package domain

import (
	"sort"
	"strconv"
)

// VenueRawBook is one venue's raw depth for market-wide combine.
type VenueRawBook struct {
	Exchange Exchange
	Symbol   string
	Book     RawOrderBook
	Err      string
}

// CombinedWall is a wall tagged with the venue it came from.
type CombinedWall struct {
	Exchange          string
	Side              string
	Price             string
	Quantity          string
	Notional          string
	DistancePct       string
	Share             float64 // fraction of combined side notional
	Behavior          string
	AgeSeconds        float64
	PresentForSeconds float64
	VisibleSeconds    float64
	AppearCount       int
	Iceberg           bool
	IcebergRefills    int
	IcebergClip       string
}

// CombinedVenueSlice is one venue's contribution inside the shared price band.
type CombinedVenueSlice struct {
	Exchange    Exchange
	Symbol      string
	Live        bool
	Source      string
	BidNotional string
	AskNotional string
	BidQuantity string
	AskQuantity string
	Imbalance   float64
	Pressure    string
	BidLevels   int
	AskLevels   int
	Error       string
}

// CombinedOrderBookAnalysis is buy/sell pressure from all venues in one shared band.
type CombinedOrderBookAnalysis struct {
	Symbol           string
	RangePct         float64 // requested ±%
	MidPrice         string
	UsedLow          string
	UsedHigh         string
	UsedRangePct     float64 // symmetric ±% actually summed (min of the two sides)
	RequestedReached bool    // true when every venue covers the requested ±rangePct
	BidNotional      string
	AskNotional      string
	BidQuantity      string
	AskQuantity      string
	Imbalance        float64
	Pressure         string
	BidLevels        int
	AskLevels        int
	CoveredBidPct    string
	CoveredAskPct    string
	Walls            []CombinedWall
	Bands            []OrderBookBand
	Venues           []CombinedVenueSlice
	VenueCount       int
}

// SharedBookMid is the median mid of venues that have both sides (robust to one stale book).
func SharedBookMid(books []VenueRawBook) float64 {
	var mids []float64
	for _, b := range books {
		if b.Err != "" {
			continue
		}
		m := midPrice(b.Book)
		if m > 0 {
			mids = append(mids, m)
		}
	}
	return medianFloat(mids)
}

// bookExtent is the deepest bid and ask this snapshot actually reaches.
func bookExtent(raw RawOrderBook) (minBid, maxAsk float64, ok bool) {
	for _, lv := range raw.Bids {
		if !validLevel(lv) {
			continue
		}
		if minBid == 0 || lv.Price < minBid {
			minBid = lv.Price
		}
	}
	for _, lv := range raw.Asks {
		if !validLevel(lv) {
			continue
		}
		if lv.Price > maxAsk {
			maxAsk = lv.Price
		}
	}
	return minBid, maxAsk, minBid > 0 && maxAsk > 0
}

// resolveCombinedRange is the price window every contributing venue can reach.
// If that window covers the requested ±rangePct of mid, the requested band is used.
func resolveCombinedRange(mid, rangePct float64, books []VenueRawBook) (lo, hi float64, reached bool) {
	reqLo, reqHi := bandBounds(mid, rangePct)
	var commonLo, commonHi float64
	n := 0
	for _, b := range books {
		if b.Err != "" {
			continue
		}
		minBid, maxAsk, ok := bookExtent(b.Book)
		if !ok {
			continue
		}
		if n == 0 {
			commonLo, commonHi = minBid, maxAsk
		} else {
			if minBid > commonLo {
				commonLo = minBid
			}
			if maxAsk < commonHi {
				commonHi = maxAsk
			}
		}
		n++
	}
	if n == 0 || commonLo <= 0 || commonHi <= 0 || commonLo >= commonHi {
		return reqLo, reqHi, false
	}
	bidPct := (mid - commonLo) / mid * 100
	askPct := (commonHi - mid) / mid * 100
	if bidPct < 0 {
		bidPct = 0
	}
	if askPct < 0 {
		askPct = 0
	}
	usedPct := bidPct
	if askPct < usedPct {
		usedPct = askPct
	}
	// Same ±% on both sides so one side cannot include more distance than the other.
	if usedPct+1e-12 >= rangePct {
		return reqLo, reqHi, true
	}
	if usedPct <= 0 {
		return mid, mid, false
	}
	lo, hi = bandBounds(mid, usedPct)
	return lo, hi, false
}

// CombineOrderBooks sums bid/ask notional only inside the common reachable band.
func CombineOrderBooks(symbol string, mid, rangePct float64, books []VenueRawBook) CombinedOrderBookAnalysis {
	rangePct = ClampRangePct(rangePct)
	out := CombinedOrderBookAnalysis{
		Symbol:   symbol,
		RangePct: rangePct,
		Pressure: OrderBookPressureBalanced,
		Walls:    []CombinedWall{},
		Bands:    []OrderBookBand{},
		Venues:   []CombinedVenueSlice{},
	}
	if mid <= 0 {
		return out
	}
	out.MidPrice = formatFixed(mid, decimalsForStep(mid/10000)+1)
	lo, hi, reached := resolveCombinedRange(mid, rangePct, books)
	out.RequestedReached = reached
	out.UsedLow = formatFixed(lo, decimalsForStep(mid/10000)+1)
	out.UsedHigh = formatFixed(hi, decimalsForStep(mid/10000)+1)
	if mid > 0 && lo > 0 && hi > lo {
		out.UsedRangePct = round4((hi - lo) / 2 / mid * 100)
	}
	if lo <= 0 || hi <= lo {
		return out
	}

	var totBidN, totAskN, totBidQ, totAskQ float64
	var totBidL, totAskL int
	type bandAcc struct {
		bidN, askN, bidQ, askQ float64
		bidL, askL             int
	}
	acc := map[float64]*bandAcc{}
	for _, pct := range analysisBandPcts {
		bLo, bHi := bandBounds(mid, pct)
		if lo <= bLo+1e-12 && hi+1e-12 >= bHi {
			acc[pct] = &bandAcc{}
		}
	}
	var walls []CombinedWall

	for _, vb := range books {
		slice := CombinedVenueSlice{Exchange: vb.Exchange, Symbol: vb.Symbol, Error: vb.Err}
		if vb.Err != "" {
			out.Venues = append(out.Venues, slice)
			continue
		}
		st := summarizeRange(vb.Book, mid, lo, hi)
		slice.Live = vb.Book.Live
		slice.Source = vb.Book.Source
		if slice.Source == "" {
			slice.Source = OrderBookSourceREST
		}
		slice.BidNotional = formatQty(st.bidNotional)
		slice.AskNotional = formatQty(st.askNotional)
		slice.BidQuantity = formatQty(st.bidQty)
		slice.AskQuantity = formatQty(st.askQty)
		slice.Imbalance = st.imbalance
		slice.Pressure = pressureFromImbalance(st.imbalance)
		slice.BidLevels = st.bidLevels
		slice.AskLevels = st.askLevels
		out.Venues = append(out.Venues, slice)
		out.VenueCount++

		totBidN += st.bidNotional
		totAskN += st.askNotional
		totBidQ += st.bidQty
		totAskQ += st.askQty
		totBidL += st.bidLevels
		totAskL += st.askLevels
		for pct, a := range acc {
			b := summarizeBand(vb.Book, mid, pct)
			a.bidN += b.bidNotional
			a.askN += b.askNotional
			a.bidQ += b.bidQty
			a.askQ += b.askQty
			a.bidL += b.bidLevels
			a.askL += b.askLevels
		}
		for _, w := range detectWallsInRange(vb.Book, mid, lo, hi) {
			walls = append(walls, CombinedWall{
				Exchange:    string(vb.Exchange),
				Side:        w.Side,
				Price:       w.Price,
				Quantity:    w.Quantity,
				Notional:    w.Notional,
				DistancePct: w.DistancePct,
			})
		}
	}

	out.BidNotional = formatQty(totBidN)
	out.AskNotional = formatQty(totAskN)
	out.BidQuantity = formatQty(totBidQ)
	out.AskQuantity = formatQty(totAskQ)
	out.BidLevels = totBidL
	out.AskLevels = totAskL
	if mid > lo {
		out.CoveredBidPct = formatFixed(round4((mid-lo)/mid*100), 3)
	}
	if hi > mid {
		out.CoveredAskPct = formatFixed(round4((hi-mid)/mid*100), 3)
	}
	if tot := totBidN + totAskN; tot > 0 {
		out.Imbalance = round4((totBidN - totAskN) / tot)
	}
	out.Pressure = pressureFromImbalance(out.Imbalance)

	for _, w := range walls {
		n, _ := strconv.ParseFloat(w.Notional, 64)
		sideTot := totAskN
		if w.Side == "bid" {
			sideTot = totBidN
		}
		if sideTot > 0 {
			w.Share = round4(n / sideTot)
		}
		out.Walls = append(out.Walls, w)
	}
	sort.Slice(out.Walls, func(i, j int) bool {
		ni, _ := strconv.ParseFloat(out.Walls[i].Notional, 64)
		nj, _ := strconv.ParseFloat(out.Walls[j].Notional, 64)
		return ni > nj
	})
	if len(out.Walls) > MaxAnalysisWalls {
		out.Walls = out.Walls[:MaxAnalysisWalls]
	}

	for _, pct := range analysisBandPcts {
		a, ok := acc[pct]
		if !ok {
			continue
		}
		imb := 0.0
		if tot := a.bidN + a.askN; tot > 0 {
			imb = round4((a.bidN - a.askN) / tot)
		}
		out.Bands = append(out.Bands, OrderBookBand{
			RangePct:    pct,
			BidNotional: formatQty(a.bidN),
			AskNotional: formatQty(a.askN),
			BidQuantity: formatQty(a.bidQ),
			AskQuantity: formatQty(a.askQ),
			Imbalance:   imb,
			BidLevels:   a.bidL,
			AskLevels:   a.askL,
		})
	}
	return out
}
