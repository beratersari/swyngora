package handler

import (
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetLiquiditySweeps handles GET /api/v1/market/liquidity-sweeps.
func (h *MarketHandler) GetLiquiditySweeps(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.GetLiquiditySweeps(r.Context(), q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, liquiditySweepToDTO(got))
}

type liquiditySweepDTO struct {
	Side            string    `json:"side"`
	Level           string    `json:"level"`
	LevelTime       time.Time `json:"levelTime"`
	Tests           int       `json:"tests"`
	PiercedAt       time.Time `json:"piercedAt"`
	ReclaimedAt     time.Time `json:"reclaimedAt,omitempty"`
	Extreme         string    `json:"extreme"`
	Excursion       string    `json:"excursion"`
	ExcursionPct    string    `json:"excursionPct"`
	Duration        string    `json:"duration"`
	DurationSeconds int       `json:"durationSeconds"`
	Volume          string    `json:"volume"`
	BuyVolume       string    `json:"buyVolume"`
	SellVolume      string    `json:"sellVolume"`
	BuySellKnown    bool      `json:"buySellKnown"`
	Bars            int       `json:"bars"`
	Status          string    `json:"status"`
	Interval        string    `json:"interval"`
	Title           string    `json:"title"`
	Summary         string    `json:"summary"`
}

type liquiditySweepVenueDTO struct {
	Exchange  string              `json:"exchange"`
	Symbol    string              `json:"symbol"`
	Interval  string              `json:"interval"`
	LastPrice string              `json:"lastPrice"`
	Sweeps    []liquiditySweepDTO `json:"sweeps"`
	Current   *liquiditySweepDTO  `json:"current,omitempty"`
	Summary   string              `json:"summary"`
	Error     string              `json:"error,omitempty"`
}

type liquiditySweepResponse struct {
	Symbol   string                   `json:"symbol"`
	Exchange string                   `json:"exchange"`
	AsOf     time.Time                `json:"asOf"`
	Venues   []liquiditySweepVenueDTO `json:"venues"`
	Summary  string                   `json:"summary"`
	Note     string                   `json:"note"`
}

func liquiditySweepToDTO(a *domain.LiquiditySweepReport) liquiditySweepResponse {
	if a == nil {
		return liquiditySweepResponse{}
	}
	venues := make([]liquiditySweepVenueDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		venues = append(venues, liquiditySweepVenueToDTO(v))
	}
	return liquiditySweepResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, AsOf: a.AsOf.UTC(),
		Venues: venues, Summary: a.Summary, Note: a.Note,
	}
}

func liquiditySweepVenueToDTO(v domain.LiquiditySweepVenue) liquiditySweepVenueDTO {
	rows := make([]liquiditySweepDTO, 0, len(v.Sweeps))
	for _, s := range v.Sweeps {
		rows = append(rows, liquiditySweepItemToDTO(s))
	}
	out := liquiditySweepVenueDTO{
		Exchange: string(v.Exchange), Symbol: v.Symbol, Interval: v.Interval,
		LastPrice: formatHistQty(v.LastPrice), Sweeps: rows, Summary: v.Summary, Error: v.Error,
	}
	if v.Current != nil {
		c := liquiditySweepItemToDTO(*v.Current)
		out.Current = &c
	}
	return out
}

func liquiditySweepItemToDTO(s domain.LiquiditySweep) liquiditySweepDTO {
	out := liquiditySweepDTO{
		Side: s.Side, Level: formatHistQty(s.Level), LevelTime: s.LevelTime.UTC(),
		Tests: s.Tests, PiercedAt: s.PiercedAt.UTC(), Extreme: formatHistQty(s.Extreme),
		Excursion: formatHistQty(s.Excursion), ExcursionPct: formatHistQty(s.ExcursionPct),
		Duration: s.Duration, DurationSeconds: s.DurationSeconds,
		Volume: formatHistQty(s.Volume), BuyVolume: formatHistQty(s.BuyVolume),
		SellVolume: formatHistQty(s.SellVolume), BuySellKnown: s.BuySellKnown,
		Bars: s.Bars, Status: s.Status, Interval: s.Interval, Title: s.Title, Summary: s.Summary,
	}
	if !s.ReclaimedAt.IsZero() {
		out.ReclaimedAt = s.ReclaimedAt.UTC()
	}
	return out
}
