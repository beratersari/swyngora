package domain

import (
	"strings"
	"testing"
	"time"
)

func TestApplyPriceDiffOppTick_PeakAndFade(t *testing.T) {
	t0 := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	o := &PriceDiffOpportunity{}
	InitPriceDiffOpportunity(o, PriceDiffOppSample{At: t0, BuyPrice: 100, SellPrice: 101, NetDiffPct: 0.40, Profit: 40})
	if o.OpenNetDiffPct != 0.40 || o.PeakNetDiffPct != 0.40 || o.FadeStartedAt != nil {
		t.Fatalf("init %+v", o)
	}

	ApplyPriceDiffOppTick(o, PriceDiffOppSample{At: t0.Add(30 * time.Second), NetDiffPct: 0.80, Profit: 80, BuyPrice: 100, SellPrice: 101.8})
	if o.PeakNetDiffPct != 0.80 || o.PeakAt != t0.Add(30*time.Second) || o.FadeStartedAt != nil {
		t.Fatalf("peak %+v", o)
	}

	ApplyPriceDiffOppTick(o, PriceDiffOppSample{At: t0.Add(60 * time.Second), NetDiffPct: 0.795, Profit: 79.5})
	if o.FadeStartedAt != nil {
		t.Fatalf("drop under 0.01%% should not start a fade: %+v", o.FadeStartedAt)
	}
	ApplyPriceDiffOppTick(o, PriceDiffOppSample{At: t0.Add(90 * time.Second), NetDiffPct: 0.78, Profit: 78})
	if o.FadeStartedAt == nil || !o.FadeStartedAt.Equal(t0.Add(90*time.Second)) {
		t.Fatalf("fade %+v", o.FadeStartedAt)
	}

	ApplyPriceDiffOppTick(o, PriceDiffOppSample{At: t0.Add(2 * time.Minute), NetDiffPct: 0.95, Profit: 95})
	if o.PeakNetDiffPct != 0.95 || o.FadeStartedAt != nil {
		t.Fatalf("new peak should clear fade %+v", o)
	}
}

func TestClosePriceDiffOpportunity_ReasonAndLast(t *testing.T) {
	t0 := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	o := &PriceDiffOpportunity{Status: PriceDiffOppOpen, NetDiffPct: 0.4}
	InitPriceDiffOpportunity(o, PriceDiffOppSample{At: t0, NetDiffPct: 0.4, Profit: 40})
	last := PriceDiffOppSample{At: t0.Add(time.Minute), NetDiffPct: 0.05, Profit: 5}
	ClosePriceDiffOpportunity(o, t0.Add(time.Minute), PriceDiffCloseBelowMinProfit, &last)
	if o.Status != PriceDiffOppClosed || o.CloseReason != PriceDiffCloseBelowMinProfit {
		t.Fatalf("%+v", o)
	}
	if o.ClosedAt == nil || o.CloseNetDiffPct != 0.05 || o.CloseProfit != 5 {
		t.Fatalf("close fields %+v", o)
	}
}

func TestPriceDiffCloseReasonFromQuote(t *testing.T) {
	w := &PriceDiffWatch{MinProfit: 10, MinNetDiffPct: 0.5}
	if got := PriceDiffCloseReasonFromQuote(nil, w); got != PriceDiffCloseThinBook {
		t.Fatalf("%s", got)
	}
	thin := &PriceDiffQuote{Executable: false, ProfitPct: 1, profitAmount: 20}
	if got := PriceDiffCloseReasonFromQuote(thin, w); got != PriceDiffCloseThinBook {
		t.Fatalf("%s", got)
	}
	lowProfit := &PriceDiffQuote{Executable: true, ProfitPct: 1, profitAmount: 2}
	if got := PriceDiffCloseReasonFromQuote(lowProfit, w); got != PriceDiffCloseBelowMinProfit {
		t.Fatalf("%s", got)
	}
	lowNet := &PriceDiffQuote{Executable: true, ProfitPct: 0.1, profitAmount: 20}
	if got := PriceDiffCloseReasonFromQuote(lowNet, w); got != PriceDiffCloseBelowMinNet {
		t.Fatalf("%s", got)
	}
}

func TestBuildPriceDiffOppReview_Summary(t *testing.T) {
	t0 := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	peak := t0.Add(2 * time.Minute)
	fade := t0.Add(3 * time.Minute)
	closed := t0.Add(10 * time.Minute)
	o := PriceDiffOpportunity{
		Status:         PriceDiffOppClosed,
		OpenedAt:       t0,
		OpenNetDiffPct: 0.40,
		OpenProfit:     40,
		PeakNetDiffPct: 0.81,
		PeakAt:         peak,
		FadeStartedAt:  &fade,
		ClosedAt:       &closed,
		CloseReason:    PriceDiffCloseBelowMinProfit,
		CloseNetDiffPct: 0.08,
		NetDiffPct:     0.08,
	}
	rev := BuildPriceDiffOppReview(o, closed)
	if rev.OpenDurationSec < 599 || rev.OpenDurationSec > 601 {
		t.Fatalf("duration %v", rev.OpenDurationSec)
	}
	if rev.PeakDeltaPct < 0.40 || rev.FadeFromPeakSec == nil || *rev.FadeFromPeakSec < 59 {
		t.Fatalf("%+v", rev)
	}
	if !strings.Contains(rev.Summary, "Peaked at 0.810%") || !strings.Contains(rev.Summary, "min profit") {
		t.Fatalf("summary %q", rev.Summary)
	}
}
