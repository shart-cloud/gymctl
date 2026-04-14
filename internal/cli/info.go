package cli

import (
	"fmt"
	"os"
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

	// Header with exercise name and status
	fmt.Fprintln(out)
	ColorHeader.Fprintln(out, "┌─ "+strings.ToUpper(exercise.Metadata.Title)+" ─")
	ColorDim.Fprintln(out, "│")

	// Current status
	status := "not_started"
	if st, ok := prog.Exercises[exercise.Metadata.Name]; ok {
		status = st.Status
	}

	statusText := map[string]string{
		"not_started": "Not Started",
		"in_progress": "In Progress",
		"completed":   "✅ Completed",
	}[status]

	ColorDim.Fprint(out, "│ Status: ")
	switch status {
	case "completed":
		ColorSuccess.Fprintln(out, statusText)
	case "in_progress":
		ColorWarning.Fprintln(out, statusText)
	default:
		ColorDim.Fprintln(out, statusText)
	}

	// Metadata
	ColorDim.Fprint(out, "│ Difficulty: ")
	fmt.Fprintln(out, DifficultyBadge(exercise.Spec.Difficulty))

	if exercise.Spec.EstimatedTime != "" {
		ColorDim.Fprint(out, "│ Time: ")
		ColorTime.Fprintln(out, exercise.Spec.EstimatedTime)
	}

	if exercise.Spec.Points > 0 {
		ColorDim.Fprint(out, "│ Points: ")
		ColorInfo.Fprintln(out, exercise.Spec.Points)
	}

	ColorDim.Fprintln(out, "│")

	// GRIM Ticket
	if exercise.Spec.Tasking != nil || exercise.Spec.Description != "" {
		ui.RenderGRIMTicket(out, exercise.Spec.Tasking, exercise.Metadata.Name, exercise.Spec.Description)
		fmt.Fprintln(out)
	}

	// Description
	if exercise.Spec.Description != "" {
		ColorDim.Fprintln(out, "│ Description:")
		for _, line := range strings.Split(exercise.Spec.Description, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				ColorDim.Fprint(out, "│   ")
				fmt.Fprintln(out, line)
			} else {
				ColorDim.Fprintln(out, "│")
			}
		}
		ColorDim.Fprintln(out, "│")
	}

	// Learning outcomes
	if len(exercise.Spec.LearningOutcomes) > 0 {
		ColorDim.Fprintln(out, "│ 📝 Learning Objectives:")
		for _, outcome := range exercise.Spec.LearningOutcomes {
			ColorDim.Fprint(out, "│   • ")
			fmt.Fprintln(out, outcome)
		}
		ColorDim.Fprintln(out, "│")
	}

	// Available hints
	if len(exercise.Spec.Hints) > 0 {
		ColorDim.Fprintln(out, "│ 💡 Available Hints:")
		for i, hint := range exercise.Spec.Hints {
			if hint.Cost == 0 {
				ColorDim.Fprintf(out, "│   %d. Free hint - run: ", i+1)
				ColorInfo.Fprintf(out, "gymctl hint %s\n", exercise.Metadata.Name)
			} else {
				ColorDim.Fprintf(out, "│   %d. Costs %d points - run: ", i+1, hint.Cost)
				ColorInfo.Fprintf(out, "gymctl hint %s --hint %d\n", exercise.Metadata.Name, i+1)
			}
		}
		ColorDim.Fprintln(out, "│")
	}

	// Current directory info
	wd, err := os.Getwd()
	if err == nil {
		ColorDim.Fprint(out, "│ Work Directory: ")
		ColorInfo.Fprintln(out, wd)
		ColorDim.Fprintln(out, "│")
	}

	// Footer with useful commands
	ColorDim.Fprintln(out, "│ 🔧 Useful Commands:")
	ColorDim.Fprint(out, "│   Check progress: ")
	ColorInfo.Fprintln(out, "gymctl check")
	ColorDim.Fprint(out, "│   Get hints: ")
	ColorInfo.Fprintln(out, "gymctl hint")
	ColorDim.Fprint(out, "│   View all exercises: ")
	ColorInfo.Fprintln(out, "gymctl list")
	ColorDim.Fprint(out, "│   Return to TUI: ")
	ColorInfo.Fprintln(out, "gymctl shell")

	ColorDim.Fprintln(out, "└─")
	fmt.Fprintln(out)

	return nil
}
