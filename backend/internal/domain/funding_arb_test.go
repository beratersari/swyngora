package domain

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestResolveFundingArbInputs(t *testing.T) {
	n, err := ResolveFundingArbNotional(0)
	if err != nil || n != DefaultFundingArbNotional {
		t.Fatalf("default notional %v %v", n, err)
	}
	if _, err := ResolveFundingArbNotional(-1); err == nil {
		t.Fatal("neg notional")
	}
	if _, err := ResolveFundingArbNotional(MaxFundingArbNotional + 1); err == nil {
		t.Fatal("huge notional")
	}
	h, err := ResolveFundingArbHoldHours(0)
	if err != nil || h != DefaultFundingArbHoldHours {
		t.Fatalf("default hold %v %v", h, err)
	}
	if _, err := ResolveFundingArbHoldHours(MaxFundingArbHoldHours + 1); err == nil {
		t.Fatal("huge hold")
	}
	def, err := ResolveFundingArbFeeRate(ExchangeBinance, nil)
	if err != nil || math.Abs(def-BinanceFeeRate) > 1e-12 {
		t.Fatalf("default fee %v %v", def, err)
	}
	zero := 0.0
	got, err := ResolveFundingArbFeeRate(ExchangeBinance, &zero)
	if err != nil || got != 0 {
		t.Fatalf("explicit zero %v %v", got, err)
	}
	pct := 0.02
	got, err = ResolveFundingArbFeeRate(ExchangeBybit, &pct)
	if err != nil || math.Abs(got-0.0002) > 1e-12 {
		t.Fatalf("override %v %v", got, err)
	}
	bad := 9.0
	if _, err := ResolveFundingArbFeeRate(ExchangeBinance, &bad); err == nil {
		t.Fatal("fee cap")
	}
}

func TestClampFundingArbScanLimit(t *testing.T) {
	if ClampFundingArbScanLimit(0) != FundingArbScanDefault {
		t.Fatal("default")
	}
	if ClampFundingArbScanLimit(99) != FundingArbScanMax {
		t.Fatal("max")
	}
	if ClampFundingArbScanLimit(8) != 8 {
		t.Fatal("pass")
	}
}

func TestBuildFundingArbReport_LongCheapShortRich(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	bin := FundingArbVenueInput{
		Exchange: ExchangeBinance, FundingRate: 0.0001, IntervalHours: 8,
		NextFundingTime: now.Add(time.Hour), PerpLast: 100, PerpMark: 100, SpotIndex: 100,
		FeeRate: 0.001,
	}
	byb := FundingArbVenueInput{
		Exchange: ExchangeBybit, FundingRate: 0.0003, IntervalHours: 8,
		NextFundingTime: now.Add(time.Hour), PerpLast: 100.2, PerpMark: 100.2, SpotIndex: 100,
		FeeRate: 0.001,
	}
	got := BuildFundingArbReport("btcusdt", []FundingArbVenueInput{byb, bin}, 10_000, 24, now)
	if got.Symbol != "BTCUSDT" || got.Trade == nil {
		t.Fatalf("%+v", got)
	}
	if got.Trade.LongExchange != "binance" || got.Trade.ShortExchange != "bybit" {
		t.Fatalf("sides %+v", got.Trade)
	}
	if !strings.Contains(got.Trade.Title, "Long Binance") || !strings.Contains(got.Trade.Title, "short Bybit") {
		t.Fatalf("title %s", got.Trade.Title)
	}
	// One settlement: 10000 * (0.0003-0.0001) = 2
	if got.Trade.NextFundingAmount != "+2" && got.Trade.NextFundingAmount != "+2.00" {
		t.Fatalf("next %s", got.Trade.NextFundingAmount)
	}
	// 3 settlements in 24h at 8h: 6
	if got.Trade.HorizonFundingAmount != "+6" && got.Trade.HorizonFundingAmount != "+6.00" {
		t.Fatalf("horizon %s", got.Trade.HorizonFundingAmount)
	}
	// Open 0.1%+0.1% = 20; RT = 40
	if got.Trade.OpenFeeAmount != "20" && got.Trade.OpenFeeAmount != "20.00" {
		t.Fatalf("open fee %s", got.Trade.OpenFeeAmount)
	}
	if got.Trade.RoundTripFeeAmount != "40" && got.Trade.RoundTripFeeAmount != "40.00" {
		t.Fatalf("rt fee %s", got.Trade.RoundTripFeeAmount)
	}
	if got.Trade.WorthIt {
		t.Fatal("24h funding 6 does not cover 40 fees")
	}
	if got.Trade.BreakEvenSettlements == "" {
		t.Fatal("need break-even settlements")
	}
	// Bybit richer: short the premium. gap ~0.2%
	if !strings.Contains(got.Trade.PerpGapPct, "+") {
		t.Fatalf("expected favorable perp gap %s", got.Trade.PerpGapPct)
	}
	if len(got.Carry) != 2 {
		t.Fatalf("carry %+v", got.Carry)
	}
	for _, c := range got.Carry {
		if c.PerpSide != "short" || c.SpotSide != "long" {
			t.Fatalf("positive funding should short perp %+v", c)
		}
	}
}

func TestBuildFundingArbReport_NegativeFundingFlipsCarry(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	bin := FundingArbVenueInput{
		Exchange: ExchangeBinance, FundingRate: -0.0004, IntervalHours: 8,
		PerpLast: 99.5, SpotIndex: 100, FeeRate: 0.0002,
	}
	byb := FundingArbVenueInput{
		Exchange: ExchangeBybit, FundingRate: -0.0001, IntervalHours: 8,
		PerpLast: 99.8, SpotIndex: 100, FeeRate: 0.0002,
	}
	got := BuildFundingArbReport("ETHUSDT", []FundingArbVenueInput{bin, byb}, 5_000, 8, now)
	if got.Trade == nil {
		t.Fatal("trade")
	}
	// Long more-negative (binance -0.04%), short less-negative (bybit -0.01%)
	if got.Trade.LongExchange != "binance" || got.Trade.ShortExchange != "bybit" {
		t.Fatalf("sides %+v", got.Trade)
	}
	// Spread 0.0003 * 5000 = 1.5 next
	if !strings.Contains(got.Trade.NextFundingAmount, "1.5") && got.Trade.NextFundingAmount != "+1.50" {
		t.Fatalf("next %s", got.Trade.NextFundingAmount)
	}
	for _, c := range got.Carry {
		if c.PerpSide != "long" || c.SpotSide != "short" {
			t.Fatalf("negative funding should long perp %+v", c)
		}
	}
}

func TestBuildFundingArbReport_WorthItWithLowFees(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	bin := FundingArbVenueInput{
		Exchange: ExchangeBinance, FundingRate: 0.00005, IntervalHours: 8,
		PerpLast: 100, SpotIndex: 100, FeeRate: 0.0001,
	}
	byb := FundingArbVenueInput{
		Exchange: ExchangeBybit, FundingRate: 0.00105, IntervalHours: 8,
		PerpLast: 100, SpotIndex: 100, FeeRate: 0.0001,
	}
	// spread 0.1%/8h → 0.3%/day. On 10000 = 30. RT fees 4. Net 26.
	got := BuildFundingArbReport("SOLUSDT", []FundingArbVenueInput{bin, byb}, 10_000, 24, now)
	if got.Trade == nil || !got.Trade.WorthIt {
		t.Fatalf("should cover fees %+v", got.Trade)
	}
	if !strings.Contains(got.Trade.HorizonFundingAmount, "30") {
		t.Fatalf("horizon %s", got.Trade.HorizonFundingAmount)
	}
}

func TestBuildFundingArbReport_MissingVenue(t *testing.T) {
	got := BuildFundingArbReport("BTCUSDT", []FundingArbVenueInput{
		{Exchange: ExchangeBinance, FundingRate: 0.0001, FeeRate: 0.001},
		{Exchange: ExchangeBybit, Error: "timeout"},
	}, 10_000, 24, time.Now().UTC())
	if got.Trade != nil {
		t.Fatalf("no trade: %+v", got.Trade)
	}
	if !strings.Contains(got.Summary, "both") {
		t.Fatalf("summary %s", got.Summary)
	}
}

func TestBuildFundingArbReport_SameRate(t *testing.T) {
	got := BuildFundingArbReport("BTCUSDT", []FundingArbVenueInput{
		{Exchange: ExchangeBinance, FundingRate: 0.0001, FeeRate: 0.001, PerpLast: 100, SpotIndex: 100},
		{Exchange: ExchangeBybit, FundingRate: 0.0001, FeeRate: 0.001, PerpLast: 100, SpotIndex: 100},
	}, 10_000, 24, time.Now().UTC())
	if got.Trade != nil {
		t.Fatal("flat spread")
	}
	if !strings.Contains(got.Summary, "same") {
		t.Fatalf("summary %s", got.Summary)
	}
}

func TestFundingArbHitFromReportAndSort(t *testing.T) {
	now := time.Now().UTC()
	a := BuildFundingArbReport("AAAUSDT", []FundingArbVenueInput{
		{Exchange: ExchangeBinance, FundingRate: 0, IntervalHours: 8, FeeRate: 0.0001, PerpLast: 1, SpotIndex: 1},
		{Exchange: ExchangeBybit, FundingRate: 0.001, IntervalHours: 8, FeeRate: 0.0001, PerpLast: 1, SpotIndex: 1},
	}, 10_000, 24, now)
	b := BuildFundingArbReport("BBBUSDT", []FundingArbVenueInput{
		{Exchange: ExchangeBinance, FundingRate: 0, IntervalHours: 8, FeeRate: 0.001, PerpLast: 1, SpotIndex: 1},
		{Exchange: ExchangeBybit, FundingRate: 0.0002, IntervalHours: 8, FeeRate: 0.001, PerpLast: 1, SpotIndex: 1},
	}, 10_000, 24, now)
	ha, ok := FundingArbHitFromReport(a, 1)
	if !ok || ha.LongExchange != "binance" {
		t.Fatalf("%+v %v", ha, ok)
	}
	hb, ok := FundingArbHitFromReport(b, -1)
	if !ok {
		t.Fatal("b")
	}
	hits := []FundingArbHit{hb, ha}
	SortFundingArbHits(hits)
	if hits[0].Symbol != "AAAUSDT" {
		t.Fatalf("rank %+v", hits)
	}
}

func TestBuildFundingArbReport_UnevenIntervals(t *testing.T) {
	now := time.Now().UTC()
	// 0.0008 every 8h vs 0 every 4h → daily 0.0008*3 = 0.0024. On 1000 = 2.4 / 24h.
	got := BuildFundingArbReport("BTCUSDT", []FundingArbVenueInput{
		{Exchange: ExchangeBinance, FundingRate: 0, IntervalHours: 4, FeeRate: 0, PerpLast: 10, SpotIndex: 10},
		{Exchange: ExchangeBybit, FundingRate: 0.0008, IntervalHours: 8, FeeRate: 0, PerpLast: 10, SpotIndex: 10},
	}, 1_000, 24, now)
	if got.Trade == nil {
		t.Fatal("trade")
	}
	if got.Trade.HorizonFundingAmount != "+2.4" && got.Trade.HorizonFundingAmount != "+2.40" {
		t.Fatalf("horizon %s", got.Trade.HorizonFundingAmount)
	}
}
