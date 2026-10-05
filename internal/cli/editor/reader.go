package editor

import (
	"context"
	"io"
	"sync"
)

// chunkSize is the size of one read from stdin.
const chunkSize = 512

type chunk struct {
	b   []byte
	err error
}

// Reader owns stdin. Start launches the one goroutine that reads it, in
// chunkSize pieces, for the life of the process; Read and ReadContext serve
// those chunks to the editor. The underlying reader is never closed (F20).
type Reader struct {
	wg      sync.WaitGroup
	ctx     context.Context
	chunks  chan chunk
	pending []byte
	err     error
}

// Start begins reading in. It must be called once, before any Read. The
// goroutine stops when in returns an error, or when ctx is done and a read
// has completed; a read that is blocked stays blocked, which is why in is
// never closed to stop it.
func (r *Reader) Start(ctx context.Context, in io.Reader) {
	r.ctx = ctx
	r.chunks = make(chan chunk, 16)
	r.wg.Go(func() {
		for {
			buf := make([]byte, chunkSize)
			n, err := in.Read(buf)
			if n > 0 && !r.send(ctx, chunk{b: buf[:n]}) {
				return
			}
			if err != nil {
				r.send(ctx, chunk{err: err})
				return
			}
		}
	})
}

func (r *Reader) send(ctx context.Context, c chunk) bool {
	select {
	case r.chunks <- c:
		return true
	case <-ctx.Done():
		return false
	}
}

// Read implements io.Reader with the context given to Start.
func (r *Reader) Read(p []byte) (int, error) {
	return r.ReadContext(r.ctx, p)
}

// ReadContext reads buffered input, waiting for the next chunk until ctx is
// done. An error from stdin is returned once the data before it is consumed,
// and on every call after that.
func (r *Reader) ReadContext(ctx context.Context, p []byte) (int, error) {
	if len(r.pending) == 0 {
		if r.err != nil {
			return 0, r.err
		}
		select {
		case c := <-r.chunks:
			r.pending, r.err = c.b, c.err
		case <-ctx.Done():
			return 0, ctx.Err()
		}
		if len(r.pending) == 0 {
			return 0, r.err
		}
	}
	n := copy(p, r.pending)
	r.pending = r.pending[n:]
	return n, nil
}
