package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mustEdit(path string, w, h int) Edit {
	e, cmd := NewEdit(path, w, h)
	if cmd != nil {
		e, _ = e.Update(cmd())
	}
	return e
}

// The editor must write what you typed and not lose it on a stray esc.
func TestEditSavesAndGuardsUnsavedChanges(t *testing.T) {
	path := filepath.Join(t.TempDir(), "SKILL.md")
	if err := os.WriteFile(path, []byte("---\nname: x\n---\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	e := mustEdit(path, 80, 24)

	e, _ = e.Update(keyMsg("!"))
	if !e.dirty() {
		t.Fatal("typing should mark the buffer modified")
	}

	if _, cmd := e.Update(keyMsg("esc")); cmd != nil {
		t.Error("esc with unsaved changes must warn, not leave")
	}
	e, _ = e.Update(keyMsg("esc"))
	if e.status == "" {
		t.Error("the warning should say why esc did nothing")
	}

	e, _ = e.Update(keyMsg("ctrl+s"))
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "!") {
		t.Errorf("ctrl+s did not save the edit; file is %q", b)
	}
	if e.dirty() {
		t.Error("buffer is still modified after a save")
	}
	if _, cmd := e.Update(keyMsg("esc")); cmd == nil {
		t.Error("esc after saving should go back")
	}
}

func TestEditFormats(t *testing.T) {
	path := filepath.Join(t.TempDir(), "SKILL.md")
	if err := os.WriteFile(path, []byte("---\nname: x\n---\n# T\n\n\n* a   \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	e := mustEdit(path, 80, 24)
	e, _ = e.Update(keyMsg("ctrl+f"))
	if got := e.ta.Value(); got != "---\nname: x\n---\n\n# T\n\n- a\n" {
		t.Errorf("ctrl+f gave %q", got)
	}
	e, _ = e.Update(keyMsg("ctrl+f"))
	if e.status != "already tidy" {
		t.Errorf("second format should be a no-op; status %q", e.status)
	}
}

// The preview is the point of the split, so look at it.
func TestEditPreviewRendersMarkdown(t *testing.T) {
	b, _ := render(t, 120, 30)
	s, _ := b.current()
	e := mustEdit(s.Path(), 120, 30)
	t.Logf("\n== edit, split ==\n%s\n", e.View())

	if !strings.Contains(e.pv.View(), s.Headings()[0]) {
		t.Errorf("preview does not show the first heading %q", s.Headings()[0])
	}
	if strings.Contains(e.pv.View(), "## ") {
		t.Error("preview is showing raw markdown, not rendered")
	}
	e, _ = e.Update(keyMsg("ctrl+p"))
	if e.split() {
		t.Error("ctrl+p should hide the preview")
	}
}
