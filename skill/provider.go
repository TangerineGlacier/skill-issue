package skill

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Scope says whose copy of a skill directory we mean.
type Scope string

const (
	// Global is the per-user directory: available in every project.
	Global Scope = "global"
	// Local is the per-project directory, checked into that project's repo.
	Local Scope = "local"
)

// Provider is an agent that reads skills from a directory on disk. The paths
// are the ones each tool actually documents; nothing here is guessed at
// runtime, so an unknown provider is a code change, not a silent no-op.
type Provider struct {
	ID       string
	Label    string
	global   string // relative to $HOME
	local    string // relative to a project root
	AlsoRead []string
}

var Providers = []Provider{
	{ID: "claude", Label: "Claude Code", global: ".claude/skills", local: ".claude/skills"},
	{ID: "codex", Label: "Codex CLI", global: ".codex/skills", local: ".codex/skills"},
	{ID: "cursor", Label: "Cursor", global: ".cursor/skills", local: ".cursor/skills",
		AlsoRead: []string{".cursor/skills-cursor"}},
	{ID: "opencode", Label: "opencode", global: ".config/opencode/skills", local: ".opencode/skills"},
}

// ProviderByID looks a provider up, so a typo on the CLI is an error rather
// than an install into a directory nobody reads.
func ProviderByID(id string) (Provider, error) {
	for _, p := range Providers {
		if p.ID == id {
			return p, nil
		}
	}
	var ids []string
	for _, p := range Providers {
		ids = append(ids, p.ID)
	}
	return Provider{}, fmt.Errorf("unknown provider %q — one of: %s", id, strings.Join(ids, ", "))
}

// Dir is where this provider reads skills from, for the given scope. For Local,
// project is the project root; for Global it is ignored.
func (p Provider) Dir(scope Scope, project string) (string, error) {
	if scope == Local {
		if project == "" {
			return "", fmt.Errorf("local scope needs a project directory")
		}
		abs, err := filepath.Abs(project)
		if err != nil {
			return "", err
		}
		return filepath.Join(abs, filepath.FromSlash(p.local)), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, filepath.FromSlash(p.global)), nil
}

// Installed lists the skill folder names already present at dir. A missing
// directory is not an error: it just means nothing is installed yet.
func Installed(dir string) []string {
	es, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range es {
		// Stat rather than trust DirEntry.IsDir: skills are commonly symlinked
		// into a provider directory, and a symlink is not a dir to ReadDir.
		if _, err := os.Stat(filepath.Join(dir, e.Name(), "SKILL.md")); err == nil {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

// Sync copies one skill into a provider directory, replacing whatever is there.
// It reports whether the destination already existed, so the caller can say
// "updated" rather than "installed" and mean it.
func Sync(s Skill, dstRoot string) (dst string, replaced bool, err error) {
	dst = filepath.Join(dstRoot, s.Name)
	if _, err := os.Stat(dst); err == nil {
		replaced = true
		if err := os.RemoveAll(dst); err != nil {
			return "", false, err
		}
	}
	if err := copyTree(s.Dir, dst); err != nil {
		return "", replaced, err
	}
	return dst, replaced, nil
}

// Unsync removes a skill from a provider directory. It refuses to delete
// anything that is not a skill folder, so a mistyped name cannot take out a
// directory that happens to share it.
func Unsync(name, dstRoot string) error {
	dst := filepath.Join(dstRoot, name)
	if _, err := os.Stat(filepath.Join(dst, "SKILL.md")); err != nil {
		return fmt.Errorf("%s is not an installed skill", dst)
	}
	return os.RemoveAll(dst)
}

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		out := filepath.Join(dst, rel)
		if fi.IsDir() {
			return os.MkdirAll(out, 0o755)
		}
		if !fi.Mode().IsRegular() {
			return nil // skip symlinks and devices rather than following them
		}
		in, err := os.Open(p)
		if err != nil {
			return err
		}
		defer in.Close()
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		w, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fi.Mode().Perm())
		if err != nil {
			return err
		}
		defer w.Close()
		_, err = io.Copy(w, in)
		return err
	})
}
