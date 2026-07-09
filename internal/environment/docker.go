package environment

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gymctl/internal/runner"
	"gymctl/internal/scenario"
)

const (
	DockerManagedLabel             = "gymctl.managed"
	DockerExerciseLabel            = "gymctl.exercise"
	DockerLegacyComposeProjectName = "jerry-gym"
)

type DockerManager struct {
	WorkDir      string
	ExerciseName string
}

func (d DockerManager) Setup(ctx context.Context, entryDir string, spec scenario.DockerSpec) error {
	if err := os.MkdirAll(d.WorkDir, 0o755); err != nil {
		return fmt.Errorf("create workdir: %w", err)
	}

	for _, item := range spec.CopyFiles {
		source := resolvePath(entryDir, item.From)
		destination := filepath.Join(d.WorkDir, item.To)
		if err := copyPath(source, destination); err != nil {
			return err
		}
	}

	if spec.ComposeFile != "" {
		composePath := resolvePath(entryDir, spec.ComposeFile)
		composeDir := filepath.Dir(composePath)
		_, err := runner.RunInDir(ctx, composeDir, "docker", "compose", "-p", DockerComposeProjectName(d.ExerciseName), "-f", composePath, "up", "-d")
		return err
	}

	if len(spec.Containers) > 0 {
		for _, container := range spec.Containers {
			image := container.Image
			if container.Build != "" {
				image = fmt.Sprintf("%s:latest", container.Name)
				buildPath := resolvePath(entryDir, container.Build)
				args := []string{"build", "-t", image}
				args = appendDockerLabels(args, d.ExerciseName)
				args = append(args, ".")
				if _, err := runner.RunInDir(ctx, buildPath, "docker", args...); err != nil {
					return err
				}
			}
			if image == "" {
				return fmt.Errorf("container %s missing image or build", container.Name)
			}
			args := []string{"run", "-d", "--name", container.Name}
			args = appendDockerLabels(args, d.ExerciseName)
			for _, port := range container.Ports {
				args = append(args, "-p", port)
			}
			args = append(args, image)
			if _, err := runner.Run(ctx, "docker", args...); err != nil {
				return err
			}
		}
	}

	// If we have copyFiles but no compose/containers, that's valid - student will build manually
	return nil
}

func (d DockerManager) Teardown(ctx context.Context, entryDir string, spec scenario.DockerSpec) error {
	if spec.ComposeFile != "" {
		composePath := resolvePath(entryDir, spec.ComposeFile)
		composeDir := filepath.Dir(composePath)
		for _, projectName := range dockerComposeTeardownProjectNames(d.ExerciseName) {
			_, err := runner.RunInDir(ctx, composeDir, "docker", "compose", "-p", projectName, "-f", composePath, "down", "-v")
			if err != nil {
				return err
			}
		}
	}

	if len(spec.Containers) > 0 {
		for _, container := range spec.Containers {
			_, _ = runner.Run(ctx, "docker", "rm", "-f", container.Name)
		}
	}

	if d.WorkDir != "" {
		_ = os.RemoveAll(d.WorkDir)
	}

	return nil
}

func appendDockerLabels(args []string, exerciseName string) []string {
	args = append(args, "--label", DockerManagedLabel+"=true")
	if strings.TrimSpace(exerciseName) != "" {
		args = append(args, "--label", DockerExerciseLabel+"="+exerciseName)
	}
	return args
}

func DockerComposeProjectName(exerciseName string) string {
	name := strings.ToLower(strings.TrimSpace(exerciseName))
	var b strings.Builder
	lastHyphen := false
	for _, r := range name {
		valid := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if valid {
			b.WriteRune(r)
			lastHyphen = false
			continue
		}
		if !lastHyphen && b.Len() > 0 {
			b.WriteByte('-')
			lastHyphen = true
		}
	}
	cleaned := strings.Trim(b.String(), "-")
	if cleaned == "" {
		return "gymctl"
	}
	return "gymctl-" + cleaned
}

func dockerComposeTeardownProjectNames(exerciseName string) []string {
	current := DockerComposeProjectName(exerciseName)
	if current == DockerLegacyComposeProjectName {
		return []string{current}
	}
	return []string{current, DockerLegacyComposeProjectName}
}

func resolvePath(baseDir string, value string) string {
	if filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(baseDir, value)
}

func copyPath(source string, destination string) error {
	info, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("stat %s: %w", source, err)
	}
	if info.IsDir() {
		return copyDir(source, destination)
	}
	return copyFile(source, destination, info.Mode())
}

func copyDir(source string, destination string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relPath)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		return copyFile(path, target, info.Mode())
	})
}

func copyFile(source string, destination string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}

	return nil
}
