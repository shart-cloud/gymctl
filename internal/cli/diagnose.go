package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"gymctl/internal/progress"
	"gymctl/internal/scenario"
)

type diagnosticCheck struct {
	Name        string
	Category    string
	Check       func(ctx context.Context) (bool, string, string)
	Required    bool
	FixCommand  string
}

func newDiagnoseCmd() *cobra.Command {
	var exerciseName string
	var verbose bool

	cmd := &cobra.Command{
		Use:   "diagnose",
		Short: "Diagnose common issues and system requirements",
		Long: `Diagnose checks your system for common issues and prerequisites.
It verifies Docker, Kubernetes, permissions, and exercise-specific requirements.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			spinner := NewSpinnerManager()

			// Header
			ColorHeader.Fprintln(cmd.OutOrStdout(), "🔍 Running Diagnostics")
			fmt.Fprintln(cmd.OutOrStdout())

			// Run system checks
			systemChecks := getSystemChecks()
			categoryResults := make(map[string][]bool)
			hasFailures := false

			for _, check := range systemChecks {
				spinner.Start(fmt.Sprintf("Checking %s", check.Name))

				passed, message, fix := check.Check(ctx)
				spinner.Stop()

				// Track results by category
				categoryResults[check.Category] = append(categoryResults[check.Category], passed)

				// Display result
				if passed {
					fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n",
						ColorSuccess.Sprint(IconSuccess),
						check.Name)
					if verbose && message != "" {
						ColorDim.Fprintf(cmd.OutOrStdout(), "  └─ %s\n", message)
					}
				} else {
					hasFailures = true
					fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n",
						ColorError.Sprint(IconFail),
						check.Name)
					if message != "" {
						ColorError.Fprintf(cmd.OutOrStdout(), "  └─ Issue: %s\n", message)
					}
					if fix != "" {
						ColorInfo.Fprintf(cmd.OutOrStdout(), "  └─ Fix: %s\n", fix)
					} else if check.FixCommand != "" {
						ColorInfo.Fprintf(cmd.OutOrStdout(), "  └─ Try: %s\n", check.FixCommand)
					}
				}
			}

			// Check specific exercise if provided
			if exerciseName != "" {
				fmt.Fprintln(cmd.OutOrStdout())
				ColorBold.Fprintf(cmd.OutOrStdout(), "Exercise: %s\n", exerciseName)

				if err := diagnoseExercise(cmd, ctx, exerciseName, verbose); err != nil {
					ColorError.Fprintf(cmd.OutOrStdout(), "Failed to diagnose exercise: %v\n", err)
					hasFailures = true
				}
			}

			// Summary
			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintln(cmd.OutOrStdout(), strings.Repeat("─", 60))

			// Category summary
			for category, results := range categoryResults {
				passed := 0
				for _, p := range results {
					if p {
						passed++
					}
				}

				icon := IconSuccess
				color := ColorSuccess
				if passed < len(results) {
					icon = IconWarning
					color = ColorWarning
				}
				if passed == 0 {
					icon = IconFail
					color = ColorError
				}

				color.Fprintf(cmd.OutOrStdout(), "%s %s: %d/%d checks passed\n",
					icon, category, passed, len(results))
			}

			if hasFailures {
				fmt.Fprintln(cmd.OutOrStdout())
				ColorWarning.Fprintln(cmd.OutOrStdout(), "⚠ Some checks failed. Review the issues above.")
				return fmt.Errorf("diagnostic checks failed")
			}

			fmt.Fprintln(cmd.OutOrStdout())
			ColorSuccess.Fprintln(cmd.OutOrStdout(), "✅ All checks passed! Your system is ready.")
			return nil
		},
	}

	cmd.Flags().StringVarP(&exerciseName, "exercise", "e", "", "Diagnose specific exercise")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed output")

	return cmd
}

func getSystemChecks() []diagnosticCheck {
	return []diagnosticCheck{
		// Docker checks
		{
			Name:     "Docker installed",
			Category: "Docker",
			Required: true,
			Check: func(ctx context.Context) (bool, string, string) {
				cmd := exec.CommandContext(ctx, "docker", "version", "--format", "{{.Server.Version}}")
				output, err := cmd.Output()
				if err != nil {
					return false, "Docker is not installed or not running", "Install Docker from https://docs.docker.com/get-docker/"
				}
				version := strings.TrimSpace(string(output))
				return true, fmt.Sprintf("Version %s", version), ""
			},
		},
		{
			Name:     "Docker daemon running",
			Category: "Docker",
			Required: true,
			Check: func(ctx context.Context) (bool, string, string) {
				cmd := exec.CommandContext(ctx, "docker", "info")
				if err := cmd.Run(); err != nil {
					if runtime.GOOS == "linux" {
						return false, "Docker daemon is not running", "sudo systemctl start docker"
					} else if runtime.GOOS == "darwin" {
						return false, "Docker daemon is not running", "Start Docker Desktop"
					}
					return false, "Docker daemon is not running", "Start the Docker service"
				}
				return true, "Docker daemon is responsive", ""
			},
		},
		{
			Name:     "Docker permissions",
			Category: "Docker",
			Required: true,
			Check: func(ctx context.Context) (bool, string, string) {
				cmd := exec.CommandContext(ctx, "docker", "ps")
				if err := cmd.Run(); err != nil {
					if strings.Contains(err.Error(), "permission denied") {
						return false, "User doesn't have Docker permissions", "Add user to docker group: sudo usermod -aG docker $USER"
					}
					return false, "Cannot access Docker", ""
				}
				return true, "User has Docker access", ""
			},
		},
		{
			Name:     "Docker Compose installed",
			Category: "Docker",
			Required: false,
			Check: func(ctx context.Context) (bool, string, string) {
				// Try docker compose (v2)
				cmd := exec.CommandContext(ctx, "docker", "compose", "version")
				if err := cmd.Run(); err == nil {
					return true, "Docker Compose v2 installed", ""
				}

				// Try docker-compose (v1)
				cmd = exec.CommandContext(ctx, "docker-compose", "--version")
				if err := cmd.Run(); err == nil {
					return true, "Docker Compose v1 installed", ""
				}

				return false, "Docker Compose not installed", "Install via: docker plugin install compose"
			},
		},

		// Kubernetes checks
		{
			Name:     "kubectl installed",
			Category: "Kubernetes",
			Required: false,
			Check: func(ctx context.Context) (bool, string, string) {
				cmd := exec.CommandContext(ctx, "kubectl", "version", "--client", "--short")
				output, err := cmd.Output()
				if err != nil {
					return false, "kubectl is not installed", "Install from https://kubernetes.io/docs/tasks/tools/"
				}
				version := strings.TrimSpace(string(output))
				return true, version, ""
			},
		},
		{
			Name:     "kind installed",
			Category: "Kubernetes",
			Required: false,
			Check: func(ctx context.Context) (bool, string, string) {
				cmd := exec.CommandContext(ctx, "kind", "version")
				output, err := cmd.Output()
				if err != nil {
					return false, "kind is not installed", "Install from https://kind.sigs.k8s.io/docs/user/quick-start/#installation"
				}
				version := strings.TrimSpace(string(output))
				return true, version, ""
			},
		},

		// File system checks
		{
			Name:     "Home directory accessible",
			Category: "System",
			Required: true,
			Check: func(ctx context.Context) (bool, string, string) {
				home, err := os.UserHomeDir()
				if err != nil {
					return false, "Cannot determine home directory", ""
				}

				gymDir := fmt.Sprintf("%s/.gym", home)
				if err := os.MkdirAll(gymDir, 0755); err != nil {
					return false, fmt.Sprintf("Cannot create %s", gymDir), "Check directory permissions"
				}

				return true, fmt.Sprintf("Gym directory: %s", gymDir), ""
			},
		},
		{
			Name:     "Tasks directory exists",
			Category: "System",
			Required: true,
			Check: func(ctx context.Context) (bool, string, string) {
				if _, err := os.Stat(tasksDir); err != nil {
					return false, fmt.Sprintf("Tasks directory not found: %s", tasksDir), "Ensure you're running from the gymctl directory"
				}
				return true, fmt.Sprintf("Found: %s", tasksDir), ""
			},
		},
		{
			Name:     "Progress file accessible",
			Category: "System",
			Required: true,
			Check: func(ctx context.Context) (bool, string, string) {
				progressPath, err := resolveProgressFile()
				if err != nil {
					return false, "Cannot resolve progress file path", ""
				}

				// Try to load it
				_, err = progress.Load(progressPath)
				if err != nil {
					return false, fmt.Sprintf("Cannot load progress file: %v", err), "Try: gymctl reset --all"
				}

				return true, fmt.Sprintf("Progress file: %s", progressPath), ""
			},
		},

		// Network checks
		{
			Name:     "Docker Hub accessible",
			Category: "Network",
			Required: false,
			Check: func(ctx context.Context) (bool, string, string) {
				ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
				defer cancel()

				cmd := exec.CommandContext(ctx, "docker", "pull", "alpine:latest")
				if err := cmd.Run(); err != nil {
					return false, "Cannot pull images from Docker Hub", "Check internet connection and proxy settings"
				}
				return true, "Docker Hub is accessible", ""
			},
		},
	}
}

func diagnoseExercise(cmd *cobra.Command, ctx context.Context, exerciseName string, verbose bool) error {
	// Load exercise
	entries, err := scenario.LoadCatalog(tasksDir)
	if err != nil {
		return fmt.Errorf("load catalog: %w", err)
	}

	entry, found := scenario.FindByName(entries, exerciseName)
	if !found {
		return fmt.Errorf("exercise not found: %s", exerciseName)
	}

	exercise := entry.Exercise

	// Check exercise-specific requirements
	ColorBold.Fprintln(cmd.OutOrStdout(), "\nExercise Requirements:")

	// Environment type
	fmt.Fprintf(cmd.OutOrStdout(), "  Environment: %s\n", exercise.Spec.Environment.Type)

	switch exercise.Spec.Environment.Type {
	case "docker":
		if exercise.Spec.Environment.Docker != nil {
			docker := exercise.Spec.Environment.Docker

			// Check if work directory exists
			workDir, err := resolveWorkDir(exerciseName)
			if err == nil {
				if stat, err := os.Stat(workDir); err == nil {
					ColorSuccess.Fprintf(cmd.OutOrStdout(), "  %s Work directory exists: %s\n", IconSuccess, workDir)
					if verbose {
						ColorDim.Fprintf(cmd.OutOrStdout(), "    Size: %d bytes\n", stat.Size())
					}
				} else {
					ColorWarning.Fprintf(cmd.OutOrStdout(), "  %s Work directory not found: %s\n", IconWarning, workDir)
				}
			}

			// Check containers
			if len(docker.Containers) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "  Required containers: %d\n", len(docker.Containers))
				for _, container := range docker.Containers {
					checkContainer(cmd, ctx, container.Name, verbose)
				}
			}

			// Check compose file if specified
			if docker.ComposeFile != "" {
				composeFile := fmt.Sprintf("%s/%s", entry.Dir, docker.ComposeFile)
				if _, err := os.Stat(composeFile); err == nil {
					ColorSuccess.Fprintf(cmd.OutOrStdout(), "  %s Compose file exists: %s\n", IconSuccess, docker.ComposeFile)
				} else {
					ColorError.Fprintf(cmd.OutOrStdout(), "  %s Compose file not found: %s\n", IconFail, docker.ComposeFile)
				}
			}
		}

	case "kubernetes":
		if exercise.Spec.Environment.Kubernetes != nil {
			k8s := exercise.Spec.Environment.Kubernetes

			// Check if cluster should exist
			if k8s.CreateCluster != nil && *k8s.CreateCluster {
				checkKindCluster(cmd, ctx, "jerry-gym", verbose)
			}

			// Check namespace
			if k8s.Namespace != "" {
				checkNamespace(cmd, ctx, k8s.Namespace, verbose)
			}
		}
	}

	// Check exercise status
	progressPath, _ := resolveProgressFile()
	progressFile, _ := progress.Load(progressPath)
	if status, exists := progressFile.Exercises[exerciseName]; exists {
		fmt.Fprintln(cmd.OutOrStdout())
		ColorBold.Fprintln(cmd.OutOrStdout(), "Progress Status:")
		fmt.Fprintf(cmd.OutOrStdout(), "  Status: %s\n", status.Status)
		if status.StartedAt != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "  Started: %s\n", status.StartedAt)
		}
		if status.CompletedAt != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "  Completed: %s\n", status.CompletedAt)
		}
		if status.HintsUsed > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "  Hints used: %d\n", status.HintsUsed)
		}
	}

	return nil
}

func checkContainer(cmd *cobra.Command, ctx context.Context, name string, verbose bool) {
	execCmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--filter", fmt.Sprintf("name=%s", name), "--format", "{{.Status}}")
	output, err := execCmd.Output()
	if err != nil || strings.TrimSpace(string(output)) == "" {
		ColorWarning.Fprintf(cmd.OutOrStdout(), "    %s Container '%s' not found\n", IconWarning, name)
		return
	}

	status := strings.TrimSpace(string(output))
	if strings.HasPrefix(status, "Up") {
		ColorSuccess.Fprintf(cmd.OutOrStdout(), "    %s Container '%s' is running\n", IconSuccess, name)
	} else {
		ColorError.Fprintf(cmd.OutOrStdout(), "    %s Container '%s' is stopped: %s\n", IconFail, name, status)
	}
}

func checkNetwork(cmd *cobra.Command, ctx context.Context, name string, verbose bool) {
	execCmd := exec.CommandContext(ctx, "docker", "network", "ls", "--filter", fmt.Sprintf("name=%s", name), "--format", "{{.Name}}")
	output, err := execCmd.Output()
	if err != nil || strings.TrimSpace(string(output)) == "" {
		ColorWarning.Fprintf(cmd.OutOrStdout(), "    %s Network '%s' not found\n", IconWarning, name)
		return
	}
	ColorSuccess.Fprintf(cmd.OutOrStdout(), "    %s Network '%s' exists\n", IconSuccess, name)
}

func checkKindCluster(cmd *cobra.Command, ctx context.Context, name string, verbose bool) {
	execCmd := exec.CommandContext(ctx, "kind", "get", "clusters")
	output, err := execCmd.Output()
	if err != nil {
		ColorError.Fprintf(cmd.OutOrStdout(), "    %s Cannot check kind clusters\n", IconFail)
		return
	}

	clusters := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, cluster := range clusters {
		if cluster == name {
			ColorSuccess.Fprintf(cmd.OutOrStdout(), "    %s Kind cluster '%s' exists\n", IconSuccess, name)
			return
		}
	}
	ColorWarning.Fprintf(cmd.OutOrStdout(), "    %s Kind cluster '%s' not found\n", IconWarning, name)
}

func checkNamespace(cmd *cobra.Command, ctx context.Context, name string, verbose bool) {
	execCmd := exec.CommandContext(ctx, "kubectl", "get", "namespace", name)
	if err := execCmd.Run(); err != nil {
		ColorWarning.Fprintf(cmd.OutOrStdout(), "    %s Namespace '%s' not found\n", IconWarning, name)
		return
	}
	ColorSuccess.Fprintf(cmd.OutOrStdout(), "    %s Namespace '%s' exists\n", IconSuccess, name)
}