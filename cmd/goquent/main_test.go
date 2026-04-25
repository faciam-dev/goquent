package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestReviewCommandExitCodesAndJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "query.sql")
	if err := os.WriteFile(path, []byte("SELECT * FROM users"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"review", "--format", "json", "--fail-on", "destructive", path}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 below destructive threshold, got %d stderr=%s", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"findings"`)) {
		t.Fatalf("expected JSON review output, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"review", "--fail-on", "high", path}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1 at high threshold, got %d stderr=%s", code, stderr.String())
	}
}

func TestReviewCommandRejectsBadFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"review", "--format", "sarif"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected config error exit code, got %d", code)
	}
}
