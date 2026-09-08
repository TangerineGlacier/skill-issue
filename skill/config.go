package skill

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config is the one piece of state that lives outside the repo: where the
// library is, so `skill-issue` works from any directory on the machine.
type Config struct {
	Library string `json:"library"` // absolute path to the skills/ directory
}

// ConfigPath honours XDG_CONFIG_HOME, falling back to ~/.config.
func ConfigPath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "skill-issue", "config.json")
}

func LoadConfig() Config {
	var c Config
	b, err := os.ReadFile(ConfigPath())
	if err != nil {
		return c
	}
	_ = json.Unmarshal(b, &c)
	return c
}

func SaveConfig(c Config) error {
	p := ConfigPath()
	if p == "" {
		return fmt.Errorf("cannot determine a config directory")
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, append(b, '\n'), 0o644)
}

// ResolveRoot finds the library, in the order that respects the most specific
// intent first:
//
//	$SKILL_ISSUE_HOME   an explicit override, for scripts and CI
//	./skills upwards    you are standing inside a library
//	the config file     the library you registered with `skill-issue link`
//
// Walking up beats the config on purpose: inside a checkout, that checkout is
// what you meant, whatever is registered globally.
func ResolveRoot(dir string) (string, error) {
	if env := os.Getenv("SKILL_ISSUE_HOME"); env != "" {
		root := env
		if filepath.Base(root) != "skills" {
			root = filepath.Join(root, "skills")
		}
		if fi, err := os.Stat(root); err == nil && fi.IsDir() {
			return root, nil
		}
		return "", fmt.Errorf("SKILL_ISSUE_HOME=%s has no skills/ directory", env)
	}
	if root, err := Root(dir); err == nil {
		return root, nil
	}
	if lib := LoadConfig().Library; lib != "" {
		if fi, err := os.Stat(lib); err == nil && fi.IsDir() {
			return lib, nil
		}
		return "", fmt.Errorf("the linked library %s is gone — re-run `skill-issue link` inside it", lib)
	}
	return "", fmt.Errorf("no library found: run `skill-issue link` inside your checkout, " +
		"or set SKILL_ISSUE_HOME")
}
