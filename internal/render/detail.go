package render

import (
	"fmt"
	"io"

	"github.com/Fanatic-zer0/tfsum/internal/color"
	"github.com/Fanatic-zer0/tfsum/internal/plan"
)

// DetailedResource renders the full attribute-level view of a single
// resource change: action, risk, provider, replacement reasons, every
// changed attribute, impact notes, and a recommendation.
func DetailedResource(w io.Writer, rc *plan.ResourceChange, theme *color.Theme, maxWidth int) error {
	wd := width(maxWidth)
	sym := actionSymbol(theme.Symbols, rc.ActionKind)
	style := actionStyle(theme, rc.ActionKind)

	fmt.Fprintln(w, rule(theme, wd))
	fmt.Fprintf(w, "%s %s\n", style.Render(sym), theme.Address.Render(rc.Address))
	fmt.Fprintln(w, rule(theme, wd))
	fmt.Fprintln(w)

	level := rc.Risk.Level.String()
	fmt.Fprintf(w, "Action:      %s\n", actionDescription(rc))
	fmt.Fprintf(w, "Risk:        %s\n", theme.StyleForRisk(level).Render(level))
	fmt.Fprintf(w, "Provider:    %s\n", valueOr(rc.Provider, "unknown"))
	fmt.Fprintf(w, "Resource:    %s\n", rc.Type)
	fmt.Fprintf(w, "Module:      %s\n", moduleLabel(rc.Module))
	fmt.Fprintln(w)

	if rc.ActionKind == plan.ActionReplace && len(rc.ReplacePaths) > 0 {
		fmt.Fprintln(w, "Why replaced:")
		for _, p := range rc.ReplacePaths {
			fmt.Fprintf(w, "  %s %s changed\n", theme.Symbols.Bullet, theme.AttrPath.Render(p))
			fmt.Fprintf(w, "  %s %s forces replacement\n", theme.Symbols.Bullet, theme.AttrPath.Render(p))
		}
		fmt.Fprintln(w)
	}

	if len(rc.AttributeDiffs) > 0 {
		fmt.Fprintln(w, "Attribute changes:")
		fmt.Fprintln(w)
		for _, ad := range rc.AttributeDiffs {
			printAttributeDiff(w, theme, ad)
		}
	}

	if impact := impactBullets(rc); len(impact) > 0 {
		fmt.Fprintln(w, "Impact:")
		for _, line := range impact {
			fmt.Fprintf(w, "  %s %s\n", theme.Symbols.Bullet, line)
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, "Recommendation:")
	fmt.Fprintf(w, "  %s\n", recommendation(rc.Risk.Level))
	fmt.Fprintln(w)
	return nil
}

func actionDescription(rc *plan.ResourceChange) string {
	switch rc.ActionKind {
	case plan.ActionReplace:
		if containsAction(rc.Actions, "create") && len(rc.Actions) > 0 && rc.Actions[0] == "create" {
			return "REPLACE (create → delete)"
		}
		return "REPLACE (delete → create)"
	default:
		return actionLabel(rc.ActionKind)
	}
}

func containsAction(actions []string, a string) bool {
	for _, s := range actions {
		if s == a {
			return true
		}
	}
	return false
}

func printAttributeDiff(w io.Writer, theme *color.Theme, ad plan.AttributeDiff) {
	switch {
	case ad.Sensitive:
		fmt.Fprintf(w, "  %s %s\n", theme.Sensitive.Render(theme.Symbols.Sensitive), theme.AttrPath.Render(ad.Path))
		fmt.Fprintf(w, "      value:   %s\n", theme.Sensitive.Render("SENSITIVE \u2014 redacted"))
	case ad.Unknown:
		fmt.Fprintf(w, "  %s %s\n", theme.Unknown.Render(theme.Symbols.Unknown), theme.AttrPath.Render(ad.Path))
		fmt.Fprintf(w, "      after:   %s\n", theme.Unknown.Render("KNOWN AFTER APPLY"))
	case ad.ChangeType == "added":
		fmt.Fprintf(w, "  %s %s\n", theme.Create.Render(theme.Symbols.Create), theme.AttrPath.Render(ad.Path))
		fmt.Fprintf(w, "      before:  %s\n", theme.Dim.Render("(absent)"))
		fmt.Fprintf(w, "      after:   %s\n", theme.NewValue.Render(formatValue(ad.After)))
	case ad.ChangeType == "removed":
		fmt.Fprintf(w, "  %s %s\n", theme.Destroy.Render(theme.Symbols.Destroy), theme.AttrPath.Render(ad.Path))
		fmt.Fprintf(w, "      before:  %s\n", theme.OldValue.Render(formatValue(ad.Before)))
		fmt.Fprintf(w, "      after:   %s\n", theme.Dim.Render("(absent)"))
	default:
		fmt.Fprintf(w, "  %s %s\n", theme.Update.Render(theme.Symbols.Update), theme.AttrPath.Render(ad.Path))
		fmt.Fprintf(w, "      before:  %s\n", theme.OldValue.Render(formatValue(ad.Before)))
		fmt.Fprintf(w, "      after:   %s\n", theme.NewValue.Render(formatValue(ad.After)))
	}
	fmt.Fprintln(w)
}

func impactBullets(rc *plan.ResourceChange) []string {
	switch rc.ActionKind {
	case plan.ActionReplace:
		return []string{
			"This resource will be destroyed and recreated; dependent resources may be affected.",
			"Verify backups or snapshots exist before applying, if applicable.",
			"Confirm any maintenance window or downtime with the resource owner.",
		}
	case plan.ActionDelete:
		return []string{"This resource and its data will be permanently destroyed."}
	default:
		return nil
	}
}

func recommendation(level plan.RiskLevel) string {
	switch level {
	case plan.RiskCritical:
		return "Require manual approval before applying this change."
	case plan.RiskHigh:
		return "Review carefully before applying; consider requiring manual approval."
	case plan.RiskMedium:
		return "Review this change before applying."
	default:
		return "No special action required."
	}
}
