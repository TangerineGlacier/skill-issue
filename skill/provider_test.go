package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProviderDir(t *testing.T) {
	home, _ := os.UserHomeDir()
	cases := []struct {
		name    string
		id      string
		scope   Scope
		project string
		want    string
		wantErr bool
	}{
		{"claude global", "claude", Global, "", filepath.Join(home, ".claude/skills"), false},
		{"claude local", "claude", Local, "/tmp/proj", "/tmp/proj/.claude/skills", false},
		{"codex global", "codex", Global, "", filepath.Join(home, ".codex/skills"), false},
		{"opencode global is XDG-ish, local is not", "opencode", Local, "/tmp/proj", "/tmp/proj/.opencode/skills", false},
		{"local without a project is an error", "claude", Local, "", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := ProviderByID(tc.id)
			if err != nil {
				t.Fatal(err)
			}
			got, err := p.Dir(tc.scope, tc.project)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestProviderByIDRejectsTypos(t *testing.T) {
	_, err := ProviderByID("cluade")
	if err == nil {
		t.Fatal("a typo must be an error, not an install into a directory nobody reads")
	}
	if !strings.Contains(err.Error(), "claude") {
		t.Errorf("the error should list the valid ids, got %q", err)
	}
}

func TestSyncCopiesTheWholeFolder(t *testing.T) {
	src := filepath.Join(t.TempDir(), "docs", "write-adr")
	writeFile(t, filepath.Join(src, "SKILL.md"), "---\nname: write-adr\ndescription: d\ntags: [docs]\n---\nbody\n")
	writeFile(t, filepath.Join(src, "references", "template.md"), "tpl")
	s, err := Load(src)
	if err != nil {
		t.Fatal(err)
	}

	dst := t.TempDir()
	got, replaced, err := Sync(s, dst)
	if err != nil {
		t.Fatal(err)
	}
	if replaced {
		t.Error("first install should not report a replacement")
	}
	for _, rel := range []string{"SKILL.md", "references/template.md"} {
		if _, err := os.Stat(filepath.Join(got, filepath.FromSlash(rel))); err != nil {
			t.Errorf("%s was not copied: %v", rel, err)
		}
	}
	if names := Installed(dst); len(names) != 1 || names[0] != "write-adr" {
		t.Errorf("Installed = %v", names)
	}

	// Re-syncing must replace, not merge: a stale file left behind from the
	// previous version is the whole reason this is a remove-then-copy.
	writeFile(t, filepath.Join(got, "references", "stale.md"), "old")
	_, replaced, err = Sync(s, dst)
	if err != nil {
		t.Fatal(err)
	}
	if !replaced {
		t.Error("second install should report a replacement")
	}
	if _, err := os.Stat(filepath.Join(got, "references", "stale.md")); !os.IsNotExist(err) {
		t.Error("a file from the previous version survived the re-sync")
	}
}

func TestUnsyncRefusesNonSkills(t *testing.T) {
	dst := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dst, "important"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Unsync("important", dst); err == nil {
		t.Fatal("Unsync deleted a directory that is not a skill")
	}
	if _, err := os.Stat(filepath.Join(dst, "important")); err != nil {
		t.Error("the directory was removed anyway")
	}
}

func TestInstalledSeesSymlinkedSkills(t *testing.T) {
	real := filepath.Join(t.TempDir(), "graphify")
	writeFile(t, filepath.Join(real, "SKILL.md"), "---\nname: graphify\n---\n")
	dst := t.TempDir()
	if err := os.Symlink(real, filepath.Join(dst, "graphify")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if names := Installed(dst); len(names) != 1 {
		t.Errorf("Installed = %v; symlinked skills are the common case for shared skills", names)
	}
}
