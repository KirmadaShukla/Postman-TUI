package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"my-new-go/internal/models"
)

type namePromptKind int

const (
	namePromptNone namePromptKind = iota
	namePromptNewFolder
	namePromptRename
)

func (m *Model) openNewFolderPrompt() {
	col := m.activeCollection()
	if col == nil || m.insertParentChildren() == nil {
		m.status = "no collection"
		return
	}
	n := models.CountFolders(col.Items) + 1
	m.openNamePrompt(namePromptNewFolder, fmt.Sprintf("Folder %d", n), "New folder name")
}

func (m *Model) openRenamePrompt() {
	row, ok := m.cursorRow()
	if !ok || row.kind != rowTree {
		m.status = "select a folder or request to rename"
		return
	}
	item := itemAtMut(m.activeCollection(), row.path)
	if item == nil {
		return
	}
	title := "Rename request"
	if item.Kind == models.ItemFolder {
		title = "Rename folder"
	}
	m.openNamePrompt(namePromptRename, item.Name, title)
}

func (m *Model) openNamePrompt(kind namePromptKind, initial, title string) {
	ti := textinput.New()
	ti.Placeholder = "name"
	ti.CharLimit = 128
	ti.Width = 40
	ti.Prompt = ""
	ti.SetValue(initial)
	ti.CursorEnd()
	ti.Focus()

	m.nameReturnFocus = m.focus
	m.blurAll()
	m.nameInput = ti
	m.namePrompt = kind
	m.nameTitle = title
	m.showName = true
	m.status = title
}

func (m *Model) closeNamePrompt(commit bool) {
	kind := m.namePrompt
	name := strings.TrimSpace(m.nameInput.Value())
	ret := m.nameReturnFocus

	m.showName = false
	m.namePrompt = namePromptNone
	m.nameTitle = ""
	m.nameInput = textinput.Model{}

	if !commit {
		m.status = "cancelled"
		m.setFocus(ret)
		return
	}
	if name == "" {
		m.status = "name cannot be empty"
		m.setFocus(ret)
		return
	}

	switch kind {
	case namePromptNewFolder:
		m.createFolderNamed(name)
	case namePromptRename:
		m.renameCursorItem(name)
	}
	m.setFocus(ret)
}

func (m *Model) createFolderNamed(name string) {
	m.applyEditorToSelected()
	col := m.activeCollection()
	parent := m.insertParentChildren()
	if col == nil || parent == nil {
		return
	}
	folder := models.NewFolderItem(name)
	*parent = append(*parent, folder)
	m.setExpanded(folder.ID, true)
	m.selectPath(m.pathOfID(col.Items, folder.ID, nil))
	m.dirty = true
	m.status = "new folder: " + folder.Name
}

func (m *Model) renameCursorItem(name string) {
	row, ok := m.cursorRow()
	if !ok || row.kind != rowTree {
		return
	}
	item := itemAtMut(m.activeCollection(), row.path)
	if item == nil {
		return
	}
	item.Name = name
	if item.Kind == models.ItemRequest && item.Request != nil {
		item.Request.Name = name
	}
	m.dirty = true
	m.status = "renamed to " + name
}

func (m Model) handleNameKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.closeNamePrompt(false)
		return m, nil
	case "enter":
		m.closeNamePrompt(true)
		return m, nil
	}
	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, cmd
}

func (m Model) viewNameModal() string {
	var b strings.Builder
	title := m.nameTitle
	if title == "" {
		title = "Name"
	}
	b.WriteString(sectionLabel.Render("◆ "+title) + "\n\n")
	b.WriteString(m.nameInput.View() + "\n\n")
	b.WriteString(help(
		[2]string{"enter", "confirm"},
		[2]string{"esc", "cancel"},
	))

	modalW := min(56, max(36, m.width-10))
	box := modalStyle.Width(modalW).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}
