package tui

import (
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
)

// The palette is deliberately near-monochrome: text / dim / faint, plus ONE
// accent reserved for state (active selection, progress fill, the pet's mood,
// success). Failure is the single exception — it gets its own red so "broken"
// always reads as broken. Set GYMCTL_ACCENT=#RRGGBB to recolor the accent.

const (
	defaultAccentDark  = "#F6C177" // amber
	defaultAccentLight = "#9A6B00"
)

var (
	ColorAccent  = lipgloss.Color(defaultAccentDark)
	ColorError   = lipgloss.Color("#E26D5C")
	ColorText    = lipgloss.Color("#D7D5D1")
	ColorTextDim = lipgloss.Color("#8A8782")
	ColorFaint   = lipgloss.Color("#565450")
)

// accentHex resolves the accent hex: GYMCTL_ACCENT env override wins, else the
// per-theme default.
func accentHex(isDark bool) string {
	if hex := strings.TrimSpace(os.Getenv("GYMCTL_ACCENT")); hex != "" {
		return hex
	}
	if isDark {
		return defaultAccentDark
	}
	return defaultAccentLight
}

var (
	// Header / breadcrumb / track label.
	StyleHeader lipgloss.Style
	StyleTrack  lipgloss.Style

	// State styles. Success folds into the accent; only error keeps its own hue.
	StyleSuccess lipgloss.Style
	StyleWarning lipgloss.Style
	StyleError   lipgloss.Style
	StyleDim     lipgloss.Style

	// Selected list row.
	StyleSelected lipgloss.Style

	// Thin rules / brief chrome.
	StyleTicketBorder lipgloss.Style

	StyleBody lipgloss.Style

	StyleKey    lipgloss.Style
	StyleFooter lipgloss.Style

	// Difficulty — plain text weights, no badges.
	StyleEasy   lipgloss.Style
	StyleMedium lipgloss.Style
	StyleHard   lipgloss.Style

	StyleSectionTitle lipgloss.Style

	StyleBarFill  lipgloss.Style
	StyleBarEmpty lipgloss.Style

	// Panels are flat now: a dim title over indented body, no border, no fill.
	StyleCard      lipgloss.Style
	StyleCardTitle lipgloss.Style
	StyleCardBody  lipgloss.Style
	StyleHighlight lipgloss.Style
)

func init() {
	buildStyles()
}

// buildStyles reconstructs all Style* vars from the current Color* values.
func buildStyles() {
	accent := lipgloss.NewStyle().Foreground(ColorAccent)
	dim := lipgloss.NewStyle().Foreground(ColorTextDim)
	faint := lipgloss.NewStyle().Foreground(ColorFaint)
	text := lipgloss.NewStyle().Foreground(ColorText)

	StyleHeader = dim
	StyleTrack = accent.Bold(true)

	StyleSuccess = accent
	StyleWarning = accent
	StyleError = lipgloss.NewStyle().Foreground(ColorError)
	StyleDim = faint

	StyleSelected = accent.Bold(true)

	StyleTicketBorder = faint

	StyleBody = text

	StyleKey = accent.Bold(true)
	StyleFooter = dim

	StyleEasy = faint
	StyleMedium = dim
	StyleHard = accent

	StyleSectionTitle = dim.Bold(true)

	StyleBarFill = accent
	StyleBarEmpty = faint

	StyleCard = lipgloss.NewStyle()
	StyleCardTitle = dim
	StyleCardBody = text
	StyleHighlight = accent
}

func DifficultyStyle(difficulty string) lipgloss.Style {
	switch difficulty {
	case "easy":
		return StyleEasy
	case "medium":
		return StyleMedium
	case "hard":
		return StyleHard
	default:
		return StyleBody
	}
}

// renderProgressBar draws a flat accent/faint bar: ▓ for filled, ░ for empty.
func renderProgressBar(current, total, width int) string {
	if total == 0 {
		return StyleBarEmpty.Render("no exercises")
	}

	percentage := (current * 100) / total
	filled := (current * width) / total

	var bar strings.Builder
	for i := 0; i < width; i++ {
		if i < filled {
			bar.WriteString(StyleBarFill.Render("▓"))
		} else {
			bar.WriteString(StyleBarEmpty.Render("░"))
		}
	}
	return bar.String() + StyleHighlight.Render(fmt.Sprintf(" %d%%", percentage))
}

// renderHomeHeader renders the minimal home breadcrumb — no banner, no box.
func renderHomeHeader() string {
	return StyleHeader.Render("gymctl") +
		StyleTicketBorder.Render(" · ") +
		StyleHeader.Render("jerry's gym")
}

func applyTheme(isDark bool) {
	ColorAccent = lipgloss.Color(accentHex(isDark))
	if isDark {
		ColorText = lipgloss.Color("#D7D5D1")
		ColorTextDim = lipgloss.Color("#8A8782")
		ColorFaint = lipgloss.Color("#565450")
	} else {
		ColorText = lipgloss.Color("#20242B")
		ColorTextDim = lipgloss.Color("#6F7480")
		ColorFaint = lipgloss.Color("#A9AEB6")
	}
	buildStyles()
}
