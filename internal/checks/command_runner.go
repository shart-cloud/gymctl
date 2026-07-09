package checks

import (
	"context"

	"gymctl/internal/runner"
	"gymctl/internal/scenario"
)

type checkCommandRunner interface {
	Run(ctx context.Context, name string, args ...string) (string, error)
	RunInDir(ctx context.Context, dir string, name string, args ...string) (string, error)
}

type defaultCheckCommandRunner struct{}

func (defaultCheckCommandRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	return runner.Run(ctx, name, args...)
}

func (defaultCheckCommandRunner) RunInDir(ctx context.Context, dir string, name string, args ...string) (string, error) {
	return runner.RunInDir(ctx, dir, name, args...)
}

func newResult(check scenario.Check) Result {
	name := check.Name
	if name == "" {
		name = check.Type
	}
	return Result{Name: name}
}
