package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"my-new-go/internal/httpclient"
	"my-new-go/internal/models"
	"my-new-go/internal/store"
)

type focusArea int

const (
	focusSidebar focusArea = iota
	focusMethod
	focusURL
	focusHeaders
	focusBody
	focusResponse
)

var methods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD"}

type sendDoneMsg struct {
	result httpclient.Result
	req    models.Request
}

// Model is the root Bubble Tea model for the API TUI.
type Model struct {
	store *store.Store
	ws    models.Workspace

	width  int
	height int
	focus  focusArea
	ready  bool

	sidebarW int
	mainW    int
	editorH  int
	respH    int

	selectedIdx int
	methodIdx   int
	reqTab      int // 0 headers, 1 body
	respTab     int // 0 body, 1 headers

	urlInput   textinput.Model
	headerKeys []textinput.Model
	headerVals []textinput.Model
	headerEn   []bool
	bodyInput  textarea.Model
	respView   viewport.Model

	// Environment editor modal (ctrl+e).
	showEnv        bool
	envKeys        []textinput.Model
	envVals        []textinput.Model
	envReturnFocus focusArea

	sending bool
	status  string
	resp    httpclient.Result
	dirty   bool
}

func New(st *store.Store, ws models.Workspace) Model {
	url := textinput.New()
	url.Placeholder = "Enter URL here"
	url.CharLimit = 2048
	url.Prompt = ""
	url.Width = 40

	body := textarea.New()
	body.Placeholder = "Request body (JSON, form, text...)"
	body.SetHeight(8)
	body.ShowLineNumbers = false
	body.CharLimit = 1 << 20

	resp := viewport.New(40, 10)
	resp.SetContent("Send a request to see the response.")

	m := Model{
		store:     st,
		ws:        ws,
		urlInput:  url,
		bodyInput: body,
		respView:  resp,
		status:    fmt.Sprintf("workspace: %s", st.Path()),
	}
	m.loadSelectedIntoEditor()
	m.setFocus(focusURL)
	return m
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}
