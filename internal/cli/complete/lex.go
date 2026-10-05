package complete

import "strings"

// SplitLine splits line[:point] the way a POSIX shell splits words, with
// single quotes, double quotes and backslash escapes, and no expansion. An
// open quote is tolerated: its token runs to the end. words are the
// complete tokens; cur is the token at the cursor, "" after a space.
// point is a byte offset; one outside the line means its end.
func SplitLine(line string, point int) (words []string, cur string) {
	if point < 0 || point > len(line) {
		point = len(line)
	}
	s := line[:point]
	var (
		b       strings.Builder
		inToken bool
		quote   byte
	)
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote == '\'':
			if c == '\'' {
				quote = 0
				continue
			}
			b.WriteByte(c)
		case quote == '"':
			switch {
			case c == '"':
				quote = 0
			case c == '\\' && i+1 < len(s) && strings.IndexByte("\"\\$`\n", s[i+1]) >= 0:
				i++
				if s[i] != '\n' {
					b.WriteByte(s[i])
				}
			default:
				b.WriteByte(c)
			}
		case c == '\'' || c == '"':
			quote, inToken = c, true
		case c == '\\':
			inToken = true
			if i+1 < len(s) {
				i++
				if s[i] != '\n' {
					b.WriteByte(s[i])
				}
			}
		case c == ' ' || c == '\t' || c == '\n':
			if inToken {
				words = append(words, b.String())
				b.Reset()
				inToken = false
			}
		default:
			inToken = true
			b.WriteByte(c)
		}
	}
	if inToken {
		return words, b.String()
	}
	return words, ""
}
