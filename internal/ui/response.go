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

type ResponseModel struct {
keybindMgr *keybindings.Manager
response   *models.Response
width      int
height     int
}

func newResponseModel(km *keybindings.Manager) ResponseModel {
return ResponseModel{keybindMgr: km}
}

func (m *ResponseModel) setResponse(resp *models.Response) {
m.response = resp
}

func (m *ResponseModel) setSize(w, h int) {
m.width = w
m.height = h
}

func (m *ResponseModel) GetResponse() *models.Response {
return m.response
}

func (m *ResponseModel) StatusColor(theme config.ThemeConfig) string {
if m.response == nil {
return theme.Success
}
if m.response.IsClientError() {
return theme.Warning
}
if m.response.IsServerError() {
return theme.Error
}
return theme.Success
}

func (m *ResponseModel) ActiveTab() string {
return ""
}

func (m *ResponseModel) contentWidth() int {
w := m.width - 4
if w < 1 {
w = 1
}
return w
}

func (m *ResponseModel) contentHeight() int {
h := m.height - 4
if h < 1 {
h = 1
}
return h
}

func (m *ResponseModel) FormattedBody() string {
if m.response == nil {
return ""
}
body := m.response.Body
ct := m.response.ContentType()
if strings.Contains(ct, "json") {
if pretty, err := prettyJSON(body); err == nil {
return pretty
}
}
return body
}

func (m *ResponseModel) view(focused bool, theme config.ThemeConfig) string {
inner := lipgloss.NewStyle().
Width(m.contentWidth()).
Height(m.contentHeight()).
Render(m.renderContent(theme))

return panelBorderStyle(focused, theme).
Padding(0, 1).
Render(inner)
}

func (m *ResponseModel) renderContent(theme config.ThemeConfig) string {
if m.response == nil {
return lipgloss.NewStyle().
Foreground(lipgloss.Color("240")).
Render("No response yet\n\nPress ctrl+e to execute the request")
}

if m.response.Error != "" {
return errorStyle().
Render("Error: " + m.response.Error)
}

statusLine := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.StatusColor(theme))).
Render(fmt.Sprintf("%d %s", m.response.Status, m.response.StatusText))

meta := dimStyle().
Render(fmt.Sprintf("  %dms  %s", m.response.Duration.Milliseconds(), formatSize(m.response.Size)))

header := statusLine + meta

sep := lipgloss.NewStyle().Foreground(lipgloss.Color("#333333")).
Render(strings.Repeat("─", m.contentWidth()))

body := m.FormattedBody()
lines := strings.Split(body, "\n")
available := m.contentHeight() - 3
if available < 1 {
available = 1
}
cw := m.contentWidth()
var preview []string
for i, l := range lines {
if i >= available {
break
}
if len([]rune(l)) > cw {
l = string([]rune(l)[:cw])
}
preview = append(preview, l)
}

hint := lipgloss.NewStyle().Foreground(lipgloss.Color("#4e4e4e")).
Render("v — open viewer")

return lipgloss.JoinVertical(lipgloss.Left,
header,
sep,
strings.Join(preview, "\n"),
hint,
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

func (m *ResponseModel) totalLines() int {
return len(strings.Split(m.FormattedBody(), "\n"))
}
