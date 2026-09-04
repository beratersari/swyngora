package domain

import (
	"strings"
	"testing"
	"time"
)

func TestFillRadarAfter_LeanHit(t *testing.T) {
	now := time.Date(2026, 9, 4, 18, 0, 0, 0, time.UTC)
	got := FillRadarAfter(100, 102, CVDDirUp, now)
	if got.Direction != CVDDirUp || got.Useful == nil || !*got.Useful {
		t.Fatalf("%+v", got)
	}
	if got.ReturnPct < 1.9 || got.ReturnPct > 2.1 {
		t.Fatalf("ret %v", got.ReturnPct)
	}
	miss := FillRadarAfter(100, 98, CVDDirUp, now)
	if miss.Useful == nil || *miss.Useful {
		t.Fatalf("miss %+v", miss)
	}
	mixed := FillRadarAfter(100, 102, "mixed", now)
	if mixed.Useful != nil {
		t.Fatalf("mixed %+v", mixed)
	}
}

func TestRadarAfterDirection_Flat(t *testing.T) {
	if RadarAfterDirection(0.01) != "flat" {
		t.Fatal("flat")
	}
	if RadarAfterDirection(0.2) != CVDDirUp {
		t.Fatal("up")
	}
}

func TestAfterPriceFromCandles(t *testing.T) {
	due := time.Date(2026, 9, 4, 13, 0, 0, 0, time.UTC)
	px, ok := AfterPriceFromCandles([]Candle{
		{OpenTime: due.Add(-time.Hour), CloseTime: due.Add(-45 * time.Minute), Close: "100"},
		{OpenTime: due.Add(-15 * time.Minute), CloseTime: due, Close: "104.5"},
		{OpenTime: due.Add(time.Minute), CloseTime: due.Add(16 * time.Minute), Close: "110"},
	}, due)
	if !ok || px != 104.5 {
		t.Fatalf("%v %v", px, ok)
	}
}

func TestParseRadarSince(t *testing.T) {
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	from, id, err := ParseRadarSince("", now)
	if err != nil || id != "7d" || !from.Equal(now.Add(-7*24*time.Hour)) {
		t.Fatalf("%v %s %v", from, id, err)
	}
	if _, _, err := ParseRadarSince("nope", now); err == nil {
		t.Fatal("bad since")
	}
}

func TestExplainRadarReview(t *testing.T) {
	now := time.Date(2026, 9, 4, 18, 0, 0, 0, time.UTC)
	yes, no := true, false
	r := RadarReview{
		Since: "7d",
		Cases: []RadarCase{
			{
				Symbol: "BTCUSDT", Lean: CVDDirUp,
				After: []RadarAfter{
					{Horizon: "1h", FilledAt: &now, ReturnPct: 1.2, Direction: CVDDirUp, Useful: &yes},
					{Horizon: "4h", FilledAt: &now, ReturnPct: -0.4, Direction: CVDDirDown, Useful: &no},
				},
			},
			{
				Symbol: "ETHUSDT", Lean: CVDDirUp,
				After: []RadarAfter{
					{Horizon: "1h", FilledAt: &now, ReturnPct: 0.8, Direction: CVDDirUp, Useful: &yes},
				},
			},
		},
	}
	FinishRadarReview(&r)
	if !strings.Contains(r.Summary, "Saved 2") || !strings.Contains(r.Summary, "1h") {
		t.Fatalf("%q", r.Summary)
	}
	if len(r.Horizons) < 1 || r.Horizons[0].Horizon != "15m" {
		t.Fatalf("%+v", r.Horizons)
	}
	var h1 RadarHorizonStat
	for _, h := range r.Horizons {
		if h.Horizon == "1h" {
			h1 = h
		}
	}
	if h1.Sample != 2 || h1.LeanRight != 2 || h1.UsefulPct < 99 {
		t.Fatalf("1h %+v", h1)
	}
}

func TestNewRadarCase_Horizons(t *testing.T) {
	at := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	c := NewRadarCase(ForensicsScanHit{Symbol: "btcusdt", Exchange: "binance", Lean: CVDDirUp, Similarity: 80}, at, 100, "3d", "1h")
	if c.Symbol != "BTCUSDT" || len(c.After) != 3 || c.After[0].Horizon != "15m" {
		t.Fatalf("%+v", c)
	}
	if !c.After[1].DueAt.Equal(at.Add(time.Hour)) {
		t.Fatalf("due %+v", c.After)
	}
}
