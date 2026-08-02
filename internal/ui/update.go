package ui

import (
	"fmt"
	"strings"
	"time"

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
	if key == "q" && !m.showEnv && m.focus != focusURL && m.focus != focusHeaders && m.focus != focusBody {
		return m, tea.Quit
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

	case "ctrl+o":
		m.newCollection()
		return m, nil

	case "ctrl+w":
		m.deleteActiveCollection()
		return m, nil

	case "ctrl+d":
		m.deleteSelectedRequest()
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
		if m.focus == focusMethod {
			m.methodIdx = (m.methodIdx - 1 + len(methods)) % len(methods)
			m.dirty = true
			return m, nil
		}

	case "right", "l":
		if m.focus == focusMethod {
			m.methodIdx = (m.methodIdx + 1) % len(methods)
			m.dirty = true
			return m, nil
		}

	case "1", "2":
		if m.focus == focusSidebar || m.focus == focusMethod || m.focus == focusResponse {
			if key == "1" {
				if m.focus == focusResponse {
					m.respTab = 0
					m.refreshResponseView()
				} else {
					m.setFocus(focusHeaders)
				}
			} else {
				if m.focus == focusResponse {
					m.respTab = 1
					m.refreshResponseView()
				} else {
					m.setFocus(focusBody)
				}
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
			if m.selectedIdx > 0 {
				m.applyEditorToSelected()
				m.selectedIdx--
				m.loadSelectedIntoEditor()
			}
			return m, nil
		}

	case "down", "j":
		if m.focus == focusSidebar {
			col := m.activeCollection()
			if col != nil && m.selectedIdx < len(col.Requests)-1 {
				m.applyEditorToSelected()
				m.selectedIdx++
				m.loadSelectedIntoEditor()
			}
			return m, nil
		}

	case "enter":
		// Send from URL / method / sidebar. Headers/body keep Enter for editing.
		if m.focus == focusSidebar || m.focus == focusMethod || m.focus == focusURL {
			return m.startSend()
		}

	case "b":
		if m.focus == focusResponse || m.focus == focusSidebar || m.focus == focusMethod {
			m.respTab = 0
			m.refreshResponseView()
			m.setFocus(focusResponse)
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
	case focusBody:
		m.bodyInput, cmd = m.bodyInput.Update(msg)
		m.dirty = true
	case focusHeaders:
		cmd = m.updateHeaders(msg)
	case focusResponse:
		m.respView, cmd = m.respView.Update(msg)
	}
	return m, cmd
}

func (m *Model) updateHeaders(msg tea.Msg) tea.Cmd {
	if len(m.headerKeys) == 0 {
		return nil
	}

	var cmd tea.Cmd
	idx := len(m.headerKeys) - 1
	for i := range m.headerKeys {
		if m.headerKeys[i].Focused() {
			idx = i
			break
		}
	}
	if !m.headerKeys[idx].Focused() && !m.headerVals[idx].Focused() {
		m.headerKeys[idx].Focus()
	}
	if m.headerKeys[idx].Focused() {
		m.headerKeys[idx], cmd = m.headerKeys[idx].Update(msg)
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
			m.headerKeys[idx].Blur()
			m.headerVals[idx].Focus()
		}
	} else {
		m.headerVals[idx], cmd = m.headerVals[idx].Update(msg)
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
			m.headerVals[idx].Blur()
			last := len(m.headerKeys) - 1
			if strings.TrimSpace(m.headerKeys[last].Value()) != "" || strings.TrimSpace(m.headerVals[last].Value()) != "" {
				m.headerKeys = append(m.headerKeys, newHeaderInput("", 20))
				m.headerVals = append(m.headerVals, newHeaderInput("", 40))
				m.headerEn = append(m.headerEn, true)
			}
			next := idx + 1
			if next < len(m.headerKeys) {
				m.headerKeys[next].Focus()
			}
		}
	}
	m.dirty = true
	return cmd
}

func (m *Model) cycleFocus(dir int) {
	order := []focusArea{focusSidebar, focusMethod, focusURL, focusHeaders, focusBody, focusResponse}
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
	case focusBody:
		m.reqTab = 1
		m.bodyInput.Focus()
	case focusHeaders:
		m.reqTab = 0
		if len(m.headerKeys) > 0 {
			m.headerKeys[0].Focus()
		}
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
