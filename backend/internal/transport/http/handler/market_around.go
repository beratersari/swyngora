package handler

import (
	"fmt"
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetAround handles GET /api/v1/market/around.
func (h *MarketHandler) GetAround(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	raw := q.Get("at")
	if raw == "" {
		writeError(w, fmt.Errorf("%w: at time is required", domain.ErrInvalidArgument))
		return
	}
	at, err := parseTimeParam(raw)
	if err != nil {
		writeError(w, err)
		return
	}
	got, err := h.svc.GetAround(r.Context(), q.Get("exchange"), q.Get("symbol"), q.Get("window"), q.Get("during"), at)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, aroundToDTO(got))
}

type aroundPriceDTO struct {
	Open      string `json:"open"`
	High      string `json:"high"`
	Low       string `json:"low"`
	Close     string `json:"close"`
	Change    string `json:"change"`
	ChangePct string `json:"changePct"`
	Range     string `json:"range"`
	RangePct  string `json:"rangePct"`
	Direction string `json:"direction"`
}

type aroundFlowDTO struct {
	Volume        string `json:"volume"`
	BuyVolume     string `json:"buyVolume"`
	SellVolume    string `json:"sellVolume"`
	Delta         string `json:"delta"`
	BuyShare      string `json:"buyShare"`
	Dominant      string `json:"dominant"`
	BuySellKnown  bool   `json:"buySellKnown"`
	VWAP          string `json:"vwap"`
	VsVWAP        string `json:"vsVwap"`
	DistancePct   string `json:"distancePct"`
	Typical       string `json:"typical,omitempty"`
	VolumeRatio   string `json:"volumeRatio,omitempty"`
	VolumeGrade   string `json:"volumeGrade,omitempty"`
	TypicalSample int    `json:"typicalSample,omitempty"`
	TypicalKnown  bool   `json:"typicalKnown"`
}

type aroundProfileDTO struct {
	POC           string `json:"poc"`
	POCVolume     string `json:"pocVolume"`
	ValueAreaLow  string `json:"valueAreaLow"`
	ValueAreaHigh string `json:"valueAreaHigh"`
	LastVsArea    string `json:"lastVsArea"`
}

type aroundBookDTO struct {
	FromMid          string `json:"fromMid"`
	ToMid            string `json:"toMid"`
	MidDelta         string `json:"midDelta"`
	MidDeltaPct      string `json:"midDeltaPct"`
	BidNotionalDelta string `json:"bidNotionalDelta"`
	AskNotionalDelta string `json:"askNotionalDelta"`
	ImbalanceDelta   string `json:"imbalanceDelta"`
	WallsAdded       int    `json:"wallsAdded"`
	WallsRemoved     int    `json:"wallsRemoved"`
	Summary          string `json:"summary"`
	Complete         bool   `json:"complete"`
}

type aroundFuturesDTO struct {
	OIFrom      string `json:"oiFrom"`
	OITo        string `json:"oiTo"`
	OIChange    string `json:"oiChange"`
	OIChangePct string `json:"oiChangePct"`
	OIDirection string `json:"oiDirection"`
	FundingFrom string `json:"fundingFrom"`
	FundingTo   string `json:"fundingTo"`
	LongPctFrom string `json:"longPctFrom"`
	LongPctTo   string `json:"longPctTo"`
	LongLiq     string `json:"longLiq"`
	ShortLiq    string `json:"shortLiq"`
	Complete    bool   `json:"complete"`
}

type aroundEventDTO struct {
	Kind    string    `json:"kind"`
	Phase   string    `json:"phase,omitempty"`
	Side    string    `json:"side,omitempty"`
	Title   string    `json:"title"`
	Summary string    `json:"summary"`
	At      time.Time `json:"at"`
	Level   string    `json:"level,omitempty"`
	Score   int       `json:"score,omitempty"`
}

type aroundPhaseDTO struct {
	Phase    string            `json:"phase"`
	From     time.Time         `json:"from"`
	To       time.Time         `json:"to"`
	BarCount int               `json:"barCount"`
	Price    aroundPriceDTO    `json:"price"`
	Flow     aroundFlowDTO     `json:"flow"`
	Profile  *aroundProfileDTO `json:"profile,omitempty"`
	Book     *aroundBookDTO    `json:"book,omitempty"`
	Futures  *aroundFuturesDTO `json:"futures,omitempty"`
	Events   []aroundEventDTO  `json:"events,omitempty"`
	Summary  string            `json:"summary"`
	Complete bool              `json:"complete"`
}

type aroundChangeDTO struct {
	Metric    string `json:"metric"`
	Before    string `json:"before"`
	During    string `json:"during"`
	After     string `json:"after"`
	Path      string `json:"path,omitempty"`
	Direction string `json:"direction,omitempty"`
	Summary   string `json:"summary"`
}

type aroundVenueDTO struct {
	Exchange string            `json:"exchange"`
	Symbol   string            `json:"symbol"`
	Interval string            `json:"interval"`
	Phases   []aroundPhaseDTO  `json:"phases"`
	Changes  []aroundChangeDTO `json:"changes"`
	Events   []aroundEventDTO  `json:"events,omitempty"`
	Summary  string            `json:"summary"`
	Error    string            `json:"error,omitempty"`
}

type aroundResponse struct {
	Symbol   string           `json:"symbol"`
	Exchange string           `json:"exchange"`
	At       time.Time        `json:"at"`
	Window   string           `json:"window"`
	During   string           `json:"during"`
	From     time.Time        `json:"from"`
	To       time.Time        `json:"to"`
	AsOf     time.Time        `json:"asOf"`
	Clipped  bool             `json:"clipped"`
	Venues   []aroundVenueDTO `json:"venues"`
	Combined *aroundVenueDTO  `json:"combined,omitempty"`
	Summary  string           `json:"summary"`
	Note     string           `json:"note"`
}

func aroundToDTO(a *domain.AroundReport) aroundResponse {
	if a == nil {
		return aroundResponse{}
	}
	venues := make([]aroundVenueDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		venues = append(venues, aroundVenueToDTO(v))
	}
	out := aroundResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, At: a.At.UTC(),
		Window: a.Window, During: a.During,
		From: a.From.UTC(), To: a.To.UTC(), AsOf: a.AsOf.UTC(), Clipped: a.Clipped,
		Venues: venues, Summary: a.Summary, Note: a.Note,
	}
	if a.Combined != nil {
		c := aroundVenueToDTO(*a.Combined)
		out.Combined = &c
	}
	return out
}

func aroundVenueToDTO(v domain.AroundVenue) aroundVenueDTO {
	phases := make([]aroundPhaseDTO, 0, len(v.Phases))
	for _, p := range v.Phases {
		phases = append(phases, aroundPhaseToDTO(p))
	}
	changes := make([]aroundChangeDTO, 0, len(v.Changes))
	for _, c := range v.Changes {
		changes = append(changes, aroundChangeDTO{
			Metric: c.Metric, Before: formatAroundNum(c.Metric, c.Before),
			During: formatAroundNum(c.Metric, c.During), After: formatAroundNum(c.Metric, c.After),
			Path: c.Path, Direction: c.Direction, Summary: c.Summary,
		})
	}
	evs := aroundEventsToDTO(v.Events)
	return aroundVenueDTO{
		Exchange: string(v.Exchange), Symbol: v.Symbol, Interval: v.Interval,
		Phases: phases, Changes: changes, Events: evs, Summary: v.Summary, Error: v.Error,
	}
}

func aroundPhaseToDTO(p domain.AroundPhase) aroundPhaseDTO {
	out := aroundPhaseDTO{
		Phase: p.Phase, From: p.From.UTC(), To: p.To.UTC(), BarCount: p.BarCount,
		Price: aroundPriceDTO{
			Open: formatHistQty(p.Price.Open), High: formatHistQty(p.Price.High),
			Low: formatHistQty(p.Price.Low), Close: formatHistQty(p.Price.Close),
			Change: domain.FormatSignedQty(p.Price.Change), ChangePct: domain.FormatSignedPct(p.Price.ChangePct),
			Range: formatHistQty(p.Price.Range), RangePct: formatHistQty(p.Price.RangePct),
			Direction: p.Price.Direction,
		},
		Flow: aroundFlowDTO{
			Volume: formatHistQty(p.Flow.Volume), BuyVolume: formatHistQty(p.Flow.BuyVolume),
			SellVolume: formatHistQty(p.Flow.SellVolume), Delta: domain.FormatSignedQty(p.Flow.Delta),
			BuyShare: formatHistQty(p.Flow.BuyShare), Dominant: p.Flow.Dominant,
			BuySellKnown: p.Flow.BuySellKnown, VWAP: formatHistQty(p.Flow.VWAP),
			VsVWAP: p.Flow.VsVWAP, DistancePct: domain.FormatSignedPct(p.Flow.DistancePct),
			TypicalKnown: p.Flow.TypicalKnown,
		},
		Events: aroundEventsToDTO(p.Events), Summary: p.Summary, Complete: p.Complete,
	}
	if p.Flow.TypicalKnown {
		out.Flow.Typical = formatHistQty(p.Flow.Typical)
		out.Flow.VolumeRatio = formatHistQty(p.Flow.VolumeRatio)
		out.Flow.VolumeGrade = p.Flow.VolumeGrade
		out.Flow.TypicalSample = p.Flow.TypicalSample
	}
	if p.Profile != nil {
		out.Profile = &aroundProfileDTO{
			POC: formatHistQty(p.Profile.POC), POCVolume: formatHistQty(p.Profile.POCVolume),
			ValueAreaLow:  formatHistQty(p.Profile.ValueAreaLow),
			ValueAreaHigh: formatHistQty(p.Profile.ValueAreaHigh),
			LastVsArea:    p.Profile.LastVsArea,
		}
	}
	if p.Book != nil {
		out.Book = &aroundBookDTO{
			FromMid: formatHistQty(p.Book.FromMid), ToMid: formatHistQty(p.Book.ToMid),
			MidDelta: domain.FormatSignedQty(p.Book.MidDelta), MidDeltaPct: domain.FormatSignedPct(p.Book.MidDeltaPct),
			BidNotionalDelta: domain.FormatSignedQty(p.Book.BidNotionalDelta),
			AskNotionalDelta: domain.FormatSignedQty(p.Book.AskNotionalDelta),
			ImbalanceDelta:   domain.FormatSignedQty(p.Book.ImbalanceDelta),
			WallsAdded:       p.Book.WallsAdded, WallsRemoved: p.Book.WallsRemoved,
			Summary: p.Book.Summary, Complete: p.Book.Complete,
		}
	}
	if p.Futures != nil {
		out.Futures = &aroundFuturesDTO{
			OIFrom: formatHistQty(p.Futures.OIFrom), OITo: formatHistQty(p.Futures.OITo),
			OIChange:    domain.FormatSignedQty(p.Futures.OIChange),
			OIChangePct: domain.FormatSignedPct(p.Futures.OIChangePct),
			OIDirection: p.Futures.OIDirection,
			FundingFrom: formatHistQty(p.Futures.FundingFrom), FundingTo: formatHistQty(p.Futures.FundingTo),
			LongPctFrom: formatHistQty(p.Futures.LongPctFrom), LongPctTo: formatHistQty(p.Futures.LongPctTo),
			LongLiq: formatHistQty(p.Futures.LongLiq), ShortLiq: formatHistQty(p.Futures.ShortLiq),
			Complete: p.Futures.Complete,
		}
	}
	return out
}

func aroundEventsToDTO(in []domain.AroundEvent) []aroundEventDTO {
	if len(in) == 0 {
		return nil
	}
	out := make([]aroundEventDTO, 0, len(in))
	for _, e := range in {
		row := aroundEventDTO{
			Kind: e.Kind, Phase: e.Phase, Side: e.Side, Title: e.Title,
			Summary: e.Summary, At: e.At.UTC(), Score: e.Score,
		}
		if e.Level > 0 {
			row.Level = formatHistQty(e.Level)
		}
		out = append(out, row)
	}
	return out
}

func formatAroundNum(metric string, v float64) string {
	switch metric {
	case "price", "oi":
		return domain.FormatSignedPct(v)
	case "delta":
		return domain.FormatSignedQty(v)
	default:
		return formatHistQty(v)
	}
}
