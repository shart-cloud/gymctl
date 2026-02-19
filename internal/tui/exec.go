package tui

import (
	"fmt"
	"os"
	"os/exec"
)

// newExecCmd creates an exec.Cmd for the given shell with the working directory set.
func newExecCmd(shell, workdir string) *exec.Cmd {
	cmd := exec.Command(shell)
	cmd.Dir = workdir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

// newGymctlStartCmd creates an exec.Cmd that runs `gymctl --tasks-dir <dir> start <name>`.
func newGymctlStartCmd(tasksDir, exerciseName string) *exec.Cmd {
	cmd := exec.Command(os.Args[0], "--tasks-dir", tasksDir, "start", exerciseName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

// newWorkstationExecCmd creates an exec.Cmd that sets up and execs into the workstation container.
func newWorkstationExecCmd(exerciseName string) *exec.Cmd {
	// Create a wrapper script that ensures workstation and then execs into it
	script := fmt.Sprintf(`#!/bin/bash
set -e

# Get gymctl path
GYMCTL_PATH="$(which gymctl || echo "%s")"

# Run gymctl shell --workstation
exec "$GYMCTL_PATH" shell --workstation
`, os.Args[0])

	// Create temporary script file
	tmpFile := "/tmp/gymctl-workstation-exec"
	if err := os.WriteFile(tmpFile, []byte(script), 0755); err != nil {
		// Fallback to direct command
		cmd := exec.Command(os.Args[0], "shell", "--workstation")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd
	}

	cmd := exec.Command("bash", tmpFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}
