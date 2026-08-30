package domain

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"
)

type cascadeMinuteScore struct {
	start time.Time
	end   time.Time
	pack  cascadePack
	side  string
	grade string
	ratio float64
}

// DetectCascadeEpisodes walks 1-minute bursts and folds them into waves.
func DetectCascadeEpisodes(ex Exchange, symbol string, events []LiquidationEvent, now time.Time) []CascadeEpisode {
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	filtered := make([]LiquidationEvent, 0, len(events))
	for _, e := range events {
		if e.Notional <= 0 || e.Time.IsZero() {
			continue
		}
		if ex != "" && e.Exchange != ex {
			continue
		}
		filtered = append(filtered, e)
	}
	from := now.Add(-cascadeEpisodeLookback)
	buckets := cascadeBucketsRange(filtered, from, now)
	scores := scoreCascadeMinutes(buckets)
	raw := foldCascadeEpisodes(ex, symbol, scores, now)
	for i := range raw {
		fillCascadePrintPrices(&raw[i], filtered, now)
		raw[i].Summary = explainCascadeEpisode(raw[i])
	}
	return raw
}

// MergeCascadeEpisodes turns overlapping same-side Binance+Bybit waves into one event.
func MergeCascadeEpisodes(binance, bybit []CascadeEpisode, now time.Time) []CascadeEpisode {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	used := make([]bool, len(bybit))
	out := make([]CascadeEpisode, 0, len(binance)+len(bybit))
	for _, a := range binance {
		merged := false
		for j, b := range bybit {
			if used[j] {
				continue
			}
			if !cascadeIntervalsOverlap(a, b, now) {
				continue
			}
			side, ok := cascadeSharedSide(a.Side, b.Side)
			if !ok {
				continue
			}
			out = append(out, combineCascadeEpisodes(a, b, side, now))
			used[j] = true
			merged = true
			break
		}
		if !merged {
			out = append(out, a)
		}
	}
	for j, b := range bybit {
		if !used[j] {
			out = append(out, b)
		}
	}
	sortCascadeEpisodes(out)
	if len(out) > CascadeEpisodeMax {
		out = out[:CascadeEpisodeMax]
	}
	return out
}

// ApplyCascadeCandlePrices overwrites print prices with candle OHLC over the wave.
func ApplyCascadeCandlePrices(ep *CascadeEpisode, bars []Candle, now time.Time) {
	if ep == nil || len(bars) == 0 {
		return
	}
	end := cascadeEpisodeEnd(*ep, now)
	var (
		open, high, low, close float64
		have                   bool
	)
	for _, c := range bars {
		t := c.OpenTime.UTC()
		if t.Before(ep.StartedAt) || !t.Before(end) {
			if !c.CloseTime.IsZero() && c.CloseTime.After(ep.StartedAt) && c.OpenTime.Before(end) {
				// overlapping forming/closed bar
			} else {
				continue
			}
		}
		o, oerr := parseFloat(c.Open)
		h, herr := parseFloat(c.High)
		l, lerr := parseFloat(c.Low)
		cl, cerr := parseFloat(c.Close)
		if oerr != nil || herr != nil || lerr != nil || cerr != nil {
			continue
		}
		if !have {
			open, high, low, close = o, h, l, cl
			have = true
			continue
		}
		if h > high {
			high = h
		}
		if l < low && l > 0 {
			low = l
		}
		close = cl
	}
	if !have {
		return
	}
	ep.PriceOpen = formatQty(open)
	ep.PriceClose = formatQty(close)
	ep.PriceHigh = formatQty(high)
	ep.PriceLow = formatQty(low)
	if open > 0 {
		ep.PriceChangePct = formatSignedPct((close - open) / open * 100)
	}
	ep.Summary = explainCascadeEpisode(*ep)
}

func scoreCascadeMinutes(buckets []cascadeBucket) []cascadeMinuteScore {
	out := make([]cascadeMinuteScore, 0, len(buckets))
	for i := range buckets {
		from := i - int(cascadeLookback/cascadeStep)
		if from < 0 {
			from = 0
		}
		if i-from < cascadeEpisodeMinPrior {
			continue
		}
		priors := make([]cascadePack, 0, i-from)
		for j := from; j < i; j++ {
			priors = append(priors, cascadePack{long: buckets[j].long, short: buckets[j].short, count: buckets[j].count})
		}
		cur := cascadePack{long: buckets[i].long, short: buckets[i].short, count: buckets[i].count}
		longT := cascadeTypical(priors, true, cascadeMinTypical1m)
		shortT := cascadeTypical(priors, false, cascadeMinTypical1m)
		longR := cascadeRatio(cur.long, longT)
		shortR := cascadeRatio(cur.short, shortT)
		maxR := math.Max(longR, shortR)
		side := cascadeSideFromRatios(longR, shortR)
		grade := cascadeGrade(maxR, cur.count, 2)
		if grade == CascadeGradeQuiet {
			side = CascadeSideNone
		}
		out = append(out, cascadeMinuteScore{
			start: buckets[i].start,
			end:   buckets[i].start.Add(cascadeStep),
			pack:  cur,
			side:  side,
			grade: grade,
			ratio: maxR,
		})
	}
	return out
}

func foldCascadeEpisodes(ex Exchange, symbol string, scores []cascadeMinuteScore, now time.Time) []CascadeEpisode {
	var (
		out  []CascadeEpisode
		cur  *CascadeEpisode
		hot  cascadePack
		peak float64
		gap  int
	)
	flush := func() {
		if cur == nil {
			return
		}
		cur.LongNotional = formatQty(hot.long)
		cur.ShortNotional = formatQty(hot.short)
		cur.TotalNotional = formatQty(hot.long + hot.short)
		cur.Count = hot.count
		cur.PeakRatio = peak
		cur.Score = cascadeScore(CascadeWindowRead{MaxRatio: peak})
		end := cur.EndedAt
		if end.IsZero() {
			end = now
		}
		if end.After(cur.StartedAt) {
			cur.DurationSec = int64(end.Sub(cur.StartedAt).Seconds())
		}
		out = append(out, *cur)
		cur = nil
		hot = cascadePack{}
		peak = 0
		gap = 0
	}
	for _, m := range scores {
		hotMin := cascadeGradeRank(m.grade) >= cascadeGradeRank(CascadeGradeElevated)
		startMin := cascadeGradeRank(m.grade) >= cascadeGradeRank(CascadeGradeCascade)
		if cur == nil {
			if !startMin {
				continue
			}
			ep := CascadeEpisode{
				Symbol: symbol, Exchange: string(ex), Side: m.side, Grade: m.grade,
				StartedAt: m.start, Open: true,
			}
			cur = &ep
			hot = m.pack
			peak = m.ratio
			gap = 0
			continue
		}
		_, compat := cascadeSharedSide(cur.Side, m.side)
		if hotMin && (m.side == CascadeSideNone || compat || cur.Side == CascadeSideNone) {
			if m.side != CascadeSideNone {
				if cur.Side == CascadeSideNone {
					cur.Side = m.side
				} else if side, ok := cascadeSharedSide(cur.Side, m.side); ok {
					cur.Side = side
				}
			}
			if cascadeGradeRank(m.grade) > cascadeGradeRank(cur.Grade) {
				cur.Grade = m.grade
			}
			if m.ratio > peak {
				peak = m.ratio
			}
			hot.long += m.pack.long
			hot.short += m.pack.short
			hot.count += m.pack.count
			gap = 0
			continue
		}
		if !hotMin {
			gap++
			if gap >= cascadeEpisodeQuietGap {
				cur.Open = false
				cur.EndedAt = m.start.Add(-time.Duration(gap-1) * cascadeStep)
				flush()
			}
			continue
		}
		// Opposite-side burst: close this wave and start a new one.
		cur.Open = false
		cur.EndedAt = m.start
		flush()
		if startMin {
			ep := CascadeEpisode{
				Symbol: symbol, Exchange: string(ex), Side: m.side, Grade: m.grade,
				StartedAt: m.start, Open: true,
			}
			cur = &ep
			hot = m.pack
			peak = m.ratio
			gap = 0
		}
	}
	if cur != nil {
		if now.After(cur.StartedAt) {
			cur.DurationSec = int64(now.Sub(cur.StartedAt).Seconds())
		}
		cur.LongNotional = formatQty(hot.long)
		cur.ShortNotional = formatQty(hot.short)
		cur.TotalNotional = formatQty(hot.long + hot.short)
		cur.Count = hot.count
		cur.PeakRatio = peak
		cur.Score = cascadeScore(CascadeWindowRead{MaxRatio: peak})
		out = append(out, *cur)
	}
	sortCascadeEpisodes(out)
	if len(out) > CascadeEpisodeMax {
		out = out[:CascadeEpisodeMax]
	}
	return out
}

func fillCascadePrintPrices(ep *CascadeEpisode, events []LiquidationEvent, now time.Time) {
	end := cascadeEpisodeEnd(*ep, now)
	var first, last, high, low float64
	var firstT, lastT time.Time
	var have bool
	for _, e := range events {
		if e.Price <= 0 {
			continue
		}
		t := e.Time.UTC()
		if t.Before(ep.StartedAt) || !t.Before(end) {
			continue
		}
		if !have {
			first, last, high, low = e.Price, e.Price, e.Price, e.Price
			firstT, lastT = t, t
			have = true
			continue
		}
		if t.Before(firstT) {
			first, firstT = e.Price, t
		}
		if !t.Before(lastT) {
			last, lastT = e.Price, t
		}
		if e.Price > high {
			high = e.Price
		}
		if e.Price < low {
			low = e.Price
		}
	}
	if !have {
		return
	}
	ep.PriceOpen = formatQty(first)
	ep.PriceClose = formatQty(last)
	ep.PriceHigh = formatQty(high)
	ep.PriceLow = formatQty(low)
	if first > 0 {
		ep.PriceChangePct = formatSignedPct((last - first) / first * 100)
	}
}

func combineCascadeEpisodes(a, b CascadeEpisode, side string, now time.Time) CascadeEpisode {
	start := a.StartedAt
	if b.StartedAt.Before(start) {
		start = b.StartedAt
	}
	open := a.Open || b.Open
	end := cascadeEpisodeEnd(a, now)
	if be := cascadeEpisodeEnd(b, now); be.After(end) {
		end = be
	}
	grade := a.Grade
	if cascadeGradeRank(b.Grade) > cascadeGradeRank(grade) {
		grade = b.Grade
	}
	peak := math.Max(a.PeakRatio, b.PeakRatio)
	out := CascadeEpisode{
		Symbol: a.Symbol, Exchange: CascadeExchangeBoth, Combined: true,
		Side: side, Grade: grade, Score: math.Max(a.Score, b.Score),
		StartedAt: start, Open: open, PeakRatio: peak,
		LongNotional:  formatQty(parseQty(a.LongNotional) + parseQty(b.LongNotional)),
		ShortNotional: formatQty(parseQty(a.ShortNotional) + parseQty(b.ShortNotional)),
		TotalNotional: formatQty(parseQty(a.TotalNotional) + parseQty(b.TotalNotional)),
		Count:         a.Count + b.Count,
	}
	if !open {
		out.EndedAt = end
	}
	if end.After(start) {
		out.DurationSec = int64(end.Sub(start).Seconds())
	}
	// Prefer the longer venue path for print prices until candles overwrite.
	pick := a
	if parseQty(b.PriceOpen) > 0 && (parseQty(a.PriceOpen) <= 0 || b.DurationSec >= a.DurationSec) {
		pick = b
	}
	out.PriceOpen, out.PriceClose = pick.PriceOpen, pick.PriceClose
	out.PriceHigh, out.PriceLow = pick.PriceHigh, pick.PriceLow
	out.PriceChangePct = pick.PriceChangePct
	if ao, bo := parseQty(a.PriceOpen), parseQty(b.PriceOpen); ao > 0 && bo > 0 {
		hi := math.Max(parseQty(a.PriceHigh), parseQty(b.PriceHigh))
		lo := parseQty(a.PriceLow)
		if bl := parseQty(b.PriceLow); bl > 0 && (lo <= 0 || bl < lo) {
			lo = bl
		}
		if hi > 0 {
			out.PriceHigh = formatQty(hi)
		}
		if lo > 0 {
			out.PriceLow = formatQty(lo)
		}
	}
	out.Summary = explainCascadeEpisode(out)
	return out
}

func cascadeEpisodeEnd(ep CascadeEpisode, now time.Time) time.Time {
	if !ep.EndedAt.IsZero() {
		return ep.EndedAt.UTC()
	}
	if now.IsZero() {
		return time.Now().UTC()
	}
	return now.UTC()
}

func cascadeIntervalsOverlap(a, b CascadeEpisode, now time.Time) bool {
	as, ae := a.StartedAt, cascadeEpisodeEnd(a, now)
	bs, be := b.StartedAt, cascadeEpisodeEnd(b, now)
	return as.Before(be) && bs.Before(ae)
}

func sortCascadeEpisodes(eps []CascadeEpisode) {
	sort.Slice(eps, func(i, j int) bool {
		if !eps[i].StartedAt.Equal(eps[j].StartedAt) {
			return eps[i].StartedAt.After(eps[j].StartedAt)
		}
		if eps[i].Combined != eps[j].Combined {
			return eps[i].Combined
		}
		return eps[i].Exchange < eps[j].Exchange
	})
}

func parseQty(s string) float64 {
	if s == "" {
		return 0
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return n
}

func formatSignedPct(v float64) string {
	if !math.IsInf(v, 0) && !math.IsNaN(v) {
		return fmt.Sprintf("%+.2f", v)
	}
	return ""
}

func explainCascadeEpisode(ep CascadeEpisode) string {
	who := ep.Exchange
	if ep.Combined {
		who = "Binance+Bybit"
	}
	dur := formatCascadeDuration(ep.DurationSec)
	state := "lasted"
	if ep.Open {
		state = "running"
	}
	move := ""
	if ep.PriceChangePct != "" {
		move = fmt.Sprintf(" Price %s%%.", ep.PriceChangePct)
	}
	return fmt.Sprintf("%s %s %s %s wave %s %s (long %s / short %s).%s",
		who, ep.Symbol, ep.Grade, ep.Side, state, dur, ep.LongNotional, ep.ShortNotional, move)
}

func formatCascadeDuration(sec int64) string {
	if sec < 60 {
		if sec < 1 {
			sec = 1
		}
		return fmt.Sprintf("%ds", sec)
	}
	m := sec / 60
	if m < 60 {
		if rem := sec % 60; rem >= 30 {
			m++
		}
		return fmt.Sprintf("%dm", m)
	}
	h := m / 60
	rm := m % 60
	if rm == 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dh %dm", h, rm)
}
