package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/swing"
)

// SwingHandler serves swing-setup analysis.
type SwingHandler struct {
	svc *swing.Service
}

// NewSwingHandler constructs the handler.
func NewSwingHandler(svc *swing.Service) *SwingHandler {
	return &SwingHandler{svc: svc}
}

type swingPatternDTO struct {
	Name        string  `json:"name"`
	Score       float64 `json:"score"`
	Description string  `json:"description"`
	Timeframe   string  `json:"timeframe"`
	Fresh       bool    `json:"fresh"`
}

type swingLevelsDTO struct {
	Entry      float64 `json:"entry"`
	StopLoss   float64 `json:"stopLoss"`
	TakeProfit float64 `json:"takeProfit"`
	RiskPct    float64 `json:"riskPct"`
	RewardPct  float64 `json:"rewardPct"`
	Rr         float64 `json:"rr"`
	ATR        float64 `json:"atr"`
}

type swingDecisionDTO struct {
	Exchange   string             `json:"exchange"`
	Symbol     string             `json:"symbol"`
	Interval   string             `json:"interval"`
	Accepted   bool               `json:"accepted"`
	Stage      string             `json:"stage"`
	SetupType  string             `json:"setupType"`
	SwingScore float64            `json:"swingScore"`
	Grade      string             `json:"grade"`
	Fresh      bool               `json:"fresh"`
	BtcRegime  string             `json:"btcRegime"`
	Side       string             `json:"side"`
	Price      float64            `json:"price"`
	Adx4h      *float64           `json:"adx4h,omitempty"`
	Adx1d      *float64           `json:"adx1d,omitempty"`
	Rsi        *float64           `json:"rsi,omitempty"`
	Ema4h      string             `json:"ema4h"`
	Ema1d      string             `json:"ema1d"`
	Patterns   []swingPatternDTO  `json:"patterns"`
	Levels     *swingLevelsDTO    `json:"levels,omitempty"`
	Reasons    []string           `json:"reasons,omitempty"`
	Note       string             `json:"note"`
	BarTime    string             `json:"barTime,omitempty"`
}

func swingToDTO(d *domain.SwingDecision) swingDecisionDTO {
	if d == nil {
		return swingDecisionDTO{}
	}
	pats := make([]swingPatternDTO, 0, len(d.Patterns))
	for _, p := range d.Patterns {
		pats = append(pats, swingPatternDTO{
			Name: p.Name, Score: p.Score, Description: p.Description, Timeframe: p.Timeframe, Fresh: p.Fresh,
		})
	}
	out := swingDecisionDTO{
		Exchange: string(d.Exchange), Symbol: d.Symbol, Interval: d.Interval,
		Accepted: d.Accepted, Stage: d.Stage, SetupType: d.SetupType, SwingScore: d.SwingScore,
		Grade: d.Grade, Fresh: d.Fresh, BtcRegime: d.BTCRegime, Side: d.Side, Price: d.Price,
		Adx4h: d.ADX4h, Adx1d: d.ADX1d, Rsi: d.RSI, Ema4h: d.EMA4h, Ema1d: d.EMA1d,
		Patterns: pats, Reasons: d.Reasons, Note: d.Note,
	}
	if !d.BarTime.IsZero() {
		out.BarTime = d.BarTime.UTC().Format(time.RFC3339Nano)
	}
	if d.Levels != nil {
		out.Levels = &swingLevelsDTO{
			Entry: d.Levels.Entry, StopLoss: d.Levels.StopLoss, TakeProfit: d.Levels.TakeProfit,
			RiskPct: d.Levels.RiskPct, RewardPct: d.Levels.RewardPct, Rr: d.Levels.RR, ATR: d.Levels.ATR,
		}
	}
	return out
}

// Analyze handles GET /api/v1/market/swing
func (h *SwingHandler) Analyze(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, fmt.Errorf("%w: swing service not configured", domain.ErrUpstream))
		return
	}
	q := r.URL.Query()
	dec, err := h.svc.Analyze(r.Context(), q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, swingToDTO(dec))
}

// ListSetups handles GET /api/v1/swing/setups (watchlist scan).
func (h *SwingHandler) ListSetups(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, fmt.Errorf("%w: swing service not configured", domain.ErrUpstream))
		return
	}
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	list, err := h.svc.ScanWatchlist(r.Context(), clientIDFrom(r), q.Get("exchange"), limit)
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]swingDecisionDTO, 0, len(list))
	accepted := 0
	for i := range list {
		items = append(items, swingToDTO(&list[i]))
		if list[i].Accepted {
			accepted++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": items, "count": len(items), "accepted": accepted,
		"note": "Informational only — not financial advice.",
	})
}
