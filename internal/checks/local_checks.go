package checks

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gymctl/internal/scenario"
)

func runScriptCheck(ctx context.Context, check scenario.Check, workDir string) Result {
	return runScriptCheckWithRunner(ctx, defaultCheckCommandRunner{}, check, workDir)
}

func runScriptCheckWithRunner(ctx context.Context, commands checkCommandRunner, check scenario.Check, workDir string) Result {
	result := newResult(check)
	if check.Script == "" {
		result.Message = "missing script"
		return result
	}

	output, err := commands.RunInDir(ctx, workDir, "bash", "-c", check.Script)
	return evaluateCommandResult(output, err, check, result)
}

func runHTTPCheck(ctx context.Context, check scenario.Check) Result {
	result := newResult(check)
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

	if check.ExpectStatus != nil {
		if resp.StatusCode != *check.ExpectStatus {
			result.Message = fmt.Sprintf("expected status %d, got %d", *check.ExpectStatus, resp.StatusCode)
			return result
		}
	}

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

func runFileCheck(check scenario.Check, workDir string) Result {
	result := newResult(check)
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

	if info.IsDir() {
		if check.Value == nil && check.Operator == "" {
			result.Passed = true
			return result
		}
		result.Message = "cannot check content of directory"
		return result
	}

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
