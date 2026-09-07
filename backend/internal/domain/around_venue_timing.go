package domain

import (
	"fmt"
	"math"
	"time"
)

const (
	AroundVenueSame           = "same"
	AroundVenueBinanceEarlier = "binance_earlier"
	AroundVenueBybitEarlier   = "bybit_earlier"
	AroundVenueUnknown        = "unknown"

	aroundVenueLookback   = 30 * time.Minute
	aroundVenueSameWindow = 2 * time.Minute
)

// AroundVenueTiming compares when Binance and Bybit started the price move.
type AroundVenueTiming struct {
	Agreement    string
	BinanceStart time.Time
	BybitStart   time.Time
	Lead         time.Duration
	LeadLabel    string
	Summary      string
}

// FindAroundPriceRunStart is when this venue's during impulse first printed.
func FindAroundPriceRunStart(bars []AroundBar, plan AroundPlan) time.Time {
	during := FilterAroundBars(bars, plan.DuringFrom, plan.DuringTo)
	if len(during) == 0 {
		return time.Time{}
	}
	price := aroundPriceOf(during)
	if price.Direction != CVDDirUp && price.Direction != CVDDirDown {
		return plan.DuringFrom
	}
	lookFrom := plan.DuringFrom.Add(-aroundVenueLookback)
	if lookFrom.Before(plan.BeforeFrom) {
		lookFrom = plan.BeforeFrom
	}
	look := FilterAroundBars(bars, lookFrom, plan.DuringTo)
	if len(look) == 0 {
		return plan.DuringFrom
	}
	var pre AroundBar
	for i := len(look) - 1; i >= 0; i-- {
		if look[i].Time.Before(plan.DuringFrom) {
			pre = look[i]
			break
		}
	}
	if pre.Close <= 0 {
		pre = look[0]
	}
	thresh := math.Max(0.12, 0.18*math.Abs(price.ChangePct))
	for _, b := range look {
		if b.Close <= 0 || pre.Close <= 0 {
			continue
		}
		ch := (b.Close - pre.Close) / pre.Close * 100
		if price.Direction == CVDDirUp && ch >= thresh {
			return b.Time.UTC()
		}
		if price.Direction == CVDDirDown && ch <= -thresh {
			return b.Time.UTC()
		}
	}
	return plan.DuringFrom
}

// CompareAroundVenueTiming says whether Binance and Bybit started together.
func CompareAroundVenueTiming(venues []AroundVenue) *AroundVenueTiming {
	var bn, by time.Time
	for _, v := range venues {
		switch v.Exchange {
		case ExchangeBinance:
			bn = v.MoveStart
		case ExchangeBybit:
			by = v.MoveStart
		}
	}
	if bn.IsZero() || by.IsZero() {
		return nil
	}
	out := &AroundVenueTiming{BinanceStart: bn.UTC(), BybitStart: by.UTC()}
	delta := bn.Sub(by)
	if delta < 0 {
		delta = -delta
	}
	out.Lead = delta
	if delta <= aroundVenueSameWindow {
		out.Agreement = AroundVenueSame
		out.Summary = "Binance and Bybit started the price move at about the same time."
		return out
	}
	out.LeadLabel = formatOptionsLead(delta)
	if bn.Before(by) {
		out.Agreement = AroundVenueBinanceEarlier
		out.Summary = fmt.Sprintf("Binance started the price move %s before Bybit.", out.LeadLabel)
		return out
	}
	out.Agreement = AroundVenueBybitEarlier
	out.Summary = fmt.Sprintf("Bybit started the price move %s before Binance.", out.LeadLabel)
	return out
}
