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
			entries, err := loadCatalogEntries()
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

			if isJSONOutput() {
				type listExerciseJSON struct {
					Name           string   `json:"name"`
					Title          string   `json:"title"`
					Track          string   `json:"track"`
					Week           int      `json:"week"`
					Order          int      `json:"order"`
					Difficulty     string   `json:"difficulty"`
					EstimatedTime  string   `json:"estimatedTime"`
					Points         int      `json:"points"`
					Status         string   `json:"status"`
					Score          int      `json:"score"`
					Tags           []string `json:"tags"`
					Locked         bool     `json:"locked"`
					MissingPrereqs []string `json:"missingPrereqs"`
				}
				type listSummaryJSON struct {
					Total        int `json:"total"`
					Completed    int `json:"completed"`
					InProgress   int `json:"inProgress"`
					NotStarted   int `json:"notStarted"`
					TotalPoints  int `json:"totalPoints"`
					EarnedPoints int `json:"earnedPoints"`
				}
				type listResponseJSON struct {
					Exercises []listExerciseJSON `json:"exercises"`
					Summary   listSummaryJSON    `json:"summary"`
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

				response := listResponseJSON{Exercises: make([]listExerciseJSON, 0, len(filtered))}
				for _, entry := range filtered {
					exercise := entry.Exercise
					st := progressFile.Exercises[exercise.Metadata.Name]
					locked, missingPrereqs := isExerciseLocked(exercise, progressFile)

					status := jsonStatus(st.Status)
					if locked {
						status = "not-started"
					}

					response.Exercises = append(response.Exercises, listExerciseJSON{
						Name:           exercise.Metadata.Name,
						Title:          exercise.Metadata.Title,
						Track:          exercise.Metadata.Track,
						Week:           exercise.Metadata.Week,
						Order:          exercise.Metadata.Order,
						Difficulty:     exercise.Spec.Difficulty,
						EstimatedTime:  exercise.Spec.EstimatedTime,
						Points:         defaultPoints(exercise.Spec.Points),
						Status:         status,
						Score:          st.Score,
						Tags:           exercise.Spec.Tags,
						Locked:         locked,
						MissingPrereqs: missingPrereqs,
					})

					response.Summary.Total++
					response.Summary.TotalPoints += defaultPoints(exercise.Spec.Points)
					response.Summary.EarnedPoints += st.Score
					switch status {
					case "completed":
						response.Summary.Completed++
					case "in-progress":
						response.Summary.InProgress++
					default:
						response.Summary.NotStarted++
					}
				}

				return writeJSON(cmd.OutOrStdout(), response)
			}

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

			// Count completed and in-progress exercises
			completedCount := 0
			inProgressCount := 0
			for _, entry := range filtered {
				st := progressFile.Exercises[entry.Exercise.Metadata.Name]
				switch st.Status {
				case "completed":
					completedCount++
				case "in_progress", "started":
					inProgressCount++
				}
			}

			// GRIM sprint header
			ColorTrack.Fprintf(cmd.OutOrStdout(), "  GRIM-9 · Sprint Overview")
			ColorDim.Fprintf(cmd.OutOrStdout(), "  ·  %d/%d complete  ·  %d in progress\n", completedCount, len(filtered), inProgressCount)

			currentTrack := ""
			for _, entry := range filtered {
				exercise := entry.Exercise
				if exercise.Metadata.Track != currentTrack {
					currentTrack = exercise.Metadata.Track
					fmt.Fprintln(cmd.OutOrStdout())
					ColorTrack.Fprintf(cmd.OutOrStdout(), "▸ %s\n", strings.ToUpper(currentTrack))
				}

				// Determine lock state: exercise is locked if any prereqs are not completed
				locked, missingPrereqs := isExerciseLocked(exercise, progressFile)

				// Get status icon
				var statusIcon string
				if locked {
					statusIcon = ColorDim.Sprint("◌")
				} else {
					status := progressFile.Exercises[exercise.Metadata.Name]
					statusIcon = FormatStatus(status.Status)
				}

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

				if locked {
					// Dim the whole line and append lock indicator
					needs := ColorDim.Sprintf("[needs: %s]", strings.Join(missingPrereqs, ", "))
					fmt.Fprintf(cmd.OutOrStdout(), "  %s %s %6s  %s\n",
						ColorDim.Sprint(nameWithStatus),
						ColorDim.Sprint(diffBadge),
						ColorDim.Sprint(estimated),
						needs,
					)
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "  %s %s %6s  %s\n",
						nameWithStatus,
						diffBadge,
						estimated,
						ColorDim.Sprint(desc),
					)
				}
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

// isExerciseLocked returns true if the exercise has prerequisites that are not completed.
// The second return value lists the missing prerequisite names.
func isExerciseLocked(exercise *scenario.Exercise, pf *progress.File) (bool, []string) {
	var missing []string
	for _, prereq := range exercise.Spec.Prerequisites {
		st := pf.Exercises[prereq]
		if st.Status != "completed" {
			missing = append(missing, prereq)
		}
	}
	return len(missing) > 0, missing
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
