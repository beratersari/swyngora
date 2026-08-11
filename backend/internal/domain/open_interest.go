package domain

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	OpenInterestWindow5m  = LiquidationWindow5m
	OpenInterestWindow1h  = LiquidationWindow1h
	OpenInterestWindow4h  = LiquidationWindow4h
	OpenInterestWindow24h = LiquidationWindow24h
)

// OpenInterestWindows is the same 5m / 1h / 4h / 24h set as liquidations.
var OpenInterestWindows = LiquidationWindows

// OpenInterestPoint is one venue snapshot of outstanding futures size.
type OpenInterestPoint struct {
	Time      time.Time
	Contracts float64 // base-asset contracts as published by the venue
	Value     float64 // USDT notional
}

// OpenInterestSeries is the current reading plus history used for window deltas.
type OpenInterestSeries struct {
	Exchange Exchange
	Symbol   string
	Current  OpenInterestPoint
	History  []OpenInterestPoint // oldest first
}

// OpenInterestLevel is a formatted contracts + USDT reading.
type OpenInterestLevel struct {
	Contracts string
	Value     string
	Time      time.Time
}

// OpenInterestWindow is current vs the reading ~window ago.
type OpenInterestWindow struct {
	Window            string
	OpenInterest      string // contracts at the past sample
	OpenInterestValue string
	Change            string
	ChangePct         string
	ChangeValue       string
	ChangeValuePct    string
	Direction         string // up | down | flat
	Complete          bool
	SampleTime        time.Time
}

// OpenInterestVenueSnap is one venue inside a combined snapshot.
type OpenInterestVenueSnap struct {
	Exchange string
	Current  OpenInterestLevel
	Windows  []OpenInterestWindow
}

// OpenInterestSnapshot is current OI plus 5m/1h/4h/24h change.
type OpenInterestSnapshot struct {
	Symbol     string
	Exchange   string // binance | bybit | all
	Unit       string // base asset, e.g. BTC
	Current    OpenInterestLevel
	Windows    []OpenInterestWindow
	Venues     []OpenInterestVenueSnap
	AsOf       time.Time
	VenueCount int
	Funding    *FundingSnapshot   // current rate + recent settlements when available
	LongShort  *LongShortSnapshot // account long/short ratio when available
}

// ParseOpenInterestExchange accepts binance, bybit, or all (empty = all).
func ParseOpenInterestExchange(raw string) (string, error) {
	return ParseLiquidationExchange(raw)
}

// OpenInterestSampleSlack is how far before the target a hist sample may sit.
func OpenInterestSampleSlack(window time.Duration) time.Duration {
	s := window / 6
	if s < 3*time.Minute {
		s = 3 * time.Minute
	}
	if s > 90*time.Minute {
		s = 90 * time.Minute
	}
	return s
}

// FindOpenInterestSample returns the latest history point at or before target.
// complete is true when that point is within slack of the target.
func FindOpenInterestSample(hist []OpenInterestPoint, target time.Time, slack time.Duration) (OpenInterestPoint, bool) {
	var best OpenInterestPoint
	found := false
	for _, p := range hist {
		if p.Time.IsZero() || p.Time.After(target) {
			continue
		}
		if !found || p.Time.After(best.Time) {
			best = p
			found = true
		}
	}
	if !found {
		return OpenInterestPoint{}, false
	}
	return best, !best.Time.Before(target.Add(-slack))
}

// OpenInterestDirection labels contract % change (0.05% deadband).
func OpenInterestDirection(changePct float64) string {
	switch {
	case changePct > 0.05:
		return "up"
	case changePct < -0.05:
		return "down"
	default:
		return "flat"
	}
}

// FormatSignedQty formats a signed quantity for API/Telegram (+ / − / 0).
func FormatSignedQty(v float64) string {
	if v == 0 || math.Abs(v) < 1e-12 {
		return "0"
	}
	if v > 0 {
		return "+" + formatQty(v)
	}
	return "-" + formatQty(-v)
}

// FormatSignedPct formats a signed percent with two decimals.
func FormatSignedPct(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "0"
	}
	if v == 0 {
		return "0"
	}
	s := formatFixed(v, 2)
	if v > 0 && !strings.HasPrefix(s, "-") {
		return "+" + s
	}
	return s
}

func oiPctChange(cur, past float64) (float64, bool) {
	if past == 0 {
		return 0, cur == 0
	}
	return (cur - past) / math.Abs(past) * 100, true
}

// FormatOpenInterestLevel renders a numeric point.
func FormatOpenInterestLevel(p OpenInterestPoint) OpenInterestLevel {
	t := p.Time.UTC()
	if p.Time.IsZero() {
		t = time.Time{}
	}
	return OpenInterestLevel{
		Contracts: formatQty(p.Contracts),
		Value:     formatQty(p.Value),
		Time:      t,
	}
}

// SummarizeOpenInterestWindow compares current to the sample ~dur ago.
func SummarizeOpenInterestWindow(s *OpenInterestSeries, windowID string, dur time.Duration, now time.Time) OpenInterestWindow {
	out := OpenInterestWindow{Window: windowID}
	if s == nil {
		return out
	}
	cur := s.Current
	if cur.Time.IsZero() && cur.Contracts == 0 && cur.Value == 0 {
		return out
	}
	target := now.Add(-dur)
	past, complete := FindOpenInterestSample(s.History, target, OpenInterestSampleSlack(dur))
	if past.Time.IsZero() {
		return out
	}
	dC := cur.Contracts - past.Contracts
	dV := cur.Value - past.Value
	pC, okC := oiPctChange(cur.Contracts, past.Contracts)
	pV, okV := oiPctChange(cur.Value, past.Value)
	if !okC {
		complete = false
	}
	if !okV {
		pV = 0
	}
	out.OpenInterest = formatQty(past.Contracts)
	out.OpenInterestValue = formatQty(past.Value)
	out.Change = FormatSignedQty(dC)
	out.ChangePct = FormatSignedPct(pC)
	out.ChangeValue = FormatSignedQty(dV)
	out.ChangeValuePct = FormatSignedPct(pV)
	out.Direction = OpenInterestDirection(pC)
	out.Complete = complete
	out.SampleTime = past.Time.UTC()
	return out
}

func venueWindows(s *OpenInterestSeries, now time.Time) []OpenInterestWindow {
	out := make([]OpenInterestWindow, 0, len(OpenInterestWindows))
	for _, w := range OpenInterestWindows {
		out = append(out, SummarizeOpenInterestWindow(s, w.ID, w.Dur, now))
	}
	return out
}

func combineOpenInterestWindow(series []*OpenInterestSeries, windowID string, dur time.Duration, now time.Time) OpenInterestWindow {
	out := OpenInterestWindow{Window: windowID, Complete: true}
	var pastC, pastV, curC, curV float64
	used := 0
	var oldest time.Time
	for _, s := range series {
		if s == nil {
			continue
		}
		w := SummarizeOpenInterestWindow(s, windowID, dur, now)
		if w.SampleTime.IsZero() {
			out.Complete = false
			continue
		}
		past, _ := FindOpenInterestSample(s.History, now.Add(-dur), OpenInterestSampleSlack(dur))
		pastC += past.Contracts
		pastV += past.Value
		curC += s.Current.Contracts
		curV += s.Current.Value
		used++
		if oldest.IsZero() || w.SampleTime.Before(oldest) {
			oldest = w.SampleTime
		}
		if !w.Complete {
			out.Complete = false
		}
	}
	if used == 0 {
		out.Complete = false
		return out
	}
	dC := curC - pastC
	dV := curV - pastV
	pC, okC := oiPctChange(curC, pastC)
	pV, okV := oiPctChange(curV, pastV)
	if !okC {
		out.Complete = false
	}
	if !okV {
		pV = 0
	}
	out.OpenInterest = formatQty(pastC)
	out.OpenInterestValue = formatQty(pastV)
	out.Change = FormatSignedQty(dC)
	out.ChangePct = FormatSignedPct(pC)
	out.ChangeValue = FormatSignedQty(dV)
	out.ChangeValuePct = FormatSignedPct(pV)
	out.Direction = OpenInterestDirection(pC)
	out.SampleTime = oldest.UTC()
	return out
}

func sumCurrent(series []*OpenInterestSeries) OpenInterestPoint {
	var out OpenInterestPoint
	for _, s := range series {
		if s == nil {
			continue
		}
		out.Contracts += s.Current.Contracts
		out.Value += s.Current.Value
		if out.Time.IsZero() || s.Current.Time.After(out.Time) {
			out.Time = s.Current.Time
		}
	}
	return out
}

// SortOpenInterestHistory oldest-first and drops duplicate timestamps (keeps last).
func SortOpenInterestHistory(hist []OpenInterestPoint) []OpenInterestPoint {
	if len(hist) == 0 {
		return hist
	}
	sort.SliceStable(hist, func(i, j int) bool {
		return hist[i].Time.Before(hist[j].Time)
	})
	out := hist[:0]
	for _, p := range hist {
		if p.Time.IsZero() || p.Contracts < 0 {
			continue
		}
		if len(out) > 0 && out[len(out)-1].Time.Equal(p.Time) {
			out[len(out)-1] = p
			continue
		}
		out = append(out, p)
	}
	return out
}

// BuildOpenInterestSnapshot folds one or more venue series into the API shape.
func BuildOpenInterestSnapshot(exchange, symbol string, series []*OpenInterestSeries, now time.Time) *OpenInterestSnapshot {
	symbol = NormalizeLiquidationSymbol(symbol)
	base, _ := SplitBaseQuote(ExchangeBinance, symbol)
	now = now.UTC()
	out := &OpenInterestSnapshot{
		Symbol:   symbol,
		Exchange: exchange,
		Unit:     base,
		AsOf:     now,
		Windows:  []OpenInterestWindow{},
		Venues:   []OpenInterestVenueSnap{},
	}
	clean := make([]*OpenInterestSeries, 0, len(series))
	for _, s := range series {
		if s == nil {
			continue
		}
		s.History = SortOpenInterestHistory(s.History)
		clean = append(clean, s)
	}
	sort.Slice(clean, func(i, j int) bool {
		return string(clean[i].Exchange) < string(clean[j].Exchange)
	})
	out.VenueCount = len(clean)
	for _, s := range clean {
		out.Venues = append(out.Venues, OpenInterestVenueSnap{
			Exchange: string(s.Exchange),
			Current:  FormatOpenInterestLevel(s.Current),
			Windows:  venueWindows(s, now),
		})
	}
	if len(clean) == 0 {
		return out
	}
	if exchange != "all" && len(clean) == 1 {
		out.Current = FormatOpenInterestLevel(clean[0].Current)
		out.Windows = venueWindows(clean[0], now)
		return out
	}
	out.Current = FormatOpenInterestLevel(sumCurrent(clean))
	for _, w := range OpenInterestWindows {
		out.Windows = append(out.Windows, combineOpenInterestWindow(clean, w.ID, w.Dur, now))
	}
	return out
}

// ValidateOpenInterestSymbol rejects empty after futures-style normalize.
func ValidateOpenInterestSymbol(symbol string) (string, error) {
	s := NormalizeLiquidationSymbol(symbol)
	if s == "" {
		return "", fmt.Errorf("%w: symbol is required", ErrInvalidArgument)
	}
	return s, nil
}
