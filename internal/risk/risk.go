// Package risk implements tfsum's deterministic, YAML-configurable risk
// rule engine. It never uses an LLM to decide risk: every finding is the
// result of an explicit, auditable rule match.
package risk

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Fanatic-zer0/tfsum/internal/plan"
)

// RuleMatch declares the conditions under which a Rule applies to a
// resource change. An empty RuleMatch never matches anything, to avoid an
// accidental catch-all rule.
type RuleMatch struct {
	ResourceTypes     []string `yaml:"resource_types,omitempty"`
	Actions           []string `yaml:"actions,omitempty"`
	Attribute         string   `yaml:"attribute,omitempty"`
	Contains          string   `yaml:"contains,omitempty"`
	ChangedAttributes []string `yaml:"changed_attributes,omitempty"`
	Unknown           bool     `yaml:"unknown,omitempty"`
}

// Rule is a single deterministic risk rule, either built-in or loaded from
// YAML via --config.
type Rule struct {
	ID       string    `yaml:"id"`
	Match    RuleMatch `yaml:"match"`
	Severity string    `yaml:"severity"`
	Message  string    `yaml:"message"`
	// Override must be set when a user-supplied rule reuses a built-in
	// rule's ID; otherwise MergeRules rejects the config.
	Override bool `yaml:"override,omitempty"`
}

func (r Rule) level() plan.RiskLevel {
	lvl, _ := plan.ParseRiskLevel(strings.ToLower(r.Severity))
	return lvl
}

// MergeRules combines built-in default rules with user-supplied rules from
// --config. A user rule may override a built-in rule of the same ID only
// if it explicitly sets Override: true; otherwise the duplicate ID is a
// configuration error.
func MergeRules(defaults, user []Rule) ([]Rule, error) {
	merged := append([]Rule(nil), defaults...)
	index := make(map[string]int, len(merged))
	for i, r := range merged {
		index[r.ID] = i
	}
	for _, ur := range user {
		if ur.ID == "" {
			return nil, fmt.Errorf("config rule is missing an id")
		}
		if idx, exists := index[ur.ID]; exists {
			if !ur.Override {
				return nil, fmt.Errorf("config rule %q reuses a built-in rule id; set override: true to replace it", ur.ID)
			}
			merged[idx] = ur
			continue
		}
		merged = append(merged, ur)
		index[ur.ID] = len(merged) - 1
	}
	return merged, nil
}

// Apply evaluates every rule against every resource change and drift entry
// in report, populating each ResourceChange.Risk / ResourceDrift.Risk with
// its single highest-severity finding, report.Risks with every finding
// (for display), and report.Summary.HighestRisk with the plan-wide max.
func Apply(report *plan.PlanReport, rules []Rule) {
	report.Risks = nil
	highest := plan.RiskNone

	for i := range report.ResourceChanges {
		rc := &report.ResourceChanges[i]
		findings := evaluateResource(rc, rules)
		if len(findings) == 0 {
			continue
		}

		best := findings[0]
		reasons := make([]string, 0, len(findings))
		for _, f := range findings {
			if f.Level > best.Level {
				best = f
			}
			reasons = append(reasons, f.Reasons...)
		}
		best.Reasons = reasons
		rc.Risk = best

		report.Risks = append(report.Risks, findings...)
		if best.Level > highest {
			highest = best.Level
		}
	}

	for i := range report.Drift {
		rd := &report.Drift[i]
		rd.Risk = plan.RiskFinding{
			Level:   plan.RiskMedium,
			RuleID:  "drift-detected",
			Message: "Actual infrastructure differs from Terraform state.",
			Address: rd.Address,
		}
		if rd.Risk.Level > highest {
			highest = rd.Risk.Level
		}
	}

	sort.SliceStable(report.Risks, func(i, j int) bool {
		if report.Risks[i].Level != report.Risks[j].Level {
			return report.Risks[i].Level > report.Risks[j].Level
		}
		return report.Risks[i].Address < report.Risks[j].Address
	})

	report.Summary.HighestRisk = highest
}

func evaluateResource(rc *plan.ResourceChange, rules []Rule) []plan.RiskFinding {
	var findings []plan.RiskFinding
	for _, r := range rules {
		if !matches(rc, r) {
			continue
		}
		findings = append(findings, plan.RiskFinding{
			Level:   r.level(),
			RuleID:  r.ID,
			Message: r.Message,
			Address: rc.Address,
			Reasons: reasonsFor(rc, r),
		})
	}
	return findings
}

func matches(rc *plan.ResourceChange, r Rule) bool {
	m := r.Match
	if isEmptyMatch(m) {
		return false
	}
	if len(m.ResourceTypes) > 0 && !containsStr(m.ResourceTypes, rc.Type) {
		return false
	}
	if len(m.Actions) > 0 && !actionsEqual(rc.Actions, m.Actions) {
		return false
	}
	if m.Unknown && !hasUnknownAttr(rc.AttributeDiffs) {
		return false
	}
	if m.Attribute != "" {
		attrs := findAttribute(rc.AttributeDiffs, m.Attribute)
		if len(attrs) == 0 {
			return false
		}
		if m.Contains != "" && !anyContains(attrs, m.Contains) {
			return false
		}
	}
	if len(m.ChangedAttributes) > 0 && !onlyChangedAttributesWithin(rc.AttributeDiffs, m.ChangedAttributes) {
		return false
	}
	return true
}

func isEmptyMatch(m RuleMatch) bool {
	return len(m.ResourceTypes) == 0 && len(m.Actions) == 0 && m.Attribute == "" &&
		m.Contains == "" && len(m.ChangedAttributes) == 0 && !m.Unknown
}

func reasonsFor(rc *plan.ResourceChange, r Rule) []string {
	var reasons []string
	if len(r.Match.ResourceTypes) > 0 {
		reasons = append(reasons, fmt.Sprintf("resource type %q matched rule %q", rc.Type, r.ID))
	}
	if len(r.Match.Actions) > 0 {
		reasons = append(reasons, fmt.Sprintf("actions %v matched rule %q", rc.Actions, r.ID))
	}
	if r.Match.Attribute != "" {
		reasons = append(reasons, fmt.Sprintf("attribute %q matched rule %q", r.Match.Attribute, r.ID))
	}
	if len(r.Match.ChangedAttributes) > 0 {
		reasons = append(reasons, fmt.Sprintf("only attributes %v changed", r.Match.ChangedAttributes))
	}
	if r.Match.Unknown {
		reasons = append(reasons, "resource contains unknown (known-after-apply) values")
	}
	return reasons
}

func containsStr(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

func actionsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	as := append([]string(nil), a...)
	bs := append([]string(nil), b...)
	sort.Strings(as)
	sort.Strings(bs)
	for i := range as {
		if as[i] != bs[i] {
			return false
		}
	}
	return true
}

func hasUnknownAttr(diffs []plan.AttributeDiff) bool {
	for _, d := range diffs {
		if d.Unknown {
			return true
		}
	}
	return false
}

// findAttribute returns every attribute diff whose final path segment
// (ignoring list/set indexes) matches name, e.g. "ingress[0].cidr_blocks"
// matches the name "cidr_blocks".
func findAttribute(diffs []plan.AttributeDiff, name string) []plan.AttributeDiff {
	var out []plan.AttributeDiff
	for _, d := range diffs {
		if lastSegment(d.Path) == name {
			out = append(out, d)
		}
	}
	return out
}

func lastSegment(path string) string {
	if idx := strings.LastIndexByte(path, '.'); idx >= 0 {
		path = path[idx+1:]
	}
	if idx := strings.IndexByte(path, '['); idx >= 0 {
		path = path[:idx]
	}
	return path
}

func anyContains(diffs []plan.AttributeDiff, substr string) bool {
	for _, d := range diffs {
		if valueContains(d.After, substr) || valueContains(d.Before, substr) {
			return true
		}
	}
	return false
}

func valueContains(v any, substr string) bool {
	switch t := v.(type) {
	case string:
		return strings.Contains(t, substr)
	case []any:
		for _, e := range t {
			if valueContains(e, substr) {
				return true
			}
		}
	}
	return false
}

// onlyChangedAttributesWithin reports whether every changed attribute's
// top-level name is contained in allowed, and at least one attribute
// changed.
func onlyChangedAttributesWithin(diffs []plan.AttributeDiff, allowed []string) bool {
	if len(diffs) == 0 {
		return false
	}
	allowedSet := make(map[string]bool, len(allowed))
	for _, a := range allowed {
		allowedSet[a] = true
	}
	for _, d := range diffs {
		top := d.Path
		if idx := strings.IndexAny(top, ".["); idx >= 0 {
			top = top[:idx]
		}
		if !allowedSet[top] {
			return false
		}
	}
	return true
}
