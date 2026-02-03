package cli

import (
	"fmt"
	"sort"
	"strings"

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
				ColorWarning.Fprintln(cmd.OutOrStdout(), "No exercises found.")
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

			// Calculate statistics
			var totalPoints int
			var earnedPoints int
			var completedCount int
			var inProgressCount int
			trackStats := make(map[string]struct {
				total     int
				completed int
			})

			for _, entry := range entries {
				exercise := entry.Exercise
				track := exercise.Metadata.Track
				stats := trackStats[track]
				stats.total++

				totalPoints += defaultPoints(exercise.Spec.Points)
				status := progressFile.Exercises[exercise.Metadata.Name]
				if status.Status == "completed" {
					completedCount++
					stats.completed++
					if status.Score > 0 {
						earnedPoints += status.Score
					}
				} else if status.Status == "in_progress" {
					inProgressCount++
				}
				trackStats[track] = stats
			}

			// Print header
			ColorHeader.Fprintln(cmd.OutOrStdout(), "📊 Progress Overview")
			fmt.Fprintln(cmd.OutOrStdout())

			// Print exercises by track
			currentTrack := ""
			for _, entry := range entries {
				exercise := entry.Exercise
				if exercise.Metadata.Track != currentTrack {
					currentTrack = exercise.Metadata.Track
					stats := trackStats[currentTrack]
					fmt.Fprintln(cmd.OutOrStdout())
					ColorTrack.Fprintf(cmd.OutOrStdout(), "▸ %s ", strings.ToUpper(currentTrack))
					ColorDim.Fprintf(cmd.OutOrStdout(), "(%d/%d completed)\n", stats.completed, stats.total)
				}

				status := progressFile.Exercises[exercise.Metadata.Name]

				// Status icon and color
				statusIcon := FormatStatus(status.Status)

				// Exercise name
				nameStr := fmt.Sprintf("%s %-27s", statusIcon, exercise.Metadata.Name)

				// Status label with color
				var statusLabel string
				switch status.Status {
				case "completed":
					statusLabel = ColorSuccess.Sprintf("%-12s", "completed")
				case "in_progress":
					statusLabel = ColorWarning.Sprintf("%-12s", "in progress")
				default:
					statusLabel = ColorDim.Sprintf("%-12s", "not started")
				}

				// Points display with color
				var pointsStr string
				if status.Status == "completed" && status.Score > 0 {
					if status.Score == 100 {
						pointsStr = ColorWarning.Sprintf("⭐ %3d pts", status.Score)
					} else {
						pointsStr = ColorProgress.Sprintf("   %3d pts", status.Score)
					}
				} else {
					pointsStr = ColorDim.Sprint("      -   ")
				}

				// Time display
				timeStr := ColorDim.Sprint("-")
				if status.TimeSpent != "" {
					timeStr = ColorTime.Sprint(status.TimeSpent)
				}

				fmt.Fprintf(cmd.OutOrStdout(), "  %s %s %s %s\n",
					nameStr,
					statusLabel,
					pointsStr,
					timeStr,
				)
			}

			// Print summary section
			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintln(cmd.OutOrStdout(), strings.Repeat("─", 60))

			// Progress bar
			progressBar := ProgressBar(completedCount, len(entries), 30)
			ColorBold.Fprint(cmd.OutOrStdout(), "Completion: ")
			fmt.Fprintln(cmd.OutOrStdout(), progressBar)

			// Points summary
			ColorBold.Fprint(cmd.OutOrStdout(), "Points:     ")
			if earnedPoints > 0 {
				ColorSuccess.Fprintf(cmd.OutOrStdout(), "%d", earnedPoints)
			} else {
				fmt.Fprint(cmd.OutOrStdout(), "0")
			}
			fmt.Fprintf(cmd.OutOrStdout(), " / %d", totalPoints)

			percentage := 0
			if totalPoints > 0 {
				percentage = (earnedPoints * 100) / totalPoints
			}
			ColorDim.Fprintf(cmd.OutOrStdout(), " (%d%%)\n", percentage)

			// Status summary
			ColorBold.Fprint(cmd.OutOrStdout(), "Status:     ")
			if completedCount > 0 {
				ColorSuccess.Fprintf(cmd.OutOrStdout(), "%d completed", completedCount)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "0 completed")
			}
			if inProgressCount > 0 {
				fmt.Fprint(cmd.OutOrStdout(), ", ")
				ColorWarning.Fprintf(cmd.OutOrStdout(), "%d in progress", inProgressCount)
			}
			notStarted := len(entries) - completedCount - inProgressCount
			if notStarted > 0 {
				fmt.Fprint(cmd.OutOrStdout(), ", ")
				ColorDim.Fprintf(cmd.OutOrStdout(), "%d not started", notStarted)
			}
			fmt.Fprintln(cmd.OutOrStdout())

			// Achievement messages
			if completedCount == len(entries) {
				fmt.Fprintln(cmd.OutOrStdout())
				ColorSuccess.Fprintln(cmd.OutOrStdout(), "🏆 Congratulations! You've completed all exercises!")
			} else if completedCount >= len(entries)*3/4 {
				fmt.Fprintln(cmd.OutOrStdout())
				ColorInfo.Fprintln(cmd.OutOrStdout(), "🎯 Great progress! You're 75% complete!")
			} else if completedCount == len(entries)/2 {
				fmt.Fprintln(cmd.OutOrStdout())
				ColorInfo.Fprintln(cmd.OutOrStdout(), "🌟 Halfway there! Keep going!")
			}

			return nil
		},
	}

	return cmd
}
