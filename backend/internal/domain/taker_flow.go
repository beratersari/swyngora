package domain

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"
)

const (
	TakerWindow5m = LiquidationWindow5m
	TakerWindow1h = LiquidationWindow1h
	TakerWindow4h = LiquidationWindow4h

	TakerSideBuy  = "buy"
	TakerSideSell = "sell"
	TakerSideEven = "balanced"
)

// TakerWindows are the aggressive-flow lookbacks.
var TakerWindows = []struct {
	ID  string
	Dur time.Duration
}{
	{TakerWindow5m, 5 * time.Minute},
	{TakerWindow1h, time.Hour},
	{TakerWindow4h, 4 * time.Hour},
}

// TakerPrint is one aggressive (taker) futures fill.
type TakerPrint struct {
	Exchange Exchange
	Symbol   string
	Side     string // buy | sell — taker side
	Price    float64
	Quantity float64
	Notional float64
	Time     time.Time
}

// TakerBucket is buy/sell notional in one time bucket (usually 1m).
type TakerBucket struct {
	Exchange     Exchange
	Symbol       string
	Start        time.Time
	BuyNotional  float64
	SellNotional float64
}

// TakerWindowFlow is aggressive volume in one lookback.
type TakerWindowFlow struct {
	Window       string
	BuyNotional  float64
	SellNotional float64
	Delta        float64 // buy − sell
	DeltaPct     float64 // (buy−sell)/(buy+sell)*100
	BuyShare     float64 // 0–1
	Dominant     string  // buy | sell | balanced
	Complete     bool
	HaveSec      int64 // how long this process has actually collected
	NeedSec      int64 // window length
}

// TakerVenueFlow is one venue's taker picture plus a short read.
type TakerVenueFlow struct {
	Exchange Exchange
	Symbol   string
	Price    float64
	Windows  []TakerWindowFlow
	Dominant string // from the 5m window when present
	Summary  string
	Error    string
}

// TakerFlowReport is the API result.
type TakerFlowReport struct {
	Symbol   string
	Exchange string // binance | bybit | all
	AsOf     time.Time
	Venues   []TakerVenueFlow
	Combined *TakerVenueFlow
	Note     string
}

// TakerFlowPort loads taker buy/sell volume for one futures venue.
type TakerFlowPort interface {
	GetTakerFlow(ctx context.Context, symbol string) (*TakerVenueFlow, error)
}

// ParseTakerExchange is binance, bybit, or all.
func ParseTakerExchange(raw string) (string, error) {
	return ParseLiquidationExchange(raw)
}

// TakerDominant labels who hit the market more. 5% band around even is balanced.
func TakerDominant(buy, sell float64) string {
	tot := buy + sell
	if tot <= 0 {
		return TakerSideEven
	}
	share := buy / tot
	if share >= 0.525 {
		return TakerSideBuy
	}
	if share <= 0.475 {
		return TakerSideSell
	}
	return TakerSideEven
}

// SummarizeTakerWindow totals prints or buckets since cut.
func SummarizeTakerWindow(buy, sell float64, windowID string, complete bool) TakerWindowFlow {
	tot := buy + sell
	out := TakerWindowFlow{
		Window:       windowID,
		BuyNotional:  buy,
		SellNotional: sell,
		Delta:        buy - sell,
		Dominant:     TakerDominant(buy, sell),
		Complete:     complete,
	}
	if tot > 0 {
		out.BuyShare = buy / tot
		out.DeltaPct = (buy - sell) / tot * 100
	}
	return out
}

// BuildTakerVenueFlow builds windows from buckets (1m or 5m). A bucket is
// counted when it overlaps the lookback (so a 5m bar that started 6m ago
// still fills the current 5m window).
func BuildTakerVenueFlow(ex Exchange, symbol string, buckets []TakerBucket, now time.Time, started time.Time) TakerVenueFlow {
	return BuildTakerVenueFlowBucket(ex, symbol, buckets, now, started, time.Minute)
}

// BuildTakerVenueFlowBucket is BuildTakerVenueFlow with an explicit bar size.
func BuildTakerVenueFlowBucket(ex Exchange, symbol string, buckets []TakerBucket, now, started time.Time, bar time.Duration) TakerVenueFlow {
	symbol = NormalizeLiquidationSymbol(symbol)
	now = now.UTC()
	if bar <= 0 {
		bar = time.Minute
	}
	out := TakerVenueFlow{
		Exchange: ex,
		Symbol:   symbol,
		Windows:  make([]TakerWindowFlow, 0, len(TakerWindows)),
	}
	var newest time.Time
	for _, b := range buckets {
		if newest.IsZero() || b.Start.After(newest) {
			newest = b.Start
		}
	}
	for _, w := range TakerWindows {
		cut := now.Add(-w.Dur)
		var buy, sell float64
		for _, b := range buckets {
			end := b.Start.Add(bar)
			// Overlap the lookback, or always keep the newest bar so a just-closed
			// 5m print is not dropped when "now" sits a few minutes into the next bar.
			if end.After(cut) || (!newest.IsZero() && b.Start.Equal(newest)) {
				buy += b.BuyNotional
				sell += b.SellNotional
			}
		}
		have := time.Duration(0)
		if !started.IsZero() {
			have = now.Sub(started)
			if have < 0 {
				have = 0
			}
			if have > w.Dur {
				have = w.Dur
			}
		}
		complete := have >= w.Dur && !started.IsZero()
		win := SummarizeTakerWindow(buy, sell, w.ID, complete)
		win.HaveSec = int64(have / time.Second)
		win.NeedSec = int64(w.Dur / time.Second)
		out.Windows = append(out.Windows, win)
	}
	if len(out.Windows) > 0 {
		out.Dominant = out.Windows[0].Dominant
	}
	return out
}

// CombineTakerVenues sums notional windows across venues (never averages %).
func CombineTakerVenues(symbol string, venues []TakerVenueFlow) *TakerVenueFlow {
	if len(venues) == 0 {
		return nil
	}
	out := &TakerVenueFlow{
		Exchange: "all",
		Symbol:   NormalizeLiquidationSymbol(symbol),
		Windows:  make([]TakerWindowFlow, 0, len(TakerWindows)),
	}
	for _, w := range TakerWindows {
		var buy, sell float64
		complete := true
		any := false
		for _, v := range venues {
			for _, tw := range v.Windows {
				if tw.Window != w.ID {
					continue
				}
				buy += tw.BuyNotional
				sell += tw.SellNotional
				any = true
				if !tw.Complete {
					complete = false
				}
			}
			if v.Price > out.Price {
				out.Price = v.Price
			}
		}
		if !any {
			complete = false
		}
		out.Windows = append(out.Windows, SummarizeTakerWindow(buy, sell, w.ID, complete))
	}
	if len(out.Windows) > 0 {
		out.Dominant = out.Windows[0].Dominant
	}
	return out
}

// TakerFlowContext is optional corroboration for the short summary.
type TakerFlowContext struct {
	PriceChange1hPct float64
	OIChange1hPct    float64
	FundingRate      float64
	LongShare        float64
	Positioning      string
}

// ExplainTakerFlow writes a short read of aggressive flow plus other futures context.
func ExplainTakerFlow(v TakerVenueFlow, ctx TakerFlowContext) string {
	w := pickTakerWindow(v.Windows, TakerWindow5m)
	w1 := pickTakerWindow(v.Windows, TakerWindow1h)
	who := "neither side"
	switch w.Dominant {
	case TakerSideBuy:
		who = "buyers"
	case TakerSideSell:
		who = "sellers"
	}
	ex := string(v.Exchange)
	if ex == "" || ex == "all" {
		ex = "Combined"
	} else if ex == string(ExchangeBinance) {
		ex = "Binance"
	} else if ex == string(ExchangeBybit) {
		ex = "Bybit"
	}
	head := fmt.Sprintf("%s: %s are more aggressive in the last 5m (buy %s vs sell %s USDT, delta %s).",
		ex, who, formatQty(w.BuyNotional), formatQty(w.SellNotional), FormatSignedQty(w.Delta))
	if w.Dominant == TakerSideEven {
		head = fmt.Sprintf("%s: buy and sell taker volume are roughly even in the last 5m (buy %s vs sell %s USDT).",
			ex, formatQty(w.BuyNotional), formatQty(w.SellNotional))
	}
	if w1.BuyNotional+w1.SellNotional > 0 && w1.Dominant != w.Dominant && w.Dominant != TakerSideEven {
		head += fmt.Sprintf(" Over 1h the edge is %s.", w1.Dominant)
	}

	bits := make([]string, 0, 3)
	if !math.IsNaN(ctx.PriceChange1hPct) && math.Abs(ctx.PriceChange1hPct) >= 0.15 {
		bits = append(bits, fmt.Sprintf("price %s over 1h", FormatSignedPct(ctx.PriceChange1hPct)))
	}
	if !math.IsNaN(ctx.OIChange1hPct) && math.Abs(ctx.OIChange1hPct) >= 0.2 {
		bits = append(bits, fmt.Sprintf("OI %s", FormatSignedPct(ctx.OIChange1hPct)))
	}
	if ctx.FundingRate > 1e-12 {
		bits = append(bits, "longs pay funding")
	} else if ctx.FundingRate < -1e-12 {
		bits = append(bits, "shorts pay funding")
	}
	if ctx.Positioning != "" && ctx.Positioning != RegimeNeutral {
		bits = append(bits, RegimeLabel(ctx.Positioning))
	}
	if len(bits) == 0 {
		return head + " This is taker (aggressive) volume, not the account long/short ratio."
	}
	meaning := takerMeaning(w.Dominant, ctx)
	return head + " Together with " + joinAnd(bits) + ": " + meaning
}

func takerMeaning(dom string, ctx TakerFlowContext) string {
	priceUp := !math.IsNaN(ctx.PriceChange1hPct) && ctx.PriceChange1hPct > 0.15
	priceDown := !math.IsNaN(ctx.PriceChange1hPct) && ctx.PriceChange1hPct < -0.15
	oiUp := !math.IsNaN(ctx.OIChange1hPct) && ctx.OIChange1hPct > 0.25
	switch {
	case dom == TakerSideBuy && priceUp && oiUp:
		return "aggressive buying with rising price and OI fits long buildup — new longs hitting the offer."
	case dom == TakerSideSell && priceDown && oiUp:
		return "aggressive selling with falling price and rising OI fits short buildup — new shorts hitting the bid."
	case dom == TakerSideBuy && priceUp && !oiUp:
		return "buyers are lifting price while OI is not expanding — often short covering rather than new longs."
	case dom == TakerSideSell && priceDown && !oiUp:
		return "sellers are pressing price while OI is not expanding — often long unwinding rather than new shorts."
	case dom == TakerSideBuy:
		return "takers are lifting offers. That is demand now, separate from how many accounts are already long."
	case dom == TakerSideSell:
		return "takers are hitting bids. That is supply now, separate from how many accounts are already short."
	default:
		return "no clear taker imbalance; look at positioning and funding for the slower crowd."
	}
}

func pickTakerWindow(ws []TakerWindowFlow, id string) TakerWindowFlow {
	for _, w := range ws {
		if w.Window == id {
			return w
		}
	}
	return TakerWindowFlow{Window: id, Dominant: TakerSideEven}
}

func joinAnd(parts []string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	case 2:
		return parts[0] + " and " + parts[1]
	default:
		return parts[0] + ", " + parts[1] + ", and " + parts[2]
	}
}

// SortTakerBuckets oldest-first.
func SortTakerBuckets(in []TakerBucket) []TakerBucket {
	sort.SliceStable(in, func(i, j int) bool { return in[i].Start.Before(in[j].Start) })
	return in
}
