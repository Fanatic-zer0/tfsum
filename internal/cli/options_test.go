package cli

import (
	"testing"

	"github.com/dheeryad/tfsum/internal/color"
	"github.com/dheeryad/tfsum/internal/plan"
	"github.com/dheeryad/tfsum/internal/render"
)

func TestParseArgsPassthrough(t *testing.T) {
	opts, err := ParseArgs([]string{"--detailed", "--var-file=prod.tfvars", "-target=module.network", "--view", "executive"})
	if err != nil {
		t.Fatalf("ParseArgs: %v", err)
	}
	if !opts.Detailed {
		t.Error("expected --detailed to be recognized")
	}
	if opts.View != render.ViewExecutive {
		t.Errorf("View = %s, want executive", opts.View)
	}
	want := []string{"--var-file=prod.tfvars", "-target=module.network"}
	if len(opts.TerraformArgs) != len(want) {
		t.Fatalf("TerraformArgs = %v, want %v", opts.TerraformArgs, want)
	}
	for i, a := range want {
		if opts.TerraformArgs[i] != a {
			t.Errorf("TerraformArgs[%d] = %q, want %q", i, opts.TerraformArgs[i], a)
		}
	}
}

func TestParseArgsColorAndFailOn(t *testing.T) {
	opts, err := ParseArgs([]string{"--color=never", "--fail-on=critical", "--fail-on-destructive", "--fail-on-replacement"})
	if err != nil {
		t.Fatalf("ParseArgs: %v", err)
	}
	if opts.ColorMode != color.ModeNever {
		t.Errorf("ColorMode = %s, want never", opts.ColorMode)
	}
	if opts.FailOn != plan.RiskCritical {
		t.Errorf("FailOn = %s, want critical", opts.FailOn)
	}
	if !opts.FailOnDestructive || !opts.FailOnReplacement {
		t.Error("expected both fail-on-destructive and fail-on-replacement to be set")
	}
}

func TestParseArgsInvalidView(t *testing.T) {
	if _, err := ParseArgs([]string{"--view", "bogus"}); err == nil {
		t.Error("expected error for invalid --view value")
	}
}

func TestParseArgsInvalidMaxWidth(t *testing.T) {
	if _, err := ParseArgs([]string{"--max-width", "notanumber"}); err == nil {
		t.Error("expected error for invalid --max-width value")
	}
	if _, err := ParseArgs([]string{"--max-width", "-5"}); err == nil {
		t.Error("expected error for negative --max-width value")
	}
}

func TestParseArgsMissingFlagValue(t *testing.T) {
	if _, err := ParseArgs([]string{"--config"}); err == nil {
		t.Error("expected error when --config is missing its value")
	}
}

func TestParseArgsEqualsForm(t *testing.T) {
	opts, err := ParseArgs([]string{"--plan-file=tfplan", "--terraform-bin=/usr/local/bin/terraform"})
	if err != nil {
		t.Fatalf("ParseArgs: %v", err)
	}
	if opts.PlanFile != "tfplan" {
		t.Errorf("PlanFile = %q, want tfplan", opts.PlanFile)
	}
	if opts.TerraformBin != "/usr/local/bin/terraform" {
		t.Errorf("TerraformBin = %q", opts.TerraformBin)
	}
}

func TestParseArgsHelp(t *testing.T) {
	opts, err := ParseArgs([]string{"--help"})
	if err != nil {
		t.Fatalf("ParseArgs: %v", err)
	}
	if !opts.Help {
		t.Error("expected Help to be true")
	}
}

func TestParseArgsTableFlag(t *testing.T) {
	opts, err := ParseArgs([]string{"--table", "--var-file=prod.tfvars"})
	if err != nil {
		t.Fatalf("ParseArgs: %v", err)
	}
	if !opts.Table {
		t.Error("expected --table to be recognized")
	}
	if len(opts.TerraformArgs) != 1 || opts.TerraformArgs[0] != "--var-file=prod.tfvars" {
		t.Errorf("TerraformArgs = %v, want [--var-file=prod.tfvars]", opts.TerraformArgs)
	}
}

func TestParseArgsVerboseFlag(t *testing.T) {
	opts, err := ParseArgs([]string{"--verbose"})
	if err != nil {
		t.Fatalf("ParseArgs: %v", err)
	}
	if !opts.Verbose {
		t.Error("expected --verbose to be recognized")
	}
}
