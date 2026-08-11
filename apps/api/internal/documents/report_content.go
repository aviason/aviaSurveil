package documents

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode"

	"golang.org/x/text/language"
	"golang.org/x/text/unicode/norm"
)

const (
	ReportContentSchema      = "avia.report-content/v1"
	ReportRenderSourceSchema = "avia.report-render-source/v1"
	maximumReportText        = 200_000
	maximumReportSourceBytes = 2 << 20
	maximumReportString      = 32_768
	maximumReportSections    = 128
	maximumReportFindings    = 512
	maximumReportParagraphs  = 2_048
)

// ReportContent is deliberately composed only of ordered scalar/string
// fields. Maps and opaque HTML cannot cross the immutable report boundary.
type ReportContent struct {
	Schema           string          `json:"schema"`
	LanguageTag      string          `json:"languageTag"`
	Title            string          `json:"title"`
	ExecutiveSummary string          `json:"executiveSummary"`
	Scope            string          `json:"scope"`
	Methodology      string          `json:"methodology"`
	Sections         []ReportSection `json:"sections"`
	Findings         []ReportFinding `json:"findings"`
	Conclusion       string          `json:"conclusion"`
	Recommendations  []string        `json:"recommendations"`
}

type ReportSection struct {
	ID         string   `json:"id"`
	Heading    string   `json:"heading"`
	Paragraphs []string `json:"paragraphs"`
}

type ReportFinding struct {
	FindingID       string   `json:"findingId"`
	Reference       string   `json:"reference"`
	Title           string   `json:"title"`
	Narrative       string   `json:"narrative"`
	RegulatoryBasis []string `json:"regulatoryBasis"`
}

type ReportRenderSource struct {
	Schema          string        `json:"schema"`
	ReportVersionID string        `json:"reportVersionId"`
	ReportID        string        `json:"reportId"`
	OrganizationID  string        `json:"organizationId"`
	AuditID         string        `json:"auditId"`
	Version         int64         `json:"version"`
	ActorSubjectID  string        `json:"actorSubjectId"`
	Content         ReportContent `json:"content"`
}

func DecodeReportContent(raw []byte) (ReportContent, error) {
	var content ReportContent
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&content); err != nil {
		return ReportContent{}, fmt.Errorf("decode report content: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return ReportContent{}, fmt.Errorf("report content contains trailing JSON")
		}
		return ReportContent{}, fmt.Errorf("decode trailing report content: %w", err)
	}
	if err := ValidateReportContent(&content); err != nil {
		return ReportContent{}, err
	}
	return content, nil
}

func ValidateReportContent(content *ReportContent) error {
	if content == nil || content.Schema != ReportContentSchema {
		return fmt.Errorf("report content schema must be %s", ReportContentSchema)
	}
	parsed, err := language.Parse(strings.TrimSpace(content.LanguageTag))
	if err != nil || parsed == language.Und {
		return fmt.Errorf("report languageTag must be a valid BCP-47 language tag")
	}
	content.LanguageTag = parsed.String()
	for _, field := range []*string{
		&content.Title, &content.ExecutiveSummary, &content.Scope,
		&content.Methodology, &content.Conclusion,
	} {
		if err := normalizeReportString(field, true); err != nil {
			return err
		}
	}
	if len(content.Sections) > maximumReportSections || len(content.Findings) > maximumReportFindings ||
		len(content.Recommendations) > maximumReportParagraphs {
		return fmt.Errorf("report content exceeds bounded collection limits")
	}
	seenSectionIDs := make(map[string]struct{}, len(content.Sections))
	textCount := 0
	for index := range content.Sections {
		section := &content.Sections[index]
		if err := normalizeReportString(&section.ID, true); err != nil {
			return err
		}
		if _, exists := seenSectionIDs[section.ID]; exists {
			return fmt.Errorf("report sections contain duplicate identity")
		}
		seenSectionIDs[section.ID] = struct{}{}
		if err := normalizeReportString(&section.Heading, true); err != nil {
			return err
		}
		if len(section.Paragraphs) == 0 || len(section.Paragraphs) > maximumReportParagraphs {
			return fmt.Errorf("report section paragraphs are outside bounds")
		}
		for paragraphIndex := range section.Paragraphs {
			if err := normalizeReportString(&section.Paragraphs[paragraphIndex], true); err != nil {
				return err
			}
			textCount += len([]rune(section.Paragraphs[paragraphIndex]))
		}
	}
	seenFindingIDs := make(map[string]struct{}, len(content.Findings))
	for index := range content.Findings {
		finding := &content.Findings[index]
		for _, field := range []*string{&finding.FindingID, &finding.Reference, &finding.Title, &finding.Narrative} {
			if err := normalizeReportString(field, true); err != nil {
				return err
			}
		}
		if _, exists := seenFindingIDs[finding.FindingID]; exists {
			return fmt.Errorf("report findings contain duplicate identity")
		}
		seenFindingIDs[finding.FindingID] = struct{}{}
		for basisIndex := range finding.RegulatoryBasis {
			if err := normalizeReportString(&finding.RegulatoryBasis[basisIndex], true); err != nil {
				return err
			}
		}
		textCount += len([]rune(finding.Narrative))
	}
	for index := range content.Recommendations {
		if err := normalizeReportString(&content.Recommendations[index], true); err != nil {
			return err
		}
		textCount += len([]rune(content.Recommendations[index]))
	}
	textCount += len([]rune(content.Title)) + len([]rune(content.ExecutiveSummary)) +
		len([]rune(content.Scope)) + len([]rune(content.Methodology)) + len([]rune(content.Conclusion))
	if textCount == 0 || textCount > maximumReportText {
		return fmt.Errorf("report content text is outside bounds")
	}
	return nil
}

func normalizeReportString(value *string, required bool) error {
	if value == nil {
		return fmt.Errorf("report content contains a nil string")
	}
	normalized := norm.NFC.String(strings.TrimSpace(*value))
	if required && normalized == "" {
		return fmt.Errorf("report content contains a required empty string")
	}
	if len([]rune(normalized)) > maximumReportString {
		return fmt.Errorf("report content string exceeds bounded length")
	}
	for _, r := range normalized {
		if r == '<' || r == '>' || unicode.IsControl(r) && r != '\n' && r != '\t' {
			return fmt.Errorf("report content contains HTML or control text")
		}
	}
	*value = normalized
	return nil
}

func NewReportRenderSource(reportVersionID, reportID, organizationID, auditID string, version int64, actorSubjectID string, content ReportContent) (ReportRenderSource, []byte, string, error) {
	source := ReportRenderSource{
		Schema: ReportRenderSourceSchema, ReportVersionID: strings.TrimSpace(reportVersionID),
		ReportID: strings.TrimSpace(reportID), OrganizationID: strings.TrimSpace(organizationID),
		AuditID: strings.TrimSpace(auditID), Version: version,
		ActorSubjectID: strings.TrimSpace(actorSubjectID), Content: content,
	}
	if source.ReportVersionID == "" || source.ReportID == "" || source.OrganizationID == "" ||
		source.AuditID == "" || source.ActorSubjectID == "" || source.Version <= 0 {
		return ReportRenderSource{}, nil, "", fmt.Errorf("complete immutable report render identity is required")
	}
	if err := ValidateReportContent(&source.Content); err != nil {
		return ReportRenderSource{}, nil, "", err
	}
	encoded, err := json.Marshal(source)
	if err != nil {
		return ReportRenderSource{}, nil, "", fmt.Errorf("encode report render source: %w", err)
	}
	if len(encoded) > maximumReportSourceBytes {
		return ReportRenderSource{}, nil, "", fmt.Errorf("report render source exceeds bounded size")
	}
	digest := sha256.Sum256(encoded)
	return source, encoded, "sha256:" + hex.EncodeToString(digest[:]), nil
}
