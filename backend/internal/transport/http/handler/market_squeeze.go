package handler

import (
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetSqueezeRisk handles GET /api/v1/market/squeeze-risk.
func (h *MarketHandler) GetSqueezeRisk(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.GetSqueezeRisk(r.Context(), q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, squeezeToDTO(got))
}

type squeezeFactorDTO struct {
	ID     string  `json:"id"`
	Label  string  `json:"label"`
	Score  float64 `json:"score"`
	Weight float64 `json:"weight"`
	Detail string  `json:"detail"`
}

type squeezeSideDTO struct {
	Side    string             `json:"side"`
	Score   float64            `json:"score"`
	Level   string             `json:"level"`
	Factors []squeezeFactorDTO `json:"factors"`
	Reasons []string           `json:"reasons"`
}

type squeezeVenueDTO struct {
	Exchange          string         `json:"exchange"`
	Symbol            string         `json:"symbol"`
	Price             string         `json:"price"`
	OpenInterestValue string         `json:"openInterestValue"`
	LongPct           string         `json:"longPct"`
	ShortPct          string         `json:"shortPct"`
	FundingRate       string         `json:"fundingRate"`
	FundingPayer      string         `json:"fundingPayer"`
	OIChange1hPct     string         `json:"oiChange1hPct,omitempty"`
	OIChange4hPct     string         `json:"oiChange4hPct,omitempty"`
	PriceChange24hPct string         `json:"priceChange24hPct,omitempty"`
	LongSqueeze       squeezeSideDTO `json:"longSqueeze"`
	ShortSqueeze      squeezeSideDTO `json:"shortSqueeze"`
	CrowdedSide       string         `json:"crowdedSide"`
	HigherRisk        string         `json:"higherRisk"`
	Summary           string         `json:"summary"`
	Error             string         `json:"error,omitempty"`
}

type squeezeCombinedDTO struct {
	LongSqueeze   squeezeSideDTO `json:"longSqueeze"`
	ShortSqueeze  squeezeSideDTO `json:"shortSqueeze"`
	CrowdedSide   string         `json:"crowdedSide"`
	HigherRisk    string         `json:"higherRisk"`
	DominantVenue string         `json:"dominantVenue"`
	Summary       string         `json:"summary"`
}

type squeezeResponse struct {
	Symbol   string              `json:"symbol"`
	Exchange string              `json:"exchange"`
	AsOf     time.Time           `json:"asOf"`
	Venues   []squeezeVenueDTO   `json:"venues"`
	Combined *squeezeCombinedDTO `json:"combined,omitempty"`
	Note     string              `json:"note"`
}

func squeezeToDTO(a *domain.SqueezeReport) squeezeResponse {
	if a == nil {
		return squeezeResponse{}
	}
	venues := make([]squeezeVenueDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		venues = append(venues, squeezeVenueToDTO(v))
	}
	out := squeezeResponse{
		Symbol:   a.Symbol,
		Exchange: a.Exchange,
		AsOf:     a.AsOf.UTC(),
		Venues:   venues,
		Note:     a.Note,
	}
	if a.Combined != nil {
		c := squeezeCombinedToDTO(*a.Combined)
		out.Combined = &c
	}
	return out
}

func squeezeVenueToDTO(v domain.SqueezeVenueReport) squeezeVenueDTO {
	out := squeezeVenueDTO{
		Exchange:          string(v.Exchange),
		Symbol:            v.Symbol,
		Price:             formatHistQty(v.Price),
		OpenInterestValue: formatHistQty(v.OpenInterest),
		LongPct:           formatHistQty(v.LongShare * 100),
		ShortPct:          formatHistQty(v.ShortShare * 100),
		FundingRate:       formatHuntRate(v.FundingRate),
		FundingPayer:      v.FundingPayer,
		LongSqueeze:       squeezeSideToDTO(v.LongSqueeze),
		ShortSqueeze:      squeezeSideToDTO(v.ShortSqueeze),
		CrowdedSide:       v.CrowdedSide,
		HigherRisk:        v.HigherRisk,
		Summary:           v.Summary,
		Error:             v.Error,
	}
	if !isNaN(v.OIChange1hPct) {
		out.OIChange1hPct = domain.FormatSignedPct(v.OIChange1hPct)
	}
	if !isNaN(v.OIChange4hPct) {
		out.OIChange4hPct = domain.FormatSignedPct(v.OIChange4hPct)
	}
	if !isNaN(v.PriceChange24hPct) {
		out.PriceChange24hPct = domain.FormatSignedPct(v.PriceChange24hPct)
	}
	return out
}

func squeezeCombinedToDTO(c domain.SqueezeCombined) squeezeCombinedDTO {
	return squeezeCombinedDTO{
		LongSqueeze:   squeezeSideToDTO(c.LongSqueeze),
		ShortSqueeze:  squeezeSideToDTO(c.ShortSqueeze),
		CrowdedSide:   c.CrowdedSide,
		HigherRisk:    c.HigherRisk,
		DominantVenue: c.DominantVenue,
		Summary:       c.Summary,
	}
}

func squeezeSideToDTO(s domain.SqueezeSideRisk) squeezeSideDTO {
	factors := make([]squeezeFactorDTO, 0, len(s.Factors))
	for _, f := range s.Factors {
		factors = append(factors, squeezeFactorDTO{
			ID: f.ID, Label: f.Label, Score: f.Score, Weight: f.Weight, Detail: f.Detail,
		})
	}
	reasons := s.Reasons
	if reasons == nil {
		reasons = []string{}
	}
	return squeezeSideDTO{
		Side: s.Side, Score: s.Score, Level: s.Level,
		Factors: factors, Reasons: reasons,
	}
}

func isNaN(v float64) bool {
	return v != v
}
