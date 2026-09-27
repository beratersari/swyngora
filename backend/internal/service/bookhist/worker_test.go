package bookhist

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type scriptBook struct {
	calls int
	err   error
	book  *domain.RawOrderBook
}

func (s *scriptBook) GetCandles(context.Context, domain.CandleQuery) ([]domain.Candle, error) {
	return nil, nil
}
func (s *scriptBook) GetTicker24h(context.Context, string) (*domain.Ticker24h, error) {
	return nil, nil
}
func (s *scriptBook) GetOrderBook(context.Context, domain.OrderBookQuery) (*domain.RawOrderBook, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return s.book, nil
}
func (s *scriptBook) ListSpotMarkets(context.Context) ([]domain.SpotMarket, error) {
	return nil, nil
}
func (s *scriptBook) ListProductTags(context.Context) ([]string, error) { return nil, nil }
func (s *scriptBook) TagsByBase(context.Context) (map[string][]string, error) {
	return nil, nil
}

func TestRunFlashes_RateLimitSkipsOnlyThatVenue(t *testing.T) {
	binance := &scriptBook{err: fmt.Errorf("%w: binance", domain.ErrRateLimited)}
	book := &domain.RawOrderBook{
		Bids: []domain.PriceLevel{{Price: 100, Quantity: 1}},
		Asks: []domain.PriceLevel{{Price: 101, Quantity: 1}},
	}
	bybit := &scriptBook{book: book}
	svc := &Service{
		Store: &memStore{},
		Books: map[domain.Exchange]domain.MarketDataPort{
			domain.ExchangeBinance: binance,
			domain.ExchangeBybit:   bybit,
		},
		Seeds: []string{"BTCUSDT", "ETHUSDT"},
	}
	(&Worker{Hist: svc, now: time.Now}).RunFlashes(context.Background())
	if binance.calls != 1 {
		t.Fatalf("binance calls=%d, want the first symbol only", binance.calls)
	}
	if bybit.calls != 2 {
		t.Fatalf("bybit calls=%d, want both symbols while binance is paused", bybit.calls)
	}
}
