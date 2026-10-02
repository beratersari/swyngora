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
	Exchange       string `json:"exchange"`
	OIValue        string `json:"oiValue,omitempty"`
	OIStatus       string `json:"oiStatus"`
	FundingRate    string `json:"fundingRate,omitempty"`
	FundingRatePct string `json:"fundingRatePct,omitempty"`
	FundingPayer   string `json:"fundingPayer,omitempty"`
	FundingStatus  string `json:"fundingStatus"`
	MarkPrice      string `json:"markPrice,omitempty"`
	LiqLong        string `json:"liqLong,omitempty"`
	LiqShort       string `json:"liqShort,omitempty"`
	LiqStatus      string `json:"liqStatus"`
	LongPct        string `json:"longPct,omitempty"`
	LongStatus     string `json:"longStatus"`
	TakerBuyPct    string `json:"takerBuyPct,omitempty"`
	TakerStatus    string `json:"takerStatus"`
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
	Venues               []derivativesVenueDTO `json:"venues"`
	Coverage             marketCoverageDTO     `json:"coverage"`
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
			Venues: venues, Coverage: coverageToDTO(c.Coverage),
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
