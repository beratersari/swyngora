package domain

import (
	"testing"
	"time"
)

func TestClassifyWhaleIntent(t *testing.T) {
	intent, pct := ClassifyWhaleIntent(WhaleSideBuy, 100, 101)
	if intent != WhaleIntentOpenLong || pct < 0.9 {
		t.Fatalf("buy + oi up = %s %v", intent, pct)
	}
	intent, _ = ClassifyWhaleIntent(WhaleSideBuy, 100, 99)
	if intent != WhaleIntentCloseShort {
		t.Fatal(intent)
	}
	intent, _ = ClassifyWhaleIntent(WhaleSideSell, 100, 101)
	if intent != WhaleIntentOpenShort {
		t.Fatal(intent)
	}
	intent, _ = ClassifyWhaleIntent(WhaleSideSell, 100, 99)
	if intent != WhaleIntentCloseLong {
		t.Fatal(intent)
	}
	intent, _ = ClassifyWhaleIntent(WhaleSideBuy, 100, 100.01)
	if intent != WhaleIntentUnclear {
		t.Fatal("flat oi", intent)
	}
}

func TestAnnotateWhaleFollow_AfterWindow(t *testing.T) {
	t0 := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	ev := WhaleEvent{
		Exchange: ExchangeBinance, Symbol: "BTCUSDT", Kind: WhaleKindTrade,
		Side: WhaleSideBuy, AvgPrice: 100, FirstTime: t0, LastTime: t0,
	}
	oi := []FuturesSnapshot{
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", SampledAt: t0.Add(-5 * time.Minute), Value: 1000},
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", SampledAt: t0.Add(5 * time.Minute), Value: 1030},
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", SampledAt: t0.Add(20 * time.Minute), Value: 1040},
	}
	marks := []WhaleMark{{Time: t0.Add(15 * time.Minute), Price: 102}}
	AnnotateWhaleFollow(&ev, oi, marks, t0.Add(20*time.Minute))
	if ev.Intent != WhaleIntentOpenLong {
		t.Fatal(ev.Intent)
	}
	if !ev.AfterReady || ev.PriceAfterPct < 1.9 || ev.PriceAfterPct > 2.1 {
		t.Fatalf("price after %v ready %v", ev.PriceAfterPct, ev.AfterReady)
	}
	if ev.OIAfterPct < 0.9 || ev.OIAfterPct > 1.1 {
		t.Fatalf("oi after %v", ev.OIAfterPct)
	}
}
