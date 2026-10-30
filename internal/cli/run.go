package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"tfsum/internal/color"
	"tfsum/internal/plan"
	"tfsum/internal/policy"
	"tfsum/internal/render"
	"tfsum/internal/risk"
	"tfsum/internal/tempdir"
	"tfsum/internal/terraform"
)

// Run parses args, executes the full plan/parse/risk/render pipeline (or
// reads an existing --plan-file), and returns the process exit code.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	opts, err := ParseArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return ExitInternal
	}
	if opts.Help {
		fmt.Fprint(stdout, usageText)
		return ExitSuccess
	}
	if opts.Version {
		fmt.Fprintf(stdout, "tfsum %s\n", Version)
		return ExitSuccess
	}

	outTheme := color.New(opts.ColorMode, opts.NoColor, stdout)
	errTheme := color.New(opts.ColorMode, opts.NoColor, stderr)

	rules, err := loadRules(opts.ConfigPath)
	if err != nil {
		render.Error(stderr, errTheme, "Invalid configuration", "", debugDetail(opts, err),
			[]string{"Check your --config YAML file for syntax or rule errors."})
		return ExitInternal
	}

	bin, err := terraform.ResolveBinary(opts.TerraformBin)
	if err != nil {
		render.Error(stderr, errTheme, "Terraform binary not found", "", debugDetail(opts, err),
			[]string{
				"Install Terraform (or OpenTofu) and ensure it is on PATH.",
				"Or pass --terraform-bin=/path/to/terraform.",
			})
		return ExitTerraformFail
	}

	report, keptPlanPath, exitCode, ok := buildReport(ctx, opts, bin, stdout, stderr, outTheme, errTheme)
	if !ok {
		return exitCode
	}

	risk.Apply(report, rules)

	if err := renderReport(stdout, opts, report, outTheme); err != nil {
		render.Error(stderr, errTheme, "Failed to render report", "", debugDetail(opts, err), nil)
		return ExitInternal
	}

	if keptPlanPath != "" {
		fmt.Fprintf(stdout, "\nPlan retained at: %s\n", keptPlanPath)
		render.Warning(stdout, outTheme, "This file may contain sensitive state values; handle it securely.")
	}

	return policyExitCode(opts, report)
}

// buildReport either reads --plan-file or runs the full `terraform plan` +
// `terraform show -json` flow inside a managed, auto-cleaned temp
// directory. ok is false if an error was already reported and the caller
// should return exitCode.
func buildReport(ctx context.Context, opts *Options, bin string, stdout, stderr io.Writer, outTheme, errTheme *color.Theme) (*plan.PlanReport, string, int, bool) {
	workDir := "."

	if opts.PlanFile != "" {
		jsonData, err := terraform.ShowJSON(ctx, bin, workDir, opts.PlanFile)
		if err != nil {
			render.Error(stderr, errTheme, "Failed to read plan file", "", debugDetail(opts, err), nil)
			return nil, "", ExitTerraformFail, false
		}
		report, err := plan.Parse(jsonData)
		if err != nil {
			render.Error(stderr, errTheme, "Failed to parse plan JSON", "", debugDetail(opts, err),
				[]string{"The saved plan may be from an unsupported Terraform version."})
			return nil, "", ExitInternal, false
		}
		if ws, err := terraform.WorkspaceName(ctx, bin, workDir); err == nil {
			report.Workspace = ws
		}
		return report, "", ExitSuccess, true
	}

	if terraform.LooksUninitialized(workDir) {
		// Non-fatal hint: some valid configurations (e.g. only built-in
		// providers) never create a .terraform directory. Warn and let
		// `terraform plan` itself be the final authority.
		render.Warning(stderr, errTheme, "No .terraform directory found; if the plan below fails, run `terraform init` first.")
	}

	mgr, err := tempdir.New(workDir)
	if err != nil {
		render.Error(stderr, errTheme, "Failed to create temporary directory", "", debugDetail(opts, err), nil)
		return nil, "", ExitInternal, false
	}
	defer mgr.Cleanup()

	// By default, Terraform's own plan progress/diff output and its
	// "Saved the plan to: ..." / "To perform exactly these actions..."
	// messages are suppressed so the report starts at TERRAFORM PLAN
	// SUMMARY. They're captured instead of discarded so the real
	// Terraform output can still be shown if the plan fails. --verbose
	// restores the live-streamed experience.
	planOut, planErr := stdout, stderr
	var capturedOut, capturedErr bytes.Buffer
	if !opts.Verbose {
		planOut, planErr = &capturedOut, &capturedErr
	}

	exitCode, err := terraform.RunPlan(ctx, bin, workDir, mgr.PlanPath(), opts.TerraformArgs, planOut, planErr)
	if err != nil {
		if errors.Is(err, context.Canceled) || ctx.Err() != nil {
			render.Warning(stderr, errTheme, "Interrupted; cleaning up temporary files...")
			return nil, "", ExitInterrupted, false
		}
		render.Error(stderr, errTheme, "Failed to run terraform plan", "", debugDetail(opts, err), nil)
		return nil, "", ExitTerraformFail, false
	}
	if exitCode != 0 {
		render.Error(stderr, errTheme, "Terraform plan failed", fmt.Sprintf("Terraform exited with code %d.", exitCode),
			capturedPlanOutput(opts, &capturedOut, &capturedErr),
			[]string{
				"Run `terraform validate`.",
				"Check the supplied var-file.",
				"Re-run with `--view full` after fixing the configuration.",
			})
		return nil, "", exitCode, false
	}

	jsonData, err := terraform.ShowJSON(ctx, bin, workDir, mgr.PlanPath())
	if err != nil {
		render.Error(stderr, errTheme, "Failed to convert plan to JSON", "", debugDetail(opts, err), nil)
		return nil, "", ExitInternal, false
	}

	report, err := plan.Parse(jsonData)
	if err != nil {
		render.Error(stderr, errTheme, "Failed to parse plan JSON", "", debugDetail(opts, err),
			[]string{"This may be an unsupported Terraform plan format version."})
		return nil, "", ExitInternal, false
	}

	if ws, err := terraform.WorkspaceName(ctx, bin, workDir); err == nil {
		report.Workspace = ws
	}

	var keptPlanPath string
	if opts.KeepPlan {
		if keptPath, err := mgr.Keep(workDir); err == nil {
			keptPlanPath = keptPath
		} else {
			render.Warning(stdout, outTheme, fmt.Sprintf("Failed to retain plan file: %v", err))
		}
	}

	return report, keptPlanPath, ExitSuccess, true
}

func renderReport(w io.Writer, opts *Options, report *plan.PlanReport, theme *color.Theme) error {
	switch opts.View {
	case render.ViewJSON:
		return render.JSON(w, report)
	case render.ViewExecutive:
		return render.Executive(w, report, theme)
	case render.ViewFull:
		return render.Reviewer(w, report, theme, render.ReviewerOptions{MaxWidth: opts.MaxWidth, Detailed: true, Table: opts.Table})
	default:
		return render.Reviewer(w, report, theme, render.ReviewerOptions{MaxWidth: opts.MaxWidth, Detailed: opts.Detailed, Table: opts.Table})
	}
}

func policyExitCode(opts *Options, report *plan.PlanReport) int {
	if opts.FailOnDestructive {
		for _, rc := range report.ResourceChanges {
			if rc.ActionKind == plan.ActionDelete {
				return ExitPolicy
			}
		}
	}
	if opts.FailOnReplacement {
		for _, rc := range report.ResourceChanges {
			if rc.ActionKind == plan.ActionReplace {
				return ExitPolicy
			}
		}
	}
	if opts.FailOn != plan.RiskNone && report.Summary.HighestRisk >= opts.FailOn {
		return ExitPolicy
	}
	return ExitSuccess
}

func loadRules(configPath string) ([]risk.Rule, error) {
	defaults := risk.DefaultRules()

	path := configPath
	if path == "" {
		if discovered, ok := policy.Discover("."); ok {
			path = discovered
		}
	}
	if path == "" {
		return defaults, nil
	}

	cfg, err := policy.Load(path)
	if err != nil {
		return nil, err
	}
	return risk.MergeRules(defaults, cfg.Rules)
}

// debugDetail returns err's full text only when --debug is set; otherwise
// it returns a short, sanitized message so tfsum never prints a raw Go
// stack trace or internal error chain to users by default.
func debugDetail(opts *Options, err error) string {
	if err == nil {
		return ""
	}
	if opts.Debug {
		return err.Error()
	}
	msg := err.Error()
	if idx := strings.Index(msg, ": "); idx > 0 {
		return msg[:idx]
	}
	return msg
}

// capturedPlanOutput returns Terraform's own captured plan stdout/stderr
// for display inside an error report. It's empty when --verbose was used,
// since that output was already streamed live and shouldn't be repeated.
func capturedPlanOutput(opts *Options, stdout, stderr *bytes.Buffer) string {
	if opts.Verbose {
		return ""
	}
	return strings.TrimRight(stdout.String()+stderr.String(), "\n")
}
