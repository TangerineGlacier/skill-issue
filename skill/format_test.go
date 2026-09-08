package skill

import "testing"

func TestFormat(t *testing.T) {
	in := "---\nname: x\ntags: [a]\n---\n\n\n\n# Title\ntext   \r\n\n\n\n* one\n  + two\n\n```go\nfmt.Println(\"a  \")\n\n\n*  keep\n```\n## Next\n"
	want := "---\nname: x\ntags: [a]\n---\n\n# Title\n\ntext\n\n- one\n  - two\n\n```go\nfmt.Println(\"a  \")\n\n\n*  keep\n```\n\n## Next\n"
	got := Format(in)
	if got != want {
		t.Errorf("Format()\n got %q\nwant %q", got, want)
	}
	if again := Format(got); again != got {
		t.Errorf("Format is not idempotent:\n got %q\nwant %q", again, got)
	}
	if body := Format("no frontmatter\n\n\nhere"); body != "no frontmatter\n\nhere\n" {
		t.Errorf("plain markdown: got %q", body)
	}
}
