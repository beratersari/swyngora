package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetPerpDepthOverview handles GET /api/v1/market/perp-depth/overview.
func (h *MarketHandler) GetPerpDepthOverview(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var rangePct float64
	if raw := q.Get("rangePct"); raw != "" {
		n, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			writeError(w, fmt.Errorf("%w: rangePct must be a number", domain.ErrInvalidArgument))
			return
		}
		rangePct = n
	}
	limit := 0
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, fmt.Errorf("%w: limit must be an integer", domain.ErrInvalidArgument))
			return
		}
		limit = n
	}
	got, err := h.svc.GetPerpDepthOverview(r.Context(), q.Get("exchange"), rangePct, limit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, perpDepthOverviewToDTO(got))
}

// GetPerpDepth handles GET /api/v1/market/perp-depth.
func (h *MarketHandler) GetPerpDepth(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var rangePct float64
	if raw := q.Get("rangePct"); raw != "" {
		n, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			writeError(w, fmt.Errorf("%w: rangePct must be a number", domain.ErrInvalidArgument))
			return
		}
		rangePct = n
	}
	got, err := h.svc.GetPerpDepth(r.Context(), q.Get("exchange"), q.Get("symbol"), rangePct)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, perpDepthToDTO(got))
}

type perpDepthSideDTO struct {
	Notional  string `json:"notional"`
	Quantity  string `json:"quantity"`
	Levels    int    `json:"levels"`
	EdgePrice string `json:"edgePrice,omitempty"`
}

type perpDepthBandDTO struct {
	RangePct  string           `json:"rangePct"`
	Below     perpDepthSideDTO `json:"below"`
	Above     perpDepthSideDTO `json:"above"`
	Imbalance string           `json:"imbalance"`
	Ratio     string           `json:"ratio,omitempty"`
	Lean      string           `json:"lean"`
	Complete  bool             `json:"complete"`
}

type perpDepthVenueDTO struct {
	Exchange    string             `json:"exchange"`
	Symbol      string             `json:"symbol"`
	Mid         string             `json:"mid"`
	CoverBidPct string             `json:"coverBidPct"`
	CoverAskPct string             `json:"coverAskPct"`
	Headline    perpDepthBandDTO   `json:"headline"`
	Bands       []perpDepthBandDTO `json:"bands"`
	Summary     string             `json:"summary"`
	Error       string             `json:"error,omitempty"`
}

type perpDepthResponse struct {
	Symbol   string              `json:"symbol"`
	Exchange string              `json:"exchange"`
	RangePct string              `json:"rangePct"`
	AsOf     time.Time           `json:"asOf"`
	Venues   []perpDepthVenueDTO `json:"venues"`
	Combined *perpDepthVenueDTO  `json:"combined,omitempty"`
	Summary  string              `json:"summary"`
	Note     string              `json:"note"`
}

func perpDepthToDTO(a *domain.PerpDepthReport) perpDepthResponse {
	if a == nil {
		return perpDepthResponse{}
	}
	venues := make([]perpDepthVenueDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		venues = append(venues, perpDepthVenueToDTO(v))
	}
	out := perpDepthResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, RangePct: formatHistQty(a.RangePct),
		AsOf: a.AsOf.UTC(), Venues: venues, Summary: a.Summary, Note: a.Note,
	}
	if a.Combined != nil {
		c := perpDepthVenueToDTO(*a.Combined)
		out.Combined = &c
	}
	return out
}

func perpDepthVenueToDTO(v domain.PerpDepthVenue) perpDepthVenueDTO {
	bands := make([]perpDepthBandDTO, 0, len(v.Bands))
	for _, b := range v.Bands {
		bands = append(bands, perpDepthBandToDTO(b))
	}
	return perpDepthVenueDTO{
		Exchange: string(v.Exchange), Symbol: v.Symbol, Mid: formatHistQty(v.Mid),
		CoverBidPct: formatHistQty(v.CoverBidPct), CoverAskPct: formatHistQty(v.CoverAskPct),
		Headline: perpDepthBandToDTO(v.Headline), Bands: bands, Summary: v.Summary, Error: v.Error,
	}
}

func perpDepthBandToDTO(b domain.PerpDepthBand) perpDepthBandDTO {
	return perpDepthBandDTO{
		RangePct:  formatHistQty(b.RangePct),
		Below:     perpDepthSideToDTO(b.Below),
		Above:     perpDepthSideToDTO(b.Above),
		Imbalance: domain.FormatSignedQty(b.Imbalance),
		Ratio:     formatHistQty(b.Ratio),
		Lean:      b.Lean, Complete: b.Complete,
	}
}

func perpDepthSideToDTO(s domain.PerpDepthSide) perpDepthSideDTO {
	out := perpDepthSideDTO{
		Notional: formatHistQty(s.Notional), Quantity: formatHistQty(s.Quantity), Levels: s.Levels,
	}
	if s.EdgePrice > 0 {
		out.EdgePrice = formatHistQty(s.EdgePrice)
	}
	return out
}

type perpDepthCoinShareDTO struct {
	Symbol      string `json:"symbol"`
	Base        string `json:"base"`
	Bid         string `json:"bid"`
	Ask         string `json:"ask"`
	Total       string `json:"total"`
	SharePct    string `json:"sharePct"`
	BidSharePct string `json:"bidSharePct"`
	AskSharePct string `json:"askSharePct"`
	Lean        string `json:"lean"`
	Complete    bool   `json:"complete"`
	Error       string `json:"error,omitempty"`
}

type perpDepthShareDriverDTO struct {
	Symbol    string `json:"symbol"`
	Base      string `json:"base"`
	ChangePct string `json:"changePct"`
	Kind      string `json:"kind"`
}

type perpDepthSizeDriverDTO struct {
	Symbol string `json:"symbol"`
	Base   string `json:"base"`
	Change string `json:"change"`
	Kind   string `json:"kind"`
}

type perpDepthShareWindowDTO struct {
	Window         string                    `json:"window"`
	BidPct         string                    `json:"bidPct"`
	AskPct         string                    `json:"askPct"`
	PastBidPct     string                    `json:"pastBidPct,omitempty"`
	PastAskPct     string                    `json:"pastAskPct,omitempty"`
	ChangePct      string                    `json:"changePct,omitempty"`
	Kind           string                    `json:"kind"`
	Total          string                    `json:"total,omitempty"`
	PastTotal      string                    `json:"pastTotal,omitempty"`
	TotalChange    string                    `json:"totalChange,omitempty"`
	TotalChangePct string                    `json:"totalChangePct,omitempty"`
	DepthKind      string                    `json:"depthKind,omitempty"`
	Drivers        []perpDepthShareDriverDTO `json:"drivers,omitempty"`
	SizeDrivers    []perpDepthSizeDriverDTO  `json:"sizeDrivers,omitempty"`
	Complete       bool                      `json:"complete"`
	SampleTime     *time.Time                `json:"sampleTime,omitempty"`
	Summary        string                    `json:"summary"`
}

type perpDepthOverviewBandCoinDTO struct {
	Symbol   string `json:"symbol"`
	Base     string `json:"base"`
	Bid      string `json:"bid"`
	Ask      string `json:"ask"`
	Total    string `json:"total"`
	SharePct string `json:"sharePct"`
}

type perpDepthOverviewBandDTO struct {
	RangePct      string                         `json:"rangePct"`
	Bid           string                         `json:"bid"`
	Ask           string                         `json:"ask"`
	Total         string                         `json:"total"`
	BidPct        string                         `json:"bidPct"`
	AskPct        string                         `json:"askPct"`
	Lean          string                         `json:"lean"`
	OfHeadlinePct string                         `json:"ofHeadlinePct"`
	AddedPct      string                         `json:"addedPct"`
	AddedBid      string                         `json:"addedBid"`
	AddedAsk      string                         `json:"addedAsk"`
	Complete      bool                           `json:"complete"`
	Leaders       []string                       `json:"leaders"`
	AddedLeaders  []string                       `json:"addedLeaders,omitempty"`
	Coins         []perpDepthOverviewBandCoinDTO `json:"coins"`
	Summary       string                         `json:"summary"`
}

type perpDepthTotalPointDTO struct {
	Time  time.Time `json:"time"`
	Total string    `json:"total"`
	Price string    `json:"price,omitempty"`
}

type perpDepthBookMoveDTO struct {
	Kind           string                   `json:"kind"`
	From           time.Time                `json:"from"`
	To             time.Time                `json:"to"`
	PastTotal      string                   `json:"pastTotal"`
	Total          string                   `json:"total"`
	Change         string                   `json:"change"`
	ChangePct      string                   `json:"changePct"`
	PriceFrom      string                   `json:"priceFrom,omitempty"`
	PriceTo        string                   `json:"priceTo,omitempty"`
	PriceAfter     string                   `json:"priceAfter,omitempty"`
	PriceChangePct string                   `json:"priceChangePct,omitempty"`
	PriceAfterPct  string                   `json:"priceAfterPct,omitempty"`
	PriceKind      string                   `json:"priceKind,omitempty"`
	AfterKind      string                   `json:"afterKind,omitempty"`
	Drivers        []perpDepthMoveDriverDTO `json:"drivers,omitempty"`
	Span           string                   `json:"span,omitempty"`
	AfterTotal     string                   `json:"afterTotal,omitempty"`
	Follow         string                   `json:"follow,omitempty"`
	FollowPct      string                   `json:"followPct,omitempty"`
	Summary        string                   `json:"summary"`
}

type perpDepthMoveDriverDTO struct {
	Symbol   string `json:"symbol"`
	Base     string `json:"base"`
	Change   string `json:"change"`
	SharePct string `json:"sharePct"`
	Kind     string `json:"kind"`
}

type perpDepthOverviewResponse struct {
	Exchange  string                     `json:"exchange"`
	RangePct  string                     `json:"rangePct"`
	Universe  int                        `json:"universe"`
	Known     int                        `json:"known"`
	AsOf      time.Time                  `json:"asOf"`
	Bid       string                     `json:"bid"`
	Ask       string                     `json:"ask"`
	Total     string                     `json:"total"`
	Imbalance string                     `json:"imbalance"`
	BidPct    string                     `json:"bidPct"`
	AskPct    string                     `json:"askPct"`
	Lean      string                     `json:"lean"`
	Coins     []perpDepthCoinShareDTO    `json:"coins"`
	Leaders   []string                   `json:"leaders"`
	Missing   []string                   `json:"missing,omitempty"`
	Windows   []perpDepthShareWindowDTO  `json:"windows"`
	History   []perpDepthTotalPointDTO   `json:"history"`
	Moves     []perpDepthBookMoveDTO     `json:"moves"`
	Bands     []perpDepthOverviewBandDTO `json:"bands"`
	Summary   string                     `json:"summary"`
	Note      string                     `json:"note"`
}

func perpDepthOverviewToDTO(a *domain.PerpDepthOverview) perpDepthOverviewResponse {
	if a == nil {
		return perpDepthOverviewResponse{}
	}
	coins := make([]perpDepthCoinShareDTO, 0, len(a.Coins))
	for _, c := range a.Coins {
		coins = append(coins, perpDepthCoinShareDTO{
			Symbol: c.Symbol, Base: c.Base,
			Bid: formatHistQty(c.Bid), Ask: formatHistQty(c.Ask), Total: formatHistQty(c.Total),
			SharePct: formatHistQty(c.SharePct), BidSharePct: formatHistQty(c.BidSharePct),
			AskSharePct: formatHistQty(c.AskSharePct), Lean: c.Lean, Complete: c.Complete, Error: c.Error,
		})
	}
	wins := make([]perpDepthShareWindowDTO, 0, len(a.Windows))
	for _, w := range a.Windows {
		row := perpDepthShareWindowDTO{
			Window: w.Window, BidPct: formatHistQty(w.BidPct), AskPct: formatHistQty(w.AskPct),
			Kind: w.Kind, Complete: w.Complete, Summary: w.Summary,
		}
		if w.Complete {
			row.PastBidPct = formatHistQty(w.PastBidPct)
			row.PastAskPct = formatHistQty(w.PastAskPct)
			row.ChangePct = domain.FormatSignedPct(w.ChangePct)
			row.Total = formatHistQty(w.Total)
			row.PastTotal = formatHistQty(w.PastTotal)
			row.TotalChange = domain.FormatSignedQty(w.TotalChange)
			row.TotalChangePct = domain.FormatSignedPct(w.TotalChangePct)
			row.DepthKind = w.DepthKind
			if !w.SampleTime.IsZero() {
				t := w.SampleTime.UTC()
				row.SampleTime = &t
			}
			drivers := make([]perpDepthShareDriverDTO, 0, len(w.Drivers))
			for _, d := range w.Drivers {
				drivers = append(drivers, perpDepthShareDriverDTO{
					Symbol: d.Symbol, Base: d.Base, ChangePct: domain.FormatSignedPct(d.ChangePct), Kind: d.Kind,
				})
			}
			row.Drivers = drivers
			sizes := make([]perpDepthSizeDriverDTO, 0, len(w.SizeDrivers))
			for _, d := range w.SizeDrivers {
				sizes = append(sizes, perpDepthSizeDriverDTO{
					Symbol: d.Symbol, Base: d.Base, Change: domain.FormatSignedQty(d.Change), Kind: d.Kind,
				})
			}
			row.SizeDrivers = sizes
		}
		wins = append(wins, row)
	}
	history := make([]perpDepthTotalPointDTO, 0, len(a.History))
	for _, p := range a.History {
		row := perpDepthTotalPointDTO{Time: p.Time.UTC(), Total: formatHistQty(p.Total)}
		if p.Price > 0 {
			row.Price = formatHistQty(p.Price)
		}
		history = append(history, row)
	}
	moves := make([]perpDepthBookMoveDTO, 0, len(a.Moves))
	for _, m := range a.Moves {
		row := perpDepthBookMoveDTO{
			Kind: m.Kind, From: m.From.UTC(), To: m.To.UTC(),
			PastTotal: formatHistQty(m.PastTotal), Total: formatHistQty(m.Total),
			Change: domain.FormatSignedQty(m.Change), ChangePct: domain.FormatSignedPct(m.ChangePct),
			Summary: m.Summary,
		}
		if m.PriceFrom > 0 {
			row.PriceFrom = formatHistQty(m.PriceFrom)
			row.PriceTo = formatHistQty(m.PriceTo)
			row.PriceChangePct = domain.FormatSignedPct(m.PriceChangePct)
			row.PriceKind = m.PriceKind
		}
		if m.PriceAfter > 0 {
			row.PriceAfter = formatHistQty(m.PriceAfter)
			row.PriceAfterPct = domain.FormatSignedPct(m.PriceAfterPct)
			row.AfterKind = m.AfterKind
		}
		drivers := make([]perpDepthMoveDriverDTO, 0, len(m.Drivers))
		for _, d := range m.Drivers {
			drivers = append(drivers, perpDepthMoveDriverDTO{
				Symbol: d.Symbol, Base: d.Base, Change: domain.FormatSignedQty(d.Change),
				SharePct: domain.FormatSignedPct(d.SharePct), Kind: d.Kind,
			})
		}
		row.Drivers = drivers
		row.Span = m.Span
		if m.AfterTotal > 0 {
			row.AfterTotal = formatHistQty(m.AfterTotal)
			row.Follow = m.Follow
			row.FollowPct = domain.FormatSignedPct(m.FollowPct)
		}
		moves = append(moves, row)
	}
	bands := make([]perpDepthOverviewBandDTO, 0, len(a.Bands))
	for _, b := range a.Bands {
		rowCoins := make([]perpDepthOverviewBandCoinDTO, 0, len(b.Coins))
		for _, c := range b.Coins {
			rowCoins = append(rowCoins, perpDepthOverviewBandCoinDTO{
				Symbol: c.Symbol, Base: c.Base,
				Bid: formatHistQty(c.Bid), Ask: formatHistQty(c.Ask), Total: formatHistQty(c.Total),
				SharePct: formatHistQty(c.SharePct),
			})
		}
		bands = append(bands, perpDepthOverviewBandDTO{
			RangePct: formatHistQty(b.RangePct), Bid: formatHistQty(b.Bid), Ask: formatHistQty(b.Ask),
			Total: formatHistQty(b.Total), BidPct: formatHistQty(b.BidPct), AskPct: formatHistQty(b.AskPct),
			Lean: b.Lean, OfHeadlinePct: formatHistQty(b.OfHeadlinePct), AddedPct: formatHistQty(b.AddedPct),
			AddedBid: formatHistQty(b.AddedBid), AddedAsk: formatHistQty(b.AddedAsk), Complete: b.Complete,
			Leaders: b.Leaders, AddedLeaders: b.AddedLeaders, Coins: rowCoins, Summary: b.Summary,
		})
	}
	return perpDepthOverviewResponse{
		Exchange: a.Exchange, RangePct: formatHistQty(a.RangePct), Universe: a.Universe, Known: a.Known,
		AsOf: a.AsOf.UTC(), Bid: formatHistQty(a.Bid), Ask: formatHistQty(a.Ask), Total: formatHistQty(a.Total),
		Imbalance: domain.FormatSignedQty(a.Imbalance), BidPct: formatHistQty(a.BidPct), AskPct: formatHistQty(a.AskPct),
		Lean: a.Lean, Coins: coins, Leaders: a.Leaders, Missing: a.Missing, Windows: wins, History: history, Moves: moves, Bands: bands,
		Summary: a.Summary, Note: a.Note,
	}
}
