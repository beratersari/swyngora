package handler

import (
	"fmt"
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/portfolio"
)

type riskLimitsBody struct {
	MaxDailyLossPct   *float64 `json:"maxDailyLossPct"`
	MaxAssetWeightPct *float64 `json:"maxAssetWeightPct"`
}

func riskLimitsViewDTO(v *portfolio.RiskLimitsView) map[string]any {
	lim := map[string]any{
		"maxDailyLossPct":   v.Limits.MaxDailyLossPct,
		"maxAssetWeightPct": v.Limits.MaxAssetWeightPct,
	}
	if !v.Limits.UpdatedAt.IsZero() {
		lim["updatedAt"] = v.Limits.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}
	assets := make([]map[string]any, 0, len(v.Status.Assets))
	for _, a := range v.Status.Assets {
		assets = append(assets, map[string]any{
			"asset": a.Asset, "value": a.Value, "weightPct": a.WeightPct, "atOrOverLimit": a.AtOrOverLimit,
		})
	}
	return map[string]any{
		"clientId": v.Limits.ClientID,
		"limits":   lim,
		"status": map[string]any{
			"dayKey":            v.Status.DayKey,
			"timezone":          "UTC",
			"startOfDayEquity":  v.Status.StartOfDayEquity,
			"equity":            v.Status.Equity,
			"dailyPnl":          v.Status.DailyPnL,
			"dailyPnlPct":       v.Status.DailyPnLPct,
			"dailyLossLimitHit": v.Status.DailyLossLimitHit,
			"assets":            assets,
			"canOpenSpotBuy":    v.Status.CanOpenSpotBuy,
			"canOpenMargin":     v.Status.CanOpenMargin,
			"blockReasons":      v.Status.BlockReasons,
		},
		"note": v.Note,
	}
}

// GetRiskLimits handles GET /api/v1/portfolio/risk-limits
func (h *PortfolioHandler) GetRiskLimits(w http.ResponseWriter, r *http.Request) {
	v, err := h.svc.GetRiskLimitsView(r.Context(), clientIDFrom(r), portfolioIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, riskLimitsViewDTO(v))
}

// PutRiskLimits handles PUT /api/v1/portfolio/risk-limits
func (h *PortfolioHandler) PutRiskLimits(w http.ResponseWriter, r *http.Request) {
	var body riskLimitsBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	v, err := h.svc.SetRiskLimits(r.Context(), portfolio.RiskLimitsInput{
		ClientID: clientIDFrom(r), PortfolioID: portfolioIDFrom(r), MaxDailyLossPct: body.MaxDailyLossPct, MaxAssetWeightPct: body.MaxAssetWeightPct,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, riskLimitsViewDTO(v))
}

// DeleteRiskLimits handles DELETE /api/v1/portfolio/risk-limits
func (h *PortfolioHandler) DeleteRiskLimits(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.ClearRiskLimits(r.Context(), clientIDFrom(r), portfolioIDFrom(r)); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"deleted": true, "clientId": clientIDFrom(r),
		"note": "All risk limits removed. New buys and margin opens are unrestricted.",
	})
}
