---
name: publish-skill
description: Upload a skill from the local library to a public GitHub repo so other people and other machines can install it, and register that repo as a source. Use when asked to publish, share, upload, or release a skill, or to push skills to a public repo.
tags: [meta, git]
---

# publish-skill

## What it does

Copies one skill folder out of the library into a public repo, commits it, and
pushes — so `skill-issue registry` on any machine can find and install it.

A skill is a directory of plain files. Publishing is therefore a copy and a
push, not a build step and not a package format. Anything that looks like
packaging here is a mistake.

## When to reach for it

| Situation | Do this |
|---|---|
| A skill works, and someone else wants it | publish it |
| A skill works, and only you want it | `skill-issue sync` — no repo needed |
| A skill is half-written | finish it; a published half-skill gets installed |
| You want a skill you saw elsewhere | `skill-issue registry`, not this |

Publishing is public and hard to take back: a pushed commit is fetchable even
after a force-push. Read the skill for machine paths, project names, tokens and
client details **before** the first push, not after.

## Prerequisites

- `gh auth status` succeeds.
- `skill-issue doctor` is clean for the skill you are publishing. A skill that fails
  doctor here will fail it on everyone else's machine too.
- A target repo you can push to. One repo holds many skills; do not make a repo
  per skill.

## How to use

1. **Check the skill.** It has to pass on its own before it travels.

   ```bash
   skill-issue doctor
   ```

2. **Read it for anything private.** Skim the whole folder, `references/` and
   `scripts/` included.

   ```bash
   grep -rIn -e '/Users/' -e '/home/' -e 'localhost' -e 'token' -e 'secret' \
             -e 'api[_-]key' skills/<tag>/<name>/
   ```

   Absolute home paths are the usual find. Replace them with `~` or make them
   an argument.

3. **Copy it into the target repo**, keeping the folder name — it is the
   skill's identity, and `skill-issue` files the copy by the skill's first tag, not by
   where it sits in your repo.

   ```bash
   REPO=~/src/my-skills
   cp -R skills/<tag>/<name> "$REPO/skills/<name>"
   ```

4. **Commit and push.** One skill per commit, so a bad one can be reverted
   without taking the others with it.

   ```bash
   cd "$REPO"
   git add "skills/<name>"
   git commit -m "add <name> skill"
   git push
   ```

5. **Register the repo as a source**, once per repo, so it shows up in
   `skill-issue registry` and in the download screen.

   ```bash
   echo "owner/my-skills" >> sources.txt
   skill-issue registry | grep <name>
   ```

6. **Verify from the outside.** Install it into a throwaway directory and check
   it still parses — this catches the file you forgot to `git add`.

   ```bash
   skill-issue registry            # it should appear as "new"
   ```

See [references/checklist.md](references/checklist.md) for the same steps as a
copy-paste checklist.

## Common questions

**Does the repo need a manifest or index?** No. `skill-issue` finds skills by walking
the repo for `SKILL.md`, so any layout works. `skills/<name>/` is conventional
and keeps the repo readable.

**How do people get updates?** They re-install. `skill-issue` backs the old folder up to
`.bak` before replacing it, so a bad update is one `mv` from being undone.
There is deliberately no version field: the repo's history is the version.

**What if the skill has no tags?** Add them before publishing. Untagged skills
land in `unfiled/` on every machine that installs them, and `skill-issue doctor` will
nag each of those people rather than you.

**Can I publish the whole library at once?** You can, and you should not.
Publishing is the moment you read a skill as a stranger would; doing four at a
time is how a home directory path gets shipped.

**What about a private repo?** `skill-issue` fetches over the GitHub API and sends
`GITHUB_TOKEN` when it is set, so a private repo works for anyone whose token
can read it. It is not a way to share with people who cannot.
