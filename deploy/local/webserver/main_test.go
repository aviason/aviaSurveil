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
