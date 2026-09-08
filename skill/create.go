package skill

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
)

// Draft is a skill that does not exist yet.
type Draft struct {
	Name        string
	Description string
	Tags        []string
	Template    string   // template file stem, e.g. "doc-writer"
	Bundle      []string // extra directories: references, scripts, assets
}

// TagList renders the tags for the frontmatter line.
func (d Draft) TagList() string { return strings.Join(d.Tags, ", ") }

// Dir is where the draft will land: the first tag decides the folder.
func (d Draft) Dir(root string) string {
	tag := "unfiled"
	if len(d.Tags) > 0 {
		tag = d.Tags[0]
	}
	return filepath.Join(root, tag, d.Name)
}

// PlannedFile is one file the draft would create. Nothing touches disk until Apply.
type PlannedFile struct {
	Path string // absolute
	Body string
	Note string // shown in the dry-run list
}

var kebab = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Validate returns every reason the draft cannot be saved, so the form can show
// them all at once rather than one per attempt.
func (d Draft) Validate(root string, existing []Skill) []string {
	var errs []string
	switch {
	case d.Name == "":
		errs = append(errs, "name is required")
	case !kebab.MatchString(d.Name):
		errs = append(errs, "name must be kebab-case: lowercase letters, digits and single hyphens")
	}
	for _, s := range existing {
		if s.Name == d.Name {
			errs = append(errs, "a skill named "+d.Name+" already exists in "+s.Group+"/")
			break
		}
	}
	switch {
	case d.Description == "":
		errs = append(errs, "description is required — without it the skill never loads")
	case len(d.Description) > MaxDescription:
		errs = append(errs, fmt.Sprintf("description is %d chars, max %d", len(d.Description), MaxDescription))
	}
	if len(d.Tags) == 0 {
		errs = append(errs, "at least one tag is required — the first one decides the folder")
	}
	if d.Name != "" && len(errs) == 0 {
		if _, err := os.Stat(d.Dir(root)); err == nil {
			errs = append(errs, d.Dir(root)+" already exists")
		}
	}
	return errs
}

// Plan renders every file the draft would create. Run it on each keystroke if
// you like: it is pure, and it is what the "will write" pane shows.
func (d Draft) Plan(root, templates string) ([]PlannedFile, error) {
	name := d.Template
	if name == "" {
		name = "blank"
	}
	b, err := os.ReadFile(filepath.Join(templates, name+".md"))
	if err != nil {
		return nil, fmt.Errorf("template %q: %w", name, err)
	}
	t, err := template.New(name).Parse(string(b))
	if err != nil {
		return nil, fmt.Errorf("template %q: %w", name, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, d); err != nil {
		return nil, err
	}

	dir := d.Dir(root)
	ws := []PlannedFile{{Path: filepath.Join(dir, "SKILL.md"), Body: buf.String(), Note: "new · " + name}}
	bundle := append([]string(nil), d.Bundle...)
	sort.Strings(bundle)
	for _, sub := range bundle {
		ws = append(ws, PlannedFile{
			Path: filepath.Join(dir, sub, ".keep"),
			Body: "",
			Note: "new directory",
		})
	}
	return ws, nil
}

// Apply writes the plan. It creates nothing if any file already exists, so a
// half-created skill is not a state this can produce.
func Apply(ws []PlannedFile) error {
	for _, w := range ws {
		if _, err := os.Stat(w.Path); err == nil {
			return fmt.Errorf("%s already exists", w.Path)
		}
	}
	for _, w := range ws {
		if err := os.MkdirAll(filepath.Dir(w.Path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(w.Path, []byte(w.Body), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// TemplateNames lists the templates available to the form.
func TemplateNames(dir string) []string {
	es, err := os.ReadDir(dir)
	if err != nil {
		return []string{"blank"}
	}
	var out []string
	for _, e := range es {
		if n := strings.TrimSuffix(e.Name(), ".md"); n != e.Name() {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	if len(out) == 0 {
		out = []string{"blank"}
	}
	return out
}
