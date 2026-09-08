package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// The status bar must never cut a key in half. It drops whole bindings, and it
// always keeps "? help" — on a narrow terminal that is the only thing that
// still fits, and it is enough to find everything else.
func TestFooterHelpDegrades(t *testing.T) {
	cases := []struct {
		name string
		w    int
	}{
		{"wide", 160}, {"normal", 120}, {"narrow", 80}, {"very narrow", 40}, {"absurd", 16},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := footerHelp(keysBrowse, tc.w)
			if w := lineWidth(got); w > tc.w {
				t.Errorf("footer is %d cells wide, terminal is %d: %q", w, tc.w, got)
			}
			if !strings.HasSuffix(stripANSI(got), helpHint) {
				t.Errorf("footer must end with %q, got %q", helpHint, stripANSI(got))
			}
			// Whatever survived must be whole: no trailing partial word.
			if strings.Contains(stripANSI(got), "…") {
				t.Errorf("footer truncated mid-binding: %q", stripANSI(got))
			}
		})
	}
	wide := stripANSI(footerHelp(keysBrowse, 160))
	narrow := stripANSI(footerHelp(keysBrowse, 60))
	if len(strings.Split(narrow, "·")) >= len(strings.Split(wide, "·")) {
		t.Error("a narrower terminal should show fewer bindings, not the same")
	}
}

// The footer only advertises the common keys; the overlay has to list them all,
// or a key documented nowhere is a key nobody finds.
func TestOverlayListsEveryBinding(t *testing.T) {
	for s, bs := range map[screen][]binding{
		screenBrowse: keysBrowse, screenDoc: keysDoc, screenDoctor: keysDoctor,
		screenNew: keysNew, screenRegistry: keysRegistry, screenSync: keysSync,
		screenHistory: keysHistory,
	} {
		out := stripANSI(helpOverlay(s, 100, 34))
		for _, b := range bs {
			if !strings.Contains(out, b.keys) {
				t.Errorf("%s overlay is missing key %q", screenNames[s], b.keys)
			}
			if !strings.Contains(out, b.text()) {
				t.Errorf("%s overlay is missing %q", screenNames[s], b.text())
			}
		}
		for _, always := range []string{"ctrl+c", "esc / q"} {
			if !strings.Contains(out, always) {
				t.Errorf("%s overlay is missing the global key %q", screenNames[s], always)
			}
		}
		if lines := strings.Split(helpOverlay(s, 100, 34), "\n"); len(lines) != 34 {
			t.Errorf("%s overlay is %d lines, terminal is 34", screenNames[s], len(lines))
		}
	}
}

func TestHelpOverlayOpensAndCloses(t *testing.T) {
	app := func(t *testing.T) App {
		t.Helper()
		a := NewApp(repoSkills(t))
		m, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
		return m.(App)
	}

	cases := []struct {
		name string
		open []string // keys to reach a screen, then "?"
		want string   // the overlay should name this screen
	}{
		{"from browse", []string{"?"}, "BROWSE"},
		{"from the doc screen", []string{"tab", "enter", "?"}, "READ"},
		{"from doctor", []string{"v", "?"}, "DOCTOR"},
		{"from sync", []string{"s", "?"}, "SYNC"},
		{"from past used", []string{"u", "?"}, "PAST USED"},
		{"from the new-skill form", []string{"n", "?"}, ""}, // ? is text there
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := app(t)
			var m tea.Model = a
			for _, k := range tc.open {
				var cmd tea.Cmd
				m, cmd = m.Update(keyMsg(k))
				// Deliver the screen-switch message, bounded: a focused
				// textinput returns a cursor-blink command that returns
				// another one forever, so draining to nil never terminates.
				for i := 0; cmd != nil && i < 3; i++ {
					m, cmd = m.Update(cmd())
				}
			}
			got := m.(App)
			if tc.want == "" {
				if got.help {
					t.Fatal("? should type a character in a text field, not open the help")
				}
				return
			}
			if !got.help {
				t.Fatal("? did not open the help")
			}
			if out := stripANSI(got.View()); !strings.Contains(out, tc.want) {
				t.Errorf("overlay does not name the screen %q:\n%s", tc.want, out)
			}
			m, _ = m.Update(keyMsg("esc"))
			if m.(App).help {
				t.Error("esc did not close the help")
			}
		})
	}
}

// While the help is up it swallows input: acting on a screen you cannot see is
// the kind of thing you only notice afterwards.
func TestHelpSwallowsKeys(t *testing.T) {
	a := NewApp(repoSkills(t))
	m, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	m, _ = m.Update(keyMsg("?"))
	before := m.(App).browse.si
	for _, k := range []string{"j", "j", "j", "n", "d"} {
		m, _ = m.Update(keyMsg(k))
	}
	got := m.(App)
	if !got.help {
		t.Fatal("the help closed on a key that should have been swallowed")
	}
	if got.browse.si != before || got.scr != screenBrowse {
		t.Errorf("keys reached the screen behind the help: cursor %d→%d, screen %v",
			before, got.browse.si, got.scr)
	}
}
