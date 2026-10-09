package domain

import (
	"errors"
	"testing"
	"time"
)

func TestParseFuturesBucket(t *testing.T) {
	got, err := ParseFuturesBucket("30m")
	if err != nil || got != 30*time.Minute {
		t.Fatalf("%v %v", got, err)
	}
	if _, err := ParseFuturesBucket("1h"); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("err %v", err)
	}
	label, err := FuturesBucketLabel(4 * time.Hour)
	if err != nil || label != "4h" {
		t.Fatalf("%s %v", label, err)
	}
}

func TestHuntSnapshotBucket(t *testing.T) {
	if HuntSnapshotBucket(12*time.Hour) != 0 || HuntSnapshotBucket(24*time.Hour) != 0 {
		t.Fatal("a day stays on the raw page")
	}
	if HuntSnapshotBucket(3*24*time.Hour) != 30*time.Minute {
		t.Fatal("3d")
	}
	if HuntSnapshotBucket(7*24*time.Hour) != 30*time.Minute {
		t.Fatal("7d")
	}
	if HuntSnapshotBucket(14*24*time.Hour) != 2*time.Hour {
		t.Fatal("14d")
	}
	if HuntSnapshotBucket(30*24*time.Hour) != 2*time.Hour {
		t.Fatal("30d")
	}
	if HuntSnapshotBucket(90*24*time.Hour) != 4*time.Hour {
		t.Fatal("90d")
	}
}

func TestFuturesPaneBucketForSpan(t *testing.T) {
	if FuturesPaneBucketForSpan(12*time.Hour) != 5*time.Minute {
		t.Fatal("24h")
	}
	if FuturesPaneBucketForSpan(3*24*time.Hour) != 30*time.Minute {
		t.Fatal("7d")
	}
	if FuturesPaneBucketForSpan(12*24*time.Hour) != 2*time.Hour {
		t.Fatal("30d")
	}
	if FuturesPaneBucketForSpan(60*24*time.Hour) != 4*time.Hour {
		t.Fatal("90d")
	}
}

func TestClampFuturesPaneWindow(t *testing.T) {
	now := time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)
	from, to := ClampFuturesPaneWindow(now.Add(-200*24*time.Hour), time.Time{}, now)
	if !to.Equal(now) {
		t.Fatalf("to %s", to)
	}
	if !from.Equal(now.Add(-FuturesHistoryMaxLookback)) {
		t.Fatalf("from %s", from)
	}
}

func TestDownsampleSnapshots_LastInBucketAndNoZeroFill(t *testing.T) {
	t0 := time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC)
	rows := []FuturesSnapshot{
		{Metric: FuturesMetricOpenInterest, Exchange: ExchangeBinance, Symbol: "BTCUSDT", SampledAt: t0, Value: 100},
		{Metric: FuturesMetricOpenInterest, Exchange: ExchangeBinance, Symbol: "BTCUSDT", SampledAt: t0.Add(10 * time.Minute), Value: 140},
		{Metric: FuturesMetricOpenInterest, Exchange: ExchangeBinance, Symbol: "BTCUSDT", SampledAt: t0.Add(time.Hour), Value: 90},
		{Metric: FuturesMetricOpenInterest, Exchange: ExchangeBybit, Symbol: "BTCUSDT", SampledAt: t0.Add(5 * time.Minute), Value: 40},
	}
	got := DownsampleSnapshots(FuturesMetricOpenInterest, rows, 30*time.Minute)
	if len(got) != 3 {
		t.Fatalf("points %d %+v", len(got), got)
	}
	// Newest first. The 10:00 bucket keeps 140, not 100. 10:30 is absent.
	if !got[0].SampledAt.Equal(t0.Add(time.Hour)) || got[0].Value != 90 {
		t.Fatalf("newest %+v", got[0])
	}
	var binanceEarly FuturesSnapshot
	for _, row := range got {
		if row.Exchange == ExchangeBinance && row.SampledAt.Equal(t0.Add(10*time.Minute)) {
			binanceEarly = row
		}
		if row.SampledAt.Equal(t0.Add(30 * time.Minute)) {
			t.Fatal("missing middle bucket was filled")
		}
	}
	if binanceEarly.Value != 140 {
		t.Fatalf("early %+v", binanceEarly)
	}
}

func TestDownsampleSnapshots_FundingPrefersSettled(t *testing.T) {
	t0 := time.Date(2026, 10, 8, 8, 0, 0, 0, time.UTC)
	rows := []FuturesSnapshot{
		{Metric: FuturesMetricFunding, Exchange: ExchangeBinance, SampledAt: t0, FundingRate: 0.0001, Predicted: false},
		{Metric: FuturesMetricFunding, Exchange: ExchangeBinance, SampledAt: t0.Add(20 * time.Minute), FundingRate: 0.0005, Predicted: true},
		{Metric: FuturesMetricFunding, Exchange: ExchangeOKX, SampledAt: t0.Add(40 * time.Minute), FundingRate: -0.0002, Predicted: true},
	}
	got := DownsampleSnapshots(FuturesMetricFunding, rows, 30*time.Minute)
	if len(got) != 2 {
		t.Fatalf("%d", len(got))
	}
	for _, row := range got {
		if row.Exchange == ExchangeBinance {
			if row.Predicted || row.FundingRate != 0.0001 {
				t.Fatalf("binance %+v", row)
			}
		}
		if row.Exchange == ExchangeOKX && (row.FundingRate != -0.0002 || !row.Predicted) {
			t.Fatalf("okx %+v", row)
		}
	}
}

func TestDownsampleSnapshots_LongShortKeepsTopShare(t *testing.T) {
	t0 := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	rows := []FuturesSnapshot{
		{
			Metric: FuturesMetricLongShort, Exchange: ExchangeBinance, SampledAt: t0,
			LongShare: 0.48, ShortShare: 0.52, Ratio: 0.48 / 0.52, SizeLongShare: 0.61,
		},
		{
			Metric: FuturesMetricLongShort, Exchange: ExchangeBinance, SampledAt: t0.Add(time.Hour),
			LongShare: 0.50, ShortShare: 0.50, Ratio: 1, SizeLongShare: 0.58,
		},
		{
			Metric: FuturesMetricLongShort, Exchange: ExchangeBybit, SampledAt: t0,
			LongShare: 0.55, ShortShare: 0.45, Ratio: 0.55 / 0.45,
		},
	}
	got := DownsampleSnapshots(FuturesMetricLongShort, rows, 4*time.Hour)
	if len(got) != 2 {
		t.Fatalf("%d", len(got))
	}
	for _, row := range got {
		switch row.Exchange {
		case ExchangeBinance:
			if row.LongShare != 0.50 || row.SizeLongShare != 0.58 {
				t.Fatalf("binance %+v", row)
			}
		case ExchangeBybit:
			if row.SizeLongShare != 0 || row.TopAccLongShare != 0 || row.LongShare != 0.55 {
				t.Fatalf("bybit %+v", row)
			}
		}
	}
}

func TestDownsampleLiquidations_SumsAndSkipsQuiet(t *testing.T) {
	t0 := time.Date(2026, 10, 8, 10, 7, 0, 0, time.UTC)
	rows := []LiquidationEvent{
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: "short", Notional: 5, Quantity: 1, Time: t0},
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: "short", Notional: 2, Quantity: 1, Time: t0.Add(time.Minute)},
		{Exchange: ExchangeBybit, Symbol: "BTCUSDT", Side: "long", Notional: 4, Quantity: 1, Time: t0.Add(2 * time.Minute)},
		{Exchange: ExchangeOKX, Symbol: "BTCUSDT", Side: "long", Notional: 9, Quantity: 1, Time: t0.Add(40 * time.Minute)},
	}
	got := DownsampleLiquidations(rows, 30*time.Minute)
	if len(got) != 3 {
		t.Fatalf("%d %+v", len(got), got)
	}
	open := TruncateToBucket(t0, 30*time.Minute)
	var short, long float64
	for _, row := range got {
		if row.Time.Equal(open) && row.Side == "short" && row.Exchange == ExchangeBinance {
			short = row.Notional
		}
		if row.Time.Equal(open) && row.Side == "long" && row.Exchange == ExchangeBybit {
			long = row.Notional
		}
	}
	if short != 7 || long != 4 {
		t.Fatalf("short %v long %v", short, long)
	}
}
