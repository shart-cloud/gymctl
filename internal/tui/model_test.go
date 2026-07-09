package tui

import (
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	tea "charm.land/bubbletea/v2"

	"gymctl/internal/environment"
	"gymctl/internal/progress"
	"gymctl/internal/scenario"
)

func TestModelNavigationOpensExerciseDetail(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	catalog := testCatalog(t)
	m := NewModel(catalog, testProgress(), "tasks", filepath.Join(t.TempDir(), "progress.yaml"))
	m.height = 24

	m = updateModel(t, m, "enter")
	if m.view != viewTrackSelect {
		t.Fatalf("expected track select, got %v", m.view)
	}

	m = updateModel(t, m, "enter")
	if m.view != viewExerciseList {
		t.Fatalf("expected exercise list, got %v", m.view)
	}
	if m.selectedTrack != "docker" {
		t.Fatalf("expected docker track, got %q", m.selectedTrack)
	}

	m = updateModel(t, m, "enter")
	if m.view != viewExerciseDetail {
		t.Fatalf("expected exercise detail, got %v", m.view)
	}
	if m.selectedEx == nil || m.selectedEx.Metadata.Name != "docker-one" {
		t.Fatalf("expected docker-one selected, got %+v", m.selectedEx)
	}
}

func TestModelHintAndStartConfirmTransitions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	catalog := testCatalog(t)
	m := NewModel(catalog, testProgress(), "tasks", filepath.Join(t.TempDir(), "progress.yaml"))
	m.view = viewExerciseDetail
	m.selectedEx = catalog[0].Exercise
	m.selectedExPath = catalog[0].Dir

	m = updateModel(t, m, "h")
	if m.view != viewHintPeek {
		t.Fatalf("expected hint view, got %v", m.view)
	}
	if m.hintContent != "free hint" {
		t.Fatalf("expected free hint content, got %q", m.hintContent)
	}

	m = updateModel(t, m, "b")
	if m.view != viewExerciseDetail {
		t.Fatalf("expected detail after closing hint, got %v", m.view)
	}

	m = updateModel(t, m, "s")
	if m.view != viewStartConfirm {
		t.Fatalf("expected start confirm, got %v", m.view)
	}
	if m.startChoice != 0 {
		t.Fatalf("expected primary choice selected, got %d", m.startChoice)
	}

	m = updateModel(t, m, "down")
	if m.startChoice != 1 {
		t.Fatalf("expected secondary choice selected, got %d", m.startChoice)
	}

	m = updateModel(t, m, "up")
	if m.startChoice != 0 {
		t.Fatalf("expected primary choice selected again, got %d", m.startChoice)
	}

	m = updateModel(t, m, "esc")
	if m.view != viewExerciseDetail {
		t.Fatalf("expected detail after cancel, got %v", m.view)
	}
}

func TestModelActionDoneReloadsProgress(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	tmpDir := t.TempDir()
	progressPath := filepath.Join(tmpDir, "progress.yaml")
	if err := progress.Save(progressPath, &progress.File{
		Version: 1,
		Exercises: map[string]progress.ExerciseStatus{
			"docker-one": {Status: "completed"},
		},
	}); err != nil {
		t.Fatalf("save progress: %v", err)
	}

	catalog := testCatalog(t)
	m := NewModel(catalog, testProgress(), "tasks", progressPath)
	m.view = viewExerciseDetail
	m.selectedEx = catalog[0].Exercise
	m.selectedExPath = catalog[0].Dir

	next, cmd := m.Update(actionDoneMsg{action: "check"})
	if cmd != nil {
		t.Fatalf("expected no command")
	}
	got := next.(Model)
	if got.view != viewExerciseDetail {
		t.Fatalf("expected detail view, got %v", got.view)
	}
	if got.statusLine != "check complete" {
		t.Fatalf("expected status line, got %q", got.statusLine)
	}
	if got.prog.Exercises["docker-one"].Status != "completed" {
		t.Fatalf("expected progress reload, got %+v", got.prog.Exercises["docker-one"])
	}
	if len(got.tracks) != 1 || got.tracks[0].completed != 1 {
		t.Fatalf("expected completed track summary, got %+v", got.tracks)
	}
}

func TestModelDetailActionsUseCommandFactory(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	catalog := testCatalog(t)
	recorder := &recordingTUICommandFactory{}
	m := NewModel(catalog, testProgress(), "tasks", filepath.Join(t.TempDir(), "progress.yaml"))
	m.commands = recorder
	m.view = viewExerciseDetail
	m.selectedEx = catalog[0].Exercise
	m.selectedExPath = catalog[0].Dir
	m.exState = &environment.ExerciseState{
		Nodes: map[string]string{
			"worker-1": "node-worker",
			"cp-1":     "node-cp",
		},
	}

	for _, key := range []string{"c", "e", "k", "r", "1"} {
		// Each action dispatches from the detail view; `c` (check) transitions to
		// the streaming-run view, so reset before each key to isolate dispatch.
		m.view = viewExerciseDetail
		m.checkRunning = false
		next, cmd := m.handleKeyPress(key)
		if cmd == nil {
			t.Fatalf("expected command for key %q", key)
		}
		m = next.(Model)
	}

	want := []string{
		"check:tasks:docker-one",
		"env:tasks:docker-one",
		"kubeconfig:tasks:docker-one",
		"reset:tasks:docker-one",
		"ssh:tasks:cp-1:docker-one",
	}
	if !reflect.DeepEqual(recorder.calls, want) {
		t.Fatalf("calls = %#v, want %#v", recorder.calls, want)
	}
}

func TestModelStartConfirmUsesCommandFactory(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	catalog := testCatalog(t)
	recorder := &recordingTUICommandFactory{}
	m := NewModel(catalog, testProgress(), "tasks", filepath.Join(t.TempDir(), "progress.yaml"))
	m.commands = recorder
	m.view = viewStartConfirm
	m.selectedEx = catalog[0].Exercise
	m.selectedExPath = catalog[0].Dir

	_, cmd := m.handleKeyPress("enter")
	if cmd == nil {
		t.Fatalf("expected start command")
	}
	want := []string{"start:tasks:docker-one"}
	if !reflect.DeepEqual(recorder.calls, want) {
		t.Fatalf("calls = %#v, want %#v", recorder.calls, want)
	}
}

func updateModel(t *testing.T, m Model, key string) Model {
	t.Helper()
	next, _ := m.handleKeyPress(key)
	got, ok := next.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", next)
	}
	return got
}

type recordingTUICommandFactory struct {
	calls []string
}

func (r *recordingTUICommandFactory) Start(tasksDir, exerciseName string) *exec.Cmd {
	r.calls = append(r.calls, "start:"+tasksDir+":"+exerciseName)
	return exec.Command("true")
}

func (r *recordingTUICommandFactory) Check(tasksDir, exerciseName string) *exec.Cmd {
	r.calls = append(r.calls, "check:"+tasksDir+":"+exerciseName)
	return exec.Command("true")
}

func (r *recordingTUICommandFactory) Env(tasksDir, exerciseName string) *exec.Cmd {
	r.calls = append(r.calls, "env:"+tasksDir+":"+exerciseName)
	return exec.Command("true")
}

func (r *recordingTUICommandFactory) Kubeconfig(tasksDir, exerciseName string) *exec.Cmd {
	r.calls = append(r.calls, "kubeconfig:"+tasksDir+":"+exerciseName)
	return exec.Command("true")
}

func (r *recordingTUICommandFactory) SSH(tasksDir, nodeRef, exerciseName string) *exec.Cmd {
	r.calls = append(r.calls, "ssh:"+tasksDir+":"+nodeRef+":"+exerciseName)
	return exec.Command("true")
}

func (r *recordingTUICommandFactory) Reset(tasksDir, exerciseName string) *exec.Cmd {
	r.calls = append(r.calls, "reset:"+tasksDir+":"+exerciseName)
	return exec.Command("true")
}

func (r *recordingTUICommandFactory) Workstation(exerciseName string) *exec.Cmd {
	r.calls = append(r.calls, "workstation:"+exerciseName)
	return exec.Command("true")
}

func testProgress() *progress.File {
	return &progress.File{Version: 1, Exercises: map[string]progress.ExerciseStatus{}}
}

func testCatalog(t *testing.T) []scenario.CatalogEntry {
	t.Helper()
	dir := t.TempDir()
	ex := &scenario.Exercise{
		Metadata: scenario.ExerciseMeta{
			Name:  "docker-one",
			Title: "Docker One",
			Track: "docker",
			Order: 1,
		},
		Spec: scenario.ExerciseSpec{
			Difficulty:  "easy",
			Description: "test exercise",
			Environment: scenario.EnvironmentSpec{
				Type: "docker",
			},
			Hints: []scenario.Hint{{Cost: 0, Content: "free hint"}},
		},
	}
	return []scenario.CatalogEntry{{Exercise: ex, Dir: dir}}
}

var _ tea.Model = Model{}
