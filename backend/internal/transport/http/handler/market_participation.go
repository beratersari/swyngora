package handler

import (
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetParticipation handles GET /api/v1/market/participation.
func (h *MarketHandler) GetParticipation(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.GetParticipation(r.Context(), q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, participationToDTO(got))
}

type participationWindowDTO struct {
	Window   string   `json:"window"`
	Volume   string   `json:"volume"`
	OiDelta  string   `json:"oiDelta"`
	Ratio    string   `json:"ratio,omitempty"`
	RatioPct string   `json:"ratioPct,omitempty"`
	Transfer string   `json:"transfer,omitempty"`
	Lean     string   `json:"lean"`
	Complete bool     `json:"complete"`
	Summary  string   `json:"summary"`
	Missing  []string `json:"missing,omitempty"`
}

type participationVenueDTO struct {
	Exchange string                   `json:"exchange"`
	Symbol   string                   `json:"symbol"`
	Windows  []participationWindowDTO `json:"windows"`
	Summary  string                   `json:"summary"`
	Error    string                   `json:"error,omitempty"`
}

type participationResponse struct {
	Symbol   string                  `json:"symbol"`
	Exchange string                  `json:"exchange"`
	AsOf     time.Time               `json:"asOf"`
	Venues   []participationVenueDTO `json:"venues"`
	Combined *participationVenueDTO  `json:"combined,omitempty"`
	Note     string                  `json:"note"`
}

func participationToDTO(a *domain.ParticipationReport) participationResponse {
	if a == nil {
		return participationResponse{}
	}
	venues := make([]participationVenueDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		venues = append(venues, participationVenueToDTO(v))
	}
	out := participationResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, AsOf: a.AsOf.UTC(),
		Venues: venues, Note: a.Note,
	}
	if a.Combined != nil {
		c := participationVenueToDTO(*a.Combined)
		out.Combined = &c
	}
	return out
}

func participationVenueToDTO(v domain.ParticipationVenue) participationVenueDTO {
	wins := make([]participationWindowDTO, 0, len(v.Windows))
	for _, w := range v.Windows {
		wins = append(wins, participationWindowToDTO(w))
	}
	return participationVenueDTO{
		Exchange: string(v.Exchange), Symbol: v.Symbol, Windows: wins,
		Summary: v.Summary, Error: v.Error,
	}
}

func participationWindowToDTO(w domain.ParticipationWindow) participationWindowDTO {
	out := participationWindowDTO{
		Window: w.Window, Volume: formatHistQty(w.Volume),
		OiDelta: domain.FormatSignedQty(w.OIDelta), Lean: w.Lean,
		Complete: w.Complete, Summary: w.Summary, Missing: append([]string{}, w.Missing...),
	}
	if w.Volume > 0 {
		out.Ratio = formatHistQty(w.Ratio)
		out.RatioPct = domain.FormatSignedPct(w.RatioPct)
		out.Transfer = formatHistQty(w.Transfer)
	}
	return out
}
