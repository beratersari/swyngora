package handler

import (
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetFearGreed handles GET /api/v1/market/fear-greed.
func (h *MarketHandler) GetFearGreed(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, domain.ErrUpstream)
		return
	}
	got, err := h.svc.GetFearGreed(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, fearGreedToDTO(got))
}

type fearGreedFactorDTO struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Detail   string  `json:"detail"`
	Reading  int     `json:"reading"`
	Weight   int     `json:"weight"`
	SharePct float64 `json:"sharePct"`
	Pull     string  `json:"pull"`
	Toward   string  `json:"toward"`
}

type fearGreedSampleDTO struct {
	Time    time.Time `json:"time"`
	Score   int       `json:"score"`
	Level   string    `json:"level"`
	TopID   string    `json:"topId,omitempty"`
	TopPull string    `json:"topPull,omitempty"`
}

type fearGreedWindowDTO struct {
	Window     string     `json:"window"`
	PastScore  *int       `json:"pastScore,omitempty"`
	PastLevel  string     `json:"pastLevel,omitempty"`
	Score      int        `json:"score"`
	Level      string     `json:"level"`
	Change     *int       `json:"change,omitempty"`
	Complete   bool       `json:"complete"`
	SampleTime *time.Time `json:"sampleTime,omitempty"`
}

type fearGreedResponse struct {
	Score    int                  `json:"score"`
	Level    string               `json:"level"`
	Label    string               `json:"label"`
	AsOf     time.Time            `json:"asOf"`
	Used     int                  `json:"used"`
	Total    int                  `json:"total"`
	Complete bool                 `json:"complete"`
	Top      *fearGreedFactorDTO  `json:"top,omitempty"`
	Factors  []fearGreedFactorDTO `json:"factors"`
	Missing  []string             `json:"missing,omitempty"`
	History  []fearGreedSampleDTO `json:"history"`
	Windows  []fearGreedWindowDTO `json:"windows"`
	Summary  string               `json:"summary"`
	Note     string               `json:"note"`
}

func fearGreedToDTO(a *domain.FearGreedReport) fearGreedResponse {
	if a == nil {
		return fearGreedResponse{Factors: []fearGreedFactorDTO{}, History: []fearGreedSampleDTO{}, Windows: []fearGreedWindowDTO{}}
	}
	factors := make([]fearGreedFactorDTO, 0, len(a.Factors))
	for _, f := range a.Factors {
		factors = append(factors, fearGreedFactorToDTO(f))
	}
	var top *fearGreedFactorDTO
	if a.Top != nil {
		dto := fearGreedFactorToDTO(*a.Top)
		top = &dto
	}
	hist := make([]fearGreedSampleDTO, 0, len(a.History))
	for _, s := range a.History {
		row := fearGreedSampleDTO{Time: s.Time.UTC(), Score: s.Score, Level: s.Level, TopID: s.TopID}
		if s.TopPull != 0 {
			row.TopPull = domain.FormatSignedPct(s.TopPull)
		}
		hist = append(hist, row)
	}
	wins := make([]fearGreedWindowDTO, 0, len(a.Windows))
	for _, w := range a.Windows {
		row := fearGreedWindowDTO{Window: w.Window, Score: w.Score, Level: w.Level, Complete: w.Complete}
		if w.HasPast {
			past := w.PastScore
			change := w.Change
			row.PastScore = &past
			row.PastLevel = w.PastLevel
			row.Change = &change
			if !w.SampleTime.IsZero() {
				ts := w.SampleTime.UTC()
				row.SampleTime = &ts
			}
		}
		wins = append(wins, row)
	}
	return fearGreedResponse{
		Score: a.Score, Level: a.Level, Label: a.Label, AsOf: a.AsOf.UTC(),
		Used: a.Used, Total: a.Total, Complete: a.Complete,
		Top: top, Factors: factors, Missing: a.Missing, History: hist, Windows: wins,
		Summary: a.Summary, Note: a.Note,
	}
}

func fearGreedFactorToDTO(f domain.FearGreedFactor) fearGreedFactorDTO {
	return fearGreedFactorDTO{
		ID: f.ID, Name: f.Name, Detail: f.Detail, Reading: f.Reading, Weight: f.Weight,
		SharePct: f.SharePct, Pull: domain.FormatSignedPct(f.Pull), Toward: f.Toward,
	}
}
