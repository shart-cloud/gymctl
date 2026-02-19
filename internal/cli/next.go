package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"gymctl/internal/dialogue"
	"gymctl/internal/progress"
	"gymctl/internal/scenario"
	"gymctl/internal/ui"
)

func newNextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "next",
		Short: "Show your next assignment from GRIM",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := scenario.LoadCatalog(tasksDir)
			if err != nil {
				return HandleCommandError(cmd, err)
			}

			progressPath, err := resolveProgressFile()
			if err != nil {
				return err
			}
			pf, err := progress.Load(progressPath)
			if err != nil {
				return err
			}

			// Partition exercises by state
			type exerciseEntry struct {
				entry  scenario.CatalogEntry
				status string
			}

			var inProgress []exerciseEntry
			var available []exerciseEntry
			var locked []exerciseEntry
			var completed []exerciseEntry

			for _, entry := range entries {
				name := entry.Exercise.Metadata.Name
				st := pf.Exercises[name]
				isLocked, _ := isExerciseLocked(entry.Exercise, pf)

				switch {
				case st.Status == "completed":
					completed = append(completed, exerciseEntry{entry, st.Status})
				case st.Status == "in_progress" || st.Status == "started":
					inProgress = append(inProgress, exerciseEntry{entry, st.Status})
				case isLocked:
					locked = append(locked, exerciseEntry{entry, "locked"})
				default:
					available = append(available, exerciseEntry{entry, st.Status})
				}
			}

			out := cmd.OutOrStdout()

			// If something is in progress, that's the next thing
			if len(inProgress) > 0 {
				ex := inProgress[0].entry.Exercise
				fmt.Fprintln(out)
				ColorInfo.Fprintln(out, "  GRIM-9: You have an exercise in progress.")
				fmt.Fprintln(out)
				ui.RenderGRIMTicket(out, ex.Spec.Tasking, ex.Metadata.Name, ex.Spec.Description)
				fmt.Fprintln(out)
				dialogue.RenderJerry(out, dialogue.Jerry(ex.Spec.JerryDialog, "onStart", ex.Metadata.Name))
				fmt.Fprintln(out)
				ColorDim.Fprintf(out, "  Run: gymctl check %s\n", ex.Metadata.Name)
				fmt.Fprintln(out)
				return nil
			}

			// Sort available by (track, week, order, name)
			sort.Slice(available, func(i, j int) bool {
				ai := available[i].entry.Exercise
				aj := available[j].entry.Exercise
				if ai.Metadata.Track != aj.Metadata.Track {
					return ai.Metadata.Track < aj.Metadata.Track
				}
				if ai.Metadata.Week != aj.Metadata.Week {
					return ai.Metadata.Week < aj.Metadata.Week
				}
				if ai.Metadata.Order != aj.Metadata.Order {
					return ai.Metadata.Order < aj.Metadata.Order
				}
				return ai.Metadata.Name < aj.Metadata.Name
			})

			if len(available) > 0 {
				ex := available[0].entry.Exercise
				fmt.Fprintln(out)
				ColorInfo.Fprintln(out, "  GRIM-9 has queued your next assignment.")
				fmt.Fprintln(out)
				ui.RenderGRIMTicket(out, ex.Spec.Tasking, ex.Metadata.Name, ex.Spec.Description)
				fmt.Fprintln(out)
				dialogue.RenderJerry(out, dialogue.Jerry(ex.Spec.JerryDialog, "onStart", ex.Metadata.Name))
				fmt.Fprintln(out)
				ColorDim.Fprintf(out, "  Run: gymctl start %s\n", ex.Metadata.Name)
				fmt.Fprintln(out)
				return nil
			}

			if len(completed) == len(entries) {
				fmt.Fprintln(out)
				ColorSuccess.Fprintln(out, "  GRIM-9 MEMO: All tickets resolved. Sprint closed.")
				ColorDim.Fprintln(out, "  Jerry has been assigned to a different team.")
				fmt.Fprintln(out)
				return nil
			}

			// Only locked exercises remain
			fmt.Fprintln(out)
			ColorWarning.Fprintln(out, "  GRIM-9: All available exercises are complete.")
			ColorDim.Fprintf(out, "  %d exercise(s) remain locked pending prerequisites.\n", len(locked))
			fmt.Fprintln(out)
			for _, le := range locked {
				ex := le.entry.Exercise
				_, missing := isExerciseLocked(ex, pf)
				ColorDim.Fprintf(out, "  ◌ %s  [needs: %s]\n", ex.Metadata.Name, strings.Join(missing, ", "))
			}
			fmt.Fprintln(out)
			return nil
		},
	}
}
