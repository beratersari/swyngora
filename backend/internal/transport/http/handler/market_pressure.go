package handler

import (
	"net/http"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetMarketPressure handles GET /api/v1/market/pressure.
func (h *MarketHandler) GetMarketPressure(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.GetMarketPressure(r.Context(), q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, marketPressureToDTO(got))
}

type marketPressureComponentDTO struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Value  string `json:"value"`
	Weight string `json:"weight,omitempty"`
	Used   bool   `json:"used"`
	Lean   string `json:"lean,omitempty"`
	Detail string `json:"detail,omitempty"`
}

type marketPressureReadingDTO struct {
	Value      string                       `json:"value"`
	Lean       string                       `json:"lean"`
	Agreement  string                       `json:"agreement,omitempty"`
	Components []marketPressureComponentDTO `json:"components"`
	Headline   string                       `json:"headline,omitempty"`
	Detail     string                       `json:"detail,omitempty"`
}

type marketPressureWindowDTO struct {
	Window     string `json:"window"`
	Value      string `json:"value"`
	PastValue  string `json:"pastValue,omitempty"`
	Change     string `json:"change,omitempty"`
	Lean       string `json:"lean,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Complete   bool   `json:"complete"`
	SampleTime string `json:"sampleTime,omitempty"`
}

type marketPressureResponse struct {
	Symbol   string                    `json:"symbol"`
	Exchange string                    `json:"exchange"`
	AsOf     string                    `json:"asOf,omitempty"`
	Current  marketPressureReadingDTO  `json:"current"`
	Change1h string                    `json:"change1h,omitempty"`
	Windows  []marketPressureWindowDTO `json:"windows,omitempty"`
	Note     string                    `json:"note"`
}

func marketPressureToDTO(a *domain.MarketPressureReport) marketPressureResponse {
	if a == nil {
		return marketPressureResponse{}
	}
	asOf := ""
	if !a.AsOf.IsZero() {
		asOf = a.AsOf.UTC().Format(time.RFC3339Nano)
	}
	out := marketPressureResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, AsOf: asOf,
		Current:  marketPressureReadingDTOFrom(a.Current),
		Change1h: a.Change1h, Note: a.Note,
	}
	for _, w := range a.Windows {
		row := marketPressureWindowDTO{
			Window: w.Window, Value: w.ValueText, PastValue: w.PastText,
			Change: w.ChangeText, Lean: w.Lean, Kind: w.Kind, Complete: w.Complete,
		}
		if !w.SampleTime.IsZero() {
			row.SampleTime = w.SampleTime.UTC().Format(time.RFC3339Nano)
		}
		out.Windows = append(out.Windows, row)
	}
	return out
}

func marketPressureReadingDTOFrom(r domain.MarketPressureReading) marketPressureReadingDTO {
	comps := make([]marketPressureComponentDTO, 0, len(r.Components))
	for _, c := range r.Components {
		comps = append(comps, marketPressureComponentDTO{
			ID: c.ID, Label: c.Label, Value: domain.FormatMarketPressureValue(c.Value),
			Weight: strconv.FormatFloat(c.Weight, 'f', 2, 64), Used: c.Used, Lean: c.Lean, Detail: c.Detail,
		})
	}
	return marketPressureReadingDTO{
		Value: r.ValueText, Lean: r.Lean, Agreement: r.Agreement,
		Components: comps, Headline: r.Headline, Detail: r.Detail,
	}
}
