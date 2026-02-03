package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"gymctl/internal/environment"
	"gymctl/internal/scenario"
)

type cleanOptions struct {
	all bool
}

func newCleanCmd() *cobra.Command {
	opts := &cleanOptions{}
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean up environments",
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := scenario.LoadCatalog(tasksDir)
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			if opts.all {
				for _, entry := range entries {
					exercise := entry.Exercise
					if exercise.Spec.Environment.Type != "docker" || exercise.Spec.Environment.Docker == nil {
						continue
					}
					workDir, err := resolveWorkDir(exercise.Metadata.Name)
					if err != nil {
						return err
					}
					manager := environment.DockerManager{WorkDir: workDir}
					_ = manager.Teardown(ctx, entry.Dir, *exercise.Spec.Environment.Docker)
				}
			} else {
				current, err := loadCurrentExercise()
				if err == nil {
					if entry, found := scenario.FindByName(entries, current); found {
						exercise := entry.Exercise
						if exercise.Spec.Environment.Type == "docker" && exercise.Spec.Environment.Docker != nil {
							workDir, err := resolveWorkDir(exercise.Metadata.Name)
							if err != nil {
								return err
							}
							manager := environment.DockerManager{WorkDir: workDir}
							_ = manager.Teardown(ctx, entry.Dir, *exercise.Spec.Environment.Docker)
						}
					}
				}
			}

			manager := environment.KindManager{ClusterName: "jerry-gym"}
			_, _ = manager.Exists(ctx)
			_ = manager.Delete(ctx)

			gymDir, err := resolveGymDir()
			if err == nil {
				_ = os.RemoveAll(filepath.Join(gymDir, "workdir"))
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Cleanup complete.")
			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.all, "all", false, "Clean all exercises")
	return cmd
}
