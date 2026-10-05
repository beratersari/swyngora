package handler

import (
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetCandleFootprint handles GET /api/v1/market/flow-at-prices/candles.
func (h *MarketHandler) GetCandleFootprint(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.svc == nil {
		writeError(w, domain.ErrUpstream)
		return
	}
	q := r.URL.Query()
	var startPtr, endPtr *time.Time
	if raw := q.Get("startTime"); raw != "" {
		t, err := parseTimeParam(raw)
		if err != nil {
			writeError(w, err)
			return
		}
		startPtr = &t
	}
	if raw := q.Get("endTime"); raw != "" {
		t, err := parseTimeParam(raw)
		if err != nil {
			writeError(w, err)
			return
		}
		endPtr = &t
	}
	got, err := h.svc.GetCandleFootprint(r.Context(), q.Get("exchange"), q.Get("symbol"), q.Get("interval"), startPtr, endPtr)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, candleFootprintToDTO(got))
}

type candleFootprintRowDTO struct {
	Price      string `json:"price"`
	High       string `json:"high"`
	BuyVolume  string `json:"buyVolume"`
	SellVolume string `json:"sellVolume"`
	Delta      string `json:"delta"`
	BuyCount   int    `json:"buyCount"`
	SellCount  int    `json:"sellCount"`
	Imbalance  string `json:"imbalance,omitempty"`
	IsPoc      bool   `json:"isPoc"`
}

type candleFootprintCandleDTO struct {
	Time       time.Time               `json:"time"`
	BuyVolume  string                  `json:"buyVolume"`
	SellVolume string                  `json:"sellVolume"`
	Delta      string                  `json:"delta"`
	BuyCount   int                     `json:"buyCount"`
	SellCount  int                     `json:"sellCount"`
	POC        string                  `json:"poc,omitempty"`
	Rows       []candleFootprintRowDTO `json:"rows"`
}

type candleFootprintBookDTO struct {
	Exchange string                     `json:"exchange"`
	Candles  []candleFootprintCandleDTO `json:"candles"`
}

type candleFootprintResponse struct {
	Symbol    string                   `json:"symbol"`
	Exchange  string                   `json:"exchange"`
	Interval  string                   `json:"interval"`
	StepSec   int                      `json:"stepSec"`
	From      time.Time                `json:"from"`
	To        time.Time                `json:"to"`
	AsOf      time.Time                `json:"asOf"`
	Definite  bool                     `json:"definite"`
	Truncated bool                     `json:"truncated"`
	Missing   []string                 `json:"missing"`
	Venues    []candleFootprintBookDTO `json:"venues"`
	Combined  *candleFootprintBookDTO  `json:"combined,omitempty"`
	Summary   string                   `json:"summary"`
	Note      string                   `json:"note"`
}

func candleFootprintToDTO(a *domain.CandleFootprintReport) candleFootprintResponse {
	if a == nil {
		return candleFootprintResponse{Missing: []string{}, Venues: []candleFootprintBookDTO{}}
	}
	missing := make([]string, 0, len(a.Missing))
	for _, ex := range a.Missing {
		missing = append(missing, string(ex))
	}
	venues := make([]candleFootprintBookDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		venues = append(venues, candleFootprintBookToDTO(v))
	}
	out := candleFootprintResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, Interval: a.Interval,
		StepSec: int(a.Step.Seconds()),
		From:    a.From.UTC(), To: a.To.UTC(), AsOf: a.AsOf.UTC(),
		Definite: a.Definite, Truncated: a.Truncated, Missing: missing,
		Venues: venues, Summary: a.Summary, Note: a.Note,
	}
	if a.Combined != nil {
		c := candleFootprintBookToDTO(*a.Combined)
		out.Combined = &c
	}
	return out
}

func candleFootprintBookToDTO(v domain.CandleFootprintBook) candleFootprintBookDTO {
	candles := make([]candleFootprintCandleDTO, 0, len(v.Candles))
	for _, c := range v.Candles {
		rows := make([]candleFootprintRowDTO, 0, len(c.Rows))
		for _, row := range c.Rows {
			rows = append(rows, candleFootprintRowDTO{
				Price: formatHistQty(row.Price), High: formatHistQty(row.High),
				BuyVolume: formatHistQty(row.Buy), SellVolume: formatHistQty(row.Sell),
				Delta:    domain.FormatSignedQty(row.Buy - row.Sell),
				BuyCount: row.BuyN, SellCount: row.SellN,
				Imbalance: row.Imbalance, IsPoc: row.IsPoc,
			})
		}
		item := candleFootprintCandleDTO{
			Time: c.Open.UTC(), BuyVolume: formatHistQty(c.Buy), SellVolume: formatHistQty(c.Sell),
			Delta:    domain.FormatSignedQty(c.Buy - c.Sell),
			BuyCount: c.BuyN, SellCount: c.SellN, Rows: rows,
		}
		if c.POC > 0 {
			item.POC = formatHistQty(c.POC)
		}
		candles = append(candles, item)
	}
	return candleFootprintBookDTO{Exchange: string(v.Exchange), Candles: candles}
}
