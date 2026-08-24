package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetVolumeSurge handles GET /api/v1/market/volume-surge.
func (h *MarketHandler) GetVolumeSurge(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.GetVolumeSurge(r.Context(), q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, volumeSurgeToDTO(got))
}

// ScanVolumeSurges handles GET /api/v1/market/volume-surge/scan.
func (h *MarketHandler) ScanVolumeSurges(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var minRatio float64
	if raw := q.Get("minRatio"); raw != "" {
		n, err := strconv.ParseFloat(raw, 64)
		if err != nil || n < 0 {
			writeError(w, fmt.Errorf("%w: minRatio must be a number >= 0", domain.ErrInvalidArgument))
			return
		}
		minRatio = n
	}
	var limit int
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, fmt.Errorf("%w: limit must be an integer", domain.ErrInvalidArgument))
			return
		}
		limit = n
	}
	got, err := h.svc.ScanVolumeSurges(r.Context(), q.Get("exchange"), q.Get("quote"), minRatio, limit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, volumeSurgeScanToDTO(got))
}

type volumeSurgeWindowDTO struct {
	Window      string `json:"window"`
	Current     string `json:"current"`
	Typical     string `json:"typical"`
	Ratio       string `json:"ratio"`
	BuyCurrent  string `json:"buyCurrent"`
	BuyTypical  string `json:"buyTypical"`
	BuyRatio    string `json:"buyRatio"`
	SellCurrent string `json:"sellCurrent"`
	SellTypical string `json:"sellTypical"`
	SellRatio   string `json:"sellRatio"`
	Dominant    string `json:"dominant"`
	Grade       string `json:"grade"`
	SampleBars  int    `json:"sampleBars"`
	Complete    bool   `json:"complete"`
}

type volumeSurgeVenueDTO struct {
	Exchange     string                 `json:"exchange"`
	Symbol       string                 `json:"symbol"`
	BuySellKnown bool                   `json:"buySellKnown"`
	Windows      []volumeSurgeWindowDTO `json:"windows"`
	MaxRatio     string                 `json:"maxRatio"`
	Hottest      string                 `json:"hottest"`
	Summary      string                 `json:"summary"`
	Error        string                 `json:"error,omitempty"`
}

type volumeSurgeResponse struct {
	Symbol   string                `json:"symbol"`
	Exchange string                `json:"exchange"`
	AsOf     time.Time             `json:"asOf"`
	Venues   []volumeSurgeVenueDTO `json:"venues"`
	Summary  string                `json:"summary"`
	Note     string                `json:"note"`
}

type volumeSurgeHitDTO struct {
	Symbol       string                 `json:"symbol"`
	Exchange     string                 `json:"exchange"`
	BuySellKnown bool                   `json:"buySellKnown"`
	Windows      []volumeSurgeWindowDTO `json:"windows"`
	MaxRatio     string                 `json:"maxRatio"`
	Hottest      string                 `json:"hottest"`
	Grade        string                 `json:"grade"`
	Dominant     string                 `json:"dominant,omitempty"`
	Summary      string                 `json:"summary"`
}

type volumeSurgeScanResponse struct {
	Exchange    string              `json:"exchange"`
	Quote       string              `json:"quote"`
	MinRatio    string              `json:"minRatio"`
	SymbolLimit int                 `json:"symbolLimit"`
	AsOf        time.Time           `json:"asOf"`
	Hits        []volumeSurgeHitDTO `json:"hits"`
	Summary     string              `json:"summary"`
	Note        string              `json:"note"`
}

func volumeSurgeToDTO(a *domain.VolumeSurgeReport) volumeSurgeResponse {
	if a == nil {
		return volumeSurgeResponse{}
	}
	venues := make([]volumeSurgeVenueDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		venues = append(venues, volumeSurgeVenueToDTO(v))
	}
	return volumeSurgeResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, AsOf: a.AsOf.UTC(),
		Venues: venues, Summary: a.Summary, Note: a.Note,
	}
}

func volumeSurgeScanToDTO(a *domain.VolumeSurgeScan) volumeSurgeScanResponse {
	if a == nil {
		return volumeSurgeScanResponse{}
	}
	hits := make([]volumeSurgeHitDTO, 0, len(a.Hits))
	for _, h := range a.Hits {
		hits = append(hits, volumeSurgeHitDTO{
			Symbol: h.Symbol, Exchange: string(h.Exchange), BuySellKnown: h.BuySellKnown,
			Windows: volumeSurgeWindowsToDTO(h.Windows), MaxRatio: formatHistQty(h.MaxRatio),
			Hottest: h.Hottest, Grade: h.Grade, Dominant: h.Dominant, Summary: h.Summary,
		})
	}
	return volumeSurgeScanResponse{
		Exchange: a.Exchange, Quote: a.Quote, MinRatio: formatHistQty(a.MinRatio),
		SymbolLimit: a.SymbolLimit, AsOf: a.AsOf.UTC(), Hits: hits,
		Summary: a.Summary, Note: a.Note,
	}
}

func volumeSurgeVenueToDTO(v domain.VolumeSurgeVenue) volumeSurgeVenueDTO {
	return volumeSurgeVenueDTO{
		Exchange: string(v.Exchange), Symbol: v.Symbol, BuySellKnown: v.BuySellKnown,
		Windows: volumeSurgeWindowsToDTO(v.Windows), MaxRatio: formatHistQty(v.MaxRatio),
		Hottest: v.Hottest, Summary: v.Summary, Error: v.Error,
	}
}

func volumeSurgeWindowsToDTO(in []domain.VolumeSurgeWindow) []volumeSurgeWindowDTO {
	out := make([]volumeSurgeWindowDTO, 0, len(in))
	for _, w := range in {
		out = append(out, volumeSurgeWindowDTO{
			Window: w.Window, Current: formatHistQty(w.Current), Typical: formatHistQty(w.Typical),
			Ratio: formatHistQty(w.Ratio), BuyCurrent: formatHistQty(w.BuyCurrent),
			BuyTypical: formatHistQty(w.BuyTypical), BuyRatio: formatHistQty(w.BuyRatio),
			SellCurrent: formatHistQty(w.SellCurrent), SellTypical: formatHistQty(w.SellTypical),
			SellRatio: formatHistQty(w.SellRatio), Dominant: w.Dominant, Grade: w.Grade,
			SampleBars: w.SampleBars, Complete: w.Complete,
		})
	}
	return out
}
