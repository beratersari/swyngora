package handler

import (
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetSnapshot handles GET /api/v1/market/snapshot.
func (h *MarketHandler) GetSnapshot(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.GetSnapshot(r.Context(), q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, snapshotToDTO(got))
}

type snapChangeDTO struct {
	Window    string `json:"window"`
	Current   string `json:"current,omitempty"`
	Change    string `json:"change,omitempty"`
	ChangePct string `json:"changePct,omitempty"`
	Direction string `json:"direction"`
	Complete  bool   `json:"complete"`
}

type snapTakerDTO struct {
	Window   string `json:"window"`
	Buy      string `json:"buy,omitempty"`
	Sell     string `json:"sell,omitempty"`
	Delta    string `json:"delta,omitempty"`
	BuyShare string `json:"buyShare,omitempty"`
	Dominant string `json:"dominant,omitempty"`
	Complete bool   `json:"complete"`
}

type snapWindowDTO struct {
	Window    string        `json:"window"`
	Price     snapChangeDTO `json:"price"`
	Volume    snapChangeDTO `json:"volume"`
	MarketCap snapChangeDTO `json:"marketCap"`
	OI        snapChangeDTO `json:"openInterest"`
	Funding   snapChangeDTO `json:"funding"`
	LongPct   snapChangeDTO `json:"longPct"`
	Taker     snapTakerDTO  `json:"taker"`
}

type snapVenueDTO struct {
	Exchange string          `json:"exchange"`
	OIValue  string          `json:"openInterestValue,omitempty"`
	Funding  string          `json:"fundingRate,omitempty"`
	LongPct  string          `json:"longPct,omitempty"`
	Windows  []snapWindowDTO `json:"windows"`
	Summary  string          `json:"summary"`
	Error    string          `json:"error,omitempty"`
}

type snapSpotDTO struct {
	Price       string          `json:"price"`
	Volume24h   string          `json:"volume24h"`
	MarketCap   string          `json:"marketCap"`
	Circulating string          `json:"circulating,omitempty"`
	Windows     []snapWindowDTO `json:"windows"`
}

type snapshotResponse struct {
	Symbol   string         `json:"symbol"`
	Exchange string         `json:"exchange"`
	AsOf     time.Time      `json:"asOf"`
	Spot     snapSpotDTO    `json:"spot"`
	Venues   []snapVenueDTO `json:"venues"`
	Combined *snapVenueDTO  `json:"combined,omitempty"`
	Summary  string         `json:"summary"`
	Note     string         `json:"note"`
}

func snapshotToDTO(a *domain.SnapshotReport) snapshotResponse {
	if a == nil {
		return snapshotResponse{}
	}
	out := snapshotResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, AsOf: a.AsOf.UTC(),
		Spot: snapSpotDTO{
			Price: formatHistQty(a.Spot.Price), Volume24h: formatHistQty(a.Spot.Volume24h),
			MarketCap: formatHistQty(a.Spot.MarketCap), Windows: snapWindowsToDTO(a.Spot.Windows, false),
		},
		Summary: a.Summary, Note: a.Note,
	}
	if a.Spot.Circulating > 0 {
		out.Spot.Circulating = formatHistQty(a.Spot.Circulating)
	}
	for _, v := range a.Venues {
		out.Venues = append(out.Venues, snapVenueToDTO(v))
	}
	if a.Combined != nil {
		c := snapVenueToDTO(*a.Combined)
		out.Combined = &c
	}
	return out
}

func snapVenueToDTO(v domain.SnapshotVenue) snapVenueDTO {
	return snapVenueDTO{
		Exchange: string(v.Exchange),
		OIValue:  formatHistQty(v.OIValue),
		Funding:  domain.FormatSignedQty(v.Funding),
		LongPct:  formatHistQty(v.LongPct),
		Windows:  snapWindowsToDTO(v.Windows, true),
		Summary:  v.Summary, Error: v.Error,
	}
}

func snapWindowsToDTO(ws []domain.SnapshotWindow, futures bool) []snapWindowDTO {
	out := make([]snapWindowDTO, 0, len(ws))
	for _, w := range ws {
		row := snapWindowDTO{
			Window:    w.Window,
			Price:     snapChangeToDTO(w.Price, false),
			Volume:    snapChangeToDTO(w.Volume, false),
			MarketCap: snapChangeToDTO(w.MarketCap, false),
		}
		if futures {
			row.OI = snapChangeToDTO(w.OI, false)
			row.Funding = snapChangeToDTO(w.Funding, true)
			row.LongPct = snapChangeToDTO(w.LongPct, false)
			row.Taker = snapTakerDTO{
				Window: w.Taker.Window, Dominant: w.Taker.Dominant, Complete: w.Taker.Complete,
			}
			if w.Taker.Complete {
				row.Taker.Buy = formatHistQty(w.Taker.Buy)
				row.Taker.Sell = formatHistQty(w.Taker.Sell)
				row.Taker.Delta = domain.FormatSignedQty(w.Taker.Delta)
				row.Taker.BuyShare = domain.FormatSignedPct(w.Taker.BuyShare)
			}
		}
		out = append(out, row)
	}
	return out
}

func snapChangeToDTO(c domain.SnapshotChange, funding bool) snapChangeDTO {
	out := snapChangeDTO{Window: c.Window, Direction: c.Direction, Complete: c.Complete}
	if c.Current != 0 || c.Complete {
		if funding {
			out.Current = domain.FormatSignedQty(c.Current)
			if c.Complete {
				out.Change = domain.FormatSignedQty(c.Change)
			}
		} else {
			out.Current = formatHistQty(c.Current)
			if c.Complete {
				out.Change = domain.FormatSignedQty(c.Change)
			}
		}
	}
	if c.Complete {
		out.ChangePct = domain.FormatSignedPct(c.ChangePct)
	}
	return out
}
