package datafeed

import (
	"context"
	"fmt"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
)

// ReconciliationManifest is the closed, value-free frontier exchanged by the
// producer and AviaCore evidence adapters. It contains no payload, filename,
// secret, user identity, or free-form diagnostic material.
type ReconciliationManifest struct {
	RunID           string
	ContractVersion string
	Events          []ReconciliationEntry
}

type ReconciliationEntry struct {
	EventID                      string
	CanonicalEventSHA256         string
	DeliveryOutcome              string
	AcknowledgementReceiptDigest string
}

type ReconciliationResult struct {
	ExpectedEventCount     int
	AcknowledgedEventCount int
}

// PostgresReconciliationManifestExporter reads only immutable run membership
// and its separate delivery frontier. It never decrypts event payloads or
// reads source business tables while creating reconciliation evidence.
type PostgresReconciliationManifestExporter struct {
	Pool *database.Pool
}

func (exporter PostgresReconciliationManifestExporter) ExportReplayManifest(ctx context.Context, runID string) (ReconciliationManifest, error) {
	if exporter.Pool == nil || !validReplayUUID(runID) {
		return ReconciliationManifest{}, fmt.Errorf("datafeed reconciliation exporter requires a pool and immutable run identity")
	}
	rows, err := exporter.Pool.Query(ctx, `
		SELECT run.contract_version, member.event_id, member.canonical_event_sha256,
		       delivery.status, delivery.acknowledgement_receipt_digest
		FROM datafeed_replay_runs run
		JOIN datafeed_replay_run_events member ON member.run_id = run.run_id
		JOIN datafeed_replay_delivery_state delivery
		  ON delivery.run_id = member.run_id AND delivery.event_id = member.event_id
		WHERE run.run_id = $1
		ORDER BY member.event_id
	`, runID)
	if err != nil {
		return ReconciliationManifest{}, fmt.Errorf("read datafeed reconciliation frontier: %w", err)
	}
	defer rows.Close()
	manifest := ReconciliationManifest{RunID: runID}
	for rows.Next() {
		var entry ReconciliationEntry
		var receiptDigest *string
		if err := rows.Scan(&manifest.ContractVersion, &entry.EventID, &entry.CanonicalEventSHA256, &entry.DeliveryOutcome, &receiptDigest); err != nil {
			return ReconciliationManifest{}, fmt.Errorf("read datafeed reconciliation entry: %w", err)
		}
		if receiptDigest != nil {
			entry.AcknowledgementReceiptDigest = *receiptDigest
		}
		if entry.DeliveryOutcome == "LEASED" {
			return ReconciliationManifest{}, fmt.Errorf("datafeed reconciliation frontier contains an in-flight replay lease")
		}
		manifest.Events = append(manifest.Events, entry)
	}
	if err := rows.Err(); err != nil {
		return ReconciliationManifest{}, fmt.Errorf("iterate datafeed reconciliation frontier: %w", err)
	}
	if err := validateReconciliationManifest(manifest); err != nil {
		return ReconciliationManifest{}, fmt.Errorf("validate exported datafeed reconciliation manifest: %w", err)
	}
	return manifest, nil
}

// ReconcileFeedManifests proves an exact event-level frontier match. It never
// treats a count-only match as success and therefore cannot hide loss, an
// extra event, a content mutation, or an acknowledgement frontier mismatch.
func ReconcileFeedManifests(producer, aviaCore ReconciliationManifest) (ReconciliationResult, error) {
	if err := validateReconciliationManifest(producer); err != nil {
		return ReconciliationResult{}, fmt.Errorf("invalid producer reconciliation manifest: %w", err)
	}
	if err := validateReconciliationManifest(aviaCore); err != nil {
		return ReconciliationResult{}, fmt.Errorf("invalid AviaCore reconciliation manifest: %w", err)
	}
	if producer.RunID != aviaCore.RunID || producer.ContractVersion != aviaCore.ContractVersion {
		return ReconciliationResult{}, fmt.Errorf("reconciliation manifests have different run or contract identity")
	}
	coreByEventID := make(map[string]ReconciliationEntry, len(aviaCore.Events))
	for _, entry := range aviaCore.Events {
		coreByEventID[entry.EventID] = entry
	}
	acknowledged := 0
	for _, expected := range producer.Events {
		actual, exists := coreByEventID[expected.EventID]
		if !exists {
			return ReconciliationResult{}, fmt.Errorf("reconciliation is missing producer event %s", expected.EventID)
		}
		if actual != expected {
			return ReconciliationResult{}, fmt.Errorf("reconciliation event %s does not preserve exact digest and acknowledgement frontier", expected.EventID)
		}
		if expected.DeliveryOutcome == "ACKNOWLEDGED" {
			acknowledged++
		}
		delete(coreByEventID, expected.EventID)
	}
	if len(coreByEventID) != 0 {
		return ReconciliationResult{}, fmt.Errorf("reconciliation contains an unexplained AviaCore event")
	}
	return ReconciliationResult{ExpectedEventCount: len(producer.Events), AcknowledgedEventCount: acknowledged}, nil
}

func validateReconciliationManifest(manifest ReconciliationManifest) error {
	if !validReplayUUID(manifest.RunID) || manifest.ContractVersion != contractVersion || len(manifest.Events) == 0 {
		return fmt.Errorf("manifest requires a locked run, contract, and event set")
	}
	seen := make(map[string]struct{}, len(manifest.Events))
	for _, entry := range manifest.Events {
		if !validReplayUUID(entry.EventID) || !validSHA256(entry.CanonicalEventSHA256) {
			return fmt.Errorf("manifest event identity or canonical digest is invalid")
		}
		if _, exists := seen[entry.EventID]; exists {
			return fmt.Errorf("manifest has duplicate event identity")
		}
		seen[entry.EventID] = struct{}{}
		if entry.DeliveryOutcome != "PENDING" && entry.DeliveryOutcome != "ACKNOWLEDGED" && entry.DeliveryOutcome != "QUARANTINED" {
			return fmt.Errorf("manifest has unsupported delivery outcome")
		}
		if (entry.DeliveryOutcome == "ACKNOWLEDGED") != validSHA256(entry.AcknowledgementReceiptDigest) {
			return fmt.Errorf("manifest acknowledgement receipt shape is invalid")
		}
	}
	return nil
}
