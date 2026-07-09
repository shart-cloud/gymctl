package checks

import (
	"errors"
	"fmt"
	"os/exec"

	"gymctl/internal/scenario"
)

func evaluateCommandResult(output string, err error, check scenario.Check, result Result) Result {
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

	if check.ExpectOutput != nil {
		return checkExpectOutput(output, check.ExpectOutput, result)
	}

	result.Passed = true
	return result
}
