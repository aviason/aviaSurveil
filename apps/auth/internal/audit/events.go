package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"
)

var (
	ErrAuditUnavailable = errors.New("audit store unavailable")
	ErrAuditInvalid     = errors.New("audit event is invalid")
)

type EventType string

const (
	EventInvite         EventType = "invite"
	EventVerification   EventType = "verification"
	EventLoginSuccess   EventType = "login-success"
	EventLoginFailure   EventType = "login-failure"
	EventRefresh        EventType = "refresh"
	EventRefreshReuse   EventType = "refresh-reuse"
	EventLogout         EventType = "logout"
	EventPasswordChange EventType = "password-change"
	EventPasswordReset  EventType = "password-reset"
	EventMFAEnrollment  EventType = "mfa-enrollment"
	EventMFARecovery    EventType = "mfa-recovery"
	EventLock           EventType = "lock"
	EventLifecycle      EventType = "lifecycle"
	EventSession        EventType = "session"
	EventClientAdmin    EventType = "client-admin"
	EventKeyAdmin       EventType = "key-admin"
	EventAdminRecovery  EventType = "admin-recovery"
)

type Event struct {
	ID        string
	At        time.Time
	Type      EventType
	Outcome   string
	SubjectID string
	ActorID   string
	ClientID  string
	RequestID string
	Fields    map[string]string
}

type Store interface {
	Append(context.Context, Event) error
}

type MemoryStore struct {
	mu       sync.Mutex
	clock    func() time.Time
	capacity int
	events   []Event
}

func NewMemoryStore(capacity int, clock func() time.Time) (*MemoryStore, error) {
	if capacity < 1 || capacity > 100000 {
		return nil, errors.New("audit capacity is invalid")
	}
	if clock == nil {
		clock = time.Now
	}
	return &MemoryStore{clock: clock, capacity: capacity, events: make([]Event, 0, capacity)}, nil
}

func (store *MemoryStore) Append(_ context.Context, event Event) error {
	if event.Type == "" || strings.TrimSpace(event.Outcome) == "" {
		return ErrAuditInvalid
	}
	if event.SubjectID != "" && !validOpaqueSubject(event.SubjectID) {
		return ErrAuditInvalid
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.events) >= store.capacity {
		return ErrAuditUnavailable
	}
	if event.ID == "" {
		event.ID = digestID(event)
	}
	if event.At.IsZero() {
		event.At = store.clock().UTC()
	}
	event.Fields = redactFields(event.Fields)
	store.events = append(store.events, cloneEvent(event))
	return nil
}

func (store *MemoryStore) Snapshot() []Event {
	store.mu.Lock()
	defer store.mu.Unlock()
	result := make([]Event, len(store.events))
	for index, event := range store.events {
		result[index] = cloneEvent(event)
	}
	return result
}

func validOpaqueSubject(subject string) bool {
	return len(subject) == len("usr_")+22 && strings.HasPrefix(subject, "usr_")
}

func redactFields(fields map[string]string) map[string]string {
	result := make(map[string]string, len(fields))
	for key, value := range fields {
		lower := strings.ToLower(key)
		if containsSensitiveName(lower) {
			result[key] = "[REDACTED]"
			continue
		}
		result[key] = value
	}
	return result
}

func containsSensitiveName(value string) bool {
	for _, sensitive := range []string{"password", "secret", "token", "private", "signing", "mfa", "recovery", "cookie", "authorization", "code"} {
		if strings.Contains(value, sensitive) {
			return true
		}
	}
	return false
}

func cloneEvent(event Event) Event {
	clone := event
	clone.Fields = make(map[string]string, len(event.Fields))
	for key, value := range event.Fields {
		clone.Fields[key] = value
	}
	return clone
}

func digestID(event Event) string {
	value := string(event.Type) + "\x00" + event.SubjectID + "\x00" + event.ActorID + "\x00" + event.ClientID + "\x00" + event.RequestID + "\x00" + event.Outcome + "\x00" + event.At.UTC().Format(time.RFC3339Nano)
	digest := sha256.Sum256([]byte(value))
	return "evt_" + hex.EncodeToString(digest[:])[:32]
}
