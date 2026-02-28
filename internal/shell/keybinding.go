package shell

import (
	"fmt"
	"os"
	"strings"
)

// Keybinding represents a shell key combination like ctrl+g.
type Keybinding struct {
	Modifier string
	Letter   string
}

var (
	DefaultKey     = Keybinding{Modifier: "ctrl", Letter: "g"}
	WarpDefaultKey = Keybinding{Modifier: "ctrl", Letter: "o"}
)

// ParseKey parses values like ctrl+g or alt+f (case-insensitive).
func ParseKey(input string) (Keybinding, error) {
	raw := strings.ToLower(strings.TrimSpace(input))
	parts := strings.Split(raw, "+")
	if len(parts) != 2 {
		return Keybinding{}, fmt.Errorf("invalid key format %q: expected modifier+letter, e.g. ctrl+g", input)
	}

	modifier := strings.TrimSpace(parts[0])
	letter := strings.TrimSpace(parts[1])

	if modifier != "ctrl" && modifier != "alt" {
		return Keybinding{}, fmt.Errorf("unsupported modifier %q: use ctrl or alt", modifier)
	}
	if len(letter) != 1 || letter[0] < 'a' || letter[0] > 'z' {
		return Keybinding{}, fmt.Errorf("invalid key %q: expected a single letter a-z", letter)
	}

	return Keybinding{Modifier: modifier, Letter: letter}, nil
}

// Validate blocks keybindings that collide with essential terminal functions.
func (k Keybinding) Validate() error {
	blockedCtrl := map[string]string{
		"c": "SIGINT (interrupt process)",
		"d": "EOF (close shell)",
		"z": "SIGTSTP (suspend process)",
		"s": "XOFF (freeze terminal output)",
		"q": "XON (resume terminal output)",
	}

	if k.Modifier == "ctrl" {
		if reason, blocked := blockedCtrl[k.Letter]; blocked {
			return fmt.Errorf("key %q conflicts with essential terminal function: %s", k.raw(), reason)
		}
	}
	return nil
}

func (k Keybinding) String() string {
	if k.Modifier == "alt" {
		return fmt.Sprintf("Alt+%s", strings.ToUpper(k.Letter))
	}
	return fmt.Sprintf("Ctrl+%s", strings.ToUpper(k.Letter))
}

func (k Keybinding) ForZsh() string {
	if k.Modifier == "alt" {
		return "\\e" + k.Letter
	}
	return "\\C-" + k.Letter
}

func (k Keybinding) ForBash() string {
	if k.Modifier == "alt" {
		return "\\e" + k.Letter
	}
	return "\\C-" + k.Letter
}

func (k Keybinding) ForFish() string {
	if k.Modifier == "alt" {
		return "\\e" + k.Letter
	}
	return "\\c" + k.Letter
}

func (k Keybinding) ForPowerShell() string {
	if k.Modifier == "alt" {
		return "Alt+" + strings.ToUpper(k.Letter)
	}
	return "Ctrl+" + strings.ToUpper(k.Letter)
}

// TemplateData is passed to shell script templates.
type TemplateData struct {
	Key     string
	Comment string
}

func DetectWarp() bool {
	return os.Getenv("TERM_PROGRAM") == "WarpTerminal"
}

func (k Keybinding) raw() string {
	return k.Modifier + "+" + k.Letter
}
