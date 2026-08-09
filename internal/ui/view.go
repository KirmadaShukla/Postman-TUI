package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"my-new-go/internal/models"
)

func (m Model) View() string {
	if !m.ready {
		return "loading..."
	}
	if m.showName {
		return m.viewNameModal()
	}
	if m.showEnv {
		return m.viewEnvModal()
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

	header := titleStyle.Render("▲ API TUI") +
		subtitleStyle.Render(fmt.Sprintf("%s  ›  %s", colName, envName))
	if m.dirty {
		header += dirtyBadge.Render("● unsaved")
	}
	header = headerBar.Render(header)

	sidebar := m.viewSidebar()
	editor := m.viewEditor()
	response := m.viewResponse()

	main := lipgloss.JoinVertical(lipgloss.Left, editor, response)
	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)

	helpBar := help(
		[2]string{"enter", "send/open"},
		[2]string{"tab", "focus"},
		[2]string{"ctrl+f", "folder"},
		[2]string{"r", "rename"},
		[2]string{"ctrl+n", "req"},
		[2]string{"y", "copy"},
		[2]string{"ctrl+e", "env"},
		[2]string{"ctrl+d", "del"},
		[2]string{"q", "quit"},
	)

	var statusLine string
	switch {
	case m.sending:
		statusLine = statusBarSending.Render("⟳ sending…")
	case strings.HasPrefix(m.status, "error"):
		statusLine = statusBarError.Render("✗ " + m.status)
	case m.status == "saved":
		statusLine = statusBarSaved.Render("✓ saved")
	case strings.HasPrefix(m.status, "environment"):
		statusLine = statusBarSaved.Render("✓ " + m.status)
	default:
		statusLine = statusBarIdle.Render(m.status)
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, body, statusLine, helpBar)
}

func (m Model) viewSidebar() string {
	innerW := max(10, m.sidebarW-2)
	var b strings.Builder
	col := m.activeCollection()
	colTitle := "◆ COLLECTION"
	if col != nil {
		idx := m.activeCollectionIndex()
		colTitle = fmt.Sprintf("◆ %s (%d/%d)", truncate(col.Name, innerW-10), idx+1, len(m.ws.Collections))
	}
	b.WriteString(sectionLabel.Render(colTitle) + "\n")
	if len(m.ws.Collections) > 1 {
		b.WriteString(emptyHint.Render("[ ] switch  ctrl+o new  ctrl+f folder") + "\n")
	} else {
		b.WriteString(emptyHint.Render("ctrl+o new · ctrl+f folder") + "\n")
	}

	rows := m.sidebarRows()
	treeCount := 0
	for _, r := range rows {
		if r.kind == rowTree {
			treeCount++
		}
	}
	if col == nil || treeCount == 0 {
		b.WriteString(emptyHint.Render("empty — ctrl+n / ctrl+f") + "\n")
	}

	histStarted := false
	for i, r := range rows {
		if r.kind == rowHistory && !histStarted {
			b.WriteString("\n" + sectionLabel.Render("◷ HISTORY") + "\n")
			histStarted = true
		}

		cursor := "  "
		if i == m.sidebarCursor {
			cursor = "▸ "
		}

		var body string
		switch r.kind {
		case rowTree:
			if col == nil {
				continue
			}
			item := itemAt(col.Items, r.path)
			if item == nil {
				continue
			}
			indent := strings.Repeat("  ", r.depth)
			if item.Kind == models.ItemFolder {
				chev := "▸"
				if m.isExpanded(item.ID) {
					chev = "▾"
				}
				name := truncate(item.Name, max(4, innerW-6-r.depth*2))
				body = indent + chev + " " + name
			} else {
				method := "GET"
				name := item.Name
				if item.Request != nil {
					method = item.Request.Method
					if name == "" {
						name = item.Request.Name
					}
				}
				badge := methodStyle(method).Render(fmt.Sprintf("%-4s", method))
				name = truncate(name, max(4, innerW-9-r.depth*2))
				body = indent + badge + " " + name
			}
		case rowHistory:
			h := m.ws.History[r.histIdx]
			code := statusChip(h.StatusCode, "").Render(fmt.Sprintf("%d", h.StatusCode))
			method := labelStyle.Render(fmt.Sprintf("%-4s", h.Method))
			body = method + code + " " + truncate(h.URL, max(4, innerW-14))
		}

		line := cursor + body
		if i == m.sidebarCursor {
			pad := innerW - lipgloss.Width(line)
			if pad > 0 {
				line += strings.Repeat(" ", pad)
			}
			b.WriteString(listSel.Render(line) + "\n")
		} else {
			b.WriteString(listItem.Render(line) + "\n")
		}
	}

	if len(m.ws.History) == 0 {
		b.WriteString("\n" + sectionLabel.Render("◷ HISTORY") + "\n")
		b.WriteString(emptyHint.Render("no requests sent yet") + "\n")
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
		methodBox = methodBoxFocused.Render(methodBox)
	}

	urlView := m.urlInput.View()
	reqLine := lipgloss.JoinHorizontal(lipgloss.Center, methodBox, " ", urlView)

	tab := func(label string, id int) string {
		if m.reqTab == id {
			return tabActive.Render(" " + label + " ")
		}
		return tabIdle.Render(" " + label + " ")
	}
	tabs := tab("Auth", reqTabAuth) + tab("Params", reqTabParams) +
		tab("Headers", reqTabHeaders) + tab("Body", reqTabBody)

	var content string
	switch m.reqTab {
	case reqTabAuth:
		content = m.viewAuthTab()
	case reqTabParams:
		content = viewKVTab(m.paramKeys, m.paramVals)
	case reqTabHeaders:
		content = viewKVTab(m.headerKeys, m.headerVals)
	default:
		content = m.bodyInput.View()
	}

	inner := lipgloss.JoinVertical(lipgloss.Left, reqLine, "", tabs, content)
	style := panelStyle.Width(m.mainW).Height(m.editorH)
	if m.focus == focusMethod || m.focus == focusURL ||
		m.focus == focusAuth || m.focus == focusParams ||
		m.focus == focusHeaders || m.focus == focusBody {
		style = focusedPanel.Width(m.mainW).Height(m.editorH)
	}
	return style.Render(inner)
}

func (m Model) viewResponse() string {
	meta := sectionLabel.Render("◆ RESPONSE")
	if m.resp.Status != "" {
		pill := statusChip(m.resp.StatusCode, m.resp.Error).Render(m.resp.Status)
		meta = pill + labelStyle.Render(fmt.Sprintf("  %dms", m.resp.Duration.Milliseconds()))
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
