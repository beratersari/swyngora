package domain

import (
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	SituationCVDPriceUp   = CVDDivPriceUpCVDDown
	SituationCVDPriceDown = CVDDivPriceDownCVDUp
	SituationCVDAgrees    = CVDSplitAgrees

	SituationOIOpened = OpenInterestFlowOpened
	SituationOIClosed = OpenInterestFlowClosed
	SituationOIFlat   = OpenInterestFlowFlat

	// SituationLiqLong means the nearest estimated long-liquidation pocket
	// below last is closer than the nearest short pocket above last.
	SituationLiqLong = "long"
	// SituationLiqShort means the nearest short pocket above last is closer.
	SituationLiqShort = "short"
	// SituationLiqNone means the map is complete and neither side has a pocket.
	SituationLiqNone = "none"

	// Whale versus retail on the latest sized combined bar.
	SituationSizeWhaleBuyRetailSell = "whale_buy_retail_sell"
	SituationSizeWhaleSellRetailBuy = "whale_sell_retail_buy"
	SituationSizeBothBuy            = "both_buy"
	SituationSizeBothSell           = "both_sell"
	SituationSizeWhaleBuy           = "whale_buy"
	SituationSizeWhaleSell          = "whale_sell"
	SituationSizeRetailBuy          = "retail_buy"
	SituationSizeRetailSell         = "retail_sell"
	SituationSizeFlat               = "flat"

	situationNote = "Each clause stays blank until every venue that publishes it covers it. Funding, price versus CVD, open interest, the liquidation side, and whale versus retail need Binance, Bybit, and OKX. Top versus crowd needs Binance and OKX; Bybit does not publish a top-trader ratio. A named venue is missing from that clause and is not treated as zero. Informational only — not financial advice."
	// situationSizeFlatUSD ignores dust smaller than one USDT on a sized bar.
	situationSizeFlatUSD = 1.0
)

// CoinSituation is the short read shown on a coin page and stored on a paper fill.
// A nil clause means that part stays blank. The matching *Missing list names who is absent.
type CoinSituation struct {
	Symbol              string                 `json:"symbol"`
	AsOf                time.Time              `json:"asOf"`
	PriceCVD            *SituationPriceCVD     `json:"priceCvd,omitempty"`
	PriceCVDMissing     []string               `json:"priceCvdMissing,omitempty"`
	OpenInterest        *SituationOpenInterest `json:"openInterest,omitempty"`
	OpenInterestMissing []string               `json:"openInterestMissing,omitempty"`
	Liquidation         *SituationLiquidation  `json:"liquidation,omitempty"`
	LiquidationMissing  []string               `json:"liquidationMissing,omitempty"`
	Funding             *SituationFunding      `json:"funding,omitempty"`
	FundingMissing      []string               `json:"fundingMissing,omitempty"`
	Crowd               *SituationCrowd        `json:"crowd,omitempty"`
	CrowdMissing        []string               `json:"crowdMissing,omitempty"`
	Size                *SituationSize         `json:"size,omitempty"`
	SizeMissing         []string               `json:"sizeMissing,omitempty"`
	Note                string                 `json:"note,omitempty"`
}

// SituationFunding is who pays the OI-weighted predicted funding rate.
// Payer is long, short, or none. RatePct is the rate in percent (0.01 = 0.01%).
type SituationFunding struct {
	Payer   string  `json:"payer"`
	RatePct float64 `json:"ratePct,omitempty"`
}

// SituationCrowd is top-trader long share minus the crowd, in percentage points.
// Lean is top_more_long, top_more_short, or even. Bybit does not publish this.
type SituationCrowd struct {
	Lean   string  `json:"lean"`
	GapPct float64 `json:"gapPct,omitempty"`
}

// SituationSize says whether whale prints and smaller prints agree on the latest sized bar.
type SituationSize struct {
	Kind string `json:"kind"`
}

// SituationPriceCVD is how price and futures CVD are moving on the combined tape.
type SituationPriceCVD struct {
	Kind string `json:"kind"`
}

// SituationOpenInterest says whether the combined 1h open interest is building.
type SituationOpenInterest struct {
	Flow   string `json:"flow"`
	Window string `json:"window"`
}

// SituationLiquidation is which estimated liquidation side sits closer to last.
type SituationLiquidation struct {
	Side        string  `json:"side"`
	DistancePct float64 `json:"distancePct,omitempty"`
}

// BuildCoinSituation folds an already-loaded tape into one read.
// A clause is set only when every venue that publishes it covers it.
// funding and longs may be nil; those clauses then stay blank and name who is absent.
func BuildCoinSituation(symbol string, now time.Time, cvd *CVDReport, oi *OpenInterestSnapshot, levels *LiquidationLevelsReport, funding *FundingSnapshot, longs *LongShortSnapshot) *CoinSituation {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	out := &CoinSituation{
		Symbol: NormalizeLiquidationSymbol(symbol),
		AsOf:   now.UTC(),
		Note:   situationNote,
	}
	if kind, ok, missing := situationPriceCVD(cvd); ok {
		out.PriceCVD = &SituationPriceCVD{Kind: kind}
	} else {
		out.PriceCVDMissing = missing
	}
	if flow, ok, missing := situationOpenInterest(oi); ok {
		out.OpenInterest = &SituationOpenInterest{Flow: flow, Window: OpenInterestWindow1h}
	} else {
		out.OpenInterestMissing = missing
	}
	if side, dist, ok, missing := situationLiquidation(levels); ok {
		out.Liquidation = &SituationLiquidation{Side: side, DistancePct: dist}
	} else {
		out.LiquidationMissing = missing
	}
	if payer, rate, ok, missing := situationFunding(funding); ok {
		out.Funding = &SituationFunding{Payer: payer, RatePct: rate}
	} else {
		out.FundingMissing = missing
	}
	if lean, gap, ok, missing := situationCrowd(longs); ok {
		out.Crowd = &SituationCrowd{Lean: lean, GapPct: gap}
	} else {
		out.CrowdMissing = missing
	}
	if kind, ok, missing := situationSize(cvd); ok {
		out.Size = &SituationSize{Kind: kind}
	} else {
		out.SizeMissing = missing
	}
	return out
}

// SituationLeft reports whether the live read no longer says what the entry read said.
// Only clauses that were set at entry count. A clause the entry did not store is ignored,
// so an older fill does not alert just because a new clause appeared.
// A set entry clause that is now blank or a different kind has left.
func SituationLeft(entry, live *CoinSituation) (bool, []string) {
	if entry == nil {
		return false, nil
	}
	var changed []string
	if entry.PriceCVD != nil && entry.PriceCVD.Kind != "" {
		got := ""
		if live != nil && live.PriceCVD != nil {
			got = live.PriceCVD.Kind
		}
		if got != entry.PriceCVD.Kind {
			changed = append(changed, "priceCvd")
		}
	}
	if entry.OpenInterest != nil && entry.OpenInterest.Flow != "" {
		got := ""
		if live != nil && live.OpenInterest != nil {
			got = live.OpenInterest.Flow
		}
		if got != entry.OpenInterest.Flow {
			changed = append(changed, "openInterest")
		}
	}
	if entry.Liquidation != nil && entry.Liquidation.Side != "" {
		got := ""
		if live != nil && live.Liquidation != nil {
			got = live.Liquidation.Side
		}
		if got != entry.Liquidation.Side {
			changed = append(changed, "liquidation")
		}
	}
	if entry.Funding != nil && entry.Funding.Payer != "" {
		got := ""
		if live != nil && live.Funding != nil {
			got = live.Funding.Payer
		}
		if got != entry.Funding.Payer {
			changed = append(changed, "funding")
		}
	}
	if entry.Crowd != nil && entry.Crowd.Lean != "" {
		got := ""
		if live != nil && live.Crowd != nil {
			got = live.Crowd.Lean
		}
		if got != entry.Crowd.Lean {
			changed = append(changed, "crowd")
		}
	}
	if entry.Size != nil && entry.Size.Kind != "" {
		got := ""
		if live != nil && live.Size != nil {
			got = live.Size.Kind
		}
		if got != entry.Size.Kind {
			changed = append(changed, "size")
		}
	}
	return len(changed) > 0, changed
}

func situationFunding(snap *FundingSnapshot) (payer string, ratePct float64, ok bool, missing []string) {
	have := map[string]struct{}{}
	if snap != nil && snap.Pressure != nil {
		for _, v := range snap.Pressure.Venues {
			name := strings.ToLower(strings.TrimSpace(v.Exchange))
			if name != "" {
				have[name] = struct{}{}
			}
		}
	}
	for _, ex := range FuturesVenues() {
		if _, found := have[string(ex)]; !found {
			missing = append(missing, string(ex))
		}
	}
	if len(missing) > 0 || snap == nil || snap.Pressure == nil || snap.Pressure.Payer == "" {
		if len(missing) == 0 {
			missing = situationVenueNames()
		}
		return "", 0, false, missing
	}
	pct, _ := strconv.ParseFloat(strings.TrimSpace(snap.Pressure.RatePct), 64)
	if math.IsNaN(pct) || math.IsInf(pct, 0) {
		pct = 0
	}
	return snap.Pressure.Payer, pct, true, nil
}

func situationCrowd(snap *LongShortSnapshot) (lean string, gap float64, ok bool, missing []string) {
	by := map[string]LongShortVenueSnap{}
	if snap != nil {
		for _, v := range snap.Venues {
			by[strings.ToLower(strings.TrimSpace(v.Exchange))] = v
		}
	}
	var gaps []float64
	for _, ex := range []Exchange{ExchangeBinance, ExchangeOKX} {
		v, found := by[string(ex)]
		g := 0.0
		parsed := false
		if found && v.Gap != nil {
			n, err := strconv.ParseFloat(strings.TrimSpace(v.Gap.GapPct), 64)
			parsed = err == nil && !math.IsNaN(n) && !math.IsInf(n, 0) && v.Gap.Lean != ""
			g = n
		}
		if !parsed {
			missing = append(missing, string(ex))
			continue
		}
		gaps = append(gaps, g)
	}
	if len(missing) > 0 || len(gaps) == 0 {
		if len(missing) == 0 {
			missing = []string{string(ExchangeBinance), string(ExchangeOKX)}
		}
		return "", 0, false, missing
	}
	sum := 0.0
	for _, g := range gaps {
		sum += g
	}
	avg := sum / float64(len(gaps))
	return LongShortGapLean(avg), roundSituationPct(avg), true, nil
}

func situationSize(report *CVDReport) (kind string, ok bool, missing []string) {
	if report == nil || report.Combined == nil || !report.Combined.Complete || len(report.Combined.Points) == 0 {
		return "", false, situationCVDMissing(report)
	}
	last := report.Combined.Points[len(report.Combined.Points)-1]
	if !last.SizeKnown {
		missing = situationSizeMissing(report, last.Time)
		if len(missing) == 0 {
			missing = situationVenueNames()
		}
		return "", false, missing
	}
	return situationSizeKind(last.WhaleDelta, last.RetailDelta), true, nil
}

func situationSizeMissing(report *CVDReport, at time.Time) []string {
	want := at.UnixMilli()
	var missing []string
	for _, ex := range FuturesVenues() {
		sized := false
		if report != nil {
			for _, v := range report.Venues {
				if v.Exchange != ex || (v.Market != "" && v.Market != CVDMarketFutures) {
					continue
				}
				for _, p := range v.Points {
					if p.Time.UnixMilli() == want && p.SizeKnown {
						sized = true
						break
					}
				}
			}
		}
		if !sized {
			missing = append(missing, string(ex))
		}
	}
	return missing
}

func situationSizeKind(whale, retail float64) string {
	if math.Abs(whale) < situationSizeFlatUSD {
		whale = 0
	}
	if math.Abs(retail) < situationSizeFlatUSD {
		retail = 0
	}
	switch {
	case whale > 0 && retail < 0:
		return SituationSizeWhaleBuyRetailSell
	case whale < 0 && retail > 0:
		return SituationSizeWhaleSellRetailBuy
	case whale > 0 && retail > 0:
		return SituationSizeBothBuy
	case whale < 0 && retail < 0:
		return SituationSizeBothSell
	case whale > 0:
		return SituationSizeWhaleBuy
	case whale < 0:
		return SituationSizeWhaleSell
	case retail > 0:
		return SituationSizeRetailBuy
	case retail < 0:
		return SituationSizeRetailSell
	default:
		return SituationSizeFlat
	}
}

func situationPriceCVD(report *CVDReport) (kind string, ok bool, missing []string) {
	if report == nil || report.Combined == nil || !report.Combined.Complete {
		return "", false, situationCVDMissing(report)
	}
	split := ReadCVDSplit(*report.Combined)
	if !split.OK || split.Kind == "" {
		return "", false, situationCVDMissing(report)
	}
	return split.Kind, true, nil
}

func situationCVDMissing(report *CVDReport) []string {
	have := map[Exchange]CVDVenueSeries{}
	if report != nil {
		for _, v := range report.Venues {
			if v.Market != "" && v.Market != CVDMarketFutures {
				continue
			}
			have[v.Exchange] = v
		}
	}
	var missing []string
	for _, ex := range FuturesVenues() {
		v, found := have[ex]
		if !found || v.Error != "" || !v.Complete || len(v.Points) == 0 {
			missing = append(missing, string(ex))
		}
	}
	if len(missing) == 0 {
		for _, ex := range FuturesVenues() {
			missing = append(missing, string(ex))
		}
	}
	return missing
}

func situationOpenInterest(snap *OpenInterestSnapshot) (flow string, ok bool, missing []string) {
	byEx := map[string]OpenInterestVenueSnap{}
	if snap != nil {
		for _, v := range snap.Venues {
			byEx[v.Exchange] = v
		}
	}
	var missingOut []string
	for _, ex := range FuturesVenues() {
		v, found := byEx[string(ex)]
		w, wok := situationOIWindow(v.Windows)
		if !found || !wok || !w.Complete {
			missingOut = append(missingOut, string(ex))
		}
	}
	if len(missingOut) > 0 || snap == nil {
		if len(missingOut) == 0 {
			missingOut = situationVenueNames()
		}
		return "", false, missingOut
	}
	w, wok := situationOIWindow(snap.Windows)
	if !wok || !w.Complete || w.Flow == "" {
		return "", false, situationVenueNames()
	}
	return w.Flow, true, nil
}

func situationOIWindow(windows []OpenInterestWindow) (OpenInterestWindow, bool) {
	for _, w := range windows {
		if w.Window == OpenInterestWindow1h {
			return w, true
		}
	}
	return OpenInterestWindow{}, false
}

func situationLiquidation(report *LiquidationLevelsReport) (side string, distance float64, ok bool, missing []string) {
	if report == nil {
		return "", 0, false, situationVenueNames()
	}
	if len(report.Missing) > 0 {
		return "", 0, false, append([]string(nil), report.Missing...)
	}
	last := parseQty(report.LastPrice)
	if last <= 0 {
		return "", 0, false, situationVenueNames()
	}
	var longDist, shortDist float64
	var longN, shortN float64
	longFound, shortFound := false, false
	for _, bar := range report.Levels {
		px := parseQty(bar.Price)
		if px <= 0 {
			continue
		}
		dist := math.Abs(px-last) / last * 100
		if px < last {
			n := parseQty(bar.LongNotional)
			if n <= 0 {
				continue
			}
			if !longFound || dist < longDist || (dist == longDist && n > longN) {
				longFound = true
				longDist = dist
				longN = n
			}
		}
		if px > last {
			n := parseQty(bar.ShortNotional)
			if n <= 0 {
				continue
			}
			if !shortFound || dist < shortDist || (dist == shortDist && n > shortN) {
				shortFound = true
				shortDist = dist
				shortN = n
			}
		}
	}
	switch {
	case longFound && shortFound:
		if longDist < shortDist || (longDist == shortDist && longN >= shortN) {
			return SituationLiqLong, roundSituationPct(longDist), true, nil
		}
		return SituationLiqShort, roundSituationPct(shortDist), true, nil
	case longFound:
		return SituationLiqLong, roundSituationPct(longDist), true, nil
	case shortFound:
		return SituationLiqShort, roundSituationPct(shortDist), true, nil
	default:
		return SituationLiqNone, 0, true, nil
	}
}

func situationVenueNames() []string {
	out := make([]string, 0, 3)
	for _, ex := range FuturesVenues() {
		out = append(out, string(ex))
	}
	return out
}

func roundSituationPct(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Round(v*10) / 10
}
