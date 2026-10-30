# tfsum

`tfsum` is a detailed, color-coded, risk-aware alternative to the default `terraform plan` review experience.

It **runs Terraform's real planner internally** — it is not a replacement for
Terraform's planning engine, only for the command and output experience.
Where you'd normally run:

```sh
terraform plan
```

you instead run:

```sh
tfsum
```

and get a structured, grouped, risk-annotated change report instead of
Terraform's raw diff output.

## Table of contents

- [Why tfsum instead of `terraform plan`](#why-tfsum-instead-of-terraform-plan)
- [Installation](#installation)
- [Usage](#usage)
- [Output](#output)
- [Risk rules](#risk-rules)
- [CI usage](#ci-usage)
- [Security and sensitive-value handling](#security-and-sensitive-value-handling)
- [Limitations](#limitations)
- [Development](#development)



## Default view 

<img width="1377" height="746" alt="image" src="https://github.com/user-attachments/assets/7bbea405-a64d-4742-ba9a-3959938081d9" />

## Detailed view 

<img width="1377" height="746" alt="image" src="https://github.com/user-attachments/assets/94199bf2-4a45-4512-9148-eaf8e5604d38" />


## Why tfsum instead of `terraform plan`

- **Risk-aware**: every resource change is run through a deterministic rule
  engine (no LLM) that flags destructive changes, replacements of stateful
  resources (databases, storage, queues...), security-sensitive resources
  (IAM, security groups, public ingress), and tag-only/no-op noise.
- **Grouped, not flat**: changes are grouped by module and sorted by risk, so
  the one CRITICAL database replacement doesn't get lost in 500 lines of
  diff.
- **Multiple views for different audiences**: `executive` for a one-screen
  go/no-go summary, `reviewer` for day-to-day review, `full`/`--detailed` for
  a complete attribute-level audit, `json` for CI and tooling.
- **Semantic diffs**: nested maps/lists/sets are diffed and shown with
  dot/bracket paths (`settings.backup.retention_days`, `ingress[0].cidr_blocks`),
  not just raw before/after blobs.
- **Safe by default**: sensitive values are redacted, unknown
  (known-after-apply) values are clearly labeled, and colors degrade
  gracefully (truecolor → 256 → ANSI → none) and respect `NO_COLOR`.
- **CI-friendly exit codes**: `--fail-on`, `--fail-on-destructive`, and
  `--fail-on-replacement` let you gate merges/applies without external
  tooling.

It still uses Terraform's own plan JSON output (`terraform show -json`) for
every structured decision — tfsum never parses Terraform's colored
human-readable text.

## Installation

### Homebrew (macOS and Linux)

```sh
brew tap Fanatic-zer0/tfsum
brew install tfsum
```

### Linux packages (.deb/.rpm)

Download the `tfsum_<version>_<arch>.deb` or `.rpm` from the
[releases page](https://github.com/Fanatic-zer0/tfsum/releases) and install
with your distro's package manager, e.g.:

```sh
sudo dpkg -i tfsum_v0.1.0_amd64.deb      # Debian/Ubuntu
sudo rpm -i tfsum_v0.1.0.x86_64.rpm      # Fedora/RHEL
```

### Prebuilt binaries

Download a `.tar.gz`/`.zip` for your platform from the
[releases page](https://github.com/Fanatic-zer0/tfsum/releases), verify
against `checksums.txt`, and place the `tfsum` binary on your `PATH`.

### Build from source

```sh
git clone <this-repository>
cd tfsum
make build      # produces bin/tfsum
# or
go build -o bin/tfsum ./cmd/tfsum
```

Requires Go 1.22+ and a `terraform` (or OpenTofu) binary on `PATH`.

## Usage

```sh
tfsum                               # like `terraform plan`, with a better report
tfsum --detailed                    # full attribute-level diffs
tfsum --view executive              # one-screen summary
tfsum --view json                   # machine-readable report for CI
tfsum --var-file=prod.tfvars        # forwarded straight to `terraform plan`
tfsum -target=module.network        # forwarded straight to `terraform plan`
tfsum --fail-on=critical            # exit non-zero if any CRITICAL risk is found
tfsum --plan-file=tfplan            # convert and report on an already-saved plan
tfsum --table                       # render resource listings as color-coded tables
tfsum --verbose                     # also stream Terraform's own plan progress output live
```

By default, Terraform's own plan progress output (and its "Saved the plan
to: ..." messages) is captured, not shown — the report starts at
`TERRAFORM PLAN SUMMARY`. If the plan fails, that captured output is shown
inside the error report so you don't lose it; pass `--verbose` to stream it
live as Terraform normally would.

See [docs/USAGE.md](docs/USAGE.md) for the full flag reference, exit codes,
and the internal execution flow.

## Output

### Reviewer view (default)

```text
┌──────────────────────────────────────────────────────────────┐
│ TERRAFORM PLAN SUMMARY                                        │
└──────────────────────────────────────────────────────────────┘

Workspace:  production
Terraform:  v1.9.8
Generated:  2026-10-30 12:45:11 UTC

Changes: 18 total (3 no-op not counted)

  + 3 to create
  ~ 12 to update
  -/+ 1 to replace
  - 2 to destroy
  · 3 no-op

Risk: HIGH

Findings:
  CRITICAL  module.db.aws_db_instance.main: Replacement of a stateful resource may cause downtime or data loss.
  CRITICAL  module.network.aws_security_group.web: Resource allows inbound traffic from the public internet (0.0.0.0/0).

Module impact:
  module.database    1 replace, 1 update
  module.network      2 updates
  ...
```

### Color palette

| Kind | Color | Symbol |
|---|---|---|
| Create | green | `+` |
| Update | yellow | `~` |
| Replace | bright magenta | `-/+` |
| Destroy | bright red | `-` |
| Drift | cyan | `!` |
| Unknown (known after apply) | blue | `?` |
| Sensitive | magenta | `***` |
| Risk LOW/MEDIUM/HIGH/CRITICAL | green/yellow/red/bold-white-on-red | — |

Colors degrade gracefully (truecolor → 256-color → ANSI → none) based on
terminal support, and are disabled entirely when `NO_COLOR` is set, when
`--no-color`/`--color=never` is passed, or when stdout isn't a terminal
(unless `--color=always` forces it). Symbols fall back to plain ASCII when
the terminal's locale doesn't advertise UTF-8.

## Risk rules

tfsum ships with a deterministic (non-LLM) default rule set:

- Any `delete` action is CRITICAL.
- Any replacement (`delete`+`create`) is HIGH, or CRITICAL if the resource
  is stateful (databases, storage, queues, stateful load balancers,
  Kubernetes persistent volumes, ...).
- Security-sensitive resources (IAM, security groups, network ACLs, public
  IPs, public access settings) are HIGH.
- Security group rules allowing `0.0.0.0/0`/`::/0` are CRITICAL.
- IAM policy changes granting a wildcard (`*`) action are CRITICAL.
- Tag-only changes are LOW.
- Unknown (known-after-apply) values add a MEDIUM finding.
- Drift is always at least MEDIUM.

Rules are fully custom via `--config` (see [docs/USAGE.md](docs/USAGE.md)).
A user rule can override a built-in rule by ID only by explicitly setting
`override: true`, so config mistakes never silently shadow a default.

## CI usage

```yaml
- name: Review Terraform plan
  run: tfsum --view json --fail-on=high > plan-report.json
```

`tfsum` exits `2` when the configured risk/destructive/replacement threshold
is breached, `1` when Terraform itself fails (using Terraform's own exit
code when available), and `3` on an internal tfsum error. See
[docs/USAGE.md](docs/USAGE.md#exit-codes) for the full table.

## Security and sensitive-value handling

- Terraform's JSON plan output does **not** redact sensitive values in
  `before`/`after` by itself — it marks them via separate
  `before_sensitive`/`after_sensitive` boolean maps. tfsum uses those maps to
  redact every sensitive attribute before it is ever rendered or logged.
- The temporary plan directory (`.tfsum-<random>/`) is created with `0700`
  permissions, and any file written into it (including the saved plan) uses
  `0600` permissions, since a Terraform plan can embed state data containing
  secrets.
- The temp directory is removed on success, on Terraform failure, and on
  interruption (SIGINT/SIGTERM) — not only via `defer`, since a raw process
  kill would otherwise skip it.
- `--keep-plan` retains the plan file and prints its path along with an
  explicit warning: **that file may still contain real secret values** and
  should be handled accordingly (e.g. not committed to version control).

## Limitations

- tfsum supports `terraform plan -destroy` and `-refresh-only` only as
  passthrough arguments; it does not render a specialized view for them.
- `-detailed-exitcode` is not supported as a passthrough flag, since tfsum
  manages its own exit-code semantics (`--fail-on*`) on top of Terraform's.
- tfsum reads the JSON plan's major format version only; it will refuse to
  parse a plan from a future, incompatible major format version rather than
  guessing at its structure.

## Development

```sh
make build         # build bin/tfsum
make test          # go test ./...
make vet           # go vet ./...
make fmt-check     # verify gofmt cleanliness
```
