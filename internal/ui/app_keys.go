package ui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
)

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

	binding, found := a.keybindMgr.Resolve(key, string(a.focused))
	if !found {
		binding, found = a.keybindMgr.Resolve(key, "global")
	}

	if found {
		return a.executeAction(binding.Action, binding.Panel)
	}

	return a.routeKeyToPanel(msg)
}

func (a *App) handleResponseKey(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	if binding, ok := a.keybindMgr.Resolve(key, "response"); ok {
		switch binding.Action {
		case "next_tab":
			a.response.NextTab()
			return nil
		case "prev_tab":
			a.response.PrevTab()
			return nil
		case "scroll_down", "down":
			a.response.ScrollDown()
			return nil
		case "scroll_up", "up":
			a.response.ScrollUp()
			return nil
		default:
			return a.executeAction(binding.Action, binding.Panel)
		}
	}

	if binding, ok := a.keybindMgr.Resolve(key, "global"); ok {
		return a.executeAction(binding.Action, binding.Panel)
	}

	switch key {
	case "tab":
		a.response.NextTab()
	case "shift+tab":
		a.response.PrevTab()
	case "j", "down":
		a.response.ScrollDown()
	case "k", "up":
		a.response.ScrollUp()
	case "1":
		a.response.activeTab = responseTabBody
		a.response.scrollY = 0
	case "2":
		a.response.activeTab = responseTabHeaders
		a.response.scrollY = 0
	case "3":
		a.response.activeTab = responseTabInfo
		a.response.scrollY = 0
	}
	return nil
}

func (a *App) handleEditorKey(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	if key == "ctrl+o" && a.editor.IsSubEditing() {
		return a.openExternalEditorWithSource(a.editor.CurrentEditValue(), "editor")
	}

	if binding, ok := a.keybindMgr.Resolve(key, "editor"); ok {
		switch binding.Action {
		case "next_panel":
			a.nextPanel()
			return nil
		case "prev_panel":
			a.prevPanel()
			return nil
		case "execute", "execute_request":
			return a.executeRequest()
		case "save":
			if a.selectedReq != nil {
				_ = a.store.SaveRequest(context.Background(), a.selectedReq)
				a.setStatus("Saved")
			}
			return nil
		}

		if a.selectedReq != nil && !a.editor.IsSubEditing() {
			switch binding.Action {
			case "exit":
				return tea.Quit
			case "search":
				a.isSearching = true
				a.searchQuery = ""
				a.focused = PanelRequestList
				return nil
			case "tab_1", "tab_2", "tab_3", "tab_4", "tab_5":
				n := int(binding.Action[4] - '0')
				a.editor.JumpToTab(n)
				return nil
			case "insert_mode":
				if a.editor.CurrentCellIsText() {
					a.openCellEdit()
					return nil
				}
			}
		}
	}

	if a.selectedReq == nil {
		return nil
	}

	return a.editor.handleKey(msg, a.selectedReq)
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
	panels := []FocusedPanel{PanelRequestList, PanelEditor, PanelResponse}
	for i, p := range panels {
		if p == a.focused {
			a.focused = panels[(i+1)%len(panels)]
			return
		}
	}
}

func (a *App) prevPanel() {
	panels := []FocusedPanel{PanelRequestList, PanelEditor, PanelResponse}
	for i, p := range panels {
		if p == a.focused {
			a.focused = panels[(i-1+len(panels))%len(panels)]
			return
		}
	}
}
