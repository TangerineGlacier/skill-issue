package main

import (
	"os/exec"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type errMsg struct{ err error }

// copyCmd puts a string on the system clipboard. Shelling out to the platform
// tool beats a clipboard dependency for one feature.
func copyCmd(s string) tea.Cmd {
	return func() tea.Msg {
		var c *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			c = exec.Command("pbcopy")
		case "windows":
			c = exec.Command("clip")
		default:
			c = exec.Command("xclip", "-selection", "clipboard")
		}
		c.Stdin = strings.NewReader(s)
		if err := c.Run(); err != nil {
			return errMsg{err}
		}
		return nil
	}
}

// openCmd hands a path to the OS, or to $EDITOR when one is set and we are not
// inside the alternate screen.
func openCmd(path string) tea.Cmd {
	return func() tea.Msg {
		opener := "open"
		if runtime.GOOS == "linux" {
			opener = "xdg-open"
		}
		if err := exec.Command(opener, path).Start(); err != nil {
			return errMsg{err}
		}
		return nil
	}
}

type reloadMsg struct{}

// ago renders a timestamp the way a human reads one.
func ago(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return plural(int(d.Minutes()), "min") + " ago"
	case d < 24*time.Hour:
		return plural(int(d.Hours()), "hour") + " ago"
	case d < 30*24*time.Hour:
		return plural(int(d.Hours()/24), "day") + " ago"
	}
	return t.Format("2 Jan 2006")
}

func plural(n int, unit string) string {
	s := ""
	if n != 1 {
		s = "s"
	}
	return itoa(n) + " " + unit + s
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
