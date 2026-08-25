package handler

import (
	"fmt"
	"net/http"
	"strconv"
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

// GetAroundMoves handles GET /api/v1/market/around/moves.
func (h *MarketHandler) GetAroundMoves(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var minPct float64
	if raw := q.Get("minReturnPct"); raw != "" {
		n, err := parseAroundFloat(raw)
		if err != nil || n < 0 {
			writeError(w, fmt.Errorf("%w: minReturnPct must be a number >= 0", domain.ErrInvalidArgument))
			return
		}
		minPct = n
	}
	var limit int
	if raw := q.Get("limit"); raw != "" {
		n, err := parseAroundInt(raw)
		if err != nil {
			writeError(w, fmt.Errorf("%w: limit must be an integer", domain.ErrInvalidArgument))
			return
		}
		limit = n
	}
	got, err := h.svc.FindAroundMoves(r.Context(), q.Get("exchange"), q.Get("symbol"), q.Get("lookback"), q.Get("interval"), q.Get("direction"), minPct, limit, q.Get("window"), q.Get("during"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, aroundMovesToDTO(got))
}

// GetAroundPrecursors handles GET /api/v1/market/around/precursors.
func (h *MarketHandler) GetAroundPrecursors(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var minPct float64
	if raw := q.Get("minReturnPct"); raw != "" {
		n, err := parseAroundFloat(raw)
		if err != nil || n < 0 {
			writeError(w, fmt.Errorf("%w: minReturnPct must be a number >= 0", domain.ErrInvalidArgument))
			return
		}
		minPct = n
	}
	var limit int
	if raw := q.Get("limit"); raw != "" {
		n, err := parseAroundInt(raw)
		if err != nil {
			writeError(w, fmt.Errorf("%w: limit must be an integer", domain.ErrInvalidArgument))
			return
		}
		limit = n
	}
	got, err := h.svc.GetAroundPrecursors(r.Context(), q.Get("exchange"), q.Get("symbol"), q.Get("lookback"), q.Get("interval"), q.Get("direction"), minPct, limit, q.Get("window"), q.Get("during"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, aroundPrecursorsToDTO(got))
}

// GetAroundSimilar handles GET /api/v1/market/around/similar.
func (h *MarketHandler) GetAroundSimilar(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var minPct float64
	if raw := q.Get("minReturnPct"); raw != "" {
		n, err := parseAroundFloat(raw)
		if err != nil || n < 0 {
			writeError(w, fmt.Errorf("%w: minReturnPct must be a number >= 0", domain.ErrInvalidArgument))
			return
		}
		minPct = n
	}
	var limit int
	if raw := q.Get("limit"); raw != "" {
		n, err := parseAroundInt(raw)
		if err != nil {
			writeError(w, fmt.Errorf("%w: limit must be an integer", domain.ErrInvalidArgument))
			return
		}
		limit = n
	}
	got, err := h.svc.GetAroundSimilar(r.Context(), q.Get("exchange"), q.Get("symbol"), q.Get("lookback"), q.Get("interval"), q.Get("direction"), minPct, limit, q.Get("window"), q.Get("during"), q.Get("fields"), q.Get("weights"), q.Get("minCoverage"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, aroundSimilarToDTO(got))
}

// GetAroundCompare handles GET /api/v1/market/around/compare.
func (h *MarketHandler) GetAroundCompare(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	fromRaw, toRaw := q.Get("from"), q.Get("to")
	if fromRaw == "" || toRaw == "" {
		writeError(w, fmt.Errorf("%w: from and to times are required", domain.ErrInvalidArgument))
		return
	}
	from, err := parseTimeParam(fromRaw)
	if err != nil {
		writeError(w, err)
		return
	}
	to, err := parseTimeParam(toRaw)
	if err != nil {
		writeError(w, err)
		return
	}
	got, err := h.svc.CompareAround(r.Context(), q.Get("exchange"), q.Get("symbol"), q.Get("window"), q.Get("during"), from, to)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, aroundCompareToDTO(got))
}

type aroundCompareDeltaDTO struct {
	Metric    string `json:"metric"`
	Phase     string `json:"phase,omitempty"`
	From      string `json:"from"`
	To        string `json:"to"`
	Change    string `json:"change"`
	ChangePct string `json:"changePct"`
	Direction string `json:"direction"`
	Summary   string `json:"summary"`
}

type aroundComparePhaseDTO struct {
	Phase   string                  `json:"phase"`
	From    aroundPhaseDTO          `json:"from"`
	To      aroundPhaseDTO          `json:"to"`
	Deltas  []aroundCompareDeltaDTO `json:"deltas"`
	Summary string                  `json:"summary"`
}

type aroundCompareVenueDTO struct {
	Exchange string                  `json:"exchange"`
	Symbol   string                  `json:"symbol"`
	Phases   []aroundComparePhaseDTO `json:"phases"`
	State    []aroundCompareDeltaDTO `json:"state,omitempty"`
	Book     *aroundBookDTO          `json:"book,omitempty"`
	Summary  string                  `json:"summary"`
	Error    string                  `json:"error,omitempty"`
}

type aroundCompareResponse struct {
	Symbol   string                  `json:"symbol"`
	Exchange string                  `json:"exchange"`
	FromAt   time.Time               `json:"fromAt"`
	ToAt     time.Time               `json:"toAt"`
	Window   string                  `json:"window"`
	During   string                  `json:"during"`
	AsOf     time.Time               `json:"asOf"`
	FromMove *aroundResponse         `json:"fromMove,omitempty"`
	ToMove   *aroundResponse         `json:"toMove,omitempty"`
	Venues   []aroundCompareVenueDTO `json:"venues"`
	Combined *aroundCompareVenueDTO  `json:"combined,omitempty"`
	Summary  string                  `json:"summary"`
	Note     string                  `json:"note"`
}

func aroundCompareToDTO(a *domain.AroundCompareReport) aroundCompareResponse {
	if a == nil {
		return aroundCompareResponse{}
	}
	venues := make([]aroundCompareVenueDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		venues = append(venues, aroundCompareVenueToDTO(v))
	}
	out := aroundCompareResponse{
		Symbol: a.Symbol, Exchange: a.Exchange,
		FromAt: a.FromAt.UTC(), ToAt: a.ToAt.UTC(),
		Window: a.Window, During: a.During, AsOf: a.AsOf.UTC(),
		Venues: venues, Summary: a.Summary, Note: a.Note,
	}
	if a.FromMove != nil {
		m := aroundToDTO(a.FromMove)
		out.FromMove = &m
	}
	if a.ToMove != nil {
		m := aroundToDTO(a.ToMove)
		out.ToMove = &m
	}
	if a.Combined != nil {
		c := aroundCompareVenueToDTO(*a.Combined)
		out.Combined = &c
	}
	return out
}

func aroundCompareVenueToDTO(v domain.AroundCompareVenue) aroundCompareVenueDTO {
	phases := make([]aroundComparePhaseDTO, 0, len(v.Phases))
	for _, p := range v.Phases {
		phases = append(phases, aroundComparePhaseDTO{
			Phase: p.Phase, From: aroundPhaseToDTO(p.From), To: aroundPhaseToDTO(p.To),
			Deltas: aroundCompareDeltasToDTO(p.Deltas), Summary: p.Summary,
		})
	}
	out := aroundCompareVenueDTO{
		Exchange: string(v.Exchange), Symbol: v.Symbol,
		Phases: phases, State: aroundCompareDeltasToDTO(v.State),
		Summary: v.Summary, Error: v.Error,
	}
	if v.Book != nil {
		out.Book = &aroundBookDTO{
			FromMid: formatHistQty(v.Book.FromMid), ToMid: formatHistQty(v.Book.ToMid),
			MidDelta: domain.FormatSignedQty(v.Book.MidDelta), MidDeltaPct: domain.FormatSignedPct(v.Book.MidDeltaPct),
			BidNotionalDelta: domain.FormatSignedQty(v.Book.BidNotionalDelta),
			AskNotionalDelta: domain.FormatSignedQty(v.Book.AskNotionalDelta),
			ImbalanceDelta:   domain.FormatSignedQty(v.Book.ImbalanceDelta),
			WallsAdded:       v.Book.WallsAdded, WallsRemoved: v.Book.WallsRemoved,
			Summary: v.Book.Summary, Complete: v.Book.Complete,
		}
	}
	return out
}

func aroundCompareDeltasToDTO(in []domain.AroundCompareDelta) []aroundCompareDeltaDTO {
	if len(in) == 0 {
		return nil
	}
	out := make([]aroundCompareDeltaDTO, 0, len(in))
	for _, d := range in {
		from, to := formatAroundCompareValue(d.Metric, d.From), formatAroundCompareValue(d.Metric, d.To)
		out = append(out, aroundCompareDeltaDTO{
			Metric: d.Metric, Phase: d.Phase, From: from, To: to,
			Change: domain.FormatSignedQty(d.Change), ChangePct: domain.FormatSignedPct(d.ChangePct),
			Direction: d.Direction, Summary: d.Summary,
		})
	}
	return out
}

func formatAroundCompareValue(metric string, v float64) string {
	switch metric {
	case domain.AroundCompareMetricMove, domain.AroundCompareMetricRange,
		domain.AroundCompareMetricOIChange, domain.AroundCompareMetricFunding,
		domain.AroundCompareMetricLongPct:
		return domain.FormatSignedPct(v)
	case domain.AroundCompareMetricDelta, domain.AroundCompareMetricBookBid, domain.AroundCompareMetricBookAsk:
		return domain.FormatSignedQty(v)
	default:
		return formatHistQty(v)
	}
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

type aroundMoveHitDTO struct {
	At        time.Time       `json:"at"`
	Until     time.Time       `json:"until"`
	Direction string          `json:"direction"`
	Open      string          `json:"open"`
	High      string          `json:"high"`
	Low       string          `json:"low"`
	Close     string          `json:"close"`
	ReturnPct string          `json:"returnPct"`
	Volume    string          `json:"volume"`
	Bars      int             `json:"bars"`
	Grade     string          `json:"grade"`
	During    string          `json:"during"`
	Title     string          `json:"title"`
	Summary   string          `json:"summary"`
	Around    *aroundResponse `json:"around,omitempty"`
}

type aroundMovesResponse struct {
	Symbol       string             `json:"symbol"`
	Exchange     string             `json:"exchange"`
	Lookback     string             `json:"lookback"`
	Interval     string             `json:"interval"`
	Direction    string             `json:"direction"`
	MinReturnPct string             `json:"minReturnPct"`
	From         time.Time          `json:"from"`
	To           time.Time          `json:"to"`
	AsOf         time.Time          `json:"asOf"`
	Moves        []aroundMoveHitDTO `json:"moves"`
	Summary      string             `json:"summary"`
	Note         string             `json:"note"`
}

func aroundMovesToDTO(a *domain.AroundMovesReport) aroundMovesResponse {
	if a == nil {
		return aroundMovesResponse{}
	}
	moves := make([]aroundMoveHitDTO, 0, len(a.Moves))
	for _, m := range a.Moves {
		row := aroundMoveHitDTO{
			At: m.At.UTC(), Until: m.Until.UTC(), Direction: m.Direction,
			Open: formatHistQty(m.Open), High: formatHistQty(m.High),
			Low: formatHistQty(m.Low), Close: formatHistQty(m.Close),
			ReturnPct: domain.FormatSignedPct(m.ReturnPct), Volume: formatHistQty(m.Volume),
			Bars: m.Bars, Grade: m.Grade, During: m.During, Title: m.Title, Summary: m.Summary,
		}
		if m.Around != nil {
			rep := aroundToDTO(m.Around)
			row.Around = &rep
		}
		moves = append(moves, row)
	}
	return aroundMovesResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, Lookback: a.Lookback, Interval: a.Interval,
		Direction: a.Direction, MinReturnPct: formatHistQty(a.MinReturnPct),
		From: a.From.UTC(), To: a.To.UTC(), AsOf: a.AsOf.UTC(),
		Moves: moves, Summary: a.Summary, Note: a.Note,
	}
}

func aroundPrecursorsToDTO(a *domain.AroundPrecursorReport) aroundPrecursorsResponse {
	if a == nil {
		return aroundPrecursorsResponse{}
	}
	pats := make([]aroundPrecursorPatternDTO, 0, len(a.Patterns))
	for _, p := range a.Patterns {
		pats = append(pats, aroundPrecursorPatternDTO{
			Metric: p.Metric, Label: p.Label, Side: p.Side,
			Hits: p.Hits, Sample: p.Sample, SharePct: formatHistQty(p.SharePct),
			Median: formatHistQty(p.Median), Common: p.Common, Summary: p.Summary,
		})
	}
	moves := make([]aroundMoveHitDTO, 0, len(a.Moves))
	for _, m := range a.Moves {
		moves = append(moves, aroundMoveHitDTO{
			At: m.At.UTC(), Until: m.Until.UTC(), Direction: m.Direction,
			Open: formatHistQty(m.Open), High: formatHistQty(m.High),
			Low: formatHistQty(m.Low), Close: formatHistQty(m.Close),
			ReturnPct: domain.FormatSignedPct(m.ReturnPct), Volume: formatHistQty(m.Volume),
			Bars: m.Bars, Grade: m.Grade, During: m.During, Title: m.Title, Summary: m.Summary,
		})
	}
	combos := make([]aroundPrecursorComboDTO, 0, len(a.Combos))
	for _, c := range a.Combos {
		combos = append(combos, aroundPrecursorComboDTO{
			Metrics: append([]string(nil), c.Metrics...), Labels: append([]string(nil), c.Labels...),
			Title: c.Title, UpHits: c.UpHits, DownHits: c.DownHits,
			UpSample: c.UpSample, DownSample: c.DownSample, Hits: c.Hits, Sample: c.Sample,
			UpSharePct: formatHistQty(c.UpSharePct), DownSharePct: formatHistQty(c.DownSharePct),
			SharePct: formatHistQty(c.SharePct), Lean: c.Lean, Common: c.Common, Summary: c.Summary,
		})
	}
	return aroundPrecursorsResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, Lookback: a.Lookback, Interval: a.Interval,
		Direction: a.Direction, MinReturnPct: formatHistQty(a.MinReturnPct),
		From: a.From.UTC(), To: a.To.UTC(), AsOf: a.AsOf.UTC(),
		UpMoves: a.UpMoves, DownMoves: a.DownMoves, Sampled: a.Sampled,
		Patterns: pats, Combos: combos, Moves: moves, Summary: a.Summary, Note: a.Note,
	}
}

type aroundPrecursorPatternDTO struct {
	Metric   string `json:"metric"`
	Label    string `json:"label"`
	Side     string `json:"side"`
	Hits     int    `json:"hits"`
	Sample   int    `json:"sample"`
	SharePct string `json:"sharePct"`
	Median   string `json:"median"`
	Common   bool   `json:"common"`
	Summary  string `json:"summary"`
}

type aroundPrecursorComboDTO struct {
	Metrics      []string `json:"metrics"`
	Labels       []string `json:"labels"`
	Title        string   `json:"title"`
	UpHits       int      `json:"upHits"`
	DownHits     int      `json:"downHits"`
	UpSample     int      `json:"upSample"`
	DownSample   int      `json:"downSample"`
	Hits         int      `json:"hits"`
	Sample       int      `json:"sample"`
	UpSharePct   string   `json:"upSharePct"`
	DownSharePct string   `json:"downSharePct"`
	SharePct     string   `json:"sharePct"`
	Lean         string   `json:"lean"`
	Common       bool     `json:"common"`
	Summary      string   `json:"summary"`
}

type aroundPrecursorsResponse struct {
	Symbol       string                      `json:"symbol"`
	Exchange     string                      `json:"exchange"`
	Lookback     string                      `json:"lookback"`
	Interval     string                      `json:"interval"`
	Direction    string                      `json:"direction"`
	MinReturnPct string                      `json:"minReturnPct"`
	From         time.Time                   `json:"from"`
	To           time.Time                   `json:"to"`
	AsOf         time.Time                   `json:"asOf"`
	UpMoves      int                         `json:"upMoves"`
	DownMoves    int                         `json:"downMoves"`
	Sampled      int                         `json:"sampled"`
	Patterns     []aroundPrecursorPatternDTO `json:"patterns"`
	Combos       []aroundPrecursorComboDTO   `json:"combos"`
	Moves        []aroundMoveHitDTO          `json:"moves"`
	Summary      string                      `json:"summary"`
	Note         string                      `json:"note"`
}

func aroundSimilarToDTO(a *domain.AroundSimilarReport) aroundSimilarResponse {
	if a == nil {
		return aroundSimilarResponse{}
	}
	return aroundSimilarResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, Lookback: a.Lookback, Window: a.Window,
		Interval: a.Interval, Fields: append([]string(nil), a.Fields...),
		Weights: aroundSimilarWeightsToDTO(a.Weights), MinCoverage: formatHistQty(a.MinCoverage),
		MinReturnPct: formatHistQty(a.MinReturnPct), AsOf: a.AsOf.UTC(),
		Current: aroundPhaseToDTO(a.Current),
		Matches: aroundSimilarHitsToDTO(a.Matches), Skipped: aroundSimilarHitsToDTO(a.Skipped),
		Events: a.Events, UpAfter: a.UpAfter, DownAfter: a.DownAfter, MedianAfterPct: domain.FormatSignedPct(a.MedianAfterPct),
		AfterHorizons: aroundSimilarHorizonsToDTO(a.AfterHorizons),
		Summary:       a.Summary, Note: a.Note,
	}
}

func aroundSimilarHitsToDTO(in []domain.AroundSimilarHit) []aroundSimilarHitDTO {
	out := make([]aroundSimilarHitDTO, 0, len(in))
	for _, m := range in {
		cmp := make([]aroundSimilarFieldDTO, 0, len(m.Compared))
		for _, c := range m.Compared {
			row := aroundSimilarFieldDTO{Name: c.Name, Used: c.Used, Weight: formatHistQty(c.Weight)}
			if c.Used {
				row.Score = formatHistQty(c.Score)
			}
			cmp = append(cmp, row)
		}
		out = append(out, aroundSimilarHitDTO{
			At: m.Move.At.UTC(), Until: m.Move.Until.UTC(), Direction: m.Move.Direction,
			ReturnPct: domain.FormatSignedPct(m.Move.ReturnPct), Grade: m.Move.Grade,
			Similarity: formatHistQty(m.Similarity), Coverage: formatHistQty(m.Coverage),
			Compared: cmp, Used: append([]string(nil), m.Used...), Missing: append([]string(nil), m.Missing...),
			Matches: append([]string(nil), m.Matches...),
			Before:  aroundPhaseToDTO(m.Before), After: aroundPhaseToDTO(m.After),
			DataFrom: m.DataFrom.UTC(), DataTo: m.DataTo.UTC(),
			AfterReturnPct: domain.FormatSignedPct(m.AfterReturnPct), AfterDirection: m.AfterDirection,
			Summary: m.Summary,
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func aroundSimilarHorizonsToDTO(in []domain.AroundSimilarHorizonStat) []aroundSimilarHorizonDTO {
	if len(in) == 0 {
		return nil
	}
	out := make([]aroundSimilarHorizonDTO, 0, len(in))
	for _, h := range in {
		row := aroundSimilarHorizonDTO{Horizon: h.Horizon, Sample: h.Sample, Events: h.Events, Up: h.Up, Down: h.Down}
		if h.Sample > 0 {
			row.AveragePct = domain.FormatSignedPct(h.AveragePct)
			row.MedianPct = domain.FormatSignedPct(h.MedianPct)
		}
		out = append(out, row)
	}
	return out
}

func aroundSimilarWeightsToDTO(in []domain.AroundSimilarFieldScore) []aroundSimilarFieldDTO {
	out := make([]aroundSimilarFieldDTO, 0, len(in))
	for _, c := range in {
		out = append(out, aroundSimilarFieldDTO{
			Name: c.Name, Used: true, Weight: formatHistQty(c.Weight),
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

type aroundSimilarFieldDTO struct {
	Name   string `json:"name"`
	Used   bool   `json:"used"`
	Score  string `json:"score,omitempty"`
	Weight string `json:"weight"`
}

type aroundSimilarHitDTO struct {
	At             time.Time               `json:"at"`
	Until          time.Time               `json:"until"`
	Direction      string                  `json:"direction"`
	ReturnPct      string                  `json:"returnPct"`
	Grade          string                  `json:"grade"`
	Similarity     string                  `json:"similarity"`
	Coverage       string                  `json:"coverage"`
	Compared       []aroundSimilarFieldDTO `json:"compared"`
	Used           []string                `json:"used,omitempty"`
	Missing        []string                `json:"missing,omitempty"`
	Matches        []string                `json:"matches,omitempty"`
	Before         aroundPhaseDTO          `json:"before"`
	After          aroundPhaseDTO          `json:"after"`
	DataFrom       time.Time               `json:"dataFrom"`
	DataTo         time.Time               `json:"dataTo"`
	AfterReturnPct string                  `json:"afterReturnPct"`
	AfterDirection string                  `json:"afterDirection"`
	Summary        string                  `json:"summary"`
}

type aroundSimilarResponse struct {
	Symbol         string                    `json:"symbol"`
	Exchange       string                    `json:"exchange"`
	Lookback       string                    `json:"lookback"`
	Window         string                    `json:"window"`
	Interval       string                    `json:"interval"`
	Fields         []string                  `json:"fields"`
	Weights        []aroundSimilarFieldDTO   `json:"weights,omitempty"`
	MinCoverage    string                    `json:"minCoverage"`
	MinReturnPct   string                    `json:"minReturnPct"`
	AsOf           time.Time                 `json:"asOf"`
	Current        aroundPhaseDTO            `json:"current"`
	Matches        []aroundSimilarHitDTO     `json:"matches"`
	Skipped        []aroundSimilarHitDTO     `json:"skipped,omitempty"`
	Events         int                       `json:"events"`
	UpAfter        int                       `json:"upAfter"`
	DownAfter      int                       `json:"downAfter"`
	MedianAfterPct string                    `json:"medianAfterPct"`
	AfterHorizons  []aroundSimilarHorizonDTO `json:"afterHorizons,omitempty"`
	Summary        string                    `json:"summary"`
	Note           string                    `json:"note"`
}

type aroundSimilarHorizonDTO struct {
	Horizon    string `json:"horizon"`
	Sample     int    `json:"sample"`
	Events     int    `json:"events"`
	Up         int    `json:"up"`
	Down       int    `json:"down"`
	AveragePct string `json:"averagePct,omitempty"`
	MedianPct  string `json:"medianPct,omitempty"`
}

func parseAroundFloat(raw string) (float64, error) {
	return strconv.ParseFloat(raw, 64)
}

func parseAroundInt(raw string) (int, error) {
	return strconv.Atoi(raw)
}
