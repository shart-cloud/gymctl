package dialogue

import (
	"fmt"
	"hash/fnv"
	"io"

	"github.com/fatih/color"

	"gymctl/internal/scenario"
)

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

// Jerry returns the dialogue line for a given trigger and exercise.
// If the exercise has a specific line for this trigger, it is returned.
// Otherwise a deterministic fallback is selected based on exercise name + trigger.
func Jerry(dialog *scenario.JerryDialog, trigger, exerciseName string) string {
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
	return selectFallback(trigger, exerciseName)
}

// selectFallback picks a fallback line deterministically from the pool.
func selectFallback(trigger, exerciseName string) string {
	pool, ok := fallbacks[trigger]
	if !ok || len(pool) == 0 {
		return "i probably should have read the docs"
	}
	h := fnv.New32a()
	h.Write([]byte(exerciseName + trigger))
	idx := h.Sum32() % uint32(len(pool))
	return pool[idx]
}

var (
	jerryLabelColor = color.New(color.FgYellow, color.Faint)
	jerryTextColor  = color.New(color.Faint)
)

// RenderJerry writes Jerry's dialogue line to w in a consistent format.
func RenderJerry(w io.Writer, line string) {
	fmt.Fprintf(w, "  %s %s\n", jerryLabelColor.Sprint("Jerry:"), jerryTextColor.Sprintf("%q", line))
}
