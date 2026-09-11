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

	// BacktestBookStakePct is the share of current equity locked as margin per fill.
	BacktestBookStakePct = 10.0
	// BacktestDefaultStartBalance is the Lab default shared pot (quote units).
	BacktestDefaultStartBalance = 10000.0
	backtestBookMaxStart        = 1e9
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
	StartingBalance float64 `json:"startingBalance"`
	EndingBalance   float64 `json:"endingBalance"`
	PeakEquity      float64 `json:"peakEquity"`
	TroughEquity    float64 `json:"troughEquity"`
	StakePct        float64 `json:"stakePct"`
	TradesTaken     int     `json:"tradesTaken"`
	TradesSkipped   int     `json:"tradesSkipped"`
	OpenAtEnd       int     `json:"openAtEnd"`
}

// BacktestRunState is persisted after a custom job walks its coins.
type BacktestRunState struct {
	Markets []BacktestMarketResult `json:"markets,omitempty"`
	Book    *BacktestBook          `json:"book,omitempty"`
}

// BacktestBookFill is one first-touch trade on the shared pot.
type BacktestBookFill struct {
	Symbol  string
	EntryAt time.Time
	ExitAt  time.Time
	PnLPct  float64
	Outcome string
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

// SimulateSharedBacktestBook walks fills in time order against one cash pot.
// Each taken trade locks stakePct of current equity (cash + open margin).
// A fill is skipped when cash is short. Still-open trades are marked into the end balance.
func SimulateSharedBacktestBook(start, stakePct float64, fills []BacktestBookFill) BacktestBook {
	start = NormalizeBacktestStartBalance(start)
	if stakePct <= 0 || math.IsNaN(stakePct) {
		stakePct = BacktestBookStakePct
	}
	out := BacktestBook{
		StartingBalance: round2(start),
		EndingBalance:   round2(start),
		PeakEquity:      round2(start),
		TroughEquity:    round2(start),
		StakePct:        stakePct,
	}
	if len(fills) == 0 {
		return out
	}
	type ev struct {
		at   time.Time
		exit bool
		i    int
	}
	events := make([]ev, 0, len(fills)*2)
	for i, f := range fills {
		entry := f.EntryAt.UTC()
		exit := f.ExitAt.UTC()
		if exit.IsZero() || !exit.After(entry) {
			exit = entry.Add(time.Nanosecond)
		}
		events = append(events, ev{at: entry, exit: false, i: i})
		events = append(events, ev{at: exit, exit: true, i: i})
	}
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].at.Equal(events[j].at) {
			if events[i].exit != events[j].exit {
				return events[i].exit
			}
			return events[i].i < events[j].i
		}
		return events[i].at.Before(events[j].at)
	})
	cash := start
	type pos struct {
		stake  float64
		pnlPct float64
	}
	open := map[int]pos{}
	equity := func() float64 {
		e := cash
		for _, p := range open {
			e += p.stake
		}
		return e
	}
	mark := func() {
		eq := equity()
		if eq > out.PeakEquity {
			out.PeakEquity = eq
		}
		if eq < out.TroughEquity {
			out.TroughEquity = eq
		}
	}
	for _, e := range events {
		if e.exit {
			p, ok := open[e.i]
			if !ok {
				continue
			}
			delete(open, e.i)
			cash += p.stake + p.stake*(p.pnlPct/100)
			mark()
			continue
		}
		eq := equity()
		stake := eq * (stakePct / 100)
		if stake <= 0 || cash+1e-9 < stake {
			out.TradesSkipped++
			continue
		}
		cash -= stake
		open[e.i] = pos{stake: stake, pnlPct: fills[e.i].PnLPct}
		out.TradesTaken++
		mark()
	}
	out.OpenAtEnd = len(open)
	for _, p := range open {
		cash += p.stake + p.stake*(p.pnlPct/100)
	}
	out.EndingBalance = round2(cash)
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

// BookFillsFromSignals keeps trades that have a plan, in list order.
func BookFillsFromSignals(sigs []ScannerBacktestSignal, rangeEnd time.Time) []BacktestBookFill {
	out := make([]BacktestBookFill, 0, len(sigs))
	end := rangeEnd.UTC()
	for _, s := range sigs {
		if s.Trade == nil {
			continue
		}
		exit := end
		if s.Trade.HitAt != nil && !s.Trade.HitAt.IsZero() {
			exit = s.Trade.HitAt.UTC()
		}
		out = append(out, BacktestBookFill{
			Symbol:  s.Symbol,
			EntryAt: s.SignalAt.UTC(),
			ExitAt:  exit,
			PnLPct:  s.Trade.PnLPct,
			Outcome: s.Trade.Outcome,
		})
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
