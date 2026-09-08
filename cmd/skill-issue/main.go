package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TangerineGlacier/skill-issue/skill"
)

const usage = `skill-issue — your library of agent skills

  skill-issue                  browse the library

  skill-issue doctor           print every problem; exits 1 if any are errors
  skill-issue fix              apply every automatic fix
  skill-issue catalog          regenerate catalog.json

  skill-issue sync [flags]     copy skills into an agent's directory (see -h)
  skill-issue providers        the agents it knows about, and what is installed
  skill-issue history          skills you have actually used, and what is missing
  skill-issue registry         what the configured public sources offer

  skill-issue link             register this library so it works from anywhere
  skill-issue where            print the library it is using, and why

The library is found via $SKILL_ISSUE_HOME, then a skills/ directory above the
working directory, then the one registered by ` + "`skill-issue link`" + `.
`

func main() {
	cmd, args := "", []string(nil)
	if len(os.Args) > 1 {
		cmd, args = os.Args[1], os.Args[2:]
	}

	// providers and help need no library.
	switch cmd {
	case "providers":
		os.Exit(runProviders())
	case "-h", "--help", "help":
		fmt.Print(usage)
		return
	}

	root, err := skill.ResolveRoot(".")
	if err != nil {
		die(err)
	}

	switch cmd {
	case "":
		p := tea.NewProgram(NewApp(root), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			die(err)
		}
	case "doctor":
		os.Exit(runDoctor(root, false))
	case "fix":
		os.Exit(runDoctor(root, true))
	case "catalog":
		out, n, err := skill.WriteCatalog(root)
		if err != nil {
			die(err)
		}
		fmt.Printf("%s — %d skills\n", out, n)
	case "sync":
		os.Exit(runSync(root, args))
	case "history":
		os.Exit(runHistory(root, args))
	case "registry":
		os.Exit(runRegistry(root))
	case "link":
		os.Exit(runLink(root))
	case "where":
		fmt.Println(root)
		if env := os.Getenv("SKILL_ISSUE_HOME"); env != "" {
			fmt.Println("via $SKILL_ISSUE_HOME")
		} else if lib := skill.LoadConfig().Library; lib == root {
			fmt.Println("via " + skill.ConfigPath())
		} else {
			fmt.Println("via a skills/ directory above the working directory")
		}
	default:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
}

// runDoctor is the headless half of the doctor screen, for CI and for piping.
func runDoctor(root string, apply bool) int {
	for pass := 0; ; pass++ {
		skills, errs := skill.LoadAll(root)
		ps := skill.Check(skills, errs)
		if !apply {
			return report(ps)
		}
		fixed := false
		for _, p := range ps {
			if !p.Fixable {
				continue
			}
			what, err := skill.Fix(p)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  %s: %v\n", p.Skill, err)
				continue
			}
			fmt.Printf("  fixed %s — %s\n", p.Skill, what)
			fixed = true
			break // a fix can move a folder, invalidating every later path
		}
		if !fixed || pass > 200 {
			return report(ps)
		}
	}
}

func report(ps []skill.Problem) int {
	if len(ps) == 0 {
		fmt.Println("no problems")
		return 0
	}
	worst := 0
	for _, p := range ps {
		fmt.Printf("%-5s %-22s %s\n", p.Level, p.Skill, p.Msg)
		if p.Level == skill.Error {
			worst = 1
		}
	}
	fmt.Printf("\n%d problem(s)\n", len(ps))
	return worst
}

func die(err error) {
	fmt.Fprintln(os.Stderr, "skill-issue:", err)
	os.Exit(1)
}
