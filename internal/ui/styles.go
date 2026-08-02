package ui

import "github.com/charmbracelet/lipgloss"

var (
	accent   = lipgloss.Color("#3DDC97")
	muted    = lipgloss.Color("#7A8494")
	danger   = lipgloss.Color("#FF6B6B")
	border   = lipgloss.Color("#2E3642")
	focusCol = lipgloss.Color("#5B9CFF")
	textCol  = lipgloss.Color("#E8EDF2")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accent).
			Padding(0, 1)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(border).
			Padding(0, 1)

	focusedPanel = panelStyle.BorderForeground(focusCol)

	labelStyle = lipgloss.NewStyle().Foreground(muted)
	valueStyle = lipgloss.NewStyle().Foreground(textCol)

	methodStyles = map[string]lipgloss.Style{
		"GET":    lipgloss.NewStyle().Foreground(lipgloss.Color("#61AFEF")).Bold(true),
		"POST":   lipgloss.NewStyle().Foreground(lipgloss.Color("#98C379")).Bold(true),
		"PUT":    lipgloss.NewStyle().Foreground(lipgloss.Color("#E5C07B")).Bold(true),
		"PATCH":  lipgloss.NewStyle().Foreground(lipgloss.Color("#C678DD")).Bold(true),
		"DELETE": lipgloss.NewStyle().Foreground(danger).Bold(true),
		"HEAD":   lipgloss.NewStyle().Foreground(muted).Bold(true),
	}

	statusOK = lipgloss.NewStyle().Foreground(accent).Bold(true)
	statusBad = lipgloss.NewStyle().Foreground(danger).Bold(true)

	helpStyle = lipgloss.NewStyle().Foreground(muted)
	listItem  = lipgloss.NewStyle().PaddingLeft(1)
	listSel   = lipgloss.NewStyle().PaddingLeft(1).Foreground(accent).Bold(true)
	tabActive = lipgloss.NewStyle().Foreground(accent).Bold(true).Underline(true)
	tabIdle   = lipgloss.NewStyle().Foreground(muted)
)
