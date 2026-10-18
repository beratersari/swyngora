package domain

import (
	"fmt"
	"strings"
	"time"
)

const (
	// The recorder follows LiquidRecorderLimit names. The board opens on a
	// shorter page so the first load does not ask every venue for all of them.
	// Callers can request the full recorded set.
	derivativesBoardDefault = 36
	derivativesBoardMin     = 5

	// VenueDerivativesMaxAge is how old an open-interest or funding snapshot
	// may be before that venue is old on the derivatives board.
	VenueDerivativesMaxAge = 5 * time.Minute

	// DerivativesRatioMaxAge is how old the latest long/short bar may be.
	// The public print is a 5-minute bar, so one missed bar is still the
	// current read and two missed bars are old.
	DerivativesRatioMaxAge = 10 * time.Minute

	DerivativesWeightOI    = "oi"
	DerivativesWeightEqual = "equal"

	// VenueReadUnsupported is a venue that does not publish this column.
	// It is named and left out. It is not a zero and it is not a fresh reading.
	VenueReadUnsupported = "unsupported"

	// DerivativesLiqHeatMinCoverage is how long a venue must have been watched
	// before its 5-minute liquidation burst can be compared with its own typical rate.
	DerivativesLiqHeatMinCoverage = 6 * time.Hour

	derivativesBoardNote = "Liquid USDT perpetuals on Binance USD-M, Bybit linear, and OKX USDT-swap. Open interest, funding, crowd long percent, 1h taker buy share, and 24h liquidation totals are combined only when every expected venue is fresh for that column. A missing or old venue is named and is not counted as zero. Crowd long is the account long share, weighted by open interest when that open interest is also fresh. Taker buy share sums the last hour of aggressive buy and sell volume; a window that is not complete stays blank. Price is the mark from the fresh venue with the most open interest. Crowd-versus-top is top-trader long percent minus account long percent, in percentage points. Bybit does not publish a top-trader ratio, so that column names Bybit as unsupported and uses the venues that do. Day CVD is aggressive buy minus sell since 00:00 UTC, blank until each venue's tape reaches that open. Price versus CVD names the current run (price up while CVD falls, or the reverse) and how long it has lasted, or says the two are in line. That cell stays blank until the tape reaches 00:00 UTC, the newest minute is fresh, and the newest bar has a price. Combined reads the summed series and stays blank unless every expected venue's 24-hour tape is complete. A 4-hour window that disagrees without a current run counts as in line. Print size is the 15-minute average fill versus the 15 minutes before it, from the in-memory tape or stored flow minutes when that tape does not cover the half hour. Order count is (buy − sell) / (buy + sell) over those same 15 minutes. Counts are summed only when every expected venue's window is complete and the newest print is still fresh. A venue with no fresh prints stays blank and is not counted as zero. Unusual names the farthest high or low reading versus that coin's own stored samples. Venues are not averaged, and the cell names the venue. A coin with no standout stays blank. Liquidation heat is the 5-minute burst versus that venue's own typical rate, blank until the venue has been watched for 6 hours. Bid share is resting bid notional within 1% of mid. These extra columns do not change whether the rest of the row is definite. Hyperliquid is a separate venue. Its tape and book stay on exchange=hyperliquid and are not added into exchange=all. It does not publish liquidations or a crowd ratio. Informational only — not financial advice."

	hyperliquidBoardNote = "Hyperliquid USDC-margined perpetuals, lined up with the USDT name (BTC is BTCUSDT). The mark, open interest, and hourly funding rate are filled when that coin is listed. Open interest is the mark times coin size. The public trade tape fills day CVD, the price-versus-CVD run, 15-minute print size, 15-minute order count, and the 1-hour taker buy share when that window is complete. A short tape stays blank and is not zero. Day CVD stays blank until the tape reaches 00:00 UTC. The public book is at most 20 levels a side. Bid share within 1% fills only when those levels reach 1% of mid on both sides. Liquidation, crowd, and the crowd-versus-top gap stay blank and are named unsupported. They are not zeros and they are not a missing feed. Hyperliquid is not part of exchange=all. Open-interest history grows only while this process is sampling. The trade tape and the book grow only while this process is watching. Funding history can be read back from Hyperliquid's settled hourly prints. Informational only — not financial advice."
)

// DerivativesVenueInput is one venue's live perp readings for a coin.
type DerivativesVenueInput struct {
	Exchange    Exchange
	OI          *OpenInterestSeries
	OIValue     float64
	OIAt        time.Time
	OIOK        bool
	FundingRate float64
	FundingAt   time.Time
	FundingOK   bool
	MarkPrice   float64
	LiqLong     float64
	LiqShort    float64
	LongShare   float64
	LongShortAt time.Time
	LongShortOK bool
	TakerBuy    float64
	TakerSell   float64
	TakerOK     bool
	// TopLongShare is the top-trader long share (0–1) on venues that publish one.
	TopLongShare   float64
	TopLongOK      bool
	TopUnsupported bool
	CVDDay         float64
	CVDAt          time.Time
	CVDDayOK       bool
	// CVDSplit* is the current price-versus-CVD run. Seconds is signed.
	CVDSplitKind       string
	CVDSplitDuration   string
	CVDSplitSeconds    int
	CVDSplitAt         time.Time
	CVDSplitOK         bool
	PrintNotional      float64
	PrintCount         int
	PrintPriorNotional float64
	PrintPriorCount    int
	PrintAt            time.Time
	PrintOK            bool
	// OrderBuy and OrderSell are aggressive print counts over the last 15 minutes.
	OrderBuy    int
	OrderSell   int
	OrderAt     time.Time
	OrderOK     bool
	LiqGrade    string
	LiqRatio    float64
	LiqHeatOK   bool
	BidNotional float64
	AskNotional float64
	BookAt      time.Time
	BookOK      bool
}

// DerivativesCoinInput is one symbol before the board decides what is definite.
type DerivativesCoinInput struct {
	Symbol  string
	Venues  []DerivativesVenueInput
	LiqFeed LiquidationFeed
	// CVDSplit* is the summed series when every venue split is ready and combined CVD is complete.
	// One venue copies that venue's own split. Tests may set these without a series.
	CVDSplitKind     string
	CVDSplitDuration string
	CVDSplitSeconds  int
	CVDSplitOK       bool
}

// DerivativesVenueRow is one venue on a coin, including an incomplete read.
type DerivativesVenueRow struct {
	Exchange         string
	OIValue          string
	OIStatus         string
	FundingRate      string
	FundingRatePct   string
	FundingPayer     string
	FundingStatus    string
	MarkPrice        string
	LiqLong          string
	LiqShort         string
	LiqStatus        string
	LongPct          string
	LongStatus       string
	TakerBuyPct      string
	TakerStatus      string
	GapPct           string
	GapStatus        string
	CVDDay           string
	CVDStatus        string
	CVDSplit         string
	CVDSplitDuration string
	CVDSplitSeconds  string
	CVDSplitStatus   string
	PrintChangePct   string
	PrintStatus      string
	OrderRatio       string
	OrderLean        string
	OrderBuy         int
	OrderSell        int
	OrderStatus      string
	UnusualMetric    string
	UnusualBand      string
	UnusualDistance  string
	LiqGrade         string
	LiqHeatStatus    string
	BidPct           string
	BookStatus       string
}

// DerivativesCoin is one row. Combined fields are empty when that column is
// not definite, so a partial sum is not presented as the market total.
type DerivativesCoin struct {
	Symbol               string
	Base                 string
	Price                string
	PriceExchange        string
	FundingRate          string
	FundingRatePct       string
	FundingPayer         string
	FundingWeight        string
	FundingDefinite      bool
	OpenInterest         string
	OpenInterestDefinite bool
	Change1h             string
	Change1hPct          string
	Change1hDefinite     bool
	Change24h            string
	Change24hPct         string
	Change24hDefinite    bool
	// Change24hMissing names venues with no open-interest sample about 24
	// hours ago. It is set only when the combined change is withheld.
	Change24hMissing []string
	LiqLong          string
	LiqShort         string
	LiqDefinite      bool
	LongPct          string
	LongDefinite     bool
	TakerBuyPct      string
	TakerDefinite    bool
	// GapPct is top-trader long percent minus crowd long percent.
	// GapScope lists the venues inside that average.
	GapPct           string
	GapDefinite      bool
	GapScope         string
	CVDDay           string
	CVDDayDefinite   bool
	CVDSplit         string
	CVDSplitDuration string
	CVDSplitSeconds  string
	CVDSplitDefinite bool
	PrintChangePct   string
	PrintRead        string
	PrintDefinite    bool
	OrderRatio       string
	OrderLean        string
	OrderBuy         int
	OrderSell        int
	OrderDefinite    bool
	// Unusual* is one venue's standout versus that coin's own samples.
	// UnusualVenue names that venue. The readings are not averaged.
	UnusualMetric     string
	UnusualBand       string
	UnusualDistance   string
	UnusualPercentile string
	UnusualVenue      string
	UnusualDefinite   bool
	LiqHeat           string
	LiqHeatRatio      string
	LiqHeatVenue      string
	LiqHeatDefinite   bool
	BidPct            string
	BidDefinite       bool
	Venues            []DerivativesVenueRow
	Coverage          MarketCoverage
	// Recorded is how far back each stored series for this coin actually goes.
	Recorded []RecordedSeries
}

// DerivativesBoard is the liquid-perp desk.
type DerivativesBoard struct {
	Exchange string
	Universe int
	AsOf     time.Time
	Coins    []DerivativesCoin
	Coverage MarketCoverage
	Definite bool
	Summary  string
	Note     string
}

// ParseDerivativesBoardLimit clamps how many liquid perpetuals the board follows.
func ParseDerivativesBoardLimit(n int) int {
	if n <= 0 {
		return derivativesBoardDefault
	}
	if n < derivativesBoardMin {
		return derivativesBoardMin
	}
	if n > LiquidRecorderLimit {
		return LiquidRecorderLimit
	}
	return n
}

// BuildDerivativesBoard ranks coins by fresh open interest and withholds a
// combined figure when any expected venue is missing or old.
func BuildDerivativesBoard(exchange string, coins []DerivativesCoinInput, now time.Time) *DerivativesBoard {
	if now.IsZero() {
		now = time.Now()
	}
	now = now.UTC()
	ex := strings.ToLower(strings.TrimSpace(exchange))
	if ex == "" {
		ex = "all"
	}
	expected := ExpectedFuturesVenues(ex)
	rows := make([]DerivativesCoin, 0, len(coins))
	rank := make([]float64, 0, len(coins))
	for _, in := range coins {
		symbol := NormalizeLiquidationSymbol(in.Symbol)
		if symbol == "" {
			continue
		}
		row, oiRank := buildDerivativesCoin(symbol, expected, in, now)
		rows = append(rows, row)
		rank = append(rank, oiRank)
	}
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0; j-- {
			if rank[j] > rank[j-1] || (rank[j] == rank[j-1] && rows[j].Symbol < rows[j-1].Symbol) {
				rows[j], rows[j-1] = rows[j-1], rows[j]
				rank[j], rank[j-1] = rank[j-1], rank[j]
				continue
			}
			break
		}
	}
	note := derivativesBoardNote
	if IsMarkOnlyVenue(Exchange(ex)) {
		note = hyperliquidBoardNote
	}
	out := &DerivativesBoard{
		Exchange: ex,
		Universe: len(rows),
		AsOf:     now,
		Coins:    rows,
		Note:     note,
	}
	out.Coverage = derivativesBoardCoverage(expected, rows, now)
	out.Definite = len(rows) > 0 && out.Coverage.Complete && derivativesRowsDefinite(rows)
	if !out.Coverage.Complete {
		out.Summary = out.Coverage.Explain()
	} else if len(rows) > 0 {
		top := rows[0]
		out.Summary = fmt.Sprintf("%d coins. %s open interest %s, funding %s%%.", len(rows), top.Base, top.OpenInterest, top.FundingRatePct)
	}
	return out
}

func derivativesRowsDefinite(rows []DerivativesCoin) bool {
	for _, row := range rows {
		if !row.OpenInterestDefinite || !row.FundingDefinite || !row.Change1hDefinite || !row.Change24hDefinite || !row.LiqDefinite || !row.LongDefinite || !row.TakerDefinite {
			return false
		}
	}
	return true
}

func buildDerivativesCoin(symbol string, expected []Exchange, in DerivativesCoinInput, now time.Time) (DerivativesCoin, float64) {
	base, _ := SplitBaseQuote(ExchangeBinance, symbol)
	oiSamples := make([]VenueSample, 0, len(expected))
	fundSamples := make([]VenueSample, 0, len(expected))
	lsSamples := make([]VenueSample, 0, len(expected))
	takerSamples := make([]VenueSample, 0, len(expected))
	byEx := map[Exchange]DerivativesVenueInput{}
	for _, v := range in.Venues {
		if v.Exchange == "" {
			continue
		}
		byEx[v.Exchange] = v
	}
	var freshOI float64
	for _, ex := range expected {
		v := byEx[ex]
		if v.OIOK {
			oiSamples = append(oiSamples, VenueSample{Exchange: ex, OK: true, At: v.OIAt})
			freshOI += v.OIValue
		}
		if v.FundingOK {
			fundSamples = append(fundSamples, VenueSample{Exchange: ex, OK: true, At: v.FundingAt})
		}
		if v.LongShortOK {
			lsSamples = append(lsSamples, VenueSample{Exchange: ex, OK: true, At: v.LongShortAt})
		}
		if v.TakerOK {
			takerSamples = append(takerSamples, VenueSample{Exchange: ex, OK: true})
		}
	}
	oiCover := CoverVenues(expected, oiSamples, now, VenueDerivativesMaxAge)
	fundCover := CoverVenues(expected, fundSamples, now, VenueDerivativesMaxAge)
	lsCover := CoverVenues(expected, lsSamples, now, DerivativesRatioMaxAge)
	takerCover := CoverVenues(expected, takerSamples, now, VenueDerivativesMaxAge)
	liqCover := CoverageFromLiquidationFeed(in.LiqFeed, coverageExchange(expected), now)
	row := DerivativesCoin{
		Symbol:               symbol,
		Base:                 base,
		OpenInterestDefinite: oiCover.Complete,
		FundingDefinite:      fundCover.Complete,
		LongDefinite:         lsCover.Complete,
		TakerDefinite:        takerCover.Complete,
		LiqDefinite:          liqCover.Complete,
		Coverage:             oiCover,
		Venues:               make([]DerivativesVenueRow, 0, len(expected)),
	}
	if oiCover.Complete {
		var sum float64
		for _, ex := range expected {
			sum += byEx[ex].OIValue
		}
		row.OpenInterest = formatQty(sum)
	}
	change1h, pct1h, ok1h := derivativesOIChange(expected, byEx, OpenInterestWindow1h, time.Hour, now)
	row.Change1hDefinite = ok1h && oiCover.Complete
	if row.Change1hDefinite {
		row.Change1h = change1h
		row.Change1hPct = pct1h
	}
	change24h, pct24h, ok24h := derivativesOIChange(expected, byEx, OpenInterestWindow24h, 24*time.Hour, now)
	row.Change24hDefinite = ok24h && oiCover.Complete
	if row.Change24hDefinite {
		row.Change24h = change24h
		row.Change24hPct = pct24h
	} else if missing := derivativesOISampleMissing(expected, byEx, 24*time.Hour, now); len(missing) > 0 {
		row.Change24hMissing = missing
	}
	if fundCover.Complete {
		rate, weight := derivativesFunding(expected, byEx, oiCover.Complete)
		dec, pct := formatSignedFunding(rate)
		row.FundingRate = dec
		row.FundingRatePct = pct
		row.FundingPayer = FundingPayer(rate)
		row.FundingWeight = weight
	}
	if lsCover.Complete {
		if share, ok := derivativesLongShare(expected, byEx, oiCover.Complete); ok {
			row.LongPct = formatFixed(share*100, 2)
		} else {
			row.LongDefinite = false
		}
	}
	if takerCover.Complete {
		if share, ok := derivativesTakerBuy(expected, byEx); ok {
			row.TakerBuyPct = formatFixed(share*100, 2)
		} else {
			row.TakerDefinite = false
		}
	}
	row.Price, row.PriceExchange = derivativesPrice(expected, byEx, fundCover)
	var liqLong, liqShort float64
	oiStatus := indexVenueReads(oiCover)
	fundStatus := indexVenueReads(fundCover)
	liqStatus := indexVenueReads(liqCover)
	lsStatus := indexVenueReads(lsCover)
	takerStatus := indexVenueReads(takerCover)
	for _, ex := range expected {
		v := byEx[ex]
		venue := DerivativesVenueRow{
			Exchange:      string(ex),
			OIStatus:      oiStatus[ex],
			FundingStatus: fundStatus[ex],
			LiqStatus:     liqStatus[ex],
			LongStatus:    lsStatus[ex],
			TakerStatus:   takerStatus[ex],
		}
		if v.OIOK && venue.OIStatus == VenueReadFresh {
			venue.OIValue = formatQty(v.OIValue)
		}
		if v.FundingOK && venue.FundingStatus == VenueReadFresh {
			dec, pct := formatSignedFunding(v.FundingRate)
			venue.FundingRate = dec
			venue.FundingRatePct = pct
			venue.FundingPayer = FundingPayer(v.FundingRate)
			if v.MarkPrice > 0 {
				venue.MarkPrice = formatQty(v.MarkPrice)
			}
		}
		if venue.LiqStatus == VenueReadFresh {
			venue.LiqLong = formatQty(v.LiqLong)
			venue.LiqShort = formatQty(v.LiqShort)
			liqLong += v.LiqLong
			liqShort += v.LiqShort
		}
		if v.LongShortOK && venue.LongStatus == VenueReadFresh {
			venue.LongPct = formatFixed(v.LongShare*100, 2)
		}
		if v.TakerOK && venue.TakerStatus == VenueReadFresh {
			tot := v.TakerBuy + v.TakerSell
			if tot > 0 {
				venue.TakerBuyPct = formatFixed(v.TakerBuy/tot*100, 2)
			}
		}
		row.Venues = append(row.Venues, venue)
	}
	if liqCover.Complete {
		row.LiqLong = formatQty(liqLong)
		row.LiqShort = formatQty(liqShort)
	}
	if !oiCover.Complete {
		freshOI = 0
		for _, ex := range expected {
			if oiStatus[ex] == VenueReadFresh {
				freshOI += byEx[ex].OIValue
			}
		}
	}
	applyBoardScan(&row, expected, in, byEx, liqCover, now)
	applyMarkOnlyBoard(&row, expected)
	return row, freshOI
}

// applyMarkOnlyBoard blanks liquidation and crowd columns Hyperliquid does not publish.
// Tape, CVD, print size, order count, and the book stay on the normal coverage path.
// Unsupported is not a zero and it is not a missing feed.
func applyMarkOnlyBoard(row *DerivativesCoin, expected []Exchange) {
	if row == nil {
		return
	}
	markOnly := 0
	for _, ex := range expected {
		if !IsMarkOnlyVenue(ex) {
			continue
		}
		markOnly++
		cell := boardVenueRow(row, ex)
		if cell == nil {
			continue
		}
		cell.LiqStatus = VenueReadUnsupported
		cell.LiqLong, cell.LiqShort = "", ""
		cell.LongStatus = VenueReadUnsupported
		cell.LongPct = ""
		cell.GapStatus = VenueReadUnsupported
		cell.GapPct = ""
		cell.LiqHeatStatus = VenueReadUnsupported
		cell.LiqGrade = ""
	}
	if markOnly == 0 || markOnly != len(expected) {
		return
	}
	row.LiqDefinite = false
	row.LiqLong, row.LiqShort = "", ""
	row.LongDefinite = false
	row.LongPct = ""
	row.GapDefinite = false
	row.GapPct, row.GapScope = "", ""
	row.LiqHeatDefinite = false
	row.LiqHeat, row.LiqHeatRatio, row.LiqHeatVenue = "", "", ""
}

func coverageExchange(expected []Exchange) string {
	if len(expected) == 1 {
		return string(expected[0])
	}
	return "all"
}

func indexVenueReads(c MarketCoverage) map[Exchange]string {
	out := map[Exchange]string{}
	for _, v := range c.Venues {
		out[v.Exchange] = v.Status
	}
	return out
}

// derivativesOISampleMissing names venues whose history has no sample
// within slack of now-dur. A nil series counts as missing. The names stay
// in venue order so a partial sum is not presented as the market change.
func derivativesOISampleMissing(expected []Exchange, byEx map[Exchange]DerivativesVenueInput, dur time.Duration, now time.Time) []string {
	target := now.Add(-dur)
	slack := OpenInterestSampleSlack(dur)
	var missing []string
	for _, ex := range expected {
		v := byEx[ex]
		if v.OI == nil {
			missing = append(missing, string(ex))
			continue
		}
		if _, complete := FindOpenInterestSample(v.OI.History, target, slack); !complete {
			missing = append(missing, string(ex))
		}
	}
	return missing
}

func derivativesOIChange(expected []Exchange, byEx map[Exchange]DerivativesVenueInput, windowID string, dur time.Duration, now time.Time) (string, string, bool) {
	var pastV, curV float64
	for _, ex := range expected {
		v := byEx[ex]
		if v.OI == nil {
			return "", "", false
		}
		w := SummarizeOpenInterestWindow(v.OI, windowID, dur, now)
		if !w.Complete {
			return "", "", false
		}
		past, _ := FindOpenInterestSample(v.OI.History, now.Add(-dur), OpenInterestSampleSlack(dur))
		pastV += past.Value
		curV += v.OI.Current.Value
	}
	delta := curV - pastV
	pct, ok := oiPctChange(curV, pastV)
	if !ok {
		return "", "", false
	}
	return FormatSignedQty(delta), FormatSignedPct(pct), true
}

func derivativesLongShare(expected []Exchange, byEx map[Exchange]DerivativesVenueInput, oiDefinite bool) (float64, bool) {
	var weighted, weight, sum float64
	n := 0
	for _, ex := range expected {
		v := byEx[ex]
		if !v.LongShortOK {
			return 0, false
		}
		sum += v.LongShare
		n++
		if oiDefinite && v.OIValue > 0 {
			weighted += v.LongShare * v.OIValue
			weight += v.OIValue
		}
	}
	if n == 0 {
		return 0, false
	}
	if weight > 0 {
		return weighted / weight, true
	}
	return sum / float64(n), true
}

func derivativesTakerBuy(expected []Exchange, byEx map[Exchange]DerivativesVenueInput) (float64, bool) {
	var buy, sell float64
	for _, ex := range expected {
		v := byEx[ex]
		if !v.TakerOK {
			return 0, false
		}
		buy += v.TakerBuy
		sell += v.TakerSell
	}
	tot := buy + sell
	if tot <= 0 {
		return 0, false
	}
	return buy / tot, true
}

func derivativesFunding(expected []Exchange, byEx map[Exchange]DerivativesVenueInput, oiDefinite bool) (float64, string) {
	var weighted, weight, sum float64
	n := 0
	for _, ex := range expected {
		v := byEx[ex]
		if !v.FundingOK {
			continue
		}
		sum += v.FundingRate
		n++
		if oiDefinite && v.OIValue > 0 {
			weighted += v.FundingRate * v.OIValue
			weight += v.OIValue
		}
	}
	if weight > 0 {
		return weighted / weight, DerivativesWeightOI
	}
	if n == 0 {
		return 0, DerivativesWeightEqual
	}
	return sum / float64(n), DerivativesWeightEqual
}

func derivativesPrice(expected []Exchange, byEx map[Exchange]DerivativesVenueInput, fundCover MarketCoverage) (string, string) {
	status := indexVenueReads(fundCover)
	var price float64
	var priceEx Exchange
	var bestOI float64
	picked := false
	for _, ex := range expected {
		v := byEx[ex]
		if status[ex] != VenueReadFresh || v.MarkPrice <= 0 {
			continue
		}
		oi := v.OIValue
		if !v.OIOK {
			oi = 0
		}
		if !picked || oi > bestOI {
			price = v.MarkPrice
			priceEx = ex
			bestOI = oi
			picked = true
		}
	}
	if !picked {
		return "", ""
	}
	return formatQty(price), string(priceEx)
}

func derivativesBoardCoverage(expected []Exchange, rows []DerivativesCoin, now time.Time) MarketCoverage {
	if len(rows) == 0 {
		return CoverVenues(expected, nil, now, VenueDerivativesMaxAge)
	}
	samples := make([]VenueSample, 0, len(expected))
	for _, ex := range expected {
		missing := false
		stale := false
		var newest time.Time
		for _, row := range rows {
			status := VenueReadMissing
			var at time.Time
			for _, v := range row.Coverage.Venues {
				if v.Exchange == ex {
					status = v.Status
					at = v.AsOf
					break
				}
			}
			switch status {
			case VenueReadMissing:
				missing = true
			case VenueReadStale:
				stale = true
			}
			if !at.IsZero() && at.After(newest) {
				newest = at
			}
		}
		sample := VenueSample{Exchange: ex}
		if !missing && !stale {
			sample.OK = true
			sample.At = newest
		} else if stale && !missing {
			sample.OK = true
			sample.At = now.Add(-VenueDerivativesMaxAge - time.Second)
		}
		samples = append(samples, sample)
	}
	return CoverVenues(expected, samples, now, VenueDerivativesMaxAge)
}
