package canonicalaga

import (
	"fmt"
	"testing"
)

func TestBuildImportManifestRejectsNonExactAGAInventory(t *testing.T) {
	pkg := AcceptedPackage{Forms: make([]FormCandidate, 52)}
	if _, err := BuildImportManifest(pkg, "aga-preprod@1.0.0"); err == nil {
		t.Fatal("expected exact question inventory validation")
	}
}

func TestBuildImportManifestRecomputesDigestsAndPreservesZeroFormBoundaries(t *testing.T) {
	pkg := AcceptedPackage{Forms: make([]FormCandidate, 0, 52)}
	questionNumber := 0
	for formNumber := 1; formNumber <= 52; formNumber++ {
		count := 0
		if formNumber <= 30 {
			count = 42
		} else if formNumber == 31 {
			count = 50
		}
		form := FormCandidate{FormCode: fmt.Sprintf("FORM-%02d", formNumber), FormSHA256: fmt.Sprintf("sha256:form-%02d", formNumber), SourceMappingState: "SOURCE_MAPPING_REQUIRED"}
		for ordinal := 1; ordinal <= count; ordinal++ {
			questionNumber++
			prompt := fmt.Sprintf("Invented privacy-safe question %d", questionNumber)
			form.Questions = append(form.Questions, QuestionCandidate{
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
	if len(manifest.ImportDigest) != len("sha256:")+64 || manifest.ImportDigest[:len("sha256:")] != "sha256:" {
		t.Fatalf("aggregate digest is not a canonical SHA-256: %q", manifest.ImportDigest)
	}
	changedForm := pkg
	changedForm.Forms = append([]FormCandidate(nil), pkg.Forms...)
	changedForm.Forms[0].FormSHA256 = "sha256:changed-form-lineage"
	changedManifest, err := BuildImportManifest(changedForm, "aga-preprod@1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if changedManifest.ImportDigest == manifest.ImportDigest {
		t.Fatal("form lineage change must change the catalog root digest")
	}
	for _, row := range manifest.Rows {
		if row.QuestionDigest == "" || row.Body != "" {
			t.Fatalf("invalid catalog row %+v", row)
		}
	}
}
