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
		Background(lipgloss.Color("#00d700")).
		Foreground(lipgloss.Color("#000000"))

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
		BorderForeground(lipgloss.Color("#626262")).
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
		Background(lipgloss.Color("#005fd7")).
		Foreground(lipgloss.Color("#ffffff"))

	if mode == "EDITING" {
		modeStyle = modeStyle.Background(lipgloss.Color("#d75f00"))
	}

	status := ""
	if time.Now().Before(a.statusExpiry) {
		status = a.statusMsg
	}

	statusStyle := lipgloss.NewStyle().Padding(0, 1)

	panel := string(a.focused)
	panelStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(lipgloss.Color("#87d7ff"))

	bar := lipgloss.JoinHorizontal(lipgloss.Top,
		modeStyle.Render(mode),
		panelStyle.Render("["+panel+"]"),
		statusStyle.Render(status),
	)

	return lipgloss.NewStyle().
		Width(a.width).
		Background(lipgloss.Color("#1c1c1c")).
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
		Background(lipgloss.Color("#121212")).
		Foreground(lipgloss.Color("#626262")).
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
	modal := modalBorderStyle("#00d7ff").
		Padding(1, 3).
		Render(content)

	bg := a.renderBackground()
	return overlayCenter(bg, modal, a.width, a.height)
}

func (a *App) renderModalOverlay(content string, w int) string {
	modal := modalBorderStyle("#00d7ff").Padding(1, 2).Width(w).Render(content)
	return overlayCenter(a.renderBackground(), modal, a.width, a.height)
}

func (a *App) renderCellEditModal() string {
	modalW := a.width * 3 / 4
	if modalW > 100 {
		modalW = 100
	}
	if modalW < 40 {
		modalW = 40
	}
	contentW := modalW - 6

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00d7ff"))

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
		Background(lipgloss.Color("#1c1c2c")).
		Foreground(lipgloss.Color("#ffffff"))

	dimKey := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00d7ff"))
	dimDesc := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))

	hintsRow := a.buildModalHints("cell_edit_modal", dimKey, dimDesc)

	body := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(a.cellEditTitle),
		"",
		textAreaStyle.Render(textContent),
		"",
		hintsRow,
	)

	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00d7ff")).
		Padding(1, 2).
		Width(modalW).
		Render(body)

	bg := a.renderBackground()
	return overlayCenter(bg, modal, a.width, a.height)
}

func (a *App) renderCurlImportModal() string {
	modalW := a.width * 3 / 4
	if modalW > 110 {
		modalW = 110
	}
	if modalW < 50 {
		modalW = 50
	}
	contentW := modalW - 6

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00d7ff"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))

	runes := []rune(a.curlImportVal)
	cursor := a.curlImportCursor
	before := string(runes[:cursor])
	after := ""
	if cursor < len(runes) {
		after = string(runes[cursor:])
	}
	textContent := before + "█" + after

	inputStyle := lipgloss.NewStyle().
		Width(contentW).
		Height(4).
		Padding(0, 1).
		Background(lipgloss.Color("#1c1c2c")).
		Foreground(lipgloss.Color("#ffffff"))

	dimKey := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00d7ff"))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))

	hintsRow := a.buildModalHints("curl_import_modal", dimKey, descStyle)

	example := dimStyle.Render("e.g. curl -X POST https://api.example.com -H 'Content-Type: application/json' -d '{\"key\":\"val\"}'")

	body := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("Import from cURL"),
		"",
		example,
		"",
		inputStyle.Render(textContent),
		"",
		hintsRow,
	)

	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00d7ff")).
		Padding(1, 2).
		Width(modalW).
		Render(body)

	bg := a.renderBackground()
	return overlayCenter(bg, modal, a.width, a.height)
}

func (a *App) renderNotificationModal() string {
	icon := "✓"
	borderColor := "#00d700"
	if a.notificationIsErr {
		icon = "✗"
		borderColor = "#d70000"
	}
	content := lipgloss.NewStyle().
		Foreground(lipgloss.Color(borderColor)).
		Bold(true).
		Render(icon+" "+a.notificationMsg) + "\n\n" +
		lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render("Press any key to continue")

	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Padding(1, 3).
		Render(content)

	bg := a.renderBackground()
	return overlayCenter(bg, modal, a.width, a.height)
}

func (a *App) methodColor(method string) string {
	colors := map[string]string{
		"GET":    a.cfg.UI.Theme.MethodGet,
		"POST":   a.cfg.UI.Theme.MethodPost,
		"PUT":    a.cfg.UI.Theme.MethodPut,
		"DELETE": a.cfg.UI.Theme.MethodDelete,
		"PATCH":  a.cfg.UI.Theme.MethodPatch,
	}
	if c, ok := colors[method]; ok && c != "" {
		return c
	}
	return "#ffffff"
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
