package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"gymctl/internal/progress"
	"gymctl/internal/scenario"
)

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show overall progress",
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := scenario.LoadCatalog(tasksDir)
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No exercises found.")
				return nil
			}

			progressPath, err := resolveProgressFile()
			if err != nil {
				return err
			}
			progressFile, err := progress.Load(progressPath)
			if err != nil {
				return err
			}

			sort.Slice(entries, func(i, j int) bool {
				ai := entries[i].Exercise
				aj := entries[j].Exercise
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

			var totalPoints int
			var earnedPoints int
			var completedCount int
			for _, entry := range entries {
				totalPoints += defaultPoints(entry.Exercise.Spec.Points)
				status := progressFile.Exercises[entry.Exercise.Metadata.Name]
				if status.Status == "completed" {
					completedCount++
					if status.Score > 0 {
						earnedPoints += status.Score
					}
				}
			}

			currentTrack := ""
			for _, entry := range entries {
				exercise := entry.Exercise
				if exercise.Metadata.Track != currentTrack {
					currentTrack = exercise.Metadata.Track
					fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n", currentTrack)
				}
				status := progressFile.Exercises[exercise.Metadata.Name]
				statusLabel := "not started"
				if status.Status == "completed" {
					statusLabel = "completed"
				} else if status.Status == "in_progress" {
					statusLabel = "in progress"
				}
				points := "-"
				if status.Status == "completed" && status.Score > 0 {
					points = fmt.Sprintf("%d pts", status.Score)
				}
				timeSpent := "-"
				if status.TimeSpent != "" {
					timeSpent = status.TimeSpent
				}
				fmt.Fprintf(cmd.OutOrStdout(), "  %-28s %-12s %-8s %s\n",
					exercise.Metadata.Name,
					statusLabel,
					points,
					timeSpent,
				)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "")
			fmt.Fprintf(cmd.OutOrStdout(), "Overall: %d/%d pts | %d/%d completed\n",
				earnedPoints,
				totalPoints,
				completedCount,
				len(entries),
			)

			return nil
		},
	}

	return cmd
}
