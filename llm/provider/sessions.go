package provider

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// Sessions holds one OAuth session per provider for a run, so the run's
// provider, its listings and its model lookups share a provider's session,
// and a refresh one of them starts is one the run can wait for (0046-MADR
// D8). Without it, each call makes its own session, which its caller drops.
// The zero value is ready to use, and it is safe for concurrent use.
type Sessions struct {
	mu sync.Mutex
	m  map[string]*auth.OAuthSession
}

// get is provider id's session, made by load on first use. A load that
// fails, or finds no login, holds nothing.
func (s *Sessions) get(id string, load func() (*auth.OAuthSession, error)) (*auth.OAuthSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess, ok := s.m[id]; ok {
		return sess, nil
	}
	sess, err := load()
	if err != nil || sess == nil {
		return sess, err
	}
	if s.m == nil {
		s.m = map[string]*auth.OAuthSession{}
	}
	s.m[id] = sess
	return sess, nil
}

// Drop forgets provider id's session, so the next use loads the stored
// login again: after a sign-in or a sign-out (0046-MADR D8). A refresh the
// dropped session is running goes on, and is not waited for.
func (s *Sessions) Drop(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, id)
}

// Held are the providers whose session s holds, sorted.
func (s *Sessions) Held() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Sorted(maps.Keys(s.m))
}

// Wait waits for every held session's refresh in flight, and for a retried
// save of a rotated session, so a run that ends keeps its rotations
// (go-llmprovider-sdk auth.OAuthSession.Wait; 0046-MADR D8). It cancels
// nothing. It returns nil, or, if ctx ends first, an error naming each
// provider still refreshing and matching ctx's error.
func (s *Sessions) Wait(ctx context.Context) error {
	s.mu.Lock()
	held := maps.Clone(s.m)
	s.mu.Unlock()
	var errs []error
	for _, id := range slices.Sorted(maps.Keys(held)) {
		if err := held[id].Wait(ctx); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", id, err))
		}
	}
	return errors.Join(errs...)
}
