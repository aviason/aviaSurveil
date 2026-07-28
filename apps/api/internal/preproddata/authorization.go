package preproddata

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrAuthorizationConsumed = errors.New("operation authorization already consumed")
	ErrInvalidAuthorization  = errors.New("invalid operation authorization")
)

type Operation string

const (
	LoadEmptyTarget    Operation = "LOAD_EMPTY_TARGET"
	ResumeRun          Operation = "RESUME_RUN"
	DropRecreateTarget Operation = "DROP_RECREATE_TARGET"
)

type OperationAuthorization struct {
	SchemaVersion           string    `json:"schemaVersion"`
	Token                   string    `json:"token"`
	Operation               Operation `json:"operation"`
	Issuer                  string    `json:"issuer"`
	ExpiresAt               time.Time `json:"expiresAt"`
	Nonce                   string    `json:"nonce"`
	RunID                   string    `json:"runId"`
	IntentDigest            string    `json:"intentDigest"`
	TargetFingerprintDigest string    `json:"targetFingerprintDigest"`
}

func (authorization OperationAuthorization) Validate(
	intent IntentManifest,
	now time.Time,
) error {
	if err := intent.Validate(); err != nil {
		return err
	}
	if authorization.SchemaVersion != "preprod-operation-authorization/v1" ||
		len(authorization.Token) < 16 ||
		strings.TrimSpace(authorization.Issuer) == "" ||
		strings.TrimSpace(authorization.Nonce) == "" ||
		authorization.RunID != intent.RunID ||
		authorization.IntentDigest != intent.IntentDigest ||
		authorization.TargetFingerprintDigest != intent.TargetFingerprintDigest ||
		!authorization.ExpiresAt.After(now.UTC()) ||
		authorization.ExpiresAt.After(now.UTC().Add(15*time.Minute)) {
		return ErrInvalidAuthorization
	}
	switch authorization.Operation {
	case LoadEmptyTarget, ResumeRun, DropRecreateTarget:
	default:
		return ErrInvalidAuthorization
	}
	return nil
}

func (authorization OperationAuthorization) Hash() string {
	digest := sha256.Sum256([]byte(authorization.Token))
	return "sha256:" + hex.EncodeToString(digest[:])
}

func WriteAuthorizationFile(
	path string,
	authorization OperationAuthorization,
) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("authorization file path must be absolute")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	encoder.SetEscapeHTML(false)
	encodeErr := encoder.Encode(authorization)
	closeErr := file.Close()
	if encodeErr != nil {
		return encodeErr
	}
	return closeErr
}

func ReadAuthorizationFile(path string) (OperationAuthorization, error) {
	if !filepath.IsAbs(path) {
		return OperationAuthorization{}, fmt.Errorf(
			"%w: authorization path must be absolute",
			ErrInvalidAuthorization,
		)
	}
	info, err := os.Stat(path)
	if err != nil {
		return OperationAuthorization{}, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		return OperationAuthorization{}, fmt.Errorf(
			"%w: authorization file mode must be 0600",
			ErrInvalidAuthorization,
		)
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
		return OperationAuthorization{}, fmt.Errorf(
			"%w: %v",
			ErrInvalidAuthorization,
			err,
		)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return OperationAuthorization{}, fmt.Errorf(
			"%w: trailing authorization content",
			ErrInvalidAuthorization,
		)
	}
	return authorization, nil
}
