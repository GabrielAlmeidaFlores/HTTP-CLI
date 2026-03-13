package ui

import (
"encoding/json"
"fmt"
"sort"
"strings"

"github.com/charmbracelet/lipgloss"

"github.com/user/http-cli/internal/config"
"github.com/user/http-cli/internal/models"
"github.com/user/http-cli/internal/ui/keybindings"
)

type responseTab int

const (
responseTabBody    responseTab = 0
responseTabHeaders responseTab = 1
responseTabInfo    responseTab = 2
)

var responseTabs = []string{"Body", "Headers", "Info"}

type ResponseModel struct {
keybindMgr *keybindings.Manager
response   *models.Response
activeTab  responseTab
scrollY    int
width      int
height     int
theme      config.ThemeConfig
}

func newResponseModel(km *keybindings.Manager, theme config.ThemeConfig) ResponseModel {
return ResponseModel{keybindMgr: km, theme: theme}
}

func (m *ResponseModel) setResponse(resp *models.Response) {
m.response = resp
m.scrollY = 0
m.activeTab = responseTabBody
}

func (m *ResponseModel) setSize(w, h int) {
m.width = w
m.height = h
}

func (m *ResponseModel) GetResponse() *models.Response {
return m.response
}

func (m *ResponseModel) NextTab() {
m.activeTab = (m.activeTab + 1) % responseTab(len(responseTabs))
m.scrollY = 0
}

func (m *ResponseModel) PrevTab() {
m.activeTab = (m.activeTab + responseTab(len(responseTabs)) - 1) % responseTab(len(responseTabs))
m.scrollY = 0
}

func (m *ResponseModel) ScrollDown() {
m.scrollY++
}

func (m *ResponseModel) ScrollUp() {
if m.scrollY > 0 {
m.scrollY--
}
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
return responseTabs[m.activeTab]
}

func (m *ResponseModel) contentWidth() int {
w := m.width - 4
if w < 1 {
w = 1
}
return w
}

func (m *ResponseModel) contentHeight() int {
h := m.height - 6
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
tabBar := m.renderTabBar(theme)
content := m.renderActiveTab(theme)

inner := lipgloss.JoinVertical(lipgloss.Left, tabBar, content)

return panelBorderStyle(focused, theme).
Padding(0, 1).
Width(m.width).
Height(m.height).
Render(inner)
}

func (m *ResponseModel) renderTabBar(theme config.ThemeConfig) string {
active := lipgloss.NewStyle().
Bold(true).
Underline(true).
Foreground(lipgloss.Color(theme.Primary)).
Padding(0, 1)
inactive := dimStyle(m.theme).Padding(0, 1)

var parts []string
for i, name := range responseTabs {
label := fmt.Sprintf("%d:%s", i+1, name)
if responseTab(i) == m.activeTab {
parts = append(parts, active.Render(label))
} else {
parts = append(parts, inactive.Render(label))
}
}
bar := strings.Join(parts, " ")
underline := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.SeparatorFg)).
Render(strings.Repeat("─", m.contentWidth()))
return lipgloss.JoinVertical(lipgloss.Left, bar, underline)
}

func (m *ResponseModel) renderActiveTab(theme config.ThemeConfig) string {
switch m.activeTab {
case responseTabHeaders:
return m.renderHeadersTab()
case responseTabInfo:
return m.renderInfoTab(theme)
default:
return m.renderBodyTab(theme)
}
}

func (m *ResponseModel) renderBodyTab(theme config.ThemeConfig) string {
cw := m.contentWidth()
ch := m.contentHeight()

if m.response == nil {
execKey := "ctrl+e"
for _, h := range m.keybindMgr.GetHints("response", "") {
if h.Action == "execute" && len(h.Keys) > 0 {
execKey = h.Keys[0]
break
}
}
return dimStyle(m.theme).Render("No response yet\n\nPress " + execKey + " to execute the request")
}
if m.response.Error != "" {
return errorStyle(m.theme).Render("Error: " + m.response.Error)
}

statusStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.StatusColor(theme)))
statusLine := statusStyle.Render(fmt.Sprintf("%d %s", m.response.Status, m.response.StatusText))
meta := dimStyle(m.theme).Render(fmt.Sprintf("  %dms  %s", m.response.Duration.Milliseconds(), formatSize(m.response.Size)))
hint := buildHintsLine(m.keybindMgr, "response", "", m.theme)

sep := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.SeparatorFg)).Render(strings.Repeat("─", cw))

body := m.FormattedBody()
lines := strings.Split(body, "\n")
available := ch - 4
if available < 1 {
available = 1
}
total := len(lines)
start := m.scrollY
if start > total-1 {
start = total - 1
}
if start < 0 {
start = 0
}
end := start + available
if end > total {
end = total
}
visible := lines[start:end]
var display []string
for _, l := range visible {
if len([]rune(l)) > cw {
l = string([]rune(l)[:cw])
}
display = append(display, l)
}

scrollHint := ""
if total > available {
scrollHint = dimStyle(m.theme).Render(fmt.Sprintf("  ↑↓ scroll  %d/%d lines", start+1, total))
}

parts := []string{statusLine + meta, sep, strings.Join(display, "\n")}
if scrollHint != "" {
parts = append(parts, scrollHint)
}
parts = append(parts, hint)
return strings.Join(parts, "\n")
}

func (m *ResponseModel) renderHeadersTab() string {
cw := m.contentWidth()
ch := m.contentHeight()

if m.response == nil || len(m.response.Headers) == 0 {
return dimStyle(m.theme).Render("No headers")
}

keys := make([]string, 0, len(m.response.Headers))
for k := range m.response.Headers {
keys = append(keys, k)
}
sort.Strings(keys)

keyStyle := secondaryStyle(m.theme).Bold(true)
valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.ValueFg))
dim := dimStyle(m.theme)

keyW := 30
valW := cw - keyW - 3
if valW < 10 {
valW = 10
}

hdr := keyStyle.Render(padRight("Header", keyW)) + dim.Render(" ") + keyStyle.Render("Value")
sep := dim.Render(strings.Repeat("─", cw))

var rows []string
for _, k := range keys {
v := m.response.Headers[k]
kDisplay := padRight(k, keyW)
vDisplay := truncate(v, valW)
rows = append(rows, keyStyle.Render(kDisplay)+" "+valStyle.Render(vDisplay))
}

total := len(rows)
available := ch - 3
if available < 1 {
available = 1
}
start := m.scrollY
if start > total-1 {
start = total - 1
}
if start < 0 {
start = 0
}
end := start + available
if end > total {
end = total
}

parts := []string{hdr, sep}
parts = append(parts, rows[start:end]...)
if total > available {
parts = append(parts, dim.Render(fmt.Sprintf("  ↑↓ scroll  %d/%d", start+1, total)))
}
return strings.Join(parts, "\n")
}

func (m *ResponseModel) renderInfoTab(theme config.ThemeConfig) string {
if m.response == nil {
return dimStyle(m.theme).Render("No response yet")
}

labelStyle := secondaryStyle(m.theme).Bold(true).Width(22)
valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.ValueFg))
sep := dimStyle(m.theme).Render(strings.Repeat("─", m.contentWidth()))
sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.Primary))

statusColor := m.StatusColor(theme)
statusVal := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(statusColor)).
Render(fmt.Sprintf("%d %s", m.response.Status, m.response.StatusText))

row := func(label, value string) string {
return labelStyle.Render(label+":") + " " + valStyle.Render(value)
}

timing := m.response.Duration
ms := timing.Milliseconds()
var timingStr string
switch {
case ms < 200:
timingStr = lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Success)).Render(fmt.Sprintf("%dms (fast)", ms))
case ms < 1000:
timingStr = lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Warning)).Render(fmt.Sprintf("%dms (moderate)", ms))
default:
timingStr = lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.ModeEditingBg)).Render(fmt.Sprintf("%dms (slow)", ms))
}

serverIP := m.response.RemoteAddr
if serverIP == "" {
serverIP = dimStyle(m.theme).Render("(not available)")
}

proto := m.response.Protocol
if proto == "" {
proto = dimStyle(m.theme).Render("(unknown)")
}

ct := m.response.ContentType()
if ct == "" {
ct = dimStyle(m.theme).Render("(not set)")
}

encoding := ""
if enc, ok := m.response.Headers["Content-Encoding"]; ok {
encoding = enc
} else if enc, ok := m.response.Headers["content-encoding"]; ok {
encoding = enc
}
if encoding == "" {
encoding = dimStyle(m.theme).Render("(none)")
}

timestamp := m.response.Timestamp.Format("2006-01-02 15:04:05")

lines := []string{
sectionStyle.Render("Status"),
sep,
"  " + labelStyle.Render("Status:") + " " + statusVal,
"  " + row("Protocol", proto),
"",
sectionStyle.Render("Timing"),
sep,
"  " + labelStyle.Render("Duration:") + " " + timingStr,
"  " + row("Timestamp", timestamp),
"",
sectionStyle.Render("Connection"),
sep,
"  " + row("Server IP", serverIP),
"",
sectionStyle.Render("Content"),
sep,
"  " + row("Content-Type", ct),
"  " + row("Content-Encoding", encoding),
"  " + row("Size", formatSize(m.response.Size)),
}

ch := m.contentHeight()
total := len(lines)
start := m.scrollY
if start > total-1 {
start = total - 1
}
if start < 0 {
start = 0
}
end := start + ch
if end > total {
end = total
}
return strings.Join(lines[start:end], "\n")
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

func (m *ResponseModel) totalContentLines() int {
switch m.activeTab {
case responseTabHeaders:
return len(m.response.Headers) + 2
case responseTabInfo:
return 20
default:
return m.totalLines()
}
}
