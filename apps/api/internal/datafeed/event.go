// Package datafeed owns producer-side immutable AviaCore event construction.
// It deliberately has no AviaCore runtime dependency: its only contract input
// is the locally locked, generated v3 validator.
package datafeed

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"time"

	contractv3 "github.com/aviason/aviaSurveil/internal/aviacorecontract/v3"
)

const (
	contractID      = "aviasurveil.production-feed.events"
	contractVersion = "3.0.0"
	sourceModule    = "aviasurveil360"
	sourceSystem    = "aviasurveil-production-api"
)

// EventInput is intentionally server-owned. Command/request fields must not
// be passed through into this envelope without reconstructing authoritative
// post-transition state first.
type EventInput struct {
	EventID               string
	EventType             string
	TenantID              string
	OwningOrganizationID  string
	ActorOrganizationID   string
	CorrelationID         string
	CausationID           any
	AggregateType         string
	AggregateID           string
	AggregateRevision     int64
	EffectiveAt           time.Time
	KnownAt               time.Time
	OccurredAt            time.Time
	EmittedAt             time.Time
	VisibilityPurposeCode string
	EntityRefs            map[string]any
	StateBefore           any
	StateAfter            string
	Payload               map[string]any
}

// BuildEvent constructs exactly one v3 event and rejects anything not allowed
// by the locally locked v3 schema before a durable row can be created.
func BuildEvent(input EventInput) (map[string]any, error) {
	if input.EventID == "" {
		generated, err := NewEventID()
		if err != nil {
			return nil, err
		}
		input.EventID = generated
	}
	payloadBytes, err := json.Marshal(input.Payload)
	if err != nil {
		return nil, fmt.Errorf("encode event payload: %w", err)
	}
	payloadHash := sha256.Sum256(payloadBytes)
	event := map[string]any{
		"event_id":                input.EventID,
		"event_type":              input.EventType,
		"event_version":           1,
		"contract_id":             contractID,
		"contract_version":        contractVersion,
		"schema_version":          contractVersion,
		"source_module":           sourceModule,
		"source_system":           sourceSystem,
		"tenant_id":               input.TenantID,
		"owning_organization_id":  input.OwningOrganizationID,
		"actor_organization_id":   nullIfEmpty(input.ActorOrganizationID),
		"visibility_purpose_code": input.VisibilityPurposeCode,
		"correlation_id":          input.CorrelationID,
		"causation_id":            input.CausationID,
		"aggregate_type":          input.AggregateType,
		"aggregate_id":            input.AggregateID,
		"aggregate_revision":      input.AggregateRevision,
		"business_key":            input.AggregateID,
		"effective_at":            input.EffectiveAt.UTC().Format(time.RFC3339Nano),
		"known_at":                input.KnownAt.UTC().Format(time.RFC3339Nano),
		"occurred_at":             input.OccurredAt.UTC().Format(time.RFC3339Nano),
		"emitted_at":              input.EmittedAt.UTC().Format(time.RFC3339Nano),
		"entity_refs":             input.EntityRefs,
		"state_before":            input.StateBefore,
		"state_after":             input.StateAfter,
		"payload":                 input.Payload,
		"payload_sha256":          hex.EncodeToString(payloadHash[:]),
		"privacy_class":           "P2",
	}
	// The generated schema validator operates over decoded JSON values. Round
	// trip here so integer constraints are evaluated exactly as they will be by
	// the HTTP producer/publisher boundary.
	raw, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("encode candidate event: %w", err)
	}
	var normalized map[string]any
	if err := json.Unmarshal(raw, &normalized); err != nil {
		return nil, fmt.Errorf("normalize candidate event: %w", err)
	}
	if errors := contractv3.ValidateEvent(normalized, input.TenantID); len(errors) != 0 {
		return nil, fmt.Errorf("locked v3 contract rejected event: %v", errors)
	}
	return normalized, nil
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// CanonicalJSON is the producer's exact JSON representation used for its
// content binding. encoding/json sorts map keys deterministically.
func CanonicalJSON(event map[string]any) ([]byte, error) {
	return json.Marshal(event)
}

func CanonicalDigest(event map[string]any) string {
	raw, err := CanonicalJSON(event)
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

// NewEventID creates a RFC4122 version-four UUID owned by this producer.
func NewEventID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return "", fmt.Errorf("allocate event UUID: %w", err)
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16]), nil
}

type EncryptedPayload struct {
	Nonce      []byte
	Ciphertext []byte
}

func EncryptPayload(key, plain []byte) (EncryptedPayload, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return EncryptedPayload{}, fmt.Errorf("create payload cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return EncryptedPayload{}, fmt.Errorf("create payload envelope: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return EncryptedPayload{}, fmt.Errorf("allocate payload nonce: %w", err)
	}
	return EncryptedPayload{Nonce: nonce, Ciphertext: gcm.Seal(nil, nonce, plain, nil)}, nil
}

func DecryptPayload(key []byte, sealed EncryptedPayload) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create payload cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create payload envelope: %w", err)
	}
	plain, err := gcm.Open(nil, sealed.Nonce, sealed.Ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt payload: %w", err)
	}
	return plain, nil
}
