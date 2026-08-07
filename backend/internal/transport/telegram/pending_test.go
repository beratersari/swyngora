package telegram

import (
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestPendingStore_TakeAndExpire(t *testing.T) {
	s := newPendingStore()
	p := &pendingTrade{
		ID: "one", ChatID: 1, UserID: 2, Side: domain.TradeSideBuy,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	s.put(p)
	got := s.get("one")
	if got == nil || got.UserID != 2 {
		t.Fatalf("%+v", got)
	}
	taken := s.take("one")
	if taken == nil {
		t.Fatal("take")
	}
	if s.take("one") != nil {
		t.Fatal("second take must be empty")
	}
	s.put(&pendingTrade{ID: "old", ExpiresAt: time.Now().Add(-time.Minute)})
	if s.get("old") != nil {
		t.Fatal("expired get")
	}
}
