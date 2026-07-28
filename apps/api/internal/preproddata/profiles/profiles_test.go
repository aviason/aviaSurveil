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
