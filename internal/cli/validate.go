package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"gymctl/internal/scenario"
)

func newValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate <task.yaml>",
		Short: "Validate an exercise definition",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			exercise, err := scenario.LoadExerciseFile(args[0])
			if err != nil {
				return err
			}

			report := validateExerciseReadiness(args[0], exercise)
			if len(report.Issues) > 0 {
				for _, issue := range report.Issues {
					fmt.Fprintf(cmd.ErrOrStderr(), "validation issue: %s\n", issue)
				}
				return fmt.Errorf("exercise readiness validation failed for %s", exercise.Metadata.Name)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "OK: %s (%s) [%s]\n", exercise.Metadata.Name, exercise.Metadata.Title, report.Status)
			return nil
		},
	}

	return cmd
}
