package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"

	"my-new-go/internal/httpclient"
	"my-new-go/internal/models"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout()
		m.ready = true
		return m, nil

	case sendDoneMsg:
		return m.handleSendDone(msg)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m.updateFocused(msg)
}

func (m Model) handleSendDone(msg sendDoneMsg) (tea.Model, tea.Cmd) {
	m.sending = false
	m.resp = msg.result
	m.respTab = 0
	m.refreshResponseView()

	entry := models.HistoryEntry{
		ID:           models.NewID(),
		Timestamp:    time.Now(),
		Method:       msg.req.Method,
		URL:          msg.result.ResolvedURL,
		StatusCode:   msg.result.StatusCode,
		DurationMS:   msg.result.Duration.Milliseconds(),
		Headers:      msg.req.Headers,
		RequestBody:  msg.req.Body,
		ResponseBody: truncate(msg.result.Body, 4000),
	}
	m.ws.History = append([]models.HistoryEntry{entry}, m.ws.History...)
	_ = m.store.Save(m.ws)

	if msg.result.Error != "" {
		m.status = "error: " + msg.result.Error
	} else {
		m.status = fmt.Sprintf("%s · %dms", msg.result.Status, msg.result.Duration.Milliseconds())
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global quit — ctrl+c always; q only outside text inputs / env modal.
	if key == "ctrl+c" {
		return m, tea.Quit
	}
	if key == "q" && !m.showEnv && !m.showName &&
		m.focus != focusURL && m.focus != focusAuth && m.focus != focusParams &&
		m.focus != focusHeaders && m.focus != focusBody {
		return m, tea.Quit
	}

	if m.showName {
		return m.handleNameKey(msg)
	}
	if m.showEnv {
		return m.handleEnvKey(msg)
	}

	switch key {
	case "ctrl+s":
		if err := m.saveWorkspace(); err != nil {
			m.status = "save failed: " + err.Error()
		}
		return m, nil

	case "ctrl+e":
		m.openEnvEditor()
		return m, nil

	case "ctrl+n":
		m.newRequest()
		return m, nil

	case "ctrl+f":
		m.openNewFolderPrompt()
		return m, nil

	case "ctrl+o":
		m.newCollection()
		return m, nil

	case "ctrl+w":
		m.deleteActiveCollection()
		return m, nil

	case "ctrl+d":
		m.deleteSelected()
		return m, nil

	case "ctrl+y":
		m.clearHistory()
		return m, nil

	// ctrl+enter is unreliable on many terminals (esp. macOS); also accept
	// ctrl+r / ctrl+g / f5 / ctrl+j (common ctrl+enter alias).
	case "ctrl+enter", "ctrl+r", "ctrl+g", "ctrl+j", "f5":
		return m.startSend()

	case "tab":
		m.cycleFocus(1)
		return m, nil
	case "shift+tab":
		m.cycleFocus(-1)
		return m, nil

	case "left":
		if m.focus == focusSidebar {
			m.expandCollapse(false)
			return m, nil
		}
		if m.focus == focusMethod {
			m.methodIdx = (m.methodIdx - 1 + len(methods)) % len(methods)
			m.dirty = true
			return m, nil
		}

	case "right":
		if m.focus == focusSidebar {
			m.expandCollapse(true)
			return m, nil
		}
		if m.focus == focusMethod {
			m.methodIdx = (m.methodIdx + 1) % len(methods)
			m.dirty = true
			return m, nil
		}

	case "l":
		if m.focus == focusMethod {
			m.methodIdx = (m.methodIdx + 1) % len(methods)
			m.dirty = true
			return m, nil
		}

	case "1", "2", "3", "4":
		if m.focus == focusSidebar || m.focus == focusMethod || m.focus == focusResponse {
			if m.focus == focusResponse {
				if key == "1" {
					m.respTab = 0
					m.refreshResponseView()
				} else if key == "2" {
					m.respTab = 1
					m.refreshResponseView()
				}
				return m, nil
			}
			switch key {
			case "1":
				m.setFocus(focusAuth)
			case "2":
				m.setFocus(focusParams)
			case "3":
				m.setFocus(focusHeaders)
			case "4":
				m.setFocus(focusBody)
			}
			return m, nil
		}

	case "[":
		if m.focus == focusSidebar || m.focus == focusMethod || m.focus == focusResponse {
			m.switchCollection(-1)
			return m, nil
		}

	case "]":
		if m.focus == focusSidebar || m.focus == focusMethod || m.focus == focusResponse {
			m.switchCollection(1)
			return m, nil
		}

	case "up", "k":
		if m.focus == focusSidebar {
			m.moveSidebar(-1)
			return m, nil
		}

	case "down", "j":
		if m.focus == focusSidebar {
			m.moveSidebar(1)
			return m, nil
		}

	case "r":
		if m.focus == focusSidebar {
			m.openRenamePrompt()
			return m, nil
		}

	case "enter":
		if m.focus == focusSidebar {
			row, ok := m.cursorRow()
			if !ok {
				return m, nil
			}
			if row.kind == rowHistory {
				m.restoreHistory(row.histIdx)
				return m, nil
			}
			item := itemAtMut(m.activeCollection(), row.path)
			if item != nil && item.Kind == models.ItemFolder {
				m.toggleExpanded(item.ID)
				return m, nil
			}
			if item != nil && item.Kind == models.ItemRequest {
				m.selectedPath = copyPath(row.path)
				return m.startSend()
			}
			return m, nil
		}
		// Send from URL / method. Headers/body keep Enter for editing.
		if m.focus == focusMethod || m.focus == focusURL {
			return m.startSend()
		}

	case "b":
		if m.focus == focusResponse || m.focus == focusSidebar || m.focus == focusMethod {
			m.respTab = 0
			m.refreshResponseView()
			m.setFocus(focusResponse)
			return m, nil
		}

	case "y":
		// Yank current response tab (body/headers) to the system clipboard.
		if m.focus == focusResponse || m.focus == focusSidebar || m.focus == focusMethod {
			text := formatResponseTab(m.resp, m.respTab)
			if strings.TrimSpace(text) == "" || text == "Send a request to see the response body." || text == "Send a request to see headers." {
				m.status = "nothing to copy"
				return m, nil
			}
			if err := clipboard.WriteAll(text); err != nil {
				m.status = "copy failed: " + err.Error()
			} else {
				m.status = "copied to clipboard"
			}
			return m, nil
		}

	case "h":
		if m.focus == focusResponse {
			m.respTab = 1
			m.refreshResponseView()
			return m, nil
		}
		if m.focus == focusMethod {
			m.methodIdx = (m.methodIdx - 1 + len(methods)) % len(methods)
			m.dirty = true
			return m, nil
		}
	}

	return m.updateFocused(msg)
}

func (m Model) updateFocused(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.focus {
	case focusURL:
		m.urlInput, cmd = m.urlInput.Update(msg)
		m.dirty = true
	case focusAuth:
		return m.updateAuth(msg)
	case focusParams:
		cmd = m.updateKV(msg, &m.paramKeys, &m.paramVals, &m.paramEn)
	case focusBody:
		m.bodyInput, cmd = m.bodyInput.Update(msg)
		m.dirty = true
	case focusHeaders:
		cmd = m.updateKV(msg, &m.headerKeys, &m.headerVals, &m.headerEn)
	case focusResponse:
		m.respView, cmd = m.respView.Update(msg)
	}
	return m, cmd
}

func (m *Model) cycleFocus(dir int) {
	order := []focusArea{
		focusSidebar, focusMethod, focusURL,
		focusAuth, focusParams, focusHeaders, focusBody, focusResponse,
	}
	cur := 0
	for i, f := range order {
		if f == m.focus {
			cur = i
			break
		}
	}
	next := (cur + dir + len(order)) % len(order)
	m.setFocus(order[next])
}

func (m *Model) setFocus(f focusArea) {
	m.blurAll()
	m.focus = f
	switch f {
	case focusURL:
		m.urlInput.Focus()
	case focusAuth:
		m.reqTab = reqTabAuth
		m.authField = 0
		m.focusAuthField()
	case focusParams:
		m.reqTab = reqTabParams
		if len(m.paramKeys) > 0 {
			m.paramKeys[0].Focus()
		}
	case focusHeaders:
		m.reqTab = reqTabHeaders
		if len(m.headerKeys) > 0 {
			m.headerKeys[0].Focus()
		}
	case focusBody:
		m.reqTab = reqTabBody
		m.bodyInput.Focus()
	}
}

func (m Model) startSend() (tea.Model, tea.Cmd) {
	if m.sending {
		return m, nil
	}
	m.sending = true
	m.status = "sending..."
	return m, m.sendCmd()
}

func (m Model) sendCmd() tea.Cmd {
	req := m.collectEditorRequest()
	vars := map[string]string{}
	if env := m.activeEnv(); env != nil {
		for k, v := range env.Variables {
			vars[k] = v
		}
	}
	return func() tea.Msg {
		res := httpclient.Send(req, vars, 30*time.Second)
		return sendDoneMsg{result: res, req: req}
	}
}
