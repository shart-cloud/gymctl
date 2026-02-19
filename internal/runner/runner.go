package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func Run(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	trimmed := strings.TrimSpace(string(output))
	if err != nil {
		return trimmed, fmt.Errorf("%s %s failed: %w\n%s", name, strings.Join(args, " "), err, trimmed)
	}
	return trimmed, nil
}

func RunInDir(ctx context.Context, dir string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	trimmed := strings.TrimSpace(string(output))
	if err != nil {
		return trimmed, fmt.Errorf("%s %s failed: %w\n%s", name, strings.Join(args, " "), err, trimmed)
	}
	return trimmed, nil
}

// RunInteractive runs a command with interactive input/output (for shells, etc.)
func RunInteractive(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
