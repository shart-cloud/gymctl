package environment

import (
	"fmt"
	"os"
	"path/filepath"
)

func resolveExerciseKubeconfigPath(exerciseName string) (string, error) {
	if exerciseName == "" {
		return "", fmt.Errorf("exerciseName is required")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".gym", "kubeconfigs", exerciseName+".yaml"), nil
}
