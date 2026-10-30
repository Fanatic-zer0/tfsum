package plan

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	tfjson "github.com/hashicorp/terraform-json"

	"github.com/Fanatic-zer0/tfsum/internal/diff"
)

// supportedMajorFormatVersion is the plan JSON major format version tfsum
// knows how to interpret. Terraform bumps the minor version for additive,
// backwards-compatible changes, so only the major version is enforced.
const supportedMajorFormatVersion = "1"

// Parse decodes the output of `terraform show -json <plan-file>` into a
// PlanReport. Risk findings are left at their zero value; call the risk
// package's Apply function afterwards to populate them.
func Parse(data []byte) (*PlanReport, error) {
	var raw tfjson.Plan
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse terraform plan json: %w", err)
	}
	if err := checkFormatVersion(raw.FormatVersion); err != nil {
		return nil, err
	}

	report := &PlanReport{
		TerraformVersion: raw.TerraformVersion,
		GeneratedAt:      time.Now().UTC(),
	}

	for _, rc := range raw.ResourceChanges {
		if rc == nil || rc.Change == nil {
			continue
		}
		change := buildResourceChange(rc)
		report.ResourceChanges = append(report.ResourceChanges, change)
		tallySummary(&report.Summary, change.ActionKind)
		if hasUnknown(change.AttributeDiffs) {
			report.Summary.HasUnknowns = true
		}
	}
	report.Summary.TotalResources = len(report.ResourceChanges)

	for _, rd := range raw.ResourceDrift {
		if rd == nil || rd.Change == nil {
			continue
		}
		drift := buildDrift(rd)
		if len(drift.AttributeDiffs) == 0 {
			continue
		}
		report.Drift = append(report.Drift, drift)
		report.Summary.HasDrift = true
	}

	for name, oc := range raw.OutputChanges {
		if oc == nil {
			continue
		}
		report.OutputChanges = append(report.OutputChanges, buildOutputChange(name, oc))
	}
	sort.Slice(report.OutputChanges, func(i, j int) bool {
		return report.OutputChanges[i].Name < report.OutputChanges[j].Name
	})

	return report, nil
}

func checkFormatVersion(v string) error {
	if v == "" {
		return fmt.Errorf("unsupported terraform plan: missing format_version")
	}
	major := v
	if idx := strings.IndexByte(v, '.'); idx >= 0 {
		major = v[:idx]
	}
	if major != supportedMajorFormatVersion {
		return fmt.Errorf("unsupported terraform plan format version %q (tfsum supports major version %s.x)", v, supportedMajorFormatVersion)
	}
	return nil
}

func buildResourceChange(rc *tfjson.ResourceChange) ResourceChange {
	actions := make([]string, 0, len(rc.Change.Actions))
	for _, a := range rc.Change.Actions {
		actions = append(actions, string(a))
	}

	before, _ := rc.Change.Before.(map[string]any)
	after, _ := rc.Change.After.(map[string]any)
	unknown, _ := rc.Change.AfterUnknown.(map[string]any)

	// Pass the raw (unasserted) values to the diff engine so a
	// whole-value-unknown or whole-value-sensitive marker (a bare `true`
	// at the root) is not lost by a failed map type assertion.
	attrDiffs := diff.Compute(rc.Change.Before, rc.Change.After, rc.Change.AfterUnknown,
		rc.Change.BeforeSensitive, rc.Change.AfterSensitive)

	return ResourceChange{
		Address:        rc.Address,
		Module:         rc.ModuleAddress,
		Name:           rc.Name,
		Type:           rc.Type,
		Provider:       rc.ProviderName,
		Actions:        actions,
		ActionKind:     actionKind(rc.Change.Actions),
		Before:         before,
		After:          after,
		AfterUnknown:   unknown,
		ReplacePaths:   replacePathStrings(rc.Change.ReplacePaths),
		AttributeDiffs: attrDiffs,
	}
}

func buildDrift(rd *tfjson.ResourceChange) ResourceDrift {
	attrDiffs := diff.Compute(rd.Change.Before, rd.Change.After, rd.Change.AfterUnknown,
		rd.Change.BeforeSensitive, rd.Change.AfterSensitive)
	return ResourceDrift{
		Address:        rd.Address,
		Module:         rd.ModuleAddress,
		Type:           rd.Type,
		AttributeDiffs: attrDiffs,
	}
}

func buildOutputChange(name string, oc *tfjson.Change) OutputChange {
	actions := make([]string, 0, len(oc.Actions))
	for _, a := range oc.Actions {
		actions = append(actions, string(a))
	}
	return OutputChange{
		Name:           name,
		Actions:        actions,
		ActionKind:     actionKind(oc.Actions),
		Before:         oc.Before,
		After:          oc.After,
		Sensitive:      isSensitiveValue(oc.BeforeSensitive) || isSensitiveValue(oc.AfterSensitive),
		AttributeDiffs: diff.Compute(oc.Before, oc.After, oc.AfterUnknown, oc.BeforeSensitive, oc.AfterSensitive),
	}
}

func isSensitiveValue(v any) bool {
	b, ok := v.(bool)
	return ok && b
}

func actionKind(a tfjson.Actions) ActionKind {
	switch {
	case a.Replace():
		return ActionReplace
	case a.Forget():
		return ActionForget
	case a.Create():
		return ActionCreate
	case a.Delete():
		return ActionDelete
	case a.Update():
		return ActionUpdate
	case a.Read():
		return ActionRead
	default:
		return ActionNoOp
	}
}

func tallySummary(s *PlanSummary, kind ActionKind) {
	switch kind {
	case ActionCreate:
		s.Create++
	case ActionUpdate:
		s.Update++
	case ActionReplace:
		s.Replace++
	case ActionDelete:
		s.Delete++
	case ActionRead:
		s.Read++
	case ActionNoOp, ActionForget:
		s.NoOp++
	}
}

func hasUnknown(diffs []AttributeDiff) bool {
	for _, d := range diffs {
		if d.Unknown {
			return true
		}
	}
	return false
}

// replacePathStrings renders Terraform's replace_paths (each a slice of
// string/index segments) using the same dot/bracket notation as
// AttributeDiff.Path.
func replacePathStrings(paths []interface{}) []string {
	if len(paths) == 0 {
		return nil
	}
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		out = append(out, replacePathString(p))
	}
	return out
}

func replacePathString(p interface{}) string {
	segs, ok := p.([]interface{})
	if !ok {
		return fmt.Sprintf("%v", p)
	}
	var sb strings.Builder
	for _, seg := range segs {
		switch v := seg.(type) {
		case string:
			if sb.Len() > 0 {
				sb.WriteByte('.')
			}
			sb.WriteString(v)
		case float64:
			sb.WriteString(fmt.Sprintf("[%d]", int(v)))
		default:
			if sb.Len() > 0 {
				sb.WriteByte('.')
			}
			sb.WriteString(fmt.Sprintf("%v", v))
		}
	}
	return sb.String()
}
