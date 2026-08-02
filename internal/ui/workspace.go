package ui

import (
	"fmt"

	"my-new-go/internal/models"
)

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

func (m *Model) activeCollectionIndex() int {
	for i := range m.ws.Collections {
		if m.ws.Collections[i].ID == m.ws.ActiveCollectionID {
			return i
		}
	}
	if len(m.ws.Collections) == 0 {
		return -1
	}
	return 0
}

func (m *Model) switchCollection(delta int) {
	n := len(m.ws.Collections)
	if n <= 1 {
		return
	}
	m.applyEditorToSelected()
	idx := m.activeCollectionIndex()
	if idx < 0 {
		return
	}
	idx = (idx + delta + n) % n
	m.ws.ActiveCollectionID = m.ws.Collections[idx].ID
	m.selectedIdx = 0
	m.loadSelectedIntoEditor()
	m.status = fmt.Sprintf("collection: %s", m.ws.Collections[idx].Name)
}

func (m *Model) newCollection() {
	m.applyEditorToSelected()
	name := fmt.Sprintf("Collection %d", len(m.ws.Collections)+1)
	col := models.Collection{
		ID:   models.NewID(),
		Name: name,
		Requests: []models.Request{
			{
				ID:     models.NewID(),
				Name:   "Request 1",
				Method: "GET",
				URL:    "{{base_url}}/",
				Headers: []models.Header{
					{Key: "Accept", Value: "application/json", Enabled: true},
				},
			},
		},
	}
	m.ws.Collections = append(m.ws.Collections, col)
	m.ws.ActiveCollectionID = col.ID
	m.selectedIdx = 0
	m.loadSelectedIntoEditor()
	m.dirty = true
	m.status = "new collection: " + name
}

func (m *Model) deleteActiveCollection() {
	if len(m.ws.Collections) <= 1 {
		m.status = "cannot delete the last collection"
		return
	}
	idx := m.activeCollectionIndex()
	if idx < 0 {
		return
	}
	name := m.ws.Collections[idx].Name
	m.ws.Collections = append(m.ws.Collections[:idx], m.ws.Collections[idx+1:]...)
	if idx >= len(m.ws.Collections) {
		idx = len(m.ws.Collections) - 1
	}
	m.ws.ActiveCollectionID = m.ws.Collections[idx].ID
	m.selectedIdx = 0
	m.loadSelectedIntoEditor()
	m.dirty = true
	_ = m.saveWorkspace()
	m.status = "deleted collection: " + name
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

func (m *Model) newRequest() {
	m.applyEditorToSelected()
	col := m.activeCollection()
	if col == nil {
		return
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
}

func (m *Model) deleteSelectedRequest() {
	col := m.activeCollection()
	if col == nil || len(col.Requests) == 0 {
		return
	}
	col.Requests = append(col.Requests[:m.selectedIdx], col.Requests[m.selectedIdx+1:]...)
	if m.selectedIdx >= len(col.Requests) {
		m.selectedIdx = len(col.Requests) - 1
	}
	m.loadSelectedIntoEditor()
	m.dirty = true
	_ = m.saveWorkspace()
	m.status = "deleted request"
}

func (m *Model) clearHistory() {
	if len(m.ws.History) == 0 {
		m.status = "history already empty"
		return
	}
	n := len(m.ws.History)
	m.ws.History = nil
	if err := m.store.Save(m.ws); err != nil {
		m.status = "clear history failed: " + err.Error()
		return
	}
	m.status = fmt.Sprintf("cleared %d history entries", n)
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
