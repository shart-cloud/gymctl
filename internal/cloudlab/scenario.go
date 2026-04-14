// Package cloudlab defines the scenario format and loader for shart.cloud
// browser-based container labs. It is intentionally separate from the local
// kind/vagrant scenario format in internal/scenario.
package cloudlab

import (
	"fmt"
	"os"

	"sigs.k8s.io/yaml"
)

// ScenarioMeta holds the human-facing metadata for a lab.
type ScenarioMeta struct {
	ID               string `json:"id"                yaml:"id"`
	Title            string `json:"title"             yaml:"title"`
	Difficulty       string `json:"difficulty"        yaml:"difficulty"`
	TimeLimitMinutes int    `json:"time_limit_minutes" yaml:"time_limit_minutes"`
}

// Check is a validation rule evaluated by gymctl inside the container.
type Check struct {
	ID        string `json:"id"         yaml:"id"`
	Type      string `json:"type"       yaml:"type"` // exact_match | kubectl_output
	Prompt    string `json:"prompt"     yaml:"prompt"`
	Expected  string `json:"expected"   yaml:"expected"`
	Kubectl   string `json:"kubectl"    yaml:"kubectl"`    // kubectl_output: args passed to kubectl (no "kubectl" prefix)
	MatchType string `json:"match_type" yaml:"match_type"` // "exact" (default) | "regex"
}

// Scenario is the top-level cloud lab definition.
//
// resources contains raw Kubernetes manifests (same structure as kubectl apply
// YAML). The mock-apiserver indexes and serves them verbatim.
//
// checks are evaluated by gymctl in lab mode (Phase 3).
type Scenario struct {
	Meta      ScenarioMeta             `json:"meta"      yaml:"meta"`
	Resources []map[string]interface{} `json:"resources" yaml:"resources"`
	Checks    []Check                  `json:"checks"    yaml:"checks"`
}

// Load reads and parses a scenario YAML file.
func Load(path string) (*Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading scenario %q: %w", path, err)
	}

	var s Scenario
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing scenario YAML: %w", err)
	}

	if s.Meta.ID == "" {
		return nil, fmt.Errorf("scenario is missing meta.id")
	}
	if len(s.Resources) == 0 {
		return nil, fmt.Errorf("scenario %q has no resources", s.Meta.ID)
	}

	return &s, nil
}
