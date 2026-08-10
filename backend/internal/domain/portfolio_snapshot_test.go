package domain

import "testing"

func TestRemapPortfolioSnapshot_MainBookAndChildren(t *testing.T) {
	snap := PortfolioSnapshot{
		Book: Portfolio{ID: "alice", ClientID: "alice", Name: "Main", Currency: "USDT", StartingBalance: 10000, CashBalance: 9000},
		Positions: []Position{{ClientID: "alice", Symbol: "BTCUSDT", Quantity: 1}},
		Trades:    []Trade{{ID: "t1", ClientID: "alice", Symbol: "BTCUSDT"}},
		OpenOrders: []PendingOrder{{ID: "o1", ClientID: "alice"}},
		Lots:      []TaxLot{{ID: "l1", ClientID: "alice"}},
		Shares:    []PortfolioShare{{PortfolioID: "alice", OwnerClientID: "alice", GranteeClientID: "bob", Role: PortfolioRoleViewer}},
	}
	got := RemapPortfolioSnapshot(snap, "alice", "carol")
	if got.Book.ID != "carol" || got.Book.ClientID != "carol" {
		t.Fatalf("book %+v", got.Book)
	}
	if got.Positions[0].ClientID != "carol" || got.Trades[0].ClientID != "carol" {
		t.Fatalf("children %+v %+v", got.Positions, got.Trades)
	}
	if got.Shares[0].PortfolioID != "carol" || got.Shares[0].OwnerClientID != "carol" || got.Shares[0].GranteeClientID != "bob" {
		t.Fatalf("share %+v", got.Shares[0])
	}
}

func TestRemapPortfolioSnapshot_ExtraBookKeepsID(t *testing.T) {
	snap := PortfolioSnapshot{
		Book: Portfolio{ID: "uuid-1", ClientID: "alice", Name: "Risky", Currency: "USDT"},
		Positions: []Position{{ClientID: "uuid-1", Symbol: "ETHUSDT"}},
	}
	got := RemapPortfolioSnapshot(snap, "alice", "carol")
	if got.Book.ID != "uuid-1" || got.Book.ClientID != "carol" {
		t.Fatalf("%+v", got.Book)
	}
	if got.Positions[0].ClientID != "uuid-1" {
		t.Fatalf("pos book id remapped unexpectedly: %s", got.Positions[0].ClientID)
	}
}

func TestValidatePortfolioSnapshot(t *testing.T) {
	ok := PortfolioSnapshot{Book: Portfolio{ID: "a", ClientID: "a", Name: "Main", Currency: "USDT", StartingBalance: 1, CashBalance: 1}}
	if err := ValidatePortfolioSnapshot(ok); err != nil {
		t.Fatal(err)
	}
	bad := ok
	bad.Book.CashBalance = -1
	if err := ValidatePortfolioSnapshot(bad); err == nil {
		t.Fatal("expected error")
	}
}
