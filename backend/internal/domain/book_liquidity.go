package domain

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	BookLiquidityWindow5m  = "5m"
	BookLiquidityWindow15m = "15m"
	BookLiquidityWindow1h  = "1h"

	DefaultBookLiquidityWindow = BookLiquidityWindow5m
	MaxBookLiquidityRange      = time.Hour
	MaxBookLiquidityLevels     = 80

	BookLiquidityReadTraded = "mostly_traded"
	BookLiquidityReadPulled = "mostly_pulled"
	BookLiquidityReadAdded  = "mostly_added"
	BookLiquidityReadMixed  = "mixed"
	BookLiquidityReadFlat   = "flat"
)

// BookLiquidityWindows are the named lookbacks the API accepts.
var BookLiquidityWindows = []struct {
	ID  string
	Dur time.Duration
}{
	{BookLiquidityWindow5m, 5 * time.Minute},
	{BookLiquidityWindow15m, 15 * time.Minute},
	{BookLiquidityWindow1h, time.Hour},
}

// BookLiquidityLevel is one price on one side of the book.
type BookLiquidityLevel struct {
	Side         string // bid | ask
	Price        float64
	FromNotional float64
	ToNotional   float64
	Added        float64
	Pulled       float64
	Traded       float64
	Remaining    float64
	Read         string
}

// BookLiquiditySide is bid or ask totals for the window.
type BookLiquiditySide struct {
	Side         string
	FromNotional float64
	ToNotional   float64
	Added        float64
	Pulled       float64
	Traded       float64
	Read         string
	Levels       []BookLiquidityLevel
}

// BookLiquidityPoint is added / pulled / traded in one sample interval.
type BookLiquidityPoint struct {
	From      time.Time
	To        time.Time
	BidAdded  float64
	BidPulled float64
	BidTraded float64
	AskAdded  float64
	AskPulled float64
	AskTraded float64
}

// BookLiquidityVenue is one exchange's attribution (or combined).
type BookLiquidityVenue struct {
	Exchange  Exchange
	Symbol    string
	From      time.Time
	To        time.Time
	TickSize  float64
	Samples   int
	Bids      BookLiquiditySide
	Asks      BookLiquiditySide
	Series    []BookLiquidityPoint
	Complete  bool
	Truncated bool
	KnownFrom time.Time
	KnownTo   time.Time
	Summary   string
	Error     string
}

// BookLiquidityReport is the API result.
type BookLiquidityReport struct {
	Symbol   string
	Exchange string
	Window   string
	From     time.Time
	To       time.Time
	AsOf     time.Time
	MinPrice float64
	MaxPrice float64
	Venues   []BookLiquidityVenue
	Combined *BookLiquidityVenue
	Summary  string
	Note     string
}

// SpotPrintPort loads aggressive spot fills in [from, to].
type SpotPrintPort interface {
	ListSpotPrints(ctx context.Context, symbol string, from, to time.Time) ([]TakerPrint, error)
}

// ParseBookLiquidityWindow accepts 5m / 15m / 1h (empty = 5m).
func ParseBookLiquidityWindow(raw string) (string, time.Duration, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		s = DefaultBookLiquidityWindow
	}
	for _, w := range BookLiquidityWindows {
		if w.ID == s {
			return w.ID, w.Dur, nil
		}
	}
	return "", 0, fmt.Errorf("%w: window must be 5m, 15m, or 1h", ErrInvalidArgument)
}

// ResolveBookLiquidityRange picks [from, to] from an explicit range or a named window.
func ResolveBookLiquidityRange(window string, start, end *time.Time, now time.Time) (from, to time.Time, windowID string, err error) {
	now = now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	hasStart := start != nil && !start.IsZero()
	hasEnd := end != nil && !end.IsZero()
	if hasStart || hasEnd {
		to = now
		if hasEnd {
			to = end.UTC()
		}
		from = to.Add(-5 * time.Minute)
		if hasStart {
			from = start.UTC()
		}
		if !to.After(from) {
			return time.Time{}, time.Time{}, "", fmt.Errorf("%w: endTime must be after startTime", ErrInvalidArgument)
		}
		if to.Sub(from) > MaxBookLiquidityRange {
			return time.Time{}, time.Time{}, "", fmt.Errorf("%w: range must be <= 1h", ErrInvalidArgument)
		}
		return from, to, "custom", nil
	}
	id, dur, err := ParseBookLiquidityWindow(window)
	if err != nil {
		return time.Time{}, time.Time{}, "", err
	}
	return now.Add(-dur), now, id, nil
}

// AttributeBookSize splits a rest change into added, pulled, and traded.
// Identity: to = from + added − pulled − traded.
func AttributeBookSize(fromN, toN, traded float64) (added, pulled, eaten float64) {
	if fromN < 0 {
		fromN = 0
	}
	if toN < 0 {
		toN = 0
	}
	if traded < 0 {
		traded = 0
	}
	eaten = traded
	shrink := fromN - toN
	if shrink > 0 {
		if traded >= shrink {
			added = traded - shrink
			return added, 0, eaten
		}
		return 0, shrink - traded, eaten
	}
	return -shrink + traded, 0, eaten
}

// BuildBookLiquidityVenue attributes resting size changes using prints between snapshots.
func BuildBookLiquidityVenue(ex Exchange, symbol string, snaps []BookHistorySnapshot, prints []TakerPrint, minPrice, maxPrice float64, from, to time.Time) BookLiquidityVenue {
	symbol = NormalizeSymbol(ex, symbol)
	from, to = from.UTC(), to.UTC()
	out := BookLiquidityVenue{
		Exchange: ex,
		Symbol:   symbol,
		From:     from,
		To:       to,
		Bids:     BookLiquiditySide{Side: "bid", Levels: []BookLiquidityLevel{}},
		Asks:     BookLiquiditySide{Side: "ask", Levels: []BookLiquidityLevel{}},
		Series:   []BookLiquidityPoint{},
	}
	snaps = filterBookLiquiditySnaps(snaps, from, to)
	sort.SliceStable(snaps, func(i, j int) bool { return snaps[i].SampledAt.Before(snaps[j].SampledAt) })
	if len(snaps) < 2 {
		out.Error = "need two stored books in this range"
		out.Summary = out.Error
		return out
	}
	out.Samples = len(snaps)
	out.KnownFrom = snaps[0].SampledAt.UTC()
	out.KnownTo = snaps[len(snaps)-1].SampledAt.UTC()
	tick := snaps[0].GroupSize
	for _, s := range snaps {
		if s.GroupSize > tick {
			tick = s.GroupSize
		}
	}
	if tick <= 0 {
		tick = 1e-8
	}
	out.TickSize = tick

	first, last := snaps[0], snaps[len(snaps)-1]
	bidTraded, askTraded := tradeNotionalByPrice(prints, first.SampledAt, last.SampledAt, tick, minPrice, maxPrice)
	out.Bids = buildBookLiquiditySide("bid", first.Bids, last.Bids, bidTraded, tick, minPrice, maxPrice)
	out.Asks = buildBookLiquiditySide("ask", first.Asks, last.Asks, askTraded, tick, minPrice, maxPrice)
	out.Series = buildBookLiquiditySeries(snaps, prints, tick, minPrice, maxPrice)
	out.Complete = bookLiquidityComplete(out.KnownFrom, from, to.Sub(from))
	out.Truncated = !out.Complete
	out.Summary = ExplainBookLiquidityVenue(out)
	return out
}

// CombineBookLiquidityVenues adds bid/ask notionals at the same tick.
func CombineBookLiquidityVenues(symbol string, venues []BookLiquidityVenue, from, to time.Time) *BookLiquidityVenue {
	out := &BookLiquidityVenue{
		Exchange: "all",
		Symbol:   symbol,
		From:     from.UTC(),
		To:       to.UTC(),
		Bids:     BookLiquiditySide{Side: "bid", Levels: []BookLiquidityLevel{}},
		Asks:     BookLiquiditySide{Side: "ask", Levels: []BookLiquidityLevel{}},
		Series:   []BookLiquidityPoint{},
	}
	var usable []BookLiquidityVenue
	for _, v := range venues {
		if v.Error != "" {
			continue
		}
		usable = append(usable, v)
	}
	if len(usable) == 0 {
		out.Error = "need two stored books in this range"
		out.Summary = out.Error
		return out
	}
	tick := usable[0].TickSize
	complete := true
	var oldest, newest time.Time
	for _, v := range usable {
		if v.TickSize > tick {
			tick = v.TickSize
		}
		if !v.Complete {
			complete = false
		}
		if v.Truncated {
			out.Truncated = true
		}
		out.Samples += v.Samples
		if !v.KnownFrom.IsZero() && (oldest.IsZero() || v.KnownFrom.Before(oldest)) {
			oldest = v.KnownFrom
		}
		if !v.KnownTo.IsZero() && (newest.IsZero() || v.KnownTo.After(newest)) {
			newest = v.KnownTo
		}
		out.Bids.FromNotional += v.Bids.FromNotional
		out.Bids.ToNotional += v.Bids.ToNotional
		out.Bids.Added += v.Bids.Added
		out.Bids.Pulled += v.Bids.Pulled
		out.Bids.Traded += v.Bids.Traded
		out.Asks.FromNotional += v.Asks.FromNotional
		out.Asks.ToNotional += v.Asks.ToNotional
		out.Asks.Added += v.Asks.Added
		out.Asks.Pulled += v.Asks.Pulled
		out.Asks.Traded += v.Asks.Traded
	}
	out.TickSize = tick
	out.KnownFrom, out.KnownTo = oldest, newest
	out.Complete = complete && !out.Truncated
	out.Bids.Levels = mergeBookLiquidityLevels("bid", usable, tick)
	out.Asks.Levels = mergeBookLiquidityLevels("ask", usable, tick)
	out.Bids.Read = bookLiquidityRead(out.Bids.Added, out.Bids.Pulled, out.Bids.Traded)
	out.Asks.Read = bookLiquidityRead(out.Asks.Added, out.Asks.Pulled, out.Asks.Traded)
	out.Series = mergeBookLiquiditySeries(usable)
	out.Summary = ExplainBookLiquidityVenue(*out)
	return out
}

// ExplainBookLiquidityVenue writes a short traded-vs-pulled line.
func ExplainBookLiquidityVenue(v BookLiquidityVenue) string {
	if v.Error != "" {
		return v.Error
	}
	name := prettyBase(v.Symbol)
	head := fmt.Sprintf("%s bids: %s traded vs %s pulled (added %s). Asks: %s traded vs %s pulled (added %s).",
		name,
		formatQty(v.Bids.Traded), formatQty(v.Bids.Pulled), formatQty(v.Bids.Added),
		formatQty(v.Asks.Traded), formatQty(v.Asks.Pulled), formatQty(v.Asks.Added))
	if row := largestBookPull(v.Bids.Levels, v.Asks.Levels); row.Price > 0 && row.Pulled > 0 {
		head += fmt.Sprintf(" Largest pull: %s %s lost %s, %s traded / %s pulled.",
			row.Side, formatQty(row.Price), formatQty(row.FromNotional-row.ToNotional),
			formatQty(row.Traded), formatQty(row.Pulled))
	}
	if v.Truncated && !v.KnownFrom.IsZero() {
		head += fmt.Sprintf(" Book samples start at %s.", v.KnownFrom.UTC().Format(time.RFC3339))
	}
	return head
}

// ExplainBookLiquidityReport prefers combined, then the first venue with samples.
func ExplainBookLiquidityReport(r BookLiquidityReport) string {
	if r.Combined != nil && r.Combined.Error == "" {
		return r.Combined.Summary
	}
	for _, v := range r.Venues {
		if v.Error == "" && v.Summary != "" {
			return v.Summary
		}
	}
	for _, v := range r.Venues {
		if v.Summary != "" {
			return v.Summary
		}
	}
	return prettyBase(r.Symbol) + ": need two stored books in this range."
}

func filterBookLiquiditySnaps(snaps []BookHistorySnapshot, from, to time.Time) []BookHistorySnapshot {
	out := make([]BookHistorySnapshot, 0, len(snaps))
	for _, s := range snaps {
		t := s.SampledAt.UTC()
		if t.IsZero() {
			continue
		}
		if t.Before(from.Add(-DefaultBookHistorySlack)) || t.After(to.Add(DefaultBookHistorySlack)) {
			continue
		}
		out = append(out, s)
	}
	return out
}

func buildBookLiquiditySide(side string, fromLv, toLv []BookHistoryLevel, traded map[float64]float64, tick, minPrice, maxPrice float64) BookLiquiditySide {
	fromMap := indexBookNotional(fromLv, tick)
	toMap := indexBookNotional(toLv, tick)
	keys := map[float64]struct{}{}
	for p := range fromMap {
		keys[p] = struct{}{}
	}
	for p := range toMap {
		keys[p] = struct{}{}
	}
	for p := range traded {
		keys[p] = struct{}{}
	}
	prices := make([]float64, 0, len(keys))
	for p := range keys {
		if minPrice > 0 && p < minPrice {
			continue
		}
		if maxPrice > 0 && p > maxPrice {
			continue
		}
		prices = append(prices, p)
	}
	if side == "bid" {
		sort.Slice(prices, func(i, j int) bool { return prices[i] > prices[j] })
	} else {
		sort.Slice(prices, func(i, j int) bool { return prices[i] < prices[j] })
	}
	out := BookLiquiditySide{Side: side, Levels: make([]BookLiquidityLevel, 0, len(prices))}
	for _, p := range prices {
		a, b, t := fromMap[p], toMap[p], traded[p]
		added, pulled, eaten := AttributeBookSize(a, b, t)
		if a <= 0 && b <= 0 && eaten <= 0 {
			continue
		}
		row := BookLiquidityLevel{
			Side: side, Price: p, FromNotional: a, ToNotional: b,
			Added: added, Pulled: pulled, Traded: eaten, Remaining: b,
			Read: bookLiquidityRead(added, pulled, eaten),
		}
		out.Levels = append(out.Levels, row)
		out.FromNotional += a
		out.ToNotional += b
		out.Added += added
		out.Pulled += pulled
		out.Traded += eaten
	}
	if len(out.Levels) > MaxBookLiquidityLevels {
		out.Levels = out.Levels[:MaxBookLiquidityLevels]
	}
	out.Read = bookLiquidityRead(out.Added, out.Pulled, out.Traded)
	return out
}

func buildBookLiquiditySeries(snaps []BookHistorySnapshot, prints []TakerPrint, tick, minPrice, maxPrice float64) []BookLiquidityPoint {
	if len(snaps) < 2 {
		return nil
	}
	out := make([]BookLiquidityPoint, 0, len(snaps)-1)
	for i := 1; i < len(snaps); i++ {
		a, b := snaps[i-1], snaps[i]
		bidT, askT := tradeNotionalByPrice(prints, a.SampledAt, b.SampledAt, tick, minPrice, maxPrice)
		bid := buildBookLiquiditySide("bid", a.Bids, b.Bids, bidT, tick, minPrice, maxPrice)
		ask := buildBookLiquiditySide("ask", a.Asks, b.Asks, askT, tick, minPrice, maxPrice)
		out = append(out, BookLiquidityPoint{
			From: a.SampledAt.UTC(), To: b.SampledAt.UTC(),
			BidAdded: bid.Added, BidPulled: bid.Pulled, BidTraded: bid.Traded,
			AskAdded: ask.Added, AskPulled: ask.Pulled, AskTraded: ask.Traded,
		})
	}
	return out
}

func tradeNotionalByPrice(prints []TakerPrint, from, to time.Time, tick, minPrice, maxPrice float64) (bidHits, askHits map[float64]float64) {
	bidHits = map[float64]float64{}
	askHits = map[float64]float64{}
	from, to = from.UTC(), to.UTC()
	for _, p := range prints {
		if p.Notional <= 0 || p.Price <= 0 {
			continue
		}
		t := p.Time.UTC()
		if t.Before(from) || t.After(to) {
			continue
		}
		px := bookBucketPrice(p.Price, tick)
		if minPrice > 0 && px < minPrice {
			continue
		}
		if maxPrice > 0 && px > maxPrice {
			continue
		}
		switch p.Side {
		case TakerSideSell:
			bidHits[px] += p.Notional
		case TakerSideBuy:
			askHits[px] += p.Notional
		}
	}
	return bidHits, askHits
}

func indexBookNotional(levels []BookHistoryLevel, tick float64) map[float64]float64 {
	out := map[float64]float64{}
	for _, lv := range levels {
		if lv.Price <= 0 || lv.Notional <= 0 {
			continue
		}
		out[bookBucketPrice(lv.Price, tick)] += lv.Notional
	}
	return out
}

func bookBucketPrice(price, step float64) float64 {
	if step <= 0 {
		return price
	}
	return math.Round(price/step) * step
}

func bookLiquidityRead(added, pulled, traded float64) string {
	if added <= 0 && pulled <= 0 && traded <= 0 {
		return BookLiquidityReadFlat
	}
	switch {
	case pulled > 0 && pulled >= 2*traded && pulled >= added:
		return BookLiquidityReadPulled
	case traded > 0 && traded >= 2*pulled && traded >= added:
		return BookLiquidityReadTraded
	case added > 0 && added >= 2*pulled && added >= 2*traded:
		return BookLiquidityReadAdded
	default:
		return BookLiquidityReadMixed
	}
}

func bookLiquidityComplete(oldest, from time.Time, dur time.Duration) bool {
	if oldest.IsZero() || from.IsZero() {
		return false
	}
	slack := dur / 5
	if slack < time.Minute {
		slack = time.Minute
	}
	return !oldest.After(from.Add(slack))
}

func largestBookPull(sides ...[]BookLiquidityLevel) BookLiquidityLevel {
	var best BookLiquidityLevel
	for _, levels := range sides {
		for _, lv := range levels {
			if lv.Pulled > best.Pulled {
				best = lv
			}
		}
	}
	return best
}

func mergeBookLiquidityLevels(side string, venues []BookLiquidityVenue, tick float64) []BookLiquidityLevel {
	type acc struct{ from, to, added, pulled, traded float64 }
	m := map[float64]*acc{}
	for _, v := range venues {
		var levels []BookLiquidityLevel
		if side == "bid" {
			levels = v.Bids.Levels
		} else {
			levels = v.Asks.Levels
		}
		for _, lv := range levels {
			p := bookBucketPrice(lv.Price, tick)
			cur := m[p]
			if cur == nil {
				cur = &acc{}
				m[p] = cur
			}
			cur.from += lv.FromNotional
			cur.to += lv.ToNotional
			cur.added += lv.Added
			cur.pulled += lv.Pulled
			cur.traded += lv.Traded
		}
	}
	prices := make([]float64, 0, len(m))
	for p := range m {
		prices = append(prices, p)
	}
	if side == "bid" {
		sort.Slice(prices, func(i, j int) bool { return prices[i] > prices[j] })
	} else {
		sort.Slice(prices, func(i, j int) bool { return prices[i] < prices[j] })
	}
	out := make([]BookLiquidityLevel, 0, len(prices))
	for _, p := range prices {
		a := m[p]
		out = append(out, BookLiquidityLevel{
			Side: side, Price: p, FromNotional: a.from, ToNotional: a.to,
			Added: a.added, Pulled: a.pulled, Traded: a.traded, Remaining: a.to,
			Read: bookLiquidityRead(a.added, a.pulled, a.traded),
		})
	}
	if len(out) > MaxBookLiquidityLevels {
		out = out[:MaxBookLiquidityLevels]
	}
	return out
}

func mergeBookLiquiditySeries(venues []BookLiquidityVenue) []BookLiquidityPoint {
	type key struct{ from, to int64 }
	m := map[key]*BookLiquidityPoint{}
	for _, v := range venues {
		for _, p := range v.Series {
			k := key{p.From.Unix(), p.To.Unix()}
			cur := m[k]
			if cur == nil {
				cp := p
				m[k] = &cp
				continue
			}
			cur.BidAdded += p.BidAdded
			cur.BidPulled += p.BidPulled
			cur.BidTraded += p.BidTraded
			cur.AskAdded += p.AskAdded
			cur.AskPulled += p.AskPulled
			cur.AskTraded += p.AskTraded
		}
	}
	out := make([]BookLiquidityPoint, 0, len(m))
	for _, p := range m {
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].From.Before(out[j].From) })
	return out
}
