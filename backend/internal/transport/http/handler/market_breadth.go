package handler

import (
	"net/http"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetBreadth handles GET /api/v1/market/breadth.
func (h *MarketHandler) GetBreadth(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 0
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, domain.ErrInvalidArgument)
			return
		}
		limit = n
	}
	got, err := h.svc.GetBreadth(r.Context(), q.Get("exchange"), limit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, breadthToDTO(got))
}

type breadthMoveDTO struct {
	Symbol    string `json:"symbol"`
	ChangePct string `json:"changePct"`
}

type breadthCountsDTO struct {
	Up            int    `json:"up"`
	Down          int    `json:"down"`
	Flat          int    `json:"flat"`
	Total         int    `json:"total"`
	UpPct         string `json:"upPct"`
	DownPct       string `json:"downPct"`
	FlatPct       string `json:"flatPct"`
	VolumeUpPct   string `json:"volumeUpPct"`
	VolumeDownPct string `json:"volumeDownPct"`
}

type breadthWindowDTO struct {
	Window    string           `json:"window"`
	Counts    breadthCountsDTO `json:"counts"`
	BTC       *breadthMoveDTO  `json:"btc,omitempty"`
	ETH       *breadthMoveDTO  `json:"eth,omitempty"`
	Alignment string           `json:"alignment"`
	Title     string           `json:"title"`
	Summary   string           `json:"summary"`
	Complete  bool             `json:"complete"`
}

type breadthResponse struct {
	Exchange string             `json:"exchange"`
	Quote    string             `json:"quote"`
	Universe int                `json:"universe"`
	AsOf     time.Time          `json:"asOf"`
	Windows  []breadthWindowDTO `json:"windows"`
	Summary  string             `json:"summary"`
	Note     string             `json:"note"`
}

func breadthToDTO(a *domain.BreadthReport) breadthResponse {
	if a == nil {
		return breadthResponse{}
	}
	wins := make([]breadthWindowDTO, 0, len(a.Windows))
	for _, w := range a.Windows {
		row := breadthWindowDTO{
			Window: w.Window,
			Counts: breadthCountsDTO{
				Up: w.Counts.Up, Down: w.Counts.Down, Flat: w.Counts.Flat, Total: w.Counts.Total,
				UpPct: domain.FormatSignedPct(w.Counts.UpPct), DownPct: domain.FormatSignedPct(w.Counts.DownPct),
				FlatPct:     domain.FormatSignedPct(w.Counts.FlatPct),
				VolumeUpPct: domain.FormatSignedPct(w.Counts.VolumeUpPct), VolumeDownPct: domain.FormatSignedPct(w.Counts.VolumeDownPct),
			},
			Alignment: w.Alignment, Title: w.Title, Summary: w.Summary, Complete: w.Complete,
		}
		if w.BTC != nil && w.BTC.Known {
			row.BTC = &breadthMoveDTO{Symbol: w.BTC.Symbol, ChangePct: domain.FormatSignedPct(w.BTC.ChangePct)}
		}
		if w.ETH != nil && w.ETH.Known {
			row.ETH = &breadthMoveDTO{Symbol: w.ETH.Symbol, ChangePct: domain.FormatSignedPct(w.ETH.ChangePct)}
		}
		wins = append(wins, row)
	}
	return breadthResponse{
		Exchange: a.Exchange, Quote: a.Quote, Universe: a.Universe, AsOf: a.AsOf.UTC(),
		Windows: wins, Summary: a.Summary, Note: a.Note,
	}
}
