package market

import (
	"context"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestCapPumpScanHitsByTotalEvents(t *testing.T) {
	mk := func(sym string, rets ...float64) PumpScanHit {
		evs := make([]domain.PumpEvent, len(rets))
		for i, r := range rets {
			evs[i] = domain.PumpEvent{ReturnPct: r}
		}
		best := 0.0
		if len(evs) > 0 {
			best = evs[0].ReturnPct
		}
		return PumpScanHit{Symbol: sym, Events: evs, BestReturnPct: best}
	}

	hits := []PumpScanHit{
		mk("A", 20, 15, 12), // 3 events
		mk("B", 18, 10),     // 2
		mk("C", 16, 14, 11), // 3
	}

	t.Run("no cap leaves unchanged", func(t *testing.T) {
		got := capPumpScanHitsByTotalEvents(hits, 0)
		if len(got) != 3 {
			t.Fatalf("len=%d", len(got))
		}
		if totalEvents(got) != 8 {
			t.Fatalf("total events=%d", totalEvents(got))
		}
	})

	t.Run("cap trims later hits and partial last hit", func(t *testing.T) {
		// budget 4: keep all 3 from A, then 1 from B; drop rest of B and all of C
		got := capPumpScanHitsByTotalEvents(hits, 4)
		if totalEvents(got) != 4 {
			t.Fatalf("total events=%d want 4; hits=%+v", totalEvents(got), got)
		}
		if len(got) != 2 {
			t.Fatalf("hit count=%d want 2", len(got))
		}
		if got[0].Symbol != "A" || len(got[0].Events) != 3 {
			t.Fatalf("first hit=%+v", got[0])
		}
		if got[1].Symbol != "B" || len(got[1].Events) != 1 {
			t.Fatalf("second hit=%+v", got[1])
		}
		if got[1].BestReturnPct != 18 {
			t.Fatalf("best return after truncate=%v", got[1].BestReturnPct)
		}
	})

	t.Run("cap of one keeps single best event", func(t *testing.T) {
		got := capPumpScanHitsByTotalEvents(hits, 1)
		if totalEvents(got) != 1 || len(got) != 1 || got[0].Symbol != "A" {
			t.Fatalf("%+v", got)
		}
	})

	t.Run("cap larger than total keeps all", func(t *testing.T) {
		got := capPumpScanHitsByTotalEvents(hits, 100)
		if totalEvents(got) != 8 || len(got) != 3 {
			t.Fatalf("total=%d hits=%d", totalEvents(got), len(got))
		}
	})
}

func totalEvents(hits []PumpScanHit) int {
	n := 0
	for _, h := range hits {
		n += len(h.Events)
	}
	return n
}

// multiPumpCandles builds a series with several close-to-close jumps above minReturnPct.
func multiPumpCandles(nPumps int) []domain.Candle {
	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	// start flat, then alternate flat + big up bar for each pump
	out := make([]domain.Candle, 0, nPumps*2+1)
	price := 100.0
	t0 := base
	// seed bar
	out = append(out, domain.Candle{
		OpenTime: t0, CloseTime: t0.Add(15 * time.Minute),
		Open: "100", High: "100", Low: "100", Close: "100", Volume: "10",
	})
	t0 = t0.Add(15 * time.Minute)
	for i := 0; i < nPumps; i++ {
		// +20% bar
		next := price * 1.20
		out = append(out, domain.Candle{
			OpenTime: t0, CloseTime: t0.Add(15 * time.Minute),
			Open: fstr(price), High: fstr(next), Low: fstr(price), Close: fstr(next), Volume: "50",
		})
		price = next
		t0 = t0.Add(15 * time.Minute)
		// flat bar so next pump is isolated
		out = append(out, domain.Candle{
			OpenTime: t0, CloseTime: t0.Add(15 * time.Minute),
			Open: fstr(price), High: fstr(price), Low: fstr(price), Close: fstr(price), Volume: "10",
		})
		t0 = t0.Add(15 * time.Minute)
	}
	return out
}

func fstr(f float64) string {
	return FormatPumpReturn(f)
}

func TestScanPumpEvents_ResolvedDefaultsAndMaxTotalEvents(t *testing.T) {
	// Three symbols, each with 3+ pump bars; MaxEventsPerSymbol default 3 → up to 9 events.
	// maxTotalEvents=4 must yield total events ≤ 4 (not 4 hits).
	spot := []domain.SpotMarket{
		{Symbol: "AAAUSDT", BaseAsset: "AAA", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "3000", Volume: "10", LastPrice: "1"},
		{Symbol: "BBBUSDT", BaseAsset: "BBB", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "2000", Volume: "10", LastPrice: "1"},
		{Symbol: "CCCUSDT", BaseAsset: "CCC", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "1000", Volume: "10", LastPrice: "1"},
	}
	fm := &fakeMarket{
		candles: multiPumpCandles(4),
		spot:    spot,
	}
	svc := New(fm, nil)

	// Empty query → defaults applied and reflected on result
	res, err := svc.ScanPumpEvents(context.Background(), PumpScanQuery{
		// omit all optionals except force low threshold so pumps detect
		MinReturnPct:   5,
		LookbackHours:  0, // service default 24; use LimitBars via DetectPump with lookback
		// Use lookback 0 + limitBars path: DetectPump uses limit when lookback 0.
		// Scan uses LookbackHours default 24 which converts to many bars — fake returns same candles.
		MaxTotalEvents: 4,
		SymbolLimit:    3,
		// MaxEventsPerSymbol default 3
	})
	if err != nil {
		t.Fatal(err)
	}
	if res == nil {
		t.Fatal("nil result")
	}
	if res.Exchange != domain.ExchangeBinance {
		t.Fatalf("exchange=%s", res.Exchange)
	}
	if res.QuoteAsset != "USDT" {
		t.Fatalf("quote=%s", res.QuoteAsset)
	}
	if res.Interval != "15m" {
		t.Fatalf("interval=%s want 15m default", res.Interval)
	}
	if res.LookbackHours != 24 {
		t.Fatalf("lookbackHours=%v want 24", res.LookbackHours)
	}
	if res.MinReturnPct != 5 {
		t.Fatalf("minReturnPct=%v", res.MinReturnPct)
	}
	if res.WindowBars != 1 {
		t.Fatalf("windowBars=%d", res.WindowBars)
	}
	if res.Mode != domain.PumpModeCloseReturn {
		t.Fatalf("mode=%s", res.Mode)
	}
	if res.Direction != domain.PumpDirectionUp {
		t.Fatalf("direction=%s", res.Direction)
	}
	if res.SymbolLimit != 3 {
		t.Fatalf("symbolLimit=%d", res.SymbolLimit)
	}
	if res.MaxTotalEvents != 4 {
		t.Fatalf("maxTotalEvents=%d", res.MaxTotalEvents)
	}
	if totalEvents(res.Hits) > 4 {
		t.Fatalf("event total %d exceeds maxTotalEvents 4; hits=%+v", totalEvents(res.Hits), summarizeHits(res.Hits))
	}
	if totalEvents(res.Hits) == 0 {
		t.Fatal("expected some pump events from multiPumpCandles")
	}
	// Without the bug (cap by hits), 3 symbols would each keep 3 events = 9.
	// With maxTotalEvents=4 we must not return 9 events.
	if totalEvents(res.Hits) > 4 {
		t.Fatalf("maxTotalEvents not applied as event cap")
	}
}

func TestScanPumpEvents_DefaultMetadataWhenAllOmitted(t *testing.T) {
	fm := &fakeMarket{
		candles: multiPumpCandles(1),
		spot: []domain.SpotMarket{
			{Symbol: "BTCUSDT", BaseAsset: "BTC", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "1000", Volume: "10", LastPrice: "100"},
		},
	}
	svc := New(fm, nil)
	res, err := svc.ScanPumpEvents(context.Background(), PumpScanQuery{})
	if err != nil {
		t.Fatal(err)
	}
	// Defaults must be non-zero / non-empty
	if res.Exchange == "" || res.QuoteAsset == "" || res.Interval == "" {
		t.Fatalf("empty string metadata: exchange=%q quote=%q interval=%q", res.Exchange, res.QuoteAsset, res.Interval)
	}
	if res.LookbackHours == 0 || res.MinReturnPct == 0 || res.WindowBars == 0 {
		t.Fatalf("zero numeric metadata: lookback=%v minReturn=%v window=%d", res.LookbackHours, res.MinReturnPct, res.WindowBars)
	}
	if res.MaxTotalEvents != 30 {
		t.Fatalf("maxTotalEvents default=%d want 30", res.MaxTotalEvents)
	}
	if res.SymbolLimit != 15 {
		t.Fatalf("symbolLimit default=%d want 15", res.SymbolLimit)
	}
	if res.Mode == "" || res.Direction == "" {
		t.Fatalf("empty mode/direction: %q %q", res.Mode, res.Direction)
	}
}

func summarizeHits(hits []PumpScanHit) []string {
	out := make([]string, len(hits))
	for i, h := range hits {
		out[i] = h.Symbol + ":" + FormatPumpReturn(float64(len(h.Events)))
	}
	return out
}
