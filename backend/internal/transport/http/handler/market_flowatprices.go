package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetFlowAtPrices handles GET /api/v1/market/flow-at-prices.
func (h *MarketHandler) GetFlowAtPrices(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
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
	tick, err := optionalPositiveFloat(q.Get("tickSize"), "tickSize")
	if err != nil {
		writeError(w, err)
		return
	}
	minPx, err := optionalPositiveFloat(q.Get("minPrice"), "minPrice")
	if err != nil {
		writeError(w, err)
		return
	}
	maxPx, err := optionalPositiveFloat(q.Get("maxPrice"), "maxPrice")
	if err != nil {
		writeError(w, err)
		return
	}
	got, err := h.svc.GetFlowAtPrices(r.Context(), q.Get("exchange"), q.Get("symbol"), q.Get("window"), startPtr, endPtr, tick, minPx, maxPx)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, flowAtPricesToDTO(got))
}

func optionalPositiveFloat(raw, name string) (float64, error) {
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.ParseFloat(raw, 64)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("%w: %s must be a positive number", domain.ErrInvalidArgument, name)
	}
	return n, nil
}

type flowAtPricesLevelDTO struct {
	Price      string `json:"price"`
	High       string `json:"high"`
	BuyVolume  string `json:"buyVolume"`
	SellVolume string `json:"sellVolume"`
	Volume     string `json:"volume"`
	BuyCount   int    `json:"buyCount"`
	SellCount  int    `json:"sellCount"`
	Trades     int    `json:"trades"`
	Delta      string `json:"delta"`
	BuyPct     string `json:"buyPct"`
	SharePct   string `json:"sharePct"`
	Dominant   string `json:"dominant"`
	IsPoc      bool   `json:"isPoc"`
}

type flowAtPricesPOCDTO struct {
	Price      string `json:"price"`
	High       string `json:"high"`
	BuyVolume  string `json:"buyVolume"`
	SellVolume string `json:"sellVolume"`
	Volume     string `json:"volume"`
	Delta      string `json:"delta"`
	Trades     int    `json:"trades"`
	SharePct   string `json:"sharePct"`
}

type flowAtPricesVenueDTO struct {
	Exchange   string                 `json:"exchange"`
	Symbol     string                 `json:"symbol"`
	From       time.Time              `json:"from"`
	To         time.Time              `json:"to"`
	TickSize   string                 `json:"tickSize"`
	LastPrice  string                 `json:"lastPrice"`
	High       string                 `json:"high"`
	Low        string                 `json:"low"`
	BuyVolume  string                 `json:"buyVolume"`
	SellVolume string                 `json:"sellVolume"`
	Volume     string                 `json:"volume"`
	Delta      string                 `json:"delta"`
	BuyCount   int                    `json:"buyCount"`
	SellCount  int                    `json:"sellCount"`
	Trades     int                    `json:"trades"`
	Dominant   string                 `json:"dominant"`
	POC        flowAtPricesPOCDTO     `json:"poc"`
	Levels     []flowAtPricesLevelDTO `json:"levels"`
	Complete   bool                   `json:"complete"`
	Truncated  bool                   `json:"truncated"`
	KnownFrom  *time.Time             `json:"knownFrom,omitempty"`
	KnownTo    *time.Time             `json:"knownTo,omitempty"`
	Summary    string                 `json:"summary"`
	Error      string                 `json:"error,omitempty"`
}

type flowAtPricesResponse struct {
	Symbol   string                 `json:"symbol"`
	Exchange string                 `json:"exchange"`
	Window   string                 `json:"window"`
	From     time.Time              `json:"from"`
	To       time.Time              `json:"to"`
	AsOf     time.Time              `json:"asOf"`
	MinPrice string                 `json:"minPrice,omitempty"`
	MaxPrice string                 `json:"maxPrice,omitempty"`
	TickSize string                 `json:"tickSize"`
	Venues   []flowAtPricesVenueDTO `json:"venues"`
	Combined *flowAtPricesVenueDTO  `json:"combined,omitempty"`
	Summary  string                 `json:"summary"`
	Note     string                 `json:"note"`
}

func flowAtPricesToDTO(a *domain.FlowAtPricesReport) flowAtPricesResponse {
	if a == nil {
		return flowAtPricesResponse{}
	}
	venues := make([]flowAtPricesVenueDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		venues = append(venues, flowAtPricesVenueToDTO(v))
	}
	out := flowAtPricesResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, Window: a.Window,
		From: a.From.UTC(), To: a.To.UTC(), AsOf: a.AsOf.UTC(),
		TickSize: formatHistQty(a.TickSize), Venues: venues, Summary: a.Summary, Note: a.Note,
	}
	if a.MinPrice > 0 {
		out.MinPrice = formatHistQty(a.MinPrice)
	}
	if a.MaxPrice > 0 {
		out.MaxPrice = formatHistQty(a.MaxPrice)
	}
	if a.Combined != nil {
		c := flowAtPricesVenueToDTO(*a.Combined)
		out.Combined = &c
	}
	return out
}

func flowAtPricesVenueToDTO(v domain.FlowAtPricesVenue) flowAtPricesVenueDTO {
	levels := make([]flowAtPricesLevelDTO, 0, len(v.Levels))
	for _, lv := range v.Levels {
		levels = append(levels, flowAtPricesLevelDTO{
			Price: formatHistQty(lv.Price), High: formatHistQty(lv.High),
			BuyVolume: formatHistQty(lv.BuyVolume), SellVolume: formatHistQty(lv.SellVolume),
			Volume: formatHistQty(lv.Volume), BuyCount: lv.BuyCount, SellCount: lv.SellCount,
			Trades: lv.Trades, Delta: domain.FormatSignedQty(lv.Delta),
			BuyPct: formatHistQty(lv.BuyPct), SharePct: formatHistQty(lv.SharePct),
			Dominant: lv.Dominant, IsPoc: lv.IsPoc,
		})
	}
	out := flowAtPricesVenueDTO{
		Exchange: string(v.Exchange), Symbol: v.Symbol,
		From: v.From.UTC(), To: v.To.UTC(), TickSize: formatHistQty(v.TickSize),
		LastPrice: formatHistQty(v.LastPrice), High: formatHistQty(v.High), Low: formatHistQty(v.Low),
		BuyVolume: formatHistQty(v.BuyVolume), SellVolume: formatHistQty(v.SellVolume),
		Volume: formatHistQty(v.Volume), Delta: domain.FormatSignedQty(v.Delta),
		BuyCount: v.BuyCount, SellCount: v.SellCount, Trades: v.Trades, Dominant: v.Dominant,
		POC: flowAtPricesPOCDTO{
			Price: formatHistQty(v.POC.Price), High: formatHistQty(v.POC.High),
			BuyVolume: formatHistQty(v.POC.BuyVolume), SellVolume: formatHistQty(v.POC.SellVolume),
			Volume: formatHistQty(v.POC.Volume), Delta: domain.FormatSignedQty(v.POC.Delta),
			Trades: v.POC.Trades, SharePct: formatHistQty(v.POC.SharePct),
		},
		Levels: levels, Complete: v.Complete, Truncated: v.Truncated, Summary: v.Summary, Error: v.Error,
	}
	if !v.KnownFrom.IsZero() {
		t := v.KnownFrom.UTC()
		out.KnownFrom = &t
	}
	if !v.KnownTo.IsZero() {
		t := v.KnownTo.UTC()
		out.KnownTo = &t
	}
	return out
}
