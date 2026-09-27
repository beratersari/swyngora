package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetSpotPerpLiquidity handles GET /api/v1/market/orderbook/spot-perp.
func (h *MarketHandler) GetSpotPerpLiquidity(w http.ResponseWriter, r *http.Request) {
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
	got, err := h.svc.GetSpotPerpLiquidity(r.Context(), q.Get("symbol"), rangePct)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, spotPerpToDTO(got))
}

type spotPerpBookDTO struct {
	BidNotional  string `json:"bidNotional"`
	AskNotional  string `json:"askNotional"`
	Total        string `json:"total"`
	ThinnerSide  string `json:"thinnerSide"`
	StrongerSide string `json:"strongerSide"`
	Mid          string `json:"mid"`
	CoverPct     string `json:"coverPct"`
	VenueCount   int    `json:"venueCount"`
	Complete     bool   `json:"complete"`
}

type spotPerpBandDTO struct {
	RangePct        string `json:"rangePct"`
	SpotBid         string `json:"spotBid"`
	SpotAsk         string `json:"spotAsk"`
	PerpBid         string `json:"perpBid"`
	PerpAsk         string `json:"perpAsk"`
	ThinnerBook     string `json:"thinnerBook"`
	StrongerBook    string `json:"strongerBook"`
	SpotThinnerSide string `json:"spotThinnerSide"`
	PerpThinnerSide string `json:"perpThinnerSide"`
	Complete        bool   `json:"complete"`
}

type spotPerpVenueDTO struct {
	Exchange        string `json:"exchange"`
	SpotBid         string `json:"spotBid"`
	SpotAsk         string `json:"spotAsk"`
	PerpBid         string `json:"perpBid"`
	PerpAsk         string `json:"perpAsk"`
	ThinnerBook     string `json:"thinnerBook"`
	SpotThinnerSide string `json:"spotThinnerSide"`
	PerpThinnerSide string `json:"perpThinnerSide"`
	Complete        bool   `json:"complete"`
	Error           string `json:"error,omitempty"`
}

type spotPerpResponse struct {
	Symbol       string             `json:"symbol"`
	RangePct     string             `json:"rangePct"`
	AsOf         time.Time          `json:"asOf"`
	Spot         spotPerpBookDTO    `json:"spot"`
	Perp         spotPerpBookDTO    `json:"perp"`
	ThinnerBook  string             `json:"thinnerBook"`
	StrongerBook string             `json:"strongerBook"`
	Ratio        string             `json:"ratio"`
	Bands        []spotPerpBandDTO  `json:"bands"`
	Venues       []spotPerpVenueDTO `json:"venues"`
	Summary      string             `json:"summary"`
	Note         string             `json:"note"`
}

func spotPerpToDTO(a *domain.SpotPerpLiquidity) spotPerpResponse {
	if a == nil {
		return spotPerpResponse{}
	}
	bands := make([]spotPerpBandDTO, 0, len(a.Bands))
	for _, b := range a.Bands {
		bands = append(bands, spotPerpBandDTO{
			RangePct: formatHistQty(b.RangePct), SpotBid: formatHistQty(b.SpotBid), SpotAsk: formatHistQty(b.SpotAsk),
			PerpBid: formatHistQty(b.PerpBid), PerpAsk: formatHistQty(b.PerpAsk),
			ThinnerBook: b.ThinnerBook, StrongerBook: b.StrongerBook,
			SpotThinnerSide: b.SpotThinnerSide, PerpThinnerSide: b.PerpThinnerSide, Complete: b.Complete,
		})
	}
	venues := make([]spotPerpVenueDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		venues = append(venues, spotPerpVenueDTO{
			Exchange: string(v.Exchange), SpotBid: formatHistQty(v.SpotBid), SpotAsk: formatHistQty(v.SpotAsk),
			PerpBid: formatHistQty(v.PerpBid), PerpAsk: formatHistQty(v.PerpAsk),
			ThinnerBook: v.ThinnerBook, SpotThinnerSide: v.SpotThinnerSide, PerpThinnerSide: v.PerpThinnerSide,
			Complete: v.Complete, Error: v.Error,
		})
	}
	return spotPerpResponse{
		Symbol: a.Symbol, RangePct: formatHistQty(a.RangePct), AsOf: a.AsOf.UTC(),
		Spot: spotPerpBookToDTO(a.Spot), Perp: spotPerpBookToDTO(a.Perp),
		ThinnerBook: a.ThinnerBook, StrongerBook: a.StrongerBook, Ratio: formatHistQty(a.Ratio),
		Bands: bands, Venues: venues, Summary: a.Summary, Note: a.Note,
	}
}

func spotPerpBookToDTO(b domain.SpotPerpBook) spotPerpBookDTO {
	return spotPerpBookDTO{
		BidNotional: formatHistQty(b.BidNotional), AskNotional: formatHistQty(b.AskNotional), Total: formatHistQty(b.Total),
		ThinnerSide: b.ThinnerSide, StrongerSide: b.StrongerSide, Mid: formatHistQty(b.Mid), CoverPct: formatHistQty(b.CoverPct),
		VenueCount: b.VenueCount, Complete: b.Complete,
	}
}
