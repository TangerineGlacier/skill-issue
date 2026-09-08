package main

import (
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/TangerineGlacier/skill-issue/skill"
)

// minPreview is the narrowest terminal where a split still leaves both halves
// readable. Below it the editor takes the whole width.
const minPreview = 96

// Edit is the in-app editor. Editing a skill is the one thing you come here to
// do, so it happens in the TUI rather than by handing the terminal to $EDITOR.
// The right half renders the markdown as the read screen does, so you can see
// what you are writing without leaving the editor.
type Edit struct {
	path    string
	ta      textarea.Model
	pv      viewport.Model
	shown   string // buffer the preview was rendered from
	saved   string // contents as they are on disk
	preview bool
	w, h    int
	status  string
	warned  bool // esc pressed once with unsaved changes
}

func NewEdit(path string, w, h int) (Edit, tea.Cmd) {
	e := Edit{path: path, w: w, h: h, preview: true}
	b, err := os.ReadFile(path)
	if err != nil {
		e.status = err.Error()
	}
	e.saved = string(b)

	e.ta = textarea.New()
	e.ta.Prompt = ""
	e.ta.ShowLineNumbers = true
	e.ta.CharLimit = 0
	e.ta.SetValue(e.saved)
	focus := e.ta.Focus() // the textarea ignores keys until it has focus
	// SetValue leaves the cursor at the end of the file; you read a skill from
	// the top. alt+< is the textarea's own "go to start" binding.
	e.ta, _ = e.ta.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("<"), Alt: true})
	e.pv = viewport.New(1, 1)
	e.resize()
	e.render()
	return e, focus
}

func (e Edit) split() bool { return e.preview && e.w >= minPreview }

// editWidth is the textarea's share of the row; the preview takes the rest.
func (e Edit) editWidth() int {
	if !e.split() {
		return max(20, e.w-2)
	}
	return max(20, e.w/2-2)
}

func (e *Edit) resize() {
	rows := max(1, e.h-4)
	e.ta.SetWidth(e.editWidth())
	e.ta.SetHeight(rows)
	e.pv.Width, e.pv.Height = max(20, e.w-e.editWidth()-3), rows
	e.shown = "" // force a re-render at the new width
}

func (e Edit) dirty() bool { return e.ta.Value() != e.saved }

// Update keeps the preview in step with the buffer: one render per message,
// never one per frame.
func (e Edit) Update(msg tea.Msg) (Edit, tea.Cmd) {
	e, cmd := e.update(msg)
	if e.split() {
		e.render()
	}
	return e, cmd
}

func (e Edit) update(msg tea.Msg) (Edit, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		e.w, e.h = msg.Width, msg.Height
		e.resize()
		return e, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+s":
			if err := os.WriteFile(e.path, []byte(e.ta.Value()), 0o644); err != nil {
				e.status = err.Error()
				return e, nil
			}
			e.saved = e.ta.Value()
			e.warned = false
			e.status = "saved " + filepath.Base(e.path)
			return e, nil
		case "ctrl+f":
			f := skill.Format(e.ta.Value())
			if f == e.ta.Value() {
				e.status = "already tidy"
				return e, nil
			}
			e.ta.SetValue(f)
			e.status = "formatted — ctrl+s to save"
			return e, nil
		case "ctrl+p":
			e.preview = !e.preview
			e.resize()
			if !e.split() {
				e.status = "preview off"
			}
			return e, nil
		case "esc":
			if e.dirty() && !e.warned {
				e.warned = true
				e.status = "unsaved changes — ctrl+s to save, esc again to discard"
				return e, nil
			}
			return e, func() tea.Msg { return backMsg{} }
		default:
			e.warned, e.status = false, ""
		}
	}
	var cmd tea.Cmd
	e.ta, cmd = e.ta.Update(msg)
	return e, cmd
}

// render refreshes the preview when the text changed, and keeps it roughly
// where the cursor is.
// ponytail: the preview follows the cursor by line ratio, not by mapping source
// lines to rendered ones — good enough until wrapped paragraphs drift far.
func (e *Edit) render() {
	if e.shown != e.ta.Value() {
		e.shown = e.ta.Value()
		e.pv.SetContent(renderMD(bodyOf(e.shown), e.pv.Width))
	}
	at := float64(e.ta.Line()) / float64(max(1, e.ta.LineCount()))
	e.pv.SetYOffset(int(at * float64(e.pv.TotalLineCount())))
}

// bodyOf drops the frontmatter so the preview shows what a reader sees.
func bodyOf(src string) string {
	if s, err := skill.Parse(src); err == nil {
		return s.Body
	}
	return src
}

func (e Edit) View() string {
	if e.w == 0 {
		return "loading…"
	}
	rows := max(1, e.h-4)
	body := block(e.ta.View(), e.editWidth(), rows)
	if e.split() {
		body = lipgloss.JoinHorizontal(lipgloss.Top,
			body, vrule(rows), block(e.pv.View(), e.pv.Width, rows))
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		e.header(), body, hrule(e.w), e.footer())
}

func (e Edit) header() string {
	name := filepath.Base(filepath.Dir(e.path)) + "/" + filepath.Base(e.path)
	right := ""
	if e.dirty() {
		right = sWarn.Render("modified")
	}
	return gap(sTitle.Render(name), right, e.w) + "\n" + hrule(e.w)
}

func (e Edit) footer() string {
	if e.status != "" {
		return sAccent.Render(truncate(e.status, e.w))
	}
	return footerHelp(keysEdit, e.w)
}
