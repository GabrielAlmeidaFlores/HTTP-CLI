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
PanelRequestList FocusedPanel = "request_list"
PanelEditor      FocusedPanel = "editor"
PanelResponse    FocusedPanel = "response"
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
inputCursor int
inputAction func(string)

showCellEdit   bool
cellEditTitle  string
cellEditVal    string
cellEditCursor int
cellEditCommit func(string)

showCurlImport    bool
curlImportVal     string
curlImportCursor  int
curlImportScroll  int

showCurlExport bool
curlExportVal  string

showNotification  bool
notificationMsg   string
notificationIsErr bool

executing bool
}

type RequestsLoadedMsg struct{ Requests []*models.Request }
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
cfg:        cfg,
keybindMgr: km,
store:      store,
httpClient: httpClient,
parseCurl:  parseCurl,
theme:      theme,
focused:    PanelRequestList,
requests:   make([]*models.Request, 0),
}

app.requestList = newRequestListModel(km)
app.editor = newEditorModel(km, theme)
app.response = newResponseModel(km, theme)

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
editorH := a.mainHeight() / 2
responseH := a.mainHeight() - editorH
a.requestList.setSize(a.listWidth(), a.mainHeight()+2)
a.editor.setSize(a.mainPanelWidth(), editorH)
a.response.setSize(a.mainPanelWidth(), responseH)

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
if a.showCurlImport {
cmds = append(cmds, a.handleCurlImportModal(msg))
break
}
if a.showCurlExport {
key := msg.String()
if key == "esc" || key == "enter" || key == "q" {
a.showCurlExport = false
} else if key == "y" {
if err := clipboard.WriteAll(a.curlExportVal); err == nil {
a.setStatus("cURL copied to clipboard ✓")
}
a.showCurlExport = false
}
break
}
if a.showNotification {
a.showNotification = false
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
return a.renderModal(a.confirmMsg + "\n\n[enter] Confirm  [n/esc] Cancel")
}

if a.showInput {
runes := []rune(a.inputValue)
cur := a.inputCursor
before := string(runes[:cur])
after := ""
if cur < len(runes) {
after = string(runes[cur:])
}
display := before + "█" + after
return a.renderModal(a.inputTitle + "\n\n" + display + "\n\n[enter] Confirm  [esc] Cancel")
}

if a.showCellEdit {
return a.renderCellEditModal()
}

if a.showCurlImport {
return a.renderCurlImportModal()
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
a.currentResp = nil
a.response.setResponse(nil)
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
