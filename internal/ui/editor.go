package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"

	"my-new-go/internal/models"
)

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
