package main

import (
	"context"
	"testing"
)

func TestWorkspaceFixtureExporterRequiresExactSource(t *testing.T) {
	err := run(context.Background(), []string{"export"}, nil, nil)
	if err == nil {
		t.Fatalf("fixture source boundary error = %v", err)
	}
}
