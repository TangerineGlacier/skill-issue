package skill

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		want     Skill
		wantErr  bool
		headings []string
	}{
		{
			name: "minimal",
			in:   "---\nname: a\ndescription: does a thing\n---\n# a\n",
			want: Skill{Name: "a", Description: "does a thing", Body: "# a\n"},
		},
		{
			name: "inline tags",
			in:   "---\nname: a\ntags: [docs, architecture]\n---\n",
			want: Skill{Name: "a", Tags: []string{"docs", "architecture"}, Body: ""},
		},
		{
			name: "block tags and folded description",
			in:   "---\nname: a\ndescription: one two\n  three\ntags:\n  - go\n  - testing\n---\nbody\n",
			want: Skill{Name: "a", Description: "one two three",
				Tags: []string{"go", "testing"}, Body: "body\n"},
		},
		{
			name:    "no frontmatter",
			in:      "# just markdown\n",
			wantErr: true,
		},
		{
			name:    "unterminated frontmatter",
			in:      "---\nname: a\n",
			wantErr: true,
		},
		{
			name:    "colon in description needs quoting, unquoted is a yaml error",
			in:      "---\ndescription: use when: never\n---\n",
			wantErr: true,
		},
		{
			name:     "headings skip fenced code",
			in:       "---\nname: a\n---\n## One\n```\n## not a heading\n```\n## Two\n",
			want:     Skill{Name: "a", Body: "## One\n```\n## not a heading\n```\n## Two\n"},
			headings: []string{"One", "Two"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got  %+v\nwant %+v", got, tc.want)
			}
			if tc.headings != nil && !reflect.DeepEqual(got.Headings(), tc.headings) {
				t.Errorf("headings = %v, want %v", got.Headings(), tc.headings)
			}
		})
	}
}

func TestPrimaryTag(t *testing.T) {
	cases := []struct {
		name string
		tags []string
		want string
	}{
		{"none", nil, "unfiled"},
		{"one", []string{"docs"}, "docs"},
		{"first of many", []string{"docs", "architecture"}, "docs"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := (Skill{Tags: tc.tags}).PrimaryTag(); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// LoadAll must report a broken skill rather than silently dropping it, or
// doctor can never tell you about the one folder that is actually broken.
func TestLoadAllReportsBrokenSkills(t *testing.T) {
	root := t.TempDir()
	write := func(p, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(root, "docs", "good", "SKILL.md"), "---\nname: good\n---\nhi\n")
	write(filepath.Join(root, "docs", "good", "references", "x.md"), "x")
	write(filepath.Join(root, "docs", "broken", "SKILL.md"), "no frontmatter\n")
	write(filepath.Join(root, "loose", "SKILL.md"), "---\nname: loose\n---\n")

	skills, errs := LoadAll(root)
	if len(skills) != 2 {
		t.Fatalf("loaded %d skills, want 2: %+v", len(skills), skills)
	}
	if len(errs) != 1 {
		t.Fatalf("errs = %v, want exactly 1", errs)
	}
	if skills[0].Name != "good" || skills[0].Group != "docs" {
		t.Errorf("got %q in group %q, want good in docs", skills[0].Name, skills[0].Group)
	}
	if len(skills[0].Files) != 2 || skills[0].Files[0].Rel != "SKILL.md" {
		t.Errorf("files = %+v, want SKILL.md first then references/x.md", skills[0].Files)
	}
	// A skill directly under the root has no group folder.
	if skills[1].Group != "unfiled" {
		t.Errorf("loose skill group = %q, want unfiled", skills[1].Group)
	}
}
