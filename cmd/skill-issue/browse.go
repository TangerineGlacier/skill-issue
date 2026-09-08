package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/TangerineGlacier/skill-issue/skill"
)

// Browse is the master-detail home screen: collections, skills, preview.
type Browse struct {
	root   string
	all    []skill.Skill
	errs   []error
	groups []string // "all" first, then the folders under skills/

	gi, si int  // cursors: group column, skill column
	onList bool // which column has focus

	filter    textinput.Model
	filtering bool

	preview viewport.Model
	w, h    int
	status  string
}

const (
	colGroups  = 20
	colPreview = 40
)

func NewBrowse(root string) Browse {
	all, errs := skill.LoadAll(root)

	seen := map[string]bool{}
	groups := []string{"all"}
	for _, s := range all {
		if !seen[s.Group] {
			seen[s.Group] = true
			groups = append(groups, s.Group)
		}
	}

	ti := textinput.New()
	ti.Prompt = "/"
	ti.Placeholder = "filter"
	ti.PromptStyle = sAccent
	ti.TextStyle = sBase

	b := Browse{root: root, all: all, errs: errs, groups: groups, filter: ti}
	b.preview = viewport.New(colPreview, 10)
	return b
}

// visible is the middle column: the selected group, narrowed by the filter.
func (b Browse) visible() []skill.Skill {
	group := b.groups[b.gi]
	q := b.filter.Value()
	out := make([]skill.Skill, 0, len(b.all))
	for _, s := range b.all {
		if group != "all" && s.Group != group {
			continue
		}
		if !fuzzy(s.Name+" "+strings.Join(s.Tags, " "), q) {
			continue
		}
		out = append(out, s)
	}
	return out
}

func (b Browse) current() (skill.Skill, bool) {
	v := b.visible()
	if b.si < 0 || b.si >= len(v) {
		return skill.Skill{}, false
	}
	return v[b.si], true
}

// reload re-reads the library from disk, keeping the cursors where they were.
func (b Browse) reload() Browse {
	n := NewBrowse(b.root)
	n.w, n.h = b.w, b.h
	n.gi = clamp(b.gi, 0, len(n.groups)-1)
	n.onList = b.onList
	n.filter = b.filter
	n.preview = b.preview
	n.si = clamp(b.si, 0, max(0, len(n.visible())-1))
	n.syncPreview()
	return n
}

func (b Browse) Init() tea.Cmd { return nil }

func (b Browse) Update(msg tea.Msg) (Browse, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		b.w, b.h = msg.Width, msg.Height
		b.preview.Width = b.previewWidth()
		b.preview.Height = max(1, b.h-5)
		b.syncPreview()
		return b, nil

	case tea.KeyMsg:
		if b.filtering {
			return b.updateFilter(msg)
		}
		return b.updateKeys(msg)
	}
	var cmd tea.Cmd
	b.preview, cmd = b.preview.Update(msg)
	return b, cmd
}

func (b Browse) updateFilter(msg tea.KeyMsg) (Browse, tea.Cmd) {
	switch msg.String() {
	case "esc":
		b.filtering = false
		b.filter.SetValue("")
		b.filter.Blur()
	case "enter":
		b.filtering = false
		b.filter.Blur()
		b.onList = true
	default:
		var cmd tea.Cmd
		b.filter, cmd = b.filter.Update(msg)
		b.si = 0
		b.syncPreview()
		return b, cmd
	}
	b.si = 0
	b.syncPreview()
	return b, nil
}

func (b Browse) updateKeys(msg tea.KeyMsg) (Browse, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return b, tea.Quit

	case "/":
		b.filtering = true
		b.onList = true
		b.status = ""
		return b, b.filter.Focus()

	case "esc":
		if b.filter.Value() != "" {
			b.filter.SetValue("")
			b.si = 0
			b.syncPreview()
		}
		b.status = ""

	case "tab", "h", "l", "left", "right":
		b.onList = !b.onList
		b.status = ""

	case "j", "down":
		b.move(1)
	case "k", "up":
		b.move(-1)
	case "g", "home":
		b.moveTo(0)
	case "G", "end":
		if b.onList {
			b.moveTo(len(b.visible()) - 1)
		} else {
			b.moveTo(len(b.groups) - 1)
		}

	case "y":
		if s, ok := b.current(); ok {
			b.status = "path copied: " + s.Path()
			return b, copyCmd(s.Path())
		}

	case "o":
		if s, ok := b.current(); ok {
			b.status = "opening " + s.Name
			return b, openCmd(s.Path())
		}

	case "enter", "L":
		if s, ok := b.current(); ok {
			return b, func() tea.Msg { return openDocMsg{s} }
		}

	case "v":
		return b, func() tea.Msg { return openDoctorMsg{} }

	case "n":
		return b, func() tea.Msg { return openNewMsg{} }

	case "d":
		return b, func() tea.Msg { return openRegistryMsg{} }

	case "s":
		return b, func() tea.Msg { return openSyncMsg{} }

	case "u":
		return b, func() tea.Msg { return openHistoryMsg{} }
	}
	return b, nil
}

func (b *Browse) move(d int) {
	if b.onList {
		b.moveTo(b.si + d)
		return
	}
	b.moveTo(b.gi + d)
}

func (b *Browse) moveTo(i int) {
	if b.onList {
		n := len(b.visible())
		b.si = clamp(i, 0, n-1)
	} else {
		b.gi = clamp(i, 0, len(b.groups)-1)
		b.si = 0
	}
	b.status = ""
	b.syncPreview()
}

func clamp(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	return min(max(v, lo), hi)
}

func (b Browse) previewWidth() int {
	if b.w < 100 {
		return 0 // too narrow for three columns; the preview is the first to go
	}
	return min(colPreview, b.w/3)
}

func (b *Browse) syncPreview() {
	s, ok := b.current()
	if !ok {
		b.preview.SetContent(sDim.Render("no skill selected"))
		return
	}
	w := max(20, b.previewWidth())
	var sb strings.Builder
	fmt.Fprintln(&sb, sTitle.Render(s.Name))
	fmt.Fprintln(&sb, sDim.Render(s.Group+"/"+s.Name))
	fmt.Fprintln(&sb)
	fmt.Fprintln(&sb, sHead.Render("FRONTMATTER"))
	fmt.Fprintln(&sb, field("tags", strings.Join(s.Tags, ", "), w))
	fmt.Fprintln(&sb, field("files", fmt.Sprintf("%d", len(s.Files)), w))
	fmt.Fprintln(&sb, field("size", humanSize(s.Size), w))
	fmt.Fprintln(&sb, field("edited", ago(s.ModTime), w))
	fmt.Fprintln(&sb)
	fmt.Fprintln(&sb, sHead.Render("DESCRIPTION"))
	desc := s.Description
	if desc == "" {
		desc = sErr.Render("(none — this skill will never load)")
	}
	fmt.Fprintln(&sb, lipgloss.NewStyle().Width(w).Foreground(fg).Render(desc))
	if hs := s.Headings(); len(hs) > 0 {
		fmt.Fprintln(&sb)
		fmt.Fprintln(&sb, sHead.Render("OUTLINE"))
		for _, h := range hs {
			fmt.Fprintln(&sb, sDim.Render("## ")+truncate(h, w-3))
		}
	}
	b.preview.SetContent(sb.String())
	b.preview.GotoTop()
}

func field(k, v string, w int) string {
	if v == "" {
		v = sFaint.Render("—")
	}
	return sDim.Render(pad(k, 8)) + truncate(v, max(1, w-8))
}

func (b Browse) View() string {
	if b.w == 0 {
		return "loading…"
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top, b.columns()...)
	return lipgloss.JoinVertical(lipgloss.Left, b.header(), body, hrule(b.w), b.footer())
}

func (b Browse) columns() []string {
	rows := max(1, b.h-4)
	pw := b.previewWidth()
	gw := colGroups
	if b.w < 70 {
		gw = 0 // narrowest layout: the skill list only
	}
	seps := 0
	if gw > 0 {
		seps++
	}
	if pw > 0 {
		seps++
	}
	lw := b.w - gw - pw - seps*3

	var out []string
	if gw > 0 {
		out = append(out, block(b.groupColumn(gw), gw, rows), vrule(rows))
	}
	out = append(out, block(b.listColumn(lw), lw, rows))
	if pw > 0 {
		out = append(out, vrule(rows), block(b.preview.View(), pw, rows))
	}
	return out
}

// vrule is a full-height column separator, one padded cell either side, so
// every column gets the same breathing room without per-column padding.
func vrule(rows int) string {
	line := " " + sVRule.Render("│") + " "
	ls := make([]string, rows)
	for i := range ls {
		ls[i] = line
	}
	return strings.Join(ls, "\n")
}

// block fixes a column to w x rows so JoinHorizontal cannot ragged-edge it.
func block(s string, w, rows int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > rows {
		lines = lines[:rows]
	}
	for len(lines) < rows {
		lines = append(lines, "")
	}
	for i := range lines {
		lines[i] = pad(lines[i], w)
	}
	return strings.Join(lines, "\n")
}

func (b Browse) groupColumn(w int) string {
	var sb strings.Builder
	fmt.Fprintln(&sb, sHead.Render("COLLECTIONS"))
	fmt.Fprintln(&sb)
	counts := map[string]int{"all": len(b.all)}
	for _, s := range b.all {
		counts[s.Group]++
	}
	for i, g := range b.groups {
		sel := i == b.gi
		label := g
		style := sBase
		switch {
		case sel && !b.onList:
			style = sCursor
		case sel:
			style = sSelected
		case g == "unfiled":
			style = sWarn
		}
		row := gap(label, fmt.Sprintf("%d", counts[g]), w-2)
		fmt.Fprintln(&sb, bar(sel, !b.onList)+style.Render(row))
	}
	return sb.String()
}

func (b Browse) listColumn(w int) string {
	var sb strings.Builder
	v := b.visible()
	title := strings.ToUpper(b.groups[b.gi])
	if title == "ALL" {
		title = "SKILLS"
	}
	fmt.Fprintln(&sb, gap(sHead.Render(title),
		sDim.Render(fmt.Sprintf("%d of %d", len(v), len(b.all))), w))
	if b.filtering || b.filter.Value() != "" {
		fmt.Fprintln(&sb, b.filter.View())
	} else {
		fmt.Fprintln(&sb)
	}
	if len(v) == 0 {
		fmt.Fprintln(&sb, sDim.Render("nothing matches "+b.filter.Value()))
		return sb.String()
	}
	// Scroll the window so the cursor stays on screen.
	rows := max(1, b.h-6)
	start := 0
	if b.si >= rows {
		start = b.si - rows + 1
	}
	for i := start; i < len(v) && i < start+rows; i++ {
		s := v[i]
		sel := i == b.si
		style := sBase
		switch {
		case sel && b.onList:
			style = sCursor
		case sel:
			style = sSelected
		}
		right := statusMark(s)
		fmt.Fprintln(&sb, bar(sel, b.onList)+style.Render(pad(s.Name, w-2-lipgloss.Width(right)))+right)
	}
	return sb.String()
}

// statusMark is the at-a-glance health of a skill, so browsing is also triage.
func statusMark(s skill.Skill) string {
	switch {
	case s.Description == "":
		return sErr.Render("err")
	case len(s.Description) > skill.MaxDescription:
		return sWarn.Render("long")
	case len(s.Tags) == 0:
		return sWarn.Render("untagged")
	}
	return sFaint.Render("ok")
}

func (b Browse) header() string {
	left := sTitle.Render("skill-issue") + sDim.Render("  "+
		fmt.Sprintf("%d skills · %d collections", len(b.all), len(b.groups)-1))
	right := sDim.Render(shortPath(b.root))
	line := gap(left, right, b.w)
	return line + "\n" + hrule(b.w)
}

func (b Browse) footer() string {
	if b.status != "" {
		return sAccent.Render(truncate(b.status, b.w))
	}
	if len(b.errs) > 0 {
		return sErr.Render(truncate(fmt.Sprintf("%d skill(s) failed to parse — run: skill-issue doctor", len(b.errs)), b.w))
	}
	return footerHelp(keysBrowse, b.w)
}

func shortPath(p string) string {
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}

func humanSize(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1fM", float64(n)/(1<<20))
	case n >= 1024:
		return fmt.Sprintf("%.1fk", float64(n)/1024)
	}
	return fmt.Sprintf("%dB", n)
}

var _ = filepath.Join
