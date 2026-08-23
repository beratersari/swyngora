package handler

import (
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetAbsorption handles GET /api/v1/market/absorption.
func (h *MarketHandler) GetAbsorption(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.GetAbsorption(r.Context(), q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, absorptionToDTO(got))
}

type absorptionPointDTO struct {
	Time           time.Time `json:"time"`
	Price          string    `json:"price"`
	PriceChangePct string    `json:"priceChangePct"`
	BuyNotional    string    `json:"buyNotional"`
	SellNotional   string    `json:"sellNotional"`
	Delta          string    `json:"delta"`
	Kind           string    `json:"kind,omitempty"`
	Absorber       string    `json:"absorber,omitempty"`
	Result         string    `json:"result,omitempty"`
	Score          int       `json:"score"`
	Grade          string    `json:"grade,omitempty"`
}

type absorptionWindowDTO struct {
	Window         string `json:"window"`
	BuyNotional    string `json:"buyNotional"`
	SellNotional   string `json:"sellNotional"`
	Delta          string `json:"delta"`
	Volume         string `json:"volume"`
	PriceFrom      string `json:"priceFrom"`
	PriceTo        string `json:"priceTo"`
	PriceChangePct string `json:"priceChangePct"`
	Kind           string `json:"kind,omitempty"`
	Absorber       string `json:"absorber,omitempty"`
	Result         string `json:"result,omitempty"`
	Score          int    `json:"score"`
	Grade          string `json:"grade,omitempty"`
	Title          string `json:"title"`
	Summary        string `json:"summary"`
	Complete       bool   `json:"complete"`
}

type absorptionEpisodeDTO struct {
	Kind            string    `json:"kind"`
	Absorber        string    `json:"absorber"`
	Result          string    `json:"result"`
	Score           int       `json:"score"`
	Grade           string    `json:"grade"`
	Bars            int       `json:"bars"`
	Since           time.Time `json:"since"`
	Until           time.Time `json:"until"`
	Duration        string    `json:"duration"`
	DurationSeconds int       `json:"durationSeconds"`
	BuyNotional     string    `json:"buyNotional"`
	SellNotional    string    `json:"sellNotional"`
	Delta           string    `json:"delta"`
	PriceFrom       string    `json:"priceFrom"`
	PriceTo         string    `json:"priceTo"`
	PriceChangePct  string    `json:"priceChangePct"`
	Active          bool      `json:"active"`
	Title           string    `json:"title"`
	Summary         string    `json:"summary"`
}

type absorptionVenueDTO struct {
	Exchange    string                 `json:"exchange"`
	Symbol      string                 `json:"symbol"`
	Market      string                 `json:"market,omitempty"`
	Points      []absorptionPointDTO   `json:"points"`
	Windows     []absorptionWindowDTO  `json:"windows"`
	Current     *absorptionEpisodeDTO  `json:"current,omitempty"`
	Episodes    []absorptionEpisodeDTO `json:"episodes,omitempty"`
	LastPrice   string                 `json:"lastPrice"`
	OverlapFrom *time.Time             `json:"overlapFrom,omitempty"`
	OverlapTo   *time.Time             `json:"overlapTo,omitempty"`
	Summary     string                 `json:"summary"`
	Error       string                 `json:"error,omitempty"`
	Complete    bool                   `json:"complete"`
}

type absorptionResponse struct {
	Symbol       string               `json:"symbol"`
	Exchange     string               `json:"exchange"`
	AsOf         time.Time            `json:"asOf"`
	Venues       []absorptionVenueDTO `json:"venues"`
	Combined     *absorptionVenueDTO  `json:"combined,omitempty"`
	SpotVenues   []absorptionVenueDTO `json:"spotVenues,omitempty"`
	SpotCombined *absorptionVenueDTO  `json:"spotCombined,omitempty"`
	Summary      string               `json:"summary"`
	Note         string               `json:"note"`
}

func absorptionToDTO(a *domain.AbsorptionReport) absorptionResponse {
	if a == nil {
		return absorptionResponse{}
	}
	venues := make([]absorptionVenueDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		venues = append(venues, absorptionVenueToDTO(v))
	}
	out := absorptionResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, AsOf: a.AsOf.UTC(),
		Venues: venues, Summary: a.Summary, Note: a.Note,
	}
	if a.Combined != nil {
		c := absorptionVenueToDTO(*a.Combined)
		out.Combined = &c
	}
	if len(a.SpotVenues) > 0 {
		sv := make([]absorptionVenueDTO, 0, len(a.SpotVenues))
		for _, v := range a.SpotVenues {
			sv = append(sv, absorptionVenueToDTO(v))
		}
		out.SpotVenues = sv
	}
	if a.SpotCombined != nil {
		c := absorptionVenueToDTO(*a.SpotCombined)
		out.SpotCombined = &c
	}
	return out
}

func absorptionVenueToDTO(v domain.AbsorptionVenue) absorptionVenueDTO {
	pts := make([]absorptionPointDTO, 0, len(v.Points))
	for _, p := range v.Points {
		pts = append(pts, absorptionPointDTO{
			Time: p.Time.UTC(), Price: formatHistQty(p.Price),
			PriceChangePct: domain.FormatSignedPct(p.PriceChangePct),
			BuyNotional:    formatHistQty(p.BuyNotional), SellNotional: formatHistQty(p.SellNotional),
			Delta: domain.FormatSignedQty(p.Delta), Kind: p.Kind, Absorber: p.Absorber,
			Result: p.Result, Score: p.Score, Grade: p.Grade,
		})
	}
	wins := make([]absorptionWindowDTO, 0, len(v.Windows))
	for _, w := range v.Windows {
		wins = append(wins, absorptionWindowDTO{
			Window: w.Window, BuyNotional: formatHistQty(w.BuyNotional),
			SellNotional: formatHistQty(w.SellNotional), Delta: domain.FormatSignedQty(w.Delta),
			Volume: formatHistQty(w.Volume), PriceFrom: formatHistQty(w.PriceFrom),
			PriceTo: formatHistQty(w.PriceTo), PriceChangePct: domain.FormatSignedPct(w.PriceChangePct),
			Kind: w.Kind, Absorber: w.Absorber, Result: w.Result, Score: w.Score, Grade: w.Grade,
			Title: w.Title, Summary: w.Summary, Complete: w.Complete,
		})
	}
	eps := make([]absorptionEpisodeDTO, 0, len(v.Episodes))
	for _, e := range v.Episodes {
		eps = append(eps, absorptionEpisodeToDTO(e))
	}
	out := absorptionVenueDTO{
		Exchange: string(v.Exchange), Symbol: v.Symbol, Market: v.Market,
		Points: pts, Windows: wins, Episodes: eps,
		LastPrice: formatHistQty(v.LastPrice), Summary: v.Summary, Error: v.Error, Complete: v.Complete,
	}
	if v.Current != nil {
		c := absorptionEpisodeToDTO(*v.Current)
		out.Current = &c
	}
	if v.OverlapFrom != nil && !v.OverlapFrom.IsZero() {
		t := v.OverlapFrom.UTC()
		out.OverlapFrom = &t
	}
	if v.OverlapTo != nil && !v.OverlapTo.IsZero() {
		t := v.OverlapTo.UTC()
		out.OverlapTo = &t
	}
	return out
}

func absorptionEpisodeToDTO(e domain.AbsorptionEpisode) absorptionEpisodeDTO {
	return absorptionEpisodeDTO{
		Kind: e.Kind, Absorber: e.Absorber, Result: e.Result, Score: e.Score, Grade: e.Grade,
		Bars: e.Bars, Since: e.Since.UTC(), Until: e.Until.UTC(), Duration: e.Duration,
		DurationSeconds: e.DurationSeconds, BuyNotional: formatHistQty(e.BuyNotional),
		SellNotional: formatHistQty(e.SellNotional), Delta: domain.FormatSignedQty(e.Delta),
		PriceFrom: formatHistQty(e.PriceFrom), PriceTo: formatHistQty(e.PriceTo),
		PriceChangePct: domain.FormatSignedPct(e.PriceChangePct), Active: e.Active,
		Title: e.Title, Summary: e.Summary,
	}
}
