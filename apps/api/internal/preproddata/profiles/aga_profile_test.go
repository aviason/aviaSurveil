package profiles_test

import "testing"

import "github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/profiles"

func TestCanonicalAGAExerciseProfileIsDedicatedAndExact(t *testing.T) {
	profile, err := profiles.Lookup("aga-preprod", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if profile.ResourceEnvelope.IdentityNamespace != "canonical-aga-preprod-exercise-v1" {
		t.Fatalf("unexpected exercise namespace: %s", profile.ResourceEnvelope.IdentityNamespace)
	}
	if profile.ExpectedCounts["catalogQuestions"] != 1310 || profile.ExpectedCounts["catalogForms"] != 52 {
		t.Fatalf("unexpected catalog counts: %+v", profile.ExpectedCounts)
	}
	if profile.ImplementationAllowed {
		t.Fatal("exercise profile must remain candidate-only and not production-enabled")
	}
}
