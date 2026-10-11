package handler

import (
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetNetPosition handles GET /api/v1/market/net-position.
func (h *MarketHandler) GetNetPosition(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, domain.ErrUpstream)
		return
	}
	q := r.URL.Query()
	got, err := h.svc.GetNetPosition(r.Context(), q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, netPositionToDTO(got))
}

type netPositionWindowDTO struct {
	Window      string   `json:"window"`
	NetLong     string   `json:"netLong"`
	NetShort    string   `json:"netShort"`
	Buy         string   `json:"buy"`
	Sell        string   `json:"sell"`
	OIDelta     string   `json:"oiDelta"`
	Unexplained string   `json:"unexplained,omitempty"`
	Complete    bool     `json:"complete"`
	Summary     string   `json:"summary"`
	Missing     []string `json:"missing,omitempty"`
}

type netPositionVenueDTO struct {
	Exchange string                 `json:"exchange"`
	Windows  []netPositionWindowDTO `json:"windows,omitempty"`
	Error    string                 `json:"error,omitempty"`
}

type netPositionResponse struct {
	Symbol   string                 `json:"symbol"`
	Exchange string                 `json:"exchange"`
	AsOf     time.Time              `json:"asOf"`
	Windows  []netPositionWindowDTO `json:"windows"`
	Venues   []netPositionVenueDTO  `json:"venues,omitempty"`
	Summary  string                 `json:"summary"`
	Note     string                 `json:"note"`
}

func netPositionToDTO(a *domain.NetPositionReport) netPositionResponse {
	if a == nil {
		return netPositionResponse{Windows: []netPositionWindowDTO{}}
	}
	return netPositionResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, AsOf: a.AsOf.UTC(),
		Windows: netPositionWindowsDTO(a.Windows),
		Venues:  netPositionVenuesDTO(a.Venues),
		Summary: a.Summary, Note: a.Note,
	}
}

func netPositionWindowsDTO(rows []domain.NetPositionWindow) []netPositionWindowDTO {
	out := make([]netPositionWindowDTO, 0, len(rows))
	for _, w := range rows {
		row := netPositionWindowDTO{
			Window: w.Window, Complete: w.Complete, Summary: w.Summary,
			Missing: append([]string{}, w.Missing...),
		}
		if w.Complete {
			row.NetLong = domain.FormatSignedQty(w.NetLong)
			row.NetShort = domain.FormatSignedQty(w.NetShort)
			row.Buy = domain.FormatSignedQty(w.Buy)
			row.Sell = domain.FormatSignedQty(w.Sell)
			row.OIDelta = domain.FormatSignedQty(w.OIDelta)
			if w.Unexplained != 0 {
				row.Unexplained = domain.FormatSignedQty(w.Unexplained)
			}
		}
		out = append(out, row)
	}
	return out
}

func netPositionVenuesDTO(rows []domain.NetPositionVenue) []netPositionVenueDTO {
	out := make([]netPositionVenueDTO, 0, len(rows))
	for _, v := range rows {
		out = append(out, netPositionVenueDTO{
			Exchange: v.Exchange, Windows: netPositionWindowsDTO(v.Windows), Error: v.Error,
		})
	}
	return out
}
