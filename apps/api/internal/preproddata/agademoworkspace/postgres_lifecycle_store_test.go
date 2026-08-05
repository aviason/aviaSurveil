package agademoworkspace

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	aga "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agaapplicability"
)

func TestPostgresLifecycleStoreRequiresCommandStore(t *testing.T) {
	_, err := (&PostgresStore{}).AppendLifecycleEvent(context.Background(), LifecycleEvent{LifecycleID: "aga-ws-lifecycle-0001"})
	if err == nil || !strings.Contains(err.Error(), "command store") {
		t.Fatalf("append without command store error = %v", err)
	}
	if !strings.Contains(err.Error(), "workspace lifecycle requires command store") {
		t.Fatalf("append without command store did not fail at the command boundary: %v", err)
	}
}

func TestPostgresLifecycleStoreUsesAppendOnlyEvents(t *testing.T) {
	ddl := WorkspaceAppendOnlyTriggerDDL()
	for _, required := range []string{
		"lifecycle_events_append_only",
		"preprod_aga_demo_workspace.lifecycle_events",
	} {
		if !strings.Contains(ddl, required) {
			t.Fatalf("append-only DDL missing %q", required)
		}
	}
}

func TestPostgresLifecycleStoreDoesNotExposeDirectTableDML(t *testing.T) {
	for _, grant := range WorkspaceRoleMatrix() {
		if grant.Role == WorkspaceCommandRole && grant.DirectTableDML {
			t.Fatal("workspace command role received direct table DML")
		}
	}
}

func TestMemoryLifecycleEventsAreAppendOnly(t *testing.T) {
	store := NewMemoryStore()
	store.loadedOnce = true
	store.currentGeneration = "aga-ws-generation-0001"
	payload := json.RawMessage(`{"generationId":"aga-ws-generation-0001"}`)
	makeEvent := func(sequence int, previous string) LifecycleEvent {
		event := LifecycleEvent{EventID: "event-" + string(rune('0'+sequence)), LifecycleID: "aga-ws-lifecycle-0001", Sequence: sequence, OperationID: "OPERATION", CommandKey: "command-" + string(rune('0'+sequence)), EventType: "OPERATION", Payload: payload, ActorSubjectID: "subject", CreatedAt: time.Unix(int64(sequence), 0).UTC(), PreviousDigest: previous}
		event.EventDigest, _ = aga.DigestExcludingJSONFields("AGA-DEMO-LIFECYCLE-EVENT-V1", event, "eventDigest")
		return event
	}
	first, err := store.AppendLifecycleEvent(context.Background(), makeEvent(1, ""))
	if err != nil {
		t.Fatalf("first lifecycle event: %v", err)
	}
	second, err := store.AppendLifecycleEvent(context.Background(), makeEvent(2, first.EventDigest))
	if err != nil {
		t.Fatalf("second lifecycle event: %v", err)
	}
	if _, err := store.AppendLifecycleEvent(context.Background(), LifecycleEvent{EventID: "tampered", LifecycleID: first.LifecycleID, Sequence: 3, OperationID: "OPERATION", CommandKey: "command-3", EventType: "OPERATION", Payload: payload, ActorSubjectID: "subject", CreatedAt: time.Unix(3, 0).UTC(), PreviousDigest: second.EventDigest, EventDigest: "sha256:0000000000000000000000000000000000000000000000000000000000000000"}); err == nil || !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("tampered lifecycle event error = %v", err)
	}
	events, err := store.GetLifecycleEvents(context.Background(), store.currentGeneration, first.LifecycleID)
	if err != nil || len(events) != 2 {
		t.Fatalf("lifecycle event history err=%v count=%d", err, len(events))
	}
}
