package datafeed

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
)

const maxReplayWindow = 30 * 24 * time.Hour

// ReplayRequest is an approval-bound, value-free selector for already
// immutable producer events. It can never create a new event identity or
// select acknowledged events for replay.
type ReplayRequest struct {
	RunID                 string
	ApprovalID            string
	TenantID              string
	OwningOrganizationID  string
	SourceSystem          string
	ContractVersion       string
	EventIDs              []string
	WindowStart           time.Time
	WindowEnd             time.Time
	AllowedTerminalStates []string
	RequestedAt           time.Time
}

// BackfillRequest is a separately approved, source-consistent historical
// event-API run. The cut is evidence about the source system; it never becomes
// an occurrence time, revision, or replacement identity for any event.
type BackfillRequest struct {
	RunID                string
	ApprovalID           string
	TenantID             string
	OwningOrganizationID string
	SourceSystem         string
	ContractVersion      string
	SourceCutID          string
	SourceManifestDigest string
	CutAt                time.Time
	RequestedAt          time.Time
	EventIDs             []string
}

type normalizedReplayRequest struct {
	RunID                 string   `json:"run_id"`
	ApprovalID            string   `json:"approval_id"`
	TenantID              string   `json:"tenant_id"`
	OwningOrganizationID  string   `json:"owning_organization_id"`
	SourceSystem          string   `json:"source_system"`
	ContractVersion       string   `json:"contract_version"`
	EventIDs              []string `json:"event_ids,omitempty"`
	WindowStart           string   `json:"window_start,omitempty"`
	WindowEnd             string   `json:"window_end,omitempty"`
	AllowedTerminalStates []string `json:"allowed_terminal_states"`
	RequestedAt           string   `json:"requested_at"`
}

type normalizedBackfillRequest struct {
	RunID                string   `json:"run_id"`
	ApprovalID           string   `json:"approval_id"`
	TenantID             string   `json:"tenant_id"`
	OwningOrganizationID string   `json:"owning_organization_id"`
	SourceSystem         string   `json:"source_system"`
	ContractVersion      string   `json:"contract_version"`
	SourceCutID          string   `json:"source_cut_id"`
	SourceManifestDigest string   `json:"source_manifest_digest"`
	CutAt                string   `json:"cut_at"`
	RequestedAt          string   `json:"requested_at"`
	EventIDs             []string `json:"event_ids"`
}

// ReplayRunResult is the immutable replay selection identity. EventCount is
// read from the persisted membership, never inferred from a later live query.
type ReplayRunResult struct {
	RunID         string
	RequestDigest string
	EventCount    int
}

// PostgresReplayStore records an approval-bound replay selection. It does not
// change datafeed_delivery_state: a later worker executes a distinct replay
// delivery lane while the original delivery history stays append-only.
type PostgresReplayStore struct {
	Pool *database.Pool
}

// ValidateReplayRequest rejects broad, ambiguous, or historically unsafe
// replay requests before they can reach a durable state transition.
func ValidateReplayRequest(request ReplayRequest) error {
	if !validReplayUUID(request.RunID) || !validReplayUUID(request.ApprovalID) {
		return fmt.Errorf("datafeed replay requires UUID run and approval identities")
	}
	if strings.TrimSpace(request.TenantID) == "" || strings.TrimSpace(request.OwningOrganizationID) == "" {
		return fmt.Errorf("datafeed replay requires tenant and owning organization scope")
	}
	if request.SourceSystem != sourceSystem || request.ContractVersion != contractVersion {
		return fmt.Errorf("datafeed replay requires the locked source system and contract version")
	}
	if request.RequestedAt.IsZero() {
		return fmt.Errorf("datafeed replay requires an explicit requested time")
	}
	hasEventIDs := len(request.EventIDs) > 0
	hasWindow := !request.WindowStart.IsZero() || !request.WindowEnd.IsZero()
	if hasEventIDs == hasWindow {
		return fmt.Errorf("datafeed replay requires exactly one bounded selector")
	}
	if hasEventIDs {
		if len(request.EventIDs) > 1000 {
			return fmt.Errorf("datafeed replay event selector exceeds bounded policy")
		}
		seen := make(map[string]struct{}, len(request.EventIDs))
		for _, eventID := range request.EventIDs {
			if !validReplayUUID(eventID) {
				return fmt.Errorf("datafeed replay event selector requires UUID identities")
			}
			if _, exists := seen[eventID]; exists {
				return fmt.Errorf("datafeed replay event selector has duplicate identity")
			}
			seen[eventID] = struct{}{}
		}
	} else if request.WindowStart.IsZero() || request.WindowEnd.IsZero() || !request.WindowEnd.After(request.WindowStart) || request.WindowEnd.Sub(request.WindowStart) > maxReplayWindow {
		return fmt.Errorf("datafeed replay window is outside its bounded policy")
	}
	if len(request.AllowedTerminalStates) == 0 || len(request.AllowedTerminalStates) > 3 {
		return fmt.Errorf("datafeed replay requires explicit eligible delivery states")
	}
	seenStates := make(map[string]struct{}, len(request.AllowedTerminalStates))
	for _, state := range request.AllowedTerminalStates {
		if state != "PENDING" && state != "DEAD_LETTER" && state != "QUARANTINED" {
			return fmt.Errorf("datafeed replay state is not eligible")
		}
		if _, exists := seenStates[state]; exists {
			return fmt.Errorf("datafeed replay has duplicate eligible delivery state")
		}
		seenStates[state] = struct{}{}
	}
	return nil
}

// ValidateBackfillRequest allows only the owner-approved historical event API
// mechanism selected by the v3 contract. It deliberately has no caller-set
// event timestamp, revision, or payload fields.
func ValidateBackfillRequest(request BackfillRequest) error {
	if !validReplayUUID(request.RunID) || !validReplayUUID(request.ApprovalID) {
		return fmt.Errorf("datafeed backfill requires UUID run and approval identities")
	}
	if strings.TrimSpace(request.TenantID) == "" || strings.TrimSpace(request.OwningOrganizationID) == "" {
		return fmt.Errorf("datafeed backfill requires tenant and owning organization scope")
	}
	if request.SourceSystem != sourceSystem || request.ContractVersion != contractVersion {
		return fmt.Errorf("datafeed backfill requires the locked source system and contract version")
	}
	if strings.TrimSpace(request.SourceCutID) == "" || !validSHA256(request.SourceManifestDigest) {
		return fmt.Errorf("datafeed backfill requires source-consistent cut evidence")
	}
	if request.CutAt.IsZero() || request.RequestedAt.IsZero() || request.CutAt.After(request.RequestedAt) {
		return fmt.Errorf("datafeed backfill has invalid cut and requested times")
	}
	if len(request.EventIDs) == 0 || len(request.EventIDs) > 1000 {
		return fmt.Errorf("datafeed backfill requires a bounded source-consistent event set")
	}
	seen := make(map[string]struct{}, len(request.EventIDs))
	for _, eventID := range request.EventIDs {
		if !validReplayUUID(eventID) {
			return fmt.Errorf("datafeed backfill event selector requires UUID identities")
		}
		if _, exists := seen[eventID]; exists {
			return fmt.Errorf("datafeed backfill event selector has duplicate identity")
		}
		seen[eventID] = struct{}{}
	}
	return nil
}

// ReplayRequestDigest is the stable scope binding stored beside a replay run.
// Ordering cannot change its identity, but every selector and ownership field
// is included so a changed request cannot replay an earlier authorization.
func ReplayRequestDigest(request ReplayRequest) (string, error) {
	if err := ValidateReplayRequest(request); err != nil {
		return "", err
	}
	eventIDs := append([]string(nil), request.EventIDs...)
	states := append([]string(nil), request.AllowedTerminalStates...)
	sort.Strings(eventIDs)
	sort.Strings(states)
	normalized := normalizedReplayRequest{
		RunID: request.RunID, ApprovalID: request.ApprovalID,
		TenantID: request.TenantID, OwningOrganizationID: request.OwningOrganizationID,
		SourceSystem: request.SourceSystem, ContractVersion: request.ContractVersion,
		EventIDs: eventIDs, AllowedTerminalStates: states,
		RequestedAt: request.RequestedAt.UTC().Format(time.RFC3339Nano),
	}
	if !request.WindowStart.IsZero() {
		normalized.WindowStart = request.WindowStart.UTC().Format(time.RFC3339Nano)
		normalized.WindowEnd = request.WindowEnd.UTC().Format(time.RFC3339Nano)
	}
	raw, err := json.Marshal(normalized)
	if err != nil {
		return "", fmt.Errorf("encode replay request: %w", err)
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), nil
}

func BackfillRequestDigest(request BackfillRequest) (string, error) {
	if err := ValidateBackfillRequest(request); err != nil {
		return "", err
	}
	eventIDs := append([]string(nil), request.EventIDs...)
	sort.Strings(eventIDs)
	raw, err := json.Marshal(normalizedBackfillRequest{
		RunID: request.RunID, ApprovalID: request.ApprovalID, TenantID: request.TenantID,
		OwningOrganizationID: request.OwningOrganizationID, SourceSystem: request.SourceSystem,
		ContractVersion: request.ContractVersion, SourceCutID: request.SourceCutID,
		SourceManifestDigest: request.SourceManifestDigest, CutAt: request.CutAt.UTC().Format(time.RFC3339Nano),
		RequestedAt: request.RequestedAt.UTC().Format(time.RFC3339Nano), EventIDs: eventIDs,
	})
	if err != nil {
		return "", fmt.Errorf("encode backfill request: %w", err)
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), nil
}

func (store PostgresReplayStore) CreateReplayRun(ctx context.Context, request ReplayRequest) (ReplayRunResult, error) {
	if store.Pool == nil {
		return ReplayRunResult{}, fmt.Errorf("datafeed replay store requires a pool")
	}
	if err := ValidateReplayRequest(request); err != nil {
		return ReplayRunResult{}, err
	}
	digest, err := ReplayRequestDigest(request)
	if err != nil {
		return ReplayRunResult{}, err
	}
	returnResult := ReplayRunResult{RunID: request.RunID, RequestDigest: digest}
	err = database.WithinTransaction(ctx, store.Pool, func(ctx context.Context, transaction pgx.Tx) error {
		var existingDigest string
		err := transaction.QueryRow(ctx, `SELECT request_sha256 FROM datafeed_replay_runs WHERE run_id = $1`, request.RunID).Scan(&existingDigest)
		if err == nil {
			if existingDigest != digest {
				return fmt.Errorf("datafeed replay run identity conflicts with a different request scope")
			}
			return transaction.QueryRow(ctx, `SELECT count(*) FROM datafeed_replay_run_events WHERE run_id = $1`, request.RunID).Scan(&returnResult.EventCount)
		}
		if !errorsIsNoRows(err) {
			return fmt.Errorf("read datafeed replay run: %w", err)
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO datafeed_replay_runs (
				run_id, run_kind, approval_id, request_sha256, tenant_id, owning_organization_id,
				source_system, contract_version, requested_at
			) VALUES ($1, 'REPLAY', $2, $3, $4, $5, $6, $7, $8)
		`, request.RunID, request.ApprovalID, digest, request.TenantID, request.OwningOrganizationID,
			request.SourceSystem, request.ContractVersion, request.RequestedAt.UTC()); err != nil {
			return fmt.Errorf("record immutable datafeed replay run: %w", err)
		}
		members, err := selectReplayMembers(ctx, transaction, request)
		if err != nil {
			return err
		}
		if len(members) == 0 {
			return fmt.Errorf("datafeed replay selector matched no eligible scoped events")
		}
		if len(request.EventIDs) > 0 && len(members) != len(request.EventIDs) {
			return fmt.Errorf("datafeed replay selector did not match every requested scoped event")
		}
		for _, member := range members {
			if _, err := transaction.Exec(ctx, `
				INSERT INTO datafeed_replay_run_events (
					run_id, event_id, canonical_event_sha256, effective_at, known_at, occurred_at
				) VALUES ($1, $2, $3, $4, $5, $6)
			`, request.RunID, member.eventID, member.canonicalEventSHA256, member.effectiveAt, member.knownAt, member.occurredAt); err != nil {
				return fmt.Errorf("bind immutable datafeed replay event: %w", err)
			}
			if _, err := transaction.Exec(ctx, `
				INSERT INTO datafeed_replay_delivery_state (run_id, event_id)
				VALUES ($1, $2)
			`, request.RunID, member.eventID); err != nil {
				return fmt.Errorf("create independent datafeed replay delivery lane: %w", err)
			}
		}
		returnResult.EventCount = len(members)
		return nil
	})
	if err != nil {
		return ReplayRunResult{}, err
	}
	return returnResult, nil
}

func (store PostgresReplayStore) CreateBackfillRun(ctx context.Context, request BackfillRequest) (ReplayRunResult, error) {
	if store.Pool == nil {
		return ReplayRunResult{}, fmt.Errorf("datafeed replay store requires a pool")
	}
	if err := ValidateBackfillRequest(request); err != nil {
		return ReplayRunResult{}, err
	}
	digest, err := BackfillRequestDigest(request)
	if err != nil {
		return ReplayRunResult{}, err
	}
	result := ReplayRunResult{RunID: request.RunID, RequestDigest: digest}
	err = database.WithinTransaction(ctx, store.Pool, func(ctx context.Context, transaction pgx.Tx) error {
		var existingDigest string
		err := transaction.QueryRow(ctx, `SELECT request_sha256 FROM datafeed_replay_runs WHERE run_id = $1`, request.RunID).Scan(&existingDigest)
		if err == nil {
			if existingDigest != digest {
				return fmt.Errorf("datafeed backfill run identity conflicts with a different request scope")
			}
			return transaction.QueryRow(ctx, `SELECT count(*) FROM datafeed_replay_run_events WHERE run_id = $1`, request.RunID).Scan(&result.EventCount)
		}
		if !errorsIsNoRows(err) {
			return fmt.Errorf("read datafeed backfill run: %w", err)
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO datafeed_replay_runs (
				run_id, run_kind, approval_id, request_sha256, tenant_id, owning_organization_id,
				source_system, contract_version, source_cut_id, source_manifest_sha256, cut_at, requested_at
			) VALUES ($1, 'BACKFILL', $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`, request.RunID, request.ApprovalID, digest, request.TenantID, request.OwningOrganizationID,
			request.SourceSystem, request.ContractVersion, request.SourceCutID, request.SourceManifestDigest,
			request.CutAt.UTC(), request.RequestedAt.UTC()); err != nil {
			return fmt.Errorf("record immutable datafeed backfill run: %w", err)
		}
		members, err := selectReplayMembers(ctx, transaction, ReplayRequest{
			RunID: request.RunID, ApprovalID: request.ApprovalID, TenantID: request.TenantID,
			OwningOrganizationID: request.OwningOrganizationID, SourceSystem: request.SourceSystem,
			ContractVersion: request.ContractVersion, EventIDs: request.EventIDs,
			AllowedTerminalStates: []string{"PENDING"}, RequestedAt: request.RequestedAt,
		})
		if err != nil {
			return err
		}
		if len(members) != len(request.EventIDs) {
			return fmt.Errorf("datafeed backfill selector did not match every source-consistent event")
		}
		for _, member := range members {
			if _, err := transaction.Exec(ctx, `
				INSERT INTO datafeed_replay_run_events (
					run_id, event_id, canonical_event_sha256, effective_at, known_at, occurred_at
				) VALUES ($1, $2, $3, $4, $5, $6)
			`, request.RunID, member.eventID, member.canonicalEventSHA256, member.effectiveAt, member.knownAt, member.occurredAt); err != nil {
				return fmt.Errorf("bind immutable datafeed backfill event: %w", err)
			}
			if _, err := transaction.Exec(ctx, `INSERT INTO datafeed_replay_delivery_state (run_id, event_id) VALUES ($1, $2)`, request.RunID, member.eventID); err != nil {
				return fmt.Errorf("create independent datafeed backfill delivery lane: %w", err)
			}
		}
		result.EventCount = len(members)
		return nil
	})
	if err != nil {
		return ReplayRunResult{}, err
	}
	return result, nil
}

type replayMember struct {
	eventID                          string
	canonicalEventSHA256             string
	effectiveAt, knownAt, occurredAt time.Time
}

func selectReplayMembers(ctx context.Context, transaction pgx.Tx, request ReplayRequest) ([]replayMember, error) {
	var rows pgx.Rows
	var err error
	if len(request.EventIDs) > 0 {
		rows, err = transaction.Query(ctx, `
			SELECT event.event_id, event.canonical_event_sha256, event.effective_at, event.known_at, event.occurred_at
			FROM datafeed_events event
			JOIN datafeed_delivery_state delivery ON delivery.event_id = event.event_id
			LEFT JOIN datafeed_replay_tombstones tombstone ON tombstone.event_id = event.event_id
			WHERE event.tenant_id = $1 AND event.owning_organization_id = $2
			  AND event.source_system = $3 AND event.contract_version = $4
			  AND event.event_id = ANY($5::uuid[]) AND delivery.status = ANY($6::text[])
			  AND tombstone.event_id IS NULL
			ORDER BY event.occurred_at, event.event_id
			FOR UPDATE OF delivery
		`, request.TenantID, request.OwningOrganizationID, request.SourceSystem, request.ContractVersion, request.EventIDs, request.AllowedTerminalStates)
	} else {
		rows, err = transaction.Query(ctx, `
			SELECT event.event_id, event.canonical_event_sha256, event.effective_at, event.known_at, event.occurred_at
			FROM datafeed_events event
			JOIN datafeed_delivery_state delivery ON delivery.event_id = event.event_id
			LEFT JOIN datafeed_replay_tombstones tombstone ON tombstone.event_id = event.event_id
			WHERE event.tenant_id = $1 AND event.owning_organization_id = $2
			  AND event.source_system = $3 AND event.contract_version = $4
			  AND event.occurred_at >= $5 AND event.occurred_at < $6
			  AND delivery.status = ANY($7::text[]) AND tombstone.event_id IS NULL
			ORDER BY event.occurred_at, event.event_id
			FOR UPDATE OF delivery
		`, request.TenantID, request.OwningOrganizationID, request.SourceSystem, request.ContractVersion, request.WindowStart.UTC(), request.WindowEnd.UTC(), request.AllowedTerminalStates)
	}
	if err != nil {
		return nil, fmt.Errorf("select datafeed replay members: %w", err)
	}
	defer rows.Close()
	members := make([]replayMember, 0)
	for rows.Next() {
		var member replayMember
		if err := rows.Scan(&member.eventID, &member.canonicalEventSHA256, &member.effectiveAt, &member.knownAt, &member.occurredAt); err != nil {
			return nil, fmt.Errorf("read datafeed replay member: %w", err)
		}
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate datafeed replay members: %w", err)
	}
	return members, nil
}

func errorsIsNoRows(err error) bool {
	return err == pgx.ErrNoRows
}

func validReplayUUID(value string) bool {
	var id pgtype.UUID
	return id.Scan(value) == nil && id.Valid
}

func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, character := range value {
		if !(character >= '0' && character <= '9') && !(character >= 'a' && character <= 'f') {
			return false
		}
	}
	return true
}
