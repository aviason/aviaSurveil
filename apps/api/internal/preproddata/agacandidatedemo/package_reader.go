package agacandidatedemo

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/sys/unix"
)

const (
	maxOuterBytes    = 1 << 20
	maxEntryBytes    = 8 << 20
	maxExpandedBytes = 16 << 20
)

var ErrInvalidPackage = errors.New("invalid AGA candidate demo package")

type PackageReader interface {
	ReadAndValidate(context.Context, string, ExpectedPackage) (AcceptedPackage, error)
}

type packageReader struct{}

func NewPackageReader() PackageReader { return packageReader{} }

func (packageReader) ReadAndValidate(ctx context.Context, packagePath string, expected ExpectedPackage) (AcceptedPackage, error) {
	if err := ctx.Err(); err != nil {
		return AcceptedPackage{}, err
	}
	if !filepath.IsAbs(packagePath) {
		return AcceptedPackage{}, invalid("package path must be absolute")
	}
	info, err := os.Lstat(packagePath)
	if err != nil {
		return AcceptedPackage{}, invalid("stat package: %v", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return AcceptedPackage{}, invalid("package must be a regular non-symlink file")
	}
	if info.Size() <= 0 || info.Size() > maxOuterBytes {
		return AcceptedPackage{}, invalid("package byte size is outside the bounded limit")
	}
	fd, err := unix.Open(packagePath, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return AcceptedPackage{}, invalid("open package without following links: %v", err)
	}
	file := os.NewFile(uintptr(fd), packagePath)
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil || !openedInfo.Mode().IsRegular() || openedInfo.Size() != info.Size() {
		return AcceptedPackage{}, invalid("opened package does not match the validated regular file")
	}
	zipDigest, err := digestFile(file)
	if err != nil {
		return AcceptedPackage{}, invalid("hash package: %v", err)
	}
	if err := validateExpectedDigestAndBytes(openedInfo.Size(), zipDigest, expected.ZipBytes, expected.ZipSHA256, "ZIP"); err != nil {
		return AcceptedPackage{}, err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return AcceptedPackage{}, invalid("rewind package: %v", err)
	}
	archive, err := zip.NewReader(file, openedInfo.Size())
	if err != nil {
		return AcceptedPackage{}, invalid("read ZIP: %v", err)
	}
	entries, root, err := readArchive(ctx, archive)
	if err != nil {
		return AcceptedPackage{}, err
	}
	manifestPath := root + manifestName
	manifest, ok := entries[manifestPath]
	if !ok {
		return AcceptedPackage{}, invalid("manifest is missing")
	}
	manifestDigest := sha256Digest(manifest)
	if err := validateExpectedDigestAndBytes(int64(len(manifest)), manifestDigest, 0, expected.ManifestSHA256, "manifest"); err != nil {
		return AcceptedPackage{}, err
	}
	manifestEntries, err := parseManifest(manifest)
	if err != nil {
		return AcceptedPackage{}, err
	}
	if err := validateManifestEntries(entries, root, manifestEntries); err != nil {
		return AcceptedPackage{}, err
	}
	jsonBytes, ok := entries[root+packageJSONName]
	if !ok {
		return AcceptedPackage{}, invalid("package JSON is missing")
	}
	jsonDigest := sha256Digest(jsonBytes)
	if err := validateExpectedDigestAndBytes(int64(len(jsonBytes)), jsonDigest, expected.JSONBytes, expected.JSONSHA256, "package JSON"); err != nil {
		return AcceptedPackage{}, err
	}
	if err := validateJSONSyntax(jsonBytes); err != nil {
		return AcceptedPackage{}, invalid("package JSON syntax: %v", err)
	}
	decoded, err := decodePackage(jsonBytes)
	if err != nil {
		return AcceptedPackage{}, invalid("decode package JSON: %v", err)
	}
	accepted, err := validatePackage(decoded, expected)
	if err != nil {
		return AcceptedPackage{}, err
	}
	accepted.Identity = PackageIdentity{
		ZipSHA256: zipDigest, ZipBytes: openedInfo.Size(), JSONSHA256: jsonDigest,
		JSONBytes: int64(len(jsonBytes)), ManifestSHA256: manifestDigest,
		PackageVersion: decoded.PackageVersion, PackageStatus: decoded.Status,
	}
	return accepted, nil
}

func digestFile(file *os.File) (string, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, io.LimitReader(file, maxOuterBytes+1)); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func readArchive(ctx context.Context, archive *zip.Reader) (map[string][]byte, string, error) {
	if len(archive.File) < 2 || len(archive.File) > 16 {
		return nil, "", invalid("ZIP entry count is outside the bounded limit")
	}
	entries := make(map[string][]byte, len(archive.File))
	root := ""
	var total int64
	for _, entry := range archive.File {
		if err := ctx.Err(); err != nil {
			return nil, "", err
		}
		if entry.Flags&0x1 != 0 || entry.Flags&0x8 != 0 {
			return nil, "", invalid("encrypted or data-descriptor ZIP entry %q", entry.Name)
		}
		if entry.Method != zip.Store && entry.Method != zip.Deflate {
			return nil, "", invalid("unsupported ZIP compression for %q", entry.Name)
		}
		if entry.UncompressedSize64 > maxEntryBytes || entry.CompressedSize64 > maxEntryBytes {
			return nil, "", invalid("ZIP entry %q exceeds bounded size", entry.Name)
		}
		if err := safeEntryName(entry.Name, entry.FileInfo().IsDir()); err != nil {
			return nil, "", err
		}
		if entry.Mode()&(os.ModeSymlink|os.ModeDevice|os.ModeNamedPipe|os.ModeSocket|os.ModeIrregular) != 0 {
			return nil, "", invalid("non-regular ZIP entry %q", entry.Name)
		}
		if _, exists := entries[entry.Name]; exists {
			return nil, "", invalid("duplicate ZIP entry %q", entry.Name)
		}
		if strings.HasSuffix(entry.Name, "/") {
			if root != "" || strings.Count(strings.TrimSuffix(entry.Name, "/"), "/") != 0 {
				return nil, "", invalid("exactly one package root directory is required")
			}
			root = entry.Name
			entries[entry.Name] = nil
			continue
		}
		reader, err := entry.Open()
		if err != nil {
			return nil, "", invalid("open ZIP entry %q: %v", entry.Name, err)
		}
		data, err := io.ReadAll(io.LimitReader(reader, maxEntryBytes+1))
		closeErr := reader.Close()
		if err != nil || closeErr != nil || int64(len(data)) != int64(entry.UncompressedSize64) {
			return nil, "", invalid("read ZIP entry %q", entry.Name)
		}
		if bytes.HasPrefix(data, []byte("%PDF-")) || bytes.HasPrefix(data, []byte("PK\x03\x04")) {
			return nil, "", invalid("forbidden PDF or nested archive content in %q", entry.Name)
		}
		total += int64(len(data))
		if total > maxExpandedBytes {
			return nil, "", invalid("expanded ZIP data exceeds bounded total")
		}
		entries[entry.Name] = data
	}
	if root == "" {
		return nil, "", invalid("package root directory is missing")
	}
	for name := range entries {
		if name != root && !strings.HasPrefix(name, root) {
			return nil, "", invalid("ZIP entry %q is outside package root", name)
		}
	}
	return entries, root, nil
}

func safeEntryName(name string, directory bool) error {
	if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, "\\") || path.Clean(name) != strings.TrimSuffix(name, "/") {
		return invalid("unsafe ZIP entry name %q", name)
	}
	if strings.Contains(name, "../") || (!directory && strings.HasSuffix(name, "/")) {
		return invalid("unsafe ZIP entry name %q", name)
	}
	return nil
}

func parseManifest(content []byte) (map[string]string, error) {
	lines := strings.Split(strings.TrimSuffix(string(content), "\n"), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return nil, invalid("manifest is empty")
	}
	entries := make(map[string]string, len(lines))
	for _, line := range lines {
		parts := strings.SplitN(line, "  ", 2)
		if len(parts) != 2 || len(parts[0]) != 64 || !isLowerHex(parts[0]) || parts[1] == "" || path.Base(parts[1]) != parts[1] {
			return nil, invalid("invalid manifest line")
		}
		if _, exists := entries[parts[1]]; exists {
			return nil, invalid("duplicate manifest path %q", parts[1])
		}
		entries[parts[1]] = "sha256:" + parts[0]
	}
	return entries, nil
}

func validateManifestEntries(entries map[string][]byte, root string, manifest map[string]string) error {
	if len(entries) != len(manifest)+2 {
		return invalid("ZIP entry set does not exactly match manifest")
	}
	for fileName, expectedDigest := range manifest {
		content, exists := entries[root+fileName]
		if !exists || sha256Digest(content) != expectedDigest {
			return invalid("manifest digest mismatch for %q", fileName)
		}
	}
	if _, ok := manifest[packageJSONName]; !ok {
		return invalid("manifest does not bind package JSON")
	}
	return nil
}

func validateExpectedDigestAndBytes(actualBytes int64, actualDigest string, expectedBytes int64, expectedDigest, label string) error {
	if expectedBytes > 0 && actualBytes != expectedBytes {
		return invalid("%s byte count mismatch", label)
	}
	if expectedDigest != "" && actualDigest != expectedDigest {
		return invalid("%s digest mismatch", label)
	}
	return nil
}

type packageDocument struct {
	PackageVersion       string           `json:"packageVersion"`
	Status               string           `json:"status"`
	CandidateOnly        bool             `json:"candidateOnly"`
	Archive              archiveIdentity  `json:"archive"`
	Forms                []FormCandidate  `json:"forms"`
	SourceCoverage       []SourceProposal `json:"sourceCoverage"`
	ExtractionPolicy     json.RawMessage  `json:"extractionPolicy"`
	OfficialSourcePolicy json.RawMessage  `json:"officialSourcePolicy"`
	RiskPolicy           json.RawMessage  `json:"riskPolicy"`
	FutureChangePolicy   json.RawMessage  `json:"futureChangePolicy"`
	Totals               packageTotals    `json:"totals"`
}

type archiveIdentity struct {
	SHA256            string `json:"sha256"`
	Bytes             int64  `json:"bytes"`
	PDFEntryCount     int    `json:"pdfEntryCount"`
	FormCount         int    `json:"formCount"`
	RawBytesPersisted bool   `json:"rawBytesPersisted"`
	RegisterSHA256    string `json:"registerSha256"`
	PathHint          string `json:"pathHint"`
}

type packageTotals struct {
	Forms      int `json:"forms"`
	Questions  int `json:"questions"`
	SourceRefs int `json:"sourceRefs"`
}

func decodePackage(content []byte) (packageDocument, error) {
	var decoded packageDocument
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return packageDocument{}, err
	}
	if decoder.More() {
		return packageDocument{}, errors.New("trailing JSON")
	}
	return decoded, nil
}

func validatePackage(decoded packageDocument, expected ExpectedPackage) (AcceptedPackage, error) {
	if decoded.PackageVersion != expected.PackageVersion || decoded.Status != expected.PackageStatus || !decoded.CandidateOnly {
		return AcceptedPackage{}, invalid("package identity or candidate-only state mismatch")
	}
	if !digestPattern.MatchString(decoded.Archive.SHA256) || !digestPattern.MatchString(decoded.Archive.RegisterSHA256) || decoded.Archive.RawBytesPersisted {
		return AcceptedPackage{}, invalid("archive provenance is invalid or raw bytes are persisted")
	}
	if (expected.SourceArchiveSHA256 != "" && decoded.Archive.SHA256 != expected.SourceArchiveSHA256) ||
		(expected.SourceArchiveBytes > 0 && decoded.Archive.Bytes != expected.SourceArchiveBytes) ||
		(expected.SourceArchivePDFs > 0 && decoded.Archive.PDFEntryCount != expected.SourceArchivePDFs) ||
		(expected.RegisterSHA256 != "" && decoded.Archive.RegisterSHA256 != expected.RegisterSHA256) {
		return AcceptedPackage{}, invalid("archive provenance identity mismatch")
	}
	if len(decoded.Forms) != expected.ExpectedCounts.Forms || decoded.Totals.Forms != len(decoded.Forms) || decoded.Archive.FormCount != len(decoded.Forms) {
		return AcceptedPackage{}, invalid("form count mismatch")
	}
	if len(decoded.SourceCoverage) != expected.ExpectedCounts.UniqueSourceReferences || decoded.Totals.SourceRefs != len(decoded.SourceCoverage) {
		return AcceptedPackage{}, invalid("source coverage count mismatch")
	}
	if !sameStringSequence(formCodes(decoded.Forms), expected.FormCodes) {
		return AcceptedPackage{}, invalid("form code set mismatch")
	}
	questions, proposalQuestionCount, unmappedCount, questionLinks, formLinks, blockers, zeroForms, questionRisks, formRisks, err := validateForms(decoded.Forms)
	if err != nil {
		return AcceptedPackage{}, err
	}
	counts := expected.ExpectedCounts
	if len(questions) != counts.Questions || proposalQuestionCount != counts.QuestionsWithProposals || unmappedCount != counts.UnmappedQuestions || questionLinks != counts.QuestionSourceProposalLinks || formLinks != counts.FormSourceProposalLinks || blockers != counts.ExpertRiskReviewBlockers {
		return AcceptedPackage{}, invalid("question, proposal, or risk-blocker count mismatch")
	}
	if decoded.Totals.Questions != len(questions) || !sameStringSet(zeroForms, expected.ZeroFormCodes) || countNonemptyForms(decoded.Forms) != counts.FormsWithCandidateBoundaries {
		return AcceptedPackage{}, invalid("question boundary form set mismatch")
	}
	if !sameCountMap(questionRisks, expected.RiskBands) || !sameCountMap(formRisks, expected.FormRiskBands) {
		return AcceptedPackage{}, invalid("proposed risk distribution mismatch")
	}
	if expected.StrictPolicies {
		if err := validatePolicies(decoded, expected.RiskBands); err != nil {
			return AcceptedPackage{}, err
		}
	}
	if err := validateSourceCoverage(decoded.SourceCoverage); err != nil {
		return AcceptedPackage{}, err
	}
	return AcceptedPackage{Forms: decoded.Forms, SourceCoverage: decoded.SourceCoverage}, nil
}

func validatePolicies(decoded packageDocument, riskBands map[string]int) error {
	if err := validatePolicyObject(decoded.ExtractionPolicy, map[string]string{
		"questionCountMeaning": "string", "formsWithQuestionCandidates": "number", "formsWithoutQuestionCandidates": "number", "parserState": "string", "form048Fallback": "string", "adminReviewRequiredBeforeImport": "bool", "noAutomaticImport": "bool",
	}); err != nil {
		return invalid("extraction policy: %v", err)
	}
	if err := validatePolicyObject(decoded.OfficialSourcePolicy, map[string]string{
		"localNCAACollectionManifest": "string", "localNamcatsPart139": "string", "namcarPart139": "string", "unresolvedBoundary": "string", "noAutomaticAuthority": "bool",
	}); err != nil {
		return invalid("official source policy: %v", err)
	}
	if err := validatePolicyObject(decoded.RiskPolicy, map[string]string{
		"bands": "array", "safetyCriticalIsAdvisory": "bool", "noAutomaticFindingSeverity": "bool", "humanReviewState": "string",
	}); err != nil {
		return invalid("risk policy: %v", err)
	}
	var bands []string
	var riskPolicy struct {
		Bands                      []string `json:"bands"`
		SafetyCriticalIsAdvisory   bool     `json:"safetyCriticalIsAdvisory"`
		NoAutomaticFindingSeverity bool     `json:"noAutomaticFindingSeverity"`
		HumanReviewState           string   `json:"humanReviewState"`
	}
	if err := json.Unmarshal(decoded.RiskPolicy, &riskPolicy); err != nil || !riskPolicy.SafetyCriticalIsAdvisory || !riskPolicy.NoAutomaticFindingSeverity || riskPolicy.HumanReviewState != "CANDIDATE_INTERPRETATION_REQUIRES_EXPERT_REVIEW" {
		return invalid("risk policy must remain non-authoritative")
	}
	bands = riskPolicy.Bands
	if !sameStringSet(bands, mapKeys(riskBands)) {
		return invalid("risk policy band set mismatch")
	}
	if err := validatePolicyObject(decoded.FutureChangePolicy, map[string]string{
		"sourceVersioning": "string", "changeAction": "string", "publication": "string",
	}); err != nil {
		return invalid("future change policy: %v", err)
	}
	return nil
}

func validatePolicyObject(raw json.RawMessage, expected map[string]string) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || len(object) != len(expected) {
		return errors.New("unexpected object shape")
	}
	for key, kind := range expected {
		value, exists := object[key]
		if !exists || !rawJSONHasType(value, kind) {
			return fmt.Errorf("invalid %s", key)
		}
	}
	return nil
}

func rawJSONHasType(raw json.RawMessage, kind string) bool {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return false
	}
	switch kind {
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		_, ok := value.(json.Number)
		return ok
	case "bool":
		_, ok := value.(bool)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	default:
		return false
	}
}

func mapKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func validateForms(forms []FormCandidate) ([]QuestionCandidate, int, int, int, int, int, []string, map[string]int, map[string]int, error) {
	seenForms, seenQuestions := map[string]struct{}{}, map[string]struct{}{}
	questions := make([]QuestionCandidate, 0)
	questionRisks, formRisks := map[string]int{}, map[string]int{}
	zeroForms, proposalQuestions := []string{}, 0
	unmapped, questionLinks, formLinks, blockers := 0, 0, 0, 0
	for _, form := range forms {
		if _, exists := seenForms[form.FormCode]; exists || !digestPattern.MatchString(form.FormSHA256) || !digestPattern.MatchString(form.ArchiveSHA256) || form.FormCode == "" || form.QuestionCount != len(form.Questions) || form.CandidateState != "NOT_IMPORTED" || form.SourceMappingState != "SOURCE_MAPPING_REQUIRED" || form.PublicationState != "NOT_AUTHORIZED" || !validRisk(form.ProposedRisk) {
			return nil, 0, 0, 0, 0, 0, nil, nil, nil, invalid("invalid form candidate")
		}
		seenForms[form.FormCode] = struct{}{}
		formRisks[form.ProposedRisk.Band]++
		formLinks += len(form.FormSourceProposals)
		if err := validateSourceCoverage(form.FormSourceProposals); err != nil {
			return nil, 0, 0, 0, 0, 0, nil, nil, nil, err
		}
		if len(form.Questions) == 0 {
			if form.QuestionExtractionState != "NO_PROTOCOL_QUESTION_BOUNDARY_DETECTED" || form.QuestionBoundaryWarning == nil {
				return nil, 0, 0, 0, 0, 0, nil, nil, nil, invalid("zero-boundary form is not explicit")
			}
			zeroForms = append(zeroForms, form.FormCode)
			continue
		}
		if form.QuestionExtractionState != "EXTRACTED_CANDIDATE_BOUNDARIES" {
			return nil, 0, 0, 0, 0, 0, nil, nil, nil, invalid("question-bearing form extraction state is invalid")
		}
		for index, question := range form.Questions {
			if _, exists := seenQuestions[question.ProposalID]; exists || question.ProposalID == "" || question.Ordinal != index+1 || question.Page < 1 || question.OriginalText == "" || question.SourceLocator == "" || sha256Digest([]byte(question.OriginalText)) != question.TextDigest || question.SourceMappingState != "SOURCE_MAPPING_REQUIRED" || question.SourceAuthorityState != "NOT_ATTESTED" || question.RiskClassificationState != "CANDIDATE_INTERPRETATION_REQUIRES_EXPERT_REVIEW" || question.DecisionState != "NOT_SUPPLIED" || (question.ExtractionState != "EXACT_SOURCE_BACKED" && question.ExtractionState != "EXTRACTED_CANDIDATE") || !validRisk(question.ProposedRisk) {
				return nil, 0, 0, 0, 0, 0, nil, nil, nil, invalid("invalid question candidate")
			}
			seenQuestions[question.ProposalID] = struct{}{}
			questions = append(questions, question)
			questionRisks[question.ProposedRisk.Band]++
			questionLinks += len(question.SourceProposals)
			if len(question.SourceProposals) > 0 {
				proposalQuestions++
			} else if len(question.SourceRefs) == 0 {
				unmapped++
			} else {
				return nil, 0, 0, 0, 0, 0, nil, nil, nil, invalid("question source reference lacks proposal")
			}
			if question.ProposedRisk.Band == "PROPOSED_REVIEW_REQUIRED" {
				blockers++
			}
			if err := validateSourceCoverage(question.SourceProposals); err != nil {
				return nil, 0, 0, 0, 0, 0, nil, nil, nil, err
			}
		}
	}
	return questions, proposalQuestions, unmapped, questionLinks, formLinks, blockers, zeroForms, questionRisks, formRisks, nil
}

func validateSourceCoverage(sources []SourceProposal) error {
	seen := map[string]struct{}{}
	for _, source := range sources {
		key := source.Ref
		if source.Ref == "" || source.SourceDocumentID == "" || source.SourceTitle == "" || !isUnresolvedProposalStatus(source.Status) || !isUnresolvedProposalAuthority(source.AuthorityState) {
			return invalid("source proposal is not an unresolved non-authoritative hint")
		}
		if source.SourceSHA256 != nil && !digestPattern.MatchString(*source.SourceSHA256) {
			return invalid("source proposal digest is malformed")
		}
		if _, exists := seen[key]; exists {
			return invalid("duplicate source proposal")
		}
		seen[key] = struct{}{}
	}
	return nil
}

func isUnresolvedProposalAuthority(state string) bool {
	switch state {
	case "NOT_ATTESTED", "SOURCE_OWNER_ATTESTATION_REQUIRED", "SOURCE_BYTES_NOT_LOCALLY_HASHED_SOURCE_OWNER_ATTESTATION_REQUIRED":
		return true
	default:
		return false
	}
}

func isUnresolvedProposalStatus(status string) bool {
	return status == "PROPOSED_EXTERNAL_OFFICIAL_SOURCE" || status == "PROPOSED_LOCAL_SOURCE"
}

func validRisk(risk RiskProposal) bool {
	return strings.HasPrefix(risk.Band, "PROPOSED_") && risk.Domain != "" && risk.Rationale != ""
}

func formCodes(forms []FormCandidate) []string {
	output := make([]string, 0, len(forms))
	for _, form := range forms {
		output = append(output, form.FormCode)
	}
	return output
}

func countNonemptyForms(forms []FormCandidate) int {
	count := 0
	for _, form := range forms {
		if len(form.Questions) > 0 {
			count++
		}
	}
	return count
}

func sameStringSet(actual, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	a, b := append([]string(nil), actual...), append([]string(nil), expected...)
	sort.Strings(a)
	sort.Strings(b)
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}

func sameStringSequence(actual, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	for index := range actual {
		if actual[index] != expected[index] {
			return false
		}
	}
	return true
}

func sameCountMap(actual, expected map[string]int) bool {
	if expected == nil {
		return true
	}
	if len(actual) != len(expected) {
		return false
	}
	for key, count := range expected {
		if actual[key] != count {
			return false
		}
	}
	return true
}

func sha256Digest(content []byte) string {
	digest := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func isLowerHex(value string) bool {
	for _, character := range value {
		if !(character >= '0' && character <= '9' || character >= 'a' && character <= 'f') {
			return false
		}
	}
	return true
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrInvalidPackage}, args...)...)
}
