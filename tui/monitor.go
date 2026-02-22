package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ProgressMsg struct {
	Step  int
	Total int
	Name  string
}

type LogMsg string

type DoneMsg struct {
	Err error
}

type MonitorModel struct {
	CurrentStep int
	TotalSteps  int
	StepName    string
	Logs        []string
	Err         error
	Done        bool
	Width       int
	Height      int
}

func (m MonitorModel) Init() tea.Cmd {
	return nil
}

func (m MonitorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
	case ProgressMsg:
		m.CurrentStep = msg.Step
		m.TotalSteps = msg.Total
		m.StepName = msg.Name
	case LogMsg:
		m.Logs = append(m.Logs, string(msg))
		if len(m.Logs) > 100 {
			m.Logs = m.Logs[1:]
		}
	case DoneMsg:
		m.Err = msg.Err
		m.Done = true
		return m, tea.Quit
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m MonitorModel) View() string {
	w := m.Width
	if w == 0 {
		w = 80
	}
	h := m.Height
	if h == 0 {
		h = 24
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Underline(true)
	borderStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
	sectionTitle := lipgloss.NewStyle().Bold(true)
	progressStyle := lipgloss.NewStyle().Bold(true).Reverse(true)

	header := titleStyle.Render("COSMIC Build Monitor")

	progWidth := w - 15
	if progWidth > 50 {
		progWidth = 50
	}

	percent := 0.0
	if m.TotalSteps > 0 {
		percent = float64(m.CurrentStep) / float64(m.TotalSteps)
	}

	filled := int(float64(progWidth) * percent)
	empty := progWidth - filled

	bar := "[" + progressStyle.Render(strings.Repeat("█", filled)) + strings.Repeat(" ", empty) + "]"
	progressLine := fmt.Sprintf("%s %d/%d - %s", bar, m.CurrentStep, m.TotalSteps, m.StepName)

	logBoxHeight := h - 12
	if logBoxHeight < 5 {
		logBoxHeight = 5
	}

	var logLines []string
	start := len(m.Logs) - logBoxHeight
	if start < 0 {
		start = 0
	}
	for i := start; i < len(m.Logs); i++ {
		line := m.Logs[i]
		if len(line) > w-10 {
			line = line[:w-13] + "..."
		}
		logLines = append(logLines, "  "+line)
	}
	logContent := strings.Join(logLines, "\n")

	view := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		sectionTitle.Render("Build Progress:"),
		progressLine,
		"",
		sectionTitle.Render("Activity Log:"),
		logContent,
		"",
		"Press Ctrl+C to abort",
	)

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, borderStyle.Render(view))
}
