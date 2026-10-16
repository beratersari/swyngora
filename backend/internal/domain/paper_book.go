package domain

import (
	"fmt"
	"math"
	"sort"
)

const (
	// PaperFillBook means the fill price is the average of the visible spot book.
	PaperFillBook = "book"
	// PaperFillSlippage means the fill uses the fixed equity (or test) slippage.
	PaperFillSlippage = "slippage"
)

// ErrBookExhausted means the visible book cannot hold the full size.
// The order is not filled. A missing level is not invented past the book.
var ErrBookExhausted = fmt.Errorf("%w: visible book cannot fill this size", ErrInvalidArgument)

// PaperFillQuote is one paper fill priced from a book walk or from fixed slippage.
type PaperFillQuote struct {
	Source            string
	Price             float64
	RequestedQuantity float64
	FilledQuantity    float64
	VisibleQuantity   float64
	Exhausted         bool
	LevelsUsed        int
	BestPrice         float64
	EndPrice          float64
}

// QuoteSlippageFill prices qty at reference plus the venue's fixed adverse slippage.
func QuoteSlippageFill(side TradeSide, reference, qty float64, cost TradingCost) PaperFillQuote {
	fill, _, _, _, _ := QuoteFill(side, reference, qty, cost)
	out := PaperFillQuote{
		Source:            PaperFillSlippage,
		Price:             fill,
		RequestedQuantity: qty,
		FilledQuantity:    qty,
	}
	if qty > 0 && fill > 0 {
		out.VisibleQuantity = qty
		out.BestPrice = reference
		out.EndPrice = fill
	}
	return out
}

// WalkPaperBook prices qty against one venue's visible bids or asks.
// A buy walks asks from low to high. A sell walks bids from high to low.
// limit > 0 keeps only prices at or better than that limit (asks at or below,
// bids at or above). A market order passes limit 0 and may use the whole side.
// Exhausted is true when the visible size is shorter than qty. Price is the
// average of the size the book could take, and is 0 when it could take nothing.
func WalkPaperBook(side TradeSide, qty float64, book *RawOrderBook, limit float64) PaperFillQuote {
	out := PaperFillQuote{
		Source:            PaperFillBook,
		RequestedQuantity: qty,
	}
	if qty <= 0 || math.IsNaN(qty) || math.IsInf(qty, 0) {
		return out
	}
	levels := paperBookLevels(side, book, limit)
	var visible float64
	for _, lv := range levels {
		visible += lv.Quantity
	}
	out.VisibleQuantity = visible
	if len(levels) == 0 {
		out.Exhausted = true
		return out
	}
	out.BestPrice = levels[0].Price
	var filled, spent float64
	var last float64
	remain := qty
	for _, lv := range levels {
		if remain <= 1e-12 {
			break
		}
		take := lv.Quantity
		if remain < take {
			take = remain
		}
		if take <= 0 {
			continue
		}
		filled += take
		spent += take * lv.Price
		last = lv.Price
		remain -= take
		out.LevelsUsed++
	}
	out.FilledQuantity = filled
	out.EndPrice = last
	if filled > 0 {
		out.Price = spent / filled
	}
	if remain > 1e-12 {
		out.Exhausted = true
	}
	return out
}

func paperBookLevels(side TradeSide, book *RawOrderBook, limit float64) []PriceLevel {
	if book == nil {
		return nil
	}
	buy := side != TradeSideSell
	src := book.Asks
	if !buy {
		src = book.Bids
	}
	out := make([]PriceLevel, 0, len(src))
	for _, lv := range src {
		if lv.Price <= 0 || lv.Quantity <= 0 || math.IsNaN(lv.Price) || math.IsNaN(lv.Quantity) {
			continue
		}
		if limit > 0 {
			if buy && lv.Price > limit {
				continue
			}
			if !buy && lv.Price < limit {
				continue
			}
		}
		out = append(out, lv)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Price == out[j].Price {
			return false
		}
		if buy {
			return out[i].Price < out[j].Price
		}
		return out[i].Price > out[j].Price
	})
	return out
}
