package analytics

import "context"

type ConsentRepository interface {
	PutConsent(ctx context.Context, record ConsentRecord) error
	CurrentConsents(ctx context.Context, userID string) (map[Purpose]ConsentRecord, error)
}

type Publisher interface {
	Publish(ctx context.Context, events []EnrichedEvent) error
	PublishDataDeletions(ctx context.Context, records []DataDeletionRecord) error
}

type NoopPublisher struct{}

func (NoopPublisher) Publish(context.Context, []EnrichedEvent) error { return nil }

func (NoopPublisher) PublishDataDeletions(context.Context, []DataDeletionRecord) error { return nil }
