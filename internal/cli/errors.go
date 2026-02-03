package cli

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// ErrorWithHint wraps an error with a helpful hint
type ErrorWithHint struct {
	Err     error
	Hint    string
	Command string
}

func (e *ErrorWithHint) Error() string {
	return e.Err.Error()
}

func (e *ErrorWithHint) Unwrap() error {
	return e.Err
}

// WrapErrorWithHint creates an error with a helpful hint
func WrapErrorWithHint(err error, hint string, command ...string) error {
	cmd := ""
	if len(command) > 0 {
		cmd = command[0]
	}
	return &ErrorWithHint{
		Err:     err,
		Hint:    hint,
		Command: cmd,
	}
}

// DiagnoseError analyzes an error and provides context-specific help
func DiagnoseError(err error) (string, string) {
	if err == nil {
		return "", ""
	}

	errStr := err.Error()
	hint := ""
	fix := ""

	// Docker-specific errors
	if strings.Contains(errStr, "docker daemon is not running") ||
		strings.Contains(errStr, "Cannot connect to the Docker daemon") {
		hint = "Docker daemon is not running"
		fix = "Start Docker Desktop or run: sudo systemctl start docker"
	} else if strings.Contains(errStr, "permission denied while trying to connect to the Docker daemon") {
		hint = "No permission to access Docker"
		fix = "Add yourself to docker group: sudo usermod -aG docker $USER && newgrp docker"
	} else if strings.Contains(errStr, "no such container") {
		hint = "Container doesn't exist"
		fix = "Run 'gymctl start' to create the required containers"
	} else if strings.Contains(errStr, "port is already allocated") {
		port := extractPort(errStr)
		hint = fmt.Sprintf("Port %s is already in use", port)
		fix = fmt.Sprintf("Stop the service using port %s or change the port in the exercise", port)
	} else if strings.Contains(errStr, "no space left on device") {
		hint = "Disk space is full"
		fix = "Clean up Docker images: docker system prune -a"
	}

	// Kubernetes-specific errors
	if strings.Contains(errStr, "kubectl: command not found") {
		hint = "kubectl is not installed"
		fix = "Install kubectl from: https://kubernetes.io/docs/tasks/tools/"
	} else if strings.Contains(errStr, "kind: command not found") {
		hint = "kind is not installed"
		fix = "Install kind from: https://kind.sigs.k8s.io/docs/user/quick-start/#installation"
	} else if strings.Contains(errStr, "error validating data") {
		hint = "Invalid Kubernetes manifest"
		fix = "Check the YAML syntax in your manifest files"
	} else if strings.Contains(errStr, "connection refused") && strings.Contains(errStr, "6443") {
		hint = "Cannot connect to Kubernetes cluster"
		fix = "Check if the cluster is running: kind get clusters"
	}

	// File/Permission errors
	if strings.Contains(errStr, "no such file or directory") {
		hint = "File or directory not found"
		fix = "Check if you're in the correct directory and the exercise is started"
	} else if strings.Contains(errStr, "permission denied") && !strings.Contains(errStr, "docker") {
		hint = "Permission denied accessing file"
		fix = "Check file permissions or run with appropriate privileges"
	}

	// Exercise-specific errors
	if strings.Contains(errStr, "exercise not found") {
		exerciseName := extractExerciseName(errStr)
		hint = fmt.Sprintf("Exercise '%s' doesn't exist", exerciseName)
		fix = "Run 'gymctl list' to see available exercises"
	} else if strings.Contains(errStr, "checks failed") {
		hint = "Exercise checks didn't pass"
		fix = "Run 'gymctl check --verbose' for details, or 'gymctl hint' for help"
	}

	// Network errors
	if strings.Contains(errStr, "timeout") {
		hint = "Operation timed out"
		fix = "Check your internet connection or increase timeout"
	} else if strings.Contains(errStr, "no route to host") {
		hint = "Cannot reach the target host"
		fix = "Check network connectivity and firewall rules"
	}

	return hint, fix
}

// extractPort attempts to extract port number from error message
func extractPort(errStr string) string {
	parts := strings.Split(errStr, ":")
	for i, part := range parts {
		if strings.Contains(part, "port") && i+1 < len(parts) {
			port := strings.TrimSpace(parts[i+1])
			// Extract just the number
			for _, char := range port {
				if char < '0' || char > '9' {
					break
				}
			}
			return port
		}
	}
	return "unknown"
}

// extractExerciseName attempts to extract exercise name from error
func extractExerciseName(errStr string) string {
	if idx := strings.Index(errStr, "exercise not found: "); idx >= 0 {
		name := errStr[idx+20:]
		if endIdx := strings.IndexAny(name, " \n\t"); endIdx >= 0 {
			return name[:endIdx]
		}
		return name
	}
	return "unknown"
}

// HandleCommandError provides user-friendly error handling for commands
func HandleCommandError(cmd *cobra.Command, err error) error {
	if err == nil {
		return nil
	}

	// Check if it's our custom error with hint
	if hintErr, ok := err.(*ErrorWithHint); ok {
		ColorError.Fprintf(cmd.ErrOrStderr(), "\n❌ Error: %v\n", hintErr.Err)
		if hintErr.Hint != "" {
			ColorInfo.Fprintf(cmd.ErrOrStderr(), "💡 Hint: %s\n", hintErr.Hint)
		}
		if hintErr.Command != "" {
			ColorInfo.Fprintf(cmd.ErrOrStderr(), "🔧 Try: %s\n", hintErr.Command)
		}
		return err
	}

	// Check if it's an exec.ExitError (command failure)
	if exitErr, ok := err.(*exec.ExitError); ok {
		ColorError.Fprintf(cmd.ErrOrStderr(), "\n❌ Command failed: %v\n", err)
		if len(exitErr.Stderr) > 0 {
			ColorDim.Fprintf(cmd.ErrOrStderr(), "Output: %s\n", string(exitErr.Stderr))
		}
	} else {
		ColorError.Fprintf(cmd.ErrOrStderr(), "\n❌ Error: %v\n", err)
	}

	// Try to diagnose the error
	hint, fix := DiagnoseError(err)
	if hint != "" {
		ColorInfo.Fprintf(cmd.ErrOrStderr(), "💡 Issue: %s\n", hint)
	}
	if fix != "" {
		ColorInfo.Fprintf(cmd.ErrOrStderr(), "🔧 Fix: %s\n", fix)
	}

	// Suggest diagnostic command
	ColorDim.Fprintln(cmd.ErrOrStderr(), "\nRun 'gymctl diagnose' to check system requirements")

	return err
}

// RecoverFromPanic recovers from panic and provides helpful error message
func RecoverFromPanic(cmd *cobra.Command) {
	if r := recover(); r != nil {
		ColorError.Fprintf(cmd.ErrOrStderr(), "\n🚨 Unexpected error occurred: %v\n", r)
		ColorInfo.Fprintln(cmd.ErrOrStderr(), "This might be a bug. Please report it with the following:")
		ColorDim.Fprintln(cmd.ErrOrStderr(), "1. Run 'gymctl diagnose > diagnostic.log'")
		ColorDim.Fprintln(cmd.ErrOrStderr(), "2. Include the command you ran")
		ColorDim.Fprintln(cmd.ErrOrStderr(), "3. Report at: https://github.com/yourusername/gymctl/issues")
	}
}