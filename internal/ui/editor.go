package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/user/http-cli/internal/config"
	"github.com/user/http-cli/internal/models"
	"github.com/user/http-cli/internal/ui/keybindings"
)

type EditorTab string

const (
	TabURL     EditorTab = "URL"
	TabHeaders EditorTab = "Headers"
	TabBody    EditorTab = "Body"
	TabQuery   EditorTab = "Query"
	TabAuth    EditorTab = "Auth"
)

var editorTabs = []EditorTab{TabURL, TabHeaders, TabBody, TabQuery, TabAuth}

type EditorModel struct {
	keybindMgr   *keybindings.Manager
	request      *models.Request
	activeTab    EditorTab
	editingField string
	fieldValue   string
	cursorPos    int
	width        int
	height       int
	headerIdx    int
	queryIdx     int
}

func newEditorModel(km *keybindings.Manager) EditorModel {
	return EditorModel{
		keybindMgr: km,
		activeTab:  TabURL,
	}
}

func (m *EditorModel) setRequest(req *models.Request) {
	m.request = req
	m.activeTab = TabURL
	m.editingField = ""
	m.fieldValue = ""
}

func (m *EditorModel) setSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *EditorModel) handleKey(msg tea.KeyMsg, req *models.Request) tea.Cmd {
	if req == nil {
		return nil
	}
	m.request = req

	key := msg.String()

	if m.editingField != "" {
		return m.handleEditingKey(key)
	}

	switch key {
	case "]":
		m.nextTab()
	case "[":
		m.prevTab()
	case "i", "enter":
		m.startEditing()
	}

	return nil
}

func (m *EditorModel) handleEditingKey(key string) tea.Cmd {
	switch key {
	case "esc":
		m.cancelEdit()
	case "enter":
		m.commitEdit()
	case "backspace":
		if m.cursorPos > 0 {
			runes := []rune(m.fieldValue)
			m.fieldValue = string(runes[:m.cursorPos-1]) + string(runes[m.cursorPos:])
			m.cursorPos--
		}
	default:
		if len(key) == 1 {
			runes := []rune(m.fieldValue)
			newRunes := make([]rune, len(runes)+1)
			copy(newRunes, runes[:m.cursorPos])
			newRunes[m.cursorPos] = []rune(key)[0]
			copy(newRunes[m.cursorPos+1:], runes[m.cursorPos:])
			m.fieldValue = string(newRunes)
			m.cursorPos++
		}
	}
	return nil
}

func (m *EditorModel) StartEditing() {
	m.startEditing()
}

func (m *EditorModel) Reset() {
	m.cancelEdit()
}

func (m *EditorModel) startEditing() {
	if m.request == nil {
		return
	}
	switch m.activeTab {
	case TabURL:
		m.editingField = "url"
		m.fieldValue = m.request.URL
		m.cursorPos = len([]rune(m.fieldValue))
	}
}

func (m *EditorModel) cancelEdit() {
	m.editingField = ""
	m.fieldValue = ""
	m.cursorPos = 0
}

func (m *EditorModel) commitEdit() {
	if m.request == nil {
		m.cancelEdit()
		return
	}
	switch m.editingField {
	case "url":
		m.request.URL = m.fieldValue
	}
	m.cancelEdit()
}

func (m *EditorModel) nextTab() {
	for i, t := range editorTabs {
		if t == m.activeTab {
			m.activeTab = editorTabs[(i+1)%len(editorTabs)]
			return
		}
	}
}

func (m *EditorModel) prevTab() {
	for i, t := range editorTabs {
		if t == m.activeTab {
			m.activeTab = editorTabs[(i-1+len(editorTabs))%len(editorTabs)]
			return
		}
	}
}

func (m *EditorModel) JumpToTab(n int) {
	if n >= 1 && n <= len(editorTabs) {
		m.activeTab = editorTabs[n-1]
	}
}

func (m *EditorModel) ActiveTab() string {
	return string(m.activeTab)
}

func (m *EditorModel) view(focused bool, theme config.ThemeConfig) string {
	borderColor := theme.BlurBorder
	if focused {
		borderColor = theme.FocusBorder
	}

	tabs := m.renderTabs()
	content := m.renderTabContent()

	inner := lipgloss.JoinVertical(lipgloss.Left, tabs, content)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Render(inner)
}

func (m *EditorModel) renderTabs() string {
	var parts []string
	for i, t := range editorTabs {
		style := lipgloss.NewStyle().Padding(0, 1)
		label := fmt.Sprintf("%d:%s", i+1, string(t))
		if t == m.activeTab {
			style = style.Bold(true).
				Underline(true).
				Foreground(lipgloss.Color("#00d7ff"))
		} else {
			style = style.Foreground(lipgloss.Color("#626262"))
		}
		parts = append(parts, style.Render(label))
	}
	return strings.Join(parts, " ")
}

func (m *EditorModel) renderTabContent() string {
	if m.request == nil {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("Select a request from the list")
	}

	switch m.activeTab {
	case TabURL:
		return m.renderURLTab()
	case TabHeaders:
		return m.renderHeadersTab()
	case TabBody:
		return m.renderBodyTab()
	case TabQuery:
		return m.renderQueryTab()
	case TabAuth:
		return m.renderAuthTab()
	}
	return ""
}

func (m *EditorModel) renderURLTab() string {
	label := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#87d7ff")).
		Render("URL: ")

	value := m.request.URL
	if m.editingField == "url" {
		runes := []rune(m.fieldValue)
		before := string(runes[:m.cursorPos])
		after := string(runes[m.cursorPos:])
		value = before + "█" + after
		value = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffffff")).
			Background(lipgloss.Color("#1c1c2c")).
			Render(value)
	} else {
		value = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#e4e4e4")).
			Render(value)
		value += lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("  [i/enter to edit]")
	}

	methodLabel := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#87d7ff")).
		Render("Method: ")

	methodValue := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00d700")).
		Render(string(m.request.Method))

	return lipgloss.JoinVertical(lipgloss.Left,
		label+value,
		"",
		methodLabel+methodValue,
	)
}

func (m *EditorModel) renderHeadersTab() string {
	if len(m.request.Headers) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("No headers\n[i] to add a header")
	}

	var lines []string
	for i, h := range m.request.Headers {
		enabled := "✓"
		if !h.Enabled {
			enabled = "✗"
		}
		line := fmt.Sprintf("%s  %s: %s", enabled, h.Key, h.Value)
		if i == m.headerIdx {
			line = lipgloss.NewStyle().
				Background(lipgloss.Color("#303030")).
				Render("> " + line)
		} else {
			line = "  " + line
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func (m *EditorModel) renderBodyTab() string {
	bodyType := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#87d7ff")).
		Render("Type: ")

	typeValue := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#e4e4e4")).
		Render(string(m.request.Body.Type))

	content := m.request.Body.Content
	if content == "" {
		content = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("(empty body)")
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		bodyType+typeValue,
		"",
		content,
	)
}

func (m *EditorModel) renderQueryTab() string {
	if len(m.request.QueryParams) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("No query parameters\n[i] to add a parameter")
	}

	var lines []string
	for _, p := range m.request.QueryParams {
		enabled := "✓"
		if !p.Enabled {
			enabled = "✗"
		}
		lines = append(lines, fmt.Sprintf("%s  %s = %s", enabled, p.Key, p.Value))
	}
	return strings.Join(lines, "\n")
}

func (m *EditorModel) renderAuthTab() string {
	authLabel := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#87d7ff")).
		Render("Type: ")

	authType := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#e4e4e4")).
		Render(string(m.request.Auth.Type))

	details := ""
	switch m.request.Auth.Type {
	case models.AuthBasic:
		details = fmt.Sprintf("\nUsername: %s\nPassword: %s",
			m.request.Auth.Username, maskPassword(m.request.Auth.Password))
	case models.AuthBearer:
		details = fmt.Sprintf("\nToken: %s", maskToken(m.request.Auth.Token))
	case models.AuthAPIKey:
		details = fmt.Sprintf("\nKey: %s\nValue: %s\nIn: %s",
			m.request.Auth.Key, m.request.Auth.Value, m.request.Auth.In)
	}

	return authLabel + authType + details
}

func maskPassword(s string) string {
	if len(s) == 0 {
		return ""
	}
	return strings.Repeat("*", len(s))
}

func maskToken(s string) string {
	if len(s) <= 8 {
		return strings.Repeat("*", len(s))
	}
	return s[:4] + strings.Repeat("*", len(s)-8) + s[len(s)-4:]
}
