// Package plan contains tfsum's internal representation of a Terraform
// plan, independent of Terraform's raw JSON schema, along with the parser
// that builds it from `terraform show -json` output.
package plan

import (
	"time"

	"tfsum/internal/diff"
)

// ActionKind classifies the overall effect of a resource change.
type ActionKind string

const (
	ActionCreate  ActionKind = "create"
	ActionUpdate  ActionKind = "update"
	ActionReplace ActionKind = "replace"
	ActionDelete  ActionKind = "delete"
	ActionNoOp    ActionKind = "no-op"
	ActionRead    ActionKind = "read"
	ActionForget  ActionKind = "forget"
)

// RiskLevel is an ordered severity level for risk findings.
type RiskLevel int

const (
	RiskNone RiskLevel = iota
	RiskLow
	RiskMedium
	RiskHigh
	RiskCritical
)

// String renders the risk level using the uppercase labels used throughout
// tfsum's output.
func (r RiskLevel) String() string {
	switch r {
	case RiskCritical:
		return "CRITICAL"
	case RiskHigh:
		return "HIGH"
	case RiskMedium:
		return "MEDIUM"
	case RiskLow:
		return "LOW"
	default:
		return "NONE"
	}
}

// ParseRiskLevel parses a case-insensitive risk level name. It returns
// RiskNone and false if the name is not recognized.
func ParseRiskLevel(s string) (RiskLevel, bool) {
	switch s {
	case "none", "NONE":
		return RiskNone, true
	case "low", "LOW":
		return RiskLow, true
	case "medium", "MEDIUM":
		return RiskMedium, true
	case "high", "HIGH":
		return RiskHigh, true
	case "critical", "CRITICAL":
		return RiskCritical, true
	default:
		return RiskNone, false
	}
}

// RiskFinding is a single deterministic risk-rule match.
type RiskFinding struct {
	Level   RiskLevel
	Reasons []string
	RuleID  string
	Message string
	// Address is the resource address the finding applies to, empty for
	// plan-wide findings (e.g. aggregate unknown-value warnings).
	Address string
}

// AttributeDiff is re-exported so callers only need to import plan.
type AttributeDiff = diff.AttributeDiff

// PlanSummary aggregates resource change counts for a plan.
type PlanSummary struct {
	TotalResources int
	Create         int
	Update         int
	Replace        int
	Delete         int
	NoOp           int
	Read           int
	HighestRisk    RiskLevel
	HasDrift       bool
	HasUnknowns    bool
}

// ResourceChange is tfsum's normalized view of a single planned resource
// change.
type ResourceChange struct {
	Address        string
	Module         string
	Name           string
	Type           string
	Provider       string
	Actions        []string
	ActionKind     ActionKind
	Before         map[string]any
	After          map[string]any
	AfterUnknown   map[string]any
	ReplacePaths   []string
	AttributeDiffs []AttributeDiff
	Risk           RiskFinding
}

// OutputChange describes a change to a root module output value.
type OutputChange struct {
	Name           string
	Actions        []string
	ActionKind     ActionKind
	Before         any
	After          any
	Sensitive      bool
	AttributeDiffs []AttributeDiff
}

// ResourceDrift describes a resource whose real infrastructure state
// differs from Terraform's recorded state, independent of any planned
// configuration change.
type ResourceDrift struct {
	Address        string
	Module         string
	Type           string
	AttributeDiffs []AttributeDiff
	Risk           RiskFinding
}

// PlanReport is the fully parsed, risk-annotated representation of a
// Terraform plan that all tfsum renderers consume.
type PlanReport struct {
	TerraformVersion string
	Workspace        string
	ResourceChanges  []ResourceChange
	OutputChanges    []OutputChange
	Drift            []ResourceDrift
	Summary          PlanSummary
	Risks            []RiskFinding
	GeneratedAt      time.Time
}
