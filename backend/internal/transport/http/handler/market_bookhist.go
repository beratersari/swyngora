package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetBookHistory handles GET /api/v1/market/orderbook/history.
func (h *MarketHandler) GetBookHistory(w http.ResponseWriter, r *http.Request) {
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
	at, err := optionalTimeParam(q.Get("at"))
	if err != nil {
		writeError(w, err)
		return
	}
	from, err := optionalTimeParam(q.Get("from"))
	if err != nil {
		writeError(w, err)
		return
	}
	to, err := optionalTimeParam(q.Get("to"))
	if err != nil {
		writeError(w, err)
		return
	}
	got, err := h.svc.GetBookHistory(r.Context(), q.Get("exchange"), q.Get("symbol"), at, from, to, limit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bookHistoryToDTO(got))
}

// CompareBookHistory handles GET /api/v1/market/orderbook/history/compare.
func (h *MarketHandler) CompareBookHistory(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, err := parseTimeParam(q.Get("from"))
	if err != nil {
		writeError(w, fmt.Errorf("%w: from time is required (RFC3339 or unix ms)", domain.ErrInvalidArgument))
		return
	}
	to, err := parseTimeParam(q.Get("to"))
	if err != nil {
		writeError(w, fmt.Errorf("%w: to time is required (RFC3339 or unix ms)", domain.ErrInvalidArgument))
		return
	}
	got, err := h.svc.CompareBookHistory(r.Context(), q.Get("exchange"), q.Get("symbol"), from, to)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bookDiffToDTO(got))
}

func optionalTimeParam(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	t, err := parseTimeParam(raw)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

type bookHistLevelDTO struct {
	Price    string `json:"price"`
	Quantity string `json:"quantity"`
	Notional string `json:"notional"`
	Wall     bool   `json:"wall"`
}

type bookHistWallDTO struct {
	Side        string  `json:"side"`
	Price       string  `json:"price"`
	Quantity    string  `json:"quantity"`
	Notional    string  `json:"notional"`
	DistancePct string  `json:"distancePct,omitempty"`
	Share       float64 `json:"share,omitempty"`
}

type bookHistSnapDTO struct {
	Exchange     string             `json:"exchange"`
	Symbol       string             `json:"symbol"`
	SampledAt    time.Time          `json:"sampledAt"`
	Mid          string             `json:"mid"`
	BestBid      string             `json:"bestBid"`
	BestAsk      string             `json:"bestAsk"`
	Spread       string             `json:"spread"`
	SpreadPct    string             `json:"spreadPct"`
	GroupSize    string             `json:"groupSize"`
	BidNotional  string             `json:"bidNotional"`
	AskNotional  string             `json:"askNotional"`
	Imbalance    float64            `json:"imbalance"`
	Pressure     string             `json:"pressure"`
	BidWalls     int                `json:"bidWalls"`
	AskWalls     int                `json:"askWalls"`
	Live         bool               `json:"live"`
	Complete     bool               `json:"complete"`
	SlackSeconds float64            `json:"slackSeconds,omitempty"`
	Bids         []bookHistLevelDTO `json:"bids,omitempty"`
	Asks         []bookHistLevelDTO `json:"asks,omitempty"`
	Walls        []bookHistWallDTO  `json:"walls,omitempty"`
}

type bookHistoryResponse struct {
	Symbol    string            `json:"symbol"`
	Exchange  string            `json:"exchange"`
	At        *time.Time        `json:"at,omitempty"`
	Snapshot  *bookHistSnapDTO  `json:"snapshot,omitempty"`
	Snapshots []bookHistSnapDTO `json:"snapshots,omitempty"`
	Summary   string            `json:"summary"`
	Note      string            `json:"note"`
}

type bookLevelChangeDTO struct {
	Side         string `json:"side"`
	Price        string `json:"price"`
	FromNotional string `json:"fromNotional"`
	ToNotional   string `json:"toNotional"`
	Delta        string `json:"delta"`
	Change       string `json:"change"`
}

type bookDiffResponse struct {
	Symbol           string               `json:"symbol"`
	Exchange         string               `json:"exchange"`
	From             bookHistSnapDTO      `json:"from"`
	To               bookHistSnapDTO      `json:"to"`
	MidDelta         string               `json:"midDelta"`
	MidDeltaPct      string               `json:"midDeltaPct"`
	SpreadDelta      string               `json:"spreadDelta"`
	ImbalanceDelta   string               `json:"imbalanceDelta"`
	BidNotionalDelta string               `json:"bidNotionalDelta"`
	AskNotionalDelta string               `json:"askNotionalDelta"`
	Gained           []bookLevelChangeDTO `json:"gained"`
	Lost             []bookLevelChangeDTO `json:"lost"`
	WallsAdded       []bookHistWallDTO    `json:"wallsAdded"`
	WallsRemoved     []bookHistWallDTO    `json:"wallsRemoved"`
	Summary          string               `json:"summary"`
	Note             string               `json:"note"`
}

func bookHistoryToDTO(a *domain.BookHistoryReport) bookHistoryResponse {
	if a == nil {
		return bookHistoryResponse{}
	}
	out := bookHistoryResponse{Symbol: a.Symbol, Exchange: a.Exchange, Summary: a.Summary, Note: a.Note}
	if !a.At.IsZero() {
		t := a.At.UTC()
		out.At = &t
	}
	if a.Snapshot != nil {
		s := bookSnapToDTO(*a.Snapshot, true)
		out.Snapshot = &s
	}
	if len(a.Snapshots) > 0 {
		out.Snapshots = make([]bookHistSnapDTO, 0, len(a.Snapshots))
		for _, s := range a.Snapshots {
			out.Snapshots = append(out.Snapshots, bookSnapToDTO(s, false))
		}
	}
	return out
}

func bookDiffToDTO(a *domain.BookHistoryDiff) bookDiffResponse {
	if a == nil {
		return bookDiffResponse{}
	}
	return bookDiffResponse{
		Symbol: a.Symbol, Exchange: a.Exchange,
		From: bookSnapToDTO(a.From, true), To: bookSnapToDTO(a.To, true),
		MidDelta: formatHistQty(a.MidDelta), MidDeltaPct: domain.FormatSignedPct(a.MidDeltaPct),
		SpreadDelta: formatHistQty(a.SpreadDelta), ImbalanceDelta: formatHistQty(a.ImbalanceDelta),
		BidNotionalDelta: formatHistQty(a.BidNotionalDelta), AskNotionalDelta: formatHistQty(a.AskNotionalDelta),
		Gained: levelChangesDTO(a.Gained), Lost: levelChangesDTO(a.Lost),
		WallsAdded: bookWallsDTO(a.WallsAdded), WallsRemoved: bookWallsDTO(a.WallsRemoved),
		Summary: a.Summary, Note: a.Note,
	}
}

func bookSnapToDTO(s domain.BookHistorySnapshot, full bool) bookHistSnapDTO {
	out := bookHistSnapDTO{
		Exchange: string(s.Exchange), Symbol: s.Symbol, SampledAt: s.SampledAt.UTC(),
		Mid: formatHistQty(s.Mid), BestBid: formatHistQty(s.BestBid), BestAsk: formatHistQty(s.BestAsk),
		Spread: formatHistQty(s.Spread), SpreadPct: formatHistQty(s.SpreadPct),
		GroupSize: formatHistQty(s.GroupSize), BidNotional: formatHistQty(s.BidNotional),
		AskNotional: formatHistQty(s.AskNotional), Imbalance: s.Imbalance, Pressure: s.Pressure,
		BidWalls: s.BidWalls, AskWalls: s.AskWalls, Live: s.Live, Complete: s.Complete,
		SlackSeconds: s.SlackSeconds,
	}
	if full {
		out.Bids = bookLevelsDTO(s.Bids)
		out.Asks = bookLevelsDTO(s.Asks)
		out.Walls = bookWallsDTO(s.Walls)
	}
	return out
}

func bookLevelsDTO(in []domain.BookHistoryLevel) []bookHistLevelDTO {
	out := make([]bookHistLevelDTO, 0, len(in))
	for _, lv := range in {
		out = append(out, bookHistLevelDTO{
			Price: formatHistQty(lv.Price), Quantity: formatHistQty(lv.Quantity),
			Notional: formatHistQty(lv.Notional), Wall: lv.Wall,
		})
	}
	return out
}

func bookWallsDTO(in []domain.BookHistoryWall) []bookHistWallDTO {
	out := make([]bookHistWallDTO, 0, len(in))
	for _, w := range in {
		out = append(out, bookHistWallDTO{
			Side: w.Side, Price: formatHistQty(w.Price), Quantity: formatHistQty(w.Quantity),
			Notional: formatHistQty(w.Notional), DistancePct: formatHistQty(w.DistancePct), Share: w.Share,
		})
	}
	return out
}

func levelChangesDTO(in []domain.BookHistoryLevelChange) []bookLevelChangeDTO {
	out := make([]bookLevelChangeDTO, 0, len(in))
	for _, c := range in {
		out = append(out, bookLevelChangeDTO{
			Side: c.Side, Price: formatHistQty(c.Price),
			FromNotional: formatHistQty(c.FromNotional), ToNotional: formatHistQty(c.ToNotional),
			Delta: formatHistQty(c.Delta), Change: c.Change,
		})
	}
	return out
}
