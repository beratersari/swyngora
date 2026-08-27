package market

import (
	"context"
	"strings"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func fundingArbSvc() *Service {
	now := time.Now().UTC()
	binFund := &fakeFunding{ser: &domain.FundingSeries{
		Exchange: domain.ExchangeBinance, Current: domain.FundingPoint{Time: now, Rate: 0.0001, Predicted: true},
		NextFundingTime: now.Add(time.Hour), IntervalHours: 8,
		History: []domain.FundingPoint{{Time: now.Add(-8 * time.Hour), Rate: 0.00008}},
	}}
	bybFund := &fakeFunding{ser: &domain.FundingSeries{
		Exchange: domain.ExchangeBybit, Current: domain.FundingPoint{Time: now, Rate: 0.0004, Predicted: true},
		NextFundingTime: now.Add(time.Hour), IntervalHours: 8,
		History: []domain.FundingPoint{{Time: now.Add(-8 * time.Hour), Rate: 0.0003}},
	}}
	binBasis := &fakeBasis{q: &domain.BasisQuote{
		Exchange: domain.ExchangeBinance, FuturesLast: 100, FuturesMark: 100, SpotIndex: 100,
	}}
	bybBasis := &fakeBasis{q: &domain.BasisQuote{
		Exchange: domain.ExchangeBybit, FuturesLast: 100.15, FuturesMark: 100.14, SpotIndex: 100,
	}}
	return NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{},
		domain.ExchangeBybit:   &fakeMarket{},
	}, &fakeSupply{}).
		WithFundingRate(map[domain.Exchange]domain.FundingRatePort{
			domain.ExchangeBinance: binFund, domain.ExchangeBybit: bybFund,
		}).
		WithBasis(map[domain.Exchange]domain.BasisPort{
			domain.ExchangeBinance: binBasis, domain.ExchangeBybit: bybBasis,
		})
}

func TestGetFundingArb_PicksSidesAndSizes(t *testing.T) {
	svc := fundingArbSvc()
	fee := 0.02
	got, err := svc.GetFundingArb(context.Background(), FundingArbParams{
		Symbol: "btc-usd", Notional: 10_000, HoldHours: 24,
		FeeBinancePct: &fee, FeeBybitPct: &fee,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Symbol != "BTCUSDT" || got.Trade == nil {
		t.Fatalf("%+v", got)
	}
	if got.Trade.LongExchange != "binance" || got.Trade.ShortExchange != "bybit" {
		t.Fatalf("sides %+v", got.Trade)
	}
	if !strings.Contains(got.Trade.NextFundingAmount, "3") {
		t.Fatalf("next funding 10000*(0.0004-0.0001)=3 got %s", got.Trade.NextFundingAmount)
	}
	if got.Trade.LongPerp == "" || got.Trade.ShortPerp == "" {
		t.Fatalf("missing perp prices %+v", got.Trade)
	}
	if got.Summary == "" || !strings.Contains(got.Note, "not financial advice") {
		t.Fatalf("copy %+v", got)
	}
}

func TestGetFundingArb_MissingSymbol(t *testing.T) {
	_, err := fundingArbSvc().GetFundingArb(context.Background(), FundingArbParams{})
	if err == nil {
		t.Fatal("expected symbol error")
	}
}

func TestGetFundingArb_BadNotional(t *testing.T) {
	_, err := fundingArbSvc().GetFundingArb(context.Background(), FundingArbParams{Symbol: "BTCUSDT", Notional: -5})
	if err == nil {
		t.Fatal("expected notional error")
	}
}

func TestGetFundingArb_DefaultFeesAndNotional(t *testing.T) {
	got, err := fundingArbSvc().GetFundingArb(context.Background(), FundingArbParams{Symbol: "ETHUSDT"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Notional == "" || got.HoldHours == "" || got.Trade != nil {
		t.Fatalf("default 0.10%% fees must not surface a losing trade %+v", got)
	}
	if !strings.Contains(got.Summary, "Not an opportunity") {
		t.Fatalf("summary %s", got.Summary)
	}
}

func TestScanFundingArb_RanksHits(t *testing.T) {
	fee := 0.01
	got, err := fundingArbSvc().ScanFundingArb(context.Background(), FundingArbScanParams{
		SymbolLimit: 5, FeeBinancePct: &fee, FeeBybitPct: &fee,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Hits) == 0 {
		t.Fatalf("expected hits %+v", got)
	}
	for _, h := range got.Hits {
		if !h.WorthIt || h.LongExchange == "" || h.ShortExchange == "" || h.Symbol == "" {
			t.Fatalf("hit %+v", h)
		}
	}
}

func TestScanFundingArb_HidesLosers(t *testing.T) {
	got, err := fundingArbSvc().ScanFundingArb(context.Background(), FundingArbScanParams{SymbolLimit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Hits) != 0 {
		t.Fatalf("default fees must hide losers %+v", got.Hits)
	}
}

func TestGetFundingArbHistory(t *testing.T) {
	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	bin := &fakeFunding{hist: []domain.FundingPoint{
		{Time: from.Add(8 * time.Hour), Rate: 0.0001},
		{Time: from.Add(16 * time.Hour), Rate: 0.0001},
	}}
	byb := &fakeFunding{hist: []domain.FundingPoint{
		{Time: from.Add(8 * time.Hour), Rate: 0.002},
		{Time: from.Add(16 * time.Hour), Rate: 0.002},
	}}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{},
		domain.ExchangeBybit:   &fakeMarket{},
	}, &fakeSupply{}).WithFundingRate(map[domain.Exchange]domain.FundingRatePort{
		domain.ExchangeBinance: bin, domain.ExchangeBybit: byb,
	})
	fee := 0.01
	got, err := svc.GetFundingArbHistory(context.Background(), FundingArbHistoryParams{
		Symbol: "BTCUSDT", From: from, To: to, Notional: 10_000,
		FeeBinancePct: &fee, FeeBybitPct: &fee,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Runs) != 1 || got.Runs[0].LongExchange != "binance" {
		t.Fatalf("%+v", got)
	}
}

func TestScanFundingArb_BadFee(t *testing.T) {
	bad := 12.0
	_, err := fundingArbSvc().ScanFundingArb(context.Background(), FundingArbScanParams{FeeBinancePct: &bad})
	if err == nil {
		t.Fatal("expected fee error")
	}
}
