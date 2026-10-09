package domain

import (
	"errors"
	"sort"
	"time"
)

const (
	// BinanceTapeBackfillLookback is how far the background walk asks
	// Binance USD-M aggTrades to go. The flow store can keep 30 days.
	// This cap is the public walk, not that retention.
	BinanceTapeBackfillLookback = 7 * 24 * time.Hour

	// BinanceTapeWindow is one aggTrades request. Binance requires
	// startTime and endTime to be less than one hour apart.
	BinanceTapeWindow = 59 * time.Minute

	// BinanceTapeMaxPages is how many 1000-trade pages one window may take
	// before the caller shrinks the window and tries again.
	BinanceTapeMaxPages = 40

	// BinanceTapeLiveGap is the recent edge the live sampler owns.
	// The backfill does not write those minutes.
	BinanceTapeLiveGap = 15 * time.Minute
)

// ErrTapeWindowFull means one aggTrades window still filled the page budget.
// The caller shrinks the window. The scan watermark stays put.
var ErrTapeWindowFull = errors.New("aggTrades window exceeded the page budget")

// TakerBucketsFromPrints sums aggressive buy and sell notional into closed minutes.
// A print with no notional or no time is left out. Minutes are oldest first.
func TakerBucketsFromPrints(prints []TakerPrint) []TakerBucket {
	type key struct {
		ex  Exchange
		sym string
		min int64
	}
	acc := map[key]*TakerBucket{}
	for _, p := range prints {
		if p.Notional <= 0 || p.Time.IsZero() {
			continue
		}
		sym := NormalizeLiquidationSymbol(p.Symbol)
		if sym == "" || p.Exchange == "" {
			continue
		}
		minute := p.Time.UTC().Truncate(time.Minute)
		k := key{ex: p.Exchange, sym: sym, min: minute.Unix()}
		b := acc[k]
		if b == nil {
			b = &TakerBucket{Exchange: p.Exchange, Symbol: sym, Start: minute}
			acc[k] = b
		}
		if p.Side == TakerSideSell {
			b.SellNotional += p.Notional
		} else {
			b.BuyNotional += p.Notional
		}
		AddTakerPrintSize(b, p.Side, p.Notional)
		b.SizeKnown = true
	}
	out := make([]TakerBucket, 0, len(acc))
	for _, b := range acc {
		out = append(out, *b)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Start.Equal(out[j].Start) {
			return string(out[i].Exchange) < string(out[j].Exchange)
		}
		return out[i].Start.Before(out[j].Start)
	})
	return out
}

// TapeBackfillEnd is the exclusive end of the next historical window.
// The live edge, the oldest stored minute, and a saved scan each pull it
// backward. The result is never after the live edge.
func TapeBackfillEnd(now, earliest, scan time.Time, hasEarliest, hasScan bool) time.Time {
	now = now.UTC()
	end := now.Add(-BinanceTapeLiveGap).Truncate(time.Minute)
	if hasEarliest {
		at := earliest.UTC().Truncate(time.Minute)
		if at.Before(end) {
			end = at
		}
	}
	if hasScan {
		at := scan.UTC()
		if at.Before(end) {
			end = at
		}
	}
	return end
}
