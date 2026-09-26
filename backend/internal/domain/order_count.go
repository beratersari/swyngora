package domain

import (
	"fmt"
	"sort"
	"time"
)

const (
	OrderCount1m  = "1m"
	OrderCount5m  = "5m"
	OrderCount15m = "15m"

	OrderCountLeanBuy      = "buy"
	OrderCountLeanSell     = "sell"
	OrderCountLeanBalanced = "balanced"

	orderCountLeanBand        = 0.10
	orderCountOverviewDefault = 20
	orderCountOverviewMin     = 5
	orderCountOverviewMax     = 30
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

// OrderCountMarketWindow is one lookback summed across the followed coins.
// Buy/sell counts are added first, then the ratio is divided. Coin counts
// are one vote per pair, so a busy coin cannot hide how many names lean.
type OrderCountMarketWindow struct {
	Window        string
	BuyCount      int
	SellCount     int
	Total         int
	Delta         int
	Ratio         float64
	Lean          string
	BuyCoins      int
	SellCoins     int
	BalancedCoins int
	QuietCoins    int
	Coins         int
	Complete      bool
	Summary       string
}

// OrderCountOverview is the whole-market print-count ratio.
type OrderCountOverview struct {
	Exchange string
	Quote    string
	Universe int
	AsOf     time.Time
	Windows  []OrderCountMarketWindow
	Coins    []OrderCountVenue
	Summary  string
	Note     string
}

// ParseOrderCountOverviewLimit clamps how many liquid pairs the desk follows.
func ParseOrderCountOverviewLimit(n int) int {
	if n <= 0 {
		return orderCountOverviewDefault
	}
	if n < orderCountOverviewMin {
		return orderCountOverviewMin
	}
	if n > orderCountOverviewMax {
		return orderCountOverviewMax
	}
	return n
}

// BuildOrderCountOverview sums print counts across coins, then divides.
// Coins are ordered by the 5m ratio, most buy-heavy first.
func BuildOrderCountOverview(exchange, quote string, coins []OrderCountVenue, now time.Time) OrderCountOverview {
	usable := make([]OrderCountVenue, 0, len(coins))
	for _, c := range coins {
		if c.Error != "" || c.Symbol == "" {
			continue
		}
		usable = append(usable, c)
	}
	out := OrderCountOverview{
		Exchange: exchange,
		Quote:    quote,
		Universe: len(usable),
		AsOf:     now.UTC(),
		Windows:  make([]OrderCountMarketWindow, 0, len(OrderCountWindows)),
		Coins:    usable,
	}
	for _, w := range OrderCountWindows {
		out.Windows = append(out.Windows, summarizeOrderCountMarket(usable, w.ID))
	}
	sort.SliceStable(out.Coins, func(i, j int) bool {
		ri, rj := orderCountWindowRatio(out.Coins[i], OrderCount5m), orderCountWindowRatio(out.Coins[j], OrderCount5m)
		if ri != rj {
			return ri > rj
		}
		ti, tj := orderCountWindowTotal(out.Coins[i], OrderCount5m), orderCountWindowTotal(out.Coins[j], OrderCount5m)
		if ti != tj {
			return ti > tj
		}
		return out.Coins[i].Symbol < out.Coins[j].Symbol
	})
	out.Summary = explainOrderCountOverview(out)
	return out
}

func summarizeOrderCountMarket(coins []OrderCountVenue, window string) OrderCountMarketWindow {
	var buy, sell, buyCoins, sellCoins, balanced, quiet int
	complete := false
	for _, c := range coins {
		win, ok := orderCountWindow(c, window)
		if !ok {
			continue
		}
		buy += win.BuyCount
		sell += win.SellCount
		if win.Complete {
			complete = true
		}
		if win.Total == 0 {
			quiet++
			continue
		}
		switch win.Lean {
		case OrderCountLeanBuy:
			buyCoins++
		case OrderCountLeanSell:
			sellCoins++
		default:
			balanced++
		}
	}
	stat := orderCountStat(window, buy, sell, complete)
	out := OrderCountMarketWindow{
		Window: stat.Window, BuyCount: stat.BuyCount, SellCount: stat.SellCount,
		Total: stat.Total, Delta: stat.Delta, Ratio: stat.Ratio, Lean: stat.Lean,
		BuyCoins: buyCoins, SellCoins: sellCoins, BalancedCoins: balanced, QuietCoins: quiet,
		Coins: len(coins), Complete: complete,
	}
	out.Summary = explainOrderCountMarketWindow(out)
	return out
}

func orderCountWindow(v OrderCountVenue, window string) (OrderCountWindow, bool) {
	for _, w := range v.Windows {
		if w.Window == window {
			return w, true
		}
	}
	return OrderCountWindow{}, false
}

func orderCountWindowRatio(v OrderCountVenue, window string) float64 {
	w, ok := orderCountWindow(v, window)
	if !ok {
		return 0
	}
	return w.Ratio
}

func orderCountWindowTotal(v OrderCountVenue, window string) int {
	w, ok := orderCountWindow(v, window)
	if !ok {
		return 0
	}
	return w.Total
}

func explainOrderCountMarketWindow(w OrderCountMarketWindow) string {
	if w.Total == 0 {
		return fmt.Sprintf("%s: no aggressive prints across %d coins.", w.Window, w.Coins)
	}
	lean := "balanced"
	switch w.Lean {
	case OrderCountLeanBuy:
		lean = "more market buys"
	case OrderCountLeanSell:
		lean = "more market sells"
	}
	return fmt.Sprintf("%s: %d buys, %d sells, ratio %+.2f (%s). %d coins buy, %d sell, %d balanced, %d quiet.",
		w.Window, w.BuyCount, w.SellCount, w.Ratio, lean, w.BuyCoins, w.SellCoins, w.BalancedCoins, w.QuietCoins)
}

func explainOrderCountOverview(r OrderCountOverview) string {
	for _, w := range r.Windows {
		if w.Window == OrderCount5m && w.Summary != "" {
			return w.Summary
		}
	}
	if len(r.Windows) > 0 {
		return r.Windows[0].Summary
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
