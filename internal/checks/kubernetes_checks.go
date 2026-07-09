package checks

import (
	"context"
	"fmt"
	"strings"

	"gymctl/internal/environment"
	"gymctl/internal/scenario"
)

func runJSONPathCheckWithRunner(ctx context.Context, commands checkCommandRunner, namespace string, check scenario.Check) Result {
	result := newResult(check)
	if check.Resource == "" || check.Jsonpath == "" {
		result.Message = "missing resource or jsonpath"
		return result
	}
	args := []string{"get", check.Resource, "-o", fmt.Sprintf("jsonpath=%s", check.Jsonpath)}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	output, err := commands.Run(ctx, "kubectl", args...)
	if err != nil {
		result.Message = err.Error()
		return result
	}

	passed, msg := compareValue(strings.TrimSpace(output), check.Operator, check.Value, check.ValueType)
	result.Passed = passed
	result.Message = msg
	return result
}

func runConditionCheckWithRunner(ctx context.Context, commands checkCommandRunner, namespace string, check scenario.Check) Result {
	result := newResult(check)
	if check.Resource == "" || check.Condition == "" {
		result.Message = "missing resource or condition"
		return result
	}
	status := check.Status
	if status == "" {
		status = "True"
	}
	jsonpath := fmt.Sprintf("{.status.conditions[?(@.type==\"%s\")].status}", check.Condition)
	args := []string{"get", check.Resource, "-o", "jsonpath=" + jsonpath}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	output, err := commands.Run(ctx, "kubectl", args...)
	if err != nil {
		result.Message = err.Error()
		return result
	}
	output = strings.TrimSpace(output)
	if output == status {
		result.Passed = true
		return result
	}
	result.Message = fmt.Sprintf("expected %s, got %s", status, output)
	return result
}

func runResourceExistsCheckWithRunner(ctx context.Context, commands checkCommandRunner, namespace string, check scenario.Check) Result {
	result := newResult(check)
	if check.Resource == "" {
		result.Message = "missing resource"
		return result
	}

	args := []string{"get", check.Resource}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	_, err := commands.Run(ctx, "kubectl", args...)
	exists := err == nil
	expected := true
	if check.Exists != nil {
		expected = *check.Exists
	} else if check.ShouldExist != nil {
		expected = *check.ShouldExist
	}

	if exists == expected {
		result.Passed = true
		return result
	}
	result.Message = fmt.Sprintf("expected exists=%t, got %t", expected, exists)
	return result
}

func runPodLogsCheckWithRunner(ctx context.Context, commands checkCommandRunner, namespace string, check scenario.Check) Result {
	result := newResult(check)
	if check.Selector == "" && check.Resource == "" {
		result.Message = "missing selector or resource for pod logs"
		return result
	}

	args := []string{"logs"}
	if check.Selector != "" {
		args = append(args, "-l", check.Selector)
	} else {
		args = append(args, check.Resource)
	}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	if check.Container != "" {
		args = append(args, "-c", check.Container)
	}
	if check.Timeout != "" {
		args = append(args, "--since", check.Timeout)
	}

	output, err := commands.Run(ctx, "kubectl", args...)
	if err != nil {
		result.Message = err.Error()
		return result
	}

	passed, msg := compareValue(output, check.Operator, check.Value, "string")
	result.Passed = passed
	result.Message = msg
	return result
}

func runKubernetesExecCheckWithRunner(ctx context.Context, commands checkCommandRunner, namespace string, check scenario.Check) Result {
	result := newResult(check)
	if check.Resource == "" || len(check.Command) == 0 {
		result.Message = "missing resource or command for exec"
		return result
	}

	args := []string{"exec", check.Resource}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	if check.Container != "" {
		args = append(args, "-c", check.Container)
	}
	args = append(args, "--")
	args = append(args, check.Command...)

	output, err := commands.Run(ctx, "kubectl", args...)
	return evaluateCommandResult(output, err, check, result)
}

func runNodeExecCheck(ctx context.Context, exercise *scenario.Exercise, check scenario.Check) Result {
	result := newResult(check)
	if exercise == nil || exercise.Spec.Environment.Kubernetes == nil {
		result.Message = "nodeExec requires kubernetes environment"
		return result
	}
	if strings.TrimSpace(check.Node) == "" {
		result.Message = "missing node"
		return result
	}
	if strings.TrimSpace(check.Script) == "" {
		result.Message = "missing script"
		return result
	}

	provider, err := environment.ResolveKubernetesProvider(exercise.Spec.Environment.Kubernetes)
	if err != nil {
		result.Message = err.Error()
		return result
	}

	state, err := environment.LoadExerciseState(exercise.Metadata.Name)
	if err != nil {
		result.Message = err.Error()
		return result
	}
	if state == nil {
		state = &environment.ExerciseState{ExerciseName: exercise.Metadata.Name}
	}

	output, err := provider.NodeExec(ctx, state, check.Node, check.Script, check.Timeout)
	return evaluateCommandResult(output, err, check, result)
}
