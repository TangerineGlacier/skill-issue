package main

import (
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TangerineGlacier/skill-issue/skill"
)

type screen int

const (
	screenBrowse screen = iota
	screenDoc
	screenDoctor
	screenEdit
	screenNew
	screenRegistry
	screenSync
	screenHistory
)

// Messages a child screen sends up to the root.
type (
	openDocMsg      struct{ s skill.Skill }
	openDoctorMsg   struct{}
	openEditMsg     struct{ path string }
	openNewMsg      struct{}
	openRegistryMsg struct{}
	openSyncMsg     struct{}
	openHistoryMsg  struct{}
	createdMsg      struct{ path string }
	backMsg         struct{}
)

// App owns the screen stack. Children never switch screens themselves; they
// emit a message and the root decides, so there is exactly one place that
// knows what "back" means.
type App struct {
	root     string
	scr      screen
	browse   Browse
	doc      Doc
	doctor   Doctor
	edit     Edit
	editFrom screen
	create   NewSkill
	registry Registry
	sync     SyncView
	history  HistoryView
	w, h     int
	help     bool
	err      error
}

// editing reports whether the focused screen is taking text input, so "?"
// types a question mark instead of opening the help.
func (a App) editing() bool {
	switch a.scr {
	case screenBrowse:
		return a.browse.filtering
	case screenNew:
		return a.create.focus <= fTags
	case screenRegistry:
		return a.registry.retagging
	case screenSync:
		return a.sync.editing
	case screenEdit:
		return true
	}
	return false
}

func NewApp(root string) App {
	return App{root: root, browse: NewBrowse(root)}
}

func (a App) Init() tea.Cmd { return nil }

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.w, a.h = msg.Width, msg.Height

	case openDocMsg:
		a.doc = NewDoc(msg.s, a.w, a.h)
		a.scr = screenDoc
		return a, nil

	case openDoctorMsg:
		a.doctor = NewDoctor(a.root, a.w, a.h)
		a.scr = screenDoctor
		return a, nil

	case openEditMsg:
		var cmd tea.Cmd
		a.edit, cmd = NewEdit(msg.path, a.w, a.h)
		a.editFrom = a.scr
		a.scr = screenEdit
		return a, cmd

	case openNewMsg:
		a.create = NewNewSkill(a.root, filepath.Join(filepath.Dir(a.root), "templates"), a.w, a.h)
		a.scr = screenNew
		return a, nil

	case openRegistryMsg:
		var cmd tea.Cmd
		a.registry, cmd = NewRegistry(a.root, a.w, a.h)
		a.scr = screenRegistry
		return a, cmd

	case openSyncMsg:
		a.sync = NewSyncView(a.root, a.w, a.h)
		a.scr = screenSync
		return a, nil

	case openHistoryMsg:
		var cmd tea.Cmd
		a.history, cmd = NewHistoryView(a.root, a.w, a.h)
		a.scr = screenHistory
		return a, cmd

	case createdMsg:
		a.scr = screenBrowse
		a.browse = a.browse.reload()
		a.browse.status = "created " + msg.path
		return a, nil

	case backMsg:
		if a.scr == screenEdit {
			// Back out of the editor to whatever opened it, showing the file
			// as it now stands on disk.
			a.scr = a.editFrom
			switch a.scr {
			case screenDoc:
				if s, err := skill.Load(a.doc.s.Dir); err == nil {
					a.doc = NewDoc(s, a.w, a.h)
				}
			case screenDoctor:
				a.doctor = a.doctor.reload()
			}
			return a, nil
		}
		a.scr = screenBrowse
		a.browse = a.browse.reload()
		var cmd tea.Cmd
		a.browse, cmd = a.browse.Update(tea.WindowSizeMsg{Width: a.w, Height: a.h})
		return a, cmd

	case reloadMsg:
		a.browse = a.browse.reload()

	case errMsg:
		a.err = msg.err

	case tea.KeyMsg:
		// ctrl+c always quits, whatever screen has focus.
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}
		if a.help {
			// While the help is up it is the only thing listening, so a
			// stray key cannot act on the screen you cannot currently see.
			switch msg.String() {
			case "?", "esc", "q", "enter", " ":
				a.help = false
			}
			return a, nil
		}
		if msg.String() == "?" && !a.editing() {
			a.help = true
			return a, nil
		}
	}

	var cmd tea.Cmd
	switch a.scr {
	case screenDoc:
		a.doc, cmd = a.doc.Update(msg)
	case screenDoctor:
		a.doctor, cmd = a.doctor.Update(msg)
	case screenEdit:
		a.edit, cmd = a.edit.Update(msg)
	case screenNew:
		a.create, cmd = a.create.Update(msg)
	case screenRegistry:
		a.registry, cmd = a.registry.Update(msg)
	case screenSync:
		a.sync, cmd = a.sync.Update(msg)
	case screenHistory:
		a.history, cmd = a.history.Update(msg)
	default:
		a.browse, cmd = a.browse.Update(msg)
	}
	return a, cmd
}

func (a App) View() string {
	if a.help {
		return helpOverlay(a.scr, a.w, a.h)
	}
	switch a.scr {
	case screenDoc:
		return a.doc.View()
	case screenDoctor:
		return a.doctor.View()
	case screenEdit:
		return a.edit.View()
	case screenNew:
		return a.create.View()
	case screenRegistry:
		return a.registry.View()
	case screenSync:
		return a.sync.View()
	case screenHistory:
		return a.history.View()
	}
	return a.browse.View()
}
