package tui

import (
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"gymctl/internal/pet"
	"gymctl/internal/progress"
)

// drainCheck runs the stream and collects all line output plus the terminal error.
func drainCheck(t *testing.T, cmd *exec.Cmd) (lines []string, doneErr error) {
	t.Helper()
	msg := streamCheck(cmd)()
	started, ok := msg.(checkStartedMsg)
	if !ok {
		t.Fatalf("streamCheck first msg = %T, want checkStartedMsg", msg)
	}
	for i := 0; i < 1000; i++ {
		ev := waitForCheck(started.events)().(checkEventMsg)
		if ev.done {
			return lines, ev.err
		}
		lines = append(lines, ev.line)
	}
	t.Fatal("stream never reported done")
	return nil, nil
}

func TestStreamCheckDeliversLinesThenDone(t *testing.T) {
	lines, err := drainCheck(t, exec.Command("sh", "-c", "printf 'alpha\\nbeta\\n'"))
	if err != nil {
		t.Fatalf("unexpected done err: %v", err)
	}
	if !reflect.DeepEqual(lines, []string{"alpha", "beta"}) {
		t.Fatalf("lines = %#v, want [alpha beta]", lines)
	}
}

func TestStreamCheckNonZeroExitCarriesError(t *testing.T) {
	// `gymctl check` exits non-zero when not passing; that must surface as a
	// non-nil done error (which the model reads as "not passing yet").
	lines, err := drainCheck(t, exec.Command("sh", "-c", "printf 'not passing\\n'; exit 1"))
	if err == nil {
		t.Fatal("expected non-nil done err on non-zero exit")
	}
	if !reflect.DeepEqual(lines, []string{"not passing"}) {
		t.Fatalf("lines = %#v, want [not passing]", lines)
	}
}

func TestStreamCheckMergesStderr(t *testing.T) {
	lines, err := drainCheck(t, exec.Command("sh", "-c", "echo out; echo err 1>&2"))
	if err != nil {
		t.Fatalf("unexpected done err: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("want 2 merged lines, got %#v", lines)
	}
}

func TestCheckKeyEntersRunView(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	catalog := testCatalog(t)
	recorder := &recordingTUICommandFactory{}
	m := NewModel(catalog, testProgress(), "tasks", filepath.Join(t.TempDir(), "progress.yaml"))
	m.commands = recorder
	m.view = viewExerciseDetail
	m.selectedEx = catalog[0].Exercise
	m.selectedExPath = catalog[0].Dir

	next, cmd := m.handleKeyPress("c")
	m = next.(Model)
	if m.view != viewCheckRun {
		t.Fatalf("view = %v, want viewCheckRun", m.view)
	}
	if !m.checkRunning {
		t.Fatal("expected checkRunning = true")
	}
	if m.petMood != pet.Thinking {
		t.Fatalf("mood = %v, want Thinking", m.petMood)
	}
	if cmd == nil {
		t.Fatal("expected a streamCheck command")
	}
	if got := recorder.calls; len(got) != 1 || got[0] != "check:tasks:docker-one" {
		t.Fatalf("command factory not exercised synchronously: %#v", got)
	}

	// A second `c` while running must be a no-op (no new command).
	_, cmd2 := m.handleKeyPress("c")
	if cmd2 != nil {
		t.Fatal("expected no command for a second check while one is running")
	}
}

func TestCheckEventReducerAccumulatesAndFinalizes(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	tmpDir := t.TempDir()
	progressPath := filepath.Join(tmpDir, "progress.yaml")
	if err := progress.Save(progressPath, &progress.File{
		Version:   1,
		Exercises: map[string]progress.ExerciseStatus{},
	}); err != nil {
		t.Fatalf("save progress: %v", err)
	}
	catalog := testCatalog(t)
	m := NewModel(catalog, testProgress(), "tasks", progressPath)
	m.view = viewCheckRun
	m.checkRunning = true
	m.checkFollow = true
	m.height = 24
	m.selectedEx = catalog[0].Exercise
	m.selectedExPath = catalog[0].Dir

	for _, ln := range []string{"checking foo", "checking bar"} {
		next, _ := m.Update(checkEventMsg{line: ln})
		m = next.(Model)
	}
	if !reflect.DeepEqual(m.checkOutput, []string{"checking foo", "checking bar"}) {
		t.Fatalf("checkOutput = %#v", m.checkOutput)
	}

	next, cmd := m.Update(checkEventMsg{done: true, err: nil})
	m = next.(Model)
	if cmd != nil {
		t.Fatal("expected no follow-up command once the stream is done")
	}
	if m.checkRunning {
		t.Fatal("checkRunning should be false after done")
	}
	if m.statusLine != "check complete" {
		t.Fatalf("statusLine = %q, want 'check complete'", m.statusLine)
	}
}

func TestCheckEventReducerNotPassing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	progressPath := filepath.Join(t.TempDir(), "progress.yaml")
	if err := progress.Save(progressPath, &progress.File{Version: 1, Exercises: map[string]progress.ExerciseStatus{}}); err != nil {
		t.Fatalf("save progress: %v", err)
	}
	catalog := testCatalog(t)
	m := NewModel(catalog, testProgress(), "tasks", progressPath)
	m.view = viewCheckRun
	m.checkRunning = true
	m.height = 24
	m.selectedEx = catalog[0].Exercise
	m.selectedExPath = catalog[0].Dir

	next, _ := m.Update(checkEventMsg{done: true, err: exec.ErrNotFound})
	m = next.(Model)
	if m.statusLine != "check: not passing yet" {
		t.Fatalf("statusLine = %q", m.statusLine)
	}
	if m.petMood != pet.Sad {
		t.Fatalf("mood = %v, want Sad on a non-passing check", m.petMood)
	}
}
