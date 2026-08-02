package models

import (
	"time"

	"github.com/google/uuid"
)

type Header struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}

type Request struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Method  string   `json:"method"`
	URL     string   `json:"url"`
	Headers []Header `json:"headers"`
	Body    string   `json:"body"`
}

type Collection struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Requests []Request `json:"requests"`
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
	RequestBody  string    `json:"request_body,omitempty"`
	ResponseBody string    `json:"response_body,omitempty"`
}

type Workspace struct {
	ActiveCollectionID string         `json:"active_collection_id"`
	ActiveEnvironmentID string        `json:"active_environment_id"`
	Collections        []Collection   `json:"collections"`
	Environments       []Environment  `json:"environments"`
	History            []HistoryEntry `json:"history"`
}

func NewID() string {
	return uuid.NewString()
}

func DefaultWorkspace() Workspace {
	colID := NewID()
	envID := NewID()
	reqID := NewID()

	return Workspace{
		ActiveCollectionID:  colID,
		ActiveEnvironmentID: envID,
		Collections: []Collection{
			{
				ID:   colID,
				Name: "Default",
				Requests: []Request{
					{
						ID:     reqID,
						Name:   "Get example",
						Method: "GET",
						URL:    "https://httpbin.org/get",
						Headers: []Header{
							{Key: "Accept", Value: "application/json", Enabled: true},
						},
						Body: "",
					},
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
