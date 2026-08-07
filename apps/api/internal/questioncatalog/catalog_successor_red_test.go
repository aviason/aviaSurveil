package questioncatalog

import (
	"testing"
)

func syntheticImportRows() []ImportRow {
	rows := make([]ImportRow, 0, 1310)
	for form := 1; form <= 52; form++ {
		for ordinal := 1; ordinal <= 25; ordinal++ {
			rows = append(rows, ImportRow{
				CatalogVersion:    "aga-preprod@1.0.0",
				FormCode:          formCode(form),
				ProposalID:        "proposal-" + formCode(form),
				Ordinal:           ordinal,
				QuestionVersionID: "qv-" + formCode(form) + "-" + formatOrdinal(ordinal),
				QuestionDigest:    "digest-" + formCode(form) + "-" + formatOrdinal(ordinal),
				UsageClass:        UsageClassPreprodExercise,
			})
		}
	}
	// The sealed AGA package has 1,310 questions rather than a rectangular
	// 52-by-25 grid; ten forms carry one additional question.
	for form := 1; form <= 10; form++ {
		rows = append(rows, ImportRow{
			CatalogVersion:    "aga-preprod@1.0.0",
			FormCode:          formCode(form),
			ProposalID:        "proposal-" + formCode(form),
			Ordinal:           26,
			QuestionVersionID: "qv-" + formCode(form) + "-026",
			QuestionDigest:    "digest-" + formCode(form) + "-026",
			UsageClass:        UsageClassPreprodExercise,
		})
	}
	return rows
}

func formCode(n int) string { return "F-" + formatOrdinal(n) }
func formatOrdinal(n int) string {
	if n < 10 {
		return "00" + string(rune('0'+n))
	}
	if n < 100 {
		return "0" + string(rune('0'+n/10)) + string(rune('0'+n%10))
	}
	return string(rune('0'+n/100)) + string(rune('0'+(n/10)%10)) + string(rune('0'+n%10))
}

func TestSuccessorImportPreservesExactAGAIdentitySet(t *testing.T) {
	rows := syntheticImportRows()
	if err := ValidateImport(rows, ImportPolicy{ExpectedRows: 1310, ExpectedForms: 52}); err != nil {
		t.Fatalf("RED successor import contract: exact 1,310-row/52-form validation is missing: %v", err)
	}
	if got := ImportDigest(rows); got == "" {
		t.Fatal("RED successor import contract: deterministic import digest is missing")
	}
}

func TestSuccessorCatalogSelectionIsBoundedAndCASProtected(t *testing.T) {
	catalog, err := NewCatalog(syntheticImportRows(), CatalogPolicy{UsageClass: UsageClassPreprodExercise})
	if err != nil {
		t.Fatal(err)
	}
	preview, err := catalog.PreviewSelection(SelectionRequest{QuestionVersionIDs: []string{"qv-F-001-001", "qv-F-001-002"}, MaxBatch: 500})
	if err != nil || preview.Count != 2 || preview.SelectionDigest == "" {
		t.Fatalf("RED successor scope contract: bounded preview/digest is missing: preview=%+v err=%v", preview, err)
	}
	if _, err := catalog.CommitSelection(SelectionRequest{QuestionVersionIDs: []string{"qv-F-001-001"}, ExpectedDigest: "stale"}); err == nil {
		t.Fatal("RED successor scope contract: stale selection CAS must fail closed")
	}
}

func TestSuccessorImportCannotCopyBodiesOrCrossUsageClass(t *testing.T) {
	rows := syntheticImportRows()
	rows[0].Body = "question body must remain in question_versions"
	if err := ValidateImport(rows, ImportPolicy{ExpectedRows: 1310, ExpectedForms: 52}); err == nil {
		t.Fatal("RED successor authority contract: catalog import must reject copied question bodies")
	}
	rows = syntheticImportRows()
	rows[0].UsageClass = UsageClassGovernedOperational
	if _, err := NewCatalog(rows, CatalogPolicy{UsageClass: UsageClassPreprodExercise}); err == nil {
		t.Fatal("RED successor usage boundary: exercise catalog must reject governed records")
	}
}
