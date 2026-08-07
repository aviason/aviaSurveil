package canonicalaga

import (
	"fmt"
	"testing"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agacandidatedemo"
)

func TestBuildImportManifestRejectsNonExactAGAInventory(t *testing.T) {
	pkg := agacandidatedemo.AcceptedPackage{Forms: make([]agacandidatedemo.FormCandidate, 52)}
	if _, err := BuildImportManifest(pkg, "aga-preprod@1.0.0"); err == nil {
		t.Fatal("expected exact question inventory validation")
	}
}

func TestBuildImportManifestRecomputesDigestsAndPreservesZeroFormBoundaries(t *testing.T) {
	pkg := agacandidatedemo.AcceptedPackage{Forms: make([]agacandidatedemo.FormCandidate, 0, 52)}
	questionNumber := 0
	for formNumber := 1; formNumber <= 52; formNumber++ {
		count := 0
		if formNumber <= 30 {
			count = 42
		} else if formNumber == 31 {
			count = 50
		}
		form := agacandidatedemo.FormCandidate{FormCode: fmt.Sprintf("FORM-%02d", formNumber), FormSHA256: fmt.Sprintf("sha256:form-%02d", formNumber), SourceMappingState: "SOURCE_MAPPING_REQUIRED"}
		for ordinal := 1; ordinal <= count; ordinal++ {
			questionNumber++
			prompt := fmt.Sprintf("Invented privacy-safe question %d", questionNumber)
			form.Questions = append(form.Questions, agacandidatedemo.QuestionCandidate{
				ProposalID: fmt.Sprintf("P-%04d", questionNumber), Ordinal: ordinal,
				OriginalText: prompt, TextDigest: digestText(prompt), SourceMappingState: "SOURCE_MAPPING_REQUIRED",
			})
		}
		pkg.Forms = append(pkg.Forms, form)
	}
	manifest, err := BuildImportManifest(pkg, "aga-preprod@1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Forms) != 52 || len(manifest.Rows) != 1310 || len(manifest.QuestionVersions) != 1310 {
		t.Fatalf("manifest inventory = forms %d rows %d versions %d", len(manifest.Forms), len(manifest.Rows), len(manifest.QuestionVersions))
	}
	zeroForms := 0
	for _, form := range manifest.Forms {
		if form.QuestionCount == 0 {
			zeroForms++
		}
	}
	if zeroForms != 21 {
		t.Fatalf("zero-form boundaries = %d, want 21", zeroForms)
	}
	if manifest.ImportDigest == "" {
		t.Fatal("expected aggregate import digest")
	}
	for _, row := range manifest.Rows {
		if row.QuestionDigest == "" || row.Body != "" {
			t.Fatalf("invalid catalog row %+v", row)
		}
	}
}
