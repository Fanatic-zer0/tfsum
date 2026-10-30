package cli

import (
	"errors"
	"testing"

	"tfsum/internal/plan"
)

func reportWithKinds(kinds ...plan.ActionKind) *plan.PlanReport {
	r := &plan.PlanReport{}
	for _, k := range kinds {
		r.ResourceChanges = append(r.ResourceChanges, plan.ResourceChange{ActionKind: k})
	}
	return r
}

func TestPolicyExitCodeFailOnThreshold(t *testing.T) {
	report := reportWithKinds(plan.ActionUpdate)
	report.Summary.HighestRisk = plan.RiskHigh

	opts := defaultOptions()
	opts.FailOn = plan.RiskHigh
	if got := policyExitCode(opts, report); got != ExitPolicy {
		t.Errorf("policyExitCode = %d, want %d (risk meets threshold)", got, ExitPolicy)
	}

	opts.FailOn = plan.RiskCritical
	if got := policyExitCode(opts, report); got != ExitSuccess {
		t.Errorf("policyExitCode = %d, want %d (risk below threshold)", got, ExitSuccess)
	}
}

func TestPolicyExitCodeFailOnDestructive(t *testing.T) {
	report := reportWithKinds(plan.ActionUpdate, plan.ActionDelete)
	opts := defaultOptions()
	opts.FailOnDestructive = true
	if got := policyExitCode(opts, report); got != ExitPolicy {
		t.Errorf("policyExitCode = %d, want %d", got, ExitPolicy)
	}

	report2 := reportWithKinds(plan.ActionUpdate, plan.ActionCreate)
	if got := policyExitCode(opts, report2); got != ExitSuccess {
		t.Errorf("policyExitCode = %d, want %d (no destroy present)", got, ExitSuccess)
	}
}

func TestPolicyExitCodeFailOnReplacement(t *testing.T) {
	report := reportWithKinds(plan.ActionReplace)
	opts := defaultOptions()
	opts.FailOnReplacement = true
	if got := policyExitCode(opts, report); got != ExitPolicy {
		t.Errorf("policyExitCode = %d, want %d", got, ExitPolicy)
	}
}

func TestPolicyExitCodeDefaultNone(t *testing.T) {
	report := reportWithKinds(plan.ActionDelete, plan.ActionReplace)
	report.Summary.HighestRisk = plan.RiskCritical
	opts := defaultOptions() // FailOn=none, no fail-on-destructive/replacement
	if got := policyExitCode(opts, report); got != ExitSuccess {
		t.Errorf("policyExitCode = %d, want %d (no thresholds configured)", got, ExitSuccess)
	}
}

func TestDebugDetailSanitizesByDefault(t *testing.T) {
	opts := defaultOptions()
	err := errors.New("outer context: inner cause: root cause")
	got := debugDetail(opts, err)
	if got != "outer context" {
		t.Errorf("debugDetail = %q, want %q", got, "outer context")
	}

	opts.Debug = true
	got = debugDetail(opts, err)
	if got != err.Error() {
		t.Errorf("debugDetail with --debug = %q, want full error %q", got, err.Error())
	}
}
