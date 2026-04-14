package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	internalexam "gymctl/internal/exam"
	"gymctl/internal/progress"
	"gymctl/internal/scenario"
)

type objectivesOptions struct {
	output         string
	objectivesFile string
}

type objectivesFile struct {
	Name       string                 `yaml:"name"`
	Domains    []objectivesFileDomain `yaml:"domains"`
	Objectives []objectiveDefinition  `yaml:"objectives"`
}

type objectivesFileDomain struct {
	ID         string                `yaml:"id"`
	Name       string                `yaml:"name"`
	Objectives []objectiveDefinition `yaml:"objectives"`
}

type objectiveDefinition struct {
	ID    string `yaml:"id"`
	Title string `yaml:"title"`
	Text  string `yaml:"text"`
}

type objectiveTaskStatus struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Score     int    `json:"score,omitempty"`
	TimeSpent string `json:"timeSpent,omitempty"`
	HintsUsed int    `json:"hintsUsed,omitempty"`
}

type objectiveSummary struct {
	ID     string                `json:"id"`
	Title  string                `json:"title"`
	Domain string                `json:"domain"`
	Status string                `json:"status"`
	Tasks  []objectiveTaskStatus `json:"tasks"`
}

func newObjectivesCmd() *cobra.Command {
	opts := &objectivesOptions{}
	cmd := &cobra.Command{
		Use:   "objectives <track>",
		Short: "Show objective progress for a track",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			track := strings.ToLower(strings.TrimSpace(args[0]))
			entries, err := loadCatalogEntries()
			if err != nil {
				return err
			}

			progressPath, err := resolveProgressFile()
			if err != nil {
				return err
			}
			progressFile, err := progress.Load(progressPath)
			if err != nil {
				return err
			}

			defs, examName, err := loadObjectiveDefinitions(track, opts.objectivesFile, entries)
			if err != nil {
				return err
			}

			grouped := make(map[string][]scenario.CatalogEntry)
			for _, entry := range entries {
				ex := entry.Exercise
				if ex == nil || !strings.EqualFold(ex.Metadata.Track, track) || strings.TrimSpace(ex.Metadata.GXObjective) == "" {
					continue
				}
				grouped[ex.Metadata.GXObjective] = append(grouped[ex.Metadata.GXObjective], entry)
			}

			summaries := make([]objectiveSummary, 0, len(defs))
			mastered := 0
			practiced := 0
			for _, def := range defs {
				summary := summarizeObjective(def, grouped[def.ID], progressFile)
				summaries = append(summaries, summary)
				switch summary.Status {
				case "mastered":
					mastered++
				case "practiced":
					practiced++
				}
			}

			total := len(summaries)
			notStarted := total - mastered - practiced

			switch strings.ToLower(strings.TrimSpace(opts.output)) {
			case "json":
				return writeJSON(cmd.OutOrStdout(), struct {
					Track      string             `json:"track"`
					Name       string             `json:"name"`
					Mastered   int                `json:"mastered"`
					Practiced  int                `json:"practiced"`
					NotStarted int                `json:"notStarted"`
					Total      int                `json:"total"`
					Objectives []objectiveSummary `json:"objectives"`
				}{Track: track, Name: examName, Mastered: mastered, Practiced: practiced, NotStarted: notStarted, Total: total, Objectives: summaries})
			case "markdown":
				renderObjectivesMarkdown(cmd.OutOrStdout(), track, examName, mastered, practiced, notStarted, total, summaries)
				return nil
			default:
				renderObjectivesTable(cmd.OutOrStdout(), track, examName, mastered, practiced, notStarted, total, summaries)
				return nil
			}
		},
	}

	cmd.Flags().StringVar(&opts.output, "output", "table", "Output format: table|json|markdown")
	cmd.Flags().StringVar(&opts.objectivesFile, "objectives-file", "", "Path to objectives YAML file")
	return cmd
}

func loadObjectiveDefinitions(track, path string, entries []scenario.CatalogEntry) ([]objectiveSummary, string, error) {
	examName := strings.ToUpper(track)
	if def, ok := internalexam.BuiltIn(track); ok {
		examName = def.Name
	}

	if strings.TrimSpace(path) == "" {
		path = defaultObjectivesFile(track)
	}

	if path != "" {
		if data, err := os.ReadFile(path); err == nil {
			var file objectivesFile
			if err := yaml.Unmarshal(data, &file); err != nil {
				return nil, "", fmt.Errorf("parse objectives file: %w", err)
			}
			if strings.TrimSpace(file.Name) != "" {
				examName = file.Name
			}
			defs := flattenObjectiveDefinitions(file, track)
			if len(defs) > 0 {
				return defs, examName, nil
			}
		}
	}

	defs := deriveObjectiveDefinitions(track, entries)
	if len(defs) == 0 {
		return nil, "", fmt.Errorf("no objectives found for track %q", track)
	}
	return defs, examName, nil
}

func defaultObjectivesFile(track string) string {
	baseDir := primaryTasksDir()
	if baseDir == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(baseDir), "exams", track, "objectives.yaml")
}

func flattenObjectiveDefinitions(file objectivesFile, track string) []objectiveSummary {
	defs := make([]objectiveSummary, 0)
	for _, objective := range file.Objectives {
		defs = append(defs, objectiveSummary{ID: objective.ID, Title: objectiveTitle(objective), Domain: domainNameForObjective(objective.ID, track)})
	}
	for _, domain := range file.Domains {
		for _, objective := range domain.Objectives {
			defs = append(defs, objectiveSummary{ID: objective.ID, Title: objectiveTitle(objective), Domain: domainNameForObjectiveWithFallback(objective.ID, domain.Name, track)})
		}
	}
	sort.Slice(defs, func(i, j int) bool { return defs[i].ID < defs[j].ID })
	return defs
}

func deriveObjectiveDefinitions(track string, entries []scenario.CatalogEntry) []objectiveSummary {
	seen := map[string]objectiveSummary{}
	for _, entry := range entries {
		ex := entry.Exercise
		if ex == nil || !strings.EqualFold(ex.Metadata.Track, track) || strings.TrimSpace(ex.Metadata.GXObjective) == "" {
			continue
		}
		if _, exists := seen[ex.Metadata.GXObjective]; exists {
			continue
		}
		seen[ex.Metadata.GXObjective] = objectiveSummary{
			ID:     ex.Metadata.GXObjective,
			Title:  ex.Metadata.Title,
			Domain: domainNameForObjective(ex.Metadata.GXObjective, track),
		}
	}
	defs := make([]objectiveSummary, 0, len(seen))
	for _, def := range seen {
		defs = append(defs, def)
	}
	sort.Slice(defs, func(i, j int) bool { return defs[i].ID < defs[j].ID })
	return defs
}

func summarizeObjective(def objectiveSummary, entries []scenario.CatalogEntry, progressFile *progress.File) objectiveSummary {
	summary := def
	summary.Status = "not-started"
	summary.Tasks = make([]objectiveTaskStatus, 0, len(entries))
	mastered := false
	practiced := false
	for _, entry := range entries {
		status := progressFile.Exercises[entry.Exercise.Metadata.Name]
		task := objectiveTaskStatus{
			Name:      entry.Exercise.Metadata.Name,
			Status:    jsonStatus(status.Status),
			Score:     status.Score,
			TimeSpent: status.TimeSpent,
			HintsUsed: status.HintsUsed,
		}
		summary.Tasks = append(summary.Tasks, task)
		if isMasteredObjectiveAttempt(entry.Exercise, status) {
			mastered = true
		}
		if isInProgressStatus(status.Status) || strings.EqualFold(status.Status, "completed") {
			practiced = true
		}
	}
	sort.Slice(summary.Tasks, func(i, j int) bool { return summary.Tasks[i].Name < summary.Tasks[j].Name })
	switch {
	case mastered:
		summary.Status = "mastered"
	case practiced:
		summary.Status = "practiced"
	}
	return summary
}

func isMasteredObjectiveAttempt(exercise *scenario.Exercise, status progress.ExerciseStatus) bool {
	if !strings.EqualFold(status.Status, "completed") || status.Score < 90 {
		return false
	}
	completedAt, err := time.Parse(time.RFC3339, status.CompletedAt)
	if err != nil {
		return false
	}
	startedAt, err := time.Parse(time.RFC3339, status.StartedAt)
	if err != nil {
		return false
	}
	return int(completedAt.Sub(startedAt).Minutes()) <= estimatedMinutes(exercise.Spec.EstimatedTime)
}

func renderObjectivesTable(out io.Writer, track, examName string, mastered, practiced, notStarted, total int, summaries []objectiveSummary) {
	fmt.Fprintf(out, "%s - %s\n", strings.ToUpper(track), examName)
	fmt.Fprintf(out, "%d mastered / %d practiced / %d not started / %d total\n\n", mastered, practiced, notStarted, total)
	currentDomain := ""
	for _, summary := range summaries {
		if summary.Domain != currentDomain {
			currentDomain = summary.Domain
			fmt.Fprintf(out, "%s\n", currentDomain)
		}
		fmt.Fprintf(out, "%s %s %s\n", objectiveStatusIcon(summary.Status), summary.ID, summary.Title)
		if len(summary.Tasks) == 0 {
			fmt.Fprintln(out, "  Tasks: none yet")
			continue
		}
		parts := make([]string, 0, len(summary.Tasks))
		for _, task := range summary.Tasks {
			parts = append(parts, fmt.Sprintf("%s (%s)", task.Name, task.Status))
		}
		fmt.Fprintf(out, "  Tasks: %s\n", strings.Join(parts, ", "))
	}
}

func renderObjectivesMarkdown(out io.Writer, track, examName string, mastered, practiced, notStarted, total int, summaries []objectiveSummary) {
	fmt.Fprintf(out, "# %s\n\n", examName)
	fmt.Fprintf(out, "%d mastered / %d practiced / %d not started / %d total\n\n", mastered, practiced, notStarted, total)
	currentDomain := ""
	for _, summary := range summaries {
		if summary.Domain != currentDomain {
			currentDomain = summary.Domain
			fmt.Fprintf(out, "## %s\n\n", currentDomain)
		}
		fmt.Fprintf(out, "- %s %s %s\n", objectiveStatusIcon(summary.Status), summary.ID, summary.Title)
	}
}

func objectiveStatusIcon(status string) string {
	switch status {
	case "mastered":
		return "✓"
	case "practiced":
		return "◌"
	default:
		return "-"
	}
}

func objectiveTitle(objective objectiveDefinition) string {
	if strings.TrimSpace(objective.Title) != "" {
		return objective.Title
	}
	return objective.Text
}

func domainNameForObjective(objectiveID, track string) string {
	parts := strings.Split(strings.TrimSpace(objectiveID), ".")
	if len(parts) == 0 {
		return "Objectives"
	}
	if def, ok := internalexam.BuiltIn(track); ok {
		return def.DomainNameByID(parts[0])
	}
	return "Objectives"
}

func domainNameForObjectiveWithFallback(objectiveID, fallback, track string) string {
	name := domainNameForObjective(objectiveID, track)
	if name == "Objectives" && strings.TrimSpace(fallback) != "" {
		return fallback
	}
	return name
}
