package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"my-new-go/internal/models"
)

func (m *Model) blurAuthInputs() {
	m.authToken.Blur()
	m.authUser.Blur()
	m.authPass.Blur()
	m.authKey.Blur()
	m.authValue.Blur()
}

func (m *Model) focusAuthField() {
	m.blurAuthInputs()
	fields := m.authFieldCount()
	if m.authField < 0 {
		m.authField = 0
	}
	if m.authField >= fields {
		m.authField = fields - 1
	}
	if m.authField == 0 {
		return // type selector
	}
	switch authTypes[m.authTypeIdx] {
	case models.AuthBearer:
		m.authToken.Focus()
	case models.AuthBasic:
		if m.authField == 1 {
			m.authUser.Focus()
		} else {
			m.authPass.Focus()
		}
	case models.AuthAPIKey:
		switch m.authField {
		case 1:
			m.authKey.Focus()
		case 2:
			m.authValue.Focus()
		}
	}
}

func (m *Model) authFieldCount() int {
	switch authTypes[m.authTypeIdx] {
	case models.AuthNone:
		return 1
	case models.AuthBearer:
		return 2 // type, token
	case models.AuthBasic:
		return 3 // type, user, pass
	case models.AuthAPIKey:
		return 4 // type, key, value, add-to
	default:
		return 1
	}
}

func (m Model) updateAuth(msg tea.Msg) (Model, tea.Cmd) {
	keyMsg, isKey := msg.(tea.KeyMsg)
	if isKey {
		switch keyMsg.String() {
		case "left", "h":
			if m.authField == 0 {
				m.authTypeIdx = (m.authTypeIdx - 1 + len(authTypes)) % len(authTypes)
				m.authField = 0
				m.focusAuthField()
				m.dirty = true
				return m, nil
			}
			if authTypes[m.authTypeIdx] == models.AuthAPIKey && m.authField == 3 {
				m.authAddToIdx = (m.authAddToIdx - 1 + len(authAddTo)) % len(authAddTo)
				m.dirty = true
				return m, nil
			}
		case "right", "l":
			if m.authField == 0 {
				m.authTypeIdx = (m.authTypeIdx + 1) % len(authTypes)
				m.authField = 0
				m.focusAuthField()
				m.dirty = true
				return m, nil
			}
			if authTypes[m.authTypeIdx] == models.AuthAPIKey && m.authField == 3 {
				m.authAddToIdx = (m.authAddToIdx + 1) % len(authAddTo)
				m.dirty = true
				return m, nil
			}
		case "up", "k":
			if m.authField > 0 {
				m.authField--
				m.focusAuthField()
				return m, nil
			}
		case "down", "j", "enter":
			if m.authField < m.authFieldCount()-1 {
				m.authField++
				m.focusAuthField()
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	switch {
	case m.authToken.Focused():
		m.authToken, cmd = m.authToken.Update(msg)
		m.dirty = true
	case m.authUser.Focused():
		m.authUser, cmd = m.authUser.Update(msg)
		m.dirty = true
	case m.authPass.Focused():
		m.authPass, cmd = m.authPass.Update(msg)
		m.dirty = true
	case m.authKey.Focused():
		m.authKey, cmd = m.authKey.Update(msg)
		m.dirty = true
	case m.authValue.Focused():
		m.authValue, cmd = m.authValue.Update(msg)
		m.dirty = true
	}
	return m, cmd
}

func (m Model) viewAuthTab() string {
	var b strings.Builder
	b.WriteString(labelStyle.Render("TYPE"))
	b.WriteString("  ")
	typeLabel := authTypeLabel(authTypes[m.authTypeIdx])
	if m.focus == focusAuth && m.authField == 0 {
		typeLabel = methodBoxFocused.Render(" ◂ " + typeLabel + " ▸ ")
	} else {
		typeLabel = valueStyle.Render(typeLabel)
	}
	b.WriteString(typeLabel)
	b.WriteString("\n")
	b.WriteString(emptyHint.Render("← → change type") + "\n\n")

	switch authTypes[m.authTypeIdx] {
	case models.AuthNone:
		b.WriteString(emptyHint.Render("no auth will be sent"))
	case models.AuthBearer:
		b.WriteString(labelStyle.Render("TOKEN") + "\n")
		b.WriteString(m.authToken.View())
	case models.AuthBasic:
		b.WriteString(labelStyle.Render("USERNAME") + "\n")
		b.WriteString(m.authUser.View() + "\n")
		b.WriteString(labelStyle.Render("PASSWORD") + "\n")
		b.WriteString(m.authPass.View())
	case models.AuthAPIKey:
		b.WriteString(labelStyle.Render("KEY") + "\n")
		b.WriteString(m.authKey.View() + "\n")
		b.WriteString(labelStyle.Render("VALUE") + "\n")
		b.WriteString(m.authValue.View() + "\n")
		addTo := authAddTo[m.authAddToIdx]
		if m.focus == focusAuth && m.authField == 3 {
			addTo = methodBoxFocused.Render(" ◂ " + addTo + " ▸ ")
		} else {
			addTo = valueStyle.Render(addTo)
		}
		b.WriteString(labelStyle.Render("ADD TO") + "  " + addTo)
	}
	return b.String()
}

func authTypeLabel(t string) string {
	switch t {
	case models.AuthBearer:
		return "Bearer Token"
	case models.AuthBasic:
		return "Basic Auth"
	case models.AuthAPIKey:
		return "API Key"
	default:
		return "No Auth"
	}
}

func (m *Model) updateKV(msg tea.Msg, keys, vals *[]textinput.Model, ens *[]bool) tea.Cmd {
	if len(*keys) == 0 {
		return nil
	}
	var cmd tea.Cmd
	idx := len(*keys) - 1
	for i := range *keys {
		if (*keys)[i].Focused() {
			idx = i
			break
		}
	}
	if !(*keys)[idx].Focused() && !(*vals)[idx].Focused() {
		(*keys)[idx].Focus()
	}
	if (*keys)[idx].Focused() {
		(*keys)[idx], cmd = (*keys)[idx].Update(msg)
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
			(*keys)[idx].Blur()
			(*vals)[idx].Focus()
		}
	} else {
		(*vals)[idx], cmd = (*vals)[idx].Update(msg)
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
			(*vals)[idx].Blur()
			last := len(*keys) - 1
			if strings.TrimSpace((*keys)[last].Value()) != "" || strings.TrimSpace((*vals)[last].Value()) != "" {
				*keys = append(*keys, newHeaderInput("", 20))
				*vals = append(*vals, newHeaderInput("", 40))
				*ens = append(*ens, true)
			}
			next := idx + 1
			if next < len(*keys) {
				(*keys)[next].Focus()
			}
		}
	}
	m.dirty = true
	return cmd
}

func viewKVTab(keys, vals []textinput.Model) string {
	var hb strings.Builder
	hb.WriteString(labelStyle.Render("KEY"))
	hb.WriteString(strings.Repeat(" ", 18))
	hb.WriteString(labelStyle.Render("VALUE"))
	hb.WriteString("\n")
	for i := range keys {
		hb.WriteString(keys[i].View())
		hb.WriteString("  ")
		hb.WriteString(vals[i].View())
		hb.WriteString("\n")
	}
	return hb.String()
}
