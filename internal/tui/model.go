package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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
	startChoice    int // 0 = "Work on it", 1 = "Exit TUI"
	width          int
	height         int
	err            error
}

var grimGreetings = []string{
	"GRIM-9 ONLINE. Outstanding tickets await your attention.",
	"Good cycle. Jerry has filed more tickets. Surprise.",
	"All systems nominal. Except Jerry. Jerry is never nominal.",
	"Your sprint board is full. Jerry's doing. As always.",
	"GRIM-9 reporting: ticket queue non-empty. Handle accordingly.",
	"Another day, another Jerry incident. Your queue awaits.",
}

func NewModel(catalog []scenario.CatalogEntry, prog *progress.File, tasksDir, progressPath string) Model {
	m := Model{
		view:         viewHome,
		catalog:      catalog,
		prog:         prog,
		tasksDir:     tasksDir,
		progressPath: progressPath,
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

func grimGreeting() string {
	h := time.Now().Hour()
	return grimGreetings[h%len(grimGreetings)]
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case startDoneMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		// Drop into workstation container
		workstationCmd := newWorkstationExecCmd(m.selectedEx.Metadata.Name)
		return m, tea.ExecProcess(workstationCmd, func(err error) tea.Msg {
			return shellExitMsg{err}
		})

	case shellExitMsg:
		// Reload progress and return to exercise detail
		if prog, err := progress.Load(m.progressPath); err == nil {
			m.prog = prog
			m.buildTracks()
		}
		m.view = viewExerciseDetail
		return m, nil
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch m.view {
	case viewHome:
		switch key {
		case "enter":
			m.view = viewTrackSelect
			m.trackCursor = 0
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case viewTrackSelect:
		switch key {
		case "up", "k":
			if m.trackCursor > 0 {
				m.trackCursor--
			}
		case "down", "j":
			if m.trackCursor < len(m.tracks)-1 {
				m.trackCursor++
			}
		case "enter":
			if len(m.tracks) > 0 {
				m.selectedTrack = m.tracks[m.trackCursor].name
				m.exerciseCursor = 0
				m.view = viewExerciseList
			}
		case "b", "esc":
			m.view = viewHome
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case viewExerciseList:
		exercises := m.trackExercises()
		switch key {
		case "up", "k":
			if m.exerciseCursor > 0 {
				m.exerciseCursor--
			}
		case "down", "j":
			if m.exerciseCursor < len(exercises)-1 {
				m.exerciseCursor++
			}
		case "enter":
			if len(exercises) > 0 {
				entry := exercises[m.exerciseCursor]
				m.selectedEx = entry.Exercise
				m.selectedExPath = entry.Dir
				m.hintContent = ""
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
		case "h":
			m.hintContent = m.loadFreeHint()
			m.view = viewHintPeek
		case "b", "esc":
			m.view = viewExerciseList
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case viewHintPeek:
		switch key {
		case "b", "esc", "enter":
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
				// Mode A: run gymctl start, then drop into shell
				startCmd := newGymctlStartCmd(m.tasksDir, m.selectedEx.Metadata.Name)
				return m, tea.ExecProcess(startCmd, func(err error) tea.Msg {
					return startDoneMsg{err}
				})
			}
			// Mode B: exit TUI, print reminder
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

// View implements tea.Model.
func (m Model) View() string {
	switch m.view {
	case viewHome:
		return m.viewHome()
	case viewTrackSelect:
		return m.viewTrackSelect()
	case viewExerciseList:
		return m.viewExerciseList()
	case viewExerciseDetail:
		return m.viewExerciseDetail()
	case viewHintPeek:
		return m.viewHintPeek()
	case viewStartConfirm:
		return m.viewStartConfirm()
	}
	return ""
}

func (m Model) viewHome() string {
	var b strings.Builder

	// Enhanced GRIM header
	b.WriteString(renderGRIMHeader() + "\n\n")

	// Status message with enhanced styling
	greeting := grimGreeting()
	b.WriteString("  " + StyleCard.Render("💬 GRIM Status: "+greeting) + "\n\n")

	// Enhanced progress section
	done, total := m.overallProgress()

	b.WriteString("  " + StyleSectionTitle.Render("📊 SPRINT PROGRESS") + "\n")

	bar := renderProgressBar(done, total, 40)
	progressText := fmt.Sprintf("  %s", bar)
	b.WriteString(progressText + "\n")

	// Progress details
	completionText := fmt.Sprintf("  %s exercises completed out of %s total",
		StyleHighlight.Render(fmt.Sprintf("%d", done)),
		StyleHighlight.Render(fmt.Sprintf("%d", total)))
	b.WriteString(StyleBody.Render(completionText) + "\n\n")

	// Track overview
	if len(m.tracks) > 0 {
		b.WriteString("  " + StyleSectionTitle.Render("📚 AVAILABLE TRACKS") + "\n")
		for _, track := range m.tracks {
			trackProgress := renderProgressBar(track.completed, track.total, 20)
			trackLine := fmt.Sprintf("  %s %s  %s",
				StyleHighlight.Render("▸"),
				StyleBody.Render(track.name),
				trackProgress)
			b.WriteString(trackLine + "\n")
		}
		b.WriteString("\n")
	}

	// Enhanced controls
	b.WriteString("  " + StyleSectionTitle.Render("🎮 CONTROLS") + "\n")
	b.WriteString("  " + StyleKey.Render(" Enter ") + StyleFooter.Render(" Browse Exercise Catalog") + "\n")
	b.WriteString("  " + StyleKey.Render("   q   ") + StyleFooter.Render(" Exit GRIM System") + "\n")

	return b.String()
}

func (m Model) viewTrackSelect() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("  " + StyleSectionTitle.Render("Select Track") + "\n\n")

	for i, t := range m.tracks {
		cursor := "  "
		line := fmt.Sprintf("%s  ·  %d/%d complete", t.name, t.completed, t.total)
		if i == m.trackCursor {
			cursor = "> "
			b.WriteString(StyleSelected.Render(cursor+line) + "\n")
		} else {
			b.WriteString("  " + StyleBody.Render(line) + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString("  " + StyleKey.Render("[↑↓/jk]") + StyleFooter.Render(" Move") + "   ")
	b.WriteString(StyleKey.Render("[Enter]") + StyleFooter.Render(" Select") + "   ")
	b.WriteString(StyleKey.Render("[b]") + StyleFooter.Render(" Back") + "   ")
	b.WriteString(StyleKey.Render("[q]") + StyleFooter.Render(" Quit") + "\n")

	return b.String()
}

func (m Model) viewExerciseList() string {
	var b strings.Builder
	exercises := m.trackExercises()

	b.WriteString("\n")
	b.WriteString("  " + StyleTrack.Render(m.selectedTrack) + "\n\n")

	for i, entry := range exercises {
		ex := entry.Exercise
		name := ex.Metadata.Name
		locked := m.isLocked(ex)

		// Status icon
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

		// Difficulty badge
		diff := DifficultyStyle(ex.Spec.Difficulty).Render(ex.Spec.Difficulty)

		// Time
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

		if i == m.exerciseCursor {
			b.WriteString(StyleSelected.Render("> "+lineContent) + "\n")
		} else {
			b.WriteString("  " + lineContent + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString("  " + StyleKey.Render("[↑↓/jk]") + StyleFooter.Render(" Move") + "   ")
	b.WriteString(StyleKey.Render("[Enter]") + StyleFooter.Render(" Open") + "   ")
	b.WriteString(StyleKey.Render("[b]") + StyleFooter.Render(" Back") + "   ")
	b.WriteString(StyleKey.Render("[q]") + StyleFooter.Render(" Quit") + "\n")

	return b.String()
}

func (m Model) viewExerciseDetail() string {
	var b strings.Builder
	ex := m.selectedEx

	b.WriteString("\n")
	b.WriteString(m.renderGRIMTicket(ex) + "\n")

	// Learning outcomes
	if len(ex.Spec.LearningOutcomes) > 0 {
		b.WriteString(StyleSectionTitle.Render("  Learning Outcomes") + "\n")
		for _, lo := range ex.Spec.LearningOutcomes {
			b.WriteString("  • " + StyleBody.Render(lo) + "\n")
		}
		b.WriteString("\n")
	}

	// Tags
	if len(ex.Spec.Tags) > 0 {
		b.WriteString("  " + StyleDim.Render("Tags: "+strings.Join(ex.Spec.Tags, ", ")) + "\n")
	}

	// Time / Points
	meta := []string{}
	if ex.Spec.EstimatedTime != "" {
		meta = append(meta, "⏱ "+ex.Spec.EstimatedTime)
	}
	if ex.Spec.Points > 0 {
		meta = append(meta, fmt.Sprintf("★ %d pts", ex.Spec.Points))
	}
	if len(meta) > 0 {
		b.WriteString("  " + StyleDim.Render(strings.Join(meta, "   ")) + "\n")
	}

	b.WriteString("\n")

	// Footer keybinds
	locked := m.isLocked(ex)
	if locked {
		b.WriteString("  " + StyleDim.Render("[s] Start (locked)") + "   ")
	} else {
		b.WriteString("  " + StyleKey.Render("[s]") + StyleFooter.Render(" Start") + "   ")
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
	b.WriteString(StyleKey.Render("[b]") + StyleFooter.Render(" Back") + "   ")
	b.WriteString(StyleKey.Render("[q]") + StyleFooter.Render(" Quit") + "\n")

	return b.String()
}

func (m Model) renderGRIMTicket(ex *scenario.Exercise) string {
	const boxWidth = 68

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
	// Pad to boxWidth
	b.WriteString(header + "\n")

	sep := "  ├" + strings.Repeat("─", boxWidth) + "┤"
	b.WriteString(dim(sep) + "\n")

	// Word-wrapped description
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

func (m Model) viewHintPeek() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("  " + StyleSectionTitle.Render("Hint 1 (free)") + "\n")
	b.WriteString("  " + StyleDim.Render(strings.Repeat("─", 60)) + "\n\n")

	for _, line := range strings.Split(m.hintContent, "\n") {
		b.WriteString("  " + StyleBody.Render(line) + "\n")
	}

	b.WriteString("\n")
	b.WriteString("  " + StyleKey.Render("[b]") + StyleFooter.Render(" Back") + "\n")

	return b.String()
}

func (m Model) viewStartConfirm() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("  " + StyleSectionTitle.Render("Start Exercise") + "\n\n")
	b.WriteString("  " + StyleBody.Render("How would you like to proceed?") + "\n\n")

	choices := []string{
		"Work on it (drop into shell)",
		"Exit TUI",
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

// wordWrap word-wraps text to width chars per line.
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
		for _, word := range words {
			if current == "" {
				current = word
			} else if len(current)+1+len(word) <= width {
				current += " " + word
			} else {
				lines = append(lines, current)
				current = word
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

// lipgloss.Width is used for padding calculations; suppress unused import.
var _ = lipgloss.Width
