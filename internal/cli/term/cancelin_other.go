//go:build !unix

package term

import "os"

// Gated is a POSIX terminal's cancellable reader; there is none here
// (0063-MADR D22, rows R139, R143).
type Gated struct{}

// OpenGated reports ErrNoGate: a Windows console keeps D19's rule, any
// other system reads standard input as built.
func OpenGated(*os.File, GatedOptions) (*Gated, error) { return nil, ErrNoGate }

// Read, Open, Close and Stop are never called: OpenGated gives no reader.
func (*Gated) Read([]byte) (int, error) { return 0, ErrNoGate }

// Open does nothing here.
func (*Gated) Open() {}

// Close reports a read, as a reader with no gate does (row R151).
func (*Gated) Close() bool { return true }

// Stop does nothing here.
func (*Gated) Stop() {}
