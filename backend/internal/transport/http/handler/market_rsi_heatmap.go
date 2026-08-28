package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetRSIHeatmap handles GET /api/v1/market/rsi-heatmap.
func (h *MarketHandler) GetRSIHeatmap(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var limit int
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, fmt.Errorf("%w: limit must be an integer", domain.ErrInvalidArgument))
			return
		}
		limit = n
	}
	var period int
	if raw := q.Get("period"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, fmt.Errorf("%w: period must be an integer", domain.ErrInvalidArgument))
			return
		}
		period = n
	}
	interval := q.Get("interval")
	if interval == "" {
		interval = q.Get("intervals")
	}
	got, err := h.svc.GetRSIHeatmap(r.Context(), q.Get("exchange"), q.Get("quote"), interval, q.Get("sort"), limit, period)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rsiHeatmapToDTO(got))
}

type rsiHeatmapRowDTO struct {
	Rank                 int      `json:"rank"`
	Symbol               string   `json:"symbol"`
	Base                 string   `json:"base,omitempty"`
	LastPrice            string   `json:"lastPrice,omitempty"`
	PriceChangePercent   string   `json:"priceChangePercent,omitempty"`
	QuoteVolume          string   `json:"quoteVolume,omitempty"`
	MarketCapCirculating *float64 `json:"marketCapCirculating,omitempty"`
	RSI                  *float64 `json:"rsi"`
	Zone                 string   `json:"zone,omitempty"`
	Error                string   `json:"error,omitempty"`
}

type rsiHeatmapResponse struct {
	Exchange        string             `json:"exchange"`
	Quote           string             `json:"quote"`
	Interval        string             `json:"interval"`
	Period          int                `json:"period"`
	Oversold        float64            `json:"oversold"`
	Overbought      float64            `json:"overbought"`
	Sort            string             `json:"sort"`
	AverageRSI      *float64           `json:"averageRsi"`
	OversoldCount   int                `json:"oversoldCount"`
	NeutralCount    int                `json:"neutralCount"`
	OverboughtCount int                `json:"overboughtCount"`
	AsOf            time.Time          `json:"asOf"`
	Stale           bool               `json:"stale,omitempty"`
	Items           []rsiHeatmapRowDTO `json:"items"`
	Note            string             `json:"note"`
}

func rsiHeatmapToDTO(h *domain.RSIHeatmap) rsiHeatmapResponse {
	if h == nil {
		return rsiHeatmapResponse{}
	}
	items := make([]rsiHeatmapRowDTO, 0, len(h.Items))
	for _, row := range h.Items {
		items = append(items, rsiHeatmapRowDTO{
			Rank:                 row.Rank,
			Symbol:               row.Symbol,
			Base:                 row.Base,
			LastPrice:            row.LastPrice,
			PriceChangePercent:   row.PriceChangePercent,
			QuoteVolume:          row.QuoteVolume,
			MarketCapCirculating: row.MarketCapCirculating,
			RSI:                  row.RSI,
			Zone:                 string(row.Zone),
			Error:                row.Error,
		})
	}
	return rsiHeatmapResponse{
		Exchange:        string(h.Exchange),
		Quote:           h.Quote,
		Interval:        h.Interval,
		Period:          h.Period,
		Oversold:        h.Oversold,
		Overbought:      h.Overbought,
		Sort:            h.SortBy,
		AverageRSI:      h.AverageRSI,
		OversoldCount:   h.OversoldCount,
		NeutralCount:    h.NeutralCount,
		OverboughtCount: h.OverboughtCount,
		AsOf:            h.AsOf.UTC(),
		Stale:           h.Stale,
		Items:           items,
		Note:            h.Note,
	}
}
