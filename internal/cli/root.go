package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gymctl",
	Short: "Gymctl orchestrates Jerry's chaos gym exercises",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if outputJSON {
			outputFormat = "json"
		}
		outputFormat = strings.ToLower(strings.TrimSpace(outputFormat))
		if err := validateOutputFormat(); err != nil {
			return err
		}

		if cmd.Name() == "check" || cmd.Name() == "hint" {
			specPath, err := cmd.Flags().GetString("spec")
			if err == nil && strings.TrimSpace(specPath) != "" {
				return nil
			}
		}

		// Resolve tasks directory location
		return setupTasksDirectory()
	},
}

var tasksDir string
var progressFile string
var outputFormat string
var outputJSON bool

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
	rootCmd.PersistentFlags().StringVar(&tasksDir, "tasks-dir", "tasks", "Tasks directory or comma-separated list")
	rootCmd.PersistentFlags().StringVar(&progressFile, "progress-file", "", "Progress file path (default: ~/.gym/progress.yaml)")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "Output format: table|json")
	rootCmd.PersistentFlags().BoolVar(&outputJSON, "json", false, "Output in JSON format")

	rootCmd.AddCommand(
		newValidateCmd(),
		newAuthorCmd(),
		newListCmd(),
		newStartCmd(),
		newStopCmd(),
		newCheckCmd(),
		newHintCmd(),
		newResetCmd(),
		newRecoverCmd(),
		newStatusCmd(),
		newCleanCmd(),
		newCleanupCmd(),
		newDescribeCmd(),
		newDiagnoseCmd(),
		newNextCmd(),
		newWatchCmd(),
		newShellCmd(),
		newSSHCmd(),
		newEnvCmd(),
		newKubeconfigCmd(),
		newInfoCmd(),
		newObjectivesCmd(),
		newExamCmd(),
		newTaskCmd(),
		newServeCmd(),
		newLabCmd(),
	)
}
