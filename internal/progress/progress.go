package progress

import (
	"fmt"
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"
)

type File struct {
	Version   int                       `yaml:"version"`
	Exercises map[string]ExerciseStatus `yaml:"exercises"`
}

type ExerciseStatus struct {
	Status      string `yaml:"status"`
	StartedAt   string `yaml:"startedAt,omitempty"`
	CompletedAt string `yaml:"completedAt,omitempty"`
	TimeSpent   string `yaml:"timeSpent,omitempty"`
	HintsUsed   int    `yaml:"hintsUsed,omitempty"`
	Resets      int    `yaml:"resets,omitempty"`
	Score       int    `yaml:"score,omitempty"`
}

func Load(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &File{Version: 1, Exercises: map[string]ExerciseStatus{}}, nil
		}
		return nil, fmt.Errorf("read progress: %w", err)
	}

	var file File
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse progress yaml: %w", err)
	}
	if file.Exercises == nil {
		file.Exercises = map[string]ExerciseStatus{}
	}
	if file.Version == 0 {
		file.Version = 1
	}

	return &file, nil
}

func Save(path string, file *File) error {
	data, err := yaml.Marshal(file)
	if err != nil {
		return fmt.Errorf("marshal progress: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create progress dir: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write progress: %w", err)
	}

	return nil
}
