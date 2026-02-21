package cli

import (
	"context"
	"os"

	"gymctl/internal/environment"
	"gymctl/internal/scenario"
)

func resolveKubernetesProviderAndState(exerciseName string, k8s *scenario.KubernetesSpec) (environment.KubernetesProvider, *environment.ExerciseState, error) {
	provider, err := environment.ResolveKubernetesProvider(k8s)
	if err != nil {
		return nil, nil, err
	}

	state, err := environment.LoadExerciseState(exerciseName)
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
