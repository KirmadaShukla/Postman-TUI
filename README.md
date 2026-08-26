# API TUI

A terminal-based HTTP client inspired by Postman. Send requests, organize them in nested folders, manage environments, and browse history — all from your terminal.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Features

- **HTTP methods** — GET, POST, PUT, PATCH, DELETE, HEAD
- **Request editor** — Auth, Params, Headers, and Body tabs
- **Authentication** — No Auth, Bearer Token, Basic Auth, API Key (header or query)
- **Query params** — Key/value rows appended to the request URL on send
- **Environments** — Variables with `{{name}}` substitution in URL, headers, body, auth, and params
- **Collections** — Multiple collections with nested folder trees
- **History** — Navigate, restore, or delete past requests; clear all with one shortcut
- **Response viewer** — Body and headers tabs, JSON pretty-printing, clipboard copy
- **Persistence** — Workspace saved automatically to `~/.apitui/workspace.json`

## Requirements

- Go **1.26.5** or newer
- A terminal that supports alternate screen mode

## Install

Clone the repository and build the binary:

```bash
git clone <repo-url>
cd Postman-TUI
go build -o apitui .
```

Or run without installing:

```bash
go run .
```

## Usage

```bash
./apitui
```

On first launch, a default workspace is created at `~/.apitui/workspace.json` with a sample GET request and a Local environment (`base_url`, `token`).

Use `{{variable}}` anywhere in the request (for example `{{base_url}}/users`) and set values in the environment editor (`ctrl+e`).

## Layout

```
┌─ Sidebar ─────────┬─ Request editor ─────────────────────┐
│ Collection tree   │ METHOD  URL                        │
│ (folders/requests)│ Auth | Params | Headers | Body     │
│                   ├──────────────────────────────────────┤
│ History           │ Response (Body | Headers)          │
└───────────────────┴──────────────────────────────────────┘
  status bar · keyboard shortcuts
```

## Keyboard shortcuts

### Global

| Key | Action |
|-----|--------|
| `tab` / `shift+tab` | Cycle focus between panels |
| `ctrl+s` | Save workspace |
| `ctrl+c` / `q` | Quit (`q` only outside text inputs) |
| `ctrl+enter` / `ctrl+r` / `ctrl+g` / `ctrl+j` / `f5` | Send request |
| `enter` | Send (from URL or method bar) |

### Sidebar (collection & history)

| Key | Action |
|-----|--------|
| `j` / `k` or `↑` / `↓` | Move cursor (tree + history) |
| `←` / `→` | Collapse / expand folder |
| `enter` | Toggle folder · send request · restore history entry |
| `[` / `]` | Previous / next collection |
| `ctrl+n` | New request (inside selected folder) |
| `ctrl+f` | New folder (name prompt) |
| `ctrl+o` | New collection |
| `ctrl+w` | Delete active collection |
| `ctrl+d` | Delete request, folder, or history entry |
| `ctrl+y` | Clear all history |
| `r` | Rename folder or request |

### Request editor

| Key | Action |
|-----|--------|
| `←` / `→` or `h` / `l` | Change HTTP method (when method is focused) |
| `1` | Auth tab |
| `2` | Params tab |
| `3` | Headers tab |
| `4` | Body tab |
| `enter` | Next field (headers, params, auth) |

**Auth tab** — Use `←` / `→` to change type (None, Bearer, Basic, API Key). Use `↑` / `↓` to move between fields.

**Params / Headers** — `enter` moves key → value → next row.

### Response

| Key | Action |
|-----|--------|
| `b` | Focus response body |
| `h` | Focus response headers |
| `1` / `2` | Body / Headers tab (when response focused) |
| `y` | Copy current response tab to clipboard |
| `↑` / `↓` / `pgup` / `pgdown` | Scroll response |

### Environment editor (`ctrl+e`)

| Key | Action |
|-----|--------|
| `ctrl+s` | Save and close |
| `esc` | Cancel |
| `enter` / `tab` | Next field |
| `ctrl+d` | Delete row |

### Name prompt (new folder / rename)

| Key | Action |
|-----|--------|
| `enter` | Confirm |
| `esc` | Cancel |

## Workspace file

Data is stored at:

```
~/.apitui/workspace.json
```

The file includes collections (with nested items), environments, and request history (last 100 entries). Legacy flat `requests` arrays are migrated to the tree format on load.

## Development

```bash
# Run tests
go test ./...

# Build
go build -o apitui .
```

### Project structure

```
.
├── main.go                 # Entry point
├── internal/
│   ├── models/             # Workspace, collection tree, request models
│   ├── store/              # JSON persistence (~/.apitui)
│   ├── httpclient/         # HTTP send, auth, params, {{var}} substitution
│   └── ui/                 # Bubble Tea TUI (editor, sidebar, env, auth)
```

## Tips

- Select a **folder** in the sidebar before `ctrl+n` or `ctrl+f` to create requests or subfolders inside it.
- Use valid JSON in the body tab — a missing colon after a property name will cause API errors like `Expected ':' after property name in JSON`.
- For login endpoints, use **POST** (not GET) and set `Content-Type: application/json` if needed (auto-added when the body looks like JSON).
- Terminal native text selection works; use `y` to copy the response to the system clipboard.

## License

[MIT](LICENSE)
