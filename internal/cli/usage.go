package cli

// usageText is printed for --help/-h. It intentionally stays short;
// docs/USAGE.md has the full reference.
const usageText = `tfsum - a detailed, color-coded, risk-aware alternative to "terraform plan"

Usage:
  tfsum [tfsum flags] [terraform plan flags...]

tfsum flags:
  --detailed                 Show full attribute-level diffs in the default view.
  --table                    Render each module's resource listing as a color-coded, bordered table.
  --view string              executive|reviewer|full|json (default "reviewer")
  --no-color                 Disable ANSI colors.
  --color string             auto|always|never (default "auto")
  --keep-plan                Retain the temporary Terraform plan file and print its path.
  --plan-file string         Read an existing saved Terraform plan instead of running plan.
  --fail-on string           none|low|medium|high|critical (default "none")
  --fail-on-destructive      Exit non-zero if any resource will be destroyed.
  --fail-on-replacement      Exit non-zero if any resource will be replaced.
  --max-width int            Terminal wrap width (default: terminal width or 100).
  --config string            Load risk rules and display preferences from YAML.
  --terraform-bin string     Path to the Terraform (or OpenTofu) binary to invoke.
  --debug                    Print full debug details instead of a sanitized error report.
  --verbose                  Stream Terraform's own plan output live instead of only the tfsum report.
  -h, --help                 Show this help message.
  --version                  Print the tfsum version.

Any other flag or argument is forwarded unchanged to "terraform plan".

By default, Terraform's own plan progress output is captured, not shown;
the report starts at TERRAFORM PLAN SUMMARY. If the plan fails, Terraform's
output is still shown inside the error report. Pass --verbose to see
Terraform's live output as it runs.

Examples:
  tfsum
  tfsum --detailed
  tfsum --view executive
  tfsum --view json
  tfsum --var-file=prod.tfvars
  tfsum -target=module.network
  tfsum --fail-on=critical
  tfsum --plan-file=tfplan
`
