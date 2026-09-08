// Package skill reads the skills/ tree off disk. It is the only thing that
// knows the SKILL.md contract; the TUI, the catalog and doctor all go through it.
package skill

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// MaxDescription is the cap the model's skill index enforces. Longer
// descriptions are silently truncated upstream, so we treat it as an error.
const MaxDescription = 1024

// File is one file bundled alongside SKILL.md.
type File struct {
	Rel  string
	Size int64
}

// Skill is one skills/<group>/<name>/ folder.
type Skill struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags"`

	Dir     string // absolute path to the skill folder
	Group   string // folder under skills/ it actually sits in
	Body    string // markdown after the frontmatter
	Files   []File // bundled files, SKILL.md first
	Size    int64  // total bytes in the folder
	ModTime time.Time
}

// Path is the skill's SKILL.md.
func (s Skill) Path() string { return filepath.Join(s.Dir, "SKILL.md") }

// PrimaryTag is the tag that decides which folder the skill belongs in.
func (s Skill) PrimaryTag() string {
	if len(s.Tags) == 0 {
		return "unfiled"
	}
	return s.Tags[0]
}

// Headings returns the "## " headings of the body, in order — the doc outline.
func (s Skill) Headings() []string {
	var out []string
	sc := bufio.NewScanner(strings.NewReader(s.Body))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	fenced := false
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "```") {
			fenced = !fenced
			continue
		}
		if !fenced && strings.HasPrefix(line, "## ") {
			out = append(out, strings.TrimSpace(line[3:]))
		}
	}
	return out
}

// Parse splits YAML frontmatter from the markdown body.
func Parse(src string) (Skill, error) {
	src = strings.TrimPrefix(src, "\ufeff") // strip a UTF-8 BOM
	if !strings.HasPrefix(src, "---\n") {
		return Skill{}, fmt.Errorf("no frontmatter: file must start with ---")
	}
	rest := src[4:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return Skill{}, fmt.Errorf("unterminated frontmatter")
	}
	var s Skill
	if err := yaml.Unmarshal([]byte(rest[:end]), &s); err != nil {
		return Skill{}, fmt.Errorf("bad frontmatter: %w", err)
	}
	body := rest[end+4:]
	s.Body = strings.TrimLeft(body, "-\n")
	return s, nil
}

// Load reads one skill folder.
func Load(dir string) (Skill, error) {
	b, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		return Skill{}, err
	}
	s, err := Parse(string(b))
	if err != nil {
		return Skill{}, fmt.Errorf("%s: %w", filepath.Join(dir, "SKILL.md"), err)
	}
	s.Dir = dir
	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		s.Files = append(s.Files, File{Rel: rel, Size: info.Size()})
		s.Size += info.Size()
		if info.ModTime().After(s.ModTime) {
			s.ModTime = info.ModTime()
		}
		return nil
	})
	sort.Slice(s.Files, func(i, j int) bool {
		if a, b := s.Files[i].Rel == "SKILL.md", s.Files[j].Rel == "SKILL.md"; a != b {
			return a
		}
		return s.Files[i].Rel < s.Files[j].Rel
	})
	return s, err
}

// LoadAll walks a skills/ root. Every directory holding a SKILL.md is a skill;
// a folder that fails to parse is returned in errs rather than dropped, so
// doctor can report it instead of it vanishing from the list.
func LoadAll(root string) (skills []Skill, errs []error) {
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() || p == root {
			return err
		}
		if _, statErr := os.Stat(filepath.Join(p, "SKILL.md")); statErr != nil {
			return nil
		}
		s, err := Load(p)
		if err != nil {
			errs = append(errs, err)
			return fs.SkipDir
		}
		rel, _ := filepath.Rel(root, p)
		if parts := strings.Split(rel, string(filepath.Separator)); len(parts) > 1 {
			s.Group = parts[0]
		} else {
			s.Group = "unfiled"
		}
		skills = append(skills, s)
		return fs.SkipDir // skills do not nest
	})
	if err != nil {
		errs = append(errs, err)
	}
	sort.Slice(skills, func(i, j int) bool {
		if skills[i].Group != skills[j].Group {
			return skills[i].Group < skills[j].Group
		}
		return skills[i].Name < skills[j].Name
	})
	return skills, errs
}

// Root finds the skills/ directory by walking up from dir, so the binary works
// from anywhere inside the repo.
func Root(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for {
		cand := filepath.Join(abs, "skills")
		if fi, err := os.Stat(cand); err == nil && fi.IsDir() {
			return cand, nil
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return "", fmt.Errorf("no skills/ directory found from %s upwards", dir)
		}
		abs = parent
	}
}
