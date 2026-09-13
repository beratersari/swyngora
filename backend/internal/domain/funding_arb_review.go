package domain

import (
	"fmt"
	"math"
	"time"
)

const (
	FundingArbReviewPending = "pending"
	FundingArbReviewSettled = "settled"
	FundingArbReviewMissing = "missing"
	fundingArbReviewSlack   = 2 * time.Minute
)

// FundingArbExpectedPayment is one predicted settlement at signal open.
type FundingArbExpectedPayment struct {
	Time     time.Time `json:"time"`
	Exchange Exchange  `json:"exchange"`
	Rate     float64   `json:"rate"`
	Amount   float64   `json:"amount"`
}

// FundingArbExpected is the quote we showed when the opportunity opened.
type FundingArbExpected struct {
	Notional       float64                     `json:"notional"`
	HoldHours      float64                     `json:"holdHours"`
	LongExchange   Exchange                    `json:"longExchange"`
	ShortExchange  Exchange                    `json:"shortExchange"`
	HorizonFunding float64                     `json:"horizonFunding"`
	RoundTripFee   float64                     `json:"roundTripFee"`
	ExpectedNet    float64                     `json:"expectedNet"`
	Payments       []FundingArbExpectedPayment `json:"payments"`
}

// FundingArbReviewPayment is one predicted clock vs the settled print.
type FundingArbReviewPayment struct {
	Time           time.Time
	Exchange       Exchange
	ExpectedRate   float64
	ActualRate     float64
	ExpectedAmount float64
	ActualAmount   float64
	Status         string
}

// FundingArbReview compares the predicted schedule to settled funding.
type FundingArbReview struct {
	AsOf            time.Time
	PendingCount    int
	SettledCount    int
	MissingCount    int
	Complete        bool
	ExpectedFunding float64
	ActualFunding   float64
	ExpectedNet     float64
	ActualNet       float64
	DeltaNet        float64
	StillWorthIt    bool
	Payments        []FundingArbReviewPayment
	Summary         string
}

// LastExpectedClock is the last predicted settlement, or zero.
func (e *FundingArbExpected) LastExpectedClock() time.Time {
	if e == nil {
		return time.Time{}
	}
	var last time.Time
	for _, p := range e.Payments {
		if last.IsZero() || p.Time.After(last) {
			last = p.Time
		}
	}
	return last
}

// FundingArbExpectedFromRaw snapshots the live trade math at open.
func FundingArbExpectedFromRaw(raw fundingArbTradeRaw) *FundingArbExpected {
	if len(raw.pays) == 0 {
		return nil
	}
	pays := make([]FundingArbExpectedPayment, 0, len(raw.pays))
	for _, p := range raw.pays {
		pays = append(pays, FundingArbExpectedPayment{
			Time: p.Time.UTC(), Exchange: p.Exchange, Rate: p.Rate, Amount: p.Amount,
		})
	}
	return &FundingArbExpected{
		Notional: raw.notional, HoldHours: raw.holdHours,
		LongExchange: raw.long.Exchange, ShortExchange: raw.short.Exchange,
		HorizonFunding: raw.horizonAmt, RoundTripFee: raw.rtFee,
		ExpectedNet: raw.netHorizon, Payments: pays,
	}
}

// BuildFundingArbReview scores settled prints against the predicted schedule.
// Clocks after now are pending. A clock that has passed with no print is missing.
func BuildFundingArbReview(exp *FundingArbExpected, binance, bybit []FundingPoint, now time.Time) *FundingArbReview {
	if exp == nil || len(exp.Payments) == 0 {
		return nil
	}
	now = now.UTC()
	out := &FundingArbReview{
		AsOf:            now,
		ExpectedFunding: exp.HorizonFunding,
		ExpectedNet:     exp.ExpectedNet,
		Payments:        make([]FundingArbReviewPayment, 0, len(exp.Payments)),
	}
	var settledExpected, settledActual float64
	for _, p := range exp.Payments {
		row := FundingArbReviewPayment{
			Time: p.Time.UTC(), Exchange: p.Exchange,
			ExpectedRate: p.Rate, ExpectedAmount: p.Amount,
			Status: FundingArbReviewPending,
		}
		if !p.Time.After(now) {
			pts := binance
			if p.Exchange == ExchangeBybit {
				pts = bybit
			}
			if pt, ok := findSettledFundingPrint(pts, p.Time); ok {
				row.Status = FundingArbReviewSettled
				row.ActualRate = pt.Rate
				row.ActualAmount = fundingArbSideAmount(exp.LongExchange, exp.ShortExchange, p.Exchange, exp.Notional, pt.Rate)
				settledExpected += p.Amount
				settledActual += row.ActualAmount
				out.SettledCount++
			} else {
				row.Status = FundingArbReviewMissing
				out.MissingCount++
			}
		} else {
			out.PendingCount++
		}
		out.Payments = append(out.Payments, row)
	}
	out.Complete = out.PendingCount == 0
	if out.Complete {
		out.ActualFunding = settledActual
		out.ActualNet = settledActual - exp.RoundTripFee
		out.DeltaNet = out.ActualNet - exp.ExpectedNet
		out.StillWorthIt = out.ActualNet > fundingArbWorthEps
	} else {
		out.ActualFunding = settledActual
		out.DeltaNet = settledActual - settledExpected
	}
	out.Summary = explainFundingArbReview(out, exp, settledExpected, settledActual)
	return out
}

func findSettledFundingPrint(pts []FundingPoint, want time.Time) (FundingPoint, bool) {
	want = want.UTC().Truncate(time.Minute)
	var best FundingPoint
	bestDist := time.Duration(math.MaxInt64)
	found := false
	for _, p := range pts {
		if p.Predicted || p.Time.IsZero() {
			continue
		}
		at := p.Time.UTC().Truncate(time.Minute)
		d := at.Sub(want)
		if d < 0 {
			d = -d
		}
		if d <= fundingArbReviewSlack && d < bestDist {
			best, bestDist, found = p, d, true
		}
	}
	return best, found
}

func fundingArbSideAmount(long, short, pay Exchange, notional, rate float64) float64 {
	if pay == long {
		return -notional * rate
	}
	if pay == short {
		return notional * rate
	}
	return 0
}

func explainFundingArbReview(r *FundingArbReview, exp *FundingArbExpected, settledExp, settledAct float64) string {
	if r == nil || exp == nil {
		return ""
	}
	n := len(exp.Payments)
	if r.Complete {
		if r.MissingCount == n {
			return "The hold window has passed, but no settled funding prints were found for those clocks."
		}
		verb := "was still profitable"
		if !r.StillWorthIt {
			verb = "was not profitable after fees"
		}
		return fmt.Sprintf("After %d settlement(s), actual net was %s versus expected %s (%s). The trade %s.",
			r.SettledCount, FormatSignedQty(r.ActualNet), FormatSignedQty(r.ExpectedNet),
			FormatSignedQty(r.DeltaNet), verb)
	}
	done := r.SettledCount + r.MissingCount
	return fmt.Sprintf("%d of %d predicted clocks have passed. On settled prints, funding was %s versus expected %s (%s). %d clock(s) still pending.",
		done, n, FormatSignedQty(settledAct), FormatSignedQty(settledExp),
		FormatSignedQty(settledAct-settledExp), r.PendingCount)
}
