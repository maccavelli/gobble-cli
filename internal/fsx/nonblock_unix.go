//go:build unix

package fsx

import "syscall"

// nonblock is the flag a file is opened for reading with: a FIFO's open
// returns at once instead of waiting for a writer, and the type check then
// refuses it.
const nonblock = syscall.O_NONBLOCK
