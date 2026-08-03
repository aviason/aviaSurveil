package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestAGACandidateDemoRejectsTokenOnArguments(t *testing.T) {
	err := run(context.Background(), []string{"verify-aga-demo-authorization", "/tmp/config.json", "token-value"}, &bytes.Buffer{}, commandDependencies{})
	if err == nil || !strings.Contains(err.Error(), "usage") {
		t.Fatalf("expected argv token rejection, got %v", err)
	}
}

func TestAGACandidateDemoRejectsNonAbsoluteConfig(t *testing.T) {
	err := run(context.Background(), []string{"prepare-aga-demo", "config.json"}, &bytes.Buffer{}, commandDependencies{})
	if err == nil || !strings.Contains(err.Error(), "absolute") {
		t.Fatalf("expected absolute configuration rejection, got %v", err)
	}
}
