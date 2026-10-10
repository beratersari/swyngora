package domain

import (
	"math"
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

	situationNote = "Each clause stays blank until Binance, Bybit, and OKX all cover it. A named venue is missing from that clause and is not treated as zero. Informational only — not financial advice."
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
	Note                string                 `json:"note,omitempty"`
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

// BuildCoinSituation folds an already-loaded CVD report, open-interest snapshot,
// and liquidation map into one read. A clause is set only when every futures
// venue covers it.
func BuildCoinSituation(symbol string, now time.Time, cvd *CVDReport, oi *OpenInterestSnapshot, levels *LiquidationLevelsReport) *CoinSituation {
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
	return out
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
