package domain

import (
	"fmt"
	"strings"
	"time"
)

// Realtime stream limits (per WebSocket connection).
const (
	RealtimeProtocolVersion = 1
	MaxRealtimePriceSymbols = 100
	RealtimeWSPath          = "/api/v1/ws"
)

// Price subscription reasons / server event types.
const (
	RealtimeTypeHello     = "hello"
	RealtimeTypeAck       = "ack"
	RealtimeTypePong      = "pong"
	RealtimeTypeError     = "error"
	RealtimeTypePrice     = "price"
	RealtimeTypePortfolio = "portfolio"
)

// Client ops.
const (
	RealtimeOpSubscribePrices      = "subscribe_prices"
	RealtimeOpUnsubscribePrices    = "unsubscribe_prices"
	RealtimeOpSubscribePortfolio   = "subscribe_portfolio"
	RealtimeOpUnsubscribePortfolio = "unsubscribe_portfolio"
	RealtimeOpPing                 = "ping"
)

// Portfolio change reasons pushed on the stream.
const (
	PortfolioChangeSnapshot       = "snapshot"
	PortfolioChangeOrderPlaced    = "order_placed"
	PortfolioChangeOrderAmended   = "order_amended"
	PortfolioChangeOrderCancelled = "order_cancelled"
	PortfolioChangeOrderFilled    = "order_filled"
	PortfolioChangeOrderUpdated   = "order_updated"
	PortfolioChangeCash           = "cash"
	PortfolioChangePosition       = "position"
)

// SymbolRef is an exchange+pair the client wants live prices for.
type SymbolRef struct {
	Exchange Exchange
	Symbol   string
}

// SymbolKey is a normalized map key for a subscribed market.
func SymbolKey(exchange Exchange, symbol string) string {
	ex := ParseExchange(string(exchange))
	if ex == "" {
		ex = DefaultExchange
	}
	return string(ex) + ":" + NormalizeSymbol(ex, symbol)
}

// NormalizeSymbolRef validates and normalizes a client symbol subscription.
func NormalizeSymbolRef(exchange, symbol string) (SymbolRef, error) {
	ex := ParseExchange(exchange)
	if ex == "" {
		return SymbolRef{}, fmt.Errorf("%w: exchange must be one of %v", ErrInvalidArgument, SupportedExchanges)
	}
	sym := NormalizeSymbol(ex, symbol)
	if sym == "" {
		return SymbolRef{}, fmt.Errorf("%w: symbol is required", ErrInvalidArgument)
	}
	return SymbolRef{Exchange: ex, Symbol: sym}, nil
}

// PriceTick is a live (or cached) 24h ticker pushed to subscribers.
type PriceTick struct {
	Exchange           Exchange
	Symbol             string
	LastPrice          string
	PriceChange        string
	PriceChangePercent string
	OpenPrice          string
	HighPrice          string
	LowPrice           string
	Volume             string
	QuoteVolume        string
	TradeCount         int64
	OpenTime           time.Time
	CloseTime          time.Time
	UpdatedAt          time.Time
	Halted             bool
}

// PriceTickFromTicker maps a 24h ticker onto a stream tick.
func PriceTickFromTicker(exchange Exchange, t *Ticker24h, at time.Time) PriceTick {
	if t == nil {
		return PriceTick{Exchange: exchange, UpdatedAt: at}
	}
	sym := strings.TrimSpace(t.Symbol)
	return PriceTick{
		Exchange:           exchange,
		Symbol:             sym,
		LastPrice:          t.LastPrice,
		PriceChange:        t.PriceChange,
		PriceChangePercent: t.PriceChangePercent,
		OpenPrice:          t.OpenPrice,
		HighPrice:          t.HighPrice,
		LowPrice:           t.LowPrice,
		Volume:             t.Volume,
		QuoteVolume:        t.QuoteVolume,
		TradeCount:         t.TradeCount,
		OpenTime:           t.OpenTime,
		CloseTime:          t.CloseTime,
		UpdatedAt:          at,
		Halted:             t.Halted,
	}
}

// PortfolioChange is emitted after paper order/position/cash mutations.
type PortfolioChange struct {
	PortfolioID string
	Reason      string
	Order       *PendingOrder
	Trade       *Trade
	View        *PortfolioView
	// Orders is set on subscribe snapshots (open pending orders).
	Orders []PendingOrder
}

// PortfolioChangeSink fans portfolio mutations to realtime subscribers.
type PortfolioChangeSink interface {
	OnPortfolioChange(ev PortfolioChange)
}
