package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetWhales handles GET /api/v1/market/whales.
func (h *MarketHandler) GetWhales(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	minNotional := 0.0
	if raw := q.Get("minNotional"); raw != "" {
		n, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			writeError(w, fmt.Errorf("%w: minNotional must be a number", domain.ErrInvalidArgument))
			return
		}
		minNotional = n
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
	got, err := h.svc.GetWhales(r.Context(), q.Get("exchange"), q.Get("symbol"), minNotional, limit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, whalesToDTO(got))
}

type whaleEventDTO struct {
	Exchange        string    `json:"exchange"`
	Symbol          string    `json:"symbol"`
	Kind            string    `json:"kind"`
	Side            string    `json:"side"`
	Position        string    `json:"position"`
	AvgPrice        string    `json:"avgPrice"`
	Quantity        string    `json:"quantity"`
	Notional        string    `json:"notional"`
	FirstTime       time.Time `json:"firstTime"`
	LastTime        time.Time `json:"lastTime"`
	Prints          int       `json:"prints"`
	MarketCap       string    `json:"marketCap,omitempty"`
	NotionalMcapPct string    `json:"notionalMcapPct,omitempty"`
	Unusual         bool      `json:"unusual"`
}

type whalesResponse struct {
	Symbol      string          `json:"symbol,omitempty"`
	Exchange    string          `json:"exchange"`
	AsOf        time.Time       `json:"asOf"`
	MinNotional string          `json:"minNotional"`
	Events      []whaleEventDTO `json:"events"`
	Summary     string          `json:"summary"`
	Note        string          `json:"note"`
}

func whalesToDTO(a *domain.WhaleReport) whalesResponse {
	if a == nil {
		return whalesResponse{Events: []whaleEventDTO{}}
	}
	events := make([]whaleEventDTO, 0, len(a.Events))
	for _, e := range a.Events {
		row := whaleEventDTO{
			Exchange:  string(e.Exchange),
			Symbol:    e.Symbol,
			Kind:      e.Kind,
			Side:      e.Side,
			Position:  e.Position,
			AvgPrice:  formatHistQty(e.AvgPrice),
			Quantity:  formatHistQty(e.Quantity),
			Notional:  formatHistQty(e.Notional),
			FirstTime: e.FirstTime.UTC(),
			LastTime:  e.LastTime.UTC(),
			Prints:    e.Prints,
			Unusual:   e.Unusual,
		}
		if e.MarketCap > 0 {
			row.MarketCap = formatHistQty(e.MarketCap)
			row.NotionalMcapPct = domain.FormatSignedPct(e.NotionalMcapPct)
		}
		events = append(events, row)
	}
	return whalesResponse{
		Symbol:      a.Symbol,
		Exchange:    a.Exchange,
		AsOf:        a.AsOf.UTC(),
		MinNotional: formatHistQty(a.MinNotional),
		Events:      events,
		Summary:     a.Summary,
		Note:        a.Note,
	}
}
