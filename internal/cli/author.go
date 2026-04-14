package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"gymctl/internal/scenario"
)

type authorConfig struct {
	LabPaths []string `json:"lab_paths" yaml:"lab_paths"`
}

type authorLabCheckIndex struct {
	LabID string `json:"lab_id" yaml:"lab_id"`
}

type mcqAnswer struct {
	ID         string
	Letter     string
	AnswerHash string
}

type mcqBlock struct {
	ID          string
	StartLine   int
	CorrectLine int
	OptionCount int
	Letter      string
}

type hashAnswersOptions struct {
	dryRun bool
}

func newAuthorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "author",
		Short: "Commands for lab authors",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	cmd.AddCommand(newAuthorHashAnswersCmd())
	return cmd
}

func newAuthorHashAnswersCmd() *cobra.Command {
	opts := &hashAnswersOptions{}
	cmd := &cobra.Command{
		Use:   "hash-answers <lab-id>",
		Short: "Hash MCQ answers from lab.md into checks.yaml",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			labDir, checksPath, err := resolveLabDirectory(args[0])
			if err != nil {
				return err
			}

			labPath := filepath.Join(labDir, "lab.md")
			labData, err := os.ReadFile(labPath)
			if err != nil {
				return fmt.Errorf("read lab markdown: %w", err)
			}
			checksData, err := os.ReadFile(checksPath)
			if err != nil {
				return fmt.Errorf("read checks yaml: %w", err)
			}

			result, err := hashMCQAnswers(labData, checksData)
			if err != nil {
				return err
			}
			if result.Warning != "" {
				fmt.Fprintln(cmd.ErrOrStderr(), result.Warning)
				return nil
			}

			if opts.dryRun {
				for _, answer := range result.Answers {
					shortHash := answer.AnswerHash
					if len(shortHash) > len("sha256:")+8 {
						shortHash = shortHash[:len("sha256:")+8] + "..."
					}
					fmt.Fprintf(cmd.OutOrStdout(), "%s: option %s -> %s\n", answer.ID, answer.Letter, shortHash)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Would update: %s (mcqs section)\n", checksPath)
				fmt.Fprintf(cmd.OutOrStdout(), "Would update: %s (strip [x] markers)\n", labPath)
				return nil
			}

			if err := writeFilesAtomically(map[string][]byte{
				checksPath: result.ChecksYAML,
				labPath:    result.LabMarkdown,
			}); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Updated %s\n", checksPath)
			fmt.Fprintf(cmd.OutOrStdout(), "Updated %s\n", labPath)
			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Print changes without writing files")
	return cmd
}

type hashAnswersResult struct {
	Answers     []mcqAnswer
	LabMarkdown []byte
	ChecksYAML  []byte
	Warning     string
}

func hashMCQAnswers(labData, checksData []byte) (*hashAnswersResult, error) {
	if !scenario.ContainsCheckedAnswerMarker(labData) {
		if scenario.HasMCQBlocks(labData) {
			return &hashAnswersResult{Warning: "No [x] markers found in lab.md -- nothing to do"}, nil
		}
		return &hashAnswersResult{Warning: "No MCQ blocks found in lab.md -- nothing to do"}, nil
	}

	blocks, rewrittenLab, err := parseMCQBlocks(labData)
	if err != nil {
		return nil, err
	}
	if len(blocks) == 0 {
		return &hashAnswersResult{Warning: "No MCQ blocks found in lab.md -- nothing to do"}, nil
	}

	answers := make([]mcqAnswer, 0, len(blocks))
	for _, block := range blocks {
		answers = append(answers, mcqAnswer{
			ID:         block.ID,
			Letter:     block.Letter,
			AnswerHash: hashAnswer(block.ID, block.Letter),
		})
	}
	sort.Slice(answers, func(i, j int) bool { return answers[i].ID < answers[j].ID })

	rewrittenChecks, err := rewriteChecksYAML(checksData, answers)
	if err != nil {
		return nil, err
	}

	return &hashAnswersResult{
		Answers:     answers,
		LabMarkdown: rewrittenLab,
		ChecksYAML:  rewrittenChecks,
	}, nil
}

func resolveLabDirectory(labID string) (string, string, error) {
	configPath, err := findAuthorConfig()
	if err != nil {
		return "", "", err
	}
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return "", "", fmt.Errorf("read config: %w", err)
	}

	var cfg authorConfig
	if err := yaml.Unmarshal(configData, &cfg); err != nil {
		return "", "", fmt.Errorf("parse config: %w", err)
	}
	if len(cfg.LabPaths) == 0 {
		return "", "", fmt.Errorf("config %s does not define lab_paths", configPath)
	}

	configDir := filepath.Dir(configPath)
	baseDir := filepath.Dir(configDir)
	for _, root := range cfg.LabPaths {
		resolvedRoot := root
		if !filepath.IsAbs(resolvedRoot) {
			resolvedRoot = filepath.Join(baseDir, resolvedRoot)
		}
		checksPaths, err := collectChecksFiles(resolvedRoot)
		if err != nil {
			return "", "", err
		}
		for _, checksPath := range checksPaths {
			data, err := os.ReadFile(checksPath)
			if err != nil {
				return "", "", fmt.Errorf("read checks yaml: %w", err)
			}
			var idx authorLabCheckIndex
			if err := yaml.Unmarshal(data, &idx); err != nil {
				return "", "", fmt.Errorf("parse checks yaml %s: %w", checksPath, err)
			}
			if idx.LabID == labID {
				return filepath.Dir(checksPath), checksPath, nil
			}
		}
	}

	return "", "", fmt.Errorf("lab %q not found in configured lab_paths", labID)
}

func findAuthorConfig() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}

	for dir := wd; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, ".gymctl", "config.yaml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}

	return "", fmt.Errorf("could not find .gymctl/config.yaml from %s upward", wd)
}

func collectChecksFiles(root string) ([]string, error) {
	paths := []string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Base(path) == "checks.yaml" {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan lab path %s: %w", root, err)
	}
	sort.Strings(paths)
	return paths, nil
}

func parseMCQBlocks(content []byte) ([]mcqBlock, []byte, error) {
	parsedBlocks, rewritten, err := scenario.ParseAndClearMCQMarkdownSelections(content)
	if err != nil {
		return nil, nil, err
	}

	blocks := make([]mcqBlock, 0, len(parsedBlocks))
	for _, block := range parsedBlocks {
		switch len(block.SelectedLetters) {
		case 0:
			return nil, nil, fmt.Errorf("mcq block %q has no correct answer", block.ID)
		case 1:
			blocks = append(blocks, mcqBlock{
				ID:          block.ID,
				StartLine:   block.StartLine,
				CorrectLine: 0,
				OptionCount: block.OptionCount,
				Letter:      block.SelectedLetters[0],
			})
		default:
			return nil, nil, fmt.Errorf("mcq block %q has multiple correct answers", block.ID)
		}
	}

	return blocks, rewritten, nil
}

func hashAnswer(id, letter string) string {
	return scenario.HashMCQAnswer(id, letter)
}

func rewriteChecksYAML(content []byte, answers []mcqAnswer) ([]byte, error) {
	section := renderMCQSection(answers)
	updated := replaceTopLevelYAMLSection(string(content), "mcqs", section, "checkpoints")
	return []byte(updated), nil
}

func renderMCQSection(answers []mcqAnswer) string {
	var b strings.Builder
	b.WriteString("mcqs:\n")
	for _, answer := range answers {
		b.WriteString(fmt.Sprintf("  - id: %s\n", answer.ID))
		b.WriteString(fmt.Sprintf("    answer_hash: %q\n", answer.AnswerHash))
	}
	return b.String()
}

func replaceTopLevelYAMLSection(content, key, section, insertBefore string) string {
	trailingNewline := strings.HasSuffix(content, "\n")
	trimmedContent := strings.TrimSuffix(content, "\n")
	lines := []string{}
	if trimmedContent != "" {
		lines = strings.Split(trimmedContent, "\n")
	}
	sectionLines := strings.Split(strings.TrimSuffix(section, "\n"), "\n")

	start := -1
	end := -1
	insertAt := len(lines)
	for i, line := range lines {
		if isTopLevelYAMLKey(line, key) {
			start = i
			continue
		}
		if start >= 0 && isAnyTopLevelYAMLKey(line) {
			end = i
			break
		}
		if insertBefore != "" && insertAt == len(lines) && isTopLevelYAMLKey(line, insertBefore) {
			insertAt = i
		}
	}
	if start >= 0 {
		if end < 0 {
			end = len(lines)
		}
		lines = append(append(lines[:start], sectionLines...), lines[end:]...)
	} else {
		lines = append(append(lines[:insertAt], sectionLines...), lines[insertAt:]...)
	}

	result := strings.Join(lines, "\n")
	if trailingNewline || result != "" {
		result += "\n"
	}
	return result
}

func isTopLevelYAMLKey(line, key string) bool {
	if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
		return false
	}
	return strings.HasPrefix(strings.TrimSpace(line), key+":")
}

func isAnyTopLevelYAMLKey(line string) bool {
	if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
		return false
	}
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "-") {
		return false
	}
	return strings.Contains(trimmed, ":")
}

func writeFilesAtomically(contents map[string][]byte) error {
	tmpPaths := make(map[string]string, len(contents))
	for path, data := range contents {
		dir := filepath.Dir(path)
		tmp, err := os.CreateTemp(dir, ".tmp-*")
		if err != nil {
			return fmt.Errorf("create temp file for %s: %w", path, err)
		}
		name := tmp.Name()
		if _, err := tmp.Write(data); err != nil {
			tmp.Close()
			_ = os.Remove(name)
			return fmt.Errorf("write temp file for %s: %w", path, err)
		}
		if err := tmp.Sync(); err != nil {
			tmp.Close()
			_ = os.Remove(name)
			return fmt.Errorf("sync temp file for %s: %w", path, err)
		}
		if err := tmp.Close(); err != nil {
			_ = os.Remove(name)
			return fmt.Errorf("close temp file for %s: %w", path, err)
		}
		tmpPaths[path] = name
	}

	paths := make([]string, 0, len(tmpPaths))
	for path := range tmpPaths {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		if err := os.Rename(tmpPaths[path], path); err != nil {
			return fmt.Errorf("replace %s atomically: %w", path, err)
		}
	}
	return nil
}
