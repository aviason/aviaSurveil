package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/datafeed"
)

const maxManifestBytes = 8 << 20

type manifestFile struct {
	RunID           string              `json:"run_id"`
	ContractVersion string              `json:"contract_version"`
	Events          []manifestEventFile `json:"events"`
}

type manifestEventFile struct {
	EventID                      string `json:"event_id"`
	CanonicalEventSHA256         string `json:"canonical_event_sha256"`
	DeliveryOutcome              string `json:"delivery_outcome"`
	AcknowledgementReceiptDigest string `json:"acknowledgement_receipt_digest,omitempty"`
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: data-feed-reconcile <producer-manifest.json> <aviacore-manifest.json>")
		os.Exit(64)
	}
	producer, err := loadManifest(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	core, err := loadManifest(os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	result, err := datafeed.ReconcileFeedManifests(producer, core)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := json.NewEncoder(os.Stdout).Encode(map[string]int{"expected_event_count": result.ExpectedEventCount, "acknowledged_event_count": result.AcknowledgedEventCount}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func loadManifest(path string) (datafeed.ReconciliationManifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return datafeed.ReconciliationManifest{}, fmt.Errorf("open reconciliation manifest: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, maxManifestBytes+1))
	decoder.DisallowUnknownFields()
	var wire manifestFile
	if err := decoder.Decode(&wire); err != nil {
		return datafeed.ReconciliationManifest{}, fmt.Errorf("decode reconciliation manifest: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return datafeed.ReconciliationManifest{}, fmt.Errorf("decode reconciliation manifest: trailing JSON")
	}
	events := make([]datafeed.ReconciliationEntry, 0, len(wire.Events))
	for _, event := range wire.Events {
		events = append(events, datafeed.ReconciliationEntry{
			EventID: event.EventID, CanonicalEventSHA256: event.CanonicalEventSHA256,
			DeliveryOutcome: event.DeliveryOutcome, AcknowledgementReceiptDigest: event.AcknowledgementReceiptDigest,
		})
	}
	return datafeed.ReconciliationManifest{RunID: wire.RunID, ContractVersion: wire.ContractVersion, Events: events}, nil
}
