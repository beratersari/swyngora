package domain

import (
	"math"
	"sort"
	"strings"
	"time"
)

const (
	// BacktestMarketOK ran and produced a count (including zero signals).
	BacktestMarketOK = "ok"
	// BacktestMarketNoData had no candles in the range.
	BacktestMarketNoData = "no_data"
	// BacktestMarketError failed to fetch or parse tape.
	BacktestMarketError = "error"

	// BacktestDefaultRiskPct is the share of current equity lost on a full stop.
	BacktestDefaultRiskPct = 1.0
	// BacktestBookStakePct is kept for older jobs that stored a stake share.
	BacktestBookStakePct = BacktestDefaultRiskPct
	// BacktestDefaultStartBalance is the Lab default shared pot (quote units).
	BacktestDefaultStartBalance = 10000.0
	backtestBookMaxStart        = 1e9
	backtestBookMaxRisk         = 25.0
	// BacktestDefaultMaxOpenRiskPct caps combined stop risk of open fills.
	BacktestDefaultMaxOpenRiskPct = 3.0
	backtestBookMaxOpenRisk       = 100.0
)

// BacktestMarketResult is one coin after a watchlist Lab run.
type BacktestMarketResult struct {
	Exchange    Exchange `json:"exchange"`
	Symbol      string   `json:"symbol"`
	Status      string   `json:"status"` // ok | no_data | error
	Error       string   `json:"error,omitempty"`
	SignalCount int      `json:"signalCount"`
	Bars        int      `json:"bars,omitempty"`
}

// BacktestBook is one shared cash pot used by every watchlist fill.
type BacktestBook struct {
	StartingBalance   float64 `json:"startingBalance"`
	RangeEndBalance   float64 `json:"rangeEndBalance"`
	AfterRangeBalance float64 `json:"afterRangeBalance"`
	EndingBalance     float64 `json:"endingBalance"` // same as afterRangeBalance
	PeakEquity        float64 `json:"peakEquity"`
	TroughEquity      float64 `json:"troughEquity"`
	RiskPct           float64 `json:"riskPct"`
	StakePct          float64 `json:"stakePct,omitempty"` // alias of riskPct for older clients
	MaxOpenRiskPct    float64 `json:"maxOpenRiskPct"`
	TradesTaken       int     `json:"tradesTaken"`
	TradesSkipped     int     `json:"tradesSkipped"`
	SkippedNoRisk     int     `json:"skippedNoRisk,omitempty"`
	SkippedSameCoin   int     `json:"skippedSameCoin,omitempty"`
	ClosedInRange     int     `json:"closedInRange"`
	ClosedAfterRange  int     `json:"closedAfterRange"`
	OpenAtRangeEnd    int     `json:"openAtRangeEnd"`
	OpenAtEnd         int     `json:"openAtEnd"`
}

// BacktestRunState is persisted after a custom job walks its coins.
type BacktestRunState struct {
	Markets []BacktestMarketResult `json:"markets,omitempty"`
	Book    *BacktestBook          `json:"book,omitempty"`
}

// BacktestBookFill is one first-touch trade on the shared pot.
type BacktestBookFill struct {
	Exchange       Exchange
	Symbol         string
	Side           string
	EntryAt        time.Time
	ExitAt         time.Time
	Entry          float64
	StopLoss       float64
	ExitPrice      float64
	RangeExitPrice float64
	Leverage       int
	ClosedInRange  bool
	Outcome        string
	HasExit        bool
	HitTimeUnknown bool
	Score          float64
	Marks          []BacktestPriceMark
	HasScale       bool
	ScaleAt        time.Time
	ScalePct       float64
	ScalePrice     float64
	StopToEntry    bool
}

// NormalizeBacktestStartBalance applies the Lab default and a hard cap.
func NormalizeBacktestStartBalance(v float64) float64 {
	if v <= 0 || math.IsNaN(v) {
		return BacktestDefaultStartBalance
	}
	if v > backtestBookMaxStart {
		return backtestBookMaxStart
	}
	return v
}

// NormalizeBacktestRiskPct is the stop-loss share of equity (default 1%).
func NormalizeBacktestRiskPct(v float64) float64 {
	if v <= 0 || math.IsNaN(v) {
		return BacktestDefaultRiskPct
	}
	if v > backtestBookMaxRisk {
		return backtestBookMaxRisk
	}
	return v
}

// NormalizeBacktestMaxOpenRiskPct is the combined open-stop cap (default 3%).
func NormalizeBacktestMaxOpenRiskPct(v float64) float64 {
	if v <= 0 || math.IsNaN(v) {
		return BacktestDefaultMaxOpenRiskPct
	}
	if v > backtestBookMaxOpenRisk {
		return backtestBookMaxOpenRisk
	}
	return v
}

// SimulateSharedBacktestBook walks fills in time order against one cash pot.
// Each taken trade is sized so a full stop loses about riskPct of current
// mark-to-market equity (cash + open margin + open unrealized). Combined open
// stop risk cannot exceed maxOpenRisk of that equity; skipped signals stay
// skipped. When a fill closes or scales at TP1, later signals can use the free
// risk. A new fill at the same timestamp as a TP1 scale does not take that
// room — TP1 must have already printed.
// RangeEndBalance marks still-open trades at the last in-range close.
// AfterRangeBalance applies later TP/SL only; still-open trades stay marked in-range.
func SimulateSharedBacktestBook(start, riskPct, maxOpenRisk float64, fills []BacktestBookFill, rangeEnd time.Time) BacktestBook {
	start = NormalizeBacktestStartBalance(start)
	riskPct = NormalizeBacktestRiskPct(riskPct)
	maxOpenRisk = NormalizeBacktestMaxOpenRiskPct(maxOpenRisk)
	out := BacktestBook{
		StartingBalance:   round2(start),
		RangeEndBalance:   round2(start),
		AfterRangeBalance: round2(start),
		EndingBalance:     round2(start),
		PeakEquity:        round2(start),
		TroughEquity:      round2(start),
		RiskPct:           riskPct,
		StakePct:          riskPct,
		MaxOpenRiskPct:    maxOpenRisk,
	}
	if len(fills) == 0 {
		return out
	}
	end := rangeEnd.UTC()
	type ev struct {
		at    time.Time
		exit  bool
		scale bool
		i     int
	}
	events := make([]ev, 0, len(fills)*3)
	for i, f := range fills {
		if f.EntryAt.IsZero() || f.Entry <= 0 || f.StopLoss <= 0 {
			continue
		}
		entry := f.EntryAt.UTC()
		events = append(events, ev{at: entry, i: i})
		if f.HasScale && !f.ScaleAt.IsZero() && f.ScalePct > 0 && f.ScalePct < 1 {
			sc := f.ScaleAt.UTC()
			if !sc.After(entry) {
				sc = entry.Add(time.Nanosecond)
			}
			events = append(events, ev{at: sc, scale: true, i: i})
		}
		if f.HasExit && !f.ExitAt.IsZero() {
			exit := f.ExitAt.UTC()
			if !exit.After(entry) {
				exit = entry.Add(time.Nanosecond)
			}
			events = append(events, ev{at: exit, exit: true, i: i})
		}
	}
	sort.SliceStable(events, func(i, j int) bool {
		if !events[i].at.Equal(events[j].at) {
			return events[i].at.Before(events[j].at)
		}
		// Same fill: scale before that fill's exit so TP1 is realized first.
		if events[i].i == events[j].i {
			ri, rj := sameFillEventRank(events[i].exit, events[i].scale), sameFillEventRank(events[j].exit, events[j].scale)
			if ri != rj {
				return ri < rj
			}
			return false
		}
		// Other fills: full exits first, then new entries, then TP1 scales.
		// A same-time entry must not take risk that TP1 has not yet freed.
		ri, rj := bookEventRank(events[i].exit, events[i].scale), bookEventRank(events[j].exit, events[j].scale)
		if ri != rj {
			return ri < rj
		}
		if !events[i].exit && !events[i].scale {
			si, sj := fills[events[i].i].Score, fills[events[j].i].Score
			if si != sj {
				return si > sj
			}
		}
		return events[i].i < events[j].i
	})
	cash := start
	type pos struct {
		margin   float64
		notional float64
		risk     float64
		i        int
		key      string
	}
	open := map[int]pos{}
	openCoin := map[string]int{}
	snapshotted := end.IsZero()
	equityAt := func(at time.Time) float64 {
		e := cash
		for _, p := range open {
			e += p.margin + notionalPnLAt(fills[p.i], p.notional, at)
		}
		return e
	}
	mark := func(at time.Time) {
		eq := equityAt(at)
		if eq > out.PeakEquity {
			out.PeakEquity = eq
		}
		if eq < out.TroughEquity {
			out.TroughEquity = eq
		}
	}
	snapshotRange := func() {
		if snapshotted {
			return
		}
		eq := cash
		for _, p := range open {
			eq += p.margin + notionalPnL(fills[p.i], p.notional, true)
		}
		out.RangeEndBalance = round2(eq)
		out.OpenAtRangeEnd = len(open)
		snapshotted = true
	}
	maybeSnap := func(at time.Time) {
		if !end.IsZero() && !at.After(end) {
			return
		}
		snapshotRange()
	}
	for _, e := range events {
		maybeSnap(e.at)
		if e.scale {
			p, ok := open[e.i]
			if !ok {
				continue
			}
			f := fills[e.i]
			pct := f.ScalePct
			if pct <= 0 || pct >= 1 {
				continue
			}
			closedN := p.notional * pct
			closedM := p.margin * pct
			cash += closedM + notionalPnLPrice(f, closedN, f.ScalePrice)
			p.notional -= closedN
			p.margin -= closedM
			if f.StopToEntry {
				p.risk = 0
			} else {
				p.risk *= 1 - pct
			}
			open[e.i] = p
			mark(e.at)
			continue
		}
		if e.exit {
			p, ok := open[e.i]
			if !ok {
				continue
			}
			delete(open, e.i)
			if p.key != "" {
				openCoin[p.key]--
				if openCoin[p.key] <= 0 {
					delete(openCoin, p.key)
				}
			}
			cash += p.margin + notionalPnL(fills[e.i], p.notional, false)
			if fills[e.i].ClosedInRange {
				out.ClosedInRange++
			} else {
				out.ClosedAfterRange++
			}
			mark(e.at)
			continue
		}
		f := fills[e.i]
		coin := marketKey(f.Exchange, f.Symbol)
		if coin != "|" && openCoin[coin] > 0 {
			out.TradesSkipped++
			out.SkippedSameCoin++
			continue
		}
		stopF := stopFrac(f.Entry, f.StopLoss)
		if stopF <= 0 {
			out.TradesSkipped++
			continue
		}
		eq := equityAt(e.at)
		newRisk := eq * (riskPct / 100)
		used := 0.0
		for _, p := range open {
			used += p.risk
		}
		if used+newRisk > eq*(maxOpenRisk/100)+1e-9 {
			out.TradesSkipped++
			out.SkippedNoRisk++
			continue
		}
		notional := newRisk / stopF
		lev := f.Leverage
		if lev < 1 {
			lev = 1
		}
		margin := notional / float64(lev)
		if margin <= 0 || cash+1e-9 < margin {
			out.TradesSkipped++
			continue
		}
		cash -= margin
		open[e.i] = pos{margin: margin, notional: notional, risk: newRisk, i: e.i, key: coin}
		if coin != "|" {
			openCoin[coin]++
		}
		out.TradesTaken++
		mark(e.at)
	}
	if !snapshotted {
		snapshotRange()
	}
	out.OpenAtEnd = len(open)
	for _, p := range open {
		// Still open: keep range-end mark. Do not apply a later tape price.
		cash += p.margin + notionalPnL(fills[p.i], p.notional, true)
	}
	out.AfterRangeBalance = round2(cash)
	out.EndingBalance = out.AfterRangeBalance
	if end.IsZero() {
		out.RangeEndBalance = out.AfterRangeBalance
	}
	out.PeakEquity = round2(out.PeakEquity)
	out.TroughEquity = round2(out.TroughEquity)
	if out.EndingBalance > out.PeakEquity {
		out.PeakEquity = out.EndingBalance
	}
	if out.EndingBalance < out.TroughEquity {
		out.TroughEquity = out.EndingBalance
	}
	return out
}

func stopFrac(entry, sl float64) float64 {
	if entry <= 0 {
		return 0
	}
	d := math.Abs(entry-sl) / entry
	if d < 1e-6 {
		return 0
	}
	return d
}

func unleveredMove(side string, entry, exit float64) float64 {
	if entry <= 0 {
		return 0
	}
	if NormalizeTradeSide(side) == BacktestTradeSideShort {
		return (entry - exit) / entry
	}
	return (exit - entry) / entry
}

func bookEventRank(exit, scale bool) int {
	if exit {
		return 0
	}
	if scale {
		return 2
	}
	return 1
}

func sameFillEventRank(exit, scale bool) int {
	if scale {
		return 0
	}
	if exit {
		return 1
	}
	return 2
}

func notionalPnL(f BacktestBookFill, notional float64, atRange bool) float64 {
	px := f.ExitPrice
	if atRange && f.RangeExitPrice > 0 {
		px = f.RangeExitPrice
	}
	if px <= 0 {
		px = f.Entry
	}
	return notional * unleveredMove(f.Side, f.Entry, px)
}

func notionalPnLAt(f BacktestBookFill, notional float64, at time.Time) float64 {
	px := markPriceAt(f.Marks, at, f.Entry)
	return notionalPnLPrice(f, notional, px)
}

func notionalPnLPrice(f BacktestBookFill, notional, px float64) float64 {
	if px <= 0 {
		px = f.Entry
	}
	return notional * unleveredMove(f.Side, f.Entry, px)
}

// BookFillsFromSignals keeps trades that actually filled after the signal bar closed.
func BookFillsFromSignals(sigs []ScannerBacktestSignal, rangeEnd time.Time) []BacktestBookFill {
	out := make([]BacktestBookFill, 0, len(sigs))
	end := rangeEnd.UTC()
	for _, s := range sigs {
		if s.Trade == nil || s.Trade.Entry <= 0 {
			continue
		}
		tr := s.Trade
		if tr.EnteredAt == nil || tr.EnteredAt.IsZero() {
			continue
		}
		entryAt := tr.EnteredAt.UTC()
		var exit time.Time
		closed := tradeFullyClosed(tr)
		free := TradeRiskFreeAt(tr)
		hasExit := closed && free != nil
		if hasExit {
			exit = free.UTC()
		}
		rangePx := tr.RangeExitPrice
		if rangePx <= 0 {
			rangePx = tr.ExitPrice
		}
		closedIn := hasExit && !tr.ClosedAfterRange
		if hasExit && !end.IsZero() && free.After(end) {
			closedIn = false
		}
		score := 0.0
		if s.Metrics != nil {
			score = s.Metrics["score"]
		}
		fill := BacktestBookFill{
			Exchange: s.Exchange, Symbol: s.Symbol, Side: tr.Side, EntryAt: entryAt, ExitAt: exit,
			Entry: tr.Entry, StopLoss: tr.StopLoss, ExitPrice: tr.ExitPrice,
			RangeExitPrice: rangePx, Leverage: tr.Leverage,
			ClosedInRange: closedIn, Outcome: tr.Outcome, HasExit: hasExit,
			HitTimeUnknown: tr.HitTimeUnknown, Score: score, Marks: tr.Marks,
		}
		if sc := tradeScaleEvent(tr); sc != nil {
			fill.HasScale = true
			fill.ScaleAt = sc.at
			fill.ScalePct = sc.pct
			fill.ScalePrice = sc.price
			fill.StopToEntry = sc.stopToEntry
		}
		out = append(out, fill)
	}
	return out
}

// MergeBacktestMarkets keeps every planned coin, including zero-signal and errors.
func MergeBacktestMarkets(planned []BacktestMarket, results []BacktestMarketResult, sigs []ScannerBacktestSignal) []BacktestMarketPerformance {
	byKey := map[string][]ScannerBacktestSignal{}
	for _, s := range sigs {
		key := marketKey(s.Exchange, s.Symbol)
		if key == "|" {
			continue
		}
		byKey[key] = append(byKey[key], s)
	}
	resByKey := map[string]BacktestMarketResult{}
	for _, r := range results {
		resByKey[marketKey(r.Exchange, r.Symbol)] = r
	}
	seen := map[string]struct{}{}
	out := make([]BacktestMarketPerformance, 0, len(planned)+len(results))
	appendRow := func(ex Exchange, sym string) {
		key := marketKey(ex, sym)
		if key == "|" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		g := byKey[key]
		res, has := resByKey[key]
		row := summarizeOneMarket(ex, sym, g)
		if has {
			if res.Status != "" {
				row.Status = res.Status
			}
			row.Error = res.Error
			if res.SignalCount > row.SignalCount {
				row.SignalCount = res.SignalCount
			}
		}
		if row.Status == "" {
			row.Status = BacktestMarketOK
		}
		out = append(out, row)
	}
	for _, p := range planned {
		appendRow(p.Exchange, p.Symbol)
	}
	for _, r := range results {
		appendRow(r.Exchange, r.Symbol)
	}
	return out
}

func marketKey(ex Exchange, sym string) string {
	return string(ex) + "|" + strings.ToUpper(strings.TrimSpace(sym))
}

func summarizeOneMarket(ex Exchange, sym string, g []ScannerBacktestSignal) BacktestMarketPerformance {
	one := make([]float64, 0, len(g))
	five := make([]float64, 0, len(g))
	twenty := make([]float64, 0, len(g))
	for _, s := range g {
		if s.Return1d != nil && !math.IsNaN(*s.Return1d) {
			one = append(one, *s.Return1d)
		}
		if s.Return5d != nil && !math.IsNaN(*s.Return5d) {
			five = append(five, *s.Return5d)
		}
		if s.Return20d != nil && !math.IsNaN(*s.Return20d) {
			twenty = append(twenty, *s.Return20d)
		}
	}
	return BacktestMarketPerformance{
		Exchange: string(ex), Symbol: sym, Status: BacktestMarketOK,
		SignalCount: len(g),
		Return1d:    backtestHorizonStats(1, one),
		Return5d:    backtestHorizonStats(5, five),
		Return20d:   backtestHorizonStats(20, twenty),
		Trades:      summarizeBacktestTrades(g),
	}
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
