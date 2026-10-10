package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetDerivativesGrid handles GET /api/v1/market/derivatives-grid.
func (h *MarketHandler) GetDerivativesGrid(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 0
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, fmt.Errorf("%w: limit must be an integer", domain.ErrInvalidArgument))
			return
		}
		limit = n
	}
	got, err := h.svc.GetDerivativesGrid(r.Context(), q.Get("exchange"), q.Get("metric"), q.Get("range"), limit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, derivativesGridToDTO(got))
}

type derivativesGridCellDTO struct {
	Time     string   `json:"time"`
	Value    *float64 `json:"value,omitempty"`
	Text     string   `json:"text,omitempty"`
	Heat     *float64 `json:"heat,omitempty"`
	Definite bool     `json:"definite"`
	Missing  []string `json:"missing,omitempty"`
	Weight   string   `json:"weight,omitempty"`
}

type derivativesGridRowDTO struct {
	Symbol string                   `json:"symbol"`
	Base   string                   `json:"base,omitempty"`
	Cells  []derivativesGridCellDTO `json:"cells"`
}

type derivativesGridDTO struct {
	Exchange string                  `json:"exchange"`
	Metric   string                  `json:"metric"`
	Range    string                  `json:"range"`
	StepSec  int                     `json:"stepSec"`
	AsOf     time.Time               `json:"asOf"`
	Columns  []time.Time             `json:"columns"`
	Rows     []derivativesGridRowDTO `json:"rows"`
	Universe int                     `json:"universe"`
	Note     string                  `json:"note,omitempty"`
}

func derivativesGridToDTO(got *domain.DerivativesGrid) derivativesGridDTO {
	if got == nil {
		return derivativesGridDTO{}
	}
	out := derivativesGridDTO{
		Exchange: got.Exchange,
		Metric:   got.Metric,
		Range:    got.Range,
		StepSec:  got.StepSec,
		AsOf:     got.AsOf,
		Columns:  got.Columns,
		Universe: got.Universe,
		Note:     got.Note,
		Rows:     make([]derivativesGridRowDTO, 0, len(got.Rows)),
	}
	if out.Columns == nil {
		out.Columns = []time.Time{}
	}
	for _, row := range got.Rows {
		dto := derivativesGridRowDTO{Symbol: row.Symbol, Base: row.Base, Cells: make([]derivativesGridCellDTO, 0, len(row.Cells))}
		for _, cell := range row.Cells {
			item := derivativesGridCellDTO{
				Time:     cell.Time.UTC().Format(time.RFC3339),
				Text:     cell.Text,
				Definite: cell.Definite,
				Missing:  cell.Missing,
				Weight:   cell.Weight,
			}
			if cell.Definite {
				value := cell.Value
				item.Value = &value
				if cell.HasHeat {
					heat := cell.Heat
					item.Heat = &heat
				}
			} else {
				item.Text = ""
				item.Weight = ""
			}
			dto.Cells = append(dto.Cells, item)
		}
		out.Rows = append(out.Rows, dto)
	}
	return out
}
