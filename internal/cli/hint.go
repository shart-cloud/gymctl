package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"gymctl/internal/pet"
	"gymctl/internal/progress"
	"gymctl/internal/scenario"
)

type hintOptions struct {
	revealAll bool
	spec      string
}

func newHintCmd() *cobra.Command {
	opts := &hintOptions{}
	cmd := &cobra.Command{
		Use:   "hint [exercise-name]",
		Short: "Show the next hint",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			type hintResponseJSON struct {
				Exercise       string  `json:"exercise"`
				HintIndex      *int    `json:"hintIndex"`
				Content        *string `json:"content"`
				HintsUsed      int     `json:"hintsUsed"`
				HintsTotal     int     `json:"hintsTotal"`
				HintsRemaining int     `json:"hintsRemaining"`
				NextHintCost   int     `json:"nextHintCost"`
			}

			if opts.spec != "" && len(args) > 0 {
				return fmt.Errorf("cannot use exercise name argument with --spec")
			}

			var exercise *scenario.Exercise
			baseDir := ""
			if opts.spec != "" {
				loaded, err := scenario.LoadExerciseFile(opts.spec)
				if err != nil {
					return err
				}
				exercise = loaded
				baseDir = filepath.Dir(opts.spec)
			} else {
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
				exercise = entry.Exercise
				baseDir = entry.Dir
			}

			progressPath, err := resolveProgressFile()
			if err != nil {
				return err
			}
			progressFile, err := progress.Load(progressPath)
			if err != nil {
				return err
			}

			status := progressFile.Exercises[exercise.Metadata.Name]
			startIndex := status.HintsUsed
			if startIndex >= len(exercise.Spec.Hints) {
				if isJSONOutput() {
					response := hintResponseJSON{
						Exercise:       exercise.Metadata.Name,
						HintIndex:      nil,
						Content:        nil,
						HintsUsed:      status.HintsUsed,
						HintsTotal:     len(exercise.Spec.Hints),
						HintsRemaining: 0,
						NextHintCost:   0,
					}
					return writeJSON(cmd.OutOrStdout(), response)
				}
				ColorWarning.Fprintln(cmd.OutOrStdout(), "No more hints available.")
				return nil
			}

			endIndex := startIndex + 1
			if opts.revealAll {
				endIndex = len(exercise.Spec.Hints)
			}

			var shownHint *scenario.Hint
			shownHintIndex := 0
			for i := startIndex; i < endIndex; i++ {
				hint := exercise.Spec.Hints[i]
				content, err := loadHintContent(baseDir, hint)
				if err != nil {
					return err
				}
				shownHint = &hint
				shownHintIndex = i + 1

				if isJSONOutput() {
					continue
				}
				ColorInfo.Fprintf(cmd.OutOrStdout(), "%s Hint %d: ", IconHint, i+1)
				fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(content))
				fmt.Fprintln(cmd.OutOrStdout(), "")
			}

			status.HintsUsed = endIndex
			progressFile.Exercises[exercise.Metadata.Name] = status
			if err := progress.Save(progressPath, progressFile); err != nil {
				return err
			}

			remaining := len(exercise.Spec.Hints) - status.HintsUsed
			nextHintCost := 0
			if remaining > 0 {
				nextHintCost = exercise.Spec.Hints[status.HintsUsed].Cost
			}

			if isJSONOutput() {
				var contentValue *string
				if shownHint != nil {
					content, err := loadHintContent(baseDir, *shownHint)
					if err != nil {
						return err
					}
					trimmed := strings.TrimSpace(content)
					contentValue = &trimmed
				}

				hintIndex := shownHintIndex
				response := hintResponseJSON{
					Exercise:       exercise.Metadata.Name,
					HintIndex:      &hintIndex,
					Content:        contentValue,
					HintsUsed:      status.HintsUsed,
					HintsTotal:     len(exercise.Spec.Hints),
					HintsRemaining: remaining,
					NextHintCost:   nextHintCost,
				}
				return writeJSON(cmd.OutOrStdout(), response)
			}

			pet.RenderCLI(cmd.OutOrStdout(), exercise.Spec.JerryDialog, "onHintUsed", exercise.Metadata.Name)
			fmt.Fprintln(cmd.OutOrStdout())
			if remaining == 0 {
				ColorDim.Fprintln(cmd.OutOrStdout(), "  No more hints available.")
			} else {
				if nextHintCost == 0 {
					ColorDim.Fprintf(cmd.OutOrStdout(), "  %d hint(s) remaining · next hint is free\n", remaining)
				} else {
					ColorDim.Fprintf(cmd.OutOrStdout(), "  %d hint(s) remaining · next hint costs %d pts\n", remaining, nextHintCost)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.revealAll, "reveal-all", false, "Show all remaining hints")
	cmd.Flags().StringVar(&opts.spec, "spec", "", "Path to an exercise spec file")
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
