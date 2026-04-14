package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"gymctl/internal/scenario"
)

type describeOptions struct {
	spec string
}

func newDescribeCmd() *cobra.Command {
	opts := &describeOptions{}
	cmd := &cobra.Command{
		Use:   "describe [exercise-name]",
		Short: "Describe an exercise",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.spec != "" && len(args) > 0 {
				return fmt.Errorf("cannot use exercise name argument with --spec")
			}

			exercise, exerciseDir, err := resolveDescribeTarget(args, opts.spec)
			if err != nil {
				return err
			}

			if isJSONOutput() {
				return writeJSON(cmd.OutOrStdout(), buildDescribeResponse(exercise, exerciseDir))
			}

			return writeDescribeTable(cmd, exercise)
		},
	}

	cmd.Flags().StringVar(&opts.spec, "spec", "", "Path to an exercise spec file")
	return cmd
}

func resolveDescribeTarget(args []string, spec string) (*scenario.Exercise, string, error) {
	if spec != "" {
		exercise, err := scenario.LoadExerciseFile(spec)
		if err != nil {
			return nil, "", err
		}
		return exercise, filepath.Dir(spec), nil
	}

	name := ""
	if len(args) == 1 {
		name = args[0]
	} else {
		current, err := loadCurrentExercise()
		if err != nil {
			return nil, "", fmt.Errorf("no exercise specified and no current exercise set")
		}
		name = current
	}

	entries, err := loadCatalogEntries()
	if err != nil {
		return nil, "", err
	}
	entry, found := scenario.FindByName(entries, name)
	if !found {
		return nil, "", fmt.Errorf("exercise not found: %s", name)
	}
	return entry.Exercise, entry.Dir, nil
}

func buildDescribeResponse(exercise *scenario.Exercise, exerciseDir string) any {
	type checkJSON struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	type referenceJSON struct {
		Title string `json:"title"`
		URL   string `json:"url"`
	}
	type describeResponseJSON struct {
		Name             string                `json:"name"`
		Title            string                `json:"title"`
		Track            string                `json:"track"`
		Week             int                   `json:"week,omitempty"`
		Order            int                   `json:"order,omitempty"`
		Difficulty       string                `json:"difficulty"`
		EstimatedTime    string                `json:"estimatedTime,omitempty"`
		Points           int                   `json:"points"`
		Description      string                `json:"description,omitempty"`
		LearningOutcomes []string              `json:"learningOutcomes,omitempty"`
		Tags             []string              `json:"tags,omitempty"`
		Prerequisites    []string              `json:"prerequisites,omitempty"`
		Checks           []checkJSON           `json:"checks,omitempty"`
		References       []referenceJSON       `json:"references,omitempty"`
		LabPath          string                `json:"labPath,omitempty"`
		LabSections      []scenario.LabSection `json:"labSections,omitempty"`
	}

	response := describeResponseJSON{
		Name:             exercise.Metadata.Name,
		Title:            exercise.Metadata.Title,
		Track:            exercise.Metadata.Track,
		Week:             exercise.Metadata.Week,
		Order:            exercise.Metadata.Order,
		Difficulty:       exercise.Spec.Difficulty,
		EstimatedTime:    exercise.Spec.EstimatedTime,
		Points:           defaultPoints(exercise.Spec.Points),
		Description:      exercise.Spec.Description,
		LearningOutcomes: exercise.Spec.LearningOutcomes,
		Tags:             exercise.Spec.Tags,
		Prerequisites:    exercise.Spec.Prerequisites,
	}

	if len(exercise.Spec.Checks) > 0 {
		response.Checks = make([]checkJSON, 0, len(exercise.Spec.Checks))
		for _, check := range exercise.Spec.Checks {
			label := check.Name
			if label == "" {
				label = check.Type
			}
			response.Checks = append(response.Checks, checkJSON{Name: label, Type: check.Type})
		}
	}

	if len(exercise.Spec.References) > 0 {
		response.References = make([]referenceJSON, 0, len(exercise.Spec.References))
		for _, ref := range exercise.Spec.References {
			response.References = append(response.References, referenceJSON{Title: ref.Title, URL: ref.URL})
		}
	}

	labPath := filepath.Join(exerciseDir, "lab.md")
	if data, err := os.ReadFile(labPath); err == nil {
		response.LabPath = labPath
		if sections, err := scenario.ParseLabSections(data); err == nil {
			response.LabSections = sections
		}
	}

	return response
}

func writeDescribeTable(cmd *cobra.Command, exercise *scenario.Exercise) error {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "%s\n", exercise.Metadata.Title)
	fmt.Fprintf(out, "%s\n", strings.Repeat("-", len(exercise.Metadata.Title)))
	fmt.Fprintf(out, "Name: %s\n", exercise.Metadata.Name)
	fmt.Fprintf(out, "Track: %s\n", exercise.Metadata.Track)
	if exercise.Metadata.Week > 0 {
		fmt.Fprintf(out, "Week: %d\n", exercise.Metadata.Week)
	}
	fmt.Fprintf(out, "Difficulty: %s\n", exercise.Spec.Difficulty)
	if exercise.Spec.EstimatedTime != "" {
		fmt.Fprintf(out, "Estimated time: %s\n", exercise.Spec.EstimatedTime)
	}
	fmt.Fprintf(out, "Points: %d\n", defaultPoints(exercise.Spec.Points))
	fmt.Fprintln(out, "")

	if exercise.Spec.Description != "" {
		fmt.Fprintln(out, strings.TrimSpace(exercise.Spec.Description))
		fmt.Fprintln(out, "")
	}

	if len(exercise.Spec.LearningOutcomes) > 0 {
		fmt.Fprintln(out, "Learning outcomes:")
		for _, item := range exercise.Spec.LearningOutcomes {
			fmt.Fprintf(out, "- %s\n", item)
		}
		fmt.Fprintln(out, "")
	}

	if len(exercise.Spec.Tags) > 0 {
		fmt.Fprintf(out, "Tags: %s\n", strings.Join(exercise.Spec.Tags, ", "))
	}
	if len(exercise.Spec.Prerequisites) > 0 {
		fmt.Fprintf(out, "Prerequisites: %s\n", strings.Join(exercise.Spec.Prerequisites, ", "))
	}
	if len(exercise.Spec.Tags) > 0 || len(exercise.Spec.Prerequisites) > 0 {
		fmt.Fprintln(out, "")
	}

	if len(exercise.Spec.Checks) > 0 {
		fmt.Fprintln(out, "Checks:")
		for _, check := range exercise.Spec.Checks {
			label := check.Name
			if label == "" {
				label = check.Type
			}
			fmt.Fprintf(out, "- %s (%s)\n", label, check.Type)
		}
		fmt.Fprintln(out, "")
	}

	if len(exercise.Spec.References) > 0 {
		fmt.Fprintln(out, "References:")
		for _, ref := range exercise.Spec.References {
			fmt.Fprintf(out, "- %s: %s\n", ref.Title, ref.URL)
		}
	}

	return nil
}
