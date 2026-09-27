package domain

import (
	"context"
	"fmt"
	"math"
	"sort"
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

// PerpDepthHistoryStore persists market-wide perp book samples.
type PerpDepthHistoryStore interface {
	InsertSamples(ctx context.Context, rows []PerpDepthSample) (inserted int, err error)
	ListSamples(ctx context.Context, exchange string, rangePct float64, from, to time.Time) ([]PerpDepthSample, error)
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
		Window:  id,
		BidPct:  live.BidPct,
		AskPct:  live.AskPct,
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
