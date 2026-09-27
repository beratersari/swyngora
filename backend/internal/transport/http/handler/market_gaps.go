package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetBookGaps handles GET /api/v1/market/orderbook/gaps.
func (h *MarketHandler) GetBookGaps(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	rangePct, err := parseGapRange(q.Get("rangePct"))
	if err != nil {
		writeError(w, err)
		return
	}
	got, err := h.svc.GetBookGaps(r.Context(), q.Get("exchange"), q.Get("symbol"), rangePct)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bookGapsToDTO(got))
}

// ScanBookGaps handles GET /api/v1/market/orderbook/gaps/scan.
func (h *MarketHandler) ScanBookGaps(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	rangePct, err := parseGapRange(q.Get("rangePct"))
	if err != nil {
		writeError(w, err)
		return
	}
	limit := 0
	if raw := strings.TrimSpace(q.Get("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, fmt.Errorf("%w: limit must be an integer", domain.ErrInvalidArgument))
			return
		}
		limit = n
	}
	got, err := h.svc.ScanBookGaps(r.Context(), q.Get("exchange"), q.Get("quote"), rangePct, limit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bookGapScanToDTO(got))
}

func parseGapRange(raw string) (float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: rangePct must be a number", domain.ErrInvalidArgument)
	}
	return v, nil
}

type bookGapDTO struct {
	Side      string  `json:"side"`
	Direction string  `json:"direction"`
	FromPrice string  `json:"fromPrice"`
	ToPrice   string  `json:"toPrice"`
	FromPct   float64 `json:"fromPct"`
	ToPct     float64 `json:"toPct"`
	WidthPct  float64 `json:"widthPct"`
	Notional  string  `json:"notional"`
}

type bookGapsResponse struct {
	Symbol       string       `json:"symbol"`
	Exchange     string       `json:"exchange"`
	MidPrice     string       `json:"midPrice"`
	RangePct     float64      `json:"rangePct"`
	Easier       string       `json:"easier"`
	UpNotional   string       `json:"upNotional"`
	DownNotional string       `json:"downNotional"`
	UpGapPct     float64      `json:"upGapPct"`
	DownGapPct   float64      `json:"downGapPct"`
	Gaps         []bookGapDTO `json:"gaps"`
	Summary      string       `json:"summary"`
	Note         string       `json:"note"`
}

func bookGapsToDTO(a *domain.BookGaps) bookGapsResponse {
	if a == nil {
		return bookGapsResponse{Gaps: []bookGapDTO{}}
	}
	gaps := make([]bookGapDTO, 0, len(a.Gaps))
	for _, g := range a.Gaps {
		gaps = append(gaps, bookGapDTO{
			Side: g.Side, Direction: g.Direction, FromPrice: g.FromPrice, ToPrice: g.ToPrice,
			FromPct: g.FromPct, ToPct: g.ToPct, WidthPct: g.WidthPct, Notional: g.Notional,
		})
	}
	return bookGapsResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, MidPrice: a.MidPrice, RangePct: a.RangePct,
		Easier: a.Easier, UpNotional: a.UpNotional, DownNotional: a.DownNotional,
		UpGapPct: a.UpGapPct, DownGapPct: a.DownGapPct, Gaps: gaps, Summary: a.Summary, Note: a.Note,
	}
}

type bookGapScanHitDTO struct {
	Symbol       string  `json:"symbol"`
	Scope        string  `json:"scope"`
	Easier       string  `json:"easier"`
	UpNotional   string  `json:"upNotional"`
	DownNotional string  `json:"downNotional"`
	UpGapPct     float64 `json:"upGapPct"`
	DownGapPct   float64 `json:"downGapPct"`
	WidestSide   string  `json:"widestSide"`
	WidestPct    float64 `json:"widestPct"`
	Summary      string  `json:"summary"`
	Error        string  `json:"error,omitempty"`
}

type bookGapScanResponse struct {
	Exchange string              `json:"exchange"`
	Quote    string              `json:"quote"`
	RangePct float64             `json:"rangePct"`
	Universe int                 `json:"universe"`
	Hits     []bookGapScanHitDTO `json:"hits"`
	Summary  string              `json:"summary"`
	Note     string              `json:"note"`
}

func bookGapScanToDTO(a *domain.BookGapScan) bookGapScanResponse {
	if a == nil {
		return bookGapScanResponse{Hits: []bookGapScanHitDTO{}}
	}
	hits := make([]bookGapScanHitDTO, 0, len(a.Hits))
	for _, h := range a.Hits {
		hits = append(hits, bookGapScanHitDTO{
			Symbol: h.Symbol, Scope: h.Scope, Easier: h.Easier,
			UpNotional: h.UpNotional, DownNotional: h.DownNotional,
			UpGapPct: h.UpGapPct, DownGapPct: h.DownGapPct,
			WidestSide: h.WidestSide, WidestPct: h.WidestPct, Summary: h.Summary, Error: h.Error,
		})
	}
	return bookGapScanResponse{
		Exchange: a.Exchange, Quote: a.Quote, RangePct: a.RangePct, Universe: a.Universe,
		Hits: hits, Summary: a.Summary, Note: a.Note,
	}
}
