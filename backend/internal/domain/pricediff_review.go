package domain

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	// MaxPriceDiffOppTicks is how many checker samples we keep per opportunity.
	MaxPriceDiffOppTicks = 360
	// PriceDiffFadeMinDropPct is the net-% drop from the peak that starts a fade.
	PriceDiffFadeMinDropPct = 0.01
)

// PriceDiffCloseReason says why an opportunity stopped qualifying.
type PriceDiffCloseReason string

const (
	PriceDiffCloseBelowMinProfit PriceDiffCloseReason = "below_min_profit"
	PriceDiffCloseBelowMinNet    PriceDiffCloseReason = "below_min_net"
	PriceDiffCloseThinBook       PriceDiffCloseReason = "thin_book"
	PriceDiffClosePaused         PriceDiffCloseReason = "paused"
	PriceDiffCloseVenuesChanged  PriceDiffCloseReason = "venues_changed"
)

// PriceDiffOppSample is one checker reading of a live book route.
type PriceDiffOppSample struct {
	At           time.Time
	BuyPrice     float64
	SellPrice    float64
	GrossDiffPct float64
	NetDiffPct   float64
	Profit       float64
	Executable   bool
}

// PriceDiffOppTick is a stored sample on an opportunity path.
type PriceDiffOppTick struct {
	OpportunityID string
	At            time.Time
	BuyPrice      float64
	SellPrice     float64
	GrossDiffPct  float64
	NetDiffPct    float64
	Profit        float64
	Executable    bool
}

// PriceDiffOppReview is the after-the-open story for a gap.
type PriceDiffOppReview struct {
	OpenDurationSec float64
	OpenNetDiffPct  float64
	OpenProfit      float64
	PeakNetDiffPct  float64
	PeakProfit      float64
	PeakAt          *time.Time
	PeakDeltaPct    float64
	FadeStartedAt   *time.Time
	FadeFromPeakSec *float64
	LastNetDiffPct  float64
	LastProfit      float64
	CloseReason     PriceDiffCloseReason
	Summary         string
}

// PriceDiffOppSampleFromQuote maps a live book walk onto a sample.
func PriceDiffOppSampleFromQuote(q *PriceDiffQuote, at time.Time) PriceDiffOppSample {
	if q == nil {
		return PriceDiffOppSample{At: at}
	}
	buy, _ := strconv.ParseFloat(q.AverageBuyPrice, 64)
	sell, _ := strconv.ParseFloat(q.AverageSellPrice, 64)
	return PriceDiffOppSample{
		At:           at,
		BuyPrice:     buy,
		SellPrice:    sell,
		GrossDiffPct: q.GrossPct,
		NetDiffPct:   q.ProfitPct,
		Profit:       q.AfterFeeProfit(),
		Executable:   q.Executable,
	}
}

// InitPriceDiffOpportunity copies the first qualifying sample onto open and peak.
func InitPriceDiffOpportunity(o *PriceDiffOpportunity, s PriceDiffOppSample) {
	if o == nil {
		return
	}
	o.BuyPrice = s.BuyPrice
	o.SellPrice = s.SellPrice
	o.GrossDiffPct = s.GrossDiffPct
	o.NetDiffPct = s.NetDiffPct
	o.OpenedAt = s.At
	o.LastSeenAt = s.At
	o.OpenBuyPrice = s.BuyPrice
	o.OpenSellPrice = s.SellPrice
	o.OpenGrossDiffPct = s.GrossDiffPct
	o.OpenNetDiffPct = s.NetDiffPct
	o.OpenProfit = s.Profit
	o.PeakBuyPrice = s.BuyPrice
	o.PeakSellPrice = s.SellPrice
	o.PeakGrossDiffPct = s.GrossDiffPct
	o.PeakNetDiffPct = s.NetDiffPct
	o.PeakProfit = s.Profit
	o.PeakAt = s.At
	o.FadeStartedAt = nil
}

// ApplyPriceDiffOppTick updates last / peak / fade from a later sample.
func ApplyPriceDiffOppTick(o *PriceDiffOpportunity, s PriceDiffOppSample) {
	if o == nil || s.At.IsZero() {
		return
	}
	o.BuyPrice = s.BuyPrice
	o.SellPrice = s.SellPrice
	o.GrossDiffPct = s.GrossDiffPct
	o.NetDiffPct = s.NetDiffPct
	o.LastSeenAt = s.At
	if s.NetDiffPct > o.PeakNetDiffPct+1e-12 {
		o.PeakBuyPrice = s.BuyPrice
		o.PeakSellPrice = s.SellPrice
		o.PeakGrossDiffPct = s.GrossDiffPct
		o.PeakNetDiffPct = s.NetDiffPct
		o.PeakProfit = s.Profit
		o.PeakAt = s.At
		o.FadeStartedAt = nil
		return
	}
	if o.FadeStartedAt == nil && o.PeakNetDiffPct-s.NetDiffPct >= PriceDiffFadeMinDropPct {
		t := s.At.UTC()
		o.FadeStartedAt = &t
	}
}

// ClosePriceDiffOpportunity marks the row closed and records why.
func ClosePriceDiffOpportunity(o *PriceDiffOpportunity, at time.Time, reason PriceDiffCloseReason, last *PriceDiffOppSample) {
	if o == nil {
		return
	}
	if last != nil {
		sample := *last
		if sample.At.IsZero() {
			sample.At = at
		}
		ApplyPriceDiffOppTick(o, sample)
		o.CloseNetDiffPct = last.NetDiffPct
		o.CloseProfit = last.Profit
	} else {
		o.CloseNetDiffPct = o.NetDiffPct
	}
	o.Status = PriceDiffOppClosed
	end := at.UTC()
	o.ClosedAt = &end
	o.LastSeenAt = end
	o.CloseReason = reason
}

// PriceDiffCloseReasonFromQuote classifies a no-longer-qualifying live quote.
func PriceDiffCloseReasonFromQuote(q *PriceDiffQuote, w *PriceDiffWatch) PriceDiffCloseReason {
	if q == nil || !q.Executable {
		return PriceDiffCloseThinBook
	}
	if w != nil && q.AfterFeeProfit()+1e-12 < w.MinProfit {
		return PriceDiffCloseBelowMinProfit
	}
	if w != nil && w.MinNetDiffPct > 0 && q.ProfitPct+1e-12 < w.MinNetDiffPct {
		return PriceDiffCloseBelowMinNet
	}
	return PriceDiffCloseBelowMinProfit
}

// PriceDiffOppDurationSec is open length to closedAt, lastSeenAt, or now.
func PriceDiffOppDurationSec(o PriceDiffOpportunity, now time.Time) float64 {
	start := o.OpenedAt.UTC()
	if start.IsZero() {
		return 0
	}
	end := now.UTC()
	if o.ClosedAt != nil && !o.ClosedAt.IsZero() {
		end = o.ClosedAt.UTC()
	} else if !o.LastSeenAt.IsZero() {
		end = o.LastSeenAt.UTC()
	}
	sec := end.Sub(start).Seconds()
	if sec < 0 || math.IsNaN(sec) {
		return 0
	}
	return sec
}

// BuildPriceDiffOppReview tells how the gap moved after it opened.
func BuildPriceDiffOppReview(o PriceDiffOpportunity, now time.Time) PriceDiffOppReview {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	rev := PriceDiffOppReview{
		OpenDurationSec: PriceDiffOppDurationSec(o, now),
		OpenNetDiffPct:  o.OpenNetDiffPct,
		OpenProfit:      o.OpenProfit,
		PeakNetDiffPct:  o.PeakNetDiffPct,
		PeakProfit:      o.PeakProfit,
		PeakDeltaPct:    o.PeakNetDiffPct - o.OpenNetDiffPct,
		LastNetDiffPct:  o.NetDiffPct,
		LastProfit:      o.CloseProfit,
		CloseReason:     o.CloseReason,
	}
	if o.Status == PriceDiffOppOpen {
		rev.LastProfit = 0
	}
	if !o.PeakAt.IsZero() {
		t := o.PeakAt.UTC()
		rev.PeakAt = &t
	}
	if o.FadeStartedAt != nil && !o.FadeStartedAt.IsZero() {
		t := o.FadeStartedAt.UTC()
		rev.FadeStartedAt = &t
		if rev.PeakAt != nil {
			sec := t.Sub(*rev.PeakAt).Seconds()
			if sec < 0 {
				sec = 0
			}
			rev.FadeFromPeakSec = &sec
		}
	}
	rev.Summary = priceDiffOppReviewSummary(o, rev)
	return rev
}

func priceDiffOppReviewSummary(o PriceDiffOpportunity, rev PriceDiffOppReview) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Open %s.", formatPriceDiffDuration(rev.OpenDurationSec))
	fmt.Fprintf(&b, " After-fee edge opened at %.3f%%", rev.OpenNetDiffPct)
	if rev.OpenProfit != 0 {
		fmt.Fprintf(&b, " (%.2f quote)", rev.OpenProfit)
	}
	fmt.Fprintf(&b, ".")
	if rev.PeakDeltaPct > 1e-6 && rev.PeakAt != nil {
		fmt.Fprintf(&b, " Peaked at %.3f%%", rev.PeakNetDiffPct)
		if !o.OpenedAt.IsZero() {
			fmt.Fprintf(&b, " %s after open", formatPriceDiffDuration(rev.PeakAt.Sub(o.OpenedAt.UTC()).Seconds()))
		}
		fmt.Fprintf(&b, ".")
	} else {
		b.WriteString(" Edge never rose after open.")
	}
	if rev.FadeStartedAt != nil {
		if rev.FadeFromPeakSec != nil {
			fmt.Fprintf(&b, " Started fading %s after the peak.", formatPriceDiffDuration(*rev.FadeFromPeakSec))
		} else {
			b.WriteString(" Started fading after the peak.")
		}
	}
	if o.Status == PriceDiffOppClosed {
		fmt.Fprintf(&b, " Closed: %s", priceDiffCloseReasonText(o.CloseReason))
		fmt.Fprintf(&b, " (last %.3f%%).", o.CloseNetDiffPct)
	}
	return b.String()
}

func priceDiffCloseReasonText(r PriceDiffCloseReason) string {
	switch r {
	case PriceDiffCloseBelowMinProfit:
		return "fill no longer met the min profit floor"
	case PriceDiffCloseBelowMinNet:
		return "fill no longer met the min net-% floor"
	case PriceDiffCloseThinBook:
		return "the size no longer filled on both books"
	case PriceDiffClosePaused:
		return "the watch was paused"
	case PriceDiffCloseVenuesChanged:
		return "the route's venues were removed from the watch"
	default:
		return "the route stopped qualifying"
	}
}

func formatPriceDiffDuration(sec float64) string {
	if sec < 0 || math.IsNaN(sec) || math.IsInf(sec, 0) {
		return "0s"
	}
	if sec < 60 {
		return fmt.Sprintf("%.0fs", sec)
	}
	if sec < 3600 {
		m := int(sec) / 60
		s := int(sec) % 60
		if s == 0 {
			return fmt.Sprintf("%dm", m)
		}
		return fmt.Sprintf("%dm %ds", m, s)
	}
	h := int(sec) / 3600
	m := (int(sec) % 3600) / 60
	if m == 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dh %dm", h, m)
}
