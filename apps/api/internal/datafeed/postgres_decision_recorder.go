package datafeed

import (
	"context"
	"errors"
	"fmt"
	"time"

	feedstore "github.com/MarlonJD/aviaSurveil360/apps/api/internal/datafeed/store/postgres"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// PostgresDecisionRecorder records every receipt result and its matching
// append-only attempt in one transaction. A stale lease rolls the entire
// receipt set back: a later worker must never consume an earlier receipt.
type PostgresDecisionRecorder struct {
	Pool  *database.Pool
	NewID func() (string, error)
}

func (recorder PostgresDecisionRecorder) Record(ctx context.Context, decisions []DeliveryDecision) error {
	if recorder.Pool == nil || len(decisions) == 0 {
		return fmt.Errorf("datafeed decision recorder requires a pool and decisions")
	}
	newID := recorder.NewID
	if newID == nil {
		newID = NewEventID
	}
	return database.WithinTransaction(ctx, recorder.Pool, func(ctx context.Context, transaction pgx.Tx) error {
		queries := feedstore.New(transaction)
		for _, decision := range decisions {
			eventID, err := toPGUUID(decision.EventID)
			if err != nil {
				return err
			}
			now := decision.RecordedAt.UTC()
			if now.IsZero() {
				return fmt.Errorf("datafeed decision requires an explicit recorded time")
			}
			var receiptDigest *string
			if decision.ReceiptDigest != "" {
				receiptDigest = &decision.ReceiptDigest
			}
			var outcomeCode = decision.OutcomeCode
			var diagnosticCode *string
			if decision.DiagnosticCode != "" {
				diagnosticCode = &decision.DiagnosticCode
			}
			if decision.ReplayRunID != "" {
				if err := recordReplayDecision(ctx, transaction, decision, receiptDigest, diagnosticCode); err != nil {
					return err
				}
				attemptIDText, err := newID()
				if err != nil {
					return fmt.Errorf("allocate datafeed replay delivery attempt id: %w", err)
				}
				if _, err := toPGUUID(attemptIDText); err != nil {
					return fmt.Errorf("allocate datafeed replay delivery attempt id: %w", err)
				}
				if _, err := transaction.Exec(ctx, `
					INSERT INTO datafeed_replay_delivery_attempts (
						attempt_id, run_id, event_id, lease_generation, outcome_code,
						acknowledgement_receipt_digest, diagnostic_code, occurred_at
					) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
				`, attemptIDText, decision.ReplayRunID, decision.EventID, decision.LeaseGeneration,
					outcomeCode, receiptDigest, diagnosticCode, now); err != nil {
					return fmt.Errorf("append datafeed replay delivery attempt: %w", err)
				}
				continue
			}
			switch decision.Action {
			case DeliveryAcknowledge:
				_, err = queries.AcknowledgeDelivery(ctx, feedstore.AcknowledgeDeliveryParams{
					ReceiptDigest: receiptDigest, AcknowledgedAt: timestamp(now), TerminalOutcomeCode: &outcomeCode,
					EventID: eventID, LeaseGeneration: decision.LeaseGeneration,
				})
			case DeliveryRetry:
				_, err = queries.RetryDelivery(ctx, feedstore.RetryDeliveryParams{
					NextAttemptAt: timestamp(decision.NextAttemptAt), OutcomeCode: &outcomeCode,
					UpdatedAt: timestamp(now), EventID: eventID, LeaseGeneration: decision.LeaseGeneration,
				})
			case DeliveryQuarantine:
				owner := decision.QuarantineOwnerRole
				_, err = queries.QuarantineDelivery(ctx, feedstore.QuarantineDeliveryParams{
					QuarantineOwnerRole: &owner, QuarantineSlaDueAt: timestamp(decision.QuarantineSLADueAt),
					OutcomeCode: &outcomeCode, UpdatedAt: timestamp(now), EventID: eventID,
					LeaseGeneration: decision.LeaseGeneration,
				})
			default:
				return fmt.Errorf("datafeed decision has unsupported action")
			}
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("datafeed fenced lease is stale for event %s", decision.EventID)
			}
			if err != nil {
				return fmt.Errorf("persist datafeed receipt decision: %w", err)
			}
			attemptIDText, err := newID()
			if err != nil {
				return fmt.Errorf("allocate datafeed delivery attempt id: %w", err)
			}
			attemptID, err := toPGUUID(attemptIDText)
			if err != nil {
				return fmt.Errorf("allocate datafeed delivery attempt id: %w", err)
			}
			if err := queries.AppendDeliveryAttempt(ctx, feedstore.AppendDeliveryAttemptParams{
				AttemptID: attemptID, EventID: eventID, LeaseGeneration: decision.LeaseGeneration,
				OutcomeCode: outcomeCode, AcknowledgementReceiptDigest: receiptDigest, DiagnosticCode: diagnosticCode, OccurredAt: timestamp(now),
			}); err != nil {
				return fmt.Errorf("append datafeed delivery attempt: %w", err)
			}
		}
		return nil
	})
}

func recordReplayDecision(ctx context.Context, transaction pgx.Tx, decision DeliveryDecision, receiptDigest, diagnosticCode *string) error {
	if !validReplayUUID(decision.ReplayRunID) {
		return fmt.Errorf("datafeed replay decision requires a UUID run identity")
	}
	var commandTag pgconn.CommandTag
	var err error
	switch decision.Action {
	case DeliveryAcknowledge:
		commandTag, err = transaction.Exec(ctx, `
			UPDATE datafeed_replay_delivery_state
			SET status = 'ACKNOWLEDGED', acknowledgement_receipt_digest = $1,
				acknowledged_at = $2, terminal_outcome_code = $3,
				lease_expires_at = NULL, updated_at = $2
			WHERE run_id = $4 AND event_id = $5 AND status = 'LEASED' AND lease_generation = $6
		`, receiptDigest, decision.RecordedAt.UTC(), decision.OutcomeCode, decision.ReplayRunID, decision.EventID, decision.LeaseGeneration)
	case DeliveryRetry:
		commandTag, err = transaction.Exec(ctx, `
			UPDATE datafeed_replay_delivery_state
			SET status = 'PENDING', lease_expires_at = NULL, next_attempt_at = $1,
				attempt_count = attempt_count + 1, terminal_outcome_code = $2, updated_at = $3
			WHERE run_id = $4 AND event_id = $5 AND status = 'LEASED' AND lease_generation = $6
		`, decision.NextAttemptAt.UTC(), decision.OutcomeCode, decision.RecordedAt.UTC(), decision.ReplayRunID, decision.EventID, decision.LeaseGeneration)
	case DeliveryQuarantine:
		commandTag, err = transaction.Exec(ctx, `
			UPDATE datafeed_replay_delivery_state
			SET status = 'QUARANTINED', lease_expires_at = NULL,
				quarantine_owner_role = $1, quarantine_sla_due_at = $2,
				terminal_outcome_code = $3, updated_at = $4
			WHERE run_id = $5 AND event_id = $6 AND status = 'LEASED' AND lease_generation = $7
		`, decision.QuarantineOwnerRole, decision.QuarantineSLADueAt.UTC(), decision.OutcomeCode,
			decision.RecordedAt.UTC(), decision.ReplayRunID, decision.EventID, decision.LeaseGeneration)
	default:
		return fmt.Errorf("datafeed replay decision has unsupported action")
	}
	if err != nil {
		return fmt.Errorf("persist datafeed replay receipt decision: %w", err)
	}
	if commandTag.RowsAffected() != 1 {
		return fmt.Errorf("datafeed replay fenced lease is stale for event %s", decision.EventID)
	}
	return nil
}

func toPGUUID(value string) (pgtype.UUID, error) {
	var result pgtype.UUID
	if err := result.Scan(value); err != nil || !result.Valid {
		return pgtype.UUID{}, fmt.Errorf("datafeed event identity must be a UUID")
	}
	return result, nil
}

func timestamp(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value.UTC(), Valid: true}
}
