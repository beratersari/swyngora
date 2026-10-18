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

type replayCandles struct {
	bySymbol map[string][]domain.Candle
	err      error
	failSym  string
	maxSpan  time.Duration
	ranged   int
	nilRange int
	spans    [][2]time.Time
}

func (f *replayCandles) GetCandles(_ context.Context, _, symbol, _ string, _ int, start, end *time.Time) ([]domain.Candle, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.failSym != "" && symbol == f.failSym {
		return nil, errors.New("venue down")
	}
	if start == nil || end == nil {
		f.nilRange++
		return append([]domain.Candle(nil), f.bySymbol[symbol]...), nil
	}
	if f.maxSpan > 0 && end.Sub(*start) > f.maxSpan {
		return nil, errors.New("chunk wider than the venue allows")
	}
	f.ranged++
	f.spans = append(f.spans, [2]time.Time{start.UTC(), end.UTC()})
	if f.bySymbol != nil {
		return append([]domain.Candle(nil), f.bySymbol[symbol]...), nil
	}
	return []domain.Candle{{
		OpenTime:  start.UTC(),
		Close:     "100",
		CloseTime: start.UTC().Add(24*time.Hour - time.Millisecond),
	}}, nil
}

func TestGroupReplayHoldings_SumsSpotValueAndMarginNotional(t *testing.T) {
	got := groupReplayHoldings(domain.PortfolioView{
		Positions: []domain.PositionView{
			{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Quantity: 1, MarkPrice: 100, MarketValue: 100},
			{Exchange: domain.ExchangeCoinbase, Symbol: "BTC-USD", Quantity: 1, MarkPrice: 50, MarketValue: 50},
		},
		MarginPositions: []domain.MarginPosition{
			{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Status: domain.MarginPositionOpen, Quantity: 2, MarkPrice: 100, Margin: 20},
			{Exchange: domain.ExchangeBinance, Symbol: "ETHUSDT", Status: domain.MarginPositionClosed, Quantity: 9, MarkPrice: 100},
		},
	})
	if len(got) != 2 {
		t.Fatalf("holdings %+v", got)
	}
	if got[0].symbol != "BTCUSDT" || got[0].weight != 300 {
		t.Fatalf("btc weight %+v", got[0])
	}
	if got[1].symbol != "BTC-USD" || got[1].weight != 50 {
		t.Fatalf("coinbase %+v", got[1])
	}
}

func TestStressPeriod_ReplaysDailyCloseOntoOpenBook(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100", "binance|ETHUSDT": "50"}}
	svc := newSvc(t, px)
	fake := &replayCandles{bySymbol: map[string][]domain.Candle{
		"BTCUSDT": {
			replayServiceDay(time.Date(2024, 8, 4, 0, 0, 0, 0, time.UTC), 100),
			replayServiceDay(time.Date(2024, 8, 6, 0, 0, 0, 0, time.UTC), 80),
		},
	}}
	svc.SetCandleSource(fake)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "rp1", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "rp1", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "rp1", Symbol: "ETHUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	rep, err := svc.StressPeriod(ctx, "rp1", "", "", "2024-08-05", "2024-08-06")
	if err != nil {
		t.Fatal(err)
	}
	if rep.Replay == nil || rep.Replay.Interval != "1d" {
		t.Fatalf("replay %+v", rep.Replay)
	}
	if len(rep.Replay.Moves) != 1 || rep.Replay.Moves[0].Asset != "BTC" {
		t.Fatalf("moves %+v", rep.Replay.Moves)
	}
	if math.Abs(rep.Replay.Moves[0].MovePct-(-20)) > 1e-6 {
		t.Fatalf("btc move %v", rep.Replay.Moves[0].MovePct)
	}
	if math.Abs(rep.EquityDelta-(-20)) > 0.01 {
		t.Fatalf("equity delta %v", rep.EquityDelta)
	}
	if len(rep.Replay.Missing) != 1 || rep.Replay.Missing[0].Symbol != "ETHUSDT" || rep.Replay.Missing[0].Reason != domain.ReplayReasonNoHistory {
		t.Fatalf("missing %+v", rep.Replay.Missing)
	}
	if rep.Note != domain.PortfolioReplayNote {
		t.Fatalf("note %q", rep.Note)
	}
	if len(rep.Replay.Points) < 2 {
		t.Fatalf("points %d", len(rep.Replay.Points))
	}
	last := rep.Replay.Points[len(rep.Replay.Points)-1]
	if math.Abs(last.Equity-rep.ShockedEquity) > 0.01 {
		t.Fatalf("path end %v shocked %v", last.Equity, rep.ShockedEquity)
	}
	if math.Abs(rep.Replay.LowEquity-last.Equity) > 0.01 {
		t.Fatalf("low %v end %v", rep.Replay.LowEquity, last.Equity)
	}
	plain, err := svc.Stress(ctx, "rp1", "", "", []domain.StressMove{{Asset: "BTC", MovePct: -10}})
	if err != nil {
		t.Fatal(err)
	}
	if plain.Replay != nil {
		t.Fatal("typed percents should not carry a replay")
	}
}

func TestStressPeriod_EmptyBookAndUnreadHistory(t *testing.T) {
	svc := newSvc(t, nil)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "rp2", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.StressPeriod(ctx, "rp2", "", "", "2024-08-05", "2024-08-06"); !errors.Is(err, errPriceHistoryUnset) {
		t.Fatalf("unset %v", err)
	}
	svc.SetCandleSource(&replayCandles{})
	if _, err := svc.StressPeriod(ctx, "rp2", "", "", "2024-08-05", "2024-08-06"); !errors.Is(err, errNoOpenPosition) {
		t.Fatalf("empty %v", err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "rp2", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	svc.SetCandleSource(&replayCandles{failSym: "BTCUSDT"})
	if _, err := svc.StressPeriod(ctx, "rp2", "", "", "2024-08-05", "2024-08-06"); !errors.Is(err, errPriceHistoryRead) {
		t.Fatalf("read %v", err)
	}
}

func TestLoadReplayBars_ChunksDailyHistory(t *testing.T) {
	fake := &replayCandles{maxSpan: time.Duration(stressReplayChunkBars) * 24 * time.Hour}
	svc := &Service{candles: fake}
	from := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(400 * 24 * time.Hour)
	if _, err := svc.loadReplayBars(context.Background(), domain.ExchangeBinance, "BTCUSDT", "1d", from, to); err != nil {
		t.Fatal(err)
	}
	if len(fake.spans) < 2 {
		t.Fatalf("spans %d", len(fake.spans))
	}
	for _, sp := range fake.spans {
		if sp[1].Sub(sp[0]) > fake.maxSpan {
			t.Fatalf("span %s", sp[1].Sub(sp[0]))
		}
	}
}

func TestLoadReplayBars_EquityOmitsRange(t *testing.T) {
	fake := &replayCandles{}
	svc := &Service{candles: fake}
	from := time.Date(2024, 8, 5, 0, 0, 0, 0, time.UTC)
	to := from.Add(48 * time.Hour)
	if _, err := svc.loadReplayBars(context.Background(), domain.ExchangeNasdaq, "AAPL", "1d", from, to); err != nil {
		t.Fatal(err)
	}
	if fake.nilRange != 1 || fake.ranged != 0 {
		t.Fatalf("nil %d ranged %d", fake.nilRange, fake.ranged)
	}
}

func replayServiceDay(open time.Time, px float64) domain.Candle {
	return domain.Candle{
		OpenTime:  open,
		Close:     strconv.FormatFloat(px, 'f', -1, 64),
		CloseTime: open.Add(24*time.Hour - time.Millisecond),
	}
}
