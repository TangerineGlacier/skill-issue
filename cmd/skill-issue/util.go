package main

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
)

// fuzzy reports whether pattern occurs in s as a case-insensitive subsequence,
// which is all "type a few letters to find a skill" needs. No ranking: the list
// is already sorted, and re-ordering under the cursor is worse than no ranking.
func fuzzy(s, pattern string) bool {
	if pattern == "" {
		return true
	}
	si, pi := 0, 0
	sr, pr := []rune(s), []rune(pattern)
	for si < len(sr) && pi < len(pr) {
		if unicode.ToLower(sr[si]) == unicode.ToLower(pr[pi]) {
			pi++
		}
		si++
	}
	return pi == len(pr)
}

// truncate cuts to w display cells, with an ellipsis when it had to cut.
func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= w {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && lipgloss.Width(string(r))+1 > w {
		r = r[:len(r)-1]
	}
	return string(r) + "…"
}

// pad right-pads to exactly w cells so columns stay aligned once joined.
func pad(s string, w int) string {
	s = truncate(s, w)
	if d := w - lipgloss.Width(s); d > 0 {
		return s + strings.Repeat(" ", d)
	}
	return s
}

// gap lays a label out left and a value right within w cells — the header and
// every "name ......... count" row in the app.
func gap(left, right string, w int) string {
	lw, rw := lipgloss.Width(left), lipgloss.Width(right)
	if lw+rw+1 > w {
		left = truncate(left, max(0, w-rw-1))
		lw = lipgloss.Width(left)
	}
	return left + strings.Repeat(" ", max(1, w-lw-rw)) + right
}

// hrule is a full-width horizontal rule. Two of them frame every screen: one
// under the header, one over the status bar, so the keys at the bottom read as
// a bar rather than as a stray line of text.
func hrule(w int) string {
	return sVRule.Render(strings.Repeat("─", w))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
