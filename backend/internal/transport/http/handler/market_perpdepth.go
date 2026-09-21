package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetPerpDepth handles GET /api/v1/market/perp-depth.
func (h *MarketHandler) GetPerpDepth(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var rangePct float64
	if raw := q.Get("rangePct"); raw != "" {
		n, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			writeError(w, fmt.Errorf("%w: rangePct must be a number", domain.ErrInvalidArgument))
			return
		}
		rangePct = n
	}
	got, err := h.svc.GetPerpDepth(r.Context(), q.Get("exchange"), q.Get("symbol"), rangePct)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, perpDepthToDTO(got))
}

type perpDepthSideDTO struct {
	Notional  string `json:"notional"`
	Quantity  string `json:"quantity"`
	Levels    int    `json:"levels"`
	EdgePrice string `json:"edgePrice,omitempty"`
}

type perpDepthBandDTO struct {
	RangePct  string           `json:"rangePct"`
	Below     perpDepthSideDTO `json:"below"`
	Above     perpDepthSideDTO `json:"above"`
	Imbalance string           `json:"imbalance"`
	Ratio     string           `json:"ratio,omitempty"`
	Lean      string           `json:"lean"`
	Complete  bool             `json:"complete"`
}

type perpDepthVenueDTO struct {
	Exchange    string             `json:"exchange"`
	Symbol      string             `json:"symbol"`
	Mid         string             `json:"mid"`
	CoverBidPct string             `json:"coverBidPct"`
	CoverAskPct string             `json:"coverAskPct"`
	Headline    perpDepthBandDTO   `json:"headline"`
	Bands       []perpDepthBandDTO `json:"bands"`
	Summary     string             `json:"summary"`
	Error       string             `json:"error,omitempty"`
}

type perpDepthResponse struct {
	Symbol   string              `json:"symbol"`
	Exchange string              `json:"exchange"`
	RangePct string              `json:"rangePct"`
	AsOf     time.Time           `json:"asOf"`
	Venues   []perpDepthVenueDTO `json:"venues"`
	Combined *perpDepthVenueDTO  `json:"combined,omitempty"`
	Summary  string              `json:"summary"`
	Note     string              `json:"note"`
}

func perpDepthToDTO(a *domain.PerpDepthReport) perpDepthResponse {
	if a == nil {
		return perpDepthResponse{}
	}
	venues := make([]perpDepthVenueDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		venues = append(venues, perpDepthVenueToDTO(v))
	}
	out := perpDepthResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, RangePct: formatHistQty(a.RangePct),
		AsOf: a.AsOf.UTC(), Venues: venues, Summary: a.Summary, Note: a.Note,
	}
	if a.Combined != nil {
		c := perpDepthVenueToDTO(*a.Combined)
		out.Combined = &c
	}
	return out
}

func perpDepthVenueToDTO(v domain.PerpDepthVenue) perpDepthVenueDTO {
	bands := make([]perpDepthBandDTO, 0, len(v.Bands))
	for _, b := range v.Bands {
		bands = append(bands, perpDepthBandToDTO(b))
	}
	return perpDepthVenueDTO{
		Exchange: string(v.Exchange), Symbol: v.Symbol, Mid: formatHistQty(v.Mid),
		CoverBidPct: formatHistQty(v.CoverBidPct), CoverAskPct: formatHistQty(v.CoverAskPct),
		Headline: perpDepthBandToDTO(v.Headline), Bands: bands, Summary: v.Summary, Error: v.Error,
	}
}

func perpDepthBandToDTO(b domain.PerpDepthBand) perpDepthBandDTO {
	return perpDepthBandDTO{
		RangePct:  formatHistQty(b.RangePct),
		Below:     perpDepthSideToDTO(b.Below),
		Above:     perpDepthSideToDTO(b.Above),
		Imbalance: domain.FormatSignedQty(b.Imbalance),
		Ratio:     formatHistQty(b.Ratio),
		Lean:      b.Lean, Complete: b.Complete,
	}
}

func perpDepthSideToDTO(s domain.PerpDepthSide) perpDepthSideDTO {
	out := perpDepthSideDTO{
		Notional: formatHistQty(s.Notional), Quantity: formatHistQty(s.Quantity), Levels: s.Levels,
	}
	if s.EdgePrice > 0 {
		out.EdgePrice = formatHistQty(s.EdgePrice)
	}
	return out
}
