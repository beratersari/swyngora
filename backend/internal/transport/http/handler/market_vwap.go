package handler

import (
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetVWAP handles GET /api/v1/market/vwap.
func (h *MarketHandler) GetVWAP(w http.ResponseWriter, r *http.Request) {
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
	got, err := h.svc.GetVWAP(r.Context(), q.Get("exchange"), q.Get("symbol"), q.Get("window"), startPtr, endPtr)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, vwapToDTO(got))
}

type vwapShareDTO struct {
	Exchange string `json:"exchange"`
	Volume   string `json:"volume"`
	SharePct string `json:"sharePct"`
	VWAP     string `json:"vwap"`
}

type vwapVenueDTO struct {
	Exchange    string         `json:"exchange"`
	Symbol      string         `json:"symbol"`
	From        time.Time      `json:"from"`
	To          time.Time      `json:"to"`
	Interval    string         `json:"interval"`
	VWAP        string         `json:"vwap"`
	LastPrice   string         `json:"lastPrice"`
	Distance    string         `json:"distance"`
	DistancePct string         `json:"distancePct"`
	Side        string         `json:"side"`
	Volume      string         `json:"volume"`
	BarCount    int            `json:"barCount"`
	High        string         `json:"high"`
	Low         string         `json:"low"`
	Shares      []vwapShareDTO `json:"shares,omitempty"`
	Summary     string         `json:"summary"`
	Error       string         `json:"error,omitempty"`
}

type vwapResponse struct {
	Symbol   string         `json:"symbol"`
	Exchange string         `json:"exchange"`
	Window   string         `json:"window"`
	From     time.Time      `json:"from"`
	To       time.Time      `json:"to"`
	AsOf     time.Time      `json:"asOf"`
	Venues   []vwapVenueDTO `json:"venues"`
	Combined *vwapVenueDTO  `json:"combined,omitempty"`
	Summary  string         `json:"summary"`
	Note     string         `json:"note"`
}

func vwapToDTO(a *domain.VWAPReport) vwapResponse {
	if a == nil {
		return vwapResponse{}
	}
	venues := make([]vwapVenueDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		venues = append(venues, vwapVenueToDTO(v))
	}
	out := vwapResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, Window: a.Window,
		From: a.From.UTC(), To: a.To.UTC(), AsOf: a.AsOf.UTC(),
		Venues: venues, Summary: a.Summary, Note: a.Note,
	}
	if a.Combined != nil {
		c := vwapVenueToDTO(*a.Combined)
		out.Combined = &c
	}
	return out
}

func vwapVenueToDTO(v domain.VWAPVenue) vwapVenueDTO {
	shares := make([]vwapShareDTO, 0, len(v.Shares))
	for _, s := range v.Shares {
		shares = append(shares, vwapShareDTO{
			Exchange: string(s.Exchange), Volume: formatHistQty(s.Volume),
			SharePct: formatHistQty(s.SharePct), VWAP: formatHistQty(s.VWAP),
		})
	}
	if len(shares) == 0 {
		shares = nil
	}
	return vwapVenueDTO{
		Exchange: string(v.Exchange), Symbol: v.Symbol,
		From: v.From.UTC(), To: v.To.UTC(), Interval: v.Interval,
		VWAP: formatHistQty(v.VWAP), LastPrice: formatHistQty(v.LastPrice),
		Distance: domain.FormatSignedQty(v.Distance), DistancePct: domain.FormatSignedPct(v.DistancePct),
		Side: v.Side, Volume: formatHistQty(v.Volume), BarCount: v.BarCount,
		High: formatHistQty(v.High), Low: formatHistQty(v.Low),
		Shares: shares, Summary: v.Summary, Error: v.Error,
	}
}
