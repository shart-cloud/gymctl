package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"gymctl/internal/pet"
	"gymctl/internal/progress"
	"gymctl/internal/scenario"
	"gymctl/internal/ui"
)

func newNextCmd() *cobra.Command {
	opts := struct {
		track           string
		includeScaffold bool
	}{}

	cmd := &cobra.Command{
		Use:   "next",
		Short: "Show your next recommended exercise",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := loadCatalogEntries()
			if err != nil {
				return HandleCommandError(cmd, err)
			}
			entries = filterScaffoldEntries(entries, opts.includeScaffold)

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

				if opts.track != "" && !strings.EqualFold(entry.Exercise.Metadata.Track, opts.track) {
					continue
				}

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
				ColorInfo.Fprintln(out, "  you have an exercise in progress.")
				fmt.Fprintln(out)
				ui.RenderBrief(out, ex.Spec.Brief, ex.Metadata.DisplayTitle(), ex.Spec.Description)
				fmt.Fprintln(out)
				pet.RenderCLI(out, ex.Spec.JerryDialog, "onStart", ex.Metadata.Name)
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
				ColorInfo.Fprintln(out, "  next up:")
				fmt.Fprintln(out)
				ui.RenderBrief(out, ex.Spec.Brief, ex.Metadata.DisplayTitle(), ex.Spec.Description)
				fmt.Fprintln(out)
				pet.RenderCLI(out, ex.Spec.JerryDialog, "onStart", ex.Metadata.Name)
				fmt.Fprintln(out)
				ColorDim.Fprintf(out, "  Run: gymctl start %s\n", ex.Metadata.Name)
				fmt.Fprintln(out)
				return nil
			}

			if len(completed) == len(entries) {
				fmt.Fprintln(out)
				ColorSuccess.Fprintln(out, "  all exercises complete. nice work.")
				ColorDim.Fprintln(out, "  jerry has been reassigned to another team.")
				fmt.Fprintln(out)
				return nil
			}

			// Only locked exercises remain
			fmt.Fprintln(out)
			ColorWarning.Fprintln(out, "  all available exercises are complete.")
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

	cmd.Flags().StringVar(&opts.track, "track", "", "Filter by track")
	cmd.Flags().BoolVar(&opts.includeScaffold, "include-scaffold", false, "Include scaffolded exercises")
	return cmd
}
