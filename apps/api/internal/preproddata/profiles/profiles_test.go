package profiles_test

import (
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/profiles"
)

func TestFrozenProfilesAreExactAndVersioned(t *testing.T) {
	expectedOrganizations := map[string]int64{
		"smoke": 3, "acceptance": 25, "realistic": 100, "stress": 200,
	}
	for name, organizations := range expectedOrganizations {
		profile, err := profiles.Lookup(name, "1.0.0")
		if err != nil {
			t.Fatalf("lookup %s: %v", name, err)
		}
		if profile.ExpectedCounts["organizations"] != organizations {
			t.Fatalf("%s organizations = %d", name,
				profile.ExpectedCounts["organizations"])
		}
		if !profile.ResourceEnvelope.SeedRequired {
			t.Fatalf("%s does not require a seed", name)
		}
		if !profile.ResourceEnvelope.ClockOrigin.Equal(
			time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		) {
			t.Fatalf("%s clock origin = %s", name,
				profile.ResourceEnvelope.ClockOrigin)
		}
		if profile.ImplementationAllowed {
			t.Fatalf("%s feasibility must remain not run", name)
		}
		for family, distribution := range profile.ExactDistributions {
			var total int64
			for _, value := range distribution {
				total += value
			}
			if total != profile.ExpectedCounts[family] {
				t.Fatalf(
					"%s/%s distribution = %d, count = %d",
					name,
					family,
					total,
					profile.ExpectedCounts[family],
				)
			}
		}
	}
	if _, err := profiles.Lookup("smoke", "2.0.0"); err == nil {
		t.Fatalf("unknown profile version was accepted")
	}
}

func TestOwnerApprovedLocalQualificationProfilesAreVersionedAndBounded(t *testing.T) {
	expected := []struct {
		name                 string
		organizations        int64
		version              string
		qualificationSeconds int64
		objectBytes          int64
	}{
		{"realistic", 50, "1.1.0", 900, 2 * 1024 * 1024 * 1024},
		{"stress", 100, "1.1.0", 1800, 512 * 1024 * 1024},
	}
	for _, item := range expected {
		profile, err := profiles.Lookup(item.name, item.version)
		if err != nil {
			t.Fatalf("lookup %s@%s: %v", item.name, item.version, err)
		}
		if got := profile.ExpectedCounts["organizations"]; got != item.organizations {
			t.Fatalf("%s organizations = %d", item.name, got)
		}
		if got := profile.ResourceEnvelope.QualificationSeconds; got !=
			item.qualificationSeconds {
			t.Fatalf("%s qualification seconds = %d", item.name, got)
		}
		if got := profile.ResourceEnvelope.ObjectBytes; got != item.objectBytes {
			t.Fatalf("%s object bytes = %d", item.name, got)
		}
		if profile.ExpectedCounts["routeDispositions"] != 86 ||
			profile.ExpectedCounts["visibleActionDispositions"] != 306 {
			t.Fatalf("%s did not retain complete catalogs", item.name)
		}
	}
}
