// Package fsx confines file tools to a workspace: the session's working
// directory and its additional directories. It resolves the paths a model
// gives as Pi does, opens paths inside the workspace through os.Root, writes
// files atomically, and serialises writes to one real path.
// Stability: internal
package fsx
