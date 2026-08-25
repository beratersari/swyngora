package domain

import (
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	// PriceDiffMaxLimitedBy* explain why max size stopped growing.
	PriceDiffMaxLimitedByProfit   = "profit"
	PriceDiffMaxLimitedByBuyBook  = "buy_book"
	PriceDiffMaxLimitedBySellBook = "sell_book"
	PriceDiffMaxLimitedByBoth     = "both_books"
	PriceDiffMaxLimitedByEmpty    = "empty_book"

	priceDiffQuoteNote = "Simulated market buy on one venue and market sell on the other using live resting depth. Not a quote, fill, or financial advice. Visible books may be thinner than the real market."
)

// PriceDiffQuoteQuery is a two-venue executable-size request.
// Provide exactly one of Notional (quote currency spent on the buy book,
// before the buy fee) or Quantity (base size on both legs).
type PriceDiffQuoteQuery struct {
	Symbol        string
	BuyExchange   Exchange
	SellExchange  Exchange
	BuyFeePct     float64
	SellFeePct    float64
	Notional      float64
	Quantity      float64
	MinNetDiffPct float64 // optional; used only to flag whether the fill still meets the watch
}

// PriceDiffQuote is a simulated buy-on-A / sell-on-B fill from two spot books.
type PriceDiffQuote struct {
	Symbol              string
	BuyExchange         Exchange
	SellExchange        Exchange
	RequestedNotional   string
	RequestedQuantity   string
	FilledQuantity      string
	FilledRequested     bool
	AverageBuyPrice     string
	AverageSellPrice    string
	BestAsk             string
	BestBid             string
	BuySlippagePct      float64
	SellSlippagePct     float64
	SlippagePct         float64 // buy + sell adverse vs top of book
	BuyNotional         string
	SellNotional        string
	BuyFeePct           float64
	SellFeePct          float64
	BuyFee              string
	SellFee             string
	CostAfterFees       string
	ProceedsAfterFees   string
	ProfitAfterFees     string
	ProfitPct           float64
	GrossProfit         string
	GrossPct            float64
	BuyExhausted        bool
	SellExhausted       bool
	Profitable          bool
	Executable          bool
	MeetsMinNet         bool
	MinNetDiffPct       float64
	Live                bool
	BuyLive             bool
	SellLive            bool
	AsOf                time.Time
	VisibleBuyQty       string
	VisibleBuyNotional  string
	VisibleSellQty      string
	VisibleSellNotional string
	MaxQuantity         string
	MaxNotional         string
	MaxAverageBuyPrice  string
	MaxAverageSellPrice string
	MaxProfitAfterFees  string
	MaxProfitPct        float64
	MaxLimitedBy        string
	UsedNotional        string
	UnusedNotional      string
	UsedQuantity        string
	UnusedQuantity      string
	UsedPct             float64
	Note                string
	profitAmount        float64
	usedNotionalAmount  float64
}

// ValidatePriceDiffQuoteSize requires exactly one of notional or quantity.
func ValidatePriceDiffQuoteSize(notional, quantity float64) error {
	return ValidateImpactSize(quantity, notional)
}

// ValidatePriceDiffQuoteRoute checks venues and fees for an executable quote.
func ValidatePriceDiffQuoteRoute(buy, sell Exchange, buyFeePct, sellFeePct float64) error {
	if !IsValidExchange(string(buy)) || !IsValidExchange(string(sell)) {
		return fmt.Errorf("%w: buyExchange and sellExchange must be known venues", ErrInvalidArgument)
	}
	if IsEquityExchange(buy) || IsEquityExchange(sell) {
		return fmt.Errorf("%w: price-diff quotes need crypto spot books (binance, coinbase, bybit)", ErrInvalidArgument)
	}
	if buy == sell {
		return fmt.Errorf("%w: buyExchange and sellExchange must differ", ErrInvalidArgument)
	}
	if buyFeePct < 0 || sellFeePct < 0 || buyFeePct > MaxPriceDiffFeePct || sellFeePct > MaxPriceDiffFeePct ||
		math.IsNaN(buyFeePct) || math.IsNaN(sellFeePct) || math.IsInf(buyFeePct, 0) || math.IsInf(sellFeePct, 0) {
		return fmt.Errorf("%w: fee percent out of range", ErrInvalidArgument)
	}
	return nil
}

// QuotePriceDiffRoute walks the buy venue asks and the sell venue bids for a size.
func QuotePriceDiffRoute(q PriceDiffQuoteQuery, buyBook, sellBook *RawOrderBook) (*PriceDiffQuote, error) {
	q.Symbol = strings.TrimSpace(q.Symbol)
	if q.Symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", ErrInvalidArgument)
	}
	if err := ValidatePriceDiffQuoteRoute(q.BuyExchange, q.SellExchange, q.BuyFeePct, q.SellFeePct); err != nil {
		return nil, err
	}
	if err := ValidatePriceDiffQuoteSize(q.Notional, q.Quantity); err != nil {
		return nil, err
	}
	if q.MinNetDiffPct < 0 || math.IsNaN(q.MinNetDiffPct) || math.IsInf(q.MinNetDiffPct, 0) {
		return nil, fmt.Errorf("%w: minNetDiffPct is invalid", ErrInvalidArgument)
	}

	buyLevels := quoteLevels(ImpactSideBuy, q.BuyExchange, buyBook)
	sellLevels := quoteLevels(ImpactSideSell, q.SellExchange, sellBook)

	out := &PriceDiffQuote{
		Symbol:        q.Symbol,
		BuyExchange:   q.BuyExchange,
		SellExchange:  q.SellExchange,
		BuyFeePct:     q.BuyFeePct,
		SellFeePct:    q.SellFeePct,
		MinNetDiffPct: q.MinNetDiffPct,
		Note:          priceDiffQuoteNote,
	}
	if q.Notional > 0 {
		out.RequestedNotional = formatQty(q.Notional)
	}
	if q.Quantity > 0 {
		out.RequestedQuantity = formatQty(q.Quantity)
	}
	if buyBook != nil {
		out.BuyLive = buyBook.Live
		out.AsOf = buyBook.FetchedAt
	}
	if sellBook != nil {
		out.SellLive = sellBook.Live
		if sellBook.FetchedAt.After(out.AsOf) {
			out.AsOf = sellBook.FetchedAt
		}
	}
	out.Live = out.BuyLive && out.SellLive

	bVisQty, bVisN := visibleSide(buyLevels)
	sVisQty, sVisN := visibleSide(sellLevels)
	out.VisibleBuyQty = formatQty(bVisQty)
	out.VisibleBuyNotional = formatQty(bVisN)
	out.VisibleSellQty = formatQty(sVisQty)
	out.VisibleSellNotional = formatQty(sVisN)

	max := maxProfitableSize(buyLevels, sellLevels, q.BuyFeePct, q.SellFeePct)
	out.MaxLimitedBy = max.limitedBy
	if max.qty > 0 {
		out.MaxQuantity = formatQty(max.qty)
		out.MaxNotional = formatQty(max.buyN)
		out.MaxAverageBuyPrice = formatQuotePrice(max.avgBuy)
		out.MaxAverageSellPrice = formatQuotePrice(max.avgSell)
		out.MaxProfitAfterFees = formatQty(max.profit)
		if max.buyN > 0 {
			cost := max.buyN * (1 + q.BuyFeePct/100)
			if cost > 0 {
				out.MaxProfitPct = round4(max.profit / cost * 100)
			}
		}
	}

	fill, err := matchRequestedSize(buyLevels, sellLevels, q.Notional, q.Quantity)
	if err != nil {
		return nil, err
	}
	fill = capFillToMax(fill, max, buyLevels, sellLevels, q.Notional, q.Quantity)
	applyRequestedFill(out, fill, q.BuyFeePct, q.SellFeePct, q.MinNetDiffPct, q.Notional, q.Quantity)
	applyUsedBudget(out, q.Notional, q.Quantity, fill, max)
	return out, nil
}

func quoteLevels(side string, ex Exchange, book *RawOrderBook) []ImpactSourceLevel {
	if book == nil {
		return nil
	}
	return CollectImpactLevels(side, []VenueRawBook{{Exchange: ex, Book: *book}})
}

func visibleSide(levels []ImpactSourceLevel) (qty, notional float64) {
	for _, lv := range levels {
		qty += lv.Quantity
		notional += lv.Price * lv.Quantity
	}
	return qty, notional
}

func formatQuotePrice(p float64) string {
	if p <= 0 || math.IsNaN(p) || math.IsInf(p, 0) {
		return ""
	}
	return formatFixed(p, decimalsForStep(p/10000)+1)
}

type quoteFill struct {
	qty, buyN, sellN, avgBuy, avgSell, bestAsk, bestBid float64
	buyExh, sellExh                                     bool
}

type maxFill struct {
	qty, buyN, sellN, avgBuy, avgSell, profit float64
	limitedBy                                 string
}

func walkLevels(levels []ImpactSourceLevel, quantity, notional float64) (qty, spent, avg, best float64, exhausted bool) {
	if len(levels) == 0 {
		return 0, 0, 0, 0, quantity > 0 || notional > 0
	}
	best = levels[0].Price
	remainQty := quantity
	remainQuote := notional
	useQuote := notional > 0 && quantity <= 0
	for _, lv := range levels {
		if lv.Price <= 0 || lv.Quantity <= 0 {
			continue
		}
		if useQuote {
			if remainQuote <= 1e-12 {
				break
			}
		} else if remainQty <= 1e-12 {
			break
		}
		take := lv.Quantity
		if useQuote {
			maxQty := remainQuote / lv.Price
			if maxQty < take {
				take = maxQty
			}
		} else if remainQty < take {
			take = remainQty
		}
		if take <= 0 {
			continue
		}
		cost := take * lv.Price
		qty += take
		spent += cost
		if useQuote {
			remainQuote -= cost
			if remainQuote < 0 {
				remainQuote = 0
			}
		} else {
			remainQty -= take
		}
	}
	if qty > 0 {
		avg = spent / qty
	}
	if useQuote {
		exhausted = remainQuote > 1e-8
	} else {
		exhausted = remainQty > 1e-12
	}
	return qty, spent, avg, best, exhausted
}

func matchRequestedSize(buy, sell []ImpactSourceLevel, notional, quantity float64) (quoteFill, error) {
	var out quoteFill
	if len(buy) > 0 {
		out.bestAsk = buy[0].Price
	}
	if len(sell) > 0 {
		out.bestBid = sell[0].Price
	}

	var wantQty float64
	if notional > 0 && quantity <= 0 {
		bq, bSpend, _, _, bExh := walkLevels(buy, 0, notional)
		out.buyExh = bExh
		wantQty = bq
		if bq <= 0 {
			out.buyN = bSpend
			out.buyExh = true
			out.sellExh = len(sell) == 0
			return out, nil
		}
	} else {
		wantQty = quantity
	}

	bq, bSpend, bAvg, _, bExh := walkLevels(buy, wantQty, 0)
	sq, sRecv, sAvg, _, sExh := walkLevels(sell, wantQty, 0)
	matched := bq
	if sq < matched {
		matched = sq
	}
	if matched+1e-12 < wantQty {
		// One side was thinner — rematch both legs to the same filled size.
		bq, bSpend, bAvg, _, bExh = walkLevels(buy, matched, 0)
		sq, sRecv, sAvg, _, sExh = walkLevels(sell, matched, 0)
	}
	out.qty = matched
	out.buyN = bSpend
	out.sellN = sRecv
	out.avgBuy = bAvg
	out.avgSell = sAvg
	out.buyExh = bExh || (notional > 0 && quantity <= 0 && out.buyExh)
	out.sellExh = sExh
	if notional > 0 && quantity <= 0 && matched+1e-12 < wantQty {
		out.buyExh = out.buyExh || bExh
		out.sellExh = true
	}
	return out, nil
}

func applyRequestedFill(out *PriceDiffQuote, fill quoteFill, buyFeePct, sellFeePct, minNetPct, reqNotional, reqQty float64) {
	if fill.bestAsk > 0 {
		out.BestAsk = formatQuotePrice(fill.bestAsk)
	}
	if fill.bestBid > 0 {
		out.BestBid = formatQuotePrice(fill.bestBid)
	}
	if fill.qty <= 0 {
		out.FilledQuantity = "0"
		out.BuyNotional = "0"
		out.SellNotional = "0"
		out.ProfitAfterFees = "0"
		out.BuyExhausted = fill.buyExh || len(out.VisibleBuyQty) == 0 || out.VisibleBuyQty == "0"
		out.SellExhausted = fill.sellExh || len(out.VisibleSellQty) == 0 || out.VisibleSellQty == "0"
		return
	}
	out.FilledQuantity = formatQty(fill.qty)
	out.BuyNotional = formatQty(fill.buyN)
	out.SellNotional = formatQty(fill.sellN)
	out.AverageBuyPrice = formatQuotePrice(fill.avgBuy)
	out.AverageSellPrice = formatQuotePrice(fill.avgSell)
	out.BuyExhausted = fill.buyExh
	out.SellExhausted = fill.sellExh

	if fill.bestAsk > 0 && fill.avgBuy > 0 {
		out.BuySlippagePct = adversePct(fill.avgBuy, fill.bestAsk)
	}
	if fill.bestBid > 0 && fill.avgSell > 0 {
		out.SellSlippagePct = adversePct(fill.bestBid, fill.avgSell)
	}
	out.SlippagePct = round4(out.BuySlippagePct + out.SellSlippagePct)

	buyFee := fill.buyN * (buyFeePct / 100)
	sellFee := fill.sellN * (sellFeePct / 100)
	cost := fill.buyN + buyFee
	proceeds := fill.sellN - sellFee
	profit := proceeds - cost
	out.BuyFee = formatQty(buyFee)
	out.SellFee = formatQty(sellFee)
	out.CostAfterFees = formatQty(cost)
	out.ProceedsAfterFees = formatQty(proceeds)
	out.ProfitAfterFees = formatQty(profit)
	out.GrossProfit = formatQty(fill.sellN - fill.buyN)
	if fill.buyN > 0 {
		out.GrossPct = round4((fill.sellN/fill.buyN - 1) * 100)
	}
	if cost > 0 {
		out.ProfitPct = round4(profit / cost * 100)
	}
	out.profitAmount = profit
	out.Profitable = profit > 1e-8
	out.MeetsMinNet = out.Profitable && out.ProfitPct+1e-12 >= minNetPct

	filledReq := !fill.buyExh && !fill.sellExh
	if reqNotional > 0 && reqQty <= 0 {
		// Notional is buy-book spend before fees.
		filledReq = filledReq && fill.buyN+1e-6 >= reqNotional
	} else if reqQty > 0 {
		filledReq = filledReq && fill.qty+1e-12 >= reqQty
	}
	out.FilledRequested = filledReq
	out.Executable = filledReq && out.Profitable
}

func capFillToMax(fill quoteFill, max maxFill, buy, sell []ImpactSourceLevel, notional, quantity float64) quoteFill {
	if max.qty <= 0 {
		return fill
	}
	if notional > 0 && quantity <= 0 {
		if fill.buyN <= max.buyN+1e-8 {
			return fill
		}
		capped, err := matchRequestedSize(buy, sell, max.buyN, 0)
		if err != nil {
			return fill
		}
		capped.bestAsk = fill.bestAsk
		capped.bestBid = fill.bestBid
		return capped
	}
	if quantity > 0 && fill.qty > max.qty+1e-12 {
		capped, err := matchRequestedSize(buy, sell, 0, max.qty)
		if err != nil {
			return fill
		}
		capped.bestAsk = fill.bestAsk
		capped.bestBid = fill.bestBid
		return capped
	}
	return fill
}

func applyUsedBudget(out *PriceDiffQuote, reqNotional, reqQty float64, fill quoteFill, max maxFill) {
	if out == nil {
		return
	}
	usableN, usableQ := fill.buyN, fill.qty
	if max.qty <= 0 {
		usableN, usableQ = 0, 0
	}
	out.usedNotionalAmount = usableN
	if reqNotional > 0 && reqQty <= 0 {
		out.UsedNotional = formatQty(usableN)
		unused := reqNotional - usableN
		if unused < 0 {
			unused = 0
		}
		out.UnusedNotional = formatQty(unused)
		if reqNotional > 0 {
			pct := usableN / reqNotional * 100
			if pct > 100 {
				pct = 100
			}
			out.UsedPct = round4(pct)
		}
		return
	}
	if reqQty > 0 {
		out.UsedQuantity = formatQty(usableQ)
		unused := reqQty - usableQ
		if unused < 0 {
			unused = 0
		}
		out.UnusedQuantity = formatQty(unused)
		pct := usableQ / reqQty * 100
		if pct > 100 {
			pct = 100
		}
		out.UsedPct = round4(pct)
	}
}

// maxProfitableSize is the largest size whose cumulative after-fee profit is
// still positive. A later level that loses money on its own is still taken
// while the running total stays above zero; a partial last clip is allowed.
func maxProfitableSize(buy, sell []ImpactSourceLevel, buyFeePct, sellFeePct float64) maxFill {
	out := maxFill{limitedBy: PriceDiffMaxLimitedByEmpty}
	if len(buy) == 0 && len(sell) == 0 {
		return out
	}
	if len(buy) == 0 {
		out.limitedBy = PriceDiffMaxLimitedByBuyBook
		return out
	}
	if len(sell) == 0 {
		out.limitedBy = PriceDiffMaxLimitedBySellBook
		return out
	}

	bf := buyFeePct / 100
	sf := sellFeePct / 100
	const eps = 1e-12
	bi, si := 0, 0
	bLeft, sLeft := buy[0].Quantity, sell[0].Quantity
	out.limitedBy = PriceDiffMaxLimitedByProfit

	for bi < len(buy) && si < len(sell) {
		for bi < len(buy) && (buy[bi].Price <= 0 || bLeft <= eps) {
			bi++
			if bi < len(buy) {
				bLeft = buy[bi].Quantity
			}
		}
		for si < len(sell) && (sell[si].Price <= 0 || sLeft <= eps) {
			si++
			if si < len(sell) {
				sLeft = sell[si].Quantity
			}
		}
		if bi >= len(buy) || si >= len(sell) {
			break
		}
		pb, ps := buy[bi].Price, sell[si].Price
		take := bLeft
		if sLeft < take {
			take = sLeft
		}
		if take <= eps {
			break
		}
		const profitFloor = 1e-8
		p0 := out.sellN*(1-sf) - out.buyN*(1+bf)
		edge := ps*(1-sf) - pb*(1+bf)
		use := take
		stopAfter := false
		if edge < 0 {
			if p0 <= profitFloor {
				out.limitedBy = PriceDiffMaxLimitedByProfit
				break
			}
			// p0 + use*edge = profitFloor
			use = (profitFloor - p0) / edge
			if use <= eps {
				out.limitedBy = PriceDiffMaxLimitedByProfit
				break
			}
			if use > take {
				use = take
			} else {
				stopAfter = true
			}
		}
		out.qty += use
		out.buyN += use * pb
		out.sellN += use * ps
		bLeft -= use
		sLeft -= use
		if stopAfter {
			out.limitedBy = PriceDiffMaxLimitedByProfit
			break
		}
		if bLeft <= eps {
			bi++
			if bi < len(buy) {
				bLeft = buy[bi].Quantity
			}
		}
		if sLeft <= eps {
			si++
			if si < len(sell) {
				sLeft = sell[si].Quantity
			}
		}
	}

	if out.qty > 0 {
		out.avgBuy = out.buyN / out.qty
		out.avgSell = out.sellN / out.qty
		out.profit = out.sellN*(1-sf) - out.buyN*(1+bf)
	}
	buyDone := bi >= len(buy)
	sellDone := si >= len(sell)
	if out.qty <= 0 {
		out.limitedBy = PriceDiffMaxLimitedByProfit
		return out
	}
	if buyDone && sellDone {
		out.limitedBy = PriceDiffMaxLimitedByBoth
	} else if buyDone {
		out.limitedBy = PriceDiffMaxLimitedByBuyBook
	} else if sellDone {
		out.limitedBy = PriceDiffMaxLimitedBySellBook
	} else {
		out.limitedBy = PriceDiffMaxLimitedByProfit
	}
	return out
}

// PriceDiffQuoteScan is every buy/sell venue pair quoted at the same size.
type PriceDiffQuoteScan struct {
	Symbol            string
	RequestedNotional string
	RequestedQuantity string
	BestRoute         *PriceDiffQuote
	Routes            []PriceDiffQuote
	ProfitableCount   int
	VenueCount        int
	Note              string
}

const priceDiffScanNote = "Each route walks live buy-venue asks and sell-venue bids at the same size, including fees and visible depth. Routes are ranked by after-fee profit, then by how much of the requested money can be used. Not a fill or financial advice."

// ScanPriceDiffQuotes quotes every ordered crypto-spot pair from the given books.
func ScanPriceDiffQuotes(symbol string, notional, quantity float64, fees map[Exchange]float64, minNet float64, books map[Exchange]*RawOrderBook) (*PriceDiffQuoteScan, error) {
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", ErrInvalidArgument)
	}
	if err := ValidatePriceDiffQuoteSize(notional, quantity); err != nil {
		return nil, err
	}
	if minNet < 0 || math.IsNaN(minNet) || math.IsInf(minNet, 0) {
		return nil, fmt.Errorf("%w: minNetDiffPct is invalid", ErrInvalidArgument)
	}
	venues := make([]Exchange, 0, 3)
	for _, ex := range SupportedExchanges {
		if IsEquityExchange(ex) {
			continue
		}
		if _, ok := books[ex]; ok {
			venues = append(venues, ex)
		}
	}
	if len(venues) < 2 {
		return nil, fmt.Errorf("%w: need books from at least two crypto venues", ErrInvalidArgument)
	}
	out := &PriceDiffQuoteScan{
		Symbol:     symbol,
		VenueCount: len(venues),
		Routes:     make([]PriceDiffQuote, 0, len(venues)*(len(venues)-1)),
		Note:       priceDiffScanNote,
	}
	if notional > 0 {
		out.RequestedNotional = formatQty(notional)
	}
	if quantity > 0 {
		out.RequestedQuantity = formatQty(quantity)
	}
	for i := range venues {
		for j := range venues {
			if i == j {
				continue
			}
			buy, sell := venues[i], venues[j]
			q, err := QuotePriceDiffRoute(PriceDiffQuoteQuery{
				Symbol: symbol, BuyExchange: buy, SellExchange: sell,
				BuyFeePct: fees[buy], SellFeePct: fees[sell],
				Notional: notional, Quantity: quantity, MinNetDiffPct: minNet,
			}, books[buy], books[sell])
			if err != nil {
				continue
			}
			if q.Profitable {
				out.ProfitableCount++
			}
			out.Routes = append(out.Routes, *q)
		}
	}
	sortQuoteScan(out.Routes)
	if len(out.Routes) > 0 {
		best := out.Routes[0]
		out.BestRoute = &best
	}
	return out, nil
}

func sortQuoteScan(routes []PriceDiffQuote) {
	for i := 1; i < len(routes); i++ {
		for j := i; j > 0 && quoteScanLess(routes[j], routes[j-1]); j-- {
			routes[j], routes[j-1] = routes[j-1], routes[j]
		}
	}
}

// Higher after-fee profit first; if tied, more of the entered money used.
func quoteScanLess(a, b PriceDiffQuote) bool {
	if a.profitAmount != b.profitAmount {
		return a.profitAmount > b.profitAmount
	}
	if a.usedNotionalAmount != b.usedNotionalAmount {
		return a.usedNotionalAmount > b.usedNotionalAmount
	}
	return a.UsedPct > b.UsedPct
}
