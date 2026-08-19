package handler

import (
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetVolatility handles GET /api/v1/market/volatility.
func (h *MarketHandler) GetVolatility(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.GetVolatility(r.Context(), q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, volToDTO(got))
}

type volMeasureDTO struct {
	NetPct      string `json:"netPct"`
	RangePct    string `json:"rangePct"`
	RealizedPct string `json:"realizedPct"`
	Bars        int    `json:"bars"`
	Complete    bool   `json:"complete"`
}

type volWindowDTO struct {
	Window     string        `json:"window"`
	Interval   string        `json:"interval"`
	Coin       volMeasureDTO `json:"coin"`
	Previous   volMeasureDTO `json:"previous"`
	TypicalPct string        `json:"typicalPct,omitempty"`
	VsNormal   string        `json:"vsNormal"`
	Trend      string        `json:"trend"`
	BTC        volMeasureDTO `json:"btc"`
	ETH        volMeasureDTO `json:"eth"`
	VsBTC      string        `json:"vsBtc"`
	VsETH      string        `json:"vsEth"`
	VsMarket   string        `json:"vsMarket"`
	Summary    string        `json:"summary"`
}

type volResponse struct {
	Symbol   string         `json:"symbol"`
	Exchange string         `json:"exchange"`
	AsOf     time.Time      `json:"asOf"`
	Windows  []volWindowDTO `json:"windows"`
	Summary  string         `json:"summary"`
	Note     string         `json:"note"`
}

func volToDTO(a *domain.VolatilityReport) volResponse {
	if a == nil {
		return volResponse{}
	}
	wins := make([]volWindowDTO, 0, len(a.Windows))
	for _, w := range a.Windows {
		row := volWindowDTO{
			Window: w.Window, Interval: w.Interval,
			Coin: volMeasureToDTO(w.Coin), Previous: volMeasureToDTO(w.Previous),
			VsNormal: w.VsNormal, Trend: w.Trend,
			BTC: volMeasureToDTO(w.BTC), ETH: volMeasureToDTO(w.ETH),
			VsBTC: w.VsBTC, VsETH: w.VsETH, VsMarket: w.VsMarket, Summary: w.Summary,
		}
		if w.TypicalPct > 0 {
			row.TypicalPct = domain.FormatSignedPct(w.TypicalPct)
		}
		wins = append(wins, row)
	}
	return volResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, AsOf: a.AsOf.UTC(),
		Windows: wins, Summary: a.Summary, Note: a.Note,
	}
}

func volMeasureToDTO(m domain.VolMeasure) volMeasureDTO {
	out := volMeasureDTO{Bars: m.Bars, Complete: m.Complete}
	if m.Complete {
		out.NetPct = domain.FormatSignedPct(m.NetPct)
		out.RangePct = domain.FormatSignedPct(m.RangePct)
		out.RealizedPct = domain.FormatSignedPct(m.RealizedPct)
	}
	return out
}
