package portfolio

import "sync"

// lockClient serializes all cash/position read-modify-write for one clientId.
// Matches the documented hardening: paper mutations are serialized per client
// so the filler's multi-symbol fan-out and concurrent HTTP orders cannot
// last-write-wins the same cash_balance. Mutexes are not reentrant — callers
// must not lock the same client twice on one goroutine.
func (s *Service) lockClient(clientID string) func() {
	if s == nil || clientID == "" {
		return func() {}
	}
	v, _ := s.clientMu.LoadOrStore(clientID, &sync.Mutex{})
	mu := v.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}
