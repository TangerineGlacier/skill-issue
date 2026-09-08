---
name: write-adr
description: Write an Architecture Decision Record for a decision that is hard to reverse, surprising without context, and involved a real trade-off. Use after a design discussion settles, when someone asks "why is it done this way", or when a choice was made that a future reader would otherwise undo by accident.
tags: [docs, architecture]
---

# write-adr

## What it does

Turns a decision that has already been made into a short, dated record in `docs/adr/`,
so the next person — or the next model — does not re-litigate it or silently undo it.

One decision per file. Numbered, never edited after acceptance; superseded by a new
ADR that links back.

## When to reach for it

A decision earns an ADR when **all three** are true. Two out of three is a comment in
the code, not a document.

| Test | Meaning |
|---|---|
| Hard to reverse | Undoing it means touching many files, migrating data, or breaking an interface. |
| Surprising without context | A competent reader would assume the other choice. |
| Real trade-off | You gave something up. If nothing was given up, there was no decision. |

Do **not** write one for: library choices with an obvious default, formatting,
anything still being argued about (that is a proposal, not a record).

## Prerequisites

- A `docs/adr/` directory. Create it with the first record.
- The decision is actually settled. If it is not, stop and settle it first.

## How to use

1. Find the next number: `ls docs/adr | tail -1`.
2. Write `docs/adr/NNNN-kebab-title.md` using [the template](references/template.md).
3. Fill **Context** with the forces, not the conclusion. If Context does not make the
   decision feel inevitable, the Context is incomplete.
4. Fill **Consequences** with what got worse, too. An ADR listing only benefits is
   marketing.
5. Link it from wherever the decision shows up in code.

## Common questions

**Where do the sections come from?** Michael Nygard's format: Title, Status, Context,
Decision, Consequences. Do not invent new sections; five is enough.

**Can I edit an accepted ADR?** Only typos. To change the decision, write a new ADR
with `Status: Accepted, supersedes 0004` and set the old one to
`Status: Superseded by 0011`.

**What if the decision was made months ago?** Write it anyway, dated today, with a
note that it records existing practice. A late record beats none.

**Isn't this what commit messages are for?** Commit messages explain a change. An ADR
explains a constraint that outlives every change.
