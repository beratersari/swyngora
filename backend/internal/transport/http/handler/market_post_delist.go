package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type postDelistResponse struct {
	Exchange           string      `json:"exchange"`
	Symbol             string      `json:"symbol"`
	Base               string      `json:"base,omitempty"`
	DelistTime         string      `json:"delistTime,omitempty"`
	Available          bool        `json:"available"`
	Source             string      `json:"source,omitempty"`
	SourceLabel        string      `json:"sourceLabel,omitempty"`
	Note               string      `json:"note"`
	LastPrice          string      `json:"lastPrice,omitempty"`
	PriceChangePercent string      `json:"priceChangePercent,omitempty"`
	Quote              string      `json:"quote,omitempty"`
	AsOf               string      `json:"asOf,omitempty"`
	Interval           string      `json:"interval,omitempty"`
	Candles            []candleDTO `json:"candles,omitempty"`
}

// GetPostDelist handles GET /api/v1/market/post-delist
func (h *MarketHandler) GetPostDelist(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	symbol := q.Get("symbol")
	interval := q.Get("interval")
	limit := 0
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, fmt.Errorf("%w: limit must be an integer", domain.ErrInvalidArgument))
			return
		}
		limit = n
	}
	view, err := h.svc.GetPostDelist(r.Context(), q.Get("exchange"), symbol, interval, limit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, postDelistToDTO(view))
}

func postDelistToDTO(view *domain.PostDelistView) postDelistResponse {
	if view == nil {
		return postDelistResponse{Note: "No public off-venue price was found after delist."}
	}
	out := postDelistResponse{
		Exchange:           view.Exchange,
		Symbol:             view.Symbol,
		Base:               view.Base,
		Available:          view.Available,
		Source:             view.Source,
		SourceLabel:        view.SourceLabel,
		Note:               view.Note,
		LastPrice:          view.LastPrice,
		PriceChangePercent: view.PriceChangePercent,
		Quote:              view.Quote,
		Interval:           view.Interval,
	}
	if !view.DelistTime.IsZero() {
		out.DelistTime = view.DelistTime.UTC().Format(time.RFC3339)
	}
	if !view.AsOf.IsZero() {
		out.AsOf = view.AsOf.UTC().Format(time.RFC3339)
	}
	if len(view.Candles) > 0 {
		out.Candles = make([]candleDTO, 0, len(view.Candles))
		for _, c := range view.Candles {
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
	}
	return out
}
