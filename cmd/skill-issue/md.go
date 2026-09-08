package main

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// listRe matches a bullet or an ordered item, with its indentation: sublists
// keep their shape instead of flattening into the paragraph above.
var listRe = regexp.MustCompile(`^(\s*)(?:[-*+]|(\d+)[.)])\s+(.*)$`)

// renderMD is a deliberately small markdown renderer: headings, fences, rules,
// bullets and wrapped paragraphs. A full markdown library would bring its own
// theme to fight with ours, and a SKILL.md does not use more than this.
func renderMD(src string, w int) string {
	if w < 20 {
		w = 20
	}
	var out []string
	fenced := false
	para := []string{}

	flush := func() {
		if len(para) == 0 {
			return
		}
		text := inline(strings.Join(para, " "))
		out = append(out, lipgloss.NewStyle().Width(w).Foreground(fg).Render(text))
		para = para[:0]
	}

	lines := strings.Split(src, "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		// A table is a block, not a line: gather it whole and lay it out.
		if !fenced && strings.HasPrefix(line, "|") {
			flush()
			j := i
			for j < len(lines) && strings.HasPrefix(lines[j], "|") {
				j++
			}
			out = append(out, renderTable(lines[i:j], w))
			i = j - 1
			continue
		}
		switch {
		case strings.HasPrefix(line, "```"):
			flush()
			fenced = !fenced
			out = append(out, sFaint.Render(strings.Repeat("╌", min(w, 40))))

		case fenced:
			out = append(out, sDim.Render("  "+truncate(line, w-2)))

		case strings.HasPrefix(line, "## "):
			flush()
			out = append(out, "", sHead.Render(strings.TrimSpace(line[3:])))

		case strings.HasPrefix(line, "### "):
			flush()
			out = append(out, "", sTitle.Render(strings.TrimSpace(line[4:])))

		case strings.HasPrefix(line, "# "):
			flush()
			out = append(out, sTitle.Render(strings.TrimSpace(line[2:])))

		case listRe.MatchString(line):
			flush()
			m := listRe.FindStringSubmatch(line)
			indent, marker, text := m[1], "· ", m[3]
			if m[2] != "" {
				marker = m[2] + ". " // an ordered list keeps its numbering
			}
			pad := strings.Repeat(" ", len(indent)+len(marker))
			body := lipgloss.NewStyle().Width(max(10, w-len(pad))).Foreground(fg).
				Render(inline(text))
			out = append(out, indent+sAccent.Render(marker)+
				strings.ReplaceAll(body, "\n", "\n"+pad))

		case strings.TrimSpace(line) == "":
			flush()
			out = append(out, "")

		default:
			para = append(para, strings.TrimSpace(line))
		}
	}
	flush()
	return strings.Join(out, "\n")
}

// renderTable lays a pipe table out on aligned columns. Cells wrap by
// truncation rather than reflow: a table that needs reflowing is prose.
func renderTable(rows []string, w int) string {
	var cells [][]string
	for _, r := range rows {
		r = strings.Trim(strings.TrimSpace(r), "|")
		var row []string
		sep := true
		for _, c := range strings.Split(r, "|") {
			c = strings.TrimSpace(c)
			if strings.Trim(c, "-: ") != "" {
				sep = false
			}
			row = append(row, c)
		}
		if sep {
			continue // the |---|---| divider
		}
		cells = append(cells, row)
	}
	if len(cells) == 0 {
		return ""
	}

	n := 0
	for _, r := range cells {
		n = max(n, len(r))
	}
	widths := make([]int, n)
	for _, r := range cells {
		for i, c := range r {
			widths[i] = max(widths[i], lipgloss.Width(c))
		}
	}
	// Share the overflow out of the widest column first.
	for {
		total := len(widths) * 2
		for _, x := range widths {
			total += x
		}
		if total <= w {
			break
		}
		widest := 0
		for i, x := range widths {
			if x > widths[widest] {
				widest = i
			}
		}
		if widths[widest] <= 8 {
			break
		}
		widths[widest] -= total - w
		if widths[widest] < 8 {
			widths[widest] = 8
		}
	}

	var b strings.Builder
	for ri, r := range cells {
		var line strings.Builder
		for i := 0; i < n; i++ {
			c := ""
			if i < len(r) {
				c = r[i]
			}
			style := sBase
			if ri == 0 {
				style = sHead
			}
			line.WriteString(style.Render(pad(inline(c), widths[i])) + "  ")
		}
		b.WriteString(strings.TrimRight(line.String(), " ") + "\n")
		if ri == 0 {
			total := 0
			for _, x := range widths {
				total += x + 2
			}
			b.WriteString(sVRule.Render(strings.Repeat("─", min(total-2, w))) + "\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// inline handles the three markers that actually appear in a SKILL.md.
func inline(s string) string {
	s = swap(s, "**", sTitle)
	s = swap(s, "`", sAccent)
	return s
}

// swap styles the text between paired occurrences of mark.
func swap(s, mark string, style lipgloss.Style) string {
	parts := strings.Split(s, mark)
	if len(parts) < 3 {
		return s
	}
	var b strings.Builder
	for i, p := range parts {
		if i%2 == 1 {
			b.WriteString(style.Render(p))
		} else {
			b.WriteString(p)
		}
	}
	return b.String()
}
