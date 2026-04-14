package scenario

import (
	"fmt"
	"strings"
)

type LabSection struct {
	Type     string        `json:"type"`
	Markdown string        `json:"markdown,omitempty"`
	MCQ      *LabMCQPrompt `json:"mcq,omitempty"`
}

type LabMCQPrompt struct {
	ID       string         `json:"id"`
	Question string         `json:"question,omitempty"`
	Options  []LabMCQOption `json:"options,omitempty"`
	Selected []string       `json:"selected,omitempty"`
}

type LabMCQOption struct {
	Letter   string `json:"letter"`
	Text     string `json:"text"`
	Selected bool   `json:"selected"`
}

func ParseLabSections(content []byte) ([]LabSection, error) {
	lines := splitMCQLines(string(content))
	sections := make([]LabSection, 0)
	markdownLines := make([]string, 0)

	flushMarkdown := func() {
		if len(markdownLines) == 0 {
			return
		}
		markdown := strings.TrimSpace(strings.Join(markdownLines, ""))
		if markdown != "" {
			sections = append(sections, LabSection{Type: "markdown", Markdown: markdown})
		}
		markdownLines = markdownLines[:0]
	}

	for i := 0; i < len(lines); i++ {
		open := mcqOpenFencePattern.FindStringSubmatch(strings.TrimRight(lines[i], "\r\n"))
		if open == nil || !isMCQInfoString(strings.TrimSpace(open[2])) {
			markdownLines = append(markdownLines, lines[i])
			continue
		}

		flushMarkdown()

		id, ok := parseMCQID(strings.TrimSpace(open[2]))
		if !ok {
			return nil, fmt.Errorf("mcq block on line %d is missing id=<id>", i+1)
		}

		fence := open[1]
		fenceChar := string(fence[0])
		fenceLen := len(fence)
		blockLines := make([]string, 0)
		closed := false

		for j := i + 1; j < len(lines); j++ {
			currentLine := strings.TrimRight(lines[j], "\r\n")
			if isClosingFence(currentLine, fenceChar, fenceLen) {
				sections = append(sections, LabSection{Type: "mcq", MCQ: parseLabMCQPrompt(id, blockLines)})
				closed = true
				i = j
				break
			}
			blockLines = append(blockLines, lines[j])
		}

		if !closed {
			return nil, fmt.Errorf("mcq block %q is missing a closing fence", id)
		}
	}

	flushMarkdown()
	return sections, nil
}

func parseLabMCQPrompt(id string, lines []string) *LabMCQPrompt {
	prompt := &LabMCQPrompt{ID: id, Options: make([]LabMCQOption, 0)}
	questionLines := make([]string, 0)
	seenOption := false

	for _, line := range lines {
		currentLine := strings.TrimRight(line, "\r\n")
		matches := mcqOptionLinePattern.FindStringSubmatch(currentLine)
		if matches != nil {
			seenOption = true
			letter := OptionLetter(len(prompt.Options))
			selected := strings.EqualFold(matches[2], "x")
			prompt.Options = append(prompt.Options, LabMCQOption{
				Letter:   letter,
				Text:     strings.TrimSpace(matches[3]),
				Selected: selected,
			})
			if selected {
				prompt.Selected = append(prompt.Selected, letter)
			}
			continue
		}

		if seenOption {
			continue
		}

		trimmed := strings.TrimSpace(currentLine)
		if trimmed == "" {
			if len(questionLines) > 0 {
				questionLines = append(questionLines, "")
			}
			continue
		}
		questionLines = append(questionLines, trimmed)
	}

	prompt.Question = strings.TrimSpace(strings.Join(questionLines, "\n"))
	return prompt
}

func SetMCQSelection(content []byte, questionID, letter string) ([]byte, error) {
	lines := splitMCQLines(string(content))
	letter = strings.ToUpper(strings.TrimSpace(letter))
	updated := false

	for i := 0; i < len(lines); i++ {
		open := mcqOpenFencePattern.FindStringSubmatch(strings.TrimRight(lines[i], "\r\n"))
		if open == nil || !isMCQInfoString(strings.TrimSpace(open[2])) {
			continue
		}

		id, ok := parseMCQID(strings.TrimSpace(open[2]))
		if !ok {
			return nil, fmt.Errorf("mcq block on line %d is missing id=<id>", i+1)
		}
		if id != questionID {
			continue
		}

		fence := open[1]
		fenceChar := string(fence[0])
		fenceLen := len(fence)
		optionCount := 0
		closed := false

		for j := i + 1; j < len(lines); j++ {
			currentLine := strings.TrimRight(lines[j], "\r\n")
			if isClosingFence(currentLine, fenceChar, fenceLen) {
				closed = true
				i = j
				break
			}

			matches := mcqOptionLinePattern.FindStringSubmatch(currentLine)
			if matches == nil {
				continue
			}

			currentLetter := OptionLetter(optionCount)
			mark := " "
			if currentLetter == letter {
				mark = "x"
				updated = true
			}
			lines[j] = matches[1] + "- [" + mark + "]" + matches[3]
			optionCount++
		}

		if !closed {
			return nil, fmt.Errorf("mcq block %q is missing a closing fence", id)
		}
		if !updated {
			return nil, fmt.Errorf("option %q not found for mcq %q", letter, questionID)
		}
		return []byte(strings.Join(lines, "")), nil
	}

	return nil, fmt.Errorf("mcq %q not found", questionID)
}
