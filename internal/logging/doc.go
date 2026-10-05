// Package logging builds gobble's one slog logger: JSON to a rolling,
// owner-only file in the state directory, text to stderr at warning and
// above, both redacted, fanned out with slog.NewMultiHandler. It never
// installs a global logger; callers pass the *slog.Logger it returns
// (0008-MADR D18). Under `gobble acp` the caller passes no stderr, because
// stdio is the protocol.
// Stability: internal
package logging
