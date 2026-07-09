package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"gymctl/internal/progress"
	"gymctl/internal/scenario"
	"gymctl/internal/ui"
)

func newInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info [exercise-name]",
		Short: "Show detailed information about an exercise",
		Long: `Show detailed information about the current or specified exercise.
This command is designed to work well inside containers and provides
all the information students need without requiring the full TUI.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var exerciseName string
			if len(args) > 0 {
				exerciseName = args[0]
			} else {
				// Try to get current exercise
				current, err := loadCurrentExercise()
				if err != nil {
					return fmt.Errorf("no exercise specified and no current exercise: %w", err)
				}
				exerciseName = current
			}

			// Load catalog
			entries, err := loadCatalogEntries()
			if err != nil {
				return fmt.Errorf("load catalog: %w", err)
			}

			// Find exercise
			entry, found := scenario.FindByName(entries, exerciseName)
			if !found {
				return fmt.Errorf("exercise not found: %s", exerciseName)
			}

			exercise := entry.Exercise

			// Load progress
			progressPath, err := resolveProgressFile()
			if err != nil {
				return fmt.Errorf("resolve progress file: %w", err)
			}

			prog, err := progress.Load(progressPath)
			if err != nil {
				return fmt.Errorf("load progress: %w", err)
			}

			// Display exercise information
			return displayExerciseInfo(cmd, exercise, prog)
		},
	}
}

func displayExerciseInfo(cmd *cobra.Command, exercise *scenario.Exercise, prog *progress.File) error {
	out := cmd.OutOrStdout()

	// Brief — flat title + description, no box-art.
	fmt.Fprintln(out)
	ui.RenderBrief(out, exercise.Spec.Brief, exercise.Metadata.DisplayTitle(), exercise.Spec.Description)
	fmt.Fprintln(out)

	// Status + metadata on a single dim line.
	status := "not_started"
	if st, ok := prog.Exercises[exercise.Metadata.Name]; ok {
		status = st.Status
	}
	statusText := map[string]string{
		"not_started": "not started",
		"in_progress": "in progress",
		"completed":   "completed ✓",
	}[status]

	ColorDim.Fprint(out, "  ")
	switch status {
	case "completed":
		ColorSuccess.Fprint(out, statusText)
	case "in_progress":
		ColorWarning.Fprint(out, statusText)
	default:
		ColorDim.Fprint(out, statusText)
	}
	meta := []string{DifficultyBadge(exercise.Spec.Difficulty)}
	if exercise.Spec.EstimatedTime != "" {
		meta = append(meta, ColorTime.Sprint(exercise.Spec.EstimatedTime))
	}
	if exercise.Spec.Points > 0 {
		meta = append(meta, fmt.Sprintf("%d pts", exercise.Spec.Points))
	}
	ColorDim.Fprintf(out, "   %s\n", strings.Join(meta, "   "))
	fmt.Fprintln(out)

	// Learning objectives.
	if len(exercise.Spec.LearningOutcomes) > 0 {
		ColorDim.Fprintln(out, "  learning objectives")
		for _, outcome := range exercise.Spec.LearningOutcomes {
			ColorDim.Fprint(out, "    · ")
			fmt.Fprintln(out, outcome)
		}
		fmt.Fprintln(out)
	}

	// Available hints.
	if len(exercise.Spec.Hints) > 0 {
		ColorDim.Fprintln(out, "  hints")
		for i, hint := range exercise.Spec.Hints {
			if hint.Cost == 0 {
				ColorDim.Fprintf(out, "    %d. free — ", i+1)
				ColorInfo.Fprintf(out, "gymctl hint %s\n", exercise.Metadata.Name)
			} else {
				ColorDim.Fprintf(out, "    %d. costs %d pts — ", i+1, hint.Cost)
				ColorInfo.Fprintf(out, "gymctl hint %s --hint %d\n", exercise.Metadata.Name, i+1)
			}
		}
		fmt.Fprintln(out)
	}

	// Useful commands.
	ColorDim.Fprintln(out, "  commands")
	ColorDim.Fprint(out, "    check   ")
	ColorInfo.Fprintln(out, "gymctl check")
	ColorDim.Fprint(out, "    hint    ")
	ColorInfo.Fprintln(out, "gymctl hint")
	ColorDim.Fprint(out, "    list    ")
	ColorInfo.Fprintln(out, "gymctl list")
	ColorDim.Fprint(out, "    tui     ")
	ColorInfo.Fprintln(out, "gymctl shell")
	fmt.Fprintln(out)

	return nil
}
