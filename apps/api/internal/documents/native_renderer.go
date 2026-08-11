package documents

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/signintech/gopdf"
)

// The font files are vendored from the official Noto Fonts repository and are
// hash-bound through rendererProvenance. No host font lookup is performed.
//
//go:embed fonts/NotoSans-Regular.ttf
var notoSansRegular []byte

//go:embed fonts/NotoSans-Bold.ttf
var notoSansBold []byte

const (
	pdfPageWidth    = 595.0
	pdfPageHeight   = 842.0
	pdfMarginLeft   = 54.0
	pdfMarginRight  = 54.0
	pdfMarginTop    = 52.0
	pdfMarginBottom = 50.0
	pdfHeaderY      = 28.0
	pdfFooterY      = 813.0
	pdfBodyWidth    = pdfPageWidth - pdfMarginLeft - pdfMarginRight
)

type nativeRenderer struct {
	rendererHash string
	templateHash string
	fontHash     string
}

// NewNativeRenderer creates the only production report renderer. It returns an
// error if the embedded font bytes are not valid TrueType data, keeping a bad
// release from reaching the worker loop.
func NewNativeRenderer() (Renderer, error) {
	if len(notoSansRegular) < 4 || len(notoSansBold) < 4 ||
		string(notoSansRegular[:4]) != "\x00\x01\x00\x00" ||
		string(notoSansBold[:4]) != "\x00\x01\x00\x00" {
		return nil, fmt.Errorf("embedded Noto Sans font bytes are invalid")
	}
	fontHash := digest(append(append([]byte("noto-sans-regular|"), notoSansRegular...), append([]byte("|noto-sans-bold|"), notoSansBold...)...))
	rendererHash, templateHash := rendererProvenance(fontHash)
	return nativeRenderer{rendererHash: rendererHash, templateHash: templateHash, fontHash: fontHash}, nil
}

func NativeRendererProvenance() (NativeProvenance, error) {
	if len(notoSansRegular) < 4 || len(notoSansBold) < 4 {
		return NativeProvenance{}, fmt.Errorf("embedded Noto Sans font bytes are invalid")
	}
	fontHash := digest(append(append([]byte("noto-sans-regular|"), notoSansRegular...), append([]byte("|noto-sans-bold|"), notoSansBold...)...))
	rendererHash, templateHash := rendererProvenance(fontHash)
	return NativeProvenance{
		RendererHash: rendererHash, TemplateHash: templateHash, FontHash: fontHash,
		Renderer: nativeRendererName, ModuleChecksum: nativeModuleChecksum,
		Layout: nativeLayoutVersion,
	}, nil
}

func (renderer nativeRenderer) Render(ctx context.Context, snapshot RenderSnapshot) (RenderedArtifact, error) {
	if err := ctx.Err(); err != nil {
		return RenderedArtifact{}, err
	}
	if strings.TrimSpace(snapshot.ReportVersionID) == "" ||
		strings.TrimSpace(snapshot.ReportID) == "" ||
		strings.TrimSpace(snapshot.OrganizationID) == "" ||
		snapshot.Version <= 0 {
		return RenderedArtifact{}, fmt.Errorf("complete report render identity is required")
	}
	if snapshot.Source.Schema != ReportRenderSourceSchema {
		return RenderedArtifact{}, fmt.Errorf("immutable report render source is required")
	}
	if snapshot.Source.ReportVersionID != snapshot.ReportVersionID ||
		snapshot.Source.ReportID != snapshot.ReportID ||
		snapshot.Source.OrganizationID != snapshot.OrganizationID ||
		snapshot.Source.AuditID != snapshot.AuditID ||
		snapshot.Source.Version != snapshot.Version {
		return RenderedArtifact{}, fmt.Errorf("render source identity does not match job snapshot")
	}
	if err := ValidateReportContent(&snapshot.Source.Content); err != nil {
		return RenderedArtifact{}, err
	}

	pdf := &gopdf.GoPdf{}
	pdf.Start(gopdf.Config{Unit: gopdf.UnitPT, PageSize: *gopdf.PageSizeA4})
	pdf.SetMargins(pdfMarginLeft, pdfMarginTop, pdfMarginRight, pdfMarginBottom)
	if err := pdf.AddTTFFontData("Noto Sans", notoSansRegular); err != nil {
		return RenderedArtifact{}, fmt.Errorf("register regular Noto Sans: %w", err)
	}
	if err := pdf.AddTTFFontData("Noto Sans Bold", notoSansBold); err != nil {
		return RenderedArtifact{}, fmt.Errorf("register bold Noto Sans: %w", err)
	}
	pdf.SetInfo(gopdf.PdfInfo{
		Title: snapshot.Source.Content.Title, Author: "AviaSurveil360",
		Subject: "Immutable aviation oversight report", Creator: "AviaSurveil360",
		Producer:     nativeRendererName + " " + nativeRendererVersion,
		CreationDate: time.Unix(0, 0).UTC(),
	})
	pageNumber := 0
	pdf.AddHeader(func() {
		pageNumber++
		_ = pdf.SetFont("Noto Sans", "", 8)
		pdf.SetTextColor(90, 90, 90)
		pdf.SetXY(pdfMarginLeft, pdfHeaderY)
		_ = pdf.Cell(&gopdf.Rect{W: pdfBodyWidth, H: 10}, "AviaSurveil360 · Immutable oversight report")
	})
	pdf.AddFooter(func() {
		_ = pdf.SetFont("Noto Sans", "", 8)
		pdf.SetTextColor(90, 90, 90)
		pdf.SetXY(pdfMarginLeft, pdfFooterY)
		_ = pdf.Cell(&gopdf.Rect{W: pdfBodyWidth, H: 10}, fmt.Sprintf("Page %d", pageNumber))
	})
	pdf.AddPage()

	if err := renderer.writeReport(ctx, pdf, snapshot.Source.Content); err != nil {
		return RenderedArtifact{}, err
	}
	body, err := pdf.GetBytesPdfReturnErr()
	if err != nil {
		return RenderedArtifact{}, fmt.Errorf("encode native PDF: %w", err)
	}
	if len(body) < len("%PDF-") || !bytes.Equal(body[:len("%PDF-")], []byte("%PDF-")) {
		return RenderedArtifact{}, fmt.Errorf("native renderer returned invalid PDF")
	}
	if len(body) > maximumPDFResponseSize {
		return RenderedArtifact{}, fmt.Errorf("native PDF exceeds bounded output size")
	}
	return RenderedArtifact{
		FileName:  snapshot.ReportID + ".pdf",
		MediaType: "application/pdf", Body: body,
		RendererHash: renderer.rendererHash, TemplateHash: renderer.templateHash,
		SourceHash: digest(mustRenderSourceBytes(snapshot.Source)),
	}, nil
}

func (renderer nativeRenderer) Provenance() NativeProvenance {
	return NativeProvenance{
		RendererHash: renderer.rendererHash, TemplateHash: renderer.templateHash,
		FontHash: renderer.fontHash, Renderer: nativeRendererName,
		ModuleChecksum: nativeModuleChecksum, Layout: nativeLayoutVersion,
	}
}

func mustRenderSourceBytes(source ReportRenderSource) []byte {
	_, encoded, _, err := NewReportRenderSource(source.ReportVersionID, source.ReportID,
		source.OrganizationID, source.AuditID, source.Version, source.ActorSubjectID, source.Content)
	if err != nil {
		return nil
	}
	return encoded
}

func (renderer nativeRenderer) writeReport(ctx context.Context, pdf *gopdf.GoPdf, content ReportContent) error {
	if err := renderer.writeHeading(ctx, pdf, content.Title, 20); err != nil {
		return err
	}
	for _, block := range []struct{ heading, text string }{
		{"Executive summary", content.ExecutiveSummary},
		{"Scope", content.Scope},
		{"Methodology", content.Methodology},
	} {
		if err := renderer.writeSection(ctx, pdf, block.heading, block.text); err != nil {
			return err
		}
	}
	for _, section := range content.Sections {
		if err := renderer.writeHeading(ctx, pdf, section.Heading, 14); err != nil {
			return err
		}
		for _, paragraph := range section.Paragraphs {
			if err := renderer.writeParagraph(ctx, pdf, paragraph, 10, 14); err != nil {
				return err
			}
		}
	}
	if len(content.Findings) > 0 {
		if err := renderer.writeHeading(ctx, pdf, "Findings", 14); err != nil {
			return err
		}
		for _, finding := range content.Findings {
			if err := renderer.writeHeading(ctx, pdf, finding.Reference+" · "+finding.Title, 11); err != nil {
				return err
			}
			if err := renderer.writeParagraph(ctx, pdf, finding.Narrative, 10, 14); err != nil {
				return err
			}
			if len(finding.RegulatoryBasis) > 0 {
				if err := renderer.writeParagraph(ctx, pdf, "Regulatory basis: "+strings.Join(finding.RegulatoryBasis, "; "), 9, 12); err != nil {
					return err
				}
			}
		}
	}
	if err := renderer.writeSection(ctx, pdf, "Conclusion", content.Conclusion); err != nil {
		return err
	}
	if len(content.Recommendations) > 0 {
		if err := renderer.writeHeading(ctx, pdf, "Recommendations", 14); err != nil {
			return err
		}
		for index, recommendation := range content.Recommendations {
			if err := renderer.writeParagraph(ctx, pdf, fmt.Sprintf("%d. %s", index+1, recommendation), 10, 14); err != nil {
				return err
			}
		}
	}
	return nil
}

func (renderer nativeRenderer) writeSection(ctx context.Context, pdf *gopdf.GoPdf, heading, text string) error {
	if err := renderer.writeHeading(ctx, pdf, heading, 13); err != nil {
		return err
	}
	return renderer.writeParagraph(ctx, pdf, text, 10, 14)
}

func (renderer nativeRenderer) writeHeading(ctx context.Context, pdf *gopdf.GoPdf, text string, size float64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !renderer.ensureSpace(pdf, 28) {
		pdf.AddPage()
	}
	if err := pdf.SetFont("Noto Sans Bold", "", size); err != nil {
		return err
	}
	pdf.SetTextColor(25, 45, 70)
	pdf.SetXY(pdfMarginLeft, pdf.GetY())
	if err := pdf.Cell(&gopdf.Rect{W: pdfBodyWidth, H: size + 6}, text); err != nil {
		return err
	}
	pdf.Br(size + 8)
	return nil
}

func (renderer nativeRenderer) writeParagraph(ctx context.Context, pdf *gopdf.GoPdf, text string, size, lineHeight float64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := pdf.SetFont("Noto Sans", "", size); err != nil {
		return err
	}
	pdf.SetTextColor(35, 35, 35)
	for _, paragraph := range strings.Split(text, "\n") {
		lines, err := pdf.SplitTextWithWordWrap(paragraph, pdfBodyWidth)
		if err != nil {
			if paragraph == "" {
				pdf.Br(lineHeight)
				continue
			}
			return err
		}
		for _, line := range lines {
			if err := ctx.Err(); err != nil {
				return err
			}
			if !renderer.ensureSpace(pdf, lineHeight) {
				pdf.AddPage()
			}
			pdf.SetXY(pdfMarginLeft, pdf.GetY())
			if err := pdf.Cell(&gopdf.Rect{W: pdfBodyWidth, H: lineHeight}, line); err != nil {
				return err
			}
			pdf.Br(lineHeight)
		}
		pdf.Br(4)
	}
	return nil
}

func (renderer nativeRenderer) ensureSpace(pdf *gopdf.GoPdf, height float64) bool {
	return pdf.GetY()+height <= pdfPageHeight-pdfMarginBottom-18
}
