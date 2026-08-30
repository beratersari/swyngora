package handler

import (
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetLiquidationCascade handles GET /api/v1/market/liquidation-cascade.
func (h *MarketHandler) GetLiquidationCascade(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.GetLiquidationCascade(r.Context(), q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cascadeReportToDTO(got))
}

// ScanLiquidationCascades handles GET /api/v1/market/liquidation-cascade/scan.
func (h *MarketHandler) ScanLiquidationCascades(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.ScanLiquidationCascades(r.Context(), q.Get("exchange"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cascadeScanToDTO(got))
}

type cascadeWindowDTO struct {
	Window        string  `json:"window"`
	LongNotional  string  `json:"longNotional"`
	ShortNotional string  `json:"shortNotional"`
	TotalNotional string  `json:"totalNotional"`
	LongTypical   string  `json:"longTypical"`
	ShortTypical  string  `json:"shortTypical"`
	LongRatio     float64 `json:"longRatio"`
	ShortRatio    float64 `json:"shortRatio"`
	MaxRatio      float64 `json:"maxRatio"`
	Side          string  `json:"side"`
	Grade         string  `json:"grade"`
	Count         int     `json:"count"`
	SampleBuckets int     `json:"sampleBuckets"`
	Complete      bool    `json:"complete"`
}

type cascadeVenueDTO struct {
	Exchange  string             `json:"exchange"`
	Symbol    string             `json:"symbol"`
	Windows   []cascadeWindowDTO `json:"windows"`
	Side      string             `json:"side"`
	Grade     string             `json:"grade"`
	Score     float64            `json:"score"`
	Hottest   string             `json:"hottest"`
	StartedAt string             `json:"startedAt,omitempty"`
	Summary   string             `json:"summary"`
}

type cascadeBothDTO struct {
	Agree   bool    `json:"agree"`
	Side    string  `json:"side"`
	Grade   string  `json:"grade"`
	Score   float64 `json:"score"`
	Hottest string  `json:"hottest"`
	Summary string  `json:"summary"`
}

type cascadeReportDTO struct {
	Symbol   string            `json:"symbol"`
	Exchange string            `json:"exchange"`
	AsOf     string            `json:"asOf"`
	Venues   []cascadeVenueDTO `json:"venues"`
	Both     *cascadeBothDTO   `json:"both,omitempty"`
	Summary  string            `json:"summary"`
	Note     string            `json:"note"`
}

type cascadeHitDTO struct {
	Symbol  string  `json:"symbol"`
	Side    string  `json:"side"`
	Grade   string  `json:"grade"`
	Score   float64 `json:"score"`
	Hottest string  `json:"hottest"`
	Both    bool    `json:"both"`
	Summary string  `json:"summary"`
}

type cascadeScanDTO struct {
	Exchange string           `json:"exchange"`
	AsOf     string           `json:"asOf"`
	Market   cascadeReportDTO `json:"market"`
	Hits     []cascadeHitDTO  `json:"hits"`
	Summary  string           `json:"summary"`
	Note     string           `json:"note"`
}

func cascadeReportToDTO(a *domain.CascadeReport) cascadeReportDTO {
	if a == nil {
		return cascadeReportDTO{Venues: []cascadeVenueDTO{}}
	}
	venues := make([]cascadeVenueDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		venues = append(venues, cascadeVenueToDTO(v))
	}
	out := cascadeReportDTO{
		Symbol: a.Symbol, Exchange: a.Exchange,
		AsOf: a.AsOf.UTC().Format(time.RFC3339Nano),
		Venues: venues, Summary: a.Summary, Note: a.Note,
	}
	if a.Both != nil {
		out.Both = &cascadeBothDTO{
			Agree: a.Both.Agree, Side: a.Both.Side, Grade: a.Both.Grade,
			Score: a.Both.Score, Hottest: a.Both.Hottest, Summary: a.Both.Summary,
		}
	}
	return out
}

func cascadeScanToDTO(a *domain.CascadeScan) cascadeScanDTO {
	if a == nil {
		return cascadeScanDTO{Hits: []cascadeHitDTO{}}
	}
	hits := make([]cascadeHitDTO, 0, len(a.Hits))
	for _, h := range a.Hits {
		hits = append(hits, cascadeHitDTO{
			Symbol: h.Symbol, Side: h.Side, Grade: h.Grade,
			Score: h.Score, Hottest: h.Hottest, Both: h.Both, Summary: h.Summary,
		})
	}
	return cascadeScanDTO{
		Exchange: a.Exchange,
		AsOf:     a.AsOf.UTC().Format(time.RFC3339Nano),
		Market:   cascadeReportToDTO(&a.Market),
		Hits:     hits,
		Summary:  a.Summary,
		Note:     a.Note,
	}
}

func cascadeVenueToDTO(v domain.CascadeVenue) cascadeVenueDTO {
	windows := make([]cascadeWindowDTO, 0, len(v.Windows))
	for _, w := range v.Windows {
		windows = append(windows, cascadeWindowDTO{
			Window: w.Window, LongNotional: w.LongNotional, ShortNotional: w.ShortNotional,
			TotalNotional: w.TotalNotional, LongTypical: w.LongTypical, ShortTypical: w.ShortTypical,
			LongRatio: w.LongRatio, ShortRatio: w.ShortRatio, MaxRatio: w.MaxRatio,
			Side: w.Side, Grade: w.Grade, Count: w.Count,
			SampleBuckets: w.SampleBuckets, Complete: w.Complete,
		})
	}
	started := ""
	if !v.StartedAt.IsZero() {
		started = v.StartedAt.UTC().Format(time.RFC3339Nano)
	}
	return cascadeVenueDTO{
		Exchange: string(v.Exchange), Symbol: v.Symbol, Windows: windows,
		Side: v.Side, Grade: v.Grade, Score: v.Score, Hottest: v.Hottest,
		StartedAt: started, Summary: v.Summary,
	}
}
