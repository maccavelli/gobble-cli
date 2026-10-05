package term

import "strings"

// Style is a set of SGR attributes: weights, decorations and one of the eight
// foreground colours.
type Style uint16

// Attributes and foreground colours. The colours are in SGR order, 30–37.
const (
	Bold Style = 1 << iota
	Dim
	Italic
	Underline
	Black
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
)

var attrCodes = [...]struct {
	bit  Style
	code string
}{
	{Bold, "1"}, {Dim, "2"}, {Italic, "3"}, {Underline, "4"},
	{Black, "30"}, {Red, "31"}, {Green, "32"}, {Yellow, "33"},
	{Blue, "34"}, {Magenta, "35"}, {Cyan, "36"}, {White, "37"},
}

// Wrap returns text inside the style's SGR sequence and a reset. When on is
// false, or the style is empty, it returns text unchanged.
func (s Style) Wrap(text string, on bool) string {
	if !on || s == 0 || text == "" {
		return text
	}
	codes := make([]string, 0, 4)
	for _, a := range attrCodes {
		if s&a.bit != 0 {
			codes = append(codes, a.code)
		}
	}
	return "\x1b[" + strings.Join(codes, ";") + "m" + text + "\x1b[0m"
}

// Glyphs are the symbols the renderer draws.
type Glyphs struct {
	Bullet, Arrow, Check, Cross, Ellipsis, Bar, BarEmpty string
}

// Unicode and ASCII are the two glyph sets. Caps.Unicode chooses between them.
var (
	Unicode = Glyphs{Bullet: "•", Arrow: "▸", Check: "✓", Cross: "✗", Ellipsis: "…", Bar: "━", BarEmpty: "╌"}
	ASCII   = Glyphs{Bullet: "*", Arrow: ">", Check: "+", Cross: "x", Ellipsis: "...", Bar: "#", BarEmpty: "-"}
)
