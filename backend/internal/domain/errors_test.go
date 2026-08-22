package domain

import (
	"errors"
	"fmt"
	"testing"
)

func TestSentinelErrors_DistinctAndWrappable(t *testing.T) {
	sentinels := []error{
		ErrInvalidArgument,
		ErrNotFound,
		ErrForbidden,
		ErrConflict,
		ErrIdempotencyHit,
		ErrUpstream,
		ErrRateLimited,
		ErrSupplyUnmapped,
		ErrCatalogUnmapped,
		ErrHoldersUnpublished,
	}
	for i, a := range sentinels {
		if a == nil || a.Error() == "" {
			t.Fatalf("sentinel %d empty", i)
		}
		for j, b := range sentinels {
			if i != j && errors.Is(a, b) {
				t.Fatalf("%v should not be %v", a, b)
			}
		}
		wrapped := fmt.Errorf("wrap: %w", a)
		if !errors.Is(wrapped, a) {
			t.Fatalf("not wrappable: %v", a)
		}
	}
}
