package cli

// Mode is how the default command runs (0008-MADR D1).
type Mode int

const (
	// ModeLine is the interactive line session.
	ModeLine Mode = iota
	// ModePrint runs one turn and prints its result.
	ModePrint
)

func (m Mode) String() string {
	if m == ModePrint {
		return "print"
	}
	return "line"
}

// resolveMode is Pi's rule: -p, or a stdin or stdout that is not a
// terminal, means print mode; otherwise the line session. The TUI is never
// chosen here (0005-MADR).
func resolveMode(printFlag, stdinTTY, stdoutTTY bool) Mode {
	if printFlag || !stdinTTY || !stdoutTTY {
		return ModePrint
	}
	return ModeLine
}
