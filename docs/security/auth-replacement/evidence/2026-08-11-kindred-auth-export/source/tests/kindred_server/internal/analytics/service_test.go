package analytics

import (
	"context"
	"errors"
	"testing"
	"time"

	apperrors "kindred_server/internal/platform/errors"
)

const (
	testUserID    = "user-1"
	testEventID   = "018f05b1-bd66-7d2f-bb14-f0d393d1a001"
	testSessionID = "018f05b1-bd66-7d2f-bb14-f0d393d1a002"
)

func TestIngestPublishesEnrichedEvents(t *testing.T) {
	repo := newMemoryConsentRepo()
	publisher := &memoryPublisher{}
	svc := newTestService(repo, publisher)
	_, err := svc.UpdateConsents(context.Background(), testUserID, ConsentUpdateRequest{Consents: map[string]bool{
		string(PurposeAnalytics):       true,
		string(PurposePersonalization): true,
	}})
	if err != nil {
		t.Fatalf("update consents: %v", err)
	}

	out, err := svc.Ingest(context.Background(), testUserID, BatchRequest{Events: []EventEnvelope{
		validEvent(EventItemViewed, PurposeAnalytics, PurposePersonalization),
	}})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if out.Accepted != 1 || out.Rejected != 0 {
		t.Fatalf("response = %+v", out)
	}
	if len(publisher.events) != 1 {
		t.Fatalf("published len = %d", len(publisher.events))
	}
	event := publisher.events[0]
	if event.UserID != testUserID {
		t.Errorf("UserID = %q, want authenticated user", event.UserID)
	}
	if event.DeviceIDHash == "" || event.DeviceIDHash == "install-id" {
		t.Errorf("DeviceIDHash = %q, want hashed anonymousId", event.DeviceIDHash)
	}
	if event.AnonymousID != "" {
		t.Errorf("AnonymousID = %q, want stripped from published event", event.AnonymousID)
	}
	if event.Environment != "test" || event.RecordSource != "test-api" {
		t.Errorf("enrichment = env %q source %q", event.Environment, event.RecordSource)
	}
	if event.ConsentVersion != 1 {
		t.Errorf("ConsentVersion = %d, want 1", event.ConsentVersion)
	}
}

func TestIngestRejectsUnsupportedEventAndMissingConsent(t *testing.T) {
	repo := newMemoryConsentRepo()
	publisher := &memoryPublisher{}
	svc := newTestService(repo, publisher)
	if _, err := svc.UpdateConsents(context.Background(), testUserID, ConsentUpdateRequest{Consents: map[string]bool{
		string(PurposeAnalytics): true,
	}}); err != nil {
		t.Fatal(err)
	}

	out, err := svc.Ingest(context.Background(), testUserID, BatchRequest{Events: []EventEnvelope{
		validEvent(EventName("unknown_event"), PurposeAnalytics),
		validEvent(EventLocationObserved, PurposeAnalytics, PurposePreciseLocation),
		validEvent(EventScreenViewed, PurposeAnalytics),
	}})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if out.Accepted != 1 || out.Rejected != 2 {
		t.Fatalf("response = %+v, want 1 accepted / 2 rejected", out)
	}
	if len(publisher.events) != 1 || publisher.events[0].EventName != EventScreenViewed {
		t.Fatalf("published = %#v", publisher.events)
	}
}

func TestIngestRejectsSensitiveMessageFields(t *testing.T) {
	repo := newMemoryConsentRepo()
	publisher := &memoryPublisher{}
	svc := newTestService(repo, publisher)
	if _, err := svc.UpdateConsents(context.Background(), testUserID, ConsentUpdateRequest{Consents: map[string]bool{
		string(PurposeMessagingMetadata): true,
	}}); err != nil {
		t.Fatal(err)
	}
	event := validEvent(EventMessageSentMetadata, PurposeMessagingMetadata)
	event.Properties = map[string]any{"ciphertext": "must-not-copy"}

	out, err := svc.Ingest(context.Background(), testUserID, BatchRequest{Events: []EventEnvelope{event}})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if out.Accepted != 0 || out.Rejected != 1 {
		t.Fatalf("response = %+v, want rejected", out)
	}
	if len(publisher.events) != 0 {
		t.Fatalf("sensitive event was published: %#v", publisher.events)
	}
}

func TestIngestPublisherFailure(t *testing.T) {
	repo := newMemoryConsentRepo()
	publisher := &memoryPublisher{err: errors.New("firehose down")}
	svc := newTestService(repo, publisher)
	if _, err := svc.UpdateConsents(context.Background(), testUserID, ConsentUpdateRequest{Consents: map[string]bool{
		string(PurposeAnalytics): true,
	}}); err != nil {
		t.Fatal(err)
	}

	_, err := svc.Ingest(context.Background(), testUserID, BatchRequest{Events: []EventEnvelope{
		validEvent(EventScreenViewed, PurposeAnalytics),
	}})
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.Status != 500 {
		t.Fatalf("err = %v, want internal app error", err)
	}
}

func TestConsentLedgerVersionsAndCurrentState(t *testing.T) {
	repo := newMemoryConsentRepo()
	svc := newTestService(repo, &memoryPublisher{})
	if _, err := svc.UpdateConsents(context.Background(), testUserID, ConsentUpdateRequest{Consents: map[string]bool{
		string(PurposeAnalytics): true,
	}}); err != nil {
		t.Fatal(err)
	}
	resp, err := svc.UpdateConsents(context.Background(), testUserID, ConsentUpdateRequest{Consents: map[string]bool{
		string(PurposeAnalytics): false,
		string(PurposeMarketing): true,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Consents[PurposeAnalytics].Granted || resp.Consents[PurposeAnalytics].Version != 2 {
		t.Fatalf("analytics consent = %+v, want false v2", resp.Consents[PurposeAnalytics])
	}
	if !resp.Consents[PurposeMarketing].Granted || resp.Consents[PurposeMarketing].Version != 1 {
		t.Fatalf("marketing consent = %+v, want true v1", resp.Consents[PurposeMarketing])
	}
	if len(repo.ledger) != 3 {
		t.Fatalf("ledger len = %d, want 3", len(repo.ledger))
	}
}

func TestRecordUserDataDeletionPublishesDeletionRecord(t *testing.T) {
	publisher := &memoryPublisher{}
	svc := newTestService(newMemoryConsentRepo(), publisher)
	requestedAt := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	deletedAt := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	if err := svc.RecordUserDataDeletion(context.Background(), testUserID, requestedAt, deletedAt); err != nil {
		t.Fatalf("RecordUserDataDeletion error = %v", err)
	}

	if len(publisher.deletions) != 1 {
		t.Fatalf("deletions len = %d, want 1", len(publisher.deletions))
	}
	record := publisher.deletions[0]
	if record.RecordType != "user_data_deletion" || record.UserID != testUserID || record.DeleteCategory != "account_delete" {
		t.Fatalf("deletion record = %+v", record)
	}
	if !record.RequestedAt.Equal(requestedAt) || !record.DeletedAt.Equal(deletedAt) {
		t.Fatalf("deletion dates = requested %s deleted %s", record.RequestedAt, record.DeletedAt)
	}
	if record.Environment != "test" || record.RecordSource != "test-api" {
		t.Fatalf("deletion metadata = %+v", record)
	}
}

func TestIngestRejectsOversizeBatch(t *testing.T) {
	svc := newTestService(newMemoryConsentRepo(), &memoryPublisher{})
	events := make([]EventEnvelope, 101)
	_, err := svc.Ingest(context.Background(), testUserID, BatchRequest{Events: events})
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.Status != 400 {
		t.Fatalf("err = %v, want bad request", err)
	}
}

func newTestService(repo *memoryConsentRepo, publisher *memoryPublisher) *Service {
	svc := NewService(repo, publisher, Config{
		Enabled:       true,
		SchemaVersion: 1,
		Environment:   "test",
		RecordSource:  "test-api",
	})
	svc.now = func() time.Time { return time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC) }
	return svc
}

func validEvent(name EventName, purposes ...Purpose) EventEnvelope {
	return EventEnvelope{
		EventID:       testEventID,
		EventName:     name,
		SchemaVersion: 1,
		EventTime:     time.Date(2026, 5, 4, 11, 59, 0, 0, time.UTC),
		SessionID:     testSessionID,
		AnonymousID:   "install-id",
		Source:        "ios",
		AppVersion:    "1.0.0",
		Screen:        "home",
		Purposes:      purposes,
		Properties:    map[string]any{"itemId": "item-1"},
	}
}

type memoryConsentRepo struct {
	current map[string]map[Purpose]ConsentRecord
	ledger  []ConsentRecord
}

func newMemoryConsentRepo() *memoryConsentRepo {
	return &memoryConsentRepo{current: map[string]map[Purpose]ConsentRecord{}}
}

func (r *memoryConsentRepo) PutConsent(_ context.Context, record ConsentRecord) error {
	if r.current[record.UserID] == nil {
		r.current[record.UserID] = map[Purpose]ConsentRecord{}
	}
	r.current[record.UserID][record.Purpose] = record
	r.ledger = append(r.ledger, record)
	return nil
}

func (r *memoryConsentRepo) CurrentConsents(_ context.Context, userID string) (map[Purpose]ConsentRecord, error) {
	out := map[Purpose]ConsentRecord{}
	for purpose, record := range r.current[userID] {
		out[purpose] = record
	}
	return out, nil
}

type memoryPublisher struct {
	events    []EnrichedEvent
	deletions []DataDeletionRecord
	err       error
}

func (p *memoryPublisher) Publish(_ context.Context, events []EnrichedEvent) error {
	if p.err != nil {
		return p.err
	}
	p.events = append(p.events, events...)
	return nil
}

func (p *memoryPublisher) PublishDataDeletions(_ context.Context, records []DataDeletionRecord) error {
	if p.err != nil {
		return p.err
	}
	p.deletions = append(p.deletions, records...)
	return nil
}
