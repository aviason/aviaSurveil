package agacandidatedemo

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"time"
)

// ProjectionStore is the only database seam available to the one-shot AGA
// overlay loader. Its implementations must use the dedicated overlay schema;
// it deliberately exposes no governed-domain operation.
type ProjectionStore interface {
	Preflight(context.Context, IntentManifest) error
	Materialize(context.Context, IntentManifest, AcceptedPackage, map[string]string) (SealReceipt, error)
	VerifySeal(context.Context, IntentManifest) (SealReceipt, error)
}

// SealReceipt is reconstructed exclusively from the committed final seal. It
// contains no candidate text and is the database readability receipt.
type SealReceipt struct {
	PackageDigest        string
	IntentDigest         string
	TargetDigest         string
	ReconciliationDigest string
	SealDigest           string
	SealedAt             time.Time
}

// sourceResolutionRequirements are persisted inside the reconciliation-bound
// package projection. They describe a future governed-source gate only; they
// neither attest a source nor authorize any transition.
var sourceResolutionRequirements = []string{
	"EXACT_SOURCE_BYTES",
	"EXACT_SOURCE_BYTES_SHA256",
	"EFFECTIVE_DATE",
	"CLAUSE_OR_PAGE_LOCATOR",
	"APPLICABILITY",
	"NAMED_SOURCE_OWNER_ATTESTATION",
}

type packageProjectionRow struct {
	Identity                     PackageIdentity `json:"identity"`
	SourceResolutionRequirements []string        `json:"sourceResolutionRequirements"`
	SourceReferenceCount         int             `json:"sourceReferenceCount"`
}

func projectionPackage(pkg AcceptedPackage) packageProjectionRow {
	return packageProjectionRow{
		Identity:                     pkg.Identity,
		SourceResolutionRequirements: slices.Clone(sourceResolutionRequirements),
		SourceReferenceCount:         len(pkg.SourceCoverage),
	}
}

type formProjectionRow struct {
	Ordinal                 int           `json:"ordinal"`
	Form                    FormCandidate `json:"form"`
	FormSourceProposalCount int           `json:"formSourceProposalCount"`
}
type sourceCatalogProjectionRow struct {
	Ordinal int            `json:"ordinal"`
	Source  SourceProposal `json:"source"`
}
type formSourceProjectionRow struct {
	FormCode string         `json:"formCode"`
	Ordinal  int            `json:"ordinal"`
	Source   SourceProposal `json:"source"`
}
type questionProjectionRow struct {
	FormCode            string            `json:"formCode"`
	FormOrdinal         int               `json:"formOrdinal"`
	Question            QuestionCandidate `json:"question"`
	SourceProposalCount int               `json:"sourceProposalCount"`
}
type questionSourceProjectionRow struct {
	ProposalID string         `json:"proposalId"`
	Ordinal    int            `json:"ordinal"`
	Source     SourceProposal `json:"source"`
}

func (receipt SealReceipt) Validate(intent IntentManifest) error {
	if !validDigest(receipt.PackageDigest) || receipt.IntentDigest != intent.IntentDigest || receipt.TargetDigest != intent.TargetFingerprintDigest || !validDigest(receipt.ReconciliationDigest) || !validDigest(receipt.SealDigest) || receipt.SealedAt.IsZero() {
		return fmt.Errorf("invalid AGA demo seal receipt")
	}
	return nil
}

func projectionForm(ordinal int, form FormCandidate) formProjectionRow {
	proposalCount := len(form.FormSourceProposals)
	form.FormSourceProposals = nil
	form.Questions = nil
	return formProjectionRow{
		Ordinal:                 ordinal,
		Form:                    form,
		FormSourceProposalCount: proposalCount,
	}
}

func projectionQuestion(
	formCode string,
	formOrdinal int,
	question QuestionCandidate,
) questionProjectionRow {
	proposalCount := len(question.SourceProposals)
	question.SourceProposals = nil
	return questionProjectionRow{
		FormCode:            formCode,
		FormOrdinal:         formOrdinal,
		Question:            question,
		SourceProposalCount: proposalCount,
	}
}

// RelationshipDigests is a versioned, domain-separated digest set. Each row
// digest binds all persisted/displayed fields in that row; each relation digest
// binds the sorted ordered-row sequence. This is intentionally separate from
// the package JSON digest so a projection cannot silently omit a field.
func RelationshipDigests(pkg AcceptedPackage) (map[string]string, error) {
	if err := validateAcceptedForProjection(pkg); err != nil {
		return nil, err
	}
	rows := map[string][][]byte{
		"package":                 {mustCanonical(projectionPackage(pkg))},
		"forms":                   make([][]byte, 0, len(pkg.Forms)),
		"formSourceProposals":     nil,
		"sourceReferenceCatalog":  make([][]byte, 0, len(pkg.SourceCoverage)),
		"questions":               nil,
		"questionSourceProposals": nil,
	}
	for index, source := range pkg.SourceCoverage {
		rows["sourceReferenceCatalog"] = append(rows["sourceReferenceCatalog"], mustCanonical(sourceCatalogProjectionRow{index + 1, source}))
	}
	for formIndex, form := range pkg.Forms {
		rows["forms"] = append(rows["forms"], mustCanonical(projectionForm(formIndex+1, form)))
		for index, source := range form.FormSourceProposals {
			rows["formSourceProposals"] = append(rows["formSourceProposals"], mustCanonical(formSourceProjectionRow{form.FormCode, index + 1, source}))
		}
		for _, question := range form.Questions {
			rows["questions"] = append(rows["questions"], mustCanonical(projectionQuestion(form.FormCode, formIndex+1, question)))
			for index, source := range question.SourceProposals {
				rows["questionSourceProposals"] = append(rows["questionSourceProposals"], mustCanonical(questionSourceProjectionRow{question.ProposalID, index + 1, source}))
			}
		}
	}
	output := make(map[string]string, len(rows)+2)
	for relation, values := range rows {
		output[relation] = relationDigest(relation, values)
	}
	output["sourceResolutionRequirements"] = relationDigest("sourceResolutionRequirements", [][]byte{
		[]byte("EXACT_SOURCE_BYTES"), []byte("EXACT_SOURCE_BYTES_SHA256"), []byte("EFFECTIVE_DATE"), []byte("CLAUSE_OR_PAGE_LOCATOR"), []byte("APPLICABILITY"), []byte("NAMED_SOURCE_OWNER_ATTESTATION"),
	})
	output["projection"] = relationDigest("projection", orderedDigestValues(output))
	return output, nil
}

func validateAcceptedForProjection(pkg AcceptedPackage) error {
	if !validDigest(pkg.Identity.ZipSHA256) || !validDigest(pkg.Identity.JSONSHA256) || !validDigest(pkg.Identity.ManifestSHA256) || pkg.Identity.ZipBytes <= 0 || pkg.Identity.JSONBytes <= 0 || len(pkg.Forms) == 0 {
		return fmt.Errorf("invalid accepted AGA package for projection")
	}
	return nil
}

func mustCanonical(value any) []byte {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("canonical AGA projection value: %v", err))
	}
	return encoded
}

func relationDigest(relation string, values [][]byte) string {
	rows := make([]string, 0, len(values))
	for _, value := range values {
		rows = append(rows, digestBytes(value))
	}
	slices.Sort(rows)
	return digestBytes([]byte("AVIA_AGA_CANDIDATE_DEMO_RELATION_V1\n" + relation + "\n" + joinLines(rows)))
}

func orderedDigestValues(values map[string]string) [][]byte {
	keys := slices.Collect(maps.Keys(values))
	slices.Sort(keys)
	output := make([][]byte, 0, len(keys))
	for _, key := range keys {
		output = append(output, []byte(key+"="+values[key]))
	}
	return output
}

func joinLines(values []string) string {
	if len(values) == 0 {
		return ""
	}
	output := values[0]
	for _, value := range values[1:] {
		output += "\n" + value
	}
	return output
}

func exactDigestSet(expected, actual map[string]string) error {
	if !maps.Equal(expected, actual) {
		return fmt.Errorf("AGA demo relationship digest mismatch")
	}
	return nil
}
