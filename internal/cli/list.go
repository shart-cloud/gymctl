package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"gymctl/internal/progress"
	"gymctl/internal/scenario"
)

type listOptions struct {
	track      string
	difficulty string
	week       int
}

func newListCmd() *cobra.Command {
	opts := &listOptions{}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available exercises",
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := scenario.LoadCatalog(tasksDir)
			if err != nil {
				return err
			}

			// Load progress to show completion status
			progressPath, err := resolveProgressFile()
			if err != nil {
				return err
			}
			progressFile, err := progress.Load(progressPath)
			if err != nil {
				return err
			}

			filtered := filterList(entries, opts)
			if len(filtered) == 0 {
				ColorWarning.Fprintln(cmd.OutOrStdout(), "No exercises found.")
				return nil
			}

			sort.Slice(filtered, func(i, j int) bool {
				ai := filtered[i].Exercise
				aj := filtered[j].Exercise
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

			// Count completed exercises
			completedCount := 0
			for _, entry := range filtered {
				if status, ok := progressFile.Exercises[entry.Exercise.Metadata.Name]; ok && status.Status == "completed" {
					completedCount++
				}
			}

			ColorHeader.Fprintln(cmd.OutOrStdout(), "🎯 Available Exercises")
			ColorDim.Fprintf(cmd.OutOrStdout(), "%d exercises total, %d completed\n", len(filtered), completedCount)

			currentTrack := ""
			for _, entry := range filtered {
				exercise := entry.Exercise
				if exercise.Metadata.Track != currentTrack {
					currentTrack = exercise.Metadata.Track
					fmt.Fprintln(cmd.OutOrStdout())
					ColorTrack.Fprintf(cmd.OutOrStdout(), "▸ %s\n", strings.ToUpper(currentTrack))
				}

				// Get status icon
				status := progressFile.Exercises[exercise.Metadata.Name]
				statusIcon := FormatStatus(status.Status)

				desc := firstLine(exercise.Spec.Description)
				estimated := exercise.Spec.EstimatedTime
				if estimated == "" {
					estimated = "-"
				} else {
					estimated = ColorTime.Sprint(estimated)
				}

				// Format exercise name with status
				nameWithStatus := fmt.Sprintf("%s %-27s", statusIcon, exercise.Metadata.Name)

				// Format difficulty with color
				diffBadge := DifficultyBadge(exercise.Spec.Difficulty)

				// Print the formatted line
				fmt.Fprintf(cmd.OutOrStdout(), "  %s %s %6s  %s\n",
					nameWithStatus,
					diffBadge,
					estimated,
					ColorDim.Sprint(desc),
				)
			}

			// Add a summary footer
			fmt.Fprintln(cmd.OutOrStdout())
			if completedCount == len(filtered) {
				ColorSuccess.Fprintln(cmd.OutOrStdout(), "🎉 Congratulations! All exercises completed!")
			} else if completedCount > 0 {
				progressBar := ProgressBar(completedCount, len(filtered), 20)
				fmt.Fprintln(cmd.OutOrStdout(), progressBar)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&opts.track, "track", "", "Filter by track")
	cmd.Flags().StringVar(&opts.difficulty, "difficulty", "", "Filter by difficulty")
	cmd.Flags().IntVar(&opts.week, "week", 0, "Filter by week")

	return cmd
}

func filterList(entries []scenario.CatalogEntry, opts *listOptions) []scenario.CatalogEntry {
	var filtered []scenario.CatalogEntry
	for _, entry := range entries {
		exercise := entry.Exercise
		if opts.track != "" && !strings.EqualFold(exercise.Metadata.Track, opts.track) {
			continue
		}
		if opts.difficulty != "" && !strings.EqualFold(exercise.Spec.Difficulty, opts.difficulty) {
			continue
		}
		if opts.week > 0 && exercise.Metadata.Week != opts.week {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}
