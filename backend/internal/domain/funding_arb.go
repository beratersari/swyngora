package domain

import (
	"fmt"
	"math"
	"sort"
	"time"
)

const (
	DefaultFundingArbNotional  = 10_000.0
	MaxFundingArbNotional      = 10_000_000.0
	DefaultFundingArbHoldHours = 24.0
	MaxFundingArbHoldHours     = 24.0 * 30
	FundingArbScanDefault      = 15
	FundingArbScanMax          = 40
	fundingArbFeePctCap        = 5.0
	fundingArbFlatRate         = 1e-12
	fundingArbWorthEps         = 1e-9
)

// FundingArbDisclaimer is returned on every quote and scan.
const FundingArbDisclaimer = "Cross-exchange perpetual funding: long the cheaper-funding venue and short the richer one. Earnings use the predicted next rate on the notional you enter. Fees are public taker-style paper defaults unless you override them (percent). Spot vs perpetual is shown per venue and as a possible extra if both perps move back to their own spot — that extra is not guaranteed. Predicted rates can change before settlement. Informational only — not financial advice and not an executable trade."

// FundingArbVenueInput is one venue's live funding + prices + fee.
type FundingArbVenueInput struct {
	Exchange        Exchange
	FundingRate     float64
	AvgLast3        float64
	HasAvgLast3     bool
	IntervalHours   int
	NextFundingTime time.Time
	PerpLast        float64
	PerpMark        float64
	SpotIndex       float64
	SpotLast        float64
	FeeRate         float64
	Error           string
}

// FundingArbVenueView is one venue in the API payload.
type FundingArbVenueView struct {
	Exchange        string `json:"exchange"`
	FundingRate     string `json:"fundingRate"`
	FundingRatePct  string `json:"fundingRatePct"`
	Payer           string `json:"payer"`
	AvgLast3        string `json:"avgLast3,omitempty"`
	AvgLast3Pct     string `json:"avgLast3Pct,omitempty"`
	IntervalHours   int    `json:"intervalHours"`
	NextFundingTime string `json:"nextFundingTime,omitempty"`
	PerpLast        string `json:"perpLast,omitempty"`
	PerpMark        string `json:"perpMark,omitempty"`
	Spot            string `json:"spot,omitempty"`
	SpotSource      string `json:"spotSource,omitempty"`
	Basis           string `json:"basis,omitempty"`
	BasisPct        string `json:"basisPct,omitempty"`
	BasisKind       string `json:"basisKind,omitempty"`
	FeePct          string `json:"feePct"`
	Error           string `json:"error,omitempty"`
}

// FundingArbTradeView is the recommended long/short pair and sized payout.
type FundingArbTradeView struct {
	LongExchange               string `json:"longExchange"`
	ShortExchange              string `json:"shortExchange"`
	LongRate                   string `json:"longRate"`
	LongRatePct                string `json:"longRatePct"`
	ShortRate                  string `json:"shortRate"`
	ShortRatePct               string `json:"shortRatePct"`
	SpreadPct                  string `json:"spreadPct"`
	SpreadPerDayPct            string `json:"spreadPerDayPct"`
	LongPerp                   string `json:"longPerp,omitempty"`
	ShortPerp                  string `json:"shortPerp,omitempty"`
	PerpGapPct                 string `json:"perpGapPct,omitempty"`
	PerpGapAmount              string `json:"perpGapAmount,omitempty"`
	LongBasisPct               string `json:"longBasisPct,omitempty"`
	ShortBasisPct              string `json:"shortBasisPct,omitempty"`
	LongSpot                   string `json:"longSpot,omitempty"`
	ShortSpot                  string `json:"shortSpot,omitempty"`
	OpenFeeAmount              string `json:"openFeeAmount"`
	OpenFeePct                 string `json:"openFeePct"`
	RoundTripFeeAmount         string `json:"roundTripFeeAmount"`
	RoundTripFeePct            string `json:"roundTripFeePct"`
	Notional                   string `json:"notional"`
	HoldHours                  string `json:"holdHours"`
	NextFundingAmount          string `json:"nextFundingAmount"`
	HorizonFundingAmount       string `json:"horizonFundingAmount"`
	NetNextAfterOpenFees       string `json:"netNextAfterOpenFees"`
	NetNextAfterRoundTrip      string `json:"netNextAfterRoundTrip"`
	NetHorizonAfterRoundTrip   string `json:"netHorizonAfterRoundTrip"`
	NetHorizonIfBasisConverges string `json:"netHorizonIfBasisConverges"`
	BreakEvenSettlements       string `json:"breakEvenSettlements,omitempty"`
	BreakEvenHoldHours         string `json:"breakEvenHoldHours,omitempty"`
	WorthIt                    bool   `json:"worthIt"`
	Title                      string `json:"title"`
	Summary                    string `json:"summary"`
}

// FundingArbCarryView is same-venue cash-and-carry (spot vs perp).
type FundingArbCarryView struct {
	Exchange              string `json:"exchange"`
	PerpSide              string `json:"perpSide"`
	SpotSide              string `json:"spotSide"`
	FundingRatePct        string `json:"fundingRatePct"`
	BasisPct              string `json:"basisPct,omitempty"`
	NextFundingAmount     string `json:"nextFundingAmount"`
	OpenFeeAmount         string `json:"openFeeAmount"`
	RoundTripFeeAmount    string `json:"roundTripFeeAmount"`
	BasisCaptureAmount    string `json:"basisCaptureAmount,omitempty"`
	NetNextAfterRoundTrip string `json:"netNextAfterRoundTrip"`
	Title                 string `json:"title"`
	Summary               string `json:"summary"`
}

// FundingArbReport is one coin's cross-venue funding opportunity.
type FundingArbReport struct {
	Symbol    string                `json:"symbol"`
	Notional  string                `json:"notional"`
	HoldHours string                `json:"holdHours"`
	AsOf      time.Time             `json:"asOf"`
	Venues    []FundingArbVenueView `json:"venues"`
	Trade     *FundingArbTradeView  `json:"trade,omitempty"`
	Carry     []FundingArbCarryView `json:"carry,omitempty"`
	Summary   string                `json:"summary"`
	Note      string                `json:"note"`
	// HorizonNet is the raw after-fee hold-window payout used to rank scans.
	HorizonNet float64 `json:"-"`
}

// FundingArbHit is one ranked scan row.
type FundingArbHit struct {
	Symbol                   string `json:"symbol"`
	LongExchange             string `json:"longExchange"`
	ShortExchange            string `json:"shortExchange"`
	SpreadPct                string `json:"spreadPct"`
	HorizonFundingAmount     string `json:"horizonFundingAmount"`
	NetHorizonAfterRoundTrip string `json:"netHorizonAfterRoundTrip"`
	WorthIt                  bool   `json:"worthIt"`
	Summary                  string `json:"summary"`
	// RankScore is the raw horizon net used to sort (not shown on the wire).
	RankScore float64 `json:"-"`
}

// FundingArbScan ranks liquid coins by after-fee funding over the hold window.
type FundingArbScan struct {
	Notional    string          `json:"notional"`
	HoldHours   string          `json:"holdHours"`
	SymbolLimit int             `json:"symbolLimit"`
	AsOf        time.Time       `json:"asOf"`
	Hits        []FundingArbHit `json:"hits"`
	Skipped     int             `json:"skipped"`
	Note        string          `json:"note"`
}

// ResolveFundingArbNotional defaults empty size and rejects junk.
func ResolveFundingArbNotional(n float64) (float64, error) {
	if math.IsNaN(n) || math.IsInf(n, 0) || n < 0 {
		return 0, fmt.Errorf("%w: notional must be a finite number >= 0", ErrInvalidArgument)
	}
	if n == 0 {
		return DefaultFundingArbNotional, nil
	}
	if n > MaxFundingArbNotional {
		return 0, fmt.Errorf("%w: notional must be <= %s", ErrInvalidArgument, formatQty(MaxFundingArbNotional))
	}
	return n, nil
}

// ResolveFundingArbHoldHours defaults empty hold time and rejects junk.
func ResolveFundingArbHoldHours(h float64) (float64, error) {
	if math.IsNaN(h) || math.IsInf(h, 0) || h < 0 {
		return 0, fmt.Errorf("%w: holdHours must be a finite number >= 0", ErrInvalidArgument)
	}
	if h == 0 {
		return DefaultFundingArbHoldHours, nil
	}
	if h > MaxFundingArbHoldHours {
		return 0, fmt.Errorf("%w: holdHours must be <= %g", ErrInvalidArgument, MaxFundingArbHoldHours)
	}
	return h, nil
}

// ResolveFundingArbFeeRate turns a percent override into a fraction.
// nil means the paper taker default for that venue.
func ResolveFundingArbFeeRate(ex Exchange, pct *float64) (float64, error) {
	if pct == nil {
		return TradingCostFor(ex).FeeRate, nil
	}
	if math.IsNaN(*pct) || math.IsInf(*pct, 0) || *pct < 0 || *pct > fundingArbFeePctCap {
		return 0, fmt.Errorf("%w: fee must be between 0 and %g percent", ErrInvalidArgument, fundingArbFeePctCap)
	}
	return *pct / 100, nil
}

// ClampFundingArbScanLimit bounds how many top-volume coins a scan walks.
func ClampFundingArbScanLimit(n int) int {
	if n <= 0 {
		return FundingArbScanDefault
	}
	if n > FundingArbScanMax {
		return FundingArbScanMax
	}
	return n
}

func fundingArbPerp(in FundingArbVenueInput) float64 {
	if in.PerpLast > 0 && !math.IsNaN(in.PerpLast) {
		return in.PerpLast
	}
	if in.PerpMark > 0 && !math.IsNaN(in.PerpMark) {
		return in.PerpMark
	}
	return 0
}

func fundingArbSpot(in FundingArbVenueInput) (price float64, source string) {
	if in.SpotIndex > 0 && !math.IsNaN(in.SpotIndex) {
		return in.SpotIndex, "index"
	}
	if in.SpotLast > 0 && !math.IsNaN(in.SpotLast) {
		return in.SpotLast, "last"
	}
	return 0, ""
}

func fundingArbInterval(in FundingArbVenueInput) int {
	if in.IntervalHours >= 1 && in.IntervalHours <= 24 {
		return in.IntervalHours
	}
	return DefaultFundingIntervalHrs
}

func fundingArbHourlyRate(rate float64, intervalHours int) float64 {
	if intervalHours < 1 {
		intervalHours = DefaultFundingIntervalHrs
	}
	return rate / float64(intervalHours)
}

// BuildFundingArbVenueView formats one venue for the API.
func BuildFundingArbVenueView(in FundingArbVenueInput) FundingArbVenueView {
	dec, pct := FormatFundingRate(in.FundingRate)
	out := FundingArbVenueView{
		Exchange:       string(in.Exchange),
		FundingRate:    dec,
		FundingRatePct: pct,
		Payer:          FundingPayer(in.FundingRate),
		IntervalHours:  fundingArbInterval(in),
		FeePct:         formatFixed(in.FeeRate*100, 4),
		Error:          in.Error,
	}
	if in.HasAvgLast3 {
		ad, ap := FormatFundingRate(in.AvgLast3)
		out.AvgLast3, out.AvgLast3Pct = ad, ap
	}
	if !in.NextFundingTime.IsZero() {
		out.NextFundingTime = in.NextFundingTime.UTC().Format(time.RFC3339Nano)
	}
	if in.PerpLast > 0 {
		out.PerpLast = formatQty(in.PerpLast)
	}
	if in.PerpMark > 0 {
		out.PerpMark = formatQty(in.PerpMark)
	}
	spot, src := fundingArbSpot(in)
	if spot > 0 {
		out.Spot = formatQty(spot)
		out.SpotSource = src
	}
	perp := fundingArbPerp(in)
	if perp > 0 && spot > 0 {
		d, bp, kind := ComputeBasis(perp, spot)
		out.Basis = FormatSignedQty(d)
		out.BasisPct = FormatSignedPct(bp)
		out.BasisKind = kind
	}
	return out
}

func pickFundingArbPair(legs []FundingArbVenueInput) (long, short FundingArbVenueInput, ok bool) {
	usable := make([]FundingArbVenueInput, 0, len(legs))
	for _, v := range legs {
		if v.Error != "" || math.IsNaN(v.FundingRate) {
			continue
		}
		usable = append(usable, v)
	}
	if len(usable) < 2 {
		return FundingArbVenueInput{}, FundingArbVenueInput{}, false
	}
	sort.SliceStable(usable, func(i, j int) bool {
		if usable[i].FundingRate != usable[j].FundingRate {
			return usable[i].FundingRate < usable[j].FundingRate
		}
		return string(usable[i].Exchange) < string(usable[j].Exchange)
	})
	return usable[0], usable[len(usable)-1], true
}

type fundingArbTradeRaw struct {
	long, short                        FundingArbVenueInput
	notional, holdHours                float64
	spread, dailySpread                float64
	openFee, rtFee                     float64
	nextAmt, horizonAmt                float64
	perpGapPct, perpGapAmt             float64
	longBasisPct, shortBasisPct        float64
	netBasisAmt                        float64
	netNextOpen, netNextRT, netHorizon float64
	netHorizonConverge                 float64
	breakEvenN, breakEvenH             float64
	hasPerp, hasSpot                   bool
	worthIt                            bool
}

func computeFundingArbTrade(long, short FundingArbVenueInput, notional, holdHours float64) fundingArbTradeRaw {
	out := fundingArbTradeRaw{long: long, short: short, notional: notional, holdHours: holdHours}
	li, si := fundingArbInterval(long), fundingArbInterval(short)
	out.spread = short.FundingRate - long.FundingRate
	out.dailySpread = fundingArbHourlyRate(short.FundingRate, si)*24 - fundingArbHourlyRate(long.FundingRate, li)*24
	out.nextAmt = notional * out.spread
	out.horizonAmt = notional * (fundingArbHourlyRate(short.FundingRate, si)*holdHours - fundingArbHourlyRate(long.FundingRate, li)*holdHours)
	out.openFee = notional * (long.FeeRate + short.FeeRate)
	out.rtFee = out.openFee * 2
	out.netNextOpen = out.nextAmt - out.openFee
	out.netNextRT = out.nextAmt - out.rtFee
	out.netHorizon = out.horizonAmt - out.rtFee

	lp, sp := fundingArbPerp(long), fundingArbPerp(short)
	if lp > 0 && sp > 0 {
		out.hasPerp = true
		mid := (lp + sp) / 2
		if mid > 0 {
			out.perpGapPct = (sp - lp) / mid * 100
			out.perpGapAmt = notional * out.perpGapPct / 100
		}
	}
	ls, _ := fundingArbSpot(long)
	ss, _ := fundingArbSpot(short)
	if lp > 0 && ls > 0 {
		_, out.longBasisPct, _ = ComputeBasis(lp, ls)
		out.hasSpot = true
	}
	if sp > 0 && ss > 0 {
		_, out.shortBasisPct, _ = ComputeBasis(sp, ss)
		out.hasSpot = true
	}
	if out.hasSpot {
		out.netBasisAmt = notional * (out.shortBasisPct - out.longBasisPct) / 100
	}
	out.netHorizonConverge = out.netHorizon + out.netBasisAmt
	if out.nextAmt > fundingArbWorthEps {
		out.breakEvenN = out.rtFee / out.nextAmt
	}
	hourly := out.horizonAmt / holdHours
	if holdHours > 0 && hourly > fundingArbWorthEps {
		out.breakEvenH = out.rtFee / hourly
	}
	out.worthIt = out.netHorizon > fundingArbWorthEps
	return out
}

func formatFundingArbTrade(raw fundingArbTradeRaw) FundingArbTradeView {
	lDec, lPct := FormatFundingRate(raw.long.FundingRate)
	sDec, sPct := FormatFundingRate(raw.short.FundingRate)
	_, spreadPct := FormatFundingRate(raw.spread)
	view := FundingArbTradeView{
		LongExchange:               string(raw.long.Exchange),
		ShortExchange:              string(raw.short.Exchange),
		LongRate:                   lDec,
		LongRatePct:                lPct,
		ShortRate:                  sDec,
		ShortRatePct:               sPct,
		SpreadPct:                  spreadPct,
		SpreadPerDayPct:            formatFixed(raw.dailySpread*100, 6),
		OpenFeeAmount:              formatQty(raw.openFee),
		OpenFeePct:                 formatFixed((raw.long.FeeRate+raw.short.FeeRate)*100, 4),
		RoundTripFeeAmount:         formatQty(raw.rtFee),
		RoundTripFeePct:            formatFixed((raw.long.FeeRate+raw.short.FeeRate)*200, 4),
		Notional:                   formatQty(raw.notional),
		HoldHours:                  formatFixed(raw.holdHours, 1),
		NextFundingAmount:          FormatSignedQty(raw.nextAmt),
		HorizonFundingAmount:       FormatSignedQty(raw.horizonAmt),
		NetNextAfterOpenFees:       FormatSignedQty(raw.netNextOpen),
		NetNextAfterRoundTrip:      FormatSignedQty(raw.netNextRT),
		NetHorizonAfterRoundTrip:   FormatSignedQty(raw.netHorizon),
		NetHorizonIfBasisConverges: FormatSignedQty(raw.netHorizonConverge),
		WorthIt:                    raw.worthIt,
	}
	if raw.hasPerp {
		view.LongPerp = formatQty(fundingArbPerp(raw.long))
		view.ShortPerp = formatQty(fundingArbPerp(raw.short))
		view.PerpGapPct = FormatSignedPct(raw.perpGapPct)
		view.PerpGapAmount = FormatSignedQty(raw.perpGapAmt)
	}
	ls, _ := fundingArbSpot(raw.long)
	ss, _ := fundingArbSpot(raw.short)
	if ls > 0 {
		view.LongSpot = formatQty(ls)
		view.LongBasisPct = FormatSignedPct(raw.longBasisPct)
	}
	if ss > 0 {
		view.ShortSpot = formatQty(ss)
		view.ShortBasisPct = FormatSignedPct(raw.shortBasisPct)
	}
	if raw.breakEvenN > 0 {
		view.BreakEvenSettlements = formatFixed(raw.breakEvenN, 1)
	}
	if raw.breakEvenH > 0 {
		view.BreakEvenHoldHours = formatFixed(raw.breakEvenH, 1)
	}
	view.Title = fmt.Sprintf("Long %s, short %s", fundingArbVenueName(raw.long.Exchange), fundingArbVenueName(raw.short.Exchange))
	view.Summary = explainFundingArbTrade(raw)
	return view
}

func fundingArbVenueName(ex Exchange) string {
	switch ex {
	case ExchangeBinance:
		return "Binance"
	case ExchangeBybit:
		return "Bybit"
	case ExchangeCoinbase:
		return "Coinbase"
	default:
		if ex == "" {
			return "unknown"
		}
		return string(ex)
	}
}

func explainFundingArbTrade(raw fundingArbTradeRaw) string {
	longN, shortN := fundingArbVenueName(raw.long.Exchange), fundingArbVenueName(raw.short.Exchange)
	_, lPct := FormatFundingRate(raw.long.FundingRate)
	_, sPct := FormatFundingRate(raw.short.FundingRate)
	_, spPct := FormatFundingRate(raw.spread)
	head := fmt.Sprintf("Long %s (%s%%) and short %s (%s%%). Funding spread %s%% per settlement (~%s%%/day).",
		longN, lPct, shortN, sPct, spPct, formatFixed(raw.dailySpread*100, 4))
	head += fmt.Sprintf(" On %s notional, the next funding on both legs is about %s.",
		formatQty(raw.notional), FormatSignedQty(raw.nextAmt))
	head += fmt.Sprintf(" Opening both sides costs about %s in taker fees; round-trip (open and close) is about %s.",
		formatQty(raw.openFee), formatQty(raw.rtFee))
	head += fmt.Sprintf(" After %s hours of funding minus round-trip fees: %s.",
		formatFixed(raw.holdHours, 1), FormatSignedQty(raw.netHorizon))
	if raw.breakEvenN > 0 && !raw.worthIt {
		head += fmt.Sprintf(" Need about %s settlements (~%s hours) of the same spread to cover those fees.",
			formatFixed(raw.breakEvenN, 1), formatFixed(raw.breakEvenH, 1))
	} else if raw.worthIt {
		head += " That hold window covers the fees at the predicted rates."
	}
	if raw.hasPerp && math.Abs(raw.perpGapPct) >= 0.005 {
		if raw.perpGapPct > 0 {
			head += fmt.Sprintf(" %s perp is richer than %s by %s%% (~%s on this size) — shorting the expensive book is extra if they meet.",
				shortN, longN, FormatSignedPct(raw.perpGapPct), FormatSignedQty(raw.perpGapAmt))
		} else {
			head += fmt.Sprintf(" %s perp is cheaper than %s by %s%% (~%s) — the funding trade is fighting the price gap.",
				shortN, longN, FormatSignedPct(-raw.perpGapPct), FormatSignedQty(-raw.perpGapAmt))
		}
	}
	if raw.hasSpot && math.Abs(raw.shortBasisPct-raw.longBasisPct) >= 0.005 {
		head += fmt.Sprintf(" If both perps go back to their own spot, that gap is about %s extra (not guaranteed).",
			FormatSignedQty(raw.netBasisAmt))
	}
	head += " Informational only — not financial advice."
	return head
}

func buildFundingArbCarry(in FundingArbVenueInput, notional float64) (FundingArbCarryView, bool) {
	if in.Error != "" || math.IsNaN(in.FundingRate) || math.Abs(in.FundingRate) <= fundingArbFlatRate {
		return FundingArbCarryView{}, false
	}
	perp := fundingArbPerp(in)
	spot, _ := fundingArbSpot(in)
	if perp <= 0 || spot <= 0 {
		return FundingArbCarryView{}, false
	}
	_, basisPct, _ := ComputeBasis(perp, spot)
	perpSide, spotSide := "short", "long"
	if in.FundingRate < 0 {
		perpSide, spotSide = "long", "short"
	}
	nextAmt := notional * math.Abs(in.FundingRate)
	openFee := notional * (in.FeeRate + in.FeeRate) // perp + spot
	rtFee := openFee * 2
	basisCap := notional * basisPct / 100
	if perpSide == "long" {
		basisCap = -basisCap
	}
	net := nextAmt - rtFee
	_, ratePct := FormatFundingRate(in.FundingRate)
	name := fundingArbVenueName(in.Exchange)
	title := fmt.Sprintf("%s %s perp, %s spot", name, perpSide, spotSide)
	summary := fmt.Sprintf("%s funding is %s%% so same-venue carry is %s the perpetual and %s spot. Next funding on %s is about %s; round-trip fees (spot+perp open and close) are about %s; net after those fees %s.",
		name, ratePct, perpSide, spotSide, formatQty(notional), FormatSignedQty(nextAmt), formatQty(rtFee), FormatSignedQty(net))
	if math.Abs(basisPct) >= 0.01 {
		summary += fmt.Sprintf(" Perp vs spot is %s%%; if they meet that is about %s extra (not guaranteed).",
			FormatSignedPct(basisPct), FormatSignedQty(basisCap))
	}
	return FundingArbCarryView{
		Exchange:              string(in.Exchange),
		PerpSide:              perpSide,
		SpotSide:              spotSide,
		FundingRatePct:        ratePct,
		BasisPct:              FormatSignedPct(basisPct),
		NextFundingAmount:     FormatSignedQty(nextAmt),
		OpenFeeAmount:         formatQty(openFee),
		RoundTripFeeAmount:    formatQty(rtFee),
		BasisCaptureAmount:    FormatSignedQty(basisCap),
		NetNextAfterRoundTrip: FormatSignedQty(net),
		Title:                 title,
		Summary:               summary,
	}, true
}

// BuildFundingArbReport folds venue inputs into the public quote.
func BuildFundingArbReport(symbol string, legs []FundingArbVenueInput, notional, holdHours float64, now time.Time) *FundingArbReport {
	symbol = NormalizeLiquidationSymbol(symbol)
	now = now.UTC()
	out := &FundingArbReport{
		Symbol:    symbol,
		Notional:  formatQty(notional),
		HoldHours: formatFixed(holdHours, 1),
		AsOf:      now,
		Venues:    make([]FundingArbVenueView, 0, len(legs)),
		Carry:     []FundingArbCarryView{},
		Note:      FundingArbDisclaimer,
	}
	sorted := append([]FundingArbVenueInput(nil), legs...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return string(sorted[i].Exchange) < string(sorted[j].Exchange)
	})
	for _, v := range sorted {
		out.Venues = append(out.Venues, BuildFundingArbVenueView(v))
		if c, ok := buildFundingArbCarry(v, notional); ok {
			out.Carry = append(out.Carry, c)
		}
	}
	long, short, ok := pickFundingArbPair(sorted)
	if !ok {
		out.Summary = "Need predicted funding on both Binance and Bybit to pick a long and a short venue."
		return out
	}
	if math.Abs(short.FundingRate-long.FundingRate) <= fundingArbFlatRate {
		out.Summary = "Binance and Bybit predicted funding are the same, so there is no funding spread to collect."
		return out
	}
	raw := computeFundingArbTrade(long, short, notional, holdHours)
	trade := formatFundingArbTrade(raw)
	out.Trade = &trade
	out.Summary = trade.Summary
	out.HorizonNet = raw.netHorizon
	return out
}

// NewFundingArbScan is an empty ranked result with resolved defaults.
func NewFundingArbScan(notional, holdHours float64, limit int, now time.Time) *FundingArbScan {
	return &FundingArbScan{
		Notional:    formatQty(notional),
		HoldHours:   formatFixed(holdHours, 1),
		SymbolLimit: ClampFundingArbScanLimit(limit),
		AsOf:        now.UTC(),
		Hits:        []FundingArbHit{},
		Note:        FundingArbDisclaimer,
	}
}

// FundingArbHitFromReport pulls the scan row from a quote.
func FundingArbHitFromReport(r *FundingArbReport, rank float64) (FundingArbHit, bool) {
	if r == nil || r.Trade == nil {
		return FundingArbHit{}, false
	}
	return FundingArbHit{
		Symbol:                   r.Symbol,
		LongExchange:             r.Trade.LongExchange,
		ShortExchange:            r.Trade.ShortExchange,
		SpreadPct:                r.Trade.SpreadPct,
		HorizonFundingAmount:     r.Trade.HorizonFundingAmount,
		NetHorizonAfterRoundTrip: r.Trade.NetHorizonAfterRoundTrip,
		WorthIt:                  r.Trade.WorthIt,
		Summary:                  r.Trade.Title + ". " + r.Trade.Summary,
		RankScore:                rank,
	}, true
}

// SortFundingArbHits ranks by after-fee horizon payout, then by symbol.
func SortFundingArbHits(hits []FundingArbHit) {
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].RankScore != hits[j].RankScore {
			return hits[i].RankScore > hits[j].RankScore
		}
		return hits[i].Symbol < hits[j].Symbol
	})
}
