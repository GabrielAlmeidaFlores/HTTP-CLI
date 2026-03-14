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

	case "execute_collection_request":
		return a.executeCollectionRequest()

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

	case "focus_panel_4":
		a.focused = PanelCollectionList

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
			a.collectionList.setRequests(a.requests)
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
			a.collectionList.setRequests(a.requests)
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
				a.collectionList.setRequests(a.requests)
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
					a.collectionList.setRequests(a.requests)
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
		case PanelCollectionList:
			a.collectionList.moveDown()
		}

	case "up":
		switch a.focused {
		case PanelRequestList:
			a.requestList.moveUp()
			if req := a.requestList.selected(); req != nil {
				a.selectRequest(req)
			}
		case PanelCollectionList:
			a.collectionList.moveUp()
		}

	case "goto_top":
		switch a.focused {
		case PanelRequestList:
			a.requestList.selectedIdx = 0
			a.requestList.scrollOffset = 0
			if req := a.requestList.selected(); req != nil {
				a.selectRequest(req)
			}
		case PanelCollectionList:
			a.collectionList.selectedIdx = 0
			a.collectionList.scrollOffset = 0
		}

	case "goto_bottom":
		switch a.focused {
		case PanelRequestList:
			if n := len(a.requestList.filtered); n > 0 {
				a.requestList.selectedIdx = n - 1
				a.requestList.ensureVisible()
				if req := a.requestList.selected(); req != nil {
					a.selectRequest(req)
				}
			}
		case PanelCollectionList:
			if n := len(a.collectionList.visible); n > 0 {
				a.collectionList.selectedIdx = n - 1
				a.collectionList.ensureVisible()
			}
		}

	case "cancel", "help", "left", "right":
		// global/navigation actions handled contextually in modal/search handlers; no-op at panel level

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
		var reqToExport *models.Request
		if a.focused == PanelCollectionList {
			if node := a.collectionList.selectedNode(); node != nil && node.kind == colNodeRequest {
				reqToExport = a.collectionList.requestIndex[node.requestID]
			}
		} else {
			reqToExport = a.selectedReq
		}
		if reqToExport != nil {
			a.curlExportVal = exporter.ToCurl(reqToExport)
			a.showCurlExport = true
		} else {
			a.setStatus("Select a request first")
		}

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

	case "new_collection":
		a.promptInput("Collection name:", "New Collection", func(name string) {
			col := &models.Collection{
				Name:       name,
				Variables:  make(map[string]string),
				RequestIDs: make([]string, 0),
				Folders:    make([]models.Folder, 0),
			}
			_ = a.store.SaveCollection(context.Background(), col)
			a.collections = append(a.collections, col)
			a.collectionList.setCollections(a.collections)
			a.setStatus("Created collection: " + col.Name)
		})

	case "import_collection":
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
			a.collectionList.setRequests(a.requests)
			if col != nil {
				for _, req := range reqs {
					col.RequestIDs = append(col.RequestIDs, req.ID)
				}
				_ = a.store.SaveCollection(context.Background(), col)
				a.collections = append(a.collections, col)
				a.collectionList.setCollections(a.collections)
				a.showNotify(fmt.Sprintf("Imported '%s' (%d requests)", col.Name, len(reqs)), false)
			} else {
				a.showNotify(fmt.Sprintf("Imported %d requests", len(reqs)), false)
			}
		})

	case "delete_collection":
		if col := a.collectionList.selectedCollection(); col != nil {
			colToDelete := col
			a.promptConfirm("Delete collection '"+colToDelete.Name+"'?", func() {
				_ = a.store.DeleteCollection(context.Background(), colToDelete.ID)
				for i, c := range a.collections {
					if c.ID == colToDelete.ID {
						a.collections = append(a.collections[:i], a.collections[i+1:]...)
						break
					}
				}
				a.collectionList.setCollections(a.collections)
				a.setStatus("Deleted: " + colToDelete.Name)
			})
		}

	case "rename_collection":
		if col := a.collectionList.selectedCollection(); col != nil {
			colToRename := col
			a.promptInput("Rename collection:", colToRename.Name, func(name string) {
				if name != "" {
					colToRename.Name = name
					_ = a.store.SaveCollection(context.Background(), colToRename)
					a.collectionList.setCollections(a.collections)
					a.setStatus("Renamed to '" + name + "'")
				}
			})
		}

	case "collection_select":
		node := a.collectionList.selectedNode()
		if node == nil {
			break
		}
		switch node.kind {
		case colNodeCollection, colNodeFolder:
			a.collectionList.toggle()
		case colNodeRequest:
			if req, ok := a.collectionList.requestIndex[node.requestID]; ok {
				a.selectRequest(req)
				a.focused = PanelEditor
			}
		}

	case "add_request_to_collection":
		if a.selectedReq != nil {
			if col := a.collectionList.selectedCollection(); col != nil {
				req := a.selectedReq
				alreadyIn := false
				for _, rid := range col.RequestIDs {
					if rid == req.ID {
						alreadyIn = true
						break
					}
				}
				if !alreadyIn {
					req.CollectionID = col.ID
					col.RequestIDs = append(col.RequestIDs, req.ID)
					_ = a.store.SaveRequest(context.Background(), req)
					_ = a.store.SaveCollection(context.Background(), col)
					a.collectionList.setRequests(a.requests)
					a.setStatus("Added '" + req.Name + "' to '" + col.Name + "'")
				} else {
					a.setStatus("Request already in collection")
				}
			} else {
				a.setStatus("Select a collection first")
			}
		}
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

	var envVars map[string]string
	if req.CollectionID != "" {
		for _, col := range a.collections {
			if col.ID == req.CollectionID && len(col.Variables) > 0 {
				envVars = make(map[string]string, len(col.Variables))
				for k, v := range col.Variables {
					envVars[k] = v
				}
				break
			}
		}
	}

	return func() tea.Msg {
		resp, err := httpClient.Execute(context.Background(), req, envVars)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return ResponseReceivedMsg{Response: resp}
	}
}

func (a *App) executeCollectionRequest() tea.Cmd {
	node := a.collectionList.selectedNode()
	if node == nil || node.kind != colNodeRequest {
		a.setStatus("Select a request first")
		return nil
	}
	req, ok := a.collectionList.requestIndex[node.requestID]
	if !ok || req == nil {
		a.setStatus("Request not found")
		return nil
	}
	if req.URL == "" {
		a.setStatus("URL is empty")
		return nil
	}

	a.executing = true
	a.setStatus("Executing...")

	var envVars map[string]string
	if node.collection != nil && len(node.collection.Variables) > 0 {
		envVars = make(map[string]string, len(node.collection.Variables))
		for k, v := range node.collection.Variables {
			envVars[k] = v
		}
	}

	httpClient := a.httpClient
	return func() tea.Msg {
		resp, err := httpClient.Execute(context.Background(), req, envVars)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return ResponseReceivedMsg{Response: resp}
	}
}
