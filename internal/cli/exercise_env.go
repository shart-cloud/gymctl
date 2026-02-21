package cli

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"gymctl/internal/environment"
	"gymctl/internal/scenario"
)

func collectExerciseEnv(ctx context.Context, exercise *scenario.Exercise) (map[string]string, []string, error) {
	envVars := map[string]string{}
	warnings := []string{}

	if exercise == nil {
		return envVars, warnings, fmt.Errorf("exercise is required")
	}

	envVars["GYM_EXERCISE"] = exercise.Metadata.Name

	if exercise.Spec.Environment.Type != "kubernetes" || exercise.Spec.Environment.Kubernetes == nil {
		return envVars, warnings, nil
	}

	k8s := exercise.Spec.Environment.Kubernetes
	provider, state, err := resolveKubernetesProviderAndState(exercise.Metadata.Name, k8s)
	if err != nil {
		return envVars, warnings, err
	}

	envVars["GYM_PROVIDER"] = state.Provider

	for logical, runtimeName := range state.Nodes {
		key := "GYM_NODE_" + sanitizeEnvName(logical)
		envVars[key] = runtimeName
	}

	kubeconfigPath := strings.TrimSpace(state.Kubeconfig)
	if kubeconfigPath != "" {
		if _, err := os.Stat(kubeconfigPath); err != nil {
			kubeconfigPath = ""
		}
	}

	if kubeconfigPath == "" {
		exportedPath, exportErr := provider.ExportKubeconfig(ctx, state, "")
		if exportErr != nil {
			if !environment.IsBareNodeKubernetes(k8s) {
				return envVars, warnings, exportErr
			}
			warnings = append(warnings, fmt.Sprintf("kubeconfig not exported yet (expected before kubeadm init): %v", exportErr))
		} else {
			kubeconfigPath = exportedPath
			if err := environment.SaveExerciseState(state); err != nil {
				return envVars, warnings, err
			}
		}
	}

	if kubeconfigPath != "" {
		envVars["GYM_KUBECONFIG"] = kubeconfigPath
		envVars["KUBECONFIG"] = kubeconfigPath
	}

	return envVars, warnings, nil
}

func sanitizeEnvName(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, ".", "_")
	value = strings.ReplaceAll(value, "/", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func sortedEnvKeys(envVars map[string]string) []string {
	keys := make([]string, 0, len(envVars))
	for key := range envVars {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func shellQuoteValue(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
