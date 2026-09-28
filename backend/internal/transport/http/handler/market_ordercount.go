package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetOrderCountOverview handles GET /api/v1/market/order-count/overview.
func (h *MarketHandler) GetOrderCountOverview(w http.ResponseWriter, r *http.Request) {
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
	got, err := h.svc.GetOrderCountOverview(r.Context(), q.Get("exchange"), q.Get("quote"), limit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, orderCountOverviewToDTO(got))
}

// GetOrderCount handles GET /api/v1/market/order-count.
func (h *MarketHandler) GetOrderCount(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.GetOrderCount(r.Context(), q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, orderCountToDTO(got))
}

type orderCountWindowDTO struct {
	Window    string `json:"window"`
	BuyCount  int    `json:"buyCount"`
	SellCount int    `json:"sellCount"`
	Total     int    `json:"total"`
	Delta     int    `json:"delta"`
	Ratio     string `json:"ratio"`
	Lean      string `json:"lean"`
	Complete  bool   `json:"complete"`
	Summary   string `json:"summary"`
}

type orderCountVenueDTO struct {
	Exchange string                `json:"exchange"`
	Symbol   string                `json:"symbol"`
	Windows  []orderCountWindowDTO `json:"windows"`
	Summary  string                `json:"summary"`
	Error    string                `json:"error,omitempty"`
}

type orderCountResponse struct {
	Symbol   string               `json:"symbol"`
	Exchange string               `json:"exchange"`
	AsOf     time.Time            `json:"asOf"`
	Venues   []orderCountVenueDTO `json:"venues"`
	Combined *orderCountVenueDTO  `json:"combined,omitempty"`
	Summary  string               `json:"summary"`
	Note     string               `json:"note"`
}

func orderCountToDTO(a *domain.OrderCountReport) orderCountResponse {
	if a == nil {
		return orderCountResponse{}
	}
	venues := make([]orderCountVenueDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		venues = append(venues, orderCountVenueToDTO(v))
	}
	out := orderCountResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, AsOf: a.AsOf.UTC(),
		Venues: venues, Summary: a.Summary, Note: a.Note,
	}
	if a.Combined != nil {
		c := orderCountVenueToDTO(*a.Combined)
		out.Combined = &c
	}
	return out
}

type orderCountMarketWindowDTO struct {
	Window        string `json:"window"`
	BuyCount      int    `json:"buyCount"`
	SellCount     int    `json:"sellCount"`
	Total         int    `json:"total"`
	Delta         int    `json:"delta"`
	Ratio         string `json:"ratio"`
	Lean          string `json:"lean"`
	BuyCoins      int    `json:"buyCoins"`
	SellCoins     int    `json:"sellCoins"`
	BalancedCoins int    `json:"balancedCoins"`
	QuietCoins    int    `json:"quietCoins"`
	Coins         int    `json:"coins"`
	Complete      bool   `json:"complete"`
	Summary       string `json:"summary"`
}

type venueReadDTO struct {
	Exchange string    `json:"exchange"`
	Status   string    `json:"status"`
	AsOf     time.Time `json:"asOf,omitempty"`
	AgeSec   int       `json:"ageSec"`
}

type marketCoverageDTO struct {
	Complete bool           `json:"complete"`
	Venues   []venueReadDTO `json:"venues"`
	Missing  []string       `json:"missing"`
	Stale    []string       `json:"stale"`
}

func coverageToDTO(c domain.MarketCoverage) marketCoverageDTO {
	venues := make([]venueReadDTO, 0, len(c.Venues))
	for _, v := range c.Venues {
		row := venueReadDTO{Exchange: string(v.Exchange), Status: v.Status, AgeSec: v.AgeSec}
		if !v.AsOf.IsZero() {
			row.AsOf = v.AsOf.UTC()
		}
		venues = append(venues, row)
	}
	missing := c.Missing
	if missing == nil {
		missing = []string{}
	}
	stale := c.Stale
	if stale == nil {
		stale = []string{}
	}
	return marketCoverageDTO{Complete: c.Complete, Venues: venues, Missing: missing, Stale: stale}
}

type orderCountOverviewResponse struct {
	Exchange string                      `json:"exchange"`
	Quote    string                      `json:"quote"`
	Universe int                         `json:"universe"`
	AsOf     time.Time                   `json:"asOf"`
	Windows  []orderCountMarketWindowDTO `json:"windows"`
	Coins    []orderCountVenueDTO        `json:"coins"`
	Summary  string                      `json:"summary"`
	Note     string                      `json:"note"`
	Coverage marketCoverageDTO           `json:"coverage"`
	Definite bool                        `json:"definite"`
}

func orderCountOverviewToDTO(a *domain.OrderCountOverview) orderCountOverviewResponse {
	if a == nil {
		return orderCountOverviewResponse{}
	}
	wins := make([]orderCountMarketWindowDTO, 0, len(a.Windows))
	for _, w := range a.Windows {
		wins = append(wins, orderCountMarketWindowDTO{
			Window: w.Window, BuyCount: w.BuyCount, SellCount: w.SellCount,
			Total: w.Total, Delta: w.Delta, Ratio: fmt.Sprintf("%+.2f", w.Ratio),
			Lean: w.Lean, BuyCoins: w.BuyCoins, SellCoins: w.SellCoins,
			BalancedCoins: w.BalancedCoins, QuietCoins: w.QuietCoins, Coins: w.Coins,
			Complete: w.Complete, Summary: w.Summary,
		})
	}
	coins := make([]orderCountVenueDTO, 0, len(a.Coins))
	for _, c := range a.Coins {
		coins = append(coins, orderCountVenueToDTO(c))
	}
	return orderCountOverviewResponse{
		Exchange: a.Exchange, Quote: a.Quote, Universe: a.Universe, AsOf: a.AsOf.UTC(),
		Windows: wins, Coins: coins, Summary: a.Summary, Note: a.Note,
		Coverage: coverageToDTO(a.Coverage), Definite: a.Definite,
	}
}

func orderCountVenueToDTO(v domain.OrderCountVenue) orderCountVenueDTO {
	wins := make([]orderCountWindowDTO, 0, len(v.Windows))
	for _, w := range v.Windows {
		wins = append(wins, orderCountWindowDTO{
			Window: w.Window, BuyCount: w.BuyCount, SellCount: w.SellCount,
			Total: w.Total, Delta: w.Delta, Ratio: fmt.Sprintf("%+.2f", w.Ratio),
			Lean: w.Lean, Complete: w.Complete, Summary: w.Summary,
		})
	}
	return orderCountVenueDTO{
		Exchange: string(v.Exchange), Symbol: v.Symbol, Windows: wins,
		Summary: v.Summary, Error: v.Error,
	}
}
