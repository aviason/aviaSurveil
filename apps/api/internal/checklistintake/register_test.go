package checklistintake

import "testing"

func TestMatchRegisterRequiresSuccessfulRegisterReceipt(t *testing.T) {
	files := []ImportFile{{ImportFileID: "f048", RegisterFormCode: "FSS-AGA-FORM-048"}}
	entries := []RegisterRow{{Page: 1, RowNumber: 1, FormCode: "FSS-AGA-FORM-048", Title: "Checklist"}}
	for _, receipt := range []PhaseReceipt{{Phase: PhaseRegisterParse, Outcome: ReceiptFailed}, {Phase: PhasePDFParse, Outcome: ReceiptSucceeded}} {
		if _, err := MatchRegister(receipt, entries, files); err == nil {
			t.Fatalf("receipt %+v must not authorize identity matching", receipt)
		}
	}
}

func TestMatchRegisterKeepsDuplicateAndExtraFactsExplicit(t *testing.T) {
	receipt := PhaseReceipt{Phase: PhaseRegisterParse, Outcome: ReceiptSucceeded, ResultDigest: "sha256:register"}
	entries := []RegisterRow{
		{Page: 1, RowNumber: 1, FormCode: "FSS-AGA-FORM-048", Title: "Checklist"},
		{Page: 1, RowNumber: 2, FormCode: "FSS-AGA-FORM-048", Title: "Duplicate"},
	}
	files := []ImportFile{{ImportFileID: "f048", RegisterFormCode: "FSS-AGA-FORM-048"}, {ImportFileID: "extra", RegisterFormCode: "FSS-AGA-FORM-999"}}
	matched, err := MatchRegister(receipt, entries, files)
	if err == nil {
		t.Fatalf("duplicate/extra register facts must remain an explicit error: matched=%+v", matched)
	}
}

func TestParseRegisterRejectsEmptyOrDuplicateFormCodes(t *testing.T) {
	if _, err := ParseRegister([]RegisterRow{{Page: 1, RowNumber: 1, FormCode: ""}}); err == nil {
		t.Fatal("empty form code was accepted")
	}
	if _, err := ParseRegister([]RegisterRow{{Page: 1, RowNumber: 1, FormCode: "FSS-AGA-FORM-048"}, {Page: 1, RowNumber: 2, FormCode: "FSS-AGA-FORM-048"}}); err == nil {
		t.Fatal("duplicate form code was accepted")
	}
}
