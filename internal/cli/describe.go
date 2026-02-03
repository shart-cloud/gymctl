package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"gymctl/internal/scenario"
)

func newDescribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe <exercise-name>",
		Short: "Describe an exercise",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := scenario.LoadCatalog(tasksDir)
			if err != nil {
				return err
			}
			entry, found := scenario.FindByName(entries, args[0])
			if !found {
				return fmt.Errorf("exercise not found: %s", args[0])
			}

			exercise := entry.Exercise
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "%s\n", exercise.Metadata.Title)
			fmt.Fprintf(out, "%s\n", strings.Repeat("-", len(exercise.Metadata.Title)))
			fmt.Fprintf(out, "Name: %s\n", exercise.Metadata.Name)
			fmt.Fprintf(out, "Track: %s\n", exercise.Metadata.Track)
			if exercise.Metadata.Week > 0 {
				fmt.Fprintf(out, "Week: %d\n", exercise.Metadata.Week)
			}
			fmt.Fprintf(out, "Difficulty: %s\n", exercise.Spec.Difficulty)
			if exercise.Spec.EstimatedTime != "" {
				fmt.Fprintf(out, "Estimated time: %s\n", exercise.Spec.EstimatedTime)
			}
			fmt.Fprintf(out, "Points: %d\n", defaultPoints(exercise.Spec.Points))
			fmt.Fprintln(out, "")

			if exercise.Spec.Description != "" {
				fmt.Fprintln(out, strings.TrimSpace(exercise.Spec.Description))
				fmt.Fprintln(out, "")
			}

			if len(exercise.Spec.LearningOutcomes) > 0 {
				fmt.Fprintln(out, "Learning outcomes:")
				for _, item := range exercise.Spec.LearningOutcomes {
					fmt.Fprintf(out, "- %s\n", item)
				}
				fmt.Fprintln(out, "")
			}

			if len(exercise.Spec.Tags) > 0 {
				fmt.Fprintf(out, "Tags: %s\n", strings.Join(exercise.Spec.Tags, ", "))
			}
			if len(exercise.Spec.Prerequisites) > 0 {
				fmt.Fprintf(out, "Prerequisites: %s\n", strings.Join(exercise.Spec.Prerequisites, ", "))
			}
			if len(exercise.Spec.Tags) > 0 || len(exercise.Spec.Prerequisites) > 0 {
				fmt.Fprintln(out, "")
			}

			if len(exercise.Spec.Checks) > 0 {
				fmt.Fprintln(out, "Checks:")
				for _, check := range exercise.Spec.Checks {
					label := check.Name
					if label == "" {
						label = check.Type
					}
					fmt.Fprintf(out, "- %s (%s)\n", label, check.Type)
				}
				fmt.Fprintln(out, "")
			}

			if len(exercise.Spec.References) > 0 {
				fmt.Fprintln(out, "References:")
				for _, ref := range exercise.Spec.References {
					fmt.Fprintf(out, "- %s: %s\n", ref.Title, ref.URL)
				}
			}

			return nil
		},
	}

	return cmd
}
