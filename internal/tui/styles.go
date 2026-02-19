package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Enhanced color palette
	ColorBrand     = lipgloss.Color("#00E5FF") // Electric cyan
	ColorSecondary = lipgloss.Color("#BB86FC") // Electric purple
	ColorAccent    = lipgloss.Color("#03DAC5") // Teal accent
	ColorSuccess   = lipgloss.Color("#4CAF50") // Material green
	ColorWarning   = lipgloss.Color("#FF9800") // Material orange
	ColorError     = lipgloss.Color("#F44336") // Material red
	ColorText      = lipgloss.Color("#E1E1E1") // Light gray text
	ColorTextDim   = lipgloss.Color("#666666") // Dim text
	ColorBg        = lipgloss.Color("#1A1A1A") // Dark background
	ColorBgSecond  = lipgloss.Color("#2A2A2A") // Slightly lighter background

	// Header / track with gradient effect
	StyleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorBrand).
			Background(lipgloss.AdaptiveColor{Light: "#F0F0F0", Dark: "#1A1A1A"})

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
			Foreground(lipgloss.Color("#000000")).
			Background(ColorBrand).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSecondary).
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
		BorderForeground(ColorTextDim).
		Padding(1, 2).
		MarginBottom(1)

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

// New function for creating enhanced visual elements
func renderStatusBadge(status string) string {
	switch status {
	case "completed":
		return StyleSuccess.Render(" ✓ DONE ")
	case "in_progress":
		return StyleWarning.Render(" ◉ ACTIVE ")
	case "locked":
		return StyleError.Render(" 🔒 LOCKED ")
	default:
		return StyleDim.Render(" ○ TODO ")
	}
}

func renderDifficultyBadge(difficulty string) string {
	switch difficulty {
	case "easy":
		return StyleEasy.Render(" EASY ")
	case "medium", "intermediate":
		return StyleMedium.Render(" MEDIUM ")
	case "hard", "difficult":
		return StyleHard.Render(" HARD ")
	default:
		return StyleBody.Render(" " + strings.ToUpper(difficulty) + " ")
	}
}

// Enhanced GRIM header with ASCII art
func renderGRIMHeader() string {
	header := `
  ╔═══════════════════════════════════════════════════════════════════════╗
  ║                     ▄████  ██▀███   ██▓ ███▄ ▄███▓                    ║
  ║                    ██▒ ▀█▒▓██ ▒ ██▒▓██▒▓██▒▀█▀ ██▒                    ║
  ║                   ▒██░▄▄▄░▓██ ░▄█ ▒▒██▒▓██    ▓██░                    ║
  ║                   ░▓█  ██▓▒██▀▀█▄  ░██░▒██    ▒██                     ║
  ║                   ░▒▓███▀▒░██▓ ▒██▒░██░▒██▒   ░██▒                    ║
  ║                    ░▒   ▒ ░ ▒▓ ░▒▓░░▓  ░ ▒░   ░  ░                    ║
  ║                     ░   ░   ░▒ ░ ▒░ ▒ ░░  ░      ░                    ║
  ║                   ░ ░   ░   ░░   ░  ▒ ░░      ░                       ║
  ║                         ░    ░      ░         ░                       ║
  ║                                                                       ║
  ║             🤖 GRIM-9 ONLINE — Autonomous Incident Routing            ║
  ║                          Routing Jerry's Chaos                       ║
  ╚═══════════════════════════════════════════════════════════════════════╝`

	return StyleHeader.Render(header)
}
