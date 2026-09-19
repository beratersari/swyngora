package handler

import (
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetBasis handles GET /api/v1/market/basis.
func (h *MarketHandler) GetBasis(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.GetBasis(r.Context(), q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, basisToDTO(got))
}

type basisLevelDTO struct {
	Futures  string `json:"futures"`
	Spot     string `json:"spot"`
	Delta    string `json:"delta"`
	DeltaPct string `json:"deltaPct"`
	Kind     string `json:"kind"`
	Source   string `json:"source"`
}

type basisWindowDTO struct {
	Window     string `json:"window"`
	PastPct    string `json:"pastPct,omitempty"`
	ChangePct  string `json:"changePct,omitempty"`
	Trend      string `json:"trend"`
	Complete   bool   `json:"complete"`
	SampleTime string `json:"sampleTime,omitempty"`
}

type basisContractDTO struct {
	Symbol       string `json:"symbol"`
	Kind         string `json:"kind"`
	Label        string `json:"label"`
	Expiry       string `json:"expiry,omitempty"`
	DaysToExpiry int    `json:"daysToExpiry"`
	Last         string `json:"last,omitempty"`
	Mark         string `json:"mark"`
	Index        string `json:"index,omitempty"`
	VsIndex      string `json:"vsIndex"`
	VsIndexPct   string `json:"vsIndexPct"`
}

type basisCurveDTO struct {
	Points    []basisContractDTO    `json:"points"`
	Near      *basisContractDTO     `json:"near,omitempty"`
	Far       *basisContractDTO     `json:"far,omitempty"`
	Spread    string                `json:"spread,omitempty"`
	SpreadPct string                `json:"spreadPct,omitempty"`
	Shape     string                `json:"shape"`
	Slope     string                `json:"slope"`
	Richer    string                `json:"richer"`
	Summary   string                `json:"summary"`
	Complete  bool                  `json:"complete"`
	History   []basisCurvePointDTO  `json:"history,omitempty"`
	Windows   []basisCurveWindowDTO `json:"windows,omitempty"`
	Moves     []basisCurveMoveDTO   `json:"moves,omitempty"`
}

type basisCurvePointDTO struct {
	Time      string `json:"time"`
	NearMark  string `json:"nearMark,omitempty"`
	FarMark   string `json:"farMark,omitempty"`
	Spread    string `json:"spread"`
	SpreadPct string `json:"spreadPct"`
	Shape     string `json:"shape"`
	Slope     string `json:"slope"`
}

type basisCurveWindowDTO struct {
	Window        string `json:"window"`
	PastSpreadPct string `json:"pastSpreadPct,omitempty"`
	ChangePct     string `json:"changePct,omitempty"`
	PastShape     string `json:"pastShape,omitempty"`
	Shape         string `json:"shape"`
	Change        string `json:"change"`
	Complete      bool   `json:"complete"`
	SampleTime    string `json:"sampleTime,omitempty"`
}

type basisCurveMoveDTO struct {
	Time      string `json:"time"`
	Kind      string `json:"kind"`
	FromShape string `json:"fromShape,omitempty"`
	ToShape   string `json:"toShape,omitempty"`
	SpreadPct string `json:"spreadPct,omitempty"`
	ChangePct string `json:"changePct,omitempty"`
	Summary   string `json:"summary"`
}

type basisVenueDTO struct {
	Exchange string           `json:"exchange"`
	Symbol   string           `json:"symbol"`
	Last     basisLevelDTO    `json:"last"`
	Mark     basisLevelDTO    `json:"mark"`
	Windows  []basisWindowDTO `json:"windows"`
	Trend    string           `json:"trend"`
	Curve    basisCurveDTO    `json:"curve"`
	Summary  string           `json:"summary"`
	Error    string           `json:"error,omitempty"`
}

type basisAgreementDTO struct {
	Alignment string `json:"alignment"`
	Title     string `json:"title"`
	Summary   string `json:"summary"`
}

type basisResponse struct {
	Symbol    string             `json:"symbol"`
	Exchange  string             `json:"exchange"`
	AsOf      time.Time          `json:"asOf"`
	Venues    []basisVenueDTO    `json:"venues"`
	Agreement *basisAgreementDTO `json:"agreement,omitempty"`
	Note      string             `json:"note"`
}

func basisToDTO(a *domain.BasisReport) basisResponse {
	if a == nil {
		return basisResponse{}
	}
	venues := make([]basisVenueDTO, 0, len(a.Venues))
	for _, v := range a.Venues {
		venues = append(venues, basisVenueToDTO(v))
	}
	out := basisResponse{
		Symbol: a.Symbol, Exchange: a.Exchange, AsOf: a.AsOf.UTC(),
		Venues: venues, Note: a.Note,
	}
	if a.Agreement != nil {
		out.Agreement = &basisAgreementDTO{
			Alignment: a.Agreement.Alignment, Title: a.Agreement.Title, Summary: a.Agreement.Summary,
		}
	}
	return out
}

func basisVenueToDTO(v domain.BasisVenueReport) basisVenueDTO {
	wins := make([]basisWindowDTO, 0, len(v.Windows))
	for _, w := range v.Windows {
		row := basisWindowDTO{Window: w.Window, Trend: w.Trend, Complete: w.Complete}
		if w.Complete {
			row.PastPct = domain.FormatSignedPct(w.PastPct)
			row.ChangePct = domain.FormatSignedPct(w.ChangePct)
			if !w.SampleTime.IsZero() {
				row.SampleTime = w.SampleTime.UTC().Format(time.RFC3339Nano)
			}
		}
		wins = append(wins, row)
	}
	return basisVenueDTO{
		Exchange: string(v.Exchange), Symbol: v.Symbol,
		Last: basisLevelToDTO(v.Last), Mark: basisLevelToDTO(v.Mark),
		Windows: wins, Trend: v.Trend, Curve: basisCurveToDTO(v.Curve),
		Summary: v.Summary, Error: v.Error,
	}
}

func basisCurveToDTO(c domain.BasisCurve) basisCurveDTO {
	pts := make([]basisContractDTO, 0, len(c.Points))
	for _, p := range c.Points {
		pts = append(pts, basisContractToDTO(p))
	}
	out := basisCurveDTO{
		Points: pts, Shape: c.Shape, Slope: c.Slope, Richer: c.Richer,
		Summary: c.Summary, Complete: c.Complete,
	}
	if c.Complete {
		out.Spread = domain.FormatSignedQty(c.Spread)
		out.SpreadPct = domain.FormatSignedPct(c.SpreadPct)
	}
	if c.Near != nil {
		n := basisContractToDTO(*c.Near)
		out.Near = &n
	}
	if c.Far != nil {
		f := basisContractToDTO(*c.Far)
		out.Far = &f
	}
	if len(c.History) > 0 {
		out.History = make([]basisCurvePointDTO, 0, len(c.History))
		for _, p := range c.History {
			out.History = append(out.History, basisCurvePointToDTO(p))
		}
	}
	if len(c.Windows) > 0 {
		out.Windows = make([]basisCurveWindowDTO, 0, len(c.Windows))
		for _, w := range c.Windows {
			row := basisCurveWindowDTO{Window: w.Window, Shape: w.Shape, Change: w.Change, Complete: w.Complete}
			if w.Complete {
				row.PastSpreadPct = domain.FormatSignedPct(w.PastSpreadPct)
				row.ChangePct = domain.FormatSignedPct(w.ChangePct)
				row.PastShape = w.PastShape
				if !w.SampleTime.IsZero() {
					row.SampleTime = w.SampleTime.UTC().Format(time.RFC3339Nano)
				}
			}
			out.Windows = append(out.Windows, row)
		}
	}
	if len(c.Moves) > 0 {
		out.Moves = make([]basisCurveMoveDTO, 0, len(c.Moves))
		for _, m := range c.Moves {
			row := basisCurveMoveDTO{
				Kind: m.Kind, FromShape: m.FromShape, ToShape: m.ToShape, Summary: m.Summary,
			}
			if !m.Time.IsZero() {
				row.Time = m.Time.UTC().Format(time.RFC3339Nano)
			}
			if m.SpreadPct != 0 {
				row.SpreadPct = domain.FormatSignedPct(m.SpreadPct)
			}
			if m.ChangePct != 0 {
				row.ChangePct = domain.FormatSignedPct(m.ChangePct)
			}
			out.Moves = append(out.Moves, row)
		}
	}
	return out
}

func basisCurvePointToDTO(p domain.BasisCurvePoint) basisCurvePointDTO {
	t := ""
	if !p.Time.IsZero() {
		t = p.Time.UTC().Format(time.RFC3339Nano)
	}
	return basisCurvePointDTO{
		Time: t, NearMark: formatHistQty(p.NearMark), FarMark: formatHistQty(p.FarMark),
		Spread: domain.FormatSignedQty(p.Spread), SpreadPct: domain.FormatSignedPct(p.SpreadPct),
		Shape: p.Shape, Slope: p.Slope,
	}
}

func basisContractToDTO(p domain.BasisContractPoint) basisContractDTO {
	exp := ""
	if !p.Expiry.IsZero() && p.Kind != domain.BasisContractPerp {
		exp = p.Expiry.UTC().Format(time.RFC3339)
	}
	return basisContractDTO{
		Symbol: p.Symbol, Kind: p.Kind, Label: p.Label, Expiry: exp,
		DaysToExpiry: p.DaysToExpiry,
		Last:         formatHistQty(p.Last), Mark: formatHistQty(p.Mark),
		Index:      formatHistQty(p.Index),
		VsIndex:    domain.FormatSignedQty(p.VsIndex),
		VsIndexPct: domain.FormatSignedPct(p.VsIndexPct),
	}
}

func basisLevelToDTO(lv domain.BasisLevel) basisLevelDTO {
	return basisLevelDTO{
		Futures: formatHistQty(lv.Futures), Spot: formatHistQty(lv.Spot),
		Delta: domain.FormatSignedQty(lv.Delta), DeltaPct: domain.FormatSignedPct(lv.DeltaPct),
		Kind: lv.Kind, Source: lv.Source,
	}
}
