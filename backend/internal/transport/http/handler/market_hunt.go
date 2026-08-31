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

type huntCascadeStepDTO struct {
	Index                int         `json:"index"`
	Band                 huntBandDTO `json:"band"`
	FromPrice            string      `json:"fromPrice"`
	MovePct              string      `json:"movePct"`
	HopPct               string      `json:"hopPct"`
	ZoneNotional         string      `json:"zoneNotional"`
	CumulativeNotional   string      `json:"cumulativeNotional"`
	Standalone           huntWalkDTO `json:"standalone"`
	Incremental          huntWalkDTO `json:"incremental"`
	Remaining            huntWalkDTO `json:"remaining"`
	PriorCascadeNotional string      `json:"priorCascadeNotional"`
	AssistancePct        string      `json:"assistancePct"`
	Easier               bool        `json:"easier"`
	SelfFueling          bool        `json:"selfFueling"`
	Reachable            bool        `json:"reachable"`
	Note                 string      `json:"note"`
}

type huntCascadePathDTO struct {
	Direction        string               `json:"direction"`
	Steps            []huntCascadeStepDTO `json:"steps"`
	ReachableCount   int                  `json:"reachableCount"`
	EasierCount      int                  `json:"easierCount"`
	SelfFuelingCount int                  `json:"selfFuelingCount"`
	ChainEasier      bool                 `json:"chainEasier"`
	Summary          string               `json:"summary"`
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
	Exchange           string                `json:"exchange"`
	Symbol             string                `json:"symbol"`
	Price              string                `json:"price"`
	OpenInterestValue  string                `json:"openInterestValue"`
	EstLongNotional    string                `json:"estLongNotional"`
	EstShortNotional   string                `json:"estShortNotional"`
	LongPct            string                `json:"longPct"`
	ShortPct           string                `json:"shortPct"`
	EstLongPct         string                `json:"estLongPct"`
	EstShortPct        string                `json:"estShortPct"`
	FundingRate        string                `json:"fundingRate"`
	FundingPayer       string                `json:"fundingPayer"`
	VisibleBidNotional string                `json:"visibleBidNotional"`
	VisibleAskNotional string                `json:"visibleAskNotional"`
	UpPressure         []huntBandDTO         `json:"upPressure"`
	DownPressure       []huntBandDTO         `json:"downPressure"`
	Observed           []huntClusterDTO      `json:"observed"`
	UpHunt             huntScenarioDTO       `json:"upHunt"`
	DownHunt           huntScenarioDTO       `json:"downHunt"`
	UpCascade          huntCascadePathDTO    `json:"upCascade"`
	DownCascade        huntCascadePathDTO    `json:"downCascade"`
	UpScore            huntDirectionScoreDTO `json:"upScore"`
	DownScore          huntDirectionScoreDTO `json:"downScore"`
	Bias               huntBiasDTO           `json:"bias"`
	Coverage           huntCoverageDTO       `json:"coverage"`
	Error              string                `json:"error,omitempty"`
}

type huntFactorDTO struct {
	ID       string  `json:"id"`
	Label    string  `json:"label"`
	Score    float64 `json:"score"`
	Weight   float64 `json:"weight"`
	SharePct float64 `json:"sharePct,omitempty"`
	Effect   float64 `json:"effect"`
	Detail   string  `json:"detail"`
}

type huntDirectionScoreDTO struct {
	Direction string          `json:"direction"`
	Score     float64         `json:"score"`
	Level     string          `json:"level"`
	Coverage  float64         `json:"coverage"`
	Factors   []huntFactorDTO `json:"factors"`
	Reasons   []string        `json:"reasons"`
}

type huntInputStatusDTO struct {
	ID       string  `json:"id"`
	Label    string  `json:"label"`
	Status   string  `json:"status"`
	Weight   float64 `json:"weight"`
	Detail   string  `json:"detail"`
	Have     string  `json:"have,omitempty"`
	Need     string  `json:"need,omitempty"`
	CoverPct float64 `json:"coverPct,omitempty"`
	Age      string  `json:"age,omitempty"`
	Stale    bool    `json:"stale,omitempty"`
}

type huntCoverageDTO struct {
	Score   float64              `json:"score"`
	Level   string               `json:"level"`
	Usable  bool                 `json:"usable"`
	Inputs  []huntInputStatusDTO `json:"inputs"`
	Missing []string             `json:"missing"`
	Weak    []string             `json:"weak"`
	Summary string               `json:"summary"`
}

type huntBiasDTO struct {
	Lean      string          `json:"lean"`
	Margin    float64         `json:"margin"`
	UpScore   float64         `json:"upScore"`
	DownScore float64         `json:"downScore"`
	Summary   string          `json:"summary"`
	Coverage  huntCoverageDTO `json:"coverage"`
	Included  []string        `json:"included,omitempty"`
	Excluded  []string        `json:"excluded,omitempty"`
}

type huntResponse struct {
	Symbol      string             `json:"symbol"`
	Exchange    string             `json:"exchange"`
	AsOf        time.Time          `json:"asOf"`
	Assumptions huntAssumptionsDTO `json:"assumptions"`
	Venues      []huntVenueDTO     `json:"venues"`
	Bias        *huntBiasDTO       `json:"bias,omitempty"`
	Coverage    *huntCoverageDTO   `json:"coverage,omitempty"`
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
		Venues:   venues,
		Bias:     huntBiasToDTO(a.Bias),
		Coverage: huntCoveragePtrToDTO(a.Coverage),
		Note:     huntDisclaimer,
	}
}

const huntDisclaimer = "Hypothetical model only — not evidence that any exchange moves the market, and not financial advice. Long/short is account count, not position size. Leverage mix is assumed. USD-M mark uses a multi-venue index, so one spot book may not move mark 1:1. Exchanges usually match users rather than take the other side; liquidationTake is an insurance-fund-like stand-in. bookOnlyPnl is the spot tour if you unwind on the current opposite side (usually a loss). netWithCascade assumes part of estimated liquidations becomes exit flow at the target. upScore / downScore rank which side looks easier or more likely from zone distance, visible book cost, price+OI trend, crowding/funding, and recent taker/liquidation flow. upCascade / downCascade list zones in price order and whether earlier estimated liquidations cheapen the next hop. coverage says how complete those inputs are; a failed venue is shown but excluded from the combined lean — not a prediction."

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
		UpCascade:          huntCascadeToDTO(v.UpCascade),
		DownCascade:        huntCascadeToDTO(v.DownCascade),
		UpScore:            huntDirectionToDTO(v.UpScore),
		DownScore:          huntDirectionToDTO(v.DownScore),
		Bias:               huntBiasValueToDTO(v.Bias),
		Coverage:           huntCoverageToDTO(v.Coverage),
		Error:              v.Error,
	}
}

func huntDirectionToDTO(s domain.HuntDirectionScore) huntDirectionScoreDTO {
	factors := make([]huntFactorDTO, 0, len(s.Factors))
	for _, f := range s.Factors {
		factors = append(factors, huntFactorDTO{
			ID:       f.ID,
			Label:    f.Label,
			Score:    f.Score,
			Weight:   f.Weight,
			SharePct: f.SharePct,
			Effect:   f.Effect,
			Detail:   f.Detail,
		})
	}
	reasons := s.Reasons
	if reasons == nil {
		reasons = []string{}
	}
	return huntDirectionScoreDTO{
		Direction: s.Direction,
		Score:     s.Score,
		Level:     s.Level,
		Coverage:  s.Coverage,
		Factors:   factors,
		Reasons:   reasons,
	}
}

func huntCoverageToDTO(c domain.HuntCoverage) huntCoverageDTO {
	inputs := make([]huntInputStatusDTO, 0, len(c.Inputs))
	for _, in := range c.Inputs {
		inputs = append(inputs, huntInputStatusDTO{
			ID:       in.ID,
			Label:    in.Label,
			Status:   in.Status,
			Weight:   in.Weight,
			Detail:   in.Detail,
			Have:     in.Have,
			Need:     in.Need,
			CoverPct: in.CoverPct,
			Age:      in.Age,
			Stale:    in.Stale,
		})
	}
	missing := c.Missing
	if missing == nil {
		missing = []string{}
	}
	weak := c.Weak
	if weak == nil {
		weak = []string{}
	}
	return huntCoverageDTO{
		Score:   c.Score,
		Level:   c.Level,
		Usable:  c.Usable,
		Inputs:  inputs,
		Missing: missing,
		Weak:    weak,
		Summary: c.Summary,
	}
}

func huntCoveragePtrToDTO(c *domain.HuntCoverage) *huntCoverageDTO {
	if c == nil {
		return nil
	}
	out := huntCoverageToDTO(*c)
	return &out
}

func huntBiasValueToDTO(b domain.HuntBias) huntBiasDTO {
	return huntBiasDTO{
		Lean:      b.Lean,
		Margin:    b.Margin,
		UpScore:   b.UpScore,
		DownScore: b.DownScore,
		Summary:   b.Summary,
		Coverage:  huntCoverageToDTO(b.Coverage),
		Included:  b.Included,
		Excluded:  b.Excluded,
	}
}

func huntBiasToDTO(b *domain.HuntBias) *huntBiasDTO {
	if b == nil {
		return nil
	}
	out := huntBiasValueToDTO(*b)
	return &out
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

func huntCascadeToDTO(p domain.HuntCascadePath) huntCascadePathDTO {
	steps := make([]huntCascadeStepDTO, 0, len(p.Steps))
	for _, s := range p.Steps {
		steps = append(steps, huntCascadeStepDTO{
			Index:                s.Index,
			Band:                 firstBandDTO(s.Band),
			FromPrice:            formatHistQty(s.FromPrice),
			MovePct:              domain.FormatSignedPct(s.MovePct),
			HopPct:               domain.FormatSignedPct(s.HopPct),
			ZoneNotional:         formatHistQty(s.ZoneNotional),
			CumulativeNotional:   formatHistQty(s.CumulativeNotional),
			Standalone:           huntWalkToDTO(s.Standalone),
			Incremental:          huntWalkToDTO(s.Incremental),
			Remaining:            huntWalkToDTO(s.Remaining),
			PriorCascadeNotional: formatHistQty(s.PriorCascadeNotional),
			AssistancePct:        formatHistQty(s.AssistancePct),
			Easier:               s.Easier,
			SelfFueling:          s.SelfFueling,
			Reachable:            s.Reachable,
			Note:                 s.Note,
		})
	}
	return huntCascadePathDTO{
		Direction:        p.Direction,
		Steps:            steps,
		ReachableCount:   p.ReachableCount,
		EasierCount:      p.EasierCount,
		SelfFuelingCount: p.SelfFuelingCount,
		ChainEasier:      p.ChainEasier,
		Summary:          p.Summary,
	}
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

// GetLiquidationLevels handles GET /api/v1/market/liquidation-levels.
func (h *MarketHandler) GetLiquidationLevels(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.GetLiquidationLevels(r.Context(), q.Get("exchange"), q.Get("symbol"), q.Get("range"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, liquidationLevelsToDTO(got))
}

type liquidationLeverageSliceDTO struct {
	Leverage      int    `json:"leverage"`
	LongNotional  string `json:"longNotional"`
	ShortNotional string `json:"shortNotional"`
}

type liquidationLevelBarDTO struct {
	Price         string                        `json:"price"`
	LongNotional  string                        `json:"longNotional"`
	ShortNotional string                        `json:"shortNotional"`
	TotalNotional string                        `json:"totalNotional"`
	CumLong       string                        `json:"cumLong,omitempty"`
	CumShort      string                        `json:"cumShort,omitempty"`
	CumTotal      string                        `json:"cumTotal,omitempty"`
	ByLeverage    []liquidationLeverageSliceDTO `json:"byLeverage,omitempty"`
}

type liquidationTimeBarDTO struct {
	T             string `json:"t"`
	LongNotional  string `json:"longNotional"`
	ShortNotional string `json:"shortNotional"`
	TotalNotional string `json:"totalNotional"`
	Count         int    `json:"count"`
}

type liquidationLevelsResponse struct {
	Kind       string                   `json:"kind"`
	Symbol     string                   `json:"symbol"`
	Exchange   string                   `json:"exchange"`
	Range      string                   `json:"range"`
	From       string                   `json:"from,omitempty"`
	To         string                   `json:"to,omitempty"`
	LastPrice  string                   `json:"lastPrice,omitempty"`
	LastPrices map[string]string        `json:"lastPrices,omitempty"`
	Levels     []liquidationLevelBarDTO `json:"levels"`
	Bars       []liquidationTimeBarDTO  `json:"bars"`
	Feed       liquidationFeedDTO       `json:"feed"`
	Missing    []string                 `json:"missing,omitempty"`
	Note       string                   `json:"note"`
}

func liquidationLevelsToDTO(a *domain.LiquidationLevelsReport) liquidationLevelsResponse {
	if a == nil {
		return liquidationLevelsResponse{Levels: []liquidationLevelBarDTO{}, Bars: []liquidationTimeBarDTO{}}
	}
	levels := make([]liquidationLevelBarDTO, 0, len(a.Levels))
	for _, lv := range a.Levels {
		slices := make([]liquidationLeverageSliceDTO, 0, len(lv.ByLeverage))
		for _, sl := range lv.ByLeverage {
			slices = append(slices, liquidationLeverageSliceDTO{
				Leverage: sl.Leverage, LongNotional: sl.LongNotional, ShortNotional: sl.ShortNotional,
			})
		}
		levels = append(levels, liquidationLevelBarDTO{
			Price: lv.Price, LongNotional: lv.LongNotional, ShortNotional: lv.ShortNotional,
			TotalNotional: lv.TotalNotional, CumLong: lv.CumLong, CumShort: lv.CumShort,
			CumTotal: lv.CumTotal, ByLeverage: slices,
		})
	}
	bars := make([]liquidationTimeBarDTO, 0, len(a.Bars))
	for _, b := range a.Bars {
		bars = append(bars, liquidationTimeBarDTO{
			T:            b.Time.UTC().Format(time.RFC3339Nano),
			LongNotional: b.LongNotional, ShortNotional: b.ShortNotional,
			TotalNotional: b.TotalNotional, Count: b.Count,
		})
	}
	from, to := "", ""
	if !a.From.IsZero() {
		from = a.From.UTC().Format(time.RFC3339Nano)
	}
	if !a.To.IsZero() {
		to = a.To.UTC().Format(time.RFC3339Nano)
	}
	return liquidationLevelsResponse{
		Kind: a.Kind, Symbol: a.Symbol, Exchange: a.Exchange, Range: a.Range,
		From: from, To: to, LastPrice: a.LastPrice, LastPrices: a.LastPrices,
		Levels: levels, Bars: bars, Feed: feedToDTO(a.Feed), Missing: a.Missing, Note: a.Note,
	}
}

type huntHeatmapGridDTO struct {
	Exchange      string      `json:"exchange"`
	Longs         [][]float64 `json:"longs"`
	Shorts        [][]float64 `json:"shorts"`
	Totals        [][]float64 `json:"totals"`
	MaxIntensity  float64     `json:"maxIntensity"`
	Coverage      float64     `json:"coverage"`
	ColumnsWithOi int         `json:"columnsWithOi"`
	LastPrice     float64     `json:"lastPrice,omitempty"`
	HasData       []bool      `json:"hasData,omitempty"`
}

type huntHeatmapResponse struct {
	Symbol        string               `json:"symbol"`
	Range         string               `json:"range"`
	From          time.Time            `json:"from"`
	To            time.Time            `json:"to"`
	StepSec       int                  `json:"stepSec"`
	PriceMin      float64              `json:"priceMin"`
	PriceMax      float64              `json:"priceMax"`
	PriceStep     float64              `json:"priceStep"`
	Prices        []float64            `json:"prices"`
	Times         []time.Time          `json:"times"`
	Binance       huntHeatmapGridDTO   `json:"binance"`
	Bybit         huntHeatmapGridDTO   `json:"bybit"`
	Combined      huntHeatmapGridDTO   `json:"combined"`
	Review        huntHeatmapReviewDTO `json:"review"`
	MissingVenues []string             `json:"missingVenues,omitempty"`
	Note          string               `json:"note"`
}

type huntHeatmapReviewHorizonDTO struct {
	Horizon            string  `json:"horizon"`
	Signals            int     `json:"signals"`
	Validated          int     `json:"validated"`
	Missing            int     `json:"missing"`
	PriceReady         int     `json:"priceReady"`
	PriceMissing       int     `json:"priceMissing"`
	LiqReady           int     `json:"liqReady"`
	LiqMissing         int     `json:"liqMissing"`
	Pending            int     `json:"pending"`
	Coverage           float64 `json:"coverage"`
	Hits               int     `json:"hits"`
	FalseSignals       int     `json:"falseSignals"`
	HitRate            float64 `json:"hitRate"`
	AvgTimeToHitSec    float64 `json:"avgTimeToHitSec"`
	MedianTimeToHitSec float64 `json:"medianTimeToHitSec"`
	LiqIncreased       int     `json:"liqIncreased"`
	LiqIncreaseRate    float64 `json:"liqIncreaseRate"`
	AvgLiqBefore       float64 `json:"avgLiqBefore"`
	AvgLiqAfter        float64 `json:"avgLiqAfter"`
}

type huntHeatmapReviewSignalHorizonDTO struct {
	Horizon         string  `json:"horizon"`
	Status          string  `json:"status"`
	Validated       bool    `json:"validated"`
	Hit             bool    `json:"hit"`
	Pending         bool    `json:"pending"`
	PriceReady      bool    `json:"priceReady"`
	LiqReady        bool    `json:"liqReady"`
	TimeToHitSec    float64 `json:"timeToHitSec,omitempty"`
	HorizonSec      float64 `json:"horizonSec"`
	PriceCoveredSec float64 `json:"priceCoveredSec"`
	PriceBars       int     `json:"priceBars"`
	LiqBefore       float64 `json:"liqBefore"`
	LiqAfter        float64 `json:"liqAfter"`
	LiqIncreased    bool    `json:"liqIncreased"`
	Gap             string  `json:"gap,omitempty"`
}

type huntHeatmapReviewSignalDTO struct {
	Time      time.Time                           `json:"time"`
	PriceLo   float64                             `json:"priceLo"`
	PriceHi   float64                             `json:"priceHi"`
	Intensity float64                             `json:"intensity"`
	Side      string                              `json:"side,omitempty"`
	Horizons  []huntHeatmapReviewSignalHorizonDTO `json:"horizons"`
}

type huntHeatmapReviewVenueDTO struct {
	Exchange string                        `json:"exchange"`
	Horizons []huntHeatmapReviewHorizonDTO `json:"horizons"`
	Signals  []huntHeatmapReviewSignalDTO  `json:"signals"`
}

type huntHeatmapReviewDTO struct {
	HotFrac  float64                   `json:"hotFrac"`
	Binance  huntHeatmapReviewVenueDTO `json:"binance"`
	Bybit    huntHeatmapReviewVenueDTO `json:"bybit"`
	Combined huntHeatmapReviewVenueDTO `json:"combined"`
	Note     string                    `json:"note"`
}

func huntHeatmapToDTO(a *domain.HuntHeatmapReport) huntHeatmapResponse {
	if a == nil {
		return huntHeatmapResponse{}
	}
	return huntHeatmapResponse{
		Symbol: a.Symbol, Range: a.Range, From: a.From.UTC(), To: a.To.UTC(),
		StepSec: a.StepSec, PriceMin: a.PriceMin, PriceMax: a.PriceMax, PriceStep: a.PriceStep,
		Prices: a.Prices, Times: a.Times,
		Binance:       huntGridToDTO(a.Binance),
		Bybit:         huntGridToDTO(a.Bybit),
		Combined:      huntGridToDTO(a.Combined),
		Review:        huntReviewToDTO(a.Review),
		MissingVenues: a.MissingVenues,
		Note:          a.Note,
	}
}

func huntReviewToDTO(r domain.HuntHeatmapReview) huntHeatmapReviewDTO {
	return huntHeatmapReviewDTO{
		HotFrac:  r.HotFrac,
		Binance:  huntReviewVenueToDTO(r.Binance),
		Bybit:    huntReviewVenueToDTO(r.Bybit),
		Combined: huntReviewVenueToDTO(r.Combined),
		Note:     r.Note,
	}
}

func huntReviewVenueToDTO(v domain.HuntHeatmapReviewVenue) huntHeatmapReviewVenueDTO {
	hs := make([]huntHeatmapReviewHorizonDTO, 0, len(v.Horizons))
	for _, h := range v.Horizons {
		hs = append(hs, huntHeatmapReviewHorizonDTO{
			Horizon: h.Horizon, Signals: h.Signals, Validated: h.Validated, Missing: h.Missing,
			PriceReady: h.PriceReady, PriceMissing: h.PriceMissing, LiqReady: h.LiqReady,
			LiqMissing: h.LiqMissing, Pending: h.Pending, Coverage: h.Coverage,
			Hits: h.Hits, FalseSignals: h.FalseSignals, HitRate: h.HitRate,
			AvgTimeToHitSec: h.AvgTimeToHitSec, MedianTimeToHitSec: h.MedianTimeToHitSec,
			LiqIncreased: h.LiqIncreased, LiqIncreaseRate: h.LiqIncreaseRate,
			AvgLiqBefore: h.AvgLiqBefore, AvgLiqAfter: h.AvgLiqAfter,
		})
	}
	sigs := make([]huntHeatmapReviewSignalDTO, 0, len(v.Signals))
	for _, s := range v.Signals {
		rows := make([]huntHeatmapReviewSignalHorizonDTO, 0, len(s.Horizons))
		for _, h := range s.Horizons {
			rows = append(rows, huntHeatmapReviewSignalHorizonDTO{
				Horizon: h.Horizon, Status: h.Status, Validated: h.Validated, Hit: h.Hit,
				Pending: h.Pending, PriceReady: h.PriceReady, LiqReady: h.LiqReady,
				TimeToHitSec: h.TimeToHitSec, HorizonSec: h.HorizonSec,
				PriceCoveredSec: h.PriceCoveredSec, PriceBars: h.PriceBars,
				LiqBefore: h.LiqBefore, LiqAfter: h.LiqAfter, LiqIncreased: h.LiqIncreased,
				Gap: h.Gap,
			})
		}
		sigs = append(sigs, huntHeatmapReviewSignalDTO{
			Time: s.Time.UTC(), PriceLo: s.PriceLo, PriceHi: s.PriceHi,
			Intensity: s.Intensity, Side: s.Side, Horizons: rows,
		})
	}
	return huntHeatmapReviewVenueDTO{Exchange: v.Exchange, Horizons: hs, Signals: sigs}
}

func huntGridToDTO(g domain.HuntHeatmapGrid) huntHeatmapGridDTO {
	return huntHeatmapGridDTO{
		Exchange: g.Exchange, Longs: g.Longs, Shorts: g.Shorts, Totals: g.Totals,
		MaxIntensity: g.MaxIntensity, Coverage: g.Coverage, ColumnsWithOi: g.ColumnsWithOI,
		LastPrice: g.LastPrice, HasData: g.HasData,
	}
}
