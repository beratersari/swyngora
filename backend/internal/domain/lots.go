package domain

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// Lot matching methods for paper sells.
const (
	LotMethodFIFO LotMethod = "fifo"
	LotMethodLIFO LotMethod = "lifo"
	DefaultLotMethod        = LotMethodFIFO
)

// LotMethod is fifo (oldest first) or lifo (newest first).
type LotMethod string

// TaxLot is one buy remaining in a paper book (quantity may shrink on partial sells).
type TaxLot struct {
	ID                string
	ClientID          string // book id
	Exchange          Exchange
	Symbol            string
	Quantity          float64 // remaining
	OriginalQuantity  float64
	Price             float64 // unit cost
	OpenedAt          time.Time
	SourceTradeID     string
	ClosedAt          *time.Time
}

// Open reports remaining quantity above the flat epsilon.
func (l TaxLot) Open() bool {
	return l.Quantity > PositionEpsilon && l.ClosedAt == nil
}

// TaxLotFill is how much of one lot a sell consumed.
type TaxLotFill struct {
	ID          string
	TradeID     string
	LotID       string
	Quantity    float64
	CostPrice   float64
	SellPrice   float64
	RealizedPnL float64
}

// LotOps is applied atomically with a trade fill.
type LotOps struct {
	Created []TaxLot
	Updated []TaxLot
	Fills   []TaxLotFill
}

// NormalizeLotMethod returns fifo|lifo. Empty → fifo.
func NormalizeLotMethod(s string) (LotMethod, error) {
	m := LotMethod(strings.ToLower(strings.TrimSpace(s)))
	if m == "" {
		return DefaultLotMethod, nil
	}
	switch m {
	case LotMethodFIFO, LotMethodLIFO:
		return m, nil
	default:
		return "", fmt.Errorf("%w: lotMethod must be fifo or lifo", ErrInvalidArgument)
	}
}

// SortLotsFIFO oldest OpenedAt first (stable by id).
func SortLotsFIFO(lots []TaxLot) {
	sort.SliceStable(lots, func(i, j int) bool {
		if !lots[i].OpenedAt.Equal(lots[j].OpenedAt) {
			return lots[i].OpenedAt.Before(lots[j].OpenedAt)
		}
		return lots[i].ID < lots[j].ID
	})
}

// SortLotsLIFO newest OpenedAt first (stable by id desc).
func SortLotsLIFO(lots []TaxLot) {
	sort.SliceStable(lots, func(i, j int) bool {
		if !lots[i].OpenedAt.Equal(lots[j].OpenedAt) {
			return lots[i].OpenedAt.After(lots[j].OpenedAt)
		}
		return lots[i].ID > lots[j].ID
	})
}

// OpenLotQuantity sums remaining open quantity.
func OpenLotQuantity(lots []TaxLot) float64 {
	var s float64
	for _, l := range lots {
		if l.Open() {
			s += l.Quantity
		}
	}
	return s
}

// MergeLotUpdates replaces lots in open with updated rows (same id).
func MergeLotUpdates(open, updated []TaxLot) []TaxLot {
	if len(updated) == 0 {
		return append([]TaxLot(nil), open...)
	}
	idx := map[string]TaxLot{}
	for _, u := range updated {
		idx[u.ID] = u
	}
	out := make([]TaxLot, 0, len(open))
	for _, l := range open {
		if u, ok := idx[l.ID]; ok {
			out = append(out, u)
			continue
		}
		out = append(out, l)
	}
	return out
}

// AvgCostFromLots is remaining cost / remaining qty (0 if flat).
func AvgCostFromLots(lots []TaxLot) float64 {
	var qty, cost float64
	for _, l := range lots {
		if !l.Open() {
			continue
		}
		qty += l.Quantity
		cost += l.Quantity * l.Price
	}
	if qty <= PositionEpsilon {
		return 0
	}
	return cost / qty
}

// ConsumeLots reduces open lots FIFO or LIFO. Partial lots keep remaining qty.
// sellPrice is the slipped fill; feeRate reduces net proceeds so realized PnL is after fee.
func ConsumeLots(open []TaxLot, qty, sellPrice float64, method LotMethod, now time.Time, feeRate float64) (fills []TaxLotFill, updated []TaxLot, realized float64, err error) {
	if qty <= 0 || sellPrice <= 0 || math.IsNaN(qty) || math.IsNaN(sellPrice) ||
		math.IsInf(qty, 0) || math.IsInf(sellPrice, 0) {
		return nil, nil, 0, fmt.Errorf("%w: quantity and price must be positive", ErrInvalidArgument)
	}
	if method == "" {
		method = DefaultLotMethod
	}
	avail := OpenLotQuantity(open)
	if avail+PositionEpsilon < qty {
		return nil, nil, 0, fmt.Errorf("%w: insufficient position quantity", ErrInvalidArgument)
	}
	ordered := append([]TaxLot(nil), open...)
	switch method {
	case LotMethodLIFO:
		SortLotsLIFO(ordered)
	default:
		SortLotsFIFO(ordered)
	}
	need := qty
	for i := range ordered {
		if need <= PositionEpsilon {
			break
		}
		if !ordered[i].Open() {
			continue
		}
		take := ordered[i].Quantity
		if take > need {
			take = need
		}
		net := NetSellPrice(sellPrice, feeRate)
		pnl := (net - ordered[i].Price) * take
		realized += pnl
		ordered[i].Quantity -= take
		if ordered[i].Quantity < PositionEpsilon {
			ordered[i].Quantity = 0
			t := now
			ordered[i].ClosedAt = &t
		}
		fills = append(fills, TaxLotFill{
			LotID: ordered[i].ID, Quantity: take, CostPrice: ordered[i].Price,
			SellPrice: sellPrice, RealizedPnL: pnl,
		})
		updated = append(updated, ordered[i])
		need -= take
	}
	if need > PositionEpsilon {
		return nil, nil, 0, fmt.Errorf("%w: insufficient position quantity", ErrInvalidArgument)
	}
	return fills, updated, realized, nil
}

// NewTaxLot builds an open lot for a buy fill.
func NewTaxLot(id, bookID string, ex Exchange, symbol string, qty, price float64, at time.Time, tradeID string) TaxLot {
	return TaxLot{
		ID: id, ClientID: bookID, Exchange: ex, Symbol: symbol,
		Quantity: qty, OriginalQuantity: qty, Price: price,
		OpenedAt: at, SourceTradeID: tradeID,
	}
}

// SyntheticOpeningLot covers a legacy position that has no lots yet.
func SyntheticOpeningLot(id, bookID string, ex Exchange, symbol string, qty, avgCost float64, at time.Time) TaxLot {
	return TaxLot{
		ID: id, ClientID: bookID, Exchange: ex, Symbol: symbol,
		Quantity: qty, OriginalQuantity: qty, Price: avgCost,
		OpenedAt: at, SourceTradeID: "",
	}
}
