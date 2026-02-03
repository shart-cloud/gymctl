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
	verbose   bool
	noCleanup bool
}

func newCheckCmd() *cobra.Command {
	opts := &checkOptions{}
	cmd := &cobra.Command{
		Use:   "check [exercise-name]",
		Short: "Check if the current exercise is solved",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			defer RecoverFromPanic(cmd)

			name := ""
			if len(args) == 1 {
				name = args[0]
			} else {
				current, err := loadCurrentExercise()
				if err != nil {
					return WrapErrorWithHint(
						fmt.Errorf("no exercise specified and no current exercise set"),
						"Start an exercise first or specify one",
						"gymctl start <exercise-name>",
					)
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
			// Show checking header
			ColorInfo.Fprintf(cmd.OutOrStdout(), "🔍 Checking: %s\n", entry.Exercise.Metadata.Name)
			fmt.Fprintln(cmd.OutOrStdout())

			results, allPassed := checks.RunExerciseChecks(ctx, entry.Exercise, workDir)

			// Count passed checks
			passedCount := 0
			for _, result := range results {
				if result.Passed {
					passedCount++
				}
			}

			// Show progress bar
			progressBar := ProgressBar(passedCount, len(results), 20)
			fmt.Fprintln(cmd.OutOrStdout(), progressBar)
			fmt.Fprintln(cmd.OutOrStdout())

			// Show individual check results
			for _, result := range results {
				checkLine := FormatCheckResult(result.Name, result.Passed, "")
				if opts.verbose && result.Message != "" {
					checkLine = FormatCheckResult(result.Name, result.Passed, result.Message)
				}
				fmt.Fprintln(cmd.OutOrStdout(), checkLine)
			}

			fmt.Fprintln(cmd.OutOrStdout())

			if allPassed {
				if err := markCompleted(entry.Exercise); err != nil {
					return err
				}
				ColorSuccess.Fprintln(cmd.OutOrStdout(), "🎉 Exercise complete! Well done!")

				// Run cleanup hook if not disabled
				if !opts.noCleanup {
					cleanupConfig := &CleanupConfig{
						AutoClean:       false, // Prompt by default
						SkipClean:       false,
						CleanImages:     true,
						CleanContainers: true,
						CleanVolumes:    true,
						Exercise:        entry.Exercise.Metadata.Name,
					}
					CleanupHook(cmd, entry.Exercise, cleanupConfig)
				}

				return nil
			}

			ColorWarning.Fprintf(cmd.OutOrStdout(), "⚠ Exercise not complete. %d/%d checks passed.\n", passedCount, len(results))
			if !opts.verbose {
				ColorDim.Fprintln(cmd.OutOrStdout(), "Use --verbose flag for detailed error messages.")
			}
			return fmt.Errorf("checks failed")
		},
	}

	cmd.Flags().BoolVar(&opts.verbose, "verbose", false, "Show check details")
	cmd.Flags().BoolVar(&opts.noCleanup, "no-cleanup", false, "Skip cleanup after successful check")

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
