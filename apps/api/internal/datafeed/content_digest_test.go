package datafeed

import "testing"

func TestEventContentDigestUsesTheLockedV3TypedHashProjection(t *testing.T) {
	event := map[string]any{
		"contract_id": "aviasurveil.production-feed.events", "contract_version": "3.0.0",
		"event_id": "10000000-0000-4000-8000-000000000001", "event_type": "audit.planned", "event_version": float64(1),
		"occurred_at": "2026-08-01T08:00:00Z", "emitted_at": "2026-08-01T08:00:01Z", "effective_at": "2026-08-01T08:00:00Z", "known_at": "2026-08-01T08:00:01Z",
		"source_module": "aviasurveil360", "source_system": "aviasurveil-production-api", "tenant_id": "tenant-contract-fixture", "owning_organization_id": "organization-fixture", "actor_organization_id": "caa-fixture", "visibility_purpose_code": "regulated_oversight",
		"business_key": "audit-ref-001", "correlation_id": "20000000-0000-4000-8000-000000000001", "causation_id": nil,
		"aggregate_type": "audit", "aggregate_id": "audit-ref-001", "aggregate_revision": float64(1), "entity_refs": map[string]any{"audit_id": "audit-ref-001"}, "state_before": nil, "state_after": "audit_planned", "privacy_class": "P2",
		"payload": map[string]any{"audit_scope_code": "airport_operations", "audit_program_ref": "program-ref-001", "planned_start_at": "2026-08-01T08:00:00Z"}, "payload_sha256": "47c9a236adb976eb229d74bc447c6856ec60b8fb1576a6c1dda16afd02a0e926",
	}
	got, err := EventContentDigest(event)
	if err != nil {
		t.Fatalf("calculate v3 event content digest: %v", err)
	}
	if want := "78472d253db32ee87271359dd61df33049f8bbecba700687f0c6c2b3b49b2e8d"; got != want {
		t.Fatalf("event content digest = %q, want locked AviaCore v3 reference %q", got, want)
	}
	changed := map[string]any{}
	for key, value := range event {
		changed[key] = value
	}
	changed["payload"] = map[string]any{"audit_scope_code": "changed", "audit_program_ref": "program-ref-001", "planned_start_at": "2026-08-01T08:00:00Z"}
	other, err := EventContentDigest(changed)
	if err != nil {
		t.Fatalf("calculate changed v3 event content digest: %v", err)
	}
	if got == other {
		t.Fatal("event content digest did not bind the locked payload projection")
	}
}
