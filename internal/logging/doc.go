// Package logging builds gobble's one slog logger: JSON to a rolling,
// owner-only file in the state directory, and stderr at warning and above,
// as gobble's own "Warning:" or "Error:" diagnostic when stderr is a
// terminal and as slog's text otherwise (0024-MADR D6), both redacted,
// fanned out with slog.NewMultiHandler. It never
// installs a global logger; callers pass the *slog.Logger it returns
// (0008-MADR D18). Under `gobble acp` the caller passes no stderr, because
// stdio is the protocol.
// Stability: internal
package logging
