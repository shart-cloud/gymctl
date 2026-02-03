package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"gymctl/internal/environment"
	"gymctl/internal/progress"
	"gymctl/internal/scenario"
)

func newStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop [exercise-name]",
		Short: "Stop the current exercise and clean up resources",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) == 1 {
				name = args[0]
			} else {
				current, err := loadCurrentExercise()
				if err != nil {
					return fmt.Errorf("no exercise specified and no current exercise set")
				}
				name = current
			}

			entries, err := scenario.LoadCatalog(tasksDir)
			if err != nil {
				return err
			}
			entry, found := scenario.FindByName(entries, name)
			if !found {
				return fmt.Errorf("exercise not found: %s", name)
			}
			exercise := entry.Exercise

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			// Clean up resources based on environment type
			switch exercise.Spec.Environment.Type {
			case "kubernetes":
				if exercise.Spec.Environment.Kubernetes == nil {
					return fmt.Errorf("missing kubernetes environment config")
				}
				k8s := exercise.Spec.Environment.Kubernetes
				createCluster := true
				if k8s.CreateCluster != nil {
					createCluster = *k8s.CreateCluster
				}

				if createCluster {
					manager := environment.KindManager{ClusterName: "jerry-gym"}
					fmt.Fprintln(cmd.OutOrStdout(), "Stopping kind cluster...")
					if err := manager.Delete(ctx); err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to delete cluster: %v\n", err)
					}
				}

			case "docker":
				if exercise.Spec.Environment.Docker == nil {
					return fmt.Errorf("missing docker environment config")
				}
				workDir, err := resolveWorkDir(exercise.Metadata.Name)
				if err != nil {
					return err
				}
				manager := environment.DockerManager{WorkDir: workDir}
				fmt.Fprintln(cmd.OutOrStdout(), "Stopping docker containers...")
				if err := manager.Teardown(ctx, entry.Dir, *exercise.Spec.Environment.Docker); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to teardown docker: %v\n", err)
				}

			default:
				return fmt.Errorf("unsupported environment type: %s", exercise.Spec.Environment.Type)
			}

			// Mark exercise as stopped
			if err := markStopped(exercise); err != nil {
				return err
			}

			// Get work directory for display
			workDir, _ := resolveWorkDir(exercise.Metadata.Name)

			fmt.Fprintln(cmd.OutOrStdout(), "")
			fmt.Fprintf(cmd.OutOrStdout(), "Exercise '%s' stopped.\n", exercise.Metadata.Name)
			fmt.Fprintln(cmd.OutOrStdout(), "Resources have been cleaned up.")
			if workDir != "" {
				fmt.Fprintln(cmd.OutOrStdout(), "")
				fmt.Fprintf(cmd.OutOrStdout(), "Your work is preserved in: %s\n", workDir)
				fmt.Fprintln(cmd.OutOrStdout(), "Use 'gymctl start' to resume or 'gymctl reset' to start fresh.")
			}

			return nil
		},
	}

	return cmd
}

func markStopped(exercise *scenario.Exercise) error {
	path, err := resolveProgressFile()
	if err != nil {
		return err
	}

	progressFile, err := progress.Load(path)
	if err != nil {
		return err
	}

	entry := progressFile.Exercises[exercise.Metadata.Name]
	entry.Status = "stopped"
	// Keep the StartedAt time to track when it was started
	progressFile.Exercises[exercise.Metadata.Name] = entry

	return progress.Save(path, progressFile)
}