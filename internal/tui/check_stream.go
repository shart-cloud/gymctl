package tui

import (
	"bufio"
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"
)

// checkStartedMsg is emitted once the check process has been launched (or has
// failed to launch). It hands the model the channel it will drain for output.
type checkStartedMsg struct {
	events chan checkEventMsg
}

// checkEventMsg carries one line of check output, or — when done is true — the
// terminal result. err is the process exit error: `gymctl check` exits non-zero
// when the exercise is not yet passing, so a non-nil err here means "not passing
// yet", not a harness failure.
type checkEventMsg struct {
	line string
	done bool
	err  error
}

// streamCheck runs execCmd with stdout+stderr merged into a pipe and streams the
// output back as checkEventMsg values on a channel, keeping the TUI live so Jerry
// can animate while the check runs. The *exec.Cmd is built by the caller (so the
// command factory is exercised synchronously); this only wires up the pipe.
func streamCheck(execCmd *exec.Cmd) tea.Cmd {
	return func() tea.Msg {
		ch := make(chan checkEventMsg, 128)

		fail := func(err error) tea.Msg {
			ch <- checkEventMsg{done: true, err: err}
			close(ch)
			return checkStartedMsg{events: ch}
		}

		pr, pw, err := os.Pipe()
		if err != nil {
			return fail(err)
		}
		execCmd.Stdin = nil
		execCmd.Stdout = pw
		execCmd.Stderr = pw
		if err := execCmd.Start(); err != nil {
			pw.Close()
			pr.Close()
			return fail(err)
		}
		// The parent must drop its copy of the write end so the reader sees EOF
		// once the child exits.
		pw.Close()

		go func() {
			sc := bufio.NewScanner(pr)
			sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
			for sc.Scan() {
				ch <- checkEventMsg{line: sc.Text()}
			}
			werr := execCmd.Wait()
			pr.Close()
			// Only after EOF (all lines sent) do we emit the terminal event, so
			// output always precedes the result on the channel.
			ch <- checkEventMsg{done: true, err: werr}
			close(ch)
		}()

		return checkStartedMsg{events: ch}
	}
}

// waitForCheck blocks on the next event from the check stream and delivers it to
// Update. Each event the model receives re-issues this command until done.
func waitForCheck(ch chan checkEventMsg) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return checkEventMsg{done: true}
		}
		return ev
	}
}
