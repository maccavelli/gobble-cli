// Package proctree runs a command as a tree that can be stopped whole: its
// own process group on Unix, a kill-on-close Job object on Windows. mcpclient
// stops stdio servers with it, and the shell tools stop commands with it.
// Stability: internal
package proctree
