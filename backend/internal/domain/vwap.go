package domain

import (
	"fmt"
	"sort"
	"time"
)

// VWAPShare is one venue's weight in a combined VWAP.
type VWAPShare struct {
	Exchange Exchange
	Volume   float64
	SharePct float64
	VWAP     float64
}

// VWAPVenue is volume-weighted average price for one venue (or combined).
type VWAPVenue struct {
	Exchange    Exchange
	Symbol      string
	From        time.Time
	To          time.Time
	Interval    string
	VWAP        float64
	LastPrice   float64
	Distance    float64 // last − VWAP
	DistancePct float64
	Side        string // above | below | at
	Volume      float64
	BarCount    int
	High        float64
	Low         float64
	Shares      []VWAPShare
	Summary     string
	Error       string
}

// VWAPReport is the API result.
type VWAPReport struct {
	Symbol   string
	Exchange string
	Window   string
	From     time.Time
	To       time.Time
	AsOf     time.Time
	Venues   []VWAPVenue
	Combined *VWAPVenue
	Summary  string
	Note     string
}

// TypicalPrice is (high+low+close)/3, the usual candle VWAP print.
func TypicalPrice(high, low, close float64) float64 {
	n := 0
	var sum float64
	if high > 0 {
		sum += high
		n++
	}
	if low > 0 {
		sum += low
		n++
	}
	if close > 0 {
		sum += close
		n++
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

// ComputeVWAP is Σ(typicalPrice × quoteVolume) / Σ(quoteVolume).
func ComputeVWAP(ex Exchange, symbol string, bars []VolumeProfileBar, last float64, from, to time.Time, interval CandleInterval) VWAPVenue {
	out := VWAPVenue{
		Exchange: ex, Symbol: symbol,
		From: from.UTC(), To: to.UTC(), Interval: string(interval),
		LastPrice: last, Side: VolumeProfileVsUnknown,
	}
	if last <= 0 {
		for i := len(bars) - 1; i >= 0; i-- {
			if bars[i].Close > 0 {
				last = bars[i].Close
				out.LastPrice = last
				break
			}
		}
	}
	var pv, vol float64
	for _, b := range bars {
		if b.Volume <= 0 {
			continue
		}
		tp := TypicalPrice(b.High, b.Low, b.Close)
		if tp <= 0 {
			continue
		}
		pv += tp * b.Volume
		vol += b.Volume
		out.BarCount++
		if out.Low == 0 || (b.Low > 0 && b.Low < out.Low) {
			out.Low = b.Low
		}
		if b.High > out.High {
			out.High = b.High
		}
	}
	if vol <= 0 || pv <= 0 {
		out.Error = "not enough volume in this range"
		out.Summary = out.Error
		return out
	}
	out.Volume = vol
	out.VWAP = pv / vol
	annotateVWAP(&out)
	out.Summary = ExplainVWAPVenue(out)
	return out
}

// CombineVWAP weights each venue's VWAP by that venue's quote volume.
func CombineVWAP(symbol string, venues []VWAPVenue, from, to time.Time, interval CandleInterval) *VWAPVenue {
	out := &VWAPVenue{
		Exchange: "all", Symbol: symbol,
		From: from.UTC(), To: to.UTC(), Interval: string(interval),
		Side: VolumeProfileVsUnknown,
	}
	var pv, vol, lastPV, lastW float64
	var shares []VWAPShare
	for _, v := range venues {
		if v.Error != "" || v.Volume <= 0 || v.VWAP <= 0 {
			continue
		}
		pv += v.VWAP * v.Volume
		vol += v.Volume
		out.BarCount += v.BarCount
		if out.Low == 0 || (v.Low > 0 && v.Low < out.Low) {
			out.Low = v.Low
		}
		if v.High > out.High {
			out.High = v.High
		}
		if v.LastPrice > 0 {
			lastPV += v.LastPrice * v.Volume
			lastW += v.Volume
		}
		shares = append(shares, VWAPShare{Exchange: v.Exchange, Volume: v.Volume, VWAP: v.VWAP})
	}
	if vol <= 0 {
		out.Error = "not enough volume in this range"
		out.Summary = out.Error
		return out
	}
	out.Volume = vol
	out.VWAP = pv / vol
	if lastW > 0 {
		out.LastPrice = lastPV / lastW
	}
	sort.Slice(shares, func(i, j int) bool {
		return string(shares[i].Exchange) < string(shares[j].Exchange)
	})
	for i := range shares {
		shares[i].SharePct = shares[i].Volume / vol * 100
	}
	out.Shares = shares
	annotateVWAP(out)
	out.Summary = ExplainVWAPVenue(*out)
	return out
}

// ExplainVWAPVenue writes last vs VWAP.
func ExplainVWAPVenue(v VWAPVenue) string {
	if v.Error != "" {
		return v.Error
	}
	name := prettyBase(v.Symbol)
	head := fmt.Sprintf("%s VWAP %s since the start (volume %s).",
		name, formatQty(v.VWAP), formatQty(v.Volume))
	switch v.Side {
	case VolumeProfileVsAbove:
		head += fmt.Sprintf(" Last %s is %s%% above VWAP.", formatQty(v.LastPrice), formatFixed(v.DistancePct, 2))
	case VolumeProfileVsBelow:
		head += fmt.Sprintf(" Last %s is %s%% below VWAP.", formatQty(v.LastPrice), formatFixed(-v.DistancePct, 2))
	default:
		if v.LastPrice > 0 {
			head += fmt.Sprintf(" Last %s is at VWAP.", formatQty(v.LastPrice))
		}
	}
	if len(v.Shares) > 0 {
		parts := make([]string, 0, len(v.Shares))
		for _, s := range v.Shares {
			parts = append(parts, fmt.Sprintf("%s %s%%", s.Exchange, formatFixed(s.SharePct, 0)))
		}
		head += " Volume: " + joinCommaAnd(parts) + "."
	}
	return head
}

func joinCommaAnd(parts []string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	case 2:
		return parts[0] + " and " + parts[1]
	default:
		return parts[0] + ", " + joinCommaAnd(parts[1:])
	}
}

// ExplainVWAPReport prefers combined.
func ExplainVWAPReport(r VWAPReport) string {
	if r.Combined != nil && r.Combined.Summary != "" && r.Combined.Error == "" {
		return "Combined: " + r.Combined.Summary
	}
	for _, v := range r.Venues {
		if v.Summary != "" && v.Error == "" {
			return string(v.Exchange) + ": " + v.Summary
		}
	}
	for _, v := range r.Venues {
		if v.Summary != "" {
			return v.Summary
		}
	}
	return "No VWAP yet."
}

func annotateVWAP(v *VWAPVenue) {
	if v == nil || v.VWAP <= 0 || v.LastPrice <= 0 {
		return
	}
	v.Distance = v.LastPrice - v.VWAP
	v.DistancePct = v.Distance / v.VWAP * 100
	const flat = 0.02 // 0.02%
	switch {
	case v.DistancePct > flat:
		v.Side = VolumeProfileVsAbove
	case v.DistancePct < -flat:
		v.Side = VolumeProfileVsBelow
	default:
		v.Side = "at"
	}
}
