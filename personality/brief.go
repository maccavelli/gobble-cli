package personality

import (
	"errors"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Brief is a personality's character brief, personality.md (0022-MADR D4):
// the front matter code needs, and the prose the model reads as written.
type Brief struct {
	Name    string `yaml:"name"`
	Level   int    `yaml:"level"`
	Summary string `yaml:"summary"`
	// Body is the brief after its front matter, trimmed.
	Body string `yaml:"-"`
}

// ParseBrief reads a personality.md: front matter with a name, a level
// from 1 to 5 and a summary, then the brief.
func ParseBrief(data []byte) (Brief, error) {
	s := strings.ReplaceAll(string(data), "\r\n", "\n")
	if !strings.HasPrefix(s, "---\n") {
		return Brief{}, errors.New("personality: the brief has no front matter")
	}
	end := strings.Index(s[4:], "\n---\n")
	if end < 0 {
		return Brief{}, errors.New("personality: the front matter does not close with ---")
	}
	var b Brief
	if err := yaml.Unmarshal([]byte(s[4:4+end]), &b); err != nil {
		return Brief{}, fmt.Errorf("personality: front matter: %w", err)
	}
	switch {
	case b.Name == "":
		return Brief{}, errors.New("personality: the brief has no name")
	case b.Level < 1 || b.Level > 5:
		return Brief{}, fmt.Errorf("personality: %s: level %d is not 1 to 5", b.Name, b.Level)
	}
	b.Body = strings.TrimSpace(s[4+end+5:])
	if b.Body == "" {
		return Brief{}, fmt.Errorf("personality: %s: the brief is empty", b.Name)
	}
	return b, nil
}
