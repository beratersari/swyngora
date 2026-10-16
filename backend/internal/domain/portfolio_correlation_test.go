package domain

import (
	"math"
	"testing"
)

func wave(n int, amp float64) []float64 {
	out := make([]float64, n)
	for i := range out {
		if i%2 == 0 {
			out[i] = amp
		} else {
			out[i] = -amp
		}
	}
	return out
}

func TestBuildPortfolioCorrelation_TogetherIsLargerThanApart(t *testing.T) {
	n := corrMinSamples + 1
	got := BuildPortfolioCorrelation([]PortfolioCorrInput{
		{Asset: "BTC", Notional: 100, Returns: wave(n, 1)},
		{Asset: "ETH", Notional: 100, Returns: wave(n, 1)},
	})
	if got.Together == nil || math.Abs(*got.Together-1) > 1e-9 {
		t.Fatalf("together %+v", got.Together)
	}
	if got.SwingPct == nil || got.ApartPct == nil {
		t.Fatal("swing missing")
	}
	if math.Abs(*got.SwingPct/(*got.ApartPct)-math.Sqrt(2)) > 1e-9 {
		t.Fatalf("swing %v apart %v", *got.SwingPct, *got.ApartPct)
	}
	if len(got.Pairs) != 1 || !got.Pairs[0].Complete || got.Pairs[0].Relation != CorrRelationFollows {
		t.Fatalf("pair %+v", got.Pairs)
	}
	if got.Pairs[0].Left != "BTC" || got.Pairs[0].Right != "ETH" {
		t.Fatalf("order %+v", got.Pairs[0])
	}
}

func TestBuildPortfolioCorrelation_ShortOffsetsASharedMove(t *testing.T) {
	n := corrMinSamples + 1
	got := BuildPortfolioCorrelation([]PortfolioCorrInput{
		{Asset: "BTC", Notional: 100, Returns: wave(n, 1)},
		{Asset: "ETH", Notional: -100, Returns: wave(n, 1)},
	})
	if got.Together == nil || math.Abs(*got.Together-1) > 1e-9 {
		t.Fatalf("prices still move together %+v", got.Together)
	}
	if got.SwingPct == nil || math.Abs(*got.SwingPct) > 1e-9 {
		t.Fatalf("hedged swing %+v", got.SwingPct)
	}
	if got.Holdings[1].Side != "short" {
		t.Fatalf("side %+v", got.Holdings)
	}
}

func TestBuildPortfolioCorrelation_OneCoinHasNoPair(t *testing.T) {
	n := corrMinSamples + 1
	got := BuildPortfolioCorrelation([]PortfolioCorrInput{
		{Asset: "BTC", Notional: 50, Returns: wave(n, 2)},
	})
	if got.Together != nil || len(got.Pairs) != 0 {
		t.Fatalf("pair invented %+v", got)
	}
	if got.SwingPct == nil || got.ApartPct == nil || math.Abs(*got.SwingPct-*got.ApartPct) > 1e-9 {
		t.Fatalf("solo %+v %+v", got.SwingPct, got.ApartPct)
	}
}

func TestBuildPortfolioCorrelation_ShortListIsNotZero(t *testing.T) {
	got := BuildPortfolioCorrelation([]PortfolioCorrInput{
		{Asset: "BTC", Notional: 100, Returns: wave(4, 1)},
		{Asset: "ETH", Notional: 100, Returns: wave(4, 1)},
	})
	if got.Together != nil || got.SwingPct != nil || len(got.Missing) != 2 {
		t.Fatalf("short window leaked %+v", got)
	}
}

func TestBuildPortfolioCorrelation_EmptyBook(t *testing.T) {
	got := BuildPortfolioCorrelation(nil)
	if got.Gross != 0 || got.Together != nil || len(got.Holdings) != 0 {
		t.Fatalf("%+v", got)
	}
}
