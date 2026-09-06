package handler

import (
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetOptions handles GET /api/v1/market/options.
func (h *MarketHandler) GetOptions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	asset := q.Get("asset")
	if asset == "" {
		asset = q.Get("symbol")
	}
	got, err := h.svc.GetOptions(r.Context(), q.Get("exchange"), asset)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, optionsToDTO(got))
}

type optionsATMDTO struct {
	Iv           string    `json:"iv"`
	Strike       string    `json:"strike"`
	Expiry       time.Time `json:"expiry"`
	DaysToExpiry int       `json:"daysToExpiry"`
	CallIv       string    `json:"callIv,omitempty"`
	PutIv        string    `json:"putIv,omitempty"`
}

type optionsExpiryDTO struct {
	Expiry          time.Time                `json:"expiry"`
	Label           string                   `json:"label"`
	DaysToExpiry    int                      `json:"daysToExpiry"`
	OpenInterest    string                   `json:"openInterest"`
	OpenInterestUsd string                   `json:"openInterestUsd"`
	Volume          string                   `json:"volume"`
	VolumeUsd       string                   `json:"volumeUsd"`
	CallOi          string                   `json:"callOi"`
	PutOi           string                   `json:"putOi"`
	CallVolume      string                   `json:"callVolume"`
	PutVolume       string                   `json:"putVolume"`
	PutCallOi       string                   `json:"putCallOi,omitempty"`
	SharePct        string                   `json:"sharePct"`
	ContractCount   int                      `json:"contractCount"`
	AtmIv           string                   `json:"atmIv,omitempty"`
	CallIv          string                   `json:"callIv,omitempty"`
	PutIv           string                   `json:"putIv,omitempty"`
	Skew            string                   `json:"skew,omitempty"`
	ExpectedMoveAbs string                   `json:"expectedMoveAbs,omitempty"`
	ExpectedMovePct string                   `json:"expectedMovePct,omitempty"`
	ExpectedLow     string                   `json:"expectedLow,omitempty"`
	ExpectedHigh    string                   `json:"expectedHigh,omitempty"`
	MaxPain         string                   `json:"maxPain,omitempty"`
	MaxPainPayout   string                   `json:"maxPainPayout,omitempty"`
	Strikes         []optionsExpiryStrikeDTO `json:"strikes,omitempty"`
}

type optionsExpiryStrikeDTO struct {
	Strike    string `json:"strike"`
	CallOi    string `json:"callOi"`
	PutOi     string `json:"putOi"`
	Payout    string `json:"payout"`
	IsMaxPain bool   `json:"isMaxPain,omitempty"`
	BelowSpot bool   `json:"belowSpot,omitempty"`
}

type optionsVolPointDTO struct {
	Expiry       time.Time `json:"expiry"`
	Label        string    `json:"label"`
	DaysToExpiry int       `json:"daysToExpiry"`
	Strike       string    `json:"strike,omitempty"`
	AtmIv        string    `json:"atmIv,omitempty"`
	CallIv       string    `json:"callIv,omitempty"`
	PutIv        string    `json:"putIv,omitempty"`
	Skew         string    `json:"skew,omitempty"`
}

type optionsSkewDTO struct {
	AtmSkew    string `json:"atmSkew,omitempty"`
	CallIv     string `json:"callIv,omitempty"`
	PutIv      string `json:"putIv,omitempty"`
	Lean       string `json:"lean,omitempty"`
	FrontIv    string `json:"frontIv,omitempty"`
	BackIv     string `json:"backIv,omitempty"`
	TermSlope  string `json:"termSlope,omitempty"`
	FrontDte   int    `json:"frontDte,omitempty"`
	BackDte    int    `json:"backDte,omitempty"`
	WingSkew   string `json:"wingSkew,omitempty"`
	WingPutIv  string `json:"wingPutIv,omitempty"`
	WingCallIv string `json:"wingCallIv,omitempty"`
}

type optionsResponse struct {
	Asset           string               `json:"asset"`
	Exchange        string               `json:"exchange"`
	IndexPrice      string               `json:"indexPrice"`
	AsOf            time.Time            `json:"asOf"`
	OpenInterest    string               `json:"openInterest"`
	OpenInterestUsd string               `json:"openInterestUsd"`
	Volume          string               `json:"volume"`
	VolumeUsd       string               `json:"volumeUsd"`
	CallOi          string               `json:"callOi"`
	PutOi           string               `json:"putOi"`
	CallVolume      string               `json:"callVolume"`
	PutVolume       string               `json:"putVolume"`
	PutCallOi       string               `json:"putCallOi,omitempty"`
	PutCallVolume   string               `json:"putCallVolume,omitempty"`
	Lean            string               `json:"lean,omitempty"`
	Dvol            string               `json:"dvol,omitempty"`
	DvolChange24h   string               `json:"dvolChange24h,omitempty"`
	Atm             *optionsATMDTO       `json:"atm,omitempty"`
	TermStructure   []optionsVolPointDTO `json:"termStructure"`
	Skew            *optionsSkewDTO      `json:"skew,omitempty"`
	Expiries        []optionsExpiryDTO   `json:"expiries"`
	Listed          int                  `json:"listed"`
	WithOi          int                  `json:"withOi"`
	WithIv          int                  `json:"withIv"`
	VsMarket        *optionsVsMarketDTO  `json:"vsMarket,omitempty"`
	History         *optionsHistoryDTO   `json:"history,omitempty"`
	Note            string               `json:"note"`
}

type optionsHistoryChangeDTO struct {
	Window    string `json:"window"`
	Current   string `json:"current,omitempty"`
	Past      string `json:"past,omitempty"`
	Change    string `json:"change,omitempty"`
	ChangePct string `json:"changePct,omitempty"`
	Direction string `json:"direction,omitempty"`
	Complete  bool   `json:"complete,omitempty"`
}

type optionsHistoryWindowDTO struct {
	Window   string                   `json:"window"`
	Oi       *optionsHistoryChangeDTO `json:"oi,omitempty"`
	PutCall  *optionsHistoryChangeDTO `json:"putCall,omitempty"`
	Dvol     *optionsHistoryChangeDTO `json:"dvol,omitempty"`
	AtmIv    *optionsHistoryChangeDTO `json:"atmIv,omitempty"`
	Complete bool                     `json:"complete,omitempty"`
}

type optionsHistoryPointDTO struct {
	Time            time.Time `json:"time"`
	OpenInterest    string    `json:"openInterest,omitempty"`
	OpenInterestUsd string    `json:"openInterestUsd,omitempty"`
	PutCallOi       string    `json:"putCallOi,omitempty"`
	Dvol            string    `json:"dvol,omitempty"`
	AtmIv           string    `json:"atmIv,omitempty"`
}

type optionsHistoryDTO struct {
	Windows []optionsHistoryWindowDTO `json:"windows"`
	Points  []optionsHistoryPointDTO  `json:"points"`
	From    time.Time                 `json:"from"`
	To      time.Time                 `json:"to"`
	Note    string                    `json:"note,omitempty"`
}

type optionsVsFindingDTO struct {
	Kind     string `json:"kind"`
	Topic    string `json:"topic"`
	Severity string `json:"severity"`
	Title    string `json:"title"`
	Detail   string `json:"detail"`
	Options  string `json:"options,omitempty"`
	Market   string `json:"market,omitempty"`
}

type optionsVsMarketDTO struct {
	Compared    bool                  `json:"compared"`
	SpotLast    string                `json:"spotLast,omitempty"`
	ImpliedIv   string                `json:"impliedIv,omitempty"`
	RealizedAnn string                `json:"realizedAnn,omitempty"`
	Range24hPct string                `json:"range24hPct,omitempty"`
	FundingRate string                `json:"fundingRate,omitempty"`
	LongPct     string                `json:"longPct,omitempty"`
	Summary     string                `json:"summary,omitempty"`
	Findings    []optionsVsFindingDTO `json:"findings"`
}

func optionsToDTO(a *domain.OptionsSnapshot) optionsResponse {
	if a == nil {
		return optionsResponse{}
	}
	expiries := make([]optionsExpiryDTO, 0, len(a.Expiries))
	for _, e := range a.Expiries {
		row := optionsExpiryDTO{
			Expiry:          e.Expiry.UTC(),
			Label:           e.Label,
			DaysToExpiry:    e.DaysToExpiry,
			OpenInterest:    formatHistQty(e.OpenInterest),
			OpenInterestUsd: formatHistQty(e.OpenInterestUSD),
			Volume:          formatHistQty(e.Volume),
			VolumeUsd:       formatHistQty(e.VolumeUSD),
			CallOi:          formatHistQty(e.CallOI),
			PutOi:           formatHistQty(e.PutOI),
			CallVolume:      formatHistQty(e.CallVolume),
			PutVolume:       formatHistQty(e.PutVolume),
			SharePct:        formatHistQty(e.SharePct),
			ContractCount:   e.ContractCount,
		}
		if e.HasPutCallOI {
			row.PutCallOi = formatHistQty(e.PutCallOI)
		}
		if e.HasAtmIV {
			row.AtmIv = formatHistQty(e.AtmIV)
		}
		if e.HasCallIV {
			row.CallIv = formatHistQty(e.CallIV)
		}
		if e.HasPutIV {
			row.PutIv = formatHistQty(e.PutIV)
		}
		if e.HasSkew {
			row.Skew = domain.FormatSignedQty(e.Skew)
		}
		if e.HasExpectedMove {
			row.ExpectedMoveAbs = formatHistQty(e.ExpectedMoveAbs)
			row.ExpectedMovePct = formatHistQty(e.ExpectedMovePct)
			row.ExpectedLow = formatHistQty(e.ExpectedLow)
			row.ExpectedHigh = formatHistQty(e.ExpectedHigh)
		}
		if e.HasMaxPain {
			row.MaxPain = formatHistQty(e.MaxPain)
			row.MaxPainPayout = formatHistQty(e.MaxPainPayout)
		}
		if len(e.Strikes) > 0 {
			row.Strikes = make([]optionsExpiryStrikeDTO, 0, len(e.Strikes))
			for _, s := range e.Strikes {
				row.Strikes = append(row.Strikes, optionsExpiryStrikeDTO{
					Strike: formatHistQty(s.Strike), CallOi: formatHistQty(s.CallOI),
					PutOi: formatHistQty(s.PutOI), Payout: formatHistQty(s.Payout),
					IsMaxPain: s.IsMaxPain, BelowSpot: s.BelowSpot,
				})
			}
		}
		expiries = append(expiries, row)
	}
	term := make([]optionsVolPointDTO, 0, len(a.TermStructure))
	for _, p := range a.TermStructure {
		row := optionsVolPointDTO{
			Expiry: p.Expiry.UTC(), Label: p.Label, DaysToExpiry: p.DaysToExpiry,
		}
		if p.Strike > 0 {
			row.Strike = formatHistQty(p.Strike)
		}
		if p.HasAtm {
			row.AtmIv = formatHistQty(p.AtmIV)
		}
		if p.HasCall {
			row.CallIv = formatHistQty(p.CallIV)
		}
		if p.HasPut {
			row.PutIv = formatHistQty(p.PutIV)
		}
		if p.HasSkew {
			row.Skew = domain.FormatSignedQty(p.Skew)
		}
		term = append(term, row)
	}
	out := optionsResponse{
		Asset:           a.Asset,
		Exchange:        a.Exchange,
		IndexPrice:      formatHistQty(a.IndexPrice),
		AsOf:            a.AsOf.UTC(),
		OpenInterest:    formatHistQty(a.OpenInterest),
		OpenInterestUsd: formatHistQty(a.OpenInterestUSD),
		Volume:          formatHistQty(a.Volume),
		VolumeUsd:       formatHistQty(a.VolumeUSD),
		CallOi:          formatHistQty(a.CallOI),
		PutOi:           formatHistQty(a.PutOI),
		CallVolume:      formatHistQty(a.CallVolume),
		PutVolume:       formatHistQty(a.PutVolume),
		Lean:            a.Lean,
		TermStructure:   term,
		Expiries:        expiries,
		Listed:          a.Listed,
		WithOi:          a.WithOI,
		WithIv:          a.WithIV,
		VsMarket:        optionsVsMarketToDTO(a.VsMarket),
		History:         optionsHistoryToDTO(a.History),
		Note:            a.Note,
	}
	if a.HasPutCallOI {
		out.PutCallOi = formatHistQty(a.PutCallOI)
	}
	if a.HasPutCallVol {
		out.PutCallVolume = formatHistQty(a.PutCallVolume)
	}
	if a.HasDVOL {
		out.Dvol = formatHistQty(a.DVOL)
	}
	if a.HasDVOLChange {
		out.DvolChange24h = domain.FormatSignedQty(a.DVOLChange24h)
	}
	if a.ATM != nil {
		atm := &optionsATMDTO{
			Iv:           formatHistQty(a.ATM.IV),
			Strike:       formatHistQty(a.ATM.Strike),
			Expiry:       a.ATM.Expiry.UTC(),
			DaysToExpiry: a.ATM.DaysToExpiry,
		}
		if a.ATM.HasCall {
			atm.CallIv = formatHistQty(a.ATM.CallIV)
		}
		if a.ATM.HasPut {
			atm.PutIv = formatHistQty(a.ATM.PutIV)
		}
		out.Atm = atm
	}
	if a.Skew != nil {
		sk := &optionsSkewDTO{Lean: a.Skew.Lean, FrontDte: a.Skew.FrontDTE, BackDte: a.Skew.BackDTE}
		if a.Skew.HasAtmSkew {
			sk.AtmSkew = domain.FormatSignedQty(a.Skew.AtmSkew)
			sk.CallIv = formatHistQty(a.Skew.CallIV)
			sk.PutIv = formatHistQty(a.Skew.PutIV)
		}
		if a.Skew.HasTerm {
			sk.FrontIv = formatHistQty(a.Skew.FrontIV)
			sk.BackIv = formatHistQty(a.Skew.BackIV)
			sk.TermSlope = domain.FormatSignedQty(a.Skew.TermSlope)
		}
		if a.Skew.HasWing {
			sk.WingSkew = domain.FormatSignedQty(a.Skew.WingSkew)
			sk.WingPutIv = formatHistQty(a.Skew.WingPutIV)
			sk.WingCallIv = formatHistQty(a.Skew.WingCallIV)
		}
		out.Skew = sk
	}
	return out
}

func optionsVsMarketToDTO(v *domain.OptionsVsMarket) *optionsVsMarketDTO {
	if v == nil || !v.Compared {
		return nil
	}
	out := &optionsVsMarketDTO{
		Compared: true,
		Summary:  v.Summary,
		Findings: make([]optionsVsFindingDTO, 0, len(v.Findings)),
	}
	if v.HasSpot && v.SpotLast > 0 {
		out.SpotLast = formatHistQty(v.SpotLast)
	}
	if v.ImpliedIV > 0 {
		out.ImpliedIv = formatHistQty(v.ImpliedIV)
	}
	if v.HasVol {
		out.RealizedAnn = formatHistQty(v.RealizedAnn)
	}
	if v.Range24hPct > 0 {
		out.Range24hPct = formatHistQty(v.Range24hPct)
	}
	if v.HasFunding {
		out.FundingRate = formatHistQty(v.FundingRate)
	}
	if v.HasLongPct {
		out.LongPct = formatHistQty(v.LongPct)
	}
	for _, f := range v.Findings {
		row := optionsVsFindingDTO{
			Kind: f.Kind, Topic: f.Topic, Severity: f.Severity,
			Title: f.Title, Detail: f.Detail,
		}
		if f.HasNums {
			row.Options = formatHistQty(f.Options)
			row.Market = formatHistQty(f.Market)
		}
		out.Findings = append(out.Findings, row)
	}
	return out
}

func optionsHistoryToDTO(h *domain.OptionsHistory) *optionsHistoryDTO {
	if h == nil {
		return nil
	}
	out := &optionsHistoryDTO{
		Windows: make([]optionsHistoryWindowDTO, 0, len(h.Windows)),
		Points:  make([]optionsHistoryPointDTO, 0, len(h.Points)),
		From:    h.From.UTC(),
		To:      h.To.UTC(),
		Note:    h.Note,
	}
	for _, w := range h.Windows {
		row := optionsHistoryWindowDTO{Window: w.Window, Complete: w.Complete}
		if w.OI.Complete {
			c := snapshotChangeToOptionsDTO(w.OI)
			row.Oi = &c
		}
		if w.PutCall.Complete {
			c := snapshotChangeToOptionsDTO(w.PutCall)
			row.PutCall = &c
		}
		if w.DVOL.Complete {
			c := snapshotChangeToOptionsDTO(w.DVOL)
			row.Dvol = &c
		}
		if w.AtmIV.Complete {
			c := snapshotChangeToOptionsDTO(w.AtmIV)
			row.AtmIv = &c
		}
		out.Windows = append(out.Windows, row)
	}
	for _, p := range h.Points {
		row := optionsHistoryPointDTO{Time: p.Time.UTC()}
		if p.HasOI {
			row.OpenInterest = formatHistQty(p.OpenInterest)
			if p.OpenInterestUSD > 0 {
				row.OpenInterestUsd = formatHistQty(p.OpenInterestUSD)
			}
		}
		if p.HasPutCallOI {
			row.PutCallOi = formatHistQty(p.PutCallOI)
		}
		if p.HasDVOL {
			row.Dvol = formatHistQty(p.DVOL)
		}
		if p.HasAtmIV {
			row.AtmIv = formatHistQty(p.AtmIV)
		}
		out.Points = append(out.Points, row)
	}
	return out
}

func snapshotChangeToOptionsDTO(c domain.SnapshotChange) optionsHistoryChangeDTO {
	out := optionsHistoryChangeDTO{
		Window: c.Window, Direction: c.Direction, Complete: c.Complete,
		Current: formatHistQty(c.Current),
	}
	if c.Complete {
		out.Past = formatHistQty(c.Past)
		out.Change = domain.FormatSignedQty(c.Change)
		out.ChangePct = domain.FormatSignedQty(c.ChangePct)
	}
	return out
}
