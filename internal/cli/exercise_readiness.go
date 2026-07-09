package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gymctl/internal/scenario"
)

type readinessReport struct {
	Status            string
	MissingSetup      bool
	PlaceholderChecks bool
	MissingHints      []string
	Issues            []string
}

func validateExerciseReadiness(path string, exercise *scenario.Exercise) readinessReport {
	status := scenario.NormalizeImplementationStatus(exercise.Spec.ImplementationStatus)
	report := readinessReport{Status: status}

	dir := filepath.Dir(path)
	report.MissingSetup = !hasSetupMechanism(dir, exercise)
	report.PlaceholderChecks = hasPlaceholderChecks(exercise)
	report.MissingHints = missingHintFiles(dir, exercise)

	if status == scenario.ImplementationStatusReady {
		if report.MissingSetup {
			report.Issues = append(report.Issues, "ready exercise has no setup directory or declared setup mechanism")
		}
		if report.PlaceholderChecks {
			report.Issues = append(report.Issues, "ready exercise still has scaffold placeholder checks")
		}
		for _, hint := range report.MissingHints {
			report.Issues = append(report.Issues, fmt.Sprintf("hint file not found: %s", hint))
		}
	}

	return report
}

func hasSetupMechanism(dir string, exercise *scenario.Exercise) bool {
	if _, err := os.Stat(filepath.Join(dir, "setup")); err == nil {
		return true
	}
	if exercise == nil {
		return false
	}
	env := exercise.Spec.Environment
	if len(env.CustomSetup) > 0 {
		return true
	}
	if env.Kubernetes != nil && len(env.Kubernetes.SetupManifests) > 0 {
		return true
	}
	if env.Kubernetes != nil && env.Kubernetes.CreateCluster != nil && *env.Kubernetes.CreateCluster {
		return true
	}
	if env.Docker != nil {
		return len(env.Docker.CopyFiles) > 0 || env.Docker.ComposeFile != "" || len(env.Docker.Containers) > 0
	}
	return env.Type == "local"
}

func hasPlaceholderChecks(exercise *scenario.Exercise) bool {
	if exercise == nil {
		return false
	}
	for _, check := range exercise.Spec.Checks {
		text := strings.ToLower(check.Name + "\n" + check.Script)
		if strings.Contains(text, "scaffold") || strings.Contains(text, "implement setup") {
			return true
		}
	}
	return false
}

func missingHintFiles(dir string, exercise *scenario.Exercise) []string {
	if exercise == nil {
		return nil
	}
	var missing []string
	for _, hint := range exercise.Spec.Hints {
		if strings.TrimSpace(hint.File) == "" {
			continue
		}
		path := filepath.Join(dir, hint.File)
		if _, err := os.Stat(path); err != nil {
			missing = append(missing, hint.File)
		}
	}
	return missing
}
