package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"gymctl/internal/checks"
	"gymctl/internal/progress"
	"gymctl/internal/scenario"
)

type checkOptions struct {
	verbose bool
}

func newCheckCmd() *cobra.Command {
	opts := &checkOptions{}
	cmd := &cobra.Command{
		Use:   "check [exercise-name]",
		Short: "Check if the current exercise is solved",
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

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			workDir := ""
			if entry.Exercise.Spec.Environment.Type == "docker" {
				resolved, err := resolveWorkDir(entry.Exercise.Metadata.Name)
				if err != nil {
					return err
				}
				workDir = resolved
			}
			results, allPassed := checks.RunExerciseChecks(ctx, entry.Exercise, workDir)
			for _, result := range results {
				status := "FAIL"
				if result.Passed {
					status = "OK"
				}
				if opts.verbose && result.Message != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s - %s\n", status, result.Name, result.Message)
					continue
				}
				fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s\n", status, result.Name)
			}

			if allPassed {
				if err := markCompleted(entry.Exercise); err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), "\nExercise complete.")
				return nil
			}

			fmt.Fprintln(cmd.OutOrStdout(), "\nExercise not complete.")
			return fmt.Errorf("checks failed")
		},
	}

	cmd.Flags().BoolVar(&opts.verbose, "verbose", false, "Show check details")

	return cmd
}

func markCompleted(exercise *scenario.Exercise) error {
	path, err := resolveProgressFile()
	if err != nil {
		return err
	}

	progressFile, err := progress.Load(path)
	if err != nil {
		return err
	}

	entry := progressFile.Exercises[exercise.Metadata.Name]
	entry.Status = "completed"
	entry.CompletedAt = time.Now().UTC().Format(time.RFC3339)
	entry.Score = defaultPoints(exercise.Spec.Points)
	progressFile.Exercises[exercise.Metadata.Name] = entry

	return progress.Save(path, progressFile)
}
