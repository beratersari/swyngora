package domain

import (
	"fmt"
	"math"
	"strings"
)

const (
	// PaperRouteAll is the exchange value for a best-cost spot market order.
	PaperRouteAll = "all"
	// PaperRouteSkipQuote means the venue's pair is outside this book's currency.
	PaperRouteSkipQuote = "quote"
)

// PaperRouteVenues are the spot books a best-cost market order can split across.
func PaperRouteVenues() []Exchange {
	return []Exchange{ExchangeBinance, ExchangeCoinbase, ExchangeBybit}
}

// IsPaperRouteExchange reports a request that asks for the best-cost split.
// all and best are the same route. Other order types stay on one venue.
func IsPaperRouteExchange(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case PaperRouteAll, "best":
		return true
	default:
		return false
	}
}

// QuotesAlias reports two cash units the paper book treats as one.
// USDT, USDC, and USD share a unit. TRY does not.
func QuotesAlias(a, b string) bool {
	left := AliasFxCode(a)
	right := AliasFxCode(b)
	return left != "" && left == right
}

// CanonicalRouteSymbol is the Binance-form pair a best-cost order starts from.
// BTC-USD and BTCUSDT both become BTCUSDT. A bare equity ticker is rejected.
func CanonicalRouteSymbol(symbol string) (string, error) {
	canon := CrossVenueSymbol(ExchangeBinance, symbol)
	if canon == "" {
		return "", fmt.Errorf("%w: symbol is required", ErrInvalidArgument)
	}
	_, quote := SplitBaseQuote(ExchangeBinance, canon)
	if quote == "" {
		return "", fmt.Errorf("%w: best cost is for a crypto pair", ErrInvalidArgument)
	}
	return canon, nil
}

// RouteSymbolMatchesBook accepts a canonical pair whose quote aliases the book currency.
func RouteSymbolMatchesBook(canon, currency string) error {
	_, quote := SplitBaseQuote(ExchangeBinance, canon)
	if quote == "" || !QuotesAlias(currency, quote) {
		shown := strings.ToUpper(strings.TrimSpace(currency))
		if shown == "" {
			shown = currency
		}
		return fmt.Errorf("%w: pair quote %s does not match portfolio currency %s", ErrInvalidArgument, quote, shown)
	}
	return nil
}

// ParseMaxSlippagePct turns a percent (0.5 = 0.5%) into a fraction.
// Zero means the route has no price cap. A negative or non-finite value is rejected.
func ParseMaxSlippagePct(pct float64) (float64, error) {
	if pct == 0 {
		return 0, nil
	}
	if math.IsNaN(pct) || math.IsInf(pct, 0) || pct < 0 || pct > 100 {
		return 0, fmt.Errorf("%w: max slippage must be from 0 to 100 percent", ErrInvalidArgument)
	}
	return pct / 100, nil
}

// ParseRouteCompare returns a spot venue to price the whole order against.
// An empty or unknown value means the preview has no comparison.
func ParseRouteCompare(raw string) Exchange {
	ex := ParseExchange(raw)
	for _, v := range PaperRouteVenues() {
		if v == ex {
			return ex
		}
	}
	return ""
}

// PaperRouteBook is one venue's visible spot book for a best-cost walk.
// CapQty nil means the venue can take as much as the book holds (a buy).
// A sell passes the free quantity on that venue, including zero.
// Missing names a book that did not load. Skipped names a book left out on purpose.
type PaperRouteBook struct {
	Exchange Exchange
	Symbol   string
	Quote    string
	FeeRate  float64
	Bids     []PriceLevel
	Asks     []PriceLevel
	CapQty   *float64
	Missing  string
	Skipped  string
	Detail   string
}

// RoutedFill is one venue's paper trade inside a best-cost order.
type RoutedFill struct {
	Position *Position
	Trade    Trade
	Lots     *LotOps
}

// PaperRouteLeg is the size one venue takes, at the average of the levels it filled.
type PaperRouteLeg struct {
	Exchange             Exchange
	Symbol               string
	Quote                string
	Aliased              bool
	Quantity             float64
	FillPrice            float64
	Notional             float64
	Fee                  float64
	FeeRate              float64
	CashDebit            float64
	CashCredit           float64
	LevelsUsed           int
	BestPrice            float64
	EndPrice             float64
	EstimatedRealizedPnL *float64
}

// PaperRouteSkip is a venue the walk did not use, with a short reason.
type PaperRouteSkip struct {
	Exchange Exchange
	Symbol   string
	Reason   string
	Detail   string
}

// PaperRoute is the lowest-cost split of one base quantity.
type PaperRoute struct {
	Side              TradeSide
	RequestedQuantity float64
	FilledQuantity    float64
	Exhausted         bool
	InventoryShort    bool
	Missing           []string
	Skipped           []PaperRouteSkip
	Legs              []PaperRouteLeg
	Notional          float64
	Fee               float64
	CashDebit         float64
	CashCredit        float64
	AveragePrice      float64
	VisibleQuantity   float64
	LevelsUsed        int
	// MaxSlippage is a fraction (0.005 = 0.5%). Zero means no price cap.
	MaxSlippage float64
	// TouchPrice is the best ask for a buy or the best bid for a sell.
	TouchPrice float64
	// LimitPrice is the furthest raw price the cap allows. Zero when there is no cap.
	LimitPrice float64
	// InsideQuantity is the visible size at or better than LimitPrice.
	InsideQuantity float64
	// BeyondQuantity is the visible size past the cap.
	BeyondQuantity float64
	// SlippageLimited means the size inside the cap is shorter than the order.
	SlippageLimited bool
}

// PaperRouteCompare is the same order filled on one venue.
// SavedCash is positive when the split spends less on a buy or receives more on a sell.
type PaperRouteCompare struct {
	Exchange       Exchange
	Symbol         string
	Known          bool
	Cash           float64
	SavedCash      float64
	Exhausted      bool
	InventoryShort bool
}

type routeLevel struct {
	exchange Exchange
	symbol   string
	quote    string
	aliased  bool
	price    float64
	quantity float64
	fee      float64
}

// RoutePaperBooks walks the merged books by fee-adjusted price.
// A buy takes the lowest price×(1+fee). A sell takes the highest price×(1−fee).
// Levels on one venue fold into one leg. A size the books cannot hold sets Exhausted.
// A sell that asks for more than the free quantity sets InventoryShort.
// maxSlippage is a fraction. A buy skips asks above the best ask by more than that.
// A sell skips bids below the best bid by more than that. Zero means no price cap.
// SlippageLimited is set when the size inside that cap is shorter than the order
// and some size sits past it.
func RoutePaperBooks(side TradeSide, qty float64, bookCurrency string, maxSlippage float64, books []PaperRouteBook) PaperRoute {
	if side != TradeSideSell {
		side = TradeSideBuy
	}
	if maxSlippage < 0 || math.IsNaN(maxSlippage) || math.IsInf(maxSlippage, 0) {
		maxSlippage = 0
	}
	out := PaperRoute{Side: side, RequestedQuantity: qty, Legs: []PaperRouteLeg{}, MaxSlippage: maxSlippage}
	if qty <= 0 || math.IsNaN(qty) || math.IsInf(qty, 0) {
		out.Exhausted = true
		return out
	}
	var levels []routeLevel
	var held float64
	sell := side == TradeSideSell
	for _, b := range books {
		if strings.TrimSpace(b.Missing) != "" {
			out.Missing = append(out.Missing, string(b.Exchange))
			continue
		}
		if strings.TrimSpace(b.Skipped) != "" {
			out.Skipped = append(out.Skipped, PaperRouteSkip{
				Exchange: b.Exchange, Symbol: b.Symbol, Reason: b.Skipped, Detail: b.Detail,
			})
			continue
		}
		fee := clampRouteFee(b.FeeRate)
		aliased := b.Quote != "" && bookCurrency != "" && !strings.EqualFold(b.Quote, bookCurrency) && QuotesAlias(b.Quote, bookCurrency)
		sideLevels := b.Asks
		if sell {
			sideLevels = b.Bids
		}
		for _, lv := range sideLevels {
			if !validRouteLevel(lv) {
				continue
			}
			out.VisibleQuantity += lv.Quantity
			levels = append(levels, routeLevel{
				exchange: b.Exchange, symbol: b.Symbol, quote: b.Quote, aliased: aliased,
				price: lv.Price, quantity: lv.Quantity, fee: fee,
			})
		}
		if sell {
			if b.CapQty != nil && *b.CapQty > 0 {
				held += *b.CapQty
			}
		}
	}
	if sell && qty > held+1e-12 {
		out.InventoryShort = true
	}
	out.TouchPrice = routeTouch(side, levels)
	if maxSlippage > 0 && out.TouchPrice > 0 {
		if sell {
			out.LimitPrice = out.TouchPrice * (1 - maxSlippage)
		} else {
			out.LimitPrice = out.TouchPrice * (1 + maxSlippage)
		}
	}
	sortRouteLevels(side, levels)
	remainCap := map[Exchange]float64{}
	limited := map[Exchange]bool{}
	for _, b := range books {
		if b.Missing != "" || b.Skipped != "" || b.CapQty == nil {
			continue
		}
		limited[b.Exchange] = true
		remainCap[b.Exchange] = *b.CapQty
	}
	type acc struct {
		leg PaperRouteLeg
	}
	var order []Exchange
	by := map[Exchange]*acc{}
	remain := qty
	for _, lv := range levels {
		if !routePriceAllowed(side, lv.price, out.TouchPrice, maxSlippage) {
			out.BeyondQuantity += lv.quantity
			continue
		}
		out.InsideQuantity += lv.quantity
		if remain <= 1e-12 {
			continue
		}
		room := lv.quantity
		if limited[lv.exchange] {
			room = remainCap[lv.exchange]
			if lv.quantity < room {
				room = lv.quantity
			}
		}
		take := room
		if remain < take {
			take = remain
		}
		if take <= 1e-12 {
			continue
		}
		if limited[lv.exchange] {
			remainCap[lv.exchange] -= take
		}
		remain -= take
		slot := by[lv.exchange]
		if slot == nil {
			order = append(order, lv.exchange)
			slot = &acc{leg: PaperRouteLeg{
				Exchange: lv.exchange, Symbol: lv.symbol, Quote: lv.quote, Aliased: lv.aliased,
				FeeRate: lv.fee, BestPrice: lv.price,
			}}
			by[lv.exchange] = slot
		}
		slot.leg.Quantity += take
		slot.leg.Notional += take * lv.price
		slot.leg.LevelsUsed++
		slot.leg.EndPrice = lv.price
		out.LevelsUsed++
	}
	for _, ex := range order {
		leg := by[ex].leg
		if leg.Quantity <= 0 {
			continue
		}
		leg.FillPrice = leg.Notional / leg.Quantity
		leg.Fee = FeeAmount(leg.Quantity, leg.FillPrice, leg.FeeRate)
		if side == TradeSideSell {
			leg.CashCredit = SellCashCredit(leg.Quantity, leg.FillPrice, leg.FeeRate)
			out.CashCredit += leg.CashCredit
		} else {
			leg.CashDebit = BuyCashDebit(leg.Quantity, leg.FillPrice, leg.FeeRate)
			out.CashDebit += leg.CashDebit
		}
		out.Notional += leg.Notional
		out.Fee += leg.Fee
		out.FilledQuantity += leg.Quantity
		out.Legs = append(out.Legs, leg)
	}
	if out.FilledQuantity > 0 {
		out.AveragePrice = out.Notional / out.FilledQuantity
	}
	if remain > 1e-12 {
		priceBlocked := maxSlippage > 0 && out.BeyondQuantity > 1e-12 && out.InsideQuantity+1e-12 < qty
		if priceBlocked {
			out.SlippageLimited = true
		}
		fullBookShort := out.VisibleQuantity+1e-12 < qty
		if out.InventoryShort {
			if fullBookShort {
				out.Exhausted = true
			}
		} else if !priceBlocked || fullBookShort {
			out.Exhausted = true
		}
	}
	return out
}

// ComparePaperRoute prices the whole quantity on one venue.
// SavedCash is set only when both that venue and the split can fill the size.
func ComparePaperRoute(full PaperRoute, compare Exchange, side TradeSide, qty float64, bookCurrency string, maxSlippage float64, books []PaperRouteBook) PaperRouteCompare {
	out := PaperRouteCompare{Exchange: compare}
	var one *PaperRouteBook
	for i := range books {
		if books[i].Exchange == compare {
			one = &books[i]
			break
		}
	}
	if one == nil || strings.TrimSpace(one.Missing) != "" || strings.TrimSpace(one.Skipped) != "" {
		return out
	}
	out.Known = true
	out.Symbol = one.Symbol
	single := RoutePaperBooks(side, qty, bookCurrency, maxSlippage, []PaperRouteBook{*one})
	out.Exhausted = single.Exhausted
	out.InventoryShort = single.InventoryShort
	if full.Exhausted || full.InventoryShort || full.SlippageLimited || single.Exhausted || single.InventoryShort || single.SlippageLimited || full.FilledQuantity+1e-12 < qty {
		return out
	}
	if side == TradeSideSell {
		out.Cash = single.CashCredit
		out.SavedCash = full.CashCredit - single.CashCredit
		return out
	}
	out.Cash = single.CashDebit
	out.SavedCash = single.CashDebit - full.CashDebit
	return out
}

// ExplainPaperRoute is the English sentence stored with the preview for the assistant.
func ExplainPaperRoute(bookCurrency string, route PaperRoute) string {
	if len(route.Legs) == 0 {
		if route.SlippageLimited {
			return fmt.Sprintf("Not enough size within %g%% of the best price. Nothing is filled.", route.MaxSlippage*100)
		}
		if len(route.Missing) > 0 {
			return fmt.Sprintf("No book could take this size. Missing: %s.", strings.Join(route.Missing, ", "))
		}
		return "No book could take this size."
	}
	names := make([]string, len(route.Legs))
	aliased := false
	for i, leg := range route.Legs {
		names[i] = string(leg.Exchange)
		if leg.Aliased {
			aliased = true
		}
	}
	msg := fmt.Sprintf("Split by fee-adjusted price across %s.", strings.Join(names, ", "))
	if aliased {
		cur := strings.ToUpper(strings.TrimSpace(bookCurrency))
		if cur == "" {
			cur = "this"
		}
		msg += fmt.Sprintf(" A venue quoted in another stable is booked 1:1 in this %s book.", cur)
	}
	if len(route.Missing) > 0 {
		msg += fmt.Sprintf(" Missing book: %s.", strings.Join(route.Missing, ", "))
	}
	if route.SlippageLimited {
		msg += fmt.Sprintf(" Not enough size within %g%% of the best price, so nothing is filled.", route.MaxSlippage*100)
	}
	return msg
}

func sortRouteLevels(side TradeSide, levels []routeLevel) {
	// Insertion sort keeps the walk obvious for the small ladders we merge.
	for i := 1; i < len(levels); i++ {
		item := levels[i]
		j := i
		for j > 0 && routeLevelBefore(side, item, levels[j-1]) {
			levels[j] = levels[j-1]
			j--
		}
		levels[j] = item
	}
}

func routeLevelBefore(side TradeSide, a, b routeLevel) bool {
	ae := routeEffective(side, a.price, a.fee)
	be := routeEffective(side, b.price, b.fee)
	if ae != be {
		if side == TradeSideSell {
			return ae > be
		}
		return ae < be
	}
	if a.exchange != b.exchange {
		return a.exchange < b.exchange
	}
	return false
}

func routeTouch(side TradeSide, levels []routeLevel) float64 {
	var best float64
	for _, lv := range levels {
		if best == 0 || (side == TradeSideSell && lv.price > best) || (side != TradeSideSell && lv.price < best) {
			best = lv.price
		}
	}
	return best
}

func routePriceAllowed(side TradeSide, price, touch, maxSlippage float64) bool {
	if maxSlippage <= 0 || touch <= 0 {
		return true
	}
	pad := touch * 1e-12
	if side == TradeSideSell {
		return price+pad >= touch*(1-maxSlippage)
	}
	return price <= touch*(1+maxSlippage)+pad
}

func routeEffective(side TradeSide, price, fee float64) float64 {
	if side == TradeSideSell {
		return price * (1 - fee)
	}
	return price * (1 + fee)
}

func clampRouteFee(fee float64) float64 {
	if math.IsNaN(fee) || math.IsInf(fee, 0) || fee < 0 || fee >= 1 {
		return 0
	}
	return fee
}

func validRouteLevel(lv PriceLevel) bool {
	if lv.Price <= 0 || lv.Quantity <= 0 {
		return false
	}
	if math.IsNaN(lv.Price) || math.IsNaN(lv.Quantity) || math.IsInf(lv.Price, 0) || math.IsInf(lv.Quantity, 0) {
		return false
	}
	return true
}
