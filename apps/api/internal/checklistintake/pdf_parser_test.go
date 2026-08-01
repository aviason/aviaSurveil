package checklistintake

import "testing"

func TestParseBoundedPDFRejectsEncryptionAndKeepsOutputWithinPolicy(t *testing.T) {
	policy := AGAZipPDFV1()
	parsed, err := ParseBoundedPDF([]byte("%PDF-1.7\n/Type /Page\nBT (question) ET"), policy)
	if err != nil || parsed.PageCount != 1 || parsed.Text == "" {
		t.Fatalf("parsed=%+v err=%v", parsed, err)
	}
	if _, err := ParseBoundedPDF([]byte("%PDF-1.7\n/Encrypt 7 0 R"), policy); err == nil {
		t.Fatal("encrypted PDF was accepted")
	}
}

func TestParseBoundedPDFRejectsOversizedText(t *testing.T) {
	policy := AGAZipPDFV1()
	policy.MaxExtractedTextBytes = 4
	if _, err := ParseBoundedPDF([]byte("%PDF-1.7\nBT (12345) ET"), policy); err == nil {
		t.Fatal("extracted text limit was not enforced")
	}
}
