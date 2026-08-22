package portfoliostore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// Import remints non-UUID extra book ids and already-taken ids, but keeps a
// free UUID. A UUID-shaped unused clientId can therefore be occupied as
// another tenant's first-book primary key.
func TestReview_ImportUUIDOccupiesUnusedTenantFirstBook(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "pf.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ctx := context.Background()
	now := time.Now().UTC()
	victim := "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"

	if _, err := st.CreatePortfolio(ctx, domain.Portfolio{
		ID: "attacker", ClientID: "attacker", Name: "Main", Currency: "USDT",
		StartingBalance: 100, CashBalance: 100, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	crafted := domain.PortfolioSnapshot{
		Book: domain.Portfolio{
			ID: victim, ClientID: "attacker", Name: "Squat", Currency: "USDT",
			StartingBalance: 1, CashBalance: 1, CreatedAt: now, UpdatedAt: now,
		},
	}
	mapped := domain.RemapPortfolioSnapshot(crafted, "attacker", "attacker")
	mapped = domain.RekeyPortfolioSnapshot(mapped, uuid.NewString)
	n, err := st.ImportOwnedPortfolios(ctx, "attacker", []domain.PortfolioSnapshot{mapped}, false)
	if err != nil || n != 1 {
		t.Fatalf("import n=%d err=%v", n, err)
	}

	if _, gerr := st.GetPortfolio(ctx, victim); gerr == nil {
		t.Fatal("import kept victim UUID as attacker extra book")
	}

	if _, cerr := st.CreatePortfolio(ctx, domain.Portfolio{
		ID: victim, ClientID: victim, Name: "Main", Currency: "USDT",
		StartingBalance: 10000, CashBalance: 10000, CreatedAt: now, UpdatedAt: now,
	}); cerr != nil {
		t.Fatalf("victim first book: %v", cerr)
	}
}
