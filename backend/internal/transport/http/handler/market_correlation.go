package handler

import (
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetCorrelation handles GET /api/v1/market/correlation.
func (h *MarketHandler) GetCorrelation(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.GetCorrelation(r.Context(), q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, corrToDTO(got))
}

type corrVsDTO struct {
	Reference  string `json:"reference"`
	Corr       string `json:"corr"`
	Beta       string `json:"beta"`
	SameDirPct string `json:"sameDirPct"`
	Relation   string `json:"relation"`
	Timing     string `json:"timing"`
	LagBars    int    `json:"lagBars,omitempty"`
	LagMinutes int    `json:"lagMinutes,omitempty"`
	Samples    int    `json:"samples"`
	Complete   bool   `json:"complete"`
	Self       bool   `json:"self,omitempty"`
	Error      string `json:"error,omitempty"`
}

type corrWindowDTO struct {
	Window       string    `json:"window"`
	Interval     string    `json:"interval"`
	Bars         int       `json:"bars"`
	AssetMovePct string    `json:"assetMovePct"`
	BtcMovePct   string    `json:"btcMovePct"`
	EthMovePct   string    `json:"ethMovePct"`
	BTC          corrVsDTO `json:"btc"`
	ETH          corrVsDTO `json:"eth"`
	Summary      string    `json:"summary"`
}

type corrResponse struct {
	Symbol   string          `json:"symbol"`
	Exchange string          `json:"exchange"`
	AsOf     time.Time       `json:"asOf"`
	Windows  []corrWindowDTO `json:"windows"`
	Summary  string          `json:"summary"`
	Note     string          `json:"note"`
}

func corrToDTO(a *domain.CorrelationReport) corrResponse {
	if a == nil {
		return corrResponse{}
	}
	wins := make([]corrWindowDTO, 0, len(a.Windows))
	for _, w := range a.Windows {
		wins = append(wins, corrWindowDTO{
			Window: w.Window, Interval: w.Interval, Bars: w.Bars,
			AssetMovePct: domain.FormatSignedPct(w.AssetMovePct),
			BtcMovePct:   domain.FormatSignedPct(w.BTCMovePct),
			EthMovePct:   domain.FormatSignedPct(w.ETHMovePct),
			BTC:          corrVsToDTO(w.BTC),
			ETH:          corrVsToDTO(w.ETH),
			Summary:      w.Summary,
		})
	}
	return corrResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, AsOf: a.AsOf.UTC(),
		Windows: wins, Summary: a.Summary, Note: a.Note,
	}
}

func corrVsToDTO(v domain.CorrelationVs) corrVsDTO {
	out := corrVsDTO{
		Reference: v.Reference, Relation: v.Relation, Timing: v.Timing,
		LagBars: v.LagBars, LagMinutes: v.LagMinutes, Samples: v.Samples,
		Complete: v.Complete, Self: v.Self, Error: v.Error,
	}
	if v.Complete || v.Self {
		out.Corr = domain.FormatSignedPct(v.Corr)
		out.Beta = domain.FormatSignedPct(v.Beta)
		out.SameDirPct = domain.FormatSignedPct(v.SameDirPct)
	}
	return out
}
