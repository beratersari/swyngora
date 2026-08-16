package domain

import (
	"math"
	"strings"
	"testing"
)

func TestClassifyMove(t *testing.T) {
	if ClassifyMove(0.2) != BreadthDirUp {
		t.Fatal("up")
	}
	if ClassifyMove(-0.2) != BreadthDirDown {
		t.Fatal("down")
	}
	if ClassifyMove(0.01) != BreadthDirFlat {
		t.Fatal("flat")
	}
}

func TestIsBreadthEligible(t *testing.T) {
	if !IsBreadthEligible("SOL") || IsBreadthEligible("BTCUP") || IsBreadthEligible("ETH3L") {
		t.Fatal("eligibility")
	}
}

func TestCountBreadth(t *testing.T) {
	got := CountBreadth([]CoinMove{
		{Base: "AAA", ChangePct: 1, QuoteVolume: 70, Known: true},
		{Base: "BBB", ChangePct: 2, QuoteVolume: 10, Known: true},
		{Base: "CCC", ChangePct: -1, QuoteVolume: 10, Known: true},
		{Base: "DDD", ChangePct: 0, QuoteVolume: 10, Known: true},
		{Base: "EEE", ChangePct: 3, Known: false},
	})
	if got.Up != 2 || got.Down != 1 || got.Flat != 1 || got.Total != 4 {
		t.Fatalf("%+v", got)
	}
	if math.Abs(got.UpPct-50) > 1e-9 {
		t.Fatalf("upPct %v", got.UpPct)
	}
	if math.Abs(got.VolumeUpPct-80) > 1e-9 {
		t.Fatalf("volUp %v", got.VolumeUpPct)
	}
}

func TestBuildBreadthWindow_WithMarket(t *testing.T) {
	moves := make([]CoinMove, 0, 12)
	for i := 0; i < 10; i++ {
		moves = append(moves, CoinMove{Base: "A" + string(rune('A'+i)), ChangePct: 1, QuoteVolume: 1, Known: true})
	}
	moves = append(moves,
		CoinMove{Base: "BTC", ChangePct: 0.8, QuoteVolume: 50, Known: true},
		CoinMove{Base: "ETH", ChangePct: 0.6, QuoteVolume: 30, Known: true},
	)
	got := BuildBreadthWindow(BreadthWindow1h, moves)
	if got.Alignment != BreadthAlignWithMarket {
		t.Fatalf("%s %s", got.Alignment, got.Summary)
	}
	if !strings.Contains(strings.ToLower(got.Summary), "up") {
		t.Fatalf("summary %s", got.Summary)
	}
}

func TestBuildBreadthWindow_Carrying(t *testing.T) {
	moves := []CoinMove{
		{Base: "BTC", ChangePct: 2, QuoteVolume: 80, Known: true},
		{Base: "ETH", ChangePct: 1.5, QuoteVolume: 40, Known: true},
	}
	for i := 0; i < 12; i++ {
		moves = append(moves, CoinMove{Base: "X" + string(rune('A'+i)), ChangePct: -0.8, QuoteVolume: 1, Known: true})
	}
	got := BuildBreadthWindow(BreadthWindow24h, moves)
	if got.Alignment != BreadthAlignCarrying {
		t.Fatalf("%s %s", got.Alignment, got.Summary)
	}
}

func TestBuildBreadthWindow_MajorsFlatPackUp(t *testing.T) {
	moves := []CoinMove{
		{Base: "BTC", ChangePct: 0.01, QuoteVolume: 50, Known: true},
		{Base: "ETH", ChangePct: 0.02, QuoteVolume: 30, Known: true},
	}
	for i := 0; i < 10; i++ {
		moves = append(moves, CoinMove{Base: "Z" + string(rune('A'+i)), ChangePct: 0.4, QuoteVolume: 2, Known: true})
	}
	got := BuildBreadthWindow(BreadthWindow1h, moves)
	if !strings.Contains(strings.ToLower(got.Title), "little changed") {
		t.Fatalf("%s %s", got.Title, got.Summary)
	}
}

func TestBuildBreadthWindow_Lagging(t *testing.T) {
	moves := []CoinMove{
		{Base: "BTC", ChangePct: -1, QuoteVolume: 5, Known: true},
		{Base: "ETH", ChangePct: -0.8, QuoteVolume: 4, Known: true},
	}
	for i := 0; i < 12; i++ {
		moves = append(moves, CoinMove{Base: "Y" + string(rune('A'+i)), ChangePct: 1, QuoteVolume: 2, Known: true})
	}
	got := BuildBreadthWindow(BreadthWindow4h, moves)
	if got.Alignment != BreadthAlignLagging {
		t.Fatalf("%s %s", got.Alignment, got.Summary)
	}
}

func TestExplainBreadthReport(t *testing.T) {
	w1 := BuildBreadthWindow(BreadthWindow1h, []CoinMove{
		{Base: "BTC", ChangePct: 1, QuoteVolume: 10, Known: true},
		{Base: "ETH", ChangePct: 1, QuoteVolume: 10, Known: true},
		{Base: "SOL", ChangePct: 1, QuoteVolume: 5, Known: true},
		{Base: "XRP", ChangePct: 1, QuoteVolume: 5, Known: true},
		{Base: "DOGE", ChangePct: 1, QuoteVolume: 5, Known: true},
		{Base: "ADA", ChangePct: 1, QuoteVolume: 5, Known: true},
		{Base: "AVAX", ChangePct: 1, QuoteVolume: 5, Known: true},
		{Base: "LINK", ChangePct: 1, QuoteVolume: 5, Known: true},
		{Base: "DOT", ChangePct: 1, QuoteVolume: 5, Known: true},
		{Base: "UNI", ChangePct: 1, QuoteVolume: 5, Known: true},
	})
	got := ExplainBreadthReport([]BreadthWindow{w1})
	if !strings.Contains(got, "1h") {
		t.Fatalf("%s", got)
	}
}

func TestParseBreadthLimit(t *testing.T) {
	if ParseBreadthLimit(0) != breadthDefaultN || ParseBreadthLimit(3) != breadthMinN || ParseBreadthLimit(999) != breadthMaxN {
		t.Fatal("clamp")
	}
}
