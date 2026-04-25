package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/faciam-dev/goquent/orm/manifest"
	"github.com/faciam-dev/goquent/orm/migration"
)

func TestManifestCommandGeneratesJSONAndSchema(t *testing.T) {
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.json")
	writeJSON(t, schemaPath, migration.Schema{Tables: []migration.TableSchema{{
		Name: "users",
		Columns: []migration.ColumnSchema{
			{Name: "id", Type: "bigint", Nullable: false},
			{Name: "email", Type: "text", Nullable: false},
		},
	}}})

	var stdout, stderr bytes.Buffer
	code := run([]string{"manifest", "--format", "json", "--schema", schemaPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected manifest generation success, got %d stderr=%s", code, stderr.String())
	}
	var m manifest.Manifest
	if err := json.Unmarshal(stdout.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if len(m.Tables) != 1 || m.Tables[0].Name != "users" {
		t.Fatalf("unexpected manifest output: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"manifest", "schema"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected manifest schema success, got %d stderr=%s", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Goquent Manifest")) {
		t.Fatalf("expected schema output, got %s", stdout.String())
	}
}

func TestManifestVerifyDetectsStaleAndDoctor(t *testing.T) {
	dir := t.TempDir()
	oldSchemaPath := filepath.Join(dir, "old-schema.json")
	newSchemaPath := filepath.Join(dir, "new-schema.json")
	writeJSON(t, oldSchemaPath, migration.Schema{Tables: []migration.TableSchema{{
		Name:    "users",
		Columns: []migration.ColumnSchema{{Name: "id", Type: "bigint"}},
	}}})
	writeJSON(t, newSchemaPath, migration.Schema{Tables: []migration.TableSchema{{
		Name:    "users",
		Columns: []migration.ColumnSchema{{Name: "id", Type: "uuid"}},
	}}})

	stored, err := manifest.Generate(manifest.Options{
		GeneratedAt: time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC),
		Schema:      loadTestSchema(t, oldSchemaPath),
	})
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(dir, "manifest.json")
	writeJSON(t, manifestPath, stored)

	var stdout, stderr bytes.Buffer
	code := run([]string{"manifest", "verify", "--manifest", manifestPath, "--schema", newSchemaPath, "--format", "json"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected stale verify exit code 1, got %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"fresh": false`)) {
		t.Fatalf("expected stale JSON verification, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"doctor", "--manifest", manifestPath, "--schema", newSchemaPath}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected doctor stale exit code 1, got %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Manifest Verification")) {
		t.Fatalf("expected doctor verification output, got %s", stdout.String())
	}
}

func TestReviewCommandCanFailOnStaleManifest(t *testing.T) {
	dir := t.TempDir()
	stored, err := manifest.Generate(manifest.Options{
		GeneratedAt: time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC),
		Schema: &migration.Schema{Tables: []migration.TableSchema{{
			Name:    "users",
			Columns: []migration.ColumnSchema{{Name: "id", Type: "bigint"}},
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	current, err := manifest.Generate(manifest.Options{
		GeneratedAt: time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC),
		Schema: &migration.Schema{Tables: []migration.TableSchema{{
			Name:    "users",
			Columns: []migration.ColumnSchema{{Name: "id", Type: "uuid"}},
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	stored = manifest.AttachVerification(stored, manifest.Verify(stored, current, time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC)))
	manifestPath := filepath.Join(dir, "manifest.json")
	writeJSON(t, manifestPath, stored)

	var stdout, stderr bytes.Buffer
	code := run([]string{"review", "--manifest", manifestPath, "--require-fresh-manifest", dir}, &stdout, &stderr)
	if code != 3 {
		t.Fatalf("expected stale manifest exit code 3, got %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func loadTestSchema(t *testing.T, path string) *migration.Schema {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var schema migration.Schema
	if err := json.Unmarshal(b, &schema); err != nil {
		t.Fatal(err)
	}
	return &schema
}
