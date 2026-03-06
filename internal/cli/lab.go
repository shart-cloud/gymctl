package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"gymctl/internal/cloudlab"
)

func newLabCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lab",
		Short: "Commands for running inside a shart.cloud container lab",
		Long: `Commands intended for use inside the shart.cloud browser-based
container lab environment. They read SCENARIO_PATH, SESSION_ID, USER_ID,
LAB_ID, and COMPLETION_WEBHOOK_SECRET from environment variables, all of
which are injected automatically by the container entrypoint.`,
		// lab commands don't need the tasks directory
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
	}

	cmd.AddCommand(newLabPromptCmd(), newLabCheckCmd())
	return cmd
}

func newLabPromptCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "prompt",
		Short: "Print the lab exercise prompts",
		Long:  "Loads the current scenario and prints each check prompt, numbered.",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := os.Getenv("SCENARIO_PATH")
			if path == "" {
				return fmt.Errorf("SCENARIO_PATH env var is not set")
			}
			return cloudlab.RunPrompt(os.Stdout, path)
		},
	}
}

func newLabCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Evaluate scenario checks interactively",
		Long: `Walks through each check in the current scenario:
  - exact_match: prompts the student for an answer (case-insensitive)
  - kubectl_output: runs a kubectl command and verifies stdout

When all checks pass the completion is POSTed to the lab webhook.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			path := os.Getenv("SCENARIO_PATH")
			if path == "" {
				return fmt.Errorf("SCENARIO_PATH env var is not set")
			}
			return cloudlab.RunCheck(path)
		},
	}
}
