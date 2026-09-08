package skill

import (
	"path/filepath"
	"testing"
)

// scanClaudeTranscript has to count both ways a skill gets used — the Skill
// tool and the user typing /name — and must not count terminal commands.
func TestScanClaudeTranscript(t *testing.T) {
	lines := []string{
		`{"timestamp":"2026-08-01T10:00:00Z","cwd":"/Users/x/alpha","message":{"content":[{"type":"tool_use","name":"Skill","input":{"skill":"resolv"}}]}}`,
		`{"timestamp":"2026-08-02T10:00:00Z","cwd":"/Users/x/beta","message":{"content":[{"type":"tool_use","name":"Skill","input":{"skill":"ponytail:ponytail"}}]}}`,
		`{"timestamp":"2026-08-03T10:00:00Z","cwd":"/Users/x/alpha","message":{"content":[{"type":"text","text":"<command-name>/resolv</command-name>"}]}}`,
		`{"timestamp":"2026-08-04T10:00:00Z","cwd":"/Users/x/alpha","message":{"content":"<command-name>/model</command-name>"}}`,
		`{"timestamp":"2026-08-05T10:00:00Z","cwd":"/Users/x/alpha","message":{"content":[{"type":"tool_use","name":"Bash","input":{}}]}}`,
		`not json at all`,
	}
	body := ""
	for _, l := range lines {
		body += l + "\n"
	}
	p := filepath.Join(t.TempDir(), "s.jsonl")
	writeFile(t, p, body)

	agg := map[string]*Use{}
	n := scanClaudeTranscript(p, agg)
	if n != 3 {
		t.Errorf("counted %d uses, want 3 (two tool calls + one slash command)", n)
	}
	cases := []struct {
		name     string
		want     int
		projects int
	}{
		{"resolv", 2, 2},   // tool call in alpha + slash command in alpha... one project
		{"ponytail", 1, 1}, // "plugin:skill" is recorded under the bare name
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u := agg[tc.name]
			if u == nil {
				t.Fatalf("%s not recorded; got %v", tc.name, keys(agg))
			}
			if u.Count != tc.want {
				t.Errorf("count = %d, want %d", u.Count, tc.want)
			}
		})
	}
	if agg["model"] != nil {
		t.Error("/model is a terminal command, not a skill")
	}
	if agg["Bash"] != nil {
		t.Error("non-Skill tool calls must not be counted")
	}
	if got := agg["resolv"].Last.Format("2006-01-02"); got != "2026-08-03" {
		t.Errorf("Last = %s, want the most recent use", got)
	}
	if len(agg["resolv"].Projects) != 1 {
		t.Errorf("projects = %v, want alpha once, not twice", agg["resolv"].Projects)
	}
}

func TestSuggestionsAreWhatYouUseAndDoNotHave(t *testing.T) {
	uses := []Use{
		{Name: "resolv", Count: 23, InLibrary: false},
		{Name: "write-adr", Count: 4, InLibrary: true},
	}
	got := Suggestions(uses)
	if len(got) != 1 || got[0].Name != "resolv" {
		t.Fatalf("got %+v, want just resolv", got)
	}
}

// History must report the sources it cannot read, not quietly omit them —
// "never used" and "cannot read" look identical otherwise.
func TestHistoryReportsUnreadableSources(t *testing.T) {
	_, sources := History("")
	ids := map[string]HistorySource{}
	for _, s := range sources {
		ids[s.ID] = s
	}
	for _, want := range []string{"claude", "cursor", "codex", "opencode"} {
		if _, ok := ids[want]; !ok {
			t.Errorf("source %q missing from the report", want)
		}
	}
	if ids["cursor"].Note == "" {
		t.Error("cursor is not read; the report must say why")
	}
	if ids["claude"].Note != "" {
		t.Error("claude is read; it should report counts, not a note")
	}
}

func keys(m map[string]*Use) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}
