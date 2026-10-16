package portfolio

import (
	"context"
	"errors"
	"math"
	"strconv"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type moveCandles struct {
	price       float64
	err         error
	calls       int
	nilStarts   int
	ranged      int
	rejectRange bool
	spans       [][2]time.Time
	bars        []domain.Candle
}

func (f *moveCandles) GetCandles(_ context.Context, _, _, _ string, _ int, start, end *time.Time) ([]domain.Candle, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	if start == nil || end == nil {
		f.nilStarts++
		if f.rejectRange {
			return append([]domain.Candle(nil), f.bars...), nil
		}
		return append([]domain.Candle(nil), f.bars...), nil
	}
	if f.rejectRange {
		return nil, errors.New("equity lookup passed a range")
	}
	f.ranged++
	f.spans = append(f.spans, [2]time.Time{start.UTC(), end.UTC()})
	px := f.price
	if px <= 0 {
		px = 110
	}
	var bars []domain.Candle
	for m := start.UTC().Truncate(time.Minute); m.Before(end.UTC()); m = m.Add(time.Minute) {
		bars = append(bars, domain.Candle{
			OpenTime:  m,
			Close:     strconv.FormatFloat(px, 'f', -1, 64),
			CloseTime: m.Add(time.Minute - time.Millisecond),
		})
	}
	return bars, nil
}

func TestChunkMoveMinutes_Splits24hFrom1h(t *testing.T) {
	start := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	chunks := chunkMoveMinutes([]time.Time{
		start.Add(24 * time.Hour),
		start.Add(time.Hour),
		start.Add(4 * time.Hour),
		start.Add(time.Hour),
	})
	if len(chunks) != 2 {
		t.Fatalf("chunks %d", len(chunks))
	}
	if len(chunks[0]) != 2 || !chunks[0][0].Equal(start.Add(time.Hour)) || !chunks[0][1].Equal(start.Add(4*time.Hour)) {
		t.Fatalf("first %+v", chunks[0])
	}
	if len(chunks[1]) != 1 || !chunks[1][0].Equal(start.Add(24*time.Hour)) {
		t.Fatalf("second %+v", chunks[1])
	}
}

func TestOpenLotAnchor_UsesLatestLot(t *testing.T) {
	t0 := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	got := openLotAnchor(domain.Position{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", AvgCost: 150}, []domain.TaxLot{
		{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Quantity: 1, Price: 100, OpenedAt: t0},
		{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Quantity: 1, Price: 200, OpenedAt: t0.Add(time.Minute)},
	})
	if got.price != 200 || !got.at.Equal(t0.Add(time.Minute)) {
		t.Fatalf("anchor %+v", got)
	}
	if empty := openLotAnchor(domain.Position{AvgCost: 150}, nil); empty.price != 0 || !empty.at.IsZero() {
		t.Fatalf("empty %+v", empty)
	}
}

func TestListTrades_PendingMoveDoesNotFetch(t *testing.T) {
	ctx := context.Background()
	svc := newSvc(t, nil)
	fake := &moveCandles{}
	svc.SetCandleSource(fake)
	if _, err := svc.Create(ctx, CreateInput{ClientID: "mv1", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "mv1", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	list, _, err := svc.ListTrades(ctx, "mv1", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if fake.calls != 0 {
		t.Fatalf("calls %d", fake.calls)
	}
	if len(list) != 1 || len(list[0].Moves) != 3 {
		t.Fatalf("%+v", list)
	}
	for _, m := range list[0].Moves {
		if m.Status != domain.FillMovePending || m.Pct != nil {
			t.Fatalf("pending %+v", m)
		}
	}
	view, err := svc.View(ctx, "mv1")
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Positions) != 1 || view.Positions[0].Moves[0].Status != domain.FillMovePending {
		t.Fatalf("position %+v", view.Positions)
	}
}

func TestView_MoveUsesLatestLotNotAverage(t *testing.T) {
	ctx := context.Background()
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	fake := &moveCandles{price: 220}
	svc.SetCandleSource(fake)
	if _, err := svc.Create(ctx, CreateInput{ClientID: "mv2", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "mv2", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	px.prices["binance|BTCUSDT"] = "200"
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "mv2", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	svc.moveClock = func() time.Time { return time.Now().UTC().Add(2 * time.Hour) }
	view, err := svc.View(ctx, "mv2")
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Positions) != 1 {
		t.Fatalf("positions %+v", view.Positions)
	}
	pos := view.Positions[0]
	if math.Abs(pos.AvgCost-150) > 1e-6 {
		t.Fatalf("avg %v", pos.AvgCost)
	}
	m := pos.Moves[0]
	if m.Status != domain.FillMoveReady || m.Pct == nil || math.Abs(*m.Pct-10) > 1e-9 {
		t.Fatalf("move %+v pct %v", m, m.Pct)
	}
	if pos.Moves[1].Status != domain.FillMovePending || pos.Moves[2].Status != domain.FillMovePending {
		t.Fatalf("later %+v", pos.Moves)
	}
	if fake.calls != 1 {
		t.Fatalf("calls %d", fake.calls)
	}
	if _, _, err := svc.ListTrades(ctx, "mv2", 10, 0); err != nil {
		t.Fatal(err)
	}
	if fake.calls != 1 {
		t.Fatalf("cache missed, calls %d", fake.calls)
	}
}

func TestListTrades_CandleErrorIsAMissingWindow(t *testing.T) {
	ctx := context.Background()
	svc := newSvc(t, nil)
	fake := &moveCandles{err: errors.New("upstream")}
	svc.SetCandleSource(fake)
	if _, err := svc.Create(ctx, CreateInput{ClientID: "mv3", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "mv3", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	svc.moveClock = func() time.Time { return time.Now().UTC().Add(2 * time.Hour) }
	list, _, err := svc.ListTrades(ctx, "mv3", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if list[0].Moves[0].Status != domain.FillMoveMissing || list[0].Moves[0].Pct != nil {
		t.Fatalf("%+v", list[0].Moves[0])
	}
	if list[0].Moves[1].Status != domain.FillMovePending {
		t.Fatalf("4h %+v", list[0].Moves[1])
	}
}

func TestFillMoves_NilSourceAndCancelDoNotInventAPrice(t *testing.T) {
	ctx := context.Background()
	svc := newSvc(t, nil)
	fillAt := time.Now().UTC().Add(-2 * time.Hour)
	svc.moveClock = func() time.Time { return time.Now().UTC() }
	moves := svc.fillMoves(ctx, []moveAnchor{{
		exchange: domain.ExchangeBinance, symbol: "BTCUSDT", price: 100, at: fillAt,
	}})
	if moves[0][0].Status != domain.FillMoveMissing || moves[0][0].Pct != nil {
		t.Fatalf("nil source %+v", moves[0][0])
	}

	fake := &moveCandles{price: 110}
	svc.SetCandleSource(fake)
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	moves = svc.fillMoves(cancelled, []moveAnchor{{
		exchange: domain.ExchangeBinance, symbol: "BTCUSDT", price: 100, at: fillAt,
	}})
	if moves[0][0].Status != domain.FillMoveMissing || fake.calls != 0 {
		t.Fatalf("cancel status %+v calls %d", moves[0][0], fake.calls)
	}
	moves = svc.fillMoves(ctx, []moveAnchor{{
		exchange: domain.ExchangeBinance, symbol: "BTCUSDT", price: 100, at: fillAt,
	}})
	if moves[0][0].Status != domain.FillMoveReady || moves[0][0].Pct == nil || math.Abs(*moves[0][0].Pct-10) > 1e-9 {
		t.Fatalf("after cancel %+v", moves[0][0])
	}
	if fake.calls != 1 {
		t.Fatalf("calls %d", fake.calls)
	}
}

func TestFillMoves_ReadyZeroAndTwoChunks(t *testing.T) {
	svc := newSvc(t, nil)
	fake := &moveCandles{price: 100}
	svc.SetCandleSource(fake)
	fillAt := time.Date(2026, 5, 1, 8, 15, 0, 0, time.UTC)
	svc.moveClock = func() time.Time { return fillAt.Add(25 * time.Hour) }
	moves := svc.fillMoves(context.Background(), []moveAnchor{{
		exchange: domain.ExchangeBinance, symbol: "BTCUSDT", price: 100, at: fillAt,
	}})
	for _, m := range moves[0] {
		if m.Status != domain.FillMoveReady || m.Pct == nil || *m.Pct != 0 {
			t.Fatalf("zero %+v", m)
		}
	}
	if fake.calls != 2 {
		t.Fatalf("calls %d spans %+v", fake.calls, fake.spans)
	}
	if fake.spans[0][1].Sub(fake.spans[0][0]) > 5*time.Hour {
		t.Fatalf("first span too wide %s", fake.spans[0][1].Sub(fake.spans[0][0]))
	}
	if fake.spans[1][0].Sub(fake.spans[0][0]) < 19*time.Hour {
		t.Fatalf("24h chunk not split %+v", fake.spans)
	}
}

func TestFillMoves_EquityUsesLatestMinuteOnly(t *testing.T) {
	ctx := context.Background()
	px := &fakePx{prices: map[string]string{"nasdaq|AAPL": "100"}}
	svc := newSvc(t, px)
	fake := &moveCandles{rejectRange: true}
	svc.SetCandleSource(fake)
	if _, err := svc.Create(ctx, CreateInput{ClientID: "mv4", StartingBalance: 10000, Currency: "USD"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "mv4", Exchange: "nasdaq", Symbol: "AAPL", Side: "buy", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}
	list, _, err := svc.ListTrades(ctx, "mv4", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	fillAt := list[0].CreatedAt
	hour := domain.FillMoveBarOpen(domain.FillMoveHorizon(fillAt, time.Hour))
	day := domain.FillMoveBarOpen(domain.FillMoveHorizon(fillAt, 24*time.Hour))
	fake.bars = []domain.Candle{{
		OpenTime: hour, Close: "90", CloseTime: hour, // Yahoo stamps close time at the open.
	}}
	svc.moveClock = func() time.Time { return fillAt.Add(30 * time.Hour) }
	list, _, err = svc.ListTrades(ctx, "mv4", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if fake.nilStarts != 1 || fake.ranged != 0 {
		t.Fatalf("nil %d ranged %d", fake.nilStarts, fake.ranged)
	}
	if list[0].Moves[0].Status != domain.FillMoveReady || list[0].Moves[0].Pct == nil || math.Abs(*list[0].Moves[0].Pct-(-10)) > 1e-9 {
		t.Fatalf("1h %+v", list[0].Moves[0])
	}
	if list[0].Moves[2].Status != domain.FillMoveMissing || list[0].Moves[2].Pct != nil {
		t.Fatalf("old 24h should stay blank %+v minute %s", list[0].Moves[2], day)
	}
	view, err := svc.View(ctx, "mv4")
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Positions) != 1 || view.Positions[0].Moves[0].Status != domain.FillMoveReady {
		t.Fatalf("equity position %+v", view.Positions)
	}
	if fake.nilStarts != 1 {
		t.Fatalf("position refetched equity candles, calls %d", fake.nilStarts)
	}
}

func TestFillMoves_CoarseBarIsMissing(t *testing.T) {
	svc := newSvc(t, nil)
	fillAt := time.Date(2026, 5, 2, 9, 0, 0, 0, time.UTC)
	hour := domain.FillMoveBarOpen(domain.FillMoveHorizon(fillAt, time.Hour))
	fake := &moveCandles{rejectRange: true, bars: []domain.Candle{{
		OpenTime: hour, Close: "150", CloseTime: hour.Add(time.Hour),
	}}}
	svc.SetCandleSource(fake)
	svc.moveClock = func() time.Time { return fillAt.Add(2 * time.Hour) }
	moves := svc.fillMoves(context.Background(), []moveAnchor{{
		exchange: domain.ExchangeNasdaq, symbol: "AAPL", price: 100, at: fillAt,
	}})
	if moves[0][0].Status != domain.FillMoveMissing || moves[0][0].Pct != nil {
		t.Fatalf("coarse %+v", moves[0][0])
	}
}

func TestMoveQuoteCache_ExpiryDoesNotEvictAFreshMinute(t *testing.T) {
	c := newMoveQuoteCache()
	base := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	for i := 0; i < moveQuoteMax; i++ {
		c.set(strconv.Itoa(i), moveQuote{price: 1, found: true, expires: base.Add(time.Second)})
	}
	later := base.Add(2 * time.Second)
	for i := 0; i < moveQuoteMax; i++ {
		if _, ok := c.get(strconv.Itoa(i), later); ok {
			t.Fatalf("key %d still cached after expiry", i)
		}
	}
	if len(c.items) != 0 {
		t.Fatalf("expired cache size=%d", len(c.items))
	}
	for i := 0; i < moveQuoteMax+5; i++ {
		c.set("n"+strconv.Itoa(i), moveQuote{
			price:   float64(i + 1),
			found:   true,
			expires: later.Add(time.Duration(i+1) * time.Second),
		})
	}
	if len(c.items) != moveQuoteMax {
		t.Fatalf("size=%d want %d", len(c.items), moveQuoteMax)
	}
	last := "n" + strconv.Itoa(moveQuoteMax+4)
	q, ok := c.get(last, later)
	if !ok || q.price != float64(moveQuoteMax+5) {
		t.Fatalf("fresh minute evicted: ok=%v price=%v", ok, q.price)
	}
	if _, ok := c.get("n0", later); ok {
		t.Fatal("oldest minute survived a full cache")
	}
}
