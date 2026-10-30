package risk

import (
	"testing"

	"github.com/Fanatic-zer0/tfsum/internal/plan"
)

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
