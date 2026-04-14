package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

type taskNewOptions struct {
	track     string
	objective string
	title     string
	envType   string
}

func newTaskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Task authoring commands",
	}
	cmd.AddCommand(newTaskNewCmd())
	return cmd
}

func newTaskNewCmd() *cobra.Command {
	opts := &taskNewOptions{}
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Scaffold a new task",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(opts.track) == "" {
				return fmt.Errorf("--track is required")
			}
			if strings.TrimSpace(opts.title) == "" {
				return fmt.Errorf("--title is required")
			}

			track := strings.ToLower(strings.TrimSpace(opts.track))
			title := strings.TrimSpace(opts.title)
			slug := slugifyTaskTitle(title)
			if slug == "" {
				return fmt.Errorf("title must contain letters or numbers")
			}

			baseDir := primaryTasksDir()
			if baseDir == "" {
				return fmt.Errorf("no tasks directory configured")
			}

			trackDir := filepath.Join(baseDir, track)
			if err := os.MkdirAll(trackDir, 0o755); err != nil {
				return fmt.Errorf("create track directory: %w", err)
			}

			namePrefix := track + "-" + slug
			name, err := nextTaskName(trackDir, namePrefix)
			if err != nil {
				return err
			}

			envType := strings.TrimSpace(opts.envType)
			if envType == "" {
				envType = inferTaskEnvironment(track)
			}

			taskDir := filepath.Join(trackDir, name)
			hintsDir := filepath.Join(taskDir, "hints")
			if err := os.MkdirAll(hintsDir, 0o755); err != nil {
				return fmt.Errorf("create task directories: %w", err)
			}

			taskYAMLPath := filepath.Join(taskDir, "task.yaml")
			hintPath := filepath.Join(hintsDir, "hint-1.md")

			if err := os.WriteFile(taskYAMLPath, []byte(renderTaskSkeleton(name, track, title, strings.TrimSpace(opts.objective), envType)), 0o644); err != nil {
				return fmt.Errorf("write task.yaml: %w", err)
			}
			if err := os.WriteFile(hintPath, []byte("Add your first hint here.\n"), 0o644); err != nil {
				return fmt.Errorf("write hint file: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", taskDir)
			fmt.Fprintf(cmd.OutOrStdout(), "- %s\n", taskYAMLPath)
			fmt.Fprintf(cmd.OutOrStdout(), "- %s\n", hintPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.track, "track", "", "Track for the new task")
	cmd.Flags().StringVar(&opts.objective, "objective", "", "GX objective id, for example 3.2")
	cmd.Flags().StringVar(&opts.title, "title", "", "Task title")
	cmd.Flags().StringVar(&opts.envType, "env", "", "Environment type override")
	return cmd
}

func inferTaskEnvironment(track string) string {
	if strings.HasPrefix(strings.ToLower(track), "gx-") {
		return "local"
	}
	return "kubernetes"
}

func slugifyTaskTitle(title string) string {
	title = strings.ToLower(strings.TrimSpace(title))
	re := regexp.MustCompile(`[^a-z0-9]+`)
	title = re.ReplaceAllString(title, "-")
	return strings.Trim(title, "-")
}

func nextTaskName(trackDir, prefix string) (string, error) {
	entries, err := os.ReadDir(trackDir)
	if err != nil {
		return "", fmt.Errorf("read track directory: %w", err)
	}

	maxSuffix := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, prefix+"-") {
			continue
		}
		var suffix int
		if _, err := fmt.Sscanf(strings.TrimPrefix(name, prefix+"-"), "%03d", &suffix); err == nil && suffix > maxSuffix {
			maxSuffix = suffix
		}
	}

	return fmt.Sprintf("%s-%03d", prefix, maxSuffix+1), nil
}

func renderTaskSkeleton(name, track, title, objective, envType string) string {
	metadataObjective := ""
	if objective != "" {
		metadataObjective = fmt.Sprintf("  gxObjective: %s\n", objective)
	}

	return fmt.Sprintf(`apiVersion: gymctl/v1
kind: Exercise
metadata:
  name: %s
  title: %q
  track: %s
%sspec:
  difficulty: intermediate
  estimatedTime: 20m
  description: |
    TODO: describe the exercise goal and expected outcome.
  environment:
    type: %s
  checks:
    # Replace with real validation checks.
    - name: TODO validation
      type: script
      script: |
        exit 1
      expectExitCode: 0
  hints:
    # Add more hints as needed.
    - cost: 10
      file: hints/hint-1.md
`, name, title, track, metadataObjective, envType)
}
