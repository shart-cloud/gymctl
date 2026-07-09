package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseTasksDirListTrimsAndDeduplicates(t *testing.T) {
	got := parseTasksDirList(" alpha, beta ,alpha,, gamma , beta ")
	want := []string{"alpha", "beta", "gamma"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseTasksDirList() = %#v, want %#v", got, want)
	}
}

func TestResolveTasksDirectoriesUsesTasksDirsEnv(t *testing.T) {
	restoreTasksDir(t, "tasks")

	root := t.TempDir()
	repoRoot := filepath.Join(root, "course")
	repoTasksDir := filepath.Join(repoRoot, "tasks")
	explicitTasksDir := filepath.Join(root, "standalone-tasks")
	if err := os.MkdirAll(repoTasksDir, 0o755); err != nil {
		t.Fatalf("mkdir repo tasks dir: %v", err)
	}
	if err := os.MkdirAll(explicitTasksDir, 0o755); err != nil {
		t.Fatalf("mkdir explicit tasks dir: %v", err)
	}

	t.Setenv("GYMCTL_TASKS_DIRS", repoRoot+" , "+explicitTasksDir+","+repoRoot)
	t.Setenv("GYMCTL_TASKS_DIR", filepath.Join(root, "ignored"))

	resolved, err := resolveTasksDirectories()
	if err != nil {
		t.Fatalf("resolveTasksDirectories() error = %v", err)
	}
	want := []string{repoTasksDir, explicitTasksDir}
	if !reflect.DeepEqual(resolved, want) {
		t.Fatalf("resolveTasksDirectories() = %#v, want %#v", resolved, want)
	}
}

func TestResolveTasksDirectoriesUsesLegacyTasksDirEnv(t *testing.T) {
	restoreTasksDir(t, "tasks")

	root := t.TempDir()
	repoRoot := filepath.Join(root, "course")
	repoTasksDir := filepath.Join(repoRoot, "tasks")
	if err := os.MkdirAll(repoTasksDir, 0o755); err != nil {
		t.Fatalf("mkdir repo tasks dir: %v", err)
	}

	t.Setenv("GYMCTL_TASKS_DIRS", "")
	t.Setenv("GYMCTL_TASKS_DIR", " "+repoRoot+" ")

	resolved, err := resolveTasksDirectories()
	if err != nil {
		t.Fatalf("resolveTasksDirectories() error = %v", err)
	}
	want := []string{repoTasksDir}
	if !reflect.DeepEqual(resolved, want) {
		t.Fatalf("resolveTasksDirectories() = %#v, want %#v", resolved, want)
	}
}

func TestResolveTasksDirectoriesExplicitFlagWinsOverEnv(t *testing.T) {
	root := t.TempDir()
	explicitTasksDir := filepath.Join(root, "explicit")
	envTasksDir := filepath.Join(root, "env")
	if err := os.MkdirAll(explicitTasksDir, 0o755); err != nil {
		t.Fatalf("mkdir explicit tasks dir: %v", err)
	}
	if err := os.MkdirAll(envTasksDir, 0o755); err != nil {
		t.Fatalf("mkdir env tasks dir: %v", err)
	}

	restoreTasksDir(t, explicitTasksDir)
	t.Setenv("GYMCTL_TASKS_DIRS", envTasksDir)

	resolved, err := resolveTasksDirectories()
	if err != nil {
		t.Fatalf("resolveTasksDirectories() error = %v", err)
	}
	want := []string{explicitTasksDir}
	if !reflect.DeepEqual(resolved, want) {
		t.Fatalf("resolveTasksDirectories() = %#v, want %#v", resolved, want)
	}
}

func TestResolveProvidedTasksDirectoriesUsesTasksSubdir(t *testing.T) {
	root := t.TempDir()
	tasksRoot := filepath.Join(root, "gx-prep")
	tasksDir := filepath.Join(tasksRoot, "tasks")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		t.Fatalf("mkdir tasks dir: %v", err)
	}

	resolved, err := resolveProvidedTasksDirectories([]string{tasksRoot})
	if err != nil {
		t.Fatalf("resolveProvidedTasksDirectories() error = %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("len(resolved) = %d, want 1", len(resolved))
	}
	if resolved[0] != tasksDir {
		t.Fatalf("resolved[0] = %q, want %q", resolved[0], tasksDir)
	}
}

func TestResolveProvidedTasksDirectoriesDeduplicatesAfterResolution(t *testing.T) {
	root := t.TempDir()
	tasksRoot := filepath.Join(root, "gx-prep")
	tasksDir := filepath.Join(tasksRoot, "tasks")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		t.Fatalf("mkdir tasks dir: %v", err)
	}

	resolved, err := resolveProvidedTasksDirectories([]string{tasksRoot, tasksDir})
	if err != nil {
		t.Fatalf("resolveProvidedTasksDirectories() error = %v", err)
	}
	want := []string{tasksDir}
	if !reflect.DeepEqual(resolved, want) {
		t.Fatalf("resolveProvidedTasksDirectories() = %#v, want %#v", resolved, want)
	}
}

func TestResolveProvidedTasksDirectoriesKeepsExplicitTasksDir(t *testing.T) {
	root := t.TempDir()
	tasksDir := filepath.Join(root, "tasks")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		t.Fatalf("mkdir tasks dir: %v", err)
	}

	resolved, err := resolveProvidedTasksDirectories([]string{tasksDir})
	if err != nil {
		t.Fatalf("resolveProvidedTasksDirectories() error = %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("len(resolved) = %d, want 1", len(resolved))
	}
	if resolved[0] != tasksDir {
		t.Fatalf("resolved[0] = %q, want %q", resolved[0], tasksDir)
	}
}

func restoreTasksDir(t *testing.T, value string) {
	t.Helper()
	previous := tasksDir
	tasksDir = value
	t.Cleanup(func() {
		tasksDir = previous
	})
}
