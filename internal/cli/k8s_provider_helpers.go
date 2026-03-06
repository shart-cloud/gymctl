package cli

import (
	"context"
	"os"
	"strings"

	"gymctl/internal/environment"
	"gymctl/internal/scenario"
)

func resolveKubernetesProviderAndState(exerciseName string, k8s *scenario.KubernetesSpec) (environment.KubernetesProvider, *environment.ExerciseState, error) {
	state, err := environment.LoadExerciseState(exerciseName)
	if err != nil {
		return nil, nil, err
	}

	resolvedSpec := k8s
	if state != nil && strings.TrimSpace(state.Provider) != "" {
		resolvedSpec = withProviderOverride(k8s, state.Provider)
	}

	provider, err := environment.ResolveKubernetesProvider(resolvedSpec)
	if err != nil {
		return nil, nil, err
	}

	if state == nil {
		state = &environment.ExerciseState{}
	}
	if state.Nodes == nil {
		state.Nodes = map[string]string{}
	}
	state.ExerciseName = exerciseName
	state.Provider = provider.Name()

	return provider, state, nil
}

func shouldCreateCluster(k8s *scenario.KubernetesSpec, noCluster bool) bool {
	if noCluster {
		return false
	}
	return environment.ShouldCreateKubernetesCluster(k8s)
}

func withCreateCluster(spec *scenario.KubernetesSpec, create bool) *scenario.KubernetesSpec {
	if spec == nil {
		return nil
	}
	cp := *spec
	cp.CreateCluster = &create
	return &cp
}

func withProviderOverride(spec *scenario.KubernetesSpec, provider string) *scenario.KubernetesSpec {
	if spec == nil || strings.TrimSpace(provider) == "" {
		return spec
	}

	cp := *spec
	cp.Provider = strings.ToLower(strings.TrimSpace(provider))
	if cp.Provider == environment.KubernetesProviderVagrant && cp.Vagrant == nil {
		cp.Vagrant = &scenario.VagrantProviderSpec{}
	}

	return &cp
}

func configureExerciseKubeconfigEnv(ctx context.Context, exercise *scenario.Exercise) error {
	if exercise == nil || exercise.Spec.Environment.Type != "kubernetes" || exercise.Spec.Environment.Kubernetes == nil {
		return nil
	}

	k8s := exercise.Spec.Environment.Kubernetes
	provider, state, err := resolveKubernetesProviderAndState(exercise.Metadata.Name, k8s)
	if err != nil {
		return err
	}

	if state.Kubeconfig != "" {
		if _, err := os.Stat(state.Kubeconfig); err == nil {
			return os.Setenv("KUBECONFIG", state.Kubeconfig)
		}
	}

	kubeconfigPath, err := provider.ExportKubeconfig(ctx, state, "")
	if err != nil {
		if environment.IsBareNodeKubernetes(k8s) {
			return nil
		}
		return err
	}

	if err := environment.SaveExerciseState(state); err != nil {
		return err
	}

	return os.Setenv("KUBECONFIG", kubeconfigPath)
}
