// Package ui implements the Bubble Tea TUI for the API client.
//
// Files:
//   - model.go     — root Model, New, Init
//   - workspace.go — collections, env, requests, history, save
//   - tree.go      — folder tree flatten / path helpers
//   - name.go      — new folder / rename name prompt
//   - auth.go      — Auth tab + shared KV editors
//   - editor.go    — request editor sync (URL, auth, params, headers, body)
//   - update.go    — key handling, focus, send
//   - view.go      — layout and panel rendering
//   - format.go    — response formatting helpers
//   - env.go       — environment variable editor modal
//   - styles.go    — lipgloss theme
package ui
