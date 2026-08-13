package scanner

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/aviason/aviaSurveil/internal/platform/objectstore"
)

type Result struct {
	Clean            bool
	Reason           string
	EngineVersion    string
	SignatureVersion string
	ScannedAt        time.Time
}

type Scanner interface {
	Scan(context.Context, io.Reader) (Result, error)
}

// DisabledScanner is the explicit fail-closed mode used when no content
// scanning service is available. It records that scanning was skipped and
// never promotes the object to canonical storage.
type DisabledScanner struct {
	Clock func() time.Time
}

func (scanner DisabledScanner) Scan(_ context.Context, _ io.Reader) (Result, error) {
	now := time.Now
	if scanner.Clock != nil {
		now = scanner.Clock
	}
	return Result{
		Clean:            false,
		Reason:           "content scanning is disabled",
		EngineVersion:    "disabled",
		SignatureVersion: "not-run",
		ScannedAt:        now().UTC(),
	}, nil
}

// ManagedResultProvider resolves a provider-owned decision for one exact
// immutable object version. It never accepts a bucket/key-only result.
type ManagedResultProvider interface {
	Resolve(context.Context, objectstore.ExactObject) (Result, error)
}

// SignatureScanner is deterministic and deliberately restricted to test
// profiles. Production configuration rejects this scanner mode.
type SignatureScanner struct {
	Clock func() time.Time
}

func (scanner SignatureScanner) Scan(_ context.Context, reader io.Reader) (Result, error) {
	body, err := io.ReadAll(reader)
	if err != nil {
		return Result{}, err
	}
	now := time.Now
	if scanner.Clock != nil {
		now = scanner.Clock
	}
	result := Result{
		Clean:            true,
		EngineVersion:    "deterministic-test",
		SignatureVersion: "eicar-v1",
		ScannedAt:        now().UTC(),
	}
	if bytes.Contains(body, []byte("EICAR-STANDARD-ANTIVIRUS-TEST-FILE")) {
		result.Clean = false
		result.Reason = "deterministic test signature detected"
	}
	return result, nil
}
