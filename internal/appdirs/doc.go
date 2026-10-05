// Package appdirs resolves gobble's four directory roles (config, data,
// state, cache) from the 0003-MADR table, and creates owner-only directories.
// Stability: internal
//
// Resolution is pure: Resolve takes the environment, the target OS and the
// platform base directories, so one test covers every layout on any host.
// System supplies the real ones. The owner-only helpers are copied from
// magic-cli-remote internal/appdirs (Apache-2.0); each copied file names its
// source.
package appdirs
