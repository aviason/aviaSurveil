package audit

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestAuditEventsAreAppendOnlyAndRedacted(t *testing.T) {
	store, err := NewMemoryStore(2, func() time.Time { return time.Date(2026, 8, 11, 15, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	event := Event{Type: EventLoginFailure, Outcome: "denied", SubjectID: "usr_0123456789012345678901", Fields: map[string]string{"password": "raw", "refresh_token": "raw-token", "ip": "192.0.2.1"}}
	if err := store.Append(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	snapshot := store.Snapshot()
	if len(snapshot) != 1 || snapshot[0].Fields["password"] != "[REDACTED]" || snapshot[0].Fields["refresh_token"] != "[REDACTED]" || snapshot[0].Fields["ip"] != "192.0.2.1" {
		t.Fatalf("audit snapshot = %+v", snapshot)
	}
	snapshot[0].Fields["ip"] = "mutated"
	if store.Snapshot()[0].Fields["ip"] == "mutated" {
		t.Fatal("audit snapshot mutation changed stored event")
	}
	if err := store.Append(context.Background(), Event{Type: EventLogout, Outcome: "success", SubjectID: "usr_0123456789012345678901"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Append(context.Background(), Event{Type: EventRefresh, Outcome: "success", SubjectID: "usr_0123456789012345678901"}); !errors.Is(err, ErrAuditUnavailable) {
		t.Fatalf("full audit store append = %v", err)
	}
	for _, event := range store.Snapshot() {
		for key, value := range event.Fields {
			if strings.Contains(strings.ToLower(key), "token") && value != "[REDACTED]" {
				t.Fatalf("sensitive audit field %s was not redacted", key)
			}
		}
	}
}

func TestAuditRejectsInvalidSubjectAndOutcome(t *testing.T) {
	store, err := NewMemoryStore(1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Append(context.Background(), Event{Type: EventSession, Outcome: "success", SubjectID: "email@example.invalid"}); !errors.Is(err, ErrAuditInvalid) {
		t.Fatalf("invalid subject = %v", err)
	}
	if err := store.Append(context.Background(), Event{Type: EventSession, SubjectID: "usr_0123456789012345678901"}); !errors.Is(err, ErrAuditInvalid) {
		t.Fatalf("missing outcome = %v", err)
	}
}
