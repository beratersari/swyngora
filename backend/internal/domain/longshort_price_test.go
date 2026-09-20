package domain

import (
	"testing"
	"time"
)

func TestAttachLongShortPriceDivergence_UserExample(t *testing.T) {
	now := time.Date(2026, 9, 20, 16, 0, 0, 0, time.UTC)
	s := crowdTopSeries(now, 7, 0.55, 0.62, 0.005)
	candles := priceCandles(now, 7, 100, -0.25) // 6 steps of -0.25% ≈ -1.49%
	AttachLongShortGap(s)
	AttachLongShortPriceDivergence(s, candles)
	if s.Divergence == nil {
		t.Fatal("divergence")
	}
	d := s.Divergence
	if d.Kind != LongShortPriceDivTopLongPriceDown {
		t.Fatalf("kind %s", d.Kind)
	}
	if d.Duration != "30m" {
		t.Fatalf("duration %s", d.Duration)
	}
	if !d.Since.Equal(now.Add(-30 * time.Minute)) {
		t.Fatalf("since %s", d.Since)
	}
	if d.Headline == "" || d.Detail == "" {
		t.Fatalf("copy %+v", d)
	}
	if d.PriceChangePct == "" || d.PriceChangePct[0] != '-' {
		t.Fatalf("price %s", d.PriceChangePct)
	}
	if d.TopChangePct == "" || d.TopChangePct[0] != '+' {
		t.Fatalf("top %s", d.TopChangePct)
	}
}

func TestAttachLongShortPriceDivergence_TopShortPriceUp(t *testing.T) {
	now := time.Date(2026, 9, 20, 16, 0, 0, 0, time.UTC)
	s := crowdTopSeries(now, 7, 0.50, 0.48, -0.005)
	candles := priceCandles(now, 7, 100, 0.30)
	AttachLongShortGap(s)
	AttachLongShortPriceDivergence(s, candles)
	if s.Divergence == nil || s.Divergence.Kind != LongShortPriceDivTopShortPriceUp {
		t.Fatalf("%+v", s.Divergence)
	}
}

func TestAttachLongShortPriceDivergence_SameDirectionIsNotDivergent(t *testing.T) {
	now := time.Date(2026, 9, 20, 16, 0, 0, 0, time.UTC)
	s := crowdTopSeries(now, 7, 0.55, 0.62, 0.005)
	candles := priceCandles(now, 7, 100, 0.30)
	AttachLongShortGap(s)
	AttachLongShortPriceDivergence(s, candles)
	if s.Divergence != nil {
		t.Fatalf("same way %+v", s.Divergence)
	}
}

func TestAttachLongShortPriceDivergence_ShortBlipSkipped(t *testing.T) {
	now := time.Date(2026, 9, 20, 16, 0, 0, 0, time.UTC)
	s := crowdTopSeries(now, 2, 0.55, 0.62, 0.03)
	candles := priceCandles(now, 2, 100, -1.0)
	AttachLongShortGap(s)
	AttachLongShortPriceDivergence(s, candles)
	if s.Divergence != nil {
		t.Fatalf("5m only %+v", s.Divergence)
	}
}

func crowdTopSeries(end time.Time, n int, crowd, top0, topStep float64) *LongShortSeries {
	s := &LongShortSeries{Exchange: ExchangeBinance, Period: LongShortPeriod5m}
	top := &LongShortSeries{Kind: LongShortKindTopPositions, Period: LongShortPeriod5m}
	for i := 0; i < n; i++ {
		at := end.Add(-time.Duration(n-1-i) * 5 * time.Minute)
		c := LongShortPoint{Time: at, LongShare: crowd, ShortShare: 1 - crowd}
		tp := top0 + float64(i)*topStep
		p := LongShortPoint{Time: at, LongShare: tp, ShortShare: 1 - tp}
		if i == n-1 {
			s.Current = c
			top.Current = p
		} else {
			s.History = append(s.History, c)
			top.History = append(top.History, p)
		}
	}
	s.History = SortLongShortNewestFirst(s.History)
	top.History = SortLongShortNewestFirst(top.History)
	s.TopPositions = top
	return s
}

func priceCandles(end time.Time, n int, startPx, stepPct float64) []Candle {
	out := make([]Candle, 0, n)
	px := startPx
	for i := 0; i < n; i++ {
		if i > 0 {
			px *= 1 + stepPct/100
		}
		at := end.Add(-time.Duration(n-1-i) * 5 * time.Minute)
		out = append(out, Candle{
			OpenTime: at, CloseTime: at.Add(5*time.Minute - time.Millisecond),
			Open: formatFixed(px, 4), Close: formatFixed(px, 4),
		})
	}
	return out
}
