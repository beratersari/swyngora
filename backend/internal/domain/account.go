package domain

import (
	"context"
	"fmt"
	"time"
)

// AccountCloseGrace is how long closed accounts keep data and may reopen.
const AccountCloseGrace = 7 * 24 * time.Hour

// AccountStatus is the lifecycle of a clientId “account” (opaque tenancy key).
type AccountStatus string

const (
	AccountActive AccountStatus = "active"
	// AccountClosed: user closed the account; data retained until PurgeAt.
	AccountClosed AccountStatus = "closed"
	// AccountPurged is terminal after grace purge (row may be deleted).
	AccountPurged AccountStatus = "purged"
)

// Account tracks close/reopen for a clientId.
type Account struct {
	ClientID  string
	Status    AccountStatus
	ClosedAt  *time.Time
	PurgeAt   *time.Time // ClosedAt + AccountCloseGrace when closed
	ReopenedAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// IsClosed reports whether the account blocks login/API access (closed, not yet purged).
func (a *Account) IsClosed() bool {
	return a != nil && a.Status == AccountClosed
}

// CanReopen reports whether reopen is still allowed.
func (a *Account) CanReopen(now time.Time) bool {
	if a == nil || a.Status != AccountClosed || a.PurgeAt == nil {
		return false
	}
	return now.Before(a.PurgeAt.UTC()) || now.Equal(a.PurgeAt.UTC())
}

// AccountPort persists account close/reopen state.
type AccountPort interface {
	Get(ctx context.Context, clientID string) (*Account, error)
	// UpsertActive ensures an active row exists (first touch).
	UpsertActive(ctx context.Context, clientID string, at time.Time) (*Account, error)
	// Close marks closed with purgeAt = closedAt + grace.
	Close(ctx context.Context, clientID string, closedAt, purgeAt time.Time) (*Account, error)
	// Reopen clears closed state if still within grace.
	Reopen(ctx context.Context, clientID string, at time.Time) (*Account, error)
	// ListDueForPurge returns closed accounts with purge_at <= now.
	ListDueForPurge(ctx context.Context, now time.Time, limit int) ([]Account, error)
	// MarkPurged sets status purged (or Delete).
	MarkPurged(ctx context.Context, clientID string, at time.Time) error
	// Delete removes the account row after purge.
	Delete(ctx context.Context, clientID string) error
}

// ClientDataPurger removes all product data for a clientId after the grace period.
type ClientDataPurger interface {
	PurgeClient(ctx context.Context, clientID string) error
}

// ErrAccountClosed is returned when a closed clientId tries to use the API.
// Unwraps to ErrForbidden (HTTP 403).
type ErrAccountClosed struct {
	ClientID string
	PurgeAt  *time.Time
}

func (e *ErrAccountClosed) Error() string {
	if e.PurgeAt != nil {
		return fmt.Sprintf("forbidden: account is closed; reopen before %s", e.PurgeAt.UTC().Format(time.RFC3339))
	}
	return "forbidden: account is closed"
}

func (e *ErrAccountClosed) Unwrap() error { return ErrForbidden }
