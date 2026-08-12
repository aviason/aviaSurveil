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

func TestScannerReadinessIsRequiredOnlyForRealClamAVMode(t *testing.T) {
	t.Parallel()

	probe, err := newScannerReadiness(config.Settings{
		ScannerMode:               "clamav",
		ClamAVAddress:             "clamav:3310",
		ClamAVMaximumSignatureAge: 48 * time.Hour,
	})
	if err != nil || probe == nil {
		t.Fatalf("ClamAV readiness = %T, err = %v", probe, err)
	}
	testProbe, err := newScannerReadiness(config.Settings{
		Environment: "test",
		ScannerMode: "deterministic-test",
	})
	if err != nil || testProbe != nil {
		t.Fatalf("deterministic readiness = %T, err = %v", testProbe, err)
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

func TestRuntimeReadinessKeepsConfiguredUnavailableDependenciesNamed(t *testing.T) {
	t.Parallel()

	ready := platformhealth.ProbeFunc(func(context.Context) error { return nil })
	probe, err := newRuntimeReadiness(
		config.Settings{
			ObjectStoreEndpoint:  "minio:9000",
			ScannerMode:          "clamav",
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
	for _, name := range []string{"minio", "clamav"} {
		if statuses[name] != platformhealth.DependencyStatusUnavailable {
			t.Fatalf("%s status = %q, want unavailable", name, statuses[name])
		}
	}
}
