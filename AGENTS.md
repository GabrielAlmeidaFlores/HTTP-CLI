# HTTP CLI — Development Guide

**Version**: 2.0.0
**Status**: Active Development
**Updated**: March 2026
**Module**: `github.com/user/http-cli`

---

## Table of Contents

1. [Project Overview](#1-project-overview)
2. [Architecture](#2-architecture)
3. [SOLID Principles Applied](#3-solid-principles-applied)
4. [File Structure](#4-file-structure)
5. [Keybinding System](#5-keybinding-system)
6. [Configuration Reference](#6-configuration-reference)
7. [TUI Component Model](#7-tui-component-model)
8. [Adding New Features](#8-adding-new-features)
9. [Development Workflow](#9-development-workflow)
10. [Implemented Features](#10-implemented-features)

---

## 1. Project Overview

HTTP CLI is a terminal-native HTTP client (like Postman) built in Go using the BubbleTea TUI framework. All behavior is driven by a single JSON configuration file. No hardcoded shortcuts — every action key is user-configurable.

### Technology Stack

| Concern | Library |
|---|---|
| TUI framework | `charmbracelet/bubbletea` |
| Layout & styling | `charmbracelet/lipgloss` |
| ANSI string ops | `charmbracelet/x/ansi` |
| Clipboard | `atotto/clipboard` |
| Storage | SQLite via `internal/storage` |

### Running

```bash
export PATH=$PATH:/usr/local/go/bin
go run ./cmd/http-cli
```

---

## 2. Architecture

### Layer Diagram

```
┌─────────────────────────────────────────────┐
│  TUI Layer  (internal/ui/)                  │
│                                             │
│  App (orchestrator)                         │
│    ├── RequestListModel  (left panel)       │
│    ├── EditorModel       (top-right panel)  │
│    └── ResponseModel     (bottom-right)     │
│                                             │
│  Keybinding Manager  (all key routing)      │
└────────────────┬────────────────────────────┘
                 │ interfaces (ports.go)
┌────────────────▼────────────────────────────┐
│  Service Layer                              │
│    RequestStore  →  internal/storage        │
│    HTTPExecutor  →  internal/transport      │
└─────────────────────────────────────────────┘
```

### BubbleTea Message Flow

```
KeyPress
  │
  ▼
App.Update(tea.KeyMsg)
  │
  ├─ modal active? → handleXxxModal()
  │
  ├─ PanelRequestList → handleKey() → Resolve(key, "request_list")
  │                                       → executeAction(action)
  │
  ├─ PanelEditor     → handleEditorKey()
  │                       → Resolve(key, "editor") → executeAction(action)
  │                       → editor.handleKey()
  │
  └─ PanelResponse   → handleKey() → Resolve(key, "response")
                                         → executeAction(action)
```

### Modal Priority (in Update loop)

Modals are checked in this order — the first active one wins:

1. `showConfirm` → confirm dialog
2. `showInput` → input dialog
3. `showCellEdit` → cell edit modal
4. `showCurlImport` → cURL import modal
5. `showNotification` → notification modal

All modals render as overlays on top of the live background UI via `overlayCenter()`.

---

## 3. SOLID Principles Applied

### Single Responsibility

Each file in `internal/ui/` has exactly one concern:

| File | Responsibility |
|---|---|
| `app.go` | App struct, Init, Update, View, size helpers |
| `app_actions.go` | `executeAction()` — all action dispatch |
| `app_keys.go` | Key routing: `handleKey`, `handleEditorKey`, modal handlers, panel navigation |
| `app_modals.go` | Modal state helpers, modal input handlers, `renderXxxModal()` |
| `app_render.go` | Layout rendering: topbar, main area, status bar, hints, overlay |
| `editor.go` | 5-tab request editor panel |
| `kv_table.go` | Key-value table widget (headers, query, form-data) |
| `select_box.go` | Dropdown select widget (method, body type, auth type) |
| `response.go` | Response viewer with vim-style navigation |
| `request_list.go` | Request list panel |
| `ports.go` | Interface definitions (DIP) |
| `util.go` | Shared helpers: `padRight`, `truncate`, `formatSize`, `isPrintable` |
| `keybindings/manager.go` | Keybinding loading, resolution, hint generation |

### Open/Closed

`executeAction(action string)` dispatches on action name strings. New actions are added by:
1. Adding a keybinding entry to `configs/config.json`
2. Adding a `case "action_name":` in `app_actions.go`

No existing code is modified — only extended.

### Liskov Substitution

`RequestStore` and `HTTPExecutor` interfaces (defined in `ports.go`) can be swapped for any implementation (mock, different storage, etc.) without changing `App`.

### Interface Segregation

`ports.go` defines two narrow interfaces — the UI layer only sees what it needs:

```go
type RequestStore interface {
    SaveRequest(ctx context.Context, req *models.Request) error
    DeleteRequest(ctx context.Context, id string) error
    ListRequests(ctx context.Context) ([]*models.Request, error)
    AddHistory(ctx context.Context, resp *models.Response) error
}

type HTTPExecutor interface {
    Execute(ctx context.Context, req *models.Request, envVars map[string]string) (*models.Response, error)
}
```

### Dependency Inversion

`App` depends on `RequestStore` and `HTTPExecutor` interfaces, not on `*storage.SQLiteStore` or `*transport.Client` directly. Concrete types are wired in `cmd/http-cli/main.go` (composition root).

---

## 4. File Structure

```
http-cli/
├── cmd/http-cli/main.go          # Entry point, dependency wiring
├── configs/
│   └── config.json               # ALL keybindings, hints, theme, layout
├── internal/
│   ├── config/config.go          # Config struct + JSON loading
│   ├── models/
│   │   ├── request.go            # Request, Body, Auth structs
│   │   ├── response.go           # Response struct + helpers
│   │   └── collection.go         # Collection struct
│   ├── storage/store.go          # SQLite implementation of RequestStore
│   ├── transport/
│   │   ├── client.go             # HTTP client (implements HTTPExecutor)
│   │   └── curl_parser.go        # cURL command parser
│   ├── parser/
│   │   ├── http_file.go          # .http file parser
│   │   └── postman.go            # Postman collection parser
│   ├── exporter/export.go        # Export to file formats
│   └── ui/
│       ├── ports.go              # RequestStore, HTTPExecutor interfaces
│       ├── app.go                # App struct + BubbleTea lifecycle
│       ├── app_actions.go        # executeAction() dispatch
│       ├── app_keys.go           # Key routing + modal handlers
│       ├── app_modals.go         # Modal state + render functions
│       ├── app_render.go         # Layout rendering + overlay
│       ├── editor.go             # 5-tab request editor
│       ├── kv_table.go           # Key-value table widget
│       ├── select_box.go         # Dropdown select widget
│       ├── table_editor.go       # Table/select wiring (kept for compat)
│       ├── response.go           # Response viewer
│       ├── request_list.go       # Request list panel
│       ├── util.go               # Shared helpers
│       └── keybindings/
│           └── manager.go        # Keybinding manager
└── AGENTS.md                     # This file
```

---

## 5. Keybinding System

### How It Works

Every user-facing key action is defined in `configs/config.json`. No action key is hardcoded in Go source.

```
Key Press
  → keybindMgr.Resolve(key, panel)  // looks up by key + panel
  → returns Binding{Action: "action_name"}
  → executeAction("action_name")
```

### Resolution Priority

When the same key exists in multiple panels, the most specific match wins:

| Panel | Score |
|---|---|
| Exact panel match (`editor`, `response`, etc.) | 20 |
| `global` | 5 |
| `navigation` | 3 |

### Panels

| Panel name | Used when |
|---|---|
| `global` | Always active |
| `navigation` | Always active (lower priority) |
| `request_list` | Left panel focused |
| `editor` | Editor panel focused |
| `response` | Response panel focused |

### Hint Filtering

`GetHints(panel, activeTab)` returns only bindings where:
- `b.Panel == panel` (exact match — no global bleed)
- If `b.Tab != ""`: only when `activeTab == b.Tab`
- If `b.Tab == ""`: always shown for that panel

### Adding a New Keybinding

1. Add to `configs/config.json`:
```json
"panel_name": {
  "my_action": {
    "keys": ["x"],
    "description": "Do something",
    "category": "My Category",
    "visible": true,
    "tab": "Body"
  }
}
```

2. Handle in `app_actions.go`:
```go
case "my_action":
    // implement behavior
```

Hints appear automatically in the footer for the correct panel/tab. No other changes needed.

---

## 6. Configuration Reference

File: `configs/config.json`

### Keybinding Entry Schema

```json
{
  "action_name": {
    "keys": ["key1", "key2"],
    "description": "Human-readable label for hints",
    "category": "Category Name",
    "visible": true,
    "tab": "TabName"
  }
}
```

| Field | Required | Purpose |
|---|---|---|
| `keys` | yes | List of key strings that trigger this action |
| `description` | yes | Shown in footer hints |
| `category` | yes | Groups hints together |
| `visible` | yes | `true` = show in footer hints |
| `tab` | no | Only show hint when this editor/response tab is active |

### Current Panels & Actions

#### `global`
| Action | Default Key | Description |
|---|---|---|
| `exit` | q, ctrl+c | Quit |
| `save` | ctrl+s | Save request |
| `execute_request` | ctrl+e | Execute request |
| `search` | / | Search requests |
| `cancel` | esc | Cancel / back |
| `next_panel` | tab | Focus next panel |
| `prev_panel` | shift+tab | Focus prev panel |

#### `navigation`
| Action | Default Key | Description |
|---|---|---|
| `down` | j, ↓ | Move down |
| `up` | k, ↑ | Move up |
| `focus_panel_1` | 1 | Focus requests panel |
| `focus_panel_2` | 2 | Focus editor panel |
| `focus_panel_3` | 3 | Focus response panel |

#### `request_list`
| Action | Default Key | Description |
|---|---|---|
| `new_request` | n | New request |
| `import_curl` | I | Import from cURL |
| `delete_request` | d | Delete request |
| `duplicate_request` | y | Duplicate request |
| `rename_request` | r | Rename request |
| `select_request` | enter | Open request |

#### `editor`
| Action | Default Key | Tab | Description |
|---|---|---|---|
| `execute` | ctrl+e | — | Send request |
| `tab_1..5` | 1..5 | — | Switch editor tab |
| `url_hint_e` | e | URL | Edit URL |
| `headers_hint_e` | e | Headers | Edit cell |
| `headers_hint_d` | d | Headers | Delete row |
| `headers_hint_space` | space | Headers | Toggle enabled |
| `body_hint_e` | e | Body | Edit cell / open type |
| `body_hint_t` | t | Body | Toggle text/file |
| `query_hint_e` | e | Query | Edit cell |
| `query_hint_d` | d | Query | Delete row |
| `auth_hint_e` | e | Auth | Edit field |
| `auth_hint_lr` | ←→ | Auth | Cycle auth type |

#### `response`
| Action | Default Key | Description |
|---|---|---|
| `execute` | ctrl+e | Send request |
| `copy_body` | y | Copy response body |
| `scroll_down` | j, ↓ | Scroll down |
| `scroll_up` | k, ↑ | Scroll up |
| `half_page_down` | ctrl+d | Half page down |
| `half_page_up` | ctrl+u | Half page up |
| `page_down` | ctrl+f | Full page down |
| `page_up` | ctrl+b | Full page up |
| `scroll_top` | g | Go to top |
| `scroll_bottom` | G | Go to bottom |
| `next_tab` | ] | Next tab |
| `prev_tab` | [ | Prev tab |

### UI Configuration

```json
"ui": {
  "hints": {
    "enabled": true,
    "position": "bottom",
    "show_descriptions": true,
    "key_color": "cyan",
    "description_color": "245"
  },
  "layout": {
    "left_panel_width_ratio": 0.25,
    "border_style": "rounded",
    "show_status_bar": true
  },
  "theme": {
    "primary": "#00d7ff",
    "focus_border": "#00d7ff",
    "blur_border": "#626262",
    "method_get": "#00d700",
    "method_post": "#d7d700",
    "method_delete": "#d70000"
  }
}
```

---

## 7. TUI Component Model

### App Struct (app.go)

The App holds all state. BubbleTea calls `Init → Update → View` every frame.

```go
type App struct {
    cfg        *config.Config
    keybindMgr *keybindings.Manager
    store      RequestStore    // interface, not concrete
    httpClient HTTPExecutor    // interface, not concrete
    focused    FocusedPanel
    // ... panels, modal states, etc.
}
```

### EditorModel (editor.go)

5-tab editor: URL · Headers · Body · Query · Auth

- `syncFromRequest()` — rebuilds all tables from the current request model (called on request select, tab switch)
- `syncToRequest()` — writes table state back to the request model (called on every key)
- `IsSubEditing() bool` — true when a cell edit or selectbox is open (gates tab switching)
- `CurrentCell*()` — API used by cell edit modal to read/write the selected cell

### kvTable (kv_table.go)

Key-value table widget used in Headers, Query, Body (form-data, multipart).

- Arrow keys navigate rows/columns
- `e` opens the cell edit modal (action resolved via keybinding manager)
- `space` toggles row enabled/disabled
- `d` deletes the current row
- `t` toggles text/FILE type (only on form-data with `showFileType: true`)
- Auto-adds a new row when navigating down past the last row
- Selected cell always shows a blue background at fixed width (no disappearing)
- Long values show a sliding viewport cursor in the modal

### selectBox (select_box.go)

Inline dropdown for HTTP method, body type, auth type.

- Closed: `←→` cycles values directly
- Open: `↑↓` navigates, `enter`/`space`/`e` opens, `esc` closes without change
- Dropdown items have symmetric padding (left and right)

### ResponseModel (response.go)

3-tab viewer: Body · Headers · Info

- Vim-style navigation: j/k/g/G/ctrl+d/ctrl+u/ctrl+f/ctrl+b
- Lines are clamped to panel width (no horizontal overflow)
- Scroll offset clamped to `totalLines() - contentHeight()`
- Scroll percentage shown in tab bar
- `y` copies body to clipboard

### Modal Overlay (app_render.go)

All modals render on top of the live background UI:

```go
func overlayCenter(bg, fg string, w, h int) string
```

Uses `charmbracelet/x/ansi` for ANSI-aware string slicing. The background is never replaced with a black screen.

---

## 8. Adding New Features

### New Action (e.g., duplicate response)

1. Add to `configs/config.json` under the right panel:
```json
"response": {
  "duplicate_something": { "keys": ["D"], "description": "...", "category": "Response", "visible": true }
}
```

2. Add case in `app_actions.go`:
```go
case "duplicate_something":
    // implement
```

Done. Hints appear automatically.

### New Modal

1. Add state fields to `App` struct in `app.go`:
```go
showMyModal  bool
myModalValue string
```

2. Add open helper in `app_modals.go`:
```go
func (a *App) openMyModal() {
    a.showMyModal = true
}
```

3. Add `handleMyModal(msg tea.KeyMsg) tea.Cmd` in `app_keys.go`

4. Add `renderMyModal() string` in `app_modals.go`

5. Wire into `Update()` in `app.go`:
```go
if a.showMyModal {
    cmds = append(cmds, a.handleMyModal(msg))
    break
}
```

6. Wire into `View()` in `app.go`:
```go
if a.showMyModal {
    return a.renderMyModal()
}
```

### New Editor Tab

Editor tabs are defined by `TabID` constants in `editor.go`. Add:
1. New `TabID` constant
2. Entry in `tabLabels`
3. `renderXxxTab()` method
4. Case in `renderTabContent()`
5. Case in `handleXxxTab()` and wire in `handleKey()`
6. Add tab keybinding in `config.json` under `editor`

---

## 9. Development Workflow

### Build & Run

```bash
export PATH=$PATH:/usr/local/go/bin
go build ./...                      # compile check
go run ./cmd/http-cli               # run
```

### Before Committing

```bash
go fmt ./...
go vet ./...
go build ./...
```

### Commit Convention

```
type: short description

- detail 1
- detail 2

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
```

Types: `feat`, `fix`, `refactor`, `docs`, `chore`

### Code Rules

- No comments in Go code (code is self-documenting)
- English only in all identifiers and strings
- No hardcoded action keys in Go — always use `keybindMgr.Resolve()`
- New features must add hints to `config.json`
- Modal keys (enter/esc/backspace for text input) may remain hardcoded — these are widget-internal controls, not user-configurable actions
- All business logic goes in `app_actions.go` or service layer, not in render functions
- Render functions are pure: no state mutation

### Widget Internal Keys (OK to hardcode)

These are text-widget-level controls, NOT user-configurable actions:

| Key | Context |
|---|---|
| `enter` | Confirm in modals/dialogs |
| `esc` | Cancel in modals/dialogs |
| `backspace` | Delete char in text input |
| `left`/`right` | Move cursor in text input |
| `home`/`end`/`ctrl+a`/`ctrl+e` | Cursor home/end in text input |
| `ctrl+j` | Insert newline in textarea |
| `ctrl+shift+v` | Paste in import modal |
| `up`/`down` in selectBox | Navigate open dropdown |

---

## 10. Implemented Features

### Request Management
- Create, rename, duplicate, delete requests
- Requests persisted in SQLite
- Import from cURL command (`I`)
- Request list with search (`/`)

### Request Editor (5 tabs)

**URL tab**
- HTTP method selector (GET/POST/PUT/DELETE/PATCH/HEAD/OPTIONS)
- URL input field
- Cycle method with `←→`, edit URL with `e`

**Headers tab**
- Key-value table with enabled/disabled toggle
- Edit cells via modal (`e`), delete rows (`d`), toggle (`space`)

**Body tab**
- Body type: none / json / form-data / multipart / raw / binary
- Type selector with `←→`
- JSON/raw: edit in modal
- Form-data/multipart: key-value table with file upload support
- Toggle text/FILE type with `t` on form-data rows

**Query tab**
- URL query parameters as key-value table
- Same controls as Headers

**Auth tab**
- Auth type: none / bearer / basic / api-key
- Type selector with `←→`
- Fields for each auth type

### Response Viewer (3 tabs)

**Body tab**
- Syntax-aware display (JSON pretty-printed)
- Vim navigation: j/k/g/G/ctrl+d/ctrl+u/ctrl+f/ctrl+b
- Copy to clipboard with `y`
- Scroll percentage indicator
- Lines clamped to panel width

**Headers tab**
- Response headers with scroll support

**Info tab**
- Status code, duration, size, timestamp

### Modals
- Cell edit modal: full-width text editor with save/cancel
- cURL import: paste or type cURL, imports with success/error notification
- Confirm dialog: enter=confirm, n/esc=cancel
- Input dialog: for rename, new request name
- Notification modal: green ✓ success / red ✗ error overlay

### UI/UX
- All modals overlay the live background (no black screen)
- Footer hints filtered per panel and active tab
- Status bar with timed messages (5s)
- Focused panel highlighted with cyan border
- Execute shortcut (`ctrl+e`) shown in editor and response hints
- Copy shortcut (`y`) shown in response hints

