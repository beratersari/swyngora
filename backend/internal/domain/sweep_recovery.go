package domain

import (
	"fmt"
	"math"
	"sort"
	"time"
)

const (
	SweepRecoveryTarget      = 0.80
	SweepRecoveryLookback    = 20 * time.Minute
	SweepRecoveryHorizon     = 2 * time.Hour
	SweepRecoveryMinBandPct  = 0.30
	SweepRecoveryCoverPct    = 2.5
	MaxSweepRecoveries       = 8
	SweepRecoverySampleLimit = 200
)

// SweepRecovery is how long resting size near a swept level took to refill.
type SweepRecovery struct {
	Exchange        string    `json:"exchange"`
	Symbol          string    `json:"symbol"`
	Side            string    `json:"side"`
	Level           string    `json:"level"`
	PiercedAt       time.Time `json:"piercedAt"`
	ReclaimedAt     time.Time `json:"reclaimedAt"`
	BeforeNotional  string    `json:"beforeNotional"`
	TroughNotional  string    `json:"troughNotional"`
	BackNotional    string    `json:"backNotional"`
	Recovered       bool      `json:"recovered"`
	RecoverySeconds int       `json:"recoverySeconds"`
	RecoveredPct    float64   `json:"recoveredPct"`
	Summary         string    `json:"summary"`
}

// SweepRecoveryReport is the refill clock for recent sweeps on one symbol.
type SweepRecoveryReport struct {
	Symbol                string          `json:"symbol"`
	Exchange              string          `json:"exchange"`
	AsOf                  time.Time       `json:"asOf"`
	MedianRecoverySeconds int             `json:"medianRecoverySeconds"`
	Recovered             int             `json:"recovered"`
	Pending               int             `json:"pending"`
	Rows                  []SweepRecovery `json:"rows"`
	Summary               string          `json:"summary"`
	Note                  string          `json:"note"`
}

// MeasureSweepRecovery compares stored ladders before the poke, at the thin
// point, and after price came back. Recovered means the swept side is back
// to 80% of the size that was resting just before the poke.
func MeasureSweepRecovery(exchange, symbol string, sweep LiquiditySweep, snaps []BookHistorySnapshot) SweepRecovery {
	out := SweepRecovery{
		Exchange: exchange,
		Symbol:   symbol,
		Side:     sweep.Side,
		Level:    formatQty(sweep.Level),
	}
	if sweep.Level <= 0 || sweep.PiercedAt.IsZero() || sweep.ReclaimedAt.IsZero() || sweep.Status != LiquiditySweepSwept {
		out.Summary = "This sweep has no return yet, so refill time is not measured."
		return out
	}
	out.PiercedAt = sweep.PiercedAt.UTC()
	out.ReclaimedAt = sweep.ReclaimedAt.UTC()
	band := sweep.ExcursionPct
	if band < SweepRecoveryMinBandPct {
		band = SweepRecoveryMinBandPct
	}
	ordered := append([]BookHistorySnapshot(nil), snaps...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].SampledAt.Before(ordered[j].SampledAt)
	})
	before, beforeOK := depthAtOrBefore(ordered, out.PiercedAt, sweep)
	if !beforeOK || before.notional <= 0 {
		out.Summary = "No stored book covered the level just before the poke."
		return out
	}
	out.BeforeNotional = formatQty(before.notional)
	trough := before.notional
	for _, snap := range ordered {
		at := snap.SampledAt.UTC()
		if at.Before(out.PiercedAt) || at.After(out.ReclaimedAt.Add(time.Minute)) {
			continue
		}
		n, ok := sweepDepth(snap, sweep, band)
		if !ok {
			continue
		}
		if n < trough {
			trough = n
		}
	}
	out.TroughNotional = formatQty(trough)
	target := before.notional * SweepRecoveryTarget
	var last float64
	var lastOK bool
	for _, snap := range ordered {
		at := snap.SampledAt.UTC()
		if at.Before(out.ReclaimedAt) {
			continue
		}
		if at.After(out.ReclaimedAt.Add(SweepRecoveryHorizon)) {
			break
		}
		n, ok := sweepDepth(snap, sweep, band)
		if !ok {
			continue
		}
		last, lastOK = n, true
		if n+1e-6 >= target {
			out.Recovered = true
			out.BackNotional = formatQty(n)
			out.RecoverySeconds = int(at.Sub(out.ReclaimedAt).Seconds())
			if out.RecoverySeconds < 0 {
				out.RecoverySeconds = 0
			}
			out.RecoveredPct = round4(n / before.notional * 100)
			out.Summary = fmt.Sprintf("%s depth near %s was back to %s in %s after price returned.",
				sideWord(sweep.Side), out.Level, formatQty(n), formatRecovery(out.RecoverySeconds))
			return out
		}
	}
	if lastOK {
		out.BackNotional = formatQty(last)
		out.RecoveredPct = round4(last / before.notional * 100)
	}
	out.Summary = fmt.Sprintf("%s depth near %s had not returned to 80%% of %s within %s of the return.",
		sideWord(sweep.Side), out.Level, out.BeforeNotional, formatRecovery(int(SweepRecoveryHorizon.Seconds())))
	return out
}

type depthPoint struct {
	at       time.Time
	notional float64
}

func depthAtOrBefore(snaps []BookHistorySnapshot, at time.Time, sweep LiquiditySweep) (depthPoint, bool) {
	band := sweep.ExcursionPct
	if band < SweepRecoveryMinBandPct {
		band = SweepRecoveryMinBandPct
	}
	var best depthPoint
	ok := false
	for _, snap := range snaps {
		ts := snap.SampledAt.UTC()
		if ts.After(at) || at.Sub(ts) > SweepRecoveryLookback {
			continue
		}
		n, covered := sweepDepth(snap, sweep, band)
		if !covered {
			continue
		}
		if !ok || ts.After(best.at) {
			best = depthPoint{at: ts, notional: n}
			ok = true
		}
	}
	return best, ok
}

func sweepDepth(snap BookHistorySnapshot, sweep LiquiditySweep, bandPct float64) (float64, bool) {
	if sweep.Level <= 0 || snap.Mid <= 0 {
		return 0, false
	}
	if math.Abs(snap.Mid-sweep.Level)/sweep.Level*100 > SweepRecoveryCoverPct {
		return 0, false
	}
	hi := sweep.Level * (1 + bandPct/100)
	lo := sweep.Level * (1 - bandPct/100)
	var n float64
	if sweep.Side == LiquiditySweepSideHigh {
		for _, lv := range snap.Asks {
			if lv.Price >= sweep.Level && lv.Price <= hi {
				n += levelNotional(lv)
			}
		}
		return n, true
	}
	for _, lv := range snap.Bids {
		if lv.Price <= sweep.Level && lv.Price >= lo {
			n += levelNotional(lv)
		}
	}
	return n, true
}

func levelNotional(lv BookHistoryLevel) float64 {
	if lv.Notional > 0 {
		return lv.Notional
	}
	if lv.Price > 0 && lv.Quantity > 0 {
		return lv.Price * lv.Quantity
	}
	return 0
}

func formatRecovery(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm", seconds/60)
	}
	return fmt.Sprintf("%dh %dm", seconds/3600, (seconds%3600)/60)
}

// BuildSweepRecoveryReport keeps the newest returned sweeps and the median refill time.
func BuildSweepRecoveryReport(symbol, exchange string, rows []SweepRecovery, now time.Time) SweepRecoveryReport {
	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i].ReclaimedAt.After(rows[j].ReclaimedAt)
	})
	if len(rows) > MaxSweepRecoveries {
		rows = rows[:MaxSweepRecoveries]
	}
	if rows == nil {
		rows = []SweepRecovery{}
	}
	var times []float64
	pending := 0
	for _, row := range rows {
		if row.Recovered {
			times = append(times, float64(row.RecoverySeconds))
		} else if row.BeforeNotional != "" {
			pending++
		}
	}
	median := 0
	if len(times) > 0 {
		median = int(medianFloat(times))
	}
	out := SweepRecoveryReport{
		Symbol: symbol, Exchange: exchange, AsOf: now.UTC(),
		MedianRecoverySeconds: median, Recovered: len(times), Pending: pending, Rows: rows,
	}
	if len(times) == 0 {
		out.Summary = "No stored book yet shows the swept depth coming back."
		return out
	}
	out.Summary = fmt.Sprintf("After %d sweeps the typical refill was %s. %d were still thin after two hours.",
		len(times), formatRecovery(median), pending)
	return out
}
