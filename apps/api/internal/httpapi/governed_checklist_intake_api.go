package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/application"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/checklistintake"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/httpapi/generated"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/go-chi/chi/v5"
)

var errChecklistImportIdempotencyMismatch = errors.New("checklist import idempotency key does not match receipt")

func (api *CanonicalAPI) createAdminChecklistImportBatch(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requireChecklistAdmin(writer, request)
	if !ok {
		return
	}
	headerIdempotencyKey := strings.TrimSpace(request.Header.Get("Idempotency-Key"))
	if headerIdempotencyKey == "" {
		api.respond(writer, nil, fmt.Errorf("%w: Idempotency-Key header is required", application.ErrInvalid))
		return
	}
	archive, receipt, cleanup, err := decodeChecklistImportMultipart(writer, request, headerIdempotencyKey)
	if err != nil {
		if errors.Is(err, errChecklistImportIdempotencyMismatch) {
			writeProblem(writer, http.StatusBadRequest, "Invalid idempotency binding", "Idempotency-Key header does not match the first receipt part", "IDEMPOTENCY_KEY_MISMATCH")
			return
		}
		api.respond(writer, nil, err)
		return
	}
	defer cleanup()
	result, err := api.checklistIntake.ReceiveArchiveReader(request.Context(), actor, receipt.OperationId, receipt.IdempotencyKey, receipt.ExpectedArchiveSha256, receipt.Reason, archive)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	api.respondCreated(writer, generated.ChecklistImportBatchReceiptView{Batch: checklistImportBatchView(result.Batch), Replayed: result.Replayed}, nil)
}

func decodeChecklistImportMultipart(writer http.ResponseWriter, request *http.Request, headerIdempotencyKey string) (*os.File, generated.CreateChecklistImportBatchReceiptInput, func(), error) {
	policy := checklistintake.AGAZipPDFV1()
	contentType := request.Header.Get("Content-Type")
	if !strings.HasPrefix(strings.ToLower(contentType), "multipart/form-data;") {
		return nil, generated.CreateChecklistImportBatchReceiptInput{}, func() {}, application.ErrInvalid
	}
	request.Body = http.MaxBytesReader(writer, request.Body, policy.MaxArchiveBytes+1<<20)
	multipartReader, err := request.MultipartReader()
	if err != nil {
		return nil, generated.CreateChecklistImportBatchReceiptInput{}, func() {}, fmt.Errorf("%w: invalid multipart body", application.ErrInvalid)
	}
	var archiveFile *os.File
	archiveCleanup := func() {
		if archiveFile == nil {
			return
		}
		name := archiveFile.Name()
		_ = archiveFile.Close()
		_ = os.Remove(name)
	}
	var receipt generated.CreateChecklistImportBatchReceiptInput
	archiveSeen := false
	firstPart, firstErr := multipartReader.NextPart()
	if firstErr != nil {
		archiveCleanup()
		return nil, generated.CreateChecklistImportBatchReceiptInput{}, func() {}, fmt.Errorf("%w: receipt must be the first multipart part", application.ErrInvalid)
	}
	if firstPart.FormName() != "receipt" {
		archiveCleanup()
		return nil, generated.CreateChecklistImportBatchReceiptInput{}, func() {}, fmt.Errorf("%w: receipt must be the first multipart part", application.ErrInvalid)
	}
	partType := strings.ToLower(strings.TrimSpace(strings.Split(firstPart.Header.Get("Content-Type"), ";")[0]))
	if partType != "" && partType != "application/json" {
		archiveCleanup()
		return nil, generated.CreateChecklistImportBatchReceiptInput{}, func() {}, fmt.Errorf("%w: receipt part must be JSON", application.ErrInvalid)
	}
	receiptBytes, readErr := io.ReadAll(io.LimitReader(firstPart, 64*1024+1))
	if readErr != nil || len(receiptBytes) > 64*1024 {
		archiveCleanup()
		return nil, generated.CreateChecklistImportBatchReceiptInput{}, func() {}, fmt.Errorf("%w: receipt is too large", application.ErrInvalid)
	}
	decoder := json.NewDecoder(strings.NewReader(string(receiptBytes)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&receipt); err != nil {
		archiveCleanup()
		return nil, generated.CreateChecklistImportBatchReceiptInput{}, func() {}, fmt.Errorf("%w: receipt is incomplete", application.ErrInvalid)
	}
	if receipt.IdempotencyKey != headerIdempotencyKey {
		archiveCleanup()
		return nil, generated.CreateChecklistImportBatchReceiptInput{}, func() {}, errChecklistImportIdempotencyMismatch
	}

	for {
		part, nextErr := multipartReader.NextPart()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			archiveCleanup()
			return nil, generated.CreateChecklistImportBatchReceiptInput{}, func() {}, fmt.Errorf("%w: invalid multipart body", application.ErrInvalid)
		}
		fieldName := part.FormName()
		switch fieldName {
		case "archive":
			if archiveSeen || strings.TrimSpace(part.FileName()) == "" {
				archiveCleanup()
				return nil, generated.CreateChecklistImportBatchReceiptInput{}, func() {}, fmt.Errorf("%w: exactly one binary archive part is required", application.ErrInvalid)
			}
			archiveSeen = true
			archiveFile, err = os.CreateTemp("", "avia-aga-intake-")
			if err != nil {
				archiveCleanup()
				return nil, generated.CreateChecklistImportBatchReceiptInput{}, func() {}, fmt.Errorf("%w: archive scratch file cannot be created", application.ErrInvalid)
			}
			if err := archiveFile.Chmod(0o600); err != nil {
				archiveCleanup()
				return nil, generated.CreateChecklistImportBatchReceiptInput{}, func() {}, fmt.Errorf("%w: archive scratch file cannot be secured", application.ErrInvalid)
			}
			written, copyErr := io.Copy(archiveFile, io.LimitReader(part, policy.MaxArchiveBytes+1))
			if copyErr != nil || written > policy.MaxArchiveBytes {
				archiveCleanup()
				return nil, generated.CreateChecklistImportBatchReceiptInput{}, func() {}, fmt.Errorf("%w: archive exceeds AGA_ZIP_PDF_V1 limit", application.ErrInvalid)
			}
		case "receipt":
			archiveCleanup()
			return nil, generated.CreateChecklistImportBatchReceiptInput{}, func() {}, fmt.Errorf("%w: exactly one JSON receipt part is required and must be first", application.ErrInvalid)
		default:
			archiveCleanup()
			return nil, generated.CreateChecklistImportBatchReceiptInput{}, func() {}, fmt.Errorf("%w: only archive and receipt parts are allowed", application.ErrInvalid)
		}
	}
	if !archiveSeen || strings.TrimSpace(receipt.OperationId) == "" || strings.TrimSpace(receipt.IdempotencyKey) == "" || strings.TrimSpace(receipt.ExpectedArchiveSha256) == "" || strings.TrimSpace(receipt.Reason) == "" {
		archiveCleanup()
		return nil, generated.CreateChecklistImportBatchReceiptInput{}, func() {}, fmt.Errorf("%w: receipt is incomplete and exactly one archive/receipt pair is required", application.ErrInvalid)
	}
	if _, err := archiveFile.Seek(0, io.SeekStart); err != nil {
		archiveCleanup()
		return nil, generated.CreateChecklistImportBatchReceiptInput{}, func() {}, fmt.Errorf("%w: archive scratch file cannot be rewound", application.ErrInvalid)
	}
	return archiveFile, receipt, archiveCleanup, nil
}

func checklistImportBatchView(batch checklistintake.ImportBatch) generated.ChecklistImportBatchView {
	manifest := batch.ManifestDigest
	var manifestPtr *string
	if manifest != "" {
		manifestPtr = &manifest
	}
	issues := []string{}
	if batch.Status == checklistintake.ImportBatchProcessing {
		issues = append(issues, "SECURITY_PHASES_PENDING")
	} else if !batch.IntakeSafetyEligible {
		issues = append(issues, "NO_PDF_ENTRIES")
	}
	expected := batch.ExpectedArchiveSHA
	if !strings.HasPrefix(expected, "sha256:") {
		expected = "sha256:" + expected
	}
	return generated.ChecklistImportBatchView{ImportBatchId: batch.ImportBatchID, ExpectedArchiveSha256: expected, Status: string(batch.Status), ManifestDigest: manifestPtr, FileCount: 0, RegisterCount: 0, BlockingIssues: issues}
}

func (api *CanonicalAPI) getAdminChecklistImportBatch(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requireChecklistAdmin(writer, request)
	if !ok {
		return
	}
	batch, found, err := api.checklistIntake.GetBatch(actor, chi.URLParam(request, "importBatchId"))
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	if !found {
		api.respond(writer, nil, application.ErrNotFound)
		return
	}
	api.respond(writer, checklistImportBatchView(batch), nil)
}

func (api *CanonicalAPI) listAdminChecklistImportFiles(writer http.ResponseWriter, request *http.Request) {
	if _, ok := requireChecklistAdmin(writer, request); !ok {
		return
	}
	api.respond(writer, generated.ChecklistImportFilePage{Items: []generated.ChecklistImportFileView{}, NextCursor: nil}, nil)
}

func (api *CanonicalAPI) listAdminChecklistImportReceipts(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requireChecklistAdmin(writer, request)
	if !ok {
		return
	}
	receipts, err := api.checklistIntake.ListPhaseReceipts(actor, chi.URLParam(request, "importBatchId"))
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	items := make([]map[string]any, 0, len(receipts))
	for _, receipt := range receipts {
		items = append(items, map[string]any{
			"receiptId": receipt.ReceiptID, "importBatchId": receipt.ImportBatchID, "phase": receipt.Phase,
			"inputDigest": receipt.InputDigest, "policyVersion": receipt.PolicyVersion,
			"resultDigest": receipt.ResultDigest, "outcome": receipt.Outcome, "errorCode": receipt.ErrorCode,
			"payload": receipt.Payload, "createdAt": receipt.CreatedAt,
		})
	}
	api.respond(writer, generated.ChecklistImportReceiptPage{Items: items, NextCursor: nil}, nil)
}

func (api *CanonicalAPI) createAdminChecklistImportFileExtractionReview(writer http.ResponseWriter, request *http.Request) {
	api.blockedCandidateCommand(writer, request, "private parser output and extraction decision packet are not available in the local profile")
}

func (api *CanonicalAPI) getAdminChecklistImportFileExtractionReview(writer http.ResponseWriter, request *http.Request) {
	api.blockedCandidateCommand(writer, request, "private extraction review is blocked until a durable parser-output receipt exists")
}

func (api *CanonicalAPI) resolveAdminChecklistImportFileIdentity(writer http.ResponseWriter, request *http.Request) {
	api.blockedCandidateCommand(writer, request, "Form 048 identity decisions require the named current Admin")
}

func (api *CanonicalAPI) createAdminExistingChecklistCandidate(writer http.ResponseWriter, request *http.Request) {
	api.blockedCandidateCommand(writer, request, "candidate import is blocked until the private extraction packet and identity resolution exist")
}

func (api *CanonicalAPI) listGovernedChecklistSourceReviewQueue(writer http.ResponseWriter, request *http.Request) {
	api.blockedCandidateCommand(writer, request, "source-owner assignment provisioning remains blocked")
}

func (api *CanonicalAPI) getGovernedChecklistSourceReviewItem(writer http.ResponseWriter, request *http.Request) {
	api.blockedCandidateCommand(writer, request, "source-review detail is assignment scoped and unavailable without a current assignment")
}

func (api *CanonicalAPI) listGovernedChecklistReviewerQueue(writer http.ResponseWriter, request *http.Request) {
	api.blockedCandidateCommand(writer, request, "reviewer assignment provisioning remains blocked")
}

func (api *CanonicalAPI) attestRegulatorySourceAuthority(writer http.ResponseWriter, request *http.Request) {
	api.blockedCandidateCommand(writer, request, "real source-authority decisions are an external owner dependency")
}

func (api *CanonicalAPI) getExistingChecklistCandidate(writer http.ResponseWriter, request *http.Request) {
	api.blockedCandidateCommand(writer, request, "real AGA candidate persistence is not available in the local profile")
}

func (api *CanonicalAPI) createDraftFromExistingChecklistCandidate(writer http.ResponseWriter, request *http.Request) {
	api.blockedCandidateCommand(writer, request, "candidate Draft creation requires a current immutable candidate leaf")
}

func (api *CanonicalAPI) createOfficialSourceChecklistDraft(writer http.ResponseWriter, request *http.Request) {
	api.blockedCandidateCommand(writer, request, "official-source Draft creation requires current source authority and server-derived owners")
}

func (api *CanonicalAPI) getGovernedChecklistDraft(writer http.ResponseWriter, request *http.Request) {
	api.blockedCandidateCommand(writer, request, "governed Draft persistence is unavailable in the local profile")
}

func (api *CanonicalAPI) createHybridReconciledChecklistDraft(writer http.ResponseWriter, request *http.Request) {
	api.blockedCandidateCommand(writer, request, "hybrid reconciliation requires a current candidate and complete source mapping")
}

func (api *CanonicalAPI) listGovernedChecklistReviewComments(writer http.ResponseWriter, request *http.Request) {
	api.blockedCandidateCommand(writer, request, "review comments are unavailable without a persisted candidate Draft")
}

func (api *CanonicalAPI) createGovernedChecklistReviewComment(writer http.ResponseWriter, request *http.Request) {
	api.blockedCandidateCommand(writer, request, "review comments are unavailable without a persisted candidate Draft")
}

func (api *CanonicalAPI) attestGovernedChecklistSourceMapping(writer http.ResponseWriter, request *http.Request) {
	api.blockedCandidateCommand(writer, request, "source mapping attestation requires the complete reviewed-source-set assignment")
}

func (api *CanonicalAPI) evaluateGovernedChecklistAuditPackageEligibility(writer http.ResponseWriter, request *http.Request) {
	api.blockedCandidateCommand(writer, request, "eligibility is read-only and requires a real published version")
}

func (api *CanonicalAPI) blockedCandidateCommand(writer http.ResponseWriter, request *http.Request, reason string) {
	if _, ok := requirePrincipal(writer, request); !ok {
		return
	}
	// Decode keyed mutation bodies before returning the blocker so malformed
	// requests cannot be used to probe or mutate a hidden candidate record.
	if request.Method != http.MethodGet && request.Body != nil {
		var raw json.RawMessage
		if !decodeJSON(writer, request, &raw) {
			return
		}
	}
	api.respond(writer, nil, fmt.Errorf("%w: %s", application.ErrForbidden, reason))
}

func requireChecklistAdmin(writer http.ResponseWriter, request *http.Request) (identity.Principal, bool) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return identity.Principal{}, false
	}
	if !checklistintake.CanReceiveArchive(actor) {
		writeProblem(writer, http.StatusForbidden, "Forbidden", "Admin candidate-only intake authority is required", "FORBIDDEN")
		return identity.Principal{}, false
	}
	return actor, true
}
