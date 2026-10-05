package editor

import "strings"

// coalescer gathers the lines of one submission. With bracketed paste on,
// x/term returns each pasted line separately, marked by ErrPasteIndicator;
// a typed line ending in a backslash continues. The next line ended by a
// plain Enter completes the submission.
type coalescer struct {
	parts  []string
	pasted int
}

func (c *coalescer) open() bool { return len(c.parts) > 0 }

func (c *coalescer) addPasted(line string) {
	c.parts = append(c.parts, line)
	c.pasted++
}

// addTyped adds a line ended by a plain Enter and reports whether the
// submission is complete.
func (c *coalescer) addTyped(line string) bool {
	if before, ok := strings.CutSuffix(line, `\`); ok {
		c.parts = append(c.parts, before)
		return false
	}
	c.parts = append(c.parts, line)
	return true
}

// take returns the submission and how many of its lines were pasted, and
// resets c.
func (c *coalescer) take() (text string, pasted int) {
	text, pasted = strings.Join(c.parts, "\n"), c.pasted
	*c = coalescer{}
	return text, pasted
}
