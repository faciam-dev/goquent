package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestMigratePlanFailOnAndJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "001_drop.sql")
	if err := os.WriteFile(path, []byte("DROP TABLE users;"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"migrate", "plan", "--format", "json", "--fail-on", "destructive", path}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected destructive threshold exit code 1, got %d stderr=%s", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"steps"`)) {
		t.Fatalf("expected JSON migration plan, got %s", stdout.String())
	}
}

func TestMigrateDryRunRequiresApprovalForDestructiveMigration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "001_drop.sql")
	if err := os.WriteFile(path, []byte("ALTER TABLE users DROP COLUMN legacy_id;"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"migrate", "dry-run", path}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected dry-run without approval to fail, got %d stderr=%s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"migrate", "dry-run", "--approve", "legacy column retired", path}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected approved dry-run to pass, got %d stderr=%s", code, stderr.String())
	}
}

func TestMigrateApplyChecksApprovalBeforeDSN(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "001_drop.sql")
	if err := os.WriteFile(path, []byte("DROP TABLE users;"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"migrate", "apply", path}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected apply without approval to fail before DSN validation, got %d stderr=%s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"migrate", "apply", "--approve", "approved cleanup", path}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected approved apply without DSN to return config error, got %d stderr=%s", code, stderr.String())
	}
}
