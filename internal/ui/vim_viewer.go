package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (a *App) vimViewerLines() []string {
	body := a.response.FormattedBody()
	if body == "" {
		return []string{"(empty response)"}
	}
	return strings.Split(body, "\n")
}

func (a *App) vimViewerHeight() int {
	h := a.height - 2
	if h < 1 {
		h = 1
	}
	return h
}

func (a *App) vimViewerClampOffset(lines []string) {
	max := len(lines) - a.vimViewerHeight()
	if max < 0 {
		max = 0
	}
	if a.vimViewerOffset > max {
		a.vimViewerOffset = max
	}
	if a.vimViewerOffset < 0 {
		a.vimViewerOffset = 0
	}
}

func (a *App) handleVimViewer(msg tea.KeyMsg) tea.Cmd {
	lines := a.vimViewerLines()
	h := a.vimViewerHeight()
	key := msg.String()

	switch key {
	case "q", "esc":
		a.showVimViewer = false

	case "j", "down":
		a.vimViewerOffset++

	case "k", "up":
		a.vimViewerOffset--

	case "ctrl+d":
		a.vimViewerOffset += h / 2

	case "ctrl+u":
		a.vimViewerOffset -= h / 2

	case "ctrl+f", " ":
		a.vimViewerOffset += h

	case "ctrl+b":
		a.vimViewerOffset -= h

	case "g":
		a.vimViewerOffset = 0

	case "G":
		a.vimViewerOffset = len(lines)
	}

	a.vimViewerClampOffset(lines)
	return nil
}

func (a *App) renderVimViewer() string {
	lines := a.vimViewerLines()
	h := a.vimViewerHeight()
	w := a.width

	a.vimViewerClampOffset(lines)

	start := a.vimViewerOffset
	end := start + h
	if end > len(lines) {
		end = len(lines)
	}

	lineNumWidth := len(fmt.Sprintf("%d", len(lines)))
	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#4e4e4e"))
	contentWidth := w - lineNumWidth - 2
	if contentWidth < 1 {
		contentWidth = 1
	}

	var sb strings.Builder
	for i := start; i < end; i++ {
		lineNum := lineNumStyle.Render(fmt.Sprintf("%*d", lineNumWidth, i+1))
		content := lines[i]
		if len([]rune(content)) > contentWidth {
			content = string([]rune(content)[:contentWidth])
		}
		sb.WriteString(lineNum + " " + content + "\n")
	}

	for i := end - start; i < h; i++ {
		tilde := lipgloss.NewStyle().Foreground(lipgloss.Color("#4e4e4e")).Render("~")
		sb.WriteString(tilde + "\n")
	}

	pct := 0
	total := len(lines)
	if total > 0 {
		pct = ((start + h) * 100) / total
		if pct > 100 {
			pct = 100
		}
	}

	statusLeft := lipgloss.NewStyle().
		Background(lipgloss.Color("#00d7ff")).
		Foreground(lipgloss.Color("#000000")).
		Bold(true).
		Render(" NORMAL ")

	info := ""
	if a.response.response != nil {
		statusColor := "#00d700"
		if a.response.response.IsClientError() {
			statusColor = "#d7d700"
		} else if a.response.response.IsServerError() {
			statusColor = "#d70000"
		}
		info = lipgloss.NewStyle().
			Foreground(lipgloss.Color(statusColor)).
			Bold(true).
			Render(fmt.Sprintf(" %d ", a.response.response.Status))
	}

	pos := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262")).
		Render(fmt.Sprintf(" %d/%d  %d%% ", start+1, total, pct))

	hints := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4e4e4e")).
		Render("  j↓  k↑  ctrl+d ½↓  ctrl+u ½↑  g top  G bottom  q close")

	statusLine := lipgloss.NewStyle().
		Width(w).
		Background(lipgloss.Color("#1a1a1a")).
		Render(statusLeft + info + hints + pos)

	return sb.String() + statusLine
}
