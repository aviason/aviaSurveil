package checklistintake

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
)

type Service struct {
	Store         Store
	Policy        IntakePolicy
	Clock         func() time.Time
	mu            sync.Mutex
	receipts      map[string]ImportBatch
	phaseReceipts map[string][]PhaseReceipt
}

// ReceiveResult keeps replay state explicit at the transport boundary without
// exposing the in-memory adapter as an authority source.
type ReceiveResult struct {
	Batch    ImportBatch
	Replayed bool
}

func NewService(store Store) *Service {
	return &Service{Store: store, Policy: AGAZipPDFV1(), Clock: time.Now, receipts: make(map[string]ImportBatch), phaseReceipts: make(map[string][]PhaseReceipt)}
}

func (service *Service) ReceiveArchive(ctx context.Context, principal identity.Principal, operationID, idempotencyKey, expectedSHA256, reason string, archive []byte) (ImportBatch, error) {
	result, err := service.ReceiveArchiveReader(ctx, principal, operationID, idempotencyKey, expectedSHA256, reason, bytes.NewReader(archive))
	if err != nil {
		return ImportBatch{}, err
	}
	return result.Batch, nil
}

// ReceiveArchiveReader is the bounded intake-service boundary used by HTTP.
// It spools the request to a mode-0600 task-owned file while hashing and
// enforcing the archive limit, then validates the file through ReaderAt without
// materializing the complete archive in memory.
func (service *Service) ReceiveArchiveReader(ctx context.Context, principal identity.Principal, operationID, idempotencyKey, expectedSHA256, reason string, archive io.Reader) (ReceiveResult, error) {
	if service == nil {
		return ReceiveResult{}, errors.New("checklist intake service is not configured")
	}
	if !CanReceiveArchive(principal) {
		return ReceiveResult{}, errors.New("intake authorization denied")
	}
	if archive == nil || strings.TrimSpace(operationID) == "" || strings.TrimSpace(idempotencyKey) == "" || strings.TrimSpace(expectedSHA256) == "" || strings.TrimSpace(reason) == "" {
		return ReceiveResult{}, errors.New("operation, idempotency, expected archive hash, reason, and archive are required")
	}
	temporary, err := os.CreateTemp("", "avia-aga-intake-")
	if err != nil {
		return ReceiveResult{}, fmt.Errorf("create archive scratch file: %w", err)
	}
	temporaryName := temporary.Name()
	defer func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryName)
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return ReceiveResult{}, fmt.Errorf("secure archive scratch file: %w", err)
	}
	digest := sha256.New()
	limited := io.LimitReader(archive, service.Policy.MaxArchiveBytes+1)
	archiveBytes, err := io.Copy(io.MultiWriter(temporary, digest), limited)
	if err != nil {
		return ReceiveResult{}, fmt.Errorf("stream archive: %w", err)
	}
	if archiveBytes > service.Policy.MaxArchiveBytes {
		return ReceiveResult{}, fmt.Errorf("archive exceeds AGA_ZIP_PDF_V1 limit: %w", ErrArchiveLimit)
	}
	archiveHex := hex.EncodeToString(digest.Sum(nil))
	archiveSHA := "sha256:" + archiveHex
	if strings.TrimPrefix(strings.TrimSpace(expectedSHA256), "sha256:") != archiveHex {
		return ReceiveResult{}, errors.New("expected archive hash does not match observed bytes")
	}
	key := operationID + "\x00" + idempotencyKey
	service.mu.Lock()
	if existing, exists := service.receipts[key]; exists {
		service.mu.Unlock()
		if existing.ObservedArchiveSHA != archiveSHA {
			return ReceiveResult{}, ErrIdempotencyConflict
		}
		return ReceiveResult{Batch: existing, Replayed: true}, nil
	}
	service.mu.Unlock()
	if _, err := temporary.Seek(0, io.SeekStart); err != nil {
		return ReceiveResult{}, fmt.Errorf("rewind archive scratch file: %w", err)
	}
	inventory, err := InventoryArchiveReaderAt(temporary, archiveBytes, archiveSHA, service.Policy)
	if err != nil {
		return ReceiveResult{}, fmt.Errorf("archive inventory failed: %w", err)
	}
	now := service.Clock()
	batchDigest := sha256.Sum256([]byte(operationID + ":" + archiveSHA))
	batch := ImportBatch{ImportBatchID: "import-" + hex.EncodeToString(batchDigest[:8]), OperationID: operationID, IdempotencyKey: idempotencyKey, ExpectedArchiveSHA: archiveSHA, ObservedArchiveSHA: archiveSHA, ObservedArchiveByteCount: archiveBytes, Status: ImportBatchProcessing, ManifestDigest: inventory.ManifestDigest, IntakeSafetyEligible: false, Reason: reason, CreatedBySubjectID: principal.SubjectID, CreatedAt: now}
	phaseReceipt := PhaseReceipt{ReceiptID: batch.ImportBatchID + "-archive-validate", ImportBatchID: batch.ImportBatchID, Phase: PhaseArchiveValidate, InputDigest: archiveSHA, PolicyVersion: service.Policy.Version, ResultDigest: inventory.ManifestDigest, Outcome: ReceiptSucceeded, Payload: mustJSON(map[string]any{"archiveBytes": archiveBytes, "entryCount": len(inventory.Entries), "pdfCount": inventory.PDFCount, "directoryCount": inventory.DirectoryCount}), CreatedAt: now}
	if service.Store != nil {
		if err := service.Store.WithinTransaction(ctx, func(ctx context.Context, transaction Transaction) error {
			if err := transaction.InsertImportBatch(ctx, batch); err != nil {
				return err
			}
			return transaction.InsertPhaseReceipt(ctx, phaseReceipt)
		}); err != nil {
			return ReceiveResult{}, err
		}
	}
	service.mu.Lock()
	service.receipts[key] = batch
	service.phaseReceipts[batch.ImportBatchID] = []PhaseReceipt{phaseReceipt}
	service.mu.Unlock()
	return ReceiveResult{Batch: batch}, nil
}

func (service *Service) ListPhaseReceipts(principal identity.Principal, importBatchID string) ([]PhaseReceipt, error) {
	if service == nil || !CanReceiveArchive(principal) || strings.TrimSpace(importBatchID) == "" {
		return nil, errors.New("candidate-only Admin receipts are unavailable")
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	receipts := append([]PhaseReceipt(nil), service.phaseReceipts[importBatchID]...)
	return receipts, nil
}

func mustJSON(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`{"serialization":"failed"}`)
	}
	return encoded
}

func (service *Service) ReceiveArchiveResult(ctx context.Context, principal identity.Principal, operationID, idempotencyKey, expectedSHA256, reason string, archive []byte) (ReceiveResult, error) {
	return service.ReceiveArchiveReader(ctx, principal, operationID, idempotencyKey, expectedSHA256, reason, bytes.NewReader(archive))
}

func (service *Service) GetBatch(principal identity.Principal, importBatchID string) (ImportBatch, bool, error) {
	if service == nil || !CanReceiveArchive(principal) || strings.TrimSpace(importBatchID) == "" {
		return ImportBatch{}, false, errors.New("candidate-only Admin inventory is unavailable")
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	for _, batch := range service.receipts {
		if batch.ImportBatchID == importBatchID {
			return batch, true, nil
		}
	}
	return ImportBatch{}, false, nil
}
