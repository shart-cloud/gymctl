package scenario

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

type MCQBlock struct {
	ID              string
	StartLine       int
	OptionCount     int
	SelectedLetters []string
}

var (
	mcqOpenFencePattern  = regexp.MustCompile(`^ {0,3}([` + "`" + `~]{3,})(.*)$`)
	mcqOptionLinePattern = regexp.MustCompile(`^(\s*)- \[([ xX])\](.*)$`)
	mcqIDPattern         = regexp.MustCompile(`(^|\s)id=([^\s]+)`)
)

func ParseMCQMarkdown(content []byte) ([]MCQBlock, error) {
	blocks, _, err := parseMCQMarkdown(content, false)
	if err != nil {
		return nil, err
	}
	return blocks, nil
}

func ParseAndClearMCQMarkdownSelections(content []byte) ([]MCQBlock, []byte, error) {
	return parseMCQMarkdown(content, true)
}

func HasMCQBlocks(content []byte) bool {
	lines := splitMCQLines(string(content))
	for _, line := range lines {
		open := mcqOpenFencePattern.FindStringSubmatch(strings.TrimRight(line, "\r\n"))
		if open == nil {
			continue
		}
		if isMCQInfoString(strings.TrimSpace(open[2])) {
			return true
		}
	}
	return false
}

func ContainsCheckedAnswerMarker(content []byte) bool {
	return bytes.Contains(content, []byte("[x]")) || bytes.Contains(content, []byte("[X]"))
}

func HashMCQAnswer(id, letter string) string {
	sum := sha256.Sum256([]byte(id + ":" + letter))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func OptionLetter(idx int) string {
	idx++
	letters := ""
	for idx > 0 {
		idx--
		letters = string(rune('A'+(idx%26))) + letters
		idx /= 26
	}
	return letters
}

func parseMCQMarkdown(content []byte, clearSelections bool) ([]MCQBlock, []byte, error) {
	lines := splitMCQLines(string(content))
	rewritten := append([]string(nil), lines...)
	blocks := []MCQBlock{}

	for i := 0; i < len(lines); i++ {
		open := mcqOpenFencePattern.FindStringSubmatch(strings.TrimRight(lines[i], "\r\n"))
		if open == nil {
			continue
		}

		fence := open[1]
		fenceChar := string(fence[0])
		fenceLen := len(fence)
		info := strings.TrimSpace(open[2])
		if !isMCQInfoString(info) {
			continue
		}

		id, ok := parseMCQID(info)
		if !ok {
			return nil, nil, fmt.Errorf("mcq block on line %d is missing id=<id>", i+1)
		}

		optionCount := 0
		selectedLetters := []string{}
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
			optionCount++
			if strings.EqualFold(matches[2], "x") {
				selectedLetters = append(selectedLetters, OptionLetter(optionCount-1))
				if clearSelections {
					rewritten[j] = matches[1] + "- [ ]" + matches[3]
				}
			}
		}

		if !closed {
			return nil, nil, fmt.Errorf("mcq block %q is missing a closing fence", id)
		}

		blocks = append(blocks, MCQBlock{
			ID:              id,
			StartLine:       i + 1,
			OptionCount:     optionCount,
			SelectedLetters: selectedLetters,
		})
	}

	return blocks, []byte(strings.Join(rewritten, "")), nil
}

func splitMCQLines(content string) []string {
	if content == "" {
		return []string{}
	}
	parts := strings.SplitAfter(content, "\n")
	if !strings.HasSuffix(content, "\n") {
		return parts
	}
	return parts[:len(parts)-1]
}

func isMCQInfoString(info string) bool {
	fields := strings.Fields(info)
	if len(fields) == 0 || fields[0] != "mcq" {
		return false
	}
	return true
}

func parseMCQID(info string) (string, bool) {
	matches := mcqIDPattern.FindStringSubmatch(info)
	if len(matches) != 3 {
		return "", false
	}
	return matches[2], true
}

func isClosingFence(line, fenceChar string, fenceLen int) bool {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, fenceChar) {
		return false
	}
	count := 0
	for count < len(trimmed) && trimmed[count] == fenceChar[0] {
		count++
	}
	if count < fenceLen {
		return false
	}
	return strings.TrimSpace(trimmed[count:]) == ""
}
