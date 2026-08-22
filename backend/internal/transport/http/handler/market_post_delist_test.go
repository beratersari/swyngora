package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/deliststore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
)

type handlerOffVenue struct {
	quote *domain.OffVenueQuote
}

func (h handlerOffVenue) QuoteByBase(context.Context, string) (*domain.OffVenueQuote, error) {
	return h.quote, nil
}

func (h handlerOffVenue) OHLCByBase(context.Context, string, int) ([]domain.Candle, error) {
	return []domain.Candle{{
		OpenTime: time.Unix(1, 0).UTC(),
		Open:     "0.1",
		High:     "0.2",
		Low:      "0.05",
		Close:    "0.15",
	}}, nil
}

func TestGetPostDelist_JSON(t *testing.T) {
	store := deliststore.NewMemory()
	store.ReplaceAll(domain.ExchangeBinance, []domain.SpotDelistEntry{
		{Symbol: "VICUSDT", DelistTime: time.Now().UTC().Add(-48 * time.Hour)},
	})
	svc := market.NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: stubMarket{},
	}, nil).
		WithDelistStore(store).
		WithOffVenuePrice(handlerOffVenue{quote: &domain.OffVenueQuote{LastUSD: 0.15, AsOf: time.Unix(2, 0).UTC()}})
	h := NewMarketHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/post-delist?exchange=binance&symbol=VICUSDT", nil)
	rec := httptest.NewRecorder()
	h.GetPostDelist(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var got postDelistResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !got.Available || got.Source != "coingecko" || got.LastPrice != "0.15" {
		t.Fatalf("got=%+v", got)
	}
	if len(got.Candles) != 1 {
		t.Fatalf("candles=%d", len(got.Candles))
	}
}

func TestGetPostDelist_MissingSymbol(t *testing.T) {
	h := NewMarketHandler(market.NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: stubMarket{},
	}, nil))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/post-delist?exchange=binance", nil)
	rec := httptest.NewRecorder()
	h.GetPostDelist(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
