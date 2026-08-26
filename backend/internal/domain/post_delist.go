package domain

import (
	"context"
	"time"
)

// PostDelistView is off-venue price after this exchange has halted the pair.
// Informational only — not this venue's book.
type PostDelistView struct {
	Exchange           string
	Symbol             string
	Base               string
	DelistTime         time.Time
	Available          bool
	Source             string
	SourceLabel        string
	Note               string
	LastPrice          string
	PriceChangePercent string
	Quote              string
	AsOf               time.Time
	Interval           string
	Candles            []Candle
}

// OffVenueQuote is a public global last price (e.g. CoinGecko USD).
type OffVenueQuote struct {
	Symbol     string
	Name       string
	ProviderID string
	LastUSD    float64
	ChangePct  *float64
	ChangeAbs  *float64
	AsOf       time.Time
}

// FillChangeAbs sets USD 24h delta from last and percent when the API omitted it.
func (q *OffVenueQuote) FillChangeAbs() {
	if q == nil || q.ChangeAbs != nil || q.ChangePct == nil {
		return
	}
	p := *q.ChangePct / 100
	if p <= -1 {
		return
	}
	abs := q.LastUSD * p / (1 + p)
	q.ChangeAbs = &abs
}

// OffVenuePricePort fetches a public price after a venue delist.
type OffVenuePricePort interface {
	QuoteByBase(ctx context.Context, base string) (*OffVenueQuote, error)
	OHLCByBase(ctx context.Context, base string, days int) ([]Candle, error)
}
