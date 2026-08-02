package ui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"my-new-go/internal/httpclient"
	"my-new-go/internal/models"
	"my-new-go/internal/store"
)

type focusArea int

const (
	focusSidebar focusArea = iota
	focusMethod
	focusURL
	focusHeaders
	focusBody
	focusResponse
)

var methods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD"}

type sendDoneMsg struct {
	result httpclient.Result
	req    models.Request
}

type Model struct {
	store *store.Store
	ws    models.Workspace

	width  int
	height int
	focus  focusArea
	ready  bool

	sidebarW int
	mainW    int
	editorH  int
	respH    int

	selectedIdx int
	methodIdx   int
	reqTab      int // 0 headers, 1 body
	respTab     int // 0 body, 1 headers

	urlInput   textinput.Model
	headerKeys []textinput.Model
	headerVals []textinput.Model
	headerEn   []bool
	bodyInput  textarea.Model
	respView   viewport.Model

	sending bool
	status  string
	resp    httpclient.Result
	dirty   bool
}

func New(st *store.Store, ws models.Workspace) Model {
	url := textinput.New()
	url.Placeholder = "https://api.example.com/{{path}}"
	url.CharLimit = 2048
	url.Prompt = ""
	url.Width = 40

	body := textarea.New()
	body.Placeholder = "Request body (JSON, form, text...)"
	body.SetHeight(8)
	body.ShowLineNumbers = false
	body.CharLimit = 1 << 20

	resp := viewport.New(40, 10)
	resp.SetContent("Send a request to see the response.")

	m := Model{
		store:     st,
		ws:        ws,
		urlInput:  url,
		bodyInput: body,
		respView:  resp,
		status:    fmt.Sprintf("workspace: %s", st.Path()),
	}
	m.loadSelectedIntoEditor()
	m.setFocus(focusURL)
	return m
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *Model) activeCollection() *models.Collection {
	for i := range m.ws.Collections {
		if m.ws.Collections[i].ID == m.ws.ActiveCollectionID {
			return &m.ws.Collections[i]
		}
	}
	if len(m.ws.Collections) == 0 {
		return nil
	}
	m.ws.ActiveCollectionID = m.ws.Collections[0].ID
	return &m.ws.Collections[0]
}

func (m *Model) activeEnv() *models.Environment {
	for i := range m.ws.Environments {
		if m.ws.Environments[i].ID == m.ws.ActiveEnvironmentID {
			return &m.ws.Environments[i]
		}
	}
	if len(m.ws.Environments) == 0 {
		return nil
	}
	m.ws.ActiveEnvironmentID = m.ws.Environments[0].ID
	return &m.ws.Environments[0]
}

func (m *Model) selectedRequest() *models.Request {
	col := m.activeCollection()
	if col == nil || len(col.Requests) == 0 {
		return nil
	}
	if m.selectedIdx < 0 {
		m.selectedIdx = 0
	}
	if m.selectedIdx >= len(col.Requests) {
		m.selectedIdx = len(col.Requests) - 1
	}
	return &col.Requests[m.selectedIdx]
}

func (m *Model) loadSelectedIntoEditor() {
	req := m.selectedRequest()
	if req == nil {
		m.urlInput.SetValue("")
		m.bodyInput.SetValue("")
		m.headerKeys = nil
		m.headerVals = nil
		m.headerEn = nil
		m.methodIdx = 0
		return
	}

	m.urlInput.SetValue(req.URL)
	m.bodyInput.SetValue(req.Body)
	m.methodIdx = indexOfMethod(req.Method)
	m.syncHeadersFrom(req.Headers)
	m.dirty = false
}

func (m *Model) syncHeadersFrom(headers []models.Header) {
	m.headerKeys = make([]textinput.Model, 0, len(headers)+1)
	m.headerVals = make([]textinput.Model, 0, len(headers)+1)
	m.headerEn = make([]bool, 0, len(headers)+1)
	for _, h := range headers {
		m.headerKeys = append(m.headerKeys, newHeaderInput(h.Key, 20))
		m.headerVals = append(m.headerVals, newHeaderInput(h.Value, 40))
		m.headerEn = append(m.headerEn, h.Enabled)
	}
	// trailing empty row for quick add
	m.headerKeys = append(m.headerKeys, newHeaderInput("", 20))
	m.headerVals = append(m.headerVals, newHeaderInput("", 40))
	m.headerEn = append(m.headerEn, true)
}

func newHeaderInput(val string, width int) textinput.Model {
	ti := textinput.New()
	ti.SetValue(val)
	ti.Prompt = ""
	ti.Width = width
	ti.CharLimit = 1024
	return ti
}

func indexOfMethod(method string) int {
	method = strings.ToUpper(strings.TrimSpace(method))
	for i, m := range methods {
		if m == method {
			return i
		}
	}
	return 0
}

func (m *Model) collectEditorRequest() models.Request {
	req := models.Request{
		Method: methods[m.methodIdx],
		URL:    m.urlInput.Value(),
		Body:   m.bodyInput.Value(),
	}
	if cur := m.selectedRequest(); cur != nil {
		req.ID = cur.ID
		req.Name = cur.Name
	} else {
		req.ID = models.NewID()
		req.Name = "Untitled"
	}

	for i := range m.headerKeys {
		k := strings.TrimSpace(m.headerKeys[i].Value())
		v := m.headerVals[i].Value()
		if k == "" && strings.TrimSpace(v) == "" {
			continue
		}
		en := true
		if i < len(m.headerEn) {
			en = m.headerEn[i]
		}
		req.Headers = append(req.Headers, models.Header{Key: k, Value: v, Enabled: en})
	}
	return req
}

func (m *Model) applyEditorToSelected() {
	req := m.collectEditorRequest()
	col := m.activeCollection()
	if col == nil {
		return
	}
	if len(col.Requests) == 0 {
		col.Requests = append(col.Requests, req)
		m.selectedIdx = 0
		return
	}
	col.Requests[m.selectedIdx] = req
}

func (m *Model) saveWorkspace() error {
	m.applyEditorToSelected()
	if err := m.store.Save(m.ws); err != nil {
		return err
	}
	m.dirty = false
	m.status = "saved"
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout()
		m.ready = true
		return m, nil

	case sendDoneMsg:
		m.sending = false
		m.resp = msg.result
		m.respTab = 0 // show body first
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

	case tea.KeyMsg:
		key := msg.String()

		// Global quit — ctrl+c always; q only outside text inputs.
		if key == "ctrl+c" || (key == "q" && m.focus != focusURL && m.focus != focusHeaders && m.focus != focusBody) {
			return m, tea.Quit
		}

		switch key {
		case "ctrl+s":
			if err := m.saveWorkspace(); err != nil {
				m.status = "save failed: " + err.Error()
			}
			return m, nil

		case "ctrl+n":
			m.applyEditorToSelected()
			col := m.activeCollection()
			if col == nil {
				return m, nil
			}
			col.Requests = append(col.Requests, models.Request{
				ID:     models.NewID(),
				Name:   fmt.Sprintf("Request %d", len(col.Requests)+1),
				Method: "GET",
				URL:    "{{base_url}}/",
				Headers: []models.Header{
					{Key: "Accept", Value: "application/json", Enabled: true},
				},
			})
			m.selectedIdx = len(col.Requests) - 1
			m.loadSelectedIntoEditor()
			m.dirty = true
			m.status = "new request"
			return m, nil

		case "ctrl+d":
			col := m.activeCollection()
			if col == nil || len(col.Requests) == 0 {
				return m, nil
			}
			col.Requests = append(col.Requests[:m.selectedIdx], col.Requests[m.selectedIdx+1:]...)
			if m.selectedIdx >= len(col.Requests) {
				m.selectedIdx = len(col.Requests) - 1
			}
			m.loadSelectedIntoEditor()
			m.dirty = true
			_ = m.saveWorkspace()
			m.status = "deleted request"
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
		if len(m.headerKeys) > 0 {
			// edit last non-empty-friendly row: first enabled key field for simplicity
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
					// ensure trailing blank row
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
		}
	case focusResponse:
		m.respView, cmd = m.respView.Update(msg)
	}
	return m, cmd
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
	m.urlInput.Blur()
	m.bodyInput.Blur()
	for i := range m.headerKeys {
		m.headerKeys[i].Blur()
		m.headerVals[i].Blur()
	}
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

func (m *Model) layout() {
	m.sidebarW = 28
	if m.width < 90 {
		m.sidebarW = 22
	}
	m.mainW = m.width - m.sidebarW - 6
	if m.mainW < 40 {
		m.mainW = 40
	}
	m.urlInput.Width = max(20, m.mainW-18)

	// title + status + help + borders
	avail := m.height - 5
	if avail < 16 {
		avail = 16
	}
	m.editorH = avail * 2 / 5
	if m.editorH < 10 {
		m.editorH = 10
	}
	m.respH = avail - m.editorH
	if m.respH < 8 {
		m.respH = 8
	}

	innerW := max(20, m.mainW-4)
	m.bodyInput.SetWidth(innerW)
	m.bodyInput.SetHeight(max(3, m.editorH-8))
	m.respView.Width = innerW
	m.respView.Height = max(3, m.respH-5)
}

func (m *Model) refreshResponseView() {
	m.respView.SetContent(formatResponseTab(m.resp, m.respTab))
	m.respView.GotoTop()
}

func (m Model) View() string {
	if !m.ready {
		return "loading..."
	}

	col := m.activeCollection()
	env := m.activeEnv()
	envName := "none"
	if env != nil {
		envName = env.Name
	}
	colName := "none"
	if col != nil {
		colName = col.Name
	}

	header := titleStyle.Render("API TUI") +
		labelStyle.Render(fmt.Sprintf("  %s / %s", colName, envName))
	if m.dirty {
		header += labelStyle.Render("  • unsaved")
	}

	sidebar := m.viewSidebar()
	editor := m.viewEditor()
	response := m.viewResponse()

	main := lipgloss.JoinVertical(lipgloss.Left, editor, response)
	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)

	help := helpStyle.Render("enter/ctrl+r send · tab focus · ↑↓ scroll/select · b/h response body|headers · ctrl+s save · q quit")
	status := labelStyle.Render(m.status)
	if m.sending {
		status = valueStyle.Render("sending...")
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, body, status, help)
}

func (m Model) viewSidebar() string {
	var b strings.Builder
	b.WriteString(labelStyle.Render("COLLECTION") + "\n")
	col := m.activeCollection()
	if col == nil || len(col.Requests) == 0 {
		b.WriteString(listItem.Render("(empty — ctrl+n)"))
	} else {
		for i, req := range col.Requests {
			method := req.Method
			style := listItem
			prefix := "  "
			if i == m.selectedIdx {
				style = listSel
				prefix = "▸ "
			}
			ms := methodStyle(method)
			line := prefix + ms.Render(fmt.Sprintf("%-6s", method)) + " " + truncate(req.Name, 14)
			b.WriteString(style.Render(line) + "\n")
		}
	}

	b.WriteString("\n" + labelStyle.Render("HISTORY") + "\n")
	limit := 8
	if len(m.ws.History) < limit {
		limit = len(m.ws.History)
	}
	for i := 0; i < limit; i++ {
		h := m.ws.History[i]
		line := fmt.Sprintf("%s %d %s", h.Method, h.StatusCode, truncate(h.URL, 16))
		b.WriteString(listItem.Render(line) + "\n")
	}

	h := m.editorH + m.respH
	style := panelStyle.Width(m.sidebarW).Height(h)
	if m.focus == focusSidebar {
		style = focusedPanel.Width(m.sidebarW).Height(h)
	}
	return style.Render(b.String())
}

func (m Model) viewEditor() string {
	method := methods[m.methodIdx]
	ms := methodStyle(method)
	methodBox := ms.Render(" " + method + " ")
	if m.focus == focusMethod {
		methodBox = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(focusCol).Render(methodBox)
	}

	urlView := m.urlInput.View()
	reqLine := lipgloss.JoinHorizontal(lipgloss.Center, methodBox, " ", urlView)

	headersTab := tabIdle.Render(" Headers ")
	bodyTab := tabIdle.Render(" Body ")
	if m.reqTab == 0 {
		headersTab = tabActive.Render(" Headers ")
	} else {
		bodyTab = tabActive.Render(" Body ")
	}
	tabs := headersTab + bodyTab

	var content string
	if m.reqTab == 0 {
		var hb strings.Builder
		hb.WriteString(labelStyle.Render("KEY") + strings.Repeat(" ", 18) + labelStyle.Render("VALUE") + "\n")
		for i := range m.headerKeys {
			hb.WriteString(m.headerKeys[i].View() + "  " + m.headerVals[i].View() + "\n")
		}
		content = hb.String()
	} else {
		content = m.bodyInput.View()
	}

	inner := lipgloss.JoinVertical(lipgloss.Left, reqLine, "", tabs, content)
	style := panelStyle.Width(m.mainW).Height(m.editorH)
	if m.focus == focusMethod || m.focus == focusURL || m.focus == focusHeaders || m.focus == focusBody {
		style = focusedPanel.Width(m.mainW).Height(m.editorH)
	}
	return style.Render(inner)
}

func (m Model) viewResponse() string {
	meta := labelStyle.Render("RESPONSE")
	if m.resp.Status != "" {
		st := statusOK
		if m.resp.StatusCode >= 400 || m.resp.Error != "" {
			st = statusBad
		}
		meta = st.Render(m.resp.Status) + labelStyle.Render(fmt.Sprintf("  %dms", m.resp.Duration.Milliseconds()))
	}

	bodyTab := tabIdle.Render(" Body ")
	hdrTab := tabIdle.Render(" Headers ")
	if m.respTab == 0 {
		bodyTab = tabActive.Render(" Body ")
	} else {
		hdrTab = tabActive.Render(" Headers ")
	}

	inner := lipgloss.JoinVertical(lipgloss.Left, meta, bodyTab+hdrTab, m.respView.View())
	style := panelStyle.Width(m.mainW).Height(m.respH)
	if m.focus == focusResponse {
		style = focusedPanel.Width(m.mainW).Height(m.respH)
	}
	return style.Render(inner)
}

func methodStyle(method string) lipgloss.Style {
	if s, ok := methodStyles[strings.ToUpper(method)]; ok {
		return s
	}
	return valueStyle
}

func formatResponseTab(r httpclient.Result, tab int) string {
	if r.Error != "" && r.Body == "" && tab == 0 {
		return "Error: " + r.Error
	}
	if tab == 1 {
		if len(r.Headers) == 0 {
			if r.Status == "" {
				return "Send a request to see headers."
			}
			return "(no headers)"
		}
		keys := make([]string, 0, len(r.Headers))
		for k := range r.Headers {
			keys = append(keys, k)
		}
		// stable order for readability
		sort.Strings(keys)
		var b strings.Builder
		for _, k := range keys {
			b.WriteString(fmt.Sprintf("%s: %s\n", k, strings.Join(r.Headers[k], ", ")))
		}
		return b.String()
	}

	if r.Error != "" {
		return "Error: " + r.Error + "\n\n" + prettyBody(r.Body)
	}
	if r.Status == "" {
		return "Send a request to see the response body."
	}
	return prettyBody(r.Body)
}

func prettyBody(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return "(empty)"
	}
	var anyJSON any
	if json.Unmarshal([]byte(body), &anyJSON) == nil {
		pretty, err := json.MarshalIndent(anyJSON, "", "  ")
		if err == nil {
			return string(pretty)
		}
	}
	return body
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}
