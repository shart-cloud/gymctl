package environment

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gymctl/internal/runner"
	"gymctl/internal/scenario"
)

type KindProvider struct{}

func (p *KindProvider) Name() string {
	return KubernetesProviderKind
}

func (p *KindProvider) Ensure(ctx context.Context, entryDir string, spec *scenario.KubernetesSpec, state *ExerciseState) error {
	if state == nil {
		return fmt.Errorf("exercise state is required")
	}

	clusterName := resolveKindClusterName(spec)
	state.Provider = p.Name()
	state.ClusterName = clusterName

	if !ShouldCreateKubernetesCluster(spec) {
		return nil
	}

	manager := KindManager{ClusterName: clusterName}
	exists, err := manager.Exists(ctx)
	if err != nil {
		return err
	}
	if exists {
		if err := manager.Delete(ctx); err != nil {
			return err
		}
	}

	kindConfig := ""
	if spec != nil {
		if spec.Kind != nil && spec.Kind.Config != "" {
			kindConfig = spec.Kind.Config
		} else {
			kindConfig = spec.KindConfig
		}
	}

	return manager.Create(ctx, kindConfig)
}

func (p *KindProvider) Teardown(ctx context.Context, state *ExerciseState) error {
	clusterName := "jerry-gym"
	if state != nil && state.ClusterName != "" {
		clusterName = state.ClusterName
	}

	manager := KindManager{ClusterName: clusterName}
	return manager.Delete(ctx)
}

func (p *KindProvider) Reset(ctx context.Context, entryDir string, spec *scenario.KubernetesSpec, state *ExerciseState) error {
	if err := p.Teardown(ctx, state); err != nil {
		return err
	}
	return p.Ensure(ctx, entryDir, spec, state)
}

func (p *KindProvider) NodeExec(ctx context.Context, state *ExerciseState, nodeRef string, script string, timeout string) (string, error) {
	if strings.TrimSpace(script) == "" {
		return "", fmt.Errorf("script is required")
	}

	clusterName := "jerry-gym"
	if state != nil && strings.TrimSpace(state.ClusterName) != "" {
		clusterName = strings.TrimSpace(state.ClusterName)
	}

	nodeName := resolveKindNodeName(clusterName, strings.TrimSpace(nodeRef))

	runCtx := ctx
	cancel := func() {}
	if strings.TrimSpace(timeout) != "" {
		d, err := time.ParseDuration(timeout)
		if err != nil {
			return "", fmt.Errorf("invalid timeout %q: %w", timeout, err)
		}
		runCtx, cancel = context.WithTimeout(ctx, d)
	}
	defer cancel()

	output, err := runner.Run(runCtx, "docker", "exec", nodeName, "sh", "-lc", script)
	if err != nil {
		return output, err
	}

	return output, nil
}

func (p *KindProvider) ExportKubeconfig(ctx context.Context, state *ExerciseState, destPath string) (string, error) {
	clusterName := "jerry-gym"
	if state != nil && strings.TrimSpace(state.ClusterName) != "" {
		clusterName = strings.TrimSpace(state.ClusterName)
	}

	output, err := runner.Run(ctx, "kind", "get", "kubeconfig", "--name", clusterName)
	if err != nil {
		return "", err
	}

	if strings.TrimSpace(destPath) == "" {
		exerciseName := "kind"
		if state != nil && strings.TrimSpace(state.ExerciseName) != "" {
			exerciseName = strings.TrimSpace(state.ExerciseName)
		}
		destPath, err = resolveExerciseKubeconfigPath(exerciseName)
		if err != nil {
			return "", err
		}
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return "", fmt.Errorf("create kubeconfig directory: %w", err)
	}
	if err := os.WriteFile(destPath, []byte(output+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("write kubeconfig: %w", err)
	}

	if state != nil {
		state.Kubeconfig = destPath
	}

	return destPath, nil
}

func resolveKindClusterName(spec *scenario.KubernetesSpec) string {
	if spec != nil && spec.Kind != nil && spec.Kind.ClusterName != "" {
		return spec.Kind.ClusterName
	}
	return "jerry-gym"
}

func resolveKindNodeName(clusterName, nodeRef string) string {
	if nodeRef == "" {
		return clusterName + "-control-plane"
	}
	if strings.HasPrefix(nodeRef, clusterName+"-") {
		return nodeRef
	}
	if nodeRef == "cp-1" || nodeRef == "control-plane" {
		return clusterName + "-control-plane"
	}
	if strings.HasPrefix(nodeRef, "worker-") {
		suffix := strings.TrimPrefix(nodeRef, "worker-")
		if suffix == "1" {
			return clusterName + "-worker"
		}
		return clusterName + "-worker" + suffix
	}
	if strings.HasPrefix(nodeRef, "cp-") {
		suffix := strings.TrimPrefix(nodeRef, "cp-")
		if suffix == "1" {
			return clusterName + "-control-plane"
		}
		return clusterName + "-control-plane" + suffix
	}
	return nodeRef
}
