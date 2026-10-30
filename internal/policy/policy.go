// Package policy loads, validates, and discovers tfsum's YAML
// configuration: risk rules and display preferences. It owns config
// schema and discovery only; rule evaluation lives in internal/risk.
package policy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Fanatic-zer0/tfsum/internal/plan"
	"github.com/Fanatic-zer0/tfsum/internal/risk"
)

// DiscoveryNames are the config file names auto-discovered in the current
// directory when --config is not provided.
var DiscoveryNames = []string{".tfsum.yml", ".tfsum.yaml"}

// Config is tfsum's YAML configuration schema.
type Config struct {
	Rules []risk.Rule `yaml:"rules"`
}

// Load reads and validates a YAML config file at path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("invalid config %s: %w", path, err)
	}
	return &cfg, nil
}

// Discover looks for a tfsum config file in dir, trying each name in
// DiscoveryNames in order. It returns ("", false) if none exist.
func Discover(dir string) (string, bool) {
	for _, name := range DiscoveryNames {
		candidate := filepath.Join(dir, name)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, true
		}
	}
	return "", false
}

func validate(cfg *Config) error {
	seen := make(map[string]bool, len(cfg.Rules))
	for _, r := range cfg.Rules {
		if r.ID == "" {
			return fmt.Errorf("rule is missing an id")
		}
		if seen[r.ID] {
			return fmt.Errorf("duplicate rule id %q in config", r.ID)
		}
		seen[r.ID] = true
		if r.Severity == "" {
			return fmt.Errorf("rule %q is missing severity", r.ID)
		}
		if _, ok := plan.ParseRiskLevel(strings.ToLower(r.Severity)); !ok {
			return fmt.Errorf("rule %q has invalid severity %q", r.ID, r.Severity)
		}
	}
	return nil
}
