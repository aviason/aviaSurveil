package main

import (
	"context"
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/platform/config"
	platformhealth "github.com/aviason/aviaSurveil/internal/platform/health"
)

func TestUploadServiceConfigsUseInjectedScenarioClock(t *testing.T) {
	scenarioTime := time.Date(2026, time.June, 15, 9, 0, 0, 0, time.UTC)
	clock := func() time.Time { return scenarioTime }
	idGenerator := func(prefix string) string { return prefix + "-canonical" }
	settings := config.Settings{
		QuarantineBucket: "quarantine",
		CanonicalBucket:  "canonical",
	}

	evidenceConfig, attachmentConfig := uploadServiceConfigs(settings, clock, idGenerator)
	planningDependencies := planningServiceDependencies(clock, idGenerator)
	communicationsDependencies := communicationsWorkflowDependencies(clock, idGenerator)

	if got := evidenceConfig.Clock(); !got.Equal(scenarioTime) {
		t.Fatalf("Evidence upload clock = %s, want %s", got, scenarioTime)
	}
	if got := attachmentConfig.Clock(); !got.Equal(scenarioTime) {
		t.Fatalf("Inspection Attachment upload clock = %s, want %s", got, scenarioTime)
	}
	if got := evidenceConfig.IDGenerator("evidence"); got != "evidence-canonical" {
		t.Fatalf("Evidence upload ID generator = %q", got)
	}
	if got := attachmentConfig.IDGenerator("attachment"); got != "attachment-canonical" {
		t.Fatalf("Inspection Attachment upload ID generator = %q", got)
	}
	if got := planningDependencies.Clock(); !got.Equal(scenarioTime) {
		t.Fatalf("Planning service clock = %s, want %s", got, scenarioTime)
	}
	if got := planningDependencies.IDGenerator("planning"); got != "planning-canonical" {
		t.Fatalf("Planning service ID generator = %q", got)
	}
	if got := communicationsDependencies.Clock(); !got.Equal(scenarioTime) {
		t.Fatalf("Communications workflow clock = %s, want %s", got, scenarioTime)
	}
	if got := communicationsDependencies.IDGenerator("notification"); got != "notification-canonical" {
		t.Fatalf("Communications workflow ID generator = %q", got)
	}
}

func TestScannerReadinessDoesNotRequireAnExternalScanner(t *testing.T) {
	t.Parallel()

	for _, mode := range []string{"", "disabled", "deterministic-test", "guardduty-s3"} {
		probe, err := newScannerReadiness(config.Settings{ScannerMode: mode})
		if err != nil || probe != nil {
			t.Fatalf("%s readiness = %T, err = %v", mode, probe, err)
		}
	}
}

func TestCanonicalAGAExerciseProfileRequiresDedicatedLocalNamespace(t *testing.T) {
	lookup := func(values map[string]string) func(string) (string, bool) {
		return func(key string) (string, bool) { value, ok := values[key]; return value, ok }
	}
	base := map[string]string{
		"AVIA_PREPROD_PROFILE":               "aga-preprod@1.0.0",
		"AVIA_PREPROD_PROFILE_QUALIFICATION": "true",
		"AVIA_PREPROD_IDENTITY_NAMESPACE":    "canonical-aga-preprod-exercise-v1",
		"AVIA_PREPROD_DATABASE_NAME":         "aviasurveil360_local_preprod",
		"AVIA_PREPROD_DATABASE_OWNER":        "aviasurveil360_preprod_loader",
	}
	if !canonicalAGAExerciseProfileEnabled("local-preprod", lookup(base)) {
		t.Fatal("expected the exact local disposable profile to be enabled")
	}
	for _, mutate := range []func(map[string]string){
		func(values map[string]string) {
			values["AVIA_PREPROD_PROFILE"] = "aga-preprod@1.0.0"
			values["AVIA_PREPROD_IDENTITY_NAMESPACE"] = "shared-preprod"
		},
		func(values map[string]string) { values["AVIA_PREPROD_PROFILE_QUALIFICATION"] = "false" },
		func(values map[string]string) { values["AVIA_PREPROD_DATABASE_OWNER"] = "shared-preprod-owner" },
	} {
		values := map[string]string{}
		for key, value := range base {
			values[key] = value
		}
		mutate(values)
		if canonicalAGAExerciseProfileEnabled("local-preprod", lookup(values)) {
			t.Fatalf("exercise profile enabled for unsafe settings: %+v", values)
		}
	}
	if canonicalAGAExerciseProfileEnabled("production", lookup(base)) {
		t.Fatal("exercise profile enabled outside local-preprod")
	}
}

func TestCanonicalTestExerciseProfileRequiresExplicitDisposableFlags(t *testing.T) {
	if !canonicalTestExerciseProfileEnabled(config.Settings{
		Environment:          "test",
		CanonicalSeed:        true,
		CanonicalTestProfile: true,
	}) {
		t.Fatal("expected the explicit canonical test exercise profile to be enabled")
	}
	for _, settings := range []config.Settings{
		{Environment: "test", CanonicalTestProfile: true},
		{Environment: "test", CanonicalSeed: true},
		{Environment: "local-preprod", CanonicalSeed: true, CanonicalTestProfile: true},
		{Environment: "production", CanonicalSeed: true, CanonicalTestProfile: true},
	} {
		if canonicalTestExerciseProfileEnabled(settings) {
			t.Fatalf("canonical test exercise profile enabled for unsafe settings: %+v", settings)
		}
	}
}

func TestRuntimeReadinessKeepsConfiguredUnavailableDependenciesNamed(t *testing.T) {
	t.Parallel()

	ready := platformhealth.ProbeFunc(func(context.Context) error { return nil })
	probe, err := newRuntimeReadiness(
		config.Settings{
			ObjectStoreEndpoint:  "minio:9000",
			ScannerMode:          "disabled",
			RuntimeHealthTimeout: time.Second,
		},
		ready,
		ready,
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("newRuntimeReadiness returned %v", err)
	}
	dependencies, ok := probe.(*platformhealth.Dependencies)
	if !ok {
		t.Fatalf("runtime readiness = %T, want named dependencies", probe)
	}
	report := dependencies.Readiness(context.Background())
	if report.Status != platformhealth.StatusNotReady {
		t.Fatalf("readiness status = %q, want not_ready", report.Status)
	}
	statuses := make(map[string]platformhealth.DependencyStatus)
	for _, dependency := range report.Dependencies {
		statuses[dependency.Name] = dependency.Status
	}
	for _, name := range []string{"minio"} {
		if statuses[name] != platformhealth.DependencyStatusUnavailable {
			t.Fatalf("%s status = %q, want unavailable", name, statuses[name])
		}
	}
}
