# tfsum usage reference

## Synopsis

```
tfsum [tfsum flags] [terraform plan flags...]
```

By default, `tfsum` behaves like `terraform plan`: it runs a real Terraform
plan in your current working directory and shows you the result — just in a
more structured, color-coded, risk-aware report.

## Flag passthrough

`tfsum` recognizes a fixed set of flags by exact name (listed below). Every
other argument — recognized or not, single- or double-dash, positional or
not — is forwarded **unchanged and in order** to `terraform plan`. This means
you can pass any Terraform plan flag you normally would:

```
tfsum --var-file=prod.tfvars
tfsum -target=module.network
tfsum -var 'environment=staging'
tfsum --detailed -target=module.network --var-file=prod.tfvars
```

tfsum does not use Cobra/pflag's standard flag parser for this split (it
rejects unknown flags and doesn't handle Terraform's single-dash flags
cleanly). Instead it pre-scans arguments itself before anything is forwarded.

One flag is explicitly **not** forwarded: `-detailed-exitcode` /
`--detailed-exitcode`. tfsum manages its own exit-code semantics (see
"Exit codes" below) and that flag would collide with it.

## Flags

| Flag | Description |
|---|---|
| `--detailed` | Show full attribute-level diffs in the default (reviewer) view. |
| `--table` | Render each module's resource listing as a color-coded, bordered table instead of the plain aligned list. |
| `--view <executive\|reviewer\|full\|json>` | Report layout. Default: `reviewer`. |
| `--no-color` | Disable ANSI colors. |
| `--color <auto\|always\|never>` | Color mode. Default: `auto`. |
| `--keep-plan` | Retain the temporary Terraform plan file and print its path. |
| `--plan-file <path>` | Read an existing saved Terraform plan instead of running `terraform plan`. |
| `--fail-on <none\|low\|medium\|high\|critical>` | Exit non-zero when the highest risk meets or exceeds this threshold. Default: `none`. |
| `--fail-on-destructive` | Exit non-zero if any resource will be destroyed. |
| `--fail-on-replacement` | Exit non-zero if any resource will be replaced. |
| `--max-width <int>` | Terminal wrap width. Default: terminal width or 100. |
| `--config <path>` | Load risk rules and display preferences from YAML. If omitted, auto-discovers `.tfsum.yml`/`.tfsum.yaml` in the current directory. |
| `--terraform-bin <path>` | Path to the Terraform (or OpenTofu) binary. Default: `terraform` resolved from `PATH`, or `TF_CLI_BINARY` if set. |
| `--debug` | Print full debug details (not just a sanitized message) on error. |
| `--verbose` | Stream Terraform's own plan progress output live, instead of only showing it if the plan fails. |
| `-h`, `--help` | Show help. |
| `--version` | Print the tfsum version. |

## Views

- **reviewer** (default): a summary box, change counts, risk findings,
  per-module impact, and a grouped resource listing. Pass `--detailed` to
  also expand every changed resource's full attribute diff.
- **executive**: risk, change counts, critical findings, destructive
  changes, and a single recommendation line. Never prints attribute diffs.
- **full**: every changed resource expanded with its full attribute-level
  diff (equivalent to reviewer + `--detailed`).
- **json**: a machine-readable report on stdout. Never contains ANSI escape
  codes, regardless of `--color`.

## Internal execution flow

1. Resolve the Terraform (or OpenTofu) binary.
2. Check for a `.terraform/` directory as a best-effort, non-fatal hint. If
   it's missing, tfsum prints a warning but still proceeds — some valid
   configurations (e.g. using only Terraform's built-in provider) never
   create one. Terraform's own error is authoritative if init is genuinely
   required.
3. Capture the active workspace name via `terraform workspace show` (not
   present in the JSON plan itself).
4. Create a `.tfsum-<random>` directory with `0700` permissions.
5. Run `terraform plan -input=false -out=<tmp>/plan.tfplan`. By default,
   Terraform's own progress/diff output (and its "Saved the plan to: ..." /
   "To perform exactly these actions..." messages) is captured rather than
   shown, so the report starts at `TERRAFORM PLAN SUMMARY`. If the plan
   fails, that captured output is shown inside the error report. Pass
   `--verbose` to stream it live instead, as Terraform normally would. If
   this step is interrupted (Ctrl-C), the temp directory is still cleaned
   up before tfsum exits.
6. Run `terraform show -json <tmp>/plan.tfplan` and parse the result in
   memory — tfsum never parses Terraform's colored human-readable output.
7. Apply the risk rule engine and render the selected view.
8. Delete the temp directory, unless `--keep-plan` was passed (in which case
   the plan file is moved out of the temp directory first and its path is
   printed, along with a reminder that it may contain sensitive state data).

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success; no `--fail-on*` threshold breached. |
| 1 | Terraform plan/init/show failed. Terraform's own exit code is used when available. |
| 2 | A tfsum policy threshold was breached (`--fail-on`, `--fail-on-destructive`, or `--fail-on-replacement`). |
| 3 | A tfsum-internal error occurred (invalid JSON, unsupported plan format version, invalid `--config`, etc.). |
| 130 | The plan was interrupted (SIGINT/SIGTERM) before completing. |

## Configuration file

`--config <path>` (or an auto-discovered `.tfsum.yml`/`.tfsum.yaml` in the
current directory) loads additional YAML risk rules:

```yaml
rules:
  - id: database-replacement
    match:
      resource_types: [aws_db_instance, google_sql_database_instance]
      actions: [delete, create]
    severity: critical
    message: "Database replacement may cause downtime or data loss."

  - id: public-ingress
    match:
      attribute: cidr_blocks
      contains: "0.0.0.0/0"
    severity: critical
    message: "Resource allows traffic from the public internet."

  - id: tag-only
    match:
      changed_attributes: [tags]
    severity: low
    message: "Only tags changed."
```

User rules are merged with tfsum's built-in defaults. A user rule may reuse a
built-in rule's `id` only if it sets `override: true`; otherwise tfsum fails
fast with a configuration error rather than silently shadowing a default.

## CI usage

```
tfsum --view json --fail-on=high > plan-report.json
```

- `--view json` produces a stable, ANSI-free machine-readable report.
- `--fail-on=high` makes the process exit with code `2` if any resource's
  risk is HIGH or CRITICAL, which most CI systems treat as a failed step.
- `--fail-on-destructive` / `--fail-on-replacement` are useful as additional,
  narrower gates (e.g. block any destroy regardless of its assigned risk).
