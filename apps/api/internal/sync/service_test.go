package fieldsync

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDecodeAndValidateConflictResolutionOperationEnvelope(t *testing.T) {
	raw := json.RawMessage(`{
      "operationId":"OP-CONFLICT-RESOLUTION-001",
      "protocolVersion":1,
      "offlineGrantId":"GRANT-001",
      "packageId":"PKG-001",
      "packageVersion":1,
      "packageRevision":3,
      "entityId":"RESP-001",
      "commandType":"RESOLVE_FIELD_CONFLICT",
      "baseRevision":4,
      "deviceInstanceId":"DEVICE-001",
      "actorSubject":"SUBJECT-001",
      "operationSequence":9,
      "payloadHash":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      "requestHash":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
      "profileKeyId":"sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
      "authorityProof":"candidate-proof",
      "dependencies":[],
      "clientOccurredAt":"2026-08-17T12:00:00Z",
      "payload":{
        "conflictOperationId":"OP-CONFLICT-001",
        "resolution":"KEEP_LOCAL_AS_NEW_REVISION",
        "reason":"Reviewed authoritative revision.",
        "authoritativeRevision":4,
        "localPayloadHash":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
      }
    }`)
	operation, err := decodeOperation(raw)
	if err != nil {
		t.Fatalf("decode conflict resolution: %v", err)
	}
	if err := validateOperation(operation); err != nil {
		t.Fatalf("validate conflict resolution: %v", err)
	}
	if operation.CommandType != "RESOLVE_FIELD_CONFLICT" || operation.OperationSequence != 9 {
		t.Fatalf("decoded operation = %+v", operation)
	}
}

func TestValidateOperationRejectsIncompleteAtLeastOnceEnvelope(t *testing.T) {
	operation := operation{
		OperationID:       "OP-INCOMPLETE",
		ProtocolVersion:   1,
		OfflineGrantID:    "GRANT-001",
		PackageID:         "PKG-001",
		PackageVersion:    1,
		PackageRevision:   1,
		EntityID:          "RESP-001",
		DeviceInstanceID:  "DEVICE-001",
		ActorSubject:      "SUBJECT-001",
		OperationSequence: 1,
		ClientOccurredAt:  "2026-08-17T12:00:00Z",
	}
	if err := validateOperation(operation); err == nil {
		t.Fatal("incomplete payload/request hashes must be rejected")
	}
}

func TestOperationRequestHashBindsTheProfileAndPayloadEnvelope(t *testing.T) {
	baseRevision := int64(4)
	input := operation{
		OperationID: "OP-HASH-001", ProtocolVersion: 1, OfflineGrantID: "GRANT-001",
		PackageID: "PKG-001", PackageVersion: 1, PackageRevision: 2, EntityID: "RESP-001",
		CommandType: "UPSERT_CHECKLIST_RESPONSE", BaseRevision: &baseRevision,
		DeviceInstanceID: "DEVICE-001", ActorSubject: "SUBJECT-001", OperationSequence: 2,
		PayloadHash:  "sha256:" + strings.Repeat("a", 64),
		ProfileKeyID: "sha256:" + strings.Repeat("d", 64),
		Dependencies: []string{"OP-DEPENDENCY-001"}, Payload: map[string]any{"answer": "OBSERVATION"},
	}
	first, err := operationRequestHash(input)
	if err != nil {
		t.Fatalf("request hash: %v", err)
	}
	input.Payload = map[string]any{"answer": "COMPLIANT"}
	second, err := operationRequestHash(input)
	if err != nil {
		t.Fatalf("changed request hash: %v", err)
	}
	if first == second {
		t.Fatal("request hash must change when the signed payload changes")
	}
}
