package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gymctl/internal/scenario"
)

// resolveTasksDirectories finds the tasks directory list in this order:
// 1. Explicitly set via --tasks-dir flag (comma-separated)
// 2. GYMCTL_TASKS_DIRS env var (comma-separated)
// 2. Local ./tasks directory (for development)
// 3. System-wide installation at /usr/share/gymctl/tasks
// 4. User's home directory at ~/.gym/tasks
// 5. Bundled with binary at <binary-dir>/tasks
func resolveTasksDirectories() ([]string, error) {
	// 1. Check if explicitly set
	if dirs := parseTasksDirList(tasksDir); len(dirs) > 0 && tasksDir != "tasks" {
		if err := validateTasksDirectories(dirs); err != nil {
			return nil, err
		}
		return dirs, nil
	}

	// 2. Check environment variable
	if dirs := parseTasksDirList(os.Getenv("GYMCTL_TASKS_DIRS")); len(dirs) > 0 {
		if err := validateTasksDirectories(dirs); err != nil {
			return nil, err
		}
		return dirs, nil
	}

	// 3. Check local directory (development mode)
	if _, err := os.Stat("tasks"); err == nil {
		return []string{"tasks"}, nil
	}

	// 4. Check system-wide installation
	systemTasks := "/usr/share/gymctl/tasks"
	if _, err := os.Stat(systemTasks); err == nil {
		return []string{systemTasks}, nil
	}

	// 5. Check user's home directory
	if home, err := os.UserHomeDir(); err == nil {
		userTasks := filepath.Join(home, ".gym", "tasks")
		if _, err := os.Stat(userTasks); err == nil {
			return []string{userTasks}, nil
		}
	}

	// 6. Check alongside the binary
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		exeTasks := filepath.Join(exeDir, "tasks")
		if _, err := os.Stat(exeTasks); err == nil {
			return []string{exeTasks}, nil
		}
	}

	return nil, fmt.Errorf(`tasks directory not found.

For standalone installation, exercises should be installed in one of:
  - ~/.gym/tasks (user installation)
  - /usr/share/gymctl/tasks (system-wide installation)
  - <gymctl-binary-location>/tasks (portable installation)

You can also specify a custom location with --tasks-dir flag.

To set up exercises in your home directory:
  git clone https://github.com/shart/container-course-exercises ~/.gym/tasks`)
}

func parseTasksDirList(raw string) []string {
	parts := strings.Split(raw, ",")
	dirs := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		dir := strings.TrimSpace(part)
		if dir == "" || seen[dir] {
			continue
		}
		seen[dir] = true
		dirs = append(dirs, dir)
	}
	return dirs
}

func validateTasksDirectories(dirs []string) error {
	for _, dir := range dirs {
		info, err := os.Stat(dir)
		if err != nil {
			return fmt.Errorf("specified tasks directory not found: %s", dir)
		}
		if !info.IsDir() {
			return fmt.Errorf("tasks path is not a directory: %s", dir)
		}
	}
	return nil
}

func loadCatalogEntries() ([]scenario.CatalogEntry, error) {
	dirs := parseTasksDirList(tasksDir)
	entries := make([]scenario.CatalogEntry, 0)
	seen := map[string]string{}
	for _, dir := range dirs {
		loaded, err := scenario.LoadCatalog(dir)
		if err != nil {
			return nil, err
		}
		for _, entry := range loaded {
			name := entry.Exercise.Metadata.Name
			if prior, exists := seen[name]; exists {
				return nil, fmt.Errorf("duplicate exercise name %q found in %s and %s", name, prior, entry.Path)
			}
			seen[name] = entry.Path
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

func primaryTasksDir() string {
	dirs := parseTasksDirList(tasksDir)
	if len(dirs) == 0 {
		return ""
	}
	return dirs[0]
}

// setupTasksDirectory ensures tasks directory is available
func setupTasksDirectory() error {
	resolved, err := resolveTasksDirectories()
	if err != nil {
		if isJSONOutput() {
			return err
		}

		// Provide instructions for setup
		fmt.Fprintln(os.Stderr, err)

		// Offer to download exercises
		gymDir, _ := resolveGymDir()
		tasksPath := filepath.Join(gymDir, "tasks")

		fmt.Fprintf(os.Stderr, "\nWould you like to download the exercises to %s? [y/N]: ", tasksPath)

		var answer string
		fmt.Scanln(&answer)

		if answer == "y" || answer == "Y" {
			return downloadExercises(tasksPath)
		}

		return err
	}

	tasksDir = strings.Join(resolved, ",")
	if !isJSONOutput() {
		fmt.Fprintf(os.Stderr, "Using exercises from: %s\n", tasksDir)
	}
	return nil
}

// downloadExercises downloads the exercise files
func downloadExercises(destPath string) error {
	fmt.Println("Downloading exercises...")

	// Create destination directory
	if err := os.MkdirAll(destPath, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Note: In a real implementation, you would:
	// 1. Download a tarball/zip from GitHub releases
	// 2. Extract it to destPath
	// 3. Verify the files

	// For now, we'll just inform the user
	fmt.Printf(`
Please manually download the exercises:

  git clone https://github.com/shart/container-course-exercises %s

Or download the latest release:

  curl -L https://github.com/shart/container-course/releases/latest/download/exercises.tar.gz | tar -xz -C %s

Then run gymctl again.
`, destPath, filepath.Dir(destPath))

	return fmt.Errorf("manual download required")
}
