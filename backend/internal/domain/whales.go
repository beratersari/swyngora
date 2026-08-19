package domain

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"
)

const (
	WhaleKindTrade       = "trade"
	WhaleKindLiquidation = "liquidation"

	WhaleSideBuy   = TakerSideBuy
	WhaleSideSell  = TakerSideSell
	WhaleSideLong  = LiquidationSideLong
	WhaleSideShort = LiquidationSideShort

	DefaultWhaleMinNotional = 100_000.0
	MinWhaleMinNotional     = 10_000.0
	MaxWhaleMinNotional     = 10_000_000.0
	DefaultWhaleLimit       = 30
	MaxWhaleLimit           = 100
	DefaultWhaleScanSymbols = 25
	whaleClusterGap         = 2 * time.Second
	whaleUnusualMcapPct     = 0.05 // 0.05% of circulating mcap
)

// WhaleEvent is one clustered print or a large liquidation.
type WhaleEvent struct {
	Exchange        Exchange
	Symbol          string
	Kind            string // trade | liquidation
	Side            string // buy | sell | long | short
	Position        string // long | short for futures trades (taker buy=long)
	AvgPrice        float64
	Quantity        float64
	Notional        float64
	FirstTime       time.Time
	LastTime        time.Time
	Prints          int
	MarketCap       float64
	NotionalMcapPct float64
	Unusual         bool // large versus this coin's market cap
}

// WhaleReport is the API result, biggest first.
type WhaleReport struct {
	Symbol      string
	Exchange    string
	AsOf        time.Time
	MinNotional float64
	Events      []WhaleEvent
	Summary     string
	Note        string
}

// RecentPrintPort loads the newest aggressive fills for one venue+symbol.
type RecentPrintPort interface {
	GetRecentPrints(ctx context.Context, symbol string) ([]TakerPrint, error)
}

// ParseWhaleMinNotional clamps the size floor.
func ParseWhaleMinNotional(v float64) float64 {
	if v <= 0 {
		return DefaultWhaleMinNotional
	}
	if v < MinWhaleMinNotional {
		return MinWhaleMinNotional
	}
	if v > MaxWhaleMinNotional {
		return MaxWhaleMinNotional
	}
	return v
}

// ParseWhaleLimit clamps result size.
func ParseWhaleLimit(n int) int {
	if n <= 0 {
		return DefaultWhaleLimit
	}
	if n > MaxWhaleLimit {
		return MaxWhaleLimit
	}
	return n
}

// ClusterWhalePrints groups nearby same-side fills and drops those under minNotional.
func ClusterWhalePrints(prints []TakerPrint, minNotional float64, gap time.Duration) []WhaleEvent {
	if gap <= 0 {
		gap = whaleClusterGap
	}
	if minNotional <= 0 {
		minNotional = DefaultWhaleMinNotional
	}
	type key struct {
		ex   Exchange
		sym  string
		side string
	}
	sorted := append([]TakerPrint(nil), prints...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if !sorted[i].Time.Equal(sorted[j].Time) {
			return sorted[i].Time.Before(sorted[j].Time)
		}
		return sorted[i].Notional > sorted[j].Notional
	})
	open := map[key]*WhaleEvent{}
	out := make([]WhaleEvent, 0, 16)
	flush := func(k key) {
		ev, ok := open[k]
		if !ok || ev == nil {
			return
		}
		delete(open, k)
		if ev.Quantity > 0 {
			ev.AvgPrice = ev.Notional / ev.Quantity
		}
		if ev.Notional >= minNotional {
			out = append(out, *ev)
		}
	}
	for _, p := range sorted {
		if !IsFinitePositive(p.Notional) || p.Time.IsZero() || p.Side == "" {
			continue
		}
		p.Symbol = NormalizeLiquidationSymbol(p.Symbol)
		k := key{ex: p.Exchange, sym: p.Symbol, side: p.Side}
		cur := open[k]
		if cur != nil && !p.Time.After(cur.LastTime.Add(gap)) {
			cur.Notional += p.Notional
			cur.Quantity += p.Quantity
			cur.Prints++
			if p.Time.After(cur.LastTime) {
				cur.LastTime = p.Time
			}
			if p.Time.Before(cur.FirstTime) {
				cur.FirstTime = p.Time
			}
			continue
		}
		if cur != nil {
			flush(k)
		}
		pos := WhaleSideLong
		if p.Side == TakerSideSell {
			pos = WhaleSideShort
		}
		open[k] = &WhaleEvent{
			Exchange: p.Exchange, Symbol: p.Symbol, Kind: WhaleKindTrade,
			Side: p.Side, Position: pos, Quantity: p.Quantity, Notional: p.Notional,
			FirstTime: p.Time, LastTime: p.Time, Prints: 1,
		}
	}
	for k := range open {
		flush(k)
	}
	return out
}

// WhaleFromLiquidation maps a forced close into a whale row.
func WhaleFromLiquidation(e LiquidationEvent, minNotional float64) (WhaleEvent, bool) {
	if e.Notional < minNotional || e.Notional <= 0 {
		return WhaleEvent{}, false
	}
	return WhaleEvent{
		Exchange: e.Exchange, Symbol: NormalizeLiquidationSymbol(e.Symbol),
		Kind: WhaleKindLiquidation, Side: e.Side, Position: e.Side,
		AvgPrice: e.Price, Quantity: e.Quantity, Notional: e.Notional,
		FirstTime: e.Time, LastTime: e.Time, Prints: 1,
	}, true
}

// AnnotateWhaleMcap sets market-cap context and the unusual flag.
func AnnotateWhaleMcap(ev *WhaleEvent, mcap float64) {
	if ev == nil {
		return
	}
	ev.MarketCap = mcap
	if mcap > 0 && ev.Notional > 0 {
		ev.NotionalMcapPct = ev.Notional / mcap * 100
		if ev.NotionalMcapPct >= whaleUnusualMcapPct {
			ev.Unusual = true
		}
	}
}

// SortWhalesBiggestFirst orders by notional, then unusual, then time.
func SortWhalesBiggestFirst(ev []WhaleEvent) {
	sort.SliceStable(ev, func(i, j int) bool {
		if ev[i].Notional != ev[j].Notional {
			return ev[i].Notional > ev[j].Notional
		}
		if ev[i].Unusual != ev[j].Unusual {
			return ev[i].Unusual
		}
		return ev[i].LastTime.After(ev[j].LastTime)
	})
}

// ExplainWhales writes a short biggest-first read.
func ExplainWhales(events []WhaleEvent, symbol string) string {
	if len(events) == 0 {
		if symbol != "" {
			return prettyBase(symbol) + ": no prints above the size floor in the recent tape."
		}
		return "No whale prints above the size floor in the recent tape."
	}
	top := events[0]
	label := whaleLabel(top)
	head := fmt.Sprintf("Largest: %s %s on %s at %s (total %s, %d prints, %s–%s).",
		formatQty(top.Notional), label, top.Symbol, formatQty(top.AvgPrice),
		formatQty(top.Quantity), top.Prints,
		top.FirstTime.UTC().Format("15:04:05"), top.LastTime.UTC().Format("15:04:05"))
	var unusual int
	for _, e := range events {
		if e.Unusual {
			unusual++
		}
	}
	if unusual > 0 {
		head += fmt.Sprintf(" %d print(s) are large versus that coin's market cap.", unusual)
	}
	return head
}

func whaleLabel(e WhaleEvent) string {
	if e.Kind == WhaleKindLiquidation {
		if e.Side == WhaleSideLong {
			return "long liquidation"
		}
		if e.Side == WhaleSideShort {
			return "short liquidation"
		}
		return "liquidation"
	}
	if e.Side == WhaleSideBuy {
		return "buy (aggressive long)"
	}
	if e.Side == WhaleSideSell {
		return "sell (aggressive short)"
	}
	return e.Side
}

// ClampWhaleScanSymbols bounds how many coins a market-wide scan may hit.
func ClampWhaleScanSymbols(n int) int {
	if n <= 0 {
		return DefaultWhaleScanSymbols
	}
	if n > 40 {
		return 40
	}
	return n
}

// IsFinitePositive is a tiny guard for prices and notionals.
func IsFinitePositive(v float64) bool {
	return v > 0 && !math.IsNaN(v) && !math.IsInf(v, 0)
}
