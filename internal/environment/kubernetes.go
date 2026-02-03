package environment

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"gymctl/internal/runner"
)

func ApplyManifests(ctx context.Context, namespace string, manifestPaths []string) error {
	for _, path := range manifestPaths {
		args := []string{"apply", "-f", path}
		if namespace != "" {
			args = append(args, "-n", namespace)
		}
		if _, err := runner.Run(ctx, "kubectl", args...); err != nil {
			return err
		}
	}
	return nil
}

func WaitForCondition(ctx context.Context, namespace string, resource string, condition string, timeout string) error {
	if resource == "" || condition == "" {
		return nil
	}
	if timeout == "" {
		timeout = "120s"
	}

	args := []string{"wait", "--for=condition=" + condition, "--timeout=" + timeout, resource}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	_, err := runner.Run(ctx, "kubectl", args...)
	return err
}

func ResolveManifestPaths(baseDir string, manifests []string) []string {
	paths := make([]string, 0, len(manifests))
	for _, manifest := range manifests {
		if strings.HasPrefix(manifest, "/") {
			paths = append(paths, manifest)
			continue
		}
		paths = append(paths, filepath.Join(baseDir, manifest))
	}
	return paths
}

func DescribeStart(exerciseTitle string) string {
	return fmt.Sprintf("Starting exercise: %s", exerciseTitle)
}
