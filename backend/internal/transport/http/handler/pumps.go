package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
)

type pumpEventDTO struct {
	Index       int     `json:"index"`
	OpenTime    string  `json:"openTime"`
	CloseTime   string  `json:"closeTime"`
	StartPrice  float64 `json:"startPrice"`
	EndPrice    float64 `json:"endPrice"`
	ReturnPct   float64 `json:"returnPct"`
	High        float64 `json:"high"`
	Low         float64 `json:"low"`
	Volume      float64 `json:"volume"`
	VolumeRatio float64 `json:"volumeRatio"`
	Mode        string  `json:"mode"`
	WindowBars  int     `json:"windowBars"`
}

func pumpEventsDTO(ev []domain.PumpEvent) []pumpEventDTO {
	out := make([]pumpEventDTO, 0, len(ev))
	for _, e := range ev {
		out = append(out, pumpEventDTO{
			Index:       e.Index,
			OpenTime:    e.OpenTime.UTC().Format(time.RFC3339Nano),
			CloseTime:   e.CloseTime.UTC().Format(time.RFC3339Nano),
			StartPrice:  e.StartPrice,
			EndPrice:    e.EndPrice,
			ReturnPct:   e.ReturnPct,
			High:        e.High,
			Low:         e.Low,
			Volume:      e.Volume,
			VolumeRatio: e.VolumeRatio,
			Mode:        string(e.Mode),
			WindowBars:  e.WindowBars,
		})
	}
	return out
}

// GetPumpEvents handles GET /api/v1/market/pumps — single-symbol pump detection.
func (h *MarketHandler) GetPumpEvents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	symbol := q.Get("symbol")
	if symbol == "" {
		writeError(w, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument))
		return
	}
	pq := market.PumpQuery{
		Exchange:  q.Get("exchange"),
		Symbol:    symbol,
		Interval:  q.Get("interval"),
		Mode:      domain.PumpDetectMode(q.Get("mode")),
		Direction: domain.PumpDirection(q.Get("direction")),
	}
	if raw := q.Get("lookbackHours"); raw != "" {
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			writeError(w, fmt.Errorf("%w: lookbackHours must be a number", domain.ErrInvalidArgument))
			return
		}
		pq.LookbackHours = f
	}
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, fmt.Errorf("%w: limit must be an integer", domain.ErrInvalidArgument))
			return
		}
		pq.Limit = n
	}
	if raw := q.Get("minReturnPct"); raw != "" {
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			writeError(w, fmt.Errorf("%w: minReturnPct must be a number", domain.ErrInvalidArgument))
			return
		}
		pq.MinReturnPct = f
	}
	if raw := q.Get("windowBars"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, fmt.Errorf("%w: windowBars must be an integer", domain.ErrInvalidArgument))
			return
		}
		pq.WindowBars = n
	}
	if raw := q.Get("minVolumeRatio"); raw != "" {
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			writeError(w, fmt.Errorf("%w: minVolumeRatio must be a number", domain.ErrInvalidArgument))
			return
		}
		pq.MinVolumeRatio = f
	}
	if raw := q.Get("maxEvents"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, fmt.Errorf("%w: maxEvents must be an integer", domain.ErrInvalidArgument))
			return
		}
		pq.MaxEvents = n
	}
	if raw := q.Get("startTime"); raw != "" {
		t, err := parseTimeParam(raw)
		if err != nil {
			writeError(w, err)
			return
		}
		pq.StartTime = &t
	}
	if raw := q.Get("endTime"); raw != "" {
		t, err := parseTimeParam(raw)
		if err != nil {
			writeError(w, err)
			return
		}
		pq.EndTime = &t
	}

	res, err := h.svc.DetectPumpEvents(r.Context(), pq)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"exchange":      string(res.Exchange),
		"symbol":        res.Symbol,
		"interval":      res.Interval,
		"lookbackHours": res.LookbackHours,
		"barsAnalyzed":  res.BarsAnalyzed,
		"minReturnPct":  res.MinReturnPct,
		"windowBars":    res.WindowBars,
		"mode":          string(res.Mode),
		"direction":     string(res.Direction),
		"events":        pumpEventsDTO(res.Events),
		"eventCount":    len(res.Events),
		"note":          res.Note,
	})
}

// ScanPumpEvents handles GET /api/v1/market/pumps/scan — multi-symbol scan.
func (h *MarketHandler) ScanPumpEvents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	sq := market.PumpScanQuery{
		Exchange:   q.Get("exchange"),
		QuoteAsset: q.Get("quote"),
		Interval:   q.Get("interval"),
		Mode:       domain.PumpDetectMode(q.Get("mode")),
		Direction:  domain.PumpDirection(q.Get("direction")),
	}
	if raw := q.Get("lookbackHours"); raw != "" {
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			writeError(w, fmt.Errorf("%w: lookbackHours must be a number", domain.ErrInvalidArgument))
			return
		}
		sq.LookbackHours = f
	}
	if raw := q.Get("minReturnPct"); raw != "" {
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			writeError(w, fmt.Errorf("%w: minReturnPct must be a number", domain.ErrInvalidArgument))
			return
		}
		sq.MinReturnPct = f
	}
	if raw := q.Get("windowBars"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, fmt.Errorf("%w: windowBars must be an integer", domain.ErrInvalidArgument))
			return
		}
		sq.WindowBars = n
	}
	if raw := q.Get("minVolumeRatio"); raw != "" {
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			writeError(w, fmt.Errorf("%w: minVolumeRatio must be a number", domain.ErrInvalidArgument))
			return
		}
		sq.MinVolumeRatio = f
	}
	if raw := q.Get("symbolLimit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, fmt.Errorf("%w: symbolLimit must be an integer", domain.ErrInvalidArgument))
			return
		}
		sq.SymbolLimit = n
	}
	if raw := q.Get("maxTotalEvents"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, fmt.Errorf("%w: maxTotalEvents must be an integer", domain.ErrInvalidArgument))
			return
		}
		sq.MaxTotalEvents = n
	}

	hits, err := h.svc.ScanPumpEvents(r.Context(), sq)
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]map[string]any, 0, len(hits))
	for _, hhit := range hits {
		items = append(items, map[string]any{
			"symbol":        hhit.Symbol,
			"exchange":      string(hhit.Exchange),
			"interval":      hhit.Interval,
			"bestReturnPct": hhit.BestReturnPct,
			"events":        pumpEventsDTO(hhit.Events),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"exchange":      q.Get("exchange"),
		"interval":      sq.Interval,
		"lookbackHours": sq.LookbackHours,
		"minReturnPct":  sq.MinReturnPct,
		"hits":          items,
		"hitCount":      len(items),
		"note":          "Informational only — not financial advice. Scan is a mechanical threshold filter over top-volume symbols.",
	})
}
