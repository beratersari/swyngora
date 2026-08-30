package middleware

import (
	"errors"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestWSTicket_IssueConsumeOnce(t *testing.T) {
	s := NewWSTicketIssuer()
	tok, exp, err := s.Issue(AuthIdentity{ClientID: "alice", UserKey: true, CanTrade: true, KeyID: "k1"})
	if err != nil || tok == "" || exp.Before(time.Now()) {
		t.Fatalf("issue: %q %v %v", tok, exp, err)
	}
	id, err := s.Consume(tok)
	if err != nil || id == nil || id.ClientID != "alice" || !id.CanTrade || id.KeyID != "k1" {
		t.Fatalf("consume: %+v %v", id, err)
	}
	if _, err := s.Consume(tok); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("second consume want forbidden, got %v", err)
	}
}

func TestWSTicket_UnknownAndEmpty(t *testing.T) {
	s := NewWSTicketIssuer()
	if _, err := s.Consume(""); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("empty: %v", err)
	}
	if _, err := s.Consume("deadbeef"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("unknown: %v", err)
	}
}
