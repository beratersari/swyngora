package handler

import (
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type performanceDTO struct {
	ClientID     string             `json:"clientId"`
	Currency     string             `json:"currency"`
	Period       string             `json:"period"`
	StartAt      string             `json:"startAt"`
	EndAt        string             `json:"endAt"`
	StartEquity  float64            `json:"startEquity"`
	EndEquity    float64            `json:"endEquity"`
	ChangeAmount float64            `json:"changeAmount"`
	ChangePct    *float64           `json:"changePct"`
	Partial      bool               `json:"partial"`
	PointCount   int                `json:"pointCount"`
	Points       []equityPointDTO   `json:"points"`
	Note         string             `json:"note"`
}

type equityPointDTO struct {
	T              string  `json:"t"`
	Equity         float64 `json:"equity"`
	CashBalance    float64 `json:"cashBalance"`
	PositionsValue float64 `json:"positionsValue"`
	MarginEquity   float64 `json:"marginEquity"`
}

func performanceToDTO(p *domain.PortfolioPerformance) performanceDTO {
	pts := make([]equityPointDTO, 0, len(p.Points))
	for _, pt := range p.Points {
		pts = append(pts, equityPointDTO{
			T: pt.Time.UTC().Format(time.RFC3339Nano),
			Equity: pt.Equity, CashBalance: pt.CashBalance,
			PositionsValue: pt.PositionsValue, MarginEquity: pt.MarginEquity,
		})
	}
	return performanceDTO{
		ClientID: p.ClientID, Currency: p.Currency, Period: string(p.Period),
		StartAt: p.StartAt.UTC().Format(time.RFC3339Nano),
		EndAt:   p.EndAt.UTC().Format(time.RFC3339Nano),
		StartEquity: p.StartEquity, EndEquity: p.EndEquity,
		ChangeAmount: p.ChangeAmount, ChangePct: p.ChangePct,
		Partial: p.Partial, PointCount: p.PointCount, Points: pts, Note: p.Note,
	}
}

// GetPerformance handles GET /api/v1/portfolio/performance
func (h *PortfolioHandler) GetPerformance(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	perf, err := h.svc.GetPerformance(r.Context(), clientIDFrom(r), period, portfolioIDFrom(r), ownerClientIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, performanceToDTO(perf))
}
