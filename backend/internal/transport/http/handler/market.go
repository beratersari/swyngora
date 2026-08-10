package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
)

// MarketHandler is the transport adapter for market use cases.
type MarketHandler struct {
	svc *market.Service
}

// NewMarketHandler constructs the market HTTP handler.
func NewMarketHandler(svc *market.Service) *MarketHandler {
	return &MarketHandler{svc: svc}
}

type candleDTO struct {
	OpenTime    string `json:"openTime"`
	Open        string `json:"open"`
	High        string `json:"high"`
	Low         string `json:"low"`
	Close       string `json:"close"`
	Volume      string `json:"volume"`
	CloseTime   string `json:"closeTime"`
	QuoteVolume string `json:"quoteVolume"`
	TradeCount  int64  `json:"tradeCount"`
}

type candlesResponse struct {
	Symbol   string      `json:"symbol"`
	Interval string      `json:"interval"`
	Exchange string      `json:"exchange"`
	Candles  []candleDTO `json:"candles"`
}

// GetCandles handles GET /api/v1/market/candles
func (h *MarketHandler) GetCandles(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	symbol := q.Get("symbol")
	interval := q.Get("interval")
	if interval == "" {
		interval = "1h"
	}
	limit := 0
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, fmt.Errorf("%w: limit must be an integer", domain.ErrInvalidArgument))
			return
		}
		limit = n
	}

	var startPtr, endPtr *time.Time
	if raw := q.Get("startTime"); raw != "" {
		t, err := parseTimeParam(raw)
		if err != nil {
			writeError(w, err)
			return
		}
		startPtr = &t
	}
	if raw := q.Get("endTime"); raw != "" {
		t, err := parseTimeParam(raw)
		if err != nil {
			writeError(w, err)
			return
		}
		endPtr = &t
	}

	exchange := q.Get("exchange")
	candles, err := h.svc.GetCandles(r.Context(), exchange, symbol, interval, limit, startPtr, endPtr)
	if err != nil {
		writeError(w, err)
		return
	}
	ex, _ := h.svc.ResolveExchange(exchange)

	out := candlesResponse{
		Symbol:   symbol,
		Interval: interval,
		Exchange: string(ex),
		Candles:  make([]candleDTO, 0, len(candles)),
	}
	for _, c := range candles {
		out.Candles = append(out.Candles, candleDTO{
			OpenTime:    c.OpenTime.UTC().Format(time.RFC3339Nano),
			Open:        c.Open,
			High:        c.High,
			Low:         c.Low,
			Close:       c.Close,
			Volume:      c.Volume,
			CloseTime:   c.CloseTime.UTC().Format(time.RFC3339Nano),
			QuoteVolume: c.QuoteVolume,
			TradeCount:  c.TradeCount,
		})
	}
	out.Symbol = strings.ToUpper(strings.TrimSpace(symbol))
	writeJSON(w, http.StatusOK, out)
}

type tickerResponse struct {
	Exchange           string `json:"exchange"`
	Symbol             string `json:"symbol"`
	PriceChange        string `json:"priceChange"`
	PriceChangePercent string `json:"priceChangePercent"`
	LastPrice          string `json:"lastPrice"`
	OpenPrice          string `json:"openPrice"`
	HighPrice          string `json:"highPrice"`
	LowPrice           string `json:"lowPrice"`
	Volume             string `json:"volume"`
	QuoteVolume        string `json:"quoteVolume"`
	OpenTime           string `json:"openTime"`
	CloseTime          string `json:"closeTime"`
	TradeCount         int64  `json:"tradeCount"`
}

type orderBookLevelDTO struct {
	Price              string `json:"price"`
	Quantity           string `json:"quantity"`
	Notional           string `json:"notional"`
	Cumulative         string `json:"cumulative"`
	CumulativeNotional string `json:"cumulativeNotional"`
	RawCount           int    `json:"rawCount"`
	IsWall             bool   `json:"isWall"`
}

type orderBookResponse struct {
	Exchange            string               `json:"exchange"`
	Symbol              string               `json:"symbol"`
	LastPrice           string               `json:"lastPrice"`
	BestBid             string               `json:"bestBid"`
	BestAsk             string               `json:"bestAsk"`
	Spread              string               `json:"spread"`
	SpreadPct           string               `json:"spreadPct"`
	GroupSize           string               `json:"groupSize"`
	SuggestedGroupSizes []string             `json:"suggestedGroupSizes"`
	Levels              int                  `json:"levels"`
	Bids                []orderBookLevelDTO  `json:"bids"`
	Asks                []orderBookLevelDTO  `json:"asks"`
	BidVolume           string               `json:"bidVolume"`
	AskVolume           string               `json:"askVolume"`
	Imbalance           float64              `json:"imbalance"`
	BidWalls            int                  `json:"bidWalls"`
	AskWalls            int                  `json:"askWalls"`
	Analysis            orderBookAnalysisDTO `json:"analysis"`
	UpdatedAt           string               `json:"updatedAt"`
	Live                bool                 `json:"live"`
	Source              string               `json:"source"`
	Note                string               `json:"note"`
}

type orderBookWallDTO struct {
	Side        string  `json:"side"`
	Price       string  `json:"price"`
	Quantity    string  `json:"quantity"`
	Notional    string  `json:"notional"`
	DistancePct string  `json:"distancePct"`
	Share       float64 `json:"share"`
}

type orderBookBandDTO struct {
	RangePct    float64 `json:"rangePct"`
	BidNotional string  `json:"bidNotional"`
	AskNotional string  `json:"askNotional"`
	BidQuantity string  `json:"bidQuantity"`
	AskQuantity string  `json:"askQuantity"`
	Imbalance   float64 `json:"imbalance"`
	BidLevels   int     `json:"bidLevels"`
	AskLevels   int     `json:"askLevels"`
}

type orderBookAnalysisDTO struct {
	RangePct      float64            `json:"rangePct"`
	MidPrice      string             `json:"midPrice"`
	BidNotional   string             `json:"bidNotional"`
	AskNotional   string             `json:"askNotional"`
	BidQuantity   string             `json:"bidQuantity"`
	AskQuantity   string             `json:"askQuantity"`
	Imbalance     float64            `json:"imbalance"`
	Pressure      string             `json:"pressure"`
	BidLevels     int                `json:"bidLevels"`
	AskLevels     int                `json:"askLevels"`
	CoveredBidPct string             `json:"coveredBidPct"`
	CoveredAskPct string             `json:"coveredAskPct"`
	Walls         []orderBookWallDTO `json:"walls"`
	Bands         []orderBookBandDTO `json:"bands"`
}

// GetOrderBook handles GET /api/v1/market/orderbook — grouped spot depth.
func (h *MarketHandler) GetOrderBook(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	levels := 0
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, fmt.Errorf("%w: limit must be an integer", domain.ErrInvalidArgument))
			return
		}
		levels = n
	}
	rangePct, err := domain.ParseRangePct(q.Get("rangePct"))
	if err != nil {
		writeError(w, err)
		return
	}
	book, err := h.svc.GetSpotOrderBook(r.Context(), q.Get("exchange"), q.Get("symbol"), q.Get("group"), levels, rangePct)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, orderBookToDTO(book))
}

func orderBookToDTO(book *domain.OrderBook) orderBookResponse {
	if book == nil {
		return orderBookResponse{}
	}
	mapLevels := func(in []domain.OrderBookLevel) []orderBookLevelDTO {
		out := make([]orderBookLevelDTO, 0, len(in))
		for _, lv := range in {
			out = append(out, orderBookLevelDTO{
				Price: lv.Price, Quantity: lv.Quantity, Notional: lv.Notional,
				Cumulative: lv.Cumulative, CumulativeNotional: lv.CumulativeNotional,
				RawCount: lv.RawCount, IsWall: lv.IsWall,
			})
		}
		return out
	}
	return orderBookResponse{
		Exchange: string(book.Exchange), Symbol: book.Symbol, LastPrice: book.LastPrice,
		BestBid: book.BestBid, BestAsk: book.BestAsk, Spread: book.Spread, SpreadPct: book.SpreadPct,
		GroupSize: book.GroupSize, SuggestedGroupSizes: book.SuggestedGroupSizes, Levels: book.Levels,
		Bids: mapLevels(book.Bids), Asks: mapLevels(book.Asks),
		BidVolume: book.BidVolume, AskVolume: book.AskVolume, Imbalance: book.Imbalance,
		BidWalls: book.BidWalls, AskWalls: book.AskWalls,
		Analysis:  analysisToDTO(book.Analysis),
		UpdatedAt: book.UpdatedAt.UTC().Format(time.RFC3339Nano),
		Live:      book.Live,
		Source:    book.Source,
		Note:      "Spot order book with backend grouping plus ±rangePct pressure/wall analysis over live depth (not only the first rows). Informational only.",
	}
}

func analysisToDTO(a domain.OrderBookAnalysis) orderBookAnalysisDTO {
	walls := make([]orderBookWallDTO, 0, len(a.Walls))
	for _, w := range a.Walls {
		walls = append(walls, orderBookWallDTO{
			Side: w.Side, Price: w.Price, Quantity: w.Quantity, Notional: w.Notional,
			DistancePct: w.DistancePct, Share: w.Share,
		})
	}
	bands := make([]orderBookBandDTO, 0, len(a.Bands))
	for _, b := range a.Bands {
		bands = append(bands, orderBookBandDTO{
			RangePct: b.RangePct, BidNotional: b.BidNotional, AskNotional: b.AskNotional,
			BidQuantity: b.BidQuantity, AskQuantity: b.AskQuantity, Imbalance: b.Imbalance,
			BidLevels: b.BidLevels, AskLevels: b.AskLevels,
		})
	}
	return orderBookAnalysisDTO{
		RangePct: a.RangePct, MidPrice: a.MidPrice,
		BidNotional: a.BidNotional, AskNotional: a.AskNotional,
		BidQuantity: a.BidQuantity, AskQuantity: a.AskQuantity,
		Imbalance: a.Imbalance, Pressure: a.Pressure,
		BidLevels: a.BidLevels, AskLevels: a.AskLevels,
		CoveredBidPct: a.CoveredBidPct, CoveredAskPct: a.CoveredAskPct,
		Walls: walls, Bands: bands,
	}
}

// GetTicker24h handles GET /api/v1/market/ticker/24h
func (h *MarketHandler) GetTicker24h(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	symbol := q.Get("symbol")
	exchange := q.Get("exchange")
	tkr, err := h.svc.GetTicker24h(r.Context(), exchange, symbol)
	if err != nil {
		writeError(w, err)
		return
	}
	ex, _ := h.svc.ResolveExchange(exchange)
	writeJSON(w, http.StatusOK, tickerResponse{
		Exchange:           string(ex),
		Symbol:             tkr.Symbol,
		PriceChange:        tkr.PriceChange,
		PriceChangePercent: tkr.PriceChangePercent,
		LastPrice:          tkr.LastPrice,
		OpenPrice:          tkr.OpenPrice,
		HighPrice:          tkr.HighPrice,
		LowPrice:           tkr.LowPrice,
		Volume:             tkr.Volume,
		QuoteVolume:        tkr.QuoteVolume,
		OpenTime:           tkr.OpenTime.UTC().Format(time.RFC3339Nano),
		CloseTime:          tkr.CloseTime.UTC().Format(time.RFC3339Nano),
		TradeCount:         tkr.TradeCount,
	})
}

type supplyResponse struct {
	Asset             string   `json:"asset"`
	Name              string   `json:"name"`
	ProviderID        string   `json:"providerId"`
	CirculatingSupply *float64 `json:"circulatingSupply"`
	TotalSupply       *float64 `json:"totalSupply"`
	MaxSupply         *float64 `json:"maxSupply"`
	CurrentPriceUSD   *float64 `json:"currentPriceUsd"`
	AsOf              string   `json:"asOf"`
	Source            string   `json:"source"`
	Note              string   `json:"note"`
}

// GetSupply handles GET /api/v1/market/supply
func (h *MarketHandler) GetSupply(w http.ResponseWriter, r *http.Request) {
	asset := r.URL.Query().Get("asset")
	if asset == "" {
		// Convenience: also accept symbol=BTCUSDT and strip to base.
		asset = r.URL.Query().Get("symbol")
	}
	sup, err := h.svc.GetSupply(r.Context(), asset)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, supplyResponse{
		Asset:             sup.Asset,
		Name:              sup.Name,
		ProviderID:        sup.ProviderID,
		CirculatingSupply: domain.CloneFloatPtr(sup.CirculatingSupply),
		TotalSupply:       domain.CloneFloatPtr(sup.TotalSupply),
		MaxSupply:         domain.CloneFloatPtr(sup.MaxSupply),
		CurrentPriceUSD:   domain.CloneFloatPtr(sup.CurrentPriceUSD),
		AsOf:              sup.AsOf.UTC().Format(time.RFC3339Nano),
		Source:            sup.Source,
		Note:              "Circulating / total / max supply from Binance marketing symbol list (daily snapshot @ 03:00 UTC plus startup). Max may be null when undefined. Request path is cache-only.",
	})
}

type intervalsResponse struct {
	Intervals []string `json:"intervals"`
	Exchange  string   `json:"exchange"`
}

// GetIntervals handles GET /api/v1/market/intervals
func (h *MarketHandler) GetIntervals(w http.ResponseWriter, r *http.Request) {
	exchange := r.URL.Query().Get("exchange")
	ivs, err := h.svc.ListIntervals(exchange)
	if err != nil {
		writeError(w, err)
		return
	}
	ex, _ := h.svc.ResolveExchange(exchange)
	out := make([]string, len(ivs))
	for i, iv := range ivs {
		out[i] = string(iv)
	}
	writeJSON(w, http.StatusOK, intervalsResponse{
		Intervals: out,
		Exchange:  string(ex),
	})
}

func parseTimeParam(raw string) (time.Time, error) {
	// Accept RFC3339 or Unix milliseconds.
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}
	if ms, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return time.UnixMilli(ms).UTC(), nil
	}
	return time.Time{}, fmt.Errorf("%w: time must be RFC3339 or unix milliseconds", domain.ErrInvalidArgument)
}

type spotMarketDTO struct {
	Symbol               string   `json:"symbol"`
	LastPrice            string   `json:"lastPrice"`
	PriceChange          string   `json:"priceChange"`
	PriceChangePercent   string   `json:"priceChangePercent"`
	HighPrice            string   `json:"highPrice"`
	LowPrice             string   `json:"lowPrice"`
	Volume               string   `json:"volume"`
	QuoteVolume          string   `json:"quoteVolume"`
	TradeCount           int64    `json:"tradeCount"`
	Tags                 []string `json:"tags"`
	CirculatingSupply    *float64 `json:"circulatingSupply"`
	TotalSupply          *float64 `json:"totalSupply"`
	MaxSupply            *float64 `json:"maxSupply"`
	MarketCapCirculating *float64 `json:"marketCapCirculating"`
	MarketCapTotal       *float64 `json:"marketCapTotal"`
	// MarketCapMax is a number, the string "∞" when max supply is undefined, or null.
	MarketCapMax any     `json:"marketCapMax"`
	DelistTime   *string `json:"delistTime,omitempty"`
}

type spotListResponse struct {
	Exchange string          `json:"exchange"`
	Query    string          `json:"query"`
	Tag      string          `json:"tag,omitempty"`
	Sort     string          `json:"sort"`
	Order    string          `json:"order"`
	Total    int             `json:"total"`
	Limit    int             `json:"limit"`
	Offset   int             `json:"offset"`
	Items    []spotMarketDTO `json:"items"`
}

type productTagsResponse struct {
	Exchange string   `json:"exchange"`
	Tags     []string `json:"tags"`
}

// ListProductTags handles GET /api/v1/market/tags
func (h *MarketHandler) ListProductTags(w http.ResponseWriter, r *http.Request) {
	exchange := r.URL.Query().Get("exchange")
	tags, err := h.svc.ListProductTags(r.Context(), exchange)
	if err != nil {
		writeError(w, err)
		return
	}
	if tags == nil {
		tags = []string{}
	}
	ex, _ := h.svc.ResolveExchange(exchange)
	writeJSON(w, http.StatusOK, productTagsResponse{
		Exchange: string(ex),
		Tags:     tags,
	})
}

type exchangesResponse struct {
	Exchanges []string `json:"exchanges"`
	Default   string   `json:"default"`
}

// ListExchanges handles GET /api/v1/market/exchanges
func (h *MarketHandler) ListExchanges(w http.ResponseWriter, r *http.Request) {
	exs := h.svc.ListExchanges()
	out := make([]string, len(exs))
	for i, e := range exs {
		out[i] = string(e)
	}
	writeJSON(w, http.StatusOK, exchangesResponse{
		Exchanges: out,
		Default:   string(domain.DefaultExchange),
	})
}

// ListSpotMarkets handles GET /api/v1/market/spot
func (h *MarketHandler) ListSpotMarkets(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 0
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, fmt.Errorf("%w: limit must be an integer", domain.ErrInvalidArgument))
			return
		}
		limit = n
	}
	offset := 0
	if raw := q.Get("offset"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, fmt.Errorf("%w: offset must be an integer", domain.ErrInvalidArgument))
			return
		}
		offset = n
	}

	// tag / tags query: comma-separated or repeated (OR match).
	tagFilters := append([]string{}, q["tag"]...)
	tagFilters = append(tagFilters, q["tags"]...)

	res, err := h.svc.ListSpotMarkets(r.Context(), q.Get("exchange"), domain.SpotListQuery{
		Query:      q.Get("q"),
		QuoteAsset: q.Get("quote"),
		BaseAsset:  q.Get("base"),
		Status:     q.Get("status"),
		Tags:       tagFilters,
		SortBy:     domain.SpotSortField(q.Get("sort")),
		Order:      domain.SortOrder(q.Get("order")),
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	out := spotListResponse{
		Exchange: string(res.Exchange),
		Query:    res.Query,
		Tag:      strings.Join(res.Tags, ","),
		Sort:     string(res.SortBy),
		Order:    string(res.Order),
		Total:    res.Total,
		Limit:    res.Limit,
		Offset:   res.Offset,
		Items:    make([]spotMarketDTO, 0, len(res.Items)),
	}
	for _, m := range res.Items {
		tags := m.Tags
		if tags == nil {
			tags = []string{}
		}
		dto := spotMarketDTO{
			Symbol:               m.Symbol,
			LastPrice:            m.LastPrice,
			PriceChange:          m.PriceChange,
			PriceChangePercent:   m.PriceChangePercent,
			HighPrice:            m.HighPrice,
			LowPrice:             m.LowPrice,
			Volume:               m.Volume,
			QuoteVolume:          m.QuoteVolume,
			TradeCount:           m.TradeCount,
			Tags:                 append([]string(nil), tags...),
			CirculatingSupply:    domain.CloneFloatPtr(m.CirculatingSupply),
			TotalSupply:          domain.CloneFloatPtr(m.TotalSupply),
			MaxSupply:            domain.CloneFloatPtr(m.MaxSupply),
			MarketCapCirculating: domain.CloneFloatPtr(m.MarketCapCirculating),
			MarketCapTotal:       domain.CloneFloatPtr(m.MarketCapTotal),
			MarketCapMax:         encodeMarketCapMax(m),
		}
		if m.DelistTime != nil && !m.DelistTime.IsZero() {
			s := m.DelistTime.UTC().Format(time.RFC3339)
			dto.DelistTime = &s
		}
		out.Items = append(out.Items, dto)
	}
	writeJSON(w, http.StatusOK, out)
}

func encodeMarketCapMax(m domain.SpotMarket) any {
	if m.MarketCapMaxInfinite {
		return "∞"
	}
	if m.MarketCapMax != nil {
		return *m.MarketCapMax
	}
	return nil
}

type delistScheduleResponse struct {
	Exchange string           `json:"exchange"`
	Enabled  bool             `json:"enabled"`
	Items    []delistEntryDTO `json:"items"`
}

type delistEntryDTO struct {
	Symbol     string `json:"symbol"`
	DelistTime string `json:"delistTime"`
}

// ListDelistSchedule handles GET /api/v1/market/delist-schedule
func (h *MarketHandler) ListDelistSchedule(w http.ResponseWriter, r *http.Request) {
	exchange := r.URL.Query().Get("exchange")
	if exchange == "" {
		exchange = string(domain.ExchangeBinance)
	}
	entries, err := h.svc.ListDelistSchedule(exchange)
	if err != nil {
		writeError(w, err)
		return
	}
	ex, _ := h.svc.ResolveExchange(exchange)
	out := delistScheduleResponse{
		Exchange: string(ex),
		Enabled:  h.svc.DelistEnabled(),
		Items:    make([]delistEntryDTO, 0, len(entries)),
	}
	for _, e := range entries {
		out.Items = append(out.Items, delistEntryDTO{
			Symbol:     e.Symbol,
			DelistTime: e.DelistTime.UTC().Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, out)
}
