package portfoliostore

import (
	"context"
	"math"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestSnapshot_ExportImportRoundTripAndMerge(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "pf.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ctx := context.Background()
	now := time.Now().UTC()
	book := domain.Portfolio{
		ID: "alice", ClientID: "alice", Name: "Main", Currency: "USDT",
		StartingBalance: 10000, CashBalance: 8800, RealizedPnLTotal: 10, NetDeposits: 0,
		MarginMode: domain.MarginModeIsolated, CreatedAt: now, UpdatedAt: now,
	}
	if _, err := st.CreatePortfolio(ctx, book); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertPosition(ctx, domain.Position{
		ClientID: "alice", Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
		Quantity: 1, AvgCost: 100.1, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.InsertTrade(ctx, domain.Trade{
		ID: "t1", ClientID: "alice", Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
		Side: domain.TradeSideBuy, Quantity: 1, Price: 100.05, Notional: 100.05, Fee: 0.1, LastPrice: 100, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.InsertTaxLot(ctx, domain.TaxLot{
		ID: "l1", ClientID: "alice", Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
		Quantity: 1, OriginalQuantity: 1, Price: 100.15, OpenedAt: now, SourceTradeID: "t1",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreatePendingOrder(ctx, domain.PendingOrder{
		ID: "o1", ClientID: "alice", Exchange: domain.ExchangeBinance, Symbol: "ETHUSDT",
		Type: domain.PendingLimitBuy, Side: domain.TradeSideBuy, Quantity: 2, RemainingQuantity: 2,
		TriggerPrice: 90, ReservedCash: 180, Status: domain.PendingStatusOpen, TimeInForce: domain.TimeInForceGTC,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateRecurringBuyPlan(ctx, domain.RecurringBuyPlan{
		ID: "r1", ClientID: "alice", Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
		Name: "Daily", Amount: 50, Frequency: domain.RecurringDaily, Status: domain.RecurringBuyActive,
		NextRunAt: now.Add(time.Hour), CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateMarginPosition(ctx, domain.MarginPosition{
		ID: "m1", ClientID: "alice", Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
		Side: domain.MarginLong, Mode: domain.MarginModeIsolated, Quantity: 1, EntryPrice: 100,
		Leverage: 5, Margin: 20, Status: domain.MarginPositionOpen, OpenedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreatePortfolioShare(ctx, domain.PortfolioShare{
		PortfolioID: "alice", OwnerClientID: "alice", GranteeClientID: "bob",
		Role: domain.PortfolioRoleTrader, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateAllocationBasket(ctx, domain.AllocationBasket{
		ID: "b1", ClientID: "alice", Name: "Core 60/40",
		Targets: []domain.AllocationTarget{
			{Asset: "BTC", Exchange: domain.ExchangeBinance, WeightPct: 60},
			{Asset: "USDT", WeightPct: 40},
		},
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateRebalanceResult(ctx, domain.RebalanceResult{
		ID: "rb1", ClientID: "alice", BasketID: "b1", BasketName: "Core 60/40",
		Currency: "USDT", Status: domain.RebalanceResultPartial,
		PlannedLegs: 1, FilledLegs: 1, FailedLegs: 0,
		BeforeEquity: 10000, AfterEquity: 9980, BeforeCash: 8800, AfterCash: 4000,
		RemainingNotional: 50, AttemptCount: 1, CreatedAt: now, UpdatedAt: now,
		Legs: []domain.RebalanceLegOutcome{{
			Side: domain.TradeSideSell, Asset: "BTC", Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
			Reason: "overweight", PlannedQty: 0.1, PlannedPrice: 100, PlannedNotional: 10,
			Status: domain.RebalanceLegFilled, TradeID: "t1", FilledQty: 0.1, FillPrice: 100, FillNotional: 10,
			Attempt: 1,
		}},
		Lines: []domain.RebalanceLineSnapshot{{
			Asset: "BTC", Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
			TargetPct: 60, BeforePct: 70, AfterPct: 65, Attempt: 1,
		}},
		Attempts: []domain.RebalanceAttempt{{
			Seq: 1, Status: domain.RebalanceResultPartial, PlannedLegs: 1, FilledLegs: 1,
			CreatedAt: now,
		}},
	}); err != nil {
		t.Fatal(err)
	}
	loss, weight := 5.0, 30.0
	if _, err := st.UpsertRiskLimits(ctx, domain.RiskLimits{
		ClientID: "alice", MaxDailyLossPct: &loss, MaxAssetWeightPct: &weight,
		DayKey: "2026-09-13", DayStartEquity: 10000, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	snaps, err := st.ExportOwnedPortfolios(ctx, "alice")
	if err != nil || len(snaps) != 1 {
		t.Fatalf("export n=%d err=%v", len(snaps), err)
	}
	if len(snaps[0].Positions) != 1 || len(snaps[0].Trades) != 1 || len(snaps[0].OpenOrders) != 1 ||
		len(snaps[0].Lots) != 1 || len(snaps[0].RecurringPlans) != 1 || len(snaps[0].MarginPositions) != 1 ||
		len(snaps[0].Shares) != 1 || len(snaps[0].Baskets) != 1 || len(snaps[0].Rebalances) != 1 ||
		snaps[0].RiskLimits == nil {
		t.Fatalf("incomplete snapshot %+v", snaps[0])
	}

	// Merge into same owner: duplicate, added 0
	n, err := st.ImportOwnedPortfolios(ctx, "alice", snaps, false)
	if err != nil || n != 0 {
		t.Fatalf("merge self n=%d err=%v", n, err)
	}

	// Remap and import as carol (replace)
	mapped := domain.RemapPortfolioSnapshot(snaps[0], "alice", "carol")
	mapped = domain.RekeyPortfolioSnapshot(mapped, uuid.NewString)
	n, err = st.ImportOwnedPortfolios(ctx, "carol", []domain.PortfolioSnapshot{mapped}, true)
	if err != nil || n != 1 {
		t.Fatalf("replace carol n=%d err=%v", n, err)
	}
	got, err := st.GetPortfolio(ctx, "carol")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got.CashBalance-8800) > 1e-9 || got.Name != "Main" {
		t.Fatalf("book %+v", got)
	}
	pos, err := st.GetPosition(ctx, "carol", domain.ExchangeBinance, "BTCUSDT")
	if err != nil || math.Abs(pos.Quantity-1) > 1e-9 {
		t.Fatalf("pos %+v %v", pos, err)
	}
	orders, err := st.ListPendingOrders(ctx, "carol", domain.PendingStatusOpen, 10, 0)
	if err != nil || len(orders) != 1 {
		t.Fatalf("orders %+v %v", orders, err)
	}
	shares, err := st.ListPortfolioSharesByBook(ctx, "carol")
	if err != nil || len(shares) != 1 || shares[0].GranteeClientID != "bob" || shares[0].OwnerClientID != "carol" {
		t.Fatalf("shares %+v %v", shares, err)
	}
	baskets, err := st.ListAllocationBaskets(ctx, "carol")
	if err != nil || len(baskets) != 1 || baskets[0].Name != "Core 60/40" || len(baskets[0].Targets) != 2 {
		t.Fatalf("baskets %+v %v", baskets, err)
	}
	rbs, err := st.ListRebalanceResults(ctx, "carol", baskets[0].ID, 10, 0)
	if err != nil || len(rbs) != 1 {
		t.Fatalf("rebalances %+v %v", rbs, err)
	}
	full, err := st.GetRebalanceResult(ctx, "carol", rbs[0].ID)
	if err != nil || full.Status != domain.RebalanceResultPartial || len(full.Legs) != 1 {
		t.Fatalf("rebalance %+v %v", full, err)
	}
	if full.Legs[0].TradeID == "t1" {
		t.Fatal("rebalance trade id was not reminted")
	}
	lim, err := st.GetRiskLimits(ctx, "carol")
	if err != nil || lim.MaxDailyLossPct == nil || *lim.MaxDailyLossPct != 5 ||
		lim.MaxAssetWeightPct == nil || *lim.MaxAssetWeightPct != 30 || lim.DayKey != "2026-09-13" {
		t.Fatalf("risk %+v %v", lim, err)
	}

	// Merge again: skip
	n, err = st.ImportOwnedPortfolios(ctx, "carol", []domain.PortfolioSnapshot{mapped}, false)
	if err != nil || n != 0 {
		t.Fatalf("merge skip n=%d err=%v", n, err)
	}
}

func TestSnapshot_ImportDoesNotOccupyVictimFirstBookID(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "pf.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ctx := context.Background()
	now := time.Now().UTC()

	crafted := domain.PortfolioSnapshot{
		Book: domain.Portfolio{
			ID: "victim", ClientID: "attacker", Name: "Main", Currency: "USDT",
			StartingBalance: 1, CashBalance: 1, CreatedAt: now, UpdatedAt: now,
		},
	}
	mapped := domain.RemapPortfolioSnapshot(crafted, "attacker", "attacker")
	mapped = domain.RekeyPortfolioSnapshot(mapped, uuid.NewString)
	n, err := st.ImportOwnedPortfolios(ctx, "attacker", []domain.PortfolioSnapshot{mapped}, false)
	if err != nil || n != 1 {
		t.Fatalf("import n=%d err=%v", n, err)
	}
	if got, gerr := st.GetPortfolio(ctx, "victim"); gerr == nil {
		t.Fatalf("victim first-book id occupied: %+v", got)
	}
	if _, err := st.CreatePortfolio(ctx, domain.Portfolio{
		ID: "victim", ClientID: "victim", Name: "Main", Currency: "USDT",
		StartingBalance: 10000, CashBalance: 10000, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("victim first book: %v", err)
	}
}

func TestSnapshot_ImportRemintsCollidingExtraUUID(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "pf.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ctx := context.Background()
	now := time.Now().UTC()
	secondID := "608d45b9-b24f-46cb-ac62-07b3701cdec7"

	if _, err := st.CreatePortfolio(ctx, domain.Portfolio{
		ID: "alice", ClientID: "alice", Name: "Main", Currency: "USDT",
		StartingBalance: 10000, CashBalance: 10000, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreatePortfolio(ctx, domain.Portfolio{
		ID: secondID, ClientID: "alice", Name: "Alt", Currency: "USDT",
		StartingBalance: 1000, CashBalance: 1000, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreatePortfolio(ctx, domain.Portfolio{
		ID: "bob", ClientID: "bob", Name: "Main", Currency: "USDT",
		StartingBalance: 5000, CashBalance: 5000, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	snaps, err := st.ExportOwnedPortfolios(ctx, "alice")
	if err != nil || len(snaps) != 2 {
		t.Fatalf("export n=%d err=%v", len(snaps), err)
	}
	var remapped []domain.PortfolioSnapshot
	for _, snap := range snaps {
		m := domain.RemapPortfolioSnapshot(snap, "alice", "bob")
		m = domain.RekeyPortfolioSnapshot(m, uuid.NewString)
		remapped = append(remapped, m)
	}
	n, err := st.ImportOwnedPortfolios(ctx, "bob", remapped, false)
	if err != nil || n != 1 {
		t.Fatalf("bob import n=%d err=%v", n, err)
	}
	books, err := st.ListPortfolios(ctx, "bob")
	if err != nil || len(books) != 2 {
		t.Fatalf("bob books n=%d err=%v", len(books), err)
	}
	aliceAlt, err := st.GetPortfolio(ctx, secondID)
	if err != nil || aliceAlt.ClientID != "alice" {
		t.Fatalf("alice extra book mutated: %+v %v", aliceAlt, err)
	}
}
