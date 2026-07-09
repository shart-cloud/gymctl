package checks

import (
	"context"
	"fmt"
	"os"

	"gymctl/internal/scenario"
)

type Result struct {
	Name    string
	Passed  bool
	Message string
}

type MCQResult struct {
	ID      string
	Passed  bool
	Message string
}

func RunExerciseChecks(ctx context.Context, exercise *scenario.Exercise, workDir string) ([]Result, bool) {
	return runExerciseChecksWithRunner(ctx, defaultCheckCommandRunner{}, exercise, workDir)
}

func runExerciseChecksWithRunner(ctx context.Context, commands checkCommandRunner, exercise *scenario.Exercise, workDir string) ([]Result, bool) {
	var results []Result
	allPassed := true
	for _, check := range exercise.Spec.Checks {
		result := runCheckWithRunner(ctx, commands, exercise, workDir, check)
		results = append(results, result)
		if !result.Passed {
			allPassed = false
		}
	}
	return results, allPassed
}

func RunMCQChecks(checksFile *scenario.ChecksFile, labPath string) ([]MCQResult, bool, error) {
	if checksFile == nil || len(checksFile.MCQs) == 0 {
		return nil, true, nil
	}

	data, err := os.ReadFile(labPath)
	if err != nil {
		return nil, false, fmt.Errorf("read lab markdown: %w", err)
	}

	blocks, err := scenario.ParseMCQMarkdown(data)
	if err != nil {
		return nil, false, fmt.Errorf("parse lab markdown: %w", err)
	}

	selectedByID := make(map[string]scenario.MCQBlock, len(blocks))
	for _, block := range blocks {
		selectedByID[block.ID] = block
	}

	results := make([]MCQResult, 0, len(checksFile.MCQs))
	allPassed := true
	for _, check := range checksFile.MCQs {
		result := MCQResult{ID: check.ID}
		block, ok := selectedByID[check.ID]
		if !ok {
			result.Message = "question not found in lab.md"
			results = append(results, result)
			allPassed = false
			continue
		}

		switch len(block.SelectedLetters) {
		case 0:
			result.Message = "no answer selected"
			allPassed = false
		case 1:
			result.Passed = scenario.HashMCQAnswer(check.ID, block.SelectedLetters[0]) == check.AnswerHash
			if !result.Passed {
				allPassed = false
			}
		default:
			result.Message = "multiple answers selected"
			allPassed = false
		}

		results = append(results, result)
	}

	return results, allPassed, nil
}

func runCheck(ctx context.Context, exercise *scenario.Exercise, workDir string, check scenario.Check) Result {
	return runCheckWithRunner(ctx, defaultCheckCommandRunner{}, exercise, workDir, check)
}

func runCheckWithRunner(ctx context.Context, commands checkCommandRunner, exercise *scenario.Exercise, workDir string, check scenario.Check) Result {
	result := newResult(check)
	switch check.Type {
	case "script":
		return runScriptCheckWithRunner(ctx, commands, check, workDir)
	case "http":
		return runHTTPCheck(ctx, check)
	case "file":
		return runFileCheck(check, workDir)
	}

	switch exercise.Spec.Environment.Type {
	case "kubernetes":
		if exercise.Spec.Environment.Kubernetes == nil {
			result.Message = "missing kubernetes config"
			return result
		}
		namespace := exercise.Spec.Environment.Kubernetes.Namespace
		if namespace == "" {
			namespace = "default"
		}
		if check.Namespace != "" {
			namespace = check.Namespace
		}

		switch check.Type {
		case "jsonpath":
			return runJSONPathCheckWithRunner(ctx, commands, namespace, check)
		case "condition":
			return runConditionCheckWithRunner(ctx, commands, namespace, check)
		case "resourceExists", "resource":
			return runResourceExistsCheckWithRunner(ctx, commands, namespace, check)
		case "podLogs":
			return runPodLogsCheckWithRunner(ctx, commands, namespace, check)
		case "exec":
			return runKubernetesExecCheckWithRunner(ctx, commands, namespace, check)
		case "nodeExec":
			return runNodeExecCheck(ctx, exercise, check)
		default:
			result.Message = fmt.Sprintf("unsupported check type: %s", check.Type)
			return result
		}
	case "docker":
		switch check.Type {
		case "docker-image":
			return runDockerImageCheckWithRunner(ctx, commands, check)
		case "docker-container":
			return runDockerContainerCheckWithRunner(ctx, commands, check)
		case "docker-logs":
			return runDockerLogsCheckWithRunner(ctx, commands, check)
		case "dockerfile":
			return runDockerfileCheck(check, workDir)
		case "exec":
			return runDockerExecCheckWithRunner(ctx, commands, check)
		default:
			result.Message = fmt.Sprintf("unsupported check type: %s", check.Type)
			return result
		}
	case "local":
		result.Message = fmt.Sprintf("unsupported local-only check type: %s", check.Type)
		return result
	default:
		result.Message = "unsupported environment for checks"
		return result
	}
}
