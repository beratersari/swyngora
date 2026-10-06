package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetDerivativesBoard handles GET /api/v1/market/derivatives-board.
func (h *MarketHandler) GetDerivativesBoard(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 0
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, fmt.Errorf("%w: limit must be an integer", domain.ErrInvalidArgument))
			return
		}
		limit = n
	}
	got, err := h.svc.GetDerivativesBoard(r.Context(), q.Get("exchange"), limit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, derivativesBoardToDTO(got))
}

type derivativesVenueDTO struct {
	Exchange         string `json:"exchange"`
	OIValue          string `json:"oiValue,omitempty"`
	OIStatus         string `json:"oiStatus"`
	FundingRate      string `json:"fundingRate,omitempty"`
	FundingRatePct   string `json:"fundingRatePct,omitempty"`
	FundingPayer     string `json:"fundingPayer,omitempty"`
	FundingStatus    string `json:"fundingStatus"`
	MarkPrice        string `json:"markPrice,omitempty"`
	LiqLong          string `json:"liqLong,omitempty"`
	LiqShort         string `json:"liqShort,omitempty"`
	LiqStatus        string `json:"liqStatus"`
	LongPct          string `json:"longPct,omitempty"`
	LongStatus       string `json:"longStatus"`
	TakerBuyPct      string `json:"takerBuyPct,omitempty"`
	TakerStatus      string `json:"takerStatus"`
	GapPct           string `json:"gapPct,omitempty"`
	GapStatus        string `json:"gapStatus,omitempty"`
	CVDDay           string `json:"cvdDay,omitempty"`
	CVDStatus        string `json:"cvdStatus,omitempty"`
	CVDSplit         string `json:"cvdSplit,omitempty"`
	CVDSplitDuration string `json:"cvdSplitDuration,omitempty"`
	CVDSplitSeconds  string `json:"cvdSplitSeconds,omitempty"`
	CVDSplitStatus   string `json:"cvdSplitStatus,omitempty"`
	PrintChangePct   string `json:"printChangePct,omitempty"`
	PrintStatus      string `json:"printStatus,omitempty"`
	OrderRatio       string `json:"orderRatio,omitempty"`
	OrderLean        string `json:"orderLean,omitempty"`
	OrderBuy         int    `json:"orderBuy,omitempty"`
	OrderSell        int    `json:"orderSell,omitempty"`
	OrderStatus      string `json:"orderStatus,omitempty"`
	UnusualMetric    string `json:"unusualMetric,omitempty"`
	UnusualBand      string `json:"unusualBand,omitempty"`
	UnusualDistance  string `json:"unusualDistance,omitempty"`
	LiqGrade         string `json:"liqGrade,omitempty"`
	LiqHeatStatus    string `json:"liqHeatStatus,omitempty"`
	BidPct           string `json:"bidPct,omitempty"`
	BookStatus       string `json:"bookStatus,omitempty"`
}

type derivativesCoinDTO struct {
	Symbol               string                `json:"symbol"`
	Base                 string                `json:"base"`
	Price                string                `json:"price,omitempty"`
	PriceExchange        string                `json:"priceExchange,omitempty"`
	FundingRate          string                `json:"fundingRate,omitempty"`
	FundingRatePct       string                `json:"fundingRatePct,omitempty"`
	FundingPayer         string                `json:"fundingPayer,omitempty"`
	FundingWeight        string                `json:"fundingWeight,omitempty"`
	FundingDefinite      bool                  `json:"fundingDefinite"`
	OpenInterest         string                `json:"openInterest,omitempty"`
	OpenInterestDefinite bool                  `json:"openInterestDefinite"`
	Change1h             string                `json:"change1h,omitempty"`
	Change1hPct          string                `json:"change1hPct,omitempty"`
	Change1hDefinite     bool                  `json:"change1hDefinite"`
	Change24h            string                `json:"change24h,omitempty"`
	Change24hPct         string                `json:"change24hPct,omitempty"`
	Change24hDefinite    bool                  `json:"change24hDefinite"`
	LiqLong              string                `json:"liqLong,omitempty"`
	LiqShort             string                `json:"liqShort,omitempty"`
	LiqDefinite          bool                  `json:"liqDefinite"`
	LongPct              string                `json:"longPct,omitempty"`
	LongDefinite         bool                  `json:"longDefinite"`
	TakerBuyPct          string                `json:"takerBuyPct,omitempty"`
	TakerDefinite        bool                  `json:"takerDefinite"`
	GapPct               string                `json:"gapPct,omitempty"`
	GapDefinite          bool                  `json:"gapDefinite"`
	GapScope             string                `json:"gapScope,omitempty"`
	CVDDay               string                `json:"cvdDay,omitempty"`
	CVDDayDefinite       bool                  `json:"cvdDayDefinite"`
	CVDSplit             string                `json:"cvdSplit,omitempty"`
	CVDSplitDuration     string                `json:"cvdSplitDuration,omitempty"`
	CVDSplitSeconds      string                `json:"cvdSplitSeconds,omitempty"`
	CVDSplitDefinite     bool                  `json:"cvdSplitDefinite"`
	PrintChangePct       string                `json:"printChangePct,omitempty"`
	PrintRead            string                `json:"printRead,omitempty"`
	PrintDefinite        bool                  `json:"printDefinite"`
	OrderRatio           string                `json:"orderRatio,omitempty"`
	OrderLean            string                `json:"orderLean,omitempty"`
	OrderBuy             int                   `json:"orderBuy,omitempty"`
	OrderSell            int                   `json:"orderSell,omitempty"`
	OrderDefinite        bool                  `json:"orderDefinite"`
	UnusualMetric        string                `json:"unusualMetric,omitempty"`
	UnusualBand          string                `json:"unusualBand,omitempty"`
	UnusualDistance      string                `json:"unusualDistance,omitempty"`
	UnusualPercentile    string                `json:"unusualPercentile,omitempty"`
	UnusualVenue         string                `json:"unusualVenue,omitempty"`
	UnusualDefinite      bool                  `json:"unusualDefinite"`
	LiqHeat              string                `json:"liqHeat,omitempty"`
	LiqHeatRatio         string                `json:"liqHeatRatio,omitempty"`
	LiqHeatVenue         string                `json:"liqHeatVenue,omitempty"`
	LiqHeatDefinite      bool                  `json:"liqHeatDefinite"`
	BidPct               string                `json:"bidPct,omitempty"`
	BidDefinite          bool                  `json:"bidDefinite"`
	Venues               []derivativesVenueDTO `json:"venues"`
	Coverage             marketCoverageDTO     `json:"coverage"`
	Recorded             []recordedSeriesDTO   `json:"recorded,omitempty"`
}

type recordedSeriesDTO struct {
	Kind     string    `json:"kind"`
	Exchange string    `json:"exchange"`
	Since    time.Time `json:"since"`
	SpanSec  int       `json:"spanSec"`
}

type derivativesBoardResponse struct {
	Exchange string               `json:"exchange"`
	Universe int                  `json:"universe"`
	AsOf     time.Time            `json:"asOf"`
	Coins    []derivativesCoinDTO `json:"coins"`
	Coverage marketCoverageDTO    `json:"coverage"`
	Definite bool                 `json:"definite"`
	Summary  string               `json:"summary"`
	Note     string               `json:"note"`
}

func recordedToDTO(rows []domain.RecordedSeries) []recordedSeriesDTO {
	if len(rows) == 0 {
		return nil
	}
	out := make([]recordedSeriesDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, recordedSeriesDTO{
			Kind: row.Kind, Exchange: string(row.Exchange), Since: row.Since.UTC(), SpanSec: row.SpanSec,
		})
	}
	return out
}

func derivativesBoardToDTO(got *domain.DerivativesBoard) derivativesBoardResponse {
	if got == nil {
		return derivativesBoardResponse{Coins: []derivativesCoinDTO{}, Coverage: coverageToDTO(domain.MarketCoverage{})}
	}
	coins := make([]derivativesCoinDTO, 0, len(got.Coins))
	for _, c := range got.Coins {
		venues := make([]derivativesVenueDTO, 0, len(c.Venues))
		for _, v := range c.Venues {
			venues = append(venues, derivativesVenueDTO{
				Exchange: v.Exchange, OIValue: v.OIValue, OIStatus: v.OIStatus,
				FundingRate: v.FundingRate, FundingRatePct: v.FundingRatePct, FundingPayer: v.FundingPayer,
				FundingStatus: v.FundingStatus, MarkPrice: v.MarkPrice,
				LiqLong: v.LiqLong, LiqShort: v.LiqShort, LiqStatus: v.LiqStatus,
				LongPct: v.LongPct, LongStatus: v.LongStatus,
				TakerBuyPct: v.TakerBuyPct, TakerStatus: v.TakerStatus,
				GapPct: v.GapPct, GapStatus: v.GapStatus,
				CVDDay: v.CVDDay, CVDStatus: v.CVDStatus,
				CVDSplit: v.CVDSplit, CVDSplitDuration: v.CVDSplitDuration, CVDSplitSeconds: v.CVDSplitSeconds, CVDSplitStatus: v.CVDSplitStatus,
				PrintChangePct: v.PrintChangePct, PrintStatus: v.PrintStatus,
				OrderRatio: v.OrderRatio, OrderLean: v.OrderLean, OrderBuy: v.OrderBuy, OrderSell: v.OrderSell, OrderStatus: v.OrderStatus,
				UnusualMetric: v.UnusualMetric, UnusualBand: v.UnusualBand, UnusualDistance: v.UnusualDistance,
				LiqGrade: v.LiqGrade, LiqHeatStatus: v.LiqHeatStatus,
				BidPct: v.BidPct, BookStatus: v.BookStatus,
			})
		}
		coins = append(coins, derivativesCoinDTO{
			Symbol: c.Symbol, Base: c.Base, Price: c.Price, PriceExchange: c.PriceExchange,
			FundingRate: c.FundingRate, FundingRatePct: c.FundingRatePct, FundingPayer: c.FundingPayer,
			FundingWeight: c.FundingWeight, FundingDefinite: c.FundingDefinite,
			OpenInterest: c.OpenInterest, OpenInterestDefinite: c.OpenInterestDefinite,
			Change1h: c.Change1h, Change1hPct: c.Change1hPct, Change1hDefinite: c.Change1hDefinite,
			Change24h: c.Change24h, Change24hPct: c.Change24hPct, Change24hDefinite: c.Change24hDefinite,
			LiqLong: c.LiqLong, LiqShort: c.LiqShort, LiqDefinite: c.LiqDefinite,
			LongPct: c.LongPct, LongDefinite: c.LongDefinite,
			TakerBuyPct: c.TakerBuyPct, TakerDefinite: c.TakerDefinite,
			GapPct: c.GapPct, GapDefinite: c.GapDefinite, GapScope: c.GapScope,
			CVDDay: c.CVDDay, CVDDayDefinite: c.CVDDayDefinite,
			CVDSplit: c.CVDSplit, CVDSplitDuration: c.CVDSplitDuration, CVDSplitSeconds: c.CVDSplitSeconds, CVDSplitDefinite: c.CVDSplitDefinite,
			PrintChangePct: c.PrintChangePct, PrintRead: c.PrintRead, PrintDefinite: c.PrintDefinite,
			OrderRatio: c.OrderRatio, OrderLean: c.OrderLean, OrderBuy: c.OrderBuy, OrderSell: c.OrderSell, OrderDefinite: c.OrderDefinite,
			UnusualMetric: c.UnusualMetric, UnusualBand: c.UnusualBand, UnusualDistance: c.UnusualDistance,
			UnusualPercentile: c.UnusualPercentile, UnusualVenue: c.UnusualVenue, UnusualDefinite: c.UnusualDefinite,
			LiqHeat: c.LiqHeat, LiqHeatRatio: c.LiqHeatRatio, LiqHeatVenue: c.LiqHeatVenue, LiqHeatDefinite: c.LiqHeatDefinite,
			BidPct: c.BidPct, BidDefinite: c.BidDefinite,
			Venues: venues, Coverage: coverageToDTO(c.Coverage),
			Recorded: recordedToDTO(c.Recorded),
		})
	}
	return derivativesBoardResponse{
		Exchange: got.Exchange,
		Universe: got.Universe,
		AsOf:     got.AsOf.UTC(),
		Coins:    coins,
		Coverage: coverageToDTO(got.Coverage),
		Definite: got.Definite,
		Summary:  got.Summary,
		Note:     got.Note,
	}
}
