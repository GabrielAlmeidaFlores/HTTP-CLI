package ui

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/http-cli/internal/models"
	"github.com/user/http-cli/internal/ui/keybindings"
)

type kvRow struct {
	enabled bool
	key     string
	value   string
	isFile  bool
}

type kvTable struct {
	rows         []kvRow
	rowIdx       int
	colIdx       int
	scrollOffset int
	visibleRows  int
	editing      bool
	editVal      string
	editCursor   int
	showFileType bool
	km           *keybindings.Manager
}

func newKvTable(rows []kvRow, km *keybindings.Manager) kvTable {
	t := kvTable{colIdx: 1, km: km}
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
	case "ctrl+v":
		text, err := clipboard.ReadAll()
		if err == nil {
			t.editVal, t.editCursor = insertAtCursor(t.editVal, t.editCursor, text)
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

	action := ""
	if t.km != nil {
		if b, ok := t.km.Resolve(key, "editor"); ok && b.Panel == "editor" {
			action = b.Action
		}
	}

	switch key {
	case "down":
		if t.rowIdx < n-1 {
			t.rowIdx++
		} else {
			t.rows = append(t.rows, kvRow{enabled: true})
			t.rowIdx++
		}
		t.ensureVisible()
		return true
	case "up":
		if t.rowIdx > 0 {
			t.rowIdx--
		}
		t.ensureVisible()
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
	}

	switch action {
	case "insert_mode":
		if t.colIdx > 0 && n > 0 {
			t.startEdit("")
		}
		return true
	case "toggle_enabled":
		if t.colIdx == 0 && n > 0 {
			t.rows[t.rowIdx].enabled = !t.rows[t.rowIdx].enabled
			return true
		}
		return false
	case "delete_row":
		if n > 0 {
			t.rows = append(t.rows[:t.rowIdx], t.rows[t.rowIdx+1:]...)
			if len(t.rows) == 0 {
				t.rows = append(t.rows, kvRow{enabled: true})
			}
			if t.rowIdx >= len(t.rows) {
				t.rowIdx = len(t.rows) - 1
			}
			t.ensureVisible()
		}
		return true
	case "toggle_type":
		if t.showFileType && n > 0 {
			t.rows[t.rowIdx].isFile = !t.rows[t.rowIdx].isFile
			return true
		}
	}

	return false
}

func (t *kvTable) ensureVisible() {
	if t.visibleRows <= 0 {
		return
	}
	if t.rowIdx < t.scrollOffset {
		t.scrollOffset = t.rowIdx
	}
	if t.rowIdx >= t.scrollOffset+t.visibleRows {
		t.scrollOffset = t.rowIdx - t.visibleRows + 1
	}
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
			ft := models.FormFieldText
			if r.isFile {
				ft = models.FormFieldFile
			}
			out = append(out, models.FormField{Key: r.key, Value: r.value, Enabled: r.enabled, Type: ft})
		}
	}
	return out
}

func (t *kvTable) render(width int, insertMode bool) string {
	return t.renderWithMaxRows(width, insertMode, 0)
}

func (t *kvTable) renderWithMaxRows(width int, insertMode bool, maxRows int) string {
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
	hdrStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#87d7ff"))
	enOn := lipgloss.NewStyle().Foreground(lipgloss.Color("#00af00"))
	enOff := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
	fileTag := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#d7af00"))
	textTag := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
	rowBg := lipgloss.NewStyle().Background(lipgloss.Color("#1c1c2c"))
	cellBg := lipgloss.NewStyle().Background(lipgloss.Color("#005f87")).Bold(true)
	placeholder := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
	ptrStyle := accentStyle()

	typeColW := 0
	if t.showFileType {
		typeColW = 6
	}

	keyW := 20
	valW := width - 3 - keyW - 4 - typeColW
	if valW < 10 {
		valW = 10
	}

	hdrType := ""
	if t.showFileType {
		hdrType = hdrStyle.Render(padRight("Type", typeColW))
	}
	hdr := hdrStyle.Render("  "+padRight("✓", 3)+padRight("Key", keyW+1)) + hdrType + hdrStyle.Render("Value")
	sep := dim.Render("  " + strings.Repeat("─", width-4))

	if maxRows > 0 {
		t.visibleRows = maxRows
		t.ensureVisible()
	}

	start := 0
	end := len(t.rows)
	if maxRows > 0 && len(t.rows) > maxRows {
		start = t.scrollOffset
		end = start + maxRows
		if end > len(t.rows) {
			end = len(t.rows)
		}
	}

	var rows []string
	for i := start; i < end; i++ {
		r := t.rows[i]
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
		if r.isFile && valDisplay == "" {
			valDisplay = placeholder.Render(padRight("/path/to/file", valW))
		} else if valDisplay == "" {
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

		typeStr := ""
		if t.showFileType {
			if r.isFile {
				typeStr = fileTag.Render(padRight("FILE", typeColW))
			} else {
				typeStr = textTag.Render(padRight("text", typeColW))
			}
			if isCurrentRow && t.colIdx == 3 {
				label := "text"
				if r.isFile {
					label = "FILE"
				}
				typeStr = cellBg.Width(typeColW).Render(padRight(label, typeColW))
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
			default:
				keyCell = keyDisplay
				valCell = valDisplay
			}
		} else {
			keyCell = keyDisplay
			valCell = valDisplay
		}

		ptr := "  "
		if isCurrentRow && insertMode {
			ptr = ptrStyle.Render("> ")
		}

		line := ptr + enStr + "  " + keyCell + " " + typeStr + valCell
		if isCurrentRow && insertMode {
			line = rowBg.Render(line)
		}
		rows = append(rows, line)
	}

	if len(rows) == 0 {
		rows = append(rows, dim.Render("  (empty)"))
	}

	scrollIndicator := ""
	if maxRows > 0 && len(t.rows) > maxRows {
		scrollIndicator = dim.Render(fmt.Sprintf("  ↑↓ %d/%d", t.rowIdx+1, len(t.rows)))
	}

	parts := []string{hdr, sep}
	parts = append(parts, rows...)
	if scrollIndicator != "" {
		parts = append(parts, scrollIndicator)
	}
	return strings.Join(parts, "\n")
}

func renderEditCursor(val string, cursor int, maxWidth int) string {
	runes := []rune(val)
	n := len(runes)

	contentWidth := maxWidth - 1
	if contentWidth < 1 {
		contentWidth = 1
	}

	var start int
	if n <= contentWidth {
		start = 0
	} else {
		edgePos := contentWidth - 2
		if edgePos < 1 {
			edgePos = 1
		}
		start = cursor - edgePos
		if start < 0 {
			start = 0
		}
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
