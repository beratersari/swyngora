package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetPrintSizeOverview handles GET /api/v1/market/print-size/overview.
func (h *MarketHandler) GetPrintSizeOverview(w http.ResponseWriter, r *http.Request) {
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
	got, err := h.svc.GetPrintSizeOverview(r.Context(), q.Get("exchange"), q.Get("quote"), limit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, printSizeOverviewToDTO(got))
}

// GetPrintSize handles GET /api/v1/market/print-size.
func (h *MarketHandler) GetPrintSize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.GetPrintSize(r.Context(), q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, printSizeToDTO(got))
}

type printSizeSideDTO struct {
	Avg           string `json:"avg"`
	PriorAvg      string `json:"priorAvg,omitempty"`
	Notional      string `json:"notional"`
	PriorNotional string `json:"priorNotional,omitempty"`
	Count         int    `json:"count"`
	PriorCount    int    `json:"priorCount"`
	Ratio         string `json:"ratio,omitempty"`
	Kind          string `json:"kind"`
}

type printSizeWindowDTO struct {
	Window   string           `json:"window"`
	Buy      printSizeSideDTO `json:"buy"`
	Sell     printSizeSideDTO `json:"sell"`
	Complete bool             `json:"complete"`
	From     string           `json:"from,omitempty"`
	To       string           `json:"to,omitempty"`
	Read     string           `json:"read"`
	Summary  string           `json:"summary"`
}

type printSizeVenueDTO struct {
	Exchange string               `json:"exchange"`
	Symbol   string               `json:"symbol"`
	Windows  []printSizeWindowDTO `json:"windows"`
	Summary  string               `json:"summary"`
	Error    string               `json:"error,omitempty"`
}

type printSizeResponse struct {
	Symbol   string              `json:"symbol"`
	Exchange string              `json:"exchange"`
	AsOf     time.Time           `json:"asOf"`
	Venues   []printSizeVenueDTO `json:"venues"`
	Combined *printSizeVenueDTO  `json:"combined,omitempty"`
	Note     string              `json:"note"`
}

func printSizeToDTO(a *domain.PrintSizeReport) printSizeResponse {
	if a == nil {
		return printSizeResponse{}
	}
	venues := make([]printSizeVenueDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		venues = append(venues, printSizeVenueToDTO(v))
	}
	out := printSizeResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, AsOf: a.AsOf.UTC(),
		Venues: venues, Note: a.Note,
	}
	if a.Combined != nil {
		c := printSizeVenueToDTO(*a.Combined)
		out.Combined = &c
	}
	return out
}

type printSizeMarketWindowDTO struct {
	Window         string           `json:"window"`
	Buy            printSizeSideDTO `json:"buy"`
	Sell           printSizeSideDTO `json:"sell"`
	Read           string           `json:"read"`
	LargerCoins    int              `json:"largerCoins"`
	SmallerCoins   int              `json:"smallerCoins"`
	FewLargeCoins  int              `json:"fewLargeCoins"`
	ManySmallCoins int              `json:"manySmallCoins"`
	StableCoins    int              `json:"stableCoins"`
	QuietCoins     int              `json:"quietCoins"`
	Coins          int              `json:"coins"`
	Complete       bool             `json:"complete"`
	Summary        string           `json:"summary"`
}

type printSizeOverviewResponse struct {
	Exchange string                     `json:"exchange"`
	Quote    string                     `json:"quote"`
	Universe int                        `json:"universe"`
	AsOf     time.Time                  `json:"asOf"`
	Windows  []printSizeMarketWindowDTO `json:"windows"`
	Coins    []printSizeVenueDTO        `json:"coins"`
	Summary  string                     `json:"summary"`
	Note     string                     `json:"note"`
}

func printSizeOverviewToDTO(a *domain.PrintSizeOverview) printSizeOverviewResponse {
	if a == nil {
		return printSizeOverviewResponse{}
	}
	wins := make([]printSizeMarketWindowDTO, 0, len(a.Windows))
	for _, w := range a.Windows {
		wins = append(wins, printSizeMarketWindowDTO{
			Window: w.Window, Buy: printSizeSideToDTO(w.Buy), Sell: printSizeSideToDTO(w.Sell),
			Read: w.Read, LargerCoins: w.LargerCoins, SmallerCoins: w.SmallerCoins,
			FewLargeCoins: w.FewLargeCoins, ManySmallCoins: w.ManySmallCoins,
			StableCoins: w.StableCoins, QuietCoins: w.QuietCoins, Coins: w.Coins,
			Complete: w.Complete, Summary: w.Summary,
		})
	}
	coins := make([]printSizeVenueDTO, 0, len(a.Coins))
	for _, c := range a.Coins {
		coins = append(coins, printSizeVenueToDTO(c))
	}
	return printSizeOverviewResponse{
		Exchange: a.Exchange, Quote: a.Quote, Universe: a.Universe, AsOf: a.AsOf.UTC(),
		Windows: wins, Coins: coins, Summary: a.Summary, Note: a.Note,
	}
}

func printSizeVenueToDTO(v domain.PrintSizeVenue) printSizeVenueDTO {
	wins := make([]printSizeWindowDTO, 0, len(v.Windows))
	for _, w := range v.Windows {
		row := printSizeWindowDTO{
			Window: w.Window, Buy: printSizeSideToDTO(w.Buy), Sell: printSizeSideToDTO(w.Sell),
			Complete: w.Complete, Read: w.Read, Summary: w.Summary,
		}
		if !w.From.IsZero() {
			row.From = w.From.UTC().Format(time.RFC3339Nano)
		}
		if !w.To.IsZero() {
			row.To = w.To.UTC().Format(time.RFC3339Nano)
		}
		wins = append(wins, row)
	}
	return printSizeVenueDTO{
		Exchange: string(v.Exchange), Symbol: v.Symbol,
		Windows: wins, Summary: v.Summary, Error: v.Error,
	}
}

func printSizeSideToDTO(s domain.PrintSizeSide) printSizeSideDTO {
	out := printSizeSideDTO{
		Avg: formatHistQty(s.Avg), Notional: formatHistQty(s.Notional),
		Count: s.Count, PriorCount: s.PriorCount, Kind: s.Kind,
	}
	if s.PriorAvg > 0 {
		out.PriorAvg = formatHistQty(s.PriorAvg)
	}
	if s.PriorNotional > 0 {
		out.PriorNotional = formatHistQty(s.PriorNotional)
	}
	if s.Ratio > 0 {
		out.Ratio = formatHistQty(s.Ratio)
	}
	return out
}
