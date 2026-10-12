// Package config is gobble's settings: settings.yaml, read strictly with
// go.yaml.in/yaml/v3, layered as defaults, the global file, a trusted
// project's .gobble/settings.yaml and --set overrides, validated against one
// key table, and explained by layer and line (0018-MADR).
// Stability: internal
//
// Keys is the table every part reads: the reader, the merge, the schema,
// the embedded template and its tests. A project's file is read and
// reported always, but applied only when the caller says the project is
// trusted, which until 0005-PLAN F6 is never. Credentials never belong
// here; they live in auth.json. internal/cli binds flags and their
// environment variables on top of what Resolve returns, and is the only
// package that imports Kong.
package config
