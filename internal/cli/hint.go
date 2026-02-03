package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"gymctl/internal/progress"
	"gymctl/internal/scenario"
)

type hintOptions struct {
	revealAll bool
}

func newHintCmd() *cobra.Command {
	opts := &hintOptions{}
	cmd := &cobra.Command{
		Use:   "hint [exercise-name]",
		Short: "Show the next hint",
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

			entries, err := scenario.LoadCatalog(tasksDir)
			if err != nil {
				return err
			}
			entry, found := scenario.FindByName(entries, name)
			if !found {
				return fmt.Errorf("exercise not found: %s", name)
			}

			progressPath, err := resolveProgressFile()
			if err != nil {
				return err
			}
			progressFile, err := progress.Load(progressPath)
			if err != nil {
				return err
			}

			status := progressFile.Exercises[entry.Exercise.Metadata.Name]
			startIndex := status.HintsUsed
			if startIndex >= len(entry.Exercise.Spec.Hints) {
				ColorWarning.Fprintln(cmd.OutOrStdout(), "No more hints available.")
				return nil
			}

			endIndex := startIndex + 1
			if opts.revealAll {
				endIndex = len(entry.Exercise.Spec.Hints)
			}

			for i := startIndex; i < endIndex; i++ {
				hint := entry.Exercise.Spec.Hints[i]
				content, err := loadHintContent(entry.Dir, hint)
				if err != nil {
					return err
				}
				ColorInfo.Fprintf(cmd.OutOrStdout(), "%s Hint %d: ", IconHint, i+1)
				fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(content))
				fmt.Fprintln(cmd.OutOrStdout(), "")
			}

			status.HintsUsed = endIndex
			progressFile.Exercises[entry.Exercise.Metadata.Name] = status
			return progress.Save(progressPath, progressFile)
		},
	}

	cmd.Flags().BoolVar(&opts.revealAll, "reveal-all", false, "Show all remaining hints")
	return cmd
}

func loadHintContent(baseDir string, hint scenario.Hint) (string, error) {
	if hint.Content != "" {
		return hint.Content, nil
	}
	if hint.File == "" {
		return "", fmt.Errorf("hint has no content or file")
	}
	path := hint.File
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read hint file: %w", err)
	}
	return string(data), nil
}
