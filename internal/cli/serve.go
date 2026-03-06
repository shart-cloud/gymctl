package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"gymctl/internal/cloudlab"
)

type serveOptions struct {
	scenario string
	port     int
}

func newServeCmd() *cobra.Command {
	opts := &serveOptions{}

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start a mock Kubernetes API server for a cloud lab scenario",
		Long: `Loads a scenario YAML file and serves a read-only subset of the
Kubernetes API wire protocol on the specified port. Designed to run inside
the shart.cloud container lab environment alongside ttyd.

kubectl can then be pointed at this server via a synthetic kubeconfig:

  server: http://localhost:6443

Supported endpoints: namespaces, pods, services, configmaps, deployments.`,
		// serve doesn't need the tasks directory — override the root PersistentPreRunE
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.scenario == "" {
				opts.scenario = os.Getenv("SCENARIO_PATH")
			}
			if opts.scenario == "" {
				return fmt.Errorf("--scenario flag or SCENARIO_PATH env var is required")
			}
			return cloudlab.Serve(opts.scenario, opts.port)
		},
	}

	cmd.Flags().StringVarP(&opts.scenario, "scenario", "s", "", "path to scenario YAML (or set SCENARIO_PATH)")
	cmd.Flags().IntVarP(&opts.port, "port", "p", 6443, "port to listen on")

	return cmd
}
