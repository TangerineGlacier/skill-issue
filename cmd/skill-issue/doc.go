package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/TangerineGlacier/skill-issue/skill"
)

// Doc is the read screen: outline and bundled files on the left, the rendered
// SKILL.md in a viewport on the right.
type Doc struct {
	s       skill.Skill
	vp      viewport.Model
	outline []string
	oi      int
	w, h    int
	status  string
}

const colOutline = 26

func NewDoc(s skill.Skill, w, h int) Doc {
	d := Doc{s: s, outline: s.Headings(), w: w, h: h}
	d.vp = viewport.New(d.bodyWidth(), max(1, h-4))
	d.vp.SetContent(renderMD(s.Body, d.bodyWidth()))
	return d
}

func (d Doc) bodyWidth() int {
	if d.w < 80 {
		return max(20, d.w-2)
	}
	return d.w - colOutline - 3
}

func (d Doc) Update(msg tea.Msg) (Doc, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.w, d.h = msg.Width, msg.Height
		d.vp.Width, d.vp.Height = d.bodyWidth(), max(1, d.h-4)
		d.vp.SetContent(renderMD(d.s.Body, d.bodyWidth()))
		return d, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "left", "h":
			return d, func() tea.Msg { return backMsg{} }
		case "tab", "n":
			d.jump(1)
			return d, nil
		case "shift+tab", "N":
			d.jump(-1)
			return d, nil
		case "e":
			path := d.s.Path()
			return d, func() tea.Msg { return openEditMsg{path} }
		case "o":
			d.status = "opening " + d.s.Path()
			return d, openCmd(d.s.Path())
		case "y":
			d.status = "path copied"
			return d, copyCmd(d.s.Path())
		case "r":
			d.status = "SKILL.md copied as prompt"
			return d, copyCmd("---\n" + d.s.Body)
		}
	}
	var cmd tea.Cmd
	d.vp, cmd = d.vp.Update(msg)
	return d, cmd
}

// jump scrolls to the next "## " heading and keeps the outline cursor in step.
func (d *Doc) jump(delta int) {
	if len(d.outline) == 0 {
		return
	}
	d.oi = clamp(d.oi+delta, 0, len(d.outline)-1)
	target := d.outline[d.oi]
	for i, line := range strings.Split(d.vp.View(), "\n") {
		_ = i
		_ = line
		break
	}
	// Count rendered lines up to the heading rather than guessing an offset.
	rendered := strings.Split(renderMD(d.s.Body, d.bodyWidth()), "\n")
	for i, line := range rendered {
		if strings.Contains(stripANSI(line), target) && strings.TrimSpace(stripANSI(line)) == target {
			d.vp.SetYOffset(max(0, i-1))
			return
		}
	}
}

func (d Doc) View() string {
	if d.w == 0 {
		return "loading…"
	}
	rows := max(1, d.h-4)
	body := block(d.vp.View(), d.bodyWidth(), rows)
	if d.w >= 80 {
		body = lipgloss.JoinHorizontal(lipgloss.Top,
			block(d.sidebar(colOutline), colOutline, rows), vrule(rows), body)
	}
	return lipgloss.JoinVertical(lipgloss.Left, d.header(), body, hrule(d.w), d.footer())
}

func (d Doc) header() string {
	left := sTitle.Render(d.s.Name)
	if len(d.s.Tags) > 0 {
		left += sDim.Render("  " + strings.Join(d.s.Tags, " · "))
	}
	right := sDim.Render(fmt.Sprintf("%s · %s", humanSize(d.s.Size), ago(d.s.ModTime)))
	return gap(left, right, d.w) + "\n" + hrule(d.w)
}

func (d Doc) sidebar(w int) string {
	var sb strings.Builder
	fmt.Fprintln(&sb, sHead.Render("OUTLINE"))
	if len(d.outline) == 0 {
		fmt.Fprintln(&sb, sDim.Render("no ## headings"))
	}
	for i, h := range d.outline {
		fmt.Fprintln(&sb, bar(i == d.oi, true)+
			pick(i == d.oi, sCursor, sBase).Render(truncate(h, w-2)))
	}
	fmt.Fprintln(&sb)
	fmt.Fprintln(&sb, sHead.Render("FILES"))
	for _, f := range d.s.Files {
		style := sBase
		if f.Rel == "SKILL.md" {
			style = sAccent
		}
		fmt.Fprintln(&sb, "  "+style.Render(gap(truncate(f.Rel, w-9), humanSize(f.Size), w-2)))
	}
	return sb.String()
}

func pick(cond bool, a, b lipgloss.Style) lipgloss.Style {
	if cond {
		return a
	}
	return b
}

func (d Doc) footer() string {
	if d.status != "" {
		return sAccent.Render(truncate(d.status, d.w))
	}
	pct := fmt.Sprintf("%3.0f%%", d.vp.ScrollPercent()*100)
	return gap(footerHelp(keysDoc, d.w-6), sDim.Render(pct), d.w)
}

// stripANSI removes styling so heading text can be matched in rendered output.
func stripANSI(s string) string {
	var b strings.Builder
	esc := false
	for _, r := range s {
		switch {
		case r == 0x1b:
			esc = true
		case esc && (r == 'm' || r == 'K'):
			esc = false
		case !esc:
			b.WriteRune(r)
		}
	}
	return b.String()
}
