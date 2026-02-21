package environment

import (
	"fmt"
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"
)

type ExerciseState struct {
	ExerciseName string            `yaml:"exerciseName"`
	Provider     string            `yaml:"provider"`
	WorkspaceDir string            `yaml:"workspaceDir,omitempty"`
	Kubeconfig   string            `yaml:"kubeconfig,omitempty"`
	ClusterName  string            `yaml:"clusterName,omitempty"`
	Nodes        map[string]string `yaml:"nodes,omitempty"`
}

func LoadExerciseState(exerciseName string) (*ExerciseState, error) {
	statePath, err := resolveStatePath(exerciseName)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read exercise state: %w", err)
	}

	var state ExerciseState
	if err := yaml.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parse exercise state yaml: %w", err)
	}
	if state.Nodes == nil {
		state.Nodes = map[string]string{}
	}

	return &state, nil
}

func SaveExerciseState(state *ExerciseState) error {
	if state == nil {
		return fmt.Errorf("exercise state is nil")
	}
	if state.ExerciseName == "" {
		return fmt.Errorf("exerciseName is required")
	}

	statePath, err := resolveStatePath(state.ExerciseName)
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal exercise state: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(statePath), 0o755); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}

	if err := os.WriteFile(statePath, data, 0o644); err != nil {
		return fmt.Errorf("write exercise state: %w", err)
	}

	return nil
}

func DeleteExerciseState(exerciseName string) error {
	statePath, err := resolveStatePath(exerciseName)
	if err != nil {
		return err
	}
	if err := os.Remove(statePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete exercise state: %w", err)
	}
	return nil
}

func resolveStatePath(exerciseName string) (string, error) {
	if exerciseName == "" {
		return "", fmt.Errorf("exerciseName is required")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}

	return filepath.Join(home, ".gym", "state", exerciseName+".yaml"), nil
}
