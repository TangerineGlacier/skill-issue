package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/TangerineGlacier/skill-issue/skill"
)

// Field order in the form. The dry-run pane below updates on every keystroke.
const (
	fName = iota
	fDescription
	fTags
	fTemplate
	fBundle
	nFields
)

var bundleDirs = []string{"references", "scripts", "assets"}

// NewSkill is the create form: a slice of inputs plus a focus index, with a
// live plan of exactly what will be written underneath.
type NewSkill struct {
	root, templates string
	existing        []skill.Skill

	inputs    []textinput.Model
	focus     int
	tmplNames []string
	tmpl      int
	bundle    map[string]bool
	bcur      int

	plan   []skill.PlannedFile
	errs   []string
	status string
	w, h   int
}

func NewNewSkill(root, templates string, w, h int) NewSkill {
	existing, _ := skill.LoadAll(root)
	n := NewSkill{
		root: root, templates: templates, existing: existing,
		tmplNames: skill.TemplateNames(templates),
		bundle:    map[string]bool{},
		w:         w, h: h,
	}
	labels := []string{"my-new-skill", "One sentence: when should the model reach for this?", "docs, architecture"}
	for i := range labels {
		ti := textinput.New()
		ti.Placeholder = labels[i]
		ti.Prompt = ""
		ti.CharLimit = skill.MaxDescription
		ti.Width = max(20, w/2)
		ti.TextStyle = sBase
		ti.PlaceholderStyle = sFaint
		n.inputs = append(n.inputs, ti)
	}
	// doc-writer is the template most skills should use.
	for i, t := range n.tmplNames {
		if t == "doc-writer" {
			n.tmpl = i
		}
	}
	n.inputs[fName].Focus()
	return n.recompute()
}

func (n NewSkill) draft() skill.Draft {
	var tags []string
	for _, t := range strings.Split(n.inputs[fTags].Value(), ",") {
		if t = strings.TrimSpace(t); t != "" {
			tags = append(tags, t)
		}
	}
	var bundle []string
	for _, d := range bundleDirs {
		if n.bundle[d] {
			bundle = append(bundle, d)
		}
	}
	return skill.Draft{
		Name:        strings.TrimSpace(n.inputs[fName].Value()),
		Description: strings.TrimSpace(n.inputs[fDescription].Value()),
		Tags:        tags,
		Template:    n.tmplNames[n.tmpl],
		Bundle:      bundle,
	}
}

func (n NewSkill) recompute() NewSkill {
	d := n.draft()
	n.errs = d.Validate(n.root, n.existing)
	n.plan = nil
	// A plan is a promise about what enter will do, so an invalid draft has no
	// plan at all rather than a plan you are not allowed to run.
	if len(n.errs) == 0 {
		if p, err := d.Plan(n.root, n.templates); err == nil {
			n.plan = p
		} else {
			n.errs = append(n.errs, err.Error())
		}
	}
	return n
}

func (n NewSkill) Update(msg tea.Msg) (NewSkill, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		n.w, n.h = msg.Width, msg.Height
		for i := range n.inputs {
			n.inputs[i].Width = max(20, n.w/2)
		}
		return n, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return n, func() tea.Msg { return backMsg{} }

		case "tab", "down":
			n.focus = (n.focus + 1) % nFields
			return n.refocus(), nil
		case "shift+tab", "up":
			n.focus = (n.focus + nFields - 1) % nFields
			return n.refocus(), nil

		case "left", "right", " ":
			switch n.focus {
			case fTemplate:
				d := 1
				if msg.String() == "left" {
					d = -1
				}
				n.tmpl = (n.tmpl + len(n.tmplNames) + d) % len(n.tmplNames)
				return n.recompute(), nil
			case fBundle:
				if msg.String() == " " {
					d := bundleDirs[n.bundleCursor()]
					n.bundle[d] = !n.bundle[d]
				} else {
					n.bcur = clamp(n.bcur+map[bool]int{true: -1, false: 1}[msg.String() == "left"], 0, len(bundleDirs)-1)
				}
				return n.recompute(), nil
			}

		case "enter":
			if len(n.errs) > 0 {
				n.status = n.errs[0]
				return n, nil
			}
			if err := skill.Apply(n.plan); err != nil {
				n.status = "failed: " + err.Error()
				return n, nil
			}
			created := n.plan[0].Path
			return n, func() tea.Msg { return createdMsg{path: created} }
		}
	}

	if n.focus <= fTags {
		var cmd tea.Cmd
		n.inputs[n.focus], cmd = n.inputs[n.focus].Update(msg)
		return n.recompute(), cmd
	}
	return n, nil
}

func (n NewSkill) bundleCursor() int { return clamp(n.bcur, 0, len(bundleDirs)-1) }

func (n NewSkill) refocus() NewSkill {
	for i := range n.inputs {
		n.inputs[i].Blur()
	}
	if n.focus <= fTags {
		n.inputs[n.focus].Focus()
	}
	n.status = ""
	return n
}

func (n NewSkill) View() string {
	if n.w == 0 {
		return "loading…"
	}
	labelW := 14
	var b strings.Builder

	row := func(i int, label, value string) {
		style := sDim
		if n.focus == i {
			style = sAccent
		}
		fmt.Fprintln(&b, bar(n.focus == i, true)+style.Render(pad(label, labelW))+value)
	}
	row(fName, "name", n.inputs[fName].View())
	fmt.Fprintln(&b)
	row(fDescription, "description", n.inputs[fDescription].View())
	// A single-line input scrolls to the cursor, so the start of a long
	// description is invisible in the field itself. Echo it wrapped underneath —
	// this is the field that decides whether the skill ever loads, so it has to
	// be readable while you write it.
	indent := strings.Repeat(" ", labelW+2)
	if v := n.inputs[fDescription].Value(); v != "" {
		wrapped := lipgloss.NewStyle().Width(n.w - labelW - 22).Foreground(dim).Render(v)
		for _, l := range strings.Split(wrapped, "\n") {
			fmt.Fprintln(&b, indent+l)
		}
	}
	fmt.Fprintln(&b, indent+sFaint.Render(
		fmt.Sprintf("%d / %d", len(n.inputs[fDescription].Value()), skill.MaxDescription)))
	fmt.Fprintln(&b)
	row(fTags, "tags", n.inputs[fTags].View())
	fmt.Fprintln(&b, indent+sFaint.Render("the first tag decides the folder"))
	fmt.Fprintln(&b)
	row(fTemplate, "template", radios(n.tmplNames, n.tmpl))
	fmt.Fprintln(&b)
	row(fBundle, "bundle", checks(bundleDirs, n.bundle, n.bundleCursor(), n.focus == fBundle))

	// The form grows as the description wraps, so split on what it actually
	// takes rather than a fixed number of rows.
	rows := max(1, n.h-4)
	form := strings.TrimRight(b.String(), "\n")
	top := clamp(strings.Count(form, "\n")+1, 1, max(1, rows-4))
	return lipgloss.JoinVertical(lipgloss.Left,
		n.header(),
		block(form, n.w, top),
		hrule(n.w),
		block(n.planView(), n.w, rows-top-1),
		hrule(n.w),
		n.footer())
}

func radios(opts []string, sel int) string {
	var out []string
	for i, o := range opts {
		if i == sel {
			out = append(out, sAccent.Render("(•) "+o))
		} else {
			out = append(out, sDim.Render("( ) "+o))
		}
	}
	return strings.Join(out, "  ")
}

func checks(opts []string, on map[string]bool, cursor int, focused bool) string {
	var out []string
	for i, o := range opts {
		mark := "[ ]"
		style := sDim
		if on[o] {
			mark, style = "[x]", sAccent
		}
		s := style.Render(mark + " " + o)
		if focused && i == cursor {
			s = sCursor.Render(mark + " " + o)
		}
		out = append(out, s)
	}
	return strings.Join(out, "  ")
}

func (n NewSkill) planView() string {
	var b strings.Builder
	if len(n.errs) > 0 {
		fmt.Fprintln(&b, sHead.Render("NOT READY"))
		for _, e := range n.errs {
			fmt.Fprintln(&b, sErr.Render("  · ")+sBase.Render(truncate(e, n.w-4)))
		}
		return b.String()
	}
	fmt.Fprintln(&b, sHead.Render("WILL WRITE")+sDim.Render("   nothing is on disk until you press enter"))
	for _, w := range n.plan {
		rel := strings.TrimPrefix(w.Path, n.root+"/")
		fmt.Fprintln(&b, "  "+sAccent.Render("+ ")+
			pad(truncate("skills/"+rel, n.w-24), n.w-22)+sDim.Render(w.Note))
	}
	return b.String()
}

func (n NewSkill) header() string {
	left := sTitle.Render("new skill")
	right := sDim.Render(fmt.Sprintf("%d existing", len(n.existing)))
	return gap(left, right, n.w) + "\n" + hrule(n.w)
}

func (n NewSkill) footer() string {
	if n.status != "" {
		return sErr.Render(truncate(n.status, n.w))
	}
	return footerHelp(keysNew, n.w)
}
