package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"gymctl/internal/progress"
	"gymctl/internal/scenario"
)

func newQuestionCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "question [exercise-name]",
		Aliases: []string{"q"},
		Short:   "Re-show the current exercise's question",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) == 1 {
				name = args[0]
			} else {
				current, err := loadCurrentExercise()
				if err != nil {
					return fmt.Errorf("no exercise specified and no current exercise set (run gymctl start <exercise>)")
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
			printQuestion(cmd, entry.Exercise)
			return nil
		},
	}
}

func printQuestion(cmd *cobra.Command, exercise *scenario.Exercise) {
	out := cmd.OutOrStdout()
	title := exercise.Metadata.DisplayTitle()
	ColorHeader.Fprintln(out, title)

	meta := exercise.Metadata.Name
	if exercise.Metadata.GXObjective != "" {
		meta += "  ·  objective " + exercise.Metadata.GXObjective
	}
	if exercise.Spec.EstimatedTime != "" {
		meta += "  ·  " + exercise.Spec.EstimatedTime
	}
	if elapsed, hints := attemptStatus(exercise.Metadata.Name); elapsed != "" {
		meta += fmt.Sprintf("  ·  %s elapsed, %d hint(s) used", elapsed, hints)
	}
	ColorDim.Fprintln(out, meta)
	fmt.Fprintln(out)
	fmt.Fprintln(out, exercise.Spec.Description)

	env := localWorkspaceEnv(exercise)
	if workDir := env["GYM_WORKDIR"]; workDir != "" {
		ColorDim.Fprintf(out, "Workspace: %s\n", workDir)
		if _, err := os.Stat(filepath.Join(workDir, workspaceQuestionFile)); err == nil {
			ColorDim.Fprintf(out, "Question:  %s\n", filepath.Join(workDir, workspaceQuestionFile))
		}
	}
	if pcap := env["PCAP"]; pcap != "" {
		ColorDim.Fprintf(out, "$PCAP:     %s\n", pcap)
	}
}

// attemptStatus reports elapsed time and hints for an in-progress attempt.
func attemptStatus(name string) (string, int) {
	path, err := resolveProgressFile()
	if err != nil {
		return "", 0
	}
	file, err := progress.Load(path)
	if err != nil {
		return "", 0
	}
	entry, ok := file.Exercises[name]
	if !ok || entry.Status != "in_progress" || entry.StartedAt == "" {
		return "", 0
	}
	started, err := time.Parse(time.RFC3339, entry.StartedAt)
	if err != nil {
		return "", 0
	}
	return formatDuration(time.Since(started).Round(time.Second)), entry.HintsUsed
}
