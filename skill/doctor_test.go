package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func codes(ps []Problem) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.Code
	}
	return out
}

func has(ps []Problem, code string) bool {
	for _, p := range ps {
		if p.Code == code {
			return true
		}
	}
	return false
}

func TestCheck(t *testing.T) {
	long := strings.Repeat("x", MaxDescription+1)
	cases := []struct {
		name  string
		skill Skill
		want  []string
		none  []string
	}{
		{
			name:  "healthy skill has no problems",
			skill: Skill{Name: "a", Description: "d", Tags: []string{"docs"}, Dir: "/r/docs/a", Group: "docs"},
			want:  nil,
		},
		{
			name:  "missing description is an error",
			skill: Skill{Name: "a", Tags: []string{"docs"}, Dir: "/r/docs/a", Group: "docs"},
			want:  []string{"no-description"},
		},
		{
			name:  "over-long description is a warning, not an error",
			skill: Skill{Name: "a", Description: long, Tags: []string{"docs"}, Dir: "/r/docs/a", Group: "docs"},
			want:  []string{"description-too-long"},
			none:  []string{"no-description"},
		},
		{
			name:  "untagged",
			skill: Skill{Name: "a", Description: "d", Dir: "/r/docs/a", Group: "docs"},
			want:  []string{"untagged"},
		},
		{
			name:  "primary tag must match the folder",
			skill: Skill{Name: "a", Description: "d", Tags: []string{"go"}, Dir: "/r/docs/a", Group: "docs"},
			want:  []string{"misfiled"},
		},
		{
			name:  "extra tags beyond the first do not have to match",
			skill: Skill{Name: "a", Description: "d", Tags: []string{"docs", "go"}, Dir: "/r/docs/a", Group: "docs"},
			none:  []string{"misfiled"},
		},
		{
			name:  "name must match the folder",
			skill: Skill{Name: "b", Description: "d", Tags: []string{"docs"}, Dir: "/r/docs/a", Group: "docs"},
			want:  []string{"name-mismatch"},
		},
		{
			name: "reference nobody links to",
			skill: Skill{Name: "a", Description: "d", Tags: []string{"docs"}, Dir: "/r/docs/a", Group: "docs",
				Files: []File{{Rel: "SKILL.md"}, {Rel: "references/x.md"}}},
			want: []string{"unlinked-reference"},
		},
		{
			name: "linked reference is fine",
			skill: Skill{Name: "a", Description: "d", Tags: []string{"docs"}, Dir: "/r/docs/a", Group: "docs",
				Body:  "see [x](references/x.md)",
				Files: []File{{Rel: "SKILL.md"}, {Rel: "references/x.md"}}},
			none: []string{"unlinked-reference"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Check([]Skill{tc.skill}, nil)
			for _, w := range tc.want {
				if !has(got, w) {
					t.Errorf("missing %q, got %v", w, codes(got))
				}
			}
			for _, n := range tc.none {
				if has(got, n) {
					t.Errorf("unexpected %q in %v", n, codes(got))
				}
			}
			if tc.want == nil && tc.none == nil && len(got) != 0 {
				t.Errorf("want no problems, got %v", codes(got))
			}
		})
	}
}

func TestCheckErrorsSortFirst(t *testing.T) {
	ps := Check([]Skill{
		{Name: "z", Description: "d", Dir: "/r/docs/z", Group: "docs"},       // warn: untagged
		{Name: "a", Tags: []string{"docs"}, Dir: "/r/docs/a", Group: "docs"}, // error
	}, nil)
	if len(ps) < 2 || ps[0].Level != Error {
		t.Fatalf("errors must sort first, got %v", codes(ps))
	}
}

func TestFix(t *testing.T) {
	cases := []struct {
		name   string
		src    string
		code   string
		arg    string
		want   string // substring the SKILL.md must contain afterwards
		absent string
	}{
		{
			name: "add tags to an untagged skill",
			src:  "---\nname: a\ndescription: d\n---\nbody\n",
			code: "untagged", arg: "docs", want: "tags: [docs]",
		},
		{
			name: "replace an inline tags line",
			src:  "---\nname: a\ntags: [go]\n---\nbody\n",
			code: "untagged", arg: "docs", want: "tags: [docs]", absent: "[go]",
		},
		{
			name: "replace a block tags list",
			src:  "---\nname: a\ntags:\n  - go\n  - testing\n---\nbody\n",
			code: "untagged", arg: "docs", want: "tags: [docs]", absent: "testing",
		},
		{
			name: "rename to match the folder",
			src:  "---\nname: wrong\ndescription: d\n---\nbody\n",
			code: "name-mismatch", arg: "a", want: "name: a", absent: "wrong",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "docs", "a")
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, "SKILL.md")
			if err := os.WriteFile(path, []byte(tc.src), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := Fix(Problem{Dir: dir, Code: tc.code, arg: tc.arg, Fixable: true}); err != nil {
				t.Fatal(err)
			}
			b, _ := os.ReadFile(path)
			got := string(b)
			if !strings.Contains(got, tc.want) {
				t.Errorf("want %q in:\n%s", tc.want, got)
			}
			if tc.absent != "" && strings.Contains(got, tc.absent) {
				t.Errorf("old value %q survived:\n%s", tc.absent, got)
			}
			if !strings.Contains(got, "body\n") {
				t.Errorf("body was lost:\n%s", got)
			}
			// The fix must leave a file that still parses.
			if _, err := Parse(got); err != nil {
				t.Errorf("fix produced unparseable frontmatter: %v\n%s", err, got)
			}
		})
	}
}

func TestFixMisfiledMovesTheFolder(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "docs", "a")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("---\nname: a\ntags: [go]\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Fix(Problem{Dir: src, Code: "misfiled", arg: "go", Fixable: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "go", "a", "SKILL.md")); err != nil {
		t.Fatalf("skill was not moved to go/: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("old folder still exists")
	}
}

func TestFixRefusesManualProblems(t *testing.T) {
	if _, err := Fix(Problem{Code: "no-description"}); err == nil {
		t.Fatal("want an error for a non-fixable problem")
	}
}

// An untagged skill in unfiled/ must not be "fixed" by tagging it "unfiled" —
// that writes the absence of a decision back into the file as if it were one.
func TestUntaggedInUnfiledIsNotAutoFixable(t *testing.T) {
	cases := []struct {
		name        string
		group       string
		wantFixable bool
		wantLevel   Level
	}{
		{"in a real folder, tag it from the folder", "docs", true, Warn},
		{"in unfiled, a human has to choose", "unfiled", false, Error},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ps := Check([]Skill{{Name: "a", Description: "d",
				Dir: "/r/" + tc.group + "/a", Group: tc.group}}, nil)
			var got *Problem
			for i := range ps {
				if ps[i].Code == "untagged" {
					got = &ps[i]
				}
			}
			if got == nil {
				t.Fatalf("no untagged problem in %v", codes(ps))
			}
			if got.Fixable != tc.wantFixable {
				t.Errorf("Fixable = %v, want %v", got.Fixable, tc.wantFixable)
			}
			if got.Level != tc.wantLevel {
				t.Errorf("Level = %v, want %v", got.Level, tc.wantLevel)
			}
		})
	}
}
