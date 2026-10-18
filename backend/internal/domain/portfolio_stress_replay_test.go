package domain

import (
	"math"
	"strconv"
	"testing"
	"time"
)

func TestParseStressWindow_InclusiveDatesUseDailyCloses(t *testing.T) {
	now := time.Date(2026, 10, 18, 15, 0, 0, 0, time.UTC)
	from, to, iv, err := ParseStressWindow("2024-08-05", "2024-08-06", now)
	if err != nil {
		t.Fatal(err)
	}
	if !from.Equal(time.Date(2024, 8, 5, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("from %s", from)
	}
	if !to.Equal(time.Date(2024, 8, 7, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("to %s", to)
	}
	if iv != Interval1d {
		t.Fatalf("interval %s", iv)
	}
}

func TestParseStressWindow_ShortInstantUsesHourly(t *testing.T) {
	now := time.Date(2026, 10, 18, 15, 0, 0, 0, time.UTC)
	from, to, iv, err := ParseStressWindow("2024-08-05T12:00:00Z", "2024-08-05T18:00:00Z", now)
	if err != nil {
		t.Fatal(err)
	}
	if to.Sub(from) != 6*time.Hour || iv != Interval1h {
		t.Fatalf("from %s to %s iv %s", from, to, iv)
	}
}

func TestParseStressWindow_RejectsOpenTodayAndMixedBounds(t *testing.T) {
	now := time.Date(2026, 10, 18, 0, 30, 0, 0, time.UTC)
	if _, _, _, err := ParseStressWindow("2026-10-18", "2026-10-18", now); err == nil {
		t.Fatal("expected a window that has not been open an hour")
	}
	if _, _, _, err := ParseStressWindow("2024-08-05", "2024-08-05T18:00:00Z", now); err == nil {
		t.Fatal("expected mixed bounds to fail")
	}
	if _, _, _, err := ParseStressWindow("2024-08-06", "2024-08-05", now); err == nil {
		t.Fatal("expected reversed dates to fail")
	}
	far := time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, _, _, err := ParseStressWindow(far.Format(time.RFC3339), now.Format(time.RFC3339), now); err == nil {
		t.Fatal("expected a window past 10 years to fail")
	}
}

func TestParseStressWindow_ClampsFutureEnd(t *testing.T) {
	now := time.Date(2026, 10, 18, 15, 0, 0, 0, time.UTC)
	_, to, iv, err := ParseStressWindow("2024-08-05T00:00:00Z", "2026-10-19T00:00:00Z", now)
	if err != nil {
		t.Fatal(err)
	}
	if !to.Equal(now) || iv != Interval1d {
		t.Fatalf("to %s iv %s", to, iv)
	}
}

func TestBuildReplayPlan_WeightsVenuesAndNamesTheGap(t *testing.T) {
	now := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	from := time.Date(2024, 8, 5, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 8, 7, 0, 0, 0, 0, time.UTC)
	plan := BuildReplayPlan([]ReplaySeries{
		{Exchange: "binance", Symbol: "BTCUSDT", Weight: 100, Bars: replayDays(t, 100, 80)},
		{Exchange: "coinbase", Symbol: "BTC-USD", Weight: 300, Bars: replayDays(t, 100, 50)},
		{Exchange: "binance", Symbol: "ETHUSDT", Weight: 40, Failed: true},
	}, from, to, Interval1d, now)
	if len(plan.Moves) != 2 {
		t.Fatalf("moves %+v", plan.Moves)
	}
	gotPct := map[string]float64{}
	for _, m := range plan.Moves {
		gotPct[m.Exchange+"|"+m.Symbol] = m.MovePct
	}
	if math.Abs(gotPct["binance|BTCUSDT"]-(-20)) > 1e-9 || math.Abs(gotPct["coinbase|BTC-USD"]-(-50)) > 1e-9 {
		t.Fatalf("venue moves %+v", gotPct)
	}
	// 100*(-20) + 300*(-50) = -17000, / 400 = -42.5. That summary is not applied as one percent.
	if len(plan.Replay.Moves) != 1 || math.Abs(plan.Replay.Moves[0].MovePct-(-42.5)) > 1e-9 {
		t.Fatalf("blend %+v", plan.Replay.Moves)
	}
	if len(plan.Replay.Moves[0].Venues) != 2 {
		t.Fatalf("venues %+v", plan.Replay.Moves[0].Venues)
	}
	if len(plan.Replay.Missing) != 1 || plan.Replay.Missing[0].Symbol != "ETHUSDT" || plan.Replay.Missing[0].Reason != ReplayReasonFetch {
		t.Fatalf("missing %+v", plan.Replay.Missing)
	}
	if plan.FetchFailures != 1 {
		t.Fatalf("failures %d", plan.FetchFailures)
	}
}

func TestBuildReplayPlan_SortsCoinsByAbsoluteMove(t *testing.T) {
	now := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	from := time.Date(2024, 8, 5, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 8, 7, 0, 0, 0, 0, time.UTC)
	plan := BuildReplayPlan([]ReplaySeries{
		{Exchange: "binance", Symbol: "ETHUSDT", Weight: 10, Bars: replayDays(t, 100, 110)},
		{Exchange: "binance", Symbol: "ADAUSDT", Weight: 10, Bars: replayDays(t, 100, 80)},
		{Exchange: "binance", Symbol: "SOLUSDT", Weight: 10, Bars: replayDays(t, 100, 140)},
		{Exchange: "binance", Symbol: "BTCUSDT", Weight: 10, Bars: replayDays(t, 100, 50)},
		{Exchange: "binance", Symbol: "DOGEUSDT", Weight: 10, Bars: replayDays(t, 100, 120)},
	}, from, to, Interval1d, now)
	var names []string
	for _, m := range plan.Replay.Moves {
		names = append(names, m.Asset)
	}
	// −50, +40, then −20 and +20 by name, then +10.
	want := []string{"BTC", "SOL", "ADA", "DOGE", "ETH"}
	if len(names) != len(want) {
		t.Fatalf("names %v", names)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("order %v", names)
		}
	}
}

func TestReplayPath_KeepsTheMiddleDipAndALiquidation(t *testing.T) {
	now := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	from := time.Date(2024, 8, 5, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 8, 7, 0, 0, 0, 0, time.UTC)
	bars := []Candle{
		replayDay(time.Date(2024, 8, 4, 0, 0, 0, 0, time.UTC), 100),
		replayDay(time.Date(2024, 8, 5, 0, 0, 0, 0, time.UTC), 50),
		replayDay(time.Date(2024, 8, 6, 0, 0, 0, 0, time.UTC), 90),
	}
	series := []ReplaySeries{{Exchange: "binance", Symbol: "BTCUSDT", Weight: 100, Bars: bars}}
	plan := BuildReplayPlan(series, from, to, Interval1d, now)
	view := PortfolioView{
		Equity: 10000, PositionsValue: 100,
		Positions: []PositionView{{
			Exchange: ExchangeBinance, Symbol: "BTCUSDT", Quantity: 1,
			AvgCost: 100, MarkPrice: 100, MarketValue: 100,
		}},
	}
	points := ReplayPath(view, series, plan, to, Interval1d, now)
	if len(points) != 3 {
		t.Fatalf("points %d %+v", len(points), points)
	}
	if math.Abs(points[0].Equity-10000) > 0.01 || math.Abs(points[1].Equity-9950) > 0.01 {
		t.Fatalf("path %v then %v", points[0].Equity, points[1].Equity)
	}
	end, err := StressPortfolio(view, plan.Moves)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(points[2].Equity-end.ShockedEquity) > 0.01 {
		t.Fatalf("end path %v stress %v", points[2].Equity, end.ShockedEquity)
	}
	low, ok := PathLow(points)
	if !ok || math.Abs(low.Equity-9950) > 0.01 || !low.At.Equal(points[1].At) {
		t.Fatalf("low %+v", low)
	}

	liqBars := []Candle{
		replayDay(time.Date(2024, 8, 4, 0, 0, 0, 0, time.UTC), 100),
		replayDay(time.Date(2024, 8, 5, 0, 0, 0, 0, time.UTC), 80),
		replayDay(time.Date(2024, 8, 6, 0, 0, 0, 0, time.UTC), 120),
	}
	liqSeries := []ReplaySeries{{Exchange: "binance", Symbol: "BTCUSDT", Weight: 20, Bars: liqBars}}
	liqPlan := BuildReplayPlan(liqSeries, from, to, Interval1d, now)
	margin := PortfolioView{
		Equity: 10000, MarginEquity: 20,
		MarginPositions: []MarginPosition{{
			Exchange: ExchangeBinance, Symbol: "BTCUSDT", Status: MarginPositionOpen,
			Side: MarginLong, Quantity: 1, EntryPrice: 100, MarkPrice: 100,
			Margin: 20, LiquidationPrice: 91,
		}},
	}
	liqPoints := ReplayPath(margin, liqSeries, liqPlan, to, Interval1d, now)
	if len(liqPoints) != 3 || liqPoints[1].Liquidated != 1 || liqPoints[2].Liquidated != 1 {
		t.Fatalf("liq path %+v", liqPoints)
	}
	if math.Abs(liqPoints[1].Equity-9980) > 0.01 || math.Abs(liqPoints[2].Equity-9980) > 0.01 {
		t.Fatalf("liquidation did not stick: %v then %v", liqPoints[1].Equity, liqPoints[2].Equity)
	}
}

func TestBuildReplayPlan_FlatCloseIsARealMove(t *testing.T) {
	now := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	from := time.Date(2024, 8, 5, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 8, 7, 0, 0, 0, 0, time.UTC)
	plan := BuildReplayPlan([]ReplaySeries{
		{Exchange: "binance", Symbol: "BTCUSDT", Weight: 10, Bars: replayDays(t, 100, 100)},
	}, from, to, Interval1d, now)
	if len(plan.Moves) != 1 || plan.Moves[0].MovePct != 0 {
		t.Fatalf("moves %+v missing %+v", plan.Moves, plan.Replay.Missing)
	}
}

func TestBuildReplayPlan_NamesMissingStartAndOutOfRange(t *testing.T) {
	now := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	from := time.Date(2024, 8, 5, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 8, 7, 0, 0, 0, 0, time.UTC)
	late := []Candle{replayDay(time.Date(2024, 8, 6, 0, 0, 0, 0, time.UTC), 80)}
	plan := BuildReplayPlan([]ReplaySeries{
		{Exchange: "binance", Symbol: "SOLUSDT", Weight: 10, Bars: late},
		{Exchange: "binance", Symbol: "DOGEUSDT", Weight: 10, Bars: replayDays(t, 100, 700)},
	}, from, to, Interval1d, now)
	if len(plan.Moves) != 0 {
		t.Fatalf("moves %+v", plan.Moves)
	}
	reasons := map[string]string{}
	for _, g := range plan.Replay.Missing {
		reasons[g.Symbol] = g.Reason
	}
	if reasons["SOLUSDT"] != ReplayReasonNoStart || reasons["DOGEUSDT"] != ReplayReasonOutOfRange {
		t.Fatalf("reasons %+v", reasons)
	}
}

func TestReplayBarClose_EquityBarEndsOneDayLater(t *testing.T) {
	open := time.Date(2024, 8, 4, 0, 0, 0, 0, time.UTC)
	bar := Candle{OpenTime: open, CloseTime: open, Close: "100"}
	got := ReplayBarClose(bar, 24*time.Hour)
	want := open.Add(24 * time.Hour)
	if !got.Equal(want) {
		t.Fatalf("close %s", got)
	}
	from := time.Date(2024, 8, 5, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 8, 7, 0, 0, 0, 0, time.UTC)
	end := Candle{OpenTime: time.Date(2024, 8, 6, 0, 0, 0, 0, time.UTC), CloseTime: time.Date(2024, 8, 6, 0, 0, 0, 0, time.UTC), Close: "80"}
	plan := BuildReplayPlan([]ReplaySeries{
		{Exchange: "nasdaq", Symbol: "AAPL", Weight: 1, Bars: []Candle{bar, end}},
	}, from, to, Interval1d, time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC))
	if len(plan.Moves) != 1 || math.Abs(plan.Moves[0].MovePct-(-20)) > 1e-9 {
		t.Fatalf("move %+v missing %+v", plan.Moves, plan.Replay.Missing)
	}
}

func TestBuildReplayPlan_FeedsStressLiquidation(t *testing.T) {
	now := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	from := time.Date(2024, 8, 5, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 8, 7, 0, 0, 0, 0, time.UTC)
	plan := BuildReplayPlan([]ReplaySeries{
		{Exchange: "binance", Symbol: "BTCUSDT", Weight: 100, Bars: replayDays(t, 100, 90)},
	}, from, to, Interval1d, now)
	view := PortfolioView{
		Equity: 10000, MarginEquity: 20,
		Positions: []PositionView{
			{Symbol: "BTCUSDT", Quantity: 1, AvgCost: 100, MarkPrice: 100, MarketValue: 100},
		},
		MarginPositions: []MarginPosition{
			{Symbol: "BTCUSDT", Side: MarginLong, Status: MarginPositionOpen, Quantity: 1, EntryPrice: 100, MarkPrice: 100, Margin: 20, LiquidationPrice: 91},
		},
	}
	got, err := StressPortfolio(view, plan.Moves)
	if err != nil {
		t.Fatal(err)
	}
	if got.Liquidated != 1 {
		t.Fatalf("10%% drop crosses 91, liquidated=%d move=%v", got.Liquidated, plan.Moves[0].MovePct)
	}
}

func replayDays(t *testing.T, start, end float64) []Candle {
	t.Helper()
	return []Candle{
		replayDay(time.Date(2024, 8, 4, 0, 0, 0, 0, time.UTC), start),
		replayDay(time.Date(2024, 8, 6, 0, 0, 0, 0, time.UTC), end),
	}
}

func replayDay(open time.Time, px float64) Candle {
	return Candle{
		OpenTime:  open,
		Close:     strconv.FormatFloat(px, 'f', -1, 64),
		CloseTime: open.Add(24*time.Hour - time.Millisecond),
	}
}
