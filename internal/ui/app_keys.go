package ui

import (
	"context"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (a *App) fpListHeight() int {
	h := a.height/2 - 7
	if h < 3 {
		h = 3
	}
	if h > 20 {
		h = 20
	}
	return h
}

func (a *App) fpUpdateScroll() {
	listH := a.fpListHeight()
	fp := &a.fp
	if fp.cursor < fp.scrollOff {
		fp.scrollOff = fp.cursor
	}
	if fp.cursor >= fp.scrollOff+listH {
		fp.scrollOff = fp.cursor - listH + 1
	}
}

func (a *App) handleFilePicker(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()
	fp := &a.fp
	n := len(fp.filtered)

	if binding, ok := a.keybindMgr.Resolve(key, "file_picker"); ok {
		switch binding.Action {
		case "down":
			if fp.cursor < n-1 {
				fp.cursor++
				a.fpUpdateScroll()
			}
		case "up":
			if fp.cursor > 0 {
				fp.cursor--
				a.fpUpdateScroll()
			}
		case "select":
			if n == 0 {
				return nil
			}
			isDir := fp.enterSelected()
			if !isDir {
				path := fp.selectedPath()
				a.showFilePicker = false
				if fp.onSelect != nil {
					fp.onSelect(path)
				}
			} else {
				a.fpUpdateScroll()
			}
		case "go_up":
			if fp.search != "" {
				runes := []rune(fp.search)
				fp.applySearch(string(runes[:len(runes)-1]))
			} else {
				fp.goUp()
			}
			a.fpUpdateScroll()
		case "home_dir":
			if home, err := os.UserHomeDir(); err == nil {
				fp.navigate(home)
			}
			a.fpUpdateScroll()
		case "cancel":
			if fp.search != "" {
				fp.applySearch("")
				a.fpUpdateScroll()
			} else {
				a.showFilePicker = false
			}
		}
		return nil
	}

	if isPrintable(key) {
		fp.applySearch(fp.search + key)
		a.fpUpdateScroll()
	}
	return nil
}

func (a *App) handleKey(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	if a.isSearching {
		switch key {
		case "esc":
			a.isSearching = false
			a.searchQuery = ""
			a.requestList.setFilter("")
		case "enter":
			a.isSearching = false
		case "backspace":
			if len(a.searchQuery) > 0 {
				a.searchQuery = a.searchQuery[:len(a.searchQuery)-1]
				a.requestList.setFilter(a.searchQuery)
			}
		default:
			if len(key) == 1 {
				a.searchQuery += key
				a.requestList.setFilter(a.searchQuery)
			}
		}
		return nil
	}

	if a.focused == PanelEditor {
		return a.handleEditorKey(msg)
	}

	if a.focused == PanelResponse {
		return a.handleResponseKey(msg)
	}

	if a.focused == PanelCollectionList {
		return a.handleCollectionListKey(msg)
	}

	binding, found := a.keybindMgr.Resolve(key, string(a.focused))
	if !found {
		binding, found = a.keybindMgr.Resolve(key, "global")
	}

	if found {
		if binding.Action == "cancel" && a.focused == PanelRequestList {
			a.focused = PanelEditor
			return nil
		}
		return a.executeAction(binding.Action, binding.Panel)
	}

	return a.routeKeyToPanel(msg)
}

func (a *App) handleResponseKey(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	binding, ok := a.keybindMgr.Resolve(key, "response")
	if !ok {
		binding, ok = a.keybindMgr.Resolve(key, "global")
		if ok {
			return a.executeAction(binding.Action, binding.Panel)
		}
		return nil
	}

	switch binding.Action {
	case "scroll_down":
		a.response.ScrollDown()
	case "scroll_up":
		a.response.ScrollUp()
	case "next_tab":
		a.response.NextTab()
	case "prev_tab":
		a.response.PrevTab()
	case "scroll_top":
		a.response.scrollY = 0
	case "scroll_bottom":
		a.response.scrollY = a.response.totalContentLines() - a.response.contentHeight()
		if a.response.scrollY < 0 {
			a.response.scrollY = 0
		}
	case "half_page_down":
		a.response.scrollY += a.response.contentHeight() / 2
	case "half_page_up":
		a.response.scrollY -= a.response.contentHeight() / 2
		if a.response.scrollY < 0 {
			a.response.scrollY = 0
		}
	case "tab_1":
		a.response.activeTab = responseTabBody
		a.response.scrollY = 0
	case "tab_2":
		a.response.activeTab = responseTabHeaders
		a.response.scrollY = 0
	case "tab_3":
		a.response.activeTab = responseTabInfo
		a.response.scrollY = 0
	case "execute", "execute_request":
		return a.executeRequest()
	case "next_panel":
		a.nextPanel()
	case "prev_panel":
		a.prevPanel()
	case "cancel":
		a.focused = PanelEditor
	default:
		return a.executeAction(binding.Action, binding.Panel)
	}

	return nil
}

func (a *App) handleEditorKey(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	binding, ok := a.keybindMgr.Resolve(key, "editor")
	if !ok {
		if a.selectedReq != nil {
			return a.editor.handleKey(msg, a.selectedReq)
		}
		return nil
	}

	switch binding.Action {
	case "open_external_editor":
		if a.editor.IsSubEditing() {
			return a.openExternalEditorWithSource(a.editor.CurrentEditValue(), "editor")
		}
	case "next_panel":
		a.nextPanel()
	case "prev_panel":
		a.prevPanel()
	case "execute", "execute_request":
		return a.executeRequest()
	case "save":
		if a.selectedReq != nil {
			if err := a.store.SaveRequest(context.Background(), a.selectedReq); err != nil {
				a.setStatus("Save failed: " + err.Error())
			} else {
				a.setStatus("Saved")
			}
		}
	case "exit":
		if !a.editor.IsSubEditing() {
			return tea.Quit
		}
	case "search":
		if a.selectedReq != nil && !a.editor.IsSubEditing() {
			a.isSearching = true
			a.searchQuery = ""
			a.focused = PanelRequestList
		}
	case "tab_1", "tab_2", "tab_3", "tab_4", "tab_5":
		if a.selectedReq != nil && !a.editor.IsSubEditing() {
			n := int(binding.Action[4] - '0')
			a.editor.JumpToTab(n)
		}
	case "insert_mode":
		if a.selectedReq != nil && !a.editor.IsSubEditing() && a.editor.CurrentCellIsText() {
			if a.editor.CurrentCellIsFilePath() {
				a.openFilePicker(func(path string) {
					a.editor.CommitCellValue(path)
					_ = a.store.SaveRequest(context.Background(), a.selectedReq)
					a.setStatus("Saved")
				})
			} else {
				a.openCellEdit()
			}
		}
	case "normal_mode":
		if a.editor.IsSubEditing() {
			a.editor.CancelSubEdit()
		} else {
			a.focused = PanelRequestList
		}
	default:
		if (binding.Panel == "navigation" || binding.Panel == "global") &&
			strings.HasPrefix(binding.Action, "focus_panel_") {
			return a.executeAction(binding.Action, binding.Panel)
		}
		if a.selectedReq != nil {
			return a.editor.handleKey(msg, a.selectedReq)
		}
	}

	return nil
}

func (a *App) handleCollectionListKey(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()
	binding, found := a.keybindMgr.Resolve(key, string(PanelCollectionList))
	if !found {
		binding, found = a.keybindMgr.Resolve(key, "global")
	}
	if found {
		if binding.Action == "cancel" {
			a.focused = PanelRequestList
			return nil
		}
		return a.executeAction(binding.Action, binding.Panel)
	}
	return nil
}

func (a *App) openCellEdit() {
	a.cellEditTitle = a.editor.CurrentCellTitle()
	a.cellEditVal = a.editor.CurrentCellValue()
	a.cellEditCursor = len([]rune(a.cellEditVal))
	a.cellEditCommit = func(val string) {
		a.editor.CommitCellValue(val)
	}
	a.showCellEdit = true
}

func (a *App) routeKeyToPanel(msg tea.KeyMsg) tea.Cmd {
	switch a.focused {
	case PanelEditor:
		return a.editor.handleKey(msg, a.selectedReq)
	}
	return nil
}

func (a *App) nextPanel() {
	panels := []FocusedPanel{PanelRequestList, PanelCollectionList, PanelEditor, PanelResponse}
	for i, p := range panels {
		if p == a.focused {
			a.focused = panels[(i+1)%len(panels)]
			return
		}
	}
}

func (a *App) prevPanel() {
	panels := []FocusedPanel{PanelRequestList, PanelCollectionList, PanelEditor, PanelResponse}
	for i, p := range panels {
		if p == a.focused {
			a.focused = panels[(i-1+len(panels))%len(panels)]
			return
		}
	}
}
