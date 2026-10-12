package domain

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// situationAtNote is the past read. The live note stays situationNote.
const situationAtNote = "This read is the stored five-minute minute at the candle. Funding, price versus CVD, open interest, measured liquidation prints, and whale versus retail need Binance, Bybit, and OKX. Top versus crowd needs Binance and OKX. A named venue is a minute the archive does not have and is not treated as zero. The estimated liquidation pocket is not part of this read. Informational only — not financial advice."

// SituationAtInput is the archive already loaded up to At.
// CVD is the combined tape built with now=At, and its last bar starts before At.
// Open interest, funding, and crowd are stored snapshots at or before At.
// Prints are five-minute venue buckets. The estimated pocket is not an input.
type SituationAtInput struct {
	At           time.Time
	Symbol       string
	CVD          *CVDReport
	OpenInterest []FuturesSnapshot
	Funding      []FuturesSnapshot
	FundingOI    []FuturesSnapshot
	Crowd        []FuturesSnapshot
	Prints       []SituationPrintMinute
	Coverage     []LiquidationCoverage
}

// BuildSituationAt rebuilds the coin read at one candle close.
// A clause stays blank when the archive has no five-minute minute within
// situationHeldGap of At, and the missing list names that venue.
// The estimated liquidation pocket is left unset.
// Since is the clock of that kind as of At. It stays zero when the minutes
// do not reach a different kind.
func BuildSituationAt(in SituationAtInput) *CoinSituation {
	if in.At.IsZero() {
		return nil
	}
	at := in.At.UTC()
	symbol := NormalizeLiquidationSymbol(in.Symbol)
	out := &CoinSituation{
		Symbol: symbol,
		AsOf:   at,
		Note:   situationAtNote,
	}
	applySituationAtCVD(out, in.CVD, at)
	if flow, ok, missing := situationOpenInterestAt(in.OpenInterest, at); ok {
		out.OpenInterest = &SituationOpenInterest{Flow: flow, Window: OpenInterestWindow1h}
	} else {
		out.OpenInterestMissing = missing
	}
	if row, ok, missing := situationPrintsAt(symbol, at, in.Prints, in.Coverage); ok {
		out.Prints = &row
	} else {
		out.PrintsMissing = missing
	}
	if payer, rate, ok, missing := situationFundingAt(in.Funding, in.FundingOI, at); ok {
		out.Funding = &SituationFunding{Payer: payer, RatePct: rate}
	} else {
		out.FundingMissing = missing
	}
	if lean, gap, ok, missing := situationCrowdAt(in.Crowd, at); ok {
		out.Crowd = &SituationCrowd{Lean: lean, GapPct: gap}
	} else {
		out.CrowdMissing = missing
	}
	var points []CVDPoint
	if in.CVD != nil && in.CVD.Combined != nil {
		points = in.CVD.Combined.Points
	}
	StampSituationHeld(out, SituationHeldInput{
		Now:          at,
		Symbol:       symbol,
		CVD:          points,
		OpenInterest: in.OpenInterest,
		Funding:      formatFundingPressurePoints(buildFundingPressureHistory(in.Funding, in.FundingOI, "all", at)),
		Crowd:        situationCrowdSnaps(in.Crowd, at),
		Prints:       in.Prints,
		Coverage:     in.Coverage,
	})
	return out
}

func situationAtFresh(sample, at time.Time) bool {
	if sample.IsZero() || at.IsZero() || sample.After(at) {
		return false
	}
	return at.Sub(sample) <= situationHeldGap
}

func applySituationAtCVD(out *CoinSituation, report *CVDReport, at time.Time) {
	if out == nil {
		return
	}
	last := situationAtCVDLast(report)
	fresh := situationAtFresh(last, at)
	kind, ok, missing := situationPriceCVD(report)
	if ok && fresh {
		out.PriceCVD = &SituationPriceCVD{Kind: kind}
	} else if ok {
		out.PriceCVDMissing = situationAtStaleVenues(report, at)
	} else {
		out.PriceCVDMissing = missing
	}
	skind, sok, smissing := situationSize(report)
	if sok && fresh {
		out.Size = &SituationSize{Kind: skind}
	} else if sok {
		out.SizeMissing = situationAtStaleVenues(report, at)
	} else {
		out.SizeMissing = smissing
	}
}

func situationAtCVDLast(report *CVDReport) time.Time {
	if report == nil || report.Combined == nil || len(report.Combined.Points) == 0 {
		return time.Time{}
	}
	return report.Combined.Points[len(report.Combined.Points)-1].Time
}

func situationAtStaleVenues(report *CVDReport, at time.Time) []string {
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
		if !found || v.Error != "" || len(v.Points) == 0 {
			missing = append(missing, string(ex))
			continue
		}
		last := v.Points[len(v.Points)-1]
		if !situationAtFresh(last.Time, at) {
			missing = append(missing, string(ex))
		}
	}
	if len(missing) == 0 {
		return situationVenueNames()
	}
	return missing
}

func situationOpenInterestAt(rows []FuturesSnapshot, at time.Time) (string, bool, []string) {
	byEx := situationOIReadings(rows, at)
	var missing []string
	var curSum, pastSum float64
	for _, ex := range FuturesVenues() {
		cur, curOK := latestSituationOI(byEx[ex], at)
		if !curOK {
			missing = append(missing, string(ex))
			continue
		}
		past, pastOK := oiReadingAt(byEx[ex], cur.at.Add(-time.Hour), OpenInterestSampleSlack(time.Hour))
		if _, pctOK := oiPctChange(cur.contracts, past.contracts); !pastOK || !pctOK {
			missing = append(missing, string(ex))
			continue
		}
		curSum += cur.contracts
		pastSum += past.contracts
	}
	if len(missing) > 0 {
		return "", false, missing
	}
	pct, pctOK := oiPctChange(curSum, pastSum)
	if !pctOK {
		return "", false, situationVenueNames()
	}
	return situationFlowFromDirection(OpenInterestDirection(pct)), true, nil
}

func situationOIReadings(rows []FuturesSnapshot, at time.Time) map[Exchange][]readingOI {
	byEx := map[Exchange][]readingOI{}
	for _, row := range rows {
		if row.SampledAt.IsZero() || row.SampledAt.After(at) || !isFuturesSituationVenue(row.Exchange) || row.Contracts <= 0 {
			continue
		}
		byEx[row.Exchange] = append(byEx[row.Exchange], readingOI{
			at:        TruncateToBucket(row.SampledAt.UTC(), situationHeldStep),
			sampled:   row.SampledAt.UTC(),
			contracts: row.Contracts,
		})
	}
	for ex, list := range byEx {
		sort.Slice(list, func(i, j int) bool {
			if list[i].at.Equal(list[j].at) {
				return list[i].sampled.Before(list[j].sampled)
			}
			return list[i].at.Before(list[j].at)
		})
		compact := list[:0]
		for _, row := range list {
			if len(compact) > 0 && compact[len(compact)-1].at.Equal(row.at) {
				compact[len(compact)-1] = row
				continue
			}
			compact = append(compact, row)
		}
		byEx[ex] = compact
	}
	return byEx
}

func latestSituationOI(rows []readingOI, at time.Time) (readingOI, bool) {
	var best readingOI
	found := false
	for _, row := range rows {
		if row.sampled.After(at) || row.at.After(at) {
			break
		}
		best = row
		found = true
	}
	if !found || !situationAtFresh(best.at, at) {
		return readingOI{}, false
	}
	return best, true
}

func situationFundingAt(funding, oi []FuturesSnapshot, at time.Time) (string, float64, bool, []string) {
	from := at.Add(-FundingPressureLookback)
	type key struct {
		ex Exchange
		t  int64
	}
	rates := map[key]float64{}
	ois := map[key]float64{}
	var latest int64
	found := false
	for _, row := range funding {
		if !row.Predicted || !isFuturesSituationVenue(row.Exchange) {
			continue
		}
		sampled := row.SampledAt.UTC()
		if sampled.IsZero() || sampled.After(at) || sampled.Before(from) {
			continue
		}
		bucket := TruncateToBucket(sampled, fundingPressureBucket)
		if !situationAtFresh(bucket, at) {
			continue
		}
		rates[key{row.Exchange, bucket.Unix()}] = row.FundingRate
		if !found || bucket.Unix() > latest {
			latest = bucket.Unix()
			found = true
		}
	}
	for _, row := range oi {
		if !isFuturesSituationVenue(row.Exchange) || row.Value <= 0 {
			continue
		}
		sampled := row.SampledAt.UTC()
		if sampled.IsZero() || sampled.After(at) || sampled.Before(from.Add(-fundingPressureBucket)) {
			continue
		}
		bucket := TruncateToBucket(sampled, fundingPressureBucket)
		ois[key{row.Exchange, bucket.Unix()}] = row.Value
	}
	if !found {
		return "", 0, false, situationVenueNames()
	}
	prev := latest - int64(fundingPressureBucket/time.Second)
	var weights []fundingWeight
	var missing []string
	for _, ex := range FuturesVenues() {
		rate, hasR := rates[key{ex, latest}]
		oiV, hasO := ois[key{ex, latest}]
		if !hasO {
			oiV, hasO = ois[key{ex, prev}]
		}
		if !hasR || !hasO || oiV <= 0 {
			missing = append(missing, string(ex))
			continue
		}
		weights = append(weights, fundingWeight{Exchange: ex, Rate: rate, OI: oiV})
	}
	if len(missing) > 0 {
		return "", 0, false, missing
	}
	raw, ok := blendFundingPressure(weights, time.Unix(latest, 0).UTC())
	if !ok || raw.Rate != raw.Rate {
		return "", 0, false, situationVenueNames()
	}
	_, pct := FormatFundingRate(raw.Rate)
	n, err := strconv.ParseFloat(strings.TrimSpace(pct), 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
		n = 0
	}
	return FundingPayer(raw.Rate), n, true, nil
}

func situationCrowdAt(rows []FuturesSnapshot, at time.Time) (string, float64, bool, []string) {
	var gaps []float64
	var missing []string
	for _, ex := range []Exchange{ExchangeBinance, ExchangeOKX} {
		gap, ok := latestSituationCrowd(rows, ex, at)
		if !ok {
			missing = append(missing, string(ex))
			continue
		}
		gaps = append(gaps, gap)
	}
	if len(missing) > 0 || len(gaps) != 2 {
		if len(missing) == 0 {
			missing = []string{string(ExchangeBinance), string(ExchangeOKX)}
		}
		return "", 0, false, missing
	}
	avg := (gaps[0] + gaps[1]) / 2
	return LongShortGapLean(avg), roundSituationPct(avg), true, nil
}

func latestSituationCrowd(rows []FuturesSnapshot, ex Exchange, at time.Time) (float64, bool) {
	var bestBucket, bestSampled time.Time
	var gap float64
	found := false
	for _, row := range rows {
		if Exchange(strings.ToLower(strings.TrimSpace(string(row.Exchange)))) != ex {
			continue
		}
		sampled := row.SampledAt.UTC()
		if sampled.IsZero() || sampled.After(at) {
			continue
		}
		crowd, top, ok := situationCrowdShares(row)
		if !ok {
			continue
		}
		bucket := TruncateToBucket(sampled, situationHeldStep)
		if !situationAtFresh(bucket, at) {
			continue
		}
		if found && (bucket.Before(bestBucket) || (bucket.Equal(bestBucket) && !sampled.After(bestSampled))) {
			continue
		}
		bestBucket = bucket
		bestSampled = sampled
		gap = (top - crowd) * 100
		found = true
	}
	return gap, found
}

func situationCrowdShares(row FuturesSnapshot) (crowd, top float64, ok bool) {
	if row.LongShare <= 0 && row.ShortShare <= 0 {
		return 0, 0, false
	}
	top = row.SizeLongShare
	if top <= 0 {
		top = row.TopAccLongShare
	}
	if top <= 0 {
		return 0, 0, false
	}
	return row.LongShare, top, true
}

func situationCrowdSnaps(rows []FuturesSnapshot, at time.Time) []LongShortVenueSnap {
	type acc struct {
		sampled time.Time
		crowd   float64
		top     float64
	}
	by := map[Exchange]map[int64]acc{}
	for _, row := range rows {
		ex := Exchange(strings.ToLower(strings.TrimSpace(string(row.Exchange))))
		if ex != ExchangeBinance && ex != ExchangeOKX {
			continue
		}
		sampled := row.SampledAt.UTC()
		if sampled.IsZero() || sampled.After(at) {
			continue
		}
		crowd, top, ok := situationCrowdShares(row)
		if !ok {
			continue
		}
		bucket := TruncateToBucket(sampled, situationHeldStep).Unix()
		if by[ex] == nil {
			by[ex] = map[int64]acc{}
		}
		prev, found := by[ex][bucket]
		if found && !sampled.After(prev.sampled) {
			continue
		}
		by[ex][bucket] = acc{sampled: sampled, crowd: crowd, top: top}
	}
	var out []LongShortVenueSnap
	for _, ex := range []Exchange{ExchangeBinance, ExchangeOKX} {
		points := by[ex]
		if len(points) == 0 {
			continue
		}
		hist := make([]LongShortGapPoint, 0, len(points))
		for unix, row := range points {
			gap := (row.top - row.crowd) * 100
			hist = append(hist, LongShortGapPoint{
				Time:   time.Unix(unix, 0).UTC(),
				GapPct: FormatSignedPct(gap),
				Lean:   LongShortGapLean(gap),
			})
		}
		sort.Slice(hist, func(i, j int) bool { return hist[i].Time.Before(hist[j].Time) })
		out = append(out, LongShortVenueSnap{Exchange: string(ex), GapHistory: hist})
	}
	return out
}

func situationPrintsAt(symbol string, at time.Time, minutes []SituationPrintMinute, coverage []LiquidationCoverage) (SituationPrints, bool, []string) {
	if at.IsZero() {
		return SituationPrints{}, false, situationVenueNames()
	}
	from := at.Add(-24 * time.Hour)
	byEx := map[Exchange][]struct {
		at          time.Time
		long, short float64
	}{}
	for _, row := range minutes {
		if row.At.IsZero() || !isFuturesSituationVenue(row.Exchange) {
			continue
		}
		byEx[row.Exchange] = append(byEx[row.Exchange], struct {
			at          time.Time
			long, short float64
		}{at: row.At.UTC(), long: row.Long, short: row.Short})
	}
	var missing []string
	var longN, shortN float64
	for _, ex := range FuturesVenues() {
		if !situationPrintWindowRecorded(printCoverage(ex, symbol, coverage), from, at) {
			missing = append(missing, string(ex))
			continue
		}
		longN, shortN = addPrintWindow(byEx[ex], from, at, longN, shortN)
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
