package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/TangerineGlacier/skill-issue/skill"
)

// Doctor lists everything wrong with the library, worst first, with the
// selected problem explained underneath.
type Doctor struct {
	root     string
	problems []skill.Problem
	cursor   int
	w, h     int
	status   string
}

func NewDoctor(root string, w, h int) Doctor {
	d := Doctor{root: root, w: w, h: h}
	return d.reload()
}

func (d Doctor) reload() Doctor {
	skills, errs := skill.LoadAll(d.root)
	d.problems = skill.Check(skills, errs)
	d.cursor = clamp(d.cursor, 0, max(0, len(d.problems)-1))
	return d
}

func (d Doctor) counts() (errs, warns, fixable int) {
	for _, p := range d.problems {
		if p.Level == skill.Error {
			errs++
		} else {
			warns++
		}
		if p.Fixable {
			fixable++
		}
	}
	return
}

func (d Doctor) Update(msg tea.Msg) (Doctor, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.w, d.h = msg.Width, msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return d, func() tea.Msg { return backMsg{} }
		case "j", "down":
			d.cursor = clamp(d.cursor+1, 0, max(0, len(d.problems)-1))
			d.status = ""
		case "k", "up":
			d.cursor = clamp(d.cursor-1, 0, max(0, len(d.problems)-1))
			d.status = ""
		case "f":
			return d.fixOne()
		case "F":
			return d.fixAll()
		case "r":
			d.status = "rechecked"
			return d.reload(), nil
		case "o":
			if p, ok := d.current(); ok {
				return d, openCmd(p.Dir)
			}
		case "e":
			if p, ok := d.current(); ok {
				path := p.Dir + "/SKILL.md"
				return d, func() tea.Msg { return openEditMsg{path} }
			}
		}
	}
	return d, nil
}

func (d Doctor) current() (skill.Problem, bool) {
	if d.cursor < 0 || d.cursor >= len(d.problems) {
		return skill.Problem{}, false
	}
	return d.problems[d.cursor], true
}

func (d Doctor) fixOne() (Doctor, tea.Cmd) {
	p, ok := d.current()
	if !ok {
		return d, nil
	}
	if !p.Fixable {
		d.status = p.Code + " has no automatic fix — " + p.Msg
		return d, nil
	}
	what, err := skill.Fix(p)
	if err != nil {
		d.status = "fix failed: " + err.Error()
		return d, nil
	}
	d.status = p.Skill + ": " + what
	return d.reload(), nil
}

func (d Doctor) fixAll() (Doctor, tea.Cmd) {
	var done, failed int
	// Re-check between fixes: a move invalidates every later Dir.
	for {
		p, ok := firstFixable(d.problems)
		if !ok {
			break
		}
		if _, err := skill.Fix(p); err != nil {
			failed++
			d.problems = without(d.problems, p)
			continue
		}
		done++
		d = d.reload()
		if done > 200 {
			break // a fix that does not clear its own problem would loop forever
		}
	}
	d = d.reload()
	d.status = fmt.Sprintf("fixed %d, %d failed, %d left needing a human", done, failed, len(d.problems))
	return d, nil
}

func firstFixable(ps []skill.Problem) (skill.Problem, bool) {
	for _, p := range ps {
		if p.Fixable {
			return p, true
		}
	}
	return skill.Problem{}, false
}

func without(ps []skill.Problem, drop skill.Problem) []skill.Problem {
	out := ps[:0]
	for _, p := range ps {
		if p.Dir != drop.Dir || p.Code != drop.Code {
			out = append(out, p)
		}
	}
	return out
}

func (d Doctor) View() string {
	if d.w == 0 {
		return "loading…"
	}
	detailRows := 6
	tableRows := max(1, d.h-4-detailRows-1)
	parts := []string{
		d.header(),
		block(d.table(tableRows), d.w, tableRows),
		hrule(d.w),
		block(d.detail(), d.w, detailRows),
		hrule(d.w),
		d.footer(),
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (d Doctor) header() string {
	errs, warns, fixable := d.counts()
	left := sTitle.Render("doctor") + "  "
	if len(d.problems) == 0 {
		left += sAccent.Render("all clear")
	} else {
		left += sErr.Render(fmt.Sprintf("%d err", errs)) + sDim.Render("  ") +
			sWarn.Render(fmt.Sprintf("%d warn", warns))
	}
	right := sDim.Render(fmt.Sprintf("%d auto-fixable", fixable))
	return gap(left, right, d.w) + "\n" + hrule(d.w)
}

func (d Doctor) table(rows int) string {
	if len(d.problems) == 0 {
		return "\n" + sAccent.Render("  Nothing to fix. Every skill has a name, a description, "+
			"tags that match its folder, and no orphan references.")
	}
	nameW := 22
	fixW := 8
	msgW := max(10, d.w-2-nameW-fixW-4)

	var sb strings.Builder
	fmt.Fprintln(&sb, "  "+sDim.Render(pad("SKILL", nameW)+pad("PROBLEM", msgW+2)+"FIX"))

	start := 0
	if d.cursor >= rows-1 {
		start = d.cursor - rows + 2
	}
	for i := start; i < len(d.problems) && i < start+rows-1; i++ {
		p := d.problems[i]
		mark := sWarn.Render("w")
		if p.Level == skill.Error {
			mark = sErr.Render("e")
		}
		fix := sFaint.Render("manual")
		if p.Fixable {
			fix = sAccent.Render("auto")
		}
		style := pick(i == d.cursor, sSelected, sBase)
		fmt.Fprintln(&sb, bar(i == d.cursor, true)+mark+" "+
			style.Render(pad(p.Skill, nameW-2)+pad(p.Msg, msgW))+"  "+fix)
	}
	return sb.String()
}

func (d Doctor) detail() string {
	p, ok := d.current()
	if !ok {
		return ""
	}
	head := sHead.Render("DETAIL") + sDim.Render("   "+p.Skill+" · "+p.Code)
	body := lipgloss.NewStyle().Width(d.w - 2).Foreground(fg).Render(p.Detail)
	return head + "\n" + body
}

func (d Doctor) footer() string {
	if d.status != "" {
		return sAccent.Render(truncate(d.status, d.w))
	}
	return footerHelp(keysDoctor, d.w)
}
