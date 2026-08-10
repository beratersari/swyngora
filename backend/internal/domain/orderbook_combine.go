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
	Exchange    string
	Side        string
	Price       string
	Quantity    string
	Notional    string
	DistancePct string
	Share       float64 // fraction of combined side notional
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

// CombinedOrderBookAnalysis is buy/sell pressure from all venues in the same ±% band.
type CombinedOrderBookAnalysis struct {
	Symbol        string
	RangePct      float64
	MidPrice      string
	BidNotional   string
	AskNotional   string
	BidQuantity   string
	AskQuantity   string
	Imbalance     float64
	Pressure      string
	BidLevels     int
	AskLevels     int
	CoveredBidPct string
	CoveredAskPct string
	Walls         []CombinedWall
	Bands         []OrderBookBand
	Venues        []CombinedVenueSlice
	VenueCount    int
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

// CombineOrderBooks sums bid/ask notional from every venue inside ±rangePct of mid.
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

	var totBidN, totAskN, totBidQ, totAskQ float64
	var totBidL, totAskL int
	var coverBid, coverAsk float64
	type bandAcc struct {
		bidN, askN, bidQ, askQ float64
		bidL, askL             int
	}
	acc := map[float64]*bandAcc{}
	for _, pct := range analysisBandPcts {
		acc[pct] = &bandAcc{}
	}
	var walls []CombinedWall

	for _, vb := range books {
		slice := CombinedVenueSlice{Exchange: vb.Exchange, Symbol: vb.Symbol, Error: vb.Err}
		if vb.Err != "" {
			out.Venues = append(out.Venues, slice)
			continue
		}
		an := AnalyzeOrderBookAt(vb.Book, mid, rangePct)
		slice.Live = vb.Book.Live
		slice.Source = vb.Book.Source
		if slice.Source == "" {
			slice.Source = OrderBookSourceREST
		}
		slice.BidNotional = an.BidNotional
		slice.AskNotional = an.AskNotional
		slice.BidQuantity = an.BidQuantity
		slice.AskQuantity = an.AskQuantity
		slice.Imbalance = an.Imbalance
		slice.Pressure = an.Pressure
		slice.BidLevels = an.BidLevels
		slice.AskLevels = an.AskLevels
		out.Venues = append(out.Venues, slice)
		out.VenueCount++

		st := summarizeBand(vb.Book, mid, rangePct)
		totBidN += st.bidNotional
		totAskN += st.askNotional
		totBidQ += st.bidQty
		totAskQ += st.askQty
		totBidL += st.bidLevels
		totAskL += st.askLevels
		if st.coveredBid > coverBid {
			coverBid = st.coveredBid
		}
		if st.coveredAsk > coverAsk {
			coverAsk = st.coveredAsk
		}
		for _, pct := range analysisBandPcts {
			b := summarizeBand(vb.Book, mid, pct)
			a := acc[pct]
			a.bidN += b.bidNotional
			a.askN += b.askNotional
			a.bidQ += b.bidQty
			a.askQ += b.askQty
			a.bidL += b.bidLevels
			a.askL += b.askLevels
		}
		for _, w := range detectBandWalls(vb.Book, mid, rangePct) {
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
	out.CoveredBidPct = formatFixed(coverBid, 3)
	out.CoveredAskPct = formatFixed(coverAsk, 3)
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
		a := acc[pct]
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
