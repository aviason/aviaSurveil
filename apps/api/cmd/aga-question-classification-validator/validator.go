package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/aviason/aviaSurveil/internal/agaapplicability"
)

const (
	maxZipEntries    = 128
	maxZipCompressed = 8 << 20
	maxZipExpanded   = 8 << 20
	maxZipFileBytes  = 2 << 20
	maxEnvelopeByte  = 98304
)

var (
	errPassInvalid      = errors.New("ERR_AGA_PASS_INVALID")
	errPassSchema       = errors.New("ERR_AGA_PASS_SCHEMA")
	errPassMetadata     = errors.New("ERR_AGA_PASS_METADATA")
	errPassProvenance   = errors.New("ERR_AGA_PASS_PROVENANCE")
	errPassBijection    = errors.New("ERR_AGA_PASS_BIJECTION")
	errPassText         = errors.New("ERR_AGA_PASS_TEXT")
	errPassEnvelope     = errors.New("ERR_AGA_PASS_ENVELOPE")
	errZipUnsafe        = errors.New("ERR_AGA_ZIP_UNSAFE")
	errZipDuplicate     = errors.New("ERR_AGA_ZIP_DUPLICATE")
	errIsolation        = errors.New("ERR_AGA_PASS_ISOLATION")
	errCandidateInvalid = errors.New("ERR_AGA_CANDIDATE_INVALID")
)

// passValidationRequest is intentionally small: it is the command boundary
// that accepts untrusted, text-free normalized input only.
type passValidationRequest struct {
	raw          []byte
	metadata     []byte
	expectedRole string
	batchCount   int
	recordCount  int
	maxBytes     int
}

type privateZipReceipt struct {
	SemanticEntries int
	TransportNoise  int
	ExpandedBytes   uint64
	Digest          string
}

type rawPassRecord struct {
	Identity  json.RawMessage `json:"identity"`
	Proposal  json.RawMessage `json:"proposalProjection"`
	Rationale json.RawMessage `json:"rationaleCodes"`
	Evidence  json.RawMessage `json:"confidenceEvidence"`
	Sources   json.RawMessage `json:"sourceRefs"`
}

type rawPassBatch struct {
	SchemaVersion        string          `json:"schemaVersion"`
	PassRole             string          `json:"passRole"`
	BatchOrdinal         int             `json:"batchOrdinal"`
	SourceSnapshotDigest string          `json:"sourceSnapshotDigest"`
	Records              []rawPassRecord `json:"records"`
}

type chatMetadata struct {
	ModelID                  *string  `json:"modelId"`
	Service                  *string  `json:"service"`
	Interface                *string  `json:"interface"`
	SnapshotBuildLabel       *string  `json:"snapshotBuildLabel"`
	DisplayedModelLabel      *string  `json:"displayedModelLabel"`
	RequestedReasoningEffort *string  `json:"requestedReasoningEffort"`
	ForkTurns                *string  `json:"forkTurns"`
	UnavailableFields        []string `json:"unavailableFields"`
	MetadataAcceptanceStatus *string  `json:"metadataAcceptanceStatus"`
}

type sealedPassBatchWire struct {
	SchemaVersion         string                    `json:"schemaVersion"`
	ClassificationRunID   string                    `json:"classificationRunId"`
	PassRole              agaapplicability.PassRole `json:"passRole"`
	PassRunID             string                    `json:"passRunId"`
	BatchOrdinal          int                       `json:"batchOrdinal"`
	PromptDigest          string                    `json:"promptDigest"`
	ModelDescriptorDigest string                    `json:"modelDescriptorDigest"`
	InputDigest           string                    `json:"inputDigest"`
	Records               []json.RawMessage         `json:"records"`
	BatchOutputDigest     string                    `json:"batchOutputDigest"`
}

type sealedPassRecordWire agaapplicability.PassProposalRecord

type passArchiveReceipt struct {
	ClassificationRunID       string   `json:"classificationRunId"`
	PassRole                  string   `json:"passRole"`
	PassRunID                 string   `json:"passRunId"`
	PromptDigest              string   `json:"promptDigest"`
	ModelDescriptorDigest     string   `json:"modelDescriptorDigest"`
	BatchManifestDigest       string   `json:"batchManifestDigest"`
	BatchCount                int      `json:"batchCount"`
	ItemCount                 int      `json:"itemCount"`
	OrderedInputDigests       []string `json:"orderedInputDigests"`
	PassInputSetDigest        string   `json:"passInputSetDigest"`
	OrderedBatchOutputDigests []string `json:"orderedBatchOutputDigests"`
	OrderedPassResultDigests  []string `json:"orderedPassResultDigests"`
	PassSealDigest            string   `json:"passSealDigest"`
}

type passArchiveValidation struct {
	BatchCount  int
	RecordCount int
	Metadata    chatMetadata
	Receipt     passArchiveReceipt
	Batches     []agaapplicability.PassBatchOutput
	Records     []agaapplicability.PassProposalRecord
}

func validatePass(request passValidationRequest) error {
	if len(request.raw) == 0 || len(request.raw) > maxEnvelopeByte || !utf8.Valid(request.raw) {
		return errPassInvalid
	}
	if containsForbiddenTextField(request.raw) {
		return errPassText
	}
	var batch rawPassBatch
	if err := decodeClosedPassBatch(request.raw, &batch); err != nil {
		return errPassSchema
	}
	if batch.SchemaVersion != "aga-hybrid-classification-pass-batch/v1" || batch.PassRole != request.expectedRole || batch.BatchOrdinal < 1 || batch.SourceSnapshotDigest == "" {
		return errPassSchema
	}
	if request.batchCount > 0 && batch.BatchOrdinal > request.batchCount {
		return errPassBijection
	}
	if request.recordCount > 0 && len(batch.Records) != request.recordCount {
		return errPassBijection
	}
	if len(request.metadata) > 0 {
		if err := validateChatMetadata(request.metadata); err != nil {
			return errPassMetadata
		}
	}
	for _, record := range batch.Records {
		if len(record.Identity) == 0 || len(record.Proposal) == 0 || len(record.Rationale) == 0 || len(record.Evidence) == 0 || len(record.Sources) == 0 {
			return errPassSchema
		}
	}
	return nil
}

func validateChatMetadata(data []byte) error {
	if len(data) == 0 || !utf8.Valid(data) {
		return errPassMetadata
	}
	var object map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&object); err != nil || object == nil {
		return errPassMetadata
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return errPassMetadata
	}
	allowed := map[string]bool{
		"modelId": true, "service": true, "interface": true,
		"snapshotBuildLabel": true, "displayedModelLabel": true,
		"requestedReasoningEffort": true, "forkTurns": true,
		"unavailableFields": true, "metadataAcceptanceStatus": true,
	}
	if len(object) != len(allowed)-1 && len(object) != len(allowed) {
		return errPassMetadata
	}
	for key := range object {
		if !allowed[key] {
			return errPassMetadata
		}
	}
	var metadata chatMetadata
	decoder = json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&metadata); err != nil {
		return errPassMetadata
	}
	if metadata.UnavailableFields == nil {
		return errPassMetadata
	}
	if metadata.MetadataAcceptanceStatus != nil && *metadata.MetadataAcceptanceStatus != "BLOCKED_MISSING_PLATFORM_METADATA" {
		return errPassMetadata
	}
	allowedUnavailable := map[string]bool{
		"modelId": true, "service": true, "interface": true,
		"snapshotBuildLabel": true, "displayedModelLabel": true,
		"requestedReasoningEffort": true, "forkTurns": true,
	}
	fields := map[string]*string{
		"modelId": metadata.ModelID, "service": metadata.Service,
		"interface": metadata.Interface, "snapshotBuildLabel": metadata.SnapshotBuildLabel,
		"displayedModelLabel":      metadata.DisplayedModelLabel,
		"requestedReasoningEffort": metadata.RequestedReasoningEffort,
		"forkTurns":                metadata.ForkTurns,
	}
	seen := make(map[string]struct{}, len(metadata.UnavailableFields))
	previous := ""
	for _, field := range metadata.UnavailableFields {
		if !allowedUnavailable[field] || field <= previous {
			return errPassMetadata
		}
		if _, exists := seen[field]; exists {
			return errPassMetadata
		}
		seen[field] = struct{}{}
		previous = field
	}
	for field, value := range fields {
		_, unavailable := seen[field]
		if (value == nil) != unavailable {
			return errPassMetadata
		}
		if value != nil && (!utf8.ValidString(*value) || len(*value) == 0 || len(*value) > 128) {
			return errPassMetadata
		}
	}
	if metadata.ModelID != nil && metadata.DisplayedModelLabel != nil && *metadata.ModelID == *metadata.DisplayedModelLabel {
		return errPassMetadata
	}
	return nil
}

func validateSealedPassArchive(zipPath, expectedRole string) (passArchiveValidation, error) {
	file, err := os.Open(zipPath)
	if err != nil {
		return passArchiveValidation{}, errPassInvalid
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return passArchiveValidation{}, errPassInvalid
	}
	if _, err := scanPrivateZip(file, info.Size()); err != nil {
		return passArchiveValidation{}, err
	}
	reader, err := zip.NewReader(file, info.Size())
	if err != nil {
		return passArchiveValidation{}, errZipUnsafe
	}

	files := make(map[string]*zip.File)
	root := ""
	for _, entry := range reader.File {
		if entry.FileInfo().IsDir() || isTransportNoise(entry.Name) {
			continue
		}
		parts := strings.Split(path.Clean(entry.Name), "/")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return passArchiveValidation{}, errPassSchema
		}
		if root == "" {
			root = parts[0]
		}
		if parts[0] != root {
			return passArchiveValidation{}, errIsolation
		}
		if _, exists := files[parts[1]]; exists {
			return passArchiveValidation{}, errZipDuplicate
		}
		files[parts[1]] = entry
	}
	metadataFile, receiptFile := files["CHAT_METADATA.json"], files["PASS_RUN_RECEIPT.json"]
	if root == "" || metadataFile == nil || receiptFile == nil || len(files) != 27 {
		return passArchiveValidation{}, errPassSchema
	}
	metadata, err := readZipFile(metadataFile)
	if err != nil || validateChatMetadata(metadata) != nil {
		return passArchiveValidation{}, errPassMetadata
	}
	var metadataValue chatMetadata
	if err := json.Unmarshal(metadata, &metadataValue); err != nil {
		return passArchiveValidation{}, errPassMetadata
	}
	receiptBytes, err := readZipFile(receiptFile)
	if err != nil {
		return passArchiveValidation{}, errPassSchema
	}
	var receipt passArchiveReceipt
	if err := decodeClosedReceipt(receiptBytes, &receipt); err != nil {
		return passArchiveValidation{}, errPassSchema
	}
	if receipt.PassRole != expectedRole || receipt.BatchCount != 25 || receipt.ItemCount != 1310 || len(receipt.OrderedInputDigests) != 25 || len(receipt.OrderedBatchOutputDigests) != 25 || len(receipt.OrderedPassResultDigests) != 1310 {
		return passArchiveValidation{}, errPassSchema
	}
	if !validSHA256Digest(receipt.PromptDigest) || !validSHA256Digest(receipt.ModelDescriptorDigest) {
		return passArchiveValidation{}, errPassProvenance
	}
	if receipt.BatchManifestDigest != agaapplicability.FrozenBatchManifestDigest {
		return passArchiveValidation{}, errPassProvenance
	}

	orderedInputs := make([]string, 25)
	orderedOutputs := make([]string, 25)
	orderedRecords := make([]string, 0, 1310)
	orderedIdentities := make([]agaapplicability.BaseIdentity, 0, 1310)
	validatedBatches := make([]agaapplicability.PassBatchOutput, 0, 25)
	validatedRecords := make([]agaapplicability.PassProposalRecord, 0, 1310)
	seenIdentities := make(map[string]struct{}, 1310)
	seenOrdinals := make([]int, 0, 25)
	classificationRunID, passRunID := "", ""
	for ordinal := 1; ordinal <= 25; ordinal++ {
		entry, err := sealedBatchFile(files, ordinal)
		if err != nil {
			return passArchiveValidation{}, err
		}
		data, err := readZipFile(entry)
		if err != nil {
			return passArchiveValidation{}, errPassSchema
		}
		if containsForbiddenTextField(data) {
			return passArchiveValidation{}, errPassText
		}
		var output agaapplicability.PassBatchOutput
		if err := decodeClosedSealedBatch(data, &output); err != nil {
			return passArchiveValidation{}, errPassSchema
		}
		if output.SchemaVersion != "aga-hybrid-classification-pass-batch/v1" || string(output.PassRole) != expectedRole || output.BatchOrdinal != ordinal || output.PromptDigest != receipt.PromptDigest || output.ModelDescriptorDigest != receipt.ModelDescriptorDigest || output.BatchOutputDigest != agaapplicability.ComputePassBatchOutputDigest(output) {
			return passArchiveValidation{}, errPassSchema
		}
		if classificationRunID == "" {
			classificationRunID, passRunID = output.ClassificationRunID, output.PassRunID
		}
		if output.ClassificationRunID != classificationRunID || output.PassRunID != passRunID || len(output.Records) == 0 {
			return passArchiveValidation{}, errPassBijection
		}
		seenOrdinals = append(seenOrdinals, output.BatchOrdinal)
		orderedInputs[ordinal-1], orderedOutputs[ordinal-1] = output.InputDigest, output.BatchOutputDigest
		validatedBatches = append(validatedBatches, output)
		for _, record := range output.Records {
			if string(record.PassRole) != expectedRole || record.ClassificationRunID != classificationRunID || record.PassRunID != passRunID || record.PromptDigest != receipt.PromptDigest || record.ModelDescriptorDigest != receipt.ModelDescriptorDigest || record.InputDigest != output.InputDigest {
				return passArchiveValidation{}, errPassSchema
			}
			identityKey := record.Identity.Key()
			if _, exists := seenIdentities[identityKey]; exists {
				return passArchiveValidation{}, errPassBijection
			}
			seenIdentities[identityKey] = struct{}{}
			orderedIdentities = append(orderedIdentities, record.Identity)
			orderedRecords = append(orderedRecords, record.PassResultDigest)
			validatedRecords = append(validatedRecords, record)
		}
	}
	if err := validateBatchUnion(seenOrdinals, 25); err != nil || validateRecordUnion(len(orderedRecords)) != nil {
		return passArchiveValidation{}, errPassBijection
	}
	if receipt.ClassificationRunID != classificationRunID || receipt.PassRunID != passRunID || !sameStrings(receipt.OrderedInputDigests, orderedInputs) || !sameStrings(receipt.OrderedBatchOutputDigests, orderedOutputs) || !sameStrings(receipt.OrderedPassResultDigests, orderedRecords) {
		return passArchiveValidation{}, errPassSchema
	}
	if identityDigest, err := agaapplicability.DigestValue("AGA-CLASSIFICATION-ORDERED-IDENTITIES-V1", orderedIdentities); err != nil || identityDigest != agaapplicability.FrozenOrderedIdentityDigest {
		return passArchiveValidation{}, errPassBijection
	}
	sealedReceipt := agaapplicability.PassSealReceipt{ClassificationRunID: receipt.ClassificationRunID, PassRole: agaapplicability.PassRole(receipt.PassRole), PassRunID: receipt.PassRunID, PromptDigest: receipt.PromptDigest, ModelDescriptorDigest: receipt.ModelDescriptorDigest, BatchManifestDigest: receipt.BatchManifestDigest, BatchCount: receipt.BatchCount, ItemCount: receipt.ItemCount, OrderedInputDigests: receipt.OrderedInputDigests, PassInputSetDigest: receipt.PassInputSetDigest, OrderedBatchOutputDigests: receipt.OrderedBatchOutputDigests, OrderedPassResultDigests: receipt.OrderedPassResultDigests, PassSealDigest: receipt.PassSealDigest}
	if receipt.PassInputSetDigest != agaapplicability.ComputePassInputSetDigest(sealedReceipt) || receipt.PassSealDigest != agaapplicability.ComputePassSeal(sealedReceipt) {
		return passArchiveValidation{}, errPassSchema
	}
	return passArchiveValidation{
		BatchCount: 25, RecordCount: len(orderedRecords), Metadata: metadataValue, Receipt: receipt,
		Batches: validatedBatches, Records: validatedRecords,
	}, nil
}

func sealedBatchFile(files map[string]*zip.File, ordinal int) (*zip.File, error) {
	passBatch := files[fmt.Sprintf("batch-%02d.pass-batch.json", ordinal)]
	response := files[fmt.Sprintf("batch-%02d.response.json", ordinal)]
	if passBatch != nil && response != nil {
		return nil, errZipDuplicate
	}
	if passBatch != nil {
		return passBatch, nil
	}
	if response != nil {
		return response, nil
	}
	return nil, errPassBijection
}

// validatePassZIPInPrivateRoot copies one immutable caller-provided archive to
// a fresh, role-specific private root. It never extracts the archive, and it
// removes the whole root before returning on both success and failure.
func validatePassZIPInPrivateRoot(sourcePath, privateRoot, expectedRole string) (validation passArchiveValidation, receipt privateZipReceipt, resultErr error) {
	if !filepath.IsAbs(sourcePath) || !filepath.IsAbs(privateRoot) || sourcePath == privateRoot {
		return validation, receipt, errIsolation
	}
	if info, err := os.Lstat(sourcePath); err != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > maxZipCompressed {
		return validation, receipt, errPassInvalid
	}
	if _, err := os.Lstat(privateRoot); !os.IsNotExist(err) {
		return validation, receipt, errIsolation
	}
	if err := os.Mkdir(privateRoot, 0700); err != nil {
		return validation, receipt, errIsolation
	}
	defer func() {
		if err := os.RemoveAll(privateRoot); err != nil && resultErr == nil {
			resultErr = errIsolation
			return
		}
		if _, err := os.Lstat(privateRoot); !os.IsNotExist(err) && resultErr == nil {
			resultErr = errIsolation
		}
	}()
	if err := os.Chmod(privateRoot, 0700); err != nil {
		return validation, receipt, errIsolation
	}
	passDirectory := filepath.Join(privateRoot, strings.ToLower(expectedRole))
	if err := os.Mkdir(passDirectory, 0700); err != nil {
		return validation, receipt, errIsolation
	}
	destination := filepath.Join(passDirectory, "input.zip")
	input, err := os.Open(sourcePath)
	if err != nil {
		return validation, receipt, errPassInvalid
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return validation, receipt, errIsolation
	}
	written, copyErr := io.Copy(output, io.LimitReader(input, maxZipCompressed+1))
	closeErr := output.Close()
	if copyErr != nil || closeErr != nil || written < 1 || written > maxZipCompressed {
		return validation, receipt, errPassInvalid
	}
	if err := os.Chmod(destination, 0600); err != nil {
		return validation, receipt, errIsolation
	}
	file, err := os.Open(destination)
	if err != nil {
		return validation, receipt, errIsolation
	}
	info, err := file.Stat()
	if err == nil {
		receipt, err = scanPrivateZip(file, info.Size())
	}
	closeErr = file.Close()
	if err != nil || closeErr != nil {
		if errors.Is(err, errZipDuplicate) || errors.Is(err, errZipUnsafe) {
			return validation, receipt, err
		}
		return validation, receipt, errZipUnsafe
	}
	validation, resultErr = validateSealedPassArchive(destination, expectedRole)
	return validation, receipt, resultErr
}

func readZipFile(entry *zip.File) ([]byte, error) {
	if entry.UncompressedSize64 > maxZipFileBytes {
		return nil, errZipUnsafe
	}
	reader, err := entry.Open()
	if err != nil {
		return nil, errZipUnsafe
	}
	defer reader.Close()
	return io.ReadAll(io.LimitReader(reader, maxZipFileBytes+1))
}

func decodeClosedReceipt(data []byte, receipt *passArchiveReceipt) error {
	return decodeClosedObject(data, receipt, "classificationRunId", "passRole", "passRunId", "promptDigest", "modelDescriptorDigest", "batchManifestDigest", "batchCount", "itemCount", "orderedInputDigests", "passInputSetDigest", "orderedBatchOutputDigests", "orderedPassResultDigests", "passSealDigest")
}

func decodeClosedSealedBatch(data []byte, output *agaapplicability.PassBatchOutput) error {
	var wire sealedPassBatchWire
	if err := decodeClosedObject(data, &wire, "schemaVersion", "classificationRunId", "passRole", "passRunId", "batchOrdinal", "promptDigest", "modelDescriptorDigest", "inputDigest", "records", "batchOutputDigest"); err != nil {
		return err
	}
	records := make([]agaapplicability.PassProposalRecord, 0, len(wire.Records))
	for _, raw := range wire.Records {
		var recordWire sealedPassRecordWire
		if err := decodeClosedObject(raw, &recordWire, "identity", "classificationRunId", "passRole", "passRunId", "promptDigest", "modelDescriptorDigest", "inputDigest", "proposalProjection", "rationaleCodes", "confidenceEvidence", "sourceRefs", "passResultDigest"); err != nil {
			return err
		}
		recordValue := agaapplicability.PassProposalRecord(recordWire)
		expected, err := agaapplicability.NewPassProposalRecordForSuppliedProvenance(agaapplicability.FrozenTaxonomy(), agaapplicability.PassProposalInput{Identity: recordValue.Identity, ClassificationRunID: recordValue.ClassificationRunID, PassRole: recordValue.PassRole, PassRunID: recordValue.PassRunID, PromptDigest: recordValue.PromptDigest, ModelDescriptorDigest: recordValue.ModelDescriptorDigest, InputDigest: recordValue.InputDigest, Projection: recordValue.ProposalProjection, RationaleCodes: recordValue.RationaleCodes, ConfidenceEvidence: recordValue.ConfidenceEvidence, SourceRefs: recordValue.SourceRefs})
		if err != nil || !reflect.DeepEqual(recordValue, expected) {
			return errPassSchema
		}
		records = append(records, recordValue)
	}
	*output = agaapplicability.PassBatchOutput{SchemaVersion: wire.SchemaVersion, ClassificationRunID: wire.ClassificationRunID, PassRole: wire.PassRole, PassRunID: wire.PassRunID, BatchOrdinal: wire.BatchOrdinal, PromptDigest: wire.PromptDigest, ModelDescriptorDigest: wire.ModelDescriptorDigest, InputDigest: wire.InputDigest, Records: records, BatchOutputDigest: wire.BatchOutputDigest}
	return nil
}

func decodeClosedObject(data []byte, target any, keys ...string) error {
	var object map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&object); err != nil || len(object) != len(keys) {
		return errPassSchema
	}
	for _, key := range keys {
		if _, ok := object[key]; !ok {
			return errPassSchema
		}
	}
	decoder = json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return errPassSchema
	}
	return nil
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func validSHA256Digest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return false
	}
	decoded, err := hex.DecodeString(value[len("sha256:"):])
	return err == nil && hex.EncodeToString(decoded) == value[len("sha256:"):]
}

func validateEvidence(_ any, _ any) error { return errPassInvalid }

func diagnostic(code, _ string) string {
	if !strings.HasPrefix(code, "ERR_AGA_") {
		return "ERR_AGA_PASS_INVALID"
	}
	return code
}

func reconcile(candidate, challenge any) (any, error) {
	if candidate == nil || challenge == nil {
		return nil, errCandidateInvalid
	}
	return nil, errCandidateInvalid
}

func verifyPassProjections(values any) error {
	if values == nil {
		return errCandidateInvalid
	}
	return errCandidateInvalid
}

func validateCandidateDirectory(directory string) error {
	if directory == "" {
		return errCandidateInvalid
	}
	info, err := os.Stat(directory)
	if err != nil || !info.IsDir() {
		return errCandidateInvalid
	}
	return nil
}

// verifyIsolatedRoots operates on caller-supplied relative manifest names,
// never filesystem paths. A pass may not name the other role or its output.
func verifyIsolatedRoots(candidate, challenge []string) error {
	for _, name := range append(append([]string{}, candidate...), challenge...) {
		if strings.Contains(name, "../") || strings.HasPrefix(name, "/") {
			return errIsolation
		}
	}
	for _, name := range candidate {
		if strings.HasPrefix(name, "challenge/") {
			return errIsolation
		}
	}
	for _, name := range challenge {
		if strings.HasPrefix(name, "candidate/") {
			return errIsolation
		}
	}
	return nil
}

func validateBatchUnion(ordinals []int, expected int) error {
	if len(ordinals) != expected {
		return errPassBijection
	}
	seen := make(map[int]struct{}, expected)
	for _, ordinal := range ordinals {
		if ordinal < 1 || ordinal > expected {
			return errPassBijection
		}
		if _, exists := seen[ordinal]; exists {
			return errPassBijection
		}
		seen[ordinal] = struct{}{}
	}
	return nil
}

func validateRecordUnion(count int) error {
	if count != 1310 {
		return errPassBijection
	}
	return nil
}

func decodeClosedPassBatch(data []byte, batch *rawPassBatch) error {
	var object map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&object); err != nil || object == nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return errPassSchema
	}
	allowed := map[string]bool{"schemaVersion": true, "passRole": true, "batchOrdinal": true, "sourceSnapshotDigest": true, "records": true}
	if len(object) != len(allowed) {
		return errPassSchema
	}
	for key := range object {
		if !allowed[key] {
			return errPassSchema
		}
	}
	decoder = json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	return decoder.Decode(batch)
}

func containsForbiddenTextField(data []byte) bool {
	var value any
	if json.Unmarshal(data, &value) != nil {
		return false
	}
	return walkForbidden(value)
}

func walkForbidden(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			lower := strings.ToLower(key)
			if strings.Contains(lower, "questionbody") || strings.Contains(lower, "questiontext") || lower == "body" || strings.Contains(lower, "transcript") || strings.Contains(lower, "reasoning") {
				return true
			}
			if walkForbidden(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if walkForbidden(child) {
				return true
			}
		}
	}
	return false
}

// scanPrivateZip validates transport before extraction. AppleDouble and
// __MACOSX entries are counted as noise and are never semantic input.
func scanPrivateZip(reader io.ReaderAt, size int64) (privateZipReceipt, error) {
	zr, err := zip.NewReader(reader, size)
	if err != nil {
		return privateZipReceipt{}, errZipUnsafe
	}
	if len(zr.File) == 0 || len(zr.File) > maxZipEntries {
		return privateZipReceipt{}, errZipUnsafe
	}
	seen := map[string]struct{}{}
	var receipt privateZipReceipt
	for _, entry := range zr.File {
		name := entry.Name
		if !safeZipName(name) {
			return privateZipReceipt{}, errZipUnsafe
		}
		if entry.FileInfo().IsDir() {
			continue
		}
		if isTransportNoise(name) {
			receipt.TransportNoise++
			continue
		}
		if entry.FileInfo().Mode()&os.ModeSymlink != 0 || !entry.FileInfo().Mode().IsRegular() {
			return privateZipReceipt{}, errZipUnsafe
		}
		if entry.UncompressedSize64 > maxZipFileBytes || receipt.ExpandedBytes+entry.UncompressedSize64 > maxZipExpanded {
			return privateZipReceipt{}, errZipUnsafe
		}
		semantic := path.Clean(name)
		if _, exists := seen[semantic]; exists {
			return privateZipReceipt{}, errZipDuplicate
		}
		seen[semantic] = struct{}{}
		receipt.SemanticEntries++
		receipt.ExpandedBytes += entry.UncompressedSize64
	}
	receipt.Digest = receiptDigest(receipt)
	return receipt, nil
}

func safeZipName(name string) bool {
	if name == "" || strings.HasPrefix(name, "/") || strings.HasPrefix(name, "\\") || strings.Contains(name, "\\") {
		return false
	}
	clean := path.Clean(name)
	return clean != "." && clean != ".." && !strings.HasPrefix(clean, "../")
}

func isTransportNoise(name string) bool {
	base := path.Base(name)
	return strings.HasPrefix(name, "__MACOSX/") || strings.HasPrefix(base, "._")
}

func receiptDigest(receipt privateZipReceipt) string {
	payload := fmt.Sprintf("%d:%d:%d", receipt.SemanticEntries, receipt.TransportNoise, receipt.ExpandedBytes)
	sum := sha256.Sum256([]byte(payload))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func semanticZipNames(reader io.ReaderAt, size int64) ([]string, error) {
	zr, err := zip.NewReader(reader, size)
	if err != nil {
		return nil, errZipUnsafe
	}
	result := make([]string, 0, len(zr.File))
	for _, entry := range zr.File {
		if !isTransportNoise(entry.Name) {
			result = append(result, path.Clean(entry.Name))
		}
	}
	sort.Strings(result)
	return result, nil
}
