package handler

import (
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetLiquidationHunt handles GET /api/v1/market/liquidation-hunt.
func (h *MarketHandler) GetLiquidationHunt(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.GetLiquidationHunt(r.Context(), q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, huntToDTO(got))
}

type huntAssumptionsDTO struct {
	MaintenanceMargin   string  `json:"maintenanceMargin"`
	AccountBlend        float64 `json:"accountBlend"`
	LiquidationTakeRate string  `json:"liquidationTakeRate"`
	SpotTakerFee        string  `json:"spotTakerFee"`
	CascadeFillRate     float64 `json:"cascadeFillRate"`
	LongShortIsAccounts bool    `json:"longShortIsAccounts"`
	LeverageMix         []struct {
		Leverage float64 `json:"leverage"`
		Weight   string  `json:"weight"`
	} `json:"leverageMix"`
}

type huntBandDTO struct {
	Side             string `json:"side"`
	Direction        string `json:"direction"`
	Leverage         string `json:"leverage,omitempty"`
	Price            string `json:"price"`
	MovePct          string `json:"movePct"`
	EstNotional      string `json:"estNotional"`
	ObservedNotional string `json:"observedNotional,omitempty"`
	Source           string `json:"source"`
}

type huntClusterDTO struct {
	Side     string `json:"side"`
	Price    string `json:"price"`
	MovePct  string `json:"movePct"`
	Notional string `json:"notional"`
	Count    int    `json:"count"`
}

type huntWalkDTO struct {
	Side              string `json:"side"`
	TargetPrice       string `json:"targetPrice"`
	Quantity          string `json:"quantity"`
	Notional          string `json:"notional"`
	AveragePrice      string `json:"averagePrice"`
	EndPrice          string `json:"endPrice"`
	Reachable         bool   `json:"reachable"`
	Exhausted         bool   `json:"exhausted"`
	MaxReachablePrice string `json:"maxReachablePrice,omitempty"`
	VisibleNotional   string `json:"visibleNotional"`
}

type huntScenarioDTO struct {
	Direction           string      `json:"direction"`
	Thesis              string      `json:"thesis"`
	Target              huntBandDTO `json:"target"`
	Spot                huntWalkDTO `json:"spot"`
	EstLiquidated       string      `json:"estLiquidated"`
	CascadeExitNotional string      `json:"cascadeExitNotional"`
	BookOnlyPnl         string      `json:"bookOnlyPnl"`
	CascadeInventoryPnl string      `json:"cascadeInventoryPnl"`
	LiquidationTake     string      `json:"liquidationTake"`
	Fees                string      `json:"fees"`
	NetBookOnly         string      `json:"netBookOnly"`
	NetWithCascade      string      `json:"netWithCascade"`
	HouseEdge           string      `json:"houseEdge"`
	Efficiency          string      `json:"efficiency"`
}

type huntVenueDTO struct {
	Exchange           string           `json:"exchange"`
	Symbol             string           `json:"symbol"`
	Price              string           `json:"price"`
	OpenInterestValue  string           `json:"openInterestValue"`
	EstLongNotional    string           `json:"estLongNotional"`
	EstShortNotional   string           `json:"estShortNotional"`
	LongPct            string           `json:"longPct"`
	ShortPct           string           `json:"shortPct"`
	EstLongPct         string           `json:"estLongPct"`
	EstShortPct        string           `json:"estShortPct"`
	FundingRate        string           `json:"fundingRate"`
	FundingPayer       string           `json:"fundingPayer"`
	VisibleBidNotional string           `json:"visibleBidNotional"`
	VisibleAskNotional string           `json:"visibleAskNotional"`
	UpPressure         []huntBandDTO    `json:"upPressure"`
	DownPressure       []huntBandDTO    `json:"downPressure"`
	Observed           []huntClusterDTO `json:"observed"`
	UpHunt             huntScenarioDTO  `json:"upHunt"`
	DownHunt           huntScenarioDTO  `json:"downHunt"`
	Error              string           `json:"error,omitempty"`
}

type huntResponse struct {
	Symbol      string             `json:"symbol"`
	Exchange    string             `json:"exchange"`
	AsOf        time.Time          `json:"asOf"`
	Assumptions huntAssumptionsDTO `json:"assumptions"`
	Venues      []huntVenueDTO     `json:"venues"`
	Note        string             `json:"note"`
}

func huntToDTO(a *domain.HuntReport) huntResponse {
	if a == nil {
		return huntResponse{}
	}
	mix := make([]struct {
		Leverage float64 `json:"leverage"`
		Weight   string  `json:"weight"`
	}, 0, len(a.Assumptions.LeverageMix))
	for _, b := range a.Assumptions.LeverageMix {
		mix = append(mix, struct {
			Leverage float64 `json:"leverage"`
			Weight   string  `json:"weight"`
		}{Leverage: b.Leverage, Weight: formatHistQty(b.Weight)})
	}
	venues := make([]huntVenueDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		venues = append(venues, huntVenueToDTO(v))
	}
	return huntResponse{
		Symbol:   a.Symbol,
		Exchange: a.Exchange,
		AsOf:     a.AsOf.UTC(),
		Assumptions: huntAssumptionsDTO{
			MaintenanceMargin:   formatHistQty(a.Assumptions.MaintenanceMargin),
			AccountBlend:        a.Assumptions.AccountBlend,
			LiquidationTakeRate: formatHistQty(a.Assumptions.LiquidationTakeRate),
			SpotTakerFee:        formatHistQty(a.Assumptions.SpotTakerFee),
			CascadeFillRate:     a.Assumptions.CascadeFillRate,
			LongShortIsAccounts: true,
			LeverageMix:         mix,
		},
		Venues: venues,
		Note:   huntDisclaimer,
	}
}

const huntDisclaimer = "Hypothetical model only — not evidence that any exchange moves the market, and not financial advice. Long/short is account count, not position size. Leverage mix is assumed. USD-M mark uses a multi-venue index, so one spot book may not move mark 1:1. Exchanges usually match users rather than take the other side; liquidationTake is an insurance-fund-like stand-in. bookOnlyPnl is the spot tour if you unwind on the current opposite side (usually a loss). netWithCascade assumes part of estimated liquidations becomes exit flow at the target."

func huntVenueToDTO(v domain.HuntVenueReport) huntVenueDTO {
	return huntVenueDTO{
		Exchange:           string(v.Exchange),
		Symbol:             v.Symbol,
		Price:              formatHistQty(v.Price),
		OpenInterestValue:  formatHistQty(v.OpenInterestValue),
		EstLongNotional:    formatHistQty(v.EstLongNotional),
		EstShortNotional:   formatHistQty(v.EstShortNotional),
		LongPct:            formatHistQty(v.LongShare * 100),
		ShortPct:           formatHistQty(v.ShortShare * 100),
		EstLongPct:         formatHistQty(v.EstLongShare * 100),
		EstShortPct:        formatHistQty(v.EstShortShare * 100),
		FundingRate:        formatHuntRate(v.FundingRate),
		FundingPayer:       v.FundingPayer,
		VisibleBidNotional: formatHistQty(v.VisibleBidNotional),
		VisibleAskNotional: formatHistQty(v.VisibleAskNotional),
		UpPressure:         huntBandsToDTO(v.UpPressure),
		DownPressure:       huntBandsToDTO(v.DownPressure),
		Observed:           huntClustersToDTO(v.Observed),
		UpHunt:             huntScenarioToDTO(v.UpHunt),
		DownHunt:           huntScenarioToDTO(v.DownHunt),
		Error:              v.Error,
	}
}

func huntBandsToDTO(in []domain.HuntBand) []huntBandDTO {
	out := make([]huntBandDTO, 0, len(in))
	for _, b := range in {
		row := huntBandDTO{
			Side:        b.Side,
			Direction:   b.Direction,
			Price:       formatHistQty(b.Price),
			MovePct:     domain.FormatSignedPct(b.MovePct),
			EstNotional: formatHistQty(b.EstNotional),
			Source:      b.Source,
		}
		if b.Leverage > 0 {
			row.Leverage = formatHistQty(b.Leverage)
		}
		if b.ObservedNotional > 0 {
			row.ObservedNotional = formatHistQty(b.ObservedNotional)
		}
		out = append(out, row)
	}
	return out
}

func huntClustersToDTO(in []domain.HuntCluster) []huntClusterDTO {
	out := make([]huntClusterDTO, 0, len(in))
	for _, c := range in {
		out = append(out, huntClusterDTO{
			Side:     c.Side,
			Price:    formatHistQty(c.Price),
			MovePct:  domain.FormatSignedPct(c.MovePct),
			Notional: formatHistQty(c.Notional),
			Count:    c.Count,
		})
	}
	return out
}

func huntScenarioToDTO(s domain.HuntScenario) huntScenarioDTO {
	return huntScenarioDTO{
		Direction:           s.Direction,
		Thesis:              s.Thesis,
		Target:              firstBandDTO(s.Target),
		Spot:                huntWalkToDTO(s.Spot),
		EstLiquidated:       formatHistQty(s.EstLiquidated),
		CascadeExitNotional: formatHistQty(s.CascadeExitNotional),
		BookOnlyPnl:         domain.FormatSignedQty(s.BookOnlyPnl),
		CascadeInventoryPnl: domain.FormatSignedQty(s.CascadeInventoryPnl),
		LiquidationTake:     formatHistQty(s.LiquidationTake),
		Fees:                formatHistQty(s.Fees),
		NetBookOnly:         domain.FormatSignedQty(s.NetBookOnly),
		NetWithCascade:      domain.FormatSignedQty(s.NetWithCascade),
		HouseEdge:           s.HouseEdge,
		Efficiency:          formatHistQty(s.Efficiency),
	}
}

func firstBandDTO(b domain.HuntBand) huntBandDTO {
	if b.Price == 0 && b.EstNotional == 0 && b.Side == "" {
		return huntBandDTO{}
	}
	return huntBandsToDTO([]domain.HuntBand{b})[0]
}

func huntWalkToDTO(w domain.BookReach) huntWalkDTO {
	out := huntWalkDTO{
		Side:            w.Side,
		TargetPrice:     formatHistQty(w.TargetPrice),
		Quantity:        formatHistQty(w.Quantity),
		Notional:        formatHistQty(w.Notional),
		AveragePrice:    formatHistQty(w.AveragePrice),
		EndPrice:        formatHistQty(w.EndPrice),
		Reachable:       w.Reachable,
		Exhausted:       w.Exhausted,
		VisibleNotional: formatHistQty(w.VisibleNotional),
	}
	if w.MaxReachablePrice > 0 {
		out.MaxReachablePrice = formatHistQty(w.MaxReachablePrice)
	}
	return out
}

func formatHuntRate(v float64) string {
	dec, _ := domain.FormatFundingRate(v)
	return dec
}

// GetLiquidationHuntHeatmap handles GET /api/v1/market/liquidation-hunt/heatmap.
func (h *MarketHandler) GetLiquidationHuntHeatmap(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.GetLiquidationHuntHeatmap(r.Context(), q.Get("exchange"), q.Get("symbol"), q.Get("range"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, huntHeatmapToDTO(got))
}

type huntHeatmapGridDTO struct {
	Exchange      string      `json:"exchange"`
	Longs         [][]float64 `json:"longs"`
	Shorts        [][]float64 `json:"shorts"`
	Totals        [][]float64 `json:"totals"`
	MaxIntensity  float64     `json:"maxIntensity"`
	Coverage      float64     `json:"coverage"`
	ColumnsWithOi int         `json:"columnsWithOi"`
}

type huntHeatmapResponse struct {
	Symbol    string             `json:"symbol"`
	Range     string             `json:"range"`
	From      time.Time          `json:"from"`
	To        time.Time          `json:"to"`
	StepSec   int                `json:"stepSec"`
	PriceMin  float64            `json:"priceMin"`
	PriceMax  float64            `json:"priceMax"`
	PriceStep float64            `json:"priceStep"`
	Prices    []float64          `json:"prices"`
	Times     []time.Time        `json:"times"`
	Binance   huntHeatmapGridDTO `json:"binance"`
	Bybit     huntHeatmapGridDTO `json:"bybit"`
	Combined  huntHeatmapGridDTO `json:"combined"`
	Note      string             `json:"note"`
}

func huntHeatmapToDTO(a *domain.HuntHeatmapReport) huntHeatmapResponse {
	if a == nil {
		return huntHeatmapResponse{}
	}
	return huntHeatmapResponse{
		Symbol: a.Symbol, Range: a.Range, From: a.From.UTC(), To: a.To.UTC(),
		StepSec: a.StepSec, PriceMin: a.PriceMin, PriceMax: a.PriceMax, PriceStep: a.PriceStep,
		Prices: a.Prices, Times: a.Times,
		Binance:  huntGridToDTO(a.Binance),
		Bybit:    huntGridToDTO(a.Bybit),
		Combined: huntGridToDTO(a.Combined),
		Note:     a.Note,
	}
}

func huntGridToDTO(g domain.HuntHeatmapGrid) huntHeatmapGridDTO {
	return huntHeatmapGridDTO{
		Exchange: g.Exchange, Longs: g.Longs, Shorts: g.Shorts, Totals: g.Totals,
		MaxIntensity: g.MaxIntensity, Coverage: g.Coverage, ColumnsWithOi: g.ColumnsWithOI,
	}
}
