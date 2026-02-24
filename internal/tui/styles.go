package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

var (
	ColorBrand     = lipgloss.Color("#56D4E6")
	ColorSecondary = lipgloss.Color("#F6C177")
	ColorAccent    = lipgloss.Color("#9CCFD8")
	ColorSuccess   = lipgloss.Color("#8BD49C")
	ColorWarning   = lipgloss.Color("#F2AE49")
	ColorError     = lipgloss.Color("#E26D5C")
	ColorText      = lipgloss.Color("#E8E6E3")
	ColorTextDim   = lipgloss.Color("#7F7B77")
	ColorBg        = lipgloss.Color("#12161B")
	ColorBgSecond  = lipgloss.Color("#1B222A")

	// Header / track with gradient effect
	StyleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorBrand).
			Background(ColorBg)

	StyleTrack = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorBrand).
			Background(ColorBgSecond).
			Padding(0, 1).
			MarginBottom(1)

	// Enhanced status styles with background
	StyleSuccess = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000")).
			Background(ColorSuccess).
			Padding(0, 1)

	StyleWarning = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000")).
			Background(ColorWarning).
			Padding(0, 1)

	StyleError = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFF")).
			Background(ColorError).
			Padding(0, 1)

	StyleDim = lipgloss.NewStyle().
			Foreground(ColorTextDim)

	// Priority with enhanced styling
	StyleP0 = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFF")).
		Background(ColorError).
		Padding(0, 1)

	StyleP1 = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#000")).
		Background(ColorWarning).
		Padding(0, 1)

	StyleP2 = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#000")).
		Background(ColorAccent).
		Padding(0, 1)

	StyleP3 = lipgloss.NewStyle().
		Foreground(ColorTextDim).
		Background(ColorBgSecond).
		Padding(0, 1)

	// Enhanced selection highlight with border
	StyleSelected = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorBg).
			Background(ColorSecondary).
			Padding(0, 1)

	// Enhanced boxes and containers
	StyleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSecondary).
			Background(ColorBgSecond).
			Padding(1, 2)

	StyleTicketBox = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(ColorBrand).
			Background(ColorBg).
			Padding(1, 2).
			MarginBottom(1)

	// Ticket border
	StyleTicketBorder = lipgloss.NewStyle().
				Foreground(ColorBrand)

	// Enhanced body text
	StyleBody = lipgloss.NewStyle().
			Foreground(ColorText)

	// Enhanced footer with better contrast
	StyleKey = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorBrand).
			Background(ColorBgSecond).
			Padding(0, 1)

	StyleFooter = lipgloss.NewStyle().
			Foreground(ColorTextDim)

	// Enhanced difficulty badges
	StyleEasy = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000")).
			Background(ColorSuccess).
			Padding(0, 1)

	StyleMedium = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000")).
			Background(ColorWarning).
			Padding(0, 1)

	StyleHard = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFF")).
			Background(ColorError).
			Padding(0, 1)

	// Enhanced section title with underline accent
	StyleSectionTitle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorSecondary).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(ColorAccent).
				MarginBottom(1)

	// Enhanced progress bar with gradients
	StyleBarFill = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Background(ColorBgSecond)

	StyleBarEmpty = lipgloss.NewStyle().
			Foreground(ColorTextDim).
			Background(ColorBg)

	// New styles for enhanced UI elements
	StyleBadge = lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder())

	StyleCard = lipgloss.NewStyle().
			Background(ColorBgSecond).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBrand).
			Padding(1, 2).
			MarginBottom(1)

	StyleCardTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary)

	StyleCardBody = lipgloss.NewStyle().
			Foreground(ColorText)

	StyleHighlight = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorAccent)
)

func PriorityStyle(priority string) lipgloss.Style {
	switch priority {
	case "P0":
		return StyleP0
	case "P1":
		return StyleP1
	case "P2":
		return StyleP2
	default:
		return StyleP3
	}
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

// Enhanced progress bar with better visual design
func renderProgressBar(current, total, width int) string {
	if total == 0 {
		return StyleBarEmpty.Render("[ No exercises ]")
	}

	percentage := 0
	if total > 0 {
		percentage = (current * 100) / total
	}

	filled := 0
	if total > 0 {
		filled = (current * width) / total
	}

	// Create progress bar with different fill characters
	bar := ""
	for i := 0; i < width; i++ {
		if i < filled {
			if i == filled-1 && filled < width {
				// Partial fill character at the boundary
				bar += StyleBarFill.Render("▶")
			} else {
				bar += StyleBarFill.Render("█")
			}
		} else {
			bar += StyleBarEmpty.Render("░")
		}
	}

	// Add percentage
	percentStr := fmt.Sprintf(" %d%%", percentage)

	return StyleTicketBorder.Render("[") +
		bar +
		StyleTicketBorder.Render("]") +
		StyleHighlight.Render(percentStr)
}

// renderGRIMHeader renders the banner, fitting to the available terminal width.
func renderGRIMHeader(termWidth int) string {
	const minInner = 50
	const maxInner = 75

	inner := termWidth - 4 // 2-char indent + 1 border char each side
	if inner < minInner {
		inner = minInner
	}
	if inner > maxInner {
		inner = maxInner
	}

	pad := strings.Repeat("━", inner)
	line1 := centerText("GRIM-9 // OPS DESK v2", inner)
	line2 := centerText("Incident Sim Lab • Ticket Arcade • Co-op Mode", inner)

	header := fmt.Sprintf("  ┏%s┓\n  ┃%s┃\n  ┃%s┃\n  ┗%s┛", pad, line1, line2, pad)
	return StyleHeader.Render(header)
}

// centerText pads s with spaces so its display width equals width.
// Uses lipgloss.Width so multi-byte Unicode chars are measured correctly.
func centerText(s string, width int) string {
	sWidth := lipgloss.Width(s)
	if sWidth >= width {
		return s
	}
	total := width - sWidth
	left := total / 2
	right := total - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}
