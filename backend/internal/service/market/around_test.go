package market

import (
	"context"
	"errors"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func aroundCandle(at time.Time, open, high, low, close, vol, buy string) domain.Candle {
	return domain.Candle{
		OpenTime: at, Open: open, High: high, Low: low, Close: close,
		QuoteVolume: vol, TakerBuyQuote: buy, CloseTime: at.Add(time.Minute),
	}
}

func aroundTape(at time.Time) []domain.Candle {
	var out []domain.Candle
	for i := 10; i >= 1; i-- {
		t0 := at.Add(-time.Duration(i) * time.Hour)
		out = append(out, aroundCandle(t0, "100", "101", "99", "100", "1000", "500"))
	}
	out = append(out, aroundCandle(at.Add(-30*time.Minute), "100", "101", "99", "100.2", "1200", "600"))
	out = append(out, aroundCandle(at, "100.2", "108", "100", "107", "9000", "7000"))
	out = append(out, aroundCandle(at.Add(20*time.Minute), "107", "107.5", "104", "105", "2000", "700"))
	return out
}

func TestGetAround_PhasesAndCombine(t *testing.T) {
	at := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	// ResolveAroundPlan rejects at more than 30d in the past vs time.Now, so
	// keep the fixture recent: shift relative to now while preserving shape.
	now := time.Now().UTC().Truncate(time.Minute)
	at = now.Add(-2 * time.Hour)
	shift := at.Sub(time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC))
	tape := aroundTape(time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC))
	for i := range tape {
		tape[i].OpenTime = tape[i].OpenTime.Add(shift)
		tape[i].CloseTime = tape[i].CloseTime.Add(shift)
	}
	bn := &fakeMarket{candles: tape, ticker: &domain.Ticker24h{LastPrice: "105"}}
	by := &fakeMarket{candles: tape, ticker: &domain.Ticker24h{LastPrice: "105"}}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: bn, domain.ExchangeBybit: by,
	}, &fakeSupply{})
	got, err := svc.GetAround(context.Background(), "all", "BTCUSDT", "1h", "15m", at)
	if err != nil {
		t.Fatal(err)
	}
	if got.Window != "1h" || got.During != "15m" || len(got.Venues) != 2 {
		t.Fatalf("report %+v", got)
	}
	if got.Combined == nil || got.Summary == "" {
		t.Fatalf("combined/summary %+v", got)
	}
	var during *domain.AroundPhase
	for i := range got.Venues[0].Phases {
		if got.Venues[0].Phases[i].Phase == domain.AroundPhaseDuring {
			during = &got.Venues[0].Phases[i]
		}
	}
	if during == nil || !during.Complete || during.Price.Close < 106 {
		t.Fatalf("during %+v", during)
	}
	if during.Flow.Volume != 9000 {
		t.Fatalf("during vol %+v", during.Flow)
	}
	if got.Combined == nil {
		t.Fatal("combined")
	}
	var cdur *domain.AroundPhase
	for i := range got.Combined.Phases {
		if got.Combined.Phases[i].Phase == domain.AroundPhaseDuring {
			cdur = &got.Combined.Phases[i]
		}
	}
	if cdur == nil || cdur.Flow.Volume != 18_000 {
		t.Fatalf("combined during %+v", cdur)
	}
}

func TestGetAround_AttachesBookAndFutures(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Minute)
	at := now.Add(-2 * time.Hour)
	shift := at.Sub(time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC))
	tape := aroundTape(time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC))
	for i := range tape {
		tape[i].OpenTime = tape[i].OpenTime.Add(shift)
		tape[i].CloseTime = tape[i].CloseTime.Add(shift)
	}
	svc := New(&fakeMarket{candles: tape, ticker: &domain.Ticker24h{LastPrice: "105"}}, &fakeSupply{})
	svc.WithBookHistory(stubAroundBook{
		from: domain.BookHistorySnapshot{Symbol: "BTCUSDT", Exchange: domain.ExchangeBinance, SampledAt: at.Add(-time.Minute), Mid: 100, BidNotional: 5000, AskNotional: 4000},
		to:   domain.BookHistorySnapshot{Symbol: "BTCUSDT", Exchange: domain.ExchangeBinance, SampledAt: at.Add(10 * time.Minute), Mid: 107, BidNotional: 2000, AskNotional: 6000},
	})
	svc.SetFuturesHistory(stubAroundFut{
		snaps: []domain.FuturesSnapshot{
			{Metric: domain.FuturesMetricOpenInterest, Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", SampledAt: at.Add(-time.Minute), Value: 1000},
			{Metric: domain.FuturesMetricOpenInterest, Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", SampledAt: at.Add(10 * time.Minute), Value: 1200},
		},
		liqs: []domain.LiquidationEvent{
			{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideShort, Notional: 50_000, Time: at.Add(2 * time.Minute)},
		},
	})
	got, err := svc.GetAround(context.Background(), "binance", "BTCUSDT", "1h", "15m", at)
	if err != nil {
		t.Fatal(err)
	}
	var during *domain.AroundPhase
	for i := range got.Venues[0].Phases {
		if got.Venues[0].Phases[i].Phase == domain.AroundPhaseDuring {
			during = &got.Venues[0].Phases[i]
		}
	}
	if during == nil || during.Book == nil || !during.Book.Complete {
		t.Fatalf("book %+v", during)
	}
	if during.Futures == nil || !during.Futures.Complete || during.Futures.OITo != 1200 {
		t.Fatalf("futures %+v", during.Futures)
	}
	if during.Futures.ShortLiq != 50_000 {
		t.Fatalf("liqs %+v", during.Futures)
	}
}

func TestGetAround_BadSymbol(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	if _, err := svc.GetAround(context.Background(), "all", "  ", "1h", "15m", time.Now().UTC().Add(-time.Hour)); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}

func TestGetAround_RequiresAt(t *testing.T) {
	svc := New(&fakeMarket{candles: aroundTape(time.Now().UTC())}, &fakeSupply{})
	if _, err := svc.GetAround(context.Background(), "binance", "BTCUSDT", "1h", "15m", time.Time{}); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}

func aroundQuietTape(at time.Time) []domain.Candle {
	var out []domain.Candle
	for i := 10; i >= 1; i-- {
		t0 := at.Add(-time.Duration(i) * time.Hour)
		out = append(out, aroundCandle(t0, "100", "100.4", "99.8", "100", "1000", "500"))
	}
	out = append(out, aroundCandle(at.Add(-30*time.Minute), "100", "100.3", "99.9", "100.1", "800", "400"))
	out = append(out, aroundCandle(at, "100.1", "101", "100", "100.7", "1500", "800"))
	out = append(out, aroundCandle(at.Add(20*time.Minute), "100.7", "100.8", "100.4", "100.5", "700", "300"))
	return out
}

func TestCompareAround_SecondMoveIsLarger(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Minute)
	quiet := now.Add(-6 * time.Hour)
	loud := now.Add(-2 * time.Hour)
	tape := append(aroundQuietTape(quiet), aroundTape(loud)...)
	svc := New(&fakeMarket{candles: tape, ticker: &domain.Ticker24h{LastPrice: "105"}}, &fakeSupply{})
	got, err := svc.CompareAround(context.Background(), "binance", "BTCUSDT", "1h", "15m", quiet, loud)
	if err != nil {
		t.Fatal(err)
	}
	if got.FromAt.Unix() != quiet.Unix() || got.ToAt.Unix() != loud.Unix() {
		t.Fatalf("times %+v", got)
	}
	if got.FromMove == nil || got.ToMove == nil || len(got.Venues) == 0 {
		t.Fatalf("moves %+v", got)
	}
	var during *domain.AroundComparePhase
	for i := range got.Venues[0].Phases {
		if got.Venues[0].Phases[i].Phase == domain.AroundPhaseDuring {
			during = &got.Venues[0].Phases[i]
		}
	}
	if during == nil {
		t.Fatal("during")
	}
	var vol domain.AroundCompareDelta
	for _, d := range during.Deltas {
		if d.Metric == domain.AroundCompareMetricVolume {
			vol = d
		}
	}
	if vol.To <= vol.From {
		t.Fatalf("expected louder second move %+v", during.Deltas)
	}
	if got.Summary == "" {
		t.Fatal("summary")
	}
}

func TestFindAroundMoves_AttachesTape(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Minute)
	at := now.Add(-2 * time.Hour)
	svc := New(&fakeMarket{candles: aroundTape(at), ticker: &domain.Ticker24h{LastPrice: "105"}}, &fakeSupply{})
	got, err := svc.FindAroundMoves(context.Background(), "binance", "BTCUSDT", "24h", "15m", "both", 1.5, 8, "1h", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Moves) == 0 {
		t.Fatal("expected at least one important move")
	}
	if got.Moves[0].ReturnPct < 3 {
		t.Fatalf("largest %+v", got.Moves[0])
	}
	if got.Moves[0].Around == nil || got.Summary == "" {
		t.Fatalf("around/summary %+v", got.Moves[0])
	}
}

func TestGetAroundPrecursors_UsesBeforeWindows(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Minute)
	var tape []domain.Candle
	for _, hoursAgo := range []int{20, 14, 8, 3} {
		at := now.Add(-time.Duration(hoursAgo) * time.Hour)
		tape = append(tape, aroundTape(at)...)
	}
	svc := New(&fakeMarket{candles: tape, ticker: &domain.Ticker24h{LastPrice: "105"}}, &fakeSupply{})
	got, err := svc.GetAroundPrecursors(context.Background(), "binance", "BTCUSDT", "24h", "15m", "both", 1.5, 8, "1h", "15m")
	if err != nil {
		t.Fatal(err)
	}
	if got.Sampled == 0 || got.Summary == "" {
		t.Fatalf("precursors %+v", got)
	}
	if got.Lookback != "24h" {
		t.Fatalf("lookback %s", got.Lookback)
	}
}

func TestGetAroundSimilar_ReturnsReport(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Minute)
	var tape []domain.Candle
	for _, hoursAgo := range []int{20, 8, 3} {
		tape = append(tape, aroundTape(now.Add(-time.Duration(hoursAgo)*time.Hour))...)
	}
	svc := New(&fakeMarket{candles: tape, ticker: &domain.Ticker24h{LastPrice: "105"}}, &fakeSupply{})
	got, err := svc.GetAroundSimilar(context.Background(), "binance", "BTCUSDT", "24h", "15m", "both", 1.5, 5, "1h", "15m", "volume,book,oi", "book:3,oi:3,volume:1", "60")
	if err != nil {
		t.Fatal(err)
	}
	if got.Summary == "" || got.Current.Phase == "" && !got.Current.Complete {
		// current may be complete from last hour of tape
		if got.Note == "" {
			t.Fatalf("%+v", got)
		}
	}
	if got.MinCoverage != 60 {
		t.Fatalf("minCoverage %v", got.MinCoverage)
	}
	if len(got.Fields) != 3 {
		t.Fatalf("fields %+v", got.Fields)
	}
	var bookW, volW float64
	for _, w := range got.Weights {
		switch w.Name {
		case domain.AroundSimilarFieldBook:
			bookW = w.Weight
		case domain.AroundSimilarFieldVolume:
			volW = w.Weight
		}
	}
	if bookW != 3 || volW != 1 {
		t.Fatalf("weights %+v", got.Weights)
	}
	if len(got.Matches) > 0 && len(got.AfterHorizons) != 3 {
		t.Fatalf("afterHorizons %+v", got.AfterHorizons)
	}
	if got.Events != len(got.Matches) {
		t.Fatalf("events %d matches %d", got.Events, len(got.Matches))
	}
}

func TestGetAroundSimilar_BadWeights(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	if _, err := svc.GetAroundSimilar(context.Background(), "binance", "BTCUSDT", "24h", "15m", "both", 1.5, 5, "1h", "15m", "volume,book", "price:2", "60"); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
	if _, err := svc.GetAroundSimilar(context.Background(), "binance", "BTCUSDT", "24h", "15m", "both", 1.5, 5, "1h", "15m", "volume,book", "", "101"); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}

func TestFindAroundMoves_BadSymbol(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	if _, err := svc.FindAroundMoves(context.Background(), "all", "  ", "24h", "15m", "both", 0, 0, "", ""); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}

func TestCompareAround_SameTimeRejected(t *testing.T) {
	at := time.Now().UTC().Add(-2 * time.Hour)
	svc := New(&fakeMarket{candles: aroundTape(at)}, &fakeSupply{})
	if _, err := svc.CompareAround(context.Background(), "binance", "BTCUSDT", "1h", "15m", at, at); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}

type stubAroundBook struct {
	from, to domain.BookHistorySnapshot
}

func (s stubAroundBook) SnapshotAt(_ context.Context, _, _ string, at time.Time) (*domain.BookHistorySnapshot, error) {
	var best *domain.BookHistorySnapshot
	for _, row := range []domain.BookHistorySnapshot{s.from, s.to} {
		if row.SampledAt.IsZero() || row.SampledAt.After(at) {
			continue
		}
		cp := row
		if best == nil || cp.SampledAt.After(best.SampledAt) {
			best = &cp
		}
	}
	if best == nil {
		return nil, domain.ErrNotFound
	}
	return best, nil
}
func (s stubAroundBook) List(context.Context, domain.BookHistoryQuery) ([]domain.BookHistorySnapshot, error) {
	return []domain.BookHistorySnapshot{s.from, s.to}, nil
}
func (s stubAroundBook) Compare(_ context.Context, _, _ string, _, _ time.Time) (*domain.BookHistoryDiff, error) {
	d := domain.CompareBookHistory(s.from, s.to)
	return &d, nil
}
func (stubAroundBook) Note(string, string) {}

type stubAroundFut struct {
	snaps []domain.FuturesSnapshot
	liqs  []domain.LiquidationEvent
}

func (s stubAroundFut) History(_ context.Context, q domain.FuturesHistoryQuery) (any, error) {
	if q.Metric == "liquidations" {
		return s.liqs, nil
	}
	var out []domain.FuturesSnapshot
	for _, r := range s.snaps {
		if r.Metric == q.Metric {
			out = append(out, r)
		}
	}
	return out, nil
}
