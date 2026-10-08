package fsx

import (
	"path/filepath"
	"sync"
)

// The mutation queue: one mutex per real path, process-wide, so writes to
// one file are serialised and writes to different files are not (Pi
// file-mutation-queue.ts; 0005-MADR "Mutation queue"). A session's calls
// run one after another, but one process serves many sessions.
var (
	locksMu sync.Mutex
	locks   = map[string]*pathLock{}
)

type pathLock struct {
	mu   sync.Mutex
	refs int
}

// Lock waits until no other caller holds abs, then holds it until unlock is
// called. abs is keyed by its real path, or by itself when it does not
// exist yet.
func Lock(abs string) (unlock func()) {
	key := lockKey(abs)
	locksMu.Lock()
	l := locks[key]
	if l == nil {
		l = &pathLock{}
		locks[key] = l
	}
	l.refs++
	locksMu.Unlock()

	l.mu.Lock()
	var once sync.Once
	return func() {
		once.Do(func() {
			l.mu.Unlock()
			locksMu.Lock()
			l.refs--
			if l.refs == 0 {
				delete(locks, key)
			}
			locksMu.Unlock()
		})
	}
}

func lockKey(abs string) string {
	if r, err := filepath.EvalSymlinks(abs); err == nil {
		abs = r
	} else if a, err := filepath.Abs(abs); err == nil {
		abs = a
	}
	return foldKey(filepath.Clean(abs))
}
