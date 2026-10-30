// Package terraform wraps invocation of the Terraform (or OpenTofu) CLI:
// binary resolution, init-state checking, workspace detection, and
// streaming `plan` / `show -json` execution.
package terraform

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ResolveBinary determines which Terraform (or OpenTofu) binary to invoke,
// preferring an explicit --terraform-bin flag, then TF_CLI_BINARY, then
// "terraform" resolved from PATH.
func ResolveBinary(flagPath string) (string, error) {
	if flagPath != "" {
		abs, err := exec.LookPath(flagPath)
		if err != nil {
			return "", fmt.Errorf("--terraform-bin %q not found or not executable: %w", flagPath, err)
		}
		return abs, nil
	}
	if envBin := os.Getenv("TF_CLI_BINARY"); envBin != "" {
		abs, err := exec.LookPath(envBin)
		if err != nil {
			return "", fmt.Errorf("TF_CLI_BINARY %q not found or not executable: %w", envBin, err)
		}
		return abs, nil
	}
	abs, err := exec.LookPath("terraform")
	if err != nil {
		return "", fmt.Errorf("terraform binary not found in PATH: %w", err)
	}
	return abs, nil
}

// LooksUninitialized reports whether the working directory shows no sign
// of having been initialized with `terraform init` (no .terraform
// directory). This is a best-effort, non-fatal hint rather than a hard
// gate: some valid configurations (e.g. ones using only Terraform's
// built-in "terraform" provider, with no external providers to download or
// lock) never create a .terraform directory even when fully usable.
// Callers should surface this as a warning and still attempt `terraform
// plan`, letting Terraform's own error surface if init is genuinely
// required.
func LooksUninitialized(workDir string) bool {
	info, err := os.Stat(filepath.Join(workDir, ".terraform"))
	return err != nil || !info.IsDir()
}

// disallowedPassthroughFlags are terraform plan flags tfsum manages itself
// and must never be forwarded, since they would collide with tfsum's own
// exit-code semantics.
var disallowedPassthroughFlags = []string{"-detailed-exitcode", "--detailed-exitcode"}

// FilterDisallowedArgs strips flags tfsum must not forward to `terraform
// plan` (currently just -detailed-exitcode).
func FilterDisallowedArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		skip := false
		for _, bad := range disallowedPassthroughFlags {
			if a == bad || strings.HasPrefix(a, bad+"=") {
				skip = true
				break
			}
		}
		if !skip {
			out = append(out, a)
		}
	}
	return out
}

// WorkspaceName runs `terraform workspace show` to determine the active
// workspace name, since it is not included in the JSON plan output.
func WorkspaceName(ctx context.Context, bin, workDir string) (string, error) {
	cmd := exec.CommandContext(ctx, bin, "workspace", "show")
	cmd.Dir = workDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("terraform workspace show: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

// RunPlan runs `terraform plan -input=false -out=planPath <extraArgs...>`,
// streaming Terraform's own stdout/stderr live to stdout/stderr so the
// user keeps their normal Terraform progress experience. It returns
// Terraform's exact process exit code; a non-zero code is not itself a Go
// error. A non-nil error means tfsum failed to run/observe the process at
// all (e.g. binary missing, or ctx was cancelled by a signal).
func RunPlan(ctx context.Context, bin, workDir, planPath string, extraArgs []string, stdout, stderr io.Writer) (int, error) {
	args := append([]string{"plan", "-input=false", "-out=" + planPath}, FilterDisallowedArgs(extraArgs)...)
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = workDir
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err == nil {
		return 0, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), nil
	}
	if ctx.Err() != nil {
		return -1, ctx.Err()
	}
	return -1, fmt.Errorf("run terraform plan: %w", err)
}

// ShowJSON runs `terraform show -json planPath` and returns its stdout.
func ShowJSON(ctx context.Context, bin, workDir, planPath string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, bin, "show", "-json", planPath)
	cmd.Dir = workDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("terraform show -json: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

// Version runs `terraform version -json` style fallback via plain
// `terraform version` and returns the first line, used only for
// diagnostics when the plan JSON doesn't carry a version (e.g.
// --plan-file).
func Version(ctx context.Context, bin, workDir string) (string, error) {
	cmd := exec.CommandContext(ctx, bin, "version")
	cmd.Dir = workDir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("terraform version: %w", err)
	}
	line := strings.SplitN(stdout.String(), "\n", 2)[0]
	return strings.TrimSpace(line), nil
}
