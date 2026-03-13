package ui

import (
"fmt"
"strings"

"github.com/atotto/clipboard"
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
keybindMgr *keybindings.Manager
request    *models.Request
activeTab  EditorTab
width      int
height     int

urlRowIdx  int
methodSel  selectBox
urlEditing bool
urlEditVal string
urlCursor  int

headersTable kvTable

bodyRowIdx    int
bodyTypeSel   selectBox
bodyEditing   bool
bodyEditVal   string
bodyCursor    int
bodyFormTable kvTable

queryTable kvTable

authRowIdx  int
authTypeSel selectBox
authEditing bool
authEditVal string
authCursor  int
}

func newEditorModel(km *keybindings.Manager) EditorModel {
ft := newKvTable(nil, km)
ft.showFileType = true
return EditorModel{
keybindMgr:    km,
activeTab:     TabURL,
methodSel:     newSelectBox(methodOptions(), "GET"),
bodyTypeSel:   newSelectBox(bodyTypeOptions(), "none"),
authTypeSel:   newSelectBox(authTypeOptions(), "none"),
headersTable:  newKvTable(nil, km),
queryTable:    newKvTable(nil, km),
bodyFormTable: ft,
}
}

func methodOptions() []string {
return []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}
}

func bodyTypeOptions() []string {
return []string{"none", "json", "raw", "form-data", "urlencoded"}
}

func authTypeOptions() []string {
return []string{"none", "bearer", "basic", "apikey"}
}

func (m *EditorModel) setRequest(req *models.Request) {
m.request = req
m.syncFromRequest()
}

func (m *EditorModel) HasRequest() bool {
return m.request != nil
}

func (m *EditorModel) syncFromRequest() {
if m.request == nil {
return
}
m.methodSel = newSelectBox(methodOptions(), string(m.request.Method))
m.urlEditing = false
m.urlEditVal = m.request.URL
m.urlCursor = len([]rune(m.urlEditVal))
m.urlRowIdx = 0

hRows := make([]kvRow, len(m.request.Headers))
for i, h := range m.request.Headers {
hRows[i] = kvRow{enabled: h.Enabled, key: h.Key, value: h.Value}
}
m.headersTable = newKvTable(hRows, m.keybindMgr)

m.bodyTypeSel = newSelectBox(bodyTypeOptions(), string(m.request.Body.Type))
m.bodyEditing = false
m.bodyEditVal = m.request.Body.Content
m.bodyCursor = len([]rune(m.bodyEditVal))
m.bodyRowIdx = 0
fRows := make([]kvRow, len(m.request.Body.FormData))
for i, f := range m.request.Body.FormData {
fRows[i] = kvRow{enabled: f.Enabled, key: f.Key, value: f.Value, isFile: f.Type == models.FormFieldFile}
}
{ ft := newKvTable(fRows, m.keybindMgr); ft.showFileType = true; m.bodyFormTable = ft }

qRows := make([]kvRow, len(m.request.QueryParams))
for i, p := range m.request.QueryParams {
qRows[i] = kvRow{enabled: p.Enabled, key: p.Key, value: p.Value}
}
m.queryTable = newKvTable(qRows, m.keybindMgr)

m.authTypeSel = newSelectBox(authTypeOptions(), string(m.request.Auth.Type))
m.authEditing = false
m.authEditVal = ""
m.authCursor = 0
m.authRowIdx = 0
}

func (m *EditorModel) syncToRequest() {
if m.request == nil {
return
}
m.request.Method = models.HTTPMethod(m.methodSel.value())
m.request.URL = m.urlEditVal
m.request.Headers = m.headersTable.toHeaders()
m.request.Body.Type = models.BodyType(m.bodyTypeSel.value())
m.request.Body.Content = m.bodyEditVal
m.request.Body.FormData = m.bodyFormTable.toFormFields()
m.request.QueryParams = m.queryTable.toQueryParams()
m.request.Auth.Type = models.AuthType(m.authTypeSel.value())
}

func (m *EditorModel) setSize(w, h int) {
m.width = w
m.height = h
}

func (m *EditorModel) IsSubEditing() bool {
switch m.activeTab {
case TabURL:
return m.methodSel.isOpen() || m.urlEditing
case TabHeaders:
return m.headersTable.isSubEditing()
case TabBody:
return m.bodyTypeSel.isOpen() || m.bodyEditing || m.bodyFormTable.isSubEditing()
case TabQuery:
return m.queryTable.isSubEditing()
case TabAuth:
return m.authTypeSel.isOpen() || m.authEditing
}
return false
}

func (m *EditorModel) CancelSubEdit() {
switch m.activeTab {
case TabURL:
if m.methodSel.isOpen() {
m.methodSel.open = false
} else if m.urlEditing {
m.urlEditing = false
m.urlEditVal = m.request.URL
m.urlCursor = len([]rune(m.urlEditVal))
}
case TabHeaders:
m.headersTable.cancelEdit()
case TabBody:
if m.bodyTypeSel.isOpen() {
m.bodyTypeSel.open = false
} else if m.bodyEditing {
m.bodyEditing = false
m.bodyEditVal = m.request.Body.Content
m.bodyCursor = len([]rune(m.bodyEditVal))
} else {
m.bodyFormTable.cancelEdit()
}
case TabQuery:
m.queryTable.cancelEdit()
case TabAuth:
if m.authTypeSel.isOpen() {
m.authTypeSel.open = false
} else if m.authEditing {
m.authEditing = false
m.authEditVal = ""
m.authCursor = 0
}
}
}

func (m *EditorModel) Reset() {
m.methodSel.open = false
m.urlEditing = false
if m.request != nil {
m.urlEditVal = m.request.URL
}
m.urlCursor = len([]rune(m.urlEditVal))
m.headersTable.cancelEdit()
m.bodyTypeSel.open = false
m.bodyEditing = false
if m.request != nil {
m.bodyEditVal = m.request.Body.Content
}
m.bodyCursor = len([]rune(m.bodyEditVal))
m.bodyFormTable.cancelEdit()
m.queryTable.cancelEdit()
m.authTypeSel.open = false
m.authEditing = false
m.authEditVal = ""
m.authCursor = 0
}

func (m *EditorModel) StartEditing(req *models.Request) {
if req != nil {
m.request = req
m.syncFromRequest()
}
}

func (m *EditorModel) resolveAction(key string) string {
if m.keybindMgr == nil {
return ""
}
if b, ok := m.keybindMgr.Resolve(key, "editor"); ok && b.Panel == "editor" {
return b.Action
}
return ""
}

func (m *EditorModel) handleKey(msg tea.KeyMsg, req *models.Request) tea.Cmd {
if req == nil {
return nil
}
m.request = req
key := msg.String()

action := m.resolveAction(key)

if key == "esc" || action == "normal_mode" {
m.CancelSubEdit()
return nil
}

switch action {
case "next_tab":
m.syncToRequest()
m.nextTab()
m.syncFromRequest()
return nil
case "prev_tab":
m.syncToRequest()
m.prevTab()
m.syncFromRequest()
return nil
}

switch m.activeTab {
case TabURL:
m.handleURLKey(key, action)
case TabHeaders:
m.handleHeadersKey(key)
case TabBody:
m.handleBodyKey(key, action)
case TabQuery:
m.handleQueryKey(key)
case TabAuth:
m.handleAuthKey(key, action)
}

m.syncToRequest()
return nil
}

func (m *EditorModel) handleURLKey(key string, action string) {
if m.urlRowIdx == 0 {
if m.methodSel.isOpen() {
consumed, changed := m.methodSel.handleKey(key, action)
if consumed {
if changed {
m.request.Method = models.HTTPMethod(m.methodSel.value())
}
return
}
} else {
switch {
case key == "down":
m.urlRowIdx = 1
case key == "enter" || key == " " || action == "insert_mode":
m.methodSel.open = true
case key == "right":
m.methodSel.next()
case key == "left":
m.methodSel.prev()
}
}
return
}

if m.urlEditing {
switch key {
case "enter":
m.urlEditing = false
m.request.URL = m.urlEditVal
case "esc":
m.urlEditing = false
m.urlEditVal = m.request.URL
m.urlCursor = len([]rune(m.urlEditVal))
case "backspace":
if m.urlCursor > 0 {
runes := []rune(m.urlEditVal)
m.urlEditVal = string(runes[:m.urlCursor-1]) + string(runes[m.urlCursor:])
m.urlCursor--
}
case "left":
if m.urlCursor > 0 {
m.urlCursor--
}
case "right":
if m.urlCursor < len([]rune(m.urlEditVal)) {
m.urlCursor++
}
case "ctrl+v":
text, err := clipboard.ReadAll()
if err == nil {
m.urlEditVal, m.urlCursor = insertAtCursor(m.urlEditVal, m.urlCursor, text)
}
default:
if isPrintable(key) {
runes := []rune(m.urlEditVal)
r := []rune(key)[0]
newRunes := make([]rune, len(runes)+1)
copy(newRunes, runes[:m.urlCursor])
newRunes[m.urlCursor] = r
copy(newRunes[m.urlCursor+1:], runes[m.urlCursor:])
m.urlEditVal = string(newRunes)
m.urlCursor++
}
}
} else {
switch {
case key == "up":
m.urlRowIdx = 0
case action == "insert_mode":
m.urlEditing = true
m.urlCursor = len([]rune(m.urlEditVal))
}
}
}

func (m *EditorModel) handleHeadersKey(key string) {
m.headersTable.handleKey(key)
}

func (m *EditorModel) handleQueryKey(key string) {
m.queryTable.handleKey(key)
}

func (m *EditorModel) handleBodyKey(key string, action string) {
bt := models.BodyType(m.bodyTypeSel.value())

if m.bodyRowIdx == 0 {
if m.bodyTypeSel.isOpen() {
consumed, changed := m.bodyTypeSel.handleKey(key, action)
if consumed {
if changed {
m.request.Body.Type = models.BodyType(m.bodyTypeSel.value())
}
return
}
} else {
switch {
case key == "down":
m.bodyRowIdx = 1
case key == "enter" || key == " " || action == "insert_mode":
m.bodyTypeSel.open = true
case key == "right":
m.bodyTypeSel.next()
case key == "left":
m.bodyTypeSel.prev()
}
}
return
}

bt = models.BodyType(m.bodyTypeSel.value())

switch bt {
case models.BodyNone:
if key == "up" {
m.bodyRowIdx = 0
}
case models.BodyRaw, models.BodyJSON:
m.handleBodyTextKey(key, action)
case models.BodyFormData, models.BodyURLEncoded:
if (key == "up") && !m.bodyFormTable.editing && m.bodyFormTable.rowIdx == 0 {
m.bodyRowIdx = 0
return
}
m.bodyFormTable.handleKey(key)
}
}

func (m *EditorModel) handleBodyTextKey(key string, action string) {
if m.bodyEditing {
switch key {
case "enter":
m.bodyEditing = false
m.request.Body.Content = m.bodyEditVal
case "esc":
m.bodyEditing = false
m.bodyEditVal = m.request.Body.Content
m.bodyCursor = len([]rune(m.bodyEditVal))
case "backspace":
if m.bodyCursor > 0 {
runes := []rune(m.bodyEditVal)
m.bodyEditVal = string(runes[:m.bodyCursor-1]) + string(runes[m.bodyCursor:])
m.bodyCursor--
}
case "left":
if m.bodyCursor > 0 {
m.bodyCursor--
}
case "right":
if m.bodyCursor < len([]rune(m.bodyEditVal)) {
m.bodyCursor++
}
case "ctrl+v":
text, err := clipboard.ReadAll()
if err == nil {
m.bodyEditVal, m.bodyCursor = insertAtCursor(m.bodyEditVal, m.bodyCursor, text)
}
default:
if isPrintable(key) {
runes := []rune(m.bodyEditVal)
r := []rune(key)[0]
newRunes := make([]rune, len(runes)+1)
copy(newRunes, runes[:m.bodyCursor])
newRunes[m.bodyCursor] = r
copy(newRunes[m.bodyCursor+1:], runes[m.bodyCursor:])
m.bodyEditVal = string(newRunes)
m.bodyCursor++
}
}
} else {
switch {
case key == "up":
m.bodyRowIdx = 0
case action == "insert_mode":
m.bodyEditing = true
m.bodyCursor = len([]rune(m.bodyEditVal))
}
}
}

func (m *EditorModel) authFieldNames() []string {
switch models.AuthType(m.authTypeSel.value()) {
case models.AuthBearer:
return []string{"Token"}
case models.AuthBasic:
return []string{"Username", "Password"}
case models.AuthAPIKey:
return []string{"Key", "Value", "In"}
}
return nil
}

func (m *EditorModel) getAuthFieldValue(idx int) string {
switch models.AuthType(m.authTypeSel.value()) {
case models.AuthBearer:
if idx == 0 {
return m.request.Auth.Token
}
case models.AuthBasic:
switch idx {
case 0:
return m.request.Auth.Username
case 1:
return m.request.Auth.Password
}
case models.AuthAPIKey:
switch idx {
case 0:
return m.request.Auth.Key
case 1:
return m.request.Auth.Value
case 2:
return m.request.Auth.In
}
}
return ""
}

func (m *EditorModel) setAuthFieldValue(idx int, val string) {
switch models.AuthType(m.authTypeSel.value()) {
case models.AuthBearer:
if idx == 0 {
m.request.Auth.Token = val
}
case models.AuthBasic:
switch idx {
case 0:
m.request.Auth.Username = val
case 1:
m.request.Auth.Password = val
}
case models.AuthAPIKey:
switch idx {
case 0:
m.request.Auth.Key = val
case 1:
m.request.Auth.Value = val
case 2:
m.request.Auth.In = val
}
}
}

func (m *EditorModel) handleAuthKey(key string, action string) {
fields := m.authFieldNames()

if m.authRowIdx == 0 {
if m.authTypeSel.isOpen() {
consumed, changed := m.authTypeSel.handleKey(key, action)
if consumed {
if changed {
m.request.Auth.Type = models.AuthType(m.authTypeSel.value())
m.authRowIdx = 0
}
return
}
} else {
switch {
case key == "down":
if len(fields) > 0 {
m.authRowIdx = 1
}
case key == "enter" || key == " " || action == "insert_mode":
m.authTypeSel.open = true
case key == "right":
m.authTypeSel.next()
m.request.Auth.Type = models.AuthType(m.authTypeSel.value())
case key == "left":
m.authTypeSel.prev()
m.request.Auth.Type = models.AuthType(m.authTypeSel.value())
}
}
return
}

fieldIdx := m.authRowIdx - 1
if m.authEditing {
switch key {
case "enter":
m.setAuthFieldValue(fieldIdx, m.authEditVal)
m.authEditing = false
case "esc":
m.authEditing = false
m.authEditVal = ""
m.authCursor = 0
case "backspace":
if m.authCursor > 0 {
runes := []rune(m.authEditVal)
m.authEditVal = string(runes[:m.authCursor-1]) + string(runes[m.authCursor:])
m.authCursor--
}
case "left":
if m.authCursor > 0 {
m.authCursor--
}
case "right":
if m.authCursor < len([]rune(m.authEditVal)) {
m.authCursor++
}
case "ctrl+v":
text, err := clipboard.ReadAll()
if err == nil {
m.authEditVal, m.authCursor = insertAtCursor(m.authEditVal, m.authCursor, text)
}
default:
if isPrintable(key) {
runes := []rune(m.authEditVal)
r := []rune(key)[0]
newRunes := make([]rune, len(runes)+1)
copy(newRunes, runes[:m.authCursor])
newRunes[m.authCursor] = r
copy(newRunes[m.authCursor+1:], runes[m.authCursor:])
m.authEditVal = string(newRunes)
m.authCursor++
}
}
return
}

switch {
case key == "down":
if fieldIdx < len(fields)-1 {
m.authRowIdx++
}
case key == "up":
if fieldIdx > 0 {
m.authRowIdx--
} else {
m.authRowIdx = 0
}
case action == "insert_mode":
m.authEditVal = m.getAuthFieldValue(fieldIdx)
m.authCursor = len([]rune(m.authEditVal))
m.authEditing = true
}
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
m.syncToRequest()
m.activeTab = editorTabs[n-1]
m.syncFromRequest()
}
}

func (m *EditorModel) ActiveTab() string {
return string(m.activeTab)
}

func (m *EditorModel) CurrentCellIsText() bool {
switch m.activeTab {
case TabURL:
return m.urlRowIdx == 1
case TabHeaders:
return m.headersTable.colIdx != 0
case TabBody:
if m.bodyRowIdx == 0 {
return false
}
bt := models.BodyType(m.bodyTypeSel.value())
if bt == models.BodyNone {
return false
}
if bt == models.BodyRaw || bt == models.BodyJSON {
return true
}
return m.bodyFormTable.colIdx != 0
case TabQuery:
return m.queryTable.colIdx != 0
case TabAuth:
return m.authRowIdx > 0
}
return false
}

func (m *EditorModel) CurrentCellTitle() string {
switch m.activeTab {
case TabURL:
return "URL"
case TabHeaders:
if m.headersTable.colIdx == 1 {
return "Header · Key"
}
return "Header · Value"
case TabBody:
bt := models.BodyType(m.bodyTypeSel.value())
if bt == models.BodyRaw || bt == models.BodyJSON {
return "Body Content"
}
if m.bodyFormTable.colIdx == 1 {
return "Form · Key"
}
isFileRow := m.bodyFormTable.rowIdx < len(m.bodyFormTable.rows) && m.bodyFormTable.rows[m.bodyFormTable.rowIdx].isFile
if isFileRow {
return "Form · File Path"
}
return "Form · Value"
case TabQuery:
if m.queryTable.colIdx == 1 {
return "Query · Key"
}
return "Query · Value"
case TabAuth:
fields := m.authFieldNames()
idx := m.authRowIdx - 1
if idx >= 0 && idx < len(fields) {
return "Auth · " + fields[idx]
}
return "Auth"
}
return "Edit"
}

func (m *EditorModel) CurrentCellValue() string {
switch m.activeTab {
case TabURL:
return m.urlEditVal
case TabHeaders:
return m.headersTable.currentCellVal()
case TabBody:
bt := models.BodyType(m.bodyTypeSel.value())
if bt == models.BodyRaw || bt == models.BodyJSON {
return m.bodyEditVal
}
return m.bodyFormTable.currentCellVal()
case TabQuery:
return m.queryTable.currentCellVal()
case TabAuth:
if m.authRowIdx > 0 {
return m.getAuthFieldValue(m.authRowIdx - 1)
}
}
return ""
}

func (m *EditorModel) CommitCellValue(val string) {
switch m.activeTab {
case TabURL:
m.urlEditVal = val
case TabHeaders:
if m.headersTable.rowIdx < len(m.headersTable.rows) {
if m.headersTable.colIdx == 1 {
m.headersTable.rows[m.headersTable.rowIdx].key = val
} else {
m.headersTable.rows[m.headersTable.rowIdx].value = val
}
}
case TabBody:
bt := models.BodyType(m.bodyTypeSel.value())
if bt == models.BodyRaw || bt == models.BodyJSON {
m.bodyEditVal = val
} else if m.bodyFormTable.rowIdx < len(m.bodyFormTable.rows) {
if m.bodyFormTable.colIdx == 1 {
m.bodyFormTable.rows[m.bodyFormTable.rowIdx].key = val
} else {
m.bodyFormTable.rows[m.bodyFormTable.rowIdx].value = val
}
}
case TabQuery:
if m.queryTable.rowIdx < len(m.queryTable.rows) {
if m.queryTable.colIdx == 1 {
m.queryTable.rows[m.queryTable.rowIdx].key = val
} else {
m.queryTable.rows[m.queryTable.rowIdx].value = val
}
}
case TabAuth:
if m.authRowIdx > 0 {
m.setAuthFieldValue(m.authRowIdx-1, val)
}
}
m.syncToRequest()
}

func (m *EditorModel) view(focused bool, theme config.ThemeConfig) string {
tabs := m.renderTabs()
content := m.renderTabContent(focused)
inner := lipgloss.JoinVertical(lipgloss.Left, tabs, content)
return panelBorderStyle(focused, theme).
Width(m.width).
Height(m.height).
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

func (m *EditorModel) renderTabContent(focused bool) string {
if m.request == nil {
return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Select a request from the list")
}
switch m.activeTab {
case TabURL:
return m.renderURLTab(focused)
case TabHeaders:
return m.renderHeadersTab(focused)
case TabBody:
return m.renderBodyTab(focused)
case TabQuery:
return m.renderQueryTab(focused)
case TabAuth:
return m.renderAuthTab(focused)
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

func (m *EditorModel) tableMaxRows() int {
rows := m.height - 10
if rows < 3 {
return 3
}
return rows
}

func (m *EditorModel) renderURLTab(focused bool) string {
dim := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
hdrStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#87d7ff"))
rowBg := lipgloss.NewStyle().Background(lipgloss.Color("#1c1c2c"))
cw := m.contentWidth()

hdr := hdrStyle.Render("  " + padRight("Field", 10) + " Value")
sep := dim.Render("  " + strings.Repeat("─", cw-2))

methodFocused := focused && m.urlRowIdx == 0
urlFocused := focused && m.urlRowIdx == 1

methodLine := "  " + padRight("Method", 10) + " " + m.methodSel.renderInline(methodFocused)
if methodFocused && !m.methodSel.isOpen() {
methodLine = rowBg.Render("> "+padRight("Method", 10)+" ") + m.methodSel.renderInline(methodFocused)
} else if methodFocused && m.methodSel.isOpen() {
methodLine = "> " + padRight("Method", 10) + " " + m.methodSel.renderInline(methodFocused)
}

var urlValStr string
if m.urlEditing && urlFocused {
urlValStr = renderEditCursor(m.urlEditVal, m.urlCursor, cw-14)
} else {
urlValStr = m.urlEditVal
if urlFocused {
urlValStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#00d7ff")).Render(urlValStr)
}
}

urlLine := "  " + padRight("URL", 10) + " " + urlValStr
if urlFocused {
urlLine = rowBg.Render("> "+padRight("URL", 10)+" ") + urlValStr
}

hint := dim.Render("  ↑↓ navigate  ←→ cycle method  e edit URL  1-5 tabs")
return strings.Join([]string{hdr, sep, methodLine, urlLine, "", hint}, "\n")
}

func (m *EditorModel) renderHeadersTab(focused bool) string {
dim := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
cw := m.contentWidth()
hdr := dim.Render("  " + strings.Repeat("─", cw-2))
hint := dim.Render("  ↑↓←→ navigate  e edit cell  space toggle  d delete")
content := m.headersTable.renderWithMaxRows(cw, focused, m.tableMaxRows())
return strings.Join([]string{content, hdr, hint}, "\n")
}

func (m *EditorModel) renderQueryTab(focused bool) string {
dim := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
cw := m.contentWidth()
hdr := dim.Render("  " + strings.Repeat("─", cw-2))
hint := dim.Render("  ↑↓←→ navigate  e edit cell  space toggle  d delete")
content := m.queryTable.renderWithMaxRows(cw, focused, m.tableMaxRows())
return strings.Join([]string{content, hdr, hint}, "\n")
}

func (m *EditorModel) renderBodyTab(focused bool) string {
dim := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
rowBg := lipgloss.NewStyle().Background(lipgloss.Color("#1c1c2c"))
cw := m.contentWidth()

typeFocused := focused && m.bodyRowIdx == 0
typeLine := "  " + padRight("Type", 10) + " " + m.bodyTypeSel.renderInline(typeFocused)
if typeFocused && !m.bodyTypeSel.isOpen() {
typeLine = rowBg.Render("> "+padRight("Type", 10)+" ") + m.bodyTypeSel.renderInline(typeFocused)
} else if typeFocused && m.bodyTypeSel.isOpen() {
typeLine = "> " + padRight("Type", 10) + " " + m.bodyTypeSel.renderInline(typeFocused)
}

bt := models.BodyType(m.bodyTypeSel.value())
var content string
contentFocused := focused && m.bodyRowIdx == 1

switch bt {
case models.BodyNone:
content = dim.Render("  (no body)")
case models.BodyRaw, models.BodyJSON:
if m.bodyEditing && contentFocused {
content = "  " + renderEditCursor(m.bodyEditVal, m.bodyCursor, cw-4)
} else {
body := m.bodyEditVal
if body == "" {
body = dim.Render("(empty — press enter to edit)")
}
if contentFocused {
content = rowBg.Render("> ") + body
} else {
content = "  " + body
}
}
case models.BodyFormData, models.BodyURLEncoded:
content = m.bodyFormTable.renderWithMaxRows(cw, contentFocused, m.tableMaxRows())
}

hint := dim.Render("  ↑↓ navigate  ←→ cycle type  e edit  1-5 tabs")
return strings.Join([]string{typeLine, "", content, "", hint}, "\n")
}

func (m *EditorModel) renderAuthTab(focused bool) string {
dim := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
lbl := lipgloss.NewStyle().Foreground(lipgloss.Color("#87d7ff"))
rowBg := lipgloss.NewStyle().Background(lipgloss.Color("#1c1c2c"))
hdrStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#87d7ff"))
cw := m.contentWidth()

typeFocused := focused && m.authRowIdx == 0
typeLine := "  " + padRight("Type", 10) + " " + m.authTypeSel.renderInline(typeFocused)
if typeFocused && !m.authTypeSel.isOpen() {
typeLine = rowBg.Render("> "+padRight("Type", 10)+" ") + m.authTypeSel.renderInline(typeFocused)
} else if typeFocused && m.authTypeSel.isOpen() {
typeLine = "> " + padRight("Type", 10) + " " + m.authTypeSel.renderInline(typeFocused)
}

fields := m.authFieldNames()
sep := dim.Render("  " + strings.Repeat("─", cw-2))
hdr := hdrStyle.Render("  " + padRight("Field", 12) + " Value")

var fieldLines []string
for i, name := range fields {
rowIdx := i + 1
isFocused := focused && m.authRowIdx == rowIdx
var valStr string
if isFocused && m.authEditing {
valStr = renderEditCursor(m.authEditVal, m.authCursor, cw-16)
} else {
val := m.getAuthFieldValue(i)
if name == "Password" || name == "Token" {
val = maskSecret(val)
}
valStr = val
if isFocused {
valStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#00d7ff")).Render(valStr)
}
}
ptr := "  "
if isFocused {
ptr = lipgloss.NewStyle().Foreground(lipgloss.Color("#00d7ff")).Render("> ")
}
line := ptr + lbl.Render(padRight(name, 12)) + " " + valStr
if isFocused {
line = rowBg.Render(line)
}
fieldLines = append(fieldLines, line)
}

hint := dim.Render("  ↑↓ navigate  ←→ cycle type  e edit field")
parts := []string{typeLine, hdr, sep}
parts = append(parts, fieldLines...)
parts = append(parts, "", hint)
return strings.Join(parts, "\n")
}

func maskSecret(s string) string {
if len(s) == 0 {
return ""
}
if len(s) <= 8 {
return strings.Repeat("*", len(s))
}
return s[:4] + strings.Repeat("*", len(s)-8) + s[len(s)-4:]
}


func (m *EditorModel) CurrentEditValue() string {
switch m.activeTab {
case TabURL:
if m.urlEditing {
return m.urlEditVal
}
case TabBody:
if m.bodyEditing {
return m.bodyEditVal
}
if m.bodyFormTable.isSubEditing() {
return m.bodyFormTable.editVal
}
case TabHeaders:
if m.headersTable.isSubEditing() {
return m.headersTable.editVal
}
case TabQuery:
if m.queryTable.isSubEditing() {
return m.queryTable.editVal
}
case TabAuth:
if m.authEditing {
return m.authEditVal
}
}
return ""
}

func (m *EditorModel) CommitExternalEdit(val string) {
switch m.activeTab {
case TabURL:
if m.urlEditing {
m.urlEditVal = val
m.urlEditing = false
m.urlCursor = len([]rune(val))
if m.request != nil {
m.request.URL = val
}
}
case TabBody:
if m.bodyEditing {
m.bodyEditVal = val
m.bodyEditing = false
m.bodyCursor = len([]rune(val))
if m.request != nil {
m.request.Body.Content = val
}
} else if m.bodyFormTable.isSubEditing() {
m.bodyFormTable.editVal = val
m.bodyFormTable.commitEdit(false)
}
case TabHeaders:
if m.headersTable.isSubEditing() {
m.headersTable.editVal = val
m.headersTable.commitEdit(false)
}
case TabQuery:
if m.queryTable.isSubEditing() {
m.queryTable.editVal = val
m.queryTable.commitEdit(false)
}
case TabAuth:
if m.authEditing {
m.authEditVal = val
m.authEditing = false
m.authCursor = len([]rune(val))
}
}
m.syncToRequest()
}
