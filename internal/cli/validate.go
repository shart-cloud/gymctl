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

			fmt.Fprintf(cmd.OutOrStdout(), "OK: %s (%s)\n", exercise.Metadata.Name, exercise.Metadata.Title)
			return nil
		},
	}

	return cmd
}
