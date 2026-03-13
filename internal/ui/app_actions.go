package ui

import (
	"context"
	"fmt"
	"os"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/http-cli/internal/exporter"
	"github.com/user/http-cli/internal/models"
	"github.com/user/http-cli/internal/parser"
)

func (a *App) executeAction(action, _ string) tea.Cmd {
	switch action {
	case "exit":
		return tea.Quit

	case "execute_request":
		return a.executeRequest()

	case "search":
		a.isSearching = true
		a.searchQuery = ""
		a.focused = PanelRequestList

	case "next_panel":
		a.nextPanel()

	case "prev_panel":
		a.prevPanel()

	case "next_tab":
		if a.focused == PanelEditor {
			a.editor.nextTab()
		}

	case "prev_tab":
		if a.focused == PanelEditor {
			a.editor.prevTab()
		}

	case "focus_panel_1":
		a.focused = PanelRequestList

	case "focus_panel_2":
		a.focused = PanelEditor

	case "focus_panel_3":
		a.focused = PanelResponse

	case "tab_1":
		if a.focused == PanelEditor {
			a.editor.JumpToTab(1)
		}

	case "tab_2":
		if a.focused == PanelEditor {
			a.editor.JumpToTab(2)
		}

	case "tab_3":
		if a.focused == PanelEditor {
			a.editor.JumpToTab(3)
		}

	case "tab_4":
		if a.focused == PanelEditor {
			a.editor.JumpToTab(4)
		}

	case "tab_5":
		if a.focused == PanelEditor {
			a.editor.JumpToTab(5)
		}

	case "new_request":
		a.promptInput("Request name:", "New Request", func(name string) {
			req := &models.Request{
				Name:        name,
				Method:      models.MethodGET,
				URL:         "",
				Headers:     []models.Header{},
				QueryParams: []models.QueryParam{},
				Body:        models.Body{Type: models.BodyNone},
				Auth:        models.Auth{Type: models.AuthNone},
			}
			_ = a.store.SaveRequest(context.Background(), req)
			a.requests = append(a.requests, req)
			a.requestList.setRequests(a.requests)
			a.selectRequest(req)
		})

	case "import_curl":
		a.curlImportVal = ""
		a.curlImportCursor = 0
		a.showCurlImport = true

	case "duplicate_request":
		if a.selectedReq != nil {
			src := a.selectedReq
			dup := &models.Request{
				Name:        src.Name + " (copy)",
				Method:      src.Method,
				URL:         src.URL,
				Headers:     append([]models.Header{}, src.Headers...),
				QueryParams: append([]models.QueryParam{}, src.QueryParams...),
				Body:        src.Body,
				Auth:        src.Auth,
			}
			_ = a.store.SaveRequest(context.Background(), dup)
			a.requests = append(a.requests, dup)
			a.requestList.setRequests(a.requests)
			a.selectRequest(dup)
			a.setStatus("Duplicated: " + dup.Name)
		}

	case "delete_request":
		if a.selectedReq != nil {
			req := a.selectedReq
			a.promptConfirm("Delete '"+req.Name+"'?", func() {
				_ = a.store.DeleteRequest(context.Background(), req.ID)
				for i, r := range a.requests {
					if r.ID == req.ID {
						a.requests = append(a.requests[:i], a.requests[i+1:]...)
						break
					}
				}
				a.requestList.setRequests(a.requests)
				if a.selectedReq != nil && a.selectedReq.ID == req.ID {
					if len(a.requests) > 0 {
						a.selectRequest(a.requests[0])
					} else {
						a.selectedReq = nil
					}
				}
				a.setStatus("Deleted: " + req.Name)
			})
		}

	case "rename_request":
		if a.selectedReq != nil {
			req := a.selectedReq
			a.promptInput("Rename request:", req.Name, func(name string) {
				if name != "" {
					req.Name = name
					_ = a.store.SaveRequest(context.Background(), req)
					a.requestList.setRequests(a.requests)
					a.setStatus("Renamed to '" + name + "'")
				}
			})
		}

	case "select_request":
		if req := a.requestList.selected(); req != nil {
			a.selectRequest(req)
			a.focused = PanelEditor
		}

	case "down":
		switch a.focused {
		case PanelRequestList:
			a.requestList.moveDown()
			if req := a.requestList.selected(); req != nil {
				a.selectRequest(req)
			}
		}

	case "up":
		switch a.focused {
		case PanelRequestList:
			a.requestList.moveUp()
			if req := a.requestList.selected(); req != nil {
				a.selectRequest(req)
			}
		}

	case "copy_body":
		body := a.response.FormattedBody()
		if body == "" {
			a.setStatus("Nothing to copy")
		} else if err := clipboard.WriteAll(body); err != nil {
			a.setStatus("Copy failed: " + err.Error())
		} else {
			a.setStatus("Copied to clipboard ✓")
		}

	case "open_viewer":
		if a.response.GetResponse() != nil {
			return a.openResponseInEditor()
		}

	case "save":
		if a.selectedReq != nil {
			_ = a.store.SaveRequest(context.Background(), a.selectedReq)
			a.setStatus("Saved")
		}

	case "export_curl":
		if a.selectedReq != nil {
			a.curlExportVal = exporter.ToCurl(a.selectedReq)
			a.showCurlExport = true
		}

	case "import_postman":
		a.promptInput("Postman collection path:", "", func(path string) {
			reqs, col, err := parser.ParsePostmanCollection(path)
			if err != nil {
				a.showNotify("Import failed: "+err.Error(), true)
				return
			}
			for _, req := range reqs {
				_ = a.store.SaveRequest(context.Background(), req)
				a.requests = append(a.requests, req)
			}
			a.requestList.setRequests(a.requests)
			name := path
			if col != nil {
				name = col.Name
			}
			a.showNotify(fmt.Sprintf("Imported %d requests from '%s'", len(reqs), name), false)
		})

	case "export_postman":
		a.promptInput("Export filename (.json):", "collection.json", func(path string) {
			data, err := exporter.ToPostmanCollection("http-cli", a.requests)
			if err != nil {
				a.showNotify("Export failed: "+err.Error(), true)
				return
			}
			if err := os.WriteFile(path, data, 0600); err != nil {
				a.showNotify("Write failed: "+err.Error(), true)
				return
			}
			a.showNotify(fmt.Sprintf("Exported %d requests to '%s'", len(a.requests), path), false)
		})
	}

	return nil
}

func (a *App) executeRequest() tea.Cmd {
	if a.selectedReq == nil {
		return nil
	}
	if a.selectedReq.URL == "" {
		a.setStatus("URL is empty")
		return nil
	}

	a.executing = true
	a.setStatus("Executing...")

	req := a.selectedReq
	httpClient := a.httpClient

	return func() tea.Msg {
		resp, err := httpClient.Execute(context.Background(), req, nil)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return ResponseReceivedMsg{Response: resp}
	}
}
