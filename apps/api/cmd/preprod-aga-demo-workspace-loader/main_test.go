package main

import (
	"context"
	"strings"
	"testing"
)

func TestWorkspaceLoaderRequiresTextFreeClassificationResult(t *testing.T) {
	err := run(context.Background(), []string{"validate"}, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "paths") {
		t.Fatalf("loader path error = %v", err)
	}
}
