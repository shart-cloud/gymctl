package environment

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"gymctl/internal/runner"
)

const (
	WorkstationContainerName = "gymctl-workstation"
	WorkstationImageName     = "gymctl-workstation:latest"
)

type WorkstationManager struct{}

// EnsureRunning ensures the workstation container is built and running
func (w *WorkstationManager) EnsureRunning(ctx context.Context, gymctlBinaryPath string) error {
	// Check if image exists, build if not
	exists, err := w.imageExists(ctx)
	if err != nil {
		return fmt.Errorf("check image exists: %w", err)
	}

	if !exists {
		if err := w.buildImage(ctx); err != nil {
			return fmt.Errorf("build workstation image: %w", err)
		}
	}

	// Check if container is running
	running, err := w.isRunning(ctx)
	if err != nil {
		return fmt.Errorf("check container running: %w", err)
	}

	if !running {
		if err := w.startContainer(ctx, gymctlBinaryPath); err != nil {
			return fmt.Errorf("start workstation container: %w", err)
		}
	}

	return nil
}

// ExecShell execs into the workstation container with a shell
func (w *WorkstationManager) ExecShell(ctx context.Context, workDir string) error {
	// Change to the work directory inside container
	cmd := []string{"docker", "exec", "-it", WorkstationContainerName, "sh", "-c",
		fmt.Sprintf("cd %s && exec zsh", workDir)}

	return runner.RunInteractive(ctx, cmd[0], cmd[1:]...)
}

// CopyExerciseFiles copies exercise files to the workstation container
func (w *WorkstationManager) CopyExerciseFiles(ctx context.Context, hostPath, containerPath string) error {
	_, err := runner.Run(ctx, "docker", "cp", hostPath,
		fmt.Sprintf("%s:%s", WorkstationContainerName, containerPath))
	return err
}

// InstallGymctl copies the gymctl binary to the workstation container
func (w *WorkstationManager) InstallGymctl(ctx context.Context, binaryPath string) error {
	// Copy gymctl binary to container
	_, err := runner.Run(ctx, "docker", "cp", binaryPath,
		fmt.Sprintf("%s:/usr/local/bin/gymctl", WorkstationContainerName))
	if err != nil {
		return err
	}

	// Make it executable
	_, err = runner.Run(ctx, "docker", "exec", WorkstationContainerName,
		"chmod", "+x", "/usr/local/bin/gymctl")
	return err
}

func (w *WorkstationManager) imageExists(ctx context.Context) (bool, error) {
	_, err := runner.Run(ctx, "docker", "image", "inspect", WorkstationImageName)
	if err != nil {
		// Image doesn't exist
		return false, nil
	}
	return true, nil
}

func (w *WorkstationManager) buildImage(ctx context.Context) error {
	// Get the directory containing Dockerfile.workstation
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	_, err = runner.RunInDir(ctx, wd, "docker", "build",
		"-f", "Dockerfile.workstation",
		"-t", WorkstationImageName,
		".")
	return err
}

func (w *WorkstationManager) isRunning(ctx context.Context) (bool, error) {
	out, err := runner.Run(ctx, "docker", "ps", "--filter",
		fmt.Sprintf("name=%s", WorkstationContainerName), "--format", "{{.Names}}")
	if err != nil {
		return false, err
	}
	return out == WorkstationContainerName, nil
}

func (w *WorkstationManager) startContainer(ctx context.Context, gymctlBinaryPath string) error {
	// Remove existing container if it exists but isn't running
	runner.Run(ctx, "docker", "rm", "-f", WorkstationContainerName)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	gymDir := filepath.Join(homeDir, ".gym")
	if err := os.MkdirAll(gymDir, 0755); err != nil {
		return fmt.Errorf("create .gym directory: %w", err)
	}

	// Start container with mounted volumes
	args := []string{
		"run", "-d", "--name", WorkstationContainerName,
		// Mount .gym directory for persistent progress
		"-v", fmt.Sprintf("%s:/home/student/.gym", gymDir),
		// Mount current directory for exercise access
		"-v", fmt.Sprintf("%s:/workspace", filepath.Dir(gymctlBinaryPath)),
		// Mount docker socket for docker-in-docker
		"-v", "/var/run/docker.sock:/var/run/docker.sock",
		// Keep container running
		WorkstationImageName,
		"sleep", "infinity",
	}

	if _, err := runner.Run(ctx, "docker", args...); err != nil {
		return err
	}

	// Install gymctl in the container
	return w.InstallGymctl(ctx, gymctlBinaryPath)
}