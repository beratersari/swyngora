package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetWideSpread handles GET /api/v1/market/orderbook/spread.
func (h *MarketHandler) GetWideSpread(w http.ResponseWriter, r *http.Request) {
	got, err := h.svc.GetWideSpread(r.Context(), r.URL.Query().Get("exchange"), r.URL.Query().Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, wideSpreadToDTO(got))
}

// ScanWideSpreads handles GET /api/v1/market/orderbook/spreads/scan.
func (h *MarketHandler) ScanWideSpreads(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 0
	if raw := strings.TrimSpace(q.Get("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, fmt.Errorf("%w: limit must be an integer", domain.ErrInvalidArgument))
			return
		}
		limit = n
	}
	got, err := h.svc.ScanWideSpreads(r.Context(), q.Get("exchange"), q.Get("quote"), limit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, wideSpreadScanToDTO(got))
}

type wideSpreadHitDTO struct {
	Exchange   string    `json:"exchange"`
	Symbol     string    `json:"symbol"`
	Base       string    `json:"base"`
	CurrentPct string    `json:"currentPct"`
	TypicalPct string    `json:"typicalPct"`
	Ratio      string    `json:"ratio"`
	Samples    int       `json:"samples"`
	AsOf       time.Time `json:"asOf"`
}

type wideSpreadScanResponse struct {
	Exchange string             `json:"exchange"`
	Quote    string             `json:"quote"`
	Known    int                `json:"known"`
	Ready    int                `json:"ready"`
	Hits     []wideSpreadHitDTO `json:"hits"`
	Summary  string             `json:"summary"`
	Note     string             `json:"note"`
}

type wideSpreadResponse struct {
	Symbol     string    `json:"symbol"`
	Exchange   string    `json:"exchange"`
	CurrentPct string    `json:"currentPct"`
	TypicalPct string    `json:"typicalPct"`
	Ratio      string    `json:"ratio"`
	Samples    int       `json:"samples"`
	Opened     bool      `json:"opened"`
	AsOf       time.Time `json:"asOf,omitempty"`
	Summary    string    `json:"summary"`
	Note       string    `json:"note"`
}

func wideSpreadScanToDTO(a *domain.SpreadScan) wideSpreadScanResponse {
	if a == nil {
		return wideSpreadScanResponse{Hits: []wideSpreadHitDTO{}}
	}
	hits := make([]wideSpreadHitDTO, 0, len(a.Hits))
	for _, h := range a.Hits {
		hits = append(hits, wideSpreadHitDTO{
			Exchange: string(h.Exchange), Symbol: h.Symbol, Base: h.Base,
			CurrentPct: formatHistQty(h.CurrentPct), TypicalPct: formatHistQty(h.TypicalPct),
			Ratio: formatHistQty(h.Ratio), Samples: h.Samples, AsOf: h.AsOf.UTC(),
		})
	}
	return wideSpreadScanResponse{
		Exchange: a.Exchange, Quote: a.Quote, Known: a.Known, Ready: a.Ready,
		Hits: hits, Summary: a.Summary, Note: a.Note,
	}
}

func wideSpreadToDTO(a *domain.SpreadReport) wideSpreadResponse {
	if a == nil {
		return wideSpreadResponse{}
	}
	out := wideSpreadResponse{
		Symbol: a.Symbol, Exchange: a.Exchange,
		CurrentPct: formatHistQty(a.CurrentPct), TypicalPct: formatHistQty(a.TypicalPct),
		Ratio: formatHistQty(a.Ratio), Samples: a.Samples, Opened: a.Opened,
		Summary: a.Summary, Note: a.Note,
	}
	if !a.AsOf.IsZero() {
		out.AsOf = a.AsOf.UTC()
	}
	return out
}
