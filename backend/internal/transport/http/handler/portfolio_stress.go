package handler

import (
	"encoding/json"
	"net/http"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type stressBody struct {
	PortfolioID   string       `json:"portfolioId"`
	OwnerClientID string       `json:"ownerClientId"`
	Moves         []stressMove `json:"moves"`
}

type stressMove struct {
	Asset   string  `json:"asset"`
	MovePct float64 `json:"movePct"`
}

// StressPortfolio handles POST /api/v1/portfolio/stress.
func (h *PortfolioHandler) StressPortfolio(w http.ResponseWriter, r *http.Request) {
	var body stressBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, err)
		return
	}
	moves := make([]domain.StressMove, 0, len(body.Moves))
	for _, m := range body.Moves {
		moves = append(moves, domain.StressMove{Asset: m.Asset, MovePct: m.MovePct})
	}
	pid := body.PortfolioID
	if pid == "" {
		pid = portfolioIDFrom(r)
	}
	owner := body.OwnerClientID
	if owner == "" {
		owner = ownerClientIDFrom(r)
	}
	rep, err := h.svc.Stress(r.Context(), clientIDFrom(r), pid, owner, moves)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stressDTO(rep))
}

func stressDTO(rep *domain.StressReport) map[string]any {
	lines := make([]map[string]any, 0, len(rep.Lines))
	for _, ln := range rep.Lines {
		lines = append(lines, map[string]any{
			"kind": ln.Kind, "symbol": ln.Symbol, "side": ln.Side, "asset": ln.Asset,
			"movePct": ln.MovePct, "mark": ln.Mark, "shockedMark": ln.ShockedMark,
			"value": ln.Value, "shockedValue": ln.ShockedValue,
			"unrealized": ln.Unrealized, "shockedUnrealized": ln.ShockedUnrealized,
			"liquidated": ln.Liquidated,
		})
	}
	return map[string]any{
		"equity": rep.Equity, "shockedEquity": rep.ShockedEquity, "equityDelta": rep.EquityDelta,
		"totalPnl": rep.TotalPnL, "shockedTotalPnl": rep.ShockedTotalPnL,
		"unrealizedPnl": rep.UnrealizedPnL, "shockedUnrealizedPnl": rep.ShockedUnrealizedPnL,
		"positionsValue": rep.PositionsValue, "shockedPositionsValue": rep.ShockedPositionsValue,
		"marginEquity": rep.MarginEquity, "shockedMarginEquity": rep.ShockedMarginEquity,
		"liquidated": rep.Liquidated, "lines": lines, "note": rep.Note,
	}
}
