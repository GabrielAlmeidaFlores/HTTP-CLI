package ui

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

func (a *App) promptConfirm(msg string, action func()) {
	a.showConfirm = true
	a.confirmMsg = msg
	a.confirmAction = action
}

func (a *App) promptInput(title, defaultVal string, action func(string)) {
	a.showInput = true
	a.inputTitle = title
	a.inputValue = defaultVal
	a.inputCursor = len([]rune(defaultVal))
	a.inputViewOffset = 0
	a.inputAction = action
}

func (a *App) showNotify(msg string, isErr bool) {
	a.notificationMsg = msg
	a.notificationIsErr = isErr
	a.showNotification = true
}

func (a *App) handleConfirmInput(msg tea.KeyMsg) tea.Cmd {
	if binding, ok := a.keybindMgr.Resolve(msg.String(), "confirm_modal"); ok {
		switch binding.Action {
		case "confirm":
			a.showConfirm = false
			if a.confirmAction != nil {
				a.confirmAction()
			}
		case "cancel":
			a.showConfirm = false
		}
	}
	return nil
}

func (a *App) handleInputDialog(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()
	runes := []rune(a.inputValue)
	n := len(runes)

	if binding, ok := a.keybindMgr.Resolve(key, "input_modal"); ok && binding.Panel == "input_modal" {
		switch binding.Action {
		case "confirm":
			a.showInput = false
			if a.inputAction != nil {
				a.inputAction(a.inputValue)
			}
			return nil
		case "cancel":
			a.showInput = false
			return nil
		}
	}

	switch key {
	case "backspace":
		if a.inputCursor > 0 {
			a.inputValue = string(runes[:a.inputCursor-1]) + string(runes[a.inputCursor:])
			a.inputCursor--
		}
	case "left":
		if a.inputCursor > 0 {
			a.inputCursor--
		}
	case "right":
		if a.inputCursor < n {
			a.inputCursor++
		}
	case "home", "ctrl+a":
		a.inputCursor = 0
	case "end":
		a.inputCursor = n
	default:
		if isPasteKey(key) {
			text, err := clipboard.ReadAll()
			if err == nil {
				a.inputValue, a.inputCursor = insertAtCursor(a.inputValue, a.inputCursor, text)
			}
		} else if len(key) == 1 {
			a.inputValue, a.inputCursor = insertAtCursor(a.inputValue, a.inputCursor, key)
		}
	}
	return nil
}

func (a *App) handleCellEditModal(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()
	runes := []rune(a.cellEditVal)
	n := len(runes)

	if binding, ok := a.keybindMgr.Resolve(key, "cell_edit_modal"); ok && binding.Panel == "cell_edit_modal" {
		switch binding.Action {
		case "confirm_exit":
			if a.cellEditCommit != nil {
				a.cellEditCommit(a.cellEditVal)
			}
			a.showCellEdit = false
			if a.selectedReq != nil {
				_ = a.store.SaveRequest(context.Background(), a.selectedReq)
				a.setStatus("Saved")
			}
			return nil
		case "save_only":
			if a.cellEditCommit != nil {
				a.cellEditCommit(a.cellEditVal)
			}
			if a.selectedReq != nil {
				_ = a.store.SaveRequest(context.Background(), a.selectedReq)
				a.setStatus("Saved")
			}
			return nil
		case "cancel":
			a.showCellEdit = false
			return nil
		case "newline":
			newRunes := make([]rune, n+1)
			copy(newRunes, runes[:a.cellEditCursor])
			newRunes[a.cellEditCursor] = '\n'
			copy(newRunes[a.cellEditCursor+1:], runes[a.cellEditCursor:])
			a.cellEditVal = string(newRunes)
			a.cellEditCursor++
			return nil
		case "open_external_editor":
			return a.openExternalEditor(a.cellEditVal)
		}
	}

	switch key {
	case "backspace":
		if a.cellEditCursor > 0 {
			a.cellEditVal = string(runes[:a.cellEditCursor-1]) + string(runes[a.cellEditCursor:])
			a.cellEditCursor--
		}
	case "delete":
		if a.cellEditCursor < n {
			a.cellEditVal = string(runes[:a.cellEditCursor]) + string(runes[a.cellEditCursor+1:])
		}
	case "left":
		if a.cellEditCursor > 0 {
			a.cellEditCursor--
		}
	case "right":
		if a.cellEditCursor < n {
			a.cellEditCursor++
		}
	case "home", "ctrl+a":
		a.cellEditCursor = 0
	case "end", "ctrl+e":
		a.cellEditCursor = n
	default:
		if isPasteKey(key) {
			text, err := clipboard.ReadAll()
			if err == nil {
				a.cellEditVal, a.cellEditCursor = insertAtCursor(a.cellEditVal, a.cellEditCursor, text)
			}
		} else {
			r := []rune(key)
			if len(r) == 1 && r[0] >= 32 && r[0] != 127 {
				newRunes := make([]rune, n+1)
				copy(newRunes, runes[:a.cellEditCursor])
				newRunes[a.cellEditCursor] = r[0]
				copy(newRunes[a.cellEditCursor+1:], runes[a.cellEditCursor:])
				a.cellEditVal = string(newRunes)
				a.cellEditCursor++
			}
		}
	}
	return nil
}

func (a *App) handleCurlImportModal(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()
	runes := []rune(a.curlImportVal)
	n := len(runes)

	if binding, ok := a.keybindMgr.Resolve(key, "curl_import_modal"); ok && binding.Panel == "curl_import_modal" {
		switch binding.Action {
		case "import":
			req, err := a.parseCurl(a.curlImportVal)
			if err != nil {
				a.showCurlImport = false
				a.showNotify("Import failed: "+err.Error(), true)
				return nil
			}
			_ = a.store.SaveRequest(context.Background(), req)
			a.requests = append(a.requests, req)
			a.requestList.setRequests(a.requests)
			a.collectionList.setRequests(a.requests)
			a.selectRequest(req)
			a.showCurlImport = false
			a.showNotify("Imported: "+req.Name, false)
			return nil
		case "paste":
			text, err := clipboard.ReadAll()
			if err == nil {
				newRunes := make([]rune, n+len([]rune(text)))
				copy(newRunes, runes[:a.curlImportCursor])
				copy(newRunes[a.curlImportCursor:], []rune(text))
				copy(newRunes[a.curlImportCursor+len([]rune(text)):], runes[a.curlImportCursor:])
				a.curlImportVal = string(newRunes)
				a.curlImportCursor += len([]rune(text))
			}
			return nil
		case "cancel":
			a.showCurlImport = false
			return nil
		}
	}

	switch key {
	case "backspace":
		if a.curlImportCursor > 0 {
			a.curlImportVal = string(runes[:a.curlImportCursor-1]) + string(runes[a.curlImportCursor:])
			a.curlImportCursor--
		}
	case "left":
		if a.curlImportCursor > 0 {
			a.curlImportCursor--
		}
	case "right":
		if a.curlImportCursor < n {
			a.curlImportCursor++
		}
	case "home", "ctrl+a":
		a.curlImportCursor = 0
	case "end", "ctrl+e":
		a.curlImportCursor = n
	default:
		if isPasteKey(key) {
			text, err := clipboard.ReadAll()
			if err == nil {
				a.curlImportVal, a.curlImportCursor = insertAtCursor(a.curlImportVal, a.curlImportCursor, text)
			}
		} else {
			r := []rune(key)
			if len(r) == 1 && r[0] >= 32 && r[0] != 127 {
				newRunes := make([]rune, n+1)
				copy(newRunes, runes[:a.curlImportCursor])
				newRunes[a.curlImportCursor] = r[0]
				copy(newRunes[a.curlImportCursor+1:], runes[a.curlImportCursor:])
				a.curlImportVal = string(newRunes)
				a.curlImportCursor++
			}
		}
	}
	return nil
}

func (a *App) openExternalEditor(initialContent string) tea.Cmd {
	return a.openExternalEditorWithSource(initialContent, "cell_edit")
}

func (a *App) openExternalEditorWithSource(initialContent, source string) tea.Cmd {
	editorCmd := os.ExpandEnv(a.cfg.ExternalEditor)
	if editorCmd == "" {
		editorCmd = os.Getenv("EDITOR")
	}
	if editorCmd == "" {
		editorCmd = "vi"
	}

	tmp, err := os.CreateTemp("", "http-cli-*.txt")
	if err != nil {
		a.setStatus("Could not create temp file: " + err.Error())
		return nil
	}
	tmpPath := tmp.Name()
	_, _ = tmp.WriteString(initialContent)
	_ = tmp.Close()

	parts := strings.Fields(editorCmd)
	if len(parts) == 0 {
		parts = []string{"vi"}
	}
	args := append(parts[1:], tmpPath)
	cmd := exec.Command(parts[0], args...)

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		defer os.Remove(tmpPath)
		if err != nil {
			return StatusMsg{Text: "Editor error: " + err.Error()}
		}
		data, readErr := os.ReadFile(tmpPath)
		if readErr != nil {
			return StatusMsg{Text: "Could not read temp file: " + readErr.Error()}
		}
		return externalEditorDoneMsg{content: string(data), source: source}
	})
}

func (a *App) openResponseInEditor() tea.Cmd {
	editorCmd := os.ExpandEnv(a.cfg.ExternalEditor)
	if editorCmd == "" {
		editorCmd = os.Getenv("EDITOR")
	}
	if editorCmd == "" {
		editorCmd = "vi"
	}

	body := a.response.FormattedBody()

	tmp, err := os.CreateTemp("", "http-cli-response-*.json")
	if err != nil {
		a.setStatus("Could not create temp file: " + err.Error())
		return nil
	}
	tmpPath := tmp.Name()
	_, _ = tmp.WriteString(body)
	_ = tmp.Close()

	parts := strings.Fields(editorCmd)
	if len(parts) == 0 {
		parts = []string{"vi"}
	}
	args := append(parts[1:], tmpPath)
	cmd := exec.Command(parts[0], args...)

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		defer os.Remove(tmpPath)
		if err != nil {
			return StatusMsg{Text: "Editor error: " + err.Error()}
		}
		return StatusMsg{Text: ""}
	})
}
