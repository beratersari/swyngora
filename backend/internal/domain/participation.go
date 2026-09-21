package domain

import (
	"fmt"
	"math"
	"sort"
	"time"
)

const (
	ParticipationWindow15m = PrintSizeWindow15m
	ParticipationWindow1h  = OpenInterestWindow1h
	ParticipationWindow4h  = OpenInterestWindow4h

	ParticipationEntering = "entering"
	ParticipationExiting  = "exiting"
	ParticipationTransfer = "transfer"

	// |OI delta / volume| below this percent is mostly book transfer, not new size.
	participationTransferPct = 5.0
)

// ParticipationWindows is 15m / 1h / 4h.
var ParticipationWindows = []struct {
	ID  string
	Dur time.Duration
}{
	{ParticipationWindow15m, 15 * time.Minute},
	{ParticipationWindow1h, time.Hour},
	{ParticipationWindow4h, 4 * time.Hour},
}

// ParticipationWindow is OI delta as a share of futures volume in one lookback.
// 2B volume and OI +300M is ratio 0.15 (15% of volume stuck as new positions).
type ParticipationWindow struct {
	Window   string
	Volume   float64 // USDT taker volume (buy + sell)
	OIDelta  float64 // USDT open-interest value change
	Ratio    float64 // oiDelta / volume
	RatioPct float64 // ratio as percent (15 = 15%)
	Transfer float64 // volume − |oiDelta|, floored at 0
	Lean     string  // entering | exiting | transfer
	Complete bool
	Summary  string
}

// ParticipationVenue is one venue's 15m / 1h / 4h participation.
type ParticipationVenue struct {
	Exchange Exchange
	Symbol   string
	Windows  []ParticipationWindow
	Summary  string
	Error    string
}

// ParticipationReport is OI-delta vs futures volume (Hyblock-style).
type ParticipationReport struct {
	Symbol   string
	Exchange string
	AsOf     time.Time
	Venues   []ParticipationVenue
	Combined *ParticipationVenue
	Note     string
}

// ClassifyParticipation labels a signed ratio percent.
func ClassifyParticipation(ratioPct float64) string {
	switch {
	case ratioPct >= participationTransferPct:
		return ParticipationEntering
	case ratioPct <= -participationTransferPct:
		return ParticipationExiting
	default:
		return ParticipationTransfer
	}
}

// OpenInterestValueDelta is current USDT OI minus the sample ~dur ago.
func OpenInterestValueDelta(s *OpenInterestSeries, dur time.Duration, now time.Time) (delta float64, complete bool) {
	if s == nil {
		return 0, false
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	past, complete := FindOpenInterestSample(s.History, now.Add(-dur), OpenInterestSampleSlack(dur))
	if past.Time.IsZero() {
		return 0, false
	}
	return s.Current.Value - past.Value, complete
}

// InferBucketBar guesses 1m vs 5m from adjacent bucket starts.
func InferBucketBar(buckets []TakerBucket) time.Duration {
	if len(buckets) < 2 {
		return time.Minute
	}
	sorted := append([]TakerBucket(nil), buckets...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Start.Before(sorted[j].Start) })
	gap := time.Duration(0)
	for i := 1; i < len(sorted); i++ {
		d := sorted[i].Start.Sub(sorted[i-1].Start)
		if d <= 0 {
			continue
		}
		if gap == 0 || d < gap {
			gap = d
		}
	}
	if gap >= 4*time.Minute {
		return 5 * time.Minute
	}
	if gap > 0 {
		return gap
	}
	return time.Minute
}

// TakerVolumeInWindow is buy+sell notional overlapping [now−dur, now].
func TakerVolumeInWindow(buckets []TakerBucket, now time.Time, dur, bar time.Duration) (vol float64, complete bool) {
	if len(buckets) == 0 || dur <= 0 {
		return 0, false
	}
	now = now.UTC()
	if bar <= 0 {
		bar = InferBucketBar(buckets)
	}
	cut := now.Add(-dur)
	var oldest, newest time.Time
	for _, b := range buckets {
		if oldest.IsZero() || b.Start.Before(oldest) {
			oldest = b.Start
		}
		if newest.IsZero() || b.Start.After(newest) {
			newest = b.Start
		}
	}
	for _, b := range buckets {
		end := b.Start.Add(bar)
		if end.After(cut) || (!newest.IsZero() && b.Start.Equal(newest)) {
			vol += b.BuyNotional + b.SellNotional
		}
	}
	return vol, !oldest.IsZero() && !oldest.After(cut)
}

// BuildParticipationWindow is OI delta / volume for one lookback.
func BuildParticipationWindow(oi *OpenInterestSeries, buckets []TakerBucket, windowID string, dur time.Duration, now time.Time) ParticipationWindow {
	out := ParticipationWindow{Window: windowID, Lean: ParticipationTransfer}
	vol, volOK := TakerVolumeInWindow(buckets, now, dur, InferBucketBar(buckets))
	delta, oiOK := OpenInterestValueDelta(oi, dur, now)
	out.Volume = vol
	out.OIDelta = delta
	if vol > 0 {
		out.Ratio = delta / vol
		out.RatioPct = out.Ratio * 100
		out.Lean = ClassifyParticipation(out.RatioPct)
		xfer := vol - math.Abs(delta)
		if xfer < 0 {
			xfer = 0
		}
		out.Transfer = xfer
	}
	out.Complete = volOK && oiOK && vol > 0
	out.Summary = ExplainParticipationWindow(out)
	return out
}

// BuildParticipationVenue fills 15m / 1h / 4h for one venue.
func BuildParticipationVenue(ex Exchange, symbol string, oi *OpenInterestSeries, buckets []TakerBucket, now time.Time) ParticipationVenue {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	out := ParticipationVenue{
		Exchange: ex,
		Symbol:   NormalizeLiquidationSymbol(symbol),
		Windows:  make([]ParticipationWindow, 0, len(ParticipationWindows)),
	}
	for _, w := range ParticipationWindows {
		out.Windows = append(out.Windows, BuildParticipationWindow(oi, buckets, w.ID, w.Dur, now))
	}
	out.Summary = ExplainParticipationVenue(out)
	if oi == nil && len(buckets) == 0 {
		out.Error = "participation not configured"
		out.Summary = out.Error
	}
	return out
}

// CombineParticipationVenues sums volume and OI delta, then recomputes the ratio.
func CombineParticipationVenues(symbol string, venues []ParticipationVenue) *ParticipationVenue {
	if len(venues) == 0 {
		return nil
	}
	out := &ParticipationVenue{
		Exchange: "all",
		Symbol:   NormalizeLiquidationSymbol(symbol),
		Windows:  make([]ParticipationWindow, 0, len(ParticipationWindows)),
	}
	for _, spec := range ParticipationWindows {
		var vol, delta float64
		complete := true
		any := false
		for _, v := range venues {
			for _, w := range v.Windows {
				if w.Window != spec.ID {
					continue
				}
				vol += w.Volume
				delta += w.OIDelta
				any = true
				if !w.Complete {
					complete = false
				}
			}
		}
		win := ParticipationWindow{Window: spec.ID, Volume: vol, OIDelta: delta, Lean: ParticipationTransfer, Complete: any && complete && vol > 0}
		if vol > 0 {
			win.Ratio = delta / vol
			win.RatioPct = win.Ratio * 100
			win.Lean = ClassifyParticipation(win.RatioPct)
			xfer := vol - math.Abs(delta)
			if xfer < 0 {
				xfer = 0
			}
			win.Transfer = xfer
		} else {
			win.Complete = false
		}
		win.Summary = ExplainParticipationWindow(win)
		out.Windows = append(out.Windows, win)
	}
	out.Summary = ExplainParticipationVenue(*out)
	return out
}

// ExplainParticipationWindow is the desk line for one lookback.
func ExplainParticipationWindow(w ParticipationWindow) string {
	if w.Volume <= 0 {
		return fmt.Sprintf("%s: not enough futures volume to score participation.", w.Window)
	}
	head := fmt.Sprintf("%s: %s volume, OI %s (%s of volume)",
		w.Window, formatQty(w.Volume), FormatSignedQty(w.OIDelta), formatParticipationPct(w.RatioPct))
	switch w.Lean {
	case ParticipationEntering:
		return head + " is new positions."
	case ParticipationExiting:
		return head + " is positions closing."
	default:
		return head + " is mostly transfer between existing books."
	}
}

// ExplainParticipationVenue prefers the 15m window, then 1h, then 4h.
func ExplainParticipationVenue(v ParticipationVenue) string {
	pick := func(id string) *ParticipationWindow {
		for i := range v.Windows {
			if v.Windows[i].Window == id && v.Windows[i].Volume > 0 {
				return &v.Windows[i]
			}
		}
		return nil
	}
	if w := pick(ParticipationWindow15m); w != nil {
		return w.Summary
	}
	if w := pick(ParticipationWindow1h); w != nil {
		return w.Summary
	}
	if w := pick(ParticipationWindow4h); w != nil {
		return w.Summary
	}
	if v.Error != "" {
		return v.Error
	}
	return "Not enough open-interest and volume history to score participation."
}

func formatParticipationPct(v float64) string {
	s := formatFixed(v, 1)
	if v > 0 && s != "0" && s[0] != '+' && s[0] != '-' {
		return "+" + s + "%"
	}
	return s + "%"
}
