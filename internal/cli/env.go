package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"gymctl/internal/scenario"
)

type envOptions struct {
	format string
}

func newEnvCmd() *cobra.Command {
	opts := &envOptions{}

	cmd := &cobra.Command{
		Use:   "env [exercise-name]",
		Short: "Print environment variables for an exercise",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) == 1 {
				name = args[0]
			} else {
				current, err := loadCurrentExercise()
				if err != nil {
					return fmt.Errorf("no exercise specified and no current exercise set")
				}
				name = current
			}

			entries, err := loadCatalogEntries()
			if err != nil {
				return err
			}
			entry, found := scenario.FindByName(entries, name)
			if !found {
				return fmt.Errorf("exercise not found: %s", name)
			}

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			envVars, warnings, err := collectExerciseEnv(ctx, entry.Exercise)
			if err != nil {
				return err
			}

			for _, warning := range warnings {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", warning)
			}

			switch strings.ToLower(strings.TrimSpace(opts.format)) {
			case "shell":
				for _, key := range sortedEnvKeys(envVars) {
					fmt.Fprintf(cmd.OutOrStdout(), "export %s=%s\n", key, shellQuoteValue(envVars[key]))
				}
			case "powershell", "ps1":
				for _, key := range sortedEnvKeys(envVars) {
					fmt.Fprintf(cmd.OutOrStdout(), "$env:%s = %q\n", key, envVars[key])
				}
			case "text":
				for _, key := range sortedEnvKeys(envVars) {
					fmt.Fprintf(cmd.OutOrStdout(), "%s=%s\n", key, envVars[key])
				}
			default:
				return fmt.Errorf("unsupported format: %s (use shell, powershell, or text)", opts.format)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&opts.format, "format", "shell", "Output format: shell|powershell|text")

	return cmd
}
