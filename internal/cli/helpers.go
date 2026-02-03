package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func resolveProgressFile() (string, error) {
	if progressFile != "" {
		return progressFile, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}

	return filepath.Join(home, ".gym", "progress.yaml"), nil
}

func resolveGymDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".gym"), nil
}

func resolveCurrentFile() (string, error) {
	gymDir, err := resolveGymDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(gymDir, "current"), nil
}

func resolveWorkDir(exerciseName string) (string, error) {
	gymDir, err := resolveGymDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(gymDir, "workdir", exerciseName), nil
}

func loadCurrentExercise() (string, error) {
	currentFile, err := resolveCurrentFile()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(currentFile)
	if err != nil {
		return "", fmt.Errorf("read current exercise: %w", err)
	}
	name := strings.TrimSpace(string(data))
	if name == "" {
		return "", fmt.Errorf("current exercise not set")
	}
	return name, nil
}

func writeCurrentExercise(name string) error {
	currentFile, err := resolveCurrentFile()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(currentFile), 0o755); err != nil {
		return err
	}
	return os.WriteFile(currentFile, []byte(name+"\n"), 0o644)
}

func firstLine(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if idx := strings.IndexByte(text, '\n'); idx >= 0 {
		return strings.TrimSpace(text[:idx])
	}
	return text
}

func defaultPoints(points int) int {
	if points == 0 {
		return 100
	}
	return points
}
