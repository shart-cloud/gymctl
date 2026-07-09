package cli

import (
	"path/filepath"
	"testing"

	"gymctl/internal/scenario"
)

func TestValidateExerciseReadinessAllowsScaffoldPlaceholder(t *testing.T) {
	ex := &scenario.Exercise{Spec: scenario.ExerciseSpec{
		ImplementationStatus: "scaffold",
		Environment:          scenario.EnvironmentSpec{Type: "kubernetes"},
		Checks: []scenario.Check{{
			Name:   "Executable lab checks implemented",
			Type:   "script",
			Script: "echo scaffolded",
		}},
		Hints: []scenario.Hint{{Cost: 0, Content: "hint"}},
	}}

	report := validateExerciseReadiness(filepath.Join(t.TempDir(), "task.yaml"), ex)
	if len(report.Issues) != 0 {
		t.Fatalf("expected scaffold placeholder to be allowed, got %#v", report.Issues)
	}
	if !report.PlaceholderChecks {
		t.Fatalf("expected placeholder check to be detected")
	}
}

func TestValidateExerciseReadinessRejectsReadyPlaceholder(t *testing.T) {
	ex := &scenario.Exercise{Spec: scenario.ExerciseSpec{
		Environment: scenario.EnvironmentSpec{Type: "kubernetes"},
		Checks: []scenario.Check{{
			Name:   "Executable lab checks implemented",
			Type:   "script",
			Script: "echo scaffolded",
		}},
		Hints: []scenario.Hint{{Cost: 0, Content: "hint"}},
	}}

	report := validateExerciseReadiness(filepath.Join(t.TempDir(), "task.yaml"), ex)
	if len(report.Issues) == 0 {
		t.Fatal("expected ready placeholder to fail readiness validation")
	}
}
