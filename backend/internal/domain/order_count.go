package domain

import (
	"fmt"
	"time"
)

const (
	OrderCount1m  = "1m"
	OrderCount5m  = "5m"
	OrderCount15m = "15m"

	OrderCountLeanBuy      = "buy"
	OrderCountLeanSell     = "sell"
	OrderCountLeanBalanced = "balanced"

	orderCountLeanBand = 0.10
)

// OrderCountWindows are the market-order count lookbacks.
var OrderCountWindows = []struct {
	ID  string
	Dur time.Duration
}{
	{OrderCount1m, time.Minute},
	{OrderCount5m, 5 * time.Minute},
	{OrderCount15m, 15 * time.Minute},
}

// OrderCountLookback is how far back prints are loaded.
func OrderCountLookback() time.Duration { return 15 * time.Minute }

// OrderCountWindow is buy vs sell print counts and their ratio.
type OrderCountWindow struct {
	Window    string
	BuyCount  int
	SellCount int
	Total     int
	Delta     int     // buy − sell
	Ratio     float64 // (buy − sell) / (buy + sell), from −1 to +1
	Lean      string  // buy | sell | balanced
	Complete  bool
	Summary   string
}

// OrderCountVenue is one venue's count ratio.
type OrderCountVenue struct {
	Exchange Exchange
	Symbol   string
	Windows  []OrderCountWindow
	Summary  string
	Error    string
}

// OrderCountReport is the API result.
type OrderCountReport struct {
	Symbol   string
	Exchange string
	AsOf     time.Time
	Venues   []OrderCountVenue
	Combined *OrderCountVenue
	Summary  string
	Note     string
}

// BuildOrderCount folds aggressive prints into 1m / 5m / 15m count ratios.
func BuildOrderCount(ex Exchange, symbol string, prints []TakerPrint, now time.Time) OrderCountVenue {
	symbol = NormalizeLiquidationSymbol(symbol)
	now = now.UTC()
	out := OrderCountVenue{Exchange: ex, Symbol: symbol, Windows: make([]OrderCountWindow, 0, len(OrderCountWindows))}
	var oldest time.Time
	clean := make([]TakerPrint, 0, len(prints))
	for _, p := range prints {
		if p.Time.IsZero() || p.Time.After(now) {
			continue
		}
		if p.Side != TakerSideBuy && p.Side != TakerSideSell {
			continue
		}
		p.Time = p.Time.UTC()
		clean = append(clean, p)
		if oldest.IsZero() || p.Time.Before(oldest) {
			oldest = p.Time
		}
	}
	for _, w := range OrderCountWindows {
		out.Windows = append(out.Windows, summarizeOrderCount(clean, w.ID, now.Add(-w.Dur), oldest))
	}
	out.Summary = explainOrderCountVenue(out)
	return out
}

// CombineOrderCount sums print counts, then recomputes the ratio.
func CombineOrderCount(symbol string, venues []OrderCountVenue) *OrderCountVenue {
	symbol = NormalizeLiquidationSymbol(symbol)
	out := &OrderCountVenue{Exchange: "all", Symbol: symbol, Windows: make([]OrderCountWindow, 0, len(OrderCountWindows))}
	usable := 0
	for _, v := range venues {
		if v.Error == "" {
			usable++
		}
	}
	if usable < 2 {
		return nil
	}
	for _, w := range OrderCountWindows {
		var buy, sell int
		complete := true
		for _, v := range venues {
			if v.Error != "" {
				continue
			}
			for _, win := range v.Windows {
				if win.Window != w.ID {
					continue
				}
				buy += win.BuyCount
				sell += win.SellCount
				if !win.Complete {
					complete = false
				}
			}
		}
		stat := orderCountStat(w.ID, buy, sell, complete)
		out.Windows = append(out.Windows, stat)
	}
	out.Summary = explainOrderCountVenue(*out)
	return out
}

func summarizeOrderCount(prints []TakerPrint, window string, from, oldest time.Time) OrderCountWindow {
	var buy, sell int
	for _, p := range prints {
		if p.Time.Before(from) {
			continue
		}
		if p.Side == TakerSideBuy {
			buy++
		} else {
			sell++
		}
	}
	complete := !oldest.IsZero() && !oldest.After(from.Add(time.Minute))
	return orderCountStat(window, buy, sell, complete)
}

func orderCountStat(window string, buy, sell int, complete bool) OrderCountWindow {
	total := buy + sell
	delta := buy - sell
	ratio := 0.0
	if total > 0 {
		ratio = float64(delta) / float64(total)
	}
	out := OrderCountWindow{
		Window: window, BuyCount: buy, SellCount: sell, Total: total,
		Delta: delta, Ratio: ratio, Lean: orderCountLean(ratio, total), Complete: complete,
	}
	out.Summary = explainOrderCountWindow(out)
	return out
}

func orderCountLean(ratio float64, total int) string {
	if total == 0 || ratio > -orderCountLeanBand && ratio < orderCountLeanBand {
		return OrderCountLeanBalanced
	}
	if ratio > 0 {
		return OrderCountLeanBuy
	}
	return OrderCountLeanSell
}

func explainOrderCountWindow(w OrderCountWindow) string {
	if w.Total == 0 {
		return fmt.Sprintf("%s: no aggressive prints.", w.Window)
	}
	lean := "balanced"
	switch w.Lean {
	case OrderCountLeanBuy:
		lean = "more market buys"
	case OrderCountLeanSell:
		lean = "more market sells"
	}
	return fmt.Sprintf("%s: %d buys, %d sells, ratio %+.2f (%s).", w.Window, w.BuyCount, w.SellCount, w.Ratio, lean)
}

func explainOrderCountVenue(v OrderCountVenue) string {
	if v.Error != "" {
		return v.Error
	}
	for _, w := range v.Windows {
		if w.Window == OrderCount5m && w.Summary != "" {
			return w.Summary
		}
	}
	if len(v.Windows) > 0 {
		return v.Windows[0].Summary
	}
	return "No aggressive print counts yet."
}

// ExplainOrderCountReport prefers the combined 5m line.
func ExplainOrderCountReport(r OrderCountReport) string {
	if r.Combined != nil && r.Combined.Summary != "" {
		return r.Combined.Summary
	}
	for _, v := range r.Venues {
		if v.Error == "" && v.Summary != "" {
			return string(v.Exchange) + ": " + v.Summary
		}
	}
	return "No aggressive print counts yet."
}
