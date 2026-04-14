package scenario

import "testing"

func TestParseLabSections(t *testing.T) {
	content := []byte("# Title\n\nIntro paragraph.\n\n```mcq id=q1\nWhat does ssh do?\n- [ ] Opens a browser\n- [x] Connects to a remote shell\n- [ ] Deletes files\n```\n\n## Next\nMore practice.\n")

	sections, err := ParseLabSections(content)
	if err != nil {
		t.Fatalf("ParseLabSections() error = %v", err)
	}
	if len(sections) != 3 {
		t.Fatalf("expected 3 sections, got %d", len(sections))
	}

	if sections[0].Type != "markdown" || sections[0].Markdown == "" {
		t.Fatalf("unexpected first section: %+v", sections[0])
	}
	if sections[1].Type != "mcq" || sections[1].MCQ == nil {
		t.Fatalf("unexpected second section: %+v", sections[1])
	}
	if sections[1].MCQ.ID != "q1" {
		t.Fatalf("expected q1, got %+v", sections[1].MCQ)
	}
	if sections[1].MCQ.Question != "What does ssh do?" {
		t.Fatalf("unexpected question: %q", sections[1].MCQ.Question)
	}
	if len(sections[1].MCQ.Options) != 3 {
		t.Fatalf("expected 3 options, got %+v", sections[1].MCQ.Options)
	}
	if len(sections[1].MCQ.Selected) != 1 || sections[1].MCQ.Selected[0] != "B" {
		t.Fatalf("unexpected selected answers: %+v", sections[1].MCQ.Selected)
	}
	if sections[2].Type != "markdown" || sections[2].Markdown == "" {
		t.Fatalf("unexpected final section: %+v", sections[2])
	}
}

func TestParseLabSectionsMissingMCQID(t *testing.T) {
	_, err := ParseLabSections([]byte("```mcq\nQuestion\n- [ ] A\n```\n"))
	if err == nil {
		t.Fatalf("expected error for missing id")
	}
}
