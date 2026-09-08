package skill

import (
	"path/filepath"
	"testing"
)

func TestRemoteTarget(t *testing.T) {
	cases := []struct {
		name string
		tags []string
		want string
	}{
		{"first tag decides the folder", []string{"docs", "go"}, "docs/write-adr"},
		{"no tags means unfiled", nil, "unfiled/write-adr"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := Remote{Name: "write-adr", Tags: tc.tags}
			if got := r.Target("/r/skills"); got != filepath.Join("/r/skills", tc.want) {
				t.Errorf("got %q, want .../%s", got, tc.want)
			}
		})
	}
}

func TestSources(t *testing.T) {
	repo := t.TempDir()
	root := filepath.Join(repo, "skills")
	writeFile(t, filepath.Join(repo, "sources.txt"),
		"# a comment\n\nowner/one\n  owner/two  \n")
	got := Sources(root)
	want := []string{"owner/one", "owner/two"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got %v, want %v", got, want)
		}
	}
	if s := Sources(filepath.Join(t.TempDir(), "skills")); s != nil {
		t.Errorf("a missing sources.txt should give nil, got %v", s)
	}
}

func TestRemoteTagPrecedence(t *testing.T) {
	cases := []struct {
		name  string
		tags  []string
		retag string
		want  string
	}{
		{"retag beats the frontmatter tag", []string{"docs"}, "go", "go"},
		{"frontmatter tag when there is no retag", []string{"docs", "go"}, "", "docs"},
		{"untagged and un-retagged goes to unfiled", nil, "", "unfiled"},
		{"retag rescues an untagged skill", nil, "go", "go"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := Remote{Name: "x", Tags: tc.tags, Retag: tc.retag}
			if got := r.Tag(); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDedupePrefersTheShallowestPath(t *testing.T) {
	in := []Remote{
		{Name: "a", Dir: "plugins/a"},
		{Name: "a", Dir: "x/y/z/a"},
		{Name: "b", Dir: "b"},
	}
	got := dedupe(in)
	if len(got) != 2 {
		t.Fatalf("got %d remotes, want 2", len(got))
	}
	if got[0].Dir != "plugins/a" {
		t.Errorf("kept %q, want the first (shallowest) entry", got[0].Dir)
	}
}
