package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
)

type stubMarket struct{}

func (stubMarket) GetCandles(_ context.Context, q domain.CandleQuery) ([]domain.Candle, error) {
	if q.Symbol == "BAD" {
		return nil, domain.ErrNotFound
	}
	return []domain.Candle{{
		OpenTime:    time.Unix(0, 0).UTC(),
		Open:        "1",
		High:        "2",
		Low:         "0.5",
		Close:       "1.5",
		Volume:      "10",
		CloseTime:   time.Unix(60, 0).UTC(),
		QuoteVolume: "15",
		TradeCount:  3,
	}}, nil
}

func (stubMarket) ListSpotMarkets(_ context.Context) ([]domain.SpotMarket, error) {
	return []domain.SpotMarket{
		{Symbol: "BTCUSDT", BaseAsset: "BTC", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "1000", LastPrice: "100", TradeCount: 9, Tags: []string{"Payments"}},
		{Symbol: "ETHUSDT", BaseAsset: "ETH", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "500", LastPrice: "50", TradeCount: 3, Tags: []string{"Layer1_Layer2"}},
	}, nil
}

func (stubMarket) TagsByBase(_ context.Context) (map[string][]string, error) {
	return map[string][]string{}, nil
}

func (stubMarket) ListProductTags(_ context.Context) ([]string, error) {
	return []string{"Layer1_Layer2", "Payments"}, nil
}

func (stubMarket) GetTicker24h(_ context.Context, symbol string) (*domain.Ticker24h, error) {
	return &domain.Ticker24h{
		Symbol:      symbol,
		LastPrice:   "100",
		Volume:      "50",
		QuoteVolume: "5000",
		OpenTime:    time.Unix(0, 0).UTC(),
		CloseTime:   time.Unix(1, 0).UTC(),
	}, nil
}

func (stubMarket) GetOrderBook(_ context.Context, q domain.OrderBookQuery) (*domain.RawOrderBook, error) {
	return &domain.RawOrderBook{
		Symbol: q.Symbol,
		Bids: []domain.PriceLevel{
			{Price: 100, Quantity: 2}, {Price: 99.5, Quantity: 4}, {Price: 99.0, Quantity: 2},
			{Price: 80, Quantity: 5_000},
		},
		Asks: []domain.PriceLevel{
			{Price: 100.1, Quantity: 1.5}, {Price: 100.5, Quantity: 3}, {Price: 101, Quantity: 1},
			{Price: 120, Quantity: 5_000},
		},
	}, nil
}

type stubSupply struct{}

func (stubSupply) Refresh(context.Context) (int, error) { return 0, nil }

func (stubSupply) GetSupply(_ context.Context, asset string) (*domain.AssetSupply, error) {
	max := 21_000_000.0
	return &domain.AssetSupply{
		Asset:     asset,
		Name:      "Bitcoin",
		MaxSupply: &max,
		AsOf:      time.Unix(0, 0).UTC(),
		Source:    "binance",
	}, nil
}

func newTestHandler() *MarketHandler {
	return NewMarketHandler(market.New(stubMarket{}, stubSupply{}))
}

func TestGetCandles_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/candles?symbol=BTCUSDT&interval=1h&limit=5", nil)
	rr := httptest.NewRecorder()
	h.GetCandles(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body candlesResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Exchange != "binance" || len(body.Candles) != 1 || body.Candles[0].Close != "1.5" {
		t.Fatalf("body=%+v", body)
	}
}

func TestGetCandles_MissingSymbol(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/candles", nil)
	rr := httptest.NewRecorder()
	h.GetCandles(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestGetOrderBook_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/orderbook?symbol=BTCUSDT&group=0.1&limit=10", nil)
	rr := httptest.NewRecorder()
	h.GetOrderBook(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body orderBookResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "BTCUSDT" || body.GroupSize != "0.1" || len(body.Bids) == 0 || len(body.Asks) == 0 {
		t.Fatalf("body=%+v", body)
	}
	if len(body.SuggestedGroupSizes) == 0 {
		t.Fatal("expected suggested groups")
	}
	if body.Analysis.RangePct != 2 || body.Analysis.Pressure == "" || body.Analysis.BidLevels < 1 {
		t.Fatalf("analysis %+v", body.Analysis)
	}
	if body.Analysis.BidLevels > 3 {
		t.Fatalf("far depth leaked into 2%% band: %+v", body.Analysis)
	}
	for _, w := range body.Analysis.Walls {
		if w.Behavior == "" {
			t.Fatalf("wall missing behavior: %+v", w)
		}
	}
}

func TestGetOrderBook_BadRange(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/orderbook?symbol=BTCUSDT&rangePct=-1", nil)
	rr := httptest.NewRecorder()
	h.GetOrderBook(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestGetOrderBookImpact_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/orderbook/impact?symbol=BTCUSDT&quantity=1&exchange=binance", nil)
	rr := httptest.NewRecorder()
	h.GetOrderBookImpact(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body orderBookImpactResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.AveragePrice == "" || body.Side != "buy" || body.Exhausted {
		t.Fatalf("%+v", body)
	}
	if !body.ImpactAvailable || body.ImpactPct != 0 {
		t.Fatalf("partial best ask must be 0 impact: %+v", body)
	}
}

func TestGetOrderBookImpact_BadSize(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/orderbook/impact?symbol=BTCUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetOrderBookImpact(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestGetLiquidations_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/liquidations?symbol=BTCUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetLiquidations(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body liquidationsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "BTCUSDT" {
		t.Fatalf("%+v", body)
	}
}

func TestGetOpenInterest_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/open-interest?symbol=BTCUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetOpenInterest(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body openInterestResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "BTCUSDT" || body.Unit != "BTC" {
		t.Fatalf("%+v", body)
	}
}

func TestGetFundingRate_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/funding-rate?symbol=BTCUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetFundingRate(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body fundingRateResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "BTCUSDT" {
		t.Fatalf("%+v", body)
	}
}

func TestGetBasis_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/basis?symbol=BTCUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetBasis(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body basisResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "BTCUSDT" {
		t.Fatalf("%+v", body)
	}
}

func TestGetBasis_BadSymbol(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/basis", nil)
	rr := httptest.NewRecorder()
	h.GetBasis(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestGetCorrelation_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/correlation?symbol=SOLUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetCorrelation(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body corrResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "SOLUSDT" || len(body.Windows) != 3 {
		t.Fatalf("%+v", body)
	}
}

func TestGetVolatility_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/volatility?symbol=SOLUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetVolatility(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body volResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "SOLUSDT" || len(body.Windows) != 3 {
		t.Fatalf("%+v", body)
	}
}

func TestGetVolatility_BadSymbol(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/volatility", nil)
	rr := httptest.NewRecorder()
	h.GetVolatility(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestGetBreadth_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/breadth", nil)
	rr := httptest.NewRecorder()
	h.GetBreadth(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body breadthResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Windows) != 3 {
		t.Fatalf("%+v", body)
	}
}

func TestGetBreadth_BadLimit(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/breadth?limit=nope", nil)
	rr := httptest.NewRecorder()
	h.GetBreadth(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestGetCorrelation_BadSymbol(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/correlation", nil)
	rr := httptest.NewRecorder()
	h.GetCorrelation(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestGetTakerFlow_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/taker-flow?symbol=BTCUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetTakerFlow(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body takerFlowResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "BTCUSDT" {
		t.Fatalf("%+v", body)
	}
}

func TestGetTakerFlow_BadSymbol(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/taker-flow", nil)
	rr := httptest.NewRecorder()
	h.GetTakerFlow(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestGetVenueDivergence_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/venue-divergence?symbol=BTCUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetVenueDivergence(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body venueDivergenceResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "BTCUSDT" {
		t.Fatalf("%+v", body)
	}
}

func TestGetVenueDivergence_BadSymbol(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/venue-divergence", nil)
	rr := httptest.NewRecorder()
	h.GetVenueDivergence(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestGetPositioning_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/positioning?symbol=BTCUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetPositioning(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body positioningResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "BTCUSDT" || body.Note == "" {
		t.Fatalf("%+v", body)
	}
}

func TestGetPositioning_BadSymbol(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/positioning", nil)
	rr := httptest.NewRecorder()
	h.GetPositioning(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestGetSqueezeRisk_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/squeeze-risk?symbol=BTCUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetSqueezeRisk(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body squeezeResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "BTCUSDT" || body.Note == "" {
		t.Fatalf("%+v", body)
	}
}

func TestGetSqueezeRisk_BadSymbol(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/squeeze-risk", nil)
	rr := httptest.NewRecorder()
	h.GetSqueezeRisk(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestGetLiquidationHunt_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/liquidation-hunt?symbol=BTCUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetLiquidationHunt(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body huntResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "BTCUSDT" || body.Note == "" {
		t.Fatalf("%+v", body)
	}
}

func TestGetLiquidationHunt_BadSymbol(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/liquidation-hunt", nil)
	rr := httptest.NewRecorder()
	h.GetLiquidationHunt(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestGetFuturesHistory_NotConfigured(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/futures-history?metric=open_interest&symbol=BTCUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetFuturesHistory(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

type stubFutHist struct{}

func (stubFutHist) History(_ context.Context, q domain.FuturesHistoryQuery) (any, error) {
	return []domain.FuturesSnapshot{{
		Metric: domain.FuturesMetricOpenInterest, Exchange: domain.ExchangeBinance,
		Symbol: q.Symbol, SampledAt: time.Unix(1_700_000_000, 0).UTC(),
		Contracts: 100, Value: 6_400_000,
	}}, nil
}

func TestGetFuturesHistory_OK(t *testing.T) {
	svc := market.New(stubMarket{}, stubSupply{})
	svc.SetFuturesHistory(stubFutHist{})
	h := NewMarketHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/futures-history?metric=open_interest&symbol=BTCUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetFuturesHistory(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["metric"] != "open_interest" || body["symbol"] != "BTCUSDT" {
		t.Fatalf("%+v", body)
	}
	items, ok := body["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items=%v", body["items"])
	}
}

func TestGetLongShortRatio_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/long-short-ratio?symbol=BTCUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetLongShortRatio(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body longShortResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "BTCUSDT" {
		t.Fatalf("%+v", body)
	}
}

func TestGetFundingRate_BadLimit(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/funding-rate?symbol=BTCUSDT&limit=nope", nil)
	rr := httptest.NewRecorder()
	h.GetFundingRate(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestGetOpenInterest_BadSymbol(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/open-interest", nil)
	rr := httptest.NewRecorder()
	h.GetOpenInterest(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestGetMarketLiquidity_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/orderbook/liquidity?symbol=BTCUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetMarketLiquidity(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body marketLiquidityResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol == "" || len(body.Market.Bands) != 3 || body.VenueCount < 1 {
		t.Fatalf("%+v", body)
	}
}

func TestGetCombinedOrderBook_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/orderbook/combined?symbol=BTCUSDT&rangePct=2", nil)
	rr := httptest.NewRecorder()
	h.GetCombinedOrderBook(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body combinedOrderBookResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.VenueCount < 1 || body.Pressure == "" || body.Symbol == "" {
		t.Fatalf("%+v", body)
	}
}

func TestGetOrderBook_BadGroup(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/orderbook?symbol=BTCUSDT&group=-1", nil)
	rr := httptest.NewRecorder()
	h.GetOrderBook(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestGetTicker24h_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/ticker/24h?symbol=ETHUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetTicker24h(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestGetSupply_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/supply?asset=BTC", nil)
	rr := httptest.NewRecorder()
	h.GetSupply(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
	var body supplyResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.MaxSupply == nil || *body.MaxSupply != 21_000_000 {
		t.Fatalf("body=%+v", body)
	}
}

func TestGetIntervals(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/intervals", nil)
	rr := httptest.NewRecorder()
	h.GetIntervals(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestGetCandles_NotFound(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/candles?symbol=BAD&interval=1h", nil)
	rr := httptest.NewRecorder()
	h.GetCandles(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestGetCandles_BadLimit(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/candles?symbol=BTCUSDT&limit=abc", nil)
	rr := httptest.NewRecorder()
	h.GetCandles(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestGetCandles_TimeParams(t *testing.T) {
	h := newTestHandler()
	// Unix ms
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/candles?symbol=BTCUSDT&interval=1h&startTime=1700000000000&endTime=1700003600000", nil)
	rr := httptest.NewRecorder()
	h.GetCandles(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("unix ms status=%d body=%s", rr.Code, rr.Body.String())
	}

	// RFC3339
	req = httptest.NewRequest(http.MethodGet, "/api/v1/market/candles?symbol=BTCUSDT&interval=1h&startTime=2024-01-01T00:00:00Z&endTime=2024-01-02T00:00:00Z", nil)
	rr = httptest.NewRecorder()
	h.GetCandles(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("rfc3339 status=%d body=%s", rr.Code, rr.Body.String())
	}

	// Invalid time
	req = httptest.NewRequest(http.MethodGet, "/api/v1/market/candles?symbol=BTCUSDT&startTime=not-a-time", nil)
	rr = httptest.NewRecorder()
	h.GetCandles(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("bad time status=%d", rr.Code)
	}
}

func TestGetSupply_ViaSymbolParam(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/supply?symbol=BTCUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetSupply(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestGetSupply_MissingAsset(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/supply", nil)
	rr := httptest.NewRecorder()
	h.GetSupply(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestGetTicker24h_MissingSymbol(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/ticker/24h", nil)
	rr := httptest.NewRecorder()
	h.GetTicker24h(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestListSpotMarkets_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/spot?q=btc&sort=quoteVolume&order=desc&limit=10", nil)
	rr := httptest.NewRecorder()
	h.ListSpotMarkets(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body spotListResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Exchange != "binance" || body.Total != 1 || len(body.Items) != 1 || body.Items[0].Symbol != "BTCUSDT" {
		t.Fatalf("body=%+v", body)
	}
	// base/quote/status removed from public DTO
	if body.Items[0].LastPrice == "" {
		t.Fatalf("expected lastPrice on item")
	}
}

func TestEncodeMarketCapMax(t *testing.T) {
	if encodeMarketCapMax(domain.SpotMarket{MarketCapMaxInfinite: true}) != "∞" {
		t.Fatal("want infinity symbol")
	}
	v := 1.5
	got := encodeMarketCapMax(domain.SpotMarket{MarketCapMax: &v})
	if got != v {
		t.Fatalf("got %v", got)
	}
	if encodeMarketCapMax(domain.SpotMarket{}) != nil {
		t.Fatal("want null")
	}
}

func TestListSpotMarkets_BadSort(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/spot?sort=nope", nil)
	rr := httptest.NewRecorder()
	h.ListSpotMarkets(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestListSpotMarkets_BadLimit(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/spot?limit=x", nil)
	rr := httptest.NewRecorder()
	h.ListSpotMarkets(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}
