package command

import (
	"regexp"
	"strings"
	"unicode"
)

// Command is one slash command as an ACP client lists it.
type Command struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	// Hint names the arguments, such as "[instructions]"; empty for none.
	Hint string `json:"hint,omitzero"`
}

var validName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

// ValidName reports whether name can be a command's name.
func ValidName(name string) bool { return validName.MatchString(name) }

// Parse reads a prompt as a command: the text, trimmed, is "/" and a valid
// name, then optional whitespace and the arguments. ok is false for any
// other text, which is a prompt for the model.
func Parse(text string) (name, args string, ok bool) {
	text = strings.TrimSpace(text)
	rest, slash := strings.CutPrefix(text, "/")
	if !slash {
		return "", "", false
	}
	end := strings.IndexFunc(rest, unicode.IsSpace)
	if end < 0 {
		end = len(rest)
	}
	name = rest[:end]
	if !ValidName(name) {
		return "", "", false
	}
	return name, strings.TrimSpace(rest[end:]), true
}
