package handler

import (
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetBotTracker handles GET /api/v1/market/bot-tracker.
func (h *MarketHandler) GetBotTracker(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.GetBotTracker(r.Context(), q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, botTrackerToDTO(got))
}

type botClusterDTO struct {
	Exchange      string    `json:"exchange"`
	Side          string    `json:"side"`
	Notional      string    `json:"notional"`
	Count         int       `json:"count"`
	SpanSec       int       `json:"spanSec"`
	From          time.Time `json:"from"`
	To            time.Time `json:"to"`
	TotalNotional string    `json:"totalNotional"`
}

type botTrackerVenueDTO struct {
	Exchange  string          `json:"exchange"`
	Symbol    string          `json:"symbol"`
	Clusters  []botClusterDTO `json:"clusters"`
	BuyCount  int             `json:"buyCount"`
	SellCount int             `json:"sellCount"`
	Delta     int             `json:"delta"`
	Summary   string          `json:"summary"`
	Error     string          `json:"error,omitempty"`
}

type botTrackerResponse struct {
	Symbol    string               `json:"symbol"`
	Exchange  string               `json:"exchange"`
	AsOf      time.Time            `json:"asOf"`
	Window    string               `json:"window"`
	Burst     string               `json:"burst"`
	MinCount  int                  `json:"minCount"`
	Venues    []botTrackerVenueDTO `json:"venues"`
	Clusters  []botClusterDTO      `json:"clusters"`
	BuyCount  int                  `json:"buyCount"`
	SellCount int                  `json:"sellCount"`
	Delta     int                  `json:"delta"`
	Summary   string               `json:"summary"`
	Note      string               `json:"note"`
}

func botTrackerToDTO(a *domain.BotTrackerReport) botTrackerResponse {
	if a == nil {
		return botTrackerResponse{}
	}
	venues := make([]botTrackerVenueDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		venues = append(venues, botTrackerVenueToDTO(v))
	}
	return botTrackerResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, AsOf: a.AsOf.UTC(),
		Window: a.Window, Burst: a.Burst, MinCount: a.MinCount,
		Venues: venues, Clusters: botClustersToDTO(a.Clusters),
		BuyCount: a.BuyCount, SellCount: a.SellCount, Delta: a.Delta,
		Summary: a.Summary, Note: a.Note,
	}
}

func botTrackerVenueToDTO(v domain.BotTrackerVenue) botTrackerVenueDTO {
	return botTrackerVenueDTO{
		Exchange: string(v.Exchange), Symbol: v.Symbol,
		Clusters: botClustersToDTO(v.Clusters),
		BuyCount: v.BuyCount, SellCount: v.SellCount, Delta: v.Delta,
		Summary: v.Summary, Error: v.Error,
	}
}

func botClustersToDTO(in []domain.BotCluster) []botClusterDTO {
	out := make([]botClusterDTO, 0, len(in))
	for _, c := range in {
		sec := int(c.Span.Round(time.Second) / time.Second)
		if sec < 0 {
			sec = 0
		}
		out = append(out, botClusterDTO{
			Exchange: string(c.Exchange), Side: c.Side,
			Notional: formatHistQty(c.Notional), Count: c.Count, SpanSec: sec,
			From: c.From.UTC(), To: c.To.UTC(),
			TotalNotional: formatHistQty(c.TotalNotional),
		})
	}
	return out
}
