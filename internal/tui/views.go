package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"gymctl/internal/environment"
	"gymctl/internal/pet"
	"gymctl/internal/scenario"
)

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
	b.WriteString("  " + renderHomeHeader() + "\n\n")

	done, total := m.overallProgress()

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

	b.WriteString("  " + renderPanel("jerry says", m.greeting) + "\n")
	b.WriteString("  " + renderPanel("progress", progressText) + "\n")
	b.WriteString("  " + renderPanel("tracks", tracksBody) + "\n")

	if active := m.activeExerciseName(); active != "" {
		b.WriteString("  " + renderPanel("active exercise",
			StyleHighlight.Render("◉ ")+StyleBody.Render(active)+StyleFooter.Render(" — press enter to browse")) + "\n")
	}

	b.WriteString("  " + renderPanel("controls", strings.Join([]string{
		StyleKey.Render("enter") + StyleFooter.Render(" browse exercises"),
		StyleKey.Render("q") + StyleFooter.Render("     quit"),
	}, "\n")) + "\n")

	// Jerry idles at the bottom of the home screen, blinking.
	b.WriteString("\n")
	for _, ln := range pet.Sprite(pet.Idle, m.frame) {
		b.WriteString("  " + StyleDim.Render(ln) + "\n")
	}

	return b.String()
}

func (m Model) viewTrackSelect() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("  " + StyleSectionTitle.Render("select a track") + "\n\n")

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
	b.WriteString("  " + renderPanel("tracks", strings.Join(lines, "\n")) + "\n")

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

		if i == m.exerciseCursor {
			lines = append(lines, StyleSelected.Render(fmt.Sprintf("> %d. %s", i+1, lineContent)))
		} else {
			lines = append(lines, fmt.Sprintf("  %d. %s", i+1, lineContent))
		}
	}
	if len(exercises) == 0 {
		lines = append(lines, StyleDim.Render("No exercises in this track."))
	}

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

	b.WriteString("  " + renderPanel("exercises", strings.Join(lines, "\n")+scrollHint) + "\n")

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
	b.WriteString(m.renderBrief(ex) + "\n")

	var brief []string

	if len(ex.Spec.LearningOutcomes) > 0 {
		for _, lo := range ex.Spec.LearningOutcomes {
			brief = append(brief, "• "+StyleBody.Render(lo))
		}
	}

	if len(ex.Spec.Tags) > 0 {
		brief = append(brief, StyleDim.Render("tags: "+strings.Join(ex.Spec.Tags, ", ")))
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
		b.WriteString("  " + renderPanel("brief", strings.Join(brief, "\n")) + "\n")
	}

	b.WriteString("\n")
	envContent := strings.TrimSpace(m.renderEnvironmentPanel(ex))
	if envContent == "" {
		envContent = StyleDim.Render("No environment info available")
	}
	b.WriteString("  " + renderPanel("environment", envContent) + "\n")
	b.WriteString("\n")
	b.WriteString(m.renderPetBlock() + "\n\n")

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
		statusStyle := StyleDim
		switch {
		case strings.Contains(m.statusLine, "failed") || strings.Contains(m.statusLine, "error"):
			statusStyle = StyleError
		case strings.Contains(m.statusLine, "complete"):
			statusStyle = StyleSuccess
		}
		b.WriteString("  " + renderPanel("latest", statusStyle.Render(m.statusLine)) + "\n")
	}

	return b.String()
}

// renderBrief renders the flat, minimal exercise brief: a bold lowercase title,
// a difficulty marker, a thin rule, and the word-wrapped description. No
// box-art, no ticket metadata. Reads from Brief data when present.
func (m Model) renderBrief(ex *scenario.Exercise) string {
	width := m.width - 4
	if width < 50 {
		width = 50
	}
	if width > 96 {
		width = 96
	}

	var summary, description string
	if t := ex.Spec.Brief; t != nil {
		summary = t.Summary
		description = t.Description
	}
	if summary == "" {
		summary = ex.Metadata.Title
	}
	if summary == "" {
		summary = ex.Metadata.Name
	}
	if description == "" {
		description = ex.Spec.Description
	}

	var b strings.Builder

	title := StyleSectionTitle.Render(strings.ToLower(summary))
	if diff := ex.Spec.Difficulty; diff != "" {
		marker := StyleHighlight.Render("●") + " " + DifficultyStyle(diff).Render(diff)
		gap := width - lipgloss.Width(title) - lipgloss.Width(marker)
		if gap < 2 {
			gap = 2
		}
		b.WriteString("  " + title + strings.Repeat(" ", gap) + marker + "\n")
	} else {
		b.WriteString("  " + title + "\n")
	}

	b.WriteString("  " + StyleTicketBorder.Render(strings.Repeat("─", width)) + "\n")

	for _, line := range wordWrap(description, width) {
		if line == "" {
			b.WriteString("\n")
		} else {
			b.WriteString("  " + StyleBody.Render(line) + "\n")
		}
	}

	return b.String()
}

// petMoodForSelected returns Jerry's resting mood for the selected exercise:
// celebrate when it's solved, otherwise nervously watching you poke at his mess.
// Lifecycle actions (check/reset/shell) override this via the petMood field.
func (m Model) petMoodForSelected() pet.Mood {
	if m.selectedEx != nil {
		if st, ok := m.prog.Exercises[m.selectedEx.Metadata.Name]; ok && st.Status == "completed" {
			return pet.Celebrate
		}
	}
	return pet.Nervous
}

// renderPetBlock draws Jerry's animated sprite plus a mood caption. The frame
// counter (driven by tickMsg) makes him blink.
func (m Model) renderPetBlock() string {
	var b strings.Builder
	for _, ln := range pet.Sprite(m.petMood, m.frame) {
		b.WriteString("  " + StyleDim.Render(ln) + "\n")
	}
	b.WriteString("  " + StyleFooter.Render("jerry · "+m.petMood.Label()))
	return b.String()
}

// renderPanel renders a flat panel: a dim lowercase title over the body, no
// border and no fill — the de-boxed, codex-style look.
func renderPanel(title, body string) string {
	header := StyleCardTitle.Render(title)
	content := StyleCardBody.Render(strings.TrimSpace(body))
	return header + "\n" + content
}

func (m Model) viewHintPeek() string {
	var b strings.Builder

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

func wordWrap(text string, width int) []string {
	var lines []string
	for i, paragraph := range strings.Split(strings.TrimRight(text, "\n"), "\n\n") {
		paragraph = strings.TrimSpace(strings.ReplaceAll(paragraph, "\n", " "))
		if i > 0 {
			lines = append(lines, "")
		}
		if paragraph == "" {
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
