package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/user/http-cli/internal/config"
	"github.com/user/http-cli/internal/models"
	"github.com/user/http-cli/internal/ui/keybindings"
)

type RequestListModel struct {
	keybindMgr  *keybindings.Manager
	requests    []*models.Request
	filtered    []*models.Request
	filter      string
	selectedIdx int
	scrollOffset int
	width       int
	height      int
}

func newRequestListModel(km *keybindings.Manager) RequestListModel {
	return RequestListModel{
		keybindMgr: km,
		requests:   make([]*models.Request, 0),
		filtered:   make([]*models.Request, 0),
	}
}

func (m *RequestListModel) setRequests(reqs []*models.Request) {
	m.requests = reqs
	m.applyFilter()
}

func (m *RequestListModel) setFilter(q string) {
	m.filter = q
	m.applyFilter()
	m.selectedIdx = 0
}

func (m *RequestListModel) applyFilter() {
	if m.filter == "" {
		m.filtered = m.requests
		return
	}
	q := strings.ToLower(m.filter)
	m.filtered = make([]*models.Request, 0)
	for _, r := range m.requests {
		if strings.Contains(strings.ToLower(r.Name), q) ||
			strings.Contains(strings.ToLower(r.URL), q) {
			m.filtered = append(m.filtered, r)
		}
	}
}

func (m *RequestListModel) setSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *RequestListModel) moveDown() {
	if m.selectedIdx < len(m.filtered)-1 {
		m.selectedIdx++
		m.ensureVisible()
	}
}

func (m *RequestListModel) moveUp() {
	if m.selectedIdx > 0 {
		m.selectedIdx--
		m.ensureVisible()
	}
}

func (m *RequestListModel) selected() *models.Request {
	if len(m.filtered) == 0 || m.selectedIdx >= len(m.filtered) {
		return nil
	}
	return m.filtered[m.selectedIdx]
}

func (m *RequestListModel) ensureVisible() {
	if m.selectedIdx < m.scrollOffset {
		m.scrollOffset = m.selectedIdx
	}
	visible := m.height - 2
	if visible < 1 {
		visible = 1
	}
	if m.selectedIdx >= m.scrollOffset+visible {
		m.scrollOffset = m.selectedIdx - visible + 1
	}
}

func (m *RequestListModel) view(focused bool, theme config.ThemeConfig) string {
	borderColor := theme.BlurBorder
	if focused {
		borderColor = theme.FocusBorder
	}

	title := "Requests"
	if m.filter != "" {
		title = fmt.Sprintf("Requests [/%s]", m.filter)
	}

	var lines []string
	visible := m.height - 2
	if visible < 1 {
		visible = 1
	}

	end := m.scrollOffset + visible
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	for i := m.scrollOffset; i < end; i++ {
		req := m.filtered[i]
		line := m.renderLine(req, i == m.selectedIdx, theme)
		lines = append(lines, line)
	}

	if len(m.filtered) == 0 {
		empty := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("No requests\nPress n to create")
		lines = append(lines, empty)
	}

	content := strings.Join(lines, "\n")

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Padding(0, 1).
		Render(title + "\n" + content)
}

func (m *RequestListModel) renderLine(req *models.Request, selected bool, theme config.ThemeConfig) string {
	methodColors := map[string]string{
		"GET":    theme.MethodGet,
		"POST":   theme.MethodPost,
		"PUT":    theme.MethodPut,
		"DELETE": theme.MethodDelete,
		"PATCH":  theme.MethodPatch,
	}

	color := methodColors[string(req.Method)]
	if color == "" {
		color = "#ffffff"
	}

	methodStr := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(color)).
		Render(fmt.Sprintf("%-7s", string(req.Method)))

	name := req.Name
	maxLen := m.width - 12
	if maxLen > 0 && len(name) > maxLen {
		name = name[:maxLen-3] + "..."
	}

	line := methodStr + " " + name

	if selected {
		return lipgloss.NewStyle().
			Background(lipgloss.Color("#303030")).
			Render("> " + line)
	}
	return "  " + line
}
