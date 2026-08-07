// Package canonicalaga adapts the sealed AGA intake package to the canonical
// question catalog.  It is an import/provenance boundary, not a stakeholder
// lifecycle store: question text is written only through question_versions;
// catalog rows retain immutable IDs, digests, and lineage.
package canonicalaga

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agacandidatedemo"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/questioncatalog"
)

type QuestionVersionImport struct {
	ID                  string
	QuestionID          string
	Version             int
	Prompt              string
	ConfiguredReference string
	ExpectedEvidence    string
	FormCode            string
	ProposalID          string
	Ordinal             int
	TextDigest          string
	SourceLocator       string
	SourceGapState      string
	ProposedDomain      string
	ProposedTopic       string
	ProposedRiskBand    string
	UsageClass          questioncatalog.UsageClass
}

type ImportManifest struct {
	CatalogVersion   string
	UsageClass       questioncatalog.UsageClass
	Forms            []CatalogFormImport
	Rows             []questioncatalog.ImportRow
	QuestionVersions []QuestionVersionImport
	ImportDigest     string
}

// CatalogFormImport preserves every form boundary, including zero-question
// forms. It contains lineage only; immutable question bodies remain owned by
// question_versions.
type CatalogFormImport struct {
	FormCode       string
	FormDigest     string
	ArchiveDigest  string
	QuestionCount  int
	SourceGapState string
}

func BuildImportManifest(pkg agacandidatedemo.AcceptedPackage, catalogVersion string) (ImportManifest, error) {
	if strings.TrimSpace(catalogVersion) == "" {
		return ImportManifest{}, fmt.Errorf("catalog version is required")
	}
	if len(pkg.Forms) != 52 {
		return ImportManifest{}, fmt.Errorf("AGA import requires 52 forms, got %d", len(pkg.Forms))
	}
	forms := make([]CatalogFormImport, 0, len(pkg.Forms))
	rows := make([]questioncatalog.ImportRow, 0, 1310)
	versions := make([]QuestionVersionImport, 0, 1310)
	for _, form := range pkg.Forms {
		if strings.TrimSpace(form.FormCode) == "" {
			return ImportManifest{}, fmt.Errorf("form code is required")
		}
		forms = append(forms, CatalogFormImport{
			FormCode: form.FormCode, FormDigest: form.FormSHA256,
			ArchiveDigest: form.ArchiveSHA256, QuestionCount: len(form.Questions),
			SourceGapState: form.SourceMappingState,
		})
		for _, question := range form.Questions {
			if strings.TrimSpace(question.ProposalID) == "" || question.Ordinal < 1 || strings.TrimSpace(question.OriginalText) == "" {
				return ImportManifest{}, fmt.Errorf("invalid AGA question identity in %s", form.FormCode)
			}
			computedDigest := digestText(question.OriginalText)
			if computedDigest != question.TextDigest {
				return ImportManifest{}, fmt.Errorf("text digest mismatch for %s/%s", form.FormCode, question.ProposalID)
			}
			id := "qv:aga-preprod:" + catalogVersion + ":" + form.FormCode + ":" + question.ProposalID
			questionID := "aga:" + form.FormCode + ":" + question.ProposalID
			gapState := question.SourceMappingState
			if strings.TrimSpace(gapState) == "" {
				gapState = "SOURCE_MAPPING_REQUIRED"
			}
			row := questioncatalog.ImportRow{
				CatalogVersion: catalogVersion,
				FormCode:       form.FormCode, ProposalID: question.ProposalID,
				Ordinal: question.Ordinal, QuestionVersionID: id,
				QuestionDigest: computedDigest,
				UsageClass:     questioncatalog.UsageClassPreprodExercise,
			}
			rows = append(rows, row)
			versions = append(versions, QuestionVersionImport{
				ID: id, QuestionID: questionID, Version: 1,
				Prompt:              question.OriginalText,
				ConfiguredReference: strings.Join(question.SourceRefs, ","),
				ExpectedEvidence:    "Evidence required for the selected inspection question",
				FormCode:            form.FormCode, ProposalID: question.ProposalID,
				Ordinal: question.Ordinal, TextDigest: computedDigest,
				SourceLocator: question.SourceLocator, SourceGapState: gapState,
				ProposedDomain: question.ProposedRisk.Domain, ProposedRiskBand: question.ProposedRisk.Band,
				UsageClass: questioncatalog.UsageClassPreprodExercise,
			})
		}
	}
	if len(rows) != 1310 {
		return ImportManifest{}, fmt.Errorf("AGA import requires 1,310 questions, got %d", len(rows))
	}
	// Twenty-one accepted forms intentionally have zero question rows; the
	// complete 52-form boundary is validated by the form projection above.
	if err := questioncatalog.ValidateImport(rows, questioncatalog.ImportPolicy{ExpectedRows: 1310}); err != nil {
		return ImportManifest{}, err
	}
	return ImportManifest{
		CatalogVersion: catalogVersion,
		UsageClass:     questioncatalog.UsageClassPreprodExercise,
		Forms:          forms,
		Rows:           rows, QuestionVersions: versions,
		ImportDigest: questioncatalog.ImportDigest(rows),
	}, nil
}

func digestText(value string) string {
	digest := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(digest[:])
}
