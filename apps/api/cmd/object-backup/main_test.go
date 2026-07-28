package main

import (
	"strings"
	"testing"
	"time"
)

func TestValidateRecoveryPointID(t *testing.T) {
	t.Parallel()

	for _, candidate := range []string{
		"rp-20260726T174500Z-a1b2c3",
		"rp-controlled-change-002",
	} {
		if err := validateRecoveryPointID(candidate); err != nil {
			t.Fatalf("validateRecoveryPointID(%q): %v", candidate, err)
		}
	}

	for _, candidate := range []string{"", "../escape", "rp/escape", "RP UPPER"} {
		if err := validateRecoveryPointID(candidate); err == nil {
			t.Fatalf("validateRecoveryPointID(%q) succeeded", candidate)
		}
	}
}

func TestDestinationKeyPreservesSourceIdentity(t *testing.T) {
	t.Parallel()

	key := destinationKey(
		"rp-20260726T174500Z-a1b2c3",
		"evidence-clean",
		"finding/AVIA-42/report.pdf",
		"version-7",
	)
	for _, expected := range []string{
		"rp-20260726T174500Z-a1b2c3",
		"evidence-clean",
		"finding/AVIA-42/report.pdf",
		"version-7",
	} {
		if !strings.Contains(key, expected) {
			t.Fatalf("destination key %q does not contain %q", key, expected)
		}
	}
}

func TestManifestDigestIsStableAcrossEntryOrder(t *testing.T) {
	t.Parallel()

	left := []manifestEntry{
		{SourceBucket: "b", Key: "two", VersionID: "v2", SHA256: "22"},
		{SourceBucket: "a", Key: "one", VersionID: "v1", SHA256: "11"},
	}
	right := []manifestEntry{left[1], left[0]}

	leftDigest, _, err := manifestDigest(left)
	if err != nil {
		t.Fatal(err)
	}
	rightDigest, _, err := manifestDigest(right)
	if err != nil {
		t.Fatal(err)
	}
	if leftDigest != rightDigest {
		t.Fatalf("manifest digests differ: %s != %s", leftDigest, rightDigest)
	}
}

func TestValidateObjectManifestRejectsChecksumMismatch(t *testing.T) {
	t.Parallel()

	entries := []manifestEntry{{
		SourceBucket: "evidence-clean",
		Key:          "finding/AVIA-42/report.pdf",
		VersionID:    "v1",
		SHA256:       strings.Repeat("a", 64),
		IsLatest:     true,
		LastModified: "2026-07-26T12:00:00Z",
	}}
	digest, canonical, err := manifestDigest(entries)
	if err != nil {
		t.Fatal(err)
	}
	manifest := objectManifest{
		SchemaVersion:   1,
		ArtifactStatus:  "candidate-only",
		RecoveryPointID: "rp-20260726T120000Z-drill",
		Entries:         canonical,
		SHA256:          digest,
	}

	if _, err := validateObjectManifest(
		manifest,
		manifest.RecoveryPointID,
		strings.Repeat("b", 64),
	); err == nil || !strings.Contains(err.Error(), "manifest checksum mismatch") {
		t.Fatalf("unexpected checksum validation result: %v", err)
	}
}

func TestRestoreOrderPreservesVersionHistoryAndLatestObject(t *testing.T) {
	t.Parallel()

	entries := []manifestEntry{
		{
			SourceBucket:         "evidence-clean",
			Key:                  "finding/AVIA-42/report.pdf",
			VersionID:            "v2",
			IsLatest:             true,
			LastModified:         "2026-07-26T12:05:00Z",
			SHA256:               strings.Repeat("b", 64),
			DestinationKey:       "recovery-points/rp-test/objects/report/v2",
			DestinationVersionID: "backup-v2",
		},
		{
			SourceBucket:         "evidence-clean",
			Key:                  "finding/AVIA-42/report.pdf",
			VersionID:            "v1",
			IsLatest:             false,
			LastModified:         "2026-07-26T12:00:00Z",
			SHA256:               strings.Repeat("a", 64),
			DestinationKey:       "recovery-points/rp-test/objects/report/v1",
			DestinationVersionID: "backup-v1",
		},
	}

	ordered, err := restoreOrder(entries)
	if err != nil {
		t.Fatal(err)
	}
	if ordered[0].VersionID != "v1" || ordered[1].VersionID != "v2" {
		t.Fatalf("restore order was %q then %q", ordered[0].VersionID, ordered[1].VersionID)
	}
	if !ordered[len(ordered)-1].IsLatest {
		t.Fatal("latest source version must be restored last")
	}
	if _, err := time.Parse(time.RFC3339, ordered[0].LastModified); err != nil {
		t.Fatalf("restore timestamp is not RFC3339: %v", err)
	}
}
