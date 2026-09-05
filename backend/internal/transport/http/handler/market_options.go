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
	Expiry          time.Time `json:"expiry"`
	Label           string    `json:"label"`
	DaysToExpiry    int       `json:"daysToExpiry"`
	OpenInterest    string    `json:"openInterest"`
	OpenInterestUsd string    `json:"openInterestUsd"`
	Volume          string    `json:"volume"`
	VolumeUsd       string    `json:"volumeUsd"`
	CallOi          string    `json:"callOi"`
	PutOi           string    `json:"putOi"`
	CallVolume      string    `json:"callVolume"`
	PutVolume       string    `json:"putVolume"`
	PutCallOi       string    `json:"putCallOi,omitempty"`
	SharePct        string    `json:"sharePct"`
	ContractCount   int       `json:"contractCount"`
}

type optionsResponse struct {
	Asset           string             `json:"asset"`
	Exchange        string             `json:"exchange"`
	IndexPrice      string             `json:"indexPrice"`
	AsOf            time.Time          `json:"asOf"`
	OpenInterest    string             `json:"openInterest"`
	OpenInterestUsd string             `json:"openInterestUsd"`
	Volume          string             `json:"volume"`
	VolumeUsd       string             `json:"volumeUsd"`
	CallOi          string             `json:"callOi"`
	PutOi           string             `json:"putOi"`
	CallVolume      string             `json:"callVolume"`
	PutVolume       string             `json:"putVolume"`
	PutCallOi       string             `json:"putCallOi,omitempty"`
	PutCallVolume   string             `json:"putCallVolume,omitempty"`
	Lean            string             `json:"lean,omitempty"`
	Dvol            string             `json:"dvol,omitempty"`
	DvolChange24h   string             `json:"dvolChange24h,omitempty"`
	Atm             *optionsATMDTO     `json:"atm,omitempty"`
	Expiries        []optionsExpiryDTO `json:"expiries"`
	Listed          int                `json:"listed"`
	WithOi          int                `json:"withOi"`
	WithIv          int                `json:"withIv"`
	Note            string             `json:"note"`
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
		expiries = append(expiries, row)
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
	return out
}
