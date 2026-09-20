package handler

import (
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetBookLiquidity handles GET /api/v1/market/orderbook/liquidity-flow.
func (h *MarketHandler) GetBookLiquidity(w http.ResponseWriter, r *http.Request) {
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
	minPx, err := optionalPositiveFloat(q.Get("minPrice"), "minPrice")
	if err != nil {
		writeError(w, err)
		return
	}
	maxPx, err := optionalPositiveFloat(q.Get("maxPrice"), "maxPrice")
	if err != nil {
		writeError(w, err)
		return
	}
	got, err := h.svc.GetBookLiquidity(r.Context(), q.Get("exchange"), q.Get("symbol"), q.Get("window"), startPtr, endPtr, minPx, maxPx)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bookLiquidityToDTO(got))
}

type bookLiquidityLevelDTO struct {
	Side         string `json:"side"`
	Price        string `json:"price"`
	FromNotional string `json:"fromNotional"`
	ToNotional   string `json:"toNotional"`
	Added        string `json:"added"`
	Pulled       string `json:"pulled"`
	Traded       string `json:"traded"`
	Remaining    string `json:"remaining"`
	Read         string `json:"read"`
}

type bookLiquiditySideDTO struct {
	Side         string                  `json:"side"`
	FromNotional string                  `json:"fromNotional"`
	ToNotional   string                  `json:"toNotional"`
	Added        string                  `json:"added"`
	Pulled       string                  `json:"pulled"`
	Traded       string                  `json:"traded"`
	Read         string                  `json:"read"`
	Levels       []bookLiquidityLevelDTO `json:"levels"`
}

type bookLiquidityPointDTO struct {
	From      time.Time `json:"from"`
	To        time.Time `json:"to"`
	BidAdded  string    `json:"bidAdded"`
	BidPulled string    `json:"bidPulled"`
	BidTraded string    `json:"bidTraded"`
	AskAdded  string    `json:"askAdded"`
	AskPulled string    `json:"askPulled"`
	AskTraded string    `json:"askTraded"`
}

type bookLiquidityVenueDTO struct {
	Exchange  string                  `json:"exchange"`
	Symbol    string                  `json:"symbol"`
	From      time.Time               `json:"from"`
	To        time.Time               `json:"to"`
	TickSize  string                  `json:"tickSize"`
	Samples   int                     `json:"samples"`
	Bids      bookLiquiditySideDTO    `json:"bids"`
	Asks      bookLiquiditySideDTO    `json:"asks"`
	Series    []bookLiquidityPointDTO `json:"series"`
	Complete  bool                    `json:"complete"`
	Truncated bool                    `json:"truncated"`
	KnownFrom *time.Time              `json:"knownFrom,omitempty"`
	KnownTo   *time.Time              `json:"knownTo,omitempty"`
	Summary   string                  `json:"summary"`
	Error     string                  `json:"error,omitempty"`
}

type bookLiquidityResponse struct {
	Symbol   string                  `json:"symbol"`
	Exchange string                  `json:"exchange"`
	Window   string                  `json:"window"`
	From     time.Time               `json:"from"`
	To       time.Time               `json:"to"`
	AsOf     time.Time               `json:"asOf"`
	MinPrice string                  `json:"minPrice,omitempty"`
	MaxPrice string                  `json:"maxPrice,omitempty"`
	Venues   []bookLiquidityVenueDTO `json:"venues"`
	Combined *bookLiquidityVenueDTO  `json:"combined,omitempty"`
	Summary  string                  `json:"summary"`
	Note     string                  `json:"note"`
}

func bookLiquidityToDTO(a *domain.BookLiquidityReport) bookLiquidityResponse {
	if a == nil {
		return bookLiquidityResponse{}
	}
	venues := make([]bookLiquidityVenueDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		venues = append(venues, bookLiquidityVenueToDTO(v))
	}
	out := bookLiquidityResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, Window: a.Window,
		From: a.From.UTC(), To: a.To.UTC(), AsOf: a.AsOf.UTC(),
		Venues: venues, Summary: a.Summary, Note: a.Note,
	}
	if a.MinPrice > 0 {
		out.MinPrice = formatHistQty(a.MinPrice)
	}
	if a.MaxPrice > 0 {
		out.MaxPrice = formatHistQty(a.MaxPrice)
	}
	if a.Combined != nil {
		c := bookLiquidityVenueToDTO(*a.Combined)
		out.Combined = &c
	}
	return out
}

func bookLiquidityVenueToDTO(v domain.BookLiquidityVenue) bookLiquidityVenueDTO {
	out := bookLiquidityVenueDTO{
		Exchange: string(v.Exchange), Symbol: v.Symbol,
		From: v.From.UTC(), To: v.To.UTC(), TickSize: formatHistQty(v.TickSize),
		Samples: v.Samples, Bids: bookLiquiditySideToDTO(v.Bids), Asks: bookLiquiditySideToDTO(v.Asks),
		Series: bookLiquiditySeriesToDTO(v.Series), Complete: v.Complete, Truncated: v.Truncated,
		Summary: v.Summary, Error: v.Error,
	}
	if !v.KnownFrom.IsZero() {
		t := v.KnownFrom.UTC()
		out.KnownFrom = &t
	}
	if !v.KnownTo.IsZero() {
		t := v.KnownTo.UTC()
		out.KnownTo = &t
	}
	return out
}

func bookLiquiditySideToDTO(s domain.BookLiquiditySide) bookLiquiditySideDTO {
	levels := make([]bookLiquidityLevelDTO, 0, len(s.Levels))
	for _, lv := range s.Levels {
		levels = append(levels, bookLiquidityLevelDTO{
			Side: lv.Side, Price: formatHistQty(lv.Price),
			FromNotional: formatHistQty(lv.FromNotional), ToNotional: formatHistQty(lv.ToNotional),
			Added: formatHistQty(lv.Added), Pulled: formatHistQty(lv.Pulled),
			Traded: formatHistQty(lv.Traded), Remaining: formatHistQty(lv.Remaining), Read: lv.Read,
		})
	}
	return bookLiquiditySideDTO{
		Side: s.Side, FromNotional: formatHistQty(s.FromNotional), ToNotional: formatHistQty(s.ToNotional),
		Added: formatHistQty(s.Added), Pulled: formatHistQty(s.Pulled), Traded: formatHistQty(s.Traded),
		Read: s.Read, Levels: levels,
	}
}

func bookLiquiditySeriesToDTO(in []domain.BookLiquidityPoint) []bookLiquidityPointDTO {
	out := make([]bookLiquidityPointDTO, 0, len(in))
	for _, p := range in {
		out = append(out, bookLiquidityPointDTO{
			From: p.From.UTC(), To: p.To.UTC(),
			BidAdded: formatHistQty(p.BidAdded), BidPulled: formatHistQty(p.BidPulled), BidTraded: formatHistQty(p.BidTraded),
			AskAdded: formatHistQty(p.AskAdded), AskPulled: formatHistQty(p.AskPulled), AskTraded: formatHistQty(p.AskTraded),
		})
	}
	return out
}
