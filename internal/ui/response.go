package ui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/user/http-cli/internal/config"
	"github.com/user/http-cli/internal/models"
	"github.com/user/http-cli/internal/ui/keybindings"
)

type ResponseTab string

const (
	RespTabBody    ResponseTab = "Body"
	RespTabHeaders ResponseTab = "Headers"
	RespTabInfo    ResponseTab = "Info"
)

var responseTabs = []ResponseTab{RespTabBody, RespTabHeaders, RespTabInfo}

type ResponseModel struct {
	keybindMgr   *keybindings.Manager
	response     *models.Response
	activeTab    ResponseTab
	scrollOffset int
	width        int
	height       int
}

func newResponseModel(km *keybindings.Manager) ResponseModel {
	return ResponseModel{
		keybindMgr: km,
		activeTab:  RespTabBody,
	}
}

func (m *ResponseModel) setResponse(resp *models.Response) {
	m.response = resp
	m.scrollOffset = 0
}

func (m *ResponseModel) setSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *ResponseModel) scrollDown(n int) {
	m.scrollOffset += n
}

func (m *ResponseModel) scrollUp(n int) {
	m.scrollOffset -= n
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

func (m *ResponseModel) nextTab() {
	for i, t := range responseTabs {
		if t == m.activeTab {
			m.activeTab = responseTabs[(i+1)%len(responseTabs)]
			return
		}
	}
}

func (m *ResponseModel) JumpToTab(n int) {
	if n >= 1 && n <= len(responseTabs) {
		m.activeTab = responseTabs[n-1]
	}
}

func (m *ResponseModel) view(focused bool, theme config.ThemeConfig) string {
	borderColor := theme.BlurBorder
	if focused {
		borderColor = theme.FocusBorder
	}

	tabs := m.renderTabs()
	content := m.renderContent(theme)

	inner := lipgloss.JoinVertical(lipgloss.Left, tabs, content)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Render(inner)
}

func (m *ResponseModel) renderTabs() string {
	var parts []string
	for i, t := range responseTabs {
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

	statusStr := ""
	if m.response != nil {
		color := "#00d700"
		if m.response.IsClientError() {
			color = "#d7d700"
		} else if m.response.IsServerError() {
			color = "#d70000"
		}
		statusStr = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(color)).
			Render(fmt.Sprintf(" %d", m.response.Status))
	}

	return strings.Join(parts, " ") + statusStr
}

func (m *ResponseModel) renderContent(theme config.ThemeConfig) string {
	if m.response == nil {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("No response yet\nPress ctrl+e to execute the request")
	}

	if m.response.Error != "" {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Error)).
			Render("Error: " + m.response.Error)
	}

	switch m.activeTab {
	case RespTabBody:
		return m.renderBody()
	case RespTabHeaders:
		return m.renderHeaders()
	case RespTabInfo:
		return m.renderInfo()
	}
	return ""
}

func (m *ResponseModel) renderBody() string {
	body := m.response.Body

	if strings.Contains(m.response.ContentType(), "json") {
		if pretty, err := prettyJSON(body); err == nil {
			body = pretty
		}
	}

	lines := strings.Split(body, "\n")
	visible := m.height - 4
	if visible < 1 {
		visible = 1
	}

	start := m.scrollOffset
	if start >= len(lines) {
		start = len(lines) - 1
	}
	if start < 0 {
		start = 0
	}

	end := start + visible
	if end > len(lines) {
		end = len(lines)
	}

	return strings.Join(lines[start:end], "\n")
}

func (m *ResponseModel) renderHeaders() string {
	var lines []string
	for k, v := range m.response.Headers {
		key := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#87d7ff")).
			Render(k)
		lines = append(lines, key+": "+v)
	}
	return strings.Join(lines, "\n")
}

func (m *ResponseModel) renderInfo() string {
	duration := m.response.Duration.Milliseconds()
	size := formatSize(m.response.Size)

	return fmt.Sprintf(
		"Status:   %d %s\nDuration: %dms\nSize:     %s\nTime:     %s",
		m.response.Status,
		m.response.StatusText,
		duration,
		size,
		m.response.Timestamp.Format("15:04:05"),
	)
}

func prettyJSON(s string) (string, error) {
	var v interface{}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
