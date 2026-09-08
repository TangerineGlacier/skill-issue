# skill-issue

`go install github.com/TangerineGlacier/skill-issue/cmd/skill-issue@latest`

Our own library of agent skills, plus two ways to read it: a Go TUI and a React
docs site. Both read the same generated catalog, so they never disagree.

```
skills/<tag>/<name>/SKILL.md   the source of truth — plain files, no database
catalog.json                   generated cache; delete it any time
cmd/skill-issue/               the TUI and CLI — go + bubbletea
skill/                         the library package both of them read
web/                           vite + react docs site
templates/                     what `skill-issue new` renders from
sources.txt                    public repos the registry offers
docs/adr/                      decisions worth not re-litigating
```

## Install

```bash
go install github.com/TangerineGlacier/skill-issue/cmd/skill-issue@latest
```

Then point it at a library — either clone this one, or start your own:

```bash
git clone https://github.com/TangerineGlacier/skill-issue.git
cd skill-issue && skill-issue link
```

`go install` puts the binary in `$(go env GOPATH)/bin` — make sure that is on
your `PATH`. If `skill-issue: command not found`, that is why:

```bash
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc
```

After `skill-issue link` it works from any directory on the machine. It finds
the library in this order, most specific first:

| Order | Source | For |
|---|---|---|
| 1 | `$SKILL_ISSUE_HOME` | scripts and CI |
| 2 | a `skills/` directory above the working directory | you are inside a checkout |
| 3 | the path registered by `skill-issue link` | everywhere else |

Walking up beats the registered path on purpose: inside a checkout, that
checkout is what you meant. `skill-issue where` prints which one won and why.

## Working on it

Edit anything under `cmd/skill-issue/` or `skill/`, then reinstall:

```bash
go install ./cmd/skill-issue
```

That overwrites the binary in `$(go env GOPATH)/bin`, so the next
`skill-issue` is your build. It must be run from inside the repo; from
anywhere else, point at it:

```bash
go build -C ~/Documents/skill-issue -o "$(go env GOPATH)/bin/skill-issue" ./cmd/skill-issue
```

While iterating, skip the install entirely:

```bash
go run ./cmd/skill-issue            # the TUI, straight from source
go run ./cmd/skill-issue doctor     # or any subcommand
go test ./...                       # before you install
```

To see a layout change without launching anything, the snapshot tests print
every screen as text:

```bash
go test ./cmd/skill-issue -run Snapshot -v
```

## The TUI

```bash
skill-issue          browse
```

| Screen | Key from browse | What it is for |
|---|---|---|
| browse | — | collections, skills, live SKILL.md preview |
| read | `⏎` | outline, bundled files, rendered body |
| edit | `e` | edit a SKILL.md in place, `ctrl+s` to save |
| new | `n` | create a skill; nothing is written until you confirm |
| registry | `d` | install from a public repo, filed by tag |
| sync | `s` | copy skills into an agent's directory, global or per-project |
| past used | `u` | what you have actually reached for, and what is missing |
| doctor | `v` | everything wrong with the library, worst first |

`?` shows the full keymap for whatever screen you are on. `esc`/`q` always
goes back one level; `ctrl+c` always quits.

The status bar shows as many keys as fit and drops whole ones as the terminal
narrows — it never cuts a key in half, and it always keeps `? help`.

Headless equivalents, for CI and for piping:

```bash
skill-issue doctor     list every problem; exits 1 if any are errors
skill-issue fix        apply every automatic fix
skill-issue catalog    regenerate catalog.json
skill-issue registry   list what the configured sources offer
skill-issue history    skills you have used, and which of them you do not have
skill-issue providers  the agents it knows about, and what is installed for each
skill-issue sync       copy skills into an agent's directory (see `skill-issue sync -h`)
skill-issue where      which library skill-issue is using, and why
```

## Global and local

A skill in this repo does nothing until it is copied where an agent will read
it. `skill-issue sync` does that, on two axes.

**Which agent** — `-provider`:

| id | agent | global | local |
|---|---|---|---|
| `claude` | Claude Code | `~/.claude/skills` | `<project>/.claude/skills` |
| `codex` | Codex CLI | `~/.codex/skills` | `<project>/.codex/skills` |
| `cursor` | Cursor | `~/.cursor/skills` | `<project>/.cursor/skills` |
| `opencode` | opencode | `~/.config/opencode/skills` | `<project>/.opencode/skills` |

**Which scope** — `-scope`: `global` is per-user and applies everywhere;
`local` writes into one project, where it can be committed and shared with
whoever clones it.

```bash
skill-issue sync -list                                 what is installed for Claude, globally
skill-issue sync write-adr go-table-tests               copy two skills there
skill-issue sync -all -provider codex                   the whole library, for Codex
skill-issue sync -scope local -project ~/work/api -all  into one project instead
skill-issue sync -remove write-adr                      take it back out
```

Copying is replace-then-write, not merge: a file left over from a previous
version of a skill is worse than a missing one.

## Past used skills

`skill-issue history` reads the agent logs on this machine and reports what you
have
actually reached for, so the library grows from evidence rather than from
guessing.

```bash
skill-issue history            everything, most used first
skill-issue history -missing   only skills you use that this library does not have
```

The last column is the point. A skill you have used twenty times and do not
have is the next one worth writing or installing.

Only Claude Code keeps a machine-readable log — both `Skill` tool calls and
`/name` invocations, with a timestamp and a project. Cursor stores chats in an
undocumented SQLite schema, and Codex and opencode keep no skill log at all.
Those three are listed in the output with a note saying so: "never used" and
"cannot read" must not look the same.

## The docs site

```bash
go run ./tui catalog      # catalog.json is generated, not committed
npm install --prefix web
npm run dev --prefix web
```

`npm run build --prefix web` validates the catalog against what the site reads
before building, so a stale catalog fails loudly instead of rendering
`undefined`.

## The SKILL.md contract

```markdown
---
name: write-adr
description: One sentence saying when to reach for this. It is the only thing
  the model matches against, so it decides whether the skill ever loads.
tags: [docs, architecture]
---

# write-adr

## What it does
## When to reach for it
## Prerequisites
## Common questions
```

- `name` must equal the folder name, kebab-case.
- `description` is required, max 1024 characters.
- **The first tag decides the folder.** `tags: [docs, architecture]` lives in
  `skills/docs/`. No tags means `skills/unfiled/`, and doctor keeps asking.
- Optional siblings: `references/` (loaded on demand), `scripts/`, `assets/`.
  A `references/` file the body never links is dead weight, and doctor says so.

## What doctor checks

| Code | Level | Auto-fix |
|---|---|---|
| `unreadable` | error | no — the file has to parse first |
| `no-description` | error | no — only you know when to reach for it |
| `untagged` in `unfiled/` | error | no — only you know what it is about |
| `no-name` | error | yes — set to the folder name |
| `description-too-long` | warn | no — cutting it is an editorial call |
| `untagged` elsewhere | warn | yes — tag it from the folder it sits in |
| `misfiled` | warn | yes — move the folder to match the first tag |
| `name-mismatch` | warn | yes — the folder wins, the frontmatter follows |
| `unlinked-reference` | warn | no — link it or delete it |

## Tests

```bash
go test ./...              # includes one network test against sources.txt
go test -short ./...       # offline
npm run test --prefix web  # catalog matches what the site reads
```

`go test ./cmd/skill-issue -run Snapshot -v` prints the screens as text, which is how the
TUI layout is reviewed.

```bash
go test ./cmd/skill-issue -run xxx -bench . -benchtime 20x
```

The benchmarks exist to answer one question — is any `View()` slow enough to
need caching — because [ADR
0001](docs/adr/0001-tui-design-follows-bubbletea-designer.md)
says no on measured numbers. Re-run them before adding a cache.

## Design

The TUI's structure comes from the `bubbletea-designer` skill in this library
(`skills/go/bubbletea-designer/`). What was taken from it, what was not, and
why, is recorded in
[ADR 0001](docs/adr/0001-tui-design-follows-bubbletea-designer.md).
