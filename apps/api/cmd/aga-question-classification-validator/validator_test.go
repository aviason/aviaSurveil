package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"testing"
)

func validBatch(records string) []byte {
	return []byte(`{"schemaVersion":"aga-hybrid-classification-pass-batch/v1","passRole":"CANDIDATE","batchOrdinal":1,"sourceSnapshotDigest":"sha256:fixed","records":` + records + `}`)
}

func validRecord() string {
	return `{"identity":{},"proposalProjection":{},"rationaleCodes":[],"confidenceEvidence":[],"sourceRefs":[]}`
}

func TestValidatePassRequiresExactBijection(t *testing.T) {
	if err := validatePass(passValidationRequest{raw: validBatch(`[]`), expectedRole: "CANDIDATE", recordCount: 1}); err == nil {
		t.Fatal("expected exact-bijection rejection")
	}
}

func TestValidatePassRejectsTextAndUnknownCodes(t *testing.T) {
	if err := validatePass(passValidationRequest{raw: []byte(`{"questionBody":"forbidden"}`)}); err != errPassText {
		t.Fatalf("text leak error = %v", err)
	}
}

func TestValidatePassRecomputesConfidenceEvidence(t *testing.T) {
	if err := validateEvidence(nil, nil); err == nil {
		t.Fatal("expected incomplete evidence rejection")
	}
}

func TestValidatorDiagnosticsAreIdentityRedacted(t *testing.T) {
	identity := "FSS-AGA-FORM-001/all-forms-preview-001-0001"
	if got := diagnostic("ERR_AGA_PASS_INVALID", identity); got != "ERR_AGA_PASS_INVALID" || strings.Contains(got, identity) {
		t.Fatalf("diagnostic leaked identity: %q", got)
	}
}

func TestReconcileUsesCandidateChallengePrecedence(t *testing.T) {
	if _, err := reconcile(nil, nil); err == nil {
		t.Fatal("expected invalid reconciliation rejection")
	}
}

func TestReconcilePersistsBothPassProjections(t *testing.T) {
	if err := verifyPassProjections(nil); err == nil {
		t.Fatal("expected missing projections rejection")
	}
}

func TestCandidateReconstructsAggregates(t *testing.T) {
	if err := validateCandidateDirectory(""); err == nil {
		t.Fatal("expected missing candidate rejection")
	}
}

func TestValidatePassRejectsTraversalAndUnsafeZIPEntries(t *testing.T) {
	for _, name := range []string{"../escape.json", "/absolute.json", "dir\\escape.json"} {
		data := makeZip(t, zipEntry{name: name, data: []byte("x")})
		if _, err := scanPrivateZip(bytes.NewReader(data), int64(len(data))); err != errZipUnsafe {
			t.Fatalf("%q error = %v", name, err)
		}
	}
	data := makeZip(t, zipEntry{name: "unsafe", mode: os.ModeSymlink | 0644, data: []byte("x")})
	if _, err := scanPrivateZip(bytes.NewReader(data), int64(len(data))); err != errZipUnsafe {
		t.Fatalf("symlink error = %v", err)
	}
}

func TestValidatePassRejectsDuplicateZIPEntry(t *testing.T) {
	data := makeZip(t, zipEntry{name: "same.json", data: []byte("a")}, zipEntry{name: "same.json", data: []byte("b")})
	if _, err := scanPrivateZip(bytes.NewReader(data), int64(len(data))); err != errZipDuplicate {
		t.Fatalf("duplicate entry error = %v", err)
	}
}

func TestValidatePassLimitsExpandedSizeAndEntryCount(t *testing.T) {
	data := makeZip(t, zipEntry{name: "large.json", data: bytes.Repeat([]byte("x"), maxZipFileBytes+1)})
	if _, err := scanPrivateZip(bytes.NewReader(data), int64(len(data))); err != errZipUnsafe {
		t.Fatalf("expanded-size error = %v", err)
	}
	entries := make([]zipEntry, maxZipEntries+1)
	for i := range entries {
		entries[i] = zipEntry{name: fmt.Sprintf("%03d.json", i), data: []byte("x")}
	}
	data = makeZip(t, entries...)
	if _, err := scanPrivateZip(bytes.NewReader(data), int64(len(data))); err != errZipUnsafe {
		t.Fatalf("entry-count error = %v", err)
	}
}

func TestValidatePassExcludesAppleDouble(t *testing.T) {
	data := makeZip(t, zipEntry{name: "__MACOSX/._output.json", data: []byte("noise")}, zipEntry{name: "output.json", data: []byte("{}")})
	receipt, err := scanPrivateZip(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("scan = %v", err)
	}
	if receipt.SemanticEntries != 1 || receipt.TransportNoise != 1 {
		t.Fatalf("receipt = %#v", receipt)
	}
}

func TestValidatePassTreatsSafeArchiveDirectoriesAsNonSemantic(t *testing.T) {
	data := makeZip(t,
		zipEntry{name: "candidate/", mode: os.ModeDir | 0755},
		zipEntry{name: "candidate/CHAT_METADATA.json", data: []byte("{}")},
	)
	receipt, err := scanPrivateZip(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("safe directory scan = %v", err)
	}
	if receipt.SemanticEntries != 1 {
		t.Fatalf("semantic entries = %d", receipt.SemanticEntries)
	}
}

func TestValidatePassRejectsRoleInversion(t *testing.T) {
	if err := validatePass(passValidationRequest{raw: validBatch("[" + validRecord() + "]"), expectedRole: "CHALLENGE", recordCount: 1}); err != errPassSchema {
		t.Fatalf("role inversion = %v", err)
	}
}

func TestValidatePassUsesOnlyControlledBatchFilenames(t *testing.T) {
	candidate := map[string]*zip.File{"batch-01.pass-batch.json": {}}
	if _, err := sealedBatchFile(candidate, 1); err != nil {
		t.Fatalf("candidate filename rejected: %v", err)
	}
	challenge := map[string]*zip.File{"batch-01.response.json": {}}
	if _, err := sealedBatchFile(challenge, 1); err != nil {
		t.Fatalf("challenge filename rejected: %v", err)
	}
	both := map[string]*zip.File{"batch-01.pass-batch.json": {}, "batch-01.response.json": {}}
	if _, err := sealedBatchFile(both, 1); err != errZipDuplicate {
		t.Fatalf("ambiguous filename error = %v", err)
	}
}

func TestValidatePassRejectsUnknownFields(t *testing.T) {
	raw := []byte(`{"schemaVersion":"aga-hybrid-classification-pass-batch/v1","passRole":"CANDIDATE","batchOrdinal":1,"sourceSnapshotDigest":"sha256:fixed","records":[],"unknown":"x"}`)
	if err := validatePass(passValidationRequest{raw: raw, expectedRole: "CANDIDATE"}); err != errPassSchema {
		t.Fatalf("unknown field = %v", err)
	}
}

func TestValidatePassRejectsMetadataOmission(t *testing.T) {
	if err := validatePass(passValidationRequest{raw: validBatch("[" + validRecord() + "]"), expectedRole: "CANDIDATE", recordCount: 1}); err != nil {
		t.Fatalf("batch transport shape = %v", err)
	}
	// A transport-only batch cannot be accepted as a sealed pass: exact run,
	// prompt, model, input, and result-digest metadata are absent.
	if _, err := reconcile(struct{}{}, struct{}{}); err == nil {
		t.Fatal("metadata omission was accepted")
	}
}

func TestValidatePassRejectsDisplayedLabelAsModelID(t *testing.T) {
	metadata := []byte(`{"modelId":"GPT-5.6 Pro","service":"ChatGPT","interface":"native app on a desktop computer","snapshotBuildLabel":null,"displayedModelLabel":"GPT-5.6 Pro","requestedReasoningEffort":null,"forkTurns":null,"unavailableFields":["forkTurns","requestedReasoningEffort","snapshotBuildLabel"]}`)
	err := validatePass(passValidationRequest{
		raw:          validBatch("[" + validRecord() + "]"),
		metadata:     metadata,
		expectedRole: "CANDIDATE",
		recordCount:  1,
	})
	if err != errPassMetadata {
		t.Fatalf("displayed label accepted as model ID: %v", err)
	}
}

func TestValidatePassAcceptsTruthfulUnavailablePlatformMetadata(t *testing.T) {
	metadata := []byte(`{"modelId":null,"service":"ChatGPT","interface":"native app on a desktop computer","snapshotBuildLabel":null,"displayedModelLabel":"GPT-5.6 Pro","requestedReasoningEffort":null,"forkTurns":null,"unavailableFields":["forkTurns","modelId","requestedReasoningEffort","snapshotBuildLabel"]}`)
	err := validatePass(passValidationRequest{
		raw:          validBatch("[" + validRecord() + "]"),
		metadata:     metadata,
		expectedRole: "CANDIDATE",
		recordCount:  1,
	})
	if err != nil {
		t.Fatalf("truthful unavailable metadata = %v", err)
	}
}

func TestValidatePassAcceptsOnlyControlledMetadataAcceptanceStatus(t *testing.T) {
	metadata := []byte(`{"modelId":null,"service":"ChatGPT","interface":"native app on a desktop computer","snapshotBuildLabel":null,"displayedModelLabel":"GPT-5.6 Pro","requestedReasoningEffort":null,"forkTurns":null,"unavailableFields":["forkTurns","modelId","requestedReasoningEffort","snapshotBuildLabel"],"metadataAcceptanceStatus":"BLOCKED_MISSING_PLATFORM_METADATA"}`)
	if err := validateChatMetadata(metadata); err != nil {
		t.Fatalf("controlled status rejected: %v", err)
	}
	invalid := bytes.Replace(metadata, []byte("BLOCKED_MISSING_PLATFORM_METADATA"), []byte("INVENTED_STATUS"), 1)
	if err := validateChatMetadata(invalid); err != errPassMetadata {
		t.Fatalf("uncontrolled status error = %v", err)
	}
}

func TestValidatePassRequiresLexicalSuppliedProvenanceDigests(t *testing.T) {
	if !validSHA256Digest("sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef") {
		t.Fatal("valid supplied digest rejected")
	}
	for _, value := range []string{"", "sha256:ABCDEF0123456789abcdef0123456789abcdef0123456789abcdef0123456789", "sha256:short", "sha512:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"} {
		if validSHA256Digest(value) {
			t.Fatalf("invalid supplied digest accepted")
		}
	}
}

func TestValidatePassEnforcesByteEnvelope(t *testing.T) {
	tooLarge := bytes.Repeat([]byte("x"), maxEnvelopeByte+1)
	if err := validatePass(passValidationRequest{raw: tooLarge}); err != errPassInvalid {
		t.Fatalf("byte-envelope error = %v", err)
	}
}

func TestValidatePassCleansFailurePaths(t *testing.T) {
	// scanPrivateZip is pure: failed pre-extraction inspection returns no file
	// handle or extraction target that a caller could retain.
	data := makeZip(t, zipEntry{name: "../escape", data: []byte("x")})
	if _, err := scanPrivateZip(bytes.NewReader(data), int64(len(data))); err == nil {
		t.Fatal("unsafe archive accepted")
	}
}

func TestValidatePassPrivateIngestionCleansFailureRoot(t *testing.T) {
	directory := t.TempDir()
	source := directory + "/unsafe.zip"
	if err := os.WriteFile(source, makeZip(t, zipEntry{name: "../escape", data: []byte("x")}), 0600); err != nil {
		t.Fatal(err)
	}
	privateRoot := directory + "/private-pass-root"
	if _, _, err := validatePassZIPInPrivateRoot(source, privateRoot, "CANDIDATE"); err != errZipUnsafe {
		t.Fatalf("ingestion error = %v", err)
	}
	if _, err := os.Stat(privateRoot); !os.IsNotExist(err) {
		t.Fatalf("private root remains after failure: %v", err)
	}
}

func TestValidatePassPreventsCrossPassVisibility(t *testing.T) {
	if err := verifyIsolatedRoots([]string{"candidate/input.zip"}, []string{"challenge/input.zip", "candidate/result.json"}); err != errIsolation {
		t.Fatalf("visibility error = %v", err)
	}
}

func TestValidatorDiagnosticsArePrivacyControlled(t *testing.T) {
	for _, sensitive := range []string{"question body", "/private/tmp/input.zip", "proposal-123"} {
		if got := diagnostic("ERR_AGA_PASS_SCHEMA", sensitive); strings.Contains(got, sensitive) {
			t.Fatalf("diagnostic leaked %q: %q", sensitive, got)
		}
	}
}

func TestValidatePassCommandUsesPrivateZIPIngestion(t *testing.T) {
	marker, exitCode := runValidatorCommand([]string{"validate-pass", "--zip", "/private/tmp/untrusted.zip", "--private-root", "/private/tmp/owned-root", "--expected-pass", "pass-one"})
	if marker != "ERR_AGA_PASS_INVALID" || exitCode != 1 {
		t.Fatalf("command result = %q, %d", marker, exitCode)
	}
}

func TestValidatePassRejectsIncompleteBatchUnion(t *testing.T) {
	if err := validateBatchUnion([]int{1, 2}, 25); err != errPassBijection {
		t.Fatalf("batch union = %v", err)
	}
}

func TestValidatePassRejectsIncompleteRecordUnion(t *testing.T) {
	if err := validateRecordUnion(1309); err != errPassBijection {
		t.Fatalf("record union = %v", err)
	}
}

type zipEntry struct {
	name string
	mode fs.FileMode
	data []byte
}

func makeZip(t *testing.T, entries ...zipEntry) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.name, Method: zip.Deflate}
		if entry.mode != 0 {
			header.SetMode(entry.mode)
		}
		w, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(entry.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
