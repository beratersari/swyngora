package handler

import (
	"fmt"
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetOrderCount handles GET /api/v1/market/order-count.
func (h *MarketHandler) GetOrderCount(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.GetOrderCount(r.Context(), q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, orderCountToDTO(got))
}

type orderCountWindowDTO struct {
	Window    string `json:"window"`
	BuyCount  int    `json:"buyCount"`
	SellCount int    `json:"sellCount"`
	Total     int    `json:"total"`
	Delta     int    `json:"delta"`
	Ratio     string `json:"ratio"`
	Lean      string `json:"lean"`
	Complete  bool   `json:"complete"`
	Summary   string `json:"summary"`
}

type orderCountVenueDTO struct {
	Exchange string                `json:"exchange"`
	Symbol   string                `json:"symbol"`
	Windows  []orderCountWindowDTO `json:"windows"`
	Summary  string                `json:"summary"`
	Error    string                `json:"error,omitempty"`
}

type orderCountResponse struct {
	Symbol   string               `json:"symbol"`
	Exchange string               `json:"exchange"`
	AsOf     time.Time            `json:"asOf"`
	Venues   []orderCountVenueDTO `json:"venues"`
	Combined *orderCountVenueDTO  `json:"combined,omitempty"`
	Summary  string               `json:"summary"`
	Note     string               `json:"note"`
}

func orderCountToDTO(a *domain.OrderCountReport) orderCountResponse {
	if a == nil {
		return orderCountResponse{}
	}
	venues := make([]orderCountVenueDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		venues = append(venues, orderCountVenueToDTO(v))
	}
	out := orderCountResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, AsOf: a.AsOf.UTC(),
		Venues: venues, Summary: a.Summary, Note: a.Note,
	}
	if a.Combined != nil {
		c := orderCountVenueToDTO(*a.Combined)
		out.Combined = &c
	}
	return out
}

func orderCountVenueToDTO(v domain.OrderCountVenue) orderCountVenueDTO {
	wins := make([]orderCountWindowDTO, 0, len(v.Windows))
	for _, w := range v.Windows {
		wins = append(wins, orderCountWindowDTO{
			Window: w.Window, BuyCount: w.BuyCount, SellCount: w.SellCount,
			Total: w.Total, Delta: w.Delta, Ratio: fmt.Sprintf("%+.2f", w.Ratio),
			Lean: w.Lean, Complete: w.Complete, Summary: w.Summary,
		})
	}
	return orderCountVenueDTO{
		Exchange: string(v.Exchange), Symbol: v.Symbol, Windows: wins,
		Summary: v.Summary, Error: v.Error,
	}
}
