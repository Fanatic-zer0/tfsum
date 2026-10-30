// Package render turns a parsed, risk-annotated plan.PlanReport into
// tfsum's terminal (executive/reviewer/full) and machine-readable (json)
// output formats.
package render

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"tfsum/internal/color"
	"tfsum/internal/plan"
)

// View selects which report layout to render.
type View string

const (
	ViewExecutive View = "executive"
	ViewReviewer  View = "reviewer"
	ViewFull      View = "full"
	ViewJSON      View = "json"
)

// ParseView validates a --view flag value.
func ParseView(s string) (View, error) {
	switch View(s) {
	case ViewExecutive, ViewReviewer, ViewFull, ViewJSON:
		return View(s), nil
	default:
		return "", fmt.Errorf("invalid --view value %q (expected executive, reviewer, full, or json)", s)
	}
}

const defaultMaxWidth = 100

// width clamps a requested terminal width to a sane minimum.
func width(maxWidth int) int {
	if maxWidth <= 0 {
		return defaultMaxWidth
	}
	if maxWidth < 40 {
		return 40
	}
	return maxWidth
}

// actionLabel returns the upper-case label used for a resource action.
func actionLabel(kind plan.ActionKind) string {
	switch kind {
	case plan.ActionCreate:
		return "CREATE"
	case plan.ActionUpdate:
		return "UPDATE"
	case plan.ActionReplace:
		return "REPLACE"
	case plan.ActionDelete:
		return "DESTROY"
	case plan.ActionRead:
		return "READ"
	case plan.ActionForget:
		return "FORGET"
	default:
		return "NO-OP"
	}
}

// actionSymbol returns the change-kind glyph for a resource action.
func actionSymbol(s color.Symbols, kind plan.ActionKind) string {
	switch kind {
	case plan.ActionCreate:
		return s.Create
	case plan.ActionUpdate:
		return s.Update
	case plan.ActionReplace:
		return s.Replace
	case plan.ActionDelete:
		return s.Destroy
	default:
		return s.NoOp
	}
}

// actionStyle returns the color style for a resource action.
func actionStyle(t *color.Theme, kind plan.ActionKind) lipgloss.Style {
	switch kind {
	case plan.ActionCreate:
		return t.Create
	case plan.ActionUpdate:
		return t.Update
	case plan.ActionReplace:
		return t.Replace
	case plan.ActionDelete:
		return t.Destroy
	default:
		return t.Dim
	}
}

// isChangeKind reports whether kind represents an actual planned change
// (as opposed to no-op/read), i.e. whether it should appear in "changed
// resources" counts and listings.
func isChangeKind(kind plan.ActionKind) bool {
	switch kind {
	case plan.ActionCreate, plan.ActionUpdate, plan.ActionReplace, plan.ActionDelete:
		return true
	default:
		return false
	}
}

// changedResourceCount returns the number of resources with an actual
// planned change (create/update/replace/delete), excluding no-ops/reads.
func changedResourceCount(report *plan.PlanReport) int {
	return report.Summary.Create + report.Summary.Update + report.Summary.Replace + report.Summary.Delete
}

type moduleStat struct {
	name    string
	create  int
	update  int
	replace int
	destroy int
}

func (m moduleStat) parts() []string {
	var parts []string
	if m.destroy > 0 {
		parts = append(parts, pluralize(m.destroy, "destroy", "destroys"))
	}
	if m.replace > 0 {
		parts = append(parts, pluralize(m.replace, "replace", "replaces"))
	}
	if m.update > 0 {
		parts = append(parts, pluralize(m.update, "update", "updates"))
	}
	if m.create > 0 {
		parts = append(parts, pluralize(m.create, "create", "creates"))
	}
	return parts
}

func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, singular)
	}
	return fmt.Sprintf("%d %s", n, plural)
}

func moduleLabel(m string) string {
	if m == "" {
		return "(root)"
	}
	return m
}

// groupModuleStats groups changed resources by module and tallies their
// action kinds, returning modules sorted by name with root ("(root)")
// first.
func groupModuleStats(resources []plan.ResourceChange) []moduleStat {
	stats := map[string]*moduleStat{}
	for _, rc := range resources {
		if !isChangeKind(rc.ActionKind) {
			continue
		}
		s, ok := stats[rc.Module]
		if !ok {
			s = &moduleStat{name: rc.Module}
			stats[rc.Module] = s
		}
		switch rc.ActionKind {
		case plan.ActionCreate:
			s.create++
		case plan.ActionUpdate:
			s.update++
		case plan.ActionReplace:
			s.replace++
		case plan.ActionDelete:
			s.destroy++
		}
	}
	out := make([]moduleStat, 0, len(stats))
	for _, s := range stats {
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

// groupByModule groups changed resources by module, sorted by address
// within each module, with modules sorted by name (root first).
func groupByModule(resources []plan.ResourceChange) ([]string, map[string][]plan.ResourceChange) {
	grouped := map[string][]plan.ResourceChange{}
	for _, rc := range resources {
		if !isChangeKind(rc.ActionKind) {
			continue
		}
		grouped[rc.Module] = append(grouped[rc.Module], rc)
	}
	modules := make([]string, 0, len(grouped))
	for m, list := range grouped {
		sort.Slice(list, func(i, j int) bool { return list[i].Address < list[j].Address })
		grouped[m] = list
		modules = append(modules, m)
	}
	sort.Strings(modules)
	return modules, grouped
}

// formatValue renders a raw attribute value (string, number, bool,
// list, map, or nil) for display.
func formatValue(v any) string {
	switch t := v.(type) {
	case nil:
		return "(absent)"
	case string:
		return fmt.Sprintf("%q", t)
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return fmt.Sprintf("%g", t)
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return fmt.Sprintf("%v", t)
		}
		return string(b)
	}
}

func box(theme *color.Theme, title string, w int) string {
	s := theme.Symbols
	inner := w - 2
	if inner < len(title)+2 {
		inner = len(title) + 2
	}
	top := s.BoxTopLeft + strings.Repeat(s.BoxHorizontal, inner) + s.BoxTopRight
	mid := s.BoxVertical + " " + title + strings.Repeat(" ", inner-len(title)-1) + s.BoxVertical
	bottom := s.BoxBottomLeft + strings.Repeat(s.BoxHorizontal, inner) + s.BoxBottomRight
	return theme.Header.Render(top) + "\n" + theme.Header.Render(mid) + "\n" + theme.Header.Render(bottom)
}

func rule(theme *color.Theme, w int) string {
	return theme.Dim.Render(strings.Repeat(theme.Symbols.Rule, w))
}
