// Package buildinfo reports gobble's build identity: the SemVer version,
// commit, commit time, toolchain and platform, whether the build is a
// release, and the HTTP User-Agent.
// Stability: internal
//
// The stamps and the release rule belong to go-selfupdate-lib's buildinfo,
// which this package wraps, so the version gobble reports and the one
// self-update decides on cannot disagree (0004-MADR amendment of
// 2026-10-05). gobble links no stamps of its own.
package buildinfo
