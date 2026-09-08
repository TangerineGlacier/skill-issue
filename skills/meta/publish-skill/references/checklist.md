# Publish checklist

One skill at a time. Stop at the first unchecked box.

- [ ] `skill-issue doctor` is clean
- [ ] `grep -rIn -e '/Users/' -e '/home/' -e 'token' -e 'secret' skills/<tag>/<name>/` is empty
- [ ] the description says *when to reach for it*, not what the skill contains
- [ ] `references/` files are all linked from the body (doctor checks this)
- [ ] `scripts/` are executable and have no absolute paths
- [ ] the folder name, the `name:` field, and the heading all match
- [ ] tags are set, and the first one is the collection it should land in
- [ ] copied into the target repo, folder name unchanged
- [ ] one commit, this skill only
- [ ] pushed
- [ ] the repo is in `sources.txt`
- [ ] `skill-issue registry` lists it as `new`
