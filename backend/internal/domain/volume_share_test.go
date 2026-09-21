package domain

import (
	"math"
	"testing"
	"time"
)

func TestBuildVolumeShare_ExampleMixAndChange(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	// Current 1h: BN 55, BY 30, OKX 15. Prior 1h: BN 50, BY 35, OKX 15.
	series := map[Exchange][]FuturesVolumeBar{
		ExchangeBinance: volumeBars(now, 55, 50),
		ExchangeBybit:   volumeBars(now, 30, 35),
		ExchangeOKX:     volumeBars(now, 15, 15),
	}
	got := BuildVolumeShare("all", "BTCUSDT", series, now)
	if got == nil || len(got.Windows) != 3 {
		t.Fatalf("%+v", got)
	}
	var h1 *VolumeShareWindow
	for i := range got.Windows {
		if got.Windows[i].Window == VolumeShareWindow1h {
			h1 = &got.Windows[i]
		}
	}
	if h1 == nil || !h1.Complete || len(h1.Venues) != 3 {
		t.Fatalf("1h %+v", h1)
	}
	if math.Abs(h1.Venues[0].SharePct-55) > 0.2 || h1.Venues[0].Exchange != ExchangeBinance {
		t.Fatalf("top %+v", h1.Venues[0])
	}
	var bn, by VolumeShareVenue
	for _, v := range h1.Venues {
		switch v.Exchange {
		case ExchangeBinance:
			bn = v
		case ExchangeBybit:
			by = v
		}
	}
	if bn.Kind != VolumeShareGained || by.Kind != VolumeShareLost {
		t.Fatalf("kinds bn=%s by=%s change bn=%v by=%v", bn.Kind, by.Kind, bn.ChangePct, by.ChangePct)
	}
	if len(h1.Gained) != 1 || h1.Gained[0] != "Binance" || len(h1.Lost) != 1 || h1.Lost[0] != "Bybit" {
		t.Fatalf("gained=%v lost=%v summary=%s", h1.Gained, h1.Lost, h1.Summary)
	}
}

func TestClassifyVolumeShare(t *testing.T) {
	t.Parallel()
	if ClassifyVolumeShare(3) != VolumeShareGained {
		t.Fatal("gained")
	}
	if ClassifyVolumeShare(-2) != VolumeShareLost {
		t.Fatal("lost")
	}
	if ClassifyVolumeShare(0.4) != VolumeShareStable {
		t.Fatal("stable")
	}
}

func TestExplainVolumeShareWindow(t *testing.T) {
	t.Parallel()
	s := ExplainVolumeShareWindow(VolumeShareWindow{
		Window: "1h",
		Total:  100,
		Venues: []VolumeShareVenue{
			{Exchange: ExchangeBinance, SharePct: 55},
			{Exchange: ExchangeBybit, SharePct: 30},
			{Exchange: ExchangeOKX, SharePct: 15},
		},
		Gained: []string{"Binance"},
		Lost:   []string{"Bybit"},
	})
	if s != "1h: Binance 55%, Bybit 30%, and OKX 15%. Binance gained share; Bybit lost share." {
		t.Fatalf("%q", s)
	}
}

func volumeBars(now time.Time, curShare, priorShare float64) []FuturesVolumeBar {
	// 12 current 5m bars + 12 prior 5m bars. Total current+prior = 100 units * 12 bars.
	out := make([]FuturesVolumeBar, 0, 24)
	for i := 0; i < 12; i++ {
		out = append(out, FuturesVolumeBar{
			Time:        now.Add(-time.Duration(i+1) * 5 * time.Minute),
			QuoteVolume: curShare,
		})
	}
	for i := 0; i < 12; i++ {
		out = append(out, FuturesVolumeBar{
			Time:        now.Add(-time.Hour - time.Duration(i+1)*5*time.Minute),
			QuoteVolume: priorShare,
		})
	}
	return out
}
