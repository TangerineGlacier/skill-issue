package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TangerineGlacier/skill-issue/skill"
)

// runLink registers the library this command was run inside, so `skill-issue` works
// from anywhere afterwards.
func runLink(root string) int {
	if err := skill.SaveConfig(skill.Config{Library: root}); err != nil {
		fmt.Fprintln(os.Stderr, "skill-issue:", err)
		return 1
	}
	fmt.Printf("linked %s\n%s\n", root, skill.ConfigPath())
	return 0
}

func runProviders() int {
	home, _ := os.UserHomeDir()
	short := func(p string) string {
		if home != "" && strings.HasPrefix(p, home) {
			return "~" + p[len(home):]
		}
		return p
	}
	fmt.Printf("%-10s %-14s %-34s %s\n", "ID", "AGENT", "GLOBAL", "INSTALLED")
	for _, p := range skill.Providers {
		dir, err := p.Dir(skill.Global, "")
		if err != nil {
			continue
		}
		names := skill.Installed(dir)
		for _, extra := range p.AlsoRead {
			names = append(names, skill.Installed(filepath.Join(home, extra))...)
		}
		fmt.Printf("%-10s %-14s %-34s %d\n", p.ID, p.Label, short(dir), len(names))
	}
	fmt.Printf("\nlocal scope writes to <project>/<provider dir> instead; see `skill-issue sync -h`\n")
	return 0
}

// runSync copies skills from the library into a provider's directory.
func runSync(root string, args []string) int {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	provider := fs.String("provider", "claude", "which agent to install for")
	scope := fs.String("scope", "global", "global (per-user) or local (per-project)")
	project := fs.String("project", ".", "project root, for -scope local")
	list := fs.Bool("list", false, "show what is installed there and exit")
	remove := fs.Bool("remove", false, "remove the named skills instead of copying them")
	all := fs.Bool("all", false, "every skill in the library")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, "skill-issue sync [flags] [skill...]\n\n"+
			"  skill-issue sync -list                      what is installed for Claude, globally\n"+
			"  skill-issue sync write-adr go-table-tests   copy two skills there\n"+
			"  skill-issue sync -all -provider codex       copy the whole library to Codex\n"+
			"  skill-issue sync -scope local -project ~/x  install into that project instead\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}

	p, err := skill.ProviderByID(*provider)
	if err != nil {
		fmt.Fprintln(os.Stderr, "skill-issue:", err)
		return 2
	}
	sc := skill.Scope(*scope)
	if sc != skill.Global && sc != skill.Local {
		fmt.Fprintln(os.Stderr, "skill-issue: -scope must be global or local")
		return 2
	}
	dst, err := p.Dir(sc, *project)
	if err != nil {
		fmt.Fprintln(os.Stderr, "skill-issue:", err)
		return 2
	}

	if *list {
		fmt.Println(dst)
		names := skill.Installed(dst)
		if len(names) == 0 {
			fmt.Println("  (nothing installed)")
			return 0
		}
		for _, n := range names {
			fmt.Println("  " + n)
		}
		return 0
	}

	if *remove {
		if len(fs.Args()) == 0 {
			fmt.Fprintln(os.Stderr, "skill-issue: name at least one skill to remove")
			return 2
		}
		for _, n := range fs.Args() {
			if err := skill.Unsync(n, dst); err != nil {
				fmt.Fprintln(os.Stderr, "  "+err.Error())
				continue
			}
			fmt.Printf("  removed %s\n", n)
		}
		return 0
	}

	library, _ := skill.LoadAll(root)
	want := map[string]bool{}
	for _, n := range fs.Args() {
		want[n] = true
	}
	if !*all && len(want) == 0 {
		fs.Usage()
		return 2
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "skill-issue:", err)
		return 1
	}

	n := 0
	for _, s := range library {
		if !*all && !want[s.Name] {
			continue
		}
		delete(want, s.Name)
		_, replaced, err := skill.Sync(s, dst)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s: %v\n", s.Name, err)
			continue
		}
		verb := "installed"
		if replaced {
			verb = "updated  "
		}
		fmt.Printf("  %s %s\n", verb, s.Name)
		n++
	}
	code := 0
	for name := range want {
		fmt.Fprintf(os.Stderr, "  no skill named %q in the library\n", name)
		code = 1
	}
	fmt.Printf("\n%d skill(s) → %s (%s, %s)\n", n, dst, p.Label, sc)
	return code
}

// runHistory prints what you have actually reached for, and what of that is
// missing from the library.
func runHistory(root string, args []string) int {
	fs := flag.NewFlagSet("history", flag.ContinueOnError)
	missing := fs.Bool("missing", false, "only skills you use that the library does not have")
	limit := fs.Int("n", 25, "how many rows")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	uses, sources := skill.History(root)
	if *missing {
		uses = skill.Suggestions(uses)
	}
	if len(uses) == 0 {
		fmt.Println("no skill usage found")
	} else {
		fmt.Printf("%-28s %6s  %-14s %-6s %s\n", "SKILL", "USED", "LAST", "IN LIB", "PROJECTS")
		for i, u := range uses {
			if i >= *limit {
				break
			}
			in := "no"
			if u.InLibrary {
				in = "yes"
			}
			fmt.Printf("%-28s %6d  %-14s %-6s %s\n", u.Name, u.Count,
				u.Last.Format("2006-01-02"), in, strings.Join(u.Projects, ", "))
		}
	}
	fmt.Println("\nSOURCES")
	for _, s := range sources {
		note := s.Note
		if note == "" {
			note = fmt.Sprintf("%d transcripts, %d uses", s.Files, s.Uses)
		}
		fmt.Printf("  %-10s %s\n", s.ID, note)
	}
	return 0
}

// runRegistry lists what the configured public sources offer.
func runRegistry(root string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	sources := skill.Sources(root)
	if len(sources) == 0 {
		fmt.Fprintln(os.Stderr, "skill-issue: no sources — add owner/repo lines to sources.txt")
		return 2
	}
	local, _ := skill.LoadAll(root)
	rs, errs := skill.Fetch(ctx, sources, local)
	for _, e := range errs {
		fmt.Fprintln(os.Stderr, "skill-issue:", e)
	}
	for _, r := range rs {
		state := "new"
		if r.Installed {
			state = "local"
		}
		fmt.Printf("%-6s %-28s %-24s %s\n", state, r.Name, strings.Join(r.Tags, ","), r.Source)
	}
	fmt.Printf("\n%d skills across %d source(s)\n", len(rs), len(sources))
	return 0
}
