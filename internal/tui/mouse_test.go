package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestClickedListIndex(t *testing.T) {
	content := strings.Join([]string{
		"title",
		"  1. first",
		"\x1b[32m> 2. second\x1b[0m",
	}, "\n")

	idx, ok := clickedListIndex(content, 2)
	if !ok {
		t.Fatalf("expected hit")
	}
	if idx != 1 {
		t.Fatalf("expected index 1, got %d", idx)
	}
}

func TestClickedListIndexMiss(t *testing.T) {
	content := "header\nno list row here"

	if _, ok := clickedListIndex(content, 1); ok {
		t.Fatalf("expected miss")
	}
	if _, ok := clickedListIndex(content, -1); ok {
		t.Fatalf("expected miss for negative y")
	}
}

func TestClickedFooterKey(t *testing.T) {
	line := "  [Enter] Open   [q] Quit"
	content := "x\n" + line
	x := strings.Index(line, "[q]") + 1

	token, ok := clickedFooterKey(content, x, 1)
	if !ok {
		t.Fatalf("expected footer token hit")
	}
	if token != "[q]" {
		t.Fatalf("expected [q], got %q", token)
	}
}

func TestMapFooterTokenToKey(t *testing.T) {
	key, ok := mapFooterTokenToKey(viewExerciseDetail, "[Enter]")
	if !ok || key != "enter" {
		t.Fatalf("expected enter mapping, got key=%q ok=%v", key, ok)
	}

	if _, ok := mapFooterTokenToKey(viewExerciseDetail, "[unknown]"); ok {
		t.Fatalf("expected unknown token to miss")
	}
}

func TestOnMouseTrackRowClickEmitsMessage(t *testing.T) {
	m := Model{view: viewTrackSelect}
	content := "title\n  1. track-one\n  2. track-two"

	h := m.onMouse(content)
	if h == nil {
		t.Fatalf("expected handler")
	}
	cmd := h(tea.MouseClickMsg{X: 3, Y: 1, Button: tea.MouseLeft})
	if cmd == nil {
		t.Fatalf("expected command")
	}

	msg := cmd()
	clicked, ok := msg.(trackClickedMsg)
	if !ok {
		t.Fatalf("expected trackClickedMsg, got %T", msg)
	}
	if clicked.index != 0 {
		t.Fatalf("expected index 0, got %d", clicked.index)
	}
}

func TestOnMouseFooterClickEmitsAction(t *testing.T) {
	m := Model{view: viewExerciseDetail}
	line := "  [q] Quit"
	h := m.onMouse(line)
	if h == nil {
		t.Fatalf("expected handler")
	}

	x := strings.Index(line, "[q]") + 1
	cmd := h(tea.MouseClickMsg{X: x, Y: 0, Button: tea.MouseLeft})
	if cmd == nil {
		t.Fatalf("expected command")
	}

	msg := cmd()
	action, ok := msg.(footerActionMsg)
	if !ok {
		t.Fatalf("expected footerActionMsg, got %T", msg)
	}
	if action.key != "q" {
		t.Fatalf("expected q action, got %q", action.key)
	}
}
