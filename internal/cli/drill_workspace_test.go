package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gymctl/internal/scenario"
)

func TestPrepareLocalWorkspace(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	taskDir := t.TempDir()
	art := filepath.Join(taskDir, "artifacts")
	if err := os.MkdirAll(filepath.Join(art, "logs"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"capture.pcap":            "pcap",
		"capture.answer-key.json": "{}",
		"scenario.key.yaml":       "k",
		"logs/conn.log":           "conn",
	} {
		if err := os.WriteFile(filepath.Join(art, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	ex := &scenario.Exercise{}
	ex.Metadata.Name = "sec503-demo-001"
	ex.Metadata.Title = "Demo"
	ex.Metadata.GXObjective = "1.5"
	ex.Spec.Description = "Find the thing.\nWrite it to ~/answer.txt"
	ex.Spec.Environment.Type = "local"

	workDir, err := prepareLocalWorkspace(ex, taskDir)
	if err != nil {
		t.Fatalf("prepareLocalWorkspace: %v", err)
	}
	if want := filepath.Join(home, ".gym", "workdir", "sec503-demo-001"); workDir != want {
		t.Fatalf("workDir = %s, want %s", workDir, want)
	}

	for _, name := range []string{"capture.pcap", "logs/conn.log"} {
		if _, err := os.Stat(filepath.Join(workDir, "artifacts", name)); err != nil {
			t.Errorf("expected %s copied: %v", name, err)
		}
	}
	for _, name := range []string{"capture.answer-key.json", "scenario.key.yaml"} {
		if _, err := os.Stat(filepath.Join(workDir, "artifacts", name)); err == nil {
			t.Errorf("answer key %s must not be copied into the workspace", name)
		}
	}

	q, err := os.ReadFile(filepath.Join(workDir, "QUESTION.md"))
	if err != nil {
		t.Fatalf("QUESTION.md: %v", err)
	}
	for _, want := range []string{"# Demo", "objective 1.5", "Find the thing.", "artifacts/capture.pcap", "$PCAP"} {
		if !strings.Contains(string(q), want) {
			t.Errorf("QUESTION.md missing %q:\n%s", want, q)
		}
	}

	env := localWorkspaceEnv(ex)
	if env["PCAP"] != filepath.Join(workDir, "artifacts", "capture.pcap") {
		t.Errorf("PCAP = %q", env["PCAP"])
	}
	if env["GYM_WORKDIR"] != workDir {
		t.Errorf("GYM_WORKDIR = %q", env["GYM_WORKDIR"])
	}

	// A second start refreshes artifacts but keeps the user's own files.
	notes := filepath.Join(workDir, "notes.txt")
	if err := os.WriteFile(notes, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "artifacts", "capture.pcap"), []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := prepareLocalWorkspace(ex, taskDir); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(workDir, "artifacts", "capture.pcap")); string(b) != "pcap" {
		t.Errorf("artifact not refreshed: %q", b)
	}
	if _, err := os.Stat(notes); err != nil {
		t.Errorf("user notes removed: %v", err)
	}
}

func TestWorkspacePcapAmbiguous(t *testing.T) {
	dir := t.TempDir()
	art := filepath.Join(dir, "artifacts")
	_ = os.MkdirAll(art, 0o755)
	_ = os.WriteFile(filepath.Join(art, "a.pcap"), nil, 0o644)
	_ = os.WriteFile(filepath.Join(art, "b.pcapng"), nil, 0o644)
	if got := workspacePcap(dir); got != "" {
		t.Errorf("workspacePcap with two captures = %q, want empty", got)
	}
}

func TestShellDirectivesEmit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "directives")
	t.Setenv(shellDirectivesEnv, path)
	err := shellDirectives{
		unset:  []string{"PCAP"},
		export: map[string]string{"PCAP": "/tmp/it's here.pcap"},
		cd:     "/tmp/work dir",
	}.emit()
	if err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	want := "unset PCAP\nexport PCAP='/tmp/it'\"'\"'s here.pcap'\ncd '/tmp/work dir'\n"
	if string(got) != want {
		t.Errorf("directives =\n%s\nwant\n%s", got, want)
	}
}
