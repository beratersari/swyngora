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
	PerpDepthShareBidGained = "bid_gained"
	PerpDepthShareAskGained = "ask_gained"
	PerpDepthShareStable    = "stable"

	PerpDepthSizeThicker = "thicker"
	PerpDepthSizeThinner = "thinner"
	PerpDepthSizeSteady  = "steady"

	perpDepthShareMovePP      = 1.0
	perpDepthSizeMovePct      = 5.0
	perpDepthShareMaxDrivers  = 5
	PerpDepthHistoryListLimit = 8000
	perpDepthSampleBucket     = time.Minute

	// PerpDepthTotalChartSpan is how far back the total-book line is drawn.
	PerpDepthTotalChartSpan = 6 * time.Hour

	PerpDepthMoveJump = "jump"
	PerpDepthMoveDrop = "drop"

	PerpDepthPriceUp   = "up"
	PerpDepthPriceDown = "down"
	PerpDepthPriceFlat = "flat"

	// How much of a book move sits in the coins that moved with it.
	PerpDepthMoveSpanOne   = "one"
	PerpDepthMoveSpanFew   = "few"
	PerpDepthMoveSpanBroad = "broad"

	perpDepthMoveMaxSpan    = 15 * time.Minute
	perpDepthMoveAfter      = 15 * time.Minute
	perpDepthMoveAfterSlack = 5 * time.Minute
	perpDepthMoveMinPct     = 5.0
	perpDepthMoveKeep       = 3
	perpDepthPriceFlatPct   = 0.2
	perpDepthMoveOneShare   = 50.0
	perpDepthMoveFewShare   = 60.0
)

// PerpDepthShareWindows is 15m / 1h / 4h of market bid vs ask mix.
var PerpDepthShareWindows = ParticipationWindows

// PerpDepthSample is one coin's bid/ask at a sampled minute.
type PerpDepthSample struct {
	Time     time.Time
	Exchange string
	RangePct float64
	Symbol   string
	Base     string
	Bid      float64
	Ask      float64
}

// PerpDepthTotalPoint is the summed bid+ask of the tracked book at one minute,
// plus BTC's close so the chart can show price next to that size.
type PerpDepthTotalPoint struct {
	Time  time.Time
	Total float64
	Price float64
}

// PerpDepthBookMove is a sharp rise or fall in total bid+ask, with BTC around it.
type PerpDepthBookMove struct {
	Kind           string // jump | drop
	From           time.Time
	To             time.Time
	PastTotal      float64
	Total          float64
	Change         float64
	ChangePct      float64
	PriceFrom      float64
	PriceTo        float64
	PriceAfter     float64
	PriceChangePct float64
	PriceAfterPct  float64
	PriceKind      string // up | down | flat
	AfterKind      string
	Drivers        []PerpDepthMoveDriver
	Span           string // one | few | broad
	Summary        string
}

// PerpDepthMoveDriver is one coin's share of a book jump or drop.
type PerpDepthMoveDriver struct {
	Symbol   string
	Base     string
	Change   float64 // USDT, end − start
	SharePct float64 // this change as a percent of the market move
	Kind     string  // thicker | thinner | steady
}

// PerpDepthHistoryStore persists market-wide perp book samples.
type PerpDepthHistoryStore interface {
	InsertSamples(ctx context.Context, rows []PerpDepthSample) (inserted int, err error)
	ListSamples(ctx context.Context, exchange string, rangePct float64, from, to time.Time) ([]PerpDepthSample, error)
	// ListMinuteTotals returns one summed bid+ask per minute, oldest first.
	ListMinuteTotals(ctx context.Context, exchange string, rangePct float64, from, to time.Time) ([]PerpDepthTotalPoint, error)
	PurgeOlderThan(ctx context.Context, cutoff time.Time) (int, error)
}

// PerpDepthShareDriver is one coin's contribution to the market bid-share change.
type PerpDepthShareDriver struct {
	Symbol    string
	Base      string
	ChangePct float64 // percentage points of market bid share
	Kind      string  // bid_gained | ask_gained | stable
}

// PerpDepthSizeDriver is one coin's signed change in bid+ask notional.
type PerpDepthSizeDriver struct {
	Symbol string
	Base   string
	Change float64 // USDT, now − past
	Kind   string  // thicker | thinner | steady
}

// PerpDepthShareWindow is live bid/ask mix and total book size vs that lookback.
type PerpDepthShareWindow struct {
	Window         string
	BidPct         float64
	AskPct         float64
	PastBidPct     float64
	PastAskPct     float64
	ChangePct      float64 // bidPct now − past, percentage points
	Kind           string
	Total          float64 // live bid+ask
	PastTotal      float64
	TotalChange    float64 // now − past, USDT
	TotalChangePct float64 // percent change of total
	DepthKind      string  // thicker | thinner | steady
	Drivers        []PerpDepthShareDriver
	SizeDrivers    []PerpDepthSizeDriver
	Complete       bool
	SampleTime     time.Time
	Summary        string
}

// PerpBookBidPct is bid as a percent of bid+ask.
func PerpBookBidPct(bid, ask float64) float64 {
	tot := bid + ask
	if tot <= 0 {
		return 0
	}
	return round4(100 * bid / tot)
}

// ClassifyPerpDepthShare labels a bid-share percentage-point change.
func ClassifyPerpDepthShare(changePct float64) string {
	switch {
	case changePct >= perpDepthShareMovePP:
		return PerpDepthShareBidGained
	case changePct <= -perpDepthShareMovePP:
		return PerpDepthShareAskGained
	default:
		return PerpDepthShareStable
	}
}

// ClassifyPerpDepthSize labels a percent change in total bid+ask.
// A move of at least 5% is thicker or thinner; smaller moves stay steady.
func ClassifyPerpDepthSize(changePct float64) string {
	switch {
	case changePct >= perpDepthSizeMovePct:
		return PerpDepthSizeThicker
	case changePct <= -perpDepthSizeMovePct:
		return PerpDepthSizeThinner
	default:
		return PerpDepthSizeSteady
	}
}

// BucketPerpDepthSampleTime truncates to the one-minute sample grid.
func BucketPerpDepthSampleTime(t time.Time) time.Time {
	if t.IsZero() {
		return t
	}
	return t.UTC().Truncate(perpDepthSampleBucket)
}

// AttachPerpDepthWindows fills 15m / 1h / 4h bid-share vs now from stored samples.
func AttachPerpDepthWindows(live *PerpDepthOverview, samples []PerpDepthSample) {
	if live == nil {
		return
	}
	live.Windows = make([]PerpDepthShareWindow, 0, len(PerpDepthShareWindows))
	groups := groupPerpDepthSamples(samples)
	now := live.AsOf
	if now.IsZero() {
		now = time.Now().UTC()
	}
	for _, spec := range PerpDepthShareWindows {
		live.Windows = append(live.Windows, buildPerpDepthShareWindow(spec.ID, spec.Dur, live, groups, now))
	}
	live.Summary = ExplainPerpDepthOverview(*live)
}

// AttachPerpDepthHistory keeps summed bid+ask minutes inside the chart span.
func AttachPerpDepthHistory(live *PerpDepthOverview, points []PerpDepthTotalPoint) {
	if live == nil {
		return
	}
	asOf := live.AsOf
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	from := asOf.Add(-PerpDepthTotalChartSpan)
	out := make([]PerpDepthTotalPoint, 0, len(points))
	for _, p := range points {
		if p.Time.IsZero() || p.Total <= 0 {
			continue
		}
		t := BucketPerpDepthSampleTime(p.Time)
		if t.Before(from) || t.After(asOf.Add(time.Minute)) {
			continue
		}
		out = append(out, PerpDepthTotalPoint{Time: t, Total: round4(p.Total)})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	live.History = out
	AttachPerpDepthMoves(live)
}

// AttachPerpDepthPrices fills each history minute with the BTC close from that minute.
func AttachPerpDepthPrices(live *PerpDepthOverview, candles []Candle) {
	if live == nil || len(live.History) == 0 || len(candles) == 0 {
		return
	}
	by := map[int64]float64{}
	for _, c := range candles {
		t := c.OpenTime
		if t.IsZero() {
			t = c.CloseTime
		}
		px, err := strconv.ParseFloat(strings.TrimSpace(c.Close), 64)
		if err != nil || px <= 0 || t.IsZero() {
			continue
		}
		by[BucketPerpDepthSampleTime(t).UnixMilli()] = px
	}
	for i := range live.History {
		px, ok := by[live.History[i].Time.UnixMilli()]
		if !ok {
			continue
		}
		live.History[i].Price = round4(px)
	}
	AttachPerpDepthMoves(live)
}

type perpDepthMoveCand struct {
	from, to int
	change   float64
	pct      float64
}

// AttachPerpDepthMoves ranks the largest jumps and drops that finish within 15 minutes.
// A move must change total bid+ask by at least 5%. Up to three of each are kept, and
// they do not overlap. BTC is the close at the start, at the end, and 15 minutes later.
func AttachPerpDepthMoves(live *PerpDepthOverview) {
	if live == nil {
		return
	}
	pts := live.History
	var jumps, drops []perpDepthMoveCand
	for i := 0; i < len(pts); i++ {
		if pts[i].Total <= 0 {
			continue
		}
		for j := i + 1; j < len(pts); j++ {
			dt := pts[j].Time.Sub(pts[i].Time)
			if dt < time.Minute {
				continue
			}
			if dt > perpDepthMoveMaxSpan {
				break
			}
			chg := pts[j].Total - pts[i].Total
			pct := 100 * chg / pts[i].Total
			switch {
			case pct >= perpDepthMoveMinPct:
				jumps = append(jumps, perpDepthMoveCand{i, j, chg, pct})
			case pct <= -perpDepthMoveMinPct:
				drops = append(drops, perpDepthMoveCand{i, j, chg, pct})
			}
		}
	}
	picked := append(pickPerpDepthMoves(jumps), pickPerpDepthMoves(drops)...)
	out := make([]PerpDepthBookMove, 0, len(picked))
	for _, c := range picked {
		out = append(out, buildPerpDepthBookMove(pts, c))
	}
	sort.SliceStable(out, func(i, j int) bool {
		ai, aj := math.Abs(out[i].Change), math.Abs(out[j].Change)
		if ai != aj {
			return ai > aj
		}
		return out[i].From.Before(out[j].From)
	})
	live.Moves = out
}

func pickPerpDepthMoves(cands []perpDepthMoveCand) []perpDepthMoveCand {
	sort.SliceStable(cands, func(i, j int) bool {
		ai, aj := math.Abs(cands[i].change), math.Abs(cands[j].change)
		if ai != aj {
			return ai > aj
		}
		return cands[i].from < cands[j].from
	})
	picked := make([]perpDepthMoveCand, 0, perpDepthMoveKeep)
	for _, c := range cands {
		over := false
		for _, p := range picked {
			if c.from < p.to && p.from < c.to {
				over = true
				break
			}
		}
		if over {
			continue
		}
		picked = append(picked, c)
		if len(picked) == perpDepthMoveKeep {
			break
		}
	}
	return picked
}

func buildPerpDepthBookMove(pts []PerpDepthTotalPoint, c perpDepthMoveCand) PerpDepthBookMove {
	a, b := pts[c.from], pts[c.to]
	kind := PerpDepthMoveJump
	if c.change < 0 {
		kind = PerpDepthMoveDrop
	}
	m := PerpDepthBookMove{
		Kind: kind, From: a.Time, To: b.Time,
		PastTotal: round4(a.Total), Total: round4(b.Total),
		Change: round4(c.change), ChangePct: round4(c.pct),
		Drivers: []PerpDepthMoveDriver{},
	}
	if a.Price > 0 && b.Price > 0 {
		m.PriceFrom = round4(a.Price)
		m.PriceTo = round4(b.Price)
		m.PriceChangePct = round4(100 * (b.Price - a.Price) / a.Price)
		m.PriceKind = classifyPerpDepthPrice(m.PriceChangePct)
	}
	if after, ok := perpDepthPriceAfter(pts, b.Time); ok {
		m.PriceAfter = round4(after.Price)
		if b.Price > 0 {
			m.PriceAfterPct = round4(100 * (after.Price - b.Price) / b.Price)
			m.AfterKind = classifyPerpDepthPrice(m.PriceAfterPct)
		}
	}
	m.Summary = explainPerpDepthBookMove(m)
	return m
}

func perpDepthPriceAfter(pts []PerpDepthTotalPoint, end time.Time) (PerpDepthTotalPoint, bool) {
	want := end.Add(perpDepthMoveAfter)
	late := want.Add(perpDepthMoveAfterSlack)
	for _, p := range pts {
		if p.Price <= 0 || p.Time.Before(want) {
			continue
		}
		if p.Time.After(late) {
			break
		}
		return p, true
	}
	return PerpDepthTotalPoint{}, false
}

func classifyPerpDepthPrice(pct float64) string {
	switch {
	case pct >= perpDepthPriceFlatPct:
		return PerpDepthPriceUp
	case pct <= -perpDepthPriceFlatPct:
		return PerpDepthPriceDown
	default:
		return PerpDepthPriceFlat
	}
}

func explainPerpDepthBookMove(m PerpDepthBookMove) string {
	verb := "jumped"
	if m.Kind == PerpDepthMoveDrop {
		verb = "dropped"
	}
	s := fmt.Sprintf("Book %s from %s to %s (%s) between %s and %s.",
		verb, formatQty(m.PastTotal), formatQty(m.Total), FormatSignedPct(m.ChangePct),
		m.From.UTC().Format("15:04"), m.To.UTC().Format("15:04"))
	if m.PriceFrom > 0 && m.PriceTo > 0 {
		s += fmt.Sprintf(" BTC %s → %s (%s).", formatQty(m.PriceFrom), formatQty(m.PriceTo), FormatSignedPct(m.PriceChangePct))
	}
	if m.PriceAfter > 0 {
		s += fmt.Sprintf(" 15m later BTC %s (%s).", formatQty(m.PriceAfter), FormatSignedPct(m.PriceAfterPct))
	}
	if coins := perpDepthMoveCoinSentence(m); coins != "" {
		s += " " + coins
	}
	return s
}

// AttachPerpDepthMoveCoins names the coins behind each jump or drop.
// One coin at half the move or more is span one. Two coins at 60% or more
// is span few. Anything wider is spread across the book.
func AttachPerpDepthMoveCoins(live *PerpDepthOverview, samples []PerpDepthSample) {
	if live == nil {
		return
	}
	groups := groupPerpDepthSamples(samples)
	for i := range live.Moves {
		m := &live.Moves[i]
		m.Drivers = []PerpDepthMoveDriver{}
		m.Span = ""
		if m.Change == 0 {
			m.Summary = explainPerpDepthBookMove(*m)
			continue
		}
		_, past, okPast := nearestPerpDepthGroup(groups, m.From, time.Minute)
		_, now, okNow := nearestPerpDepthGroup(groups, m.To, time.Minute)
		if !okPast || !okNow {
			m.Summary = explainPerpDepthBookMove(*m)
			continue
		}
		sized := perpDepthSizeDrivers(perpDepthCoinsAsShares(now), past, m.Change)
		drivers := make([]PerpDepthMoveDriver, 0, len(sized))
		for _, d := range sized {
			drivers = append(drivers, PerpDepthMoveDriver{
				Symbol: d.Symbol, Base: d.Base, Change: d.Change, Kind: d.Kind,
				SharePct: round4(100 * d.Change / m.Change),
			})
		}
		m.Drivers = drivers
		m.Span = classifyPerpDepthMoveSpan(drivers, m.Change)
		m.Summary = explainPerpDepthBookMove(*m)
	}
}

func perpDepthCoinsAsShares(in []PerpDepthCoinInput) []PerpDepthCoinShare {
	out := make([]PerpDepthCoinShare, 0, len(in))
	for _, c := range in {
		out = append(out, PerpDepthCoinShare{Symbol: c.Symbol, Base: c.Base, Bid: c.Bid, Ask: c.Ask})
	}
	return out
}

func classifyPerpDepthMoveSpan(drivers []PerpDepthMoveDriver, marketChange float64) string {
	same := make([]float64, 0, 2)
	for _, d := range drivers {
		if marketChange > 0 && d.Change <= 0 {
			continue
		}
		if marketChange < 0 && d.Change >= 0 {
			continue
		}
		same = append(same, d.SharePct)
		if len(same) == 2 {
			break
		}
	}
	if len(same) == 0 {
		return PerpDepthMoveSpanBroad
	}
	if same[0] >= perpDepthMoveOneShare {
		return PerpDepthMoveSpanOne
	}
	two := same[0]
	if len(same) > 1 {
		two += same[1]
	}
	if two >= perpDepthMoveFewShare {
		return PerpDepthMoveSpanFew
	}
	return PerpDepthMoveSpanBroad
}

func perpDepthMoveCoinSentence(m PerpDepthBookMove) string {
	names := make([]string, 0, 2)
	for _, d := range m.Drivers {
		if len(names) >= 2 {
			break
		}
		if m.Change > 0 && d.Change <= 0 {
			continue
		}
		if m.Change < 0 && d.Change >= 0 {
			continue
		}
		if d.Base == "" {
			continue
		}
		names = append(names, d.Base)
	}
	switch m.Span {
	case PerpDepthMoveSpanOne:
		if len(names) == 0 {
			return ""
		}
		return names[0] + " caused most of it."
	case PerpDepthMoveSpanFew:
		if len(names) == 0 {
			return ""
		}
		return joinList(names) + " caused most of it."
	case PerpDepthMoveSpanBroad:
		return "Spread across the book."
	default:
		return ""
	}
}

func groupPerpDepthSamples(rows []PerpDepthSample) map[int64][]PerpDepthCoinInput {
	out := map[int64][]PerpDepthCoinInput{}
	for _, r := range rows {
		if r.Symbol == "" {
			continue
		}
		ms := BucketPerpDepthSampleTime(r.Time).UnixMilli()
		out[ms] = append(out[ms], PerpDepthCoinInput{
			Symbol: r.Symbol, Base: r.Base, Bid: r.Bid, Ask: r.Ask,
		})
	}
	return out
}

func buildPerpDepthShareWindow(id string, dur time.Duration, live *PerpDepthOverview, groups map[int64][]PerpDepthCoinInput, now time.Time) PerpDepthShareWindow {
	out := PerpDepthShareWindow{
		Window:      id,
		BidPct:      live.BidPct,
		AskPct:      live.AskPct,
		Kind:        PerpDepthShareStable,
		Drivers:     []PerpDepthShareDriver{},
		SizeDrivers: []PerpDepthSizeDriver{},
	}
	target := now.Add(-dur)
	slack := perpDepthShareSlack(dur)
	pastTime, pastCoins, ok := nearestPerpDepthGroup(groups, target, slack)
	if !ok || live.Total <= 0 {
		out.Summary = fmt.Sprintf("%s: not enough book history yet.", id)
		return out
	}
	var pastBid, pastAsk float64
	for _, c := range pastCoins {
		pastBid += c.Bid
		pastAsk += c.Ask
	}
	pastTotal := pastBid + pastAsk
	if pastTotal <= 0 {
		out.Summary = fmt.Sprintf("%s: not enough book history yet.", id)
		return out
	}
	out.Complete = true
	out.SampleTime = pastTime
	out.PastBidPct = PerpBookBidPct(pastBid, pastAsk)
	out.PastAskPct = round4(100 - out.PastBidPct)
	out.ChangePct = round4(out.BidPct - out.PastBidPct)
	out.Kind = ClassifyPerpDepthShare(out.ChangePct)
	out.Total = live.Total
	out.PastTotal = pastTotal
	out.TotalChange = round4(live.Total - pastTotal)
	out.TotalChangePct = round4(100 * out.TotalChange / pastTotal)
	out.DepthKind = ClassifyPerpDepthSize(out.TotalChangePct)
	out.Drivers = perpDepthShareDrivers(live.Coins, pastCoins, live.Total, pastTotal, out.ChangePct)
	out.SizeDrivers = perpDepthSizeDrivers(live.Coins, pastCoins, out.TotalChange)
	out.Summary = ExplainPerpDepthShareWindow(out)
	return out
}

func perpDepthShareSlack(dur time.Duration) time.Duration {
	slack := dur / 10
	if slack < 3*time.Minute {
		slack = 3 * time.Minute
	}
	if slack > 15*time.Minute {
		slack = 15 * time.Minute
	}
	return slack
}

func nearestPerpDepthGroup(groups map[int64][]PerpDepthCoinInput, at time.Time, slack time.Duration) (time.Time, []PerpDepthCoinInput, bool) {
	if len(groups) == 0 || at.IsZero() {
		return time.Time{}, nil, false
	}
	want := at.UTC().UnixMilli()
	maxDelta := slack.Milliseconds()
	var best int64
	bestDelta := int64(math.MaxInt64)
	for ms := range groups {
		d := ms - want
		if d < 0 {
			d = -d
		}
		if d <= maxDelta && d < bestDelta {
			best, bestDelta = ms, d
		}
	}
	if bestDelta == int64(math.MaxInt64) {
		return time.Time{}, nil, false
	}
	return time.UnixMilli(best).UTC(), groups[best], true
}

func perpDepthShareDrivers(now []PerpDepthCoinShare, past []PerpDepthCoinInput, nowTotal, pastTotal, marketDelta float64) []PerpDepthShareDriver {
	if nowTotal <= 0 || pastTotal <= 0 {
		return []PerpDepthShareDriver{}
	}
	pastBy := map[string]PerpDepthCoinInput{}
	for _, c := range past {
		pastBy[NormalizeLiquidationSymbol(c.Symbol)] = c
	}
	seen := map[string]struct{}{}
	out := make([]PerpDepthShareDriver, 0, len(now)+len(past))
	add := func(symbol, base string, bidNow, bidPast float64) {
		symbol = NormalizeLiquidationSymbol(symbol)
		if symbol == "" {
			return
		}
		if _, ok := seen[symbol]; ok {
			return
		}
		seen[symbol] = struct{}{}
		if base == "" {
			base = prettyBase(symbol)
		}
		chg := round4(100 * (bidNow/nowTotal - bidPast/pastTotal))
		out = append(out, PerpDepthShareDriver{
			Symbol: symbol, Base: base, ChangePct: chg, Kind: ClassifyPerpDepthShare(chg),
		})
	}
	for _, c := range now {
		p := pastBy[c.Symbol]
		add(c.Symbol, c.Base, c.Bid, p.Bid)
	}
	for _, p := range past {
		sym := NormalizeLiquidationSymbol(p.Symbol)
		if _, ok := seen[sym]; ok {
			continue
		}
		add(p.Symbol, p.Base, 0, p.Bid)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if marketDelta >= perpDepthShareMovePP {
			if out[i].ChangePct != out[j].ChangePct {
				return out[i].ChangePct > out[j].ChangePct
			}
		} else if marketDelta <= -perpDepthShareMovePP {
			if out[i].ChangePct != out[j].ChangePct {
				return out[i].ChangePct < out[j].ChangePct
			}
		} else if math.Abs(out[i].ChangePct) != math.Abs(out[j].ChangePct) {
			return math.Abs(out[i].ChangePct) > math.Abs(out[j].ChangePct)
		}
		return out[i].Symbol < out[j].Symbol
	})
	if len(out) > perpDepthShareMaxDrivers {
		out = out[:perpDepthShareMaxDrivers]
	}
	return out
}

func perpDepthSizeDrivers(now []PerpDepthCoinShare, past []PerpDepthCoinInput, marketChange float64) []PerpDepthSizeDriver {
	pastBy := map[string]PerpDepthCoinInput{}
	for _, c := range past {
		pastBy[NormalizeLiquidationSymbol(c.Symbol)] = c
	}
	seen := map[string]struct{}{}
	out := make([]PerpDepthSizeDriver, 0, len(now)+len(past))
	add := func(symbol, base string, nowTotal, pastTotal float64) {
		symbol = NormalizeLiquidationSymbol(symbol)
		if symbol == "" {
			return
		}
		if _, ok := seen[symbol]; ok {
			return
		}
		seen[symbol] = struct{}{}
		if base == "" {
			base = prettyBase(symbol)
		}
		chg := round4(nowTotal - pastTotal)
		if chg == 0 {
			return
		}
		out = append(out, PerpDepthSizeDriver{
			Symbol: symbol, Base: base, Change: chg, Kind: perpDepthSizeKind(chg),
		})
	}
	for _, c := range now {
		p := pastBy[c.Symbol]
		add(c.Symbol, c.Base, c.Bid+c.Ask, p.Bid+p.Ask)
	}
	for _, p := range past {
		sym := NormalizeLiquidationSymbol(p.Symbol)
		if _, ok := seen[sym]; ok {
			continue
		}
		add(p.Symbol, p.Base, 0, p.Bid+p.Ask)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if marketChange > 0 {
			if out[i].Change != out[j].Change {
				return out[i].Change > out[j].Change
			}
		} else if marketChange < 0 {
			if out[i].Change != out[j].Change {
				return out[i].Change < out[j].Change
			}
		} else if math.Abs(out[i].Change) != math.Abs(out[j].Change) {
			return math.Abs(out[i].Change) > math.Abs(out[j].Change)
		}
		return out[i].Symbol < out[j].Symbol
	})
	if len(out) > perpDepthShareMaxDrivers {
		out = out[:perpDepthShareMaxDrivers]
	}
	return out
}

func perpDepthSizeKind(change float64) string {
	switch {
	case change > 0:
		return PerpDepthSizeThicker
	case change < 0:
		return PerpDepthSizeThinner
	default:
		return PerpDepthSizeSteady
	}
}

// ExplainPerpDepthShareWindow writes bid share then → now plus who pushed it.
func ExplainPerpDepthShareWindow(w PerpDepthShareWindow) string {
	if !w.Complete {
		return fmt.Sprintf("%s: not enough book history yet.", w.Window)
	}
	head := fmt.Sprintf("%s: bids were %s%% of the book, now %s%% (%s). Book was %s, now %s (%s) — %s.",
		w.Window, formatQty(w.PastBidPct), formatQty(w.BidPct), FormatSignedPct(w.ChangePct),
		formatQty(w.PastTotal), formatQty(w.Total), FormatSignedPct(w.TotalChangePct), perpDepthSizeWord(w.DepthKind))
	names := make([]string, 0, 3)
	for _, d := range w.Drivers {
		if len(names) >= 3 {
			break
		}
		if w.Kind == PerpDepthShareBidGained && d.ChangePct <= 0 {
			continue
		}
		if w.Kind == PerpDepthShareAskGained && d.ChangePct >= 0 {
			continue
		}
		if w.Kind == PerpDepthShareStable && math.Abs(d.ChangePct) < perpDepthShareMovePP {
			continue
		}
		names = append(names, d.Base)
	}
	if len(names) == 0 {
		return head
	}
	switch w.Kind {
	case PerpDepthShareBidGained:
		head += " " + joinList(names) + " pushed bid share up the most."
	case PerpDepthShareAskGained:
		head += " " + joinList(names) + " pushed ask share up the most."
	default:
		head += " " + joinList(names) + " moved the mix the most."
	}
	if size := perpDepthSizeDriverSentence(w); size != "" {
		head += " " + size
	}
	return head
}

func perpDepthSizeDriverSentence(w PerpDepthShareWindow) string {
	names := make([]string, 0, 3)
	for _, d := range w.SizeDrivers {
		if len(names) >= 3 {
			break
		}
		if w.DepthKind == PerpDepthSizeThicker && d.Change <= 0 {
			continue
		}
		if w.DepthKind == PerpDepthSizeThinner && d.Change >= 0 {
			continue
		}
		if d.Base == "" {
			continue
		}
		names = append(names, d.Base)
	}
	if len(names) == 0 {
		return ""
	}
	switch w.DepthKind {
	case PerpDepthSizeThicker:
		return joinList(names) + " added the most size."
	case PerpDepthSizeThinner:
		return joinList(names) + " removed the most size."
	default:
		return joinList(names) + " moved the book size the most."
	}
}

func perpDepthSizeWord(kind string) string {
	switch kind {
	case PerpDepthSizeThicker:
		return "thicker"
	case PerpDepthSizeThinner:
		return "thinner"
	default:
		return "about the same"
	}
}

// PerpDepthSamplesFromOverview is the minute-bucket rows to persist.
func PerpDepthSamplesFromOverview(o *PerpDepthOverview) []PerpDepthSample {
	if o == nil || o.AsOf.IsZero() {
		return nil
	}
	at := BucketPerpDepthSampleTime(o.AsOf)
	out := make([]PerpDepthSample, 0, len(o.Coins))
	for _, c := range o.Coins {
		if c.Symbol == "" || c.Error != "" || c.Total <= 0 {
			continue
		}
		out = append(out, PerpDepthSample{
			Time: at, Exchange: o.Exchange, RangePct: o.RangePct,
			Symbol: c.Symbol, Base: c.Base, Bid: c.Bid, Ask: c.Ask,
		})
	}
	return out
}
