package skill

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Remote is a skill living in a public repo, before it is installed.
type Remote struct {
	Source      string // owner/repo
	Name        string
	Description string
	Tags        []string
	Dir         string   // directory inside the repo
	Files       []string // repo-relative paths under Dir
	Installed   bool     // a skill of this name already exists locally
	Retag       string   // operator override: file it here regardless of tags
}

// Tag is the folder this skill belongs in. An explicit retag wins, then the
// first frontmatter tag, and an untagged skill goes to unfiled/ where doctor
// will keep asking about it.
func (r Remote) Tag() string {
	switch {
	case r.Retag != "":
		return r.Retag
	case len(r.Tags) > 0:
		return r.Tags[0]
	}
	return "unfiled"
}

// Target is where Install will put it.
func (r Remote) Target(root string) string {
	return filepath.Join(root, r.Tag(), r.Name)
}

var client = &http.Client{Timeout: 20 * time.Second}

func get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "skill-issue")
	// A token lifts the 60/hour anonymous limit. Optional: without it the
	// registry still works, just not many times in a row.
	if t := os.Getenv("GITHUB_TOKEN"); t != "" {
		req.Header.Set("Authorization", "Bearer "+t)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusForbidden && strings.Contains(string(b), "rate limit") {
			return nil, fmt.Errorf("github rate limit reached — set GITHUB_TOKEN to raise it")
		}
		return nil, fmt.Errorf("%s: %s", url, resp.Status)
	}
	return b, nil
}

type treeResp struct {
	Tree []struct {
		Path string `json:"path"`
		Type string `json:"type"`
	} `json:"tree"`
	Truncated bool `json:"truncated"`
}

// FetchSource lists every skill in one public repo. One API call for the tree,
// then one raw fetch per SKILL.md — raw.githubusercontent is not on the API
// rate limit, so a big repo costs one API request, not fifty.
func FetchSource(ctx context.Context, source string) ([]Remote, error) {
	b, err := get(ctx, "https://api.github.com/repos/"+source+"/git/trees/HEAD?recursive=1")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", source, err)
	}
	var tr treeResp
	if err := json.Unmarshal(b, &tr); err != nil {
		return nil, fmt.Errorf("%s: %w", source, err)
	}

	dirs := map[string]bool{}
	files := map[string][]string{}
	for _, e := range tr.Tree {
		if e.Type != "blob" {
			continue
		}
		if path.Base(e.Path) == "SKILL.md" {
			dirs[path.Dir(e.Path)] = true
		}
	}
	for _, e := range tr.Tree {
		if e.Type != "blob" {
			continue
		}
		for d := range dirs {
			if e.Path == path.Join(d, path.Base(e.Path)) || strings.HasPrefix(e.Path, d+"/") {
				files[d] = append(files[d], e.Path)
				break
			}
		}
	}

	out := make([]Remote, 0, len(dirs))
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for d := range dirs {
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			raw, err := get(ctx, "https://raw.githubusercontent.com/"+source+"/HEAD/"+d+"/SKILL.md")
			if err != nil {
				return
			}
			s, err := Parse(string(raw))
			if err != nil {
				return
			}
			name := s.Name
			if name == "" {
				name = path.Base(d)
			}
			sort.Strings(files[d])
			mu.Lock()
			out = append(out, Remote{Source: source, Name: name, Description: s.Description,
				Tags: s.Tags, Dir: d, Files: files[d]})
			mu.Unlock()
		}(d)
	}
	wg.Wait()
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		// Same skill published at two paths (plugins/x and skills/x, say):
		// prefer the shallower one, then the alphabetically first.
		di, dj := strings.Count(out[i].Dir, "/"), strings.Count(out[j].Dir, "/")
		if di != dj {
			return di < dj
		}
		return out[i].Dir < out[j].Dir
	})
	return dedupe(out), nil
}

// dedupe keeps one copy per name. A repo that ships the same skill under two
// paths is offering one skill, and installing both would just race for the
// same destination folder.
func dedupe(rs []Remote) []Remote {
	seen := map[string]bool{}
	out := rs[:0]
	for _, r := range rs {
		if seen[r.Name] {
			continue
		}
		seen[r.Name] = true
		out = append(out, r)
	}
	return out
}

// Fetch lists every skill across every source, marking the ones already here.
func Fetch(ctx context.Context, sources []string, local []Skill) ([]Remote, []error) {
	have := map[string]bool{}
	for _, s := range local {
		have[s.Name] = true
	}
	var all []Remote
	var errs []error
	for _, src := range sources {
		rs, err := FetchSource(ctx, src)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for i := range rs {
			rs[i].Installed = have[rs[i].Name]
		}
		all = append(all, rs...)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Installed != all[j].Installed {
			return !all[i].Installed // not-yet-installed first: that is what you came for
		}
		return all[i].Name < all[j].Name
	})
	return all, errs
}

// Install downloads one remote skill into the folder its first tag names.
// An existing folder is moved aside rather than merged, because a merge leaves
// files from two different versions in one skill.
func Install(ctx context.Context, root string, r Remote) (string, error) {
	dst := r.Target(root)
	if _, err := os.Stat(dst); err == nil {
		bak := dst + ".bak"
		_ = os.RemoveAll(bak)
		if err := os.Rename(dst, bak); err != nil {
			return "", err
		}
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return "", err
	}
	for _, f := range r.Files {
		rel, err := filepath.Rel(r.Dir, f)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		out := filepath.Join(dst, filepath.FromSlash(rel))
		// Refuse to write outside the destination, whatever the repo claims.
		if !strings.HasPrefix(out, dst+string(os.PathSeparator)) && out != dst {
			continue
		}
		b, err := get(ctx, "https://raw.githubusercontent.com/"+r.Source+"/HEAD/"+f)
		if err != nil {
			return "", fmt.Errorf("%s: %w", f, err)
		}
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(out, b, 0o644); err != nil {
			return "", err
		}
	}
	// Record the retag in the file itself. Without this the skill lands in the
	// chosen folder with frontmatter that disagrees, and doctor immediately
	// reports it as misfiled — technically correct and completely unhelpful.
	if r.Retag != "" {
		if err := setFrontmatter(filepath.Join(dst, "SKILL.md"), "tags", "["+r.Retag+"]"); err != nil {
			return dst, err
		}
	}
	return dst, nil
}

// Sources reads sources.txt next to the skills root: one owner/repo per line,
// # comments ignored. A missing file means no remote sources, not an error.
func Sources(root string) []string {
	b, err := os.ReadFile(filepath.Join(filepath.Dir(root), "sources.txt"))
	if err != nil {
		return nil
	}
	var out []string
	for _, l := range strings.Split(string(b), "\n") {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		out = append(out, l)
	}
	return out
}
