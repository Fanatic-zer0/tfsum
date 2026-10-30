package cli

// Exit codes. Terraform's own exit codes are passed through unchanged for
// plan/init/show failures (see docs/USAGE.md); tfsum reserves this range
// for its own outcomes so they are distinguishable in CI pipelines.
const (
	ExitSuccess       = 0
	ExitTerraformFail = 1
	ExitPolicy        = 2
	ExitInternal      = 3
	// ExitInterrupted is returned when the plan subprocess is interrupted
	// by SIGINT/SIGTERM, following the common `128+signal` convention.
	ExitInterrupted = 130
)
