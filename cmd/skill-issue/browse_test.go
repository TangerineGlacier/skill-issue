package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TangerineGlacier/skill-issue/skill"
)

var wd, _ = os.Getwd()

// repoRoot is the skills/ dir of this repository — the tests run against the
// real library, so a broken skill breaks the build.
func repoSkills(t *testing.T) string    { return repoDir(t, "skills") }
func repoTemplates(t *testing.T) string { return repoDir(t, "templates") }

// repoDir resolves a directory at the repo root. Tests run from cmd/si, so
// the root is two levels up.
func repoDir(t *testing.T, name string) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(wd, "..", "..", name)
}

// render drives the model like the runtime does: size it, then feed keys.
func render(t *testing.T, w, h int, keys ...string) (Browse, string) {
	t.Helper()
	b := NewBrowse(repoSkills(t))
	b, _ = b.Update(tea.WindowSizeMsg{Width: w, Height: h})
	for _, k := range keys {
		b, _ = b.Update(keyMsg(k))
	}
	return b, b.View()
}

func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEscape}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestBrowseView(t *testing.T) {
	cases := []struct {
		name  string
		w, h  int
		keys  []string
		group string // navigate to this collection rather than counting keystrokes
		want  []string
		avoid []string
	}{
		{
			name: "wide shows all three columns",
			w:    120, h: 30,
			want: []string{"skill-issue", "COLLECTIONS", "docs", "write-adr",
				"DESCRIPTION", "OUTLINE", "filter", "doctor", "│"},
		},
		{
			name: "preview drops below 100 columns",
			w:    90, h: 30,
			want:  []string{"COLLECTIONS", "write-adr"},
			avoid: []string{"DESCRIPTION"},
		},
		{
			name: "list only below 70 columns",
			w:    60, h: 20,
			want:  []string{"write-adr"},
			avoid: []string{"COLLECTIONS"},
		},
		{
			name: "filter narrows the list",
			w:    120, h: 30,
			keys:  []string{"/", "t", "a", "b"},
			want:  []string{"go-table-tests"},
			avoid: []string{"write-adr"},
		},
		{
			name: "selecting a collection filters to it",
			w:    120, h: 30,
			group: "obsidian",
			want:  []string{"vault-note"},
			avoid: []string{"write-adr"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, out := render(t, tc.w, tc.h, tc.keys...)
			if tc.group != "" {
				// Counting "j" presses breaks the moment a collection is added,
				// which says nothing about the code under test.
				for b.groups[b.gi] != tc.group {
					before := b.gi
					b, _ = b.Update(keyMsg("j"))
					if b.gi == before {
						t.Fatalf("no collection named %q in %v", tc.group, b.groups)
					}
				}
				out = b.View()
			}
			for _, w := range tc.want {
				if !strings.Contains(out, w) {
					t.Errorf("view missing %q\n%s", w, out)
				}
			}
			for _, a := range tc.avoid {
				if strings.Contains(out, a) {
					t.Errorf("view should not contain %q\n%s", a, out)
				}
			}
			for i, line := range strings.Split(out, "\n") {
				if got := lineWidth(line); got > tc.w {
					t.Fatalf("line %d is %d cells wide, terminal is %d:\n%s", i, got, tc.w, line)
				}
			}
		})
	}
}

func TestBrowseNavigationStaysInBounds(t *testing.T) {
	cases := []struct {
		name string
		keys []string
	}{
		{"past the end of the collections", []string{"j", "j", "j", "j", "j", "j", "j", "j"}},
		{"before the start of the collections", []string{"k", "k", "k"}},
		{"past the end of the list", []string{"tab", "j", "j", "j", "j", "j", "j", "j", "j"}},
		{"filter to nothing then move", []string{"/", "z", "z", "z", "z", "enter", "j", "j"}},
		{"clear filter with esc", []string{"/", "z", "z", "esc", "j"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, out := render(t, 120, 30, tc.keys...)
			if b.gi < 0 || b.gi >= len(b.groups) {
				t.Fatalf("group cursor %d out of range 0..%d", b.gi, len(b.groups)-1)
			}
			if n := len(b.visible()); b.si < 0 || (n > 0 && b.si >= n) {
				t.Fatalf("skill cursor %d out of range for %d visible", b.si, n)
			}
			if out == "" {
				t.Fatal("empty view")
			}
		})
	}
}

// The preview must track the cursor — a stale preview is worse than none.
func TestPreviewFollowsSelection(t *testing.T) {
	_, first := render(t, 120, 30, "tab")
	_, second := render(t, 120, 30, "tab", "j")
	if first == second {
		t.Fatal("moving the cursor did not change the view")
	}
}

func lineWidth(s string) int {
	n, inEsc := 0, false
	for _, r := range s {
		switch {
		case r == 0x1b:
			inEsc = true
		case inEsc && (r == 'm' || r == 'K'):
			inEsc = false
		case !inEsc:
			n++
		}
	}
	return n
}

// TestSnapshot is not an assertion — it prints the screen so a human can look
// at it. `go test ./tui -run Snapshot -v`.
func TestSnapshot(t *testing.T) {
	for _, sz := range [][2]int{{120, 26}, {80, 20}} {
		_, out := render(t, sz[0], sz[1], "tab", "j")
		t.Logf("\n== %dx%d ==\n%s\n", sz[0], sz[1], out)
	}
}

func TestDocAndDoctorSnapshot(t *testing.T) {
	b, _ := render(t, 120, 26, "tab")
	s, _ := b.current()
	d := NewDoc(s, 120, 26)
	t.Logf("\n== doc 120x26 ==\n%s\n", d.View())
	d, _ = d.Update(keyMsg("tab"))
	t.Logf("\n== doc after tab (section 2) ==\n%s\n", d.View())

	dr := NewDoctor(brokenLibrary(t), 120, 26)
	t.Logf("\n== doctor 120x26 ==\n%s\n", dr.View())
	dr, _ = dr.Update(keyMsg("j"))
	t.Logf("\n== doctor after fix all ==\n%s\n", must(dr.Update(keyMsg("F"))).View())
}

func must(d Doctor, _ tea.Cmd) Doctor { return d }

// brokenLibrary is a skills/ tree with one of every problem doctor knows about.
func brokenLibrary(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "skills")
	write := func(rel, body string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("docs/api-reference/SKILL.md", "---\nname: api-reference\ntags: [docs]\n---\nbody\n")
	write("obsidian/vault-sync/SKILL.md", "no frontmatter at all\n")
	write("docs/diagram-from-code/SKILL.md",
		"---\nname: diagram-from-code\ntags: [docs]\ndescription: "+strings.Repeat("x", 1100)+"\n---\n")
	write("unfiled/go-table-tests/SKILL.md", "---\nname: go-table-tests\ndescription: d\n---\n")
	write("docs/write-adr/SKILL.md", "---\nname: write-adr\ndescription: d\ntags: [docs]\n---\nbody\n")
	write("docs/write-adr/references/examples.md", "x")
	write("docs/renamed/SKILL.md", "---\nname: old-name\ndescription: d\ntags: [docs]\n---\n")
	return root
}

func TestDoctorScreen(t *testing.T) {
	root := brokenLibrary(t)
	d := NewDoctor(root, 120, 26)
	if len(d.problems) < 5 {
		t.Fatalf("fixture should trip at least 5 rules, got %d: %+v", len(d.problems), d.problems)
	}
	errs, warns, fixable := d.counts()
	if errs == 0 || warns == 0 || fixable == 0 {
		t.Fatalf("want errors, warnings and fixables; got %d/%d/%d", errs, warns, fixable)
	}
	for _, line := range strings.Split(d.View(), "\n") {
		if lineWidth(line) > 120 {
			t.Fatalf("doctor line overflows: %q", line)
		}
	}

	// Fix-all must terminate, fix everything fixable, and leave the rest.
	d, _ = d.Update(keyMsg("F"))
	_, _, fixable = d.counts()
	if fixable != 0 {
		t.Errorf("fix all left %d fixable problems", fixable)
	}
	if len(d.problems) == 0 {
		t.Error("fix all should not have cleared the manual problems too")
	}
	for _, p := range d.problems {
		if p.Fixable {
			t.Errorf("%s still fixable after fix all", p.Code)
		}
	}
}

func TestDocScreenNavigation(t *testing.T) {
	b, _ := render(t, 120, 26, "tab")
	s, _ := b.current()
	cases := []struct {
		name string
		keys []string
	}{
		{"tab past the last section", []string{"tab", "tab", "tab", "tab", "tab", "tab", "tab", "tab"}},
		{"shift-tab before the first", []string{"shift+tab", "shift+tab"}},
		{"scroll to the bottom", []string{"G", "j", "j", "j"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDoc(s, 120, 26)
			for _, k := range tc.keys {
				d, _ = d.Update(keyMsg(k))
			}
			if d.oi < 0 || d.oi >= max(1, len(d.outline)) {
				t.Fatalf("outline cursor %d out of range for %d headings", d.oi, len(d.outline))
			}
			for _, line := range strings.Split(d.View(), "\n") {
				if lineWidth(line) > 120 {
					t.Fatalf("doc line overflows: %q", line)
				}
			}
		})
	}
}

// A narrow terminal drops the sidebar rather than truncating the body.
func TestDocNarrow(t *testing.T) {
	b, _ := render(t, 120, 26, "tab")
	s, _ := b.current()
	d := NewDoc(s, 70, 20)
	if strings.Contains(d.View(), "OUTLINE") {
		t.Error("sidebar should be dropped below 80 columns")
	}
}

func TestNewSkillForm(t *testing.T) {
	root := brokenLibrary(t)
	tmpl := repoTemplates(t)

	cases := []struct {
		name     string
		typed    map[int]string
		wantErr  string
		wantPath string
	}{
		{
			name:    "empty form is not ready",
			typed:   nil,
			wantErr: "name is required",
		},
		{
			name:    "rejects a name that is not kebab-case",
			typed:   map[int]string{fName: "My Skill", fDescription: "d", fTags: "docs"},
			wantErr: "kebab-case",
		},
		{
			name:    "requires a description",
			typed:   map[int]string{fName: "a-skill", fTags: "docs"},
			wantErr: "description is required",
		},
		{
			name:    "requires a tag, because the tag picks the folder",
			typed:   map[int]string{fName: "a-skill", fDescription: "d"},
			wantErr: "at least one tag",
		},
		{
			name:     "first tag decides the folder",
			typed:    map[int]string{fName: "a-skill", fDescription: "d", fTags: "go, testing"},
			wantPath: filepath.Join(root, "go", "a-skill", "SKILL.md"),
		},
		{
			name:    "refuses a name that already exists",
			typed:   map[int]string{fName: "write-adr", fDescription: "d", fTags: "docs"},
			wantErr: "already exists",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n := NewNewSkill(root, tmpl, 120, 30)
			for field, text := range tc.typed {
				n.inputs[field].SetValue(text)
			}
			n = n.recompute()
			if tc.wantErr != "" {
				if !strings.Contains(strings.Join(n.errs, "; "), tc.wantErr) {
					t.Fatalf("errs = %v, want one containing %q", n.errs, tc.wantErr)
				}
				if len(n.plan) > 0 {
					t.Errorf("an invalid draft must not produce a plan: %+v", n.plan)
				}
				return
			}
			if len(n.errs) > 0 {
				t.Fatalf("unexpected errors: %v", n.errs)
			}
			if len(n.plan) == 0 || n.plan[0].Path != tc.wantPath {
				t.Fatalf("plan[0] = %+v, want %s", n.plan, tc.wantPath)
			}
			// Nothing may exist before enter.
			if _, err := os.Stat(tc.wantPath); !os.IsNotExist(err) {
				t.Fatal("the form wrote to disk before confirm")
			}
			n, _ = n.Update(keyMsg("enter"))
			if _, err := os.Stat(tc.wantPath); err != nil {
				t.Fatalf("enter did not create the skill: %v", err)
			}
			// And the result must parse and pass doctor's basic checks.
			s, err := skillLoad(tc.wantPath)
			if err != nil {
				t.Fatalf("created SKILL.md does not parse: %v", err)
			}
			if s.Name != "a-skill" || len(s.Tags) != 2 {
				t.Errorf("created skill = %+v", s)
			}
		})
	}
}

func TestNewSkillBundleDirs(t *testing.T) {
	root := brokenLibrary(t)
	n := NewNewSkill(root, repoTemplates(t), 120, 30)
	n.inputs[fName].SetValue("bundled")
	n.inputs[fDescription].SetValue("d")
	n.inputs[fTags].SetValue("docs")
	n.bundle["references"] = true
	n = n.recompute()
	if len(n.plan) != 2 {
		t.Fatalf("plan = %+v, want SKILL.md plus references/.keep", n.plan)
	}
	if !strings.HasSuffix(n.plan[1].Path, filepath.Join("references", ".keep")) {
		t.Errorf("plan[1] = %s", n.plan[1].Path)
	}
}

func TestRegistryWithNoSources(t *testing.T) {
	root := brokenLibrary(t) // no sources.txt next to it
	r, cmd := NewRegistry(root, 120, 26)
	if cmd != nil {
		t.Error("with no sources there is nothing to fetch")
	}
	if !strings.Contains(r.View(), "sources.txt") {
		t.Errorf("the empty state should say how to add a source:\n%s", r.View())
	}
	for _, line := range strings.Split(r.View(), "\n") {
		if lineWidth(line) > 120 {
			t.Fatalf("registry line overflows: %q", line)
		}
	}
}

func skillLoad(path string) (skill.Skill, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return skill.Skill{}, err
	}
	return skill.Parse(string(b))
}

func TestSprint3Snapshot(t *testing.T) {
	root := brokenLibrary(t)
	n := NewNewSkill(root, repoTemplates(t), 110, 24)
	t.Logf("\n== new skill, empty ==\n%s\n", n.View())
	n.inputs[fName].SetValue("obsidian-daily-note")
	n.inputs[fDescription].SetValue("Append a timestamped entry to today's daily note, creating it from the template if it does not exist yet.")
	n.inputs[fTags].SetValue("obsidian, notes")
	n.bundle["references"] = true
	n = n.recompute()
	t.Logf("\n== new skill, filled ==\n%s\n", n.View())

	r, _ := NewRegistry(root, 110, 20)
	r.loading = false
	r.remotes = []skill.Remote{
		{Source: "mattpocock/skills", Name: "grill-with-docs", Tags: []string{"docs", "planning"}, Files: make([]string, 3)},
		{Source: "mattpocock/skills", Name: "domain-model", Tags: []string{"docs"}, Files: make([]string, 2)},
		{Source: "mattpocock/skills", Name: "tdd", Files: make([]string, 1)},
		{Source: "mattpocock/skills", Name: "write-adr", Tags: []string{"docs"}, Installed: true, Files: make([]string, 4)},
	}
	r.sources = []string{"mattpocock/skills"}
	r.sel["grill-with-docs"] = true
	r.sel["tdd"] = true
	t.Logf("\n== registry ==\n%s\n", r.View())
	r, _ = r.Update(keyMsg("t"))
	t.Logf("\n== registry, retagging ==\n%s\n", r.View())
}

func TestRegistryRetagChangesTheTarget(t *testing.T) {
	r, _ := NewRegistry(brokenLibrary(t), 110, 20)
	r.loading = false
	r.remotes = []skill.Remote{{Name: "tdd"}} // untagged
	r.sel["tdd"] = true
	if got := r.remotes[0].Tag(); got != "unfiled" {
		t.Fatalf("untagged remote should target unfiled, got %q", got)
	}
	r, _ = r.Update(keyMsg("t"))
	if !r.retagging {
		t.Fatal("t did not open the retag input")
	}
	r.retag.SetValue("testing")
	r, _ = r.Update(keyMsg("enter"))
	if got := r.remotes[0].Tag(); got != "testing" {
		t.Errorf("after retag the target is %q, want testing", got)
	}
	if r.retagging {
		t.Error("enter should close the retag input")
	}
}

// Every screen must render exactly as many lines as the terminal is tall. One
// line too many and the alt screen scrolls, which shifts the status bar off the
// bottom and leaves a torn row at the top.
func TestScreensFillTheTerminalExactly(t *testing.T) {
	const w, h = 110, 24
	root := brokenLibrary(t)

	b, _ := render(t, w, h)
	s, _ := b.current()
	reg, _ := NewRegistry(root, w, h)

	hv, hcmd := NewHistoryView(root, w, h)
	hv, _ = hv.Update(hcmd())

	views := map[string]string{
		"sync":     NewSyncView(root, w, h).View(),
		"history":  hv.View(),
		"browse":   b.View(),
		"doc":      NewDoc(s, w, h).View(),
		"doctor":   NewDoctor(root, w, h).View(),
		"new":      NewNewSkill(root, repoTemplates(t), w, h).View(),
		"registry": reg.View(),
		"edit":     mustEdit(s.Path(), w, h).View(),
	}
	for name, v := range views {
		t.Run(name, func(t *testing.T) {
			lines := strings.Split(v, "\n")
			if len(lines) != h {
				t.Errorf("rendered %d lines, terminal is %d tall", len(lines), h)
			}
			// The status bar is the last line; the line above it is its rule.
			if len(lines) >= 2 && !strings.Contains(lines[len(lines)-2], "─") {
				t.Errorf("no rule above the status bar; got %q", lines[len(lines)-2])
			}
		})
	}
}

func TestSyncAndHistoryScreens(t *testing.T) {
	const w, h = 120, 26
	root := repoSkills(t)

	sv := NewSyncView(root, w, h)
	t.Logf("\n== sync, global/claude ==\n%s\n", sv.View())
	sv, _ = sv.Update(keyMsg(" "))   // select the row under the cursor
	sv, _ = sv.Update(keyMsg("tab")) // next provider
	sv, _ = sv.Update(keyMsg("l"))   // local scope
	t.Logf("\n== sync, local/codex, one selected ==\n%s\n", sv.View())

	hv, cmd := NewHistoryView(root, w, h)
	if cmd == nil {
		t.Fatal("history must load in a command, not block Update")
	}
	hv, _ = hv.Update(cmd())
	t.Logf("\n== past used ==\n%s\n", hv.View())
	hv, _ = hv.Update(keyMsg("m"))
	t.Logf("\n== past used, suggestions only ==\n%s\n", hv.View())
}

func TestSyncViewTargetsAndScope(t *testing.T) {
	root := repoSkills(t)
	home, _ := os.UserHomeDir()
	cases := []struct {
		name string
		keys []string
		want string
	}{
		{"defaults to claude, globally", nil, filepath.Join(home, ".claude", "skills")},
		{"tab moves to the next provider", []string{"tab"}, filepath.Join(home, ".codex", "skills")},
		{"tab wraps rather than running off the end",
			[]string{"tab", "tab", "tab", "tab"}, filepath.Join(home, ".claude", "skills")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := NewSyncView(root, 120, 26)
			for _, k := range tc.keys {
				v, _ = v.Update(keyMsg(k))
			}
			got, err := v.dst()
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("target = %q, want %q", got, tc.want)
			}
		})
	}

	// With nothing selected, enter acts on the row under the cursor — not on
	// the whole library, and not on nothing.
	v := NewSyncView(root, 120, 26)
	if n := len(v.targets()); n != 1 {
		t.Errorf("unselected targets = %d, want just the cursor row", n)
	}
	v, _ = v.Update(keyMsg("a"))
	if n := len(v.targets()); n != len(v.library) {
		t.Errorf("after 'a', targets = %d, want all %d", n, len(v.library))
	}
	v, _ = v.Update(keyMsg("a"))
	if n := len(v.targets()); n != 1 {
		t.Errorf("'a' should toggle off, leaving the cursor row; got %d", n)
	}
}

// The whole point of the sync screen: copy into a real directory, and see it
// reported back as installed.
func TestSyncViewCopiesIntoAProject(t *testing.T) {
	root := repoSkills(t)
	proj := t.TempDir()
	v := NewSyncView(root, 120, 26)
	v.scope = "local"
	v.project.SetValue(proj)
	v, _ = v.Update(keyMsg("a")) // select everything
	v, _ = v.Update(keyMsg("enter"))

	dst := filepath.Join(proj, ".claude", "skills")
	got := len(skillInstalled(dst))
	if got != len(v.library) {
		t.Fatalf("installed %d of %d skills into %s", got, len(v.library), dst)
	}
	if !strings.Contains(v.status, "installed") {
		t.Errorf("status = %q, should report what happened", v.status)
	}
	// And removing them again must leave the directory empty, not the project.
	v, _ = v.Update(keyMsg("x"))
	if n := len(skillInstalled(dst)); n != 0 {
		t.Errorf("%d skills survived the remove", n)
	}
	if _, err := os.Stat(proj); err != nil {
		t.Error("the project directory itself was removed")
	}
}

func skillInstalled(dir string) []string { return skill.Installed(dir) }

func TestHistoryViewMissingFilter(t *testing.T) {
	v := HistoryView{w: 120, h: 26, uses: []skill.Use{
		{Name: "resolv", Count: 23, InLibrary: false},
		{Name: "write-adr", Count: 4, InLibrary: true},
	}, sources: []skill.HistorySource{{ID: "claude", Files: 1, Uses: 27}}}
	if len(v.rows()) != 2 {
		t.Fatalf("rows = %d, want both", len(v.rows()))
	}
	v, _ = v.Update(keyMsg("m"))
	if len(v.rows()) != 1 || v.rows()[0].Name != "resolv" {
		t.Fatalf("filtered rows = %+v, want only the missing one", v.rows())
	}
	if !strings.Contains(v.View(), "missing") {
		t.Error("a skill you use and do not have should be marked missing")
	}
	for _, line := range strings.Split(v.View(), "\n") {
		if lineWidth(line) > 120 {
			t.Fatalf("history line overflows: %q", line)
		}
	}
}

func TestHelpSnapshot(t *testing.T) {
	for _, s := range []screen{screenBrowse, screenSync} {
		t.Logf("\n== help · %s ==\n%s\n", screenNames[s], helpOverlay(s, 100, 26))
	}
	t.Logf("\n== status bar at 120 / 80 / 46 ==\n%s\n%s\n%s\n",
		footerHelp(keysBrowse, 120), footerHelp(keysBrowse, 80), footerHelp(keysBrowse, 46))
}
