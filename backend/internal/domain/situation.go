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

	// Measured force-order prints over the last 24 hours. These are separate
	// from the estimated pocket above.
	SituationPrintLong  = "long"
	SituationPrintShort = "short"
	SituationPrintQuiet = "quiet"
	SituationPrintEven  = "even"

	// situationPrintLead is the share the larger side needs before the window
	// is called long or short. Under this, both sides printed and the read is even.
	situationPrintLead = 0.6

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

	situationNote = "Each clause stays blank until every venue that publishes it covers it. Funding, price versus CVD, open interest, the liquidation model, measured liquidation prints, and whale versus retail need Binance, Bybit, and OKX. Top versus crowd needs Binance and OKX; Bybit does not publish a top-trader ratio. A named venue is missing from that clause and is not treated as zero. The model names the closer estimated pocket. Measured prints are force-orders over the last 24 hours, and that total is definite only when the window was recorded. Informational only — not financial advice."
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
	Prints              *SituationPrints       `json:"prints,omitempty"`
	PrintsMissing       []string               `json:"printsMissing,omitempty"`
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

// SituationPrints is measured force-order notional over the last 24 hours.
// Side is long, short, even, or quiet. Notionals are USDT.
type SituationPrints struct {
	Side          string  `json:"side"`
	LongNotional  float64 `json:"longNotional,omitempty"`
	ShortNotional float64 `json:"shortNotional,omitempty"`
	Window        string  `json:"window"`
}

// SituationPrintInput is the 24h print archive plus the live coverage clock.
// ArchiveOK is false when the stored sum could not be read. The clause then
// stays blank, because the memory book alone is not the whole window.
type SituationPrintInput struct {
	ArchiveOK bool
	Coverage  []LiquidationCoverage
	Sides     []LiquidationPrintSideRow
}

// BuildCoinSituation folds an already-loaded tape into one read.
// A clause is set only when every venue that publishes it covers it.
// funding and longs may be nil; those clauses then stay blank and name who is absent.
func BuildCoinSituation(symbol string, now time.Time, cvd *CVDReport, oi *OpenInterestSnapshot, levels *LiquidationLevelsReport, funding *FundingSnapshot, longs *LongShortSnapshot, prints SituationPrintInput) *CoinSituation {
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
	if row, ok, missing := situationPrints(symbol, now, prints); ok {
		out.Prints = &row
	} else {
		out.PrintsMissing = missing
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

const (
	// SituationLeftChanged means the entry clause still has a live kind, and it differs.
	SituationLeftChanged = "changed"
	// SituationLeftMissing means the entry clause is blank on the later read.
	// That is not a new opinion.
	SituationLeftMissing = "missing"
)

// SituationClauseLeft is one stored clause that is no longer the same opinion.
// Status is changed or missing. Entry and Live are the kind tokens
// (priceCvd.kind, openInterest.flow, liquidation.side, prints.side, funding.payer, crowd.lean, size.kind).
// Missing names the venues when the later read says who is absent.
type SituationClauseLeft struct {
	Clause  string   `json:"clause"`
	Status  string   `json:"status"`
	Entry   string   `json:"entry,omitempty"`
	Live    string   `json:"live,omitempty"`
	Missing []string `json:"missing,omitempty"`
}

// SituationComparison is the entry read against a later read.
// Same is true only when the entry stored at least one clause and every one of
// those clauses still has the same kind. A clause the entry did not store is omitted.
// A blank later clause is missing, not a new kind.
type SituationComparison struct {
	Same    bool                  `json:"same"`
	Clauses []SituationClauseLeft `json:"clauses,omitempty"`
}

// SituationExit is one buy a sell closed, compared with the read stored on that sell.
type SituationExit struct {
	SourceTradeID string
	EntryAt       time.Time
	Entry         *CoinSituation
	Comparison    SituationComparison
}

// CompareCoinSituation names the stored clauses whose kind changed, and the
// stored clauses the later read left blank. A clause only the later read has
// stays quiet. Print dollars can change without a new side.
func CompareCoinSituation(entry, live *CoinSituation) SituationComparison {
	var out SituationComparison
	if entry == nil {
		return out
	}
	stored := 0
	add := func(set bool, entryKind, liveKind, clause string, missing []string) {
		if !set || entryKind == "" {
			return
		}
		stored++
		if liveKind == "" {
			row := SituationClauseLeft{Clause: clause, Status: SituationLeftMissing, Entry: entryKind}
			if len(missing) > 0 {
				row.Missing = append([]string(nil), missing...)
			}
			out.Clauses = append(out.Clauses, row)
			return
		}
		if liveKind != entryKind {
			out.Clauses = append(out.Clauses, SituationClauseLeft{
				Clause: clause, Status: SituationLeftChanged, Entry: entryKind, Live: liveKind,
			})
		}
	}
	add(entry.PriceCVD != nil, kindOf(entry.PriceCVD), kindOf(livePrice(live)), "priceCvd", missingOf(live, "priceCvd"))
	add(entry.OpenInterest != nil, flowOf(entry.OpenInterest), flowOf(liveOI(live)), "openInterest", missingOf(live, "openInterest"))
	add(entry.Liquidation != nil, sideOfLiq(entry.Liquidation), sideOfLiq(liveLiq(live)), "liquidation", missingOf(live, "liquidation"))
	add(entry.Prints != nil, sideOfPrints(entry.Prints), sideOfPrints(livePrints(live)), "prints", missingOf(live, "prints"))
	add(entry.Funding != nil, payerOf(entry.Funding), payerOf(liveFunding(live)), "funding", missingOf(live, "funding"))
	add(entry.Crowd != nil, leanOf(entry.Crowd), leanOf(liveCrowd(live)), "crowd", missingOf(live, "crowd"))
	add(entry.Size != nil, sizeKindOf(entry.Size), sizeKindOf(liveSize(live)), "size", missingOf(live, "size"))
	out.Same = stored > 0 && len(out.Clauses) == 0
	return out
}

// SituationLeft reports whether the live read no longer says what the entry read said.
// Only clauses that were set at entry count. A clause the entry did not store is ignored,
// so an older fill does not alert just because a new clause appeared.
// A set entry clause that is now blank or a different kind has left.
// The names are the same clauses CompareCoinSituation returns, in that order.
func SituationLeft(entry, live *CoinSituation) (bool, []string) {
	cmp := CompareCoinSituation(entry, live)
	if len(cmp.Clauses) == 0 {
		return false, nil
	}
	names := make([]string, len(cmp.Clauses))
	for i := range cmp.Clauses {
		names[i] = cmp.Clauses[i].Clause
	}
	return true, names
}

func kindOf(v *SituationPriceCVD) string {
	if v == nil {
		return ""
	}
	return v.Kind
}

func flowOf(v *SituationOpenInterest) string {
	if v == nil {
		return ""
	}
	return v.Flow
}

func sideOfLiq(v *SituationLiquidation) string {
	if v == nil {
		return ""
	}
	return v.Side
}

func sideOfPrints(v *SituationPrints) string {
	if v == nil {
		return ""
	}
	return v.Side
}

func payerOf(v *SituationFunding) string {
	if v == nil {
		return ""
	}
	return v.Payer
}

func leanOf(v *SituationCrowd) string {
	if v == nil {
		return ""
	}
	return v.Lean
}

func sizeKindOf(v *SituationSize) string {
	if v == nil {
		return ""
	}
	return v.Kind
}

func livePrice(live *CoinSituation) *SituationPriceCVD {
	if live == nil {
		return nil
	}
	return live.PriceCVD
}

func liveOI(live *CoinSituation) *SituationOpenInterest {
	if live == nil {
		return nil
	}
	return live.OpenInterest
}

func liveLiq(live *CoinSituation) *SituationLiquidation {
	if live == nil {
		return nil
	}
	return live.Liquidation
}

func livePrints(live *CoinSituation) *SituationPrints {
	if live == nil {
		return nil
	}
	return live.Prints
}

func liveFunding(live *CoinSituation) *SituationFunding {
	if live == nil {
		return nil
	}
	return live.Funding
}

func liveCrowd(live *CoinSituation) *SituationCrowd {
	if live == nil {
		return nil
	}
	return live.Crowd
}

func liveSize(live *CoinSituation) *SituationSize {
	if live == nil {
		return nil
	}
	return live.Size
}

func missingOf(live *CoinSituation, clause string) []string {
	if live == nil {
		return nil
	}
	switch clause {
	case "priceCvd":
		return live.PriceCVDMissing
	case "openInterest":
		return live.OpenInterestMissing
	case "liquidation":
		return live.LiquidationMissing
	case "prints":
		return live.PrintsMissing
	case "funding":
		return live.FundingMissing
	case "crowd":
		return live.CrowdMissing
	case "size":
		return live.SizeMissing
	default:
		return nil
	}
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

func situationPrints(symbol string, now time.Time, in SituationPrintInput) (SituationPrints, bool, []string) {
	if !in.ArchiveOK {
		return SituationPrints{}, false, situationVenueNames()
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	from := now.Add(-24 * time.Hour)
	byEx := map[Exchange]LiquidationPrintSideRow{}
	for _, row := range in.Sides {
		byEx[row.Exchange] = row
	}
	var missing []string
	var longN, shortN float64
	for _, ex := range FuturesVenues() {
		cov := printCoverage(ex, symbol, in.Coverage)
		if !situationPrintWindowRecorded(cov, from, now) {
			missing = append(missing, string(ex))
			continue
		}
		row := byEx[ex]
		if row.Long > 0 {
			longN += row.Long
		}
		if row.Short > 0 {
			shortN += row.Short
		}
	}
	if len(missing) > 0 {
		return SituationPrints{}, false, missing
	}
	return SituationPrints{
		Side:          situationPrintSide(longN, shortN),
		LongNotional:  roundSituationUSD(longN),
		ShortNotional: roundSituationUSD(shortN),
		Window:        LiquidationWindow24h,
	}, true, nil
}

// situationPrintWindowRecorded is true when the watch started at or before
// the window open and no known gap meets [from, until).
func situationPrintWindowRecorded(cov LiquidationCoverage, from, until time.Time) bool {
	if cov.FirstWatch.IsZero() || cov.FirstWatch.After(from) {
		return false
	}
	return !printWindowOverlapsGap(cov.Gaps, from, until)
}

func situationPrintSide(longN, shortN float64) string {
	total := longN + shortN
	if total < situationSizeFlatUSD {
		return SituationPrintQuiet
	}
	if longN > 0 && shortN > 0 {
		larger := longN
		if shortN > larger {
			larger = shortN
		}
		if larger/total < situationPrintLead {
			return SituationPrintEven
		}
	}
	if shortN > longN {
		return SituationPrintShort
	}
	if longN > shortN {
		return SituationPrintLong
	}
	return SituationPrintEven
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

func roundSituationUSD(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return 0
	}
	return math.Round(v*100) / 100
}
