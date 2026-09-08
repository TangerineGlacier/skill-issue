package skill

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestBuildAndWrite(t *testing.T) {
	repo := t.TempDir()
	root := filepath.Join(repo, "skills")
	dir := filepath.Join(root, "docs", "a")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := "---\nname: a\ndescription: d\ntags: [docs]\n---\n## One\n## Two\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	// A second, deliberately broken skill so the catalog has a problem to carry.
	bad := filepath.Join(root, "docs", "b")
	os.MkdirAll(bad, 0o755)
	os.WriteFile(filepath.Join(bad, "SKILL.md"), []byte("---\nname: b\ntags: [docs]\n---\n"), 0o644)

	out, n, err := WriteCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("wrote %d skills, want 2", n)
	}
	if out != filepath.Join(repo, "catalog.json") {
		t.Fatalf("catalog written to %s", out)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var c Catalog
	if err := json.Unmarshal(b, &c); err != nil {
		t.Fatalf("catalog is not valid json: %v", err)
	}
	if len(c.Skills) != 2 || c.Skills[0].Name != "a" {
		t.Fatalf("skills = %+v", c.Skills)
	}
	if got := c.Skills[0].Path; got != "skills/docs/a" {
		t.Errorf("path = %q, want skills/docs/a (repo-relative, forward slashes)", got)
	}
	if len(c.Skills[0].Headings) != 2 {
		t.Errorf("headings = %v", c.Skills[0].Headings)
	}
	if len(c.Skills[1].Problems) == 0 {
		t.Error("the broken skill carries no problems, so the site cannot badge it")
	}
	if len(c.Problems) == 0 {
		t.Error("catalog-level problem list is empty")
	}
	// No temp file left behind.
	if _, err := os.Stat(out + ".tmp"); !os.IsNotExist(err) {
		t.Error("catalog.json.tmp was left behind")
	}
}
