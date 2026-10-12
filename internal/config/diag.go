package config

import "strconv"

// Severity is a diagnostic's weight. An error means a value or a file was
// not used; a warning means something was ignored that may be meant.
type Severity int

// The severities.
const (
	SeverityError Severity = iota
	SeverityWarning
)

// Diagnostic is one finding about a settings file, in 0018-MADR D7's
// forms. It names a key's path and line, never its value (D7, D15).
type Diagnostic struct {
	File     string
	Line     int // 0 when the finding is about the whole file
	Path     string
	Severity Severity
	Text     string
}

func (d Diagnostic) String() string {
	loc := d.File
	if d.Line > 0 {
		loc += ":" + strconv.Itoa(d.Line)
	}
	if d.Path != "" {
		return loc + ": " + d.Path + ": " + d.Text
	}
	return loc + ": " + d.Text
}

// Unknown reports whether d is an unknown key, which `gobble config validate
// --strict` fails on (0018-MADR D7).
func (d Diagnostic) Unknown() bool { return d.Text == textUnknown }

// D7's texts.
const (
	textSecondDocument = "settings are one YAML document; a second one starts here"
	textNotMapping     = "settings must be a mapping of keys"
	textUnknown        = "not used by this gobble"
	textGlobalOnly     = "only the global settings file can set this"
	textCredential     = "credentials go in auth.json, not settings.yaml"
	textUntrusted      = "project settings apply once the project is trusted (0005-PLAN F6)"
)

func errorAt(file string, line int, path, text string) Diagnostic {
	return Diagnostic{File: file, Line: line, Path: path, Severity: SeverityError, Text: text}
}

func warningAt(file string, line int, path, text string) Diagnostic {
	return Diagnostic{File: file, Line: line, Path: path, Severity: SeverityWarning, Text: text}
}

// HasErrors reports whether any diagnostic is an error.
func HasErrors(ds []Diagnostic) bool {
	for _, d := range ds {
		if d.Severity == SeverityError {
			return true
		}
	}
	return false
}
