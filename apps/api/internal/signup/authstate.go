package signup

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// AuthState holds the state/nonce/PKCE verifier bound to a single authorize
// request. All three fields travel together so that SIGNUP-02 (state),
// SIGNUP-03 (nonce), and SIGNUP-04 (verifier) share one record with one
// lifetime.
//
// Verifier MUST NOT leave apps/api; only the Challenge is placed on the wire.
type AuthState struct {
	State    string
	Nonce    string
	Verifier string // PKCE code_verifier; SIGNUP-04
}

type stateRecord struct {
	st        AuthState
	expiresAt time.Time
}

// Store is an in-memory, goroutine-safe store for pending auth states.
// Records are short-lived (TTL set at construction) and single-use:
// Consume reads and deletes atomically so a replay attempt always fails.
type Store struct {
	mu  sync.Mutex
	ttl time.Duration
	m   map[string]stateRecord
}

// NewStore returns a Store whose records expire after ttl.
// The spec mandates short TTL (≤10 min); callers are responsible for
// passing a compliant value.
func NewStore(ttl time.Duration) *Store {
	return &Store{ttl: ttl, m: make(map[string]stateRecord)}
}

// Save binds st to key (the state value, which the browser echoes back at
// callback). Returns an error if a live record already exists for key.
func (s *Store) Save(key string, st AuthState) error {
	if key == "" {
		return errors.New("signup: store key must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, ok := s.m[key]; ok && time.Now().Before(r.expiresAt) {
		return fmt.Errorf("signup: state record already exists for key")
	}
	s.m[key] = stateRecord{st: st, expiresAt: time.Now().Add(s.ttl)}
	return nil
}

// Consume looks up and atomically deletes the record for key.
// Returns an error if the key is unknown or the record has expired.
// After a successful Consume the key is gone; any replay returns an error.
func (s *Store) Consume(key string) (AuthState, error) {
	if key == "" {
		return AuthState{}, errors.New("signup: store key must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.m[key]
	if !ok {
		return AuthState{}, errors.New("signup: state record not found")
	}
	delete(s.m, key)
	if time.Now().After(r.expiresAt) {
		return AuthState{}, errors.New("signup: state record expired")
	}
	return r.st, nil
}
