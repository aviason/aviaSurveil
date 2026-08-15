package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadSecretFileRejectsSymlinkAndNewline(t *testing.T) {
	root := t.TempDir()
	valid := filepath.Join(root, "database-url")
	if err := os.WriteFile(valid, []byte("postgres://bootstrap@example.invalid/surveil"), 0o400); err != nil {
		t.Fatal(err)
	}
	if value, err := readSecretFile(valid); err != nil || value == "" {
		t.Fatalf("valid database URL file was rejected: value=%q err=%v", value, err)
	}

	malformed := filepath.Join(root, "malformed")
	if err := os.WriteFile(malformed, []byte("postgres://example.invalid\nsecond-line"), 0o400); err != nil {
		t.Fatal(err)
	}
	if _, err := readSecretFile(malformed); err == nil {
		t.Fatal("database URL file with a newline was accepted")
	}

	link := filepath.Join(root, "link")
	if err := os.Symlink(valid, link); err != nil {
		t.Fatal(err)
	}
	if _, err := readSecretFile(link); err == nil {
		t.Fatal("symlinked database URL file was accepted")
	}
}

func TestRunRejectsInlineDatabaseURL(t *testing.T) {
	err := run(context.Background(), []string{
		"load",
		"--package", filepath.Join(string(filepath.Separator), "app", "catalog.zip"),
		"--manifest", filepath.Join(string(filepath.Separator), "app", "catalog.json"),
		"--manifest-sha256", "sha256:" + strings.Repeat("a", 64),
		"--target", "namibia/demo",
		"--database-url", "postgres://inline-value@example.invalid/surveil",
	}, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "invalid approved AGA loader arguments") {
		t.Fatalf("inline database URL flag was accepted: %v", err)
	}
}
