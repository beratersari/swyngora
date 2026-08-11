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
	Side              string  `json:"side"`
	Price             string  `json:"price"`
	Quantity          string  `json:"quantity"`
	Notional          string  `json:"notional"`
	DistancePct       string  `json:"distancePct"`
	Share             float64 `json:"share"`
	Behavior          string  `json:"behavior,omitempty"`
	AgeSeconds        float64 `json:"ageSeconds,omitempty"`
	PresentForSeconds float64 `json:"presentForSeconds,omitempty"`
	VisibleSeconds    float64 `json:"visibleSeconds,omitempty"`
	AppearCount       int     `json:"appearCount,omitempty"`
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

// GetCombinedOrderBook handles GET /api/v1/market/orderbook/combined.
func (h *MarketHandler) GetCombinedOrderBook(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	rangePct, err := domain.ParseRangePct(q.Get("rangePct"))
	if err != nil {
		writeError(w, err)
		return
	}
	got, err := h.svc.GetCombinedOrderBookAnalysis(r.Context(), q.Get("symbol"), rangePct)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, combinedBookToDTO(got))
}

// GetLiquidations handles GET /api/v1/market/liquidations.
func (h *MarketHandler) GetLiquidations(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.GetLiquidations(r.Context(), q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, liquidationsToDTO(got))
}

// GetOpenInterest handles GET /api/v1/market/open-interest.
func (h *MarketHandler) GetOpenInterest(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.GetOpenInterest(r.Context(), q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, openInterestToDTO(got))
}

// GetFundingRate handles GET /api/v1/market/funding-rate.
func (h *MarketHandler) GetFundingRate(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 0
	if raw := strings.TrimSpace(q.Get("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, fmt.Errorf("%w: limit must be an integer", domain.ErrInvalidArgument))
			return
		}
		limit = n
	}
	got, err := h.svc.GetFundingRate(r.Context(), q.Get("exchange"), q.Get("symbol"), limit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, fundingToDTO(got))
}

// GetLongShortRatio handles GET /api/v1/market/long-short-ratio.
func (h *MarketHandler) GetLongShortRatio(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 0
	if raw := strings.TrimSpace(q.Get("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, fmt.Errorf("%w: limit must be an integer", domain.ErrInvalidArgument))
			return
		}
		limit = n
	}
	got, err := h.svc.GetLongShortRatio(r.Context(), q.Get("exchange"), q.Get("symbol"), limit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, longShortToDTO(got))
}

// GetMarketLiquidity handles GET /api/v1/market/orderbook/liquidity.
func (h *MarketHandler) GetMarketLiquidity(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.GetMarketLiquidity(r.Context(), q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, liquidityToDTO(got))
}

// GetOrderBookImpact handles GET /api/v1/market/orderbook/impact.
func (h *MarketHandler) GetOrderBookImpact(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var qty, notional float64
	if raw := strings.TrimSpace(q.Get("quantity")); raw != "" {
		n, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			writeError(w, fmt.Errorf("%w: quantity must be a number", domain.ErrInvalidArgument))
			return
		}
		qty = n
	}
	if raw := strings.TrimSpace(q.Get("notional")); raw != "" {
		n, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			writeError(w, fmt.Errorf("%w: notional must be a number", domain.ErrInvalidArgument))
			return
		}
		notional = n
	}
	got, err := h.svc.EstimateOrderBookImpact(r.Context(), q.Get("exchange"), q.Get("symbol"), q.Get("side"), qty, notional)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, impactToDTO(got))
}

type impactFillDTO struct {
	Exchange           string `json:"exchange"`
	Price              string `json:"price"`
	Quantity           string `json:"quantity"`
	Notional           string `json:"notional"`
	CumulativeQuantity string `json:"cumulativeQuantity"`
	CumulativeNotional string `json:"cumulativeNotional"`
}

type orderBookImpactResponse struct {
	Symbol            string          `json:"symbol"`
	Scope             string          `json:"scope"`
	Side              string          `json:"side"`
	MidPrice          string          `json:"midPrice"`
	BestPrice         string          `json:"bestPrice"`
	AveragePrice      string          `json:"averagePrice"`
	EndPrice          string          `json:"endPrice"`
	NewBestPrice      string          `json:"newBestPrice,omitempty"`
	RequestedQuantity string          `json:"requestedQuantity,omitempty"`
	RequestedNotional string          `json:"requestedNotional,omitempty"`
	FilledQuantity    string          `json:"filledQuantity"`
	SpentNotional     string          `json:"spentNotional"`
	UnfilledQuantity  string          `json:"unfilledQuantity,omitempty"`
	UnfilledNotional  string          `json:"unfilledNotional,omitempty"`
	VisibleQuantity   string          `json:"visibleQuantity"`
	VisibleNotional   string          `json:"visibleNotional"`
	SlippagePct       float64         `json:"slippagePct"`
	SlippageVsBestPct float64         `json:"slippageVsBestPct"`
	ImpactPct         float64         `json:"impactPct"`
	ImpactAvailable   bool            `json:"impactAvailable"`
	ImpactNote        string          `json:"impactNote,omitempty"`
	Exhausted         bool            `json:"exhausted"`
	LevelsUsed        int             `json:"levelsUsed"`
	VenueCount        int             `json:"venueCount"`
	Live              bool            `json:"live"`
	Fills             []impactFillDTO `json:"fills"`
	Note              string          `json:"note"`
}

func impactToDTO(a *domain.OrderBookImpact) orderBookImpactResponse {
	if a == nil {
		return orderBookImpactResponse{}
	}
	fills := make([]impactFillDTO, 0, len(a.Fills))
	for _, f := range a.Fills {
		fills = append(fills, impactFillDTO{
			Exchange: f.Exchange, Price: f.Price, Quantity: f.Quantity, Notional: f.Notional,
			CumulativeQuantity: f.CumulativeQuantity, CumulativeNotional: f.CumulativeNotional,
		})
	}
	return orderBookImpactResponse{
		Symbol: a.Symbol, Scope: a.Scope, Side: a.Side,
		MidPrice: a.MidPrice, BestPrice: a.BestPrice, AveragePrice: a.AveragePrice, EndPrice: a.EndPrice,
		NewBestPrice:      a.NewBestPrice,
		RequestedQuantity: a.RequestedQuantity, RequestedNotional: a.RequestedNotional,
		FilledQuantity: a.FilledQuantity, SpentNotional: a.SpentNotional,
		UnfilledQuantity: a.UnfilledQuantity, UnfilledNotional: a.UnfilledNotional,
		VisibleQuantity: a.VisibleQuantity, VisibleNotional: a.VisibleNotional,
		SlippagePct: a.SlippagePct, SlippageVsBestPct: a.SlippageVsBestPct, ImpactPct: a.ImpactPct,
		ImpactAvailable: a.ImpactAvailable, ImpactNote: a.ImpactNote,
		Exhausted: a.Exhausted, LevelsUsed: a.LevelsUsed, VenueCount: a.VenueCount, Live: a.Live,
		Fills: fills,
		Note:  "Simulated market order walking live resting depth. Not a quote, fill, or financial advice. Visible book may be thinner than the real market.",
	}
}

type combinedWallDTO struct {
	Exchange          string  `json:"exchange"`
	Side              string  `json:"side"`
	Price             string  `json:"price"`
	Quantity          string  `json:"quantity"`
	Notional          string  `json:"notional"`
	DistancePct       string  `json:"distancePct"`
	Share             float64 `json:"share"`
	Behavior          string  `json:"behavior,omitempty"`
	AgeSeconds        float64 `json:"ageSeconds,omitempty"`
	PresentForSeconds float64 `json:"presentForSeconds,omitempty"`
	VisibleSeconds    float64 `json:"visibleSeconds,omitempty"`
	AppearCount       int     `json:"appearCount,omitempty"`
}

type combinedVenueDTO struct {
	Exchange    string  `json:"exchange"`
	Symbol      string  `json:"symbol"`
	Live        bool    `json:"live"`
	Source      string  `json:"source,omitempty"`
	BidNotional string  `json:"bidNotional,omitempty"`
	AskNotional string  `json:"askNotional,omitempty"`
	BidQuantity string  `json:"bidQuantity,omitempty"`
	AskQuantity string  `json:"askQuantity,omitempty"`
	Imbalance   float64 `json:"imbalance"`
	Pressure    string  `json:"pressure,omitempty"`
	BidLevels   int     `json:"bidLevels"`
	AskLevels   int     `json:"askLevels"`
	Error       string  `json:"error,omitempty"`
}

type combinedOrderBookResponse struct {
	Symbol           string             `json:"symbol"`
	RangePct         float64            `json:"rangePct"`
	MidPrice         string             `json:"midPrice"`
	UsedLow          string             `json:"usedLow"`
	UsedHigh         string             `json:"usedHigh"`
	UsedRangePct     float64            `json:"usedRangePct"`
	RequestedReached bool               `json:"requestedReached"`
	BidNotional      string             `json:"bidNotional"`
	AskNotional      string             `json:"askNotional"`
	BidQuantity      string             `json:"bidQuantity"`
	AskQuantity      string             `json:"askQuantity"`
	Imbalance        float64            `json:"imbalance"`
	Pressure         string             `json:"pressure"`
	BidLevels        int                `json:"bidLevels"`
	AskLevels        int                `json:"askLevels"`
	CoveredBidPct    string             `json:"coveredBidPct"`
	CoveredAskPct    string             `json:"coveredAskPct"`
	Walls            []combinedWallDTO  `json:"walls"`
	Bands            []orderBookBandDTO `json:"bands"`
	Venues           []combinedVenueDTO `json:"venues"`
	VenueCount       int                `json:"venueCount"`
	Note             string             `json:"note"`
}

func combinedBookToDTO(a *domain.CombinedOrderBookAnalysis) combinedOrderBookResponse {
	if a == nil {
		return combinedOrderBookResponse{}
	}
	walls := make([]combinedWallDTO, 0, len(a.Walls))
	for _, w := range a.Walls {
		walls = append(walls, combinedWallDTO{
			Exchange: w.Exchange, Side: w.Side, Price: w.Price, Quantity: w.Quantity,
			Notional: w.Notional, DistancePct: w.DistancePct, Share: w.Share,
			Behavior: w.Behavior, AgeSeconds: w.AgeSeconds, PresentForSeconds: w.PresentForSeconds,
			VisibleSeconds: w.VisibleSeconds, AppearCount: w.AppearCount,
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
	venues := make([]combinedVenueDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		venues = append(venues, combinedVenueDTO{
			Exchange: string(v.Exchange), Symbol: v.Symbol, Live: v.Live, Source: v.Source,
			BidNotional: v.BidNotional, AskNotional: v.AskNotional,
			BidQuantity: v.BidQuantity, AskQuantity: v.AskQuantity,
			Imbalance: v.Imbalance, Pressure: v.Pressure,
			BidLevels: v.BidLevels, AskLevels: v.AskLevels, Error: v.Error,
		})
	}
	return combinedOrderBookResponse{
		Symbol: a.Symbol, RangePct: a.RangePct, MidPrice: a.MidPrice,
		UsedLow: a.UsedLow, UsedHigh: a.UsedHigh, UsedRangePct: a.UsedRangePct, RequestedReached: a.RequestedReached,
		BidNotional: a.BidNotional, AskNotional: a.AskNotional,
		BidQuantity: a.BidQuantity, AskQuantity: a.AskQuantity,
		Imbalance: a.Imbalance, Pressure: a.Pressure,
		BidLevels: a.BidLevels, AskLevels: a.AskLevels,
		CoveredBidPct: a.CoveredBidPct, CoveredAskPct: a.CoveredAskPct,
		Walls: walls, Bands: bands, Venues: venues, VenueCount: a.VenueCount,
		Note: "Market-wide spot depth: only the symmetric ±usedRangePct band every venue can reach on both sides is summed. If all venues cover ±rangePct both ways, that requested band is used. USD and USDT treated as 1:1. Informational only.",
	}
}

type liquidityBandDTO struct {
	RangePct      float64 `json:"rangePct"`
	BidNotional   string  `json:"bidNotional"`
	AskNotional   string  `json:"askNotional"`
	BidQuantity   string  `json:"bidQuantity"`
	AskQuantity   string  `json:"askQuantity"`
	TotalNotional string  `json:"totalNotional"`
	Imbalance     float64 `json:"imbalance"`
	Score         float64 `json:"score"`
}

type liquidityScoreDTO struct {
	MidPrice     string             `json:"midPrice"`
	UsedRangePct float64            `json:"usedRangePct"`
	Score        float64            `json:"score"`
	Grade        string             `json:"grade"`
	WeakerSide   string             `json:"weakerSide"`
	Weakness     float64            `json:"weakness"`
	Bands        []liquidityBandDTO `json:"bands"`
}

type venueLiquidityDTO struct {
	Exchange string `json:"exchange"`
	Symbol   string `json:"symbol"`
	Live     bool   `json:"live"`
	Error    string `json:"error,omitempty"`
	liquidityScoreDTO
}

type marketLiquidityResponse struct {
	Symbol     string              `json:"symbol"`
	VenueCount int                 `json:"venueCount"`
	Market     liquidityScoreDTO   `json:"market"`
	Venues     []venueLiquidityDTO `json:"venues"`
	Note       string              `json:"note"`
}

func liquidityToDTO(a *domain.MarketLiquidity) marketLiquidityResponse {
	if a == nil {
		return marketLiquidityResponse{}
	}
	mapScore := func(s domain.LiquidityScore) liquidityScoreDTO {
		bands := make([]liquidityBandDTO, 0, len(s.Bands))
		for _, b := range s.Bands {
			bands = append(bands, liquidityBandDTO{
				RangePct: b.RangePct, BidNotional: b.BidNotional, AskNotional: b.AskNotional,
				BidQuantity: b.BidQuantity, AskQuantity: b.AskQuantity, TotalNotional: b.TotalNotional,
				Imbalance: b.Imbalance, Score: b.Score,
			})
		}
		return liquidityScoreDTO{
			MidPrice: s.MidPrice, UsedRangePct: s.UsedRangePct,
			Score: s.Score, Grade: s.Grade,
			WeakerSide: s.WeakerSide, Weakness: s.Weakness, Bands: bands,
		}
	}
	venues := make([]venueLiquidityDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		venues = append(venues, venueLiquidityDTO{
			Exchange: string(v.Exchange), Symbol: v.Symbol, Live: v.Live, Error: v.Error,
			liquidityScoreDTO: mapScore(v.LiquidityScore),
		})
	}
	return marketLiquidityResponse{
		Symbol: a.Symbol, VenueCount: a.VenueCount, Market: mapScore(a.Market), Venues: venues,
		Note: "Liquidity score 0–100 from resting bid/ask notional. Only ±0.1 / ±0.5 / ±1% bands the book actually reaches on both sides are shown. Market-wide uses the symmetric ±% every venue can reach. weakerSide is the thinner side in the widest included band. Informational only — not a quote.",
	}
}

type liquidationHitDTO struct {
	Exchange string `json:"exchange"`
	Side     string `json:"side"`
	Price    string `json:"price"`
	Quantity string `json:"quantity"`
	Notional string `json:"notional"`
	Time     string `json:"time"`
}

type liquidationWindowDTO struct {
	Window          string             `json:"window"`
	LongNotional    string             `json:"longNotional"`
	ShortNotional   string             `json:"shortNotional"`
	TotalNotional   string             `json:"totalNotional"`
	Count           int                `json:"count"`
	Biggest         *liquidationHitDTO `json:"biggest,omitempty"`
	CoverageSeconds int64              `json:"coverageSeconds"`
	Complete        bool               `json:"complete"`
}

type liquidationsResponse struct {
	Symbol          string                 `json:"symbol"`
	Exchange        string                 `json:"exchange"`
	CollectingSince string                 `json:"collectingSince,omitempty"`
	Live            bool                   `json:"live"`
	VenueCount      int                    `json:"venueCount"`
	Windows         []liquidationWindowDTO `json:"windows"`
	Note            string                 `json:"note"`
}

func liquidationsToDTO(a *domain.LiquidationSnapshot) liquidationsResponse {
	if a == nil {
		return liquidationsResponse{}
	}
	wins := make([]liquidationWindowDTO, 0, len(a.Windows))
	for _, w := range a.Windows {
		row := liquidationWindowDTO{
			Window: w.Window, LongNotional: w.LongNotional, ShortNotional: w.ShortNotional,
			TotalNotional: w.TotalNotional, Count: w.Count,
			CoverageSeconds: w.CoverageSeconds, Complete: w.Complete,
		}
		if w.Biggest != nil {
			row.Biggest = &liquidationHitDTO{
				Exchange: w.Biggest.Exchange, Side: w.Biggest.Side,
				Price: w.Biggest.Price, Quantity: w.Biggest.Quantity, Notional: w.Biggest.Notional,
				Time: w.Biggest.Time.UTC().Format(time.RFC3339Nano),
			}
		}
		wins = append(wins, row)
	}
	since := ""
	if !a.CollectingSince.IsZero() {
		since = a.CollectingSince.UTC().Format(time.RFC3339Nano)
	}
	return liquidationsResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, CollectingSince: since,
		Live: a.Live, VenueCount: a.VenueCount, Windows: wins,
		Note: "Binance USD-M and Bybit linear perpetual liquidations. complete counts only time the websocket was actually live for that coin and venue. A dropped or never-connected stream does not grow coverage. Notional is quote (USDT). Informational only.",
	}
}

type openInterestLevelDTO struct {
	Contracts string `json:"contracts"`
	Value     string `json:"value"`
	Time      string `json:"time,omitempty"`
}

type openInterestWindowDTO struct {
	Window            string `json:"window"`
	OpenInterest      string `json:"openInterest"`
	OpenInterestValue string `json:"openInterestValue"`
	Change            string `json:"change"`
	ChangePct         string `json:"changePct"`
	ChangeValue       string `json:"changeValue"`
	ChangeValuePct    string `json:"changeValuePct"`
	Direction         string `json:"direction"`
	Complete          bool   `json:"complete"`
	SampleTime        string `json:"sampleTime,omitempty"`
}

type openInterestVenueDTO struct {
	Exchange string                  `json:"exchange"`
	Current  openInterestLevelDTO    `json:"current"`
	Windows  []openInterestWindowDTO `json:"windows"`
}

type openInterestResponse struct {
	Symbol     string                  `json:"symbol"`
	Exchange   string                  `json:"exchange"`
	Unit       string                  `json:"unit"`
	Current    openInterestLevelDTO    `json:"current"`
	Windows    []openInterestWindowDTO `json:"windows"`
	Venues     []openInterestVenueDTO  `json:"venues"`
	Funding    *fundingRateResponse    `json:"funding,omitempty"`
	LongShort  *longShortResponse      `json:"longShort,omitempty"`
	AsOf       string                  `json:"asOf,omitempty"`
	VenueCount int                     `json:"venueCount"`
	Note       string                  `json:"note"`
}

type fundingPrintDTO struct {
	Time      string `json:"time,omitempty"`
	Rate      string `json:"rate"`
	RatePct   string `json:"ratePct"`
	Payer     string `json:"payer"`
	MarkPrice string `json:"markPrice,omitempty"`
	Predicted bool   `json:"predicted,omitempty"`
}

type fundingCurrentDTO struct {
	Rate            string `json:"rate"`
	RatePct         string `json:"ratePct"`
	Payer           string `json:"payer"`
	NextFundingTime string `json:"nextFundingTime,omitempty"`
	IntervalHours   int    `json:"intervalHours"`
	Time            string `json:"time,omitempty"`
}

type fundingVenueDTO struct {
	Exchange    string            `json:"exchange"`
	Current     fundingCurrentDTO `json:"current"`
	LastSettled *fundingPrintDTO  `json:"lastSettled,omitempty"`
	AvgLast3    string            `json:"avgLast3,omitempty"`
	AvgLast3Pct string            `json:"avgLast3Pct,omitempty"`
	History     []fundingPrintDTO `json:"history"`
}

type fundingRateResponse struct {
	Symbol     string             `json:"symbol"`
	Exchange   string             `json:"exchange"`
	Current    *fundingCurrentDTO `json:"current,omitempty"`
	Venues     []fundingVenueDTO  `json:"venues"`
	History    []fundingPrintDTO  `json:"history,omitempty"`
	AsOf       string             `json:"asOf,omitempty"`
	VenueCount int                `json:"venueCount"`
	Note       string             `json:"note"`
}

type longShortLevelDTO struct {
	Time      string `json:"time,omitempty"`
	LongPct   string `json:"longPct"`
	ShortPct  string `json:"shortPct"`
	Ratio     string `json:"ratio"`
	Bias      string `json:"bias"`
	LongShare string `json:"longShare"`
}

type longShortVenueDTO struct {
	Exchange string              `json:"exchange"`
	Kind     string              `json:"kind"`
	Period   string              `json:"period"`
	Current  longShortLevelDTO   `json:"current"`
	Change   string              `json:"change,omitempty"`
	History  []longShortLevelDTO `json:"history"`
}

type longShortResponse struct {
	Symbol     string              `json:"symbol"`
	Exchange   string              `json:"exchange"`
	Kind       string              `json:"kind"`
	Period     string              `json:"period"`
	Current    *longShortLevelDTO  `json:"current,omitempty"`
	Venues     []longShortVenueDTO `json:"venues"`
	History    []longShortLevelDTO `json:"history,omitempty"`
	AsOf       string              `json:"asOf,omitempty"`
	VenueCount int                 `json:"venueCount"`
	Note       string              `json:"note"`
}

func oiLevelDTO(l domain.OpenInterestLevel) openInterestLevelDTO {
	t := ""
	if !l.Time.IsZero() {
		t = l.Time.UTC().Format(time.RFC3339Nano)
	}
	return openInterestLevelDTO{Contracts: l.Contracts, Value: l.Value, Time: t}
}

func oiWindowsDTO(in []domain.OpenInterestWindow) []openInterestWindowDTO {
	out := make([]openInterestWindowDTO, 0, len(in))
	for _, w := range in {
		row := openInterestWindowDTO{
			Window: w.Window, OpenInterest: w.OpenInterest, OpenInterestValue: w.OpenInterestValue,
			Change: w.Change, ChangePct: w.ChangePct, ChangeValue: w.ChangeValue, ChangeValuePct: w.ChangeValuePct,
			Direction: w.Direction, Complete: w.Complete,
		}
		if !w.SampleTime.IsZero() {
			row.SampleTime = w.SampleTime.UTC().Format(time.RFC3339Nano)
		}
		out = append(out, row)
	}
	return out
}

func openInterestToDTO(a *domain.OpenInterestSnapshot) openInterestResponse {
	if a == nil {
		return openInterestResponse{}
	}
	venues := make([]openInterestVenueDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		venues = append(venues, openInterestVenueDTO{
			Exchange: v.Exchange, Current: oiLevelDTO(v.Current), Windows: oiWindowsDTO(v.Windows),
		})
	}
	asOf := ""
	if !a.AsOf.IsZero() {
		asOf = a.AsOf.UTC().Format(time.RFC3339Nano)
	}
	out := openInterestResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, Unit: a.Unit,
		Current: oiLevelDTO(a.Current), Windows: oiWindowsDTO(a.Windows),
		Venues: venues, AsOf: asOf, VenueCount: a.VenueCount,
		Note: "Binance USD-M and Bybit linear perpetual open interest. contracts is outstanding size in the base asset (Bybit uses singleOpenInterest — one side — not the 2× both-sides openInterest). value is USDT notional. Change is current minus the reading ~window ago. funding is the predicted next rate plus recent settlements. longShort is the share of accounts that are long vs short. Combined all sums OI across venues; funding and long/short stay per venue. Informational only.",
	}
	if a.Funding != nil {
		f := fundingToDTO(a.Funding)
		out.Funding = &f
	}
	if a.LongShort != nil {
		ls := longShortToDTO(a.LongShort)
		out.LongShort = &ls
	}
	return out
}

func fundingPrintDTOFrom(p domain.FundingPrint) fundingPrintDTO {
	t := ""
	if !p.Time.IsZero() {
		t = p.Time.UTC().Format(time.RFC3339Nano)
	}
	return fundingPrintDTO{
		Time: t, Rate: p.Rate, RatePct: p.RatePct, Payer: p.Payer,
		MarkPrice: p.MarkPrice, Predicted: p.Predicted,
	}
}

func fundingCurrentDTOFrom(c domain.FundingCurrent) fundingCurrentDTO {
	out := fundingCurrentDTO{
		Rate: c.Rate, RatePct: c.RatePct, Payer: c.Payer, IntervalHours: c.IntervalHours,
	}
	if !c.NextFundingTime.IsZero() {
		out.NextFundingTime = c.NextFundingTime.UTC().Format(time.RFC3339Nano)
	}
	if !c.Time.IsZero() {
		out.Time = c.Time.UTC().Format(time.RFC3339Nano)
	}
	return out
}

func fundingToDTO(a *domain.FundingSnapshot) fundingRateResponse {
	if a == nil {
		return fundingRateResponse{}
	}
	venues := make([]fundingVenueDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		row := fundingVenueDTO{
			Exchange: v.Exchange, Current: fundingCurrentDTOFrom(v.Current),
			AvgLast3: v.AvgLast3, AvgLast3Pct: v.AvgLast3Pct,
			History: make([]fundingPrintDTO, 0, len(v.History)),
		}
		if v.LastSettled != nil {
			p := fundingPrintDTOFrom(*v.LastSettled)
			row.LastSettled = &p
		}
		for _, h := range v.History {
			row.History = append(row.History, fundingPrintDTOFrom(h))
		}
		venues = append(venues, row)
	}
	hist := make([]fundingPrintDTO, 0, len(a.History))
	for _, h := range a.History {
		hist = append(hist, fundingPrintDTOFrom(h))
	}
	asOf := ""
	if !a.AsOf.IsZero() {
		asOf = a.AsOf.UTC().Format(time.RFC3339Nano)
	}
	out := fundingRateResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, Venues: venues, History: hist,
		AsOf: asOf, VenueCount: a.VenueCount,
		Note: "Binance USD-M and Bybit linear perpetual funding. rate is decimal (0.0001 = 0.01%). ratePct is percent. payer is who pays at settlement (positive rate → longs pay shorts). current is the predicted next payment, not yet settled. history is recent settlements, newest first. Combined all does not average venues. Informational only.",
	}
	if a.Current != nil {
		cur := fundingCurrentDTOFrom(*a.Current)
		out.Current = &cur
	}
	return out
}

func longShortLevelDTOFrom(p domain.LongShortLevel) longShortLevelDTO {
	t := ""
	if !p.Time.IsZero() {
		t = p.Time.UTC().Format(time.RFC3339Nano)
	}
	return longShortLevelDTO{
		Time: t, LongPct: p.LongPct, ShortPct: p.ShortPct,
		Ratio: p.Ratio, Bias: p.Bias, LongShare: p.LongShare,
	}
}

func longShortToDTO(a *domain.LongShortSnapshot) longShortResponse {
	if a == nil {
		return longShortResponse{}
	}
	venues := make([]longShortVenueDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		hist := make([]longShortLevelDTO, 0, len(v.History))
		for _, h := range v.History {
			hist = append(hist, longShortLevelDTOFrom(h))
		}
		venues = append(venues, longShortVenueDTO{
			Exchange: v.Exchange, Kind: v.Kind, Period: v.Period,
			Current: longShortLevelDTOFrom(v.Current), Change: v.Change, History: hist,
		})
	}
	hist := make([]longShortLevelDTO, 0, len(a.History))
	for _, h := range a.History {
		hist = append(hist, longShortLevelDTOFrom(h))
	}
	asOf := ""
	if !a.AsOf.IsZero() {
		asOf = a.AsOf.UTC().Format(time.RFC3339Nano)
	}
	out := longShortResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, Kind: a.Kind, Period: a.Period,
		Venues: venues, History: hist, AsOf: asOf, VenueCount: a.VenueCount,
		Note: "Share of accounts that are long vs short (not position size). ratio is long/short. bias is long if ratio≥1.05, short if ≤0.95. 5m samples. Combined all does not average venues. Informational only.",
	}
	if a.Current != nil {
		cur := longShortLevelDTOFrom(*a.Current)
		out.Current = &cur
	}
	return out
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
			Behavior: w.Behavior, AgeSeconds: w.AgeSeconds, PresentForSeconds: w.PresentForSeconds,
			VisibleSeconds: w.VisibleSeconds, AppearCount: w.AppearCount,
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
