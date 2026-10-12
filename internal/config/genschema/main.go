// Command genschema writes internal/config's JSON Schema to
// internal/config/testdata/settings.schema.json, or to the path given. Run
// it from the repository root, after changing a key:
//
//	go run ./internal/config/genschema
package main

import (
	"fmt"
	"os"

	"github.com/maccavelli/gobble-cli/internal/config"
)

func main() {
	out := "internal/config/testdata/settings.schema.json"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	b, err := config.Schema()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(out, b, 0o644); err != nil { //nolint:gosec // G306: a committed schema, world-readable like every source file
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
