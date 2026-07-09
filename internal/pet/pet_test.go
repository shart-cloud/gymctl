package pet

import (
	"bytes"
	"strings"
	"testing"

	"gymctl/internal/scenario"
)

func TestMoodForTrigger(t *testing.T) {
	cases := map[string]Mood{
		"onStart":     Nervous,
		"onHintUsed":  Nervous,
		"onCheckPass": Happy,
		"onCheckFail": Sad,
		"onComplete":  Celebrate,
		"onCheck":     Thinking,
		"checking":    Thinking,
		"":            Idle,
		"bogus":       Idle,
	}
	for trigger, want := range cases {
		if got := MoodForTrigger(trigger); got != want {
			t.Errorf("MoodForTrigger(%q) = %v, want %v", trigger, got, want)
		}
	}
}

func TestReactPrefersDialogOverride(t *testing.T) {
	dialog := &scenario.JerryDialog{OnStart: "custom start line"}
	mood, line := React(dialog, "onStart", "ex-1")
	if mood != Nervous {
		t.Errorf("mood = %v, want Nervous", mood)
	}
	if line != "custom start line" {
		t.Errorf("line = %q, want the dialog override", line)
	}
}

func TestReactFallbackWhenNoOverride(t *testing.T) {
	// Empty override falls through to the deterministic pool.
	mood, line := React(&scenario.JerryDialog{}, "onStart", "ex-1")
	if mood != Nervous {
		t.Errorf("mood = %v, want Nervous", mood)
	}
	if !contains(fallbacks["onStart"], line) {
		t.Errorf("line %q not drawn from the onStart pool", line)
	}
}

func TestFallbackDeterministic(t *testing.T) {
	// Same (exercise, trigger) must pick the same line every time.
	a := fallback("onCheckFail", "cks-001")
	b := fallback("onCheckFail", "cks-001")
	if a != b {
		t.Errorf("fallback not deterministic: %q != %q", a, b)
	}
	if !contains(fallbacks["onCheckFail"], a) {
		t.Errorf("line %q not in the onCheckFail pool", a)
	}
}

func TestFallbackUnknownTrigger(t *testing.T) {
	if got := fallback("no-such-trigger", "ex"); got == "" {
		t.Error("fallback for an unknown trigger should still return a line")
	}
}

func TestSpriteEveryMoodRenders(t *testing.T) {
	moods := []Mood{Idle, Thinking, Nervous, Happy, Sad, Celebrate}
	for _, m := range moods {
		// Sweep a range of frames to exercise every animation branch.
		for frame := 0; frame < 48; frame++ {
			lines := Sprite(m, frame)
			if len(lines) == 0 {
				t.Fatalf("Sprite(%v, %d) returned no lines", m, frame)
			}
			for _, ln := range lines {
				if strings.TrimSpace(ln) == "" {
					t.Errorf("Sprite(%v, %d) has a blank line", m, frame)
				}
			}
		}
	}
}

func TestSpriteIdleBlinks(t *testing.T) {
	// The resting frame shows open eyes; some frame within a blink window closes them.
	rest := strings.Join(Sprite(Idle, 0), "\n")
	if !strings.Contains(rest, "o   o") {
		t.Errorf("resting idle sprite should have open eyes, got:\n%s", rest)
	}
	blinked := false
	for frame := 1; frame < 44; frame++ {
		if strings.Contains(strings.Join(Sprite(Idle, frame), "\n"), "-   -") {
			blinked = true
			break
		}
	}
	if !blinked {
		t.Error("idle Jerry never blinked across two cycles")
	}
}

func TestRenderCLIIncludesSpriteAndLine(t *testing.T) {
	var buf bytes.Buffer
	dialog := &scenario.JerryDialog{OnComplete: "nice work"}
	RenderCLI(&buf, dialog, "onComplete", "ex-1")
	out := buf.String()
	if !strings.Contains(out, "jerry:") {
		t.Errorf("RenderCLI output missing the jerry caption:\n%s", out)
	}
	if !strings.Contains(out, "nice work") {
		t.Errorf("RenderCLI output missing the reaction line:\n%s", out)
	}
	if !strings.Contains(out, "╰──┬─┬──╯") {
		t.Errorf("RenderCLI output missing the sprite:\n%s", out)
	}
}

func contains(pool []string, s string) bool {
	for _, p := range pool {
		if p == s {
			return true
		}
	}
	return false
}
