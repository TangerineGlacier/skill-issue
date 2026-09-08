package skill

import "strings"

// Format tidies a SKILL.md: the whitespace rules everyone agrees on, and
// nothing else. Prose is never reflowed and frontmatter keys are left in the
// order they were written — a formatter that rewrites what you meant is one
// people turn off.
//
// Fenced code blocks are copied through untouched.
func Format(src string) string {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	head, body := splitFrontmatter(src)

	var out []string
	fenced := false
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			fenced = !fenced
			out = append(out, strings.TrimRight(line, " \t"))
			continue
		}
		if fenced {
			out = append(out, line)
			continue
		}
		line = strings.TrimRight(line, " \t")
		if b := strings.TrimLeft(line, " "); strings.HasPrefix(b, "* ") || strings.HasPrefix(b, "+ ") {
			line = line[:len(line)-len(b)] + "-" + b[1:]
		}
		blank := line == ""
		// One blank line is a paragraph break; more is just slack. A heading
		// always gets one above it, so sections do not run together.
		if blank && len(out) > 0 && out[len(out)-1] == "" {
			continue
		}
		if len(out) > 0 && out[len(out)-1] != "" &&
			(strings.HasPrefix(line, "#") || (!blank && strings.HasPrefix(out[len(out)-1], "#"))) {
			out = append(out, "")
		}
		if blank && len(out) == 0 {
			continue
		}
		out = append(out, line)
	}

	text := strings.TrimRight(strings.Join(out, "\n"), "\n") + "\n"
	if head == "" {
		return text
	}
	return head + "\n\n" + text
}

// splitFrontmatter returns the frontmatter block (delimiters included, trailing
// newline trimmed) and the rest. Frontmatter is YAML: leave it alone.
func splitFrontmatter(src string) (head, body string) {
	if !strings.HasPrefix(src, "---\n") {
		return "", src
	}
	end := strings.Index(src[4:], "\n---")
	if end < 0 {
		return "", src
	}
	cut := 4 + end + len("\n---")
	if nl := strings.IndexByte(src[cut:], '\n'); nl >= 0 {
		cut += nl + 1
	} else {
		cut = len(src)
	}
	return strings.TrimRight(src[:cut], "\n"), src[cut:]
}
