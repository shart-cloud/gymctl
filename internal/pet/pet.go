// Package pet renders Jerry — the terminal companion who broke your
// environment and now watches you fix it. He reacts to lifecycle events
// (start / check pass / check fail / hint / complete) with a mood and a line.
//
// This replaces the old internal/dialogue package: the reaction-line pools
// live here now, re-keyed to moods instead of raw triggers.
//
// The ASCII art in Sprite is a deliberately simple placeholder. It is isolated
// behind the Sprite variable so a richer animated creature (à la pi-pokepet /
// codex pets / the Claude Code squid) can be dropped in later without touching
// the mood mapping, reaction lines, or any caller.
package pet

import (
	"fmt"
	"hash/fnv"
	"io"

	"github.com/fatih/color"

	"gymctl/internal/scenario"
)

// Mood is Jerry's emotional state, derived from a lifecycle trigger.
type Mood int

const (
	Idle Mood = iota
	Thinking
	Nervous
	Happy
	Sad
	Celebrate
)

// Label returns the lowercase mood name (used in the TUI caption).
func (m Mood) Label() string {
	switch m {
	case Thinking:
		return "thinking"
	case Nervous:
		return "nervous"
	case Happy:
		return "happy"
	case Sad:
		return "sad"
	case Celebrate:
		return "celebrate"
	default:
		return "idle"
	}
}

// MoodForTrigger maps a lifecycle trigger to the mood Jerry shows.
func MoodForTrigger(trigger string) Mood {
	switch trigger {
	case "onStart", "onHintUsed":
		return Nervous
	case "onCheckPass":
		return Happy
	case "onCheckFail":
		return Sad
	case "onComplete":
		return Celebrate
	case "onCheck", "checking":
		return Thinking
	default:
		return Idle
	}
}

// Sprite returns Jerry's multi-line sprite for a mood. frame drives the
// idle-blink and thinking-dots so he feels alive when animated in the TUI;
// pass frame=0 for a static frame (CLI output).
//
// Swap this variable to replace the art wholesale.
var Sprite = defaultSprite

// defaultSprite draws Jerry as a small boxed face, ~4 lines tall. Every mood
// shares the same head outline so he reads as one character across states; the
// eyes, mouth, and a per-mood accent (thinking dots, nervous sweat, celebrate
// sparkles) carry the emotion. frame animates those accents in the TUI; frame=0
// is the resting frame used for static CLI output.
func defaultSprite(m Mood, frame int) []string {
	// blink returns true for a couple of frames every ~3s, but never on the
	// resting frame so static CLI output shows open eyes.
	blink := frame > 0 && frame%22 < 2

	switch m {
	case Thinking:
		dots := []string{"   ", ".  ", ".. ", "..."}[(frame/2)%4]
		return []string{
			"╭───────╮",
			"│ o   o │ " + dots,
			"│   ─   │",
			"╰──┬─┬──╯",
		}
	case Nervous:
		sweat := " "
		if frame/3%2 == 0 { // a bead of sweat drips on alternating beats
			sweat = "°"
		}
		return []string{
			"╭───────╮",
			"│ ◕   ◕ │ " + sweat,
			"│  ~~~  │",
			"╰──┬─┬──╯",
		}
	case Happy:
		return []string{
			"╭───────╮",
			"│ ^   ^ │",
			"│  \\_/  │",
			"╰──┬─┬──╯",
		}
	case Sad:
		return []string{
			"╭───────╮",
			"│ ;   ; │",
			"│  /¯\\  │",
			"╰──┬─┬──╯",
		}
	case Celebrate:
		spark := "  "
		if frame/2%2 == 0 {
			spark = "✦ "
		}
		return []string{
			"╭───────╮ " + spark,
			"│ ◠   ◠ │",
			"│  \\▽/  │",
			"╰──┬─┬──╯",
		}
	default: // Idle
		eyes := "o   o"
		if blink {
			eyes = "-   -"
		}
		return []string{
			"╭───────╮",
			"│ " + eyes + " │",
			"│   ─   │",
			"╰──┬─┬──╯",
		}
	}
}

// React returns the mood and reaction line for a trigger. Per-exercise lines
// from JerryDialog win; otherwise a deterministic fallback is chosen.
func React(dialog *scenario.JerryDialog, trigger, exerciseName string) (Mood, string) {
	return MoodForTrigger(trigger), line(dialog, trigger, exerciseName)
}

var (
	labelColor = color.New(color.FgYellow, color.Faint)
	artColor   = color.New(color.Faint)
	textColor  = color.New(color.Faint)
)

// RenderCLI writes a static Jerry — sprite plus reaction line — to w for a
// lifecycle trigger. This is the plain-terminal counterpart to the animated
// TUI pet. Replaces the old dialogue.RenderJerry.
func RenderCLI(w io.Writer, dialog *scenario.JerryDialog, trigger, exerciseName string) {
	mood, said := React(dialog, trigger, exerciseName)
	art := Sprite(mood, 0)
	for i, ln := range art {
		if i == len(art)-1 {
			fmt.Fprintf(w, "  %s   %s %s\n",
				artColor.Sprint(ln),
				labelColor.Sprint("jerry:"),
				textColor.Sprintf("%q", said))
		} else {
			fmt.Fprintf(w, "  %s\n", artColor.Sprint(ln))
		}
	}
}

// line selects the reaction text: a per-exercise override if present, else a
// deterministic fallback from the pool for that trigger.
func line(dialog *scenario.JerryDialog, trigger, exerciseName string) string {
	if dialog != nil {
		switch trigger {
		case "onStart":
			if dialog.OnStart != "" {
				return dialog.OnStart
			}
		case "onCheckFail":
			if dialog.OnCheckFail != "" {
				return dialog.OnCheckFail
			}
		case "onCheckPass":
			if dialog.OnCheckPass != "" {
				return dialog.OnCheckPass
			}
		case "onHintUsed":
			if dialog.OnHintUsed != "" {
				return dialog.OnHintUsed
			}
		case "onComplete":
			if dialog.OnComplete != "" {
				return dialog.OnComplete
			}
		}
	}
	return fallback(trigger, exerciseName)
}

var fallbacks = map[string][]string{
	"onStart": {
		"i swear i only changed one line",
		"it was working on my machine",
		"this is probably fine, right?",
		"i may have pushed directly to main",
		"i was going to write tests but ran out of time",
		"production and staging are basically the same environment",
	},
	"onCheckFail": {
		"okay but did you try turning it off and on again",
		"it worked in the demo though",
		"i don't think that check is measuring the right thing",
		"this is probably a flake, run it again",
		"have you considered that the check might be wrong",
		"the logs aren't that red",
	},
	"onCheckPass": {
		"oh nice, you're actually good at this",
		"see, i knew it wasn't that broken",
		"okay that one i definitely didn't break",
		"green is my favorite color",
	},
	"onHintUsed": {
		"i definitely read that documentation",
		"i would have gotten there",
		"that's basically what i was thinking",
		"yeah no i knew that",
		"okay i didn't know that",
		"in my defense the docs are very long",
	},
	"onComplete": {
		"okay yeah that was probably my fault",
		"i'll do better next time, but also no promises",
		"thanks, don't tell my manager",
		"i would have fixed it eventually",
		"it was a learning experience for everyone",
		"cool cool cool, who's buying lunch",
	},
}

func fallback(trigger, exerciseName string) string {
	pool, ok := fallbacks[trigger]
	if !ok || len(pool) == 0 {
		return "i probably should have read the docs"
	}
	h := fnv.New32a()
	h.Write([]byte(exerciseName + trigger))
	return pool[h.Sum32()%uint32(len(pool))]
}
