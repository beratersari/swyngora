package market

import (
	"context"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type gapBookPort struct {
	book domain.RawOrderBook
}

func (p gapBookPort) GetCandles(context.Context, domain.CandleQuery) ([]domain.Candle, error) {
	return nil, nil
}
func (p gapBookPort) GetTicker24h(context.Context, string) (*domain.Ticker24h, error) {
	return &domain.Ticker24h{}, nil
}
func (p gapBookPort) GetOrderBook(context.Context, domain.OrderBookQuery) (*domain.RawOrderBook, error) {
	b := p.book
	return &b, nil
}
func (p gapBookPort) ListSpotMarkets(context.Context) ([]domain.SpotMarket, error) {
	return []domain.SpotMarket{{
		Symbol: "SOLUSDT", BaseAsset: "SOL", QuoteAsset: "USDT", QuoteVolume: "1000000", Status: "TRADING",
	}}, nil
}
func (p gapBookPort) ListProductTags(context.Context) ([]string, error) { return nil, nil }
func (p gapBookPort) TagsByBase(context.Context) (map[string][]string, error) {
	return map[string][]string{}, nil
}

func TestGetBookGaps_FromLiveBook(t *testing.T) {
	s := &Service{markets: map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: gapBookPort{book: gapBookRaw()},
	}}
	got, err := s.GetBookGaps(context.Background(), "binance", "SOLUSDT", 2)
	if err != nil {
		t.Fatal(err)
	}
	if got.Easier != domain.BookGapEasierUp || got.Symbol != "SOLUSDT" {
		t.Fatalf("%+v", got)
	}
	if got.Note == "" {
		t.Fatal("note")
	}
}

func gapBookRaw() domain.RawOrderBook {
	bids := make([]domain.PriceLevel, 0, 20)
	for i := 1; i <= 20; i++ {
		bids = append(bids, domain.PriceLevel{Price: 100 - float64(i)*0.1, Quantity: 20})
	}
	return domain.RawOrderBook{
		Symbol: "SOLUSDT",
		Bids:   bids,
		Asks: []domain.PriceLevel{
			{Price: 100.05, Quantity: 2},
			{Price: 101.5, Quantity: 2},
		},
	}
}
