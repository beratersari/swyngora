package handler

import (
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetPositioning handles GET /api/v1/market/positioning.
func (h *MarketHandler) GetPositioning(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.GetPositioning(r.Context(), q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, positioningToDTO(got))
}

type positioningWindowDTO struct {
	Window         string  `json:"window"`
	PriceChangePct string  `json:"priceChangePct"`
	OIChangePct    string  `json:"oiChangePct"`
	PriceDir       string  `json:"priceDir"`
	OIDir          string  `json:"oiDir"`
	Regime         string  `json:"regime"`
	Label          string  `json:"label"`
	Confidence     float64 `json:"confidence"`
}

type positioningVenueDTO struct {
	Exchange          string                 `json:"exchange"`
	Symbol            string                 `json:"symbol"`
	Price             string                 `json:"price"`
	OpenInterestValue string                 `json:"openInterestValue"`
	LongPct           string                 `json:"longPct"`
	ShortPct          string                 `json:"shortPct"`
	FundingRate       string                 `json:"fundingRate"`
	FundingPayer      string                 `json:"fundingPayer"`
	Primary           positioningWindowDTO   `json:"primary"`
	Windows           []positioningWindowDTO `json:"windows"`
	Regime            string                 `json:"regime"`
	Label             string                 `json:"label"`
	Confidence        float64                `json:"confidence"`
	Reasons           []string               `json:"reasons"`
	Summary           string                 `json:"summary"`
	Error             string                 `json:"error,omitempty"`
}

type positioningCombinedDTO struct {
	Regime        string   `json:"regime"`
	Label         string   `json:"label"`
	Confidence    float64  `json:"confidence"`
	Agreement     string   `json:"agreement"`
	DominantVenue string   `json:"dominantVenue"`
	Summary       string   `json:"summary"`
	Reasons       []string `json:"reasons"`
}

type positioningResponse struct {
	Symbol   string                  `json:"symbol"`
	Exchange string                  `json:"exchange"`
	AsOf     time.Time               `json:"asOf"`
	Venues   []positioningVenueDTO   `json:"venues"`
	Combined *positioningCombinedDTO `json:"combined,omitempty"`
	Note     string                  `json:"note"`
}

func positioningToDTO(a *domain.PositioningReport) positioningResponse {
	if a == nil {
		return positioningResponse{}
	}
	venues := make([]positioningVenueDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		venues = append(venues, positioningVenueToDTO(v))
	}
	out := positioningResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, AsOf: a.AsOf.UTC(),
		Venues: venues, Note: a.Note,
	}
	if a.Combined != nil {
		c := positioningCombinedToDTO(*a.Combined)
		out.Combined = &c
	}
	return out
}

func positioningVenueToDTO(v domain.PositioningVenueReport) positioningVenueDTO {
	wins := make([]positioningWindowDTO, 0, len(v.Windows))
	for _, w := range v.Windows {
		wins = append(wins, positioningWindowToDTO(w))
	}
	reasons := v.Reasons
	if reasons == nil {
		reasons = []string{}
	}
	return positioningVenueDTO{
		Exchange: string(v.Exchange), Symbol: v.Symbol,
		Price: formatHistQty(v.Price), OpenInterestValue: formatHistQty(v.OpenInterest),
		LongPct: formatHistQty(v.LongShare * 100), ShortPct: formatHistQty(v.ShortShare * 100),
		FundingRate: formatHuntRate(v.FundingRate), FundingPayer: v.FundingPayer,
		Primary: positioningWindowToDTO(v.Primary), Windows: wins,
		Regime: v.Regime, Label: v.Label, Confidence: v.Confidence,
		Reasons: reasons, Summary: v.Summary, Error: v.Error,
	}
}

func positioningWindowToDTO(w domain.PositioningWindow) positioningWindowDTO {
	return positioningWindowDTO{
		Window:         w.Window,
		PriceChangePct: domain.FormatSignedPct(w.PriceChangePct),
		OIChangePct:    domain.FormatSignedPct(w.OIChangePct),
		PriceDir:       w.PriceDir, OIDir: w.OIDir,
		Regime: w.Regime, Label: w.Label, Confidence: w.Confidence,
	}
}

func positioningCombinedToDTO(c domain.PositioningCombined) positioningCombinedDTO {
	reasons := c.Reasons
	if reasons == nil {
		reasons = []string{}
	}
	return positioningCombinedDTO{
		Regime: c.Regime, Label: c.Label, Confidence: c.Confidence,
		Agreement: c.Agreement, DominantVenue: c.DominantVenue,
		Summary: c.Summary, Reasons: reasons,
	}
}
