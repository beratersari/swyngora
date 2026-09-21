package market

import (
	"context"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/feargreedstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestFearGreedInputsFromReports(t *testing.T) {
	breadth := &domain.BreadthReport{Windows: []domain.BreadthWindow{{
		Window: domain.BreadthWindow24h,
		Counts: domain.BreadthCounts{Total: 80, UpPct: 62},
		BTC:    &domain.CoinMove{Symbol: "BTCUSDT", ChangePct: 3.2, Known: true},
	}}}
	vol := &domain.VolatilityReport{Windows: []domain.VolWindow{{
		Window:     domain.VolWindow24h,
		Coin:       domain.VolMeasure{RangePct: 4, NetPct: 3, Complete: true},
		TypicalPct: 2,
	}}}
	fund := &domain.FundingSnapshot{Pressure: &domain.FundingPressure{RatePct: "0.01", Payer: "long"}}
	ls := &domain.LongShortSnapshot{Current: &domain.LongShortLevel{LongPct: "65"}}
	liq := &domain.LiquidationOverview{Windows: []domain.LiquidationWindowTotals{{
		Window: domain.LiquidationWindow24h, LongNotional: "30", ShortNotional: "70", Complete: true,
	}}}
	book := &domain.PerpDepthOverview{Known: 20, Total: 100, BidPct: 58}
	taker := &domain.TakerFlowReport{Combined: &domain.TakerVenueFlow{Windows: []domain.TakerWindowFlow{{
		Window: domain.TakerWindow1h, BuyNotional: 60, SellNotional: 40, BuyShare: 0.6,
	}}}}
	in := fearGreedInputsFrom(breadth, vol, fund, ls, liq, book, taker)
	if !in.HasBreadth || in.BreadthUpPct != 62 || !in.HasMomentum || in.BTCChangePct != 3.2 {
		t.Fatalf("breadth/momentum %+v", in)
	}
	if !in.HasVol || in.VolRatio < 1.9 || in.VolRatio > 2.1 {
		t.Fatalf("vol %+v", in)
	}
	if !in.HasFunding || in.FundingRatePct != 0.01 || in.FundingPayer != "long" {
		t.Fatalf("funding %+v", in)
	}
	if !in.HasLong || in.LongPct != 65 || !in.HasLiq || in.LiqShort != 70 || !in.HasBook || in.BidPct != 58 {
		t.Fatalf("positioning %+v", in)
	}
	if !in.HasTaker || in.TakerBuyPct != 60 || in.TakerWindow != "1h" {
		t.Fatalf("taker %+v", in)
	}
}

func TestFearGreedInputsSkipThinLiquidations(t *testing.T) {
	in := fearGreedInputsFrom(nil, nil, nil, nil, &domain.LiquidationOverview{Windows: []domain.LiquidationWindowTotals{{
		Window: domain.LiquidationWindow24h, LongNotional: "0", ShortNotional: "10", Complete: false,
	}}}, nil, nil)
	if in.HasLiq {
		t.Fatal("a partial liquidation window must not be labeled as 24h")
	}
}

func TestRememberFearGreedStoresScore(t *testing.T) {
	mem := feargreedstore.NewMemory()
	svc := New(nil, nil).WithFearGreedHistory(mem)
	now := time.Date(2026, 9, 21, 12, 7, 0, 0, time.UTC)
	rep := domain.BuildFearGreed(domain.FearGreedInputs{
		HasBreadth: true, BreadthUpPct: 80,
		HasMomentum: true, BTCChangePct: 4,
		HasFunding: true, FundingRatePct: 0.03, FundingPayer: "long",
	}, now)
	if rep.Used < domain.FearGreedMinFactors {
		t.Fatalf("used %d", rep.Used)
	}
	svc.rememberFearGreed(context.Background(), rep)
	if len(rep.History) != 1 || rep.History[0].Score != rep.Score {
		t.Fatalf("history %+v", rep.History)
	}
	if !rep.History[0].Time.Equal(domain.FearGreedSampleTime(now)) {
		t.Fatalf("bucket %s", rep.History[0].Time)
	}
	rows, err := mem.ListFearGreed(context.Background(), now.Add(-time.Hour))
	if err != nil || len(rows) != 1 {
		t.Fatalf("%+v %v", rows, err)
	}
}
