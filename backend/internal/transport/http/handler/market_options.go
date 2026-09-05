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
	Note            string               `json:"note"`
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
