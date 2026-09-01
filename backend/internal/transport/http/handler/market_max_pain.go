package handler

import (
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetLiquidationMaxPain handles GET /api/v1/market/liquidation-max-pain.
func (h *MarketHandler) GetLiquidationMaxPain(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.GetLiquidationMaxPain(r.Context(), q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, maxPainToDTO(got))
}

type maxPainPocketDTO struct {
	Exchange         string `json:"exchange,omitempty"`
	Side             string `json:"side"`
	Direction        string `json:"direction"`
	Price            string `json:"price"`
	MovePct          string `json:"movePct"`
	Notional         string `json:"notional"`
	EstNotional      string `json:"estNotional"`
	ObservedNotional string `json:"observedNotional,omitempty"`
	Leverage         string `json:"leverage,omitempty"`
	Source           string `json:"source"`
}

type maxPainVenueDTO struct {
	Exchange          string             `json:"exchange"`
	Symbol            string             `json:"symbol"`
	Price             string             `json:"price"`
	OpenInterestValue string             `json:"openInterestValue"`
	Above             *maxPainPocketDTO  `json:"above,omitempty"`
	Below             *maxPainPocketDTO  `json:"below,omitempty"`
	AboveLevels       []maxPainPocketDTO `json:"aboveLevels"`
	BelowLevels       []maxPainPocketDTO `json:"belowLevels"`
	Error             string             `json:"error,omitempty"`
}

type maxPainResponse struct {
	Symbol      string             `json:"symbol"`
	Exchange    string             `json:"exchange"`
	AsOf        time.Time          `json:"asOf"`
	Above       *maxPainPocketDTO  `json:"above,omitempty"`
	Below       *maxPainPocketDTO  `json:"below,omitempty"`
	Venues      []maxPainVenueDTO  `json:"venues"`
	Summary     string             `json:"summary"`
	Assumptions huntAssumptionsDTO `json:"assumptions"`
	Note        string             `json:"note"`
}

func maxPainToDTO(a *domain.MaxPainReport) maxPainResponse {
	if a == nil {
		return maxPainResponse{Venues: []maxPainVenueDTO{}}
	}
	venues := make([]maxPainVenueDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		venues = append(venues, maxPainVenueToDTO(v))
	}
	return maxPainResponse{
		Symbol:   a.Symbol,
		Exchange: a.Exchange,
		AsOf:     a.AsOf.UTC(),
		Above:    maxPainPocketToDTO(a.Above),
		Below:    maxPainPocketToDTO(a.Below),
		Venues:   venues,
		Summary:  a.Summary,
		Assumptions: huntAssumptionsDTO{
			MaintenanceMargin:   formatHistQty(a.Assumptions.MaintenanceMargin),
			AccountBlend:        a.Assumptions.AccountBlend,
			LiquidationTakeRate: formatHistQty(a.Assumptions.LiquidationTakeRate),
			SpotTakerFee:        formatHistQty(a.Assumptions.SpotTakerFee),
			CascadeFillRate:     a.Assumptions.CascadeFillRate,
			LongShortIsAccounts: true,
		},
		Note: a.Note,
	}
}

func maxPainVenueToDTO(v domain.MaxPainVenue) maxPainVenueDTO {
	return maxPainVenueDTO{
		Exchange:          string(v.Exchange),
		Symbol:            v.Symbol,
		Price:             formatHistQty(v.Price),
		OpenInterestValue: formatHistQty(v.OpenInterestValue),
		Above:             maxPainPocketToDTO(v.Above),
		Below:             maxPainPocketToDTO(v.Below),
		AboveLevels:       maxPainPocketsToDTO(v.AboveLevels),
		BelowLevels:       maxPainPocketsToDTO(v.BelowLevels),
		Error:             v.Error,
	}
}

func maxPainPocketsToDTO(in []domain.MaxPainPocket) []maxPainPocketDTO {
	out := make([]maxPainPocketDTO, 0, len(in))
	for i := range in {
		if p := maxPainPocketToDTO(&in[i]); p != nil {
			out = append(out, *p)
		}
	}
	return out
}

func maxPainPocketToDTO(p *domain.MaxPainPocket) *maxPainPocketDTO {
	if p == nil {
		return nil
	}
	out := maxPainPocketDTO{
		Exchange:    p.Exchange,
		Side:        p.Side,
		Direction:   p.Direction,
		Price:       formatHistQty(p.Price),
		MovePct:     domain.FormatSignedPct(p.MovePct),
		Notional:    formatHistQty(p.Notional),
		EstNotional: formatHistQty(p.EstNotional),
		Source:      p.Source,
	}
	if p.ObservedNotional > 0 {
		out.ObservedNotional = formatHistQty(p.ObservedNotional)
	}
	if p.Leverage > 0 {
		out.Leverage = formatHistQty(p.Leverage)
	}
	return &out
}
