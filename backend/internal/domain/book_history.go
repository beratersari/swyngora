package domain

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultBookHistoryLimit  = 60
	MaxBookHistoryLimit      = 500
	DefaultBookHistoryLevels = 25
	DefaultBookHistoryBucket = time.Minute
	DefaultBookHistorySlack  = 3 * time.Minute
	BookHistoryChangeGained  = "gained"
	BookHistoryChangeLost    = "lost"
	BookHistoryChangeAdded   = "added"
	BookHistoryChangeRemoved = "removed"
)

// DefaultBookHistorySymbols are always sampled by the background worker.
var DefaultBookHistorySymbols = DefaultFuturesHistorySymbols

// BookHistoryLevel is one stored bid or ask bucket.
type BookHistoryLevel struct {
	Price    float64
	Quantity float64
	Notional float64
	Wall     bool
}

// BookHistoryWall is a large resting cluster at sample time.
type BookHistoryWall struct {
	Side        string
	Price       float64
	Quantity    float64
	Notional    float64
	DistancePct float64
	Share       float64
}

// BookHistorySnapshot is one durable order-book sample.
type BookHistorySnapshot struct {
	Exchange     Exchange
	Symbol       string
	SampledAt    time.Time
	Mid          float64
	BestBid      float64
	BestAsk      float64
	Spread       float64
	SpreadPct    float64
	GroupSize    float64
	BidNotional  float64
	AskNotional  float64
	Imbalance    float64
	Pressure     string
	BidWalls     int
	AskWalls     int
	Live         bool
	Bids         []BookHistoryLevel
	Asks         []BookHistoryLevel
	Walls        []BookHistoryWall
	Complete     bool
	SlackSeconds float64
}

// BookHistoryQuery lists or finds stored books.
type BookHistoryQuery struct {
	Exchange string
	Symbol   string
	From     time.Time
	To       time.Time
	At       time.Time
	Limit    int
}

// BookHistoryLevelChange is how one price bucket moved between two samples.
type BookHistoryLevelChange struct {
	Side         string
	Price        float64
	FromNotional float64
	ToNotional   float64
	Delta        float64
	Change       string // gained | lost | added | removed
}

// BookHistoryDiff is a before/after read of the same book.
type BookHistoryDiff struct {
	Symbol           string
	Exchange         string
	From             BookHistorySnapshot
	To               BookHistorySnapshot
	MidDelta         float64
	MidDeltaPct      float64
	SpreadDelta      float64
	ImbalanceDelta   float64
	BidNotionalDelta float64
	AskNotionalDelta float64
	Gained           []BookHistoryLevelChange
	Lost             []BookHistoryLevelChange
	WallsAdded       []BookHistoryWall
	WallsRemoved     []BookHistoryWall
	Summary          string
	Note             string
}

// BookHistoryReport is a point-in-time book or a list of samples.
type BookHistoryReport struct {
	Symbol    string
	Exchange  string
	At        time.Time
	Snapshot  *BookHistorySnapshot
	Snapshots []BookHistorySnapshot
	Summary   string
	Note      string
}

// BookHistoryStore persists regular order-book samples.
type BookHistoryStore interface {
	InsertSnapshot(ctx context.Context, rec BookHistorySnapshot) (inserted bool, err error)
	NearestAt(ctx context.Context, exchange, symbol string, at time.Time) (*BookHistorySnapshot, error)
	ListSnapshots(ctx context.Context, q BookHistoryQuery) ([]BookHistorySnapshot, error)
	PurgeOlderThan(ctx context.Context, cutoff time.Time) (int, error)
}

// ParseBookHistoryLimit clamps list size.
func ParseBookHistoryLimit(n int) int {
	if n <= 0 {
		return DefaultBookHistoryLimit
	}
	if n > MaxBookHistoryLimit {
		return MaxBookHistoryLimit
	}
	return n
}

// CaptureBookHistory builds a compact sample from a live raw book.
func CaptureBookHistory(ex Exchange, raw RawOrderBook, sampledAt time.Time) BookHistorySnapshot {
	if sampledAt.IsZero() {
		sampledAt = time.Now().UTC()
	}
	sampledAt = TruncateToBucket(sampledAt.UTC(), DefaultBookHistoryBucket)
	grouped := GroupOrderBook(raw, 0, DefaultBookHistoryLevels)
	an := AnalyzeOrderBook(raw, DefaultOrderBookRangePct)
	group, _ := strconv.ParseFloat(grouped.GroupSize, 64)
	mid := parseHistFloat(grouped.LastPrice)
	if mid <= 0 {
		mid = bookMid(raw)
	}
	out := BookHistorySnapshot{
		Exchange:    ex,
		Symbol:      raw.Symbol,
		SampledAt:   sampledAt,
		Mid:         mid,
		BestBid:     parseHistFloat(grouped.BestBid),
		BestAsk:     parseHistFloat(grouped.BestAsk),
		Spread:      parseHistFloat(grouped.Spread),
		SpreadPct:   parseHistFloat(grouped.SpreadPct),
		GroupSize:   group,
		BidNotional: parseHistFloat(an.BidNotional),
		AskNotional: parseHistFloat(an.AskNotional),
		Imbalance:   an.Imbalance,
		Pressure:    an.Pressure,
		BidWalls:    grouped.BidWalls,
		AskWalls:    grouped.AskWalls,
		Live:        raw.Live,
		Bids:        bookLevelsFromGrouped(grouped.Bids),
		Asks:        bookLevelsFromGrouped(grouped.Asks),
		Walls:       bookWallsFromAnalysis(an.Walls),
		Complete:    true,
	}
	if out.Symbol == "" {
		out.Symbol = grouped.Symbol
	}
	if out.BidNotional <= 0 {
		out.BidNotional = bookSideNotional(out.Bids)
	}
	if out.AskNotional <= 0 {
		out.AskNotional = bookSideNotional(out.Asks)
	}
	return out
}

// NearestBookSnapshot picks the sample closest to at, preferring at-or-before.
func NearestBookSnapshot(rows []BookHistorySnapshot, at time.Time, slack time.Duration) (BookHistorySnapshot, bool) {
	if slack <= 0 {
		slack = DefaultBookHistorySlack
	}
	if at.IsZero() || len(rows) == 0 {
		return BookHistorySnapshot{}, false
	}
	at = at.UTC()
	var before, after BookHistorySnapshot
	haveBefore, haveAfter := false, false
	for _, r := range rows {
		if r.SampledAt.IsZero() {
			continue
		}
		t := r.SampledAt.UTC()
		if !t.After(at) {
			if !haveBefore || t.After(before.SampledAt) {
				before, haveBefore = r, true
			}
			continue
		}
		if !haveAfter || t.Before(after.SampledAt) {
			after, haveAfter = r, true
		}
	}
	if haveBefore && at.Sub(before.SampledAt.UTC()) <= slack {
		return markSlack(before, at), true
	}
	if haveAfter && after.SampledAt.UTC().Sub(at) <= slack {
		return markSlack(after, at), true
	}
	if haveBefore {
		got := markSlack(before, at)
		got.Complete = got.SlackSeconds <= slack.Seconds()
		return got, true
	}
	if haveAfter {
		got := markSlack(after, at)
		got.Complete = got.SlackSeconds <= slack.Seconds()
		return got, true
	}
	return BookHistorySnapshot{}, false
}

// CompareBookHistory diffs two samples of the same book.
func CompareBookHistory(from, to BookHistorySnapshot) BookHistoryDiff {
	out := BookHistoryDiff{
		Symbol:   firstNonEmpty(to.Symbol, from.Symbol),
		Exchange: firstNonEmpty(string(to.Exchange), string(from.Exchange)),
		From:     from,
		To:       to,
		Note:     bookHistoryDisclaimer,
	}
	out.MidDelta = to.Mid - from.Mid
	if from.Mid > 0 {
		out.MidDeltaPct = out.MidDelta / from.Mid * 100
	}
	out.SpreadDelta = to.Spread - from.Spread
	out.ImbalanceDelta = to.Imbalance - from.Imbalance
	out.BidNotionalDelta = to.BidNotional - from.BidNotional
	out.AskNotionalDelta = to.AskNotional - from.AskNotional

	step := from.GroupSize
	if to.GroupSize > step {
		step = to.GroupSize
	}
	if step <= 0 {
		step = 1e-8
	}
	fromMap := map[string]float64{}
	toMap := map[string]float64{}
	indexSide := func(side string, levels []BookHistoryLevel, dest map[string]float64) {
		for _, lv := range levels {
			if lv.Price <= 0 {
				continue
			}
			dest[bookLevelKey(side, lv.Price, step)] += lv.Notional
		}
	}
	indexSide("bid", from.Bids, fromMap)
	indexSide("ask", from.Asks, fromMap)
	indexSide("bid", to.Bids, toMap)
	indexSide("ask", to.Asks, toMap)

	seen := map[string]struct{}{}
	var changes []BookHistoryLevelChange
	addChange := func(side string, price, a, b float64) {
		ch := BookHistoryLevelChange{Side: side, Price: price, FromNotional: a, ToNotional: b, Delta: b - a}
		switch {
		case a <= 0 && b > 0:
			ch.Change = BookHistoryChangeAdded
		case a > 0 && b <= 0:
			ch.Change = BookHistoryChangeRemoved
		case b > a:
			ch.Change = BookHistoryChangeGained
		case b < a:
			ch.Change = BookHistoryChangeLost
		default:
			return
		}
		changes = append(changes, ch)
	}
	for k, a := range fromMap {
		seen[k] = struct{}{}
		side, price := parseBookLevelKey(k)
		addChange(side, price, a, toMap[k])
	}
	for k, b := range toMap {
		if _, ok := seen[k]; ok {
			continue
		}
		side, price := parseBookLevelKey(k)
		addChange(side, price, 0, b)
	}
	for _, ch := range changes {
		switch ch.Change {
		case BookHistoryChangeGained, BookHistoryChangeAdded:
			out.Gained = append(out.Gained, ch)
		case BookHistoryChangeLost, BookHistoryChangeRemoved:
			out.Lost = append(out.Lost, ch)
		}
	}
	sort.SliceStable(out.Gained, func(i, j int) bool { return out.Gained[i].Delta > out.Gained[j].Delta })
	sort.SliceStable(out.Lost, func(i, j int) bool { return out.Lost[i].Delta < out.Lost[j].Delta })

	fromWalls := wallKeySet(from.Walls, step)
	toWalls := wallKeySet(to.Walls, step)
	for k, w := range toWalls {
		if _, ok := fromWalls[k]; !ok {
			out.WallsAdded = append(out.WallsAdded, w)
		}
	}
	for k, w := range fromWalls {
		if _, ok := toWalls[k]; !ok {
			out.WallsRemoved = append(out.WallsRemoved, w)
		}
	}
	out.Summary = ExplainBookDiff(out)
	return out
}

// ExplainBookHistory writes a short point-in-time read.
func ExplainBookHistory(s BookHistorySnapshot) string {
	if s.SampledAt.IsZero() && s.Mid <= 0 {
		return "No stored order book at that time."
	}
	name := prettyBase(s.Symbol)
	when := s.SampledAt.UTC().Format("15:04:05")
	return fmt.Sprintf("%s %s at %s: mid %s, spread %s (%s%%), bid %s vs ask %s (%s), %d bid wall(s) / %d ask wall(s).",
		titleExchange(s.Exchange), name, when,
		formatQty(s.Mid), formatQty(s.Spread), formatQty(s.SpreadPct),
		formatQty(s.BidNotional), formatQty(s.AskNotional), s.Pressure,
		s.BidWalls, s.AskWalls)
}

// ExplainBookDiff writes a short before/after read.
func ExplainBookDiff(d BookHistoryDiff) string {
	if d.From.SampledAt.IsZero() || d.To.SampledAt.IsZero() {
		return "Need two stored books to compare."
	}
	name := prettyBase(d.Symbol)
	head := fmt.Sprintf("%s from %s to %s: mid %s (%s%%), bid liquidity %s, ask liquidity %s.",
		name, d.From.SampledAt.UTC().Format("15:04:05"), d.To.SampledAt.UTC().Format("15:04:05"),
		formatSignedQty(d.MidDelta), formatQty(d.MidDeltaPct),
		formatSignedQty(d.BidNotionalDelta), formatSignedQty(d.AskNotionalDelta))
	if len(d.Gained) > 0 {
		g := d.Gained[0]
		head += fmt.Sprintf(" Largest gain: %s %s %+s.", g.Side, formatQty(g.Price), formatQty(g.Delta))
	}
	if len(d.Lost) > 0 {
		l := d.Lost[0]
		head += fmt.Sprintf(" Largest loss: %s %s %s.", l.Side, formatQty(l.Price), formatSignedQty(l.Delta))
	}
	if n := len(d.WallsAdded); n > 0 {
		head += fmt.Sprintf(" %d wall(s) appeared.", n)
	}
	if n := len(d.WallsRemoved); n > 0 {
		head += fmt.Sprintf(" %d wall(s) pulled.", n)
	}
	return head
}

// StripBookLevels drops ladders for compact list rows.
func StripBookLevels(s BookHistorySnapshot) BookHistorySnapshot {
	s.Bids = nil
	s.Asks = nil
	s.Walls = nil
	return s
}

const bookHistoryDisclaimer = "Stored books are 1-minute samples of the live spot ladder (top grouped levels, ±2% liquidity and walls), not a full 24h tape. Compare uses the nearest sample to each time. Informational only — not financial advice."

func bookLevelsFromGrouped(in []OrderBookLevel) []BookHistoryLevel {
	out := make([]BookHistoryLevel, 0, len(in))
	for _, lv := range in {
		p := parseHistFloat(lv.Price)
		q := parseHistFloat(lv.Quantity)
		n := parseHistFloat(lv.Notional)
		if p <= 0 || q <= 0 {
			continue
		}
		if n <= 0 {
			n = p * q
		}
		out = append(out, BookHistoryLevel{Price: p, Quantity: q, Notional: n, Wall: lv.IsWall})
	}
	return out
}

func bookWallsFromAnalysis(in []OrderBookWall) []BookHistoryWall {
	out := make([]BookHistoryWall, 0, len(in))
	for _, w := range in {
		p := parseHistFloat(w.Price)
		if p <= 0 {
			continue
		}
		out = append(out, BookHistoryWall{
			Side:  strings.ToLower(strings.TrimSpace(w.Side)),
			Price: p, Quantity: parseHistFloat(w.Quantity),
			Notional:    parseHistFloat(w.Notional),
			DistancePct: parseHistFloat(w.DistancePct), Share: w.Share,
		})
	}
	return out
}

func bookSideNotional(levels []BookHistoryLevel) float64 {
	var n float64
	for _, lv := range levels {
		n += lv.Notional
	}
	return n
}

func bookMid(raw RawOrderBook) float64 {
	if len(raw.Bids) == 0 || len(raw.Asks) == 0 {
		return 0
	}
	return (raw.Bids[0].Price + raw.Asks[0].Price) / 2
}

func parseHistFloat(s string) float64 {
	s = strings.TrimSpace(strings.TrimPrefix(s, "+"))
	if s == "" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return v
}

func bookLevelKey(side string, price, step float64) string {
	if step <= 0 {
		step = 1e-8
	}
	bucket := math.Round(price/step) * step
	return side + ":" + strconv.FormatFloat(bucket, 'f', 10, 64)
}

func parseBookLevelKey(k string) (side string, price float64) {
	i := strings.IndexByte(k, ':')
	if i < 0 {
		return "", 0
	}
	side = k[:i]
	price, _ = strconv.ParseFloat(k[i+1:], 64)
	return side, price
}

func wallKeySet(walls []BookHistoryWall, step float64) map[string]BookHistoryWall {
	out := map[string]BookHistoryWall{}
	for _, w := range walls {
		if w.Price <= 0 {
			continue
		}
		out[bookLevelKey(w.Side, w.Price, step)] = w
	}
	return out
}

func markSlack(s BookHistorySnapshot, at time.Time) BookHistorySnapshot {
	d := s.SampledAt.UTC().Sub(at.UTC())
	if d < 0 {
		d = -d
	}
	s.SlackSeconds = d.Seconds()
	s.Complete = true
	return s
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func formatSignedQty(v float64) string {
	return FormatSignedQty(v)
}

func titleExchange(ex Exchange) string {
	s := string(ex)
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
