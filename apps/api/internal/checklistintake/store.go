package checklistintake

import (
	"context"
	"errors"

	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrIdempotencyConflict = errors.New("checklist intake idempotency conflict")
	ErrStaleRevision       = errors.New("checklist intake stale current leaf")
	ErrAppendOnlyViolation = errors.New("checklist intake append-only violation")
)

// Transaction exposes the append-only primitives used by the worker and
// authoring services. Implementations must lock/re-read current leaves before
// appending successors.
type Transaction interface {
	InsertImportBatch(context.Context, ImportBatch) error
	InsertImportFile(context.Context, ImportFile) error
	InsertPhaseReceipt(context.Context, PhaseReceipt) error
	InsertRegisterEntry(context.Context, RegisterEntry) error
	InsertObjectIntent(context.Context, ObjectIntent) error
	InsertAttempt(context.Context, Attempt) error
	InsertAttemptEvent(context.Context, AttemptEvent) error
	InsertIdentityResolution(context.Context, IdentityResolution) error
	InsertExtractionPacket(context.Context, ExtractionReviewPacket) error
	InsertExtractionProposal(context.Context, ExtractionProposal) error
	InsertExtractionDecisionSet(context.Context, ExtractionDecisionSet) error
	InsertExtractionDecision(context.Context, ExtractionDecision) error
	InsertExistingCandidate(context.Context, ExistingCandidate) error
	InsertExistingCandidateQuestion(context.Context, ExistingCandidateQuestion) error
}

type Store interface {
	WithinTransaction(context.Context, func(context.Context, Transaction) error) error
	CreateImportBatch(context.Context, ImportBatch) error
}

type PostgresStore struct {
	Pool *database.Pool
}

func NewPostgresStore(pool *database.Pool) *PostgresStore { return &PostgresStore{Pool: pool} }

func (store *PostgresStore) WithinTransaction(ctx context.Context, fn func(context.Context, Transaction) error) error {
	if store == nil || store.Pool == nil {
		return errors.New("checklist intake store is not configured")
	}
	return database.WithinTransaction(ctx, store.Pool, func(ctx context.Context, tx pgx.Tx) error {
		return fn(ctx, &postgresTransaction{tx: tx})
	})
}

// databaseTx is the small subset of pgx.Tx used by this package. Keeping the
// adapter local makes the append-only transaction boundary explicit.
type databaseTx interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func (store *PostgresStore) CreateImportBatch(ctx context.Context, batch ImportBatch) error {
	if store == nil || store.Pool == nil {
		return errors.New("checklist intake store is not configured")
	}
	return (&postgresTransaction{tx: store.Pool}).InsertImportBatch(ctx, batch)
}
