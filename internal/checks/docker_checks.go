package checks

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gymctl/internal/scenario"
)

func runDockerImageCheckWithRunner(ctx context.Context, commands checkCommandRunner, check scenario.Check) Result {
	result := newResult(check)
	if check.Image == "" || check.Property == "" {
		result.Message = "missing image or property"
		return result
	}

	switch check.Property {
	case "size":
		output, err := commands.Run(ctx, "docker", "image", "inspect", check.Image, "--format", "{{.Size}}")
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
		output, err := commands.Run(ctx, "docker", "image", "inspect", check.Image, "--format", "{{len .RootFS.Layers}}")
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
		output, err := commands.Run(ctx, "docker", "image", "inspect", check.Image, "--format", "{{.ContainerConfig.Image}}")
		if err != nil {
			result.Message = err.Error()
			return result
		}
		passed, msg := compareValue(strings.TrimSpace(output), check.Operator, check.Value, "string")
		result.Passed = passed
		result.Message = msg
		return result
	case "labels":
		output, err := commands.Run(ctx, "docker", "image", "inspect", check.Image, "--format", "{{.Config.Labels}}")
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

func runDockerContainerCheckWithRunner(ctx context.Context, commands checkCommandRunner, check scenario.Check) Result {
	result := newResult(check)
	if check.Container == "" || check.Property == "" {
		result.Message = "missing container or property"
		return result
	}

	switch check.Property {
	case "state":
		output, err := commands.Run(ctx, "docker", "inspect", check.Container, "--format", "{{.State.Status}}")
		if err != nil {
			result.Message = err.Error()
			return result
		}
		passed, msg := compareValue(strings.TrimSpace(output), check.Operator, check.Value, "string")
		result.Passed = passed
		result.Message = msg
		return result
	case "health":
		output, err := commands.Run(ctx, "docker", "inspect", check.Container, "--format", "{{.State.Health.Status}}")
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
		output, err := commands.Run(ctx, "docker", "inspect", check.Container, "--format", "{{.State.ExitCode}}")
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
		output, err := commands.Run(ctx, "docker", "port", check.Container)
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

func runDockerLogsCheckWithRunner(ctx context.Context, commands checkCommandRunner, check scenario.Check) Result {
	result := newResult(check)
	if check.Container == "" {
		result.Message = "missing container"
		return result
	}
	args := []string{"logs"}
	if check.Timeout != "" {
		args = append(args, "--since", check.Timeout)
	}
	args = append(args, check.Container)
	output, err := commands.Run(ctx, "docker", args...)
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
	result := newResult(check)
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
	lastFrom := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		upper := strings.ToUpper(trimmed)
		if strings.HasPrefix(upper, "FROM ") {
			fromCount++
			fromImage := strings.TrimSpace(trimmed[5:])
			if firstFrom == "" {
				firstFrom = fromImage
			}
			lastFrom = fromImage
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
		passed, msg := compareValue(lastFrom, check.Operator, check.Value, "string")
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

func runDockerExecCheckWithRunner(ctx context.Context, commands checkCommandRunner, check scenario.Check) Result {
	result := newResult(check)
	if check.Container == "" || len(check.Command) == 0 {
		result.Message = "missing container or command for exec"
		return result
	}

	args := []string{"exec", check.Container}
	args = append(args, check.Command...)

	output, err := commands.Run(ctx, "docker", args...)
	return evaluateCommandResult(output, err, check, result)
}
