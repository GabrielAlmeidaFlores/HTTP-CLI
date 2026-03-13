package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/user/http-cli/internal/ui/keybindings"
)

func (a *App) renderTopBar() string {
	method := ""
	url := ""
	if a.selectedReq != nil {
		method = string(a.selectedReq.Method)
		url = a.selectedReq.URL
	}

	methodStyle := lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1).
		Foreground(lipgloss.Color(methodColor(method, a.cfg.UI.Theme)))

	urlStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Width(a.width - 25)

	sendStyle := lipgloss.NewStyle().
		Bold(true).
		Padding(0, 2).
		Background(lipgloss.Color(a.theme.Success)).
		Foreground(lipgloss.Color(a.theme.Black))

	executing := ""
	if a.executing {
		executing = " ..."
	}

	execKey := "ctrl+e"
	for _, h := range a.keybindMgr.GetHints("editor", "") {
		if h.Action == "execute" || h.Action == "execute_request" {
			if len(h.Keys) > 0 {
				execKey = h.Keys[0]
			}
			break
		}
	}
	sendLabel := fmt.Sprintf("Send [%s]", execKey) + executing

	bar := lipgloss.JoinHorizontal(lipgloss.Top,
		methodStyle.Render(fmt.Sprintf("[%s]", method)),
		urlStyle.Render(url),
		sendStyle.Render(sendLabel),
	)

	return lipgloss.NewStyle().
		Width(a.width).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(a.cfg.UI.Theme.BlurBorder)).
		Render(bar)
}

func (a *App) renderMainArea() string {
	leftPanel := a.requestList.view(a.focused == PanelRequestList, a.cfg.UI.Theme)
	rightTop := a.editor.view(a.focused == PanelEditor, a.cfg.UI.Theme)
	rightBottom := a.response.view(a.focused == PanelResponse, a.cfg.UI.Theme)

	right := lipgloss.JoinVertical(lipgloss.Left, rightTop, rightBottom)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, right)
}

func (a *App) renderStatusBar() string {
	if !a.cfg.UI.Layout.ShowStatusBar {
		return ""
	}

	mode := "NORMAL"
	if a.isSearching {
		mode = "SEARCH: " + a.searchQuery
	} else if a.focused == PanelEditor && a.editor.IsSubEditing() {
		mode = "EDITING"
	}

	modeStyle := lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1).
		Background(lipgloss.Color(a.theme.ModeBg)).
		Foreground(lipgloss.Color(a.theme.TextFg))

	if mode == "EDITING" {
		modeStyle = modeStyle.Background(lipgloss.Color(a.theme.ModeEditingBg))
	}

	status := ""
	if time.Now().Before(a.statusExpiry) {
		status = a.statusMsg
	}

	statusStyle := lipgloss.NewStyle().Padding(0, 1)

	panel := string(a.focused)
	panelStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(lipgloss.Color(a.cfg.UI.Theme.Secondary))

	bar := lipgloss.JoinHorizontal(lipgloss.Top,
		modeStyle.Render(mode),
		panelStyle.Render("["+panel+"]"),
		statusStyle.Render(status),
	)

	return lipgloss.NewStyle().
		Width(a.width).
		Background(lipgloss.Color(a.theme.AppBg)).
		Render(bar)
}

func (a *App) renderHints() string {
	if !a.cfg.UI.Hints.Enabled {
		return ""
	}

	activeTab := ""
	if a.focused == PanelEditor {
		activeTab = a.editor.ActiveTab()
	}

	hints := a.keybindMgr.GetHints(string(a.focused), activeTab)

	if a.editor.HasRequest() {
		executeHints := a.keybindMgr.GetHints("editor", "")
		alreadyHasExecute := false
		for _, h := range hints {
			if h.Action == "execute" {
				alreadyHasExecute = true
				break
			}
		}
		if !alreadyHasExecute {
			for _, h := range executeHints {
				if h.Action == "execute" {
					hints = append([]keybindings.Binding{h}, hints...)
					break
				}
			}
		}
	}

	keyStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(a.cfg.UI.Hints.KeyColor))

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(a.cfg.UI.Hints.DescriptionColor))

	sep := a.cfg.UI.Hints.Separator
	if sep == "" {
		sep = "  "
	}

	var parts []string
	for _, h := range hints {
		if len(h.Keys) == 0 {
			continue
		}
		keyStr := keyStyle.Render(h.Keys[0])
		if a.cfg.UI.Hints.ShowDescriptions {
			parts = append(parts, keyStr+descStyle.Render(" "+h.Description))
		} else {
			parts = append(parts, keyStr)
		}
	}

	hintsText := strings.Join(parts, sep)

	return lipgloss.NewStyle().
		Width(a.width).
		Height(a.cfg.UI.Hints.Height).
		Padding(0, 1).
		Background(lipgloss.Color(a.theme.StatusBg)).
		Foreground(lipgloss.Color(a.theme.Dim)).
		Render(hintsText)
}

func (a *App) renderBackground() string {
	topBar := a.renderTopBar()
	mainArea := a.renderMainArea()
	statusBar := a.renderStatusBar()
	hints := a.renderHints()
	return lipgloss.JoinVertical(lipgloss.Left, topBar, mainArea, statusBar, hints)
}

func (a *App) renderModal(content string) string {
	modal := modalBorderStyle(a.theme.Primary).
		Padding(1, 3).
		Render(content)

	bg := a.renderBackground()
	return overlayCenter(bg, modal, a.width, a.height)
}

func (a *App) renderModalOverlay(content string, w int) string {
	modal := modalBorderStyle(a.theme.Primary).Padding(1, 2).Width(w).Render(content)
	return overlayCenter(a.renderBackground(), modal, a.width, a.height)
}

func (a *App) renderCellEditModal() string {
	modalW := modalWidth(a.width)
	contentW := modalW - 6

	titleStyle := accentStyle(a.theme).Bold(true)

	runes := []rune(a.cellEditVal)
	cursor := a.cellEditCursor
	before := string(runes[:cursor])
	after := ""
	if cursor < len(runes) {
		after = string(runes[cursor:])
	}
	textContent := before + "█" + after

	textAreaStyle := lipgloss.NewStyle().
		Width(contentW).
		Height(8).
		Padding(1, 1).
		Background(lipgloss.Color(a.theme.InputBg)).
		Foreground(lipgloss.Color(a.theme.TextFg))

	dimKey := accentStyle(a.theme).Bold(true)
	dimDesc := dimStyle(a.theme)

	hintsRow := a.buildModalHints("cell_edit_modal", dimKey, dimDesc)

	body := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(a.cellEditTitle),
		"",
		textAreaStyle.Render(textContent),
		"",
		hintsRow,
	)

	return a.renderModalOverlay(body, modalW)
}

func (a *App) renderCurlImportModal() string {
	const visibleLines = 5

	modalW := a.width * 3 / 4
	if modalW > 110 {
		modalW = 110
	}
	if modalW < 50 {
		modalW = 50
	}
	contentW := modalW - 6
	lineW := contentW - 2

	titleStyle := accentStyle(a.theme).Bold(true)
	dim := dimStyle(a.theme)

	runes := []rune(a.curlImportVal)
	cursor := a.curlImportCursor

	wrapped := wrapRunesIntoLines(runes, lineW)

	cursorLine, cursorCol := cursorLineCol(runes, cursor, lineW)

	a.curlImportScroll = syncScrollLine(a.curlImportScroll, cursorLine, visibleLines)

	inputStyle := lipgloss.NewStyle().
		Width(contentW).
		Height(visibleLines).
		Padding(0, 1).
		Background(lipgloss.Color(a.theme.InputBg)).
		Foreground(lipgloss.Color(a.theme.TextFg))

	var renderedLines []string
	start := a.curlImportScroll
	end := start + visibleLines
	if end > len(wrapped) {
		end = len(wrapped)
	}
	for lineIdx := start; lineIdx < end; lineIdx++ {
		line := wrapped[lineIdx]
		if lineIdx == cursorLine {
			before := string(line[:cursorCol])
			after := ""
			if cursorCol < len(line) {
				after = string(line[cursorCol:])
			}
			renderedLines = append(renderedLines, before+"█"+after)
		} else {
			renderedLines = append(renderedLines, string(line))
		}
	}
	if len(renderedLines) == 0 {
		renderedLines = []string{"█"}
	}

	scrollInfo := ""
	totalLines := len(wrapped)
	if totalLines > visibleLines {
		scrollInfo = dim.Render(fmt.Sprintf(" (%d/%d lines)", cursorLine+1, totalLines))
	}

	dimKey := accentStyle(a.theme).Bold(true)
	descStyle := dimStyle(a.theme)
	hintsRow := a.buildModalHints("curl_import_modal", dimKey, descStyle)

	example := dim.Render("e.g. curl -X POST https://api.example.com -H 'Content-Type: application/json' -d '{}'")

	body := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("Import from cURL"),
		"",
		example,
		"",
		inputStyle.Render(strings.Join(renderedLines, "\n")),
		scrollInfo,
		"",
		hintsRow,
	)

	return a.renderModalOverlay(body, modalW)
}

func (a *App) renderCurlExportModal() string {
	const visibleLines = 8

	modalW := a.width * 3 / 4
	if modalW > 120 {
		modalW = 120
	}
	if modalW < 50 {
		modalW = 50
	}
	contentW := modalW - 6
	lineW := contentW - 2

	titleStyle := accentStyle(a.theme).Bold(true)
	dim := dimStyle(a.theme)

	runes := []rune(a.curlExportVal)
	wrapped := wrapRunesIntoLines(runes, lineW)

	scroll := 0
	start := scroll
	end := start + visibleLines
	if end > len(wrapped) {
		end = len(wrapped)
	}

	inputStyle := lipgloss.NewStyle().
		Width(contentW).
		Height(visibleLines).
		Padding(0, 1).
		Background(lipgloss.Color(a.theme.InputBg)).
		Foreground(lipgloss.Color(a.theme.ValueFg))

	var lines []string
	for i := start; i < end; i++ {
		lines = append(lines, string(wrapped[i]))
	}

	totalLines := len(wrapped)
	scrollInfo := ""
	if totalLines > visibleLines {
		scrollInfo = dim.Render(fmt.Sprintf("  (%d lines total)", totalLines))
	}

	hints := a.buildModalHints("curl_export_modal", accentStyle(a.theme).Bold(true), dimStyle(a.theme))

	body := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("Export as cURL"),
		"",
		inputStyle.Render(strings.Join(lines, "\n")),
		scrollInfo,
		"",
		hints,
	)

	return a.renderModalOverlay(body, modalW)
}

func (a *App) renderNotificationModal() string {
	icon := "✓"
	borderColor := a.theme.Success
	if a.notificationIsErr {
		icon = "✗"
		borderColor = a.theme.Error
	}
	content := lipgloss.NewStyle().
		Foreground(lipgloss.Color(borderColor)).
		Bold(true).
		Render(icon+" "+a.notificationMsg) + "\n\n" +
		dimStyle(a.theme).Render("Press any key to continue")

	modal := modalBorderStyle(borderColor).
		Padding(1, 3).
		Render(content)

	bg := a.renderBackground()
	return overlayCenter(bg, modal, a.width, a.height)
}

func (a *App) buildModalHints(panel string, keyStyle, descStyle lipgloss.Style) string {
	hints := a.keybindMgr.GetHints(panel, "")
	var parts []string
	for _, h := range hints {
		if len(h.Keys) == 0 {
			continue
		}
		parts = append(parts, keyStyle.Render(h.Keys[0])+" "+descStyle.Render(h.Description))
	}
	return strings.Join(parts, "   ")
}
