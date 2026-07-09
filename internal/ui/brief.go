package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"

	"gymctl/internal/scenario"
)

const ruleWidth = 68 // width of the thin horizontal rule under the title

var (
	titleColor = color.New(color.Bold)
	ruleColor  = color.New(color.Faint)
	bodyColor  = color.New()
)

// wrapText reflows text to width chars per line. Paragraphs are separated by
// blank lines; soft newlines within a paragraph (author hard-wrapping) are
// collapsed to spaces so wrapping is clean regardless of source formatting.
func wrapText(text string, width int) []string {
	var lines []string
	for i, paragraph := range strings.Split(strings.TrimRight(text, "\n"), "\n\n") {
		paragraph = strings.TrimSpace(strings.ReplaceAll(paragraph, "\n", " "))
		if i > 0 {
			lines = append(lines, "")
		}
		if paragraph == "" {
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

// RenderBrief writes a flat, minimal exercise brief to w: a bold title, a thin
// rule, and the word-wrapped description. No box-art — the "codex/claude-code"
// look. If brief carries a summary/description they win, otherwise it falls back
// to the exercise name and description.
func RenderBrief(w io.Writer, brief *scenario.Brief, exerciseName, exerciseDesc string) {
	title := exerciseName
	description := exerciseDesc
	if brief != nil {
		if brief.Summary != "" {
			title = brief.Summary
		}
		if brief.Description != "" {
			description = brief.Description
		}
	}

	fmt.Fprintln(w, "  "+titleColor.Sprint(strings.ToLower(title)))
	fmt.Fprintln(w, "  "+ruleColor.Sprint(strings.Repeat("─", ruleWidth)))
	for _, line := range wrapText(description, ruleWidth) {
		if line == "" {
			fmt.Fprintln(w)
		} else {
			fmt.Fprintln(w, "  "+bodyColor.Sprint(line))
		}
	}
}
