package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

func validateOutputFormat() error {
	switch outputFormat {
	case "table", "json":
		return nil
	default:
		return fmt.Errorf("invalid output format %q (supported: table, json)", outputFormat)
	}
}

func isJSONOutput() bool {
	return outputFormat == "json"
}

func writeJSON(out io.Writer, value any) error {
	encoder := json.NewEncoder(out)
	return encoder.Encode(value)
}

func jsonStatus(status string) string {
	switch strings.ToLower(status) {
	case "completed":
		return "completed"
	case "in_progress", "started":
		return "in-progress"
	default:
		return "not-started"
	}
}

func isInProgressStatus(status string) bool {
	switch strings.ToLower(status) {
	case "in_progress", "started":
		return true
	default:
		return false
	}
}
