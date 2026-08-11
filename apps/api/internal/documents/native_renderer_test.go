package documents

import (
	"bytes"
	"context"
	"os"
	"testing"
)

func testRenderSnapshot(t *testing.T) RenderSnapshot {
	t.Helper()
	content := ReportContent{
		Schema: ReportContentSchema, LanguageTag: "tr-TR",
		Title:            "Emniyet Gözetim Raporu",
		ExecutiveSummary: "İşletmenin emniyet yönetim sistemi incelendi; résultats vérifiables et ação documentada.",
		Scope:            "2026 kabin ve operasyon gözetimi.", Methodology: "Saha görüşmeleri, kayıt incelemesi ve örneklem doğrulaması.",
		Sections:        []ReportSection{{ID: "scope", Heading: "Kapsam", Paragraphs: []string{"İnceleme kapsamı ve sınırlar açıkça kaydedildi."}}},
		Findings:        []ReportFinding{{FindingID: "finding-1", Reference: "OPS-001", Title: "Kayıt bütünlüğü", Narrative: "İşletme kayıtları düzenli ve izlenebilir tutulmalıdır.", RegulatoryBasis: []string{"ICAO Annex 19"}}},
		Conclusion:      "Kapanış için doğrulanmış kanıt ve sorumlu onayı gereklidir; la clôture reste contrôlée.",
		Recommendations: []string{"Aksiyon sahiplerini ve son tarihleri gözden geçirin.", "Reveja os responsáveis e os prazos da ação."},
	}
	source, _, _, err := NewReportRenderSource("rv-1", "report-1", "org-1", "audit-1", 1, "subject-1", content)
	if err != nil {
		t.Fatalf("NewReportRenderSource() error = %v", err)
	}
	return RenderSnapshot{
		ReportVersionID: "rv-1", ReportID: "report-1", Kind: "FINAL", OrganizationID: "org-1",
		AuditID: "audit-1", Version: 1, CreatedBySubject: "subject-1", Source: source,
	}
}

func TestNativeRendererIsDeterministicAndNarrativeBound(t *testing.T) {
	renderer, err := NewNativeRenderer()
	if err != nil {
		t.Fatalf("NewNativeRenderer() error = %v", err)
	}
	snapshot := testRenderSnapshot(t)
	first, err := renderer.Render(context.Background(), snapshot)
	if err != nil {
		t.Fatalf("first Render() error = %v", err)
	}
	second, err := renderer.Render(context.Background(), snapshot)
	if err != nil {
		t.Fatalf("second Render() error = %v", err)
	}
	if first.MediaType != "application/pdf" || first.FileName != "report-1.pdf" ||
		!bytes.HasPrefix(first.Body, []byte("%PDF-")) || len(first.Body) < 2_000 {
		t.Fatalf("native artifact = filename %q media %q bytes %d", first.FileName, first.MediaType, len(first.Body))
	}
	if !bytes.Equal(first.Body, second.Body) || first.RendererHash != second.RendererHash ||
		first.TemplateHash != second.TemplateHash || first.SourceHash != second.SourceHash {
		t.Fatal("native renderer output or provenance is not deterministic")
	}
	if !validSHA256(first.RendererHash) || !validSHA256(first.TemplateHash) || !validSHA256(first.SourceHash) {
		t.Fatalf("invalid provenance: %+v", first)
	}
	if outputPath := os.Getenv("AVIA_NATIVE_PDF_OUT"); outputPath != "" {
		if err := os.WriteFile(outputPath, first.Body, 0o600); err != nil {
			t.Fatalf("write native PDF fixture: %v", err)
		}
	}
}

func TestNativeRendererRejectsMissingImmutableNarrative(t *testing.T) {
	renderer, err := NewNativeRenderer()
	if err != nil {
		t.Fatalf("NewNativeRenderer() error = %v", err)
	}
	snapshot := testRenderSnapshot(t)
	snapshot.Source.Content = ReportContent{}
	if _, err := renderer.Render(context.Background(), snapshot); err == nil {
		t.Fatal("Render() accepted a missing immutable narrative")
	}
}
