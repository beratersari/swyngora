package domain

import (
	"strings"
	"testing"
	"time"
)

func TestBlendFundingPressure_OIWeighted(t *testing.T) {
	now := time.Date(2026, 8, 11, 15, 0, 0, 0, time.UTC)
	raw, ok := blendFundingPressure([]fundingWeight{
		{Exchange: ExchangeBinance, Rate: 0.0001, OI: 8e9},
		{Exchange: ExchangeBybit, Rate: -0.0001, OI: 1e9},
	}, now)
	if !ok {
		t.Fatal("blend")
	}
	// (8e9*0.0001 + 1e9*-0.0001) / 9e9 = 0.0000777…
	if raw.Rate < 0.000077 || raw.Rate > 0.000078 {
		t.Fatalf("rate %v", raw.Rate)
	}
	if raw.OI != 9e9 || len(raw.Venues) != 2 || raw.Venues[0].Exchange != ExchangeBinance {
		t.Fatalf("oi/venues %+v", raw)
	}
}

func TestBlendFundingPressure_SkipZeroOI(t *testing.T) {
	raw, ok := blendFundingPressure([]fundingWeight{
		{Exchange: ExchangeBinance, Rate: 0.0001, OI: 8e9},
		{Exchange: ExchangeBybit, Rate: -0.01, OI: 0},
	}, time.Now().UTC())
	if !ok || raw.Rate != 0.0001 || len(raw.Venues) != 1 {
		t.Fatalf("%+v ok=%v", raw, ok)
	}
	if _, ok := blendFundingPressure(nil, time.Now().UTC()); ok {
		t.Fatal("empty")
	}
}

func TestAttachFundingPressure_UserExample(t *testing.T) {
	now := time.Date(2026, 8, 11, 15, 0, 0, 0, time.UTC)
	snap := BuildFundingSnapshot("all", "BTCUSDT", []*FundingSeries{
		{
			Exchange:        ExchangeBinance,
			Current:         FundingPoint{Time: now, Rate: 0.0001, Predicted: true},
			NextFundingTime: now.Add(time.Hour),
			IntervalHours:   8,
		},
		{
			Exchange:        ExchangeBybit,
			Current:         FundingPoint{Time: now, Rate: -0.0001, Predicted: true},
			NextFundingTime: now.Add(2 * time.Hour),
			IntervalHours:   8,
		},
	}, now)
	AttachFundingPressure(snap, []*FundingSeries{
		{Exchange: ExchangeBinance, Current: FundingPoint{Rate: 0.0001}, NextFundingTime: now.Add(time.Hour), IntervalHours: 8},
		{Exchange: ExchangeBybit, Current: FundingPoint{Rate: -0.0001}, NextFundingTime: now.Add(2 * time.Hour), IntervalHours: 8},
	}, []*OpenInterestSeries{
		{Exchange: ExchangeBinance, Current: OpenInterestPoint{Value: 8e9, Time: now}},
		{Exchange: ExchangeBybit, Current: OpenInterestPoint{Value: 1e9, Time: now}},
	}, now)
	if snap.Pressure == nil {
		t.Fatal("pressure")
	}
	p := snap.Pressure
	if p.Payer != "long" || p.DominantVenue != "binance" || p.VenueCount != 2 {
		t.Fatalf("%+v", p)
	}
	if p.Venues[0].WeightPct != "88.9" || p.Venues[1].WeightPct != "11.1" {
		t.Fatalf("weights %+v", p.Venues)
	}
	if p.IntervalHours != 8 || p.NextFundingTime != now.Add(time.Hour) {
		t.Fatalf("clock %+v", p)
	}
	if !strings.Contains(p.Summary, "88.9%") || !strings.Contains(p.Summary, "Combined pressure") {
		t.Fatalf("summary %s", p.Summary)
	}
	if p.rate < 0.000077 || p.rate > 0.000078 {
		t.Fatalf("raw rate %v", p.rate)
	}
}

func TestWithholdFundingPressure_NamesAbsentVenue(t *testing.T) {
	snap := &FundingSnapshot{
		Exchange: "all",
		Pressure: &FundingPressure{
			Rate: "0.0001", RatePct: "0.0100%", Payer: "long",
			OpenInterestValue: "9000000000", VenueCount: 2, DominantVenue: "binance",
			rate: 0.000078,
			Venues: []FundingPressureVenue{
				{Exchange: "binance"},
				{Exchange: "bybit"},
			},
		},
		PressureWindows: []FundingPressureWindow{
			{Window: "1h", ChangePct: "+0.01", PastRate: "0.00005", PastRatePct: "0.005%", Complete: true},
		},
	}
	WithholdFundingPressure(snap, []Exchange{ExchangeBinance, ExchangeBybit, ExchangeOKX})
	p := snap.Pressure
	if p.Rate != "" || p.RatePct != "" || p.Payer != "" || p.OpenInterestValue != "" || p.VenueCount != 0 || p.DominantVenue != "" || p.rate != 0 {
		t.Fatalf("blanked %+v", p)
	}
	if len(p.Venues) != 2 {
		t.Fatal("blended venue rows stay")
	}
	if len(p.Missing) != 1 || p.Missing[0] != "okx" || !strings.Contains(p.Summary, "okx") {
		t.Fatalf("%+v", p)
	}
	w := snap.PressureWindows[0]
	if w.Complete || w.ChangePct != "" || w.PastRate != "" || w.PastRatePct != "" {
		t.Fatalf("window %+v", w)
	}

	both := &FundingSnapshot{
		Exchange: "all",
		Pressure: &FundingPressure{
			Rate: "0.0001", Payer: "long", VenueCount: 2, DominantVenue: "binance",
			Venues: []FundingPressureVenue{{Exchange: "binance"}, {Exchange: "bybit"}},
		},
	}
	WithholdFundingPressure(both, []Exchange{ExchangeBinance, ExchangeBybit})
	if both.Pressure.Rate != "0.0001" || both.Pressure.Payer != "long" || both.Pressure.DominantVenue != "binance" {
		t.Fatalf("should keep %+v", both.Pressure)
	}
}

func TestAttachFundingPressureHistory_WindowsAndFlip(t *testing.T) {
	now := time.Date(2026, 8, 11, 16, 0, 0, 0, time.UTC)
	bucket := func(h int) time.Time {
		return TruncateToBucket(now.Add(time.Duration(h)*time.Hour), fundingPressureBucket)
	}
	var fund, oi []FuturesSnapshot
	// 24h ago: shorts pay on the big book. Now: longs pay.
	samples := []struct {
		at           time.Time
		bin, byb     float64
		binOI, bybOI float64
	}{
		{bucket(-24), -0.0001, -0.0001, 8e9, 1e9},
		{bucket(-4), -0.00002, 0.00001, 8e9, 1e9},
		{bucket(-1), 0.00005, -0.00001, 8e9, 1e9},
		{bucket(0), 0.0001, -0.0001, 8e9, 1e9},
	}
	for _, s := range samples {
		fund = append(fund,
			FuturesSnapshot{Metric: FuturesMetricFunding, Exchange: ExchangeBinance, SampledAt: s.at, Predicted: true, FundingRate: s.bin},
			FuturesSnapshot{Metric: FuturesMetricFunding, Exchange: ExchangeBybit, SampledAt: s.at, Predicted: true, FundingRate: s.byb},
		)
		oi = append(oi,
			FuturesSnapshot{Metric: FuturesMetricOpenInterest, Exchange: ExchangeBinance, SampledAt: s.at, Value: s.binOI},
			FuturesSnapshot{Metric: FuturesMetricOpenInterest, Exchange: ExchangeBybit, SampledAt: s.at, Value: s.bybOI},
		)
	}
	snap := &FundingSnapshot{Symbol: "BTCUSDT", Exchange: "all", AsOf: now}
	AttachFundingPressure(snap, []*FundingSeries{
		{Exchange: ExchangeBinance, Current: FundingPoint{Rate: 0.0001}},
		{Exchange: ExchangeBybit, Current: FundingPoint{Rate: -0.0001}},
	}, []*OpenInterestSeries{
		{Exchange: ExchangeBinance, Current: OpenInterestPoint{Value: 8e9}},
		{Exchange: ExchangeBybit, Current: OpenInterestPoint{Value: 1e9}},
	}, now)
	AttachFundingPressureHistory(snap, fund, oi, now)
	if len(snap.PressureHistory) < 3 {
		t.Fatalf("history %d", len(snap.PressureHistory))
	}
	var saw24 bool
	for _, w := range snap.PressureWindows {
		if w.Window == SnapshotWindow24h {
			saw24 = true
			if w.PastPayer != "short" || w.Payer != "long" || w.Kind != FundingPressureFlipped {
				t.Fatalf("24h %+v", w)
			}
			if !w.Complete {
				t.Fatalf("24h should be complete %+v", w)
			}
		}
	}
	if !saw24 {
		t.Fatal("missing 24h window")
	}
	if len(snap.PressureMoves) == 0 {
		t.Fatal("expected flip/steepen moves")
	}
	foundFlip := false
	for _, m := range snap.PressureMoves {
		if m.Kind == FundingPressureFlipped {
			foundFlip = true
			if !strings.Contains(m.Summary, "flipped") {
				t.Fatalf("summary %s", m.Summary)
			}
		}
	}
	if !foundFlip {
		t.Fatalf("moves %+v", snap.PressureMoves)
	}
}

func TestFundingPressureKind(t *testing.T) {
	if fundingPressureKind(-0.0001, 0.0001, fundingPressureWindowMove) != FundingPressureFlipped {
		t.Fatal("flip")
	}
	if fundingPressureKind(0.0001, 0.0002, fundingPressureWindowMove) != FundingPressureSteeper {
		t.Fatal("steeper")
	}
	if fundingPressureKind(0.0002, 0.0001, fundingPressureWindowMove) != FundingPressureFlatter {
		t.Fatal("flatter")
	}
	if fundingPressureKind(0.0001, 0.00010001, fundingPressureWindowMove) != FundingPressureStable {
		t.Fatal("stable")
	}
}

func TestBuildFundingPressureHistory_IgnoresSettled(t *testing.T) {
	now := time.Date(2026, 8, 11, 16, 0, 0, 0, time.UTC)
	at := TruncateToBucket(now, fundingPressureBucket)
	pts := buildFundingPressureHistory([]FuturesSnapshot{
		{Predicted: false, Exchange: ExchangeBinance, SampledAt: at, FundingRate: 0.01},
		{Predicted: true, Exchange: ExchangeBinance, SampledAt: at, FundingRate: 0.0001},
	}, []FuturesSnapshot{
		{Exchange: ExchangeBinance, SampledAt: at, Value: 1e9},
	}, "binance", now)
	if len(pts) != 1 || pts[0].Rate != 0.0001 {
		t.Fatalf("%+v", pts)
	}
}
