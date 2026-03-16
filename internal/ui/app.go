package ui

import (
	"context"
	"fmt"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/user/http-cli/internal/config"
	"github.com/user/http-cli/internal/models"
	"github.com/user/http-cli/internal/ui/keybindings"
)

type FocusedPanel string

const (
	PanelRequestList    FocusedPanel = "request_list"
	PanelCollectionList FocusedPanel = "collection_list"
	PanelEditor         FocusedPanel = "editor"
	PanelResponse       FocusedPanel = "response"
)

type App struct {
	cfg        *config.Config
	keybindMgr *keybindings.Manager
	store      RequestStore
	httpClient HTTPExecutor
	parseCurl  func(string) (*models.Request, error)
	theme      config.ThemeConfig

	focused FocusedPanel
	width   int
	height  int

	requests    []*models.Request
	selectedReq *models.Request
	currentResp *models.Response

	collections []*models.Collection

	requestList    RequestListModel
	collectionList CollectionListModel
	editor         EditorModel
	response       ResponseModel

	statusMsg    string
	statusExpiry time.Time

	searchQuery string
	isSearching bool

	showConfirm   bool
	confirmMsg    string
	confirmAction func()

	showInput       bool
	inputTitle      string
	inputValue      string
	inputCursor     int
	inputViewOffset int
	inputAction     func(string)

	showCellEdit   bool
	cellEditTitle  string
	cellEditVal    string
	cellEditCursor int
	cellEditCommit func(string)

	showCurlImport   bool
	curlImportVal    string
	curlImportCursor int
	curlImportScroll int

	showCurlExport bool
	curlExportVal  string

	showNotification  bool
	notificationMsg   string
	notificationIsErr bool

	showVarsModal   bool
	varsCollection  *models.Collection
	varsTable       kvTable

	showFilePicker bool
	fp             filePicker

	executing bool

	lastResponse map[string]*models.Response
}

type RequestsLoadedMsg struct{ Requests []*models.Request }
type CollectionsLoadedMsg struct{ Collections []*models.Collection }
type ResponseReceivedMsg struct{ Response *models.Response }
type ErrorMsg struct{ Err error }
type StatusMsg struct{ Text string }
type externalEditorDoneMsg struct {
	content string
	source  string
}

func NewApp(cfg *config.Config, store RequestStore, httpClient HTTPExecutor, parseCurl func(string) (*models.Request, error)) *App {
	km := keybindings.NewManager(cfg)
	theme := cfg.UI.Theme

	app := &App{
		cfg:          cfg,
		keybindMgr:   km,
		store:        store,
		httpClient:   httpClient,
		parseCurl:    parseCurl,
		theme:        theme,
		focused:      PanelRequestList,
		requests:     make([]*models.Request, 0),
		collections:  make([]*models.Collection, 0),
		lastResponse: make(map[string]*models.Response),
	}

	app.requestList = newRequestListModel(km)
	app.collectionList = newCollectionListModel(km)
	app.editor = newEditorModel(km, theme)
	app.response = newResponseModel(km, theme)

	return app
}

func (a *App) Init() tea.Cmd {
	return tea.Batch(a.loadRequests(), a.loadCollections())
}

func (a *App) loadRequests() tea.Cmd {
	store := a.store
	return func() tea.Msg {
		reqs, err := store.ListRequests(context.Background())
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return RequestsLoadedMsg{Requests: reqs}
	}
}

func (a *App) loadCollections() tea.Cmd {
	store := a.store
	return func() tea.Msg {
		cols, err := store.ListCollections(context.Background())
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return CollectionsLoadedMsg{Collections: cols}
	}
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		editorH := a.mainHeight() / 2
		responseH := a.mainHeight() - editorH
		listH := a.mainHeight()
		reqListH := listH / 2
		colListH := listH - reqListH
		lw := a.listWidth() - 2
		mw := a.mainPanelWidth() - 2
		a.requestList.setSize(lw, reqListH)
		a.collectionList.setSize(lw, colListH)
		a.editor.setSize(mw, editorH)
		a.response.setSize(mw, responseH)

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}
		if a.showConfirm {
			cmds = append(cmds, a.handleConfirmInput(msg))
			break
		}
		if a.showFilePicker {
			cmds = append(cmds, a.handleFilePicker(msg))
			break
		}
		if a.showInput {
			cmds = append(cmds, a.handleInputDialog(msg))
			break
		}
		if a.showCellEdit {
			cmds = append(cmds, a.handleCellEditModal(msg))
			break
		}
		if a.showCurlImport {
			cmds = append(cmds, a.handleCurlImportModal(msg))
			break
		}
		if a.showVarsModal {
			cmds = append(cmds, a.handleVarsModal(msg))
			break
		}
		if a.showCurlExport {
			if binding, ok := a.keybindMgr.Resolve(msg.String(), "curl_export_modal"); ok {
				switch binding.Action {
				case "close":
					a.showCurlExport = false
				case "copy_clipboard":
					if err := clipboard.WriteAll(a.curlExportVal); err == nil {
						a.setStatus("cURL copied to clipboard ✓")
					}
					a.showCurlExport = false
				}
			}
			break
		}
		if a.showNotification {
			a.showNotification = false
			break
		}
		cmds = append(cmds, a.handleKey(msg))
		if a.focused == PanelEditor && a.selectedReq != nil {
			a.collectionList.rebuild()
		}

	case RequestsLoadedMsg:
		a.requests = msg.Requests
		a.requestList.setRequests(msg.Requests)
		a.collectionList.setRequests(msg.Requests)
		if a.selectedReq == nil && len(msg.Requests) > 0 {
			a.selectRequest(msg.Requests[0])
		}

	case CollectionsLoadedMsg:
		a.collections = msg.Collections
		a.collectionList.setCollections(msg.Collections)

	case ResponseReceivedMsg:
		a.executing = false
		a.currentResp = msg.Response
		a.response.setResponse(msg.Response)
		if msg.Response.RequestID != "" {
			a.lastResponse[msg.Response.RequestID] = msg.Response
		}
		_ = a.store.AddHistory(context.Background(), msg.Response)
		a.setStatus(fmt.Sprintf("%d %s  %dms  %s",
			msg.Response.Status, msg.Response.StatusText,
			msg.Response.Duration.Milliseconds(),
			formatSize(msg.Response.Size)))

	case ErrorMsg:
		a.executing = false
		a.setStatus("Error: " + msg.Err.Error())

	case StatusMsg:
		a.setStatus(msg.Text)

	case externalEditorDoneMsg:
		switch msg.source {
		case "cell_edit":
			if a.cellEditCommit != nil {
				a.cellEditCommit(msg.content)
				a.showCellEdit = false
				if a.selectedReq != nil {
					_ = a.store.SaveRequest(context.Background(), a.selectedReq)
					a.setStatus("Saved")
				}
			}
		case "editor":
			a.editor.CommitExternalEdit(msg.content)
			if a.selectedReq != nil {
				_ = a.store.SaveRequest(context.Background(), a.selectedReq)
				a.setStatus("Saved")
			}
		}
	}

	return a, tea.Batch(cmds...)
}

func (a *App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	if a.showConfirm {
		return a.renderConfirmModal()
	}

	if a.showFilePicker {
		return a.renderFilePickerModal()
	}

	if a.showInput {
		return a.renderInputModal()
	}

	if a.showCellEdit {
		return a.renderCellEditModal()
	}

	if a.showCurlImport {
		return a.renderCurlImportModal()
	}

	if a.showVarsModal {
		return a.renderVarsModal()
	}

	if a.showCurlExport {
		return a.renderCurlExportModal()
	}

	if a.showNotification {
		return a.renderNotificationModal()
	}

	topBar := a.renderTopBar()
	mainArea := a.renderMainArea()
	statusBar := a.renderStatusBar()
	hints := a.renderHints()

	return lipgloss.JoinVertical(lipgloss.Left,
		topBar,
		mainArea,
		statusBar,
		hints,
	)
}

func (a *App) setStatus(text string) {
	a.statusMsg = text
	a.statusExpiry = time.Now().Add(5 * time.Second)
}

func (a *App) selectRequest(req *models.Request) {
	a.selectedReq = req
	a.editor.setRequest(req)
	if last, ok := a.lastResponse[req.ID]; ok {
		a.currentResp = last
		a.response.setResponse(last)
	} else {
		a.currentResp = nil
		a.response.setResponse(nil)
	}
}

func (a *App) selectCollectionNode() {
	node := a.collectionList.selectedNode()
	if node == nil || node.kind != colNodeRequest {
		return
	}
	if req, ok := a.collectionList.requestIndex[node.requestID]; ok {
		a.selectRequest(req)
	}
}

func (a *App) listWidth() int {
	ratio := a.cfg.UI.Layout.LeftPanelWidthRatio
	if ratio <= 0 || ratio >= 1 {
		ratio = 0.25
	}
	return int(float64(a.width) * ratio)
}

func (a *App) mainPanelWidth() int {
	return a.width - a.listWidth()
}

func (a *App) mainHeight() int {
	reserved := 6
	if a.cfg.UI.Layout.ShowStatusBar {
		reserved++
	}
	if a.cfg.UI.Hints.Enabled {
		reserved += a.cfg.UI.Hints.Height
	}
	h := a.height - reserved
	if h < 5 {
		return 5
	}
	return h
}
