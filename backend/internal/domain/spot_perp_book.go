package domain

import (
	"fmt"
	"math"
	"time"
)

const (
	DefaultSpotPerpRangePct = 1.0
	MinSpotPerpRangePct     = 0.1
	MaxSpotPerpRangePct     = 1.0

	SpotPerpSpot     = "spot"
	SpotPerpPerp     = "perp"
	SpotPerpEven     = "even"
	SpotPerpBid      = "bid"
	SpotPerpAsk      = "ask"
	SpotPerpBalanced = "balanced"

	// A book is even with the other when the gap is under 10% of the larger total.
	spotPerpEvenFrac = 0.10
)

// SpotPerpBandPcts are the ±% windows compared on both books.
var SpotPerpBandPcts = []float64{0.1, 0.5, 1.0}

// SpotPerpBook is resting bid and ask notional inside one band.
type SpotPerpBook struct {
	BidNotional  float64
	AskNotional  float64
	Total        float64
	ThinnerSide  string
	StrongerSide string
	Mid          float64
	CoverPct     float64
	VenueCount   int
	Complete     bool
}

// SpotPerpBand compares spot and perp at one distance from mid.
type SpotPerpBand struct {
	RangePct        float64
	SpotBid         float64
	SpotAsk         float64
	PerpBid         float64
	PerpAsk         float64
	ThinnerBook     string
	StrongerBook    string
	SpotThinnerSide string
	PerpThinnerSide string
	Complete        bool
}

// SpotPerpVenue is one exchange that has both a spot book and a perp book.
type SpotPerpVenue struct {
	Exchange        Exchange
	SpotBid         float64
	SpotAsk         float64
	PerpBid         float64
	PerpAsk         float64
	ThinnerBook     string
	SpotThinnerSide string
	PerpThinnerSide string
	Complete        bool
	Error           string
}

// SpotPerpLiquidity is spot vs perpetual resting size at the same % of mid.
type SpotPerpLiquidity struct {
	Symbol       string
	RangePct     float64
	AsOf         time.Time
	Spot         SpotPerpBook
	Perp         SpotPerpBook
	ThinnerBook  string
	StrongerBook string
	Ratio        float64 // perp total / spot total at the headline band
	Bands        []SpotPerpBand
	Venues       []SpotPerpVenue
	Summary      string
	Note         string
}

// ParseSpotPerpRangePct clamps the headline ±% (empty = 1). Bands stay 0.1 / 0.5 / 1.
func ParseSpotPerpRangePct(v float64) (float64, error) {
	if v == 0 {
		return DefaultSpotPerpRangePct, nil
	}
	if v != v || v < MinSpotPerpRangePct || v > MaxSpotPerpRangePct {
		return 0, fmt.Errorf("%w: rangePct must be between %g and %g", ErrInvalidArgument, MinSpotPerpRangePct, MaxSpotPerpRangePct)
	}
	return v, nil
}

// CompareSpotPerpLiquidity sums spot venues and perp venues inside the same ±% bands.
// A venue counts in a band only when its book reaches that % on both sides.
// Same-venue rows are Binance and Bybit, the venues that list both books.
func CompareSpotPerpLiquidity(symbol string, spotBooks, perpBooks []VenueRawBook, rangePct float64, asOf time.Time) SpotPerpLiquidity {
	if rangePct <= 0 {
		rangePct = DefaultSpotPerpRangePct
	}
	out := SpotPerpLiquidity{
		Symbol:   symbol,
		RangePct: rangePct,
		AsOf:     asOf.UTC(),
		Bands:    []SpotPerpBand{},
		Venues:   []SpotPerpVenue{},
		Note:     "Spot sums Binance, Coinbase, and Bybit. Perp sums Binance USD-M, Bybit linear, and OKX USDT-swap. A band counts a venue only when that book reaches the % on both sides. Binance and Bybit rows compare that venue's own spot book with its perp. Informational only — not financial advice.",
	}
	spotReady := readyBooks(spotBooks)
	perpReady := readyBooks(perpBooks)
	var headline *SpotPerpBand
	for _, pct := range SpotPerpBandPcts {
		if pct-rangePct > 1e-9 {
			continue
		}
		band := compareBand(pct, spotReady, perpReady)
		out.Bands = append(out.Bands, band)
		if band.Complete {
			copied := band
			headline = &copied
		}
	}
	if headline == nil && len(out.Bands) > 0 {
		last := out.Bands[len(out.Bands)-1]
		headline = &last
	}
	if headline != nil {
		out.RangePct = headline.RangePct
		out.Spot = bookFromTotals(headline.SpotBid, headline.SpotAsk, spotReady, headline.RangePct, headline.Complete)
		out.Perp = bookFromTotals(headline.PerpBid, headline.PerpAsk, perpReady, headline.RangePct, headline.Complete)
		out.ThinnerBook = headline.ThinnerBook
		out.StrongerBook = headline.StrongerBook
		if headline.SpotBid+headline.SpotAsk > 0 {
			out.Ratio = round4((headline.PerpBid + headline.PerpAsk) / (headline.SpotBid + headline.SpotAsk))
		}
	}
	out.Venues = sameVenueRows(spotBooks, perpBooks, out.RangePct)
	out.Summary = explainSpotPerp(out)
	return out
}

func readyBooks(books []VenueRawBook) []VenueRawBook {
	out := make([]VenueRawBook, 0, len(books))
	for _, b := range books {
		if b.Err != "" || (len(b.Book.Bids) == 0 && len(b.Book.Asks) == 0) {
			continue
		}
		out = append(out, b)
	}
	return out
}

func compareBand(pct float64, spotBooks, perpBooks []VenueRawBook) SpotPerpBand {
	spotBid, spotAsk, spotN := sumCovered(spotBooks, pct)
	perpBid, perpAsk, perpN := sumCovered(perpBooks, pct)
	complete := spotN > 0 && perpN > 0
	thinner, stronger := SpotPerpEven, SpotPerpEven
	spotSide, perpSide := SpotPerpBalanced, SpotPerpBalanced
	if complete {
		thinner, stronger = thinnerBook(spotBid+spotAsk, perpBid+perpAsk)
		spotSide, _ = sideLean(spotBid, spotAsk)
		perpSide, _ = sideLean(perpBid, perpAsk)
	}
	return SpotPerpBand{
		RangePct:        pct,
		SpotBid:         spotBid,
		SpotAsk:         spotAsk,
		PerpBid:         perpBid,
		PerpAsk:         perpAsk,
		ThinnerBook:     thinner,
		StrongerBook:    stronger,
		SpotThinnerSide: spotSide,
		PerpThinnerSide: perpSide,
		Complete:        complete,
	}
}

func sumCovered(books []VenueRawBook, pct float64) (bid, ask float64, n int) {
	for _, b := range books {
		mid := midPrice(b.Book)
		if mid <= 0 || !bandFullyCovered(bookCoverPct(b.Book, mid), pct) {
			continue
		}
		st := summarizeBand(b.Book, mid, pct)
		bid += st.bidNotional
		ask += st.askNotional
		n++
	}
	return bid, ask, n
}

func bookFromTotals(bid, ask float64, books []VenueRawBook, pct float64, complete bool) SpotPerpBook {
	thin, strong := sideLean(bid, ask)
	if !complete {
		thin, strong = SpotPerpBalanced, SpotPerpBalanced
	}
	var midSum float64
	var cover float64
	n := 0
	for _, b := range books {
		mid := midPrice(b.Book)
		if mid <= 0 || !bandFullyCovered(bookCoverPct(b.Book, mid), pct) {
			continue
		}
		midSum += mid
		c := bookCoverPct(b.Book, mid)
		if n == 0 || c < cover {
			cover = c
		}
		n++
	}
	mid := 0.0
	if n > 0 {
		mid = midSum / float64(n)
	}
	return SpotPerpBook{
		BidNotional:  bid,
		AskNotional:  ask,
		Total:        bid + ask,
		ThinnerSide:  thin,
		StrongerSide: strong,
		Mid:          mid,
		CoverPct:     cover,
		VenueCount:   n,
		Complete:     complete,
	}
}

func sideLean(bid, ask float64) (thinner, stronger string) {
	tot := bid + ask
	imb := 0.0
	if tot > 0 {
		imb = (bid - ask) / tot
	}
	switch weaker, _ := weakerFromImbalance(imb, tot); weaker {
	case LiquidityWeakerSell:
		return SpotPerpAsk, SpotPerpBid
	case LiquidityWeakerBuy:
		return SpotPerpBid, SpotPerpAsk
	default:
		return SpotPerpBalanced, SpotPerpBalanced
	}
}

func thinnerBook(spotTot, perpTot float64) (thinner, stronger string) {
	maxTot := spotTot
	if perpTot > maxTot {
		maxTot = perpTot
	}
	if maxTot <= 0 || math.Abs(spotTot-perpTot)/maxTot < spotPerpEvenFrac {
		return SpotPerpEven, SpotPerpEven
	}
	if spotTot < perpTot {
		return SpotPerpSpot, SpotPerpPerp
	}
	return SpotPerpPerp, SpotPerpSpot
}

func sameVenueRows(spotBooks, perpBooks []VenueRawBook, pct float64) []SpotPerpVenue {
	want := []Exchange{ExchangeBinance, ExchangeBybit}
	rows := make([]SpotPerpVenue, 0, len(want))
	for _, ex := range want {
		spot := findVenueBook(spotBooks, ex)
		perp := findVenueBook(perpBooks, ex)
		row := SpotPerpVenue{Exchange: ex}
		if spot == nil && perp == nil {
			continue
		}
		if spot == nil || perp == nil || spot.Err != "" || perp.Err != "" {
			row.Error = spotPerpVenueError(spot, perp)
			rows = append(rows, row)
			continue
		}
		band := compareBand(pct, []VenueRawBook{*spot}, []VenueRawBook{*perp})
		row.SpotBid = band.SpotBid
		row.SpotAsk = band.SpotAsk
		row.PerpBid = band.PerpBid
		row.PerpAsk = band.PerpAsk
		row.ThinnerBook = band.ThinnerBook
		row.SpotThinnerSide = band.SpotThinnerSide
		row.PerpThinnerSide = band.PerpThinnerSide
		row.Complete = band.Complete
		if !band.Complete {
			row.Error = "book does not reach this % on both sides"
		}
		rows = append(rows, row)
	}
	return rows
}

func spotPerpVenueError(spot, perp *VenueRawBook) string {
	switch {
	case spot == nil || spot.Err != "":
		if perp == nil || perp.Err != "" {
			return "spot and perp books are not both available"
		}
		return "spot book is not available"
	default:
		return "perp book is not available"
	}
}

func findVenueBook(books []VenueRawBook, ex Exchange) *VenueRawBook {
	for i := range books {
		if books[i].Exchange == ex {
			return &books[i]
		}
	}
	return nil
}

func explainSpotPerp(r SpotPerpLiquidity) string {
	if !r.Spot.Complete || !r.Perp.Complete {
		return fmt.Sprintf("Spot and perp do not both reach %.1f%% of mid, so this comparison is incomplete.", r.RangePct)
	}
	book := "about the same size"
	switch r.ThinnerBook {
	case SpotPerpSpot:
		book = "the spot book is thinner"
	case SpotPerpPerp:
		book = "the perp book is thinner"
	}
	return fmt.Sprintf(
		"Within %.1f%% of mid, %s (spot %s, perp %s). On spot, %s. On perp, %s.",
		r.RangePct,
		book,
		formatQty(r.Spot.Total),
		formatQty(r.Perp.Total),
		spotPerpSidePhrase(r.Spot.ThinnerSide),
		spotPerpSidePhrase(r.Perp.ThinnerSide),
	)
}

func spotPerpSidePhrase(side string) string {
	switch side {
	case SpotPerpBid:
		return "bids are thinner"
	case SpotPerpAsk:
		return "asks are thinner"
	default:
		return "the two sides are about even"
	}
}
