package environment

import (
	"context"
	"fmt"
	"os"
	"strings"

	"gymctl/internal/runner"
)

type KindManager struct {
	ClusterName string
}

func (k KindManager) Create(ctx context.Context, kindConfig string) error {
	args := []string{"create", "cluster", "--name", k.ClusterName}
	var tempFile string
	if kindConfig != "" {
		file, err := os.CreateTemp("", "gymctl-kind-*.yaml")
		if err != nil {
			return fmt.Errorf("create kind config temp file: %w", err)
		}
		defer os.Remove(file.Name())
		if _, err := file.WriteString(kindConfig); err != nil {
			return fmt.Errorf("write kind config: %w", err)
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("close kind config: %w", err)
		}
		tempFile = file.Name()
		args = append(args, "--config", tempFile)
	}

	_, err := runner.Run(ctx, "kind", args...)
	return err
}

func (k KindManager) Delete(ctx context.Context) error {
	_, err := runner.Run(ctx, "kind", "delete", "cluster", "--name", k.ClusterName)
	return err
}

func (k KindManager) Exists(ctx context.Context) (bool, error) {
	output, err := runner.Run(ctx, "kind", "get", "clusters")
	if err != nil {
		return false, err
	}

	for _, line := range splitLines(output) {
		if line == k.ClusterName {
			return true, nil
		}
	}

	return false, nil
}

func (k KindManager) LoadImage(ctx context.Context, image string) error {
	_, err := runner.Run(ctx, "kind", "load", "docker-image", "--name", k.ClusterName, image)
	return err
}

func splitLines(value string) []string {
	if value == "" {
		return nil
	}
	var lines []string
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
