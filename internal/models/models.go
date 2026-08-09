package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	ItemFolder  = "folder"
	ItemRequest = "request"
)

type Header struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}

const (
	AuthNone   = "none"
	AuthBearer = "bearer"
	AuthBasic  = "basic"
	AuthAPIKey = "apikey"
)

// Auth is per-request authentication (applied at send time).
type Auth struct {
	Type     string `json:"type"`               // none | bearer | basic | apikey
	Token    string `json:"token,omitempty"`    // bearer token
	Username string `json:"username,omitempty"` // basic
	Password string `json:"password,omitempty"` // basic
	Key      string `json:"key,omitempty"`      // apikey name
	Value    string `json:"value,omitempty"`    // apikey value
	AddTo    string `json:"add_to,omitempty"`   // header | query
}

type Request struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Method  string   `json:"method"`
	URL     string   `json:"url"`
	Auth    Auth     `json:"auth"`
	Params  []Header `json:"params,omitempty"` // query params
	Headers []Header `json:"headers"`
	Body    string   `json:"body"`
}

// Item is a folder or request node in a collection tree.
type Item struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Kind     string   `json:"kind"` // ItemFolder | ItemRequest
	Request  *Request `json:"request,omitempty"`
	Children []Item   `json:"children,omitempty"`
}

type Collection struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Items    []Item    `json:"items"`
	Requests []Request `json:"requests,omitempty"` // legacy flat list; migrated on load
}

type Environment struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Variables map[string]string `json:"variables"`
}

type HistoryEntry struct {
	ID           string    `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	Method       string    `json:"method"`
	URL          string    `json:"url"`
	StatusCode   int       `json:"status_code"`
	DurationMS   int64     `json:"duration_ms"`
	Headers      []Header  `json:"headers,omitempty"`
	RequestBody  string    `json:"request_body,omitempty"`
	ResponseBody string    `json:"response_body,omitempty"`
}

type Workspace struct {
	ActiveCollectionID  string         `json:"active_collection_id"`
	ActiveEnvironmentID string         `json:"active_environment_id"`
	Collections         []Collection   `json:"collections"`
	Environments        []Environment  `json:"environments"`
	History             []HistoryEntry `json:"history"`
}

func NewID() string {
	return uuid.NewString()
}

func NewRequestItem(req Request) Item {
	if req.ID == "" {
		req.ID = NewID()
	}
	return Item{
		ID:      req.ID,
		Name:    req.Name,
		Kind:    ItemRequest,
		Request: &req,
	}
}

func NewFolderItem(name string) Item {
	return Item{
		ID:       NewID(),
		Name:     name,
		Kind:     ItemFolder,
		Children: []Item{},
	}
}

// MigrateWorkspace lifts legacy flat Collection.Requests into Items.
func MigrateWorkspace(ws *Workspace) {
	for i := range ws.Collections {
		MigrateCollection(&ws.Collections[i])
	}
	if ws.History == nil {
		ws.History = []HistoryEntry{}
	}
}

func MigrateCollection(col *Collection) {
	if len(col.Items) == 0 && len(col.Requests) > 0 {
		col.Items = make([]Item, 0, len(col.Requests))
		for _, req := range col.Requests {
			r := req
			col.Items = append(col.Items, NewRequestItem(r))
		}
	}
	col.Requests = nil
	normalizeItems(col.Items)
}

func normalizeItems(items []Item) {
	for i := range items {
		if items[i].Kind == "" {
			if items[i].Request != nil {
				items[i].Kind = ItemRequest
			} else {
				items[i].Kind = ItemFolder
			}
		}
		if items[i].Kind == ItemFolder && items[i].Children == nil {
			items[i].Children = []Item{}
		}
		if items[i].Kind == ItemRequest && items[i].Request != nil && items[i].Name == "" {
			items[i].Name = items[i].Request.Name
		}
		normalizeItems(items[i].Children)
	}
}

func CountRequests(items []Item) int {
	n := 0
	for _, it := range items {
		if it.Kind == ItemRequest {
			n++
		}
		n += CountRequests(it.Children)
	}
	return n
}

func CountFolders(items []Item) int {
	n := 0
	for _, it := range items {
		if it.Kind == ItemFolder {
			n++
			n += CountFolders(it.Children)
		}
	}
	return n
}

func DefaultWorkspace() Workspace {
	colID := NewID()
	envID := NewID()
	reqID := NewID()

	req := Request{
		ID:     reqID,
		Name:   "Get example",
		Method: "GET",
		URL:    "https://httpbin.org/get",
		Headers: []Header{
			{Key: "Accept", Value: "application/json", Enabled: true},
		},
		Body: "",
	}

	return Workspace{
		ActiveCollectionID:  colID,
		ActiveEnvironmentID: envID,
		Collections: []Collection{
			{
				ID:   colID,
				Name: "Default",
				Items: []Item{
					NewRequestItem(req),
				},
			},
		},
		Environments: []Environment{
			{
				ID:   envID,
				Name: "Local",
				Variables: map[string]string{
					"base_url": "https://httpbin.org",
					"token":    "",
				},
			},
		},
		History: []HistoryEntry{},
	}
}
