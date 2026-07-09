package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"gymctl/internal/scenario"
)

func newBackupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup [exercise-name]",
		Short: "Create a backup of an exercise's work directory and progress",
		Long: `Backup creates a tar.gz archive of the exercise's work directory
and progress metadata, stored in ~/.gym/backups/. Restore currently restores
the work directory contents from the archive.

Restore a backup with: gymctl recover <exercise-name> --backup <path>`,
		Args: cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			defer RecoverFromPanic(cmd)

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
			exercise := entry.Exercise

			spinner := NewSpinnerManager()
			spinner.Start(fmt.Sprintf("Backing up %s", exercise.Metadata.Name))

			backupPath, err := createBackup(exercise.Metadata.Name)
			if err != nil {
				spinner.Fail("Backup failed")
				return err
			}
			spinner.Success("Backup created")

			fmt.Fprintln(cmd.OutOrStdout())
			ColorSuccess.Fprintf(cmd.OutOrStdout(), "✓ Backup saved: %s\n", backupPath)
			ColorInfo.Fprintln(cmd.OutOrStdout(), "Restore with: gymctl recover "+exercise.Metadata.Name+" --backup "+backupPath)

			return nil
		},
	}

	return cmd
}
