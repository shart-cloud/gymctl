package tui

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

	"gymctl/internal/environment"
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
type progressReloadedMsg struct{ prog *progress.File }
type actionDoneMsg struct {
	action string
	err    error
}

type trackClickedMsg struct{ index int }
type exerciseClickedMsg struct{ index int }
type footerActionMsg struct{ key string }

var (
	listRowPattern  = regexp.MustCompile(`^\s*>?\s*(\d+)\.\s+`)
	ansiCodePattern = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)
)

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
}

var grimGreetings = []string{
	"GRIM-9 online. Let's clean up Jerry's latest messes.",
	"Fresh queue loaded. Pick a ticket and speedrun it.",
	"Ops desk ready. Chaos level: manageable.",
	"Mission board updated. You're on triage duty.",
	"Welcome back. The cluster is weird again.",
	"Ticket arcade open. Score points by fixing outages.",
}

func NewModel(catalog []scenario.CatalogEntry, prog *progress.File, tasksDir, progressPath string) Model {
	m := Model{
		view:         viewHome,
		catalog:      catalog,
		prog:         prog,
		tasksDir:     tasksDir,
		progressPath: progressPath,
		greeting:     grimGreetings[rand.Intn(len(grimGreetings))],
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
	return tea.RequestBackgroundColor
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

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
				sshCmd := newGymctlSSHCmd(m.tasksDir, "cp-1", ex.Metadata.Name)
				return m, tea.ExecProcess(sshCmd, func(err error) tea.Msg {
					return shellExitMsg{err}
				})
			case ex.Spec.Environment.Type == "docker":
				// Docker: environment is now running locally, just return to detail view.
				return m, func() tea.Msg { return shellExitMsg{nil} }
			}
		}
		// Default (Kind): drop into workstation container.
		workstationCmd := newWorkstationExecCmd(m.selectedEx.Metadata.Name)
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
				resetCmd := newGymctlResetCmd(m.tasksDir, m.selectedEx.Metadata.Name)
				return m, tea.ExecProcess(resetCmd, func(err error) tea.Msg {
					return actionDoneMsg{action: "reset", err: err}
				})
			}
		case "c":
			checkCmd := newGymctlCheckCmd(m.tasksDir, m.selectedEx.Metadata.Name)
			return m, tea.ExecProcess(checkCmd, func(err error) tea.Msg {
				return actionDoneMsg{action: "check", err: err}
			})
		case "e":
			envCmd := newGymctlEnvCmd(m.tasksDir, m.selectedEx.Metadata.Name)
			return m, tea.ExecProcess(envCmd, func(err error) tea.Msg {
				return actionDoneMsg{action: "env", err: err}
			})
		case "k":
			kubeconfigCmd := newGymctlKubeconfigCmd(m.tasksDir, m.selectedEx.Metadata.Name)
			return m, tea.ExecProcess(kubeconfigCmd, func(err error) tea.Msg {
				return actionDoneMsg{action: "kubeconfig", err: err}
			})
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			nodes := m.selectedNodeRefs()
			idx := int(key[0] - '1')
			if idx >= 0 && idx < len(nodes) {
				sshCmd := newGymctlSSHCmd(m.tasksDir, nodes[idx], m.selectedEx.Metadata.Name)
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
				startCmd := newGymctlStartCmd(m.tasksDir, m.selectedEx.Metadata.Name)
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

// renderEnvironmentPanel uses the cached exState — no disk reads inside View().
func (m Model) renderEnvironmentPanel(ex *scenario.Exercise) string {
	if ex == nil || ex.Spec.Environment.Type != "kubernetes" {
		return ""
	}
	state := m.exState
	if state == nil {
		return StyleDim.Render("Environment: not started yet")
	}

	var b strings.Builder
	provider := state.Provider
	if provider == "" {
		provider = "kubernetes"
	}
	b.WriteString(StyleBody.Render(fmt.Sprintf("Provider: %s", provider)) + "\n")

	if ex.Spec.Environment.Kubernetes != nil && ex.Spec.Environment.Kubernetes.Provider == "vagrant" {
		mode := environment.ResolveVagrantBootstrapMode(ex.Spec.Environment.Kubernetes)
		b.WriteString(StyleBody.Render(fmt.Sprintf("Mode: %s", mode)) + "\n")
	}

	if state.Kubeconfig != "" {
		b.WriteString(StyleBody.Render("Kubeconfig: ready") + "\n")
	} else {
		b.WriteString(StyleDim.Render("Kubeconfig: not exported") + "\n")
	}

	nodes := m.selectedNodeRefs()
	if len(nodes) > 0 {
		for i, node := range nodes {
			runtimeName := state.Nodes[node]
			b.WriteString(StyleDim.Render(fmt.Sprintf("[%d] ssh %s (%s)", i+1, node, runtimeName)) + "\n")
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

// View implements tea.Model.
func (m Model) View() tea.View {
	var content string
	switch m.view {
	case viewHome:
		content = m.viewHome()
	case viewTrackSelect:
		content = m.viewTrackSelect()
	case viewExerciseList:
		content = m.viewExerciseList()
	case viewExerciseDetail:
		content = m.viewExerciseDetail()
	case viewHintPeek:
		content = m.viewHintPeek()
	case viewStartConfirm:
		content = m.viewStartConfirm()
	}
	v := tea.NewView(content)
	v.AltScreen = true
	v.ReportFocus = true
	v.MouseMode = tea.MouseModeCellMotion
	v.WindowTitle = "gymctl shell"
	if m.selectedEx != nil {
		v.WindowTitle = "gymctl: " + m.selectedEx.Metadata.Name
	}
	v.OnMouse = m.onMouse(content)
	return v
}

func (m Model) viewHome() string {
	var b strings.Builder
	b.WriteString(renderGRIMHeader(m.width) + "\n\n")

	done, total := m.overallProgress()

	// Scale progress bar to terminal width.
	barWidth := 40
	if m.width > 0 && m.width-20 < barWidth {
		barWidth = m.width - 20
	}
	if barWidth < 10 {
		barWidth = 10
	}
	bar := renderProgressBar(done, total, barWidth)
	progressText := fmt.Sprintf("%s\n\n%s exercises completed out of %s total",
		bar,
		StyleHighlight.Render(fmt.Sprintf("%d", done)),
		StyleHighlight.Render(fmt.Sprintf("%d", total)))

	tracksBody := "No tracks found."
	if len(m.tracks) > 0 {
		lines := make([]string, 0, len(m.tracks))
		for _, track := range m.tracks {
			trackProgress := renderProgressBar(track.completed, track.total, 20)
			lines = append(lines, fmt.Sprintf("%s %s  %s", StyleHighlight.Render("▸"), StyleBody.Render(track.name), trackProgress))
		}
		tracksBody = strings.Join(lines, "\n")
	}

	b.WriteString("  " + renderPanel("Shift Brief", "💬 "+m.greeting) + "\n")
	b.WriteString("  " + renderPanel("Sprint Progress", progressText) + "\n")
	b.WriteString("  " + renderPanel("Tracks", tracksBody) + "\n")

	// Surface the currently active exercise so students can quickly resume.
	if active := m.activeExerciseName(); active != "" {
		b.WriteString("  " + renderPanel("Active Exercise",
			StyleWarning.Render("◉  ")+StyleBody.Render(active)+" — press Enter to browse") + "\n")
	}

	b.WriteString("  " + renderPanel("Controls", strings.Join([]string{
		StyleKey.Render(" Enter ") + StyleFooter.Render(" Browse Exercise Catalog"),
		StyleKey.Render("   q   ") + StyleFooter.Render(" Exit Ops Desk"),
	}, "\n")) + "\n")

	return b.String()
}

func (m Model) viewTrackSelect() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("  " + StyleSectionTitle.Render("Mission Tracks") + "\n\n")

	var lines []string
	if len(m.tracks) == 0 {
		lines = append(lines, StyleDim.Render("No tracks discovered in tasks catalog."))
	}

	for i, t := range m.tracks {
		line := fmt.Sprintf("%d. %s  ·  %d/%d complete", i+1, t.name, t.completed, t.total)
		if i == m.trackCursor {
			lines = append(lines, StyleSelected.Render("> "+line))
		} else {
			lines = append(lines, "  "+StyleBody.Render(line))
		}
	}
	b.WriteString("  " + renderPanel("Track Queue", strings.Join(lines, "\n")) + "\n")

	b.WriteString("\n")
	b.WriteString("  " + StyleKey.Render("[↑↓/jk/tab]") + StyleFooter.Render(" Move") + "   ")
	b.WriteString(StyleKey.Render("[Enter]") + StyleFooter.Render(" Select") + "   ")
	b.WriteString(StyleKey.Render("[1-9]") + StyleFooter.Render(" Quick Select") + "   ")
	b.WriteString(StyleKey.Render("[b]") + StyleFooter.Render(" Back") + "   ")
	b.WriteString(StyleKey.Render("[q]") + StyleFooter.Render(" Quit") + "\n")

	return b.String()
}

func (m Model) viewExerciseList() string {
	var b strings.Builder
	exercises := m.trackExercises()
	visRows := m.visibleListRows()

	b.WriteString("\n")
	b.WriteString("  " + StyleTrack.Render(m.selectedTrack) + "\n\n")

	// Compute the visible window.
	start := m.listScroll
	end := start + visRows
	if end > len(exercises) {
		end = len(exercises)
	}

	var lines []string
	for i := start; i < end; i++ {
		entry := exercises[i]
		ex := entry.Exercise
		name := ex.Metadata.Name
		locked := m.isLocked(ex)

		var statusIcon string
		if st, ok := m.prog.Exercises[name]; ok {
			switch st.Status {
			case "completed":
				statusIcon = StyleSuccess.Render("✓")
			case "started", "in_progress":
				statusIcon = StyleWarning.Render("◉")
			default:
				statusIcon = StyleDim.Render("◌")
			}
		} else {
			statusIcon = StyleDim.Render("◌")
		}

		diff := DifficultyStyle(ex.Spec.Difficulty).Render(ex.Spec.Difficulty)
		timeStr := ""
		if ex.Spec.EstimatedTime != "" {
			timeStr = StyleDim.Render(" · " + ex.Spec.EstimatedTime)
		}

		title := ex.Metadata.Title
		if title == "" {
			title = name
		}

		var lineContent string
		if locked {
			prereqs := strings.Join(ex.Spec.Prerequisites, ", ")
			lineContent = fmt.Sprintf("%s  %s%s  [needs: %s]",
				statusIcon,
				StyleDim.Render(title),
				timeStr,
				StyleDim.Render(prereqs),
			)
		} else {
			lineContent = fmt.Sprintf("%s  %s  %s%s",
				statusIcon,
				title,
				diff,
				timeStr,
			)
		}

		// Labels use absolute indices so [1-9] quick-select is always consistent.
		if i == m.exerciseCursor {
			lines = append(lines, StyleSelected.Render(fmt.Sprintf("> %d. %s", i+1, lineContent)))
		} else {
			lines = append(lines, fmt.Sprintf("  %d. %s", i+1, lineContent))
		}
	}
	if len(exercises) == 0 {
		lines = append(lines, StyleDim.Render("No exercises in this track."))
	}

	// Append a scroll position indicator when the list doesn't fit on screen.
	scrollHint := ""
	if len(exercises) > visRows {
		indicator := fmt.Sprintf("showing %d–%d of %d", start+1, end, len(exercises))
		if m.listScroll > 0 {
			indicator = "↑ " + indicator
		}
		if end < len(exercises) {
			indicator += " ↓"
		}
		scrollHint = "\n  " + StyleDim.Render(indicator)
	}

	b.WriteString("  " + renderPanel("Mission Queue", strings.Join(lines, "\n")+scrollHint) + "\n")

	b.WriteString("\n")
	b.WriteString("  " + StyleKey.Render("[↑↓/jk]") + StyleFooter.Render(" Move") + "   ")
	b.WriteString(StyleKey.Render("[Enter]") + StyleFooter.Render(" Open") + "   ")
	b.WriteString(StyleKey.Render("[1-9]") + StyleFooter.Render(" Quick Open") + "   ")
	b.WriteString(StyleKey.Render("[b]") + StyleFooter.Render(" Back") + "   ")
	b.WriteString(StyleKey.Render("[q]") + StyleFooter.Render(" Quit") + "\n")

	return b.String()
}

func (m Model) viewExerciseDetail() string {
	var b strings.Builder
	ex := m.selectedEx

	b.WriteString("\n")
	b.WriteString(m.renderGRIMTicket(ex) + "\n")

	var brief []string

	if len(ex.Spec.LearningOutcomes) > 0 {
		for _, lo := range ex.Spec.LearningOutcomes {
			brief = append(brief, "• "+StyleBody.Render(lo))
		}
	}

	if len(ex.Spec.Tags) > 0 {
		brief = append(brief, StyleDim.Render("Tags: "+strings.Join(ex.Spec.Tags, ", ")))
	}

	meta := []string{}
	if ex.Spec.EstimatedTime != "" {
		meta = append(meta, "⏱ "+ex.Spec.EstimatedTime)
	}
	if ex.Spec.Points > 0 {
		meta = append(meta, fmt.Sprintf("★ %d pts", ex.Spec.Points))
	}
	if len(meta) > 0 {
		brief = append(brief, StyleDim.Render(strings.Join(meta, "   ")))
	}
	if len(brief) > 0 {
		b.WriteString("  " + renderPanel("Mission Brief", strings.Join(brief, "\n")) + "\n")
	}

	b.WriteString("\n")
	envContent := strings.TrimSpace(m.renderEnvironmentPanel(ex))
	if envContent == "" {
		envContent = StyleDim.Render("No environment info available")
	}
	b.WriteString("  " + renderPanel("Environment", envContent) + "\n")
	b.WriteString("\n")

	// Footer keybinds
	locked := m.isLocked(ex)
	if locked {
		b.WriteString("  " + StyleDim.Render("[s] Start (locked)") + "   ")
		b.WriteString(StyleDim.Render("[r] Reset (locked)") + "   ")
	} else {
		b.WriteString("  " + StyleKey.Render("[s]") + StyleFooter.Render(" Start") + "   ")
		b.WriteString(StyleKey.Render("[r]") + StyleFooter.Render(" Reset") + "   ")
	}

	hasFreeHint := false
	for _, h := range ex.Spec.Hints {
		if h.Cost == 0 {
			hasFreeHint = true
			break
		}
	}
	if hasFreeHint {
		b.WriteString(StyleKey.Render("[h]") + StyleFooter.Render(" Hint") + "   ")
	}
	b.WriteString(StyleKey.Render("[c]") + StyleFooter.Render(" Check") + "   ")
	b.WriteString(StyleKey.Render("[e]") + StyleFooter.Render(" Env") + "   ")
	b.WriteString(StyleKey.Render("[k]") + StyleFooter.Render(" Kubeconfig") + "   ")
	b.WriteString(StyleKey.Render("[b]") + StyleFooter.Render(" Back") + "   ")
	b.WriteString(StyleKey.Render("[q]") + StyleFooter.Render(" Quit") + "\n")

	if m.statusLine != "" {
		b.WriteString("\n")
		// Colour the status line based on outcome.
		statusStyle := StyleDim
		switch {
		case strings.Contains(m.statusLine, "failed") || strings.Contains(m.statusLine, "error"):
			statusStyle = StyleError
		case strings.Contains(m.statusLine, "complete"):
			statusStyle = StyleSuccess
		}
		b.WriteString("  " + renderPanel("Latest Action", statusStyle.Render(m.statusLine)) + "\n")
	}

	return b.String()
}

// renderGRIMTicket renders the JIRA-style ticket box, responsive to terminal width.
func (m Model) renderGRIMTicket(ex *scenario.Exercise) string {
	boxWidth := m.width - 6 // subtract indent + border chars
	if boxWidth < 50 {
		boxWidth = 50
	}
	if boxWidth > 100 {
		boxWidth = 100
	}

	var b strings.Builder
	t := ex.Spec.Tasking

	var ticket, priority, summary, description, reporter string
	if t != nil {
		ticket = t.Ticket
		priority = t.Priority
		summary = t.Summary
		description = t.Description
		reporter = t.Reporter
	} else {
		ticket = syntheticTicketID(ex.Metadata.Name)
		priority = "P2"
		summary = ex.Metadata.Title
		if summary == "" {
			summary = ex.Metadata.Name
		}
		description = ex.Spec.Description
		reporter = "GRIM-9"
	}
	if reporter == "" {
		reporter = "GRIM-9"
	}

	dim := StyleDim.Render
	pStyle := PriorityStyle(priority)

	top := "  ┌─ GRIM-9 " + strings.Repeat("─", boxWidth-10) + "┐"
	b.WriteString(dim(top) + "\n")

	header := fmt.Sprintf("  │  %s  ·  %s — %s",
		ticket,
		pStyle.Render(priority),
		StyleSectionTitle.Render(strings.ToUpper(summary)),
	)
	b.WriteString(header + "\n")

	sep := "  ├" + strings.Repeat("─", boxWidth) + "┤"
	b.WriteString(dim(sep) + "\n")

	for _, line := range wordWrap(description, boxWidth-4) {
		if line == "" {
			b.WriteString("  │\n")
		} else {
			b.WriteString("  │  " + StyleBody.Render(line) + "\n")
		}
	}

	b.WriteString("  │\n")
	footer := fmt.Sprintf("  │  Reporter: %s  ·  Assignee: You (by elimination)", reporter)
	b.WriteString(footer + "\n")

	bot := "  └" + strings.Repeat("─", boxWidth) + "┘"
	b.WriteString(dim(bot) + "\n")

	return b.String()
}

func renderPanel(title, body string) string {
	header := StyleCardTitle.Render(title)
	content := StyleCardBody.Render(strings.TrimSpace(body))
	return StyleCard.Render(header + "\n" + content)
}

func (m Model) viewHintPeek() string {
	var b strings.Builder

	// Count free hints for an accurate title.
	freeHintCount := 0
	for _, h := range m.selectedEx.Spec.Hints {
		if h.Cost == 0 {
			freeHintCount++
		}
	}
	hintTitle := "Free Hint"
	if freeHintCount > 1 {
		hintTitle = fmt.Sprintf("Free Hints (%d available)", freeHintCount)
	}

	b.WriteString("\n")
	b.WriteString("  " + StyleSectionTitle.Render(hintTitle) + "\n")
	b.WriteString("  " + StyleDim.Render(strings.Repeat("─", 60)) + "\n\n")

	lines := strings.Split(m.hintContent, "\n")
	visRows := m.visibleHintRows()
	start := m.hintScroll
	end := start + visRows
	if end > len(lines) {
		end = len(lines)
	}

	for _, line := range lines[start:end] {
		// Render lines that look like shell commands (start with $) in accent colour.
		if strings.HasPrefix(strings.TrimSpace(line), "$") || strings.HasPrefix(line, "    ") {
			b.WriteString("  " + StyleHighlight.Render(line) + "\n")
		} else {
			b.WriteString("  " + StyleBody.Render(line) + "\n")
		}
	}

	if len(lines) > visRows {
		b.WriteString("\n")
		b.WriteString("  " + StyleDim.Render(
			fmt.Sprintf("[↑↓/jk · PgUp/PgDn — line %d/%d]", m.hintScroll+1, len(lines))) + "\n")
	}

	b.WriteString("\n")
	b.WriteString("  " + StyleKey.Render("[b/Esc]") + StyleFooter.Render(" Back") + "\n")

	return b.String()
}

func (m Model) viewStartConfirm() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("  " + StyleSectionTitle.Render("Start Exercise") + "\n\n")
	b.WriteString("  " + StyleBody.Render("How would you like to proceed?") + "\n\n")

	// Primary action label reflects the exercise type so the choice is unambiguous.
	primaryAction := "Start exercise"
	ex := m.selectedEx
	if ex != nil {
		switch {
		case ex.Spec.Environment.Type == "kubernetes" &&
			ex.Spec.Environment.Kubernetes != nil &&
			ex.Spec.Environment.Kubernetes.Provider == "vagrant":
			primaryAction = "Start VMs & SSH into control plane"
		case ex.Spec.Environment.Type == "docker":
			primaryAction = "Start docker environment"
		case ex.Spec.Environment.Type == "kubernetes":
			primaryAction = "Start cluster & open workstation shell"
		}
	}

	choices := []string{
		primaryAction,
		"Exit TUI — continue in terminal",
	}

	for i, c := range choices {
		if i == m.startChoice {
			b.WriteString(StyleSelected.Render("> "+c) + "\n")
		} else {
			b.WriteString("  " + StyleBody.Render(c) + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString("  " + StyleKey.Render("[↑↓]") + StyleFooter.Render(" Choose") + "   ")
	b.WriteString(StyleKey.Render("[Enter]") + StyleFooter.Render(" Confirm") + "   ")
	b.WriteString(StyleKey.Render("[Esc]") + StyleFooter.Render(" Cancel") + "\n")

	return b.String()
}

// wordWrap word-wraps text to at most width visual characters per line.
// Uses utf8.RuneCountInString so multi-byte characters are measured correctly.
func wordWrap(text string, width int) []string {
	var lines []string
	for _, paragraph := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			lines = append(lines, "")
			continue
		}
		words := strings.Fields(paragraph)
		current := ""
		currentLen := 0
		for _, word := range words {
			wordLen := utf8.RuneCountInString(word)
			if current == "" {
				current = word
				currentLen = wordLen
			} else if currentLen+1+wordLen <= width {
				current += " " + word
				currentLen += 1 + wordLen
			} else {
				lines = append(lines, current)
				current = word
				currentLen = wordLen
			}
		}
		if current != "" {
			lines = append(lines, current)
		}
	}
	return lines
}

// syntheticTicketID generates a deterministic ticket ID from an exercise name.
func syntheticTicketID(exerciseName string) string {
	h := uint32(2166136261)
	for _, c := range exerciseName {
		h ^= uint32(c)
		h *= 16777619
	}
	return fmt.Sprintf("TASK-%04d", h%9000+1000)
}
