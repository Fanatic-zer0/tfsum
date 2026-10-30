package render

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tfsum/internal/color"
	"tfsum/internal/plan"
	"tfsum/internal/risk"
)

func loadFixtureReport(t *testing.T) *plan.PlanReport {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "plans", "sample.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	report, err := plan.Parse(data)
	if err != nil {
		t.Fatalf("plan.Parse: %v", err)
	}
	report.Workspace = "production"
	report.GeneratedAt = time.Date(2026, 10, 30, 12, 45, 11, 0, time.UTC)
	risk.Apply(report, risk.DefaultRules())
	return report
}

func compareGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "golden", name)
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("write golden %s: %v", path, err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v (run with UPDATE_GOLDEN=1 to create it)", path, err)
	}
	if !bytes.Equal(want, got) {
		t.Errorf("golden mismatch for %s\n--- want ---\n%s\n--- got ---\n%s", name, want, got)
	}
}

func noColorTheme(w *bytes.Buffer) *color.Theme {
	return color.New(color.ModeNever, false, w)
}

func TestReviewerGolden(t *testing.T) {
	report := loadFixtureReport(t)
	var buf bytes.Buffer
	if err := Reviewer(&buf, report, noColorTheme(&buf), ReviewerOptions{MaxWidth: 80}); err != nil {
		t.Fatalf("Reviewer: %v", err)
	}
	compareGolden(t, "reviewer.golden", buf.Bytes())
}

func TestReviewerTableMode(t *testing.T) {
	report := loadFixtureReport(t)
	var buf bytes.Buffer
	if err := Reviewer(&buf, report, noColorTheme(&buf), ReviewerOptions{MaxWidth: 80, Table: true}); err != nil {
		t.Fatalf("Reviewer: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "RESOURCE") || !strings.Contains(out, "ACTION") || !strings.Contains(out, "RISK") {
		t.Errorf("expected table headers in output, got:\n%s", out)
	}
	if !strings.Contains(out, "module.db.aws_db_instance.main") {
		t.Errorf("expected resource address in table output, got:\n%s", out)
	}
}

func TestExecutiveGolden(t *testing.T) {
	report := loadFixtureReport(t)
	var buf bytes.Buffer
	if err := Executive(&buf, report, noColorTheme(&buf)); err != nil {
		t.Fatalf("Executive: %v", err)
	}
	compareGolden(t, "executive.golden", buf.Bytes())
}

func TestJSONGolden(t *testing.T) {
	report := loadFixtureReport(t)
	var buf bytes.Buffer
	if err := JSON(&buf, report); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	compareGolden(t, "report.golden.json", buf.Bytes())
}

func TestJSONOutputIsValidAndHasNoANSI(t *testing.T) {
	report := loadFixtureReport(t)
	var buf bytes.Buffer
	if err := JSON(&buf, report); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if bytes.ContainsRune(buf.Bytes(), 0x1b) {
		t.Error("JSON output must never contain ANSI escape codes")
	}
	summary, ok := decoded["summary"].(map[string]any)
	if !ok {
		t.Fatal("summary field missing or wrong type")
	}
	// total must exclude no-ops: create(1)+update(4)+replace(1)+delete(1) = 7
	if total, _ := summary["total"].(float64); total != 7 {
		t.Errorf("summary.total = %v, want 7 (no-ops excluded)", summary["total"])
	}
}

func TestDetailedResourceNoANSIWhenDisabled(t *testing.T) {
	report := loadFixtureReport(t)
	var rc plan.ResourceChange
	for _, c := range report.ResourceChanges {
		if c.Address == "module.db.aws_db_instance.main" {
			rc = c
		}
	}
	var buf bytes.Buffer
	if err := DetailedResource(&buf, &rc, noColorTheme(&buf), 80); err != nil {
		t.Fatalf("DetailedResource: %v", err)
	}
	if bytes.ContainsRune(buf.Bytes(), 0x1b) {
		t.Error("detailed resource output must not contain ANSI codes when color is disabled")
	}
	if !bytes.Contains(buf.Bytes(), []byte("SENSITIVE")) {
		t.Error("expected redacted password attribute to be labeled SENSITIVE")
	}
	if !bytes.Contains(buf.Bytes(), []byte("KNOWN AFTER APPLY")) {
		t.Error("expected unknown attribute to be labeled KNOWN AFTER APPLY")
	}
}
