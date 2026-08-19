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

type venueDiffDTO struct {
	Metric       string `json:"metric"`
	Label        string `json:"label"`
	Binance      string `json:"binance"`
	Bybit        string `json:"bybit"`
	Alignment    string `json:"alignment"`
	Important    bool   `json:"important"`
	WhyItMatters string `json:"whyItMatters"`
}

type venueDivergenceResponse struct {
	Symbol          string         `json:"symbol"`
	AsOf            time.Time      `json:"asOf"`
	Alignment       string         `json:"alignment"`
	BinanceLean     string         `json:"binanceLean"`
	BybitLean       string         `json:"bybitLean"`
	Important       bool           `json:"important"`
	Title           string         `json:"title"`
	Summary         string         `json:"summary"`
	Diffs           []venueDiffDTO `json:"diffs"`
	BinanceRegime   string         `json:"binanceRegime"`
	BybitRegime     string         `json:"bybitRegime"`
	BinanceOIChange string         `json:"binanceOiChange,omitempty"`
	BybitOIChange   string         `json:"bybitOiChange,omitempty"`
	Note            string         `json:"note"`
}

func divergenceToDTO(a *domain.VenueDivergenceReport) venueDivergenceResponse {
	if a == nil {
		return venueDivergenceResponse{}
	}
	diffs := make([]venueDiffDTO, 0, len(a.Diffs))
	for _, d := range a.Diffs {
		diffs = append(diffs, venueDiffDTO{
			Metric: d.Metric, Label: d.Label,
			Binance: d.Binance, Bybit: d.Bybit,
			Alignment: d.Alignment, Important: d.Important,
			WhyItMatters: d.WhyItMatters,
		})
	}
	return venueDivergenceResponse{
		Symbol: a.Symbol, AsOf: a.AsOf.UTC(),
		Alignment: a.Alignment, BinanceLean: a.BinanceLean, BybitLean: a.BybitLean,
		Important: a.Important, Title: a.Title, Summary: a.Summary,
		Diffs: diffs, BinanceRegime: a.BinanceRegime, BybitRegime: a.BybitRegime,
		BinanceOIChange: a.BinanceOIChange, BybitOIChange: a.BybitOIChange,
		Note: a.Note,
	}
}
