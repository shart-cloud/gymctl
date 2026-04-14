package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHashMCQAnswersHappyPath(t *testing.T) {
	lab := "" +
		"# Lab\n\n" +
		"````mcq id=q1\n" +
		"- [ ] Alpha\n" +
		"- [x] Bravo\n" +
		"- [ ] Charlie\n" +
		"````\n\n" +
		"```mcq id=q2\n" +
		"- [ ] One\n" +
		"- [ ] Two\n" +
		"- [x] Three\n" +
		"- [ ] Four\n" +
		"```\n"
	checks := "lab_id: week-1/day-1\ncheckpoints:\n  - id: done\n"

	result, err := hashMCQAnswers([]byte(lab), []byte(checks))
	if err != nil {
		t.Fatalf("hashMCQAnswers() error = %v", err)
	}
	if result.Warning != "" {
		t.Fatalf("unexpected warning: %s", result.Warning)
	}

	gotLab := string(result.LabMarkdown)
	if strings.Contains(gotLab, "[x]") {
		t.Fatalf("rewritten lab markdown still contains [x]:\n%s", gotLab)
	}

	gotChecks := string(result.ChecksYAML)
	if !strings.Contains(gotChecks, "mcqs:\n") {
		t.Fatalf("checks yaml missing mcqs section:\n%s", gotChecks)
	}
	if !strings.Contains(gotChecks, "id: q1") || !strings.Contains(gotChecks, hashAnswer("q1", "B")) {
		t.Fatalf("checks yaml missing q1 hash:\n%s", gotChecks)
	}
	if !strings.Contains(gotChecks, "id: q2") || !strings.Contains(gotChecks, hashAnswer("q2", "C")) {
		t.Fatalf("checks yaml missing q2 hash:\n%s", gotChecks)
	}
	if strings.Index(gotChecks, "mcqs:") > strings.Index(gotChecks, "checkpoints:") {
		t.Fatalf("mcqs section should be inserted before checkpoints:\n%s", gotChecks)
	}
}

func TestHashMCQAnswersValidationFailure(t *testing.T) {
	checks := []byte("lab_id: week-1/day-1\ncheckpoints:\n  - id: done\n")
	tests := []struct {
		name string
		lab  string
		want string
	}{
		{
			name: "zero correct",
			lab:  "```mcq id=q1\n- [ ] A\n- [ ] B\n```\n\n```mcq id=q2\n- [x] A\n- [ ] B\n```\n",
			want: "has no correct answer",
		},
		{
			name: "multiple correct",
			lab:  "```mcq id=q1\n- [x] A\n- [x] B\n```\n",
			want: "has multiple correct answers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := hashMCQAnswers([]byte(tt.lab), checks)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error containing %q, got %v", tt.want, err)
			}
		})
	}
}

func TestHashMCQAnswersIdempotentRerun(t *testing.T) {
	lab := "```mcq id=q1\n- [ ] A\n- [ ] B\n- [ ] C\n```\n"
	checks := "lab_id: week-1/day-1\nmcqs:\n  - id: q1\n    answer_hash: \"sha256:abc\"\n"

	result, err := hashMCQAnswers([]byte(lab), []byte(checks))
	if err != nil {
		t.Fatalf("hashMCQAnswers() error = %v", err)
	}
	if result.Warning != "No [x] markers found in lab.md -- nothing to do" {
		t.Fatalf("unexpected warning: %q", result.Warning)
	}
}

func TestHashMCQAnswersNoMCQBlocks(t *testing.T) {
	result, err := hashMCQAnswers([]byte("# no mcqs here\n"), []byte("lab_id: week-1/day-1\n"))
	if err != nil {
		t.Fatalf("hashMCQAnswers() error = %v", err)
	}
	if result.Warning != "No MCQ blocks found in lab.md -- nothing to do" {
		t.Fatalf("unexpected warning: %q", result.Warning)
	}
}

func TestHashMCQAnswersSupportsMoreThanFourOptions(t *testing.T) {
	lab := "```mcq id=q9\n- [ ] A\n- [ ] B\n- [ ] C\n- [ ] D\n- [x] E\n```\n"
	result, err := hashMCQAnswers([]byte(lab), []byte("lab_id: edge\n"))
	if err != nil {
		t.Fatalf("hashMCQAnswers() error = %v", err)
	}
	if len(result.Answers) != 1 || result.Answers[0].Letter != "E" {
		t.Fatalf("expected option E, got %+v", result.Answers)
	}
	if result.Answers[0].AnswerHash != hashAnswer("q9", "E") {
		t.Fatalf("unexpected hash: %s", result.Answers[0].AnswerHash)
	}
}

func TestHashAnswersCommandDryRunAndWrite(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, ".gymctl")
	labDir := filepath.Join(root, "labs", "week-1-linux-basics", "day-1-first-steps")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(labDir, 0o755); err != nil {
		t.Fatal(err)
	}

	config := "lab_paths:\n  - labs\n"
	checks := "lab_id: week-1/day-1\ncheckpoints:\n  - id: done\n"
	lab := "```mcq id=q1\n- [ ] A\n- [x] B\n```\n"

	writeFile(t, filepath.Join(configDir, "config.yaml"), config)
	writeFile(t, filepath.Join(labDir, "checks.yaml"), checks)
	writeFile(t, filepath.Join(labDir, "lab.md"), lab)

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(prevWD) }()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	t.Run("dry-run", func(t *testing.T) {
		cmd := newAuthorHashAnswersCmd()
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.SetOut(&stdout)
		cmd.SetErr(&stderr)
		cmd.SetArgs([]string{"--dry-run", "week-1/day-1"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if !strings.Contains(stdout.String(), "q1: option B -> sha256:") {
			t.Fatalf("unexpected dry-run stdout:\n%s", stdout.String())
		}
		if !strings.Contains(stdout.String(), "Would update: ") {
			t.Fatalf("missing dry-run file updates:\n%s", stdout.String())
		}
		if got := readFile(t, filepath.Join(labDir, "lab.md")); got != lab {
			t.Fatalf("dry-run modified lab.md:\n%s", got)
		}
	})

	t.Run("write", func(t *testing.T) {
		cmd := newAuthorHashAnswersCmd()
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.SetOut(&stdout)
		cmd.SetErr(&stderr)
		cmd.SetArgs([]string{"week-1/day-1"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		updatedChecks := readFile(t, filepath.Join(labDir, "checks.yaml"))
		if !strings.Contains(updatedChecks, hashAnswer("q1", "B")) {
			t.Fatalf("checks.yaml missing hash:\n%s", updatedChecks)
		}
		updatedLab := readFile(t, filepath.Join(labDir, "lab.md"))
		if strings.Contains(updatedLab, "[x]") {
			t.Fatalf("lab.md still contains [x]:\n%s", updatedLab)
		}
	})
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
