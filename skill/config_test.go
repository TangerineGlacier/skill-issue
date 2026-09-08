package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveRootPrecedence(t *testing.T) {
	// A library that only the config knows about.
	linked := filepath.Join(t.TempDir(), "linked")
	writeFile(t, filepath.Join(linked, "skills", "docs", "a", "SKILL.md"), "---\nname: a\n---\n")
	// A library you are standing inside.
	here := filepath.Join(t.TempDir(), "here")
	writeFile(t, filepath.Join(here, "skills", "docs", "b", "SKILL.md"), "---\nname: b\n---\n")
	// A third, named by the environment.
	env := filepath.Join(t.TempDir(), "env")
	writeFile(t, filepath.Join(env, "skills", "docs", "c", "SKILL.md"), "---\nname: c\n---\n")

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := SaveConfig(Config{Library: filepath.Join(linked, "skills")}); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		env  string
		from string
		want string
	}{
		{"env wins over everything", env, here, filepath.Join(env, "skills")},
		{"standing inside a library beats the linked one", "", here, filepath.Join(here, "skills")},
		{"falls back to the linked library", "", t.TempDir(), filepath.Join(linked, "skills")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("SKILL_ISSUE_HOME", tc.env)
			got, err := ResolveRoot(tc.from)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveRootErrors(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Run("bad SKILL_ISSUE_HOME says so instead of falling through", func(t *testing.T) {
		t.Setenv("SKILL_ISSUE_HOME", filepath.Join(t.TempDir(), "nope"))
		if _, err := ResolveRoot(t.TempDir()); err == nil {
			t.Fatal("want an error")
		}
	})
	t.Run("nothing configured explains what to do", func(t *testing.T) {
		t.Setenv("SKILL_ISSUE_HOME", "")
		_, err := ResolveRoot(t.TempDir())
		if err == nil {
			t.Fatal("want an error")
		}
		if got := err.Error(); !strings.Contains(got, "skill-issue link") {
			t.Errorf("the error should name the fix, got %q", got)
		}
	})
	t.Run("SKILL_ISSUE_HOME may point at the repo or at skills/", func(t *testing.T) {
		repo := t.TempDir()
		writeFile(t, filepath.Join(repo, "skills", "docs", "a", "SKILL.md"), "---\nname: a\n---\n")
		for _, v := range []string{repo, filepath.Join(repo, "skills")} {
			t.Setenv("SKILL_ISSUE_HOME", v)
			got, err := ResolveRoot(os.TempDir())
			if err != nil || got != filepath.Join(repo, "skills") {
				t.Errorf("SKILL_ISSUE_HOME=%s → %q, %v", v, got, err)
			}
		}
	})
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && filepath.Base(s) != sub && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
