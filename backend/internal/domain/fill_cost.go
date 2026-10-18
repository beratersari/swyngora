package domain

import "time"

const (
	// ExecutionQuoteTTL is how long a comparison can be named on the order that follows it.
	// The ticket refreshes the comparison on its own clock. An older quote is not the one on screen.
	ExecutionQuoteTTL = 2 * time.Minute

	// FillCostOpen means slices are still working. The difference is not final.
	FillCostOpen = "open"
	// FillCostComplete means the order finished, including a spread that was canceled with fills kept.
	FillCostComplete = "complete"
)

// StoredExecutionQuote is one comparison the ticket can send back when it places.
type StoredExecutionQuote struct {
	ID        string
	ClientID  string
	Compare   ExecutionCompare
	CreatedAt time.Time
}

// FillCostCheck is the quote kept for one placed order and what the fills actually paid.
// Slippage is quote currency versus the touch the quote used, before the fee.
// A positive delta means the fill cost more than the quote.
type FillCostCheck struct {
	ID               string
	ClientID         string
	Style            ExecutionStyle
	Symbol           string
	Side             TradeSide
	ExpectedFee      float64
	ExpectedSlippage float64
	TouchPrice       float64
	QuotedQuantity   float64
	QuotedAt         time.Time
	ActualFee        float64
	ActualSlippage   float64
	FilledQuantity   float64
	Status           string
	LeadTradeID      string
	SpreadOrderID    string
}

// FeeDelta is actual fee minus the quoted fee.
func (c FillCostCheck) FeeDelta() float64 {
	return c.ActualFee - c.ExpectedFee
}

// SlippageDelta is actual slippage minus the quoted slippage.
func (c FillCostCheck) SlippageDelta() float64 {
	return c.ActualSlippage - c.ExpectedSlippage
}

// SlippageCash is the quote-currency distance from touch to price, times quantity.
// Positive means a buy paid more than the touch, or a sell received less.
func SlippageCash(side TradeSide, touch, price, quantity float64) float64 {
	if touch <= 0 || price <= 0 || quantity <= 0 {
		return 0
	}
	if side == TradeSideSell {
		return (touch - price) * quantity
	}
	return (price - touch) * quantity
}

// SlippageCash is this row's slippage in quote currency, using the same touch as the percent.
func (q ExecutionQuote) SlippageCash(side TradeSide) float64 {
	if q.Status != ExecutionFilled {
		return 0
	}
	return SlippageCash(side, q.TouchPrice, q.AveragePrice, q.FilledQuantity)
}

// QuoteFresh reports whether a comparison saved at created is still the one a place at now may use.
func QuoteFresh(created, now time.Time) bool {
	if created.IsZero() || now.IsZero() {
		return false
	}
	if created.After(now.Add(5 * time.Second)) {
		return false
	}
	return now.Sub(created) <= ExecutionQuoteTTL
}
