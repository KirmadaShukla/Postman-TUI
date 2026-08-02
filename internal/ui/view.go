package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if !m.ready {
		return "loading..."
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
		[2]string{"enter", "send"},
		[2]string{"tab", "focus"},
		[2]string{"[/]", "collection"},
		[2]string{"ctrl+e", "env"},
		[2]string{"ctrl+n", "new req"},
		[2]string{"ctrl+o", "new col"},
		[2]string{"ctrl+d", "del req"},
		[2]string{"ctrl+y", "clear hist"},
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
		b.WriteString(emptyHint.Render("[ ] switch  ctrl+o new") + "\n")
	} else {
		b.WriteString(emptyHint.Render("ctrl+o new collection") + "\n")
	}
	if col == nil || len(col.Requests) == 0 {
		b.WriteString(emptyHint.Render("empty — ctrl+n to add") + "\n")
	} else {
		for i, req := range col.Requests {
			method := req.Method
			prefix := "  "
			ms := methodStyle(method)
			badge := ms.Render(fmt.Sprintf("%-4s", method))
			name := truncate(req.Name, innerW-9)
			line := prefix + badge + " " + name
			if i == m.selectedIdx {
				line = "▸ " + badge + " " + name
				pad := innerW - lipgloss.Width(line)
				if pad > 0 {
					line += strings.Repeat(" ", pad)
				}
				b.WriteString(listSel.Render(line) + "\n")
			} else {
				b.WriteString(listItem.Render(line) + "\n")
			}
		}
	}

	b.WriteString("\n" + sectionLabel.Render("◷ HISTORY") + "\n")
	if len(m.ws.History) == 0 {
		b.WriteString(emptyHint.Render("no requests sent yet") + "\n")
	}
	limit := 8
	if len(m.ws.History) < limit {
		limit = len(m.ws.History)
	}
	for i := 0; i < limit; i++ {
		h := m.ws.History[i]
		code := statusChip(h.StatusCode, "").Render(fmt.Sprintf("%d", h.StatusCode))
		method := labelStyle.Render(fmt.Sprintf("%-4s", h.Method))
		line := method + code + " " + truncate(h.URL, innerW-14)
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
		methodBox = methodBoxFocused.Render(methodBox)
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
		hb.WriteString(labelStyle.Render("KEY"))
		hb.WriteString(strings.Repeat(" ", 18))
		hb.WriteString(labelStyle.Render("VALUE"))
		hb.WriteString("\n")
		for i := range m.headerKeys {
			hb.WriteString(m.headerKeys[i].View())
			hb.WriteString("  ")
			hb.WriteString(m.headerVals[i].View())
			hb.WriteString("\n")
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
