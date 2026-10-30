package plan

import (
	"os"
	"testing"
)

func loadSample(t *testing.T) *PlanReport {
	t.Helper()
	data, err := os.ReadFile("../../testdata/plans/sample.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	report, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return report
}

func TestParseSummaryCounts(t *testing.T) {
	report := loadSample(t)

	want := PlanSummary{
		TotalResources: 8,
		Create:         1,
		Update:         4,
		Replace:        1,
		Delete:         1,
		NoOp:           1,
	}
	got := report.Summary
	if got.TotalResources != want.TotalResources || got.Create != want.Create ||
		got.Update != want.Update || got.Replace != want.Replace ||
		got.Delete != want.Delete || got.NoOp != want.NoOp {
		t.Fatalf("Summary = %+v, want %+v", got, want)
	}
	if !got.HasUnknowns {
		t.Error("expected HasUnknowns to be true (final_snapshot_identifier, id)")
	}
}

func TestParseActionKinds(t *testing.T) {
	report := loadSample(t)
	want := map[string]ActionKind{
		"module.db.aws_db_instance.main":        ActionReplace,
		"module.network.aws_security_group.web": ActionUpdate,
		"module.compute.aws_instance.web[0]":    ActionUpdate,
		"module.compute.aws_instance.web[1]":    ActionDelete,
		"module.compute.aws_instance.web[3]":    ActionCreate,
		"module.iam.aws_iam_role_policy.admin":  ActionUpdate,
		"module.storage.aws_s3_bucket.assets":   ActionUpdate,
		"module.network.aws_route53_record.api": ActionNoOp,
	}
	got := map[string]ActionKind{}
	for _, rc := range report.ResourceChanges {
		got[rc.Address] = rc.ActionKind
	}
	for addr, kind := range want {
		if got[addr] != kind {
			t.Errorf("address %s: ActionKind = %s, want %s", addr, got[addr], kind)
		}
	}
}

func TestParseReplacePaths(t *testing.T) {
	report := loadSample(t)
	for _, rc := range report.ResourceChanges {
		if rc.Address != "module.db.aws_db_instance.main" {
			continue
		}
		if len(rc.ReplacePaths) != 1 || rc.ReplacePaths[0] != "engine_version" {
			t.Errorf("ReplacePaths = %v, want [engine_version]", rc.ReplacePaths)
		}
		return
	}
	t.Fatal("db instance resource change not found")
}

func TestParseSensitiveRedaction(t *testing.T) {
	report := loadSample(t)
	for _, rc := range report.ResourceChanges {
		if rc.Address != "module.db.aws_db_instance.main" {
			continue
		}
		found := false
		for _, ad := range rc.AttributeDiffs {
			if ad.Path == "password" {
				found = true
				if !ad.Sensitive {
					t.Error("password attribute diff should be marked Sensitive")
				}
				if ad.Before != nil || ad.After != nil {
					t.Error("password attribute diff values should be redacted to nil")
				}
			}
		}
		if !found {
			t.Error("expected a password attribute diff")
		}
		return
	}
	t.Fatal("db instance resource change not found")
}

func TestParseDrift(t *testing.T) {
	report := loadSample(t)
	if len(report.Drift) != 1 {
		t.Fatalf("expected 1 drift entry, got %d", len(report.Drift))
	}
	if !report.Summary.HasDrift {
		t.Error("expected HasDrift to be true")
	}
	if report.Drift[0].Address != "module.compute.aws_instance.web[0]" {
		t.Errorf("drift address = %s, want module.compute.aws_instance.web[0]", report.Drift[0].Address)
	}
}

func TestParseOutputChanges(t *testing.T) {
	report := loadSample(t)
	if len(report.OutputChanges) != 1 {
		t.Fatalf("expected 1 output change, got %d", len(report.OutputChanges))
	}
	if report.OutputChanges[0].Name != "api_endpoint" {
		t.Errorf("output name = %s, want api_endpoint", report.OutputChanges[0].Name)
	}
}

func TestCheckFormatVersion(t *testing.T) {
	cases := []struct {
		version string
		wantErr bool
	}{
		{"1.2", false},
		{"1.0", false},
		{"2.0", true},
		{"", true},
	}
	for _, tc := range cases {
		err := checkFormatVersion(tc.version)
		if (err != nil) != tc.wantErr {
			t.Errorf("checkFormatVersion(%q) error = %v, wantErr %v", tc.version, err, tc.wantErr)
		}
	}
}

func TestParseRiskLevel(t *testing.T) {
	cases := []struct {
		in   string
		want RiskLevel
		ok   bool
	}{
		{"none", RiskNone, true},
		{"low", RiskLow, true},
		{"medium", RiskMedium, true},
		{"high", RiskHigh, true},
		{"critical", RiskCritical, true},
		{"bogus", RiskNone, false},
	}
	for _, tc := range cases {
		got, ok := ParseRiskLevel(tc.in)
		if ok != tc.ok || (ok && got != tc.want) {
			t.Errorf("ParseRiskLevel(%q) = (%v, %v), want (%v, %v)", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}
