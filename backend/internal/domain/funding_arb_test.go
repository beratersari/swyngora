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

func TestFundingClocksInWindow_DocsSchedule(t *testing.T) {
	// 8h clocks from 16:00: 16:00, 00:00, 08:00 — official Binance/Bybit 8h times.
	from := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	next := time.Date(2026, 8, 26, 16, 0, 0, 0, time.UTC)
	got := FundingClocksInWindow(next, 8, from, to)
	if len(got) != 3 || !got[0].Equal(next) || got[1].Hour() != 0 || got[2].Hour() != 8 {
		t.Fatalf("%v", got)
	}
	// Hold that ends before 16:00 — no payment.
	if n := FundingClocksInWindow(next, 8, from, from.Add(3*time.Hour)); len(n) != 0 {
		t.Fatalf("expected none %v", n)
	}
	if FundingClocksInWindow(time.Time{}, 8, from, to) != nil {
		t.Fatal("missing next")
	}
}

func TestBuildFundingArbReport_LongCheapShortRich(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	next := now.Add(time.Hour)
	bin := FundingArbVenueInput{
		Exchange: ExchangeBinance, FundingRate: 0.0001, IntervalHours: 8,
		NextFundingTime: next, PerpLast: 100, PerpMark: 100, SpotIndex: 100,
		FeeRate: 0.001,
	}
	byb := FundingArbVenueInput{
		Exchange: ExchangeBybit, FundingRate: 0.0003, IntervalHours: 8,
		NextFundingTime: next, PerpLast: 100.2, PerpMark: 100.2, SpotIndex: 100,
		FeeRate: 0.001,
	}
	got := BuildFundingArbReport("btcusdt", []FundingArbVenueInput{byb, bin}, 10_000, 24, now)
	if got.Symbol != "BTCUSDT" {
		t.Fatalf("%+v", got)
	}
	// 3 clocks × 2 venues, each 8h from 13:00. Fees 40 beat funding 6 — not an opportunity.
	if got.Trade != nil {
		t.Fatalf("losing trade must not be an opportunity %+v", got.Trade)
	}
	if !strings.Contains(got.Summary, "Not an opportunity") {
		t.Fatalf("summary %s", got.Summary)
	}
	if len(got.Payments) != 6 {
		t.Fatalf("want 6 venue clocks, got %d", len(got.Payments))
	}
	if len(got.Carry) != 0 {
		t.Fatalf("losing carry must be hidden %+v", got.Carry)
	}
}

func TestBuildFundingArbReport_NoClockInWindowIsZero(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	next := time.Date(2026, 8, 26, 16, 0, 0, 0, time.UTC)
	got := BuildFundingArbReport("BTCUSDT", []FundingArbVenueInput{
		{Exchange: ExchangeBinance, FundingRate: 0.001, IntervalHours: 8, NextFundingTime: next, FeeRate: 0},
		{Exchange: ExchangeBybit, FundingRate: 0.002, IntervalHours: 8, NextFundingTime: next, FeeRate: 0},
	}, 10_000, 2, now)
	if got.Trade != nil {
		t.Fatalf("no clock in 2h window %+v", got.Trade)
	}
	if len(got.Payments) != 0 || !strings.Contains(got.Summary, "no") {
		t.Fatalf("%s pays=%d", got.Summary, len(got.Payments))
	}
}

func TestBuildFundingArbReport_NegativeFundingSides(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	next := now.Add(time.Hour)
	bin := FundingArbVenueInput{
		Exchange: ExchangeBinance, FundingRate: -0.0004, IntervalHours: 8,
		NextFundingTime: next, PerpLast: 99.5, SpotIndex: 100, FeeRate: 0,
	}
	byb := FundingArbVenueInput{
		Exchange: ExchangeBybit, FundingRate: -0.0001, IntervalHours: 8,
		NextFundingTime: next, PerpLast: 99.8, SpotIndex: 100, FeeRate: 0,
	}
	got := BuildFundingArbReport("ETHUSDT", []FundingArbVenueInput{bin, byb}, 5_000, 8, now)
	if got.Trade == nil {
		t.Fatal("trade")
	}
	if got.Trade.LongExchange != "binance" || got.Trade.ShortExchange != "bybit" {
		t.Fatalf("sides %+v", got.Trade)
	}
	// One 8h clock in an 8h window. Long pays -5000*(-0.0004)=+2; short pays 5000*(-0.0001)=-0.5; net +1.5
	if !strings.Contains(got.Trade.NextFundingAmount, "1.5") && got.Trade.NextFundingAmount != "+1.50" {
		t.Fatalf("next %s", got.Trade.NextFundingAmount)
	}
}

func TestBuildFundingArbReport_WorthItWithLowFees(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	next := now.Add(time.Hour)
	bin := FundingArbVenueInput{
		Exchange: ExchangeBinance, FundingRate: 0.00005, IntervalHours: 8,
		NextFundingTime: next, PerpLast: 100, SpotIndex: 100, FeeRate: 0.0001,
	}
	byb := FundingArbVenueInput{
		Exchange: ExchangeBybit, FundingRate: 0.00105, IntervalHours: 8,
		NextFundingTime: next, PerpLast: 100, SpotIndex: 100, FeeRate: 0.0001,
	}
	// 3 clocks × spread 0.001 = 30. RT fees 4. Net 26.
	got := BuildFundingArbReport("SOLUSDT", []FundingArbVenueInput{bin, byb}, 10_000, 24, now)
	if got.Trade == nil || !got.Trade.WorthIt {
		t.Fatalf("should cover fees %+v %s", got.Trade, got.Summary)
	}
	if !strings.Contains(got.Trade.HorizonFundingAmount, "30") {
		t.Fatalf("horizon %s", got.Trade.HorizonFundingAmount)
	}
	if got.Trade.PaymentCount != 6 {
		t.Fatalf("clocks %d", got.Trade.PaymentCount)
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
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	next := now.Add(time.Hour)
	a := BuildFundingArbReport("AAAUSDT", []FundingArbVenueInput{
		{Exchange: ExchangeBinance, FundingRate: 0, IntervalHours: 8, NextFundingTime: next, FeeRate: 0.0001, PerpLast: 1, SpotIndex: 1},
		{Exchange: ExchangeBybit, FundingRate: 0.001, IntervalHours: 8, NextFundingTime: next, FeeRate: 0.0001, PerpLast: 1, SpotIndex: 1},
	}, 10_000, 24, now)
	b := BuildFundingArbReport("BBBUSDT", []FundingArbVenueInput{
		{Exchange: ExchangeBinance, FundingRate: 0, IntervalHours: 8, NextFundingTime: next, FeeRate: 0.001, PerpLast: 1, SpotIndex: 1},
		{Exchange: ExchangeBybit, FundingRate: 0.0002, IntervalHours: 8, NextFundingTime: next, FeeRate: 0.001, PerpLast: 1, SpotIndex: 1},
	}, 10_000, 24, now)
	ha, ok := FundingArbHitFromReport(a, 1)
	if !ok || ha.LongExchange != "binance" {
		t.Fatalf("%+v %v summary=%s", ha, ok, a.Summary)
	}
	if _, ok := FundingArbHitFromReport(b, -1); ok {
		t.Fatal("losing row must not be an opportunity")
	}
	hits := []FundingArbHit{ha}
	SortFundingArbHits(hits)
	if hits[0].Symbol != "AAAUSDT" {
		t.Fatalf("rank %+v", hits)
	}
}

func TestBuildFundingArbReport_UnevenIntervals(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	next := now.Add(time.Hour)
	// Bybit 0.0008 at 13:00, 21:00, 05:00. Binance rate 0 on 4h clocks. On 1000 = 2.4
	got := BuildFundingArbReport("BTCUSDT", []FundingArbVenueInput{
		{Exchange: ExchangeBinance, FundingRate: 0, IntervalHours: 4, NextFundingTime: next, FeeRate: 0, PerpLast: 10, SpotIndex: 10},
		{Exchange: ExchangeBybit, FundingRate: 0.0008, IntervalHours: 8, NextFundingTime: next, FeeRate: 0, PerpLast: 10, SpotIndex: 10},
	}, 1_000, 24, now)
	if got.Trade == nil {
		t.Fatal("trade")
	}
	if got.Trade.HorizonFundingAmount != "+2.4" && got.Trade.HorizonFundingAmount != "+2.40" {
		t.Fatalf("horizon %s pays=%d", got.Trade.HorizonFundingAmount, got.Trade.PaymentCount)
	}
}

func TestResolveFundingArbRange(t *testing.T) {
	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(7 * 24 * time.Hour)
	gotFrom, gotTo, err := ResolveFundingArbRange(from, to)
	if err != nil || !gotFrom.Equal(from) || !gotTo.Equal(to) {
		t.Fatalf("%v %v %v", gotFrom, gotTo, err)
	}
	if _, _, err := ResolveFundingArbRange(time.Time{}, to); err == nil {
		t.Fatal("empty")
	}
	if _, _, err := ResolveFundingArbRange(to, from); err == nil {
		t.Fatal("order")
	}
	if _, _, err := ResolveFundingArbRange(from, from.Add(31*24*time.Hour)); err == nil {
		t.Fatal("span")
	}
}

func TestBuildFundingArbHistory_WinningRun(t *testing.T) {
	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	bin := []FundingPoint{
		{Time: from.Add(8 * time.Hour), Rate: 0.0001},
		{Time: from.Add(16 * time.Hour), Rate: 0.0001},
	}
	byb := []FundingPoint{
		{Time: from.Add(8 * time.Hour), Rate: 0.002},
		{Time: from.Add(16 * time.Hour), Rate: 0.002},
	}
	// 08:00 is the entry signal (not collected). 16:00 is the first hold:
	// 10000 × 0.0019 = 19. Fees 4. Net 15.
	got := BuildFundingArbHistory("BTCUSDT", bin, byb, 10_000, 0.0001, 0.0001, from, to)
	if len(got.Runs) != 1 {
		t.Fatalf("%+v", got)
	}
	run := got.Runs[0]
	if run.LongExchange != "binance" || run.ShortExchange != "bybit" {
		t.Fatalf("sides %+v", run)
	}
	if run.PaymentCount != 2 {
		t.Fatalf("pays %d (08:00 must not be collected)", run.PaymentCount)
	}
	if !strings.Contains(run.StartedAt, "T08:00") {
		t.Fatalf("startedAt should be the 08:00 entry signal, got %s", run.StartedAt)
	}
	if len(run.Payments) == 0 || !strings.Contains(run.Payments[0].Time, "T16:00") {
		t.Fatalf("first collected payment must be 16:00, got %+v", run.Payments)
	}
	if run.DurationHours != "8" && run.DurationHours != "8.0" {
		t.Fatalf("duration %s", run.DurationHours)
	}
	if !strings.Contains(run.NetAfterFees, "15") {
		t.Fatalf("net %s fund=%s", run.NetAfterFees, run.FundingAmount)
	}
}

func TestResolveFundingArbMinProfit(t *testing.T) {
	got, err := ResolveFundingArbMinProfit(5)
	if err != nil || got != 5 {
		t.Fatalf("%v %v", got, err)
	}
	if _, err := ResolveFundingArbMinProfit(0); err == nil {
		t.Fatal("zero")
	}
	if _, err := ResolveFundingArbMinProfit(-1); err == nil {
		t.Fatal("neg")
	}
	if _, err := ResolveFundingArbMinProfit(math.NaN()); err == nil {
		t.Fatal("nan")
	}
	if _, err := ResolveFundingArbMinProfit(MaxFundingArbMinProfit + 1); err == nil {
		t.Fatal("max")
	}
}

func TestBuildFundingArbHistory_SameClockNotProfit(t *testing.T) {
	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	at := from.Add(8 * time.Hour)
	got := BuildFundingArbHistory("BTCUSDT",
		[]FundingPoint{{Time: at, Rate: 0.0001}},
		[]FundingPoint{{Time: at, Rate: 0.05}},
		10_000, 0, 0, from, to)
	if len(got.Runs) != 0 {
		t.Fatalf("08:00 start=end must not collect that payment %+v", got.Runs)
	}
}

func TestBuildFundingArbHistory_SkipLosersAndFlip(t *testing.T) {
	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(3 * 24 * time.Hour)
	bin := []FundingPoint{
		{Time: from.Add(8 * time.Hour), Rate: 0.0001},
		{Time: from.Add(16 * time.Hour), Rate: 0.003},
	}
	byb := []FundingPoint{
		{Time: from.Add(8 * time.Hour), Rate: 0.00012},
		{Time: from.Add(16 * time.Hour), Rate: 0.0001},
	}
	// First print pair: tiny spread, will lose after 40 of default-like fees.
	// Second print: long bybit / short binance, one print of 10000*(0.003-0.0001)=29, still lose vs 40.
	got := BuildFundingArbHistory("ETHUSDT", bin, byb, 10_000, 0.001, 0.001, from, to)
	if len(got.Runs) != 0 || got.SkippedUnprofitable < 1 {
		t.Fatalf("%+v", got)
	}
}
