package handler

import (
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetVenueDivergence handles GET /api/v1/market/venue-divergence.
func (h *MarketHandler) GetVenueDivergence(w http.ResponseWriter, r *http.Request) {
	got, err := h.svc.GetVenueDivergence(r.Context(), r.URL.Query().Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, divergenceToDTO(got))
}

type venueReadingDTO struct {
	Exchange string `json:"exchange"`
	Value    string `json:"value"`
}

type venueDiffDTO struct {
	Metric       string            `json:"metric"`
	Label        string            `json:"label"`
	Readings     []venueReadingDTO `json:"readings"`
	Same         []string          `json:"same,omitempty"`
	Different    []string          `json:"different,omitempty"`
	Binance      string            `json:"binance,omitempty"`
	Bybit        string            `json:"bybit,omitempty"`
	Alignment    string            `json:"alignment"`
	Important    bool              `json:"important"`
	WhyItMatters string            `json:"whyItMatters"`
}

type venueLeanDTO struct {
	Exchange string `json:"exchange"`
	Lean     string `json:"lean"`
	Regime   string `json:"regime"`
	OIChange string `json:"oiChange,omitempty"`
}

type venueGroupDTO struct {
	Lean      string   `json:"lean"`
	Exchanges []string `json:"exchanges"`
}

type venueDivergenceResponse struct {
	Symbol          string          `json:"symbol"`
	AsOf            time.Time       `json:"asOf"`
	Alignment       string          `json:"alignment"`
	Venues          []venueLeanDTO  `json:"venues"`
	Groups          []venueGroupDTO `json:"groups"`
	Outliers        []string        `json:"outliers,omitempty"`
	BinanceLean     string          `json:"binanceLean,omitempty"`
	BybitLean       string          `json:"bybitLean,omitempty"`
	OKXLean         string          `json:"okxLean,omitempty"`
	Important       bool            `json:"important"`
	Title           string          `json:"title"`
	Summary         string          `json:"summary"`
	Diffs           []venueDiffDTO  `json:"diffs"`
	BinanceRegime   string          `json:"binanceRegime,omitempty"`
	BybitRegime     string          `json:"bybitRegime,omitempty"`
	OKXRegime       string          `json:"okxRegime,omitempty"`
	BinanceOIChange string          `json:"binanceOiChange,omitempty"`
	BybitOIChange   string          `json:"bybitOiChange,omitempty"`
	OKXOIChange     string          `json:"okxOiChange,omitempty"`
	Note            string          `json:"note"`
}

func divergenceToDTO(a *domain.VenueDivergenceReport) venueDivergenceResponse {
	if a == nil {
		return venueDivergenceResponse{}
	}
	diffs := make([]venueDiffDTO, 0, len(a.Diffs))
	for _, d := range a.Diffs {
		readings := make([]venueReadingDTO, 0, len(d.Readings))
		for _, r := range d.Readings {
			readings = append(readings, venueReadingDTO{Exchange: r.Exchange, Value: r.Value})
		}
		diffs = append(diffs, venueDiffDTO{
			Metric: d.Metric, Label: d.Label, Readings: readings,
			Same: d.Same, Different: d.Different,
			Binance: d.Binance, Bybit: d.Bybit,
			Alignment: d.Alignment, Important: d.Important,
			WhyItMatters: d.WhyItMatters,
		})
	}
	venues := make([]venueLeanDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		venues = append(venues, venueLeanDTO{Exchange: v.Exchange, Lean: v.Lean, Regime: v.Regime, OIChange: v.OIChange})
	}
	groups := make([]venueGroupDTO, 0, len(a.Groups))
	for _, g := range a.Groups {
		groups = append(groups, venueGroupDTO{Lean: g.Lean, Exchanges: g.Exchanges})
	}
	return venueDivergenceResponse{
		Symbol: a.Symbol, AsOf: a.AsOf.UTC(),
		Alignment: a.Alignment, Venues: venues, Groups: groups, Outliers: a.Outliers,
		BinanceLean: a.BinanceLean, BybitLean: a.BybitLean, OKXLean: a.OKXLean,
		Important: a.Important, Title: a.Title, Summary: a.Summary,
		Diffs: diffs, BinanceRegime: a.BinanceRegime, BybitRegime: a.BybitRegime, OKXRegime: a.OKXRegime,
		BinanceOIChange: a.BinanceOIChange, BybitOIChange: a.BybitOIChange, OKXOIChange: a.OKXOIChange,
		Note: a.Note,
	}
}
