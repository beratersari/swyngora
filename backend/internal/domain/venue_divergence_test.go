package domain

import (
	"testing"
	"time"
)

func TestCompareVenuePositioning_OppositeAlignment(t *testing.T) {
	now := time.Date(2026, 8, 13, 18, 0, 0, 0, time.UTC)
	bin := BuildPositioningVenue(PositioningInputs{
		Exchange: ExchangeBinance, Symbol: "ETHUSDT",
		Price4hPct: 1.5, OI4hPct: 2, LongShare: 0.66, FundingRate: 0.00025,
	})
	byb := BuildPositioningVenue(PositioningInputs{
		Exchange: ExchangeBybit, Symbol: "ETHUSDT",
		Price4hPct: -1.2, OI4hPct: 2, LongShare: 0.34, FundingRate: -0.0002,
	})
	got := CompareVenuePositioning("ETHUSDT", bin, byb, now)
	if got.Alignment != AlignOpposite {
		t.Fatalf("align %s summary=%s diffs=%+v", got.Alignment, got.Summary, got.Diffs)
	}
	if !got.Important || !HasMeaningfulDivergence(got) {
		t.Fatalf("%+v", got)
	}
	if got.BinanceLean != LeanBullish || got.BybitLean != LeanBearish {
		t.Fatalf("leans %s / %s", got.BinanceLean, got.BybitLean)
	}
	nImp := 0
	for _, d := range got.Diffs {
		if d.Important {
			nImp++
		}
		if d.WhyItMatters == "" {
			t.Fatalf("empty why %+v", d)
		}
	}
	if nImp < 2 {
		t.Fatalf("expected several important diffs: %+v", got.Diffs)
	}
}

func TestCompareVenuePositioning_Same(t *testing.T) {
	now := time.Date(2026, 8, 13, 18, 0, 0, 0, time.UTC)
	a := BuildPositioningVenue(PositioningInputs{
		Exchange: ExchangeBinance, Price4hPct: 1.2, OI4hPct: 2, LongShare: 0.6, FundingRate: 0.0001,
	})
	b := BuildPositioningVenue(PositioningInputs{
		Exchange: ExchangeBybit, Price4hPct: 1.0, OI4hPct: 1.5, LongShare: 0.58, FundingRate: 0.00008,
	})
	got := CompareVenuePositioning("BTCUSDT", a, b, now)
	if got.Alignment != AlignSame {
		t.Fatalf("%s %+v", got.Alignment, got)
	}
	if got.Important {
		t.Fatalf("same should not be important alert: %+v", got)
	}
}
