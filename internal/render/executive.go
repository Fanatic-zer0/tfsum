package render

import (
	"bufio"
	"fmt"
	"io"

	"github.com/Fanatic-zer0/tfsum/internal/color"
	"github.com/Fanatic-zer0/tfsum/internal/plan"
)

// Executive renders tfsum's high-level view: risk, change counts, critical
// findings, destructive changes, and a single recommendation line. It
// never prints per-attribute diffs.
func Executive(w io.Writer, report *plan.PlanReport, theme *color.Theme) error {
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	fmt.Fprintf(bw, "TERRAFORM PLAN %s %s\n\n", theme.Symbols.Arrow, valueOr(report.Workspace, "default"))

	level := report.Summary.HighestRisk.String()
	fmt.Fprintf(bw, "Risk: %s\n", theme.StyleForRisk(level).Render(level))
	fmt.Fprintf(bw, "Changes: %d\n\n", changedResourceCount(report))

	s := report.Summary
	sym := theme.Symbols
	if s.Create > 0 {
		fmt.Fprintf(bw, "  %s %d create\n", theme.Create.Render(sym.Create), s.Create)
	}
	if s.Update > 0 {
		fmt.Fprintf(bw, "  %s %d update\n", theme.Update.Render(sym.Update), s.Update)
	}
	if s.Replace > 0 {
		fmt.Fprintf(bw, "  %s %d replace\n", theme.Replace.Render(sym.Replace), s.Replace)
	}
	if s.Delete > 0 {
		fmt.Fprintf(bw, "  %s %d destroy\n", theme.Destroy.Render(sym.Destroy), s.Delete)
	}
	fmt.Fprintln(bw)

	var critical []plan.RiskFinding
	for _, f := range report.Risks {
		if f.Level == plan.RiskCritical {
			critical = append(critical, f)
		}
	}
	if len(critical) > 0 {
		fmt.Fprintln(bw, "Critical findings:")
		for _, f := range critical {
			fmt.Fprintf(bw, "  %s %s: %s\n", sym.Bullet, theme.Address.Render(f.Address), f.Message)
		}
		fmt.Fprintln(bw)
	}

	var destructive []string
	for _, rc := range report.ResourceChanges {
		if rc.ActionKind == plan.ActionDelete || rc.ActionKind == plan.ActionReplace {
			destructive = append(destructive, rc.Address)
		}
	}
	if len(destructive) > 0 {
		fmt.Fprintln(bw, "Destructive changes:")
		for _, addr := range destructive {
			fmt.Fprintf(bw, "  %s %s\n", sym.Bullet, theme.Address.Render(addr))
		}
		fmt.Fprintln(bw)
	}

	fmt.Fprintln(bw, "Recommendation:")
	fmt.Fprintf(bw, "  %s\n", recommendationLabel(report.Summary.HighestRisk))

	return bw.Flush()
}

func recommendationLabel(level plan.RiskLevel) string {
	switch {
	case level >= plan.RiskHigh:
		return "MANUAL APPROVAL REQUIRED"
	case level == plan.RiskMedium:
		return "REVIEW RECOMMENDED"
	default:
		return "SAFE TO APPLY"
	}
}
