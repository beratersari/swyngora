package domain

import "testing"

func TestRemapPortfolioSnapshot_MainBookAndChildren(t *testing.T) {
	snap := PortfolioSnapshot{
		Book:       Portfolio{ID: "alice", ClientID: "alice", Name: "Main", Currency: "USDT", StartingBalance: 10000, CashBalance: 9000},
		Positions:  []Position{{ClientID: "alice", Symbol: "BTCUSDT", Quantity: 1}},
		Trades:     []Trade{{ID: "t1", ClientID: "alice", Symbol: "BTCUSDT"}},
		OpenOrders: []PendingOrder{{ID: "o1", ClientID: "alice"}},
		Lots:       []TaxLot{{ID: "l1", ClientID: "alice"}},
		Shares:     []PortfolioShare{{PortfolioID: "alice", OwnerClientID: "alice", GranteeClientID: "bob", Role: PortfolioRoleViewer}},
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
	const extra = "608d45b9-b24f-46cb-ac62-07b3701cdec7"
	snap := PortfolioSnapshot{
		Book:      Portfolio{ID: extra, ClientID: "alice", Name: "Risky", Currency: "USDT"},
		Positions: []Position{{ClientID: extra, Symbol: "ETHUSDT"}},
	}
	got := RemapPortfolioSnapshot(snap, "alice", "carol")
	if got.Book.ID != extra || got.Book.ClientID != "carol" {
		t.Fatalf("%+v", got.Book)
	}
	if got.Positions[0].ClientID != extra {
		t.Fatalf("pos book id remapped unexpectedly: %s", got.Positions[0].ClientID)
	}
}

func TestMustRemintImportedBookID(t *testing.T) {
	extra := "608d45b9-b24f-46cb-ac62-07b3701cdec7"
	if MustRemintImportedBookID("carol", "carol") {
		t.Fatal("importer first book must be kept")
	}
	if !MustRemintImportedBookID(extra, "carol") {
		t.Fatal("extra UUID book must be reminted so it cannot squat another tenant")
	}
	if !MustRemintImportedBookID("victim", "attacker") {
		t.Fatal("non-UUID extra id would squat another tenant first book")
	}
}

func TestRebindPortfolioSnapshotBookID(t *testing.T) {
	snap := PortfolioSnapshot{
		Book:      Portfolio{ID: "victim", ClientID: "attacker", Name: "Main"},
		Positions: []Position{{ClientID: "victim", Symbol: "BTCUSDT"}},
		Shares:    []PortfolioShare{{PortfolioID: "victim", OwnerClientID: "attacker"}},
	}
	got := RebindPortfolioSnapshotBookID(snap, "new-id")
	if got.Book.ID != "new-id" || got.Book.ClientID != "attacker" {
		t.Fatalf("book %+v", got.Book)
	}
	if got.Positions[0].ClientID != "new-id" || got.Shares[0].PortfolioID != "new-id" {
		t.Fatalf("children %+v %+v", got.Positions, got.Shares)
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

func TestRemapPortfolioSnapshot_BasketsRebalancesAndRisk(t *testing.T) {
	loss := 5.0
	snap := PortfolioSnapshot{
		Book: Portfolio{ID: "alice", ClientID: "alice", Name: "Main", Currency: "USDT"},
		Baskets: []AllocationBasket{{ID: "b1", ClientID: "alice", Name: "Core"}},
		Rebalances: []RebalanceResult{{ID: "r1", ClientID: "alice", BasketID: "b1", Status: RebalanceResultCompleted}},
		RiskLimits: &RiskLimits{ClientID: "alice", MaxDailyLossPct: &loss, DayKey: "2026-09-13"},
	}
	got := RemapPortfolioSnapshot(snap, "alice", "carol")
	if got.Baskets[0].ClientID != "carol" || got.Rebalances[0].ClientID != "carol" {
		t.Fatalf("book-scoped children %+v %+v", got.Baskets, got.Rebalances)
	}
	if got.Baskets[0].ID != "b1" || got.Rebalances[0].BasketID != "b1" {
		t.Fatalf("basket ids remapped unexpectedly")
	}
	if got.RiskLimits == nil || got.RiskLimits.ClientID != "carol" || got.RiskLimits.MaxDailyLossPct == nil || *got.RiskLimits.MaxDailyLossPct != 5 {
		t.Fatalf("risk %+v", got.RiskLimits)
	}
	if snap.RiskLimits.ClientID != "alice" {
		t.Fatal("remap mutated original risk limits")
	}
}

func TestRekeyPortfolioSnapshot_BasketsRebalancesAndTradeIDs(t *testing.T) {
	n := 0
	newID := func() string {
		n++
		return "n" + string(rune('0'+n))
	}
	snap := PortfolioSnapshot{
		Book:   Portfolio{ID: "carol", ClientID: "carol", Name: "Main"},
		Trades: []Trade{{ID: "t1", ClientID: "carol"}},
		Baskets: []AllocationBasket{{ID: "b1", ClientID: "carol", Name: "Core"}},
		Rebalances: []RebalanceResult{{
			ID: "rb1", ClientID: "carol", BasketID: "b1",
			Legs: []RebalanceLegOutcome{{TradeID: "t1", Status: RebalanceLegFilled}},
			Attempts: []RebalanceAttempt{{
				Seq: 1,
				Legs: []RebalanceLegOutcome{{TradeID: "t1", Status: RebalanceLegFilled}},
			}},
		}},
	}
	got := RekeyPortfolioSnapshot(snap, newID)
	if got.Baskets[0].ID == "b1" || got.Rebalances[0].ID == "rb1" {
		t.Fatalf("ids not reminted: basket=%s rebalance=%s", got.Baskets[0].ID, got.Rebalances[0].ID)
	}
	if got.Rebalances[0].BasketID != got.Baskets[0].ID {
		t.Fatalf("rebalance basket %s != %s", got.Rebalances[0].BasketID, got.Baskets[0].ID)
	}
	if got.Trades[0].ID == "t1" {
		t.Fatal("trade id not reminted")
	}
	if got.Rebalances[0].Legs[0].TradeID != got.Trades[0].ID {
		t.Fatalf("leg trade %s != %s", got.Rebalances[0].Legs[0].TradeID, got.Trades[0].ID)
	}
	if got.Rebalances[0].Attempts[0].Legs[0].TradeID != got.Trades[0].ID {
		t.Fatalf("attempt trade %s != %s", got.Rebalances[0].Attempts[0].Legs[0].TradeID, got.Trades[0].ID)
	}
}
