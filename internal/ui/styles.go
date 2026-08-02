package ui

import "github.com/charmbracelet/lipgloss"

// ── Palette ─────────────────────────────────────────────────────────────
// A calm dark theme: cool slate surfaces, a minty accent for primary
// actions/success, warm amber for caution, and coral for danger. Chosen so
// method badges and status codes each get a distinct, legible color.
var (
	bg       = lipgloss.Color("#11151C") // app background (terminal default assumed dark)
	surface  = lipgloss.Color("#1A2029") // panel fill / chip background
	surface2 = lipgloss.Color("#212836") // slightly raised surface (active tab, selected row)
	border   = lipgloss.Color("#2B3240")
	borderLo = lipgloss.Color("#232833")

	accent   = lipgloss.Color("#5EEAD4") // mint — primary accent / 2xx
	accentDk = lipgloss.Color("#0F3D36") // dark mint for chip backgrounds
	focusCol = lipgloss.Color("#7DAFFF") // soft blue — focus ring
	focusDk  = lipgloss.Color("#16233F")

	textCol = lipgloss.Color("#E7ECF3")
	muted   = lipgloss.Color("#7C8898")
	dim     = lipgloss.Color("#4B5666")

	warn     = lipgloss.Color("#FFC978")
	warnDk   = lipgloss.Color("#3D2E12")
	danger   = lipgloss.Color("#FF7A93")
	dangerDk = lipgloss.Color("#3B1620")
)

// ── Chrome ──────────────────────────────────────────────────────────────

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(bg).
			Background(accent).
			Padding(0, 2)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(muted).
			Background(surface).
			Padding(0, 1).
			MarginLeft(1)

	dirtyBadge = lipgloss.NewStyle().
			Foreground(warn).
			Background(warnDk).
			Bold(true).
			Padding(0, 1).
			MarginLeft(1)

	headerBar = lipgloss.NewStyle().
			MarginBottom(1)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(border).
			Padding(0, 1)

	focusedPanel = lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(focusCol).
			Padding(0, 1)

	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(accent).
			Background(surface).
			Padding(1, 2)

	panelTitle = lipgloss.NewStyle().
			Foreground(muted).
			Bold(true)

	labelStyle = lipgloss.NewStyle().Foreground(muted)
	valueStyle = lipgloss.NewStyle().Foreground(textCol)
	dimStyle   = lipgloss.NewStyle().Foreground(dim)
)

// ── Method badges ───────────────────────────────────────────────────────
// Rendered as small filled "chips" (background + bold foreground) rather
// than plain colored text, so the request list reads like a scannable
// table instead of a wall of text.

var methodStyles = map[string]lipgloss.Style{
	"GET":    chip("#61AFEF", "#132A3E"),
	"POST":   chip("#98C379", "#1C2E1B"),
	"PUT":    chip("#E5C07B", "#332A12"),
	"PATCH":  chip("#C678DD", "#2A1C33"),
	"DELETE": chip(string(danger), string(dangerDk)),
	"HEAD":   chip(string(muted), string(surface)),
}

func chip(fg, bgc string) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(fg)).
		Background(lipgloss.Color(bgc)).
		Bold(true).
		Padding(0, 1)
}

var methodBoxFocused = lipgloss.NewStyle().
	Border(lipgloss.ThickBorder()).
	BorderForeground(focusCol).
	Padding(0)

// ── Status / response ───────────────────────────────────────────────────

var (
	statusOK   = lipgloss.NewStyle().Foreground(accent).Bold(true)
	statusBad  = lipgloss.NewStyle().Foreground(danger).Bold(true)
	statusWarn = lipgloss.NewStyle().Foreground(warn).Bold(true)

	statusPill = lipgloss.NewStyle().Padding(0, 1).Bold(true)
)

// statusChip picks a color for an HTTP status code the way most API
// clients do: 2xx mint, 3xx blue, 4xx amber, 5xx coral.
func statusChip(code int, errText string) lipgloss.Style {
	switch {
	case errText != "":
		return statusPill.Foreground(danger).Background(dangerDk)
	case code >= 500:
		return statusPill.Foreground(danger).Background(dangerDk)
	case code >= 400:
		return statusPill.Foreground(warn).Background(warnDk)
	case code >= 300:
		return statusPill.Foreground(focusCol).Background(focusDk)
	case code >= 200:
		return statusPill.Foreground(accent).Background(accentDk)
	default:
		return statusPill.Foreground(muted).Background(surface)
	}
}

// ── Sidebar / lists ─────────────────────────────────────────────────────

var (
	sectionLabel = lipgloss.NewStyle().
			Foreground(dim).
			Bold(true)

	listItem = lipgloss.NewStyle().
			PaddingLeft(1).
			Foreground(textCol)

	listSel = lipgloss.NewStyle().
		PaddingLeft(1).
		Foreground(bg).
		Background(surface2).
		Bold(true)

	emptyHint = lipgloss.NewStyle().
			Foreground(dim).
			Italic(true).
			PaddingLeft(1)
)

// ── Tabs ────────────────────────────────────────────────────────────────

var (
	tabIdle = lipgloss.NewStyle().
		Foreground(muted).
		Padding(0, 2)

	tabActive = lipgloss.NewStyle().
			Foreground(bg).
			Background(accent).
			Bold(true).
			Padding(0, 2)
)

// ── Footer / help ───────────────────────────────────────────────────────

var (
	helpStyle = lipgloss.NewStyle().Foreground(dim)
	helpKey   = lipgloss.NewStyle().Foreground(focusCol).Bold(true)

	statusBarSending = lipgloss.NewStyle().Foreground(warn).Bold(true)
	statusBarSaved   = lipgloss.NewStyle().Foreground(accent)
	statusBarError   = lipgloss.NewStyle().Foreground(danger).Bold(true)
	statusBarIdle    = lipgloss.NewStyle().Foreground(muted)
)

// help renders the footer hint bar with the keys highlighted, e.g.
// "enter send · tab focus" but with the key tokens in accent color.
func help(pairs ...[2]string) string {
	parts := make([]string, 0, len(pairs))
	for _, p := range pairs {
		parts = append(parts, helpKey.Render(p[0])+helpStyle.Render(" "+p[1]))
	}
	sep := helpStyle.Render("  ·  ")
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += sep
		}
		out += p
	}
	return out
}
