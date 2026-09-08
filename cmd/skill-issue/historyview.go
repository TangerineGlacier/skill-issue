package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/TangerineGlacier/skill-issue/skill"
)

// HistoryView is what you have actually reached for, read out of the agent
// logs on this machine. Its point is the last column: a skill you use often
// and do not have in the library is the next thing worth writing.
type HistoryView struct {
	root        string
	uses        []skill.Use
	sources     []skill.HistorySource
	cursor      int
	onlyMissing bool
	loading     bool
	w, h        int
	status      string
}

type historyMsg struct {
	uses    []skill.Use
	sources []skill.HistorySource
}

func NewHistoryView(root string, w, h int) (HistoryView, tea.Cmd) {
	v := HistoryView{root: root, loading: true, w: w, h: h}
	return v, func() tea.Msg {
		// Reading 150 transcripts takes a moment, so it is a command, not
		// work done inside Update.
		uses, sources := skill.History(root)
		return historyMsg{uses, sources}
	}
}

func (v HistoryView) rows() []skill.Use {
	if !v.onlyMissing {
		return v.uses
	}
	return skill.Suggestions(v.uses)
}

func (v HistoryView) Update(msg tea.Msg) (HistoryView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.w, v.h = msg.Width, msg.Height

	case historyMsg:
		v.loading = false
		v.uses, v.sources = msg.uses, msg.sources

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return v, func() tea.Msg { return backMsg{} }
		case "j", "down":
			v.cursor = clamp(v.cursor+1, 0, max(0, len(v.rows())-1))
		case "k", "up":
			v.cursor = clamp(v.cursor-1, 0, max(0, len(v.rows())-1))
		case "m":
			v.onlyMissing = !v.onlyMissing
			v.cursor = 0
		case "d":
			// Look the highlighted skill up in the registry, which is the
			// obvious next move for something you use but do not have.
			if u, ok := v.current(); ok && !u.InLibrary {
				v.status = "searching the registry for " + u.Name
				return v, func() tea.Msg { return openRegistryMsg{} }
			}
		}
	}
	return v, nil
}

func (v HistoryView) current() (skill.Use, bool) {
	r := v.rows()
	if v.cursor < 0 || v.cursor >= len(r) {
		return skill.Use{}, false
	}
	return r[v.cursor], true
}

func (v HistoryView) View() string {
	if v.w == 0 {
		return "loading…"
	}
	srcRows := len(v.sources) + 2
	rows := max(1, v.h-4-srcRows-1)
	return lipgloss.JoinVertical(lipgloss.Left,
		v.header(),
		block(v.table(rows), v.w, rows),
		hrule(v.w),
		block(v.sourcesView(), v.w, srcRows),
		hrule(v.w),
		v.footer())
}

func (v HistoryView) header() string {
	missing := len(skill.Suggestions(v.uses))
	left := sTitle.Render("past used")
	if v.loading {
		left += sDim.Render("   reading transcripts…")
	} else {
		left += sDim.Render(fmt.Sprintf("   %d skills used · ", len(v.uses))) +
			sWarn.Render(fmt.Sprintf("%d not in the library", missing))
	}
	right := sDim.Render("all")
	if v.onlyMissing {
		right = sAccent.Render("suggestions only")
	}
	return gap(left, right, v.w) + "\n" + hrule(v.w)
}

func (v HistoryView) table(rows int) string {
	r := v.rows()
	if v.loading {
		return "\n  " + sDim.Render("reading ~/.claude/projects…")
	}
	if len(r) == 0 {
		return "\n  " + sDim.Render("nothing recorded yet")
	}
	nameW, useW, lastW, libW := 26, 6, 14, 8
	projW := max(10, v.w-2-nameW-useW-lastW-libW-4)

	var b strings.Builder
	fmt.Fprintln(&b, "  "+sDim.Render(pad("SKILL", nameW)+pad("USED", useW)+
		pad("LAST", lastW)+pad("IN LIB", libW)+"  PROJECTS"))

	start := 0
	visible := rows - 1
	if v.cursor >= visible {
		start = v.cursor - visible + 1
	}
	for i := start; i < len(r) && i < start+visible; i++ {
		u := r[i]
		lib := sAccent.Render(pad("yes", libW))
		if !u.InLibrary {
			lib = sWarn.Render(pad("missing", libW))
		}
		last := "never"
		if !u.Last.IsZero() {
			last = ago(u.Last)
		}
		fmt.Fprintln(&b, bar(i == v.cursor, true)+
			pick(i == v.cursor, sSelected, sBase).Render(pad(u.Name, nameW))+
			sDim.Render(pad(fmt.Sprintf("%d", u.Count), useW)+pad(last, lastW))+
			lib+"  "+sFaint.Render(truncate(strings.Join(u.Projects, ", "), projW)))
	}
	return b.String()
}

func (v HistoryView) sourcesView() string {
	var b strings.Builder
	fmt.Fprintln(&b, sHead.Render("SOURCES")+sDim.Render("   where this came from, and what could not be read"))
	for _, s := range v.sources {
		note := s.Note
		style := sFaint
		if note == "" {
			note = fmt.Sprintf("%d transcripts · %d uses", s.Files, s.Uses)
			style = sDim
		}
		fmt.Fprintln(&b, "  "+sBase.Render(pad(s.ID, 11))+style.Render(truncate(note, v.w-14)))
	}
	return b.String()
}

func (v HistoryView) footer() string {
	if v.status != "" {
		return sAccent.Render(truncate(v.status, v.w))
	}
	return footerHelp(keysHistory, v.w)
}
