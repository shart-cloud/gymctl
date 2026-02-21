package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"gymctl/internal/environment"
	"gymctl/internal/runner"
	"gymctl/internal/scenario"
)

func newSSHCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssh <node-ref> [exercise-name]",
		Short: "Open interactive shell on an exercise node",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			nodeRef := strings.TrimSpace(args[0])
			exerciseName := ""
			if len(args) == 2 {
				exerciseName = args[1]
			} else {
				current, err := loadCurrentExercise()
				if err != nil {
					return fmt.Errorf("no exercise specified and no current exercise set")
				}
				exerciseName = current
			}

			entries, err := scenario.LoadCatalog(tasksDir)
			if err != nil {
				return err
			}
			entry, found := scenario.FindByName(entries, exerciseName)
			if !found {
				return fmt.Errorf("exercise not found: %s", exerciseName)
			}
			exercise := entry.Exercise
			if exercise.Spec.Environment.Type != "kubernetes" || exercise.Spec.Environment.Kubernetes == nil {
				return fmt.Errorf("exercise %s is not a kubernetes exercise", exerciseName)
			}

			provider, state, err := resolveKubernetesProviderAndState(exerciseName, exercise.Spec.Environment.Kubernetes)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			switch provider.Name() {
			case environment.KubernetesProviderVagrant:
				vmName := deriveVagrantVMName(state, exerciseName, nodeRef)
				workspace := strings.TrimSpace(state.WorkspaceDir)
				if workspace == "" {
					gymDir, err := resolveGymDir()
					if err != nil {
						return err
					}
					workspace = filepath.Join(gymDir, "providers", "vagrant", exerciseName)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Connecting to %s (%s)...\n", nodeRef, vmName)
				return runner.RunInteractiveInDir(ctx, workspace, "vagrant", "ssh", vmName)
			case environment.KubernetesProviderKind:
				clusterName := "jerry-gym"
				if state != nil && strings.TrimSpace(state.ClusterName) != "" {
					clusterName = strings.TrimSpace(state.ClusterName)
				}
				nodeName := deriveKindNodeName(clusterName, nodeRef)
				fmt.Fprintf(cmd.OutOrStdout(), "Connecting to %s (%s)...\n", nodeRef, nodeName)
				return runner.RunInteractive(ctx, "docker", "exec", "-it", nodeName, "sh")
			default:
				return fmt.Errorf("ssh is not supported for provider: %s", provider.Name())
			}
		},
	}

	return cmd
}

func deriveVagrantVMName(state *environment.ExerciseState, exerciseName, nodeRef string) string {
	if state != nil {
		if runtimeName, ok := state.Nodes[nodeRef]; ok && strings.TrimSpace(runtimeName) != "" {
			return runtimeName
		}
	}
	if strings.HasPrefix(nodeRef, "cp-") || strings.HasPrefix(nodeRef, "worker-") {
		name := strings.ToLower(strings.TrimSpace(exerciseName + "-" + nodeRef))
		name = strings.ReplaceAll(name, "_", "-")
		name = strings.ReplaceAll(name, " ", "-")
		return name
	}
	return nodeRef
}

func deriveKindNodeName(clusterName, nodeRef string) string {
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
