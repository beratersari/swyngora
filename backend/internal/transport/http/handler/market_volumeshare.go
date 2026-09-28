package handler

import (
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetVolumeShare handles GET /api/v1/market/volume-share.
func (h *MarketHandler) GetVolumeShare(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.GetVolumeShare(r.Context(), q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, volumeShareToDTO(got))
}

type volumeShareVenueDTO struct {
	Exchange      string `json:"exchange"`
	Volume        string `json:"volume"`
	SharePct      string `json:"sharePct"`
	PriorVolume   string `json:"priorVolume,omitempty"`
	PriorSharePct string `json:"priorSharePct,omitempty"`
	ChangePct     string `json:"changePct,omitempty"`
	Kind          string `json:"kind"`
	Complete      bool   `json:"complete"`
	PriorKnown    bool   `json:"priorKnown"`
}

type volumeShareWindowDTO struct {
	Window   string                `json:"window"`
	Total    string                `json:"total"`
	Prior    string                `json:"prior,omitempty"`
	Venues   []volumeShareVenueDTO `json:"venues"`
	Gained   []string              `json:"gained,omitempty"`
	Lost     []string              `json:"lost,omitempty"`
	Missing  []string              `json:"missing,omitempty"`
	Complete bool                  `json:"complete"`
	Summary  string                `json:"summary"`
}

type volumeShareResponse struct {
	Symbol   string                 `json:"symbol"`
	Exchange string                 `json:"exchange"`
	AsOf     time.Time              `json:"asOf"`
	Windows  []volumeShareWindowDTO `json:"windows"`
	Note     string                 `json:"note"`
}

func volumeShareToDTO(a *domain.VolumeShareReport) volumeShareResponse {
	if a == nil {
		return volumeShareResponse{}
	}
	wins := make([]volumeShareWindowDTO, 0, len(a.Windows))
	for _, w := range a.Windows {
		venues := make([]volumeShareVenueDTO, 0, len(w.Venues))
		for _, v := range w.Venues {
			row := volumeShareVenueDTO{
				Exchange: string(v.Exchange), Volume: formatHistQty(v.Volume),
				SharePct: formatHistQty(v.SharePct), Kind: v.Kind, Complete: v.Complete,
				PriorKnown: v.PriorKnown,
			}
			if v.PriorKnown {
				row.PriorVolume = formatHistQty(v.PriorVolume)
				row.PriorSharePct = formatHistQty(v.PriorSharePct)
				row.ChangePct = domain.FormatSignedPct(v.ChangePct)
			}
			venues = append(venues, row)
		}
		win := volumeShareWindowDTO{
			Window: w.Window, Total: formatHistQty(w.Total), Venues: venues,
			Gained: w.Gained, Lost: w.Lost, Missing: w.Missing, Complete: w.Complete, Summary: w.Summary,
		}
		if w.Prior > 0 {
			win.Prior = formatHistQty(w.Prior)
		}
		wins = append(wins, win)
	}
	return volumeShareResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, AsOf: a.AsOf.UTC(),
		Windows: wins, Note: a.Note,
	}
}
