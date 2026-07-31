package datafeed

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	contractv3 "github.com/MarlonJD/aviaSurveil360/apps/api/internal/aviacorecontract/v3"
)

func TestBuildEventProducesLockedImmutableV3Envelope(t *testing.T) {
	now := time.Date(2026, time.August, 1, 8, 0, 1, 0, time.UTC)
	event, err := BuildEvent(EventInput{
		EventID:               "10000000-0000-4000-8000-000000000001",
		EventType:             "audit.planned",
		TenantID:              "tenant-contract-fixture",
		OwningOrganizationID:  "organization-fixture",
		ActorOrganizationID:   "caa-fixture",
		CorrelationID:         "20000000-0000-4000-8000-000000000001",
		AggregateType:         "audit",
		AggregateID:           "audit-ref-001",
		AggregateRevision:     1,
		EffectiveAt:           now.Add(-time.Second),
		KnownAt:               now,
		OccurredAt:            now.Add(-time.Second),
		EmittedAt:             now,
		VisibilityPurposeCode: "regulated_oversight",
		EntityRefs:            map[string]any{"audit_id": "audit-ref-001"},
		StateBefore:           nil,
		StateAfter:            "audit_planned",
		Payload: map[string]any{
			"audit_program_ref": "program-ref-001",
			"audit_scope_code":  "airport_operations",
			"planned_start_at":  "2026-08-01T08:00:00Z",
		},
	})
	if err != nil {
		t.Fatalf("build event: %v", err)
	}
	if errors := contractv3.ValidateEvent(event, "tenant-contract-fixture"); len(errors) != 0 {
		t.Fatalf("locked contract rejected event: %+v", errors)
	}
	if got, want := event["contract_version"], "3.0.0"; got != want {
		t.Fatalf("contract version = %v, want %q", got, want)
	}
	if got, want := event["payload_sha256"], "47c9a236adb976eb229d74bc447c6856ec60b8fb1576a6c1dda16afd02a0e926"; got != want {
		t.Fatalf("payload digest = %v, want %q", got, want)
	}
	first, err := CanonicalJSON(event)
	if err != nil {
		t.Fatal(err)
	}
	second, err := CanonicalJSON(event)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("canonical event bytes were not stable")
	}
	sum := sha256.Sum256(first)
	if got, want := CanonicalDigest(event), hex.EncodeToString(sum[:]); got != want {
		t.Fatalf("canonical digest = %q, want %q", got, want)
	}
}

func TestBuildEventFailsClosedOnUnapprovedPayloadAndTenantlessIdentity(t *testing.T) {
	_, err := BuildEvent(EventInput{
		EventType: "audit.planned",
		Payload:   map[string]any{"forbidden": "value"},
	})
	if err == nil {
		t.Fatal("invalid event was accepted")
	}
}

func TestEncryptPayloadKeepsPlaintextOutOfStoredCopy(t *testing.T) {
	plain, err := json.Marshal(map[string]any{"audit_scope_code": "airport_operations"})
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := EncryptPayload(testEncryptionKey, plain)
	if err != nil {
		t.Fatal(err)
	}
	if string(sealed.Ciphertext) == string(plain) {
		t.Fatal("payload stored as plaintext")
	}
	opened, err := DecryptPayload(testEncryptionKey, sealed)
	if err != nil {
		t.Fatal(err)
	}
	if string(opened) != string(plain) {
		t.Fatalf("decrypted payload = %s, want %s", opened, plain)
	}
}

func TestNewWriterFailsClosedWithoutDedicatedPayloadKeyAndTenant(t *testing.T) {
	if _, err := NewWriter(WriterConfig{TenantID: "tenant-a", PayloadKey: []byte("short")}); err == nil {
		t.Fatal("short payload key was accepted")
	}
	if _, err := NewWriter(WriterConfig{PayloadKey: testEncryptionKey}); err == nil {
		t.Fatal("missing platform tenant was accepted")
	}
}

var testEncryptionKey = []byte("0123456789abcdef0123456789abcdef")
