package handler

import (
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetBasis handles GET /api/v1/market/basis.
func (h *MarketHandler) GetBasis(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.GetBasis(r.Context(), q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, basisToDTO(got))
}

type basisLevelDTO struct {
	Futures  string `json:"futures"`
	Spot     string `json:"spot"`
	Delta    string `json:"delta"`
	DeltaPct string `json:"deltaPct"`
	Kind     string `json:"kind"`
	Source   string `json:"source"`
}

type basisWindowDTO struct {
	Window     string `json:"window"`
	PastPct    string `json:"pastPct,omitempty"`
	ChangePct  string `json:"changePct,omitempty"`
	Trend      string `json:"trend"`
	Complete   bool   `json:"complete"`
	SampleTime string `json:"sampleTime,omitempty"`
}

type basisVenueDTO struct {
	Exchange string           `json:"exchange"`
	Symbol   string           `json:"symbol"`
	Last     basisLevelDTO    `json:"last"`
	Mark     basisLevelDTO    `json:"mark"`
	Windows  []basisWindowDTO `json:"windows"`
	Trend    string           `json:"trend"`
	Summary  string           `json:"summary"`
	Error    string           `json:"error,omitempty"`
}

type basisAgreementDTO struct {
	Alignment string `json:"alignment"`
	Title     string `json:"title"`
	Summary   string `json:"summary"`
}

type basisResponse struct {
	Symbol    string             `json:"symbol"`
	Exchange  string             `json:"exchange"`
	AsOf      time.Time          `json:"asOf"`
	Venues    []basisVenueDTO    `json:"venues"`
	Agreement *basisAgreementDTO `json:"agreement,omitempty"`
	Note      string             `json:"note"`
}

func basisToDTO(a *domain.BasisReport) basisResponse {
	if a == nil {
		return basisResponse{}
	}
	venues := make([]basisVenueDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		venues = append(venues, basisVenueToDTO(v))
	}
	out := basisResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, AsOf: a.AsOf.UTC(),
		Venues: venues, Note: a.Note,
	}
	if a.Agreement != nil {
		out.Agreement = &basisAgreementDTO{
			Alignment: a.Agreement.Alignment, Title: a.Agreement.Title, Summary: a.Agreement.Summary,
		}
	}
	return out
}

func basisVenueToDTO(v domain.BasisVenueReport) basisVenueDTO {
	wins := make([]basisWindowDTO, 0, len(v.Windows))
	for _, w := range v.Windows {
		row := basisWindowDTO{Window: w.Window, Trend: w.Trend, Complete: w.Complete}
		if w.Complete {
			row.PastPct = domain.FormatSignedPct(w.PastPct)
			row.ChangePct = domain.FormatSignedPct(w.ChangePct)
			if !w.SampleTime.IsZero() {
				row.SampleTime = w.SampleTime.UTC().Format(time.RFC3339Nano)
			}
		}
		wins = append(wins, row)
	}
	return basisVenueDTO{
		Exchange: string(v.Exchange), Symbol: v.Symbol,
		Last: basisLevelToDTO(v.Last), Mark: basisLevelToDTO(v.Mark),
		Windows: wins, Trend: v.Trend, Summary: v.Summary, Error: v.Error,
	}
}

func basisLevelToDTO(lv domain.BasisLevel) basisLevelDTO {
	return basisLevelDTO{
		Futures: formatHistQty(lv.Futures), Spot: formatHistQty(lv.Spot),
		Delta: domain.FormatSignedQty(lv.Delta), DeltaPct: domain.FormatSignedPct(lv.DeltaPct),
		Kind: lv.Kind, Source: lv.Source,
	}
}
