package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetVolumeProfile handles GET /api/v1/market/volume-profile.
func (h *MarketHandler) GetVolumeProfile(w http.ResponseWriter, r *http.Request) {
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
	var tick float64
	if raw := q.Get("tickSize"); raw != "" {
		n, err := strconv.ParseFloat(raw, 64)
		if err != nil || n <= 0 {
			writeError(w, badTickSize())
			return
		}
		tick = n
	}
	got, err := h.svc.GetVolumeProfile(r.Context(), q.Get("exchange"), q.Get("symbol"), q.Get("window"), startPtr, endPtr, tick)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, volumeProfileToDTO(got))
}

func badTickSize() error {
	return fmt.Errorf("%w: tickSize must be a positive number", domain.ErrInvalidArgument)
}

type volumeProfileShareDTO struct {
	Exchange string `json:"exchange"`
	Volume   string `json:"volume"`
	SharePct string `json:"sharePct"`
}

type volumeProfileBinDTO struct {
	Price       string                  `json:"price"`
	High        string                  `json:"high"`
	Volume      string                  `json:"volume"`
	BuyVolume   string                  `json:"buyVolume"`
	SellVolume  string                  `json:"sellVolume"`
	BuyPct      string                  `json:"buyPct"`
	SharePct    string                  `json:"sharePct"`
	IsPoc       bool                    `json:"isPoc"`
	InValueArea bool                    `json:"inValueArea"`
	IsHvn       bool                    `json:"isHvn"`
	Shares      []volumeProfileShareDTO `json:"shares,omitempty"`
}

type volumeProfilePOCDTO struct {
	Price      string `json:"price"`
	High       string `json:"high"`
	Volume     string `json:"volume"`
	BuyVolume  string `json:"buyVolume"`
	SellVolume string `json:"sellVolume"`
	SharePct   string `json:"sharePct"`
}

type volumeProfileValueAreaDTO struct {
	Low       string `json:"low"`
	High      string `json:"high"`
	Volume    string `json:"volume"`
	VolumePct string `json:"volumePct"`
	BinCount  int    `json:"binCount"`
}

type volumeProfileVenueDTO struct {
	Exchange       string                    `json:"exchange"`
	Symbol         string                    `json:"symbol"`
	From           time.Time                 `json:"from"`
	To             time.Time                 `json:"to"`
	Interval       string                    `json:"interval"`
	TickSize       string                    `json:"tickSize"`
	LastPrice      string                    `json:"lastPrice"`
	High           string                    `json:"high"`
	Low            string                    `json:"low"`
	TotalVolume    string                    `json:"totalVolume"`
	BuyVolume      string                    `json:"buyVolume"`
	SellVolume     string                    `json:"sellVolume"`
	BuySellKnown   bool                      `json:"buySellKnown"`
	BuySellPartial bool                      `json:"buySellPartial"`
	LastVsArea     string                    `json:"lastVsValueArea"`
	POC            volumeProfilePOCDTO       `json:"poc"`
	ValueArea      volumeProfileValueAreaDTO `json:"valueArea"`
	Bins           []volumeProfileBinDTO     `json:"bins"`
	BarCount       int                       `json:"barCount"`
	Summary        string                    `json:"summary"`
	Error          string                    `json:"error,omitempty"`
}

type volumeProfileResponse struct {
	Symbol   string                  `json:"symbol"`
	Exchange string                  `json:"exchange"`
	Window   string                  `json:"window"`
	From     time.Time               `json:"from"`
	To       time.Time               `json:"to"`
	AsOf     time.Time               `json:"asOf"`
	Venues   []volumeProfileVenueDTO `json:"venues"`
	Combined *volumeProfileVenueDTO  `json:"combined,omitempty"`
	Summary  string                  `json:"summary"`
	Note     string                  `json:"note"`
}

func volumeProfileToDTO(a *domain.VolumeProfileReport) volumeProfileResponse {
	if a == nil {
		return volumeProfileResponse{}
	}
	venues := make([]volumeProfileVenueDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		venues = append(venues, volumeProfileVenueToDTO(v))
	}
	out := volumeProfileResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, Window: a.Window,
		From: a.From.UTC(), To: a.To.UTC(), AsOf: a.AsOf.UTC(),
		Venues: venues, Summary: a.Summary, Note: a.Note,
	}
	if a.Combined != nil {
		c := volumeProfileVenueToDTO(*a.Combined)
		out.Combined = &c
	}
	return out
}

func volumeProfileVenueToDTO(v domain.VolumeProfileVenue) volumeProfileVenueDTO {
	bins := make([]volumeProfileBinDTO, 0, len(v.Bins))
	for _, b := range v.Bins {
		bins = append(bins, volumeProfileBinDTO{
			Price: formatHistQty(b.Price), High: formatHistQty(b.High),
			Volume: formatHistQty(b.Volume), BuyVolume: formatHistQty(b.BuyVolume),
			SellVolume: formatHistQty(b.SellVolume), BuyPct: formatHistQty(b.BuyPct),
			SharePct: formatHistQty(b.SharePct), IsPoc: b.IsPoc, InValueArea: b.InValueArea,
			IsHvn: b.IsHvn, Shares: volumeProfileSharesToDTO(b.Shares),
		})
	}
	return volumeProfileVenueDTO{
		Exchange: string(v.Exchange), Symbol: v.Symbol,
		From: v.From.UTC(), To: v.To.UTC(), Interval: v.Interval,
		TickSize: formatHistQty(v.TickSize), LastPrice: formatHistQty(v.LastPrice),
		High: formatHistQty(v.High), Low: formatHistQty(v.Low),
		TotalVolume: formatHistQty(v.TotalVolume), BuyVolume: formatHistQty(v.BuyVolume),
		SellVolume: formatHistQty(v.SellVolume), BuySellKnown: v.BuySellKnown,
		BuySellPartial: v.BuySellPartial, LastVsArea: v.LastVsArea,
		POC: volumeProfilePOCDTO{
			Price: formatHistQty(v.POC.Price), High: formatHistQty(v.POC.High),
			Volume: formatHistQty(v.POC.Volume), BuyVolume: formatHistQty(v.POC.BuyVolume),
			SellVolume: formatHistQty(v.POC.SellVolume), SharePct: formatHistQty(v.POC.SharePct),
		},
		ValueArea: volumeProfileValueAreaDTO{
			Low: formatHistQty(v.ValueArea.Low), High: formatHistQty(v.ValueArea.High),
			Volume: formatHistQty(v.ValueArea.Volume), VolumePct: formatHistQty(v.ValueArea.VolumePct),
			BinCount: v.ValueArea.BinCount,
		},
		Bins: bins, BarCount: v.BarCount, Summary: v.Summary, Error: v.Error,
	}
}

func volumeProfileSharesToDTO(in []domain.VolumeProfileShare) []volumeProfileShareDTO {
	if len(in) == 0 {
		return nil
	}
	out := make([]volumeProfileShareDTO, 0, len(in))
	for _, s := range in {
		out = append(out, volumeProfileShareDTO{
			Exchange: string(s.Exchange),
			Volume:   formatHistQty(s.Volume),
			SharePct: formatHistQty(s.SharePct),
		})
	}
	return out
}
