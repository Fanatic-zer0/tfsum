package risk

import (
	"os"
	"testing"

	"tfsum/internal/plan"
)

func loadSampleReport(t *testing.T) *plan.PlanReport {
	t.Helper()
	data, err := os.ReadFile("../../testdata/plans/sample.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	report, err := plan.Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return report
}

func riskByAddress(report *plan.PlanReport, addr string) plan.RiskFinding {
	for _, rc := range report.ResourceChanges {
		if rc.Address == addr {
			return rc.Risk
		}
	}
	return plan.RiskFinding{}
}

func TestApplyDefaultRules(t *testing.T) {
	report := loadSampleReport(t)
	Apply(report, DefaultRules())

	cases := []struct {
		address  string
		wantRule string
		wantMin  plan.RiskLevel
	}{
		{"module.db.aws_db_instance.main", "stateful-replacement", plan.RiskCritical},
		{"module.network.aws_security_group.web", "public-ingress-ipv4", plan.RiskCritical},
		{"module.compute.aws_instance.web[1]", "resource-deletion", plan.RiskCritical},
		{"module.iam.aws_iam_role_policy.admin", "iam-wildcard-action", plan.RiskCritical},
		{"module.storage.aws_s3_bucket.assets", "tag-only-change", plan.RiskLow},
	}
	for _, tc := range cases {
		got := riskByAddress(report, tc.address)
		if got.Level < tc.wantMin {
			t.Errorf("%s: Risk.Level = %s, want >= %s", tc.address, got.Level, tc.wantMin)
		}
	}

	if report.Summary.HighestRisk != plan.RiskCritical {
		t.Errorf("Summary.HighestRisk = %s, want CRITICAL", report.Summary.HighestRisk)
	}

	if len(report.Drift) == 0 {
		t.Fatal("expected drift entries")
	}
	if report.Drift[0].Risk.Level < plan.RiskMedium {
		t.Errorf("drift risk = %s, want at least MEDIUM", report.Drift[0].Risk.Level)
	}
}

func TestEvaluateResourceUnknownValues(t *testing.T) {
	report := loadSampleReport(t)
	Apply(report, DefaultRules())

	found := false
	for _, f := range report.Risks {
		if f.RuleID == "unknown-values" {
			found = true
		}
	}
	if !found {
		t.Error("expected at least one unknown-values finding")
	}
}

func TestMatchesActionSetExact(t *testing.T) {
	rc := &plan.ResourceChange{
		Address: "test.resource",
		Type:    "aws_instance",
		Actions: []string{"update"},
	}
	rule := Rule{ID: "delete-only", Match: RuleMatch{Actions: []string{"delete"}}, Severity: "critical"}
	if matches(rc, rule) {
		t.Error("update-only resource should not match a delete-only rule")
	}

	rc.Actions = []string{"delete"}
	if !matches(rc, rule) {
		t.Error("delete resource should match a delete-only rule")
	}
}

func TestMatchesReplacementActionSetOrderIndependent(t *testing.T) {
	rule := Rule{ID: "replace", Match: RuleMatch{Actions: []string{"delete", "create"}}, Severity: "high"}
	rc1 := &plan.ResourceChange{Actions: []string{"delete", "create"}}
	rc2 := &plan.ResourceChange{Actions: []string{"create", "delete"}}
	if !matches(rc1, rule) || !matches(rc2, rule) {
		t.Error("replacement rule should match regardless of action order")
	}
}

func TestEmptyMatchNeverMatches(t *testing.T) {
	rule := Rule{ID: "empty", Match: RuleMatch{}, Severity: "critical"}
	rc := &plan.ResourceChange{Type: "aws_instance", Actions: []string{"update"}}
	if matches(rc, rule) {
		t.Error("a rule with an empty match should never match any resource")
	}
}

func TestOnlyChangedAttributesWithin(t *testing.T) {
	diffs := []plan.AttributeDiff{{Path: "tags.Environment"}, {Path: "tags.Team"}}
	if !onlyChangedAttributesWithin(diffs, []string{"tags"}) {
		t.Error("expected tags-only change to match allowed=[tags]")
	}
	diffs = append(diffs, plan.AttributeDiff{Path: "instance_type"})
	if onlyChangedAttributesWithin(diffs, []string{"tags"}) {
		t.Error("expected mixed changes to not match tags-only allowed set")
	}
}

func TestMergeRulesOverride(t *testing.T) {
	defaults := []Rule{{ID: "resource-deletion", Severity: "critical"}}
	user := []Rule{{ID: "resource-deletion", Severity: "high", Override: true}}
	merged, err := MergeRules(defaults, user)
	if err != nil {
		t.Fatalf("MergeRules: %v", err)
	}
	if len(merged) != 1 || merged[0].Severity != "high" {
		t.Errorf("expected override to replace severity, got %+v", merged)
	}
}

func TestMergeRulesDuplicateWithoutOverrideFails(t *testing.T) {
	defaults := []Rule{{ID: "resource-deletion", Severity: "critical"}}
	user := []Rule{{ID: "resource-deletion", Severity: "low"}}
	if _, err := MergeRules(defaults, user); err == nil {
		t.Error("expected error for duplicate rule id without override: true")
	}
}

func TestMergeRulesAppendsNewRule(t *testing.T) {
	defaults := []Rule{{ID: "a", Severity: "low"}}
	user := []Rule{{ID: "b", Severity: "high"}}
	merged, err := MergeRules(defaults, user)
	if err != nil {
		t.Fatalf("MergeRules: %v", err)
	}
	if len(merged) != 2 {
		t.Errorf("expected 2 merged rules, got %d", len(merged))
	}
}
