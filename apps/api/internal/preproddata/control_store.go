package preproddata

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var (
	ErrAppendOnlyConflict = errors.New("append-only control record conflicts")
	ErrRunIDConflict      = errors.New("run ID is already bound to another intent")
	ErrNoSuccessfulResult = errors.New("run has no successful result")
	ErrNoCheckpoints      = errors.New("run has no checkpoints")
)

type FileControlStore struct {
	root string
}

type authorizationConsumptionRecord struct {
	SchemaVersion           string    `json:"schemaVersion"`
	AuthorizationHash       string    `json:"authorizationHash"`
	Operation               Operation `json:"operation"`
	Issuer                  string    `json:"issuer"`
	ExpiresAt               time.Time `json:"expiresAt"`
	Nonce                   string    `json:"nonce"`
	RunID                   string    `json:"runId"`
	IntentDigest            string    `json:"intentDigest"`
	TargetFingerprintDigest string    `json:"targetFingerprintDigest"`
	ConsumedAt              time.Time `json:"consumedAt"`
}

func NewFileControlStore(root string) (*FileControlStore, error) {
	if !filepath.IsAbs(root) {
		return nil, fmt.Errorf("control store path must be absolute")
	}
	root = filepath.Clean(root)
	if root == string(filepath.Separator) {
		return nil, fmt.Errorf("control store cannot use the filesystem root")
	}
	for _, directory := range []string{
		root,
		filepath.Join(root, "intents", "by-digest"),
		filepath.Join(root, "runs"),
		filepath.Join(root, "results"),
		filepath.Join(root, "checkpoints"),
		filepath.Join(root, "authorizations"),
		filepath.Join(root, "cleanup"),
	} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return nil, err
		}
		info, err := os.Lstat(directory)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() ||
			info.Mode()&os.ModeSymlink != 0 ||
			info.Mode().Perm()&0o077 != 0 {
			return nil, fmt.Errorf(
				"control store directories must be private real directories",
			)
		}
	}
	return &FileControlStore{root: root}, nil
}

func (store *FileControlStore) Root() string {
	return store.root
}

func (store *FileControlStore) AppendIntent(intent IntentManifest) error {
	if !runIDPattern.MatchString(intent.RunID) ||
		!digestPattern.MatchString(intent.IntentDigest) {
		return ErrInvalidIntent
	}
	runBindingPath := filepath.Join(store.root, "runs", intent.RunID+".intent")
	if binding, err := os.ReadFile(runBindingPath); err == nil {
		existingDigest := strings.TrimSpace(string(binding))
		existingPath := filepath.Join(
			store.root,
			"intents",
			"by-digest",
			digestFilename(existingDigest)+".json",
		)
		existing, readErr := os.ReadFile(existingPath)
		current, canonicalErr := canonicalJSON(intent)
		if readErr == nil && canonicalErr == nil && bytes.Equal(existing, current) {
			return nil
		}
		return ErrRunIDConflict
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := intent.Validate(); err != nil {
		return err
	}
	encoded, err := canonicalJSON(intent)
	if err != nil {
		return err
	}
	intentPath := filepath.Join(
		store.root,
		"intents",
		"by-digest",
		digestFilename(intent.IntentDigest)+".json",
	)
	if err := appendImmutable(intentPath, encoded); err != nil {
		return err
	}
	return appendImmutable(
		runBindingPath,
		[]byte(intent.IntentDigest+"\n"),
	)
}

func (store *FileControlStore) AppendResult(result ResultManifest) error {
	if err := result.Validate(); err != nil {
		return err
	}
	intent, err := store.intentForRun(result.RunID)
	if err != nil {
		return fmt.Errorf("read result intent binding: %w", err)
	}
	if intent.IntentDigest != result.IntentDigest {
		return fmt.Errorf("result intent does not match the run binding")
	}
	if result.Outcome == "SUCCEEDED" {
		if err := validateReconciliation(intent, Reconciliation{
			ActualCounts:        result.ActualCounts,
			RelationshipDigests: result.RelationshipDigests,
		}); err != nil {
			return err
		}
	}
	encoded, err := canonicalJSON(result)
	if err != nil {
		return err
	}
	directory := filepath.Join(store.root, "results", result.RunID)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	if err := appendImmutable(
		filepath.Join(directory, digestFilename(result.ResultDigest)+".json"),
		encoded,
	); err != nil {
		return err
	}
	if result.Outcome != "SUCCEEDED" {
		return nil
	}
	return appendImmutable(
		filepath.Join(store.root, "runs", result.RunID+".success"),
		[]byte(result.ResultDigest+"\n"),
	)
}

func (store *FileControlStore) SuccessfulResult(
	runID string,
	intentDigest string,
) (ResultManifest, error) {
	if !runIDPattern.MatchString(runID) ||
		!digestPattern.MatchString(intentDigest) {
		return ResultManifest{}, fmt.Errorf("invalid successful-result identity")
	}
	binding, err := os.ReadFile(
		filepath.Join(store.root, "runs", runID+".success"),
	)
	if errors.Is(err, fs.ErrNotExist) {
		return ResultManifest{}, ErrNoSuccessfulResult
	}
	if err != nil {
		return ResultManifest{}, err
	}
	resultDigest := strings.TrimSpace(string(binding))
	if !digestPattern.MatchString(resultDigest) {
		return ResultManifest{}, fmt.Errorf("successful-result binding is malformed")
	}
	var result ResultManifest
	if err := decodeJSONFile(
		filepath.Join(
			store.root,
			"results",
			runID,
			digestFilename(resultDigest)+".json",
		),
		&result,
	); err != nil {
		return ResultManifest{}, err
	}
	if err := result.Validate(); err != nil {
		return ResultManifest{}, err
	}
	if result.RunID != runID ||
		result.IntentDigest != intentDigest ||
		result.ResultDigest != resultDigest ||
		result.Outcome != "SUCCEEDED" {
		return ResultManifest{}, fmt.Errorf(
			"successful-result binding does not match the requested run",
		)
	}
	return result, nil
}

func (store *FileControlStore) AppendCheckpoint(
	checkpoint Checkpoint,
) error {
	if err := validateCheckpoint(checkpoint); err != nil {
		return err
	}
	encoded, err := canonicalJSON(checkpoint)
	if err != nil {
		return err
	}
	directory := filepath.Join(store.root, "checkpoints", checkpoint.RunID)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	return appendOnce(
		filepath.Join(
			directory,
			fmt.Sprintf("%012d.json", checkpoint.Sequence),
		),
		encoded,
	)
}

func (store *FileControlStore) RunCheckpoints(
	runID string,
	intentDigest string,
) ([]Checkpoint, error) {
	if !runIDPattern.MatchString(runID) ||
		!digestPattern.MatchString(intentDigest) {
		return nil, fmt.Errorf("invalid checkpoint identity")
	}
	directory := filepath.Join(store.root, "checkpoints", runID)
	entries, err := os.ReadDir(directory)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNoCheckpoints
	}
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(left, right int) bool {
		return entries[left].Name() < entries[right].Name()
	})
	checkpoints := make([]Checkpoint, 0, len(entries))
	var previousSequence int64
	var previousApplied int64
	for _, entry := range entries {
		if entry.IsDir() {
			return nil, fmt.Errorf("checkpoint directory contains a subdirectory")
		}
		var checkpoint Checkpoint
		if err := decodeJSONFile(
			filepath.Join(directory, entry.Name()),
			&checkpoint,
		); err != nil {
			return nil, err
		}
		if err := validateCheckpoint(checkpoint); err != nil {
			return nil, err
		}
		if checkpoint.RunID != runID ||
			checkpoint.IntentDigest != intentDigest ||
			checkpoint.Sequence != previousSequence+1 ||
			checkpoint.AppliedCommands < previousApplied {
			return nil, fmt.Errorf("checkpoint history is not monotonic")
		}
		checkpoints = append(checkpoints, checkpoint)
		previousSequence = checkpoint.Sequence
		previousApplied = checkpoint.AppliedCommands
	}
	if len(checkpoints) == 0 {
		return nil, ErrNoCheckpoints
	}
	return checkpoints, nil
}

func validateCheckpoint(checkpoint Checkpoint) error {
	if checkpoint.SchemaVersion != "preprod-run-checkpoint/v1" ||
		!runIDPattern.MatchString(checkpoint.RunID) ||
		!digestPattern.MatchString(checkpoint.IntentDigest) ||
		checkpoint.Sequence < 1 ||
		strings.TrimSpace(checkpoint.Name) == "" ||
		checkpoint.AppliedCommands < 0 ||
		checkpoint.RecordedAt.IsZero() ||
		(checkpoint.AppliedCommands == 0 &&
			checkpoint.LastOperationID != "") ||
		(checkpoint.AppliedCommands > 0 &&
			strings.TrimSpace(checkpoint.LastOperationID) == "") {
		return fmt.Errorf("invalid checkpoint")
	}
	return nil
}

func (store *FileControlStore) ConsumeAuthorization(
	authorization OperationAuthorization,
	consumedAt time.Time,
) error {
	intent, err := store.intentForRun(authorization.RunID)
	if err != nil {
		return fmt.Errorf("read authorization intent: %w", err)
	}
	if err := authorization.Validate(intent, consumedAt.UTC()); err != nil {
		return err
	}
	// Only the SHA-256 authorization hash and bound public claims are retained.
	authorizationHash := authorization.Hash()
	record := authorizationConsumptionRecord{
		SchemaVersion:     "preprod-authorization-consumption/v1",
		AuthorizationHash: authorizationHash,
		Operation:         authorization.Operation, Issuer: authorization.Issuer,
		ExpiresAt: authorization.ExpiresAt.UTC(), Nonce: authorization.Nonce,
		RunID: authorization.RunID, IntentDigest: authorization.IntentDigest,
		TargetFingerprintDigest: authorization.TargetFingerprintDigest,
		ConsumedAt:              consumedAt.UTC(),
	}
	encoded, err := canonicalJSON(record)
	if err != nil {
		return err
	}
	path := filepath.Join(
		store.root,
		"authorizations",
		digestFilename(authorizationHash)+".json",
	)
	if err := appendOnce(path, encoded); errors.Is(err, ErrAppendOnlyConflict) {
		return ErrAuthorizationConsumed
	} else {
		return err
	}
}

func (store *FileControlStore) AuthorizationRecords() ([]byte, error) {
	entries, err := os.ReadDir(filepath.Join(store.root, "authorizations"))
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(left, right int) bool {
		return entries[left].Name() < entries[right].Name()
	})
	var output bytes.Buffer
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		record, err := os.ReadFile(
			filepath.Join(store.root, "authorizations", entry.Name()),
		)
		if err != nil {
			return nil, err
		}
		output.Write(record)
		output.WriteByte('\n')
	}
	return output.Bytes(), nil
}

func (store *FileControlStore) AppendCleanupAttestation(
	attestation CleanupAttestation,
) error {
	if attestation.SchemaVersion != "preprod-cleanup-attestation/v1" ||
		!runIDPattern.MatchString(attestation.RunID) ||
		!digestPattern.MatchString(attestation.IntentDigest) ||
		!digestPattern.MatchString(attestation.ResultDigest) ||
		!digestPattern.MatchString(attestation.TargetDigest) ||
		!digestPattern.MatchString(attestation.AuthorizationHash) ||
		attestation.CleanedAt.IsZero() {
		return fmt.Errorf("invalid cleanup attestation")
	}
	intent, err := store.intentForRun(attestation.RunID)
	if err != nil {
		return fmt.Errorf("read cleanup intent: %w", err)
	}
	if intent.IntentDigest != attestation.IntentDigest ||
		intent.TargetFingerprintDigest != attestation.TargetDigest {
		return fmt.Errorf("cleanup attestation target or intent mismatch")
	}
	result, err := store.SuccessfulResult(
		attestation.RunID,
		attestation.IntentDigest,
	)
	if err != nil {
		return fmt.Errorf("read cleanup result: %w", err)
	}
	if result.ResultDigest != attestation.ResultDigest {
		return fmt.Errorf("cleanup attestation result mismatch")
	}
	var authorization authorizationConsumptionRecord
	if err := decodeJSONFile(
		filepath.Join(
			store.root,
			"authorizations",
			digestFilename(attestation.AuthorizationHash)+".json",
		),
		&authorization,
	); err != nil {
		return fmt.Errorf("read cleanup authorization: %w", err)
	}
	if authorization.AuthorizationHash != attestation.AuthorizationHash ||
		authorization.Operation != DropRecreateTarget ||
		authorization.RunID != attestation.RunID ||
		authorization.IntentDigest != attestation.IntentDigest ||
		authorization.TargetFingerprintDigest != attestation.TargetDigest ||
		authorization.ConsumedAt.After(attestation.CleanedAt.UTC()) {
		return fmt.Errorf(
			"cleanup attestation requires a consumed DROP_RECREATE_TARGET authorization",
		)
	}
	copy := attestation
	copy.AttestationDigest = ""
	payload, err := canonicalJSON(copy)
	if err != nil {
		return err
	}
	expectedDigest := sha256Digest(payload)
	if attestation.AttestationDigest == "" {
		attestation.AttestationDigest = expectedDigest
	} else if attestation.AttestationDigest != expectedDigest {
		return fmt.Errorf("cleanup attestation digest mismatch")
	}
	encoded, err := canonicalJSON(attestation)
	if err != nil {
		return err
	}
	directory := filepath.Join(store.root, "cleanup", attestation.RunID)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	return appendImmutable(
		filepath.Join(
			directory,
			digestFilename(attestation.AttestationDigest)+".json",
		),
		encoded,
	)
}

func (store *FileControlStore) intentForRun(
	runID string,
) (IntentManifest, error) {
	if !runIDPattern.MatchString(runID) {
		return IntentManifest{}, ErrInvalidIntent
	}
	binding, err := os.ReadFile(
		filepath.Join(store.root, "runs", runID+".intent"),
	)
	if err != nil {
		return IntentManifest{}, err
	}
	intentDigest := strings.TrimSpace(string(binding))
	if !digestPattern.MatchString(intentDigest) {
		return IntentManifest{}, fmt.Errorf("run intent binding is malformed")
	}
	var intent IntentManifest
	if err := decodeJSONFile(
		filepath.Join(
			store.root,
			"intents",
			"by-digest",
			digestFilename(intentDigest)+".json",
		),
		&intent,
	); err != nil {
		return IntentManifest{}, err
	}
	if err := intent.Validate(); err != nil {
		return IntentManifest{}, err
	}
	if intent.RunID != runID || intent.IntentDigest != intentDigest {
		return IntentManifest{}, fmt.Errorf(
			"run intent binding does not match stored intent",
		)
	}
	return intent, nil
}

func appendImmutable(path string, contents []byte) error {
	file, err := os.OpenFile(
		path,
		os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		0o600,
	)
	if err == nil {
		if _, writeErr := file.Write(contents); writeErr != nil {
			_ = file.Close()
			return writeErr
		}
		return file.Close()
	}
	if !errors.Is(err, fs.ErrExist) {
		return err
	}
	existing, readErr := os.ReadFile(path)
	if readErr != nil {
		return readErr
	}
	if bytes.Equal(existing, contents) {
		return nil
	}
	return ErrAppendOnlyConflict
}

func appendOnce(path string, contents []byte) error {
	file, err := os.OpenFile(
		path,
		os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		0o600,
	)
	if errors.Is(err, fs.ErrExist) {
		return ErrAppendOnlyConflict
	}
	if err != nil {
		return err
	}
	if _, err := file.Write(contents); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func digestFilename(digest string) string {
	return strings.TrimPrefix(digest, "sha256:")
}

func decodeJSONFile(path string, output any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 2*1024*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return fmt.Errorf("JSON record contains trailing content")
	}
	return nil
}
