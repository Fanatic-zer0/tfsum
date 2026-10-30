package render

import (
	"encoding/json"
	"io"

	"github.com/Fanatic-zer0/tfsum/internal/plan"
)

type jsonSummary struct {
	Total       int    `json:"total"`
	Create      int    `json:"create"`
	Update      int    `json:"update"`
	Replace     int    `json:"replace"`
	Delete      int    `json:"delete"`
	NoOp        int    `json:"no_op"`
	HighestRisk string `json:"highest_risk"`
}

type jsonRiskFinding struct {
	Level   string   `json:"level"`
	RuleID  string   `json:"rule_id"`
	Message string   `json:"message"`
	Address string   `json:"address,omitempty"`
	Reasons []string `json:"reasons,omitempty"`
}

type jsonAttributeDiff struct {
	Path       string `json:"path"`
	Before     any    `json:"before,omitempty"`
	After      any    `json:"after,omitempty"`
	Unknown    bool   `json:"unknown,omitempty"`
	Sensitive  bool   `json:"sensitive,omitempty"`
	ChangeType string `json:"change_type"`
}

type jsonResource struct {
	Address        string              `json:"address"`
	Module         string              `json:"module,omitempty"`
	Type           string              `json:"type"`
	Name           string              `json:"name"`
	Provider       string              `json:"provider,omitempty"`
	Actions        []string            `json:"actions"`
	ActionKind     string              `json:"action_kind"`
	ReplacePaths   []string            `json:"replace_paths,omitempty"`
	Risk           jsonRiskFinding     `json:"risk"`
	AttributeDiffs []jsonAttributeDiff `json:"attribute_diffs,omitempty"`
}

type jsonOutputChange struct {
	Name       string `json:"name"`
	ActionKind string `json:"action_kind"`
	Sensitive  bool   `json:"sensitive"`
}

type jsonDrift struct {
	Address        string              `json:"address"`
	Module         string              `json:"module,omitempty"`
	Type           string              `json:"type"`
	Risk           jsonRiskFinding     `json:"risk"`
	AttributeDiffs []jsonAttributeDiff `json:"attribute_diffs,omitempty"`
}

type jsonReport struct {
	TerraformVersion string             `json:"terraform_version"`
	Workspace        string             `json:"workspace"`
	Summary          jsonSummary        `json:"summary"`
	Risks            []jsonRiskFinding  `json:"risks"`
	Resources        []jsonResource     `json:"resources"`
	Outputs          []jsonOutputChange `json:"outputs"`
	Drift            []jsonDrift        `json:"drift"`
}

// JSON renders report as machine-readable JSON. Output never contains
// ANSI escape codes regardless of the active color theme.
func JSON(w io.Writer, report *plan.PlanReport) error {
	out := jsonReport{
		TerraformVersion: report.TerraformVersion,
		Workspace:        report.Workspace,
		Summary: jsonSummary{
			Total:       changedResourceCount(report),
			Create:      report.Summary.Create,
			Update:      report.Summary.Update,
			Replace:     report.Summary.Replace,
			Delete:      report.Summary.Delete,
			NoOp:        report.Summary.NoOp,
			HighestRisk: lowerRisk(report.Summary.HighestRisk),
		},
		Risks:     toJSONRisks(report.Risks),
		Resources: make([]jsonResource, 0, len(report.ResourceChanges)),
		Outputs:   make([]jsonOutputChange, 0, len(report.OutputChanges)),
		Drift:     make([]jsonDrift, 0, len(report.Drift)),
	}

	for _, rc := range report.ResourceChanges {
		out.Resources = append(out.Resources, jsonResource{
			Address:        rc.Address,
			Module:         rc.Module,
			Type:           rc.Type,
			Name:           rc.Name,
			Provider:       rc.Provider,
			Actions:        rc.Actions,
			ActionKind:     string(rc.ActionKind),
			ReplacePaths:   rc.ReplacePaths,
			Risk:           toJSONRisk(rc.Risk),
			AttributeDiffs: toJSONAttrDiffs(rc.AttributeDiffs),
		})
	}
	for _, oc := range report.OutputChanges {
		out.Outputs = append(out.Outputs, jsonOutputChange{
			Name:       oc.Name,
			ActionKind: string(oc.ActionKind),
			Sensitive:  oc.Sensitive,
		})
	}
	for _, d := range report.Drift {
		out.Drift = append(out.Drift, jsonDrift{
			Address:        d.Address,
			Module:         d.Module,
			Type:           d.Type,
			Risk:           toJSONRisk(d.Risk),
			AttributeDiffs: toJSONAttrDiffs(d.AttributeDiffs),
		})
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(out)
}

func lowerRisk(l plan.RiskLevel) string {
	switch l {
	case plan.RiskCritical:
		return "critical"
	case plan.RiskHigh:
		return "high"
	case plan.RiskMedium:
		return "medium"
	case plan.RiskLow:
		return "low"
	default:
		return "none"
	}
}

func toJSONRisk(f plan.RiskFinding) jsonRiskFinding {
	return jsonRiskFinding{
		Level:   lowerRisk(f.Level),
		RuleID:  f.RuleID,
		Message: f.Message,
		Address: f.Address,
		Reasons: f.Reasons,
	}
}

func toJSONRisks(risks []plan.RiskFinding) []jsonRiskFinding {
	out := make([]jsonRiskFinding, 0, len(risks))
	for _, r := range risks {
		out = append(out, toJSONRisk(r))
	}
	return out
}

func toJSONAttrDiffs(diffs []plan.AttributeDiff) []jsonAttributeDiff {
	out := make([]jsonAttributeDiff, 0, len(diffs))
	for _, d := range diffs {
		out = append(out, jsonAttributeDiff{
			Path:       d.Path,
			Before:     d.Before,
			After:      d.After,
			Unknown:    d.Unknown,
			Sensitive:  d.Sensitive,
			ChangeType: string(d.ChangeType),
		})
	}
	return out
}
