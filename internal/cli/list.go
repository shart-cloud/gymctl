package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

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

			filtered := filterList(entries, opts)
			if len(filtered) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No exercises found.")
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

			fmt.Fprintln(cmd.OutOrStdout(), "Available exercises:")
			currentTrack := ""
			for _, entry := range filtered {
				exercise := entry.Exercise
				if exercise.Metadata.Track != currentTrack {
					currentTrack = exercise.Metadata.Track
					fmt.Fprintf(cmd.OutOrStdout(), "\nTRACK: %s\n", currentTrack)
				}
				desc := firstLine(exercise.Spec.Description)
				estimated := exercise.Spec.EstimatedTime
				if estimated == "" {
					estimated = "-"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "  %-28s [%-12s] %6s  - %s\n",
					exercise.Metadata.Name,
					exercise.Spec.Difficulty,
					estimated,
					desc,
				)
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
