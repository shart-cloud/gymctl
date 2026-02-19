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
func RunCustomSetup(ctx context.Context, baseDir string, steps []scenario.CustomSetupStep) error {
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
