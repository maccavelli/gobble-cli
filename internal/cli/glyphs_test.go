package cli

import (
	"testing"
	"unicode/utf8"

	"github.com/maccavelli/gobble-cli/internal/cli/term"
)

// The glyphs the CLI draws, from go-tui-lib's glyph sets: each set, chosen by
// the terminal's Unicode, has every one, and ASCII's are ASCII (0035-MADR D14
// step 2).
func TestGlyphSets(t *testing.T) {
	for _, unicodeOK := range []bool{true, false} {
		g := glyphsFor(term.Caps{Unicode: unicodeOK})
		for name, s := range map[string]string{"Bullet": g.Bullet, "Heading": g.Heading, "Done": g.Done, "Failed": g.Failed,
			"Ellipsis": g.Ellipsis, "MeterFill": g.MeterFill, "MeterTrack": g.MeterTrack} {
			if s == "" {
				t.Errorf("Unicode %v: %s is empty", unicodeOK, name)
			}
			for _, r := range s {
				if !unicodeOK && r >= utf8.RuneSelf {
					t.Errorf("ASCII %s = %q is not ASCII", name, s)
				}
			}
		}
	}
}
