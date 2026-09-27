package handler

import (
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetSweepRecovery handles GET /api/v1/market/orderbook/recovery.
func (h *MarketHandler) GetSweepRecovery(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.GetSweepRecovery(r.Context(), q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sweepRecoveryToDTO(got))
}

type sweepRecoveryDTO struct {
	Exchange        string    `json:"exchange"`
	Symbol          string    `json:"symbol"`
	Side            string    `json:"side"`
	Level           string    `json:"level"`
	PiercedAt       time.Time `json:"piercedAt"`
	ReclaimedAt     time.Time `json:"reclaimedAt"`
	BeforeNotional  string    `json:"beforeNotional"`
	TroughNotional  string    `json:"troughNotional"`
	BackNotional    string    `json:"backNotional"`
	Recovered       bool      `json:"recovered"`
	RecoverySeconds int       `json:"recoverySeconds"`
	RecoveredPct    float64   `json:"recoveredPct"`
	Summary         string    `json:"summary"`
}

type sweepRecoveryResponse struct {
	Symbol                string             `json:"symbol"`
	Exchange              string             `json:"exchange"`
	AsOf                  time.Time          `json:"asOf"`
	MedianRecoverySeconds int                `json:"medianRecoverySeconds"`
	Recovered             int                `json:"recovered"`
	Pending               int                `json:"pending"`
	Rows                  []sweepRecoveryDTO `json:"rows"`
	Summary               string             `json:"summary"`
	Note                  string             `json:"note"`
}

func sweepRecoveryToDTO(a *domain.SweepRecoveryReport) sweepRecoveryResponse {
	if a == nil {
		return sweepRecoveryResponse{Rows: []sweepRecoveryDTO{}}
	}
	rows := make([]sweepRecoveryDTO, 0, len(a.Rows))
	for _, row := range a.Rows {
		rows = append(rows, sweepRecoveryDTO{
			Exchange: row.Exchange, Symbol: row.Symbol, Side: row.Side, Level: row.Level,
			PiercedAt: row.PiercedAt.UTC(), ReclaimedAt: row.ReclaimedAt.UTC(),
			BeforeNotional: row.BeforeNotional, TroughNotional: row.TroughNotional, BackNotional: row.BackNotional,
			Recovered: row.Recovered, RecoverySeconds: row.RecoverySeconds, RecoveredPct: row.RecoveredPct,
			Summary: row.Summary,
		})
	}
	return sweepRecoveryResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, AsOf: a.AsOf.UTC(),
		MedianRecoverySeconds: a.MedianRecoverySeconds, Recovered: a.Recovered, Pending: a.Pending,
		Rows: rows, Summary: a.Summary, Note: a.Note,
	}
}
