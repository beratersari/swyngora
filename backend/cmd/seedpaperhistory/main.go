// seedpaperhistory creates a paper book with many backdated trades, equity
// snapshots (last ~90 days), and open pending orders for equity-chart demos.
//
//	cd backend && go run ./cmd/seedpaperhistory
//
// Env:
//
//	PORTFOLIO_DB_PATH   default: data/portfolio.db
//	SEED_CLIENT_ID      default: paper-history-demo
//	SEED_CLIENT_IDS     optional comma list (overrides SEED_CLIENT_ID when set)
//	SEED_STARTING       default: 50000
//	SEED_DAYS           default: 90
//	SEED_BOOK_NAME      default: History Demo
//	SEED_REPLACE        default: 1 — delete existing books with SEED_BOOK_NAME only
package main

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/portfoliostore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func main() {
	dbPath := envOr("PORTFOLIO_DB_PATH", "data/portfolio.db")
	if !filepath.IsAbs(dbPath) {
		if _, err := os.Stat(dbPath); err != nil {
			if _, err2 := os.Stat(filepath.Join("backend", dbPath)); err2 == nil {
				dbPath = filepath.Join("backend", dbPath)
			}
		}
	}
	clients := parseClientIDs()
	startBal, _ := strconv.ParseFloat(envOr("SEED_STARTING", "50000"), 64)
	if startBal < domain.MinStartingBalance {
		startBal = 50000
	}
	days, _ := strconv.Atoi(envOr("SEED_DAYS", "90"))
	if days < 7 {
		days = 90
	}
	bookName := envOr("SEED_BOOK_NAME", "History Demo")
	replace := envOr("SEED_REPLACE", "1") != "0"

	store, err := portfoliostore.Open(dbPath)
	if err != nil {
		fail("open portfolio db: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	for i, clientID := range clients {
		fmt.Printf("\n=== seeding client %s ===\n", clientID)
		if err := seedOne(ctx, store, clientID, bookName, startBal, days, replace, int64(42+i)); err != nil {
			fail("seed %s: %v", clientID, err)
		}
	}
	fmt.Printf("\nDone. In the browser console on the app origin:\n")
	fmt.Printf("  localStorage.setItem('swyngora.clientId', %q)\n", clients[0])
	fmt.Printf("then hard-refresh → /portfolio and select book %q\n", bookName)
	if len(clients) > 1 {
		fmt.Printf("Other seeded clientIds: %s\n", strings.Join(clients[1:], ", "))
	}
}

func parseClientIDs() []string {
	if raw := strings.TrimSpace(os.Getenv("SEED_CLIENT_IDS")); raw != "" {
		var out []string
		for _, p := range strings.Split(raw, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return []string{envOr("SEED_CLIENT_ID", "paper-history-demo")}
}

func seedOne(ctx context.Context, store *portfoliostore.SQLite, clientID, bookName string, startBal float64, days int, replace bool, seed int64) error {
	now := time.Now().UTC()
	created := now.Add(-time.Duration(days) * 24 * time.Hour).Truncate(time.Hour)

	list, err := store.ListPortfolios(ctx, clientID)
	if err != nil {
		return err
	}
	if replace {
		for _, p := range list {
			if !strings.EqualFold(p.Name, bookName) {
				continue
			}
			if err := store.DeletePortfolio(ctx, clientID, p.ID); err != nil {
				return fmt.Errorf("delete %s: %w", p.ID, err)
			}
			fmt.Printf("removed existing book %s (%s)\n", p.ID, p.Name)
		}
		list, err = store.ListPortfolios(ctx, clientID)
		if err != nil {
			return err
		}
	}

	bookID := uuid.NewString()
	if len(list) == 0 {
		// First book for this client uses legacy id = clientId.
		bookID = clientID
	}
	p := domain.Portfolio{
		ID:              bookID,
		ClientID:        clientID,
		Name:            bookName,
		Currency:        domain.DefaultPaperCurrency,
		StartingBalance: startBal,
		CashBalance:     startBal,
		MarginMode:      domain.MarginModeIsolated,
		CreatedAt:       created,
		UpdatedAt:       created,
	}
	if _, err := store.CreatePortfolio(ctx, p); err != nil {
		return fmt.Errorf("create portfolio: %w", err)
	}
	_, _ = store.ApplyCashMovement(ctx, &p, domain.CashMovement{
		ID: uuid.NewString(), Kind: domain.CashMovementDeposit,
		Amount: startBal, CashAfter: startBal, NetDepositsAfter: 0,
		Note: "Opening balance (seed)", CreatedAt: created,
	})

	type asset struct {
		symbol string
		base   float64
		vol    float64
	}
	assets := []asset{
		{symbol: "BTCUSDT", base: 62000, vol: 0.025},
		{symbol: "ETHUSDT", base: 3200, vol: 0.03},
		{symbol: "SOLUSDT", base: 145, vol: 0.04},
	}

	rng := rand.New(rand.NewSource(seed))
	type lotState struct {
		lots []domain.TaxLot
		qty  float64
		avg  float64
	}
	pos := map[string]*lotState{}
	for _, a := range assets {
		pos[a.symbol] = &lotState{}
	}

	cash := startBal
	realizedTotal := 0.0
	prices := map[string]float64{}
	for _, a := range assets {
		prices[a.symbol] = a.base
	}

	tradeEvery := 18 * time.Hour
	var tradeCount, buyCount, sellCount int
	var lastSnapEquity = startBal
	depAt := created.Add(time.Duration(days/2) * 24 * time.Hour)
	depAmt := 2500.0
	deposited := false

	writeSnap := func(at time.Time) error {
		posVal := 0.0
		unreal := 0.0
		for sym, st := range pos {
			px := prices[sym]
			posVal += st.qty * px
			unreal += domain.UnrealizedPnL(st.qty, st.avg, px)
		}
		eq := cash + posVal
		lastSnapEquity = eq
		return store.UpsertEquitySnapshot(ctx, domain.EquitySnapshot{
			ClientID: bookID, BucketAt: domain.SnapshotBucket(at, 24*time.Hour),
			TakenAt: at, Equity: eq, CashBalance: cash, PositionsValue: posVal,
			UnrealizedPnL: unreal, RealizedPnL: realizedTotal,
		})
	}
	if err := writeSnap(created); err != nil {
		return err
	}

	for t := created.Add(6 * time.Hour); !t.After(now); t = t.Add(tradeEvery) {
		for _, a := range assets {
			shock := 1 + rng.NormFloat64()*a.vol*0.35 + 0.0004
			prices[a.symbol] = math.Max(prices[a.symbol]*shock, a.base*0.4)
		}

		if !deposited && !t.Before(depAt) {
			cash += depAmt
			p.CashBalance = cash
			p.NetDeposits += depAmt
			p.UpdatedAt = t
			if _, err := store.ApplyCashMovement(ctx, &p, domain.CashMovement{
				ID: uuid.NewString(), Kind: domain.CashMovementDeposit,
				Amount: depAmt, CashAfter: cash, NetDepositsAfter: p.NetDeposits,
				Note: "Seed mid-period top-up", CreatedAt: t,
			}); err != nil {
				return fmt.Errorf("deposit: %w", err)
			}
			if fresh, err := store.GetPortfolio(ctx, bookID); err == nil && fresh != nil {
				cash = fresh.CashBalance
				p = *fresh
			}
			deposited = true
			_ = writeSnap(t)
		}

		if t.Hour() < 12 {
			_ = writeSnap(t)
		}

		a := assets[rng.Intn(len(assets))]
		st := pos[a.symbol]
		last := prices[a.symbol]
		side := domain.TradeSideBuy
		if st.qty > 0 && (cash < startBal*0.15 || rng.Float64() < 0.45) {
			side = domain.TradeSideSell
		}
		if st.qty <= domain.PositionEpsilon {
			side = domain.TradeSideBuy
		}

		cost := domain.TradingCostFor(domain.ExchangeBinance)
		fill := domain.ApplySlippage(last, side, cost.SlippageRate)
		if fill <= 0 {
			continue
		}

		if side == domain.TradeSideBuy {
			notional := startBal * (0.01 + rng.Float64()*0.03)
			if cash < notional*1.02 {
				notional = cash * 0.5
			}
			if notional < 50 {
				continue
			}
			qty := math.Round((notional/fill)*1e6) / 1e6
			if qty < domain.MinTradeQuantity {
				continue
			}
			debit := domain.BuyCashDebit(qty, fill, cost.FeeRate)
			if cash+1e-9 < debit {
				continue
			}
			unit := domain.BuyUnitCost(fill, cost.FeeRate)
			cash -= debit
			tradeID := uuid.NewString()
			lot := domain.NewTaxLot(uuid.NewString(), bookID, domain.ExchangeBinance, a.symbol, qty, unit, t, tradeID)
			st.lots = append(st.lots, lot)
			st.qty += qty
			st.avg = domain.AvgCostFromLots(st.lots)
			fee := domain.FeeAmount(qty, fill, cost.FeeRate)
			p.CashBalance = cash
			p.RealizedPnLTotal = realizedTotal
			p.UpdatedAt = t
			posOut := &domain.Position{
				ClientID: bookID, Exchange: domain.ExchangeBinance, Symbol: a.symbol,
				Quantity: st.qty, AvgCost: st.avg, UpdatedAt: t,
			}
			tr := domain.Trade{
				ID: tradeID, ClientID: bookID, Exchange: domain.ExchangeBinance, Symbol: a.symbol,
				Side: side, Quantity: qty, Price: fill, Notional: qty * fill,
				LotMethod: domain.LotMethodFIFO, Fee: fee, LastPrice: last, CreatedAt: t,
			}
			if err := store.ExecuteTrade(ctx, &p, posOut, tr, &domain.LotOps{Created: []domain.TaxLot{lot}}); err != nil {
				return fmt.Errorf("buy trade: %w", err)
			}
			buyCount++
			tradeCount++
			continue
		}

		frac := 0.2 + rng.Float64()*0.5
		qty := math.Round(st.qty*frac*1e6) / 1e6
		if qty < domain.MinTradeQuantity || qty > st.qty {
			if st.qty >= domain.MinTradeQuantity {
				qty = st.qty
			} else {
				continue
			}
		}
		fills, updated, realized, err := domain.ConsumeLots(st.lots, qty, fill, domain.LotMethodFIFO, t, cost.FeeRate)
		if err != nil {
			continue
		}
		st.lots = domain.MergeLotUpdates(st.lots, updated)
		open := st.lots[:0]
		for _, l := range st.lots {
			if l.Open() {
				open = append(open, l)
			}
		}
		st.lots = open
		st.qty -= qty
		if st.qty < domain.PositionEpsilon {
			st.qty = 0
			st.avg = 0
			st.lots = nil
		} else {
			st.avg = domain.AvgCostFromLots(st.lots)
		}
		credit := domain.SellCashCredit(qty, fill, cost.FeeRate)
		cash += credit
		realizedTotal += realized
		tradeID := uuid.NewString()
		for i := range fills {
			fills[i].TradeID = tradeID
			if fills[i].ID == "" {
				fills[i].ID = tradeID + "-f" + strconv.Itoa(i)
			}
		}
		fee := domain.FeeAmount(qty, fill, cost.FeeRate)
		p.CashBalance = cash
		p.RealizedPnLTotal = realizedTotal
		p.UpdatedAt = t
		posOut := &domain.Position{
			ClientID: bookID, Exchange: domain.ExchangeBinance, Symbol: a.symbol,
			Quantity: st.qty, AvgCost: st.avg, UpdatedAt: t,
		}
		if st.qty <= domain.PositionEpsilon {
			posOut.Quantity = 0
			posOut.AvgCost = 0
		}
		tr := domain.Trade{
			ID: tradeID, ClientID: bookID, Exchange: domain.ExchangeBinance, Symbol: a.symbol,
			Side: side, Quantity: qty, Price: fill, Notional: qty * fill, RealizedPnL: realized,
			LotMethod: domain.LotMethodFIFO, LotFills: fills, Fee: fee, LastPrice: last, CreatedAt: t,
		}
		if err := store.ExecuteTrade(ctx, &p, posOut, tr, &domain.LotOps{Updated: updated, Fills: fills}); err != nil {
			return fmt.Errorf("sell trade: %w", err)
		}
		sellCount++
		tradeCount++
	}

	_ = writeSnap(now)

	// Open pending orders (resting — not filled) so Orders table is non-empty.
	// Keep reserves small enough to leave cash free for the chart.
	cost := domain.TradingCostFor(domain.ExchangeBinance)
	nowOrd := now
	openOrders := 0

	// limit buy BTC far below market
	btcPx := prices["BTCUSDT"]
	buyQty := 0.005
	buyTrigger := math.Round(btcPx*0.85*100) / 100
	buyReserve := domain.BuyReserveCash(buyQty, buyTrigger, cost)
	if cash > buyReserve+100 {
		o := domain.PendingOrder{
			ID: uuid.NewString(), ClientID: bookID, Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
			Type: domain.PendingLimitBuy, Side: domain.TradeSideBuy,
			Quantity: buyQty, RemainingQuantity: buyQty, TriggerPrice: buyTrigger,
			ReservedCash: buyReserve, TimeInForce: domain.TimeInForceGTC,
			Status: domain.PendingStatusOpen, CreatedAt: nowOrd, UpdatedAt: nowOrd,
		}
		if _, err := store.CreatePendingOrder(ctx, o); err != nil {
			return fmt.Errorf("limit buy: %w", err)
		}
		openOrders++
	}

	// limit sell ETH above market (if we hold ETH)
	if st := pos["ETHUSDT"]; st != nil && st.qty > 0.01 {
		sellQty := math.Min(0.05, st.qty*0.25)
		sellQty = math.Round(sellQty*1e6) / 1e6
		if sellQty >= domain.MinTradeQuantity {
			ethPx := prices["ETHUSDT"]
			sellTrigger := math.Round(ethPx*1.12*100) / 100
			o := domain.PendingOrder{
				ID: uuid.NewString(), ClientID: bookID, Exchange: domain.ExchangeBinance, Symbol: "ETHUSDT",
				Type: domain.PendingLimitSell, Side: domain.TradeSideSell,
				Quantity: sellQty, RemainingQuantity: sellQty, TriggerPrice: sellTrigger,
				ReservedQuantity: sellQty, TimeInForce: domain.TimeInForceGTC,
				Status: domain.PendingStatusOpen, LotMethod: domain.LotMethodFIFO,
				CreatedAt: nowOrd, UpdatedAt: nowOrd,
			}
			if _, err := store.CreatePendingOrder(ctx, o); err != nil {
				return fmt.Errorf("limit sell: %w", err)
			}
			openOrders++
		}
	}

	// stop-loss on SOL if held
	if st := pos["SOLUSDT"]; st != nil && st.qty > 0.5 {
		slQty := math.Min(5, st.qty*0.3)
		slQty = math.Round(slQty*1e6) / 1e6
		if slQty >= domain.MinTradeQuantity {
			solPx := prices["SOLUSDT"]
			slTrigger := math.Round(solPx*0.9*100) / 100
			o := domain.PendingOrder{
				ID: uuid.NewString(), ClientID: bookID, Exchange: domain.ExchangeBinance, Symbol: "SOLUSDT",
				Type: domain.PendingStopLoss, Side: domain.TradeSideSell,
				Quantity: slQty, RemainingQuantity: slQty, TriggerPrice: slTrigger,
				ReservedQuantity: slQty, TimeInForce: domain.TimeInForceGTC,
				Status: domain.PendingStatusOpen, LotMethod: domain.LotMethodFIFO,
				CreatedAt: nowOrd, UpdatedAt: nowOrd,
			}
			if _, err := store.CreatePendingOrder(ctx, o); err != nil {
				return fmt.Errorf("stop loss: %w", err)
			}
			openOrders++
		}
	}

	// trailing stop on BTC if held
	if st := pos["BTCUSDT"]; st != nil && st.qty > 0.001 {
		trQty := math.Min(0.002, st.qty*0.2)
		trQty = math.Round(trQty*1e6) / 1e6
		if trQty >= domain.MinTradeQuantity {
			btcPx := prices["BTCUSDT"]
			trail := 0.05
			peak := btcPx
			stop := domain.TrailStopPrice(peak, trail, domain.TrailTypePercent)
			o := domain.PendingOrder{
				ID: uuid.NewString(), ClientID: bookID, Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
				Type: domain.PendingTrailingStop, Side: domain.TradeSideSell,
				Quantity: trQty, RemainingQuantity: trQty, TriggerPrice: stop,
				ReservedQuantity: trQty, TimeInForce: domain.TimeInForceGTC,
				TrailType: domain.TrailTypePercent, TrailValue: trail, TrailPeak: peak,
				Status: domain.PendingStatusOpen, LotMethod: domain.LotMethodFIFO,
				CreatedAt: nowOrd, UpdatedAt: nowOrd,
			}
			if _, err := store.CreatePendingOrder(ctx, o); err != nil {
				return fmt.Errorf("trailing stop: %w", err)
			}
			openOrders++
		}
	}

	// Ensure open-order reserves don't break cash view: CreatePendingOrder only stores reservations;
	// available cash is computed as cash - sum(reserved). Good.

	posVal := 0.0
	for sym, st := range pos {
		posVal += st.qty * prices[sym]
	}
	fmt.Printf("  portfolioId: %s\n", bookID)
	fmt.Printf("  name:        %s\n", bookName)
	fmt.Printf("  createdAt:   %s\n", created.Format(time.RFC3339))
	fmt.Printf("  trades:      %d (%d buys / %d sells)\n", tradeCount, buyCount, sellCount)
	fmt.Printf("  open orders: %d\n", openOrders)
	fmt.Printf("  cash:        %.2f USDT\n", cash)
	fmt.Printf("  positions≈:  %.2f USDT\n", posVal)
	fmt.Printf("  equity≈:     %.2f (last snap %.2f)\n", cash+posVal, lastSnapEquity)
	fmt.Printf("  realized:    %.2f\n", realizedTotal)
	return nil
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
