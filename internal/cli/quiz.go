package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"gymctl/internal/scenario"
)

type quizOptions struct {
	spec string
}

type quizSaveMsg struct {
	questionIndex int
	letter        string
	err           error
}

type quizModel struct {
	exerciseName string
	labPath      string
	questions    []*scenario.LabMCQPrompt
	current      int
	cursor       int
	status       string
	saving       bool
	width        int
	height       int
	quitting     bool
}

func newQuizCmd() *cobra.Command {
	opts := &quizOptions{}
	cmd := &cobra.Command{
		Use:   "quiz [exercise-name]",
		Short: "Answer quiz questions from lab.md",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			exercise, exerciseDir, err := resolveDescribeTarget(args, opts.spec)
			if err != nil {
				return err
			}

			labPath := filepath.Join(exerciseDir, "lab.md")
			data, err := os.ReadFile(labPath)
			if err != nil {
				return fmt.Errorf("read lab markdown: %w", err)
			}

			sections, err := scenario.ParseLabSections(data)
			if err != nil {
				return fmt.Errorf("parse lab markdown: %w", err)
			}

			questions := make([]*scenario.LabMCQPrompt, 0)
			for _, section := range sections {
				if section.MCQ != nil {
					prompt := *section.MCQ
					questions = append(questions, &prompt)
				}
			}
			if len(questions) == 0 {
				return fmt.Errorf("no quiz questions found in %s", labPath)
			}

			m := newQuizModel(exercise.Metadata.Name, labPath, questions)
			p := tea.NewProgram(m)
			result, err := p.Run()
			if err != nil {
				return err
			}

			finalModel, ok := result.(quizModel)
			if !ok {
				return nil
			}
			if finalModel.status != "" {
				fmt.Fprintln(cmd.OutOrStdout(), finalModel.status)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Saved quiz answers in %s\n", labPath)
			fmt.Fprintf(cmd.OutOrStdout(), "Run %q to grade your answers.\n", "gymctl check "+exercise.Metadata.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.spec, "spec", "", "Path to an exercise spec file")
	return cmd
}

func newQuizModel(exerciseName, labPath string, questions []*scenario.LabMCQPrompt) quizModel {
	m := quizModel{
		exerciseName: exerciseName,
		labPath:      labPath,
		questions:    questions,
		status:       "Select an answer with Enter. Use left/right to move between questions.",
	}
	m.syncCursorToQuestion()
	return m
}

func (m quizModel) Init() tea.Cmd { return nil }

func (m quizModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case quizSaveMsg:
		m.saving = false
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not save answer: %v", msg.err)
			return m, nil
		}
		q := m.questions[msg.questionIndex]
		q.Selected = []string{msg.letter}
		for i := range q.Options {
			q.Options[i].Selected = q.Options[i].Letter == msg.letter
		}
		m.status = fmt.Sprintf("Saved %s -> %s", q.ID, msg.letter)
		return m, nil
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m quizModel) View() tea.View {
	if len(m.questions) == 0 {
		return tea.NewView("No quiz questions found.\n")
	}

	q := m.questions[m.current]
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Quiz: %s\n\n", m.exerciseName))
	b.WriteString(renderQuizProgress(m.questions, m.current))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Question %d/%d  %s\n", m.current+1, len(m.questions), q.ID))
	b.WriteString(strings.Repeat("-", max(24, len(q.ID)+16)))
	b.WriteString("\n")
	if q.Question != "" {
		b.WriteString(q.Question)
		b.WriteString("\n\n")
	}

	for i, option := range q.Options {
		cursor := "  "
		if i == m.cursor {
			cursor = "› "
		}
		selected := "[ ]"
		if option.Selected {
			selected = "[x]"
		}
		b.WriteString(fmt.Sprintf("%s%s %s. %s\n", cursor, selected, option.Letter, option.Text))
	}

	b.WriteString("\n")
	if m.saving {
		b.WriteString("Saving selection...\n")
	} else if m.status != "" {
		b.WriteString(m.status)
		b.WriteString("\n")
	}
	b.WriteString("\nKeys: ↑/↓ move  enter select  ←/→ question  q quit\n")
	v := tea.NewView(b.String())
	v.AltScreen = true
	v.WindowTitle = "gymctl quiz"
	return v
}

func (m quizModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if len(m.questions) == 0 {
		return m, tea.Quit
	}

	switch msg.String() {
	case "ctrl+c", "q":
		m.quitting = true
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case "down", "j":
		if m.cursor < len(m.questions[m.current].Options)-1 {
			m.cursor++
		}
		return m, nil
	case "left", "h":
		if m.current > 0 {
			m.current--
			m.syncCursorToQuestion()
		}
		return m, nil
	case "right", "l":
		if m.current < len(m.questions)-1 {
			m.current++
			m.syncCursorToQuestion()
		}
		return m, nil
	case "enter", " ":
		if m.saving {
			return m, nil
		}
		option := m.questions[m.current].Options[m.cursor]
		m.saving = true
		m.status = fmt.Sprintf("Saving %s -> %s...", m.questions[m.current].ID, option.Letter)
		questionIndex := m.current
		return m, func() tea.Msg {
			data, err := os.ReadFile(m.labPath)
			if err != nil {
				return quizSaveMsg{questionIndex: questionIndex, letter: option.Letter, err: fmt.Errorf("read lab markdown: %w", err)}
			}
			updated, err := scenario.SetMCQSelection(data, m.questions[questionIndex].ID, option.Letter)
			if err != nil {
				return quizSaveMsg{questionIndex: questionIndex, letter: option.Letter, err: err}
			}
			if err := os.WriteFile(m.labPath, updated, 0o644); err != nil {
				return quizSaveMsg{questionIndex: questionIndex, letter: option.Letter, err: fmt.Errorf("write lab markdown: %w", err)}
			}
			return quizSaveMsg{questionIndex: questionIndex, letter: option.Letter}
		}
	}

	return m, nil
}

func (m *quizModel) syncCursorToQuestion() {
	if len(m.questions) == 0 || len(m.questions[m.current].Options) == 0 {
		m.cursor = 0
		return
	}
	q := m.questions[m.current]
	for i, option := range q.Options {
		if option.Selected {
			m.cursor = i
			return
		}
	}
	m.cursor = 0
}

func renderQuizProgress(questions []*scenario.LabMCQPrompt, current int) string {
	parts := make([]string, 0, len(questions))
	for i, q := range questions {
		status := "○"
		if len(q.Selected) > 0 {
			status = "●"
		}
		if i == current {
			parts = append(parts, fmt.Sprintf("[%s %s]", status, q.ID))
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %s", status, q.ID))
	}
	return strings.Join(parts, "   ")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
