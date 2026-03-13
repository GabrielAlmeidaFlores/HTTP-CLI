package ui

import (
"strings"

"github.com/charmbracelet/lipgloss"
"github.com/user/http-cli/internal/models"
)

type selectBox struct {
options []string
current int
open    bool
}

func newSelectBox(options []string, initial string) selectBox {
sb := selectBox{options: options}
for i, o := range options {
if o == initial {
sb.current = i
break
}
}
return sb
}

func (s *selectBox) value() string {
if len(s.options) == 0 {
return ""
}
return s.options[s.current]
}

func (s *selectBox) set(val string) {
for i, o := range s.options {
if o == val {
s.current = i
return
}
}
}

func (s *selectBox) next() {
if len(s.options) > 0 {
s.current = (s.current + 1) % len(s.options)
}
}

func (s *selectBox) prev() {
if len(s.options) > 0 {
s.current = (s.current - 1 + len(s.options)) % len(s.options)
}
}

func (s *selectBox) handleKey(key string) (bool, bool) {
if s.open {
switch key {
case "down":
prev := s.current
s.next()
return true, s.current != prev
case "up":
prev := s.current
s.prev()
return true, s.current != prev
case "enter", " ":
s.open = false
return true, false
case "esc":
s.open = false
return true, false
}
return false, false
}
switch key {
case "enter", " ", "e":
s.open = true
return true, false
case "right":
prev := s.current
s.next()
return true, s.current != prev
case "left":
prev := s.current
s.prev()
return true, s.current != prev
}
return false, false
}

func (s *selectBox) isOpen() bool {
return s.open
}

func (s *selectBox) renderInline(focused bool) string {
val := s.value()
if !focused {
return "[" + val + " ▾]"
}
if !s.open {
return lipgloss.NewStyle().
Bold(true).
Foreground(lipgloss.Color("#00d7ff")).
Render("[" + val + " ▾]")
}
header := lipgloss.NewStyle().
Bold(true).
Foreground(lipgloss.Color("#00d7ff")).
Render("[" + val + " ▾]")
var items []string
for i, opt := range s.options {
if i == s.current {
items = append(items, lipgloss.NewStyle().
Background(lipgloss.Color("#00d7ff")).
Foreground(lipgloss.Color("#000000")).
Render("▶ "+opt))
} else {
items = append(items, "  "+opt)
}
}
dropdown := lipgloss.NewStyle().
Border(lipgloss.RoundedBorder()).
BorderForeground(lipgloss.Color("#00d7ff")).
Render(strings.Join(items, "\n"))
return header + "\n" + dropdown
}

type kvRow struct {
enabled bool
key     string
value   string
}

type kvTable struct {
rows       []kvRow
rowIdx     int
colIdx     int
editing    bool
editVal    string
editCursor int
}

func newKvTable(rows []kvRow) kvTable {
t := kvTable{colIdx: 1}
if len(rows) > 0 {
t.rows = make([]kvRow, len(rows))
copy(t.rows, rows)
}
t.rows = append(t.rows, kvRow{enabled: true})
return t
}

func (t *kvTable) cancelEdit() {
t.editing = false
t.editVal = ""
t.editCursor = 0
}

func (t *kvTable) isSubEditing() bool {
return t.editing
}

func (t *kvTable) currentCellVal() string {
if t.rowIdx >= len(t.rows) {
return ""
}
if t.colIdx == 1 {
return t.rows[t.rowIdx].key
}
return t.rows[t.rowIdx].value
}

func (t *kvTable) startEdit(extra string) {
if t.colIdx == 0 || t.rowIdx >= len(t.rows) {
return
}
t.editVal = t.currentCellVal() + extra
t.editCursor = len([]rune(t.editVal))
t.editing = true
}

func (t *kvTable) commitEdit(advanceCol bool) {
if t.rowIdx < len(t.rows) {
if t.colIdx == 1 {
t.rows[t.rowIdx].key = t.editVal
} else if t.colIdx == 2 {
t.rows[t.rowIdx].value = t.editVal
}
}
t.editing = false
t.editVal = ""
t.editCursor = 0
if advanceCol {
if t.colIdx < 2 {
t.colIdx++
t.startEdit("")
} else {
t.colIdx = 1
}
}
}

func (t *kvTable) handleKey(key string) bool {
if t.editing {
return t.handleEditKey(key)
}
return t.handleNavKey(key)
}

func (t *kvTable) handleEditKey(key string) bool {
switch key {
case "esc":
	t.cancelEdit()
	return true
case "enter":
t.commitEdit(false)
return true
case "tab":
t.commitEdit(true)
return true
case "backspace":
if t.editCursor > 0 {
runes := []rune(t.editVal)
t.editVal = string(runes[:t.editCursor-1]) + string(runes[t.editCursor:])
t.editCursor--
}
return true
case "left":
if t.editCursor > 0 {
t.editCursor--
}
return true
case "right":
if t.editCursor < len([]rune(t.editVal)) {
t.editCursor++
}
return true
default:
if isPrintable(key) {
runes := []rune(t.editVal)
r := []rune(key)[0]
newRunes := make([]rune, len(runes)+1)
copy(newRunes, runes[:t.editCursor])
newRunes[t.editCursor] = r
copy(newRunes[t.editCursor+1:], runes[t.editCursor:])
t.editVal = string(newRunes)
t.editCursor++
return true
}
}
return false
}

func (t *kvTable) handleNavKey(key string) bool {
n := len(t.rows)
switch key {
case "down":
if t.rowIdx < n-1 {
t.rowIdx++
} else {
t.rows = append(t.rows, kvRow{enabled: true})
t.rowIdx++
}
return true
case "up":
if t.rowIdx > 0 {
t.rowIdx--
}
return true
case "left":
if t.colIdx > 0 {
t.colIdx--
}
return true
case "right":
if t.colIdx < 2 {
t.colIdx++
}
return true
case "e":
if t.colIdx > 0 && n > 0 {
t.startEdit("")
}
return true
case " ":
if t.colIdx == 0 && n > 0 {
t.rows[t.rowIdx].enabled = !t.rows[t.rowIdx].enabled
return true
}
return false
case "d":
if n > 0 {
t.rows = append(t.rows[:t.rowIdx], t.rows[t.rowIdx+1:]...)
if len(t.rows) == 0 {
t.rows = append(t.rows, kvRow{enabled: true})
}
if t.rowIdx >= len(t.rows) {
t.rowIdx = len(t.rows) - 1
}
}
return true
}
return false
}

func (t *kvTable) toHeaders() []models.Header {
var out []models.Header
for _, r := range t.rows {
if r.key != "" {
out = append(out, models.Header{Key: r.key, Value: r.value, Enabled: r.enabled})
}
}
return out
}

func (t *kvTable) toQueryParams() []models.QueryParam {
var out []models.QueryParam
for _, r := range t.rows {
if r.key != "" {
out = append(out, models.QueryParam{Key: r.key, Value: r.value, Enabled: r.enabled})
}
}
return out
}

func (t *kvTable) toFormFields() []models.FormField {
var out []models.FormField
for _, r := range t.rows {
if r.key != "" {
out = append(out, models.FormField{Key: r.key, Value: r.value, Enabled: r.enabled, Type: models.FormFieldText})
}
}
return out
}

func (t *kvTable) render(width int, insertMode bool) string {
dim := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
hdrStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#87d7ff"))
enOn := lipgloss.NewStyle().Foreground(lipgloss.Color("#00af00"))
enOff := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
rowBg := lipgloss.NewStyle().Background(lipgloss.Color("#1c1c2c"))
cellBg := lipgloss.NewStyle().Background(lipgloss.Color("#005f87")).Bold(true)
placeholder := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))

keyW := 22
valW := width - 3 - keyW - 4
if valW < 10 {
valW = 10
}

hdr := hdrStyle.Render("  "+padRight("✓", 3)+padRight("Key", keyW+1)) + hdrStyle.Render("Value")
sep := dim.Render("  " + strings.Repeat("─", width-4))

var rows []string
for i, r := range t.rows {
isCurrentRow := insertMode && i == t.rowIdx

enStr := enOn.Render("✓")
if !r.enabled {
enStr = enOff.Render("✗")
}

keyDisplay := r.key
if keyDisplay == "" {
keyDisplay = placeholder.Render(padRight("…", keyW))
} else {
keyDisplay = padRight(r.key, keyW)
}

valDisplay := r.value
if valDisplay == "" {
valDisplay = placeholder.Render(padRight("…", valW))
} else {
valDisplay = truncate(r.value, valW)
}

if isCurrentRow && t.editing {
if t.colIdx == 1 {
keyDisplay = renderEditCursor(t.editVal, t.editCursor, keyW)
} else if t.colIdx == 2 {
valDisplay = renderEditCursor(t.editVal, t.editCursor, valW)
}
}

var keyCell, valCell string
if isCurrentRow && !t.editing {
switch t.colIdx {
case 0:
enStr = cellBg.Render(enStr)
keyCell = keyDisplay
valCell = valDisplay
case 1:
keyCell = cellBg.Width(keyW).Render(padRight(r.key, keyW))
valCell = valDisplay
case 2:
keyCell = keyDisplay
valCell = cellBg.Width(valW).Render(padRight(r.value, valW))
}
} else {
keyCell = keyDisplay
valCell = valDisplay
}

ptr := "  "
if isCurrentRow && insertMode {
ptr = lipgloss.NewStyle().Foreground(lipgloss.Color("#00d7ff")).Render("> ")
}

line := ptr + enStr + "  " + keyCell + " " + valCell
if isCurrentRow && insertMode {
line = rowBg.Render(line)
}
rows = append(rows, line)
}

if len(rows) == 0 {
rows = append(rows, dim.Render("  (empty)"))
}

parts := []string{hdr, sep}
parts = append(parts, rows...)
return strings.Join(parts, "\n")
}

func renderEditCursor(val string, cursor int, maxWidth int) string {
runes := []rune(val)
n := len(runes)

contentWidth := maxWidth - 1
if contentWidth < 1 {
contentWidth = 1
}

idealBefore := contentWidth * 2 / 3
start := cursor - idealBefore
if start < 0 {
start = 0
}

beforeRunes := runes[start:cursor]

afterCount := contentWidth - len(beforeRunes)
afterEnd := cursor + afterCount
if afterEnd > n {
afterEnd = n
}
afterRunes := runes[cursor:afterEnd]

beforeStr := string(beforeRunes)
afterStr := string(afterRunes)

if start > 0 && len(beforeRunes) > 0 {
br := []rune(beforeStr)
br[0] = '‹'
beforeStr = string(br)
} else if start > 0 {
beforeStr = "‹"
}

result := padRight(beforeStr+"█"+afterStr, maxWidth)

return lipgloss.NewStyle().
Background(lipgloss.Color("#005f87")).
Foreground(lipgloss.Color("#ffffff")).
Bold(true).
Render(result)
}

func isPrintable(key string) bool {
runes := []rune(key)
if len(runes) != 1 {
return false
}
r := runes[0]
return r >= 32 && r != 127
}
