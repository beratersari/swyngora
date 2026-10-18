package domain

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// PortfolioReplayNote is the what-if shown when moves come from a past window.
	PortfolioReplayNote = "What-if that replays each open position's own close-to-close change over the chosen window onto the current book. The path is that book at today's size on each close, so a drop in the middle stays visible if the end recovers. A margin position that is liquidated on the path stays liquidated afterward. A coin on two venues keeps each venue's move. movePct on the coin is the value-weighted average of the venues that priced. A venue with no close is named and that position stays unchanged. Cash does not change. Nothing is traded. Informational only."

	// ReplayReasonNoStart means no bar closed at or before the start.
	ReplayReasonNoStart = "no_start"
	// ReplayReasonNoEnd means no bar closed at or before the end.
	ReplayReasonNoEnd = "no_end"
	// ReplayReasonSameClose means the window still has only one close.
	ReplayReasonSameClose = "same_close"
	// ReplayReasonOutOfRange means the percent is outside the stress bounds.
	ReplayReasonOutOfRange = "out_of_range"
	// ReplayReasonNoHistory means the venue returned no usable bars.
	ReplayReasonNoHistory = "no_history"
	// ReplayReasonFetch means the candle read failed.
	ReplayReasonFetch = "fetch_failed"
)

// stressReplayMaxSpan is the longest window a replay will ask for.
const stressReplayMaxSpan = 3660 * 24 * time.Hour

// StressReplay is the past window that produced the stress moves.
type StressReplay struct {
	From      time.Time
	To        time.Time
	Interval  string
	Moves     []StressReplayMove
	Missing   []StressReplayGap
	Points    []StressPathPoint
	LowEquity float64
	LowAt     time.Time
}

// StressPathPoint is the current book repriced at one close in the window.
type StressPathPoint struct {
	At         time.Time
	Equity     float64
	Delta      float64
	DeltaPct   *float64
	Liquidated int
}

// StressReplayMove is one coin's percent, plus each venue that priced it.
type StressReplayMove struct {
	Asset   string
	MovePct float64
	Venues  []StressReplayVenue
}

// StressReplayVenue is one holding's close at each end of the window.
type StressReplayVenue struct {
	Exchange string
	Symbol   string
	Start    float64
	End      float64
	StartAt  time.Time
	EndAt    time.Time
	MovePct  float64
	Weight   float64
}

// StressReplayGap is a holding that was left unchanged.
type StressReplayGap struct {
	Asset    string
	Exchange string
	Symbol   string
	Reason   string
}

// ReplaySeries is one open holding and the candles used to price it.
// Failed means the candle read itself failed.
type ReplaySeries struct {
	Asset    string
	Exchange string
	Symbol   string
	Weight   float64
	Bars     []Candle
	Failed   bool
}

// ReplayPlan is the moves a past window can apply, and the holdings it cannot.
type ReplayPlan struct {
	Moves         []StressMove
	Replay        StressReplay
	FetchFailures int
	Series        int
}

// ParseStressWindow reads a past period.
// A YYYY-MM-DD pair is inclusive UTC days: the start price is the close entering
// the first day, and the end price is the close of the last day.
// An RFC3339 pair uses the last close at or before each instant.
// A window of at least 24 hours uses daily closes. A shorter window uses hourly closes.
// An end in the future is cut back to now.
func ParseStressWindow(fromRaw, toRaw string, now time.Time) (from, to time.Time, interval CandleInterval, err error) {
	fromRaw = strings.TrimSpace(fromRaw)
	toRaw = strings.TrimSpace(toRaw)
	if fromRaw == "" || toRaw == "" {
		return time.Time{}, time.Time{}, "", fmt.Errorf("%w: from and to are both required", ErrInvalidArgument)
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	from, fromDate, err := classifyStressBound("from", fromRaw)
	if err != nil {
		return time.Time{}, time.Time{}, "", err
	}
	to, toDate, err := classifyStressBound("to", toRaw)
	if err != nil {
		return time.Time{}, time.Time{}, "", err
	}
	if fromDate != toDate {
		return time.Time{}, time.Time{}, "", fmt.Errorf("%w: from and to must both be dates or both be times", ErrInvalidArgument)
	}
	if fromDate {
		to = to.Add(24 * time.Hour)
	}
	if to.After(now) {
		to = now
	}
	if !to.After(from) {
		return time.Time{}, time.Time{}, "", fmt.Errorf("%w: the period has to end after it starts", ErrInvalidArgument)
	}
	if to.Sub(from) < time.Hour {
		return time.Time{}, time.Time{}, "", fmt.Errorf("%w: the window has to be at least an hour", ErrInvalidArgument)
	}
	if to.Sub(from) > stressReplayMaxSpan {
		return time.Time{}, time.Time{}, "", fmt.Errorf("%w: the window can be at most 10 years", ErrInvalidArgument)
	}
	interval = Interval1d
	if to.Sub(from) < 24*time.Hour {
		interval = Interval1h
	}
	return from, to, interval, nil
}

func classifyStressBound(field, raw string) (time.Time, bool, error) {
	if len(raw) == 10 {
		if t, err := time.ParseInLocation("2006-01-02", raw, time.UTC); err == nil {
			return t, true, nil
		}
	}
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("%w: %s must be YYYY-MM-DD or an RFC3339 time", ErrInvalidArgument, field)
	}
	return t.UTC(), false, nil
}

// ReplayBarClose is when that candle's close is known.
// A close published after the open is used as-is.
// A missing or identical close time is treated as one interval after the open,
// which is how equity bars are stored.
func ReplayBarClose(c Candle, step time.Duration) time.Time {
	if !c.CloseTime.IsZero() && c.CloseTime.After(c.OpenTime) {
		return c.CloseTime.UTC()
	}
	if !c.OpenTime.IsZero() && step > 0 {
		return c.OpenTime.UTC().Add(step)
	}
	if !c.CloseTime.IsZero() {
		return c.CloseTime.UTC()
	}
	return time.Time{}
}

// BuildReplayPlan turns held candles into one percent per coin.
// A venue that cannot be priced is listed and left out of that coin's percent.
func BuildReplayPlan(in []ReplaySeries, from, to time.Time, interval CandleInterval, now time.Time) ReplayPlan {
	step := IntervalDuration(interval)
	plan := ReplayPlan{
		Series: len(in),
		Replay: StressReplay{
			From: from.UTC(), To: to.UTC(), Interval: string(interval),
		},
	}
	type group struct {
		asset string
		rows  []ReplaySeries
	}
	var order []string
	by := map[string]*group{}
	for _, row := range in {
		asset := stressAsset(row.Asset)
		if asset == "" {
			asset = stressAsset(row.Symbol)
		}
		if asset == "" {
			plan.Replay.Missing = append(plan.Replay.Missing, StressReplayGap{
				Exchange: row.Exchange, Symbol: row.Symbol, Reason: ReplayReasonNoHistory,
			})
			continue
		}
		g, ok := by[asset]
		if !ok {
			g = &group{asset: asset}
			by[asset] = g
			order = append(order, asset)
		}
		g.rows = append(g.rows, row)
	}
	for _, asset := range order {
		g := by[asset]
		var venues []StressReplayVenue
		var weight, weighted float64
		for _, row := range g.rows {
			if row.Failed {
				plan.FetchFailures++
				plan.Replay.Missing = append(plan.Replay.Missing, StressReplayGap{
					Asset: asset, Exchange: row.Exchange, Symbol: row.Symbol, Reason: ReplayReasonFetch,
				})
				continue
			}
			venue, reason := priceReplayVenue(row, step, from, to, now)
			if reason != "" {
				plan.Replay.Missing = append(plan.Replay.Missing, StressReplayGap{
					Asset: asset, Exchange: row.Exchange, Symbol: row.Symbol, Reason: reason,
				})
				continue
			}
			venue.Exchange = row.Exchange
			venue.Symbol = row.Symbol
			venue.Weight = row.Weight
			venues = append(venues, venue)
			if row.Weight > 0 {
				weight += row.Weight
				weighted += row.Weight * venue.MovePct
			}
		}
		if len(venues) == 0 || weight <= 0 {
			continue
		}
		move := weighted / weight
		if !validStressPct(move) {
			plan.Replay.Missing = append(plan.Replay.Missing, StressReplayGap{
				Asset: asset, Reason: ReplayReasonOutOfRange,
			})
			continue
		}
		for _, venue := range venues {
			plan.Moves = append(plan.Moves, StressMove{
				Asset: asset, Exchange: venue.Exchange, Symbol: venue.Symbol, MovePct: venue.MovePct,
			})
		}
		plan.Replay.Moves = append(plan.Replay.Moves, StressReplayMove{
			Asset: asset, MovePct: move, Venues: venues,
		})
	}
	sortReplayMoves(plan.Replay.Moves)
	return plan
}

// sortReplayMoves lists the largest price change first, whether it was up or down.
// Equal sizes stay in asset name order.
func sortReplayMoves(moves []StressReplayMove) {
	sort.SliceStable(moves, func(i, j int) bool {
		left := math.Abs(moves[i].MovePct)
		right := math.Abs(moves[j].MovePct)
		if math.Abs(left-right) > 1e-9 {
			return left > right
		}
		return moves[i].Asset < moves[j].Asset
	})
}

func priceReplayVenue(row ReplaySeries, step time.Duration, from, to, now time.Time) (StressReplayVenue, string) {
	bars := closedReplayBars(row.Bars, step, now)
	if len(bars) == 0 {
		return StressReplayVenue{}, ReplayReasonNoHistory
	}
	start, startAt, ok := replayCloseAt(bars, step, from)
	if !ok {
		return StressReplayVenue{}, ReplayReasonNoStart
	}
	end, endAt, ok := replayCloseAt(bars, step, to)
	if !ok {
		return StressReplayVenue{}, ReplayReasonNoEnd
	}
	if !endAt.After(startAt) {
		return StressReplayVenue{}, ReplayReasonSameClose
	}
	pct := (end/start - 1) * 100
	if !validStressPct(pct) {
		return StressReplayVenue{}, ReplayReasonOutOfRange
	}
	return StressReplayVenue{Start: start, End: end, StartAt: startAt, EndAt: endAt, MovePct: pct}, ""
}

func closedReplayBars(bars []Candle, step time.Duration, now time.Time) []Candle {
	if now.IsZero() {
		return bars
	}
	out := make([]Candle, 0, len(bars))
	for _, b := range bars {
		closeAt := ReplayBarClose(b, step)
		if closeAt.IsZero() || closeAt.After(now) {
			continue
		}
		out = append(out, b)
	}
	return out
}

func replayCloseAt(bars []Candle, step time.Duration, at time.Time) (float64, time.Time, bool) {
	var best time.Time
	var px float64
	found := false
	for _, b := range bars {
		closeAt := ReplayBarClose(b, step)
		if closeAt.IsZero() || closeAt.After(at) {
			continue
		}
		p, ok := parseReplayClose(b.Close)
		if !ok {
			continue
		}
		if !found || closeAt.After(best) {
			found = true
			best = closeAt
			px = p
		}
	}
	return px, best, found
}

func parseReplayClose(raw string) (float64, bool) {
	f, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || f <= 0 || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false
	}
	return f, true
}

func validStressPct(pct float64) bool {
	return pct > -100 && pct <= 500 && !math.IsNaN(pct) && !math.IsInf(pct, 0)
}
