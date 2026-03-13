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

var allBodyTypes = []models.BodyType{
models.BodyNone,
models.BodyRaw,
models.BodyJSON,
models.BodyFormData,
models.BodyURLEncoded,
}

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
tableRow     int
tableCol     int
tableEditing bool
bodyTypes    []models.BodyType
bodyTypeIdx  int
}

func newEditorModel(km *keybindings.Manager) EditorModel {
return EditorModel{
keybindMgr: km,
activeTab:  TabURL,
bodyTypes:  allBodyTypes,
tableCol:   1,
}
}

func (m *EditorModel) setRequest(req *models.Request) {
m.request = req
m.activeTab = TabURL
m.editingField = ""
m.fieldValue = ""
m.tableRow = 0
m.tableCol = 1
m.tableEditing = false
m.headerIdx = 0
m.queryIdx = 0
m.bodyTypeIdx = 0
if req != nil {
for i, bt := range allBodyTypes {
if bt == req.Body.Type {
m.bodyTypeIdx = i
break
}
}
}
}

func (m *EditorModel) setSize(w, h int) {
m.width = w
m.height = h
}

func (m *EditorModel) IsSubEditing() bool {
return m.tableEditing
}

func (m *EditorModel) handleKey(msg tea.KeyMsg, req *models.Request) tea.Cmd {
if req == nil {
return nil
}
m.request = req
key := msg.String()
switch m.editingField {
case "":
return m.handleNormalKey(key)
case "table_nav":
return m.handleTableNavKey(key)
case "table_cell", "body_raw", "auth_token":
return m.handleTextEditKey(key)
}
return nil
}

func (m *EditorModel) handleNormalKey(key string) tea.Cmd {
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

func (m *EditorModel) handleTableNavKey(key string) tea.Cmd {
switch key {
case "]":
m.nextTab()
m.tableRow = 0
m.tableCol = 1
return nil
case "[":
m.prevTab()
m.tableRow = 0
m.tableCol = 1
return nil
}
switch m.activeTab {
case TabURL:
return m.handleURLNavKey(key)
case TabHeaders:
return m.handleHeadersNavKey(key)
case TabQuery:
return m.handleQueryNavKey(key)
case TabBody:
return m.handleBodyNavKey(key)
case TabAuth:
return m.handleAuthNavKey(key)
}
return nil
}

func (m *EditorModel) handleURLNavKey(key string) tea.Cmd {
switch key {
case "j", "down":
if m.tableRow < 1 {
m.tableRow++
}
case "k", "up":
if m.tableRow > 0 {
m.tableRow--
}
case "h", "left":
if m.tableRow == 0 {
m.cycleMethod(-1)
}
case "l", "right":
if m.tableRow == 0 {
m.cycleMethod(1)
}
case "enter":
if m.tableRow == 0 {
m.cycleMethod(1)
} else {
m.startCellEdit(m.request.URL)
}
case "i":
if m.tableRow == 1 {
m.startCellEdit(m.request.URL)
} else {
m.cycleMethod(1)
}
}
return nil
}

func (m *EditorModel) handleHeadersNavKey(key string) tea.Cmd {
n := len(m.request.Headers)
switch key {
case "j", "down":
if m.tableRow < n-1 {
m.tableRow++
}
m.headerIdx = m.tableRow
case "k", "up":
if m.tableRow > 0 {
m.tableRow--
}
m.headerIdx = m.tableRow
case "tab":
if m.tableCol == 1 {
m.tableCol = 2
} else {
m.tableCol = 1
}
case "enter", "i":
if n > 0 {
m.startCellEdit(m.currentCellValue())
}
case "n":
m.request.Headers = append(m.request.Headers, models.Header{Enabled: true})
m.tableRow = len(m.request.Headers) - 1
m.headerIdx = m.tableRow
m.tableCol = 1
m.startCellEdit("")
case "d":
if n > 0 && m.tableRow < n {
m.request.Headers = append(m.request.Headers[:m.tableRow], m.request.Headers[m.tableRow+1:]...)
if m.tableRow >= len(m.request.Headers) && m.tableRow > 0 {
m.tableRow--
}
m.headerIdx = m.tableRow
}
case "space":
if n > 0 && m.tableRow < n {
m.request.Headers[m.tableRow].Enabled = !m.request.Headers[m.tableRow].Enabled
}
}
return nil
}

func (m *EditorModel) handleQueryNavKey(key string) tea.Cmd {
n := len(m.request.QueryParams)
switch key {
case "j", "down":
if m.tableRow < n-1 {
m.tableRow++
}
m.queryIdx = m.tableRow
case "k", "up":
if m.tableRow > 0 {
m.tableRow--
}
m.queryIdx = m.tableRow
case "tab":
if m.tableCol == 1 {
m.tableCol = 2
} else {
m.tableCol = 1
}
case "enter", "i":
if n > 0 {
m.startCellEdit(m.currentCellValue())
}
case "n":
m.request.QueryParams = append(m.request.QueryParams, models.QueryParam{Enabled: true})
m.tableRow = len(m.request.QueryParams) - 1
m.queryIdx = m.tableRow
m.tableCol = 1
m.startCellEdit("")
case "d":
if n > 0 && m.tableRow < n {
m.request.QueryParams = append(m.request.QueryParams[:m.tableRow], m.request.QueryParams[m.tableRow+1:]...)
if m.tableRow >= len(m.request.QueryParams) && m.tableRow > 0 {
m.tableRow--
}
m.queryIdx = m.tableRow
}
case "space":
if n > 0 && m.tableRow < n {
m.request.QueryParams[m.tableRow].Enabled = !m.request.QueryParams[m.tableRow].Enabled
}
}
return nil
}

func (m *EditorModel) handleBodyNavKey(key string) tea.Cmd {
bt := m.request.Body.Type
n := len(m.request.Body.FormData)
switch key {
case "t":
m.cycleBodyType()
case "j", "down":
if (bt == models.BodyFormData || bt == models.BodyURLEncoded) && m.tableRow < n-1 {
m.tableRow++
}
case "k", "up":
if (bt == models.BodyFormData || bt == models.BodyURLEncoded) && m.tableRow > 0 {
m.tableRow--
}
case "tab":
if bt == models.BodyFormData {
if m.tableCol < 3 {
m.tableCol++
} else {
m.tableCol = 1
}
} else if bt == models.BodyURLEncoded {
if m.tableCol == 1 {
m.tableCol = 2
} else {
m.tableCol = 1
}
}
case "enter", "i":
if bt == models.BodyRaw || bt == models.BodyJSON {
m.editingField = "body_raw"
m.fieldValue = m.request.Body.Content
m.cursorPos = len([]rune(m.fieldValue))
m.tableEditing = true
} else if (bt == models.BodyFormData || bt == models.BodyURLEncoded) && n > 0 {
if bt == models.BodyFormData && m.tableCol == 3 {
m.toggleFormDataType()
} else {
m.startCellEdit(m.currentCellValue())
}
}
case "n":
if bt == models.BodyFormData || bt == models.BodyURLEncoded {
m.request.Body.FormData = append(m.request.Body.FormData, models.FormField{
Enabled: true,
Type:    models.FormFieldText,
})
m.tableRow = len(m.request.Body.FormData) - 1
m.tableCol = 1
m.startCellEdit("")
}
case "d":
if (bt == models.BodyFormData || bt == models.BodyURLEncoded) && n > 0 && m.tableRow < n {
m.request.Body.FormData = append(m.request.Body.FormData[:m.tableRow], m.request.Body.FormData[m.tableRow+1:]...)
if m.tableRow >= len(m.request.Body.FormData) && m.tableRow > 0 {
m.tableRow--
}
}
case "space":
if bt == models.BodyFormData && n > 0 && m.tableRow < n {
if m.tableCol == 3 {
m.toggleFormDataType()
} else {
m.request.Body.FormData[m.tableRow].Enabled = !m.request.Body.FormData[m.tableRow].Enabled
}
} else if bt == models.BodyURLEncoded && n > 0 && m.tableRow < n {
m.request.Body.FormData[m.tableRow].Enabled = !m.request.Body.FormData[m.tableRow].Enabled
}
case "f":
if bt == models.BodyFormData && n > 0 && m.tableRow < n {
m.toggleFormDataType()
}
}
return nil
}

func (m *EditorModel) handleAuthNavKey(key string) tea.Cmd {
switch key {
case "enter", "i":
m.editingField = "auth_token"
m.fieldValue = m.request.Auth.Token
m.cursorPos = len([]rune(m.fieldValue))
m.tableEditing = true
}
return nil
}

func (m *EditorModel) handleTextEditKey(key string) tea.Cmd {
switch key {
case "esc":
m.cancelCellEdit()
case "enter":
if m.editingField == "table_cell" {
m.commitCellEdit(false)
} else {
m.storeTextEdit()
m.editingField = "table_nav"
m.fieldValue = ""
m.cursorPos = 0
m.tableEditing = false
}
case "tab":
if m.editingField == "table_cell" {
m.commitCellEdit(true)
}
case "left":
if m.cursorPos > 0 {
m.cursorPos--
}
case "right":
runes := []rune(m.fieldValue)
if m.cursorPos < len(runes) {
m.cursorPos++
}
case "home":
m.cursorPos = 0
case "end":
m.cursorPos = len([]rune(m.fieldValue))
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

func (m *EditorModel) startCellEdit(value string) {
m.editingField = "table_cell"
m.fieldValue = value
m.cursorPos = len([]rune(value))
m.tableEditing = true
}

func (m *EditorModel) cancelCellEdit() {
m.editingField = "table_nav"
m.fieldValue = ""
m.cursorPos = 0
m.tableEditing = false
}

func (m *EditorModel) commitCellEdit(advanceCol bool) {
m.storeTableCellValue()
m.tableEditing = false
if advanceCol {
maxCol := 2
if m.activeTab == TabBody && m.request.Body.Type == models.BodyFormData {
maxCol = 3
}
if m.tableCol < maxCol {
m.tableCol++
if m.tableCol == 3 {
m.editingField = "table_nav"
m.fieldValue = ""
m.cursorPos = 0
return
}
m.editingField = "table_cell"
m.fieldValue = m.currentCellValue()
m.cursorPos = len([]rune(m.fieldValue))
m.tableEditing = true
return
}
m.tableCol = 1
}
m.editingField = "table_nav"
m.fieldValue = ""
m.cursorPos = 0
}

func (m *EditorModel) storeTextEdit() {
switch m.editingField {
case "body_raw":
m.request.Body.Content = m.fieldValue
case "auth_token":
m.request.Auth.Token = m.fieldValue
if m.request.Auth.Type == models.AuthNone {
m.request.Auth.Type = models.AuthBearer
}
}
}

func (m *EditorModel) storeTableCellValue() {
switch m.activeTab {
case TabURL:
if m.tableRow == 1 {
m.request.URL = m.fieldValue
}
case TabHeaders:
if m.tableRow < len(m.request.Headers) {
if m.tableCol == 1 {
m.request.Headers[m.tableRow].Key = m.fieldValue
} else {
m.request.Headers[m.tableRow].Value = m.fieldValue
}
}
case TabQuery:
if m.tableRow < len(m.request.QueryParams) {
if m.tableCol == 1 {
m.request.QueryParams[m.tableRow].Key = m.fieldValue
} else {
m.request.QueryParams[m.tableRow].Value = m.fieldValue
}
}
case TabBody:
if m.tableRow < len(m.request.Body.FormData) {
if m.tableCol == 1 {
m.request.Body.FormData[m.tableRow].Key = m.fieldValue
} else if m.tableCol == 2 {
m.request.Body.FormData[m.tableRow].Value = m.fieldValue
}
}
}
}

func (m *EditorModel) currentCellValue() string {
switch m.activeTab {
case TabURL:
if m.tableRow == 1 {
return m.request.URL
}
case TabHeaders:
if m.tableRow < len(m.request.Headers) {
if m.tableCol == 1 {
return m.request.Headers[m.tableRow].Key
}
return m.request.Headers[m.tableRow].Value
}
case TabQuery:
if m.tableRow < len(m.request.QueryParams) {
if m.tableCol == 1 {
return m.request.QueryParams[m.tableRow].Key
}
return m.request.QueryParams[m.tableRow].Value
}
case TabBody:
if m.tableRow < len(m.request.Body.FormData) {
if m.tableCol == 1 {
return m.request.Body.FormData[m.tableRow].Key
}
return m.request.Body.FormData[m.tableRow].Value
}
}
return ""
}

func (m *EditorModel) cycleMethod(dir int) {
methods := models.AllMethods
for i, mm := range methods {
if mm == m.request.Method {
m.request.Method = methods[(i+dir+len(methods))%len(methods)]
return
}
}
m.request.Method = models.MethodGET
}

func (m *EditorModel) cycleBodyType() {
m.bodyTypeIdx = (m.bodyTypeIdx + 1) % len(m.bodyTypes)
m.request.Body.Type = m.bodyTypes[m.bodyTypeIdx]
m.tableRow = 0
m.tableCol = 1
}

func (m *EditorModel) toggleFormDataType() {
if m.tableRow < len(m.request.Body.FormData) {
if m.request.Body.FormData[m.tableRow].Type == models.FormFieldFile {
m.request.Body.FormData[m.tableRow].Type = models.FormFieldText
} else {
m.request.Body.FormData[m.tableRow].Type = models.FormFieldFile
}
}
}

func (m *EditorModel) StartEditing(req *models.Request) {
if req != nil {
m.request = req
}
m.startEditing()
}

func (m *EditorModel) Reset() {
m.editingField = ""
m.fieldValue = ""
m.cursorPos = 0
m.tableEditing = false
}

func (m *EditorModel) startEditing() {
if m.request == nil {
return
}
m.editingField = "table_nav"
m.tableEditing = false
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
style = style.Bold(true).Underline(true).Foreground(lipgloss.Color("#00d7ff"))
} else {
style = style.Foreground(lipgloss.Color("#626262"))
}
parts = append(parts, style.Render(label))
}
return strings.Join(parts, " ")
}

func (m *EditorModel) renderTabContent() string {
if m.request == nil {
return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Select a request from the list")
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

func (m *EditorModel) contentWidth() int {
w := m.width - 4
if w < 30 {
return 30
}
return w
}

func (m *EditorModel) renderCursor() string {
runes := []rune(m.fieldValue)
before := string(runes[:m.cursorPos])
after := string(runes[m.cursorPos:])
return lipgloss.NewStyle().
Foreground(lipgloss.Color("#ffffff")).
Background(lipgloss.Color("#1c1c2c")).
Render(before + "█" + after)
}

func (m *EditorModel) cellVal(rowIdx, colIdx int, raw string) string {
isRow := m.editingField != "" && rowIdx == m.tableRow
isCol := m.tableCol == colIdx
if isRow && m.tableEditing && isCol {
return m.renderCursor()
}
if isRow && !m.tableEditing && isCol {
return lipgloss.NewStyle().Foreground(lipgloss.Color("#00d7ff")).Bold(true).Render(raw)
}
return raw
}

func (m *EditorModel) rowPtr(rowIdx int) string {
if m.editingField != "" && rowIdx == m.tableRow {
return lipgloss.NewStyle().Foreground(lipgloss.Color("#00d7ff")).Render("> ")
}
return "  "
}

func (m *EditorModel) renderURLTab() string {
cw := m.contentWidth()
dim := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
lbl := lipgloss.NewStyle().Foreground(lipgloss.Color("#87d7ff"))

hdr := dim.Render("  " + padRight("Field", 10) + " Value")
sep := dim.Render("  " + strings.Repeat("─", cw-2))

row0 := m.rowPtr(0) + lbl.Render(padRight("Method", 10)) + " " + m.cellVal(0, 1, string(m.request.Method))
row1 := m.rowPtr(1) + lbl.Render(padRight("URL", 10)) + " " + m.cellVal(1, 1, m.request.URL)

hint := dim.Render("  ←/→ or h/l: cycle method  enter/i: edit URL  j/k: move  [ ]: tabs")
return strings.Join([]string{hdr, sep, row0, row1, "", hint}, "\n")
}

func (m *EditorModel) renderHeadersTab() string {
dim := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
enOn := lipgloss.NewStyle().Foreground(lipgloss.Color("#00d700"))
cw := m.contentWidth()
keyW := 22
valW := cw - 2 - 3 - keyW - 2
if valW < 10 {
valW = 10
}

hdr := dim.Render("  " + padRight("✓", 3) + padRight("Key", keyW+1) + "Value")
sep := dim.Render("  " + strings.Repeat("─", cw-2))

var rows []string
for i, h := range m.request.Headers {
en := enOn.Render("✓")
if !h.Enabled {
en = dim.Render("✗")
}
key := m.cellVal(i, 1, padRight(h.Key, keyW))
val := m.cellVal(i, 2, truncate(h.Value, valW))
rows = append(rows, m.rowPtr(i)+en+"  "+key+" "+val)
}
if len(rows) == 0 {
rows = append(rows, dim.Render("  (no headers)"))
}

hint := dim.Render("  [n] add  [d] delete  [space] toggle  [enter/i] edit  [tab] next col")
parts := []string{hdr, sep}
parts = append(parts, rows...)
parts = append(parts, "", hint)
return strings.Join(parts, "\n")
}

func (m *EditorModel) renderQueryTab() string {
dim := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
enOn := lipgloss.NewStyle().Foreground(lipgloss.Color("#00d700"))
cw := m.contentWidth()
keyW := 22
valW := cw - 2 - 3 - keyW - 2
if valW < 10 {
valW = 10
}

hdr := dim.Render("  " + padRight("✓", 3) + padRight("Key", keyW+1) + "Value")
sep := dim.Render("  " + strings.Repeat("─", cw-2))

var rows []string
for i, p := range m.request.QueryParams {
en := enOn.Render("✓")
if !p.Enabled {
en = dim.Render("✗")
}
key := m.cellVal(i, 1, padRight(p.Key, keyW))
val := m.cellVal(i, 2, truncate(p.Value, valW))
rows = append(rows, m.rowPtr(i)+en+"  "+key+" "+val)
}
if len(rows) == 0 {
rows = append(rows, dim.Render("  (no query parameters)"))
}

hint := dim.Render("  [n] add  [d] delete  [space] toggle  [enter/i] edit  [tab] next col")
parts := []string{hdr, sep}
parts = append(parts, rows...)
parts = append(parts, "", hint)
return strings.Join(parts, "\n")
}

func (m *EditorModel) renderBodyTab() string {
dim := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
lbl := lipgloss.NewStyle().Foreground(lipgloss.Color("#87d7ff"))
act := lipgloss.NewStyle().Foreground(lipgloss.Color("#00d7ff")).Bold(true)

typeStr := lbl.Render("Type: ") + act.Render(string(m.request.Body.Type))
typeHint := dim.Render("  [t] cycle: none → raw → json → form-data → urlencoded")

var content string
switch m.request.Body.Type {
case models.BodyNone:
content = dim.Render("  (no body)")
case models.BodyRaw, models.BodyJSON:
if m.editingField == "body_raw" {
content = m.renderCursor()
content += "\n" + dim.Render("  [enter] confirm  [esc] cancel  ←/→ cursor")
} else {
body := m.request.Body.Content
if body == "" {
body = dim.Render("  (empty body)  [enter/i] to edit")
}
content = body
}
case models.BodyURLEncoded:
content = m.renderFormTable(false)
case models.BodyFormData:
content = m.renderFormTable(true)
}

return strings.Join([]string{typeStr, typeHint, "", content}, "\n")
}

func (m *EditorModel) renderFormTable(withType bool) string {
dim := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
enOn := lipgloss.NewStyle().Foreground(lipgloss.Color("#00d700"))
cw := m.contentWidth()

var keyW, valW, typeW int
if withType {
keyW = 14
typeW = 8
valW = cw - 2 - 3 - keyW - typeW - 4
if valW < 10 {
valW = 10
}
} else {
keyW = 22
valW = cw - 2 - 3 - keyW - 2
if valW < 10 {
valW = 10
}
}

hdrStr := "  " + padRight("✓", 3) + padRight("Key", keyW+1) + padRight("Value", valW+1)
if withType {
hdrStr += "Type"
}
hdr := dim.Render(hdrStr)
sep := dim.Render("  " + strings.Repeat("─", cw-2))

var rows []string
for i, f := range m.request.Body.FormData {
en := enOn.Render("✓")
if !f.Enabled {
en = dim.Render("✗")
}
key := m.cellVal(i, 1, padRight(f.Key, keyW))
val := m.cellVal(i, 2, truncate(f.Value, valW))
row := m.rowPtr(i) + en + "  " + key + " " + val
if withType {
tv := string(f.Type)
if tv == "" {
tv = "text"
}
row += " " + m.cellVal(i, 3, tv)
}
rows = append(rows, row)
}
if len(rows) == 0 {
rows = append(rows, dim.Render("  (no fields)"))
}

hintStr := "[n] add  [d] delete  [space] toggle  [enter/i] edit  [tab] col"
if withType {
hintStr += "  [f] toggle text/file"
}
hint := dim.Render("  " + hintStr)

parts := []string{hdr, sep}
parts = append(parts, rows...)
parts = append(parts, "", hint)
return strings.Join(parts, "\n")
}

func (m *EditorModel) renderAuthTab() string {
dim := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
lbl := lipgloss.NewStyle().Foreground(lipgloss.Color("#87d7ff"))

if m.editingField == "auth_token" {
input := lbl.Render("Bearer token: ") + m.renderCursor()
return input + "\n" + dim.Render("  [enter] confirm  [esc] cancel  ←/→ cursor")
}

authLabel := lbl.Render("Type: ")
authType := lipgloss.NewStyle().Foreground(lipgloss.Color("#e4e4e4")).Render(string(m.request.Auth.Type))

details := ""
switch m.request.Auth.Type {
case models.AuthBasic:
details = fmt.Sprintf("\n%s%s\n%s%s",
lbl.Render("Username: "), m.request.Auth.Username,
lbl.Render("Password: "), maskPassword(m.request.Auth.Password))
case models.AuthBearer:
details = fmt.Sprintf("\n%s%s", lbl.Render("Token: "), maskToken(m.request.Auth.Token))
case models.AuthAPIKey:
details = fmt.Sprintf("\n%s%s\n%s%s\n%s%s",
lbl.Render("Key: "), m.request.Auth.Key,
lbl.Render("Value: "), m.request.Auth.Value,
lbl.Render("In: "), m.request.Auth.In)
}

hint := "\n" + dim.Render("  [i/enter] edit token")
return authLabel + authType + details + hint
}

func padRight(s string, width int) string {
runes := []rune(s)
if len(runes) >= width {
return string(runes[:width])
}
return s + strings.Repeat(" ", width-len(runes))
}

func truncate(s string, width int) string {
runes := []rune(s)
if len(runes) <= width {
return s
}
return string(runes[:width-1]) + "…"
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
