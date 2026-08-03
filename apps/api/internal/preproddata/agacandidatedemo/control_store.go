package agacandidatedemo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type ControlStore struct{ root string }

func NewControlStore(root string) (*ControlStore, error) {
	if !filepath.IsAbs(root) {
		return nil, fmt.Errorf("AGA demo control store path must be absolute")
	}
	root = filepath.Join(filepath.Clean(root), "aga-demo")
	for _, name := range []string{"intents", "authorizations", "results", "cleanup"} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(path, 0o700); err != nil {
			return nil, err
		}
		info, err := os.Lstat(path)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 {
			return nil, fmt.Errorf("AGA demo control store directory is not private")
		}
	}
	return &ControlStore{root: root}, nil
}

func (store *ControlStore) AppendIntent(intent IntentManifest) error {
	if err := intent.Validate(); err != nil {
		return err
	}
	data, err := json.Marshal(intent)
	if err != nil {
		return err
	}
	path := filepath.Join(store.root, "intents", digestFilename(intent.IntentDigest)+".json")
	if existing, err := os.ReadFile(path); err == nil {
		if string(existing) == string(append(data, '\n')) {
			return nil
		}
		return fmt.Errorf("AGA demo intent digest conflicts with stored content")
	} else if !os.IsNotExist(err) {
		return err
	}
	return appendOnly(path, data)
}

func (store *ControlStore) ConsumeAuthorization(authorization OperationAuthorization, operation string, now time.Time) error {
	cleaned, err := store.IsCleaned(authorization.RunID, authorization.IntentDigest)
	if err != nil {
		return err
	}
	if cleaned {
		return fmt.Errorf("AGA demo run was cleaned and is non-replayable")
	}
	data, err := os.ReadFile(filepath.Join(store.root, "intents", digestFilename(authorization.IntentDigest)+".json"))
	if err != nil {
		return fmt.Errorf("read AGA demo intent: %w", err)
	}
	var intent IntentManifest
	if err := json.Unmarshal(data, &intent); err != nil {
		return err
	}
	if err := authorization.Validate(intent, operation, now); err != nil {
		return err
	}
	record := struct {
		SchemaVersion           string    `json:"schemaVersion"`
		AuthorizationHash       string    `json:"authorizationHash"`
		Operation               string    `json:"operation"`
		Issuer                  string    `json:"issuer"`
		Nonce                   string    `json:"nonce"`
		RunID                   string    `json:"runId"`
		IntentDigest            string    `json:"intentDigest"`
		TargetFingerprintDigest string    `json:"targetFingerprintDigest"`
		ExpiresAt               time.Time `json:"expiresAt"`
		ConsumedAt              time.Time `json:"consumedAt"`
	}{"preprod-aga-candidate-demo-authorization-consumption/v1", authorization.Hash(), authorization.Operation, authorization.Issuer, authorization.Nonce, authorization.RunID, authorization.IntentDigest, authorization.TargetFingerprintDigest, authorization.ExpiresAt.UTC(), now.UTC()}
	encoded, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return appendOnly(filepath.Join(store.root, "authorizations", digestFilename(authorization.Hash())+".json"), encoded)
}

func (store *ControlStore) AuthorizationRecords() ([]byte, error) {
	entries, err := os.ReadDir(filepath.Join(store.root, "authorizations"))
	if err != nil {
		return nil, err
	}
	var output []byte
	for _, entry := range entries {
		if entry.IsDir() {
			return nil, fmt.Errorf("invalid authorization record directory")
		}
		data, err := os.ReadFile(filepath.Join(store.root, "authorizations", entry.Name()))
		if err != nil {
			return nil, err
		}
		output = append(output, data...)
	}
	return output, nil
}

func (store *ControlStore) AppendResult(result ResultManifest) error {
	if err := result.Validate(); err != nil {
		return err
	}
	data, err := os.ReadFile(filepath.Join(store.root, "intents", digestFilename(result.IntentDigest)+".json"))
	if err != nil {
		return fmt.Errorf("read AGA demo intent: %w", err)
	}
	var intent IntentManifest
	if err := json.Unmarshal(data, &intent); err != nil || intent.RunID != result.RunID || intent.IntentDigest != result.IntentDigest {
		return fmt.Errorf("result does not bind the stored AGA demo intent")
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return appendOnly(filepath.Join(store.root, "results", digestFilename(result.ResultDigest)+".json"), encoded)
}

// Result returns the single immutable result bound to an intent. It never
// reads database state, so cleanup can tombstone a completed overlay only after
// its durable control-plane receipt exists.
func (store *ControlStore) Result(runID, intentDigest string) (ResultManifest, error) {
	if !runIDPattern.MatchString(runID) || !validDigest(intentDigest) {
		return ResultManifest{}, fmt.Errorf("invalid AGA demo result lookup")
	}
	entries, err := os.ReadDir(filepath.Join(store.root, "results"))
	if err != nil {
		return ResultManifest{}, err
	}
	var matched *ResultManifest
	for _, entry := range entries {
		if entry.IsDir() {
			return ResultManifest{}, fmt.Errorf("invalid AGA demo result directory")
		}
		data, err := os.ReadFile(filepath.Join(store.root, "results", entry.Name()))
		if err != nil {
			return ResultManifest{}, err
		}
		var result ResultManifest
		if err := json.Unmarshal(data, &result); err != nil || result.Validate() != nil {
			return ResultManifest{}, fmt.Errorf("invalid AGA demo result")
		}
		if result.RunID == runID && result.IntentDigest == intentDigest {
			if matched != nil {
				return ResultManifest{}, fmt.Errorf("multiple AGA demo results bind the same intent")
			}
			copy := result
			matched = &copy
		}
	}
	if matched == nil {
		return ResultManifest{}, fmt.Errorf("AGA demo result is missing")
	}
	return *matched, nil
}

func (store *ControlStore) AppendCleanupTombstone(tombstone CleanupTombstone) error {
	if err := tombstone.Validate(); err != nil {
		return err
	}
	resultPath := filepath.Join(store.root, "results", digestFilename(tombstone.ResultDigest)+".json")
	data, err := os.ReadFile(resultPath)
	if err != nil {
		return fmt.Errorf("cleanup tombstone requires stored result: %w", err)
	}
	var result ResultManifest
	if err := json.Unmarshal(data, &result); err != nil || result.Validate() != nil || result.RunID != tombstone.RunID || result.IntentDigest != tombstone.IntentDigest || result.ResultDigest != tombstone.ResultDigest {
		return fmt.Errorf("cleanup tombstone result binding mismatch")
	}
	encoded, err := json.Marshal(tombstone)
	if err != nil {
		return err
	}
	return appendOnly(filepath.Join(store.root, "cleanup", digestFilename(tombstone.TombstoneDigest)+".json"), encoded)
}

func (store *ControlStore) IsCleaned(runID, intentDigest string) (bool, error) {
	if !runIDPattern.MatchString(runID) || !validDigest(intentDigest) {
		return false, fmt.Errorf("invalid AGA demo cleanup lookup")
	}
	entries, err := os.ReadDir(filepath.Join(store.root, "cleanup"))
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return false, fmt.Errorf("invalid cleanup record directory")
		}
		data, err := os.ReadFile(filepath.Join(store.root, "cleanup", entry.Name()))
		if err != nil {
			return false, err
		}
		var tombstone CleanupTombstone
		if err := json.Unmarshal(data, &tombstone); err != nil || tombstone.Validate() != nil {
			return false, fmt.Errorf("invalid cleanup tombstone")
		}
		if tombstone.RunID == runID && tombstone.IntentDigest == intentDigest {
			return true, nil
		}
	}
	return false, nil
}

func appendOnly(path string, data []byte) error {
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("append-only record already exists")
	} else if !os.IsNotExist(err) {
		return err
	}
	directory := filepath.Dir(path)
	temporary := filepath.Join(directory, ".tmp-"+filepath.Base(path)+"-"+strconv.FormatInt(time.Now().UnixNano(), 10))
	file, err := os.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer os.Remove(temporary)
	if _, err := file.Write(append(data, '\n')); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Link(temporary, path); err != nil {
		return err
	}
	directoryFile, err := os.Open(directory)
	if err != nil {
		return err
	}
	defer directoryFile.Close()
	return directoryFile.Sync()
}
func digestFilename(digest string) string { return strings.TrimPrefix(digest, "sha256:") }
