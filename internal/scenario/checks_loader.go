package scenario

import (
	"fmt"
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"
)

type ChecksFile struct {
	LabID string     `yaml:"lab_id,omitempty"`
	Title string     `yaml:"title,omitempty"`
	MCQs  []MCQCheck `yaml:"mcqs,omitempty"`
}

type MCQCheck struct {
	ID         string `yaml:"id"`
	AnswerHash string `yaml:"answer_hash"`
}

func LoadChecksFile(path string) (*ChecksFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read checks file: %w", err)
	}

	var checks ChecksFile
	if err := yaml.Unmarshal(data, &checks); err != nil {
		return nil, fmt.Errorf("parse checks yaml: %w", err)
	}

	return &checks, nil
}

func LoadChecksFileIfExists(dir string) (*ChecksFile, string, error) {
	path := filepath.Join(dir, "checks.yaml")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, "", nil
		}
		return nil, "", fmt.Errorf("stat checks file: %w", err)
	}

	checks, err := LoadChecksFile(path)
	if err != nil {
		return nil, "", err
	}
	return checks, path, nil
}
