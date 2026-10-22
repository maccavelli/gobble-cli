package editor

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"
)

// chunkSize is the size of one read from stdin.
const chunkSize = 512

// errResized and errWaitOver end a ReadUntil with no bytes: the terminal
// was resized, or the wait for its cursor's place is over.
var (
	errResized  = errors.New("editor: terminal resized")
	errWaitOver = errors.New("editor: no reply from the terminal")
)

type chunk struct {
	b   []byte
	err error
}

// Reader owns stdin. Start launches the one goroutine that reads it, in
// chunkSize pieces, for the life of the process; Read and ReadContext serve
// those chunks to the editor. The underlying reader is never closed (F20).
//
// It reads only when asked: one read of stdin for each ReadContext that
// finds nothing buffered, none ahead. So a read starts in the terminal mode
// of whoever asked, raw for the editor and cooked for a question; on
// Windows a console read keeps the mode it started in (0027-MADR D5, as
// amended for P5).
type Reader struct {
	wg      sync.WaitGroup
	ctx     context.Context
	asks    chan struct{}
	chunks  chan chunk
	asked   bool // a read was asked for and has not been served
	pending []byte
	err     error
}

// Start makes in the reader's stdin. It must be called once, before any
// Read; it reads nothing until asked. The goroutine stops when in returns
// an error, or when ctx is done and no read is blocked; a read that is
// blocked stays blocked, which is why in is never closed to stop it.
func (r *Reader) Start(ctx context.Context, in io.Reader) {
	r.ctx = ctx
	r.asks = make(chan struct{}, 1)
	r.chunks = make(chan chunk, 1)
	r.wg.Go(func() {
		for {
			select {
			case <-r.asks:
			case <-ctx.Done():
				return
			}
			buf := make([]byte, chunkSize)
			n, err := in.Read(buf)
			for n == 0 && err == nil {
				n, err = in.Read(buf)
			}
			if !r.send(ctx, chunk{b: buf[:n], err: err}) || err != nil {
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

// ReadUntil is ReadContext that also stops waiting, with no bytes, when
// resize or timeout delivers: errResized or errWaitOver. A read asked for
// stays asked, and is served to the next call. A nil channel never
// delivers.
func (r *Reader) ReadUntil(ctx context.Context, p []byte, resize <-chan struct{}, timeout <-chan time.Time) (int, error) {
	if len(r.pending) == 0 && r.err == nil {
		if !r.asked {
			r.asks <- struct{}{}
			r.asked = true
		}
		select {
		case c := <-r.chunks:
			r.pending, r.err, r.asked = c.b, c.err, false
		case <-resize:
			return 0, errResized
		case <-timeout:
			return 0, errWaitOver
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}
	return r.ReadContext(ctx, p)
}

// ReadContext reads buffered input; with none, it asks for one read of
// stdin and waits for it until ctx is done. A read asked for and not
// served by then is served to the next call. An error from stdin is
// returned once the data before it is consumed, and on every call after
// that.
func (r *Reader) ReadContext(ctx context.Context, p []byte) (int, error) {
	if len(r.pending) == 0 {
		if r.err != nil {
			return 0, r.err
		}
		if !r.asked {
			r.asks <- struct{}{}
			r.asked = true
		}
		select {
		case c := <-r.chunks:
			r.pending, r.err, r.asked = c.b, c.err, false
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
