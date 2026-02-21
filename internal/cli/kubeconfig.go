package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"gymctl/internal/environment"
	"gymctl/internal/scenario"
)

type kubeconfigOptions struct {
	format string
	out    string
}

func newKubeconfigCmd() *cobra.Command {
	opts := &kubeconfigOptions{}

	cmd := &cobra.Command{
		Use:   "kubeconfig [exercise-name]",
		Short: "Export kubeconfig for a kubernetes exercise",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) == 1 {
				name = args[0]
			} else {
				current, err := loadCurrentExercise()
				if err != nil {
					return fmt.Errorf("no exercise specified and no current exercise set")
				}
				name = current
			}

			entries, err := scenario.LoadCatalog(tasksDir)
			if err != nil {
				return err
			}
			entry, found := scenario.FindByName(entries, name)
			if !found {
				return fmt.Errorf("exercise not found: %s", name)
			}
			exercise := entry.Exercise

			if exercise.Spec.Environment.Type != "kubernetes" || exercise.Spec.Environment.Kubernetes == nil {
				return fmt.Errorf("exercise %s is not a kubernetes exercise", name)
			}

			provider, state, err := resolveKubernetesProviderAndState(name, exercise.Spec.Environment.Kubernetes)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			kubeconfigPath, err := provider.ExportKubeconfig(ctx, state, opts.out)
			if err != nil {
				if environment.IsBareNodeKubernetes(exercise.Spec.Environment.Kubernetes) {
					return fmt.Errorf("failed to export kubeconfig (barenode mode usually needs kubeadm init first): %w", err)
				}
				return err
			}

			if err := environment.SaveExerciseState(state); err != nil {
				return err
			}

			switch strings.ToLower(opts.format) {
			case "env":
				fmt.Fprintf(cmd.OutOrStdout(), "export KUBECONFIG=%q\n", kubeconfigPath)
			case "powershell", "ps1":
				fmt.Fprintf(cmd.OutOrStdout(), "$env:KUBECONFIG = %q\n", kubeconfigPath)
			default:
				fmt.Fprintf(cmd.OutOrStdout(), "Kubeconfig written: %s\n", kubeconfigPath)
				fmt.Fprintf(cmd.OutOrStdout(), "Run (bash/zsh):       export KUBECONFIG=%q\n", kubeconfigPath)
				fmt.Fprintf(cmd.OutOrStdout(), "Run (PowerShell):     $env:KUBECONFIG = %q\n", kubeconfigPath)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&opts.format, "format", "text", "Output format: text|env|powershell")
	cmd.Flags().StringVar(&opts.out, "out", "", "Write kubeconfig to specific path")

	return cmd
}
