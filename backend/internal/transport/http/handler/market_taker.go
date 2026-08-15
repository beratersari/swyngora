package handler

import (
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetTakerFlow handles GET /api/v1/market/taker-flow.
func (h *MarketHandler) GetTakerFlow(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.GetTakerFlow(r.Context(), q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, takerToDTO(got))
}

type takerWindowDTO struct {
	Window       string `json:"window"`
	BuyNotional  string `json:"buyNotional"`
	SellNotional string `json:"sellNotional"`
	Delta        string `json:"delta"`
	DeltaPct     string `json:"deltaPct"`
	BuyShare     string `json:"buyShare"`
	Dominant     string `json:"dominant"`
	Complete     bool   `json:"complete"`
}

type takerVenueDTO struct {
	Exchange string           `json:"exchange"`
	Symbol   string           `json:"symbol"`
	Price    string           `json:"price,omitempty"`
	Windows  []takerWindowDTO `json:"windows"`
	Dominant string           `json:"dominant"`
	Summary  string           `json:"summary"`
	Error    string           `json:"error,omitempty"`
}

type takerFlowResponse struct {
	Symbol   string          `json:"symbol"`
	Exchange string          `json:"exchange"`
	AsOf     time.Time       `json:"asOf"`
	Venues   []takerVenueDTO `json:"venues"`
	Combined *takerVenueDTO  `json:"combined,omitempty"`
	Note     string          `json:"note"`
}

func takerToDTO(a *domain.TakerFlowReport) takerFlowResponse {
	if a == nil {
		return takerFlowResponse{}
	}
	venues := make([]takerVenueDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		venues = append(venues, takerVenueToDTO(v))
	}
	out := takerFlowResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, AsOf: a.AsOf.UTC(),
		Venues: venues, Note: a.Note,
	}
	if a.Combined != nil {
		c := takerVenueToDTO(*a.Combined)
		out.Combined = &c
	}
	return out
}

func takerVenueToDTO(v domain.TakerVenueFlow) takerVenueDTO {
	wins := make([]takerWindowDTO, 0, len(v.Windows))
	for _, w := range v.Windows {
		wins = append(wins, takerWindowDTO{
			Window:      w.Window,
			BuyNotional: formatHistQty(w.BuyNotional), SellNotional: formatHistQty(w.SellNotional),
			Delta: domain.FormatSignedQty(w.Delta), DeltaPct: domain.FormatSignedPct(w.DeltaPct),
			BuyShare: formatHistQty(w.BuyShare), Dominant: w.Dominant, Complete: w.Complete,
		})
	}
	return takerVenueDTO{
		Exchange: string(v.Exchange), Symbol: v.Symbol, Price: formatHistQty(v.Price),
		Windows: wins, Dominant: v.Dominant, Summary: v.Summary, Error: v.Error,
	}
}
