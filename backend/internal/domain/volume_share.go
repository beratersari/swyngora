package domain

import (
	"fmt"
	"sort"
	"time"
)

const (
	VolumeShareWindow15m = ParticipationWindow15m
	VolumeShareWindow1h  = ParticipationWindow1h
	VolumeShareWindow4h  = ParticipationWindow4h

	VolumeShareGained  = "gained"
	VolumeShareLost    = "lost"
	VolumeShareStable  = "stable"
	VolumeShareMissing = "missing"

	// 1 percentage point before calling a venue's mix a gain or loss.
	volumeShareMovePts = 1.0
	futuresVolumeBar   = 5 * time.Minute
	FuturesVolumeBars  = 120 // 10h of 5m bars covers 4h current + 4h prior
)

// VolumeShareWindows is 15m / 1h / 4h of futures quote volume.
var VolumeShareWindows = ParticipationWindows

// FuturesVolumeBar is one 5m perpetual quote-volume print.
type FuturesVolumeBar struct {
	Time        time.Time
	QuoteVolume float64
}

// VolumeShareVenue is one venue's share of futures volume in a window.
type VolumeShareVenue struct {
	Exchange      Exchange
	Volume        float64
	SharePct      float64
	PriorVolume   float64
	PriorSharePct float64
	ChangePct     float64 // percentage points vs the prior window, among venues that have both
	Kind          string  // gained | lost | stable | missing
	Complete      bool
	PriorKnown    bool // false when this venue has no bars in the earlier window
}

// VolumeShareWindow is the mix of venues in one lookback, vs the same length just before.
type VolumeShareWindow struct {
	Window   string
	Total    float64
	Prior    float64
	Venues   []VolumeShareVenue
	Gained   []string
	Lost     []string
	Missing  []string // venues with no earlier window; not treated as 0% share
	Complete bool
	Summary  string
}

// VolumeShareReport is how BTC (or another) futures volume splits across venues.
type VolumeShareReport struct {
	Symbol   string
	Exchange string
	AsOf     time.Time
	Windows  []VolumeShareWindow
	Note     string
}

// SumFuturesVolume totals quote volume for bars in [from, to).
func SumFuturesVolume(bars []FuturesVolumeBar, from, to time.Time) (vol float64, n int) {
	if to.Before(from) {
		return 0, 0
	}
	from, to = from.UTC(), to.UTC()
	for _, b := range bars {
		t := b.Time.UTC()
		if t.Before(from) || !t.Before(to) || b.QuoteVolume <= 0 {
			continue
		}
		vol += b.QuoteVolume
		n++
	}
	return vol, n
}

// ClassifyVolumeShare labels a percentage-point mix change.
func ClassifyVolumeShare(changePct float64) string {
	switch {
	case changePct >= volumeShareMovePts:
		return VolumeShareGained
	case changePct <= -volumeShareMovePts:
		return VolumeShareLost
	default:
		return VolumeShareStable
	}
}

// BuildVolumeShare folds per-venue 5m bars into 15m / 1h / 4h mix and 1h-style share change.
func BuildVolumeShare(exchange, symbol string, series map[Exchange][]FuturesVolumeBar, now time.Time) *VolumeShareReport {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	symbol = NormalizeLiquidationSymbol(symbol)
	out := &VolumeShareReport{
		Symbol:   symbol,
		Exchange: exchange,
		AsOf:     now,
		Windows:  make([]VolumeShareWindow, 0, len(VolumeShareWindows)),
		Note:     "Perpetual quote volume share across Binance USD-M, Bybit linear, and OKX USDT-swap. Current mix vs the same length just before (1h now vs the hour before). Informational only — not financial advice.",
	}
	order := volumeShareOrder(exchange, series)
	for _, spec := range VolumeShareWindows {
		out.Windows = append(out.Windows, buildVolumeShareWindow(spec.ID, spec.Dur, order, series, now))
	}
	return out
}

func volumeShareOrder(exchange string, series map[Exchange][]FuturesVolumeBar) []Exchange {
	if exchange != "" && exchange != "all" {
		return []Exchange{Exchange(exchange)}
	}
	_ = series
	return FuturesVenues()
}

func buildVolumeShareWindow(id string, dur time.Duration, order []Exchange, series map[Exchange][]FuturesVolumeBar, now time.Time) VolumeShareWindow {
	curFrom := now.Add(-dur)
	priorFrom := now.Add(-2 * dur)
	type row struct {
		ex         Exchange
		vol, prior float64
		nCur, nPri int
	}
	rows := make([]row, 0, len(order))
	var total, priorTotal float64
	for _, ex := range order {
		v, n := SumFuturesVolume(series[ex], curFrom, now)
		p, pn := SumFuturesVolume(series[ex], priorFrom, curFrom)
		rows = append(rows, row{ex: ex, vol: v, prior: p, nCur: n, nPri: pn})
		total += v
		priorTotal += p
	}
	out := VolumeShareWindow{Window: id, Total: total, Prior: priorTotal, Complete: total > 0 && priorTotal > 0}
	var cmpCur, cmpPri float64
	for _, r := range rows {
		if r.nCur > 0 && r.nPri > 0 {
			cmpCur += r.vol
			cmpPri += r.prior
		}
	}
	venues := make([]VolumeShareVenue, 0, len(rows))
	var gained, lost, missing []string
	for _, r := range rows {
		priorKnown := r.nPri > 0
		curKnown := r.nCur > 0
		v := VolumeShareVenue{
			Exchange: r.ex, Volume: r.vol, PriorVolume: r.prior,
			Kind: VolumeShareStable, Complete: curKnown, PriorKnown: priorKnown,
		}
		if total > 0 && curKnown {
			v.SharePct = r.vol / total * 100
		}
		if !priorKnown || !curKnown {
			v.Kind = VolumeShareMissing
			out.Complete = false
			missing = append(missing, prettyFuturesVenue(r.ex))
		} else if cmpCur > 0 && cmpPri > 0 {
			v.PriorSharePct = r.prior / cmpPri * 100
			v.ChangePct = r.vol/cmpCur*100 - v.PriorSharePct
			v.Kind = ClassifyVolumeShare(v.ChangePct)
		} else {
			out.Complete = false
		}
		switch v.Kind {
		case VolumeShareGained:
			gained = append(gained, prettyFuturesVenue(r.ex))
		case VolumeShareLost:
			lost = append(lost, prettyFuturesVenue(r.ex))
		}
		venues = append(venues, v)
	}
	sort.Slice(venues, func(i, j int) bool {
		if venues[i].SharePct == venues[j].SharePct {
			return string(venues[i].Exchange) < string(venues[j].Exchange)
		}
		return venues[i].SharePct > venues[j].SharePct
	})
	out.Venues = venues
	out.Gained = gained
	out.Lost = lost
	out.Missing = missing
	out.Summary = ExplainVolumeShareWindow(out)
	return out
}

// ExplainVolumeShareWindow is the desk line: mix plus who gained or lost share.
func ExplainVolumeShareWindow(w VolumeShareWindow) string {
	if w.Total <= 0 || len(w.Venues) == 0 {
		return fmt.Sprintf("%s: not enough futures volume to split across venues.", w.Window)
	}
	parts := make([]string, 0, len(w.Venues))
	for _, v := range w.Venues {
		if v.Kind == VolumeShareMissing && v.SharePct == 0 {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %s%%", prettyFuturesVenue(v.Exchange), formatFixed(v.SharePct, 0)))
	}
	head := w.Window + ": " + joinList(parts) + "."
	if len(w.Missing) > 0 {
		head += " No earlier window for " + joinList(w.Missing) + "."
	}
	switch {
	case len(w.Gained) > 0 && len(w.Lost) > 0:
		return head + " " + joinList(w.Gained) + " gained share; " + joinList(w.Lost) + " lost share."
	case len(w.Gained) > 0:
		return head + " " + joinList(w.Gained) + " gained share."
	case len(w.Lost) > 0:
		return head + " " + joinList(w.Lost) + " lost share."
	default:
		if w.Prior > 0 {
			return head + " Mix is little changed vs the prior window."
		}
		return head
	}
}

func prettyFuturesVenue(ex Exchange) string {
	switch Exchange(ex) {
	case ExchangeBinance:
		return "Binance"
	case ExchangeBybit:
		return "Bybit"
	case ExchangeOKX:
		return "OKX"
	default:
		s := string(ex)
		if s == "" {
			return "venue"
		}
		return s
	}
}
