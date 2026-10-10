package domain

import (
	"math"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestTapeBacktestCVDCrossScoresForwardReturns(t *testing.T) {
	day := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	hour1 := day.Add(10 * time.Hour)
	hour2 := day.Add(11 * time.Hour)
	now := day.Add(48 * time.Hour)
	buckets := append(fillTapeHour(ExchangeBinance, hour1, 50, 40, 60), fillTapeHour(ExchangeBinance, hour2, 50, 70, 30)...)
	q := tapeQuery(t, "binance", TapeConditionCVD, TapeSideBuy, 0.65, hour1, now, now)
	got, err := BuildTapeBacktest(TapeBacktestInput{
		Query: q, Buckets: buckets, Candles: tapeCandles(hour2.Add(time.Hour), 100, 110, 90, 120),
		CandleStep: 5 * time.Minute, PriceExchange: "binance",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Definite || got.Combined != nil {
		t.Fatalf("single venue definite=%v combined=%v", got.Definite, got.Combined)
	}
	if len(got.Venues) != 1 || got.Venues[0].SignalCount != 1 {
		t.Fatalf("%+v", got.Venues)
	}
	sig := got.Venues[0].Signals[0]
	if !sig.At.Equal(hour2.Add(time.Hour)) || math.Abs(sig.Value-0.7) > 0.0001 || sig.Close != 100 {
		t.Fatalf("%+v", sig)
	}
	if sig.Return1h == nil || *sig.Return1h != 10 || sig.Return4h == nil || *sig.Return4h != -10 || sig.Return24h == nil || *sig.Return24h != 20 {
		t.Fatalf("returns %+v", sig)
	}
	h := got.Venues[0].Horizons
	if h[0].SampleCount != 1 || h[0].Average == nil || *h[0].Average != 10 || h[0].WinRate == nil || *h[0].WinRate != 100 {
		t.Fatalf("1h %+v", h[0])
	}
	if h[1].WinRate == nil || *h[1].WinRate != 0 {
		t.Fatalf("4h %+v", h[1])
	}
}

func TestTapeBacktestCVDDoesNotRepeatOrSkipAGap(t *testing.T) {
	day := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)
	now := day.Add(48 * time.Hour)
	stay := append(fillTapeHour(ExchangeBinance, day.Add(10*time.Hour), 50, 70, 30), fillTapeHour(ExchangeBinance, day.Add(11*time.Hour), 50, 80, 20)...)
	q := tapeQuery(t, "binance", TapeConditionCVD, "", 0, day, now, now)
	got, err := BuildTapeBacktest(TapeBacktestInput{Query: q, Buckets: stay, CandleStep: 5 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if got.Venues[0].SignalCount != 0 {
		t.Fatalf("stay above fired %d", got.Venues[0].SignalCount)
	}
	gapped := append(fillTapeHour(ExchangeBinance, day.Add(10*time.Hour), 50, 40, 60), fillTapeHour(ExchangeBinance, day.Add(12*time.Hour), 50, 80, 20)...)
	got, err = BuildTapeBacktest(TapeBacktestInput{Query: q, Buckets: gapped, CandleStep: 5 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if got.Venues[0].SignalCount != 0 {
		t.Fatalf("gap fired %d", got.Venues[0].SignalCount)
	}
	short := fillTapeHour(ExchangeBinance, day.Add(10*time.Hour), 40, 40, 60)
	short = append(short, fillTapeHour(ExchangeBinance, day.Add(11*time.Hour), 50, 80, 20)...)
	got, err = BuildTapeBacktest(TapeBacktestInput{Query: q, Buckets: short, CandleStep: 5 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if got.Venues[0].SignalCount != 0 || got.Venues[0].Bars != 1 {
		t.Fatalf("incomplete previous %+v", got.Venues[0])
	}
}

func TestTapeBacktestCombinedCVDNeedsEveryVenue(t *testing.T) {
	day := time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)
	now := day.Add(48 * time.Hour)
	var rows []TakerBucket
	for _, ex := range []Exchange{ExchangeBinance, ExchangeBybit, ExchangeOKX} {
		rows = append(rows, fillTapeHour(ex, day.Add(10*time.Hour), 50, 30, 70)...)
		rows = append(rows, fillTapeHour(ex, day.Add(11*time.Hour), 50, 80, 20)...)
	}
	q := tapeQuery(t, "all", TapeConditionCVD, TapeSideBuy, 65, day, now, now)
	if math.Abs(q.Threshold-0.65) > 0.0001 {
		t.Fatalf("percent threshold %v", q.Threshold)
	}
	got, err := BuildTapeBacktest(TapeBacktestInput{Query: q, Buckets: rows, CandleStep: 5 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Definite || got.Combined == nil || got.Combined.SignalCount != 1 {
		t.Fatalf("definite=%v combined=%+v missing=%v", got.Definite, got.Combined, got.Missing)
	}
	if math.Abs(got.Combined.Signals[0].Value-0.8) > 0.0001 {
		t.Fatalf("combined share %+v", got.Combined.Signals[0])
	}
	kept := rows[:0]
	for _, b := range rows {
		if b.Exchange != ExchangeOKX {
			kept = append(kept, b)
		}
	}
	got, err = BuildTapeBacktest(TapeBacktestInput{Query: q, Buckets: kept, CandleStep: 5 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if got.Definite || got.Combined != nil || len(got.Missing) != 1 || got.Missing[0] != "okx" {
		t.Fatalf("withheld %+v missing=%v", got.Combined, got.Missing)
	}
	var bin int
	for _, v := range got.Venues {
		if v.Exchange == "binance" {
			bin = v.SignalCount
		}
	}
	if bin != 1 {
		t.Fatalf("binance still scores alone, got %d", bin)
	}
}

func TestTapeBacktestLiquidationBurstIsACrossOfPrints(t *testing.T) {
	start := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	now := start.Add(48 * time.Hour)
	var events []LiquidationEvent
	for i := 0; i < 5; i++ {
		n := 100.0
		if i == 4 {
			n = 1000
		}
		events = append(events, LiquidationEvent{
			Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: LiquidationSideLong,
			Notional: n, Time: start.Add(time.Duration(i) * 5 * time.Minute),
		})
	}
	events = append(events, LiquidationEvent{
		Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: LiquidationSideShort,
		Notional: 9000, Time: start.Add(4 * 5 * time.Minute),
	})
	q := tapeQuery(t, "binance", TapeConditionLiquidation, TapeSideLong, 4, start, now, now)
	signalAt := start.Add(4 * 5 * time.Minute).Add(TapeLiqBucket)
	got, err := BuildTapeBacktest(TapeBacktestInput{
		Query: q, Liquidations: events,
		Candles: []Candle{{
			OpenTime: signalAt.Add(-5 * time.Minute), Close: "100",
		}, {
			OpenTime: signalAt.Add(time.Hour).Add(-5 * time.Minute), Close: "80",
		}},
		CandleStep: 5 * time.Minute, PriceExchange: "binance",
	})
	if err != nil {
		t.Fatal(err)
	}
	ven := got.Venues[0]
	if ven.SignalCount != 1 || math.Abs(ven.Signals[0].Value-10) > 0.01 {
		t.Fatalf("%+v", ven)
	}
	if ven.Signals[0].Return1h == nil || *ven.Signals[0].Return1h != -20 {
		t.Fatalf("return %+v", ven.Signals[0])
	}
	if !strings.Contains(got.Note, "not an estimated leverage level") {
		t.Fatalf("note %s", got.Note)
	}
	thin := events[:4]
	got, err = BuildTapeBacktest(TapeBacktestInput{Query: q, Liquidations: thin, CandleStep: 5 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if got.Venues[0].SignalCount != 0 || got.Venues[0].Note == "" {
		t.Fatalf("thin baseline %+v", got.Venues[0])
	}
}

func TestTapeBacktestBookCrossUsesThePreviousBlock(t *testing.T) {
	start := time.Date(2026, 10, 5, 8, 0, 0, 0, time.UTC)
	now := start.Add(6 * time.Hour)
	snaps := []BookHistorySnapshot{
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", SampledAt: start.Add(time.Minute), BidNotional: 40, AskNotional: 60},
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", SampledAt: start.Add(15 * time.Minute).Add(2 * time.Minute), BidNotional: 80, AskNotional: 20},
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", SampledAt: start.Add(45 * time.Minute), BidNotional: 90, AskNotional: 10},
	}
	q := tapeQuery(t, "binance", TapeConditionBook, TapeSideBid, 0.65, start, now, now)
	got, err := BuildTapeBacktest(TapeBacktestInput{Query: q, Books: snaps, CandleStep: 5 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if got.Venues[0].SignalCount != 1 || !got.Venues[0].Signals[0].At.Equal(start.Add(30*time.Minute)) {
		t.Fatalf("%+v", got.Venues[0])
	}
	if _, err := ParseTapeBacktestQuery("okx", "BTCUSDT", TapeConditionBook, "", 0, time.Time{}, time.Time{}, now); err == nil {
		t.Fatal("okx spot book should be rejected")
	}
}

func TestTapeBacktestWorstDropIsTheLowestCloseOnThePath(t *testing.T) {
	day := time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC)
	hour1 := day.Add(10 * time.Hour)
	hour2 := day.Add(11 * time.Hour)
	now := day.Add(48 * time.Hour)
	signalAt := hour2.Add(time.Hour)
	buckets := append(fillTapeHour(ExchangeBinance, hour1, 50, 40, 60), fillTapeHour(ExchangeBinance, hour2, 50, 70, 30)...)
	step := 5 * time.Minute
	candles := tapeCandles(signalAt, 100, 110, 90, 120)
	candles = append(candles, Candle{OpenTime: signalAt.Add(30 * time.Minute).Add(-step), Close: "80"})
	q := tapeQuery(t, "binance", TapeConditionCVD, TapeSideBuy, 0.65, hour1, now, now)
	got, err := BuildTapeBacktest(TapeBacktestInput{
		Query: q, Buckets: buckets, Candles: candles, CandleStep: step, PriceExchange: "binance",
	})
	if err != nil {
		t.Fatal(err)
	}
	sig := got.Venues[0].Signals[0]
	if sig.Return1h == nil || *sig.Return1h != 10 || sig.Drop1h == nil || *sig.Drop1h != -20 {
		t.Fatalf("path %+v", sig)
	}
	if got.Venues[0].Horizons[0].WorstDrop == nil || *got.Venues[0].Horizons[0].WorstDrop != -20 {
		t.Fatalf("horizon %+v", got.Venues[0].Horizons[0])
	}
	if got.Venues[0].Horizons[1].WorstDrop == nil || *got.Venues[0].Horizons[1].WorstDrop != -20 {
		t.Fatalf("4h drop %+v", got.Venues[0].Horizons[1])
	}
}

func TestTapeBacktestFundingCrossUsesSettledPrints(t *testing.T) {
	day := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)
	now := day.Add(48 * time.Hour)
	first := day.Add(8 * time.Hour)
	second := day.Add(16 * time.Hour)
	q := tapeQuery(t, "binance", TapeConditionFunding, TapeSideLong, 0.01, day, now, now)
	rows := []FuturesSnapshot{
		{Exchange: ExchangeBinance, SampledAt: first, Predicted: true, FundingRate: 0.0003},
		{Exchange: ExchangeBinance, SampledAt: first, FundingRate: 0.00005},
		{Exchange: ExchangeBinance, SampledAt: second, FundingRate: 0.0002},
	}
	got, err := BuildTapeBacktest(TapeBacktestInput{
		Query: q, Snapshots: rows, Candles: tapeCandles(second, 100, 101, 102, 103),
		CandleStep: 5 * time.Minute, PriceExchange: "binance",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Venues[0].SignalCount != 1 || math.Abs(got.Venues[0].Signals[0].Value-0.02) > 0.0001 {
		t.Fatalf("%+v", got.Venues[0])
	}
	if !got.Venues[0].Signals[0].At.Equal(second) {
		t.Fatalf("at %s", got.Venues[0].Signals[0].At)
	}
}

func TestTapeBacktestOpenInterestNeedsBothSamples(t *testing.T) {
	day := time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC)
	now := day.Add(72 * time.Hour)
	base := day.Add(10 * time.Hour)
	q := tapeQuery(t, "binance", TapeConditionOpenInterest, TapeSideUp, 5, day, now, now)
	rows := []FuturesSnapshot{
		{Exchange: ExchangeBinance, SampledAt: base, Value: 100},
		{Exchange: ExchangeBinance, SampledAt: base.Add(time.Hour), Value: 100},
		{Exchange: ExchangeBinance, SampledAt: base.Add(24 * time.Hour), Value: 102},
		{Exchange: ExchangeBinance, SampledAt: base.Add(25 * time.Hour), Value: 110},
	}
	got, err := BuildTapeBacktest(TapeBacktestInput{Query: q, Snapshots: rows, CandleStep: 5 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if got.Venues[0].SignalCount != 1 || math.Abs(got.Venues[0].Signals[0].Value-10) > 0.01 {
		t.Fatalf("%+v", got.Venues[0])
	}
	if got.Venues[0].Bars < 2 {
		t.Fatalf("bars %d", got.Venues[0].Bars)
	}
}

func TestTapeBacktestGapLeavesBybitOut(t *testing.T) {
	day := time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC)
	now := day.Add(48 * time.Hour)
	if _, err := ParseTapeBacktestQuery("bybit", "BTCUSDT", TapeConditionGap, "", 0, day, now, now); err == nil {
		t.Fatal("expected bybit to be rejected")
	}
	q := tapeQuery(t, "all", TapeConditionGap, TapeSideTop, 3, day, now, now)
	if len(q.Expected) != 2 {
		t.Fatalf("expected binance and okx, got %v", q.Expected)
	}
	hour1 := day.Add(10 * time.Hour)
	hour2 := hour1.Add(time.Hour)
	rows := []FuturesSnapshot{
		{Exchange: ExchangeBinance, SampledAt: hour1.Add(30 * time.Minute), LongShare: 0.50, SizeLongShare: 0.52},
		{Exchange: ExchangeBinance, SampledAt: hour2.Add(30 * time.Minute), LongShare: 0.50, SizeLongShare: 0.56},
		{Exchange: ExchangeOKX, SampledAt: hour1.Add(20 * time.Minute), LongShare: 0.50, TopAccLongShare: 0.51},
		{Exchange: ExchangeOKX, SampledAt: hour2.Add(20 * time.Minute), LongShare: 0.50, TopAccLongShare: 0.55},
		{Exchange: ExchangeBybit, SampledAt: hour2, LongShare: 0.60},
	}
	got, err := BuildTapeBacktest(TapeBacktestInput{Query: q, Snapshots: rows, CandleStep: 5 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Definite || got.Combined == nil || got.Combined.SignalCount != 1 {
		t.Fatalf("definite %v combined %+v missing %v", got.Definite, got.Combined, got.Missing)
	}
	if math.Abs(got.Combined.Signals[0].Value-5.5) > 0.01 {
		t.Fatalf("combined gap %v", got.Combined.Signals[0].Value)
	}
}

func TestTapeBacktestSizeUsesKnownPrintsOnly(t *testing.T) {
	day := time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)
	now := day.Add(48 * time.Hour)
	hour1 := day.Add(10 * time.Hour)
	hour2 := hour1.Add(time.Hour)
	q := tapeQuery(t, "binance", TapeConditionSize, TapeSideWhale, 0.40, hour1, now, now)
	rows := append(fillTapeSizeHour(ExchangeBinance, hour1, 50, 20, 80), fillTapeSizeHour(ExchangeBinance, hour2, 50, 50, 50)...)
	rows = append(rows, TakerBucket{
		Exchange: ExchangeBinance, Symbol: "BTCUSDT", Start: hour2.Add(10 * time.Minute),
		BuyNotional: 1_000_000, SizeKnown: false,
	})
	got, err := BuildTapeBacktest(TapeBacktestInput{Query: q, Buckets: rows, CandleStep: 5 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if got.Venues[0].SignalCount != 1 || math.Abs(got.Venues[0].Signals[0].Value-0.5) > 0.0001 {
		t.Fatalf("%+v", got.Venues[0])
	}
	shortQ := tapeQuery(t, "binance", TapeConditionSize, TapeSideWhale, 0, hour1, now, now)
	shortRows := fillTapeSizeHour(ExchangeBinance, hour1, 49, 80, 20)
	short, err := BuildTapeBacktest(TapeBacktestInput{Query: shortQ, Buckets: shortRows, CandleStep: 5 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if short.Venues[0].Bars != 0 || short.Venues[0].SignalCount != 0 {
		t.Fatalf("short hour %+v", short.Venues[0])
	}
}

func TestTapeBacktestRejectsARangePastThirtyDays(t *testing.T) {
	now := time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC)
	if _, err := ParseTapeBacktestQuery("all", "BTCUSDT", TapeConditionCVD, TapeSideBuy, 0, now.Add(-40*24*time.Hour), now, now); err == nil {
		t.Fatal("expected the 30 day cap")
	}
	if _, err := ParseTapeBacktestQuery("all", "BTCUSDT", TapeConditionCVD, TapeSideBuy, 0.4, now.Add(-24*time.Hour), now, now); err == nil {
		t.Fatal("expected the share band")
	}
	if _, err := ParseTapeBacktestQuery("all", " ", TapeConditionCVD, "", 0, time.Time{}, time.Time{}, now); err == nil {
		t.Fatal("expected symbol error")
	}
}

func tapeQuery(t *testing.T, exchange, condition, side string, threshold float64, from, to, now time.Time) TapeBacktestQuery {
	t.Helper()
	q, err := ParseTapeBacktestQuery(exchange, "BTCUSDT", condition, side, threshold, from, to, now)
	if err != nil {
		t.Fatal(err)
	}
	return q
}

func fillTapeSizeHour(ex Exchange, start time.Time, minutes int, whale, retail float64) []TakerBucket {
	out := make([]TakerBucket, 0, minutes)
	for i := 0; i < minutes; i++ {
		out = append(out, TakerBucket{
			Exchange: ex, Symbol: "BTCUSDT", Start: start.Add(time.Duration(i) * time.Minute),
			BuyNotional: (whale + retail) / float64(minutes), SizeKnown: true,
			WhaleBuy: whale / float64(minutes), RetailBuy: retail / float64(minutes),
		})
	}
	return out
}

func fillTapeHour(ex Exchange, start time.Time, minutes int, buy, sell float64) []TakerBucket {
	out := make([]TakerBucket, 0, minutes)
	for i := 0; i < minutes; i++ {
		out = append(out, TakerBucket{
			Exchange: ex, Symbol: "BTCUSDT", Start: start.Add(time.Duration(i) * time.Minute),
			BuyNotional: buy / float64(minutes), SellNotional: sell / float64(minutes),
		})
	}
	return out
}

func tapeCandles(signalAt time.Time, entry, h1, h4, h24 float64) []Candle {
	step := 5 * time.Minute
	return []Candle{
		{OpenTime: signalAt.Add(-step), Close: formatTapePx(entry)},
		{OpenTime: signalAt.Add(time.Hour).Add(-step), Close: formatTapePx(h1)},
		{OpenTime: signalAt.Add(4 * time.Hour).Add(-step), Close: formatTapePx(h4)},
		{OpenTime: signalAt.Add(24 * time.Hour).Add(-step), Close: formatTapePx(h24)},
	}
}

func formatTapePx(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
