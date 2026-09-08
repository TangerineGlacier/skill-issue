---
name: vault-note
description: Append a timestamped entry to a note in the Obsidian vault, creating the note from its folder template when it does not exist yet, and linking it from the relevant index. Use when asked to jot something down, log a decision, or capture a note into the vault.
tags: [obsidian, notes]
---

# vault-note

## What it does

Writes a note into the Obsidian vault at `~/Documents/notebook`, in the folder
the content belongs to, with the frontmatter that folder's notes already use —
so the note is indexed and linked rather than orphaned.

## When to reach for it

- "Jot this down", "capture this", "add this to my notes".
- A decision or finding worth keeping past the current conversation.
- Anything the vault already has a home for.

Do not reach for it to dump a whole transcript. A note is the conclusion, not the
log that produced it.

## Prerequisites

- The vault exists at `~/Documents/notebook` and is a working directory.
- Read a sibling note **before writing**. Frontmatter keys vary per folder and
  guessing them breaks Dataview queries.

## How to use

1. Pick the folder from the content. Check with `ls ~/Documents/notebook`.
2. Read one existing note in that folder and copy its frontmatter keys exactly.
3. Write the note. Append with a `## YYYY-MM-DD` heading if the note exists.
4. Add `[[wikilinks]]` to at least one existing note, and add a line to that
   folder's index note. A note nothing links to will not be found again.

## Common questions

**Filename?** Title Case with spaces, matching the `title:` field. The vault does
not use kebab-case for note filenames.

**Tags?** Reuse tags that already exist in the vault. A tag used once is noise.

**What if no folder fits?** That is a signal the vault needs a new folder, which is
a decision for the user — ask rather than inventing one.
