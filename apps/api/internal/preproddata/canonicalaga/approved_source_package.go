package canonicalaga

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

const (
	approvedPackageJSONName    = "AGA_ALL_FORMS_APPROVED_SOURCE_V2.json"
	approvedSourceManifestName = "AGA_APPROVED_SOURCE_MANIFEST.json"
	approvedPackageVersion     = "AGA_ALL_FORMS_APPROVED_SOURCE_V2"
	approvedPackageStatus      = "IMPORTED_APPROVED_SOURCE"
	approvedCatalogUsage       = "GOVERNED_OPERATIONAL"
	approvedCatalogOrigin      = "IMPORTED_APPROVED_SOURCE"
)

// ApprovedPackageExpectation is the release-pinned package boundary. It is a
// technical integrity contract, not a human approval record.
type ApprovedPackageExpectation struct {
	ZipBytes              int64
	ZipSHA256             string
	JSONBytes             int64
	JSONSHA256            string
	ManifestSHA256        string
	SourceManifestSHA256  string
	CatalogRootDigest     string
	SourceArchiveSHA256   string
	SourceArchiveBytes    int64
	SourceArchivePDFs     int
	HistoricalV1ZipSHA256 string
	FormCount             int
	QuestionBoundaryForms int
	QuestionCount         int
}

func ExactApprovedSourcePackage() ApprovedPackageExpectation {
	return ApprovedPackageExpectation{
		ZipBytes:             326448,
		ZipSHA256:            "sha256:f6a67d16b49d7a9a856bf458a0337f116ae2d139f7c12d7f952a5c00890764bc",
		JSONBytes:            3393847,
		JSONSHA256:           "sha256:57abb7b87ce91dc7383fb3b24426ccd542811ce979f33c6b17bcf938c3907973",
		ManifestSHA256:       "sha256:c13a113fb1d7bc667aa0146ae15ae5ddf000ff846c2499a27818e26fe02347be",
		SourceManifestSHA256: "sha256:53679bd6eccb77b2d4bf1c909cb16a3b925ca1433584d206daeaf212031877f8",
		CatalogRootDigest:    "sha256:972f06005ba7befecb480d477334ea8cee542555d39d2604607f082deeee6e48",
		SourceArchiveSHA256:  "sha256:dd819cfa6a670760e0cfceed94496e2e466dc53bac13e6fd792b1128314d6e32",
		SourceArchiveBytes:   12227415, SourceArchivePDFs: 53,
		HistoricalV1ZipSHA256: "sha256:30700a88aeb5b26514bf7eb76bef050deb08b96294db94117d185de5c9f163b2",
		FormCount:             52, QuestionBoundaryForms: 31, QuestionCount: 1310,
	}
}

type ApprovedSourcePackage struct {
	Identity             PackageIdentity
	SourceManifestSHA256 string
	CatalogRootDigest    string
	Forms                []ApprovedSourceForm
}

type ApprovedSourceForm struct {
	FormCode              string
	SourceFormSHA256      string
	SourceArchiveSHA256   string
	QuestionCount         int
	QuestionBoundaryState string
	Questions             []ApprovedSourceQuestion
}

type ApprovedSourceQuestion struct {
	QuestionID        string
	QuestionVersionID string
	Version           int
	Ordinal           int
	ProtocolCode      *string
	Page              *int
	Text              string
	TextDigest        string
	SourceLocator     string
	ProposalID        string
}

type approvedPackageDocument struct {
	SchemaVersion            int                          `json:"schemaVersion"`
	PackageVersion           string                       `json:"packageVersion"`
	Status                   string                       `json:"status"`
	CandidateOnly            bool                         `json:"candidateOnly"`
	SourceClassification     approvedSourceClassification `json:"sourceClassification"`
	SourceManifestSHA256     string                       `json:"sourceManifestSha256"`
	CatalogRootDigest        string                       `json:"catalogRootDigest"`
	CatalogUsageClass        string                       `json:"catalogUsageClass"`
	CatalogOrigin            string                       `json:"catalogOrigin"`
	OptionalEnrichmentPolicy json.RawMessage              `json:"optionalEnrichmentPolicy"`
	OriginalSourceArchive    approvedOriginalArchive      `json:"originalSourceArchive"`
	HistoricalV1             approvedHistoricalV1         `json:"historicalV1"`
	Totals                   approvedTotals               `json:"totals"`
	Forms                    []approvedFormDocument       `json:"forms"`
}

type approvedSourceClassification struct {
	Origin                string `json:"origin"`
	Authority             string `json:"authority"`
	HumanApprovalRequired bool   `json:"humanApprovalRequired"`
	DeploymentBlocking    bool   `json:"deploymentBlocking"`
}

type approvedOriginalArchive struct {
	SHA256   string `json:"sha256"`
	PDFCount int    `json:"pdfCount"`
	Bytes    int64  `json:"bytes"`
}

type approvedHistoricalV1 struct {
	ZipSHA256             string `json:"zipSha256"`
	JSONSHA256            string `json:"jsonSha256"`
	RegisterSHA256        string `json:"registerSha256"`
	DerivedRegisterSHA256 string `json:"derivedRegisterSha256"`
	PackageVersion        string `json:"packageVersion"`
}

type approvedTotals struct {
	Forms                 int `json:"forms"`
	QuestionBoundaryForms int `json:"questionBoundaryForms"`
	Questions             int `json:"questions"`
}

type approvedFormDocument struct {
	FormCode              string                     `json:"formCode"`
	DocumentTitle         *string                    `json:"documentTitle"`
	FormKind              *string                    `json:"formKind"`
	PageCount             *int                       `json:"pageCount"`
	SourceArchivePath     *string                    `json:"sourceArchivePath"`
	SourceFormSHA256      string                     `json:"sourceFormSha256"`
	SourceArchiveSHA256   string                     `json:"sourceArchiveSha256"`
	QuestionCount         int                        `json:"questionCount"`
	QuestionBoundaryState string                     `json:"questionBoundaryState"`
	Questions             []approvedQuestionDocument `json:"questions"`
}

type approvedQuestionDocument struct {
	ImmutableQuestionID        string          `json:"immutableQuestionId"`
	ImmutableQuestionVersionID string          `json:"immutableQuestionVersionId"`
	Version                    int             `json:"version"`
	Ordinal                    int             `json:"ordinal"`
	ProtocolCode               *string         `json:"protocolCode"`
	Page                       *int            `json:"page"`
	Text                       string          `json:"text"`
	TextDigest                 string          `json:"textDigest"`
	SourceLocator              string          `json:"sourceLocator"`
	ProposalID                 string          `json:"proposalId"`
	OptionalEnrichment         json.RawMessage `json:"optionalEnrichment"`
}

func ReadApprovedSourcePackage(ctx context.Context, packagePath string, expected ApprovedPackageExpectation) (ApprovedSourcePackage, error) {
	if err := ctx.Err(); err != nil {
		return ApprovedSourcePackage{}, err
	}
	if !filepath.IsAbs(packagePath) {
		return ApprovedSourcePackage{}, invalid("approved package path must be absolute")
	}
	info, err := os.Lstat(packagePath)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 || info.Size() > maxOuterBytes {
		return ApprovedSourcePackage{}, invalid("approved package must be a bounded regular non-symlink file")
	}
	fd, err := unix.Open(packagePath, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return ApprovedSourcePackage{}, invalid("open approved package: %v", err)
	}
	file := os.NewFile(uintptr(fd), packagePath)
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil || openedInfo.Size() != info.Size() || !openedInfo.Mode().IsRegular() {
		return ApprovedSourcePackage{}, invalid("approved package changed while opening")
	}
	zipDigest, err := digestFile(file)
	if err != nil {
		return ApprovedSourcePackage{}, invalid("hash approved package: %v", err)
	}
	if err := validateExpectedDigestAndBytes(openedInfo.Size(), zipDigest, expected.ZipBytes, expected.ZipSHA256, "approved ZIP"); err != nil {
		return ApprovedSourcePackage{}, err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return ApprovedSourcePackage{}, err
	}
	archive, err := zip.NewReader(file, openedInfo.Size())
	if err != nil {
		return ApprovedSourcePackage{}, invalid("read approved ZIP: %v", err)
	}
	entries, root, err := readArchive(ctx, archive)
	if err != nil {
		return ApprovedSourcePackage{}, err
	}
	manifest, ok := entries[root+manifestName]
	if !ok {
		return ApprovedSourcePackage{}, invalid("approved package manifest is missing")
	}
	manifestDigest := sha256Digest(manifest)
	if err := validateExpectedDigestAndBytes(int64(len(manifest)), manifestDigest, 0, expected.ManifestSHA256, "approved manifest"); err != nil {
		return ApprovedSourcePackage{}, err
	}
	manifestEntries, err := parseManifest(manifest)
	if err != nil {
		return ApprovedSourcePackage{}, err
	}
	if err := validateManifestEntriesFor(entries, root, manifestEntries, approvedPackageJSONName); err != nil {
		return ApprovedSourcePackage{}, err
	}
	jsonBytes, ok := entries[root+approvedPackageJSONName]
	if !ok {
		return ApprovedSourcePackage{}, invalid("approved package JSON is missing")
	}
	jsonDigest := sha256Digest(jsonBytes)
	if err := validateExpectedDigestAndBytes(int64(len(jsonBytes)), jsonDigest, expected.JSONBytes, expected.JSONSHA256, "approved package JSON"); err != nil {
		return ApprovedSourcePackage{}, err
	}
	sourceManifestBytes, ok := entries[root+approvedSourceManifestName]
	if !ok {
		return ApprovedSourcePackage{}, invalid("approved source manifest is missing")
	}
	sourceManifestDigest := sha256Digest(sourceManifestBytes)
	if expected.SourceManifestSHA256 != "" && sourceManifestDigest != expected.SourceManifestSHA256 {
		return ApprovedSourcePackage{}, invalid("approved source manifest digest mismatch")
	}
	var document approvedPackageDocument
	if err := decodeStrictJSON(jsonBytes, &document); err != nil {
		return ApprovedSourcePackage{}, invalid("approved package JSON: %v", err)
	}
	if document.SchemaVersion != 2 || document.PackageVersion != approvedPackageVersion || document.Status != approvedPackageStatus || document.CandidateOnly || document.CatalogUsageClass != approvedCatalogUsage || document.CatalogOrigin != approvedCatalogOrigin || document.SourceManifestSHA256 != sourceManifestDigest || document.CatalogRootDigest != expected.CatalogRootDigest {
		return ApprovedSourcePackage{}, invalid("approved package identity or lineage is invalid")
	}
	if document.SourceClassification.Origin != approvedCatalogOrigin || document.SourceClassification.Authority != "DIRECT_AVIATION_SOURCE_APPROVED_BY_PROJECT_OWNER" || document.SourceClassification.HumanApprovalRequired || document.SourceClassification.DeploymentBlocking {
		return ApprovedSourcePackage{}, invalid("approved source classification is invalid")
	}
	if document.OriginalSourceArchive.SHA256 != expected.SourceArchiveSHA256 || document.OriginalSourceArchive.Bytes != expected.SourceArchiveBytes || document.OriginalSourceArchive.PDFCount != expected.SourceArchivePDFs || document.HistoricalV1.ZipSHA256 != expected.HistoricalV1ZipSHA256 || document.HistoricalV1.PackageVersion != "AGA_ALL_FORMS_SOURCE_RISK_DRAFT_V1" {
		return ApprovedSourcePackage{}, invalid("approved historical source identity is invalid")
	}
	if document.Totals.Forms != expected.FormCount || document.Totals.QuestionBoundaryForms != expected.QuestionBoundaryForms || document.Totals.Questions != expected.QuestionCount || len(document.Forms) != expected.FormCount {
		return ApprovedSourcePackage{}, invalid("approved source counts are invalid")
	}
	forms := make([]ApprovedSourceForm, 0, len(document.Forms))
	seenForm := map[string]struct{}{}
	seenQuestion := map[string]struct{}{}
	rowLines := make([]string, 0, expected.QuestionCount+expected.FormCount)
	boundaryForms := 0
	questionCount := 0
	for _, form := range document.Forms {
		if form.FormCode == "" || !digestPattern.MatchString(form.SourceFormSHA256) || !digestPattern.MatchString(form.SourceArchiveSHA256) || form.QuestionCount != len(form.Questions) {
			return ApprovedSourcePackage{}, invalid("approved form identity is invalid")
		}
		if _, exists := seenForm[form.FormCode]; exists {
			return ApprovedSourcePackage{}, invalid("duplicate approved form %q", form.FormCode)
		}
		seenForm[form.FormCode] = struct{}{}
		if len(form.Questions) > 0 {
			boundaryForms++
		}
		questions := make([]ApprovedSourceQuestion, 0, len(form.Questions))
		rowLines = append(rowLines, fmt.Sprintf("form\x00%s\x00%s\x00%s\x00%d\n", form.FormCode, form.SourceFormSHA256, form.SourceArchiveSHA256, form.QuestionCount))
		for index, question := range form.Questions {
			if question.Ordinal != index+1 || question.Version != 1 || question.Text == "" || question.SourceLocator == "" || question.ProposalID == "" || !digestPattern.MatchString(question.TextDigest) || sha256Digest([]byte(question.Text)) != question.TextDigest || question.ImmutableQuestionID == "" || question.ImmutableQuestionVersionID == "" {
				return ApprovedSourcePackage{}, invalid("approved question identity is invalid for %s/%d", form.FormCode, index+1)
			}
			if _, exists := seenQuestion[question.ImmutableQuestionID]; exists {
				return ApprovedSourcePackage{}, invalid("duplicate approved question ID %q", question.ImmutableQuestionID)
			}
			if _, exists := seenQuestion[question.ImmutableQuestionVersionID]; exists {
				return ApprovedSourcePackage{}, invalid("duplicate approved question version ID %q", question.ImmutableQuestionVersionID)
			}
			seenQuestion[question.ImmutableQuestionID] = struct{}{}
			seenQuestion[question.ImmutableQuestionVersionID] = struct{}{}
			questions = append(questions, ApprovedSourceQuestion{QuestionID: question.ImmutableQuestionID, QuestionVersionID: question.ImmutableQuestionVersionID, Version: question.Version, Ordinal: question.Ordinal, ProtocolCode: question.ProtocolCode, Page: question.Page, Text: question.Text, TextDigest: question.TextDigest, SourceLocator: question.SourceLocator, ProposalID: question.ProposalID})
			rowLines = append(rowLines, fmt.Sprintf("question\x00%s\x00%d\x00%s\x00%s\x00%s\x00%s\n", form.FormCode, question.Ordinal, question.ImmutableQuestionID, question.ImmutableQuestionVersionID, question.TextDigest, question.SourceLocator))
			questionCount++
		}
		forms = append(forms, ApprovedSourceForm{FormCode: form.FormCode, SourceFormSHA256: form.SourceFormSHA256, SourceArchiveSHA256: form.SourceArchiveSHA256, QuestionCount: form.QuestionCount, QuestionBoundaryState: form.QuestionBoundaryState, Questions: questions})
	}
	if boundaryForms != expected.QuestionBoundaryForms || questionCount != expected.QuestionCount || sha256Digest([]byte(strings.Join(rowLines, ""))) != expected.CatalogRootDigest {
		return ApprovedSourcePackage{}, invalid("approved source root or question ordering mismatch")
	}
	return ApprovedSourcePackage{Identity: PackageIdentity{ZipSHA256: zipDigest, ZipBytes: openedInfo.Size(), JSONSHA256: jsonDigest, JSONBytes: int64(len(jsonBytes)), ManifestSHA256: manifestDigest, PackageVersion: document.PackageVersion, PackageStatus: document.Status}, SourceManifestSHA256: sourceManifestDigest, CatalogRootDigest: document.CatalogRootDigest, Forms: forms}, nil
}

func decodeStrictJSON(content []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.More() {
		return errors.New("trailing JSON")
	}
	return nil
}
