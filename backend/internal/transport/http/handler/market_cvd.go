package handler

import (
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetCVD handles GET /api/v1/market/cvd.
func (h *MarketHandler) GetCVD(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.GetCVD(r.Context(), q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cvdToDTO(got))
}

type cvdPointDTO struct {
	Time         time.Time `json:"time"`
	Price        string    `json:"price"`
	BuyNotional  string    `json:"buyNotional"`
	SellNotional string    `json:"sellNotional"`
	Delta        string    `json:"delta"`
	CVD          string    `json:"cvd"`
}

type cvdWindowDTO struct {
	Window         string `json:"window"`
	CVDChange      string `json:"cvdChange"`
	CVDChangePct   string `json:"cvdChangePct"`
	PriceChangePct string `json:"priceChangePct"`
	BuyNotional    string `json:"buyNotional"`
	SellNotional   string `json:"sellNotional"`
	VsPrice        string `json:"vsPrice"`
	Title          string `json:"title"`
	Summary        string `json:"summary"`
	Complete       bool   `json:"complete"`
}

type cvdVenueDTO struct {
	Exchange  string         `json:"exchange"`
	Symbol    string         `json:"symbol"`
	Points    []cvdPointDTO  `json:"points"`
	Windows   []cvdWindowDTO `json:"windows"`
	LastCVD   string         `json:"lastCvd"`
	LastPrice string         `json:"lastPrice"`
	Summary   string         `json:"summary"`
	Error     string         `json:"error,omitempty"`
	Complete  bool           `json:"complete"`
}

type cvdResponse struct {
	Symbol   string        `json:"symbol"`
	Exchange string        `json:"exchange"`
	AsOf     time.Time     `json:"asOf"`
	Venues   []cvdVenueDTO `json:"venues"`
	Combined *cvdVenueDTO  `json:"combined,omitempty"`
	Summary  string        `json:"summary"`
	Note     string        `json:"note"`
}

func cvdToDTO(a *domain.CVDReport) cvdResponse {
	if a == nil {
		return cvdResponse{}
	}
	venues := make([]cvdVenueDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		venues = append(venues, cvdVenueToDTO(v))
	}
	out := cvdResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, AsOf: a.AsOf.UTC(),
		Venues: venues, Summary: a.Summary, Note: a.Note,
	}
	if a.Combined != nil {
		c := cvdVenueToDTO(*a.Combined)
		out.Combined = &c
	}
	return out
}

func cvdVenueToDTO(v domain.CVDVenueSeries) cvdVenueDTO {
	pts := make([]cvdPointDTO, 0, len(v.Points))
	for _, p := range v.Points {
		pts = append(pts, cvdPointDTO{
			Time: p.Time.UTC(), Price: formatHistQty(p.Price),
			BuyNotional: formatHistQty(p.BuyNotional), SellNotional: formatHistQty(p.SellNotional),
			Delta: domain.FormatSignedQty(p.Delta), CVD: domain.FormatSignedQty(p.CVD),
		})
	}
	wins := make([]cvdWindowDTO, 0, len(v.Windows))
	for _, w := range v.Windows {
		wins = append(wins, cvdWindowDTO{
			Window: w.Window, CVDChange: domain.FormatSignedQty(w.CVDChange),
			CVDChangePct:   domain.FormatSignedPct(w.CVDChangePct),
			PriceChangePct: domain.FormatSignedPct(w.PriceChangePct),
			BuyNotional:    formatHistQty(w.BuyNotional), SellNotional: formatHistQty(w.SellNotional),
			VsPrice: w.VsPrice, Title: w.Title, Summary: w.Summary, Complete: w.Complete,
		})
	}
	return cvdVenueDTO{
		Exchange: string(v.Exchange), Symbol: v.Symbol, Points: pts, Windows: wins,
		LastCVD: domain.FormatSignedQty(v.LastCVD), LastPrice: formatHistQty(v.LastPrice),
		Summary: v.Summary, Error: v.Error, Complete: v.Complete,
	}
}
