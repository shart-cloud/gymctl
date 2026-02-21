package cli

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"gymctl/internal/environment"
	"gymctl/internal/progress"
	"gymctl/internal/scenario"
	"gymctl/internal/tui"
)

func newShellCmd() *cobra.Command {
	var useWorkstation bool

	cmd := &cobra.Command{
		Use:   "shell",
		Short: "Open the interactive GRIM sprint browser or exec into workstation",
		Long: `Open the interactive GRIM sprint browser to navigate exercises,
or directly exec into the workstation container if --workstation is used.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if useWorkstation {
				return execIntoWorkstation(cmd)
			}

			catalog, err := scenario.LoadCatalog(tasksDir)
			if err != nil {
				return fmt.Errorf("load catalog: %w", err)
			}

			progressPath, err := resolveProgressFile()
			if err != nil {
				return fmt.Errorf("resolve progress file: %w", err)
			}

			prog, err := progress.Load(progressPath)
			if err != nil {
				return fmt.Errorf("load progress: %w", err)
			}

			m := tui.NewModel(catalog, prog, tasksDir, progressPath)
			p := tea.NewProgram(m, tea.WithAltScreen())
			_, err = p.Run()
			return err
		},
	}

	cmd.Flags().BoolVar(&useWorkstation, "workstation", false, "Exec directly into workstation container")

	return cmd
}

func execIntoWorkstation(cmd *cobra.Command) error {
	ctx := cmd.Context()
	envVars := map[string]string{}

	// Get current exercise if one is active
	currentExercise, err := loadCurrentExercise()
	if err != nil {
		fmt.Fprintf(cmd.OutOrStderr(), "Warning: No current exercise found (%v)\n", err)
	} else {
		entries, loadErr := scenario.LoadCatalog(tasksDir)
		if loadErr == nil {
			if entry, found := scenario.FindByName(entries, currentExercise); found {
				collected, warnings, envErr := collectExerciseEnv(ctx, entry.Exercise)
				if envErr != nil {
					fmt.Fprintf(cmd.OutOrStderr(), "Warning: Could not resolve exercise environment variables: %v\n", envErr)
				} else {
					envVars = collected
				}
				for _, warning := range warnings {
					fmt.Fprintf(cmd.OutOrStderr(), "Warning: %s\n", warning)
				}
			}
		}

		if state, stateErr := environment.LoadExerciseState(currentExercise); stateErr == nil && state != nil {
			if state.Kubeconfig != "" {
				containerPath := state.Kubeconfig
				if homeDir, homeErr := os.UserHomeDir(); homeErr == nil {
					prefix := filepath.Join(homeDir, ".gym")
					if rel, relErr := filepath.Rel(prefix, state.Kubeconfig); relErr == nil {
						if len(rel) > 0 && rel[0] != '.' {
							containerPath = filepath.Join("/home/student/.gym", rel)
						}
					}
				}
				envVars["GYM_KUBECONFIG"] = containerPath
				envVars["KUBECONFIG"] = containerPath
			}
		}
	}

	// Ensure workstation is running
	manager := &environment.WorkstationManager{}

	// Get gymctl binary path
	gymctlPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Ensuring workstation container is ready...\n")
	if err := manager.EnsureRunning(ctx, gymctlPath); err != nil {
		return fmt.Errorf("ensure workstation running: %w", err)
	}

	// Determine work directory
	workDir := "/workspace"
	if currentExercise != "" {
		homeDir, _ := os.UserHomeDir()
		exerciseWorkDir := filepath.Join(homeDir, ".gym", "workdir", currentExercise)

		// Copy current exercise files to container if they exist
		if _, err := os.Stat(exerciseWorkDir); err == nil {
			containerPath := fmt.Sprintf("/workspace/%s", currentExercise)
			if err := manager.CopyExerciseFiles(ctx, exerciseWorkDir, containerPath); err != nil {
				fmt.Fprintf(cmd.OutOrStderr(), "Warning: Could not copy exercise files: %v\n", err)
			} else {
				workDir = containerPath
			}
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Dropping into workstation container...\n")
	fmt.Fprintf(cmd.OutOrStdout(), "Type 'exit' to return to host shell.\n\n")

	return manager.ExecShell(ctx, workDir, envVars)
}
