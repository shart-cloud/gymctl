package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gymctl",
	Short: "Gymctl orchestrates Jerry's chaos gym exercises",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Resolve tasks directory location
		return setupTasksDirectory()
	},
}

var tasksDir string
var progressFile string

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&tasksDir, "tasks-dir", "tasks", "Tasks directory")
	rootCmd.PersistentFlags().StringVar(&progressFile, "progress-file", "", "Progress file path (default: ~/.gym/progress.yaml)")

	rootCmd.AddCommand(
		newValidateCmd(),
		newListCmd(),
		newStartCmd(),
		newStopCmd(),
		newCheckCmd(),
		newHintCmd(),
		newResetCmd(),
		newStatusCmd(),
		newCleanCmd(),
		newDescribeCmd(),
	)
}
