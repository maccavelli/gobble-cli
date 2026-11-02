//go:build !unix

package fsx

// nonblock is 0: off Unix, opening a file does not wait for a FIFO's
// writer.
const nonblock = 0
