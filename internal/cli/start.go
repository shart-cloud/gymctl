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

	"gymctl/internal/dialogue"
	"gymctl/internal/environment"
	"gymctl/internal/progress"
	"gymctl/internal/scenario"
	"gymctl/internal/ui"
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
			defer RecoverFromPanic(cmd)

			entries, err := scenario.LoadCatalog(tasksDir)
			if err != nil {
				return HandleCommandError(cmd, err)
			}
			entry, found := scenario.FindByName(entries, args[0])
			if !found {
				return HandleCommandError(cmd, WrapErrorWithHint(
					fmt.Errorf("exercise not found: %s", args[0]),
					"Check the exercise name is correct",
					"gymctl list",
				))
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
				createCluster := shouldCreateCluster(k8s, opts.noCluster)
				provider, state, err := resolveKubernetesProviderAndState(exercise.Metadata.Name, k8s)
				if err != nil {
					return err
				}

				ensureSpec := k8s
				if !createCluster {
					ensureSpec = withCreateCluster(k8s, false)
				}
				namespace := k8s.Namespace
				if namespace == "" {
					namespace = "default"
				}

				if createCluster {
					if provider.Name() == environment.KubernetesProviderVagrant {
						// Vagrant streams its own output; skip the spinner to avoid
						// interleaved terminal writes. Vagrant up can take 5-15 minutes
						// on Windows/macOS so live output is important.
						fmt.Fprintln(cmd.OutOrStdout(), "Starting Vagrant VMs with VirtualBox (this may take several minutes)...")
						err = provider.Ensure(ctx, entry.Dir, ensureSpec, state)
					} else {
						err = WithSpinner("Preparing kubernetes environment (this may take a minute)", func() error {
							return provider.Ensure(ctx, entry.Dir, ensureSpec, state)
						})
					}
				} else {
					err = provider.Ensure(ctx, entry.Dir, ensureSpec, state)
				}
				if err != nil {
					return err
				}

				if err := environment.SaveExerciseState(state); err != nil {
					return err
				}

				if !environment.IsBareNodeKubernetes(k8s) {
					kubeconfigPath, err := provider.ExportKubeconfig(ctx, state, "")
					if err != nil {
						return err
					}
					if err := environment.SaveExerciseState(state); err != nil {
						return err
					}
					_ = os.Setenv("KUBECONFIG", kubeconfigPath)
				}

				if environment.IsBareNodeKubernetes(k8s) {
					ColorInfo.Fprintln(cmd.OutOrStdout(), "Vagrant bare-node mode: skipping manifest apply and wait conditions until cluster bootstrap is done.")
				}

				manifests := environment.ResolveManifestPaths(entry.Dir, k8s.SetupManifests)
				if len(manifests) > 0 && !environment.IsBareNodeKubernetes(k8s) {
					err = WithSpinner("Applying setup manifests", func() error {
						return environment.ApplyManifests(ctx, namespace, manifests)
					})
					if err != nil {
						return err
					}
				}

				if !environment.IsBareNodeKubernetes(k8s) {
					for _, wait := range k8s.WaitFor {
						err = WithSpinner(fmt.Sprintf("Waiting for %s", wait.Resource), func() error {
							return environment.WaitForCondition(ctx, namespace, wait.Resource, wait.Condition, wait.Timeout)
						})
						if err != nil {
							return err
						}
					}
				}

				if len(exercise.Spec.Environment.CustomSetup) > 0 {
					err = WithSpinner("Running custom setup steps", func() error {
						return environment.RunCustomSetup(ctx, entry.Dir, exercise, exercise.Spec.Environment.CustomSetup)
					})
					if err != nil {
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
				err = WithSpinner("Setting up docker environment", func() error {
					return manager.Setup(ctx, entry.Dir, *exercise.Spec.Environment.Docker)
				})
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported environment type: %s", exercise.Spec.Environment.Type)
			}

			ui.RenderGRIMTicket(cmd.OutOrStdout(), exercise.Spec.Tasking, exercise.Metadata.Name, exercise.Spec.Description)
			fmt.Fprintln(cmd.OutOrStdout())
			dialogue.RenderJerry(cmd.OutOrStdout(), dialogue.Jerry(exercise.Spec.JerryDialog, "onStart", exercise.Metadata.Name))
			fmt.Fprintln(cmd.OutOrStdout())
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
					// Handle both files and directories
					if strings.HasSuffix(copySpec.From, "/") {
						// Source ends with /, treat as directory contents
						srcPath = strings.TrimSuffix(srcPath, "/")
					}
					if err := copyPath(srcPath, dstPath); err != nil {
						return fmt.Errorf("copy %s: %w", copySpec.From, err)
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

	cmd.Flags().BoolVar(&opts.noCluster, "no-cluster", false, "Skip kubernetes cluster creation")

	return cmd
}

func printExerciseIntro(cmd *cobra.Command, exercise *scenario.Exercise) {
	out := cmd.OutOrStdout()

	// Print title with color
	fmt.Fprintln(out, "")
	ColorHeader.Fprintln(out, exercise.Metadata.Title)
	ColorDim.Fprintln(out, strings.Repeat("═", len(exercise.Metadata.Title)))

	// Print metadata
	ColorInfo.Fprintf(out, "📚 Difficulty: ")
	fmt.Fprintln(out, DifficultyBadge(exercise.Spec.Difficulty))
	if exercise.Spec.EstimatedTime != "" {
		ColorInfo.Fprint(out, "⏱  Estimated Time: ")
		ColorTime.Fprintln(out, exercise.Spec.EstimatedTime)
	}

	if exercise.Spec.Description != "" {
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, exercise.Spec.Description)
	}

	if len(exercise.Spec.LearningOutcomes) > 0 {
		fmt.Fprintln(out, "")
		ColorBold.Fprintln(out, "📝 Learning Objectives:")
		for _, item := range exercise.Spec.LearningOutcomes {
			fmt.Fprintf(out, "  • %s\n", item)
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

func copyPath(srcPath, dstPath string) error {
	// Check if source is a file or directory
	info, err := os.Stat(srcPath)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return copyDir(srcPath, dstPath)
	}
	return copyFile(srcPath, dstPath)
}

func copyDir(source string, destination string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relPath)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		_, err = entry.Info()
		if err != nil {
			return err
		}
		return copyFile(path, target)
	})
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
