package main

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestServeFileDoesNotRedirectIndexHTML(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	filename := filepath.Join(root, "index.html")
	if err := os.WriteFile(filename, []byte("<!doctype html><title>Avia</title>"), 0o600); err != nil {
		t.Fatalf("write test document: %v", err)
	}

	request := httptest.NewRequest("GET", "/index.html", nil)
	response := httptest.NewRecorder()
	serveFile(response, request, filename)

	if response.Code != 200 {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if location := response.Header().Get("Location"); location != "" {
		t.Fatalf("Location = %q, want no redirect", location)
	}
	if body := response.Body.String(); body != "<!doctype html><title>Avia</title>" {
		t.Fatalf("body = %q, want test document", body)
	}
}

func TestArtifactCachePolicy(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		path   string
		policy string
	}{
		{path: "/", policy: "no-store, no-transform"},
		{path: "/sw.js", policy: "no-store, no-transform"},
		{path: "/app-shell-assets.json", policy: "no-store, no-transform"},
		{path: "/http-config.json", policy: "no-store, no-transform"},
		{path: "/assets/app-abcdef12.js", policy: "public, max-age=31536000, immutable"},
		{path: "/assets/app.js", policy: "no-store, no-transform"},
		{path: "/operations/dashboard", policy: "no-store, no-transform"},
	} {
		response := httptest.NewRecorder()
		setArtifactCachePolicy(response, test.path)
		if got := response.Header().Get("Cache-Control"); got != test.policy {
			t.Errorf("%s Cache-Control = %q, want %q", test.path, got, test.policy)
		}
	}
}
