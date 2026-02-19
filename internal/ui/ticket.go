package ui

import (
	"fmt"
	"hash/fnv"
	"io"
	"regexp"
	"strings"

	"github.com/fatih/color"

	"gymctl/internal/scenario"
)

const (
	boxWidth  = 70 // total width including │ borders
	textWidth = 64 // available chars for word-wrapped body text
)

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// visibleLen returns the display width of s, ignoring ANSI escape codes.
func visibleLen(s string) int {
	return len(ansiEscape.ReplaceAllString(s, ""))
}

// boxLine wraps content in a │ bordered line padded to boxWidth.
// Left padding is 2 spaces; right side is filled with spaces.
func boxLine(content string) string {
	innerWidth := boxWidth - 2 // 68 chars between │ borders
	leftPad := 2
	available := innerWidth - leftPad // 66 chars for content + right padding
	vlen := visibleLen(content)
	rightPad := available - vlen
	if rightPad < 0 {
		rightPad = 0
	}
	return "│" + strings.Repeat(" ", leftPad) + content + strings.Repeat(" ", rightPad) + "│"
}

// emptyLine returns a blank interior line with borders.
func emptyLine() string {
	return "│" + strings.Repeat(" ", boxWidth-2) + "│"
}

// wrapText word-wraps text to fit within width chars per line.
// Blank lines in the input are preserved as empty entries.
func wrapText(text string, width int) []string {
	var lines []string
	for _, paragraph := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			lines = append(lines, "")
			continue
		}
		words := strings.Fields(paragraph)
		current := ""
		for _, word := range words {
			if current == "" {
				current = word
			} else if len(current)+1+len(word) <= width {
				current += " " + word
			} else {
				lines = append(lines, current)
				current = word
			}
		}
		if current != "" {
			lines = append(lines, current)
		}
	}
	return lines
}

// priorityColor returns the color for a GRIM priority level.
func priorityColor(priority string) *color.Color {
	switch priority {
	case "P0":
		return color.New(color.FgRed, color.Bold)
	case "P1":
		return color.New(color.FgYellow, color.Bold)
	case "P2":
		return color.New(color.FgCyan)
	default: // P3 and unknown
		return color.New(color.Faint)
	}
}

// syntheticTicket generates a deterministic ticket ID from an exercise name.
func syntheticTicket(exerciseName string) string {
	h := fnv.New32a()
	h.Write([]byte(exerciseName))
	return fmt.Sprintf("TASK-%04d", h.Sum32()%9000+1000)
}

// RenderGRIMTicket writes a box-drawn GRIM-9 ticket to w.
// If tasking is nil a generic ticket is generated from exerciseName and exerciseDesc.
func RenderGRIMTicket(w io.Writer, tasking *scenario.GRIMTasking, exerciseName, exerciseDesc string) {
	var ticket, priority, summary, description, reporter string

	if tasking != nil {
		ticket = tasking.Ticket
		priority = tasking.Priority
		summary = tasking.Summary
		description = tasking.Description
		reporter = tasking.Reporter
	} else {
		ticket = syntheticTicket(exerciseName)
		priority = "P2"
		summary = exerciseName
		description = exerciseDesc
		reporter = "GRIM-9"
	}
	if reporter == "" {
		reporter = "GRIM-9"
	}

	// Top border: ┌─ GRIM-9 ──...──┐
	label := " GRIM-9 "
	dashCount := boxWidth - 2 - 1 - len(label) // -2 for ┌┐, -1 for leading ─
	if dashCount < 0 {
		dashCount = 0
	}
	fmt.Fprintln(w, "┌─"+label+strings.Repeat("─", dashCount)+"┐")

	// Header line: ticket · priority — SUMMARY
	pColor := priorityColor(priority)
	priorityStr := pColor.Sprint(priority)
	headerText := ticket + "  ·  " + priorityStr + " — " + strings.ToUpper(summary)
	fmt.Fprintln(w, boxLine(headerText))

	// Separator
	fmt.Fprintln(w, "├"+strings.Repeat("─", boxWidth-2)+"┤")

	// Empty line
	fmt.Fprintln(w, emptyLine())

	// Body: word-wrapped description
	for _, line := range wrapText(description, textWidth) {
		if line == "" {
			fmt.Fprintln(w, emptyLine())
		} else {
			fmt.Fprintln(w, boxLine(line))
		}
	}

	// Empty line
	fmt.Fprintln(w, emptyLine())

	// Footer: reporter and assignee
	footer := "Reporter: " + reporter + "  ·  Assignee: You (by elimination)"
	fmt.Fprintln(w, boxLine(footer))

	// Bottom border
	fmt.Fprintln(w, "└"+strings.Repeat("─", boxWidth-2)+"┘")
}
