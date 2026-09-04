package handler

import (
	"net/http"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// ReviewRadar handles GET /api/v1/market/forensics/radar.
func (h *MarketHandler) ReviewRadar(w http.ResponseWriter, r *http.Request) {
	if h.Radar == nil {
		writeJSON(w, http.StatusOK, radarReviewToDTO(&domain.RadarReview{
			Cases:   []domain.RadarCase{},
			Note:    domain.RadarCaseDisclaimer,
			Summary: "Radar history is not configured on this process.",
			AsOf:    time.Now().UTC(),
		}))
		return
	}
	q := r.URL.Query()
	limit := 0
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, domain.ErrInvalidArgument)
			return
		}
		limit = n
	}
	got, err := h.Radar.Review(r.Context(), q.Get("exchange"), q.Get("symbol"), q.Get("since"), limit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, radarReviewToDTO(got))
}

type radarAfterDTO struct {
	Horizon   string  `json:"horizon"`
	DueAt     string  `json:"dueAt,omitempty"`
	FilledAt  *string `json:"filledAt,omitempty"`
	ReturnPct string  `json:"returnPct,omitempty"`
	Direction string  `json:"direction,omitempty"`
	Price     string  `json:"price,omitempty"`
	Useful    *bool   `json:"useful,omitempty"`
}

type radarCaseDTO struct {
	ID         string                  `json:"id"`
	Exchange   string                  `json:"exchange"`
	Symbol     string                  `json:"symbol"`
	FoundAt    time.Time               `json:"foundAt"`
	FoundPrice string                  `json:"foundPrice,omitempty"`
	Similarity string                  `json:"similarity"`
	Coverage   string                  `json:"coverage,omitempty"`
	Lean       string                  `json:"lean"`
	WhySimilar string                  `json:"whySimilar,omitempty"`
	AfterStory string                  `json:"afterStory,omitempty"`
	Lookback   string                  `json:"lookback,omitempty"`
	Window     string                  `json:"window,omitempty"`
	Compared   []aroundSimilarFieldDTO `json:"compared,omitempty"`
	After      []radarAfterDTO         `json:"after"`
}

type radarHorizonDTO struct {
	Horizon    string `json:"horizon"`
	Sample     int    `json:"sample"`
	LeanRight  int    `json:"leanRight"`
	LeanWrong  int    `json:"leanWrong"`
	Up         int    `json:"up"`
	Down       int    `json:"down"`
	Flat       int    `json:"flat"`
	MedianPct  string `json:"medianPct,omitempty"`
	AveragePct string `json:"averagePct,omitempty"`
	UsefulPct  string `json:"usefulPct,omitempty"`
}

type radarReviewDTO struct {
	Exchange string            `json:"exchange"`
	Symbol   string            `json:"symbol,omitempty"`
	Since    string            `json:"since"`
	From     time.Time         `json:"from"`
	Limit    int               `json:"limit"`
	AsOf     time.Time         `json:"asOf"`
	Cases    []radarCaseDTO    `json:"cases"`
	Horizons []radarHorizonDTO `json:"horizons"`
	Summary  string            `json:"summary"`
	Note     string            `json:"note"`
}

func radarReviewToDTO(a *domain.RadarReview) radarReviewDTO {
	if a == nil {
		return radarReviewDTO{Cases: []radarCaseDTO{}, Note: domain.RadarCaseDisclaimer}
	}
	cases := make([]radarCaseDTO, 0, len(a.Cases))
	for _, c := range a.Cases {
		row := radarCaseDTO{
			ID: c.ID, Exchange: c.Exchange, Symbol: c.Symbol, FoundAt: c.FoundAt.UTC(),
			Similarity: formatHistQty(c.Similarity), Coverage: formatHistQty(c.Coverage),
			Lean: c.Lean, WhySimilar: c.WhySimilar, AfterStory: c.AfterStory,
			Lookback: c.Lookback, Window: c.Window, Compared: aroundSimilarFieldsToDTO(c.Compared),
		}
		if c.FoundPrice > 0 {
			row.FoundPrice = formatHistQty(c.FoundPrice)
		}
		after := make([]radarAfterDTO, 0, len(c.After))
		for _, x := range c.After {
			ad := radarAfterDTO{Horizon: x.Horizon, Direction: x.Direction, Useful: x.Useful}
			if !x.DueAt.IsZero() {
				ad.DueAt = x.DueAt.UTC().Format(time.RFC3339Nano)
			}
			if x.FilledAt != nil {
				s := x.FilledAt.UTC().Format(time.RFC3339Nano)
				ad.FilledAt = &s
				ad.ReturnPct = domain.FormatSignedPct(x.ReturnPct)
				if x.Price > 0 {
					ad.Price = formatHistQty(x.Price)
				}
			}
			after = append(after, ad)
		}
		row.After = after
		cases = append(cases, row)
	}
	hz := make([]radarHorizonDTO, 0, len(a.Horizons))
	for _, h := range a.Horizons {
		row := radarHorizonDTO{
			Horizon: h.Horizon, Sample: h.Sample, LeanRight: h.LeanRight, LeanWrong: h.LeanWrong,
			Up: h.Up, Down: h.Down, Flat: h.Flat,
		}
		if h.Sample > 0 {
			row.MedianPct = domain.FormatSignedPct(h.MedianPct)
			row.AveragePct = domain.FormatSignedPct(h.AveragePct)
		}
		if h.LeanRight+h.LeanWrong > 0 {
			row.UsefulPct = formatHistQty(h.UsefulPct)
		}
		hz = append(hz, row)
	}
	return radarReviewDTO{
		Exchange: a.Exchange, Symbol: a.Symbol, Since: a.Since, From: a.From.UTC(),
		Limit: a.Limit, AsOf: a.AsOf.UTC(), Cases: cases, Horizons: hz,
		Summary: a.Summary, Note: a.Note,
	}
}
