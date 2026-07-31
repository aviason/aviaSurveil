package v3

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
)

func contractRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate contract test")
	}
	return filepath.Join(filepath.Dir(file), "../../../../..", "integrations/aviacore/contracts")
}

func readObject(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func clone(t *testing.T, value map[string]any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func has(errors []ValidationError, code, pointer string) bool {
	for _, item := range errors {
		if item.Code == code && item.Pointer == pointer {
			return true
		}
	}
	return false
}

func TestLockedV3PositiveAndNegativeVectors(t *testing.T) {
	root := contractRoot(t)
	positive, err := os.ReadDir(filepath.Join(root, "contracts/aviasurveil-production/v3/golden/positive"))
	if err != nil {
		t.Fatal(err)
	}
	if len(positive) != 39 {
		t.Fatalf("positive vectors = %d, want 39", len(positive))
	}
	for _, entry := range positive {
		event := readObject(t, filepath.Join(root, "contracts/aviasurveil-production/v3/golden/positive", entry.Name()))
		if errors := ValidateEvent(event, event["tenant_id"].(string)); len(errors) != 0 {
			t.Fatalf("positive %s: %+v", entry.Name(), errors)
		}
		negative := readObject(t, filepath.Join(root, "contracts/aviasurveil-production/v3/golden/negative", entry.Name()))
		invalid, _ := negative["event"].(map[string]any)
		if !has(ValidateEvent(invalid, invalid["tenant_id"].(string)), "SCHEMA_ADDITIONAL_PROPERTY", "/payload/forbidden_extra") {
			t.Fatalf("negative %s was accepted", entry.Name())
		}
	}
}

func TestLockedV3BranchMatrix(t *testing.T) {
	root := contractRoot(t)
	matrix := readObject(t, filepath.Join(root, "contracts/aviasurveil-production/v3/negative-branch-matrix.json"))
	cases, _ := matrix["cases"].([]any)
	if len(cases) != 11 {
		t.Fatalf("branch cases = %d, want 11", len(cases))
	}
	for _, rawCase := range cases {
		branch := rawCase.(map[string]any)
		code, pointer := branch["expected_error_code"].(string), branch["expected_json_pointer"].(string)
		var errors []ValidationError
		switch branch["kind"] {
		case "event":
			event := readObject(t, filepath.Join(root, "contracts/aviasurveil-production/v3/golden/positive", branch["source_vector"].(string)))
			payload := event["payload"].(map[string]any)
			switch branch["mutation"] {
			case "unknown_event_type":
				event["event_type"] = "unsupported.fixture"
			case "missing_required_payload_field":
				delete(payload, branch["field"].(string))
			case "extra_payload_field":
				payload[branch["field"].(string)] = "forbidden"
				event["payload_sha256"] = digestPayload(t, payload)
			case "payload_digest_mismatch":
				event["payload_sha256"] = "0000000000000000000000000000000000000000000000000000000000000000"
			case "tenant_identity_mismatch":
				errors = ValidateEvent(event, "another-tenant")
			case "timestamp_order":
				event["known_at"] = "2026-07-28T00:00:00Z"
			case "privacy_forbidden":
				event["privacy_class"] = "P3"
			case "correction_self_reference":
				payload["corrected_event_id"] = event["event_id"]
				event["payload_sha256"] = digestPayload(t, payload)
			case "supersession_same_reference":
				payload["replacement_event_id"] = payload["superseded_event_id"]
				event["payload_sha256"] = digestPayload(t, payload)
			}
			if errors == nil {
				errors = ValidateEvent(event, event["tenant_id"].(string))
			}
		case "recovery_contract":
			recovery := readObject(t, filepath.Join(root, "contracts/aviasurveil-production/v3/recovery-contract.json"))
			delete(recovery["bootstrap"].(map[string]any), branch["field"].(string))
			errors = ValidateRecoveryContract(recovery)
		case "contract_set":
			contractSet := readObject(t, filepath.Join(root, "contracts/aviasurveil-production/v3/contract-set.json"))
			contractSet["version_compatibility"].(map[string]any)[branch["field"].(string)] = "allowed"
			errors = ValidateVersionCompatibility(contractSet)
		default:
			t.Fatalf("unsupported branch kind %v", branch["kind"])
		}
		if !has(errors, code, pointer) {
			t.Fatalf("branch %s did not fail as %s %s: %+v", branch["id"], code, pointer, errors)
		}
	}
}

func TestGeneratedEventTypesMatchLockedCatalog(t *testing.T) {
	root := contractRoot(t)
	catalog := readObject(t, filepath.Join(root, "contracts/aviasurveil-production/v3/event-catalog.json"))
	events, _ := catalog["events"].([]any)
	actual := make([]string, 0, len(GeneratedRequiredPayloadFields))
	for event := range GeneratedRequiredPayloadFields {
		actual = append(actual, event)
	}
	sort.Strings(actual)
	expected := make([]string, 0, len(events))
	for _, raw := range events {
		expected = append(expected, raw.(map[string]any)["event_type"].(string))
	}
	sort.Strings(expected)
	if stringSlice := len(actual) == len(expected); !stringSlice {
		t.Fatalf("generated event types = %d, catalog = %d", len(actual), len(expected))
	}
	for index := range actual {
		if actual[index] != expected[index] {
			t.Fatalf("generated event type %q, want %q", actual[index], expected[index])
		}
	}
}

func TestLockedSchemasRejectConstTypeEntityAndStateViolations(t *testing.T) {
	root := contractRoot(t)
	audit := readObject(t, filepath.Join(root, "contracts/aviasurveil-production/v3/golden/positive/audit-planned.json"))
	invalidContract := clone(t, audit)
	invalidContract["contract_id"] = "wrong.contract"
	if !has(ValidateEvent(invalidContract, audit["tenant_id"].(string)), "SCHEMA_CONST", "/contract_id") {
		t.Fatal("locked contract_id const was not enforced")
	}
	invalidEntity := clone(t, audit)
	entityRefs := invalidEntity["entity_refs"].(map[string]any)
	for field := range entityRefs {
		delete(entityRefs, field)
		break
	}
	if !has(ValidateEvent(invalidEntity, audit["tenant_id"].(string)), "SCHEMA_REQUIRED", "/entity_refs/audit_id") {
		t.Fatal("locked entity-ref requirement was not enforced")
	}
	invalidState := clone(t, audit)
	invalidState["state_after"] = "not_a_valid_audit_state"
	if !hasPointer(ValidateEvent(invalidState, audit["tenant_id"].(string)), "/state_after") {
		t.Fatal("locked state constraint was not enforced")
	}
	candidate := readObject(t, filepath.Join(root, "contracts/aviasurveil-production/v3/golden/positive/checklist_generation-candidate_versioned.json"))
	invalidType := clone(t, candidate)
	payload := invalidType["payload"].(map[string]any)
	payload["question_count"] = "one"
	invalidType["payload_sha256"] = digestPayload(t, payload)
	if !has(ValidateEvent(invalidType, candidate["tenant_id"].(string)), "SCHEMA_TYPE", "/payload/question_count") {
		t.Fatal("locked payload type was not enforced")
	}
}

func hasPointer(errors []ValidationError, pointer string) bool {
	for _, item := range errors {
		if item.Pointer == pointer {
			return true
		}
	}
	return false
}

func digestPayload(t *testing.T, payload map[string]any) string {
	t.Helper()
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return sha256Hex(encoded)
}
