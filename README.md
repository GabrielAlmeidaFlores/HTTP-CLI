# http-cli

**A terminal-native HTTP client — like Postman, but lives in your terminal.**

`http-cli` is a fully interactive TUI (terminal user interface) HTTP testing tool built in Go. It lets you create, organize, and execute HTTP requests without leaving the terminal. Vim-style navigation, configurable keybindings, persistent request storage, and a clean multi-panel layout make it a complete replacement for GUI HTTP clients in keyboard-driven workflows.

```
┌─────────────────────┬──────────────────────────────────────┐
│  Requests           │  Editor                              │
│                     │  URL · Headers · Body · Query · Auth │
│  > GET  /users      │                                      │
│    POST /login      │  GET  https://api.example.com/users  │
│    PUT  /users/1    │                                      │
│                     ├──────────────────────────────────────┤
│                     │  Response                            │
│                     │  Body · Headers · Info               │
│                     │                                      │
│                     │  200 OK  · 142ms · 1.2KB             │
└─────────────────────┴──────────────────────────────────────┘
 ctrl+e Send  y Copy  j↓ k↑  g Top  G Bottom  ctrl+d ½Page
```

---

## Features

- **Multi-panel TUI** — requests list, editor, and response viewer side by side
- **5-tab request editor** — URL, Headers, Body, Query, Auth
- **All HTTP methods** — GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS
- **Body types** — JSON, form-data, multipart, raw, binary
- **File uploads** — toggle any form-data field to FILE mode with `t`
- **Auth support** — Bearer token, Basic, API Key
- **Vim-style navigation** — j/k/g/G/ctrl+d/ctrl+u throughout
- **Import from cURL** — paste any `curl` command and it becomes a request
- **Persistent storage** — requests saved in SQLite automatically
- **Configurable keybindings** — every action key comes from `configs/config.json`
- **Contextual hints** — footer shows only the shortcuts relevant to what you are doing
- **Modal overlays** — cell editor, confirm dialogs, and notifications render over the live UI

---

## Installation

### Requirements

- Go 1.21+

### Build from source

```bash
git clone https://github.com/user/http-cli
cd http-cli
go build -o http-cli ./cmd/http-cli
./http-cli
```

### Or run directly

```bash
go run ./cmd/http-cli
```

---

## Usage

### Starting

```bash
./http-cli
```

The TUI opens immediately. No arguments required.

### Panel Navigation

| Key | Action |
|---|---|
| `Tab` | Focus next panel |
| `Shift+Tab` | Focus previous panel |
| `1` | Jump to Requests panel |
| `2` | Jump to Editor panel |
| `3` | Jump to Response panel |

### Requests Panel

| Key | Action |
|---|---|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `Enter` | Open request in editor |
| `n` | New request |
| `I` | Import from cURL |
| `r` | Rename request |
| `y` | Duplicate request |
| `d` | Delete request (confirm with `Enter`) |
| `/` | Search requests |

### Editor Panel

Navigate with arrow keys. Press `e` to open the cell editor modal.

| Key | Action |
|---|---|
| `↑↓←→` | Navigate rows and columns |
| `e` | Edit selected cell (opens modal) |
| `Space` | Toggle row enabled/disabled |
| `d` | Delete current row |
| `t` | Toggle text / FILE (form-data only) |
| `←→` on method/type | Cycle values |
| `1`–`5` | Switch tabs (URL/Headers/Body/Query/Auth) |
| `ctrl+e` | Execute request |
| `ctrl+s` | Save request |

#### Cell Edit Modal

| Key | Action |
|---|---|
| `Enter` | Save and close |
| `ctrl+d` | Save without closing |
| `ctrl+j` | Insert newline |
| `Esc` | Cancel |

### Response Panel

| Key | Action |
|---|---|
| `j` / `↓` | Scroll down one line |
| `k` / `↑` | Scroll up one line |
| `ctrl+d` | Half page down |
| `ctrl+u` | Half page up |
| `ctrl+f` | Full page down |
| `ctrl+b` | Full page up |
| `g` | Jump to top |
| `G` | Jump to bottom |
| `y` | Copy response body to clipboard |
| `]` / `[` | Next / previous tab |

### Global

| Key | Action |
|---|---|
| `ctrl+e` | Execute current request |
| `ctrl+s` | Save current request |
| `q` / `ctrl+c` | Quit |

---

## Importing from cURL

Press `I` in the Requests panel to open the import modal. Paste any `curl` command:

```bash
curl -X POST https://api.example.com/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"secret"}'
```

The request is parsed and added to your collection immediately.

---

## Configuration

All keybindings, hints, theme colors, and layout settings are in `configs/config.json`.

### Changing a keybinding

```json
"request_list": {
  "new_request": {
    "keys": ["a"],
    "description": "New request",
    "category": "Requests",
    "visible": true
  }
}
```

Change `"keys"` to any key or combination. Multiple keys are supported:

```json
"keys": ["n", "ctrl+n"]
```

### Hiding a hint from the footer

Set `"visible": false` on any binding to stop it from appearing in the footer hints.

### Theme colors

```json
"ui": {
  "theme": {
    "primary": "#00d7ff",
    "focus_border": "#00d7ff",
    "blur_border": "#626262",
    "method_get": "#00d700",
    "method_post": "#d7d700",
    "method_put": "#d75f00",
    "method_delete": "#d70000",
    "method_patch": "#00d7af"
  }
}
```

Values are hex colors or terminal color names.

### Full config reference

See [`configs/config.json`](configs/config.json) for the complete annotated configuration file.

---

## Project Structure

```
http-cli/
├── cmd/http-cli/main.go      # Entry point
├── configs/config.json       # All keybindings, theme, layout
└── internal/
    ├── config/               # Config loading
    ├── models/               # Request, Response, Collection types
    ├── storage/              # SQLite persistence
    ├── transport/            # HTTP client + cURL parser
    ├── parser/               # .http and Postman file parsers
    ├── exporter/             # Export to file formats
    └── ui/
        ├── app.go            # BubbleTea model lifecycle
        ├── app_actions.go    # Action dispatch
        ├── app_keys.go       # Key routing
        ├── app_modals.go     # Modal state and rendering
        ├── app_render.go     # Layout rendering
        ├── ports.go          # Storage and HTTP interfaces
        ├── editor.go         # Request editor (5 tabs)
        ├── kv_table.go       # Key-value table widget
        ├── select_box.go     # Dropdown select widget
        ├── response.go       # Response viewer
        ├── request_list.go   # Request list
        └── keybindings/      # Keybinding manager
```

---

## Architecture

`http-cli` follows SOLID principles:

- **Single Responsibility** — each file has one concern (actions, keys, rendering, modals are separate)
- **Open/Closed** — new actions are added by extending `config.json` + one `case` in `app_actions.go`, without touching existing code
- **Dependency Inversion** — the UI layer depends on `RequestStore` and `HTTPExecutor` interfaces, not concrete types; implementations are wired in `main.go`

See [`AGENTS.md`](AGENTS.md) for the full developer guide including patterns, adding new features, and contribution rules.

---

## License

MIT
