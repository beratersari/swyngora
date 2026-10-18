package domain

import (
	"math"
	"testing"
	"time"
)

func TestSlippageCash_BuyAndSell(t *testing.T) {
	if got := SlippageCash(TradeSideBuy, 100, 110, 1); math.Abs(got-10) > 1e-9 {
		t.Fatalf("buy %v", got)
	}
	if got := SlippageCash(TradeSideSell, 100, 90, 2); math.Abs(got-20) > 1e-9 {
		t.Fatalf("sell %v", got)
	}
	if got := SlippageCash(TradeSideBuy, 0, 110, 1); got != 0 {
		t.Fatalf("no touch %v", got)
	}
}

func TestExecutionQuote_SlippageCashUsesTheFill(t *testing.T) {
	q := ExecutionQuote{Status: ExecutionFilled, TouchPrice: 100, AveragePrice: 101, FilledQuantity: 2}
	if got := q.SlippageCash(TradeSideBuy); math.Abs(got-2) > 1e-9 {
		t.Fatalf("%v", got)
	}
	q.Status = ExecutionExhausted
	if got := q.SlippageCash(TradeSideBuy); got != 0 {
		t.Fatalf("unfilled %v", got)
	}
}

func TestQuoteFresh(t *testing.T) {
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	if !QuoteFresh(now.Add(-time.Minute), now) {
		t.Fatal("a minute ago is still the quote on screen")
	}
	if QuoteFresh(now.Add(-ExecutionQuoteTTL-time.Second), now) {
		t.Fatal("older than the window")
	}
	if QuoteFresh(now.Add(time.Minute), now) {
		t.Fatal("a future quote is not usable")
	}
}

func TestFillCostCheck_Deltas(t *testing.T) {
	c := FillCostCheck{ExpectedFee: 10, ActualFee: 10.4, ExpectedSlippage: 15, ActualSlippage: 12}
	if math.Abs(c.FeeDelta()-0.4) > 1e-9 || math.Abs(c.SlippageDelta()-(-3)) > 1e-9 {
		t.Fatalf("fee %v slip %v", c.FeeDelta(), c.SlippageDelta())
	}
}
