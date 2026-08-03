package agacandidatedemo_test

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"hash/crc32"
	"os"
	"path/filepath"
	"testing"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agacandidatedemo"
)

func TestPackageReaderRejectsRelativePathBeforeOpeningArchive(t *testing.T) {
	reader := agacandidatedemo.NewPackageReader()
	if _, err := reader.ReadAndValidate(context.Background(), "synthetic-valid-package.zip", syntheticExpected(t)); err == nil {
		t.Fatal("relative package path was accepted")
	}
}

func TestPackageReaderAcceptsSyntheticPackageAndRejectsExtraEntry(t *testing.T) {
	packagePath, expected := writeSyntheticPackage(t, "")
	reader := agacandidatedemo.NewPackageReader()
	accepted, err := reader.ReadAndValidate(context.Background(), packagePath, expected)
	if err != nil {
		t.Fatalf("read synthetic package: %v", err)
	}
	if len(accepted.Forms) != 1 || len(accepted.Forms[0].Questions) != 1 {
		t.Fatalf("accepted package shape = %#v", accepted)
	}

	extraPath, extraExpected := writeSyntheticPackage(t, "extra-entry")
	if _, err := reader.ReadAndValidate(context.Background(), extraPath, extraExpected); err == nil {
		t.Fatal("archive with an unmanifested entry was accepted")
	}
}

func TestPackageReaderRejectsSymlinkAndInvalidJSONShapes(t *testing.T) {
	reader := agacandidatedemo.NewPackageReader()
	packagePath, expected := writeSyntheticPackage(t, "")
	symlinkPath := filepath.Join(t.TempDir(), "package-link.zip")
	if err := os.Symlink(packagePath, symlinkPath); err != nil {
		t.Fatalf("create package symlink: %v", err)
	}
	if _, err := reader.ReadAndValidate(context.Background(), symlinkPath, expected); err == nil {
		t.Fatal("symlinked package was accepted")
	}

	for _, mutation := range []string{"duplicate-json-key", "unknown-json-field"} {
		t.Run(mutation, func(t *testing.T) {
			path, expected := writeSyntheticPackage(t, mutation)
			if _, err := reader.ReadAndValidate(context.Background(), path, expected); err == nil {
				t.Fatalf("%s package was accepted", mutation)
			}
		})
	}
}

func TestPackageReaderRejectsDigestAndCandidateStateDrift(t *testing.T) {
	reader := agacandidatedemo.NewPackageReader()
	path, expected := writeSyntheticPackage(t, "")
	expected.ZipSHA256 = "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	if _, err := reader.ReadAndValidate(context.Background(), path, expected); err == nil {
		t.Fatal("wrong expected ZIP digest was accepted")
	}
	_, expected = writeSyntheticPackage(t, "")
	expected.ZipBytes++
	if _, err := reader.ReadAndValidate(context.Background(), path, expected); err == nil {
		t.Fatal("wrong expected ZIP byte count was accepted")
	}

	for _, mutation := range []string{"wrong-text-digest", "invented-zero-boundary-question", "authoritative-question-source-state", "manifest-mismatch", "trailing-json", "invalid-form-digest", "supplied-decision"} {
		t.Run(mutation, func(t *testing.T) {
			path, expected := writeSyntheticPackage(t, mutation)
			if _, err := reader.ReadAndValidate(context.Background(), path, expected); err == nil {
				t.Fatalf("%s package was accepted", mutation)
			}
		})
	}
}

func TestPackageReaderRejectsForbiddenZIPEntryShapes(t *testing.T) {
	reader := agacandidatedemo.NewPackageReader()
	for _, mutation := range []string{"duplicate-entry", "unsafe-entry", "pdf-entry", "nested-archive-entry", "oversized-entry", "encrypted-entry", "data-descriptor-entry"} {
		t.Run(mutation, func(t *testing.T) {
			path, expected := writeSyntheticPackage(t, mutation)
			if _, err := reader.ReadAndValidate(context.Background(), path, expected); err == nil {
				t.Fatalf("%s package was accepted", mutation)
			}
		})
	}

	directoryPath := t.TempDir()
	if _, err := reader.ReadAndValidate(context.Background(), directoryPath, syntheticExpected(t)); err == nil {
		t.Fatal("directory was accepted as a package")
	}
}

func TestSyntheticFixtureIsReadable(t *testing.T) {
	fixturePath := filepath.Join("testdata", "synthetic-valid-package.zip")
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read synthetic fixture: %v", err)
	}
	digest := sha256.Sum256(content)
	expected := syntheticExpected(t)
	expected.ZipBytes = int64(len(content))
	expected.ZipSHA256 = "sha256:" + hex.EncodeToString(digest[:])
	if _, err := agacandidatedemo.NewPackageReader().ReadAndValidate(context.Background(), absolutePath(t, fixturePath), expected); err != nil {
		t.Fatalf("read synthetic fixture: %v", err)
	}
}

func TestWriteSyntheticFixture(t *testing.T) {
	if os.Getenv("AVIA_WRITE_SYNTHETIC_AGA_FIXTURE") != "1" {
		t.Skip("set AVIA_WRITE_SYNTHETIC_AGA_FIXTURE=1 to generate the synthetic committed fixture")
	}
	sourcePath, _ := writeSyntheticPackage(t, "")
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read generated fixture source: %v", err)
	}
	fixtureDirectory := filepath.Join("testdata")
	if err := os.MkdirAll(fixtureDirectory, 0o755); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fixtureDirectory, "synthetic-valid-package.zip"), content, 0o644); err != nil {
		t.Fatalf("write synthetic fixture: %v", err)
	}
}

func TestAcceptedPackage(t *testing.T) {
	packagePath := os.Getenv("AVIA_AGA_DEMO_PACKAGE_FILE")
	if packagePath == "" {
		t.Skip("AVIA_AGA_DEMO_PACKAGE_FILE is required for the explicit accepted-package check")
	}
	accepted, err := agacandidatedemo.NewPackageReader().ReadAndValidate(
		context.Background(), packagePath, agacandidatedemo.ExactAcceptedPackage(),
	)
	if err != nil {
		t.Fatalf("validate accepted package: %v", err)
	}
	questions := 0
	for _, form := range accepted.Forms {
		questions += len(form.Questions)
	}
	if len(accepted.Forms) != 52 || questions != 1310 || len(accepted.SourceCoverage) != 174 {
		t.Fatalf("accepted package counts = forms:%d questions:%d sources:%d", len(accepted.Forms), questions, len(accepted.SourceCoverage))
	}
	t.Logf("accepted package verified: forms=%d questions=%d sources=%d zip=%s json=%s", len(accepted.Forms), questions, len(accepted.SourceCoverage), accepted.Identity.ZipSHA256, accepted.Identity.JSONSHA256)
}

func syntheticExpected(t *testing.T) agacandidatedemo.ExpectedPackage {
	t.Helper()
	return agacandidatedemo.ExpectedPackage{
		PackageVersion: "SYNTHETIC_AGA_DEMO_V1",
		PackageStatus:  "PENDING_ADMIN_AND_SOURCE_OWNER_REVIEW",
		ExpectedCounts: agacandidatedemo.ExpectedCounts{
			Forms: 1, FormsWithCandidateBoundaries: 1, Questions: 1,
			QuestionsWithProposals: 1, UnmappedQuestions: 0,
			QuestionSourceProposalLinks: 1, FormSourceProposalLinks: 1,
			UniqueSourceReferences: 1, ExpertRiskReviewBlockers: 1,
		},
		FormCodes: []string{"FSS-AGA-FORM-SYNTHETIC"},
	}
}

func writeSyntheticPackage(t *testing.T, mutation string) (string, agacandidatedemo.ExpectedPackage) {
	t.Helper()
	content := []byte(`{"packageVersion":"SYNTHETIC_AGA_DEMO_V1","status":"PENDING_ADMIN_AND_SOURCE_OWNER_REVIEW","candidateOnly":true,"archive":{"sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","bytes":1,"pdfEntryCount":0,"formCount":1,"rawBytesPersisted":false,"registerSha256":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","pathHint":"synthetic"},"forms":[{"formCode":"FSS-AGA-FORM-SYNTHETIC","formSha256":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","archiveSha256":"sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd","archivePath":"synthetic.pdf","documentTitle":"Synthetic form","formKind":"SYNTHETIC","pageCount":1,"registerTitleCandidate":"Synthetic form","questionCount":1,"questionExtractionState":"EXTRACTED_CANDIDATE_BOUNDARIES","questionBoundaryWarning":null,"candidateState":"NOT_IMPORTED","sourceMappingState":"SOURCE_MAPPING_REQUIRED","publicationState":"NOT_AUTHORIZED","formSourceRefs":["SYNTHETIC-REF"],"formSourceProposals":[{"ref":"SYNTHETIC-REF","sourceDocumentId":"SYNTHETIC-SOURCE","sourceTitle":"Synthetic source","sourceUrl":null,"sourceSha256":null,"sourcePage":null,"clauseLocator":null,"status":"PROPOSED_EXTERNAL_OFFICIAL_SOURCE","authorityState":"NOT_ATTESTED"}],"proposedRisk":{"band":"PROPOSED_REVIEW_REQUIRED","domain":"Synthetic","rationale":"Synthetic only","safetyCritical":false},"questions":[{"proposalId":"SYNTHETIC-Q-001","ordinal":1,"protocolCode":null,"page":1,"sourceLocator":"Synthetic locator","originalText":"Synthetic question?","textDigest":"sha256:e3511e9a60026a35d90d2f659db7fc87502f37f408b0633d1b3238d2debf3556","sourceRefs":["SYNTHETIC-REF"],"sourceProposals":[{"ref":"SYNTHETIC-REF","sourceDocumentId":"SYNTHETIC-SOURCE","sourceTitle":"Synthetic source","sourceUrl":null,"sourceSha256":null,"sourcePage":null,"clauseLocator":null,"status":"PROPOSED_EXTERNAL_OFFICIAL_SOURCE","authorityState":"NOT_ATTESTED"}],"sourceMappingState":"SOURCE_MAPPING_REQUIRED","sourceAuthorityState":"NOT_ATTESTED","extractionState":"EXTRACTED_CANDIDATE","riskClassificationState":"CANDIDATE_INTERPRETATION_REQUIRES_EXPERT_REVIEW","decisionState":"NOT_SUPPLIED","proposedRisk":{"band":"PROPOSED_REVIEW_REQUIRED","domain":"Synthetic","rationale":"Synthetic only","safetyCritical":false}}]}],"sourceCoverage":[{"ref":"SYNTHETIC-REF","sourceDocumentId":"SYNTHETIC-SOURCE","sourceTitle":"Synthetic source","sourceUrl":null,"sourceSha256":null,"sourcePage":null,"clauseLocator":null,"status":"PROPOSED_EXTERNAL_OFFICIAL_SOURCE","authorityState":"NOT_ATTESTED"}],"extractionPolicy":{},"officialSourcePolicy":{"noAutomaticAuthority":true},"riskPolicy":{"safetyCriticalIsAdvisory":true,"noAutomaticFindingSeverity":true},"futureChangePolicy":{},"totals":{"forms":1,"questions":1,"sourceRefs":1}}`)

	switch mutation {
	case "duplicate-json-key":
		content = bytes.Replace(content, []byte(`"packageVersion":"SYNTHETIC_AGA_DEMO_V1"`), []byte(`"packageVersion":"SYNTHETIC_AGA_DEMO_V1","packageVersion":"SYNTHETIC_AGA_DEMO_V1"`), 1)
	case "unknown-json-field":
		content = bytes.Replace(content, []byte(`"totals":{`), []byte(`"unknownField":true,"totals":{`), 1)
	case "wrong-text-digest":
		content = bytes.Replace(content, []byte(`"textDigest":"sha256:e3511e9a60026a35d90d2f659db7fc87502f37f408b0633d1b3238d2debf3556"`), []byte(`"textDigest":"sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"`), 1)
	case "invented-zero-boundary-question":
		content = bytes.Replace(content, []byte(`"questionCount":1`), []byte(`"questionCount":0`), 1)
	case "authoritative-question-source-state":
		content = bytes.Replace(content, []byte(`"sourceAuthorityState":"NOT_ATTESTED"`), []byte(`"sourceAuthorityState":"ATTESTED"`), 1)
	case "trailing-json":
		content = append(content, []byte(`{}`)...)
	case "invalid-form-digest":
		content = bytes.Replace(content, []byte(`"formSha256":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"`), []byte(`"formSha256":"not-a-digest"`), 1)
	case "supplied-decision":
		content = bytes.Replace(content, []byte(`"decisionState":"NOT_SUPPLIED"`), []byte(`"decisionState":"SUPPLIED"`), 1)
	}

	directory := t.TempDir()
	path := filepath.Join(directory, "synthetic-valid-package.zip")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create synthetic package: %v", err)
	}
	archive := zip.NewWriter(file)
	entries := map[string][]byte{
		"synthetic/AGA_ALL_FORMS_SOURCE_RISK_DRAFT.json": content,
		"synthetic/README.md":                            []byte("synthetic only\n"),
	}
	manifest := ""
	for _, name := range []string{"synthetic/AGA_ALL_FORMS_SOURCE_RISK_DRAFT.json", "synthetic/README.md"} {
		digest := sha256.Sum256(entries[name])
		manifest += hex.EncodeToString(digest[:]) + "  " + filepath.Base(name) + "\n"
	}
	entries["synthetic/MANIFEST.sha256"] = []byte(manifest)
	if mutation == "manifest-mismatch" {
		entries["synthetic/README.md"] = []byte("changed after manifest")
	}
	extraEntryName, extraEntryContent := "", []byte(nil)
	switch mutation {
	case "extra-entry":
		extraEntryName, extraEntryContent = "synthetic/unmanifested.txt", []byte("forbidden")
	case "unsafe-entry":
		extraEntryName, extraEntryContent = "synthetic/../unsafe.txt", []byte("forbidden")
	case "pdf-entry":
		extraEntryName, extraEntryContent = "synthetic/forbidden.pdf", []byte("%PDF-1.7 synthetic")
	case "nested-archive-entry":
		extraEntryName, extraEntryContent = "synthetic/nested.zip", []byte("PK\x03\x04synthetic")
	}
	writeRawEntry(t, archive, "synthetic/", nil)
	for _, name := range []string{"synthetic/AGA_ALL_FORMS_SOURCE_RISK_DRAFT.json", "synthetic/README.md", "synthetic/MANIFEST.sha256"} {
		writeRawEntry(t, archive, name, entries[name])
	}
	if extraEntryName != "" {
		writeRawEntry(t, archive, extraEntryName, extraEntryContent)
	}
	if mutation == "duplicate-entry" {
		writeRawEntry(t, archive, "synthetic/README.md", entries["synthetic/README.md"])
	}
	if mutation == "oversized-entry" {
		writeRawDeflatedEntry(t, archive, "synthetic/expanded.txt", bytes.Repeat([]byte("A"), 9<<20))
	}
	if mutation == "encrypted-entry" {
		writeRawEntryWithFlags(t, archive, "synthetic/encrypted.txt", []byte("synthetic"), 0x1)
	}
	if mutation == "data-descriptor-entry" {
		writeRawEntryWithFlags(t, archive, "synthetic/data-descriptor.txt", []byte("synthetic"), 0x8)
	}
	if err := archive.Close(); err != nil {
		t.Fatalf("close archive: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close package: %v", err)
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read package: %v", err)
	}
	digest := sha256.Sum256(bytes)
	expected := syntheticExpected(t)
	expected.ZipBytes = int64(len(bytes))
	expected.ZipSHA256 = "sha256:" + hex.EncodeToString(digest[:])
	return path, expected
}

func writeRawEntry(t *testing.T, archive *zip.Writer, name string, content []byte) {
	t.Helper()
	header := &zip.FileHeader{Name: name, Method: zip.Store}
	header.CRC32 = crc32.ChecksumIEEE(content)
	header.UncompressedSize64 = uint64(len(content))
	header.CompressedSize64 = uint64(len(content))
	writer, err := archive.CreateRaw(header)
	if err != nil {
		t.Fatalf("create %s: %v", name, err)
	}
	if _, err := writer.Write(content); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func writeRawEntryWithFlags(t *testing.T, archive *zip.Writer, name string, content []byte, flags uint16) {
	t.Helper()
	header := &zip.FileHeader{Name: name, Method: zip.Store, Flags: flags}
	header.CRC32 = crc32.ChecksumIEEE(content)
	header.UncompressedSize64 = uint64(len(content))
	header.CompressedSize64 = uint64(len(content))
	writer, err := archive.CreateRaw(header)
	if err != nil {
		t.Fatalf("create %s: %v", name, err)
	}
	if _, err := writer.Write(content); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func writeRawDeflatedEntry(t *testing.T, archive *zip.Writer, name string, content []byte) {
	t.Helper()
	var compressed bytes.Buffer
	compressor, err := flate.NewWriter(&compressed, flate.BestCompression)
	if err != nil {
		t.Fatalf("create compressor: %v", err)
	}
	if _, err := compressor.Write(content); err != nil {
		t.Fatalf("compress %s: %v", name, err)
	}
	if err := compressor.Close(); err != nil {
		t.Fatalf("close compressor: %v", err)
	}
	header := &zip.FileHeader{Name: name, Method: zip.Deflate}
	header.CRC32 = crc32.ChecksumIEEE(content)
	header.UncompressedSize64 = uint64(len(content))
	header.CompressedSize64 = uint64(compressed.Len())
	writer, err := archive.CreateRaw(header)
	if err != nil {
		t.Fatalf("create deflated %s: %v", name, err)
	}
	if _, err := writer.Write(compressed.Bytes()); err != nil {
		t.Fatalf("write deflated %s: %v", name, err)
	}
}

func absolutePath(t *testing.T, relative string) string {
	t.Helper()
	path, err := filepath.Abs(relative)
	if err != nil {
		t.Fatalf("absolute path for %s: %v", relative, err)
	}
	return path
}
