package handler

import (
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetLevels handles GET /api/v1/market/levels.
func (h *MarketHandler) GetLevels(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.GetLevels(r.Context(), q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, levelsToDTO(got))
}

type levelBreakDTO struct {
	Status  string `json:"status"`
	Score   int    `json:"score"`
	Volume  string `json:"volume"`
	Book    string `json:"book"`
	Taker   string `json:"taker"`
	Summary string `json:"summary"`
}

type levelZoneDTO struct {
	Kind          string         `json:"kind"`
	Price         string         `json:"price"`
	Low           string         `json:"low"`
	High          string         `json:"high"`
	DistancePct   string         `json:"distancePct"`
	Tests         int            `json:"tests"`
	Volume        string         `json:"volume"`
	BidNotional   string         `json:"bidNotional"`
	AskNotional   string         `json:"askNotional"`
	LiquiditySide string         `json:"liquiditySide"`
	Break         *levelBreakDTO `json:"breakout,omitempty"`
}

type levelsResponse struct {
	Symbol      string         `json:"symbol"`
	Exchange    string         `json:"exchange"`
	Price       string         `json:"price"`
	Supports    []levelZoneDTO `json:"supports"`
	Resistances []levelZoneDTO `json:"resistances"`
	Active      *levelZoneDTO  `json:"active,omitempty"`
	Summary     string         `json:"summary"`
	Note        string         `json:"note"`
	AsOf        time.Time      `json:"asOf"`
}

func levelsToDTO(a *domain.LevelsReport) levelsResponse {
	if a == nil {
		return levelsResponse{}
	}
	out := levelsResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, Price: formatHistQty(a.Price),
		Supports: zoneListDTO(a.Supports), Resistances: zoneListDTO(a.Resistances),
		Summary: a.Summary, Note: a.Note, AsOf: time.Now().UTC(),
	}
	if a.Active != nil {
		z := zoneToDTO(*a.Active)
		out.Active = &z
	}
	return out
}

func zoneListDTO(in []domain.PriceLevelZone) []levelZoneDTO {
	out := make([]levelZoneDTO, 0, len(in))
	for _, z := range in {
		out = append(out, zoneToDTO(z))
	}
	return out
}

func zoneToDTO(z domain.PriceLevelZone) levelZoneDTO {
	row := levelZoneDTO{
		Kind: z.Kind, Price: formatHistQty(z.Price), Low: formatHistQty(z.Low), High: formatHistQty(z.High),
		DistancePct: domain.FormatSignedPct(z.DistancePct), Tests: z.Tests,
		Volume: formatHistQty(z.Volume), BidNotional: formatHistQty(z.BidNotional),
		AskNotional: formatHistQty(z.AskNotional), LiquiditySide: z.LiquiditySide,
	}
	if z.Break != nil {
		row.Break = &levelBreakDTO{
			Status: z.Break.Status, Score: z.Break.Score, Volume: z.Break.Volume,
			Book: z.Break.Book, Taker: z.Break.Taker, Summary: z.Break.Summary,
		}
	}
	return row
}
