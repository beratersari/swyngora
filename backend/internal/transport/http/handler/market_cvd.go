package handler

import (
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetCVD handles GET /api/v1/market/cvd.
func (h *MarketHandler) GetCVD(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.GetCVD(r.Context(), q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cvdToDTO(got))
}

type cvdShareDTO struct {
	Exchange string `json:"exchange"`
	Delta    string `json:"delta"`
	CVD      string `json:"cvd"`
	SharePct string `json:"sharePct"`
}

type cvdDivergenceDTO struct {
	Kind            string    `json:"kind,omitempty"`
	VsPrice         string    `json:"vsPrice,omitempty"`
	Title           string    `json:"title,omitempty"`
	Summary         string    `json:"summary,omitempty"`
	Bars            int       `json:"bars"`
	LastAt          time.Time `json:"lastAt,omitempty"`
	Since           time.Time `json:"since,omitempty"`
	Duration        string    `json:"duration,omitempty"`
	DurationSeconds int       `json:"durationSeconds,omitempty"`
	PriceMovePct    string    `json:"priceMovePct,omitempty"`
	CVDMove         string    `json:"cvdMove,omitempty"`
	CVDMovePct      string    `json:"cvdMovePct,omitempty"`
}

type cvdVenueSplitDTO struct {
	Alignment     string `json:"alignment"`
	Binance       string `json:"binance"`
	Bybit         string `json:"bybit"`
	BinanceChange string `json:"binanceChange"`
	BybitChange   string `json:"bybitChange"`
	Title         string `json:"title,omitempty"`
	Summary       string `json:"summary,omitempty"`
}

type cvdPointDTO struct {
	Time           time.Time     `json:"time"`
	Price          string        `json:"price"`
	PriceChangePct string        `json:"priceChangePct"`
	BuyNotional    string        `json:"buyNotional"`
	SellNotional   string        `json:"sellNotional"`
	Delta          string        `json:"delta"`
	CVD            string        `json:"cvd"`
	VsPrice        string        `json:"vsPrice"`
	Divergence     string        `json:"divergence,omitempty"`
	Shares         []cvdShareDTO `json:"shares,omitempty"`
}

type cvdWindowDTO struct {
	Window         string        `json:"window"`
	CVDChange      string        `json:"cvdChange"`
	CVDChangePct   string        `json:"cvdChangePct"`
	PriceChangePct string        `json:"priceChangePct"`
	BuyNotional    string        `json:"buyNotional"`
	SellNotional   string        `json:"sellNotional"`
	VsPrice        string        `json:"vsPrice"`
	Divergence     string        `json:"divergence,omitempty"`
	Title          string        `json:"title"`
	Summary        string        `json:"summary"`
	Complete       bool              `json:"complete"`
	Contributions  []cvdShareDTO     `json:"contributions,omitempty"`
	VenueSplit     *cvdVenueSplitDTO `json:"venueSplit,omitempty"`
}

type cvdVenueDTO struct {
	Exchange      string             `json:"exchange"`
	Symbol        string             `json:"symbol"`
	Points        []cvdPointDTO      `json:"points"`
	Windows       []cvdWindowDTO     `json:"windows"`
	LastCVD       string             `json:"lastCvd"`
	LastPrice     string             `json:"lastPrice"`
	Contributions []cvdShareDTO      `json:"contributions,omitempty"`
	OverlapFrom   *time.Time         `json:"overlapFrom,omitempty"`
	OverlapTo     *time.Time         `json:"overlapTo,omitempty"`
	Divergence    cvdDivergenceDTO   `json:"divergence"`
	VenueSplit    *cvdVenueSplitDTO  `json:"venueSplit,omitempty"`
	Summary       string             `json:"summary"`
	Error         string             `json:"error,omitempty"`
	Complete      bool               `json:"complete"`
}

type cvdResponse struct {
	Symbol   string        `json:"symbol"`
	Exchange string        `json:"exchange"`
	AsOf     time.Time     `json:"asOf"`
	Venues   []cvdVenueDTO `json:"venues"`
	Combined *cvdVenueDTO  `json:"combined,omitempty"`
	Summary  string        `json:"summary"`
	Note     string        `json:"note"`
}

func cvdToDTO(a *domain.CVDReport) cvdResponse {
	if a == nil {
		return cvdResponse{}
	}
	venues := make([]cvdVenueDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		venues = append(venues, cvdVenueToDTO(v))
	}
	out := cvdResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, AsOf: a.AsOf.UTC(),
		Venues: venues, Summary: a.Summary, Note: a.Note,
	}
	if a.Combined != nil {
		c := cvdVenueToDTO(*a.Combined)
		out.Combined = &c
	}
	return out
}

func cvdVenueToDTO(v domain.CVDVenueSeries) cvdVenueDTO {
	pts := make([]cvdPointDTO, 0, len(v.Points))
	for _, p := range v.Points {
		pts = append(pts, cvdPointDTO{
			Time: p.Time.UTC(), Price: formatHistQty(p.Price),
			PriceChangePct: domain.FormatSignedPct(p.PriceChangePct),
			BuyNotional:    formatHistQty(p.BuyNotional), SellNotional: formatHistQty(p.SellNotional),
			Delta: domain.FormatSignedQty(p.Delta), CVD: domain.FormatSignedQty(p.CVD),
			VsPrice: p.VsPrice, Divergence: p.Divergence, Shares: cvdSharesToDTO(p.Shares),
		})
	}
	wins := make([]cvdWindowDTO, 0, len(v.Windows))
	for _, w := range v.Windows {
		wins = append(wins, cvdWindowDTO{
			Window: w.Window, CVDChange: domain.FormatSignedQty(w.CVDChange),
			CVDChangePct:   domain.FormatSignedPct(w.CVDChangePct),
			PriceChangePct: domain.FormatSignedPct(w.PriceChangePct),
			BuyNotional:    formatHistQty(w.BuyNotional), SellNotional: formatHistQty(w.SellNotional),
			VsPrice: w.VsPrice, Divergence: w.Divergence,
			Title: w.Title, Summary: w.Summary, Complete: w.Complete,
			Contributions: cvdSharesToDTO(w.Contributions),
			VenueSplit:    cvdVenueSplitToDTO(w.VenueSplit),
		})
	}
	div := cvdDivergenceDTO{
		Kind: v.Divergence.Kind, VsPrice: v.Divergence.VsPrice,
		Title: v.Divergence.Title, Summary: v.Divergence.Summary,
		Bars: v.Divergence.Bars, Duration: v.Divergence.Duration,
		DurationSeconds: v.Divergence.DurationSeconds,
		PriceMovePct:    domain.FormatSignedPct(v.Divergence.PriceMovePct),
		CVDMove:         domain.FormatSignedQty(v.Divergence.CVDMove),
		CVDMovePct:      domain.FormatSignedPct(v.Divergence.CVDMovePct),
	}
	if !v.Divergence.LastAt.IsZero() {
		div.LastAt = v.Divergence.LastAt.UTC()
	}
	if !v.Divergence.Since.IsZero() {
		div.Since = v.Divergence.Since.UTC()
	}
	out := cvdVenueDTO{
		Exchange: string(v.Exchange), Symbol: v.Symbol, Points: pts, Windows: wins,
		LastCVD: domain.FormatSignedQty(v.LastCVD), LastPrice: formatHistQty(v.LastPrice),
		Contributions: cvdSharesToDTO(v.Contributions),
		Divergence:    div,
		VenueSplit:    cvdVenueSplitToDTO(v.VenueSplit),
		Summary:       v.Summary, Error: v.Error, Complete: v.Complete,
	}
	if v.OverlapFrom != nil && !v.OverlapFrom.IsZero() {
		t := v.OverlapFrom.UTC()
		out.OverlapFrom = &t
	}
	if v.OverlapTo != nil && !v.OverlapTo.IsZero() {
		t := v.OverlapTo.UTC()
		out.OverlapTo = &t
	}
	return out
}

func cvdVenueSplitToDTO(in *domain.CVDVenueSplit) *cvdVenueSplitDTO {
	if in == nil {
		return nil
	}
	return &cvdVenueSplitDTO{
		Alignment:     in.Alignment,
		Binance:       in.Binance,
		Bybit:         in.Bybit,
		BinanceChange: domain.FormatSignedQty(in.BinanceChange),
		BybitChange:   domain.FormatSignedQty(in.BybitChange),
		Title:         in.Title,
		Summary:       in.Summary,
	}
}

func cvdSharesToDTO(in []domain.CVDShare) []cvdShareDTO {
	if len(in) == 0 {
		return nil
	}
	out := make([]cvdShareDTO, 0, len(in))
	for _, s := range in {
		out = append(out, cvdShareDTO{
			Exchange: string(s.Exchange),
			Delta:    domain.FormatSignedQty(s.Delta),
			CVD:      domain.FormatSignedQty(s.CVD),
			SharePct: domain.FormatSignedPct(s.SharePct),
		})
	}
	return out
}
