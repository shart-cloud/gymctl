package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"gymctl/internal/environment"
	"gymctl/internal/progress"
	"gymctl/internal/scenario"
)

type startOptions struct {
	noCluster bool
}

func newStartCmd() *cobra.Command {
	opts := &startOptions{}
	cmd := &cobra.Command{
		Use:   "start <exercise-name>",
		Short: "Start an exercise",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := scenario.LoadCatalog(tasksDir)
			if err != nil {
				return err
			}
			entry, found := scenario.FindByName(entries, args[0])
			if !found {
				return fmt.Errorf("exercise not found: %s", args[0])
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
					exists, err := manager.Exists(ctx)
					if err != nil {
						return err
					}
					if exists {
						fmt.Fprintln(cmd.OutOrStdout(), "Cleaning existing kind cluster...")
						if err := manager.Delete(ctx); err != nil {
							return err
						}
					}
					fmt.Fprintln(cmd.OutOrStdout(), "Creating kind cluster...")
					if err := manager.Create(ctx, k8s.KindConfig); err != nil {
						return err
					}
				}

				manifests := environment.ResolveManifestPaths(entry.Dir, k8s.SetupManifests)
				if len(manifests) > 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "Applying setup manifests...")
					if err := environment.ApplyManifests(ctx, namespace, manifests); err != nil {
						return err
					}
				}

				for _, wait := range k8s.WaitFor {
					fmt.Fprintf(cmd.OutOrStdout(), "Waiting for %s...\n", wait.Resource)
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
				fmt.Fprintln(cmd.OutOrStdout(), "Setting up docker environment...")
				if err := manager.Setup(ctx, entry.Dir, *exercise.Spec.Environment.Docker); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported environment type: %s", exercise.Spec.Environment.Type)
			}

			printExerciseIntro(cmd, exercise)

			if err := markStarted(exercise); err != nil {
				return err
			}

			if err := writeCurrentExercise(exercise.Metadata.Name); err != nil {
				return err
			}

			// Create and show work directory
			workDir, err := resolveWorkDir(exercise.Metadata.Name)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(workDir, 0o755); err != nil {
				return fmt.Errorf("create work directory: %w", err)
			}

			// Copy files if Docker environment specifies copyFiles
			if exercise.Spec.Environment.Docker != nil && len(exercise.Spec.Environment.Docker.CopyFiles) > 0 {
				for _, copySpec := range exercise.Spec.Environment.Docker.CopyFiles {
					srcPath := filepath.Join(entry.Dir, copySpec.From)
					dstPath := filepath.Join(workDir, copySpec.To)
					if err := copyFile(srcPath, dstPath); err != nil {
						return fmt.Errorf("copy file %s: %w", copySpec.From, err)
					}
				}
				fmt.Fprintln(cmd.OutOrStdout(), "Exercise files copied to work directory.")
			}

			// Print work directory info
			fmt.Fprintln(cmd.OutOrStdout(), "")
			fmt.Fprintf(cmd.OutOrStdout(), "Work directory: %s\n", workDir)
			fmt.Fprintln(cmd.OutOrStdout(), "")
			fmt.Fprintln(cmd.OutOrStdout(), "To navigate to your work directory, run:")
			fmt.Fprintf(cmd.OutOrStdout(), "  cd %s\n", workDir)
			fmt.Fprintln(cmd.OutOrStdout(), "")

			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.noCluster, "no-cluster", false, "Skip kind cluster creation")

	return cmd
}

func printExerciseIntro(cmd *cobra.Command, exercise *scenario.Exercise) {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "")
	fmt.Fprintf(out, "%s\n", exercise.Metadata.Title)
	fmt.Fprintln(out, strings.Repeat("-", len(exercise.Metadata.Title)))
	if exercise.Spec.Description != "" {
		fmt.Fprintln(out, exercise.Spec.Description)
	}
	if len(exercise.Spec.LearningOutcomes) > 0 {
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "Learning objectives:")
		for _, item := range exercise.Spec.LearningOutcomes {
			fmt.Fprintf(out, "- %s\n", item)
		}
	}
	fmt.Fprintln(out, "")
}

func markStarted(exercise *scenario.Exercise) error {
	path, err := resolveProgressFile()
	if err != nil {
		return err
	}

	progressFile, err := progress.Load(path)
	if err != nil {
		return err
	}

	entry := progressFile.Exercises[exercise.Metadata.Name]
	if entry.Status == "" || entry.Status == "not_started" {
		entry.StartedAt = time.Now().UTC().Format(time.RFC3339)
		entry.HintsUsed = 0
		entry.Resets = 0
	}
	entry.Status = "in_progress"
	progressFile.Exercises[exercise.Metadata.Name] = entry

	return progress.Save(path, progressFile)
}

func copyFile(srcPath, dstPath string) error {
	// Create destination directory if needed
	dstDir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}

	// Copy file
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}

	// Preserve file permissions
	info, err := src.Stat()
	if err != nil {
		return err
	}
	return dst.Chmod(info.Mode())
}
