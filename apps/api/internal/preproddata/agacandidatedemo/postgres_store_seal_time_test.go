package agacandidatedemo

import (
	"testing"
	"time"
)

func TestPostgresTimestampPreservesPersistedSealPrecision(t *testing.T) {
	input := time.Date(2026, time.August, 2, 12, 34, 56, 987654321, time.FixedZone("offset", 3*60*60))
	got := postgresTimestamp(input)
	want := time.Date(2026, time.August, 2, 9, 34, 56, 987654000, time.UTC)
	if !got.Equal(want) || got.Location() != time.UTC {
		t.Fatalf("postgresTimestamp() = %s (%s), want %s (UTC)", got, got.Location(), want)
	}
}
