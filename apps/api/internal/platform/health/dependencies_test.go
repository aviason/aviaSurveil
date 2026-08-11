package health

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestDependenciesFailClosedOnlyForRequiredFailures(t *testing.T) {
	t.Parallel()

	dependencies, err := NewDependencies(
		Dependency{Name: "postgresql", Required: true, Probe: ProbeFunc(func(context.Context) error {
			return errors.New("postgres://user:secret@database/private")
		})},
		Dependency{Name: "mailpit", Required: false, Probe: ProbeFunc(func(context.Context) error {
			return errors.New("smtp password and provider response")
		})},
		Dependency{Name: "minio", Required: true, Probe: ProbeFunc(func(context.Context) error {
			return nil
		})},
	)
	if err != nil {
		t.Fatalf("NewDependencies() error = %v", err)
	}

	report := dependencies.Readiness(context.Background())
	if report.Status != "not_ready" {
		t.Fatalf("status = %q", report.Status)
	}
	if err := dependencies.Ready(context.Background()); err == nil {
		t.Fatal("required failure did not fail readiness")
	}
	want := []DependencyState{
		{Name: "mailpit", Required: false, Status: "unavailable"},
		{Name: "minio", Required: true, Status: "ready"},
		{Name: "postgresql", Required: true, Status: "unavailable"},
	}
	if !reflect.DeepEqual(report.Dependencies, want) {
		t.Fatalf("dependencies = %+v, want %+v", report.Dependencies, want)
	}
}

func TestOptionalFailureProducesDegradedReadyState(t *testing.T) {
	t.Parallel()

	dependencies, err := NewDependencies(
		Dependency{Name: "postgresql", Required: true, Probe: ProbeFunc(func(context.Context) error {
			return nil
		})},
		Dependency{Name: "native-pdf", Required: false, Probe: ProbeFunc(func(context.Context) error {
			return errors.New("renderer unavailable")
		})},
	)
	if err != nil {
		t.Fatalf("NewDependencies() error = %v", err)
	}

	report := dependencies.Readiness(context.Background())
	if report.Status != "degraded" {
		t.Fatalf("status = %q", report.Status)
	}
	if err := dependencies.Ready(context.Background()); err != nil {
		t.Fatalf("optional failure blocked readiness: %v", err)
	}
}

func TestDependenciesRejectUnsafeOrDuplicateNames(t *testing.T) {
	t.Parallel()

	for _, dependencies := range [][]Dependency{
		{{Name: "postgresql password", Required: true, Probe: ProbeFunc(func(context.Context) error { return nil })}},
		{
			{Name: "postgresql", Required: true, Probe: ProbeFunc(func(context.Context) error { return nil })},
			{Name: "postgresql", Required: true, Probe: ProbeFunc(func(context.Context) error { return nil })},
		},
		{{Name: "postgresql", Required: true}},
	} {
		if _, err := NewDependencies(dependencies...); err == nil {
			t.Fatalf("unsafe dependency set was accepted: %+v", dependencies)
		}
	}
}

func TestDependenciesProbeIndependentServicesConcurrently(t *testing.T) {
	t.Parallel()

	started := make(chan string, 2)
	release := make(chan struct{})
	probe := func(name string) ProbeFunc {
		return func(ctx context.Context) error {
			started <- name
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	dependencies, err := NewDependencies(
		Dependency{Name: "identity", Required: true, Probe: probe("identity"), Timeout: time.Second},
		Dependency{Name: "postgresql", Required: true, Probe: probe("postgresql"), Timeout: time.Second},
	)
	if err != nil {
		t.Fatalf("NewDependencies() error = %v", err)
	}

	reported := make(chan Report, 1)
	go func() {
		reported <- dependencies.Readiness(context.Background())
	}()

	for range 2 {
		select {
		case <-started:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("readiness probes did not start concurrently")
		}
	}
	close(release)
	select {
	case report := <-reported:
		if report.Status != StatusReady {
			t.Fatalf("status = %q", report.Status)
		}
	case <-time.After(time.Second):
		t.Fatal("concurrent readiness report did not complete")
	}
}
