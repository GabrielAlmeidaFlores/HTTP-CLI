package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/user/http-cli/internal/config"
	"github.com/user/http-cli/internal/models"
	"github.com/user/http-cli/internal/storage"
	"github.com/user/http-cli/internal/transport"
	"github.com/user/http-cli/internal/ui/keybindings"
)

type AppMode string

const (
	ModeNormal AppMode = "normal"
	ModeSearch AppMode = "search"
)

type FocusedPanel string

const (
	PanelRequestList FocusedPanel = "request_list"
	PanelEditor      FocusedPanel = "editor"
	PanelResponse    FocusedPanel = "response"
)

type App struct {
	cfg        *config.Config
	keybindMgr *keybindings.Manager
	store      *storage.Store
	httpClient *transport.Client

	mode    AppMode
	focused FocusedPanel
	width   int
	height  int

	requests    []*models.Request
	selectedReq *models.Request
	currentResp *models.Response

	requestList RequestListModel
	editor      EditorModel
	response    ResponseModel

	statusMsg    string
	statusExpiry time.Time

	searchQuery string
	isSearching bool

	showConfirm   bool
	confirmMsg    string
	confirmAction func()

	showInput   bool
	inputTitle  string
	inputValue  string
	inputAction func(string)

	showCellEdit   bool
	cellEditTitle  string
	cellEditVal    string
	cellEditCursor int
	cellEditCommit func(string)

	executing bool
	err       error
}

type RequestsLoadedMsg struct{ Requests []*models.Request }
type ResponseReceivedMsg struct{ Response *models.Response }
type ErrorMsg struct{ Err error }
type StatusMsg struct{ Text string }

func NewApp(cfg *config.Config, store *storage.Store) *App {
	km := keybindings.NewManager(cfg)
	httpClient := transport.NewClient(
		cfg.RequestDefaults.TimeoutSeconds,
		cfg.RequestDefaults.FollowRedirects,
		cfg.RequestDefaults.VerifySSL,
	)

	app := &App{
		cfg:         cfg,
		keybindMgr:  km,
		store:       store,
		httpClient:  httpClient,
		mode:        ModeNormal,
		focused:     PanelRequestList,
		requests:    make([]*models.Request, 0),
	}

	app.requestList = newRequestListModel(km)
	app.editor = newEditorModel(km)
	app.response = newResponseModel(km)

	return app
}

func (a *App) Init() tea.Cmd {
	return a.loadRequests()
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

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.requestList.setSize(a.listWidth(), a.mainHeight())
		a.editor.setSize(a.mainPanelWidth(), a.mainHeight()/2)
		a.response.setSize(a.mainPanelWidth(), a.mainHeight()/2)

	case tea.KeyMsg:
		if a.showConfirm {
			cmds = append(cmds, a.handleConfirmInput(msg))
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
		cmds = append(cmds, a.handleKey(msg))

	case RequestsLoadedMsg:
		a.requests = msg.Requests
		a.requestList.setRequests(msg.Requests)
		if a.selectedReq == nil && len(msg.Requests) > 0 {
			a.selectRequest(msg.Requests[0])
		}

	case ResponseReceivedMsg:
		a.executing = false
		a.currentResp = msg.Response
		a.response.setResponse(msg.Response)
		_ = a.store.AddHistory(context.Background(), msg.Response)
		a.setStatus(fmt.Sprintf("%d %s  %dms  %s",
			msg.Response.Status, msg.Response.StatusText,
			msg.Response.Duration.Milliseconds(),
			formatSize(msg.Response.Size)))

	case ErrorMsg:
		a.executing = false
		a.err = msg.Err
		a.setStatus("Error: " + msg.Err.Error())

	case StatusMsg:
		a.setStatus(msg.Text)
	}

	return a, tea.Batch(cmds...)
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

	binding, found := a.keybindMgr.Resolve(key, string(a.focused))
	if !found {
		binding, found = a.keybindMgr.Resolve(key, "global")
	}

	if found {
		return a.executeAction(binding.Action, binding.Panel)
	}

	return a.routeKeyToPanel(msg)
}

func (a *App) handleEditorKey(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	switch key {
	case "tab":
		a.nextPanel()
		return nil
	case "shift+tab":
		a.prevPanel()
		return nil
	case "ctrl+e":
		return a.executeRequest()
	case "ctrl+s":
		if a.selectedReq != nil {
			_ = a.store.SaveRequest(context.Background(), a.selectedReq)
			a.setStatus("Saved")
		}
		return nil
	}

	if a.selectedReq == nil {
		return nil
	}

	if !a.editor.IsSubEditing() {
		switch key {
		case "q", "ctrl+c":
			return tea.Quit
		case "/":
			a.isSearching = true
			a.searchQuery = ""
			a.focused = PanelRequestList
			return nil
		case "1":
			a.editor.JumpToTab(1)
			return nil
		case "2":
			a.editor.JumpToTab(2)
			return nil
		case "3":
			a.editor.JumpToTab(3)
			return nil
		case "4":
			a.editor.JumpToTab(4)
			return nil
		case "5":
			a.editor.JumpToTab(5)
			return nil
		case "e":
			if a.editor.CurrentCellIsText() {
				a.openCellEdit()
				return nil
			}
		}
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

func (a *App) handleCellEditModal(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()
	runes := []rune(a.cellEditVal)
	n := len(runes)

	switch key {
	case "ctrl+s", "enter":
		if a.cellEditCommit != nil {
			a.cellEditCommit(a.cellEditVal)
		}
		a.showCellEdit = false
		if a.selectedReq != nil {
			_ = a.store.SaveRequest(context.Background(), a.selectedReq)
			a.setStatus("Saved")
		}
	case "ctrl+d":
		if a.cellEditCommit != nil {
			a.cellEditCommit(a.cellEditVal)
		}
		if a.selectedReq != nil {
			_ = a.store.SaveRequest(context.Background(), a.selectedReq)
			a.setStatus("Saved")
		}
	case "esc":
		a.showCellEdit = false
	case "ctrl+j":
		newRunes := make([]rune, n+1)
		copy(newRunes, runes[:a.cellEditCursor])
		newRunes[a.cellEditCursor] = '\n'
		copy(newRunes[a.cellEditCursor+1:], runes[a.cellEditCursor:])
		a.cellEditVal = string(newRunes)
		a.cellEditCursor++
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
	return nil
}

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
		switch a.focused {
		case PanelEditor:
			a.editor.nextTab()
		case PanelResponse:
			a.response.nextTab()
		}

	case "prev_tab":
		switch a.focused {
		case PanelEditor:
			a.editor.prevTab()
		case PanelResponse:
			a.response.prevTab()
		}

	case "focus_panel_1":
		a.focused = PanelRequestList

	case "focus_panel_2":
		a.focused = PanelEditor

	case "focus_panel_3":
		a.focused = PanelResponse

	case "tab_1":
		switch a.focused {
		case PanelEditor:
			a.editor.JumpToTab(1)
		case PanelResponse:
			a.response.JumpToTab(1)
		}

	case "tab_2":
		switch a.focused {
		case PanelEditor:
			a.editor.JumpToTab(2)
		case PanelResponse:
			a.response.JumpToTab(2)
		}

	case "tab_3":
		switch a.focused {
		case PanelEditor:
			a.editor.JumpToTab(3)
		case PanelResponse:
			a.response.JumpToTab(3)
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

	case "delete_request":
		if a.selectedReq != nil {
			req := a.selectedReq
			a.promptConfirm("Delete request '"+req.Name+"'?", func() {
				_ = a.store.DeleteRequest(context.Background(), req.ID)
			})
			return a.loadRequests()
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

	case "insert_mode":
		// no-op: editor is always editable when focused

	case "down":
		switch a.focused {
		case PanelRequestList:
			a.requestList.moveDown()
		case PanelResponse:
			a.response.scrollDown(1)
		}

	case "up":
		switch a.focused {
		case PanelRequestList:
			a.requestList.moveUp()
		case PanelResponse:
			a.response.scrollUp(1)
		}

	case "page_down":
		a.response.scrollDown(10)

	case "page_up":
		a.response.scrollUp(10)

	case "copy_body":
		body := a.response.FormattedBody()
		if body == "" {
			a.setStatus("Nothing to copy")
		} else if err := clipboard.WriteAll(body); err != nil {
			a.setStatus("Copy failed: " + err.Error())
		} else {
			a.setStatus("Copied to clipboard ✓")
		}

	case "save":
		if a.selectedReq != nil {
			_ = a.store.SaveRequest(context.Background(), a.selectedReq)
			a.setStatus("Saved")
		}
	}

	return nil
}

func (a *App) routeKeyToPanel(msg tea.KeyMsg) tea.Cmd {
	switch a.focused {
	case PanelEditor:
		return a.editor.handleKey(msg, a.selectedReq)
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

func (a *App) selectRequest(req *models.Request) {
	a.selectedReq = req
	a.editor.setRequest(req)
	a.currentResp = nil
	a.response.setResponse(nil)
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

func (a *App) promptConfirm(msg string, action func()) {
	a.showConfirm = true
	a.confirmMsg = msg
	a.confirmAction = action
}

func (a *App) promptInput(title, defaultVal string, action func(string)) {
	a.showInput = true
	a.inputTitle = title
	a.inputValue = defaultVal
	a.inputAction = action
}

func (a *App) handleConfirmInput(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "y", "enter":
		a.showConfirm = false
		if a.confirmAction != nil {
			a.confirmAction()
			return a.loadRequests()
		}
	case "n", "esc":
		a.showConfirm = false
	}
	return nil
}

func (a *App) handleInputDialog(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		a.showInput = false
		if a.inputAction != nil {
			a.inputAction(a.inputValue)
		}
	case "esc":
		a.showInput = false
	case "backspace":
		if len(a.inputValue) > 0 {
			runes := []rune(a.inputValue)
			a.inputValue = string(runes[:len(runes)-1])
		}
	default:
		if len(msg.String()) == 1 {
			a.inputValue += msg.String()
		}
	}
	return nil
}

func (a *App) setStatus(text string) {
	a.statusMsg = text
	a.statusExpiry = time.Now().Add(5 * time.Second)
}

func (a *App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	if a.showConfirm {
		return a.renderModal(a.confirmMsg + "\n\n[y] Confirm  [n/esc] Cancel")
	}

	if a.showInput {
		cursor := "_"
		return a.renderModal(a.inputTitle + "\n\n" + a.inputValue + cursor + "\n\n[enter] Confirm  [esc] Cancel")
	}

	if a.showCellEdit {
		return a.renderCellEditModal()
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

func (a *App) renderTopBar() string {
	method := ""
	url := ""
	if a.selectedReq != nil {
		method = string(a.selectedReq.Method)
		url = a.selectedReq.URL
	}

	methodStyle := lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1).
		Foreground(lipgloss.Color(a.methodColor(method)))

	urlStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Width(a.width - 25)

	sendStyle := lipgloss.NewStyle().
		Bold(true).
		Padding(0, 2).
		Background(lipgloss.Color("#00d700")).
		Foreground(lipgloss.Color("#000000"))

	executing := ""
	if a.executing {
		executing = " ..."
	}
	sendLabel := "Send [ctrl+e]" + executing

	bar := lipgloss.JoinHorizontal(lipgloss.Top,
		methodStyle.Render(fmt.Sprintf("[%s]", method)),
		urlStyle.Render(url),
		sendStyle.Render(sendLabel),
	)

	return lipgloss.NewStyle().
		Width(a.width).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#626262")).
		Render(bar)
}

func (a *App) renderMainArea() string {
	leftPanel := a.requestList.view(a.focused == PanelRequestList, a.cfg.UI.Theme)
	rightTop := a.editor.view(a.focused == PanelEditor, a.cfg.UI.Theme)
	rightBottom := a.response.view(a.focused == PanelResponse, a.cfg.UI.Theme)

	right := lipgloss.JoinVertical(lipgloss.Left, rightTop, rightBottom)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, right)
}

func (a *App) renderStatusBar() string {
	if !a.cfg.UI.Layout.ShowStatusBar {
		return ""
	}

	mode := "NORMAL"
	if a.isSearching {
		mode = "SEARCH: " + a.searchQuery
	} else if a.focused == PanelEditor && a.editor.IsSubEditing() {
		mode = "EDITING"
	}

	modeStyle := lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1).
		Background(lipgloss.Color("#005fd7")).
		Foreground(lipgloss.Color("#ffffff"))

	if mode == "EDITING" {
		modeStyle = modeStyle.Background(lipgloss.Color("#d75f00"))
	}

	status := ""
	if time.Now().Before(a.statusExpiry) {
		status = a.statusMsg
	}

	statusStyle := lipgloss.NewStyle().Padding(0, 1)

	panel := string(a.focused)
	panelStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(lipgloss.Color("#87d7ff"))

	bar := lipgloss.JoinHorizontal(lipgloss.Top,
		modeStyle.Render(mode),
		panelStyle.Render("["+panel+"]"),
		statusStyle.Render(status),
	)

	return lipgloss.NewStyle().
		Width(a.width).
		Background(lipgloss.Color("#1c1c1c")).
		Render(bar)
}

func (a *App) renderHints() string {
	if !a.cfg.UI.Hints.Enabled {
		return ""
	}

	activeTab := ""
	switch a.focused {
	case PanelEditor:
		activeTab = a.editor.ActiveTab()
	case PanelResponse:
		activeTab = a.response.ActiveTab()
	}

	hints := a.keybindMgr.GetHints(string(a.focused), activeTab)

	keyStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(a.cfg.UI.Hints.KeyColor))

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(a.cfg.UI.Hints.DescriptionColor))

	sep := a.cfg.UI.Hints.Separator
	if sep == "" {
		sep = "  "
	}

	var parts []string
	for _, h := range hints {
		if len(h.Keys) == 0 {
			continue
		}
		keyStr := keyStyle.Render(h.Keys[0])
		if a.cfg.UI.Hints.ShowDescriptions {
			parts = append(parts, keyStr+descStyle.Render(" "+h.Description))
		} else {
			parts = append(parts, keyStr)
		}
	}

	hintsText := strings.Join(parts, sep)

	return lipgloss.NewStyle().
		Width(a.width).
		Height(a.cfg.UI.Hints.Height).
		Padding(0, 1).
		Background(lipgloss.Color("#121212")).
		Foreground(lipgloss.Color("#626262")).
		Render(hintsText)
}

func (a *App) renderCellEditModal() string {
	modalW := a.width * 3 / 4
	if modalW > 100 {
		modalW = 100
	}
	if modalW < 40 {
		modalW = 40
	}
	contentW := modalW - 6

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00d7ff"))

	runes := []rune(a.cellEditVal)
	cursor := a.cellEditCursor
	before := string(runes[:cursor])
	after := ""
	if cursor < len(runes) {
		after = string(runes[cursor:])
	}
	textContent := before + "█" + after

	textAreaStyle := lipgloss.NewStyle().
		Width(contentW).
		Height(8).
		Padding(1, 1).
		Background(lipgloss.Color("#1c1c2c")).
		Foreground(lipgloss.Color("#ffffff"))

	dimKey := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00d7ff"))
	dimDesc := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))

	hints := lipgloss.JoinHorizontal(lipgloss.Top,
		dimKey.Render("enter")+" "+dimDesc.Render("save & exit"),
		"   ",
		dimKey.Render("ctrl+d")+" "+dimDesc.Render("save"),
		"   ",
		dimKey.Render("esc")+" "+dimDesc.Render("cancel"),
		"   ",
		dimKey.Render("ctrl+j")+" "+dimDesc.Render("newline"),
	)

	body := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(a.cellEditTitle),
		"",
		textAreaStyle.Render(textContent),
		"",
		hints,
	)

	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00d7ff")).
		Padding(1, 2).
		Width(modalW).
		Render(body)

	return lipgloss.Place(a.width, a.height,
		lipgloss.Center, lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceBackground(lipgloss.Color("#000000")),
	)
}

func (a *App) renderModal(content string) string {
	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00d7ff")).
		Padding(1, 3).
		Render(content)

	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, modal)
}

func (a *App) methodColor(method string) string {
	colors := map[string]string{
		"GET":    a.cfg.UI.Theme.MethodGet,
		"POST":   a.cfg.UI.Theme.MethodPost,
		"PUT":    a.cfg.UI.Theme.MethodPut,
		"DELETE": a.cfg.UI.Theme.MethodDelete,
		"PATCH":  a.cfg.UI.Theme.MethodPatch,
	}
	if c, ok := colors[method]; ok && c != "" {
		return c
	}
	return "#ffffff"
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
	reserved := 4
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

func formatSize(bytes int64) string {
	switch {
	case bytes < 1024:
		return fmt.Sprintf("%dB", bytes)
	case bytes < 1024*1024:
		return fmt.Sprintf("%.1fKB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%.1fMB", float64(bytes)/1024/1024)
	}
}
