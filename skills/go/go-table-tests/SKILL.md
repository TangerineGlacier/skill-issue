---
name: go-table-tests
description: Write Go tests as table-driven subtests with t.Run and a cases slice, rather than one function per case or a chain of if-statements. Use when adding tests to Go code, when a test function is growing repeated blocks, or when asked to improve Go test coverage.
tags: [go, testing]
---

# go-table-tests

## What it does

Turns repeated Go test blocks into a single `cases` slice driven by `t.Run`, so
adding a case is one line and a failure names itself.

## When to reach for it

- A test function has two or more near-identical blocks differing only in inputs.
- You are about to write `TestFooEmpty`, `TestFooNil`, `TestFooLong`.
- A bug report arrives with a concrete input — it becomes one row.

Skip it when there is genuinely one case, or when each case needs different setup
and teardown. A table with a `switch` inside it is two tests wearing a coat.

## Prerequisites

Standard library `testing` only. No assertion framework — `got`/`want` and
`t.Errorf` are the idiom, and a third-party matcher makes failures harder to read,
not easier.

## The shape

```go
func TestParse(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    Skill
		wantErr bool
	}{
		{name: "minimal", in: "---\nname: a\n---\n", want: Skill{Name: "a"}},
		{name: "no frontmatter", in: "# a\n", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr && !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}
```

Rules that make the difference:

- `name` first, and lowercase with spaces — it becomes `TestParse/no_frontmatter`.
- `t.Fatalf` when continuing would panic; `t.Errorf` otherwise, so one run reports
  every failing field.
- Print `got` before `want`. Every Go reader expects that order.
- No `t.Parallel()` until the tests are actually slow — it reorders output for
  nothing.

## Common questions

**Map or slice for the cases?** Slice. Map iteration order is random, so failures
appear in a different order each run.

**Where do fixtures go?** `testdata/`, read in the case body. The `go` tool ignores
that directory.

**How many fields before the table is too wide?** If a case needs more than about
six fields, the function under test is doing more than one thing. Split the
function, not the table.
