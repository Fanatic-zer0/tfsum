package render

import (
	"bufio"
	"fmt"
	"io"
	"time"

	"github.com/Fanatic-zer0/tfsum/internal/color"
	"github.com/Fanatic-zer0/tfsum/internal/plan"
)

// ReviewerOptions controls optional Reviewer rendering behavior.
type ReviewerOptions struct {
	MaxWidth int
	// Detailed expands every changed resource with its full attribute diff.
	Detailed bool
	// Table renders each module's resource listing as a color-coded,
	// bordered table instead of the default plain aligned list.
	Table bool
}

// Reviewer renders tfsum's default, detailed-but-grouped report: a
// summary box, change counts, risk findings, per-module impact, and a
// grouped resource listing. When opts.Detailed is true, every changed
// resource is additionally expanded with its full attribute diff.
func Reviewer(w io.Writer, report *plan.PlanReport, theme *color.Theme, opts ReviewerOptions) error {
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	wd := width(opts.MaxWidth)
	s := theme.Symbols

	fmt.Fprintln(bw, box(theme, "TERRAFORM PLAN SUMMARY", wd))
	fmt.Fprintln(bw)
	fmt.Fprintf(bw, "Workspace:  %s\n", valueOr(report.Workspace, "default"))
	fmt.Fprintf(bw, "Terraform:  v%s\n", valueOr(report.TerraformVersion, "unknown"))
	fmt.Fprintf(bw, "Generated:  %s\n", report.GeneratedAt.Format(time.RFC1123))
	fmt.Fprintln(bw)

	changed := changedResourceCount(report)
	fmt.Fprintf(bw, "Changes: %d total", changed)
	if report.Summary.NoOp > 0 {
		fmt.Fprintf(bw, " (%d no-op not counted)", report.Summary.NoOp)
	}
	fmt.Fprintln(bw)
	fmt.Fprintln(bw)
	printCounts(bw, theme, report.Summary)
	fmt.Fprintln(bw)

	fmt.Fprintf(bw, "Risk: %s\n", theme.StyleForRisk(report.Summary.HighestRisk.String()).Render(report.Summary.HighestRisk.String()))
	fmt.Fprintln(bw)

	printFindings(bw, theme, report.Risks)

	if stats := groupModuleStats(report.ResourceChanges); len(stats) > 0 {
		fmt.Fprintln(bw, "Module impact:")
		maxName := 0
		for _, st := range stats {
			if l := len(moduleLabel(st.name)); l > maxName {
				maxName = l
			}
		}
		for _, st := range stats {
			fmt.Fprintf(bw, "  %-*s %s\n", maxName+4, moduleLabel(st.name), joinParts(st.parts()))
		}
		fmt.Fprintln(bw)
	}

	fmt.Fprintf(bw, "Changed resources: %d of %d managed resources\n", changed, report.Summary.TotalResources)
	fmt.Fprintln(bw)

	modules, grouped := groupByModule(report.ResourceChanges)
	for _, m := range modules {
		fmt.Fprintf(bw, "%s %s\n", s.ModuleMarker, moduleLabel(m))
		if opts.Table && !opts.Detailed {
			fmt.Fprintln(bw, resourceTable(theme, grouped[m]))
			fmt.Fprintln(bw)
			continue
		}
		for _, rc := range grouped[m] {
			printResourceLine(bw, theme, rc)
			if opts.Detailed {
				fmt.Fprintln(bw)
				if err := DetailedResource(bw, &rc, theme, wd); err != nil {
					return err
				}
			}
		}
		if !opts.Detailed {
			fmt.Fprintln(bw)
		}
	}

	if len(report.Drift) > 0 {
		fmt.Fprintln(bw)
		printDrift(bw, theme, report.Drift)
	}

	return bw.Flush()
}

func valueOr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func printCounts(w io.Writer, theme *color.Theme, s plan.PlanSummary) {
	sym := theme.Symbols
	if s.Create > 0 {
		fmt.Fprintf(w, "  %s %d to create\n", theme.Create.Render(sym.Create), s.Create)
	}
	if s.Update > 0 {
		fmt.Fprintf(w, "  %s %d to update\n", theme.Update.Render(sym.Update), s.Update)
	}
	if s.Replace > 0 {
		fmt.Fprintf(w, "  %s %d to replace\n", theme.Replace.Render(sym.Replace), s.Replace)
	}
	if s.Delete > 0 {
		fmt.Fprintf(w, "  %s %d to destroy\n", theme.Destroy.Render(sym.Destroy), s.Delete)
	}
	if s.NoOp > 0 {
		fmt.Fprintf(w, "  %s %d no-op\n", theme.Dim.Render(sym.NoOp), s.NoOp)
	}
}

func joinParts(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ", "
		}
		out += p
	}
	return out
}

func printFindings(w io.Writer, theme *color.Theme, risks []plan.RiskFinding) {
	const maxShown = 10
	var shown []plan.RiskFinding
	unknownAddrs := map[string]bool{}
	for _, f := range risks {
		if f.RuleID == "unknown-values" {
			unknownAddrs[f.Address] = true
			continue
		}
		if f.Level >= plan.RiskMedium {
			shown = append(shown, f)
		}
	}
	if len(shown) == 0 && len(unknownAddrs) == 0 {
		return
	}
	fmt.Fprintln(w, "Findings:")
	n := len(shown)
	if n > maxShown {
		n = maxShown
	}
	for _, f := range shown[:n] {
		level := f.Level.String()
		fmt.Fprintf(w, "  %-9s %s: %s\n", theme.StyleForRisk(level).Render(level), theme.Address.Render(f.Address), f.Message)
	}
	if len(shown) > maxShown {
		fmt.Fprintf(w, "  ... and %d more\n", len(shown)-maxShown)
	}
	if len(unknownAddrs) > 0 {
		level := plan.RiskMedium.String()
		fmt.Fprintf(w, "  %-9s %d resources contain unknown values\n", theme.StyleForRisk(level).Render(level), len(unknownAddrs))
	}
	fmt.Fprintln(w)
}

func printResourceLine(w io.Writer, theme *color.Theme, rc plan.ResourceChange) {
	sym := actionSymbol(theme.Symbols, rc.ActionKind)
	style := actionStyle(theme, rc.ActionKind)
	label := actionLabel(rc.ActionKind)
	level := rc.Risk.Level.String()
	fmt.Fprintf(w, "  %-3s %-45s %-8s %s\n",
		style.Render(sym), theme.Address.Render(rc.Address), label, theme.StyleForRisk(level).Render(level))
}

func printDrift(w io.Writer, theme *color.Theme, drift []plan.ResourceDrift) {
	fmt.Fprintf(w, "%s DRIFT DETECTED\n\n", theme.Drift.Render(theme.Symbols.Drift))
	for _, d := range drift {
		fmt.Fprintf(w, "%s\n", theme.Address.Render(d.Address))
		fmt.Fprintln(w, "  Actual infrastructure differs from Terraform state.")
		fmt.Fprintln(w)
		for _, ad := range d.AttributeDiffs {
			fmt.Fprintf(w, "  %s %s\n", theme.Drift.Render("~"), theme.AttrPath.Render(ad.Path))
			fmt.Fprintf(w, "      state:   %s\n", theme.OldValue.Render(formatValue(ad.Before)))
			fmt.Fprintf(w, "      actual:  %s\n", theme.NewValue.Render(formatValue(ad.After)))
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w, "This resource was modified outside Terraform.")
}
