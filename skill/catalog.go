package skill

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Entry is one skill as the docs site sees it. It carries the rendered body so
// the web build needs nothing but this file.
type Entry struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	Group       string    `json:"group"`
	Path        string    `json:"path"` // relative to the repo root
	Headings    []string  `json:"headings"`
	Files       []File    `json:"files"`
	Size        int64     `json:"size"`
	Updated     time.Time `json:"updated"`
	Hash        string    `json:"hash"`
	Body        string    `json:"body"`
	Problems    []Problem `json:"problems"`
}

// Catalog is the generated index both readers agree on. It is a cache: delete
// it and Build regenerates it from the filesystem.
type Catalog struct {
	Generated time.Time `json:"generated"`
	Groups    []string  `json:"groups"`
	Skills    []Entry   `json:"skills"`
	Problems  []Problem `json:"problems"`
}

// Build walks the skills root and runs every doctor check over the result.
func Build(root string) Catalog {
	skills, errs := LoadAll(root)
	problems := Check(skills, errs)

	byDir := map[string][]Problem{}
	for _, p := range problems {
		byDir[p.Dir] = append(byDir[p.Dir], p)
	}

	repo := filepath.Dir(root)
	seen := map[string]bool{}
	c := Catalog{Generated: time.Now(), Problems: problems}
	for _, s := range skills {
		rel, err := filepath.Rel(repo, s.Dir)
		if err != nil {
			rel = s.Dir
		}
		sum := sha256.Sum256([]byte(s.Body + s.Description))
		c.Skills = append(c.Skills, Entry{
			Name: s.Name, Description: s.Description, Tags: s.Tags, Group: s.Group,
			Path: filepath.ToSlash(rel), Headings: s.Headings(), Files: s.Files,
			Size: s.Size, Updated: s.ModTime, Hash: hex.EncodeToString(sum[:8]),
			Body: s.Body, Problems: byDir[s.Dir],
		})
		if !seen[s.Group] {
			seen[s.Group] = true
			c.Groups = append(c.Groups, s.Group)
		}
	}
	return c
}

// WriteCatalog regenerates the catalog next to the skills root. It is written whole
// and atomically, so a half-written catalog never reaches the web build.
func WriteCatalog(root string) (string, int, error) {
	c := Build(root)
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", 0, err
	}
	out := filepath.Join(filepath.Dir(root), "catalog.json")
	tmp := out + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return "", 0, err
	}
	if err := os.Rename(tmp, out); err != nil {
		return "", 0, err
	}
	return out, len(c.Skills), nil
}
