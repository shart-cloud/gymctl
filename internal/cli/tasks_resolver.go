package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

// resolveTasksDirectory finds the tasks directory in this order:
// 1. Explicitly set via --tasks-dir flag
// 2. Local ./tasks directory (for development)
// 3. System-wide installation at /usr/share/gymctl/tasks
// 4. User's home directory at ~/.gym/tasks
// 5. Bundled with binary at <binary-dir>/tasks
func resolveTasksDirectory() (string, error) {
	// 1. Check if explicitly set
	if tasksDir != "" && tasksDir != "tasks" {
		if _, err := os.Stat(tasksDir); err != nil {
			return "", fmt.Errorf("specified tasks directory not found: %s", tasksDir)
		}
		return tasksDir, nil
	}

	// 2. Check local directory (development mode)
	if _, err := os.Stat("tasks"); err == nil {
		return "tasks", nil
	}

	// 3. Check system-wide installation
	systemTasks := "/usr/share/gymctl/tasks"
	if _, err := os.Stat(systemTasks); err == nil {
		return systemTasks, nil
	}

	// 4. Check user's home directory
	if home, err := os.UserHomeDir(); err == nil {
		userTasks := filepath.Join(home, ".gym", "tasks")
		if _, err := os.Stat(userTasks); err == nil {
			return userTasks, nil
		}
	}

	// 5. Check alongside the binary
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		exeTasks := filepath.Join(exeDir, "tasks")
		if _, err := os.Stat(exeTasks); err == nil {
			return exeTasks, nil
		}
	}

	return "", fmt.Errorf(`tasks directory not found.

For standalone installation, exercises should be installed in one of:
  - ~/.gym/tasks (user installation)
  - /usr/share/gymctl/tasks (system-wide installation)
  - <gymctl-binary-location>/tasks (portable installation)

You can also specify a custom location with --tasks-dir flag.

To set up exercises in your home directory:
  git clone https://github.com/shart/container-course-exercises ~/.gym/tasks`)
}

// setupTasksDirectory ensures tasks directory is available
func setupTasksDirectory() error {
	resolved, err := resolveTasksDirectory()
	if err != nil {
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

	tasksDir = resolved
	fmt.Fprintf(os.Stderr, "Using exercises from: %s\n", tasksDir)
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