package policy

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func TestLoadValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "config.yml", `
rules:
  - id: database-replacement
    match:
      resource_types: [aws_db_instance]
      actions: [delete, create]
    severity: critical
    message: "Database replacement may cause downtime or data loss."
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Rules) != 1 || cfg.Rules[0].ID != "database-replacement" {
		t.Errorf("unexpected rules: %+v", cfg.Rules)
	}
}

func TestLoadRejectsMissingID(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "config.yml", `
rules:
  - severity: low
    message: "missing id"
`)
	if _, err := Load(path); err == nil {
		t.Error("expected error for rule missing id")
	}
}

func TestLoadRejectsDuplicateID(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "config.yml", `
rules:
  - id: dup
    severity: low
    message: "first"
  - id: dup
    severity: high
    message: "second"
`)
	if _, err := Load(path); err == nil {
		t.Error("expected error for duplicate rule id")
	}
}

func TestLoadRejectsInvalidSeverity(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "config.yml", `
rules:
  - id: bad-severity
    severity: extreme
    message: "invalid severity"
`)
	if _, err := Load(path); err == nil {
		t.Error("expected error for invalid severity")
	}
}

func TestLoadRejectsMalformedYAML(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "config.yml", "rules: [this is not valid yaml:")
	if _, err := Load(path); err == nil {
		t.Error("expected error for malformed YAML")
	}
}

func TestDiscoverFindsYml(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".tfsum.yml", "rules: []")
	path, ok := Discover(dir)
	if !ok {
		t.Fatal("expected Discover to find .tfsum.yml")
	}
	if filepath.Base(path) != ".tfsum.yml" {
		t.Errorf("Discover found %s, want .tfsum.yml", path)
	}
}

func TestDiscoverNoneFound(t *testing.T) {
	dir := t.TempDir()
	if _, ok := Discover(dir); ok {
		t.Error("expected Discover to find nothing in an empty directory")
	}
}
