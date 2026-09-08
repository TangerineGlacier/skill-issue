package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/TangerineGlacier/skill-issue/skill"
)

// SyncView pushes skills out of the library and into an agent's directory.
// Two axes: which agent, and global (this machine) or local (one project).
type SyncView struct {
	root    string
	library []skill.Skill
	sel     map[string]bool
	cursor  int

	provider int
	scope    skill.Scope
	project  textinput.Model
	editing  bool

	status string
	w, h   int
}

func NewSyncView(root string, w, h int) SyncView {
	lib, _ := skill.LoadAll(root)
	ti := textinput.New()
	ti.Prompt = "project  "
	ti.PromptStyle = sDim
	wd, _ := os.Getwd()
	ti.SetValue(wd)
	ti.CharLimit = 200

	return SyncView{root: root, library: lib, sel: map[string]bool{},
		scope: skill.Global, project: ti, w: w, h: h}
}

func (v SyncView) prov() skill.Provider { return skill.Providers[v.provider] }

func (v SyncView) dst() (string, error) {
	return v.prov().Dir(v.scope, v.project.Value())
}

func (v SyncView) Update(msg tea.Msg) (SyncView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.w, v.h = msg.Width, msg.Height
		v.project.Width = max(20, v.w/3)
		return v, nil

	case tea.KeyMsg:
		if v.editing {
			switch msg.String() {
			case "enter", "esc":
				v.editing = false
				v.project.Blur()
			default:
				var cmd tea.Cmd
				v.project, cmd = v.project.Update(msg)
				return v, cmd
			}
			return v, nil
		}
		switch msg.String() {
		case "q", "esc":
			return v, func() tea.Msg { return backMsg{} }
		case "j", "down":
			v.cursor = clamp(v.cursor+1, 0, max(0, len(v.library)-1))
		case "k", "up":
			v.cursor = clamp(v.cursor-1, 0, max(0, len(v.library)-1))
		case " ":
			if s, ok := v.current(); ok {
				v.sel[s.Name] = !v.sel[s.Name]
			}
		case "a":
			all := len(v.selected()) < len(v.library)
			for _, s := range v.library {
				v.sel[s.Name] = all
			}
		case "tab", "right", "left":
			d := 1
			if msg.String() == "left" {
				d = -1
			}
			v.provider = (v.provider + len(skill.Providers) + d) % len(skill.Providers)
			v.status = ""
		case "g":
			v.scope = skill.Global
			v.status = ""
		case "l":
			v.scope = skill.Local
			v.status = ""
		case "p":
			v.scope = skill.Local
			v.editing = true
			return v, v.project.Focus()
		case "x":
			return v.remove()
		case "enter":
			return v.apply()
		}
	}
	return v, nil
}

func (v SyncView) current() (skill.Skill, bool) {
	if v.cursor < 0 || v.cursor >= len(v.library) {
		return skill.Skill{}, false
	}
	return v.library[v.cursor], true
}

func (v SyncView) selected() []skill.Skill {
	var out []skill.Skill
	for _, s := range v.library {
		if v.sel[s.Name] {
			out = append(out, s)
		}
	}
	return out
}

// targets is what enter would act on: the selection, or the row under the
// cursor when nothing is selected.
func (v SyncView) targets() []skill.Skill {
	if sel := v.selected(); len(sel) > 0 {
		return sel
	}
	if s, ok := v.current(); ok {
		return []skill.Skill{s}
	}
	return nil
}

func (v SyncView) apply() (SyncView, tea.Cmd) {
	dst, err := v.dst()
	if err != nil {
		v.status = err.Error()
		return v, nil
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		v.status = err.Error()
		return v, nil
	}
	var added, updated, failed int
	for _, s := range v.targets() {
		_, replaced, err := skill.Sync(s, dst)
		switch {
		case err != nil:
			failed++
		case replaced:
			updated++
		default:
			added++
		}
	}
	v.status = fmt.Sprintf("%d installed, %d updated, %d failed → %s",
		added, updated, failed, dst)
	return v, nil
}

func (v SyncView) remove() (SyncView, tea.Cmd) {
	dst, err := v.dst()
	if err != nil {
		v.status = err.Error()
		return v, nil
	}
	var n int
	for _, s := range v.targets() {
		if err := skill.Unsync(s.Name, dst); err == nil {
			n++
		}
	}
	v.status = fmt.Sprintf("removed %d from %s", n, dst)
	return v, nil
}

func (v SyncView) View() string {
	if v.w == 0 {
		return "loading…"
	}
	rows := max(1, v.h-4)
	pw := 0
	if v.w >= 96 {
		pw = min(42, v.w/3)
	}
	lw := v.w - pw
	if pw > 0 {
		lw -= 3
	}
	cols := []string{block(v.list(rows, lw), lw, rows)}
	if pw > 0 {
		cols = append(cols, vrule(rows), block(v.target(pw), pw, rows))
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		v.header(), lipgloss.JoinHorizontal(lipgloss.Top, cols...), hrule(v.w), v.footer())
}

func (v SyncView) header() string {
	var tabs []string
	for i, p := range skill.Providers {
		s := sDim
		if i == v.provider {
			s = sCursor
		}
		tabs = append(tabs, s.Render(p.ID))
	}
	scopes := sCursor.Render("global") + sFaint.Render(" / ") + sDim.Render("local")
	if v.scope == skill.Local {
		scopes = sDim.Render("global") + sFaint.Render(" / ") + sCursor.Render("local")
	}
	left := sTitle.Render("sync") + "   " + strings.Join(tabs, sFaint.Render(" · "))
	return gap(left, scopes, v.w) + "\n" + hrule(v.w)
}

func (v SyncView) list(rows, w int) string {
	var b strings.Builder
	dst, _ := v.dst()
	there := map[string]bool{}
	for _, n := range skill.Installed(dst) {
		there[n] = true
	}

	// The state column is right-aligned per row, so its header must be too.
	fmt.Fprintln(&b, "  "+sDim.Render(gap(pad("SKILL", 28)+"COLLECTION", "THERE", w-2)))
	start := 0
	visible := rows - 1
	if v.cursor >= visible {
		start = v.cursor - visible + 1
	}
	for i := start; i < len(v.library) && i < start+visible; i++ {
		s := v.library[i]
		box := "[ ]"
		if v.sel[s.Name] {
			box = sAccent.Render("[x]")
		}
		state := sFaint.Render("—")
		if there[s.Name] {
			state = sAccent.Render("installed")
		}
		row := box + " " + pick(i == v.cursor, sSelected, sBase).
			Render(pad(s.Name, 28)+sDim.Render(pad(s.Group, 14)))
		fmt.Fprintln(&b, bar(i == v.cursor, true)+gap(row, state, w-2))
	}

	// Anything in the provider directory the library does not have. Worth
	// seeing: it is either a skill to import, or one to clean up.
	var extra []string
	for _, n := range skill.Installed(dst) {
		found := false
		for _, s := range v.library {
			if s.Name == n {
				found = true
			}
		}
		if !found {
			extra = append(extra, n)
		}
	}
	if len(extra) > 0 {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, "  "+sDim.Render(fmt.Sprintf("there but not in the library: %s",
			truncate(strings.Join(extra, ", "), w-34))))
	}
	return b.String()
}

func (v SyncView) target(w int) string {
	var b strings.Builder
	dst, err := v.dst()
	fmt.Fprintln(&b, sHead.Render("TARGET"))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, sBase.Render(v.prov().Label))
	if err != nil {
		fmt.Fprintln(&b, sErr.Render(truncate(err.Error(), w)))
	} else {
		for _, l := range strings.Split(lipgloss.NewStyle().Width(w).Render(shortPath(dst)), "\n") {
			fmt.Fprintln(&b, sAccent.Render(l))
		}
	}
	fmt.Fprintln(&b)
	if v.scope == skill.Local {
		fmt.Fprintln(&b, v.project.View())
		fmt.Fprintln(&b, sFaint.Render("p to change"))
	} else {
		fmt.Fprintln(&b, sFaint.Render("every project on this machine"))
	}
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, sHead.Render("WILL COPY"))
	t := v.targets()
	if len(t) == 0 {
		fmt.Fprintln(&b, sDim.Render("nothing selected"))
	}
	for i, s := range t {
		if i == 12 {
			fmt.Fprintln(&b, sDim.Render(fmt.Sprintf("… and %d more", len(t)-i)))
			break
		}
		fmt.Fprintln(&b, sAccent.Render("+ ")+truncate(s.Name, w-2))
	}
	return b.String()
}

func (v SyncView) footer() string {
	if v.editing {
		return v.project.View() + sDim.Render("   ⏎ done")
	}
	if v.status != "" {
		return sAccent.Render(truncate(v.status, v.w))
	}
	return footerHelp(keysSync, v.w)
}
