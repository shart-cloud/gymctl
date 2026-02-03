package scenario

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type CatalogEntry struct {
	Exercise *Exercise
	Path     string
	Dir      string
}

func LoadCatalog(tasksDir string) ([]CatalogEntry, error) {
	info, err := os.Stat(tasksDir)
	if err != nil {
		return nil, fmt.Errorf("tasks dir not found: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("tasks path is not a directory: %s", tasksDir)
	}

	var entries []CatalogEntry
	walkErr := filepath.WalkDir(tasksDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Base(path) != "task.yaml" {
			return nil
		}

		exercise, err := LoadExerciseFile(path)
		if err != nil {
			return fmt.Errorf("load %s: %w", path, err)
		}
		entries = append(entries, CatalogEntry{
			Exercise: exercise,
			Path:     path,
			Dir:      filepath.Dir(path),
		})
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	return entries, nil
}

func FindByName(entries []CatalogEntry, name string) (*CatalogEntry, bool) {
	for i := range entries {
		if entries[i].Exercise.Metadata.Name == name {
			return &entries[i], true
		}
	}
	return nil, false
}
