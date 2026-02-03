package checks

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"

	"gymctl/internal/runner"
	"gymctl/internal/scenario"
)

type Result struct {
	Name    string
	Passed  bool
	Message string
}

func RunExerciseChecks(ctx context.Context, exercise *scenario.Exercise, workDir string) ([]Result, bool) {
	var results []Result
	allPassed := true
	for _, check := range exercise.Spec.Checks {
		result := runCheck(ctx, exercise, workDir, check)
		results = append(results, result)
		if !result.Passed {
			allPassed = false
		}
	}
	return results, allPassed
}

func runCheck(ctx context.Context, exercise *scenario.Exercise, workDir string, check scenario.Check) Result {
	result := Result{Name: check.Name, Passed: false}
	if check.Name == "" {
		result.Name = check.Type
	}

	// Environment-agnostic checks
	switch check.Type {
	case "script":
		return runScriptCheck(ctx, check, workDir)
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
			return runJSONPathCheck(ctx, namespace, check)
		case "condition":
			return runConditionCheck(ctx, namespace, check)
		case "resourceExists":
			return runResourceExistsCheck(ctx, namespace, check)
		case "podLogs":
			return runPodLogsCheck(ctx, namespace, check)
		case "exec":
			return runKubernetesExecCheck(ctx, namespace, check)
		default:
			result.Message = fmt.Sprintf("unsupported check type: %s", check.Type)
			return result
		}
	case "docker":
		switch check.Type {
		case "docker-image":
			return runDockerImageCheck(ctx, check)
		case "docker-container":
			return runDockerContainerCheck(ctx, check)
		case "docker-logs":
			return runDockerLogsCheck(ctx, check)
		case "dockerfile":
			return runDockerfileCheck(check, workDir)
		case "exec":
			return runDockerExecCheck(ctx, check)
		default:
			result.Message = fmt.Sprintf("unsupported check type: %s", check.Type)
			return result
		}
	default:
		result.Message = "unsupported environment for checks"
		return result
	}
}

func runJSONPathCheck(ctx context.Context, namespace string, check scenario.Check) Result {
	result := Result{Name: check.Name}
	if check.Resource == "" || check.Jsonpath == "" {
		result.Message = "missing resource or jsonpath"
		return result
	}
	args := []string{"get", check.Resource, "-o", fmt.Sprintf("jsonpath=%s", check.Jsonpath)}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	output, err := runner.Run(ctx, "kubectl", args...)
	if err != nil {
		result.Message = err.Error()
		return result
	}

	passed, msg := compareValue(strings.TrimSpace(output), check.Operator, check.Value, check.ValueType)
	result.Passed = passed
	result.Message = msg
	return result
}

func runConditionCheck(ctx context.Context, namespace string, check scenario.Check) Result {
	result := Result{Name: check.Name}
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
	output, err := runner.Run(ctx, "kubectl", args...)
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

func runResourceExistsCheck(ctx context.Context, namespace string, check scenario.Check) Result {
	result := Result{Name: check.Name}
	if check.Resource == "" {
		result.Message = "missing resource"
		return result
	}

	args := []string{"get", check.Resource}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	_, err := runner.Run(ctx, "kubectl", args...)
	exists := err == nil
	expected := true
	if check.Exists != nil {
		expected = *check.Exists
	}

	if exists == expected {
		result.Passed = true
		return result
	}
	result.Message = fmt.Sprintf("expected exists=%t, got %t", expected, exists)
	return result
}

func compareValue(actual string, operator string, expected interface{}, valueType string) (bool, string) {
	if operator == "" {
		operator = "equals"
	}
	if operator == "exists" {
		if actual != "" {
			return true, ""
		}
		return false, "value not found"
	}

	stringExpected := ""
	if expected != nil {
		stringExpected = fmt.Sprintf("%v", expected)
	}

	switch operator {
	case "equals":
		if actual == stringExpected {
			return true, ""
		}
		return false, fmt.Sprintf("expected %s, got %s", stringExpected, actual)
	case "notEquals":
		if actual != stringExpected {
			return true, ""
		}
		return false, fmt.Sprintf("expected not %s, got %s", stringExpected, actual)
	case "contains":
		if strings.Contains(actual, stringExpected) {
			return true, ""
		}
		return false, fmt.Sprintf("expected contains %s, got %s", stringExpected, actual)
	case "regex":
		matched, err := regexp.MatchString(stringExpected, actual)
		if err != nil {
			return false, fmt.Sprintf("invalid regex: %s", err)
		}
		if matched {
			return true, ""
		}
		return false, fmt.Sprintf("expected %s to match %s", actual, stringExpected)
	case "greaterThan", "lessThan":
		return compareOrdered(actual, stringExpected, operator, valueType)
	default:
		return false, fmt.Sprintf("unsupported operator: %s", operator)
	}
}

func compareOrdered(actual string, expected string, operator string, valueType string) (bool, string) {
	switch valueType {
	case "number":
		actualNum, err := strconv.ParseFloat(actual, 64)
		if err != nil {
			return false, fmt.Sprintf("invalid number: %s", actual)
		}
		expectedNum, err := strconv.ParseFloat(expected, 64)
		if err != nil {
			return false, fmt.Sprintf("invalid expected number: %s", expected)
		}
		if operator == "greaterThan" {
			if actualNum > expectedNum {
				return true, ""
			}
			return false, fmt.Sprintf("expected %v > %v", actualNum, expectedNum)
		}
		if actualNum < expectedNum {
			return true, ""
		}
		return false, fmt.Sprintf("expected %v < %v", actualNum, expectedNum)
	case "quantity":
		actualQty, err := resource.ParseQuantity(actual)
		if err != nil {
			return false, fmt.Sprintf("invalid quantity: %s", actual)
		}
		expectedQty, err := resource.ParseQuantity(expected)
		if err != nil {
			return false, fmt.Sprintf("invalid expected quantity: %s", expected)
		}
		cmp := actualQty.Cmp(expectedQty)
		if operator == "greaterThan" {
			if cmp > 0 {
				return true, ""
			}
			return false, fmt.Sprintf("expected %s > %s", actualQty.String(), expectedQty.String())
		}
		if cmp < 0 {
			return true, ""
		}
		return false, fmt.Sprintf("expected %s < %s", actualQty.String(), expectedQty.String())
	default:
		return false, "valueType required for ordered comparison"
	}
}

func runDockerImageCheck(ctx context.Context, check scenario.Check) Result {
	result := Result{Name: check.Name}
	if check.Image == "" || check.Property == "" {
		result.Message = "missing image or property"
		return result
	}

	switch check.Property {
	case "size":
		output, err := runner.Run(ctx, "docker", "image", "inspect", check.Image, "--format", "{{.Size}}")
		if err != nil {
			result.Message = err.Error()
			return result
		}
		actualSize, err := strconv.ParseInt(strings.TrimSpace(output), 10, 64)
		if err != nil {
			result.Message = fmt.Sprintf("invalid image size: %s", output)
			return result
		}
		expectedSize, err := parseSize(fmt.Sprintf("%v", check.Value))
		if err != nil {
			result.Message = err.Error()
			return result
		}
		passed, msg := compareInt(actualSize, expectedSize, check.Operator)
		result.Passed = passed
		result.Message = msg
		return result
	case "layers":
		output, err := runner.Run(ctx, "docker", "image", "inspect", check.Image, "--format", "{{len .RootFS.Layers}}")
		if err != nil {
			result.Message = err.Error()
			return result
		}
		actualLayers, err := strconv.ParseInt(strings.TrimSpace(output), 10, 64)
		if err != nil {
			result.Message = fmt.Sprintf("invalid layer count: %s", output)
			return result
		}
		expectedLayers, err := strconv.ParseInt(fmt.Sprintf("%v", check.Value), 10, 64)
		if err != nil {
			result.Message = fmt.Sprintf("invalid expected layers: %v", check.Value)
			return result
		}
		passed, msg := compareInt(actualLayers, expectedLayers, check.Operator)
		result.Passed = passed
		result.Message = msg
		return result
	case "baseImage":
		output, err := runner.Run(ctx, "docker", "image", "inspect", check.Image, "--format", "{{.ContainerConfig.Image}}")
		if err != nil {
			result.Message = err.Error()
			return result
		}
		passed, msg := compareValue(strings.TrimSpace(output), check.Operator, check.Value, "string")
		result.Passed = passed
		result.Message = msg
		return result
	case "labels":
		output, err := runner.Run(ctx, "docker", "image", "inspect", check.Image, "--format", "{{.Config.Labels}}")
		if err != nil {
			result.Message = err.Error()
			return result
		}
		passed, msg := compareValue(strings.TrimSpace(output), check.Operator, check.Value, "string")
		result.Passed = passed
		result.Message = msg
		return result
	default:
		result.Message = fmt.Sprintf("unsupported docker image property: %s", check.Property)
		return result
	}
}

func runDockerContainerCheck(ctx context.Context, check scenario.Check) Result {
	result := Result{Name: check.Name}
	if check.Container == "" || check.Property == "" {
		result.Message = "missing container or property"
		return result
	}

	switch check.Property {
	case "state":
		output, err := runner.Run(ctx, "docker", "inspect", check.Container, "--format", "{{.State.Status}}")
		if err != nil {
			result.Message = err.Error()
			return result
		}
		passed, msg := compareValue(strings.TrimSpace(output), check.Operator, check.Value, "string")
		result.Passed = passed
		result.Message = msg
		return result
	case "health":
		output, err := runner.Run(ctx, "docker", "inspect", check.Container, "--format", "{{.State.Health.Status}}")
		if err != nil {
			result.Message = err.Error()
			return result
		}
		value := strings.TrimSpace(output)
		if check.Operator == "exists" {
			if value != "" {
				result.Passed = true
				return result
			}
			result.Message = "health status not found"
			return result
		}
		passed, msg := compareValue(value, check.Operator, check.Value, "string")
		result.Passed = passed
		result.Message = msg
		return result
	case "exitCode":
		output, err := runner.Run(ctx, "docker", "inspect", check.Container, "--format", "{{.State.ExitCode}}")
		if err != nil {
			result.Message = err.Error()
			return result
		}
		actual := strings.TrimSpace(output)
		passed, msg := compareValue(actual, check.Operator, check.Value, "number")
		result.Passed = passed
		result.Message = msg
		return result
	case "ports":
		output, err := runner.Run(ctx, "docker", "port", check.Container)
		if err != nil {
			result.Message = err.Error()
			return result
		}
		passed, msg := compareValue(strings.TrimSpace(output), check.Operator, check.Value, "string")
		result.Passed = passed
		result.Message = msg
		return result
	default:
		result.Message = fmt.Sprintf("unsupported docker container property: %s", check.Property)
		return result
	}
}

func runDockerLogsCheck(ctx context.Context, check scenario.Check) Result {
	result := Result{Name: check.Name}
	if check.Container == "" {
		result.Message = "missing container"
		return result
	}
	args := []string{"logs"}
	if check.Timeout != "" {
		args = append(args, "--since", check.Timeout)
	}
	args = append(args, check.Container)
	output, err := runner.Run(ctx, "docker", args...)
	if err != nil {
		result.Message = err.Error()
		return result
	}
	passed, msg := compareValue(output, check.Operator, check.Value, "string")
	result.Passed = passed
	result.Message = msg
	return result
}

func runDockerfileCheck(check scenario.Check, workDir string) Result {
	result := Result{Name: check.Name}
	if check.Path == "" {
		result.Message = "missing dockerfile path"
		return result
	}
	path := check.Path
	if !filepath.IsAbs(path) {
		path = filepath.Join(workDir, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		result.Message = fmt.Sprintf("read dockerfile: %s", err)
		return result
	}
	content := string(data)
	lines := strings.Split(content, "\n")
	fromCount := 0
	userFound := false
	copyFromFound := false
	firstFrom := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		upper := strings.ToUpper(trimmed)
		if strings.HasPrefix(upper, "FROM ") {
			fromCount++
			if firstFrom == "" {
				firstFrom = strings.TrimSpace(trimmed[5:])
			}
		}
		if strings.HasPrefix(upper, "USER ") {
			userFound = true
		}
		if strings.Contains(upper, "COPY --FROM=") {
			copyFromFound = true
		}
	}

	switch check.Check {
	case "multiStage":
		actual := fromCount > 1
		passed, msg := compareValue(fmt.Sprintf("%t", actual), check.Operator, check.Value, "string")
		result.Passed = passed
		result.Message = msg
		return result
	case "baseImage":
		passed, msg := compareValue(firstFrom, check.Operator, check.Value, "string")
		result.Passed = passed
		result.Message = msg
		return result
	case "copyFrom":
		passed, msg := compareValue(fmt.Sprintf("%t", copyFromFound), check.Operator, check.Value, "string")
		result.Passed = passed
		result.Message = msg
		return result
	case "userInstruction":
		if check.Operator == "exists" {
			if userFound {
				result.Passed = true
				return result
			}
			result.Message = "USER instruction not found"
			return result
		}
		passed, msg := compareValue(fmt.Sprintf("%t", userFound), check.Operator, check.Value, "string")
		result.Passed = passed
		result.Message = msg
		return result
	default:
		result.Message = fmt.Sprintf("unsupported dockerfile check: %s", check.Check)
		return result
	}
}

func compareInt(actual int64, expected int64, operator string) (bool, string) {
	if operator == "" {
		operator = "equals"
	}
	switch operator {
	case "equals":
		if actual == expected {
			return true, ""
		}
		return false, fmt.Sprintf("expected %d, got %d", expected, actual)
	case "notEquals":
		if actual != expected {
			return true, ""
		}
		return false, fmt.Sprintf("expected not %d, got %d", expected, actual)
	case "greaterThan":
		if actual > expected {
			return true, ""
		}
		return false, fmt.Sprintf("expected %d > %d", actual, expected)
	case "lessThan":
		if actual < expected {
			return true, ""
		}
		return false, fmt.Sprintf("expected %d < %d", actual, expected)
	default:
		return false, fmt.Sprintf("unsupported operator: %s", operator)
	}
}

func parseSize(value string) (int64, error) {
	value = strings.TrimSpace(strings.ToUpper(value))
	if value == "" {
		return 0, fmt.Errorf("size is empty")
	}

	multiplier := int64(1)
	switch {
	case strings.HasSuffix(value, "GB"):
		multiplier = 1024 * 1024 * 1024
		value = strings.TrimSuffix(value, "GB")
	case strings.HasSuffix(value, "MB"):
		multiplier = 1024 * 1024
		value = strings.TrimSuffix(value, "MB")
	case strings.HasSuffix(value, "KB"):
		multiplier = 1024
		value = strings.TrimSuffix(value, "KB")
	case strings.HasSuffix(value, "B"):
		multiplier = 1
		value = strings.TrimSuffix(value, "B")
	}

	numeric, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size: %s", value)
	}
	return int64(numeric * float64(multiplier)), nil
}

// runPodLogsCheck searches Kubernetes pod logs
func runPodLogsCheck(ctx context.Context, namespace string, check scenario.Check) Result {
	result := Result{Name: check.Name}
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

	output, err := runner.Run(ctx, "kubectl", args...)
	if err != nil {
		result.Message = err.Error()
		return result
	}

	passed, msg := compareValue(output, check.Operator, check.Value, "string")
	result.Passed = passed
	result.Message = msg
	return result
}

// runKubernetesExecCheck runs a command in a Kubernetes pod
func runKubernetesExecCheck(ctx context.Context, namespace string, check scenario.Check) Result {
	result := Result{Name: check.Name}
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

	output, err := runner.Run(ctx, "kubectl", args...)

	// Check exit code if specified
	if check.ExpectExitCode != nil {
		exitCode := 0
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				exitCode = exitErr.ExitCode()
			} else {
				result.Message = err.Error()
				return result
			}
		}
		if exitCode != *check.ExpectExitCode {
			result.Message = fmt.Sprintf("expected exit code %d, got %d", *check.ExpectExitCode, exitCode)
			return result
		}
	} else if err != nil {
		result.Message = err.Error()
		return result
	}

	// Check output if specified
	if check.ExpectOutput != nil {
		return checkExpectOutput(output, check.ExpectOutput, result)
	}

	result.Passed = true
	return result
}

// runDockerExecCheck runs a command in a Docker container
func runDockerExecCheck(ctx context.Context, check scenario.Check) Result {
	result := Result{Name: check.Name}
	if check.Container == "" || len(check.Command) == 0 {
		result.Message = "missing container or command for exec"
		return result
	}

	args := []string{"exec", check.Container}
	args = append(args, check.Command...)

	output, err := runner.Run(ctx, "docker", args...)

	// Check exit code if specified
	if check.ExpectExitCode != nil {
		exitCode := 0
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				exitCode = exitErr.ExitCode()
			} else {
				result.Message = err.Error()
				return result
			}
		}
		if exitCode != *check.ExpectExitCode {
			result.Message = fmt.Sprintf("expected exit code %d, got %d", *check.ExpectExitCode, exitCode)
			return result
		}
	} else if err != nil {
		result.Message = err.Error()
		return result
	}

	// Check output if specified
	if check.ExpectOutput != nil {
		return checkExpectOutput(output, check.ExpectOutput, result)
	}

	result.Passed = true
	return result
}

// runScriptCheck runs a custom shell script
func runScriptCheck(ctx context.Context, check scenario.Check, workDir string) Result {
	result := Result{Name: check.Name}
	if check.Script == "" {
		result.Message = "missing script"
		return result
	}

	output, err := runner.RunInDir(ctx, workDir, "bash", "-c", check.Script)

	// Check exit code if specified
	if check.ExpectExitCode != nil {
		exitCode := 0
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				exitCode = exitErr.ExitCode()
			} else {
				result.Message = err.Error()
				return result
			}
		}
		if exitCode != *check.ExpectExitCode {
			result.Message = fmt.Sprintf("expected exit code %d, got %d", *check.ExpectExitCode, exitCode)
			return result
		}
	} else if err != nil {
		result.Message = err.Error()
		return result
	}

	// Check output if specified
	if check.ExpectOutput != nil {
		return checkExpectOutput(output, check.ExpectOutput, result)
	}

	result.Passed = true
	return result
}

// runHTTPCheck validates an HTTP endpoint
func runHTTPCheck(ctx context.Context, check scenario.Check) Result {
	result := Result{Name: check.Name}
	if check.URL == "" {
		result.Message = "missing URL"
		return result
	}

	method := check.Method
	if method == "" {
		method = "GET"
	}

	timeout := 10 * time.Second
	if check.Timeout != "" {
		if d, err := time.ParseDuration(check.Timeout); err == nil {
			timeout = d
		}
	}

	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequestWithContext(ctx, method, check.URL, nil)
	if err != nil {
		result.Message = fmt.Sprintf("create request: %s", err)
		return result
	}

	for k, v := range check.Headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		result.Message = fmt.Sprintf("request failed: %s", err)
		return result
	}
	defer resp.Body.Close()

	// Check status code if specified
	if check.ExpectStatus != nil {
		if resp.StatusCode != *check.ExpectStatus {
			result.Message = fmt.Sprintf("expected status %d, got %d", *check.ExpectStatus, resp.StatusCode)
			return result
		}
	}

	// Check body if specified
	if check.ExpectBody != nil {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			result.Message = fmt.Sprintf("read body: %s", err)
			return result
		}
		bodyStr := string(body)

		if check.ExpectBody.Contains != "" {
			if !strings.Contains(bodyStr, check.ExpectBody.Contains) {
				result.Message = fmt.Sprintf("body does not contain: %s", check.ExpectBody.Contains)
				return result
			}
		}
		if check.ExpectBody.NotContains != "" {
			if strings.Contains(bodyStr, check.ExpectBody.NotContains) {
				result.Message = fmt.Sprintf("body contains: %s", check.ExpectBody.NotContains)
				return result
			}
		}
		if check.ExpectBody.Regex != "" {
			matched, err := regexp.MatchString(check.ExpectBody.Regex, bodyStr)
			if err != nil {
				result.Message = fmt.Sprintf("invalid regex: %s", err)
				return result
			}
			if !matched {
				result.Message = fmt.Sprintf("body does not match regex: %s", check.ExpectBody.Regex)
				return result
			}
		}
	}

	result.Passed = true
	return result
}

// runFileCheck validates file existence and content
func runFileCheck(check scenario.Check, workDir string) Result {
	result := Result{Name: check.Name}
	if check.Path == "" {
		result.Message = "missing file path"
		return result
	}

	path := check.Path
	if !filepath.IsAbs(path) {
		path = filepath.Join(workDir, path)
	}

	info, err := os.Stat(path)
	exists := err == nil

	// Check existence
	if check.Exists != nil {
		if exists != *check.Exists {
			if *check.Exists {
				result.Message = fmt.Sprintf("file does not exist: %s", check.Path)
			} else {
				result.Message = fmt.Sprintf("file exists but should not: %s", check.Path)
			}
			return result
		}
		if !*check.Exists {
			result.Passed = true
			return result
		}
	}

	if !exists {
		result.Message = fmt.Sprintf("file not found: %s", check.Path)
		return result
	}

	// If it's a directory and no content check, just verify it exists
	if info.IsDir() {
		if check.Value == nil && check.Operator == "" {
			result.Passed = true
			return result
		}
		result.Message = "cannot check content of directory"
		return result
	}

	// Read file content if value check is needed
	if check.Value != nil || check.Operator != "" {
		content, err := os.ReadFile(path)
		if err != nil {
			result.Message = fmt.Sprintf("read file: %s", err)
			return result
		}
		passed, msg := compareValue(string(content), check.Operator, check.Value, "string")
		result.Passed = passed
		result.Message = msg
		return result
	}

	result.Passed = true
	return result
}

// checkExpectOutput validates command output against expectations
func checkExpectOutput(output string, expect *scenario.ExpectOutput, result Result) Result {
	if expect.Contains != "" {
		if !strings.Contains(output, expect.Contains) {
			result.Message = fmt.Sprintf("output does not contain: %s", expect.Contains)
			return result
		}
	}
	if expect.NotContains != "" {
		if strings.Contains(output, expect.NotContains) {
			result.Message = fmt.Sprintf("output contains: %s", expect.NotContains)
			return result
		}
	}
	if expect.Regex != "" {
		matched, err := regexp.MatchString(expect.Regex, output)
		if err != nil {
			result.Message = fmt.Sprintf("invalid regex: %s", err)
			return result
		}
		if !matched {
			result.Message = fmt.Sprintf("output does not match regex: %s", expect.Regex)
			return result
		}
	}
	result.Passed = true
	return result
}
