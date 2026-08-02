package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m *Model) openEnvEditor() {
	env := m.activeEnv()
	if env == nil {
		m.status = "no environment"
		return
	}
	if env.Variables == nil {
		env.Variables = map[string]string{}
	}

	keys := make([]string, 0, len(env.Variables))
	for k := range env.Variables {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	m.envKeys = m.envKeys[:0]
	m.envVals = m.envVals[:0]
	for _, k := range keys {
		m.envKeys = append(m.envKeys, newEnvInput(k, 18))
		m.envVals = append(m.envVals, newEnvInput(env.Variables[k], 36))
	}
	m.envKeys = append(m.envKeys, newEnvInput("", 18))
	m.envVals = append(m.envVals, newEnvInput("", 36))

	m.envReturnFocus = m.focus
	m.blurAll()
	m.envKeys[0].Focus()
	m.showEnv = true
	m.status = fmt.Sprintf("editing env: %s", env.Name)
}

func (m *Model) closeEnvEditor(save bool) {
	if save {
		if err := m.applyEnvEditor(); err != nil {
			m.status = err.Error()
			return
		}
		m.status = "environment saved"
	} else {
		m.status = "environment edit cancelled"
	}
	ret := m.envReturnFocus
	m.showEnv = false
	m.envKeys = nil
	m.envVals = nil
	m.setFocus(ret)
}

func (m *Model) applyEnvEditor() error {
	env := m.activeEnv()
	if env == nil {
		return fmt.Errorf("no environment")
	}
	vars := map[string]string{}
	for i := range m.envKeys {
		k := strings.TrimSpace(m.envKeys[i].Value())
		if k == "" {
			continue
		}
		vars[k] = m.envVals[i].Value()
	}
	env.Variables = vars
	m.applyEditorToSelected()
	if err := m.store.Save(m.ws); err != nil {
		return fmt.Errorf("save failed: %w", err)
	}
	m.dirty = false
	return nil
}

func newEnvInput(val string, width int) textinput.Model {
	ti := textinput.New()
	ti.SetValue(val)
	ti.Prompt = ""
	ti.Width = width
	ti.CharLimit = 2048
	return ti
}

func (m *Model) blurAll() {
	m.urlInput.Blur()
	m.bodyInput.Blur()
	for i := range m.headerKeys {
		m.headerKeys[i].Blur()
		m.headerVals[i].Blur()
	}
	for i := range m.envKeys {
		m.envKeys[i].Blur()
		m.envVals[i].Blur()
	}
}

func (m *Model) envFocusedRow() (idx int, onKey bool) {
	idx = 0
	onKey = true
	for i := range m.envKeys {
		if m.envKeys[i].Focused() {
			return i, true
		}
		if m.envVals[i].Focused() {
			return i, false
		}
	}
	if len(m.envKeys) > 0 {
		m.envKeys[0].Focus()
	}
	return 0, true
}

func (m Model) handleEnvKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "esc":
		m.closeEnvEditor(false)
		return m, nil
	case "ctrl+s":
		m.closeEnvEditor(true)
		return m, nil
	case "ctrl+d":
		idx, _ := m.envFocusedRow()
		if len(m.envKeys) <= 1 {
			m.envKeys[0].SetValue("")
			m.envVals[0].SetValue("")
			m.envKeys[0].Focus()
			return m, nil
		}
		m.envKeys = append(m.envKeys[:idx], m.envKeys[idx+1:]...)
		m.envVals = append(m.envVals[:idx], m.envVals[idx+1:]...)
		if idx >= len(m.envKeys) {
			idx = len(m.envKeys) - 1
		}
		m.blurAll()
		m.envKeys[idx].Focus()
		return m, nil
	case "tab", "enter":
		idx, onKey := m.envFocusedRow()
		m.blurAll()
		if onKey {
			m.envVals[idx].Focus()
		} else {
			last := len(m.envKeys) - 1
			if strings.TrimSpace(m.envKeys[last].Value()) != "" || strings.TrimSpace(m.envVals[last].Value()) != "" {
				m.envKeys = append(m.envKeys, newEnvInput("", 18))
				m.envVals = append(m.envVals, newEnvInput("", 36))
			}
			next := idx + 1
			if next < len(m.envKeys) {
				m.envKeys[next].Focus()
			} else {
				m.envKeys[last].Focus()
			}
		}
		return m, nil
	case "shift+tab":
		idx, onKey := m.envFocusedRow()
		m.blurAll()
		if !onKey {
			m.envKeys[idx].Focus()
		} else if idx > 0 {
			m.envVals[idx-1].Focus()
		} else {
			m.envKeys[0].Focus()
		}
		return m, nil
	}

	return m.updateEnvFocused(msg)
}

func (m Model) updateEnvFocused(msg tea.Msg) (tea.Model, tea.Cmd) {
	if len(m.envKeys) == 0 {
		return m, nil
	}
	idx, onKey := m.envFocusedRow()
	var cmd tea.Cmd
	if onKey {
		m.envKeys[idx], cmd = m.envKeys[idx].Update(msg)
	} else {
		m.envVals[idx], cmd = m.envVals[idx].Update(msg)
	}
	return m, cmd
}

func (m Model) viewEnvModal() string {
	env := m.activeEnv()
	name := "Environment"
	if env != nil {
		name = env.Name
	}

	var b strings.Builder
	b.WriteString(sectionLabel.Render("◆ ENV · "+name) + "\n")
	b.WriteString(emptyHint.Render("use as {{name}} in URL / headers / body") + "\n\n")
	b.WriteString(labelStyle.Render("KEY"))
	b.WriteString(strings.Repeat(" ", 16))
	b.WriteString(labelStyle.Render("VALUE"))
	b.WriteString("\n")

	for i := range m.envKeys {
		b.WriteString(m.envKeys[i].View())
		b.WriteString("  ")
		b.WriteString(m.envVals[i].View())
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(help(
		[2]string{"ctrl+s", "save"},
		[2]string{"esc", "cancel"},
		[2]string{"enter", "next"},
		[2]string{"ctrl+d", "del row"},
	))

	modalW := min(72, max(40, m.width-8))
	box := modalStyle.Width(modalW).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceForeground(dim),
	)
}
