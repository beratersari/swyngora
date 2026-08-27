package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
)

// GetFundingArb handles GET /api/v1/market/funding-arb.
func (h *MarketHandler) GetFundingArb(w http.ResponseWriter, r *http.Request) {
	in, err := parseFundingArbParams(r)
	if err != nil {
		writeError(w, err)
		return
	}
	got, err := h.svc.GetFundingArb(r.Context(), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, got)
}

// ScanFundingArb handles GET /api/v1/market/funding-arb/scan.
func (h *MarketHandler) ScanFundingArb(w http.ResponseWriter, r *http.Request) {
	in, err := parseFundingArbParams(r)
	if err != nil {
		writeError(w, err)
		return
	}
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
	got, err := h.svc.ScanFundingArb(r.Context(), market.FundingArbScanParams{
		Quote:         q.Get("quote"),
		Notional:      in.Notional,
		HoldHours:     in.HoldHours,
		FeeBinancePct: in.FeeBinancePct,
		FeeBybitPct:   in.FeeBybitPct,
		SymbolLimit:   limit,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, got)
}

// GetFundingArbHistory handles GET /api/v1/market/funding-arb/history.
func (h *MarketHandler) GetFundingArbHistory(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, err := parseFundingArbTime(q.Get("from"))
	if err != nil {
		writeError(w, fmt.Errorf("%w: from must be RFC3339, YYYY-MM-DD, or unix ms", domain.ErrInvalidArgument))
		return
	}
	to, err := parseFundingArbTime(q.Get("to"))
	if err != nil {
		writeError(w, fmt.Errorf("%w: to must be RFC3339, YYYY-MM-DD, or unix ms", domain.ErrInvalidArgument))
		return
	}
	base, err := parseFundingArbParams(r)
	if err != nil {
		writeError(w, err)
		return
	}
	got, err := h.svc.GetFundingArbHistory(r.Context(), market.FundingArbHistoryParams{
		Symbol:        q.Get("symbol"),
		From:          from,
		To:            to,
		Notional:      base.Notional,
		FeeBinancePct: base.FeeBinancePct,
		FeeBybitPct:   base.FeeBybitPct,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, got)
}

func parseFundingArbTime(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("%w: time is required", domain.ErrInvalidArgument)
	}
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse("2006-01-02", raw); err == nil {
		return t.UTC(), nil
	}
	if ms, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return time.UnixMilli(ms).UTC(), nil
	}
	return time.Time{}, fmt.Errorf("%w: time", domain.ErrInvalidArgument)
}

func parseFundingArbParams(r *http.Request) (market.FundingArbParams, error) {
	q := r.URL.Query()
	in := market.FundingArbParams{Symbol: q.Get("symbol")}
	if raw := strings.TrimSpace(q.Get("notional")); raw != "" {
		n, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return in, fmt.Errorf("%w: notional must be a number", domain.ErrInvalidArgument)
		}
		in.Notional = n
	}
	if raw := strings.TrimSpace(q.Get("holdHours")); raw != "" {
		n, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return in, fmt.Errorf("%w: holdHours must be a number", domain.ErrInvalidArgument)
		}
		in.HoldHours = n
	}
	fb, err := parseOptionalFloatPtr(q.Get("feeBinancePct"))
	if err != nil {
		return in, fmt.Errorf("%w: feeBinancePct must be a number", domain.ErrInvalidArgument)
	}
	in.FeeBinancePct = fb
	fy, err := parseOptionalFloatPtr(q.Get("feeBybitPct"))
	if err != nil {
		return in, fmt.Errorf("%w: feeBybitPct must be a number", domain.ErrInvalidArgument)
	}
	in.FeeBybitPct = fy
	return in, nil
}

func parseOptionalFloatPtr(raw string) (*float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	n, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil, err
	}
	return &n, nil
}
