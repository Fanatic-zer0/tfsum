// Package cli implements tfsum's command-line flag parsing (including
// Terraform flag passthrough) and top-level run orchestration.
package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Fanatic-zer0/tfsum/internal/color"
	"github.com/Fanatic-zer0/tfsum/internal/plan"
	"github.com/Fanatic-zer0/tfsum/internal/render"
)

// Options holds every tfsum-specific flag, plus every other argument
// (terraform's own flags and positional args) collected unchanged for
// passthrough to `terraform plan`.
type Options struct {
	Detailed          bool
	Table             bool
	View              render.View
	ColorMode         color.Mode
	NoColor           bool
	KeepPlan          bool
	PlanFile          string
	FailOn            plan.RiskLevel
	FailOnDestructive bool
	FailOnReplacement bool
	MaxWidth          int
	ConfigPath        string
	TerraformBin      string
	Debug             bool
	Verbose           bool
	Help              bool
	Version           bool

	TerraformArgs []string
}

func defaultOptions() *Options {
	return &Options{
		View:      render.ViewReviewer,
		ColorMode: color.ModeAuto,
		FailOn:    plan.RiskNone,
	}
}

// ParseArgs splits args into tfsum's own flags and everything else
// (forwarded verbatim to `terraform plan`). Cobra's standard flag parser
// is not used here: it rejects unrecognized flags, and Terraform itself
// uses single-dash flags (-target, -var-file) that are easy to misparse
// against tfsum's double-dash flags. Instead this performs an explicit
// pre-scan: tfsum flags are recognized by exact name; anything else
// (recognized or not, single- or double-dash, positional) is forwarded.
func ParseArgs(args []string) (*Options, error) {
	opts := defaultOptions()
	var terraformArgs []string

	i := 0
	for i < len(args) {
		a := args[i]
		var (
			val  string
			next int
			err  error
		)
		switch {
		case a == "--help" || a == "-h":
			opts.Help = true
			i++
			continue
		case a == "--version":
			opts.Version = true
			i++
			continue
		case a == "--detailed":
			opts.Detailed = true
			i++
			continue
		case a == "--table":
			opts.Table = true
			i++
			continue
		case a == "--no-color":
			opts.NoColor = true
			i++
			continue
		case a == "--keep-plan":
			opts.KeepPlan = true
			i++
			continue
		case a == "--fail-on-destructive":
			opts.FailOnDestructive = true
			i++
			continue
		case a == "--fail-on-replacement":
			opts.FailOnReplacement = true
			i++
			continue
		case a == "--debug":
			opts.Debug = true
			i++
			continue
		case a == "--verbose":
			opts.Verbose = true
			i++
			continue
		case isFlag(a, "--view"):
			if val, next, err = flagValue(args, i, "--view"); err != nil {
				return nil, err
			}
			v, err := render.ParseView(val)
			if err != nil {
				return nil, err
			}
			opts.View = v
			i = next
			continue
		case isFlag(a, "--color"):
			if val, next, err = flagValue(args, i, "--color"); err != nil {
				return nil, err
			}
			m, err := color.ParseMode(val)
			if err != nil {
				return nil, err
			}
			opts.ColorMode = m
			i = next
			continue
		case isFlag(a, "--plan-file"):
			if val, next, err = flagValue(args, i, "--plan-file"); err != nil {
				return nil, err
			}
			opts.PlanFile = val
			i = next
			continue
		case isFlag(a, "--fail-on"):
			if val, next, err = flagValue(args, i, "--fail-on"); err != nil {
				return nil, err
			}
			lvl, ok := plan.ParseRiskLevel(strings.ToLower(val))
			if !ok {
				return nil, fmt.Errorf("invalid --fail-on value %q (expected none, low, medium, high, or critical)", val)
			}
			opts.FailOn = lvl
			i = next
			continue
		case isFlag(a, "--max-width"):
			if val, next, err = flagValue(args, i, "--max-width"); err != nil {
				return nil, err
			}
			n, convErr := strconv.Atoi(val)
			if convErr != nil || n <= 0 {
				return nil, fmt.Errorf("invalid --max-width value %q: must be a positive integer", val)
			}
			opts.MaxWidth = n
			i = next
			continue
		case isFlag(a, "--config"):
			if val, next, err = flagValue(args, i, "--config"); err != nil {
				return nil, err
			}
			opts.ConfigPath = val
			i = next
			continue
		case isFlag(a, "--terraform-bin"):
			if val, next, err = flagValue(args, i, "--terraform-bin"); err != nil {
				return nil, err
			}
			opts.TerraformBin = val
			i = next
			continue
		default:
			terraformArgs = append(terraformArgs, a)
			i++
		}
	}

	opts.TerraformArgs = terraformArgs
	return opts, nil
}

func isFlag(arg, name string) bool {
	return arg == name || strings.HasPrefix(arg, name+"=")
}

// flagValue extracts the value for the flag at args[i], supporting both
// "--flag value" and "--flag=value" forms, returning the index to resume
// parsing from.
func flagValue(args []string, i int, name string) (string, int, error) {
	a := args[i]
	if strings.HasPrefix(a, name+"=") {
		return a[len(name)+1:], i + 1, nil
	}
	if i+1 >= len(args) {
		return "", 0, fmt.Errorf("flag %s requires a value", name)
	}
	return args[i+1], i + 2, nil
}
