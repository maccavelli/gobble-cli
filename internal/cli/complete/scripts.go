package complete

import (
	"bytes"
	"embed"
	"fmt"
	"slices"
	"text/template"
)

//go:embed scripts/*.tmpl
var scriptFS embed.FS

var scripts = template.Must(template.ParseFS(scriptFS, "scripts/*.tmpl"))

// Shells are the shells Script supports.
var Shells = []string{"bash", "zsh", "fish", shellPowerShell}

// Script renders the registration script for shell, for the program name.
func Script(shell, name string) (string, error) {
	if !slices.Contains(Shells, shell) {
		return "", fmt.Errorf("complete: no script for shell %q", shell)
	}
	var b bytes.Buffer
	if err := scripts.ExecuteTemplate(&b, shell+".tmpl", struct{ Name, Protocol string }{name, Protocol}); err != nil {
		return "", fmt.Errorf("complete: render %s script: %w", shell, err)
	}
	return b.String(), nil
}
