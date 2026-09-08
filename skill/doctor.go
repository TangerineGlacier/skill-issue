package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Level orders problems worst-first.
type Level int

const (
	Warn Level = iota
	Error
)

func (l Level) String() string {
	if l == Error {
		return "err"
	}
	return "warn"
}

// Problem is one thing wrong with one skill. Fixable problems carry enough
// information for Fix to repair them without re-running the checks.
type Problem struct {
	Skill   string `json:"skill"`
	Dir     string `json:"dir"`
	Level   Level  `json:"level"`
	Code    string `json:"code"`
	Msg     string `json:"msg"`
	Detail  string `json:"detail"`
	Fixable bool   `json:"fixable"`

	arg string // fix payload: the tag to add, the folder to move to, the new name
}

// Check runs every rule over the library. loadErrs are the folders that failed
// to parse — they cannot be represented as a Skill, so they arrive separately.
func Check(skills []Skill, loadErrs []error) []Problem {
	var ps []Problem

	for _, e := range loadErrs {
		msg := e.Error()
		dir := ""
		if i := strings.Index(msg, "/SKILL.md"); i > 0 {
			dir = filepath.Dir(msg[:i+len("/SKILL.md")])
		}
		ps = append(ps, Problem{
			Skill: filepath.Base(dir), Dir: dir, Level: Error, Code: "unreadable",
			Msg:    "SKILL.md is missing or will not parse",
			Detail: msg + "\n\nUntil this parses the skill does not exist as far as any reader is concerned.",
		})
	}

	for _, s := range skills {
		switch {
		case s.Description == "":
			ps = append(ps, Problem{
				Skill: s.Name, Dir: s.Dir, Level: Error, Code: "no-description",
				Msg: "no description in frontmatter",
				Detail: "The description is the only thing the model matches against when " +
					"deciding whether to load a skill. Without it this skill will never " +
					"load, no matter how good the body is.\n\nFix: add one sentence saying " +
					"when to reach for it.",
			})
		case len(s.Description) > MaxDescription:
			ps = append(ps, Problem{
				Skill: s.Name, Dir: s.Dir, Level: Warn, Code: "description-too-long",
				Msg: fmt.Sprintf("description %d chars (max %d)", len(s.Description), MaxDescription),
				Detail: "Long descriptions are truncated by the reader, so the end of yours " +
					"is invisible. Cut it to the triggering conditions and move the rest " +
					"into the body.",
			})
		}

		if s.Name == "" {
			ps = append(ps, Problem{
				Skill: filepath.Base(s.Dir), Dir: s.Dir, Level: Error, Code: "no-name",
				Msg: "no name in frontmatter", Fixable: true, arg: filepath.Base(s.Dir),
				Detail: "Fix sets name to the folder name.",
			})
		} else if want := filepath.Base(s.Dir); s.Name != want {
			ps = append(ps, Problem{
				Skill: s.Name, Dir: s.Dir, Level: Warn, Code: "name-mismatch",
				Msg: fmt.Sprintf("name %q but folder %q", s.Name, want), Fixable: true, arg: want,
				Detail: "Every reader addresses a skill by its folder. Fix renames the " +
					"frontmatter to match the folder, not the other way round.",
			})
		}

		if len(s.Tags) == 0 {
			// A skill already sitting in a real folder can be tagged from it.
			// One in unfiled/ cannot: "unfiled" is not a tag, it is the absence
			// of one, and writing it back would only hide the problem.
			p := Problem{
				Skill: s.Name, Dir: s.Dir, Level: Warn, Code: "untagged",
				Msg: "no tags, so nothing decides its folder",
				Detail: "Tags decide which folder a skill lives in, and they are how the " +
					"docs site groups it.\n\nFix adds the folder it is already sitting in " +
					"as its first tag.",
			}
			if s.Group != "unfiled" {
				p.Fixable, p.arg = true, s.Group
			} else {
				p.Level = Error
				p.Detail = "This skill is in unfiled/ with no tags, so nothing decides where " +
					"it belongs and the docs site cannot group it.\n\nNo automatic fix: only " +
					"you know what it is about. Add a tag and run fix to file it."
			}
			ps = append(ps, p)
		} else if s.PrimaryTag() != s.Group {
			ps = append(ps, Problem{
				Skill: s.Name, Dir: s.Dir, Level: Warn, Code: "misfiled",
				Msg:     fmt.Sprintf("tagged %q but filed under %q", s.PrimaryTag(), s.Group),
				Fixable: true, arg: s.PrimaryTag(),
				Detail: "Fix moves the folder to match the first tag. The tag wins; the folder follows.",
			})
		}

		for _, f := range s.Files {
			if !strings.HasPrefix(f.Rel, "references/") {
				continue
			}
			if !strings.Contains(s.Body, filepath.Base(f.Rel)) && !strings.Contains(s.Body, f.Rel) {
				ps = append(ps, Problem{
					Skill: s.Name, Dir: s.Dir, Level: Warn, Code: "unlinked-reference",
					Msg: f.Rel + " is never linked from the body",
					Detail: "A reference file nothing points at is never loaded, so it is " +
						"dead weight in the repo. Either link it from the body or delete it.",
				})
			}
		}
	}

	sort.SliceStable(ps, func(i, j int) bool {
		if ps[i].Level != ps[j].Level {
			return ps[i].Level > ps[j].Level
		}
		return ps[i].Skill < ps[j].Skill
	})
	return ps
}

// Fix repairs one fixable problem in place. It returns what it did, so the
// caller can show it rather than claiming success silently.
func Fix(p Problem) (string, error) {
	if !p.Fixable {
		return "", fmt.Errorf("%s is not auto-fixable", p.Code)
	}
	path := filepath.Join(p.Dir, "SKILL.md")

	switch p.Code {
	case "no-name", "name-mismatch":
		if err := setFrontmatter(path, "name", p.arg); err != nil {
			return "", err
		}
		return fmt.Sprintf("set name: %s", p.arg), nil

	case "untagged":
		if err := setFrontmatter(path, "tags", "["+p.arg+"]"); err != nil {
			return "", err
		}
		return fmt.Sprintf("added tag: %s", p.arg), nil

	case "misfiled":
		skillsRoot := filepath.Dir(filepath.Dir(p.Dir))
		dst := filepath.Join(skillsRoot, p.arg, filepath.Base(p.Dir))
		if _, err := os.Stat(dst); err == nil {
			return "", fmt.Errorf("%s already exists", dst)
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return "", err
		}
		if err := os.Rename(p.Dir, dst); err != nil {
			return "", err
		}
		return fmt.Sprintf("moved to %s/", p.arg), nil
	}
	return "", fmt.Errorf("no fix implemented for %s", p.Code)
}

// setFrontmatter replaces or inserts a top-level key, leaving the rest of the
// file byte-identical. Rewriting via the YAML marshaller would reflow the whole
// document and lose comments and folding.
func setFrontmatter(path, key, value string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	src := string(b)
	if !strings.HasPrefix(src, "---\n") {
		return fmt.Errorf("%s: no frontmatter to edit", path)
	}
	end := strings.Index(src[4:], "\n---")
	if end < 0 {
		return fmt.Errorf("%s: unterminated frontmatter", path)
	}
	head, tail := src[4:4+end], src[4+end:]

	lines := strings.Split(head, "\n")
	replaced := false
	out := make([]string, 0, len(lines)+1)
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		if !strings.HasPrefix(l, key+":") {
			out = append(out, l)
			continue
		}
		out = append(out, key+": "+value)
		replaced = true
		// Drop the old value's continuation and block-list lines.
		for i+1 < len(lines) && (strings.HasPrefix(lines[i+1], " ") ||
			strings.HasPrefix(lines[i+1], "\t") || strings.HasPrefix(lines[i+1], "-")) {
			i++
		}
	}
	if !replaced {
		out = append(out, key+": "+value)
	}
	return os.WriteFile(path, []byte("---\n"+strings.Join(out, "\n")+tail), 0o644)
}
