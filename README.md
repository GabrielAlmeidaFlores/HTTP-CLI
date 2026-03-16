# HTTP CLI

A terminal-native HTTP client — like Postman, but lives in your terminal.

`http-cli` is a fully interactive TUI HTTP testing tool built in Go. Create, organize, and execute HTTP requests without leaving the terminal. Vim-style navigation, fully config-driven keybindings and colors, persistent storage, and contextual hints make it a complete replacement for GUI HTTP clients in keyboard-driven workflows.

<video width="100%" controls>
  <source src="https://raw.githubusercontent.com/GabrielAlmeidaFlores/GabrielAlmeidaFlores/main/assets/HTTP-CLI/http-cli.mp4" type="video/mp4">
  Your browser does not support the video tag.
</video>

---

## Features

- Three-panel layout — requests list, request editor, response viewer
- Five-tab request editor — URL, Headers, Body, Query, Auth
- All HTTP methods — GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS
- Body types — JSON, raw, form-data, multipart, URL-encoded
- File upload support — toggle any form-data field between text and file mode
- Auth — Bearer token, Basic auth, API Key
- Vim-style navigation — `j`/`k`/`g`/`G`/`ctrl+d`/`ctrl+u` throughout every panel
- Import requests from cURL — supports cookies, multiline, all common flags
- Export request as cURL — copy the current request as a ready-to-run curl command
- Import Postman collections — load a Postman v2.1 JSON export
- Export Postman collections — save all requests as a Postman-compatible file
- Open response in external editor — view the response body in your configured editor
- Open any cell in external editor — edit large values (headers, body) outside the TUI
- Persistent storage — requests saved automatically in SQLite
- All keybindings come from `configs/config.json` — no hardcoded keys in source
- All colors come from `configs/config.json` — fully themeable
- Contextual hints bar — footer shows only the shortcuts relevant to the active panel

---

## Requirements

- Go 1.21+

---

## Installation

```bash
git clone https://github.com/user/http-cli
cd http-cli
go build -o http-cli ./cmd/http-cli
./http-cli
```

Or run directly without building:

```bash
go run ./cmd/http-cli
```

---

## Usage

Run `./http-cli` to open the TUI. No arguments required.

### Panel Navigation

| Key | Action |
|---|---|
| `Tab` | Focus next panel |
| `Shift+Tab` | Focus previous panel |
| `q` / `ctrl+c` | Quit |

---

### Requests Panel

| Key | Action |
|---|---|
| `j` / `↓` | Move down (loads request in editor) |
| `k` / `↑` | Move up (loads request in editor) |
| `Enter` | Open request and focus editor |
| `ctrl+e` | Execute selected request |
| `n` | New request |
| `r` | Rename request |
| `y` | Duplicate request |
| `d` | Delete request |
| `/` | Search / filter requests |
| `I` | Import from cURL |
| `E` | Export current request as cURL |
| `P` | Import a Postman collection |
| `X` | Export all requests as Postman collection |

---

### Editor Panel

The editor has five tabs: URL, Headers, Body, Query, Auth. Navigate with arrow keys.

| Key | Action |
|---|---|
| `↑` / `↓` | Move between rows |
| `←` / `→` | Move between columns (or cycle method/type values) |
| `e` / `Enter` | Edit selected cell |
| `ctrl+o` | Open selected cell in external editor |
| `Space` | Toggle row enabled / disabled |
| `d` | Delete current row |
| `t` | Toggle text / FILE mode (form-data only) |
| `1` – `5` | Switch tabs (URL / Headers / Body / Query / Auth) |
| `]` / `[` | Next / previous tab |
| `ctrl+e` | Execute request |
| `ctrl+s` | Save request |
| `Tab` | Focus next panel |
| `Shift+Tab` | Focus previous panel |

#### Cell edit modal

Opens when you press `e` on a text cell. Full text editing with clipboard paste support.

| Key | Action |
|---|---|
| `Enter` | Save and close |
| `ctrl+d` | Save without closing |
| `ctrl+j` | Insert newline |
| `ctrl+o` | Open in external editor |
| `ctrl+shift+v` | Paste from clipboard |
| `Esc` | Cancel |

---

### Response Panel

Three tabs: Body, Headers, Info.

| Key | Action |
|---|---|
| `j` / `↓` | Scroll down |
| `k` / `↑` | Scroll up |
| `g` | Jump to top |
| `G` | Jump to bottom |
| `ctrl+d` | Half page down |
| `ctrl+u` | Half page up |
| `l` | Next tab |
| `h` | Previous tab |
| `1` | Body tab |
| `2` | Headers tab |
| `3` | Info tab |
| `y` | Copy response body to clipboard |
| `v` | Open response body in external editor |
| `Tab` | Focus next panel |
| `Shift+Tab` | Focus previous panel |

The **Info** tab shows timing with fast/moderate/slow label, server IP, protocol, content type, response size, and timestamp.

---

## Importing from cURL

Press `I` in the Requests panel. Paste any `curl` command, including multi-line commands with `\` continuation. Supported flags: `-X`, `-H`, `-d`, `--data-raw`, `--data-binary`, `-b`/`--cookie`, `-u`, `-A`/`--user-agent`, `-L`, and more.

Example:

```bash
curl -X POST https://api.example.com/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"secret"}'
```

---

## Exporting cURL

Press `E` in the Requests panel to see the current request as a curl command. Press `y` to copy it to the clipboard.

---

## Postman Collections

- **Import** (`P`) — provide a path to a Postman v2.1 JSON export file. All requests are added to your collection.
- **Export** (`X`) — provide a filename. All requests are written as a Postman-compatible JSON file you can import into Postman or share.

---

## External Editor

The external editor is used in two places:

- **Cell edit** — press `ctrl+o` inside the cell edit modal to open the cell value in your editor
- **Response body** — press `v` in the response panel to open the full response body in your editor

The editor is configured in `configs/config.json`:

```json
"external_editor": "vi"
```

Set it to any editor command: `"nano"`, `"nvim"`, `"code --wait"`, `"$EDITOR"`, etc. The content is written to a temporary file which is deleted after the editor closes.

---

## Configuration

Everything is in `configs/config.json`. There are no hardcoded keys or colors in the source code.

### Changing a keybinding

Find the action in the relevant panel section and change `"keys"`:

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

Multiple keys are supported: `"keys": ["n", "ctrl+n"]`

Set `"visible": false` to hide a shortcut from the hints bar without disabling it.

### Theme colors

All colors are under `"ui" → "theme"`. Values are hex colors:

```json
"theme": {
  "primary": "#00d7ff",
  "success": "#00d700",
  "error": "#d70000",
  "method_get": "#00d700",
  "method_post": "#d7d700",
  "method_delete": "#d70000"
}
```

See [`configs/config.json`](configs/config.json) for the full list of theme fields and all available panels and actions.

---

## Project Structure

```
http-cli/
├── cmd/http-cli/         Entry point
├── configs/config.json   All keybindings, colors, layout settings
└── internal/
    ├── config/           Config loading and types
    ├── models/           Request, Response, Collection types
    ├── storage/          SQLite persistence
    ├── transport/        HTTP client and cURL parser
    ├── parser/           Postman collection parser
    ├── exporter/         cURL and Postman exporters
    └── ui/               TUI — panels, editor, response, modals
```

---

## License

MIT

