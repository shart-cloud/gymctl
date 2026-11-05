package cli

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gymctl/internal/scenario"
)

// Local exercises are drilled in a per-exercise workspace under
// ~/.gym/workdir/<name> instead of inside the task repo. Start copies the
// task's artifacts in (answer keys excluded, so they can't be read while
// solving) and writes QUESTION.md so the brief is always one `cat` away.
// Checks still run from the task directory, so task.yaml paths like
// artifacts/<key>.json keep working unchanged; they find the user's output
// files (answer.txt, rules, ...) through $GYM_WORKDIR.

const (
	workspaceArtifactsDir = "artifacts"
	workspaceQuestionFile = "QUESTION.md"
	workspacePreviousDir  = "previous-attempt"
)

// workspaceEnvKeys are the variables a drill exports into the user's shell.
// They are unset before each start so values from a previous drill never leak.
var workspaceEnvKeys = []string{"GYM_EXERCISE", "GYM_WORKDIR", "PCAP"}

// prepareLocalWorkspace builds the drill workspace. A fresh attempt moves the
// previous attempt's files into previous-attempt/ so stale answers can't pass
// the new attempt; resuming (fresh=false) leaves them in place.
func prepareLocalWorkspace(exercise *scenario.Exercise, taskDir string, fresh bool) (string, error) {
	workDir, err := resolveWorkDir(exercise.Metadata.Name)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return "", fmt.Errorf("create workspace: %w", err)
	}
	if fresh {
		if err := archivePreviousAttempt(workDir); err != nil {
			return "", err
		}
	}

	// Refresh artifacts on every start so they always match the task and any
	// accidental edits from a previous attempt are undone. Everything else in
	// the workspace (notes, extracted files) is left alone.
	dst := filepath.Join(workDir, workspaceArtifactsDir)
	if err := os.RemoveAll(dst); err != nil {
		return "", fmt.Errorf("refresh workspace artifacts: %w", err)
	}
	src := filepath.Join(taskDir, workspaceArtifactsDir)
	if info, err := os.Stat(src); err == nil && info.IsDir() {
		if err := copyArtifacts(src, dst); err != nil {
			return "", err
		}
	}

	question := renderQuestion(exercise, workDir)
	if err := os.WriteFile(filepath.Join(workDir, workspaceQuestionFile), []byte(question), 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w", workspaceQuestionFile, err)
	}
	return workDir, nil
}

func isAnswerKeyFile(name string) bool {
	lower := strings.ToLower(name)
	if strings.Contains(lower, "answer-key") || strings.Contains(lower, "answer_key") {
		return true
	}
	for _, suffix := range []string{".key.json", ".key.yaml", ".key.yml"} {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}

func copyArtifacts(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !d.Type().IsRegular() || isAnswerKeyFile(d.Name()) {
			return nil
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm()|0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return fmt.Errorf("copy %s: %w", src, err)
	}
	return out.Close()
}

// workspacePcap returns the absolute path of the workspace's capture file when
// there is exactly one, so $PCAP is never ambiguous.
func workspacePcap(workDir string) string {
	var found []string
	for _, pattern := range []string{"*.pcap", "*.pcapng", "*.cap"} {
		matches, _ := filepath.Glob(filepath.Join(workDir, workspaceArtifactsDir, pattern))
		found = append(found, matches...)
	}
	if len(found) != 1 {
		return ""
	}
	return found[0]
}

// localWorkspaceEnv returns GYM_WORKDIR and PCAP for a local exercise whose
// workspace has been prepared.
func localWorkspaceEnv(exercise *scenario.Exercise) map[string]string {
	env := map[string]string{}
	if exercise.Spec.Environment.Type != "local" {
		return env
	}
	workDir, err := resolveWorkDir(exercise.Metadata.Name)
	if err != nil {
		return env
	}
	if info, err := os.Stat(workDir); err != nil || !info.IsDir() {
		return env
	}
	env["GYM_WORKDIR"] = workDir
	if pcap := workspacePcap(workDir); pcap != "" {
		env["PCAP"] = pcap
	}
	return env
}

func renderQuestion(exercise *scenario.Exercise, workDir string) string {
	var b strings.Builder
	title := exercise.Metadata.DisplayTitle()
	fmt.Fprintf(&b, "# %s\n\n", title)

	meta := []string{"`" + exercise.Metadata.Name + "`"}
	if exercise.Metadata.GXObjective != "" {
		meta = append(meta, "objective "+exercise.Metadata.GXObjective)
	}
	if exercise.Spec.Difficulty != "" {
		meta = append(meta, exercise.Spec.Difficulty)
	}
	if exercise.Spec.EstimatedTime != "" {
		meta = append(meta, exercise.Spec.EstimatedTime)
	}
	fmt.Fprintf(&b, "%s\n\n", strings.Join(meta, " · "))

	fmt.Fprintf(&b, "%s\n", strings.TrimRight(exercise.Spec.Description, "\n"))

	files := listWorkspaceArtifacts(workDir)
	if len(files) > 0 {
		b.WriteString("\n## Files\n\n")
		for _, f := range files {
			fmt.Fprintf(&b, "- %s\n", f)
		}
		if pcap := workspacePcap(workDir); pcap != "" {
			fmt.Fprintf(&b, "\n`$PCAP` = %s\n", pcap)
		}
	}

	b.WriteString("\nWrite answer files (answer.txt, rules, extracted files) in this directory.\n")
	b.WriteString("\n---\n`gymctl q` re-shows this · `gymctl check` grades · `gymctl hint` costs points\n")
	return b.String()
}

func listWorkspaceArtifacts(workDir string) []string {
	root := filepath.Join(workDir, workspaceArtifactsDir)
	var files []string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(workDir, path)
		if err == nil {
			files = append(files, rel)
		}
		return nil
	})
	sort.Strings(files)
	return files
}

// archivePreviousAttempt moves everything the user created in the workspace
// into previous-attempt/, replacing any older archive.
func archivePreviousAttempt(workDir string) error {
	entries, err := os.ReadDir(workDir)
	if err != nil {
		return fmt.Errorf("read workspace: %w", err)
	}
	var userFiles []string
	for _, e := range entries {
		switch e.Name() {
		case workspaceArtifactsDir, workspaceQuestionFile, workspacePreviousDir:
			continue
		}
		userFiles = append(userFiles, e.Name())
	}
	if len(userFiles) == 0 {
		return nil
	}
	archive := filepath.Join(workDir, workspacePreviousDir)
	if err := os.RemoveAll(archive); err != nil {
		return fmt.Errorf("clear %s: %w", workspacePreviousDir, err)
	}
	if err := os.MkdirAll(archive, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", workspacePreviousDir, err)
	}
	for _, name := range userFiles {
		if err := os.Rename(filepath.Join(workDir, name), filepath.Join(archive, name)); err != nil {
			return fmt.Errorf("archive %s: %w", name, err)
		}
	}
	return nil
}

// exportCheckEnv makes the drill workspace visible to check scripts, which
// run from the task directory and read the user's files via $GYM_WORKDIR.
func exportCheckEnv(exercise *scenario.Exercise) error {
	if exercise.Spec.Environment.Type != "local" {
		return nil
	}
	workDir, err := resolveWorkDir(exercise.Metadata.Name)
	if err != nil {
		return err
	}
	return os.Setenv("GYM_WORKDIR", workDir)
}
