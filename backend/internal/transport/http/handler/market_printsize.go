package handler

import (
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

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
