package main

import "github.com/charmbracelet/lipgloss"

// One accent, three greys, two status colours. Everything else is weight and
// space. Adaptive so the TUI is legible on light and dark terminals without a
// config file.
var (
	accent = lipgloss.AdaptiveColor{Light: "#0b7285", Dark: "#38d9a9"}
	fg     = lipgloss.AdaptiveColor{Light: "#212529", Dark: "#e9ecef"}
	dim    = lipgloss.AdaptiveColor{Light: "#868e96", Dark: "#6c757d"}
	faint  = lipgloss.AdaptiveColor{Light: "#adb5bd", Dark: "#495057"}
	rule   = lipgloss.AdaptiveColor{Light: "#dee2e6", Dark: "#343a40"}
	danger = lipgloss.AdaptiveColor{Light: "#c92a2a", Dark: "#ff8787"}
	warnc  = lipgloss.AdaptiveColor{Light: "#e67700", Dark: "#ffd43b"}

	sBase   = lipgloss.NewStyle().Foreground(fg)
	sDim    = lipgloss.NewStyle().Foreground(dim)
	sFaint  = lipgloss.NewStyle().Foreground(faint)
	sAccent = lipgloss.NewStyle().Foreground(accent)
	sErr    = lipgloss.NewStyle().Foreground(danger)
	sWarn   = lipgloss.NewStyle().Foreground(warnc)

	// Column headers: the only uppercase in the app, so they read as furniture
	// rather than content.
	sHead = lipgloss.NewStyle().Foreground(accent).Bold(true)

	sTitle = lipgloss.NewStyle().Foreground(fg).Bold(true)

	// A selected row gets an accent bar and bold text, never a full-width
	// background — background fills fight with terminal themes.
	sCursor   = lipgloss.NewStyle().Foreground(accent).Bold(true)
	sSelected = lipgloss.NewStyle().Foreground(fg).Bold(true)

	sVRule = lipgloss.NewStyle().Foreground(rule)
	sKey   = lipgloss.NewStyle().Foreground(accent)
	sHelp  = lipgloss.NewStyle().Foreground(dim)
)

// bar is the left-hand row marker. Always two cells, so text never shifts
// horizontally when the cursor moves. Three states, because a selection in an
// unfocused pane still has to be findable.
func bar(selected, focused bool) string {
	switch {
	case selected && focused:
		return sAccent.Render("▌ ")
	case selected:
		return sFaint.Render("▌ ")
	}
	return "  "
}
