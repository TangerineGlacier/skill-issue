package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// binding is one key and what it does. Every screen's footer and the ? overlay
// render from the same table, so a key can never be documented in one and
// missing from the other.
type binding struct {
	keys string
	desc string // short, for the status bar
	long string // optional fuller wording, overlay only
	full bool   // only shown in the overlay, not in the status bar
}

func (b binding) text() string {
	if b.long != "" {
		return b.long
	}
	return b.desc
}

var (
	keysBrowse = []binding{
		{keys: "j / k", desc: "move"},
		{keys: "tab", desc: "pane", long: "switch pane"},
		{keys: "⏎", desc: "read"},
		{keys: "/", desc: "filter"},
		{keys: "n", desc: "new", long: "new skill"},
		{keys: "d", desc: "get", long: "download from a source"},
		{keys: "s", desc: "sync", long: "copy into an agent's directory"},
		{keys: "u", desc: "used", long: "skills you have used before"},
		{keys: "v", desc: "doctor"},
		{keys: "y", desc: "yank path", full: true},
		{keys: "o", desc: "open in the OS", full: true},
		{keys: "g / G", desc: "first / last", full: true},
		{keys: "q", desc: "quit"},
	}
	keysDoc = []binding{
		{keys: "j / k", desc: "scroll"},
		{keys: "tab", desc: "section", long: "next section"},
		{keys: "e", desc: "edit"},
		{keys: "o", desc: "open"},
		{keys: "r", desc: "copy", long: "copy as prompt"},
		{keys: "shift+tab", desc: "previous section", full: true},
		{keys: "y", desc: "yank path", full: true},
		{keys: "q", desc: "back"},
	}
	keysDoctor = []binding{
		{keys: "j / k", desc: "move"},
		{keys: "f", desc: "fix"},
		{keys: "F", desc: "fix all"},
		{keys: "e", desc: "edit"},
		{keys: "r", desc: "recheck"},
		{keys: "o", desc: "open the folder", full: true},
		{keys: "q", desc: "back"},
	}
	keysNew = []binding{
		{keys: "tab", desc: "next field"},
		{keys: "← / →", desc: "choose"},
		{keys: "space", desc: "toggle"},
		{keys: "⏎", desc: "create"},
		{keys: "esc", desc: "cancel"},
	}
	keysRegistry = []binding{
		{keys: "space", desc: "select"},
		{keys: "a", desc: "all"},
		{keys: "t", desc: "retag"},
		{keys: "⏎", desc: "install"},
		{keys: "r", desc: "refetch"},
		{keys: "esc", desc: "back"},
	}
	keysSync = []binding{
		{keys: "space", desc: "select"},
		{keys: "a", desc: "all"},
		{keys: "tab", desc: "provider"},
		{keys: "g / l", desc: "scope"},
		{keys: "p", desc: "project"},
		{keys: "⏎", desc: "copy"},
		{keys: "x", desc: "remove"},
		{keys: "q", desc: "back"},
	}
	keysEdit = []binding{
		{keys: "ctrl+s", desc: "save"},
		{keys: "ctrl+f", desc: "format", long: "tidy whitespace, headings and list bullets"},
		{keys: "ctrl+p", desc: "preview", long: "show or hide the rendered half"},
		{keys: "esc", desc: "back", long: "back — asks once if there are unsaved changes"},
	}
	keysHistory = []binding{
		{keys: "j / k", desc: "move"},
		{keys: "m", desc: "missing only", long: "show only what you do not have"},
		{keys: "d", desc: "find", long: "look it up in the registry"},
		{keys: "q", desc: "back"},
	}
)

func keysFor(s screen) []binding {
	switch s {
	case screenDoc:
		return keysDoc
	case screenDoctor:
		return keysDoctor
	case screenEdit:
		return keysEdit
	case screenNew:
		return keysNew
	case screenRegistry:
		return keysRegistry
	case screenSync:
		return keysSync
	case screenHistory:
		return keysHistory
	}
	return keysBrowse
}

var screenNames = map[screen]string{
	screenBrowse: "browse", screenDoc: "read", screenDoctor: "doctor",
	screenEdit: "edit", screenNew: "new skill", screenRegistry: "registry", screenSync: "sync",
	screenHistory: "past used",
}

const helpHint = "? help"

// footerHelp lays the keys out for the status bar, dropping whole bindings
// rather than truncating one mid-word. "? help" is always kept: on a narrow
// terminal it is the only thing that still fits, and it is enough.
func footerHelp(bs []binding, w int) string {
	const sepW = 5 // "  ·  "
	sep := sFaint.Render("  ·  ")
	hint := sKey.Render("?") + sHelp.Render(" help")

	parts := []string{}
	used := lipgloss.Width(hint) // the hint is always kept and always last
	for _, b := range bs {
		if b.full {
			continue
		}
		item := sKey.Render(b.keys) + sHelp.Render(" "+b.desc)
		if used+sepW+lipgloss.Width(item) > w {
			break
		}
		parts = append(parts, item)
		used += sepW + lipgloss.Width(item)
	}
	parts = append(parts, hint)
	return strings.Join(parts, sep)
}

// helpOverlay is the full keymap for the current screen plus the keys that
// work everywhere. It replaces the screen rather than floating over it —
// a box drawn over a text UI hides the thing you are asking about.
func helpOverlay(s screen, w, h int) string {
	rows := max(1, h-4)
	var b strings.Builder

	global := []binding{
		{keys: "?", desc: "this help"},
		{keys: "esc / q", desc: "back one level"},
		{keys: "ctrl+c", desc: "quit, from any screen"},
	}
	// One key column for both blocks, or the global keys sit out of line with
	// the ones above them.
	keyW := 0
	for _, k := range append(append([]binding{}, keysFor(s)...), global...) {
		keyW = max(keyW, lipgloss.Width(k.keys))
	}
	row := func(k binding) {
		b.WriteString("  " + sKey.Render(pad(k.keys, keyW+3)) + sBase.Render(k.text()) + "\n")
	}

	b.WriteString(sHead.Render(strings.ToUpper(screenNames[s])) + "\n\n")
	for _, k := range keysFor(s) {
		row(k)
	}
	b.WriteString("\n" + sHead.Render("ANYWHERE") + "\n\n")
	for _, k := range global {
		row(k)
	}

	head := gap(sTitle.Render("help"), sDim.Render("skill-issue"), w) + "\n" + hrule(w)
	foot := sKey.Render("?") + sHelp.Render(" or ") + sKey.Render("esc") + sHelp.Render(" to go back")
	return lipgloss.JoinVertical(lipgloss.Left,
		head, block(b.String(), w, rows), hrule(w), foot)
}
