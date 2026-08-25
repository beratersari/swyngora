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

func (stubMarket) GetRecentPrints(_ context.Context, symbol string) ([]domain.TakerPrint, error) {
	t0 := time.Unix(1_700_000_000, 0).UTC()
	return []domain.TakerPrint{{
		Exchange: domain.ExchangeBinance, Symbol: symbol, Side: domain.TakerSideBuy,
		Price: 100, Quantity: 2000, Notional: 200_000, Time: t0,
	}}, nil
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

type stubHolders struct{}

func (stubHolders) GetHolders(_ context.Context, asset string) (*domain.AssetHolders, error) {
	top10 := 5.4
	return &domain.AssetHolders{
		Asset:          asset,
		Name:           "Bitcoin",
		HolderCount:    50_000_000,
		TopTenSharePct: &top10,
		TopHolders:     []domain.AssetHolder{{Address: "34xp4vRoCGJym3xR7yCVPFHoCNxv4Twseo", Balance: 1, SharePct: 1.18}},
		AsOf:           time.Unix(0, 0).UTC(),
		Source:         "coinmarketcap",
	}, nil
}

func newTestHandler() *MarketHandler {
	return NewMarketHandler(market.New(stubMarket{}, stubSupply{}).WithHolders(stubHolders{}))
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

func TestGetOrderBookHeatmap_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/orderbook/heatmap?symbol=BTCUSDT&window=300", nil)
	rr := httptest.NewRecorder()
	h.GetOrderBookHeatmap(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body orderBookHeatmapResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "BTCUSDT" || body.WindowSeconds != 300 || len(body.Columns) == 0 {
		t.Fatalf("%+v", body)
	}
	if body.Columns[0].T == "" || body.Columns[0].Mid == "" {
		t.Fatalf("column %+v", body.Columns[0])
	}
}

func TestGetOrderBookHeatmap_BadWindow(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/orderbook/heatmap?symbol=BTCUSDT&window=nope", nil)
	rr := httptest.NewRecorder()
	h.GetOrderBookHeatmap(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
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

func TestGetLevels_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/levels?symbol=BTCUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetLevels(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body levelsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "BTCUSDT" {
		t.Fatalf("%+v", body)
	}
}

func TestGetLevels_BadSymbol(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/levels", nil)
	rr := httptest.NewRecorder()
	h.GetLevels(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestGetSnapshot_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/snapshot?symbol=SOLUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetSnapshot(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body snapshotResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "SOLUSDT" {
		t.Fatalf("%+v", body)
	}
}

func TestGetSnapshot_BadSymbol(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/snapshot", nil)
	rr := httptest.NewRecorder()
	h.GetSnapshot(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestGetCVD_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/cvd?symbol=BTCUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetCVD(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body cvdResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "BTCUSDT" {
		t.Fatalf("%+v", body)
	}
}

func TestCVDToDTO_SharesAndDivergence(t *testing.T) {
	at := time.Date(2026, 8, 16, 12, 5, 0, 0, time.UTC)
	from := at.Add(-time.Hour)
	rep := &domain.CVDReport{
		Symbol: "BTCUSDT", Exchange: "all",
		Combined: &domain.CVDVenueSeries{
			Exchange: "all", Symbol: "BTCUSDT",
			Points: []domain.CVDPoint{{
				Time: at, Price: 101, PriceChangePct: 1, Delta: -80, CVD: 20,
				VsPrice: domain.CVDVsOpposite, Divergence: domain.CVDDivPriceUpCVDDown,
				Shares: []domain.CVDShare{
					{Exchange: domain.ExchangeBinance, Delta: -50, CVD: 10, SharePct: 50},
					{Exchange: domain.ExchangeBybit, Delta: -30, CVD: 10, SharePct: 50},
				},
			}},
			Contributions: []domain.CVDShare{
				{Exchange: domain.ExchangeBinance, CVD: 10, SharePct: 50},
			},
			OverlapFrom: &from, OverlapTo: &at,
			Divergence: domain.CVDDivergence{
				Kind: domain.CVDDivPriceUpCVDDown, VsPrice: domain.CVDVsOpposite,
				Title: "price up, CVD down", Bars: 1, LastAt: at,
			},
			Complete: false,
		},
	}
	got := cvdToDTO(rep)
	if got.Combined == nil || got.Combined.Complete || got.Combined.OverlapFrom == nil {
		t.Fatalf("%+v", got.Combined)
	}
	if got.Combined.Divergence.Kind != domain.CVDDivPriceUpCVDDown {
		t.Fatalf("div %+v", got.Combined.Divergence)
	}
	if got.Combined.Divergence.CVDMove == "" {
		t.Fatal("expected cvdMove on divergence")
	}
	if len(got.Combined.Points) != 1 || got.Combined.Points[0].Divergence != domain.CVDDivPriceUpCVDDown {
		t.Fatalf("point %+v", got.Combined.Points)
	}
	if len(got.Combined.Points[0].Shares) != 2 || got.Combined.Points[0].Shares[0].Exchange != "binance" {
		t.Fatalf("shares %+v", got.Combined.Points[0].Shares)
	}
	rep.SpotFutures = &domain.CVDSpotFutures{
		Alignment: domain.AlignOpposite, Spot: domain.CVDDirUp, Futures: domain.CVDDirDown,
		SpotChange: 10, FuturesChange: -8, Window: domain.CVDWindow1h, Summary: "split",
	}
	got = cvdToDTO(rep)
	if got.SpotFutures == nil || got.SpotFutures.Alignment != domain.AlignOpposite {
		t.Fatalf("spotFutures %+v", got.SpotFutures)
	}
}

func TestGetCVD_BadSymbol(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/cvd", nil)
	rr := httptest.NewRecorder()
	h.GetCVD(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

type volumeProfileMarket struct {
	stubMarket
}

func (volumeProfileMarket) GetCandles(_ context.Context, q domain.CandleQuery) ([]domain.Candle, error) {
	at := time.Now().UTC().Add(-2 * time.Hour)
	return []domain.Candle{{
		OpenTime: at, Open: "65000", High: "65300", Low: "65100", Close: "65200",
		Volume: "10", CloseTime: at.Add(time.Minute), QuoteVolume: "80000", TakerBuyQuote: "50000",
	}, {
		OpenTime: at.Add(time.Hour), Open: "67800", High: "68000", Low: "67700", Close: "67900",
		Volume: "2", CloseTime: at.Add(time.Hour + time.Minute), QuoteVolume: "10000", TakerBuyQuote: "2000",
	}}, nil
}

func (volumeProfileMarket) GetTicker24h(_ context.Context, symbol string) (*domain.Ticker24h, error) {
	return &domain.Ticker24h{Symbol: symbol, LastPrice: "65500"}, nil
}

func TestGetVolumeProfile_OK(t *testing.T) {
	h := NewMarketHandler(market.New(volumeProfileMarket{}, stubSupply{}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/volume-profile?symbol=BTCUSDT&window=4h&tickSize=100", nil)
	rr := httptest.NewRecorder()
	h.GetVolumeProfile(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body volumeProfileResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "BTCUSDT" || body.Window != "4h" || len(body.Venues) == 0 {
		t.Fatalf("%+v", body)
	}
	if body.Venues[0].POC.Price == "" || body.Venues[0].ValueArea.Low == "" {
		t.Fatalf("poc/va %+v", body.Venues[0])
	}
}

func TestGetVolumeProfile_BadSymbol(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/volume-profile", nil)
	rr := httptest.NewRecorder()
	h.GetVolumeProfile(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestGetVWAP_OK(t *testing.T) {
	h := NewMarketHandler(market.New(volumeProfileMarket{}, stubSupply{}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/vwap?symbol=BTCUSDT&window=4h", nil)
	rr := httptest.NewRecorder()
	h.GetVWAP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body vwapResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "BTCUSDT" || body.Window != "4h" || len(body.Venues) == 0 {
		t.Fatalf("%+v", body)
	}
	if body.Venues[0].VWAP == "" || body.Venues[0].Volume == "" {
		t.Fatalf("vwap %+v", body.Venues[0])
	}
}

func TestGetVWAP_BadSymbol(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/vwap", nil)
	rr := httptest.NewRecorder()
	h.GetVWAP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestGetAround_OK(t *testing.T) {
	h := NewMarketHandler(market.New(volumeProfileMarket{}, stubSupply{}))
	at := time.Now().UTC().Add(-90 * time.Minute).Format(time.RFC3339)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/around?symbol=BTCUSDT&window=1h&during=15m&at="+at, nil)
	rr := httptest.NewRecorder()
	h.GetAround(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body aroundResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "BTCUSDT" || body.Window != "1h" || body.During != "15m" || len(body.Venues) == 0 {
		t.Fatalf("%+v", body)
	}
	if len(body.Venues[0].Phases) != 3 {
		t.Fatalf("phases %+v", body.Venues[0].Phases)
	}
}

func TestGetAround_MissingAt(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/around?symbol=BTCUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetAround(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestGetAroundCompare_OK(t *testing.T) {
	h := NewMarketHandler(market.New(volumeProfileMarket{}, stubSupply{}))
	from := time.Now().UTC().Add(-3 * time.Hour).Format(time.RFC3339)
	to := time.Now().UTC().Add(-90 * time.Minute).Format(time.RFC3339)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/around/compare?symbol=BTCUSDT&window=1h&during=15m&from="+from+"&to="+to, nil)
	rr := httptest.NewRecorder()
	h.GetAroundCompare(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body aroundCompareResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "BTCUSDT" || body.FromMove == nil || body.ToMove == nil || len(body.Venues) == 0 {
		t.Fatalf("%+v", body)
	}
}

func TestGetAroundMoves_OK(t *testing.T) {
	h := NewMarketHandler(market.New(volumeProfileMarket{}, stubSupply{}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/around/moves?symbol=BTCUSDT&lookback=24h&minReturnPct=1", nil)
	rr := httptest.NewRecorder()
	h.GetAroundMoves(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body aroundMovesResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "BTCUSDT" || body.Lookback != "24h" {
		t.Fatalf("%+v", body)
	}
}

func TestGetAroundPrecursors_OK(t *testing.T) {
	h := NewMarketHandler(market.New(volumeProfileMarket{}, stubSupply{}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/around/precursors?symbol=BTCUSDT&lookback=24h&minReturnPct=1", nil)
	rr := httptest.NewRecorder()
	h.GetAroundPrecursors(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body aroundPrecursorsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "BTCUSDT" || body.Lookback != "24h" {
		t.Fatalf("%+v", body)
	}
}

func TestGetAroundSimilar_OK(t *testing.T) {
	h := NewMarketHandler(market.New(volumeProfileMarket{}, stubSupply{}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/around/similar?symbol=BTCUSDT&lookback=24h&fields=volume,book,oi&weights=book:3,oi:3,volume:1&minCoverage=60&horizons=30m,2h", nil)
	rr := httptest.NewRecorder()
	h.GetAroundSimilar(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body aroundSimilarResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "BTCUSDT" || body.MinCoverage != "60" {
		t.Fatalf("%+v", body)
	}
	if len(body.Horizons) != 2 || body.Horizons[0] != "30m" || body.Horizons[1] != "2h" {
		t.Fatalf("horizons %+v", body.Horizons)
	}
	if len(body.Fields) != 3 || len(body.Weights) != 3 {
		t.Fatalf("fields/weights %+v", body)
	}
	for _, hit := range append(append([]aroundSimilarHitDTO{}, body.Matches...), body.Skipped...) {
		for _, c := range hit.Compared {
			if !c.Used && c.Score != "" {
				t.Fatalf("uncompared %s should not have score %+v", c.Name, c)
			}
		}
		if !hit.DataTo.IsZero() && hit.DataTo.After(hit.At) {
			t.Fatalf("dataTo after move start %+v", hit)
		}
	}
}

func TestGetAroundSimilar_BadWeights(t *testing.T) {
	h := NewMarketHandler(market.New(volumeProfileMarket{}, stubSupply{}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/around/similar?symbol=BTCUSDT&fields=volume,book&weights=price:2", nil)
	rr := httptest.NewRecorder()
	h.GetAroundSimilar(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestGetAroundMoves_BadSymbol(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/around/moves", nil)
	rr := httptest.NewRecorder()
	h.GetAroundMoves(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestGetAroundCompare_MissingTimes(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/around/compare?symbol=BTCUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetAroundCompare(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestGetVolumeProfile_BadTick(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/volume-profile?symbol=BTCUSDT&tickSize=nope", nil)
	rr := httptest.NewRecorder()
	h.GetVolumeProfile(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestGetAbsorption_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/absorption?symbol=BTCUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetAbsorption(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body absorptionResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "BTCUSDT" {
		t.Fatalf("%+v", body)
	}
}

func TestGetAbsorption_BadSymbol(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/absorption", nil)
	rr := httptest.NewRecorder()
	h.GetAbsorption(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestGetLiquiditySweeps_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/liquidity-sweeps?symbol=BTCUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetLiquiditySweeps(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body liquiditySweepResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "BTCUSDT" {
		t.Fatalf("%+v", body)
	}
}

func TestGetLiquiditySweeps_BadSymbol(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/liquidity-sweeps", nil)
	rr := httptest.NewRecorder()
	h.GetLiquiditySweeps(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestGetVolumeSurge_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/volume-surge?symbol=BTCUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetVolumeSurge(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body volumeSurgeResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "BTCUSDT" {
		t.Fatalf("%+v", body)
	}
}

func TestGetVolumeSurge_BadSymbol(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/volume-surge", nil)
	rr := httptest.NewRecorder()
	h.GetVolumeSurge(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestScanVolumeSurges_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/volume-surge/scan", nil)
	rr := httptest.NewRecorder()
	h.ScanVolumeSurges(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body volumeSurgeScanResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Quote != "USDT" {
		t.Fatalf("%+v", body)
	}
}

func TestGetIcebergs_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/orderbook/icebergs?symbol=BTCUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetIcebergs(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body icebergsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "BTCUSDT" || body.Summary == "" {
		t.Fatalf("%+v", body)
	}
}

func TestGetIcebergs_BadSymbol(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/orderbook/icebergs", nil)
	rr := httptest.NewRecorder()
	h.GetIcebergs(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestGetBookHistory_NotConfigured(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/orderbook/history?symbol=BTCUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetBookHistory(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status=%d", rr.Code)
	}
}

type stubBookHistHTTP struct{}

func (stubBookHistHTTP) SnapshotAt(_ context.Context, _, symbol string, at time.Time) (*domain.BookHistorySnapshot, error) {
	return &domain.BookHistorySnapshot{
		Exchange: domain.ExchangeBinance, Symbol: symbol, SampledAt: at,
		Mid: 100, Spread: 0.2, BidNotional: 500, AskNotional: 400,
		Bids: []domain.BookHistoryLevel{{Price: 99.9, Quantity: 1, Notional: 99.9}},
	}, nil
}
func (stubBookHistHTTP) List(_ context.Context, q domain.BookHistoryQuery) ([]domain.BookHistorySnapshot, error) {
	return []domain.BookHistorySnapshot{{
		Exchange: domain.ExchangeBinance, Symbol: q.Symbol,
		SampledAt: time.Unix(1_700_000_000, 0).UTC(), Mid: 100, BidNotional: 500, AskNotional: 400,
	}}, nil
}
func (stubBookHistHTTP) Compare(_ context.Context, _, symbol string, from, to time.Time) (*domain.BookHistoryDiff, error) {
	fromS := domain.BookHistorySnapshot{Symbol: symbol, SampledAt: from, Mid: 100, BidNotional: 500, Bids: []domain.BookHistoryLevel{{Price: 99, Quantity: 2, Notional: 198}}}
	toS := domain.BookHistorySnapshot{Symbol: symbol, SampledAt: to, Mid: 102, BidNotional: 200, Bids: []domain.BookHistoryLevel{{Price: 99, Quantity: 1, Notional: 99}}}
	d := domain.CompareBookHistory(fromS, toS)
	return &d, nil
}
func (stubBookHistHTTP) Note(string, string) {}

func TestGetBookHistory_OK(t *testing.T) {
	svc := market.New(stubMarket{}, stubSupply{}).WithBookHistory(stubBookHistHTTP{})
	h := NewMarketHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/orderbook/history?symbol=BTCUSDT&at=2026-08-16T12:00:00Z", nil)
	rr := httptest.NewRecorder()
	h.GetBookHistory(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body bookHistoryResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Snapshot == nil || body.Snapshot.Mid == "" {
		t.Fatalf("%+v", body)
	}
}

func TestCompareBookHistory_OK(t *testing.T) {
	svc := market.New(stubMarket{}, stubSupply{}).WithBookHistory(stubBookHistHTTP{})
	h := NewMarketHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/orderbook/history/compare?symbol=BTCUSDT&from=2026-08-16T12:00:00Z&to=2026-08-16T12:05:00Z", nil)
	rr := httptest.NewRecorder()
	h.CompareBookHistory(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body bookDiffResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Summary == "" {
		t.Fatalf("%+v", body)
	}
}

func TestCompareBookHistory_MissingTimes(t *testing.T) {
	svc := market.New(stubMarket{}, stubSupply{}).WithBookHistory(stubBookHistHTTP{})
	h := NewMarketHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/orderbook/history/compare?symbol=BTCUSDT", nil)
	rr := httptest.NewRecorder()
	h.CompareBookHistory(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestGetWhales_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/whales?symbol=BTCUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetWhales(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body whalesResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Exchange != "all" || len(body.Events) == 0 || body.Events[0].AvgPrice == "" || body.Events[0].FirstTime.IsZero() {
		t.Fatalf("%+v", body)
	}
}

func TestGetWhales_ScanOK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/whales", nil)
	rr := httptest.NewRecorder()
	h.GetWhales(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestGetWhales_BadExchange(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/whales?exchange=coinbase", nil)
	rr := httptest.NewRecorder()
	h.GetWhales(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
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

func TestGetHolders_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/holders?symbol=BTCUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetHolders(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body holdersResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.HolderCount != 50_000_000 || body.Asset != "BTCUSDT" || len(body.TopHolders) != 1 {
		t.Fatalf("body=%+v", body)
	}
	if body.TopHolders[0].Label != "Binance" {
		t.Fatalf("label=%q", body.TopHolders[0].Label)
	}
}

func TestGetHolders_MissingAsset(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/holders", nil)
	rr := httptest.NewRecorder()
	h.GetHolders(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
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

type mixedQuoteMarket struct{ stubMarket }

func (mixedQuoteMarket) ListSpotMarkets(_ context.Context) ([]domain.SpotMarket, error) {
	return []domain.SpotMarket{
		{Symbol: "BTCUSDT", BaseAsset: "BTC", QuoteAsset: "USDT", Status: "TRADING", LastPrice: "100000", QuoteVolume: "1"},
		{Symbol: "ETHEUR", BaseAsset: "ETH", QuoteAsset: "EUR", Status: "TRADING", LastPrice: "3200", QuoteVolume: "1"},
		{Symbol: "ETHBTC", BaseAsset: "ETH", QuoteAsset: "BTC", Status: "TRADING", LastPrice: "0.035", QuoteVolume: "1"},
	}, nil
}

func TestListSpotMarkets_QuoteFilterReturnsRawPairLastAndOmitsQuoteAsset(t *testing.T) {
	h := NewMarketHandler(market.New(mixedQuoteMarket{}, stubSupply{}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/spot?quote=EUR", nil)
	rr := httptest.NewRecorder()
	h.ListSpotMarkets(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var raw map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	items, _ := raw["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("EUR filter items=%s", rr.Body.String())
	}
	item := items[0].(map[string]any)
	if item["symbol"] != "ETHEUR" || item["lastPrice"] != "3200" {
		t.Fatalf("EUR last must stay in pair units, got %v", item)
	}
	if _, ok := item["quoteAsset"]; ok {
		t.Fatalf("public SpotMarket DTO must not include quoteAsset (OpenAPI): %v", item)
	}
	if _, ok := item["baseAsset"]; ok {
		t.Fatalf("public SpotMarket DTO must not include baseAsset: %v", item)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/market/spot?quote=BTC", nil)
	rr = httptest.NewRecorder()
	h.ListSpotMarkets(rr, req)
	if err := json.Unmarshal(rr.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	items, _ = raw["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("BTC filter items=%s", rr.Body.String())
	}
	item = items[0].(map[string]any)
	if item["symbol"] != "ETHBTC" || item["lastPrice"] != "0.035" {
		t.Fatalf("BTC last must stay 0.035 (pair units), got %v", item)
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
