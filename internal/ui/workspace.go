package ui

import (
	"fmt"
	"net/http"
	"time"

	"my-new-go/internal/httpclient"
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
	m.resetSelectionToFirstRequest()
	m.loadSelectedIntoEditor()
	m.status = fmt.Sprintf("collection: %s", m.ws.Collections[idx].Name)
}

func (m *Model) resetSelectionToFirstRequest() {
	col := m.activeCollection()
	m.selectedPath = nil
	m.sidebarCursor = 0
	if col == nil {
		return
	}
	if p := m.firstRequestPath(col.Items, nil); p != nil {
		m.selectPath(p)
	} else if len(col.Items) > 0 {
		m.selectPath([]int{0})
	}
}

func (m *Model) newCollection() {
	m.applyEditorToSelected()
	name := fmt.Sprintf("Collection %d", len(m.ws.Collections)+1)
	req := models.Request{
		ID:     models.NewID(),
		Name:   "Request 1",
		Method: "GET",
		URL:    "{{base_url}}/",
		Headers: []models.Header{
			{Key: "Accept", Value: "application/json", Enabled: true},
		},
	}
	col := models.Collection{
		ID:    models.NewID(),
		Name:  name,
		Items: []models.Item{models.NewRequestItem(req)},
	}
	m.ws.Collections = append(m.ws.Collections, col)
	m.ws.ActiveCollectionID = col.ID
	m.resetSelectionToFirstRequest()
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
	m.resetSelectionToFirstRequest()
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
	if col == nil {
		return nil
	}
	item := itemAtMut(col, m.selectedPath)
	if item == nil || item.Kind != models.ItemRequest || item.Request == nil {
		return nil
	}
	return item.Request
}

func (m *Model) newRequest() {
	m.applyEditorToSelected()
	col := m.activeCollection()
	parent := m.insertParentChildren()
	if col == nil || parent == nil {
		return
	}
	n := models.CountRequests(col.Items) + 1
	req := models.Request{
		ID:     models.NewID(),
		Name:   fmt.Sprintf("Request %d", n),
		Method: "GET",
		URL:    "{{base_url}}/",
		Headers: []models.Header{
			{Key: "Accept", Value: "application/json", Enabled: true},
		},
	}
	*parent = append(*parent, models.NewRequestItem(req))
	// Find the path of the newly appended item.
	m.selectedPath = m.pathOfID(col.Items, req.ID, nil)
	m.selectPath(m.selectedPath)
	m.loadSelectedIntoEditor()
	m.dirty = true
	m.status = "new request"
}

func (m *Model) pathOfID(items []models.Item, id string, prefix []int) []int {
	for i, it := range items {
		p := append(copyPath(prefix), i)
		if it.ID == id {
			return p
		}
		if found := m.pathOfID(it.Children, id, p); found != nil {
			return found
		}
	}
	return nil
}

func (m *Model) deleteSelected() {
	row, ok := m.cursorRow()
	if !ok {
		return
	}
	if row.kind == rowHistory {
		m.deleteHistoryAt(row.histIdx)
		return
	}
	col := m.activeCollection()
	if col == nil {
		return
	}
	item := itemAtMut(col, row.path)
	if item == nil {
		return
	}
	wasRequest := item.Kind == models.ItemRequest
	name := item.Name
	keepID := ""
	if sel := m.selectedRequest(); sel != nil && !(wasRequest && pathsEqual(m.selectedPath, row.path)) {
		keepID = sel.ID
	}
	if !deleteAtPath(&col.Items, row.path) {
		return
	}
	if keepID != "" {
		m.selectedPath = m.pathOfID(col.Items, keepID, nil)
	} else {
		m.selectedPath = nil
	}
	if m.selectedPath == nil {
		m.resetSelectionToFirstRequest()
	} else {
		m.selectPath(m.selectedPath)
	}
	m.loadSelectedIntoEditor()
	m.dirty = true
	_ = m.saveWorkspace()
	if wasRequest {
		m.status = "deleted request"
	} else {
		m.status = "deleted folder: " + name
	}
}

func (m *Model) deleteHistoryAt(idx int) {
	if idx < 0 || idx >= len(m.ws.History) {
		return
	}
	m.ws.History = append(m.ws.History[:idx], m.ws.History[idx+1:]...)
	m.ensureSidebarCursor()
	if err := m.store.Save(m.ws); err != nil {
		m.status = "delete history failed: " + err.Error()
		return
	}
	m.status = "deleted history entry"
}

func (m *Model) clearHistory() {
	if len(m.ws.History) == 0 {
		m.status = "history already empty"
		return
	}
	n := len(m.ws.History)
	m.ws.History = nil
	m.ensureSidebarCursor()
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

func (m *Model) moveSidebar(delta int) {
	rows := m.sidebarRows()
	if len(rows) == 0 {
		return
	}
	m.ensureSidebarCursor()
	next := m.sidebarCursor + delta
	if next < 0 || next >= len(rows) {
		return
	}

	prev := rows[m.sidebarCursor]
	if prev.kind == rowTree {
		item := itemAtMut(m.activeCollection(), prev.path)
		if item != nil && item.Kind == models.ItemRequest {
			m.applyEditorToSelected()
		}
	}

	m.sidebarCursor = next
	cur := rows[m.sidebarCursor]
	if cur.kind == rowTree {
		item := itemAtMut(m.activeCollection(), cur.path)
		if item != nil && item.Kind == models.ItemRequest {
			m.selectedPath = copyPath(cur.path)
			m.loadSelectedIntoEditor()
		}
	}
}

func (m *Model) expandCollapse(open bool) {
	row, ok := m.cursorRow()
	if !ok || row.kind != rowTree || !row.folder {
		return
	}
	m.setExpanded(row.id, open)
}

func (m *Model) restoreHistory(idx int) {
	if idx < 0 || idx >= len(m.ws.History) {
		return
	}
	h := m.ws.History[idx]

	if m.selectedRequest() == nil {
		m.newRequest()
	}

	m.methodIdx = indexOfMethod(h.Method)
	m.urlInput.SetValue(h.URL)
	m.bodyInput.SetValue(h.RequestBody)
	if len(h.Headers) > 0 {
		m.syncHeadersFrom(h.Headers)
	} else {
		m.syncHeadersFrom(nil)
	}
	m.dirty = true

	status := fmt.Sprintf("%d", h.StatusCode)
	if h.StatusCode == 0 {
		status = ""
	}
	m.resp = httpclient.Result{
		StatusCode: h.StatusCode,
		Status:     status,
		Body:       h.ResponseBody,
		Duration:   time.Duration(h.DurationMS) * time.Millisecond,
		ResolvedURL: h.URL,
	}
	if status != "" {
		m.resp.Status = fmt.Sprintf("%d %s", h.StatusCode, http.StatusText(h.StatusCode))
	}
	m.respTab = 0
	m.refreshResponseView()
	m.status = "restored from history"
}
