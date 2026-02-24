package tui

import (
	"regexp"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
)

type trackClickedMsg struct{ index int }
type exerciseClickedMsg struct{ index int }
type footerActionMsg struct{ key string }

var (
	listRowPattern  = regexp.MustCompile(`^\s*>?\s*(\d+)\.\s+`)
	ansiCodePattern = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)
)

func (m Model) handleMouseWheel(msg tea.MouseWheelMsg) (tea.Model, tea.Cmd) {
	delta := 0
	if msg.Button == tea.MouseWheelUp {
		delta = -1
	}
	if msg.Button == tea.MouseWheelDown {
		delta = 1
	}
	if delta == 0 {
		return m, nil
	}

	switch m.view {
	case viewTrackSelect:
		m.trackCursor += delta
		if m.trackCursor < 0 {
			m.trackCursor = 0
		}
		if max := len(m.tracks) - 1; m.trackCursor > max {
			m.trackCursor = max
		}
	case viewExerciseList:
		exercises := m.trackExercises()
		if len(exercises) == 0 {
			return m, nil
		}
		m.exerciseCursor += delta
		if m.exerciseCursor < 0 {
			m.exerciseCursor = 0
		}
		if max := len(exercises) - 1; m.exerciseCursor > max {
			m.exerciseCursor = max
		}
		visRows := m.visibleListRows()
		if m.exerciseCursor < m.listScroll {
			m.listScroll = m.exerciseCursor
		}
		if m.exerciseCursor >= m.listScroll+visRows {
			m.listScroll = m.exerciseCursor - visRows + 1
		}
	case viewHintPeek:
		lines := strings.Split(m.hintContent, "\n")
		maxScroll := len(lines) - m.visibleHintRows()
		if maxScroll < 0 {
			maxScroll = 0
		}
		m.hintScroll += delta
		if m.hintScroll < 0 {
			m.hintScroll = 0
		}
		if m.hintScroll > maxScroll {
			m.hintScroll = maxScroll
		}
	}

	return m, nil
}

func (m Model) onMouse(content string) func(msg tea.MouseMsg) tea.Cmd {
	if m.view == viewHome {
		return nil
	}

	view := m.view
	return func(msg tea.MouseMsg) tea.Cmd {
		click, ok := msg.(tea.MouseClickMsg)
		if !ok || click.Button != tea.MouseLeft {
			return nil
		}

		if view == viewTrackSelect || view == viewExerciseList {
			index, ok := clickedListIndex(content, click.Y)
			if ok {
				switch view {
				case viewTrackSelect:
					return func() tea.Msg { return trackClickedMsg{index: index} }
				case viewExerciseList:
					return func() tea.Msg { return exerciseClickedMsg{index: index} }
				}
			}
		}

		key, ok := clickedFooterKey(content, click.X, click.Y)
		if !ok {
			return nil
		}
		mapped, ok := mapFooterTokenToKey(view, key)
		if !ok {
			return nil
		}
		return func() tea.Msg { return footerActionMsg{key: mapped} }
	}
}

func clickedListIndex(content string, y int) (int, bool) {
	if y < 0 {
		return 0, false
	}

	lines := strings.Split(content, "\n")
	if y >= len(lines) {
		return 0, false
	}

	line := stripANSIEscape(lines[y])
	matches := listRowPattern.FindStringSubmatch(line)
	if len(matches) != 2 {
		return 0, false
	}

	n, err := strconv.Atoi(matches[1])
	if err != nil || n <= 0 {
		return 0, false
	}

	return n - 1, true
}

func stripANSIEscape(s string) string {
	return ansiCodePattern.ReplaceAllString(s, "")
}

func clickedFooterKey(content string, x, y int) (string, bool) {
	if x < 0 || y < 0 {
		return "", false
	}
	lines := strings.Split(content, "\n")
	if y >= len(lines) {
		return "", false
	}
	line := stripANSIEscape(lines[y])

	tokenStart := -1
	for i, r := range line {
		if r == '[' {
			tokenStart = i
			continue
		}
		if r == ']' && tokenStart >= 0 {
			tokenEnd := i
			if x >= tokenStart && x <= tokenEnd {
				return line[tokenStart : tokenEnd+1], true
			}
			tokenStart = -1
		}
	}

	return "", false
}

func mapFooterTokenToKey(view viewState, token string) (string, bool) {
	switch token {
	case "[Enter]":
		return "enter", true
	case "[b]":
		return "b", true
	case "[q]":
		return "q", true
	case "[s]":
		return "s", true
	case "[r]":
		return "r", true
	case "[h]":
		return "h", true
	case "[c]":
		return "c", true
	case "[e]":
		return "e", true
	case "[k]":
		return "k", true
	case "[b/Esc]":
		return "b", true
	case "[Esc]":
		return "esc", true
	}

	if view == viewStartConfirm && token == "[↑↓]" {
		return "", false
	}
	return "", false
}
