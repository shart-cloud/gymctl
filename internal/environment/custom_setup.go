package environment

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gymctl/internal/runner"
	"gymctl/internal/scenario"
)

// RunCustomSetup executes scenario-defined setup steps.
//
// Supported step types:
// - "" / "script": execute step.Script with "bash -c"
// - "nodeExec": execute step.NodeExec.Script on a provider-managed node
func RunCustomSetup(ctx context.Context, baseDir string, exercise *scenario.Exercise, steps []scenario.CustomSetupStep) error {
	for i, step := range steps {
		stepType := strings.ToLower(strings.TrimSpace(step.Type))
		if stepType == "" {
			stepType = "script"
		}

		if strings.TrimSpace(step.Condition) != "" {
			ok, err := evaluateCondition(ctx, baseDir, step.Condition)
			if err != nil {
				return fmt.Errorf("customSetup[%d] condition failed: %w", i, err)
			}
			if !ok {
				continue
			}
		}

		switch stepType {
		case "script":
			if strings.TrimSpace(step.Script) == "" {
				return fmt.Errorf("customSetup[%d] missing script", i)
			}

			runCtx := ctx
			cancel := func() {}
			if strings.TrimSpace(step.Timeout) != "" {
				d, err := time.ParseDuration(step.Timeout)
				if err != nil {
					return fmt.Errorf("customSetup[%d] invalid timeout %q: %w", i, step.Timeout, err)
				}
				runCtx, cancel = context.WithTimeout(ctx, d)
			}
			_, err := runner.RunInDir(runCtx, baseDir, "bash", "-c", step.Script)
			cancel()
			if err != nil {
				return fmt.Errorf("customSetup[%d] script failed: %w", i, err)
			}
		case "nodeexec":
			if exercise == nil {
				return fmt.Errorf("customSetup[%d] missing exercise context", i)
			}
			if exercise.Spec.Environment.Kubernetes == nil {
				return fmt.Errorf("customSetup[%d] nodeExec requires kubernetes environment", i)
			}
			if step.NodeExec == nil {
				return fmt.Errorf("customSetup[%d] missing nodeExec block", i)
			}
			if strings.TrimSpace(step.NodeExec.Node) == "" {
				return fmt.Errorf("customSetup[%d] nodeExec missing node", i)
			}
			if strings.TrimSpace(step.NodeExec.Script) == "" {
				return fmt.Errorf("customSetup[%d] nodeExec missing script", i)
			}

			state, err := LoadExerciseState(exercise.Metadata.Name)
			if err != nil {
				return fmt.Errorf("customSetup[%d] load exercise state: %w", i, err)
			}
			if state == nil {
				state = &ExerciseState{ExerciseName: exercise.Metadata.Name}
			}

			provider, err := ResolveKubernetesProvider(exercise.Spec.Environment.Kubernetes)
			if err != nil {
				return fmt.Errorf("customSetup[%d] resolve kubernetes provider: %w", i, err)
			}

			timeout := strings.TrimSpace(step.NodeExec.Timeout)
			if timeout == "" {
				timeout = strings.TrimSpace(step.Timeout)
			}

			_, err = provider.NodeExec(ctx, state, step.NodeExec.Node, step.NodeExec.Script, timeout)
			if err != nil {
				return fmt.Errorf("customSetup[%d] nodeExec failed: %w", i, err)
			}
		default:
			return fmt.Errorf("customSetup[%d] unsupported type: %s", i, stepType)
		}
	}

	return nil
}

func evaluateCondition(ctx context.Context, baseDir string, condition string) (bool, error) {
	_, err := runner.RunInDir(ctx, baseDir, "bash", "-c", condition)
	if err != nil {
		// Condition false should skip the step, not fail the exercise startup.
		return false, nil
	}
	return true, nil
}
