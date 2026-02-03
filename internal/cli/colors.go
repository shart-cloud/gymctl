package cli

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

var (
	// Status colors
	ColorSuccess = color.New(color.FgGreen, color.Bold)
	ColorError   = color.New(color.FgRed, color.Bold)
	ColorWarning = color.New(color.FgYellow, color.Bold)
	ColorInfo    = color.New(color.FgCyan)

	// Text styles
	ColorBold     = color.New(color.Bold)
	ColorDim      = color.New(color.Faint)
	ColorHeader   = color.New(color.FgWhite, color.Bold, color.Underline)

	// Specific element colors
	ColorTrack      = color.New(color.FgMagenta, color.Bold)
	ColorExercise   = color.New(color.FgWhite, color.Bold)
	ColorDifficulty = color.New(color.FgYellow)
	ColorTime       = color.New(color.FgBlue)
	ColorProgress   = color.New(color.FgGreen)

	// Check status
	ColorPass = color.New(color.FgGreen)
	ColorFail = color.New(color.FgRed)

	// Icons
	IconSuccess = "✓"
	IconFail    = "✗"
	IconPending = "○"
	IconRunning = "⚡"
	IconWarning = "⚠"
	IconInfo    = "ℹ"
	IconHint    = "💡"
)

// DifficultyColor returns appropriate color for difficulty level
func DifficultyColor(difficulty string) *color.Color {
	switch strings.ToLower(difficulty) {
	case "easy":
		return color.New(color.FgGreen)
	case "medium":
		return color.New(color.FgYellow)
	case "hard":
		return color.New(color.FgRed)
	default:
		return color.New(color.FgWhite)
	}
}

// DifficultyBadge returns a colored badge for difficulty
func DifficultyBadge(difficulty string) string {
	switch strings.ToLower(difficulty) {
	case "easy":
		return color.GreenString("● Easy")
	case "medium":
		return color.YellowString("● Medium")
	case "hard":
		return color.RedString("● Hard")
	default:
		return difficulty
	}
}

// FormatStatus formats exercise status with appropriate icon and color
func FormatStatus(status string) string {
	switch status {
	case "completed":
		return ColorSuccess.Sprint(IconSuccess)
	case "started", "in_progress":
		return ColorWarning.Sprint(IconRunning)
	default:
		return ColorDim.Sprint(IconPending)
	}
}

// ProgressBar creates a simple ASCII progress bar
func ProgressBar(current, total int, width int) string {
	if total == 0 {
		return ""
	}

	percentage := float64(current) / float64(total)
	filled := int(percentage * float64(width))

	bar := "["
	for i := 0; i < width; i++ {
		if i < filled {
			bar += ColorProgress.Sprint("█")
		} else {
			bar += ColorDim.Sprint("░")
		}
	}
	bar += fmt.Sprintf("] %d/%d", current, total)

	return bar
}

// FormatCheckResult formats check results with color
func FormatCheckResult(name string, passed bool, message string) string {
	status := ColorFail.Sprintf("[%s]", IconFail)
	if passed {
		status = ColorSuccess.Sprintf("[%s]", IconSuccess)
	}

	result := fmt.Sprintf("%s %s", status, name)
	if message != "" {
		result += ColorDim.Sprintf(" - %s", message)
	}

	return result
}

// DisableColors disables all colors (useful for non-tty output)
func DisableColors() {
	color.NoColor = true
}