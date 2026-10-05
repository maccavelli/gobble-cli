package main

import (
	"context"
	"os"

	"github.com/maccavelli/gobble-cli/internal/cli"
	"github.com/maccavelli/gobble-cli/internal/cli/term"
)

// main is the only place that names the process's streams and exits
// (0008-PLAN C3). Signal handling lives in cli.Main.
func main() {
	restore, err := term.PrepareConsole(os.Stdout, os.Stderr)
	code := cli.Main(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	if err != nil && code == cli.ExitOK {
		code = cli.ExitFailure
	}
	restore()
	os.Exit(code)
}
