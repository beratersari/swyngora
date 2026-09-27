package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// ScanSuspiciousWalls handles GET /api/v1/market/orderbook/walls/scan.
func (h *MarketHandler) ScanSuspiciousWalls(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 0
	if raw := strings.TrimSpace(q.Get("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, domain.ErrInvalidArgument)
			return
		}
		limit = n
	}
	got, err := h.svc.ScanSuspiciousWalls(r.Context(), q.Get("exchange"), q.Get("quote"), limit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, wallScanToDTO(got))
}

type wallScanHitDTO struct {
	Exchange          string    `json:"exchange"`
	Symbol            string    `json:"symbol"`
	Side              string    `json:"side"`
	Price             string    `json:"price"`
	Notional          string    `json:"notional"`
	DistancePct       string    `json:"distancePct"`
	Share             float64   `json:"share"`
	Kind              string    `json:"kind"`
	AppearCount       int       `json:"appearCount"`
	AgeSeconds        float64   `json:"ageSeconds"`
	PresentForSeconds float64   `json:"presentForSeconds"`
	VisibleSeconds    float64   `json:"visibleSeconds"`
	Present           bool      `json:"present"`
	LastSeen          time.Time `json:"lastSeen"`
}

type wallScanResponse struct {
	Exchange   string           `json:"exchange"`
	Quote      string           `json:"quote"`
	Universe   int              `json:"universe"`
	AsOf       time.Time        `json:"asOf"`
	Suspicious int              `json:"suspicious"`
	Flickers   int              `json:"flickers"`
	Hits       []wallScanHitDTO `json:"hits"`
	Summary    string           `json:"summary"`
	Note       string           `json:"note"`
}

func wallScanToDTO(a *domain.WallScan) wallScanResponse {
	if a == nil {
		return wallScanResponse{Hits: []wallScanHitDTO{}}
	}
	hits := make([]wallScanHitDTO, 0, len(a.Hits))
	for _, h := range a.Hits {
		seen := h.LastSeen
		if !seen.IsZero() {
			seen = seen.UTC()
		}
		hits = append(hits, wallScanHitDTO{
			Exchange: h.Exchange, Symbol: h.Symbol, Side: h.Side, Price: h.Price,
			Notional: h.Notional, DistancePct: h.DistancePct, Share: h.Share, Kind: h.Kind,
			AppearCount: h.AppearCount, AgeSeconds: h.AgeSeconds, PresentForSeconds: h.PresentForSeconds,
			VisibleSeconds: h.VisibleSeconds, Present: h.Present, LastSeen: seen,
		})
	}
	asOf := a.AsOf
	if !asOf.IsZero() {
		asOf = asOf.UTC()
	}
	return wallScanResponse{
		Exchange: a.Exchange, Quote: a.Quote, Universe: a.Universe, AsOf: asOf,
		Suspicious: a.Suspicious, Flickers: a.Flickers, Hits: hits, Summary: a.Summary, Note: a.Note,
	}
}
