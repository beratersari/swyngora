package domain

import (
	"fmt"
	"sort"
	"time"
)

// FundingArbHistoryPayment is one settled print inside a past run.
type FundingArbHistoryPayment struct {
	Time          string `json:"time"`
	Exchange      string `json:"exchange"`
	Rate          string `json:"rate"`
	RatePct       string `json:"ratePct"`
	Amount        string `json:"amount"`
	LongExchange  string `json:"longExchange"`
	ShortExchange string `json:"shortExchange"`
}

// FundingArbHistoryRun is one stretch of the same long/short pair that
// finished after-fee positive on settled funding.
type FundingArbHistoryRun struct {
	StartedAt     string                     `json:"startedAt"`
	EndedAt       string                     `json:"endedAt"`
	DurationHours string                     `json:"durationHours"`
	LongExchange  string                     `json:"longExchange"`
	ShortExchange string                     `json:"shortExchange"`
	Payments      []FundingArbHistoryPayment `json:"payments"`
	PaymentCount  int                        `json:"paymentCount"`
	FundingAmount string                     `json:"fundingAmount"`
	RoundTripFees string                     `json:"roundTripFees"`
	NetAfterFees  string                     `json:"netAfterFees"`
	Summary       string                     `json:"summary"`
	net           float64
}

// FundingArbHistoryReport lists past profitable runs for one coin.
type FundingArbHistoryReport struct {
	Symbol              string                 `json:"symbol"`
	Notional            string                 `json:"notional"`
	From                time.Time              `json:"from"`
	To                  time.Time              `json:"to"`
	Runs                []FundingArbHistoryRun `json:"runs"`
	SkippedUnprofitable int                    `json:"skippedUnprofitable"`
	Summary             string                 `json:"summary"`
	Note                string                 `json:"note"`
}

// ResolveFundingArbRange requires from < to and a span of at most 30 days.
func ResolveFundingArbRange(from, to time.Time) (time.Time, time.Time, error) {
	if from.IsZero() || to.IsZero() {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: from and to are required", ErrInvalidArgument)
	}
	from = from.UTC()
	to = to.UTC()
	if !to.After(from) {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: to must be after from", ErrInvalidArgument)
	}
	if to.Sub(from) > MaxFundingArbHistory {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: range must be <= 30 days", ErrInvalidArgument)
	}
	return from, to, nil
}

type fundingHistEv struct {
	Time     time.Time
	Exchange Exchange
	Rate     float64
}

// BuildFundingArbHistory walks settled Binance and Bybit prints and keeps
// only after-fee winning runs of a stable long/short pair.
func BuildFundingArbHistory(symbol string, binance, bybit []FundingPoint, notional, feeBinance, feeBybit float64, from, to time.Time) *FundingArbHistoryReport {
	symbol = NormalizeLiquidationSymbol(symbol)
	from, to = from.UTC(), to.UTC()
	rtFee := notional * (feeBinance + feeBybit) * 2
	out := &FundingArbHistoryReport{
		Symbol:   symbol,
		Notional: formatQty(notional),
		From:     from,
		To:       to,
		Runs:     []FundingArbHistoryRun{},
		Note:     FundingArbDisclaimer,
	}
	evs := mergeFundingHist(binance, bybit, from, to)
	if len(evs) == 0 {
		out.Summary = "No settled Binance or Bybit funding payments in that range."
		return out
	}

	type rawRun struct {
		long, short Exchange
		start, end  time.Time
		funding     float64
		pays        []FundingArbHistoryPayment
	}
	var (
		last = map[Exchange]float64{}
		have = map[Exchange]bool{}
		cur  *rawRun
		skip int
		runs = []FundingArbHistoryRun{}
	)
	collect := func(run *rawRun, batch []fundingHistEv, long, short Exchange) {
		if run == nil {
			return
		}
		for _, e := range batch {
			amt := 0.0
			switch e.Exchange {
			case long:
				amt = -notional * e.Rate
			case short:
				amt = notional * e.Rate
			default:
				continue
			}
			dec, pct := FormatFundingRate(e.Rate)
			run.pays = append(run.pays, FundingArbHistoryPayment{
				Time: e.Time.UTC().Format(time.RFC3339Nano), Exchange: string(e.Exchange),
				Rate: dec, RatePct: pct, Amount: FormatSignedQty(amt),
				LongExchange: string(long), ShortExchange: string(short),
			})
			run.funding += amt
			run.end = e.Time
		}
	}
	flush := func() {
		if cur == nil {
			return
		}
		net := cur.funding - rtFee
		if net <= fundingArbWorthEps {
			skip++
			cur = nil
			return
		}
		dur := cur.end.Sub(cur.start).Hours()
		view := FundingArbHistoryRun{
			StartedAt:     cur.start.UTC().Format(time.RFC3339Nano),
			EndedAt:       cur.end.UTC().Format(time.RFC3339Nano),
			DurationHours: formatFixed(dur, 1),
			LongExchange:  string(cur.long),
			ShortExchange: string(cur.short),
			Payments:      cur.pays,
			PaymentCount:  len(cur.pays),
			FundingAmount: FormatSignedQty(cur.funding),
			RoundTripFees: formatQty(rtFee),
			NetAfterFees:  FormatSignedQty(net),
			net:           net,
		}
		view.Summary = fmt.Sprintf("Long %s, short %s from %s to %s (%s h, %d payments). Settled funding %s minus round-trip fees %s = %s.",
			fundingArbVenueName(cur.long), fundingArbVenueName(cur.short),
			cur.start.UTC().Format(time.RFC3339), cur.end.UTC().Format(time.RFC3339),
			view.DurationHours, len(cur.pays), view.FundingAmount, view.RoundTripFees, view.NetAfterFees)
		runs = append(runs, view)
		cur = nil
	}

	for i := 0; i < len(evs); {
		t := evs[i].Time
		j := i
		for j < len(evs) && evs[j].Time.Equal(t) {
			have[evs[j].Exchange] = true
			last[evs[j].Exchange] = evs[j].Rate
			j++
		}
		batch := evs[i:j]
		i = j
		if !have[ExchangeBinance] || !have[ExchangeBybit] {
			continue
		}
		long, short := pickHistSides(last)
		if long == "" || short == "" || long == short {
			continue
		}
		if cur != nil && (cur.long != long || cur.short != short) {
			// Direction flips at this settlement. The old position was already
			// open, so this clock still pays the old sides, then the run ends.
			collect(cur, batch, cur.long, cur.short)
			flush()
		}
		if cur == nil {
			// First clock of a run is the signal to enter. The position is not
			// open before that payment, so that print is not collected profit.
			cur = &rawRun{long: long, short: short, start: t, end: t}
			continue
		}
		collect(cur, batch, long, short)
	}
	flush()

	sort.SliceStable(runs, func(i, j int) bool {
		if runs[i].net != runs[j].net {
			return runs[i].net > runs[j].net
		}
		return runs[i].StartedAt < runs[j].StartedAt
	})
	out.Runs = runs
	out.SkippedUnprofitable = skip
	if len(runs) == 0 {
		out.Summary = fmt.Sprintf("No after-fee winning stretch in that range (%d unprofitable run(s) skipped).", skip)
		return out
	}
	top := runs[0]
	out.Summary = fmt.Sprintf("%d after-fee winning stretch(es). Best: %s", len(runs), top.Summary)
	return out
}

func pickHistSides(last map[Exchange]float64) (long, short Exchange) {
	br, bok := last[ExchangeBinance]
	yr, yok := last[ExchangeBybit]
	if !bok || !yok {
		return "", ""
	}
	if yr > br {
		return ExchangeBinance, ExchangeBybit
	}
	if br > yr {
		return ExchangeBybit, ExchangeBinance
	}
	return "", ""
}

func mergeFundingHist(binance, bybit []FundingPoint, from, to time.Time) []fundingHistEv {
	out := make([]fundingHistEv, 0, len(binance)+len(bybit))
	add := func(ex Exchange, pts []FundingPoint) {
		for _, p := range pts {
			t := p.Time.UTC()
			if t.IsZero() || t.Before(from) || t.After(to) {
				continue
			}
			out = append(out, fundingHistEv{Time: t, Exchange: ex, Rate: p.Rate})
		}
	}
	add(ExchangeBinance, binance)
	add(ExchangeBybit, bybit)
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].Time.Equal(out[j].Time) {
			return out[i].Time.Before(out[j].Time)
		}
		return string(out[i].Exchange) < string(out[j].Exchange)
	})
	return out
}
