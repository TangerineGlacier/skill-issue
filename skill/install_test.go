package skill

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// End-to-end against the real source in sources.txt. Skipped in short mode so
// `go test -short ./...` stays offline.
func TestInstallEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("network")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	rs, err := FetchSource(ctx, "Human-Frontier-Labs-Inc/human-frontier-labs-marketplace")
	if err != nil {
		t.Skipf("source unreachable: %v", err)
	}
	var target Remote
	for _, r := range rs {
		if r.Name == "effective-go" {
			target = r
		}
	}
	if target.Name == "" {
		t.Skip("effective-go not in the source any more")
	}
	target.Retag = "go"

	root := filepath.Join(t.TempDir(), "skills")
	dst, err := Install(ctx, root, target)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(root, "go", "effective-go"); dst != want {
		t.Fatalf("installed to %s, want %s", dst, want)
	}
	s, err := Load(dst)
	if err != nil {
		t.Fatalf("installed skill does not load: %v", err)
	}
	if s.PrimaryTag() != "go" {
		t.Errorf("retag was not written into the file: tags = %v", s.Tags)
	}
	// It must land clean: installing then immediately failing doctor on
	// "misfiled" would make the whole flow useless.
	skills, errs := LoadAll(root)
	for _, p := range Check(skills, errs) {
		if p.Code == "misfiled" || p.Code == "untagged" {
			t.Errorf("freshly installed skill is %s: %s", p.Code, p.Msg)
		}
	}
	if _, err := os.Stat(filepath.Join(dst, "SKILL.md")); err != nil {
		t.Error(err)
	}
}
