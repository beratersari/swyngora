package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type stressBody struct {
	PortfolioID   string       `json:"portfolioId"`
	OwnerClientID string       `json:"ownerClientId"`
	Moves         []stressMove `json:"moves"`
	From          string       `json:"from"`
	To            string       `json:"to"`
}

type stressMove struct {
	Asset   string  `json:"asset"`
	MovePct float64 `json:"movePct"`
}

// StressPortfolio handles POST /api/v1/portfolio/stress.
// moves are typed percents. from and to replay a past window onto the open book.
// Send one of those, not both. Nothing is traded.
func (h *PortfolioHandler) StressPortfolio(w http.ResponseWriter, r *http.Request) {
	var body stressBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, err)
		return
	}
	pid := body.PortfolioID
	if pid == "" {
		pid = portfolioIDFrom(r)
	}
	owner := body.OwnerClientID
	if owner == "" {
		owner = ownerClientIDFrom(r)
	}
	hasPeriod := strings.TrimSpace(body.From) != "" || strings.TrimSpace(body.To) != ""
	if hasPeriod && len(body.Moves) > 0 {
		writeError(w, fmt.Errorf("%w: send moves or a period, not both", domain.ErrInvalidArgument))
		return
	}
	var (
		rep *domain.StressReport
		err error
	)
	if hasPeriod {
		rep, err = h.svc.StressPeriod(r.Context(), clientIDFrom(r), pid, owner, body.From, body.To)
	} else {
		moves := make([]domain.StressMove, 0, len(body.Moves))
		for _, m := range body.Moves {
			moves = append(moves, domain.StressMove{Asset: m.Asset, MovePct: m.MovePct})
		}
		rep, err = h.svc.Stress(r.Context(), clientIDFrom(r), pid, owner, moves)
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stressDTO(rep))
}

func stressDTO(rep *domain.StressReport) map[string]any {
	if rep == nil {
		return map[string]any{}
	}
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
	out := map[string]any{
		"equity": rep.Equity, "shockedEquity": rep.ShockedEquity, "equityDelta": rep.EquityDelta,
		"totalPnl": rep.TotalPnL, "shockedTotalPnl": rep.ShockedTotalPnL,
		"unrealizedPnl": rep.UnrealizedPnL, "shockedUnrealizedPnl": rep.ShockedUnrealizedPnL,
		"positionsValue": rep.PositionsValue, "shockedPositionsValue": rep.ShockedPositionsValue,
		"marginEquity": rep.MarginEquity, "shockedMarginEquity": rep.ShockedMarginEquity,
		"liquidated": rep.Liquidated, "lines": lines, "note": rep.Note,
	}
	if rep.Replay != nil {
		out["replay"] = stressReplayDTO(rep.Replay)
	}
	return out
}

func stressReplayDTO(rep *domain.StressReplay) map[string]any {
	moves := make([]map[string]any, 0, len(rep.Moves))
	for _, m := range rep.Moves {
		venues := make([]map[string]any, 0, len(m.Venues))
		for _, v := range m.Venues {
			venues = append(venues, map[string]any{
				"exchange": v.Exchange, "symbol": v.Symbol,
				"start": v.Start, "end": v.End,
				"startAt": v.StartAt.UTC().Format(time.RFC3339Nano),
				"endAt":   v.EndAt.UTC().Format(time.RFC3339Nano),
				"movePct": v.MovePct, "weight": v.Weight,
			})
		}
		moves = append(moves, map[string]any{
			"asset": m.Asset, "movePct": m.MovePct, "venues": venues,
		})
	}
	missing := make([]map[string]any, 0, len(rep.Missing))
	for _, g := range rep.Missing {
		missing = append(missing, map[string]any{
			"asset": g.Asset, "exchange": g.Exchange, "symbol": g.Symbol, "reason": g.Reason,
		})
	}
	out := map[string]any{
		"from":     rep.From.UTC().Format(time.RFC3339Nano),
		"to":       rep.To.UTC().Format(time.RFC3339Nano),
		"interval": rep.Interval,
		"moves":    moves,
		"missing":  missing,
	}
	if len(rep.Points) > 0 {
		points := make([]map[string]any, 0, len(rep.Points))
		for _, p := range rep.Points {
			row := map[string]any{
				"at": p.At.UTC().Format(time.RFC3339Nano), "equity": p.Equity,
				"delta": p.Delta, "liquidated": p.Liquidated,
			}
			if p.DeltaPct != nil {
				row["deltaPct"] = *p.DeltaPct
			}
			points = append(points, row)
		}
		out["points"] = points
		out["lowEquity"] = rep.LowEquity
		if !rep.LowAt.IsZero() {
			out["lowAt"] = rep.LowAt.UTC().Format(time.RFC3339Nano)
		}
	}
	return out
}
