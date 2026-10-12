package config

import _ "embed" // for the //go:embed below

//go:embed settings.yaml
var template []byte

// Template is the commented settings.yaml gobble creates when none exists
// (0018-MADR D9): every key, commented out, at its default, with what it
// does. A copy is returned, so a caller cannot change the embedded one.
func Template() []byte { return append([]byte(nil), template...) }
