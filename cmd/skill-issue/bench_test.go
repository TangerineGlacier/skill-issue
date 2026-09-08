package main

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TangerineGlacier/skill-issue/skill"
)

// These exist to answer one question: is any View() expensive enough to need
// the caching the bubbletea-designer skill recommends? Measured on this
// library: browse 0.35ms, doc-jump 0.30ms. They are not. The only expensive
// read is the transcript scan at ~180ms, which is why History loads in a
// command instead of in Update. Re-run before adding a cache to anything.
//
//	go test ./tui -run xxx -bench . -benchtime 20x

func benchRoot(tb testing.TB) string {
	tb.Helper()
	wd, _ := os.Getwd()
	return filepath.Join(wd, "..", "..", "skills")
}

func BenchmarkBrowseView(b *testing.B) {
	m := NewBrowse(benchRoot(b))
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkDocJump(b *testing.B) {
	br := NewBrowse(benchRoot(b))
	br, _ = br.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	br, _ = br.Update(keyMsg("tab"))
	s, _ := br.current()
	d := NewDoc(s, 120, 40)
	for i := 0; i < b.N; i++ {
		d, _ = d.Update(keyMsg("tab"))
	}
}

func BenchmarkHistoryScan(b *testing.B) {
	for i := 0; i < b.N; i++ {
		skill.History(benchRoot(b))
	}
}
