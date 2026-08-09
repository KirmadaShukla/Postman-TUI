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
	focusAuth
	focusParams
	focusHeaders
	focusBody
	focusResponse
)

const (
	reqTabAuth = iota
	reqTabParams
	reqTabHeaders
	reqTabBody
)

var methods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD"}

var authTypes = []string{
	models.AuthNone,
	models.AuthBearer,
	models.AuthBasic,
	models.AuthAPIKey,
}

var authAddTo = []string{"header", "query"}

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

	sidebarCursor int
	selectedPath  []int // path to request bound to the editor
	expanded      map[string]bool

	methodIdx int
	reqTab    int // auth / params / headers / body
	respTab   int // 0 body, 1 headers

	urlInput   textinput.Model
	headerKeys []textinput.Model
	headerVals []textinput.Model
	headerEn   []bool
	paramKeys  []textinput.Model
	paramVals  []textinput.Model
	paramEn    []bool
	bodyInput  textarea.Model
	respView   viewport.Model

	authTypeIdx  int
	authAddToIdx int
	authField    int // 0=type, then type-specific fields
	authToken    textinput.Model
	authUser     textinput.Model
	authPass     textinput.Model
	authKey      textinput.Model
	authValue    textinput.Model

	// Environment editor modal (ctrl+e).
	showEnv        bool
	envKeys        []textinput.Model
	envVals        []textinput.Model
	envReturnFocus focusArea

	// Name prompt modal (new folder / rename).
	showName        bool
	namePrompt      namePromptKind
	nameTitle       string
	nameInput       textinput.Model
	nameReturnFocus focusArea

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
		expanded:  map[string]bool{},
		authToken: newAuthInput("", 48),
		authUser:  newAuthInput("", 32),
		authPass:  newAuthInput("", 32),
		authKey:   newAuthInput("X-API-Key", 20),
		authValue: newAuthInput("", 40),
		status:    fmt.Sprintf("workspace: %s", st.Path()),
	}
	m.authPass.EchoMode = textinput.EchoPassword
	m.authPass.EchoCharacter = '•'
	m.resetSelectionToFirstRequest()
	m.loadSelectedIntoEditor()
	m.setFocus(focusURL)
	return m
}

func newAuthInput(val string, width int) textinput.Model {
	ti := textinput.New()
	ti.SetValue(val)
	ti.Prompt = ""
	ti.Width = width
	ti.CharLimit = 4096
	return ti
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}
