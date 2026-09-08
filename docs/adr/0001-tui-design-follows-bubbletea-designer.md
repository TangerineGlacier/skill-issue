# 0001. Follow the bubbletea-designer skill, and record where we do not

- **Status:** Accepted
- **Date:** 2026-09-08
- **Deciders:** sreevatsansridhar

## Context

The TUI's structure was taken from the `bubbletea-designer` skill, which is
installed in this library at `skills/go/bubbletea-designer/`. Its
recommendations are generic, and a few of them cost complexity that this
application does not need. Without a record, every later contributor either
re-applies all of it or quietly ignores all of it.

The skill was fetched from its source repo
(`Human-Frontier-Labs-Inc/human-frontier-labs-marketplace`, `plugins/bubbletea-designer`)
rather than from the mcpmarket listing page, which rate-limits and renders the
SKILL.md client-side only.

## Decision

We follow the skill's structural guidance, and depart from three of its
recommendations for reasons measured on this codebase.

### Applied

| Skill guidance | Where |
|---|---|
| Pattern 2, multi-view state machine | `cmd/skill-issue/app.go` — one `screen` enum, children emit messages and never switch screens themselves |
| Pattern 4, master–detail | `browse.go`, `syncview.go`, `registryview.go` — list plus a target/preview pane |
| Pattern 5, form flow | `newskill.go` — `[]textinput.Model` plus a focus index |
| Pattern 6, progress tracker | `registryview.go` — queue, `progress.Model`, `spinner.Model` |
| `viewport.Model` for scrollable content | `doc.go`, the browse preview |
| Responsive layout on `WindowSizeMsg` | every screen drops columns rather than truncating them |
| Never block in `Update()` | the 180 ms transcript scan is a `tea.Cmd`; see below |
| Ready state before rendering | every `View()` returns early while `w == 0` |
| Discoverable help (`?`) and the help-overlay pattern | `cmd/skill-issue/help.go` |
| One component per file | one screen per file under `cmd/skill-issue/`: `browse.go`, `doc.go`, `editor.go`, `doctorview.go`, `newskill.go`, `registryview.go`, `syncview.go`, `historyview.go` |

### Not applied

**"Cache expensive renders."** Measured, not assumed: `BenchmarkBrowseView`
is 0.35 ms and `BenchmarkDocJump` is 0.30 ms on this library. A dirty-flag
cache would add a state bug for no perceptible gain. `cmd/skill-issue/bench_test.go` exists so
this can be re-checked rather than re-argued; add the cache when a benchmark
says to.

The one genuinely expensive read is `skill.History` at ~180 ms across 144
transcripts, and that is why it loads in a command instead of in `Update`.

**`list.Model` and `table.Model`.** Every list here needs a fixed-width
multi-column row with a right-aligned status marker, and the two panes must
line up with each other. Both components are hand-rolled instead. Reconsider if
a screen needs pagination or the component's filtering.

**The skill's file split (`update.go`, `view.go`, `messages.go`).** Update and
View live next to the model they belong to, one screen per file. Splitting by
method rather than by screen means a change to one screen touches four files.
Messages are in `cmd/skill-issue/app.go` because they are the contract between screens.

## Consequences

What becomes easier:

- A new screen is one file plus one entry in the `screen` enum and one in the
  keymap table.
- `cmd/skill-issue/help.go` is the single source of truth for keys: the status bar and the `?`
  overlay render from the same table, so a key cannot be documented in one and
  missing from the other. Adding a key without documenting it is now the
  awkward path.

What becomes harder:

- Adding pagination or component-level filtering means writing it, since we do
  not use `list.Model`.
- The keymap table is a second place to edit when adding a key binding. That is
  the cost of the two surfaces agreeing.

What we are now committed to:

- Measuring before optimising a render. `bench_test.go` is the check.
- Every screen exposing its keys through `keysFor`, so `?` is complete.
