package agacandidatedemo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type OperationAuthorization struct {
	SchemaVersion           string    `json:"schemaVersion"`
	Token                   string    `json:"token"`
	Operation               string    `json:"operation"`
	Issuer                  string    `json:"issuer"`
	ExpiresAt               time.Time `json:"expiresAt"`
	Nonce                   string    `json:"nonce"`
	RunID                   string    `json:"runId"`
	IntentDigest            string    `json:"intentDigest"`
	TargetFingerprintDigest string    `json:"targetFingerprintDigest"`
}

const (
	LoadOverlayOperation    = "LOAD_AGA_CANDIDATE_DEMO_OVERLAY"
	CleanupOverlayOperation = "CLEANUP_AGA_CANDIDATE_DEMO_OVERLAY"
)

func (authorization OperationAuthorization) Validate(intent IntentManifest, operation string, now time.Time) error {
	if err := intent.Validate(); err != nil {
		return err
	}
	if (operation != LoadOverlayOperation && operation != CleanupOverlayOperation) || authorization.SchemaVersion != "preprod-aga-candidate-demo-operation-authorization/v1" || authorization.Operation != operation || len(authorization.Token) < 16 || strings.TrimSpace(authorization.Issuer) == "" || strings.TrimSpace(authorization.Nonce) == "" || authorization.RunID != intent.RunID || authorization.IntentDigest != intent.IntentDigest || authorization.TargetFingerprintDigest != intent.TargetFingerprintDigest || !authorization.ExpiresAt.After(now.UTC()) || authorization.ExpiresAt.After(now.UTC().Add(15*time.Minute)) {
		return fmt.Errorf("invalid AGA demo operation authorization")
	}
	return nil
}

func (authorization OperationAuthorization) Hash() string {
	digest := sha256.Sum256([]byte(authorization.Token))
	return "sha256:" + hex.EncodeToString(digest[:])
}

func WriteAuthorizationFile(path string, authorization OperationAuthorization) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("AGA demo authorization path must be absolute")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	data, err := json.Marshal(authorization)
	if err == nil {
		_, err = file.Write(append(data, '\n'))
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func ReadAuthorizationFile(path string) (OperationAuthorization, error) {
	if !filepath.IsAbs(path) {
		return OperationAuthorization{}, fmt.Errorf("AGA demo authorization path must be absolute")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return OperationAuthorization{}, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o600 {
		return OperationAuthorization{}, fmt.Errorf("AGA demo authorization file must be a 0600 regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return OperationAuthorization{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 64*1024))
	decoder.DisallowUnknownFields()
	var authorization OperationAuthorization
	if err := decoder.Decode(&authorization); err != nil {
		return OperationAuthorization{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return OperationAuthorization{}, fmt.Errorf("trailing AGA demo authorization content")
	}
	return authorization, nil
}
