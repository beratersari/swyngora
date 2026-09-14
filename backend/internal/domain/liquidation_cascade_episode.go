package domain

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
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

// MergeCascadeEpisodes turns overlapping same-side waves on two or more venues into one event.
func MergeCascadeEpisodes(groups [][]CascadeEpisode, now time.Time) []CascadeEpisode {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var all []CascadeEpisode
	for _, g := range groups {
		all = append(all, g...)
	}
	n := len(all)
	if n == 0 {
		return nil
	}
	parent := make([]int, n)
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		if parent[i] != i {
			parent[i] = find(parent[i])
		}
		return parent[i]
	}
	union := func(a, b int) {
		pa, pb := find(a), find(b)
		if pa != pb {
			parent[pa] = pb
		}
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if all[i].Exchange == all[j].Exchange {
				continue
			}
			if !cascadeIntervalsOverlap(all[i], all[j], now) {
				continue
			}
			if _, ok := cascadeSharedSide(all[i].Side, all[j].Side); !ok {
				continue
			}
			union(i, j)
		}
	}
	clusters := map[int][]CascadeEpisode{}
	for i := range all {
		root := find(i)
		clusters[root] = append(clusters[root], all[i])
	}
	out := make([]CascadeEpisode, 0, n)
	for _, cl := range clusters {
		if len(cl) == 1 {
			out = append(out, cl[0])
			continue
		}
		side := cl[0].Side
		for _, ep := range cl[1:] {
			if s, ok := cascadeSharedSide(side, ep.Side); ok {
				side = s
			}
		}
		out = append(out, combineCascadeEpisodeGroup(cl, side, now))
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
	if have {
		ep.PriceOpen = formatQty(first)
		ep.PriceClose = formatQty(last)
		ep.PriceHigh = formatQty(high)
		ep.PriceLow = formatQty(low)
		if first > 0 {
			ep.PriceChangePct = formatSignedPct((last - first) / first * 100)
		}
	}
	var firstPrint time.Time
	for _, e := range events {
		t := e.Time.UTC()
		if t.Before(ep.StartedAt) || !t.Before(end) {
			continue
		}
		if firstPrint.IsZero() || t.Before(firstPrint) {
			firstPrint = t
		}
	}
	ep.FirstPrintAt = firstPrint
}

func combineCascadeEpisodeGroup(parts []CascadeEpisode, side string, now time.Time) CascadeEpisode {
	if len(parts) == 1 {
		return parts[0]
	}
	start := parts[0].StartedAt
	open := parts[0].Open
	end := cascadeEpisodeEnd(parts[0], now)
	grade := parts[0].Grade
	peak := parts[0].PeakRatio
	score := parts[0].Score
	var longN, shortN, totalN float64
	count := 0
	pick := parts[0]
	var hi, lo float64
	haveHi, haveLo := false, false
	for _, p := range parts {
		if p.StartedAt.Before(start) {
			start = p.StartedAt
		}
		if p.Open {
			open = true
		}
		if be := cascadeEpisodeEnd(p, now); be.After(end) {
			end = be
		}
		if cascadeGradeRank(p.Grade) > cascadeGradeRank(grade) {
			grade = p.Grade
		}
		if p.PeakRatio > peak {
			peak = p.PeakRatio
		}
		if p.Score > score {
			score = p.Score
		}
		longN += parseQty(p.LongNotional)
		shortN += parseQty(p.ShortNotional)
		totalN += parseQty(p.TotalNotional)
		count += p.Count
		if parseQty(p.PriceOpen) > 0 && (parseQty(pick.PriceOpen) <= 0 || p.DurationSec >= pick.DurationSec) {
			pick = p
		}
		if h := parseQty(p.PriceHigh); h > 0 && (!haveHi || h > hi) {
			hi, haveHi = h, true
		}
		if l := parseQty(p.PriceLow); l > 0 && (!haveLo || l < lo) {
			lo, haveLo = l, true
		}
	}
	out := CascadeEpisode{
		Symbol: parts[0].Symbol, Exchange: CascadeExchangeBoth, Combined: true,
		Side: side, Grade: grade, Score: score,
		StartedAt: start, Open: open, PeakRatio: peak,
		LongNotional:   formatQty(longN),
		ShortNotional:  formatQty(shortN),
		TotalNotional:  formatQty(totalN),
		Count:          count,
		PriceOpen:      pick.PriceOpen,
		PriceClose:     pick.PriceClose,
		PriceHigh:      pick.PriceHigh,
		PriceLow:       pick.PriceLow,
		PriceChangePct: pick.PriceChangePct,
	}
	if !open {
		out.EndedAt = end
	}
	if end.After(start) {
		out.DurationSec = int64(end.Sub(start).Seconds())
	}
	if haveHi {
		out.PriceHigh = formatQty(hi)
	}
	if haveLo {
		out.PriceLow = formatQty(lo)
	}
	fillCascadeVenueStarts(&out, parts)
	out.Summary = explainCascadeEpisode(out)
	return out
}

func fillCascadeVenueStarts(out *CascadeEpisode, parts []CascadeEpisode) {
	if out == nil || len(parts) == 0 {
		return
	}
	type rec struct {
		ex string
		t  time.Time
	}
	recs := make([]rec, 0, len(parts))
	for _, p := range parts {
		t := p.FirstPrintAt
		if t.IsZero() {
			t = p.StartedAt
		}
		if t.IsZero() {
			continue
		}
		recs = append(recs, rec{ex: p.Exchange, t: t.UTC()})
	}
	if len(recs) == 0 {
		return
	}
	sort.Slice(recs, func(i, j int) bool {
		if !recs[i].t.Equal(recs[j].t) {
			return recs[i].t.Before(recs[j].t)
		}
		return recs[i].ex < recs[j].ex
	})
	first := recs[0].t
	tied := 0
	firstSec := first.Truncate(time.Second)
	for _, r := range recs {
		if r.t.Truncate(time.Second).Equal(firstSec) {
			tied++
		}
	}
	out.OrderUnknown = tied > 1
	if !out.OrderUnknown {
		out.StartedBy = recs[0].ex
	}
	out.VenueStarts = make([]CascadeVenueStart, 0, len(recs))
	for _, r := range recs {
		lag := int64(r.t.Sub(first) / time.Second)
		if lag < 0 {
			lag = 0
		}
		out.VenueStarts = append(out.VenueStarts, CascadeVenueStart{
			Exchange: r.ex, StartedAt: r.t, LagSec: lag,
		})
	}
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
	who := cascadeEpisodeWho(ep)
	dur := formatCascadeDuration(ep.DurationSec)
	state := "lasted"
	if ep.Open {
		state = "running"
	}
	move := ""
	if ep.PriceChangePct != "" {
		move = fmt.Sprintf(" Price %s%%.", ep.PriceChangePct)
	}
	body := fmt.Sprintf("%s %s %s %s wave %s %s (long %s / short %s).%s",
		who, ep.Symbol, ep.Grade, ep.Side, state, dur, ep.LongNotional, ep.ShortNotional, move)
	if lead := explainCascadeLead(ep); lead != "" {
		return lead + ". " + body
	}
	return body
}

func cascadeEpisodeWho(ep CascadeEpisode) string {
	if !ep.Combined {
		return ep.Exchange
	}
	if len(ep.VenueStarts) == 0 {
		return "combined"
	}
	names := make([]string, 0, len(ep.VenueStarts))
	for _, v := range ep.VenueStarts {
		names = append(names, cascadeVenueTitle(v.Exchange))
	}
	return strings.Join(names, "+")
}

func explainCascadeLead(ep CascadeEpisode) string {
	if !ep.Combined || len(ep.VenueStarts) < 2 {
		return ""
	}
	if ep.OrderUnknown {
		var first, later []string
		for _, v := range ep.VenueStarts {
			name := cascadeVenueTitle(v.Exchange)
			if v.LagSec == 0 {
				first = append(first, name)
			} else {
				later = append(later, fmt.Sprintf("%s +%ds", name, v.LagSec))
			}
		}
		head := strings.Join(first, " and ") + " started together"
		if len(later) > 0 {
			return head + ", " + strings.Join(later, ", ")
		}
		return head
	}
	parts := make([]string, 0, len(ep.VenueStarts))
	for i, v := range ep.VenueStarts {
		name := cascadeVenueTitle(v.Exchange)
		if i == 0 || v.LagSec == 0 {
			parts = append(parts, name+" first")
		} else {
			parts = append(parts, fmt.Sprintf("%s +%ds", name, v.LagSec))
		}
	}
	return strings.Join(parts, ", ")
}

func cascadeVenueTitle(ex string) string {
	switch Exchange(ex) {
	case ExchangeBinance:
		return "Binance"
	case ExchangeBybit:
		return "Bybit"
	case ExchangeOKX:
		return "OKX"
	default:
		if ex == "" {
			return ""
		}
		return strings.ToUpper(ex[:1]) + ex[1:]
	}
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
