package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"gymctl/internal/checks"
	"gymctl/internal/dialogue"
	"gymctl/internal/environment"
	"gymctl/internal/progress"
	"gymctl/internal/scenario"
)

type checkOptions struct {
	verbose   bool
	noCleanup bool
	spec      string
}

func newCheckCmd() *cobra.Command {
	opts := &checkOptions{}
	cmd := &cobra.Command{
		Use:   "check [exercise-name]",
		Short: "Check if the current exercise is solved",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			defer RecoverFromPanic(cmd)

			if opts.spec != "" && len(args) > 0 {
				return fmt.Errorf("cannot use exercise name argument with --spec")
			}

			var exercise *scenario.Exercise
			if opts.spec != "" {
				loaded, err := scenario.LoadExerciseFile(opts.spec)
				if err != nil {
					return err
				}
				exercise = loaded
			} else {
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
				exercise = entry.Exercise
			}

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			if err := configureExerciseKubeconfigEnv(ctx, exercise); err != nil {
				return err
			}

			workDir := ""
			if exercise.Spec.Environment.Type == "docker" {
				resolved, err := resolveWorkDir(exercise.Metadata.Name)
				if err != nil {
					return err
				}
				workDir = resolved
			}
			if !isJSONOutput() {
				ColorInfo.Fprintf(cmd.OutOrStdout(), "🔍 Checking: %s\n", exercise.Metadata.Name)
				if exercise.Spec.Environment.Kubernetes != nil && environment.IsBareNodeKubernetes(exercise.Spec.Environment.Kubernetes) {
					ColorDim.Fprintln(cmd.OutOrStdout(), "  bare-node mode: nodeExec checks can run before kubeadm bootstrap completes")
				}
				fmt.Fprintln(cmd.OutOrStdout())
			}

			results, allPassed := checks.RunExerciseChecks(ctx, exercise, workDir)

			passedCount := 0
			for _, result := range results {
				if result.Passed {
					passedCount++
				}
			}

			if isJSONOutput() {
				type checkResultJSON struct {
					Name    string `json:"name"`
					Passed  bool   `json:"passed"`
					Message string `json:"message"`
				}
				type checkResponseJSON struct {
					Exercise        string            `json:"exercise"`
					AllPassed       bool              `json:"allPassed"`
					PassedCount     int               `json:"passedCount"`
					TotalCount      int               `json:"totalCount"`
					Score           int               `json:"score"`
					PointsAvailable int               `json:"pointsAvailable"`
					Checks          []checkResultJSON `json:"checks"`
				}

				jsonChecks := make([]checkResultJSON, 0, len(results))
				for _, result := range results {
					jsonChecks = append(jsonChecks, checkResultJSON{
						Name:    result.Name,
						Passed:  result.Passed,
						Message: result.Message,
					})
				}

				score := 0
				if allPassed {
					if err := markCompleted(exercise); err != nil {
						return err
					}
					resolvedProgressPath, err := resolveProgressFile()
					if err != nil {
						return err
					}
					progressData, err := progress.Load(resolvedProgressPath)
					if err != nil {
						return err
					}
					score = progressData.Exercises[exercise.Metadata.Name].Score
				}

				response := checkResponseJSON{
					Exercise:        exercise.Metadata.Name,
					AllPassed:       allPassed,
					PassedCount:     passedCount,
					TotalCount:      len(results),
					Score:           score,
					PointsAvailable: defaultPoints(exercise.Spec.Points),
					Checks:          jsonChecks,
				}
				if err := writeJSON(cmd.OutOrStdout(), response); err != nil {
					return err
				}
				if !allPassed {
					return fmt.Errorf("checks failed")
				}
				return nil
			}

			progressBar := ProgressBar(passedCount, len(results), 20)
			fmt.Fprintln(cmd.OutOrStdout(), progressBar)
			fmt.Fprintln(cmd.OutOrStdout())

			for _, result := range results {
				checkLine := FormatCheckResult(result.Name, result.Passed, "")
				if opts.verbose && result.Message != "" {
					checkLine = FormatCheckResult(result.Name, result.Passed, result.Message)
				}
				fmt.Fprintln(cmd.OutOrStdout(), checkLine)
			}

			fmt.Fprintln(cmd.OutOrStdout())

			if allPassed {
				if err := markCompleted(exercise); err != nil {
					return err
				}
				dialogue.RenderJerry(cmd.OutOrStdout(), dialogue.Jerry(exercise.Spec.JerryDialog, "onComplete", exercise.Metadata.Name))
				fmt.Fprintln(cmd.OutOrStdout())
				ColorSuccess.Fprintln(cmd.OutOrStdout(), "🎉 Exercise complete! Well done!")

				// Run cleanup hook if not disabled
				if !opts.noCleanup {
					cleanupConfig := &CleanupConfig{
						AutoClean:       false, // Prompt by default
						SkipClean:       false,
						CleanImages:     true,
						CleanContainers: true,
						CleanVolumes:    true,
						Exercise:        exercise.Metadata.Name,
					}
					CleanupHook(cmd, exercise, cleanupConfig)
				}

				return nil
			}

			dialogue.RenderJerry(cmd.OutOrStdout(), dialogue.Jerry(exercise.Spec.JerryDialog, "onCheckFail", exercise.Metadata.Name))
			fmt.Fprintln(cmd.OutOrStdout())
			ColorWarning.Fprintf(cmd.OutOrStdout(), "⚠ Exercise not complete. %d/%d checks passed.\n", passedCount, len(results))
			if !opts.verbose {
				ColorDim.Fprintln(cmd.OutOrStdout(), "Use --verbose flag for detailed error messages.")
			}
			return fmt.Errorf("checks failed")
		},
	}

	cmd.Flags().BoolVar(&opts.verbose, "verbose", false, "Show check details")
	cmd.Flags().BoolVar(&opts.noCleanup, "no-cleanup", false, "Skip cleanup after successful check")
	cmd.Flags().StringVar(&opts.spec, "spec", "", "Path to an exercise spec file")

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
	baseScore := defaultPoints(exercise.Spec.Points)
	penalty := 0
	for i := 0; i < entry.HintsUsed && i < len(exercise.Spec.Hints); i++ {
		penalty += exercise.Spec.Hints[i].Cost
	}
	if penalty > baseScore {
		penalty = baseScore
	}
	entry.Score = baseScore - penalty
	if entry.StartedAt != "" {
		if t, err := time.Parse(time.RFC3339, entry.StartedAt); err == nil {
			entry.TimeSpent = formatDuration(time.Since(t).Round(time.Second))
		}
	}
	progressFile.Exercises[exercise.Metadata.Name] = entry

	return progress.Save(path, progressFile)
}
