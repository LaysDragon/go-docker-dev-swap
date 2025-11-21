package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type modelOptions struct {
	maxLines               int
	initialDebuggerEnabled bool
	actionChan             chan<- Action
}

type model struct {
	width  int
	height int

	workLines      []string
	containerLines []string
	maxLines       int

	debuggerEnabled bool
	statusMessage   string

	actionChan chan<- Action
}

type workLogMsg string

type containerLogMsg string

type debuggerStateMsg struct {
	enabled bool
}

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	panelStyle = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240")).Padding(0, 1)
	keyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("60"))
)

func newModel(opts modelOptions) model {
	return model{
		maxLines:        opts.maxLines,
		debuggerEnabled: opts.initialDebuggerEnabled,
		actionChan:      opts.actionChan,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.KeyMsg:
		switch v.String() {
		case "ctrl+c", "q":
			m.sendAction(Action{Type: ActionQuit})
			return m, tea.Quit
		case "d":
			m.debuggerEnabled = !m.debuggerEnabled
			state := "關閉"
			if m.debuggerEnabled {
				state = "開啟"
			}
			m.statusMessage = fmt.Sprintf("Debugger 已切換為%s，正在套用...", state)
			m.sendAction(Action{Type: ActionToggleDebugger, Enabled: m.debuggerEnabled})
		}
	case tea.WindowSizeMsg:
		m.width = v.Width
		m.height = v.Height
	case workLogMsg:
		m.workLines = appendLine(m.workLines, string(v), m.maxLines)
	case containerLogMsg:
		m.containerLines = appendLine(m.containerLines, string(v), m.maxLines)
	case debuggerStateMsg:
		m.debuggerEnabled = v.enabled
		state := "關閉"
		if v.enabled {
			state = "開啟"
		}
		m.statusMessage = fmt.Sprintf("Debugger 已%s", state)
	}

	return m, nil
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "載入介面中..."
	}

	keyRowHeight := 1
	available := m.height - keyRowHeight
	if available < 6 {
		available = m.height
	}

	topHeight := available / 2
	if topHeight < 4 {
		topHeight = available
	}

	bottomHeight := available - topHeight
	if bottomHeight < 4 {
		bottomHeight = topHeight
	}

	work := m.renderPanel("工作日誌", m.workLines, topHeight)
	container := m.renderPanel("容器輸出", m.containerLines, bottomHeight)
	return lipgloss.JoinVertical(lipgloss.Left, work, container, m.renderKeyRow())
}

func (m model) renderPanel(title string, lines []string, height int) string {
	if height < 3 {
		height = 3
	}
	bodyLines := lastLines(lines, height-3)
	body := strings.Join(bodyLines, "\n")

	titleView := titleStyle.Render(title)
	panelView := panelStyle.Width(m.width - 2).MaxWidth(m.width).Height(height - 3).MaxHeight(height - 1).Render(body)
	return lipgloss.JoinVertical(lipgloss.Left, titleView, panelView)
}

func (m model) renderKeyRow() string {
	statusColor := lipgloss.Color("160")
	state := "OFF"
	if m.debuggerEnabled {
		statusColor = lipgloss.Color("40")
		state = "ON"
	}
	stateView := lipgloss.NewStyle().Bold(true).Foreground(statusColor).Render(state)
	//lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("60"))
	info := fmt.Sprintf("[D] Debugger %s", stateView)
	otherInfo := "   [Ctrl+C] 退出"

	if m.statusMessage != "" {
		otherInfo = fmt.Sprintf("%s  •  %s", otherInfo, m.statusMessage)
	}
	info = info + keyStyle.Render(otherInfo)
	return keyStyle.Width(m.width).Padding(0, 1).Render(info)
}

func (m model) sendAction(a Action) {
	if m.actionChan == nil {
		return
	}
	select {
	case m.actionChan <- a:
	default:
	}
}

func appendLine(lines []string, line string, max int) []string {
	if line == "" {
		return lines
	}
	lines = append(lines, line)
	if max > 0 && len(lines) > max {
		offset := len(lines) - max
		lines = lines[offset:]
	}
	return lines
}

func lastLines(lines []string, count int) []string {
	if count <= 0 || len(lines) <= count {
		return lines
	}
	return lines[len(lines)-count:]
}
