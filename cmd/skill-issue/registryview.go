package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/TangerineGlacier/skill-issue/skill"
)

type (
	remotesMsg struct {
		rs   []skill.Remote
		errs []error
	}
	installedMsg struct {
		name, dst string
		err       error
	}
)

// Registry is the download screen: remote skills on the left, where each
// selected one will land on the right, and a progress bar while installing.
type Registry struct {
	root      string
	sources   []string
	remotes   []skill.Remote
	sel       map[string]bool
	cursor    int
	loading   bool
	spin      spinner.Model
	prog      progress.Model
	queue     []skill.Remote
	retag     textinput.Model
	retagging bool
	done      int
	errs      []error
	status    string
	w, h      int
}

func NewRegistry(root string, w, h int) (Registry, tea.Cmd) {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = sAccent

	ti := textinput.New()
	ti.Prompt = "file under skills/"
	ti.Placeholder = "tag"
	ti.PromptStyle = sAccent
	ti.CharLimit = 40

	r := Registry{
		root: root, sources: skill.Sources(root), sel: map[string]bool{}, retag: ti,
		spin: sp, prog: progress.New(progress.WithoutPercentage(),
			progress.WithSolidFill("#38d9a9")),
		loading: true, w: w, h: h,
	}
	if len(r.sources) == 0 {
		r.loading = false
		r.status = "no sources — add owner/repo lines to sources.txt"
		return r, nil
	}
	return r, tea.Batch(r.spin.Tick, r.fetch())
}

func (r Registry) fetch() tea.Cmd {
	root, sources := r.root, r.sources
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		local, _ := skill.LoadAll(root)
		rs, errs := skill.Fetch(ctx, sources, local)
		return remotesMsg{rs, errs}
	}
}

func (r Registry) installNext() tea.Cmd {
	if len(r.queue) == 0 {
		return nil
	}
	next, root := r.queue[0], r.root
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		dst, err := skill.Install(ctx, root, next)
		return installedMsg{name: next.Name, dst: dst, err: err}
	}
}

func (r Registry) Update(msg tea.Msg) (Registry, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		r.w, r.h = msg.Width, msg.Height
		r.prog.Width = max(10, r.previewWidth()-4)

	case spinner.TickMsg:
		if !r.loading && len(r.queue) == 0 {
			return r, nil
		}
		var cmd tea.Cmd
		r.spin, cmd = r.spin.Update(msg)
		return r, cmd

	case remotesMsg:
		r.loading = false
		r.remotes, r.errs = msg.rs, msg.errs
		if len(r.remotes) == 0 && len(r.errs) == 0 {
			r.status = "no skills found in " + strings.Join(r.sources, ", ")
		}
		return r, nil

	case installedMsg:
		r.done++
		if len(r.queue) > 0 {
			r.queue = r.queue[1:]
		}
		if msg.err != nil {
			r.errs = append(r.errs, fmt.Errorf("%s: %w", msg.name, msg.err))
		}
		if len(r.queue) > 0 {
			return r, r.installNext()
		}
		r.status = fmt.Sprintf("installed %d, %d failed — press esc to see them", r.done-len(r.errs), len(r.errs))
		return r, func() tea.Msg { return reloadMsg{} }

	case tea.KeyMsg:
		if len(r.queue) > 0 {
			return r, nil // no input while writing to disk
		}
		if r.retagging {
			switch msg.String() {
			case "esc":
				r.retagging = false
				r.retag.Blur()
			case "enter":
				tag := strings.TrimSpace(r.retag.Value())
				r.retagging = false
				r.retag.Blur()
				if tag == "" {
					return r, nil
				}
				n := 0
				targets := r.selected()
				if len(targets) == 0 {
					if c, ok := r.current(); ok {
						targets = []skill.Remote{c}
					}
				}
				for _, t := range targets {
					for i := range r.remotes {
						if r.remotes[i].Name == t.Name {
							r.remotes[i].Retag = tag
							n++
						}
					}
				}
				r.status = fmt.Sprintf("%d skill(s) will be filed under skills/%s/", n, tag)
			default:
				var cmd tea.Cmd
				r.retag, cmd = r.retag.Update(msg)
				return r, cmd
			}
			return r, nil
		}
		switch msg.String() {
		case "q", "esc":
			return r, func() tea.Msg { return backMsg{} }
		case "j", "down":
			r.cursor = clamp(r.cursor+1, 0, max(0, len(r.remotes)-1))
		case "k", "up":
			r.cursor = clamp(r.cursor-1, 0, max(0, len(r.remotes)-1))
		case " ":
			if c, ok := r.current(); ok {
				r.sel[c.Name] = !r.sel[c.Name]
			}
		case "a":
			for _, c := range r.remotes {
				if !c.Installed {
					r.sel[c.Name] = true
				}
			}
		case "t":
			r.retagging = true
			r.retag.SetValue("")
			r.status = ""
			return r, r.retag.Focus()
		case "r":
			r.loading, r.remotes = true, nil
			return r, tea.Batch(r.spin.Tick, r.fetch())
		case "enter":
			r.queue = r.selected()
			if len(r.queue) == 0 {
				r.status = "nothing selected — space to select, a for all"
				return r, nil
			}
			r.done, r.errs = 0, nil
			return r, tea.Batch(r.spin.Tick, r.installNext())
		}
	}
	return r, nil
}

func (r Registry) current() (skill.Remote, bool) {
	if r.cursor < 0 || r.cursor >= len(r.remotes) {
		return skill.Remote{}, false
	}
	return r.remotes[r.cursor], true
}

func (r Registry) selected() []skill.Remote {
	var out []skill.Remote
	for _, c := range r.remotes {
		if r.sel[c.Name] {
			out = append(out, c)
		}
	}
	return out
}

func (r Registry) previewWidth() int {
	if r.w < 100 {
		return 0
	}
	return min(38, r.w/3)
}

func (r Registry) View() string {
	if r.w == 0 {
		return "loading…"
	}
	rows := max(1, r.h-4)
	pw := r.previewWidth()
	lw := r.w - pw
	if pw > 0 {
		lw -= 3
	}
	cols := []string{block(r.list(rows), lw, rows)}
	if pw > 0 {
		cols = append(cols, vrule(rows), block(r.target(pw), pw, rows))
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		r.header(), lipgloss.JoinHorizontal(lipgloss.Top, cols...), hrule(r.w), r.footer())
}

func (r Registry) header() string {
	left := sTitle.Render("registry")
	if r.loading {
		left += "  " + r.spin.View() + sDim.Render(" fetching")
	} else {
		left += sDim.Render(fmt.Sprintf("  %d available · %d selected", len(r.remotes), len(r.selected())))
	}
	return gap(left, sDim.Render(strings.Join(r.sources, "  ")), r.w) +
		"\n" + hrule(r.w)
}

func (r Registry) list(rows int) string {
	var b strings.Builder
	if r.loading {
		return "\n  " + sDim.Render("reading "+strings.Join(r.sources, ", ")+"…")
	}
	if len(r.errs) > 0 {
		for _, e := range r.errs {
			fmt.Fprintln(&b, sErr.Render("  "+truncate(e.Error(), r.w-4)))
		}
		fmt.Fprintln(&b)
	}
	if len(r.remotes) == 0 {
		fmt.Fprintln(&b, sDim.Render("  nothing to show"))
		return b.String()
	}
	nameW := 26
	start := 0
	visible := rows - 1
	if r.cursor >= visible {
		start = r.cursor - visible + 1
	}
	for i := start; i < len(r.remotes) && i < start+visible; i++ {
		c := r.remotes[i]
		box := "[ ]"
		style := sBase
		switch {
		case c.Installed:
			box, style = sFaint.Render("[·]"), sDim
		case r.sel[c.Name]:
			box = sAccent.Render("[x]")
		}
		right := sFaint.Render("local")
		if !c.Installed {
			right = sDim.Render(truncate(strings.Join(c.Tags, ", "), 24))
		}
		line := box + " " + style.Render(pad(c.Name, nameW))
		fmt.Fprintln(&b, bar(i == r.cursor, true)+gap(line, right, r.w-r.previewWidth()-6))
	}
	return b.String()
}

func (r Registry) target(w int) string {
	var b strings.Builder
	fmt.Fprintln(&b, sHead.Render("TARGET FOLDER"))
	fmt.Fprintln(&b)
	sel := r.selected()
	if len(sel) == 0 {
		if c, ok := r.current(); ok {
			sel = []skill.Remote{c}
			fmt.Fprintln(&b, sDim.Render("(nothing selected — showing the"))
			fmt.Fprintln(&b, sDim.Render("row under the cursor)"))
			fmt.Fprintln(&b)
		}
	}
	for _, c := range sel {
		tag := c.Tag()
		via := "tag"
		switch {
		case c.Retag != "":
			via = "retag"
		case len(c.Tags) == 0:
			via = "no tag"
		}
		fmt.Fprintln(&b, sBase.Render(truncate(c.Name, w)))
		line := sDim.Render("  "+via+" → ") + sAccent.Render(truncate("skills/"+tag+"/", w-12))
		if tag == "unfiled" {
			line = sDim.Render("  "+via+" → ") + sWarn.Render("skills/unfiled/")
		}
		fmt.Fprintln(&b, line)
		if c.Installed {
			fmt.Fprintln(&b, sWarn.Render("  ! exists, backs up to .bak"))
		}
		fmt.Fprintln(&b, sFaint.Render("  "+plural(len(c.Files), "file")))
		fmt.Fprintln(&b)
	}
	if n := len(r.queue); n > 0 {
		total := n + r.done
		fmt.Fprintln(&b, r.prog.ViewAs(float64(r.done)/float64(max(1, total))))
		fmt.Fprintln(&b, sAccent.Render(fmt.Sprintf("%s installing %d of %d", r.spin.View(), r.done+1, total)))
	}
	return b.String()
}

func (r Registry) footer() string {
	if r.retagging {
		return r.retag.View() + sDim.Render("/   ⏎ apply · esc cancel")
	}
	if r.status != "" {
		return sAccent.Render(truncate(r.status, r.w))
	}
	return footerHelp(keysRegistry, r.w)
}
