package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetIcebergs handles GET /api/v1/market/orderbook/icebergs.
func (h *MarketHandler) GetIcebergs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	minNotional := 0.0
	if raw := q.Get("minNotional"); raw != "" {
		n, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			writeError(w, fmt.Errorf("%w: minNotional must be a number", domain.ErrInvalidArgument))
			return
		}
		minNotional = n
	}
	got, err := h.svc.GetIcebergs(r.Context(), q.Get("exchange"), q.Get("symbol"), minNotional)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, icebergsToDTO(got))
}

type icebergLevelDTO struct {
	Exchange         string    `json:"exchange"`
	Symbol           string    `json:"symbol"`
	Side             string    `json:"side"`
	Price            string    `json:"price"`
	ClipQuantity     string    `json:"clipQuantity"`
	ClipNotional     string    `json:"clipNotional"`
	VisibleQuantity  string    `json:"visibleQuantity"`
	VisibleNotional  string    `json:"visibleNotional"`
	Refills          int       `json:"refills"`
	ExecutedNotional string    `json:"executedNotional"`
	PrintHits        int       `json:"printHits"`
	Confidence       string    `json:"confidence"`
	FirstSeen        time.Time `json:"firstSeen"`
	LastRefill       time.Time `json:"lastRefill,omitempty"`
	Summary          string    `json:"summary"`
}

type icebergsResponse struct {
	Symbol   string            `json:"symbol"`
	Exchange string            `json:"exchange"`
	AsOf     time.Time         `json:"asOf"`
	Asks     []icebergLevelDTO `json:"asks"`
	Bids     []icebergLevelDTO `json:"bids"`
	Summary  string            `json:"summary"`
	Note     string            `json:"note"`
}

func icebergsToDTO(a *domain.IcebergReport) icebergsResponse {
	if a == nil {
		return icebergsResponse{Asks: []icebergLevelDTO{}, Bids: []icebergLevelDTO{}}
	}
	return icebergsResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, AsOf: a.AsOf.UTC(),
		Asks: icebergListDTO(a.Asks), Bids: icebergListDTO(a.Bids),
		Summary: a.Summary, Note: a.Note,
	}
}

func icebergListDTO(in []domain.IcebergLevel) []icebergLevelDTO {
	out := make([]icebergLevelDTO, 0, len(in))
	for _, e := range in {
		out = append(out, icebergLevelDTO{
			Exchange: string(e.Exchange), Symbol: e.Symbol, Side: e.Side,
			Price: formatHistQty(e.Price), ClipQuantity: formatHistQty(e.ClipQuantity),
			ClipNotional:    formatHistQty(e.ClipNotional),
			VisibleQuantity: formatHistQty(e.VisibleQuantity), VisibleNotional: formatHistQty(e.VisibleNotional),
			Refills: e.Refills, ExecutedNotional: formatHistQty(e.ExecutedNotional),
			PrintHits: e.PrintHits, Confidence: e.Confidence,
			FirstSeen: e.FirstSeen.UTC(), LastRefill: e.LastRefill.UTC(), Summary: e.Summary,
		})
	}
	return out
}
