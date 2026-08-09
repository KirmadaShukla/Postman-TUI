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
		m.syncHeadersFrom(nil)
		m.syncParamsFrom(nil)
		m.syncAuthFrom(models.Auth{Type: models.AuthNone})
		m.methodIdx = 0
		return
	}

	m.urlInput.SetValue(req.URL)
	m.bodyInput.SetValue(req.Body)
	m.methodIdx = indexOfMethod(req.Method)
	m.syncHeadersFrom(req.Headers)
	m.syncParamsFrom(req.Params)
	m.syncAuthFrom(req.Auth)
	m.dirty = false
}

func (m *Model) syncHeadersFrom(headers []models.Header) {
	m.headerKeys, m.headerVals, m.headerEn = syncKVFrom(headers)
}

func (m *Model) syncParamsFrom(params []models.Header) {
	m.paramKeys, m.paramVals, m.paramEn = syncKVFrom(params)
}

func syncKVFrom(rows []models.Header) (keys, vals []textinput.Model, ens []bool) {
	keys = make([]textinput.Model, 0, len(rows)+1)
	vals = make([]textinput.Model, 0, len(rows)+1)
	ens = make([]bool, 0, len(rows)+1)
	for _, h := range rows {
		keys = append(keys, newHeaderInput(h.Key, 20))
		vals = append(vals, newHeaderInput(h.Value, 40))
		ens = append(ens, h.Enabled)
	}
	keys = append(keys, newHeaderInput("", 20))
	vals = append(vals, newHeaderInput("", 40))
	ens = append(ens, true)
	return keys, vals, ens
}

func collectKV(keys, vals []textinput.Model, ens []bool) []models.Header {
	var out []models.Header
	for i := range keys {
		k := strings.TrimSpace(keys[i].Value())
		v := vals[i].Value()
		if k == "" && strings.TrimSpace(v) == "" {
			continue
		}
		en := true
		if i < len(ens) {
			en = ens[i]
		}
		out = append(out, models.Header{Key: k, Value: v, Enabled: en})
	}
	return out
}

func (m *Model) syncAuthFrom(a models.Auth) {
	if a.Type == "" {
		a.Type = models.AuthNone
	}
	m.authTypeIdx = indexOfAuth(a.Type)
	m.authToken.SetValue(a.Token)
	m.authUser.SetValue(a.Username)
	m.authPass.SetValue(a.Password)
	key := a.Key
	if key == "" {
		key = "X-API-Key"
	}
	m.authKey.SetValue(key)
	m.authValue.SetValue(a.Value)
	m.authAddToIdx = 0
	if strings.ToLower(a.AddTo) == "query" {
		m.authAddToIdx = 1
	}
	m.authField = 0
}

func (m *Model) collectAuth() models.Auth {
	a := models.Auth{Type: authTypes[m.authTypeIdx]}
	switch a.Type {
	case models.AuthBearer:
		a.Token = m.authToken.Value()
	case models.AuthBasic:
		a.Username = m.authUser.Value()
		a.Password = m.authPass.Value()
	case models.AuthAPIKey:
		a.Key = m.authKey.Value()
		a.Value = m.authValue.Value()
		a.AddTo = authAddTo[m.authAddToIdx]
	}
	return a
}

func indexOfAuth(t string) int {
	t = strings.ToLower(strings.TrimSpace(t))
	for i, a := range authTypes {
		if a == t {
			return i
		}
	}
	return 0
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
		Method:  methods[m.methodIdx],
		URL:     m.urlInput.Value(),
		Body:    m.bodyInput.Value(),
		Auth:    m.collectAuth(),
		Params:  collectKV(m.paramKeys, m.paramVals, m.paramEn),
		Headers: collectKV(m.headerKeys, m.headerVals, m.headerEn),
	}
	if cur := m.selectedRequest(); cur != nil {
		req.ID = cur.ID
		req.Name = cur.Name
	} else {
		req.ID = models.NewID()
		req.Name = "Untitled"
	}
	return req
}

func (m *Model) applyEditorToSelected() {
	req := m.collectEditorRequest()
	col := m.activeCollection()
	if col == nil {
		return
	}
	item := itemAtMut(col, m.selectedPath)
	if item != nil && item.Kind == models.ItemRequest {
		cp := req
		item.Request = &cp
		item.Name = req.Name
		item.ID = req.ID
		return
	}
	// No request selected — append at root and select it.
	col.Items = append(col.Items, models.NewRequestItem(req))
	m.selectedPath = []int{len(col.Items) - 1}
	m.selectPath(m.selectedPath)
}
