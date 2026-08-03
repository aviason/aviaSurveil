package agacandidatedemo_test

import (
	"strings"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agacandidatedemo"
)

func TestOverlayIntentBindsExactPackageAndSuccessfulBaseEvidence(t *testing.T) {
	intent, err := agacandidatedemo.BuildIntent(agacandidatedemo.IntentInput{
		RunID:                       "aga-demo-run-20260801",
		BaseRunID:                   "base-run-20260731",
		BaseIntentDigest:            digest("a"),
		BaseResultDigest:            digest("b"),
		BaseTargetDigest:            digest("c"),
		CodeDigest:                  digest("d"),
		ContractDigest:              digest("e"),
		ExpectedPackage:             agacandidatedemo.ExactAcceptedPackage(),
		ExpectedRelationshipDigests: map[string]string{"forms": digest("f")},
		Target:                      validOverlayTarget(),
		CreatedAt:                   time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("build overlay intent: %v", err)
	}
	if err := intent.Validate(); err != nil {
		t.Fatalf("validate overlay intent: %v", err)
	}
	if intent.Operation != "LOAD_AGA_CANDIDATE_DEMO_OVERLAY" || intent.SchemaVersion != "preprod-aga-candidate-demo-intent/v1" {
		t.Fatalf("unexpected overlay intent identity: %#v", intent)
	}
}

func validOverlayTarget() agacandidatedemo.TargetFingerprint {
	return agacandidatedemo.TargetFingerprint{
		Environment: "local-preprod", DatabaseName: "aviasurveil360_local_preprod",
		DatabaseOwner: "aviasurveil360_preprod_loader", PostgresSystemIdentifier: "7421987349021349876",
		PostgresHost: "preprod-postgres", PostgresPort: 5432,
		ComposeProject: "aviasurveil360-local-preprod", OverlaySchema: "preprod_aga_demo",
	}
}

func digest(value string) string {
	return "sha256:" + strings.Repeat(value, 64)
}
