package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"gymctl/internal/environment"
	"gymctl/internal/progress"
	"gymctl/internal/scenario"
)

type resetOptions struct {
	noCluster bool
	keepWork  bool
}

func newResetCmd() *cobra.Command {
	opts := &resetOptions{}
	cmd := &cobra.Command{
		Use:   "reset [exercise-name]",
		Short: "Reset the current exercise",
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
				if opts.noCluster {
					createCluster = false
				}
				namespace := k8s.Namespace
				if namespace == "" {
					namespace = "default"
				}

				if createCluster {
					manager := environment.KindManager{ClusterName: "jerry-gym"}
					_ = manager.Delete(ctx)
					if err := manager.Create(ctx, k8s.KindConfig); err != nil {
						return err
					}
				}

				manifests := environment.ResolveManifestPaths(entry.Dir, k8s.SetupManifests)
				if len(manifests) > 0 {
					if err := environment.ApplyManifests(ctx, namespace, manifests); err != nil {
						return err
					}
				}

				for _, wait := range k8s.WaitFor {
					if err := environment.WaitForCondition(ctx, namespace, wait.Resource, wait.Condition, wait.Timeout); err != nil {
						return err
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
				if err := manager.Teardown(ctx, entry.Dir, *exercise.Spec.Environment.Docker); err != nil {
					return err
				}
				if err := manager.Setup(ctx, entry.Dir, *exercise.Spec.Environment.Docker); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported environment type: %s", exercise.Spec.Environment.Type)
			}

			if err := markReset(exercise); err != nil {
				return err
			}
			if err := writeCurrentExercise(exercise.Metadata.Name); err != nil {
				return err
			}

			// Handle work directory
			workDir, err := resolveWorkDir(exercise.Metadata.Name)
			if err == nil && !opts.keepWork {
				// Check if work directory exists
				if _, err := os.Stat(workDir); err == nil {
					// Ask for confirmation
					fmt.Fprintf(cmd.OutOrStdout(), "Reset work directory %s? [y/N]: ", workDir)
					reader := bufio.NewReader(os.Stdin)
					answer, _ := reader.ReadString('\n')
					answer = strings.TrimSpace(strings.ToLower(answer))

					if answer == "y" || answer == "yes" {
						if err := os.RemoveAll(workDir); err != nil {
							fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to remove work directory: %v\n", err)
						} else {
							// Recreate empty work directory
							if err := os.MkdirAll(workDir, 0o755); err != nil {
								fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to create work directory: %v\n", err)
							}
							fmt.Fprintln(cmd.OutOrStdout(), "Work directory cleared.")
						}
					} else {
						fmt.Fprintln(cmd.OutOrStdout(), "Work directory preserved.")
					}
				}
			}

			fmt.Fprintln(cmd.OutOrStdout(), "")
			fmt.Fprintln(cmd.OutOrStdout(), "Exercise reset successfully.")
			fmt.Fprintf(cmd.OutOrStdout(), "Work directory: %s\n", workDir)
			fmt.Fprintln(cmd.OutOrStdout(), "")
			fmt.Fprintln(cmd.OutOrStdout(), "To navigate to your work directory, run:")
			fmt.Fprintf(cmd.OutOrStdout(), "  cd %s\n", workDir)
			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.noCluster, "no-cluster", false, "Skip kind cluster recreation")
	cmd.Flags().BoolVar(&opts.keepWork, "keep-work", false, "Keep work directory contents")
	return cmd
}

func markReset(exercise *scenario.Exercise) error {
	path, err := resolveProgressFile()
	if err != nil {
		return err
	}
	progressFile, err := progress.Load(path)
	if err != nil {
		return err
	}
	entry := progressFile.Exercises[exercise.Metadata.Name]
	entry.Resets++
	entry.Status = "in_progress"
	entry.StartedAt = time.Now().UTC().Format(time.RFC3339)
	progressFile.Exercises[exercise.Metadata.Name] = entry
	return progress.Save(path, progressFile)
}
