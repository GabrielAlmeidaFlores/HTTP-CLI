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
	total := len(lines)
	h := a.vimViewerHeight()
	key := msg.String()

	switch key {
	case "q", "esc":
		a.showVimViewer = false

	case "j", "down":
		if a.vimViewerCursor < total-1 {
			a.vimViewerCursor++
		}

	case "k", "up":
		if a.vimViewerCursor > 0 {
			a.vimViewerCursor--
		}

	case "ctrl+d":
		a.vimViewerCursor += h / 2
		if a.vimViewerCursor >= total {
			a.vimViewerCursor = total - 1
		}

	case "ctrl+u":
		a.vimViewerCursor -= h / 2
		if a.vimViewerCursor < 0 {
			a.vimViewerCursor = 0
		}

	case "ctrl+f", " ":
		a.vimViewerCursor += h
		if a.vimViewerCursor >= total {
			a.vimViewerCursor = total - 1
		}

	case "ctrl+b":
		a.vimViewerCursor -= h
		if a.vimViewerCursor < 0 {
			a.vimViewerCursor = 0
		}

	case "g":
		a.vimViewerCursor = 0

	case "G":
		a.vimViewerCursor = total - 1
		if a.vimViewerCursor < 0 {
			a.vimViewerCursor = 0
		}
	}

	a.vimViewerSyncScroll(h)
	return nil
}

func (a *App) vimViewerSyncScroll(h int) {
	if a.vimViewerCursor < a.vimViewerOffset {
		a.vimViewerOffset = a.vimViewerCursor
	}
	if a.vimViewerCursor >= a.vimViewerOffset+h {
		a.vimViewerOffset = a.vimViewerCursor - h + 1
	}
	if a.vimViewerOffset < 0 {
		a.vimViewerOffset = 0
	}
}

func (a *App) renderVimViewer() string {
	lines := a.vimViewerLines()
	h := a.vimViewerHeight()
	w := a.width
	total := len(lines)

	a.vimViewerSyncScroll(h)

	start := a.vimViewerOffset
	end := start + h
	if end > total {
		end = total
	}

	lineNumWidth := len(fmt.Sprintf("%d", total))
	if lineNumWidth < 1 {
		lineNumWidth = 1
	}
	contentWidth := w - lineNumWidth - 2
	if contentWidth < 1 {
		contentWidth = 1
	}

	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(a.theme.LineNumFg))
	lineNumCursorStyle := secondaryStyle(a.theme).Bold(true)
	cursorLineStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(a.theme.VimCursorBg)).
		Width(w)

	var sb strings.Builder
	for i := start; i < end; i++ {
		isCursor := i == a.vimViewerCursor

		numStyle := lineNumStyle
		if isCursor {
			numStyle = lineNumCursorStyle
		}
		lineNum := numStyle.Render(fmt.Sprintf("%*d", lineNumWidth, i+1))

		content := lines[i]
		if len([]rune(content)) > contentWidth {
			content = string([]rune(content)[:contentWidth])
		}

		row := lineNum + " " + content
		if isCursor {
			row = cursorLineStyle.Render(row)
		}
		sb.WriteString(row + "\n")
	}

	for i := end - start; i < h; i++ {
		tilde := lipgloss.NewStyle().Foreground(lipgloss.Color(a.theme.LineNumFg)).Render("~")
		sb.WriteString(tilde + "\n")
	}

	pct := 0
	if total > 0 {
		pct = ((a.vimViewerCursor + 1) * 100) / total
		if pct > 100 {
			pct = 100
		}
	}

	statusLeft := lipgloss.NewStyle().
		Background(lipgloss.Color(a.theme.Primary)).
		Foreground(lipgloss.Color(a.theme.Black)).
		Bold(true).
		Render(" NORMAL ")

	info := ""
	if a.response.GetResponse() != nil {
		info = lipgloss.NewStyle().
			Foreground(lipgloss.Color(a.response.StatusColor(a.cfg.UI.Theme))).
			Bold(true).
			Render(fmt.Sprintf(" %d ", a.response.GetResponse().Status))
	}

	pos := dimStyle(a.theme).
		Render(fmt.Sprintf(" %d/%d  %d%% ", a.vimViewerCursor+1, total, pct))

	hints := buildHintsLine(a.keybindMgr, "vim_viewer", "", a.theme)

	statusLine := lipgloss.NewStyle().
		Width(w).
		Background(lipgloss.Color(a.theme.VimStatusBg)).
		Render(statusLeft + info + hints + pos)

	return sb.String() + statusLine
}
