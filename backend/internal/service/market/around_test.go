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
		to:   domain.BookHistorySnapshot{Symbol: "BTCUSDT", Exchange: domain.ExchangeBinance, SampledAt: at.Add(14 * time.Minute), Mid: 107, BidNotional: 2000, AskNotional: 6000},
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

func TestGetAround_AttachesOptions(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Minute)
	at := now.Add(-2 * time.Hour)
	shift := at.Sub(time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC))
	tape := aroundTape(time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC))
	for i := range tape {
		tape[i].OpenTime = tape[i].OpenTime.Add(shift)
		tape[i].CloseTime = tape[i].CloseTime.Add(shift)
	}
	svc := New(&fakeMarket{candles: tape, ticker: &domain.Ticker24h{LastPrice: "105"}}, &fakeSupply{})
	svc.WithOptionsHistory(stubAroundOptions{samples: []domain.OptionsSample{
		{Asset: "BTC", SampledAt: at.Add(-time.Hour), OpenInterest: 100, PutCallOI: 0.9, HasPutCallOI: true, DVOL: 40, HasDVOL: true},
		{Asset: "BTC", SampledAt: at.Add(-5 * time.Minute), OpenInterest: 120, PutCallOI: 1.25, HasPutCallOI: true, DVOL: 45, HasDVOL: true},
	}})
	got, err := svc.GetAround(context.Background(), "binance", "BTCUSDT", "1h", "15m", at)
	if err != nil {
		t.Fatal(err)
	}
	var before *domain.AroundPhase
	for i := range got.Venues[0].Phases {
		if got.Venues[0].Phases[i].Phase == domain.AroundPhaseBefore {
			before = &got.Venues[0].Phases[i]
		}
	}
	if before == nil || before.Options == nil || !before.Options.Complete {
		t.Fatalf("options %+v", before)
	}
	if before.Options.OIFrom != 100 || before.Options.OITo != 120 {
		t.Fatalf("oi %+v", before.Options)
	}
	if !before.Options.Unusual {
		t.Fatalf("expected unusual options %+v", before.Options)
	}
	if before.Options.Started != domain.AroundOptionsStartedBefore {
		t.Fatalf("started %q", before.Options.Started)
	}
}

func TestGetAround_AttachesCurve(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Hour)
	at := now.Add(-2 * time.Hour)
	tape := aroundTape(at)
	bn := &fakeMarket{candles: tape, ticker: &domain.Ticker24h{LastPrice: "105"}}
	nearExp := at.Add(40 * 24 * time.Hour)
	farExp := at.Add(130 * 24 * time.Hour)
	histNear := []domain.PriceSample{{Time: at.Add(-time.Hour), Price: 64000}, {Time: at, Price: 64000}}
	histFar := []domain.PriceSample{{Time: at.Add(-time.Hour), Price: 64600}, {Time: at, Price: 64700}}
	bin := &fakeBasis{q: &domain.BasisQuote{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Time: now,
		FuturesMark: 64000, SpotIndex: 64000,
		Contracts: []domain.BasisContractQuote{
			{Symbol: "BTCUSDT", Kind: domain.BasisContractPerp, Mark: 64000, Index: 64000},
			{Symbol: "BTCUSDT_260925", Kind: domain.BasisContractCurrentQuarter, Expiry: nearExp, Mark: 64000, Index: 64000, History: histNear},
			{Symbol: "BTCUSDT_261225", Kind: domain.BasisContractNextQuarter, Expiry: farExp, Mark: 64700, Index: 64000, History: histFar},
		},
	}}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: bn,
	}, &fakeSupply{}).WithBasis(map[domain.Exchange]domain.BasisPort{
		domain.ExchangeBinance: bin,
	})
	got, err := svc.GetAround(context.Background(), "binance", "BTCUSDT", "1h", "15m", at)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Venues) == 0 {
		t.Fatal("no venues")
	}
	var before *domain.AroundPhase
	for i := range got.Venues[0].Phases {
		if got.Venues[0].Phases[i].Phase == domain.AroundPhaseBefore {
			before = &got.Venues[0].Phases[i]
		}
	}
	if before == nil || before.Curve == nil || !before.Curve.Complete {
		t.Fatalf("curve %+v", before)
	}
	if before.Curve.Shape != domain.BasisCurveContango {
		t.Fatalf("shape %s", before.Curve.Shape)
	}
}

func TestGetAround_OneSidedOptionsNotAChange(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Minute)
	at := now.Add(-2 * time.Hour)
	shift := at.Sub(time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC))
	tape := aroundTape(time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC))
	for i := range tape {
		tape[i].OpenTime = tape[i].OpenTime.Add(shift)
		tape[i].CloseTime = tape[i].CloseTime.Add(shift)
	}
	svc := New(&fakeMarket{candles: tape, ticker: &domain.Ticker24h{LastPrice: "105"}}, &fakeSupply{})
	svc.WithOptionsHistory(stubAroundOptions{samples: []domain.OptionsSample{
		{Asset: "BTC", SampledAt: at.Add(-2 * time.Minute), OpenInterest: 120, PutCallOI: 1.3, HasPutCallOI: true},
	}})
	got, err := svc.GetAround(context.Background(), "binance", "BTCUSDT", "1h", "15m", at)
	if err != nil {
		t.Fatal(err)
	}
	for _, ph := range got.Venues[0].Phases {
		if ph.Options != nil && ph.Options.HasOI {
			t.Fatalf("one sample must not invent an OI change %+v", ph.Options)
		}
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
	got, err := svc.GetAroundSimilar(context.Background(), "binance", "BTCUSDT", "24h", "15m", "both", 1.5, 5, "1h", "15m", "volume,book,oi", "book:3,oi:3,volume:1", "60", "30m,2h", time.Time{})
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
	if len(got.Horizons) != 2 || got.Horizons[0] != "30m" || got.Horizons[1] != "2h" {
		t.Fatalf("horizons %+v", got.Horizons)
	}
	if len(got.Matches) > 0 && len(got.AfterHorizons) != 2 {
		t.Fatalf("afterHorizons %+v", got.AfterHorizons)
	}
	if got.Events < len(got.Matches) {
		t.Fatalf("events %d matches %d", got.Events, len(got.Matches))
	}
}

func TestGetAroundSimilar_BadWeights(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	if _, err := svc.GetAroundSimilar(context.Background(), "binance", "BTCUSDT", "24h", "15m", "both", 1.5, 5, "1h", "15m", "volume,book", "price:2", "60", "", time.Time{}); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
	if _, err := svc.GetAroundSimilar(context.Background(), "binance", "BTCUSDT", "24h", "15m", "both", 1.5, 5, "1h", "15m", "volume,book", "", "101", "", time.Time{}); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
	if _, err := svc.GetAroundSimilar(context.Background(), "binance", "BTCUSDT", "24h", "15m", "both", 1.5, 5, "1h", "15m", "volume,book", "", "60", "rsi", time.Time{}); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}

func TestFindAroundMoves_BadSymbol(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	if _, err := svc.FindAroundMoves(context.Background(), "all", "  ", "24h", "15m", "both", 0, 0, "", ""); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}

func TestGetForensics_ReturnsTape(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Minute)
	at := now.Add(-2 * time.Hour)
	svc := New(&fakeMarket{candles: aroundTape(at), ticker: &domain.Ticker24h{LastPrice: "105"}}, &fakeSupply{})
	got, err := svc.GetForensics(context.Background(), "binance", "BTCUSDT", "1h", "15m", at, "24h", "15m", "both", 1.5, 5, "volume,book,oi", "", "60", "30m,2h", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Tape == nil || got.Symbol != "BTCUSDT" || got.Summary == "" {
		t.Fatalf("%+v", got)
	}
	if got.At.UTC() != at.UTC() {
		t.Fatalf("at %s want %s", got.At, at)
	}
	if got.Note == "" {
		t.Fatal("missing note")
	}
	if got.Why.Headline == "" {
		t.Fatal("missing why")
	}
	if got.Similar == nil && got.SimilarError == "" {
		t.Fatal("expected similar or similarError")
	}
	if got.Similar != nil && got.Similar.At.IsZero() {
		t.Fatal("similar at should be set")
	}
}

func TestGetForensics_AsOfMustBeAtOrAfterEvent(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Minute)
	at := now.Add(-2 * time.Hour)
	svc := New(&fakeMarket{candles: aroundTape(at), ticker: &domain.Ticker24h{LastPrice: "105"}}, &fakeSupply{})
	if _, err := svc.GetForensics(context.Background(), "binance", "BTCUSDT", "1h", "15m", at, "24h", "15m", "both", 1.5, 5, "volume,book,oi", "", "60", "30m,2h", at.Add(-time.Minute)); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
	if _, err := svc.GetForensics(context.Background(), "binance", "BTCUSDT", "1h", "15m", at, "24h", "15m", "both", 1.5, 5, "volume,book,oi", "", "60", "30m,2h", now.Add(time.Hour)); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}

func TestGetForensics_ExplicitAsOfSetsKnownUntil(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Minute)
	at := now.Add(-2 * time.Hour)
	asOf := at.Add(10 * time.Minute)
	svc := New(&fakeMarket{candles: aroundTape(at), ticker: &domain.Ticker24h{LastPrice: "105"}}, &fakeSupply{})
	got, err := svc.GetForensics(context.Background(), "binance", "BTCUSDT", "1h", "15m", at, "24h", "15m", "both", 1.5, 5, "volume,book,oi", "", "60", "30m,2h", asOf)
	if err != nil {
		t.Fatal(err)
	}
	if !got.AsOfExplicit || !got.AsOf.Equal(asOf) {
		t.Fatalf("asOf %+v", got)
	}
	if !got.Timeline.KnownUntil.Equal(asOf) {
		t.Fatalf("knownUntil %s want %s", got.Timeline.KnownUntil, asOf)
	}
}

func TestScanForensics_RanksCurrentSetups(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Minute)
	at := now.Add(-2 * time.Hour)
	var tape []domain.Candle
	for _, hoursAgo := range []int{20, 8, 3} {
		tape = append(tape, aroundTape(now.Add(-time.Duration(hoursAgo)*time.Hour))...)
	}
	tape = append(tape, aroundTape(at)...)
	svc := New(&fakeMarket{candles: tape, ticker: &domain.Ticker24h{LastPrice: "105"}}, &fakeSupply{})
	got, err := svc.ScanForensics(context.Background(), "binance", "USDT", "24h", "15m", "both", 1.5, 4, "1h", "15m", "volume", "", "0", "1h")
	if err != nil {
		t.Fatal(err)
	}
	if got.Scanned == 0 || got.Summary == "" || got.Note == "" {
		t.Fatalf("%+v", got)
	}
	if got.Quote != "USDT" {
		t.Fatalf("quote %s", got.Quote)
	}
}

func TestGetAroundSimilar_UsesSelectedAt(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Minute)
	at := now.Add(-3 * time.Hour)
	svc := New(&fakeMarket{candles: aroundTape(at), ticker: &domain.Ticker24h{LastPrice: "105"}}, &fakeSupply{})
	got, err := svc.GetAroundSimilar(context.Background(), "binance", "BTCUSDT", "24h", "15m", "both", 1.5, 5, "1h", "15m", "volume", "", "0", "", at)
	if err != nil {
		t.Fatal(err)
	}
	if got.At.UTC() != at.UTC() {
		t.Fatalf("at %s want %s", got.At, at)
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
	marked, ok := domain.NearestBookSnapshot([]domain.BookHistorySnapshot{*best}, at, domain.DefaultBookHistorySlack)
	if ok {
		return &marked, nil
	}
	best.Complete = false
	return best, nil
}
func (s stubAroundBook) List(context.Context, domain.BookHistoryQuery) ([]domain.BookHistorySnapshot, error) {
	return []domain.BookHistorySnapshot{s.from, s.to}, nil
}
func (s stubAroundBook) ListLadders(ctx context.Context, q domain.BookHistoryQuery) ([]domain.BookHistorySnapshot, error) {
	return s.List(ctx, q)
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

type stubAroundOptions struct {
	samples []domain.OptionsSample
}

func (s stubAroundOptions) InsertSample(context.Context, domain.OptionsSample) (bool, error) {
	return false, nil
}

func (s stubAroundOptions) ListSamples(_ context.Context, asset string, from, to time.Time, _ int) ([]domain.OptionsSample, error) {
	var out []domain.OptionsSample
	for _, rec := range s.samples {
		if rec.Asset != asset {
			continue
		}
		if rec.SampledAt.Before(from) || rec.SampledAt.After(to) {
			continue
		}
		out = append(out, rec)
	}
	return out, nil
}

func (s stubAroundOptions) PurgeOlderThan(context.Context, time.Time) (int, error) {
	return 0, nil
}

func (stubAroundOptions) ReplaceStrikeBook(context.Context, string, time.Time, []domain.OptionsStrikeOIPoint) error {
	return nil
}

func (stubAroundOptions) StrikeBookAt(context.Context, string, time.Time) (domain.OptionsStrikeBook, bool, error) {
	return domain.OptionsStrikeBook{}, false, nil
}
