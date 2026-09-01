package domain

import (
	"fmt"
	"math"
	"sort"
	"time"
)

const (
	CascadeWindow1m  = "1m"
	CascadeWindow5m  = "5m"
	CascadeWindow15m = "15m"

	CascadeGradeQuiet     = "quiet"
	CascadeGradeElevated  = "elevated"
	CascadeGradeCascade   = "cascade"
	CascadeGradeExtreme   = "extreme"

	CascadeSideNone = "none"
	CascadeSideBoth = "both"

	cascadeLookback         = 6 * time.Hour
	cascadeEpisodeLookback  = 24 * time.Hour
	cascadeStep             = time.Minute
	cascadeMinTypical1m     = 500.0
	cascadeEpisodeMinPrior  = 20
	cascadeEpisodeQuietGap  = 2
	DefaultCascadeMin       = CascadeGradeElevated
	CascadeScanMaxHits      = 40
	CascadeEpisodeMax       = 30
	CascadeExchangeBoth     = "both"
)

// CascadeWindows are short bursts compared to that coin/venue's own typical.
var CascadeWindows = []struct {
	ID       string
	Dur      time.Duration
	MinPrior int
	MinCount int
}{
	{CascadeWindow1m, time.Minute, 20, 2},
	{CascadeWindow5m, 5 * time.Minute, 8, 3},
	{CascadeWindow15m, 15 * time.Minute, 6, 4},
}

// CascadeWindowRead is current vs typical liquidation notional for one burst size.
type CascadeWindowRead struct {
	Window        string
	LongNotional  string
	ShortNotional string
	TotalNotional string
	LongTypical   string
	ShortTypical  string
	LongRatio     float64
	ShortRatio    float64
	MaxRatio      float64
	Side          string // long | short | both | none
	Grade         string
	Count         int
	SampleBuckets int
	Complete      bool
}

// CascadeVenue is one venue's cascade read for a coin or the whole market.
type CascadeVenue struct {
	Exchange  Exchange
	Symbol    string
	Windows   []CascadeWindowRead
	Side      string
	Grade     string
	Score     float64
	Hottest   string
	StartedAt time.Time
	Summary   string
}

// CascadeBoth is true when Binance and Bybit fire the same-side burst together.
type CascadeBoth struct {
	Agree   bool
	Side    string
	Grade   string
	Score   float64
	Hottest string
	Summary string
}

// CascadeEpisode is one completed or still-open liquidation wave.
type CascadeEpisode struct {
	Symbol         string
	Exchange       string // binance | bybit | both
	Combined       bool
	Side           string
	Grade          string // peak grade in the wave
	Score          float64
	StartedAt      time.Time
	EndedAt        time.Time // zero while open
	Open           bool
	DurationSec    int64
	LongNotional   string
	ShortNotional  string
	TotalNotional  string
	Count          int
	PeakRatio      float64
	PriceOpen      string
	PriceClose     string
	PriceHigh      string
	PriceLow       string
	PriceChangePct string
	Summary        string
}

// CascadeReport is one coin (or symbol=all for the pooled market).
type CascadeReport struct {
	Symbol   string
	Exchange string
	AsOf     time.Time
	Venues   []CascadeVenue
	Both     *CascadeBoth
	Episodes []CascadeEpisode
	Summary  string
	Note     string
}

// CascadeHit is one coin in a market scan.
type CascadeHit struct {
	Symbol  string
	Side    string
	Grade   string
	Score   float64
	Hottest string
	Both    bool
	Summary string
}

// CascadeScan is market-wide risk plus coins currently bursting.
type CascadeScan struct {
	Exchange string
	AsOf     time.Time
	Market   CascadeReport
	Hits     []CascadeHit
	Summary  string
	Note     string
}

// DetectCascadeVenue compares recent long/short bursts to that stream's typical rate.
func DetectCascadeVenue(ex Exchange, symbol string, events []LiquidationEvent, now time.Time) CascadeVenue {
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	out := CascadeVenue{Exchange: ex, Symbol: symbol, Windows: []CascadeWindowRead{}}
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
	buckets := cascadeBuckets(filtered, now)
	var hottest CascadeWindowRead
	for _, spec := range CascadeWindows {
		w := measureCascadeWindow(buckets, spec.ID, spec.Dur, spec.MinPrior, spec.MinCount)
		out.Windows = append(out.Windows, w)
		if hottest.Window == "" || w.MaxRatio > hottest.MaxRatio || (w.MaxRatio == hottest.MaxRatio && cascadeGradeRank(w.Grade) > cascadeGradeRank(hottest.Grade)) {
			hottest = w
		}
	}
	out.Hottest = hottest.Window
	out.Side = hottest.Side
	out.Grade = hottest.Grade
	out.Score = cascadeScore(hottest)
	out.StartedAt = cascadeStart(filtered, hottest.Window, hottest.Side, now)
	out.Summary = explainCascadeVenue(out)
	return out
}

// DetectCascadeBoth flags the same-side burst on Binance and Bybit.
func DetectCascadeBoth(venues []CascadeVenue) *CascadeBoth {
	var bn, bb *CascadeVenue
	for i := range venues {
		switch venues[i].Exchange {
		case ExchangeBinance:
			bn = &venues[i]
		case ExchangeBybit:
			bb = &venues[i]
		}
	}
	if bn == nil || bb == nil {
		return nil
	}
	side, ok := cascadeSharedSide(bn.Side, bb.Side)
	grade := cascadeWeakerGrade(bn.Grade, bb.Grade)
	agree := ok && cascadeGradeRank(grade) >= cascadeGradeRank(CascadeGradeCascade)
	hottest := bn.Hottest
	if cascadeWindowRank(bb.Hottest) < cascadeWindowRank(hottest) {
		hottest = bb.Hottest
	}
	score := math.Min(bn.Score, bb.Score)
	out := &CascadeBoth{
		Agree:   agree,
		Side:    side,
		Grade:   grade,
		Score:   score,
		Hottest: hottest,
	}
	if agree {
		out.Summary = fmt.Sprintf("Same %s cascade on Binance and Bybit (%s, score %.0f).", side, grade, score)
	} else if ok && cascadeGradeRank(grade) >= cascadeGradeRank(CascadeGradeElevated) {
		out.Summary = fmt.Sprintf("Both venues elevated on the %s side, not yet a confirmed cascade.", side)
	} else if bn.Side != CascadeSideNone && bb.Side != CascadeSideNone && bn.Side != bb.Side && bn.Side != CascadeSideBoth && bb.Side != CascadeSideBoth {
		out.Side = CascadeSideNone
		out.Summary = "Venues are bursting on opposite sides."
	} else {
		out.Summary = "No shared cascade across Binance and Bybit."
	}
	return out
}

// BuildCascadeReport scores each requested venue and the both-venues flag.
func BuildCascadeReport(symbol, exchange string, events []LiquidationEvent, now time.Time) CascadeReport {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	ex := exchange
	if ex == "" {
		ex = "all"
	}
	want := []Exchange{ExchangeBinance, ExchangeBybit}
	if ex != "all" {
		want = []Exchange{Exchange(ex)}
	}
	out := CascadeReport{
		Symbol:   symbol,
		Exchange: ex,
		AsOf:     now.UTC(),
		Venues:   make([]CascadeVenue, 0, len(want)),
		Episodes: []CascadeEpisode{},
		Note:     cascadeNote,
	}
	for _, v := range want {
		out.Venues = append(out.Venues, DetectCascadeVenue(v, symbol, events, now))
	}
	sort.Slice(out.Venues, func(i, j int) bool {
		return string(out.Venues[i].Exchange) < string(out.Venues[j].Exchange)
	})
	if ex == "all" {
		out.Both = DetectCascadeBoth(out.Venues)
		bn := DetectCascadeEpisodes(ExchangeBinance, symbol, events, now)
		bb := DetectCascadeEpisodes(ExchangeBybit, symbol, events, now)
		out.Episodes = MergeCascadeEpisodes(bn, bb, now)
	} else {
		out.Episodes = DetectCascadeEpisodes(Exchange(ex), symbol, events, now)
	}
	out.Summary = explainCascadeReport(out)
	return out
}

// BuildCascadeScan scores the pooled market and ranks bursting coins.
func BuildCascadeScan(exchange string, events []LiquidationEvent, now time.Time) CascadeScan {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	ex := exchange
	if ex == "" {
		ex = "all"
	}
	out := CascadeScan{
		Exchange: ex,
		AsOf:     now.UTC(),
		Market:   BuildCascadeReport("all", ex, events, now),
		Hits:     []CascadeHit{},
		Note:     cascadeNote,
	}
	bySym := map[string][]LiquidationEvent{}
	for _, e := range events {
		sym := NormalizeLiquidationSymbol(e.Symbol)
		if sym == "" {
			continue
		}
		bySym[sym] = append(bySym[sym], e)
	}
	for sym, list := range bySym {
		rep := BuildCascadeReport(sym, ex, list, now)
		grade, score, side, hottest, both := cascadeReportPeak(rep)
		if cascadeGradeRank(grade) < cascadeGradeRank(CascadeGradeElevated) {
			continue
		}
		out.Hits = append(out.Hits, CascadeHit{
			Symbol:  sym,
			Side:    side,
			Grade:   grade,
			Score:   score,
			Hottest: hottest,
			Both:    both,
			Summary: rep.Summary,
		})
	}
	sort.Slice(out.Hits, func(i, j int) bool {
		if out.Hits[i].Score != out.Hits[j].Score {
			return out.Hits[i].Score > out.Hits[j].Score
		}
		if out.Hits[i].Both != out.Hits[j].Both {
			return out.Hits[i].Both
		}
		return out.Hits[i].Symbol < out.Hits[j].Symbol
	})
	if len(out.Hits) > CascadeScanMaxHits {
		out.Hits = out.Hits[:CascadeScanMaxHits]
	}
	out.Summary = explainCascadeScan(out)
	return out
}

const cascadeNote = "A cascade is a short burst of long or short liquidations far above that stream's own typical rate (1m / 5m / 15m vs the prior 6 hours). Episodes list each wave in the last 24h: start, duration, long/short notional, and price move. Binance and Bybit are scored separately; a combined episode is the same-side wave on both venues at once. Informational only."

type cascadeBucket struct {
	start time.Time
	long  float64
	short float64
	count int
}

func cascadeBuckets(events []LiquidationEvent, now time.Time) []cascadeBucket {
	return cascadeBucketsRange(events, now.Add(-cascadeLookback), now)
}

func cascadeBucketsRange(events []LiquidationEvent, from, now time.Time) []cascadeBucket {
	if !from.Before(now) {
		return nil
	}
	n := int(now.Sub(from) / cascadeStep)
	if n < 1 {
		return nil
	}
	out := make([]cascadeBucket, n)
	for i := 0; i < n; i++ {
		out[i].start = from.Add(time.Duration(i) * cascadeStep)
	}
	for _, e := range events {
		t := e.Time.UTC()
		if t.Before(from) || !t.Before(now) {
			continue
		}
		i := int(t.Sub(from) / cascadeStep)
		if i < 0 || i >= n {
			continue
		}
		switch e.Side {
		case LiquidationSideLong:
			out[i].long += e.Notional
		case LiquidationSideShort:
			out[i].short += e.Notional
		}
		out[i].count++
	}
	return out
}

type cascadePack struct {
	long, short float64
	count       int
}

func measureCascadeWindow(buckets []cascadeBucket, id string, dur time.Duration, minPrior, minCount int) CascadeWindowRead {
	step := int(dur / cascadeStep)
	if step < 1 {
		step = 1
	}
	out := CascadeWindowRead{Window: id, Side: CascadeSideNone, Grade: CascadeGradeQuiet}
	if len(buckets) < step {
		return out
	}
	var blocks []cascadePack
	for i := 0; i+step <= len(buckets); i += step {
		var p cascadePack
		for j := 0; j < step; j++ {
			p.long += buckets[i+j].long
			p.short += buckets[i+j].short
			p.count += buckets[i+j].count
		}
		blocks = append(blocks, p)
	}
	if len(blocks) == 0 {
		return out
	}
	cur := blocks[len(blocks)-1]
	priors := blocks[:len(blocks)-1]
	out.SampleBuckets = len(priors)
	out.Complete = len(priors) >= minPrior
	out.Count = cur.count
	out.LongNotional = formatQty(cur.long)
	out.ShortNotional = formatQty(cur.short)
	out.TotalNotional = formatQty(cur.long + cur.short)
	floor := cascadeMinTypical1m * float64(step)
	longT := cascadeTypical(priors, true, floor)
	shortT := cascadeTypical(priors, false, floor)
	out.LongTypical = formatQty(longT)
	out.ShortTypical = formatQty(shortT)
	out.LongRatio = cascadeRatio(cur.long, longT)
	out.ShortRatio = cascadeRatio(cur.short, shortT)
	out.MaxRatio = math.Max(out.LongRatio, out.ShortRatio)
	out.Side = cascadeSideFromRatios(out.LongRatio, out.ShortRatio)
	out.Grade = cascadeGrade(out.MaxRatio, cur.count, minCount)
	if out.Grade == CascadeGradeQuiet {
		out.Side = CascadeSideNone
	}
	return out
}

func cascadeTypical(priors []cascadePack, longSide bool, floor float64) float64 {
	vals := make([]float64, 0, len(priors))
	for _, p := range priors {
		if longSide {
			vals = append(vals, p.long)
		} else {
			vals = append(vals, p.short)
		}
	}
	med := medianFloat(vals)
	if med < floor {
		return floor
	}
	return med
}

func cascadeRatio(current, typical float64) float64 {
	if typical <= 0 {
		if current <= 0 {
			return 0
		}
		return current / cascadeMinTypical1m
	}
	return current / typical
}

func cascadeSideFromRatios(longR, shortR float64) string {
	longHot := longR >= 2
	shortHot := shortR >= 2
	switch {
	case longHot && shortHot:
		return CascadeSideBoth
	case longHot && longR >= shortR:
		return LiquidationSideLong
	case shortHot && shortR > longR:
		return LiquidationSideShort
	default:
		return CascadeSideNone
	}
}

func cascadeGrade(maxRatio float64, count, minCount int) string {
	if maxRatio < 2 || count < 2 {
		return CascadeGradeQuiet
	}
	if maxRatio >= 8 && count >= minCount+2 {
		return CascadeGradeExtreme
	}
	if maxRatio >= 4 && count >= minCount {
		return CascadeGradeCascade
	}
	if maxRatio >= 2 {
		return CascadeGradeElevated
	}
	return CascadeGradeQuiet
}

func cascadeScore(w CascadeWindowRead) float64 {
	if w.MaxRatio <= 0 {
		return 0
	}
	s := 12.5 * w.MaxRatio
	if s > 100 {
		s = 100
	}
	return math.Round(s*10) / 10
}

// CascadeGradeRank is 0 quiet, 2 elevated, 3 cascade, 4 extreme.
func CascadeGradeRank(g string) int {
	return cascadeGradeRank(g)
}

func cascadeGradeRank(g string) int {
	switch g {
	case CascadeGradeExtreme:
		return 4
	case CascadeGradeCascade:
		return 3
	case CascadeGradeElevated:
		return 2
	default:
		return 0
	}
}

func cascadeWindowRank(id string) int {
	switch id {
	case CascadeWindow1m:
		return 0
	case CascadeWindow5m:
		return 1
	case CascadeWindow15m:
		return 2
	default:
		return 9
	}
}

func cascadeWeakerGrade(a, b string) string {
	if cascadeGradeRank(a) < cascadeGradeRank(b) {
		return a
	}
	return b
}

func cascadeSharedSide(a, b string) (string, bool) {
	if a == CascadeSideNone || b == CascadeSideNone {
		return CascadeSideNone, false
	}
	if a == b {
		return a, true
	}
	if a == CascadeSideBoth {
		return b, b != CascadeSideNone
	}
	if b == CascadeSideBoth {
		return a, a != CascadeSideNone
	}
	return CascadeSideNone, false
}

func cascadeStart(events []LiquidationEvent, window, side string, now time.Time) time.Time {
	var dur time.Duration
	for _, w := range CascadeWindows {
		if w.ID == window {
			dur = w.Dur
			break
		}
	}
	if dur <= 0 {
		return time.Time{}
	}
	cut := now.Add(-dur)
	var first time.Time
	for _, e := range events {
		if e.Time.Before(cut) {
			continue
		}
		if side != CascadeSideBoth && side != CascadeSideNone && e.Side != side {
			continue
		}
		if first.IsZero() || e.Time.Before(first) {
			first = e.Time.UTC()
		}
	}
	return first
}

func cascadeReportPeak(rep CascadeReport) (grade string, score float64, side, hottest string, both bool) {
	grade = CascadeGradeQuiet
	side = CascadeSideNone
	for _, v := range rep.Venues {
		if cascadeGradeRank(v.Grade) > cascadeGradeRank(grade) || (v.Grade == grade && v.Score > score) {
			grade = v.Grade
			score = v.Score
			side = v.Side
			hottest = v.Hottest
		}
	}
	if rep.Both != nil && rep.Both.Agree {
		both = true
		if cascadeGradeRank(rep.Both.Grade) > cascadeGradeRank(grade) {
			grade = rep.Both.Grade
			score = rep.Both.Score
			side = rep.Both.Side
			hottest = rep.Both.Hottest
		}
	}
	return grade, score, side, hottest, both
}

func explainCascadeVenue(v CascadeVenue) string {
	if v.Grade == CascadeGradeQuiet || v.Side == CascadeSideNone {
		return fmt.Sprintf("%s %s is in a normal liquidation range.", v.Exchange, v.Symbol)
	}
	return fmt.Sprintf("%s %s %s %s burst (%.1fx typical on %s).", v.Exchange, v.Symbol, v.Grade, v.Side, hottestRatio(v), v.Hottest)
}

func hottestRatio(v CascadeVenue) float64 {
	for _, w := range v.Windows {
		if w.Window == v.Hottest {
			return w.MaxRatio
		}
	}
	return 0
}

func explainCascadeReport(rep CascadeReport) string {
	if rep.Both != nil && rep.Both.Agree {
		return rep.Both.Summary
	}
	grade, _, side, hottest, _ := cascadeReportPeak(rep)
	if grade == CascadeGradeQuiet {
		if rep.Symbol == "all" {
			return "Market liquidation flow is in a normal range."
		}
		return "No liquidation cascade on this coin."
	}
	who := rep.Symbol
	if who == "all" {
		who = "market"
	}
	return fmt.Sprintf("%s %s %s-side liquidation burst (%s).", who, grade, side, hottest)
}

func explainCascadeScan(s CascadeScan) string {
	n := 0
	both := 0
	for _, h := range s.Hits {
		if cascadeGradeRank(h.Grade) >= cascadeGradeRank(CascadeGradeCascade) {
			n++
		}
		if h.Both {
			both++
		}
	}
	if n == 0 && (s.Market.Both == nil || !s.Market.Both.Agree) {
		return "No coins are in a liquidation cascade right now."
	}
	if both > 0 {
		return fmt.Sprintf("%d coin(s) cascading; %d on both Binance and Bybit.", n, both)
	}
	return fmt.Sprintf("%d coin(s) in a liquidation cascade.", n)
}
