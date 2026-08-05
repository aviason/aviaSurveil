package main

import (
	"context"
	"strings"
	"testing"
)

func TestWorkspaceRoleProvisionerRequiresPrivateCredentials(t *testing.T) {
	if err := run(context.Background(), nil, nil); err == nil || !strings.Contains(err.Error(), "database URL") {
		t.Fatalf("missing credentials/URL error = %v", err)
	}
}
