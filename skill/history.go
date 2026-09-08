package skill

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Use is one skill, aggregated across every session on this machine.
type Use struct {
	Name      string    `json:"name"`
	Count     int       `json:"count"`
	Last      time.Time `json:"last"`
	Projects  []string  `json:"projects"`
	Source    string    `json:"source"` // which agent's log it came from
	InLibrary bool      `json:"in_library"`
}

// HistorySource describes one place usage was read from, including the ones
// that had nothing to give. Reporting the empty ones matters: "you have never
// used a skill in Cursor" and "we cannot read Cursor's log" look identical in
// a table that only lists what it found.
type HistorySource struct {
	ID    string
	Path  string
	Files int
	Uses  int
	Note  string
}

var slashCmd = regexp.MustCompile(`<command-name>/?([\w:.\-]+)</command-name>`)

// builtins are terminal commands, not skills. Counting them would bury the
// real signal under /model and /goal.
var builtins = map[string]bool{
	"model": true, "goal": true, "clear": true, "help": true, "compact": true,
	"config": true, "cost": true, "login": true, "logout": true, "resume": true,
	"init": true, "status": true, "doctor": true, "permissions": true,
	"remote-control": true, "tasks": true, "artifacts": true, "vim": true,
}

// History aggregates skill usage from every agent log we can actually read.
// Only Claude Code keeps a machine-readable one: Cursor stores chats in an
// undocumented SQLite schema, and Codex and opencode keep no skill log at all.
// Those are reported as sources with a note rather than silently omitted.
func History(root string) ([]Use, []HistorySource) {
	home, _ := os.UserHomeDir()
	agg := map[string]*Use{}

	claudeDir := filepath.Join(home, ".claude", "projects")
	files, _ := filepath.Glob(filepath.Join(claudeDir, "*", "*.jsonl"))
	before := 0
	for _, f := range files {
		before += scanClaudeTranscript(f, agg)
	}
	sources := []HistorySource{
		{ID: "claude", Path: claudeDir, Files: len(files), Uses: before},
		{ID: "cursor", Path: filepath.Join(home, ".cursor", "chats"),
			Note: "chats are an undocumented SQLite schema — not read"},
		{ID: "codex", Path: filepath.Join(home, ".codex"), Note: "keeps no skill log"},
		{ID: "opencode", Path: filepath.Join(home, ".config", "opencode"), Note: "keeps no skill log"},
	}

	have := map[string]bool{}
	if root != "" {
		skills, _ := LoadAll(root)
		for _, s := range skills {
			have[s.Name] = true
		}
	}

	out := make([]Use, 0, len(agg))
	for _, u := range agg {
		sort.Strings(u.Projects)
		u.InLibrary = have[u.Name]
		out = append(out, *u)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Name < out[j].Name
	})
	return out, sources
}

type transcriptLine struct {
	Timestamp string `json:"timestamp"`
	Cwd       string `json:"cwd"`
	Message   struct {
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

type contentBlock struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Text  string `json:"text"`
	Input struct {
		Skill string `json:"skill"`
	} `json:"input"`
}

// scanClaudeTranscript counts both ways a skill gets used: the Skill tool, and
// the user typing /name. They are the same event from the library's point of
// view, so they are counted together.
func scanClaudeTranscript(path string, agg map[string]*Use) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	n := 0
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 256*1024), 16*1024*1024)
	for sc.Scan() {
		raw := sc.Bytes()
		if !strings.Contains(string(raw), "Skill") && !strings.Contains(string(raw), "command-name") {
			continue
		}
		var l transcriptLine
		if err := json.Unmarshal(raw, &l); err != nil {
			continue
		}
		ts, _ := time.Parse(time.RFC3339, l.Timestamp)

		var blocks []contentBlock
		if json.Unmarshal(l.Message.Content, &blocks) == nil {
			for _, b := range blocks {
				if b.Type == "tool_use" && b.Name == "Skill" && b.Input.Skill != "" {
					record(agg, b.Input.Skill, ts, l.Cwd, "claude")
					n++
				}
				if b.Type == "text" {
					n += recordSlash(agg, b.Text, ts, l.Cwd)
				}
			}
			continue
		}
		var text string
		if json.Unmarshal(l.Message.Content, &text) == nil {
			n += recordSlash(agg, text, ts, l.Cwd)
		}
	}
	return n
}

func recordSlash(agg map[string]*Use, text string, ts time.Time, cwd string) int {
	n := 0
	for _, m := range slashCmd.FindAllStringSubmatch(text, -1) {
		if builtins[m[1]] {
			continue
		}
		record(agg, m[1], ts, cwd, "claude")
		n++
	}
	return n
}

func record(agg map[string]*Use, name string, ts time.Time, cwd, source string) {
	// Plugin skills arrive as "plugin:skill"; the library knows the bare name.
	if i := strings.LastIndex(name, ":"); i >= 0 {
		name = name[i+1:]
	}
	u := agg[name]
	if u == nil {
		u = &Use{Name: name, Source: source}
		agg[name] = u
	}
	u.Count++
	if ts.After(u.Last) {
		u.Last = ts
	}
	if cwd != "" {
		p := filepath.Base(cwd)
		for _, e := range u.Projects {
			if e == p {
				return
			}
		}
		u.Projects = append(u.Projects, p)
	}
}

// Suggestions are the skills used often enough to be worth having, that the
// library does not contain. That is the whole point of reading the history.
func Suggestions(uses []Use) []Use {
	var out []Use
	for _, u := range uses {
		if !u.InLibrary {
			out = append(out, u)
		}
	}
	return out
}
