package tui

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"gymctl/internal/environment"
	"gymctl/internal/pet"
	"gymctl/internal/progress"
	"gymctl/internal/scenario"
)

type viewState int

const (
	viewHome viewState = iota
	viewTrackSelect
	viewExerciseList
	viewExerciseDetail
	viewHintPeek
	viewStartConfirm
)

type trackSummary struct {
	name      string
	total     int
	completed int
}

type startDoneMsg struct{ err error }
type shellExitMsg struct{ err error }
type actionDoneMsg struct {
	action string
	err    error
}

type Model struct {
	view           viewState
	catalog        []scenario.CatalogEntry
	prog           *progress.File
	tasksDir       string
	progressPath   string
	tracks         []trackSummary
	trackCursor    int
	exerciseCursor int
	selectedTrack  string
	selectedEx     *scenario.Exercise
	selectedExPath string
	hintContent    string
	startChoice    int
	width          int
	height         int
	err            error
	statusLine     string
	isDarkTheme    bool
	hasFocus       bool

	// greeting is chosen once at startup so it doesn't change throughout the session.
	greeting string
	// listScroll is the scroll offset for the exercise list view.
	listScroll int
	// hintScroll is the scroll offset for the hint view.
	hintScroll int
	// exState caches the on-disk ExerciseState for the selected exercise.
	// It is reloaded when navigating to the detail view and after actions complete.
	// Keeping it cached avoids a disk read on every TUI render tick.
	exState *environment.ExerciseState
	// commands creates external commands for actions. Tests swap this seam.
	commands tuiCommandFactory
	// frame is the animation counter driven by tickMsg; it makes Jerry blink.
	frame int
	// petMood is Jerry's current mood, set by lifecycle actions.
	petMood pet.Mood
}

// tickMsg drives the pet animation (idle-blink).
type tickMsg struct{}

// petTick schedules the next animation frame.
func petTick() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
}

var greetings = []string{
	"jerry made a mess again. let's clean it up.",
	"pick an exercise and dig in.",
	"welcome back. the cluster is weird again.",
	"fresh broken environment, just for you.",
	"jerry swears it was working a minute ago.",
	"fix the outage, score the points.",
}

func NewModel(catalog []scenario.CatalogEntry, prog *progress.File, tasksDir, progressPath string) Model {
	m := Model{
		view:         viewHome,
		catalog:      catalog,
		prog:         prog,
		tasksDir:     tasksDir,
		progressPath: progressPath,
		greeting:     greetings[rand.Intn(len(greetings))],
	}
	m.buildTracks()
	return m
}

func (m *Model) buildTracks() {
	trackMap := map[string]*trackSummary{}
	for _, entry := range m.catalog {
		t := entry.Exercise.Metadata.Track
		if t == "" {
			continue
		}
		if _, ok := trackMap[t]; !ok {
			trackMap[t] = &trackSummary{name: t}
		}
		trackMap[t].total++
		name := entry.Exercise.Metadata.Name
		if st, ok := m.prog.Exercises[name]; ok && st.Status == "completed" {
			trackMap[t].completed++
		}
	}

	m.tracks = nil
	for _, ts := range trackMap {
		m.tracks = append(m.tracks, *ts)
	}
	sort.Slice(m.tracks, func(i, j int) bool {
		return m.tracks[i].name < m.tracks[j].name
	})
}

func (m Model) trackExercises() []scenario.CatalogEntry {
	var out []scenario.CatalogEntry
	for _, e := range m.catalog {
		if e.Exercise.Metadata.Track == m.selectedTrack {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		oi := out[i].Exercise.Metadata.Order
		oj := out[j].Exercise.Metadata.Order
		if oi != oj {
			return oi < oj
		}
		return out[i].Exercise.Metadata.Name < out[j].Exercise.Metadata.Name
	})
	return out
}

func (m Model) isLocked(ex *scenario.Exercise) bool {
	for _, prereq := range ex.Spec.Prerequisites {
		st, ok := m.prog.Exercises[prereq]
		if !ok || st.Status != "completed" {
			return true
		}
	}
	return false
}

func (m Model) overallProgress() (int, int) {
	total := len(m.catalog)
	done := 0
	for _, e := range m.catalog {
		if st, ok := m.prog.Exercises[e.Exercise.Metadata.Name]; ok && st.Status == "completed" {
			done++
		}
	}
	return done, total
}

// activeExerciseName returns the name of the currently in_progress exercise, if any.
func (m Model) activeExerciseName() string {
	for name, st := range m.prog.Exercises {
		if st.Status == "in_progress" {
			return name
		}
	}
	return ""
}

// visibleListRows returns how many exercise rows fit given the current terminal height.
func (m Model) visibleListRows() int {
	// overhead: top padding + track badge + panel borders + title + footer
	const overhead = 12
	v := m.height - overhead
	if v < 5 {
		return 5
	}
	return v
}

// visibleHintRows returns how many hint content lines fit given the current terminal height.
func (m Model) visibleHintRows() int {
	const overhead = 8
	v := m.height - overhead
	if v < 5 {
		return 5
	}
	return v
}

// reloadState loads ExerciseState from disk for the selected exercise and caches it.
// Call this after navigating to an exercise and after any action that may change state.
func (m *Model) reloadState() {
	if m.selectedEx == nil {
		m.exState = nil
		return
	}
	state, _ := environment.LoadExerciseState(m.selectedEx.Metadata.Name)
	m.exState = state
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(tea.RequestBackgroundColor, petTick())
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		m.frame++
		return m, petTick()

	case tea.BackgroundColorMsg:
		m.isDarkTheme = msg.IsDark()
		applyTheme(m.isDarkTheme)
		return m, nil

	case tea.FocusMsg:
		m.hasFocus = true
		return m, nil

	case tea.BlurMsg:
		m.hasFocus = false
		return m, nil

	case tea.MouseWheelMsg:
		return m.handleMouseWheel(msg)

	case trackClickedMsg:
		if msg.index >= 0 && msg.index < len(m.tracks) {
			m.trackCursor = msg.index
			m.selectedTrack = m.tracks[m.trackCursor].name
			m.exerciseCursor = 0
			m.listScroll = 0
			m.view = viewExerciseList
		}
		return m, nil

	case exerciseClickedMsg:
		exercises := m.trackExercises()
		if msg.index >= 0 && msg.index < len(exercises) {
			m.exerciseCursor = msg.index
			if m.exerciseCursor < m.listScroll {
				m.listScroll = m.exerciseCursor
			}
			if visRows := m.visibleListRows(); m.exerciseCursor >= m.listScroll+visRows {
				m.listScroll = m.exerciseCursor - visRows + 1
			}

			entry := exercises[m.exerciseCursor]
			m.selectedEx = entry.Exercise
			m.selectedExPath = entry.Dir
			m.hintContent = ""
			m.statusLine = ""
			m.reloadState()
			m.petMood = m.petMoodForSelected()
			m.view = viewExerciseDetail
		}
		return m, nil

	case footerActionMsg:
		return m.handleKeyPress(msg.key)

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case startDoneMsg:
		if msg.err != nil {
			m.err = msg.err
			m.statusLine = fmt.Sprintf("start failed: %v", msg.err)
			return m, nil
		}
		// Pick the right post-start action based on exercise type.
		ex := m.selectedEx
		if ex != nil {
			switch {
			case ex.Spec.Environment.Type == "kubernetes" &&
				ex.Spec.Environment.Kubernetes != nil &&
				ex.Spec.Environment.Kubernetes.Provider == "vagrant":
				// Vagrant: SSH into the control plane instead of launching a Docker workstation.
				sshCmd := m.actionCommands().SSH(m.tasksDir, "cp-1", ex.Metadata.Name)
				return m, tea.ExecProcess(sshCmd, func(err error) tea.Msg {
					return shellExitMsg{err}
				})
			case ex.Spec.Environment.Type == "docker":
				// Docker: environment is now running locally, just return to detail view.
				return m, func() tea.Msg { return shellExitMsg{nil} }
			}
		}
		// Default (Kind): drop into workstation container.
		workstationCmd := m.actionCommands().Workstation(m.selectedEx.Metadata.Name)
		return m, tea.ExecProcess(workstationCmd, func(err error) tea.Msg {
			return shellExitMsg{err}
		})

	case shellExitMsg:
		if prog, err := progress.Load(m.progressPath); err == nil {
			m.prog = prog
			m.buildTracks()
		}
		m.reloadState()
		m.view = viewExerciseDetail
		m.petMood = m.petMoodForSelected()
		if msg.err != nil {
			m.statusLine = fmt.Sprintf("action ended with error: %v", msg.err)
		} else {
			m.statusLine = "returned from shell"
		}
		return m, nil

	case actionDoneMsg:
		if prog, err := progress.Load(m.progressPath); err == nil {
			m.prog = prog
			m.buildTracks()
		}
		m.reloadState()
		m.view = viewExerciseDetail
		if msg.err != nil {
			m.statusLine = fmt.Sprintf("%s failed: %v", msg.action, msg.err)
		} else {
			m.statusLine = fmt.Sprintf("%s complete", msg.action)
		}
		// Jerry reacts: celebrate when the exercise is now solved, sad when a
		// check ran but didn't pass, otherwise back to nervously watching.
		m.petMood = m.petMoodForSelected()
		if msg.action == "check" && m.petMood != pet.Celebrate {
			m.petMood = pet.Sad
		} else if msg.action == "reset" {
			m.petMood = pet.Nervous
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	return m.handleKeyPress(key)
}

func (m Model) handleKeyPress(key string) (tea.Model, tea.Cmd) {
	switch m.view {
	case viewHome:
		switch key {
		case "enter", "ctrl+m":
			m.view = viewTrackSelect
			m.trackCursor = 0
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case viewTrackSelect:
		switch key {
		case "up", "k", "shift+tab":
			if m.trackCursor > 0 {
				m.trackCursor--
			}
		case "down", "j", "tab":
			if m.trackCursor < len(m.tracks)-1 {
				m.trackCursor++
			}
		case "enter", "ctrl+m", "l":
			if len(m.tracks) > 0 {
				m.selectedTrack = m.tracks[m.trackCursor].name
				m.exerciseCursor = 0
				m.listScroll = 0
				m.view = viewExerciseList
			}
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			idx := int(key[0] - '1')
			if idx >= 0 && idx < len(m.tracks) {
				m.trackCursor = idx
				m.selectedTrack = m.tracks[m.trackCursor].name
				m.exerciseCursor = 0
				m.listScroll = 0
				m.view = viewExerciseList
			}
		case "b", "esc":
			m.view = viewHome
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case viewExerciseList:
		exercises := m.trackExercises()
		visRows := m.visibleListRows()
		switch key {
		case "up", "k", "shift+tab":
			if m.exerciseCursor > 0 {
				m.exerciseCursor--
			}
			// Keep cursor within visible window.
			if m.exerciseCursor < m.listScroll {
				m.listScroll = m.exerciseCursor
			}
		case "down", "j", "tab":
			if m.exerciseCursor < len(exercises)-1 {
				m.exerciseCursor++
			}
			// Scroll down if cursor moved below the visible window.
			if m.exerciseCursor >= m.listScroll+visRows {
				m.listScroll = m.exerciseCursor - visRows + 1
			}
		case "enter", "ctrl+m", "l":
			if len(exercises) > 0 {
				entry := exercises[m.exerciseCursor]
				m.selectedEx = entry.Exercise
				m.selectedExPath = entry.Dir
				m.hintContent = ""
				m.statusLine = ""
				m.reloadState()
				m.petMood = m.petMoodForSelected()
				m.view = viewExerciseDetail
			}
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			idx := int(key[0] - '1')
			if idx >= 0 && idx < len(exercises) {
				m.exerciseCursor = idx
				// Scroll to show the quick-selected exercise.
				if m.exerciseCursor < m.listScroll {
					m.listScroll = m.exerciseCursor
				}
				if m.exerciseCursor >= m.listScroll+visRows {
					m.listScroll = m.exerciseCursor - visRows + 1
				}
				entry := exercises[m.exerciseCursor]
				m.selectedEx = entry.Exercise
				m.selectedExPath = entry.Dir
				m.hintContent = ""
				m.statusLine = ""
				m.reloadState()
				m.petMood = m.petMoodForSelected()
				m.view = viewExerciseDetail
			}
		case "b", "esc":
			m.view = viewTrackSelect
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case viewExerciseDetail:
		switch key {
		case "s":
			if !m.isLocked(m.selectedEx) {
				m.startChoice = 0
				m.view = viewStartConfirm
			}
		case "r":
			if !m.isLocked(m.selectedEx) {
				resetCmd := m.actionCommands().Reset(m.tasksDir, m.selectedEx.Metadata.Name)
				return m, tea.ExecProcess(resetCmd, func(err error) tea.Msg {
					return actionDoneMsg{action: "reset", err: err}
				})
			}
		case "c":
			checkCmd := m.actionCommands().Check(m.tasksDir, m.selectedEx.Metadata.Name)
			return m, tea.ExecProcess(checkCmd, func(err error) tea.Msg {
				return actionDoneMsg{action: "check", err: err}
			})
		case "e":
			envCmd := m.actionCommands().Env(m.tasksDir, m.selectedEx.Metadata.Name)
			return m, tea.ExecProcess(envCmd, func(err error) tea.Msg {
				return actionDoneMsg{action: "env", err: err}
			})
		case "k":
			kubeconfigCmd := m.actionCommands().Kubeconfig(m.tasksDir, m.selectedEx.Metadata.Name)
			return m, tea.ExecProcess(kubeconfigCmd, func(err error) tea.Msg {
				return actionDoneMsg{action: "kubeconfig", err: err}
			})
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			nodes := m.selectedNodeRefs()
			idx := int(key[0] - '1')
			if idx >= 0 && idx < len(nodes) {
				sshCmd := m.actionCommands().SSH(m.tasksDir, nodes[idx], m.selectedEx.Metadata.Name)
				return m, tea.ExecProcess(sshCmd, func(err error) tea.Msg {
					return actionDoneMsg{action: "ssh", err: err}
				})
			}
		case "h":
			m.hintContent = m.loadFreeHint()
			m.hintScroll = 0
			m.view = viewHintPeek
		case "b", "esc":
			// Clear status when leaving the detail view so stale messages don't persist.
			m.statusLine = ""
			m.view = viewExerciseList
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case viewHintPeek:
		lines := strings.Split(m.hintContent, "\n")
		maxScroll := len(lines) - m.visibleHintRows()
		if maxScroll < 0 {
			maxScroll = 0
		}
		switch key {
		case "up", "k":
			if m.hintScroll > 0 {
				m.hintScroll--
			}
		case "down", "j":
			if m.hintScroll < maxScroll {
				m.hintScroll++
			}
		case "pgup", "ctrl+u":
			m.hintScroll -= m.visibleHintRows() / 2
			if m.hintScroll < 0 {
				m.hintScroll = 0
			}
		case "pgdn", "ctrl+d":
			m.hintScroll += m.visibleHintRows() / 2
			if m.hintScroll > maxScroll {
				m.hintScroll = maxScroll
			}
		case "b", "esc", "enter":
			m.hintScroll = 0
			m.view = viewExerciseDetail
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case viewStartConfirm:
		switch key {
		case "up", "k":
			if m.startChoice > 0 {
				m.startChoice--
			}
		case "down", "j":
			if m.startChoice < 1 {
				m.startChoice++
			}
		case "enter":
			if m.startChoice == 0 {
				startCmd := m.actionCommands().Start(m.tasksDir, m.selectedEx.Metadata.Name)
				return m, tea.ExecProcess(startCmd, func(err error) tea.Msg {
					return startDoneMsg{err}
				})
			}
			// Exit TUI, print reminder to the terminal.
			fmt.Printf("\nRun: gymctl start %s\n", m.selectedEx.Metadata.Name)
			return m, tea.Quit
		case "esc":
			m.view = viewExerciseDetail
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m Model) loadFreeHint() string {
	for _, hint := range m.selectedEx.Spec.Hints {
		if hint.Cost == 0 {
			if hint.Content != "" {
				return hint.Content
			}
			if hint.File != "" {
				data, err := os.ReadFile(filepath.Join(m.selectedExPath, hint.File))
				if err == nil {
					return string(data)
				}
				return fmt.Sprintf("(could not read hint file: %v)", err)
			}
		}
	}
	return "(no free hint available)"
}

// selectedNodeRefs returns sorted node reference names from the cached exercise state.
// Control-plane nodes are ordered before worker nodes.
func (m Model) selectedNodeRefs() []string {
	if m.exState == nil || len(m.exState.Nodes) == 0 {
		return nil
	}
	nodes := make([]string, 0, len(m.exState.Nodes))
	for logical := range m.exState.Nodes {
		nodes = append(nodes, logical)
	}
	sort.Slice(nodes, func(i, j int) bool {
		a := nodes[i]
		b := nodes[j]
		if strings.HasPrefix(a, "cp-") && strings.HasPrefix(b, "worker-") {
			return true
		}
		if strings.HasPrefix(a, "worker-") && strings.HasPrefix(b, "cp-") {
			return false
		}
		return a < b
	})
	return nodes
}
