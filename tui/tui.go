package tui

import (
	"fmt"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Step int

const (
	StepMaintainerName Step = iota
	StepMaintainerEmail
	StepRelease
	StepWorkDir
	StepOutDir
	StepJobs
	StepSkipDeps
	StepOnly
	StepConfirm
	StepDone
)

type MenuOption struct {
	Key  string
	Text string
	Desc string
}

type Model struct {
	Step      Step
	Width     int
	Height    int
	Choices   map[string]string
	Cursor    int
	TextInput string
	CursorPos int
	Quitting  bool
	Confirmed bool
	Distro    string
	Codename  string
	Releases  []string
}

func NewModel(distro, codename string, releases []string) Model {
	m := Model{
		Step:     StepMaintainerName,
		Choices:  make(map[string]string),
		Distro:   distro,
		Codename: codename,
		Releases: releases,
	}
	m.initInput()
	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.Quitting = true
			return m, tea.Quit
		}
	}

	if m.isMenuStep() {
		return m.updateMenu(msg)
	}
	return m.updateInput(msg)
}

func (m Model) isMenuStep() bool {
	switch m.Step {
	case StepMaintainerName, StepMaintainerEmail, StepWorkDir, StepOutDir, StepOnly:
		return false
	}
	return true
}

func (m Model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	opts := m.getMenuOptions()
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
			}
		case "down", "j":
			if m.Cursor < len(opts)-1 {
				m.Cursor++
			}
		case "enter":
			if len(opts) > 0 {
				m.selectOption(opts[m.Cursor])
			}
		}
	}
	return m, nil
}

func (m Model) updateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "enter":
			m.commitInput()
		case "backspace":
			if m.CursorPos > 0 {
				m.TextInput = m.TextInput[:m.CursorPos-1] + m.TextInput[m.CursorPos:]
				m.CursorPos--
			}
		case "left":
			if m.CursorPos > 0 {
				m.CursorPos--
			}
		case "right":
			if m.CursorPos < len(m.TextInput) {
				m.CursorPos++
			}
		default:
			if len(km.String()) == 1 {
				m.TextInput = m.TextInput[:m.CursorPos] + km.String() + m.TextInput[m.CursorPos:]
				m.CursorPos++
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.Quitting {
		return ""
	}

	w := m.Width
	if w == 0 {
		w = 80
	}
	h := m.Height
	if h == 0 {
		h = 24
	}

	border := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
	title := lipgloss.NewStyle().Bold(true).Underline(true)
	highlight := lipgloss.NewStyle().Bold(true).Reverse(true).Padding(0, 1)
	sectionTitle := lipgloss.NewStyle().Bold(true)

	stepLabel, stepDesc := m.getStepInfo()
	header := title.Render(fmt.Sprintf("COSMIC Debian Builder (%s %s)", m.Distro, m.Codename))
	stepLine := sectionTitle.Render(stepLabel)

	var body string
	if m.isMenuStep() {
		opts := m.getMenuOptions()
		var lines []string
		for i, opt := range opts {
			label := fmt.Sprintf("  %s", opt.Text)
			if opt.Desc != "" {
				label += fmt.Sprintf(" (%s)", opt.Desc)
			}
			if i == m.Cursor {
				lines = append(lines, highlight.Render(label))
			} else {
				lines = append(lines, label)
			}
		}
		body = strings.Join(lines, "\n")
	} else {
		prompt := m.getInputPrompt()
		display := m.TextInput
		if m.CursorPos < len(display) {
			display = display[:m.CursorPos] + "█" + display[m.CursorPos+1:]
		} else {
			display = display + "█"
		}
		body = fmt.Sprintf("%s\n\n  > %s", prompt, display)
	}

	summary := m.buildSummary()
	var sidebar string
	if summary != "" {
		sideBox := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).Width(30)
		sidebar = sideBox.Render(sectionTitle.Render("Configuration") + "\n" + summary)
	}

	mainContent := lipgloss.JoinVertical(lipgloss.Left,
		header,
		strings.Repeat("─", 50),
		"",
		stepLine,
		stepDesc,
		"",
		body,
		"",
		strings.Repeat("─", 50),
		"Navigate: Up/Down  Select: Enter  Quit: Ctrl+C",
	)

	var page string
	if sidebar != "" {
		page = lipgloss.JoinHorizontal(lipgloss.Top, mainContent, "  ", sidebar)
	} else {
		page = mainContent
	}

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, border.Render(page))
}

func (m Model) getStepInfo() (string, string) {
	switch m.Step {
	case StepMaintainerName:
		return "[ Maintainer Name ]", "Enter your full name for package metadata"
	case StepMaintainerEmail:
		return "[ Maintainer Email ]", "Enter your email for package metadata"
	case StepRelease:
		return "[ Source Mode ]", "Select an epoch tag or build from the main branch HEAD"
	case StepWorkDir:
		return "[ Workspace ]", "Directory for compilation and source code"
	case StepOutDir:
		return "[ Output ]", "Directory where .deb files will be saved"
	case StepJobs:
		return "[ Parallelism ]", "Number of concurrent compilation jobs"
	case StepSkipDeps:
		return "[ Dependency Check ]", "Automatically install build dependencies?"
	case StepOnly:
		return "[ Targeted Build ]", "Build specific component (leave empty for all)"
	case StepConfirm:
		return "[ Confirmation ]", "Verify settings and start synthesis"
	}
	return "", ""
}

func (m Model) getMenuOptions() []MenuOption {
	switch m.Step {
	case StepRelease:
		opts := []MenuOption{
			{Key: "branch", Text: "Latest (main branch)", Desc: "HEAD of each repo's main branch — newest code"},
		}
		for _, r := range m.Releases {
			opts = append(opts, MenuOption{Key: r, Text: r, Desc: "epoch tag"})
		}
		return opts
	case StepJobs:
		n := runtime.NumCPU()
		return []MenuOption{
			{fmt.Sprintf("%d", n), "System Default", fmt.Sprintf("Use all %d cores", n)},
			{"1", "Serial", "Use 1 core (safest)"},
			{"4", "Quad", "Use 4 cores"},
		}
	case StepSkipDeps:
		return []MenuOption{
			{"n", "Yes", "Perform automatic installation"},
			{"y", "No", "Skip installation (I'll handle it myself)"},
		}
	case StepConfirm:
		return []MenuOption{
			{"y", "Proceed", "Start the build process"},
			{"n", "Abort", "Exit without building"},
		}
	}
	return nil
}

func (m *Model) selectOption(opt MenuOption) {
	switch m.Step {
	case StepRelease:
		m.Choices["release"] = opt.Key
		m.Step = StepWorkDir
		m.Cursor = 0
	case StepJobs:
		m.Choices["jobs"] = opt.Key
		m.Step = StepSkipDeps
		m.Cursor = 0
	case StepSkipDeps:
		m.Choices["skip_deps"] = opt.Key
		m.Step = StepOnly
		m.initInput()
	case StepConfirm:
		if opt.Key == "y" {
			m.Confirmed = true
			m.Quitting = true
		} else {
			m.Quitting = true
		}
	}
}

func (m *Model) initInput() {
	m.TextInput = m.getInputDefault()
	m.CursorPos = len(m.TextInput)
}

func (m Model) getInputDefault() string {
	switch m.Step {
	case StepMaintainerName:
		return "cosmic-deb"
	case StepMaintainerEmail:
		return "cosmic-deb@example.com"
	case StepWorkDir:
		return "cosmic-work"
	case StepOutDir:
		return "cosmic-packages"
	case StepOnly:
		return ""
	}
	return ""
}

func (m Model) getInputPrompt() string {
	switch m.Step {
	case StepMaintainerName:
		return "Name:"
	case StepMaintainerEmail:
		return "Email:"
	case StepWorkDir:
		return "Path:"
	case StepOutDir:
		return "Path:"
	case StepOnly:
		return "Component (e.g. cosmic-term):"
	}
	return ">"
}

func (m *Model) commitInput() {
	val := m.TextInput
	if val == "" {
		val = m.getInputDefault()
	}
	switch m.Step {
	case StepMaintainerName:
		m.Choices["maintainer_name"] = val
		m.Step = StepMaintainerEmail
	case StepMaintainerEmail:
		m.Choices["maintainer_email"] = val
		m.Step = StepRelease
		m.Cursor = 0
	case StepWorkDir:
		m.Choices["workdir"] = val
		m.Step = StepOutDir
	case StepOutDir:
		m.Choices["outdir"] = val
		m.Step = StepJobs
		m.Cursor = 0
	case StepOnly:
		m.Choices["only"] = val
		m.Step = StepConfirm
		m.Cursor = 0
	}
	m.initInput()
}

func (m Model) buildSummary() string {
	var lines []string
	if v, ok := m.Choices["release"]; ok {
		if v == "branch" {
			lines = append(lines, "Src:  main branch HEAD")
		} else {
			lines = append(lines, fmt.Sprintf("Tag:  %s", v))
		}
	} else {
		lines = append(lines, "Src:  (pending selection)")
	}
	if m.Distro != "" {
		lines = append(lines, fmt.Sprintf("OS:   %s %s", m.Distro, m.Codename))
	}
	if v, ok := m.Choices["maintainer_name"]; ok {
		lines = append(lines, fmt.Sprintf("User: %s", v))
	}
	if v, ok := m.Choices["outdir"]; ok {
		lines = append(lines, fmt.Sprintf("Out:  %s", v))
	}
	return strings.Join(lines, "\n")
}

func RunWizard(distro, codename string, releases []string) (map[string]string, bool, error) {
	p := tea.NewProgram(NewModel(distro, codename, releases), tea.WithAltScreen())
	m, err := p.Run()
	if err != nil {
		return nil, false, err
	}
	final := m.(Model)
	return final.Choices, final.Confirmed, nil
}
