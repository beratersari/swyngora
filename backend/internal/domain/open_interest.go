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

	OpenInterestFlowOpened = "opened"
	OpenInterestFlowClosed = "closed"
	OpenInterestFlowFlat   = "flat"
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
	Flow              string // opened | closed | flat
	OpenedValue       string // USDT notional added when OI rose
	ClosedValue       string // USDT notional closed when OI fell
	Complete          bool
	SampleTime        time.Time
	// Missing names venues that were passed in and lacked a usable sample.
	// A missing venue is not counted as zero.
	Missing []string
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
	// Missing names venues asked for on exchange=all that did not return a series.
	// Combined current and windows stay blank until they are in.
	Missing   []string
	Funding   *FundingSnapshot   // current rate + recent settlements when available
	LongShort *LongShortSnapshot // account long/short ratio when available
}

// ParseOpenInterestExchange accepts binance, bybit, okx, hyperliquid, or all (empty = all).
// Hyperliquid is mark, open interest, and funding only. Liquidation routes still reject it.
func ParseOpenInterestExchange(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" || s == "all" || s == "combined" {
		return "all", nil
	}
	ex := Exchange(s)
	if IsFuturesVenue(ex) || IsMarkOnlyVenue(ex) {
		return s, nil
	}
	return "", fmt.Errorf("%w: exchange must be binance, bybit, okx, hyperliquid, or all", ErrInvalidArgument)
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
	attachOpenInterestFlow(&out, dV)
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
	var pastC, pastV, curC, curV float64
	used := 0
	var oldest time.Time
	var missing []string
	for _, s := range series {
		if s == nil {
			continue
		}
		w := SummarizeOpenInterestWindow(s, windowID, dur, now)
		// A venue with no sample, or a sample outside the slack, is not zero.
		// The other venues' change is not published as the combined window.
		if w.SampleTime.IsZero() || !w.Complete {
			missing = append(missing, exchangeLabel(s.Exchange))
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
	}
	if len(missing) > 0 || used == 0 {
		return OpenInterestWindow{
			Window:   windowID,
			Complete: false,
			Missing:  uniqueSortedStrings(missing),
		}
	}
	out := OpenInterestWindow{Window: windowID, Complete: true}
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
	attachOpenInterestFlow(&out, dV)
	out.SampleTime = oldest.UTC()
	return out
}

func attachOpenInterestFlow(w *OpenInterestWindow, deltaValue float64) {
	if w == nil {
		return
	}
	switch w.Direction {
	case "up":
		w.Flow = OpenInterestFlowOpened
		if deltaValue > 0 {
			w.OpenedValue = formatQty(deltaValue)
		}
	case "down":
		w.Flow = OpenInterestFlowClosed
		if deltaValue < 0 {
			w.ClosedValue = formatQty(-deltaValue)
		}
	default:
		w.Flow = OpenInterestFlowFlat
	}
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

// WithholdCombinedOpenInterest blanks the combined current and every combined
// window when a venue that was asked for did not return. Per-venue rows stay.
// A missing venue is named and is not counted as zero.
func WithholdCombinedOpenInterest(snap *OpenInterestSnapshot, missing []string) {
	if snap == nil || snap.Exchange != "all" || len(missing) == 0 {
		return
	}
	names := uniqueSortedStrings(missing)
	snap.Missing = names
	snap.Current = OpenInterestLevel{}
	for i := range snap.Windows {
		id := snap.Windows[i].Window
		snap.Windows[i] = OpenInterestWindow{
			Window:   id,
			Complete: false,
			Missing:  append([]string{}, names...),
		}
	}
}

// exchangeLabel is the venue name used when a combined read names who is missing.
func exchangeLabel(ex Exchange) string {
	s := strings.TrimSpace(string(ex))
	if s == "" {
		return "venue"
	}
	return s
}

// uniqueSortedStrings returns a sorted copy without blanks or duplicates.
func uniqueSortedStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
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
