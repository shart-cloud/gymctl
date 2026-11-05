package checks

import (
	"path/filepath"
	"testing"
)

func TestExpandCheckPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GYM_WORKDIR", "/work/drill")

	cases := map[string]string{
		"$GYM_WORKDIR/answer.txt": "/work/drill/answer.txt",
		"~/cracked.txt":           filepath.Join(home, "cracked.txt"),
		"relative.txt":            "relative.txt",
	}
	for in, want := range cases {
		if got := expandCheckPath(in); got != want {
			t.Errorf("expandCheckPath(%q) = %q, want %q", in, got, want)
		}
	}
}
