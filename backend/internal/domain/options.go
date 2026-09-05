package domain

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	OptionsAssetBTC = "BTC"
	OptionsAssetETH = "ETH"

	OptionSideCall = "call"
	OptionSidePut  = "put"

	OptionsLeanCalls    = "calls"
	OptionsLeanPuts     = "puts"
	OptionsLeanBalanced = "balanced"

	// OptionsATMTargetDTE is the DVOL-like tenor used to pick ATM IV.
	OptionsATMTargetDTE = 30
	OptionsATMMinDTE    = 7
	OptionsATMMaxDTE    = 45

	// Deribit options settle at 08:00 UTC.
	optionsExpiryHourUTC = 8

	optionsPCRPuts  = 1.2
	optionsPCRCalls = 0.8
)

// OptionQuote is one listed option contract from a venue book summary.
type OptionQuote struct {
	Instrument   string
	Asset        string
	Expiry       time.Time
	Strike       float64
	Side         string // call | put
	OpenInterest float64
	Volume       float64
	VolumeUSD    float64
	MarkIV       float64 // percent; 0 when the venue omitted it
	Underlying   float64
}

// OptionsDVOL is Deribit's 30-day implied volatility index.
type OptionsDVOL struct {
	Value     float64
	Change24h float64 // index points; ignored unless HasChange
	HasChange bool
}

// OptionsATM is the at-the-money implied vol nearest the 30-day tenor.
type OptionsATM struct {
	IV           float64 // percent
	Strike       float64
	Expiry       time.Time
	DaysToExpiry int
	CallIV       float64
	PutIV        float64
	HasCall      bool
	HasPut       bool
}

// OptionsExpiryBucket is open interest and volume for one expiry.
type OptionsExpiryBucket struct {
	Expiry          time.Time
	Label           string
	DaysToExpiry    int
	OpenInterest    float64
	OpenInterestUSD float64
	Volume          float64
	VolumeUSD       float64
	CallOI          float64
	PutOI           float64
	CallVolume      float64
	PutVolume       float64
	PutCallOI       float64
	HasPutCallOI    bool
	SharePct        float64
	ContractCount   int
}

// OptionsSnapshot is the CoinGlass-style options overview for one asset.
type OptionsSnapshot struct {
	Asset           string
	Exchange        string
	IndexPrice      float64
	AsOf            time.Time
	OpenInterest    float64
	OpenInterestUSD float64
	Volume          float64
	VolumeUSD       float64
	CallOI          float64
	PutOI           float64
	CallVolume      float64
	PutVolume       float64
	PutCallOI       float64
	HasPutCallOI    bool
	PutCallVolume   float64
	HasPutCallVol   bool
	Lean            string
	DVOL            float64
	HasDVOL         bool
	DVOLChange24h   float64
	HasDVOLChange   bool
	ATM             *OptionsATM
	Expiries        []OptionsExpiryBucket
	Listed          int
	WithOI          int
	WithIV          int
	Note            string
}

// ParseOptionsAsset accepts BTC, ETH, or a common pair (BTCUSDT).
func ParseOptionsAsset(raw string) (string, error) {
	s := strings.ToUpper(strings.TrimSpace(raw))
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "/", "")
	s = strings.ReplaceAll(s, "_", "")
	switch s {
	case OptionsAssetBTC, "BTCUSDT", "BTCUSD", "BTCUSDC", "BTCPERP":
		return OptionsAssetBTC, nil
	case OptionsAssetETH, "ETHUSDT", "ETHUSD", "ETHUSDC", "ETHPERP":
		return OptionsAssetETH, nil
	default:
		return "", fmt.Errorf("%w: asset must be BTC or ETH", ErrInvalidArgument)
	}
}

// ParseOptionsExchange accepts deribit (default). Empty is deribit.
func ParseOptionsExchange(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" || s == string(ExchangeDeribit) {
		return string(ExchangeDeribit), nil
	}
	return "", fmt.Errorf("%w: options exchange must be deribit", ErrInvalidArgument)
}

// ParseOptionInstrument decodes Deribit names like BTC-27MAR26-100000-C.
func ParseOptionInstrument(name string) (asset string, expiry time.Time, strike float64, side string, ok bool) {
	name = strings.ToUpper(strings.TrimSpace(name))
	parts := strings.Split(name, "-")
	if len(parts) != 4 {
		return "", time.Time{}, 0, "", false
	}
	asset = parts[0]
	if asset != OptionsAssetBTC && asset != OptionsAssetETH {
		return "", time.Time{}, 0, "", false
	}
	exp, err := parseDeribitExpiry(parts[1])
	if err != nil {
		return "", time.Time{}, 0, "", false
	}
	strike, err = strconv.ParseFloat(parts[2], 64)
	if err != nil || strike <= 0 {
		return "", time.Time{}, 0, "", false
	}
	switch parts[3] {
	case "C":
		side = OptionSideCall
	case "P":
		side = OptionSidePut
	default:
		return "", time.Time{}, 0, "", false
	}
	return asset, exp, strike, side, true
}

func parseDeribitExpiry(raw string) (time.Time, error) {
	// D-MON-YY (5SEP25) or DD-MON-YY (27MAR26).
	var monStart int
	switch len(raw) {
	case 6:
		monStart = 1
	case 7:
		monStart = 2
	default:
		return time.Time{}, fmt.Errorf("bad expiry")
	}
	day, err := strconv.Atoi(raw[:monStart])
	if err != nil || day < 1 || day > 31 {
		return time.Time{}, fmt.Errorf("bad day")
	}
	mon, ok := deribitMonth[raw[monStart : monStart+3]]
	if !ok {
		return time.Time{}, fmt.Errorf("bad month")
	}
	yy, err := strconv.Atoi(raw[monStart+3:])
	if err != nil {
		return time.Time{}, fmt.Errorf("bad year")
	}
	year := 2000 + yy
	return time.Date(year, mon, day, optionsExpiryHourUTC, 0, 0, 0, time.UTC), nil
}

var deribitMonth = map[string]time.Month{
	"JAN": time.January, "FEB": time.February, "MAR": time.March,
	"APR": time.April, "MAY": time.May, "JUN": time.June,
	"JUL": time.July, "AUG": time.August, "SEP": time.September,
	"OCT": time.October, "NOV": time.November, "DEC": time.December,
}

// OptionsExpiryLabel is a short desk label (27 Mar 2026).
func OptionsExpiryLabel(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2 Jan 2006")
}

// OptionsDaysToExpiry is whole UTC days from asOf to expiry (can be 0).
func OptionsDaysToExpiry(expiry, asOf time.Time) int {
	if expiry.IsZero() {
		return 0
	}
	a := asOf.UTC().Truncate(24 * time.Hour)
	e := expiry.UTC().Truncate(24 * time.Hour)
	return int(e.Sub(a).Hours() / 24)
}

// BuildOptionsSnapshot aggregates listed quotes into the desk overview.
func BuildOptionsSnapshot(asset string, quotes []OptionQuote, index float64, dvol OptionsDVOL, asOf time.Time) *OptionsSnapshot {
	out := &OptionsSnapshot{
		Asset:    asset,
		Exchange: string(ExchangeDeribit),
		AsOf:     asOf.UTC(),
		Note:     "Deribit inverse options. Open interest and volume are in the base coin. Implied vol is percent. Informational only — not financial advice.",
	}
	if index > 0 {
		out.IndexPrice = index
	}
	if dvol.Value > 0 {
		out.DVOL = dvol.Value
		out.HasDVOL = true
	}
	if dvol.HasChange {
		out.DVOLChange24h = dvol.Change24h
		out.HasDVOLChange = true
	}

	type expKey struct {
		unix int64
	}
	buckets := map[int64]*OptionsExpiryBucket{}
	for i := range quotes {
		q := quotes[i]
		if q.Asset != "" && q.Asset != asset {
			continue
		}
		out.Listed++
		if q.OpenInterest > 0 {
			out.WithOI++
		}
		if q.MarkIV > 0 {
			out.WithIV++
		}
		if q.Underlying > 0 && out.IndexPrice <= 0 {
			out.IndexPrice = q.Underlying
		}
		px := out.IndexPrice
		if q.Underlying > 0 {
			px = q.Underlying
		}
		oiUSD := 0.0
		if px > 0 && q.OpenInterest > 0 {
			oiUSD = q.OpenInterest * px
		}
		out.OpenInterest += q.OpenInterest
		out.OpenInterestUSD += oiUSD
		out.Volume += q.Volume
		if q.VolumeUSD > 0 {
			out.VolumeUSD += q.VolumeUSD
		} else if px > 0 {
			out.VolumeUSD += q.Volume * px
		}
		switch q.Side {
		case OptionSideCall:
			out.CallOI += q.OpenInterest
			out.CallVolume += q.Volume
		case OptionSidePut:
			out.PutOI += q.OpenInterest
			out.PutVolume += q.Volume
		}
		if q.Expiry.IsZero() {
			continue
		}
		key := q.Expiry.UTC().Unix()
		b := buckets[key]
		if b == nil {
			b = &OptionsExpiryBucket{
				Expiry:       q.Expiry.UTC(),
				Label:        OptionsExpiryLabel(q.Expiry),
				DaysToExpiry: OptionsDaysToExpiry(q.Expiry, asOf),
			}
			buckets[key] = b
		}
		b.OpenInterest += q.OpenInterest
		b.OpenInterestUSD += oiUSD
		b.Volume += q.Volume
		if q.VolumeUSD > 0 {
			b.VolumeUSD += q.VolumeUSD
		} else if px > 0 {
			b.VolumeUSD += q.Volume * px
		}
		b.ContractCount++
		switch q.Side {
		case OptionSideCall:
			b.CallOI += q.OpenInterest
			b.CallVolume += q.Volume
		case OptionSidePut:
			b.PutOI += q.OpenInterest
			b.PutVolume += q.Volume
		}
	}

	if out.CallOI > 0 {
		out.PutCallOI = out.PutOI / out.CallOI
		out.HasPutCallOI = true
		out.Lean = optionsLean(out.PutCallOI)
	}
	if out.CallVolume > 0 {
		out.PutCallVolume = out.PutVolume / out.CallVolume
		out.HasPutCallVol = true
	}

	out.Expiries = make([]OptionsExpiryBucket, 0, len(buckets))
	for _, b := range buckets {
		if out.OpenInterest > 0 {
			b.SharePct = 100 * b.OpenInterest / out.OpenInterest
		}
		if b.CallOI > 0 {
			b.PutCallOI = b.PutOI / b.CallOI
			b.HasPutCallOI = true
		}
		out.Expiries = append(out.Expiries, *b)
	}
	sort.Slice(out.Expiries, func(i, j int) bool {
		return out.Expiries[i].Expiry.Before(out.Expiries[j].Expiry)
	})

	if atm := pickATMIV(quotes, out.IndexPrice, asOf); atm != nil {
		out.ATM = atm
	}
	return out
}

func optionsLean(pcr float64) string {
	switch {
	case pcr >= optionsPCRPuts:
		return OptionsLeanPuts
	case pcr <= optionsPCRCalls:
		return OptionsLeanCalls
	default:
		return OptionsLeanBalanced
	}
}

func pickATMIV(quotes []OptionQuote, index float64, asOf time.Time) *OptionsATM {
	if index <= 0 || len(quotes) == 0 {
		return nil
	}
	type expGroup struct {
		expiry time.Time
		dte    int
		quotes []OptionQuote
	}
	byExp := map[int64]*expGroup{}
	for i := range quotes {
		q := quotes[i]
		if q.Expiry.IsZero() || q.MarkIV <= 0 || q.Strike <= 0 {
			continue
		}
		dte := OptionsDaysToExpiry(q.Expiry, asOf)
		if dte < 0 {
			continue
		}
		key := q.Expiry.UTC().Unix()
		g := byExp[key]
		if g == nil {
			g = &expGroup{expiry: q.Expiry.UTC(), dte: dte}
			byExp[key] = g
		}
		g.quotes = append(g.quotes, q)
	}
	if len(byExp) == 0 {
		return nil
	}
	var preferred []*expGroup
	var all []*expGroup
	for _, g := range byExp {
		all = append(all, g)
		if g.dte >= OptionsATMMinDTE && g.dte <= OptionsATMMaxDTE {
			preferred = append(preferred, g)
		}
	}
	pool := preferred
	if len(pool) == 0 {
		pool = all
	}
	best := pool[0]
	bestDist := absInt(best.dte - OptionsATMTargetDTE)
	for _, g := range pool[1:] {
		d := absInt(g.dte - OptionsATMTargetDTE)
		if d < bestDist || (d == bestDist && g.dte < best.dte) {
			best = g
			bestDist = d
		}
	}

	// Closest strike to index; average call+put when both sides exist.
	type strikeIV struct {
		strike float64
		call   float64
		put    float64
		hasC   bool
		hasP   bool
	}
	byStrike := map[float64]*strikeIV{}
	for _, q := range best.quotes {
		s := byStrike[q.Strike]
		if s == nil {
			s = &strikeIV{strike: q.Strike}
			byStrike[q.Strike] = s
		}
		switch q.Side {
		case OptionSideCall:
			s.call = q.MarkIV
			s.hasC = true
		case OptionSidePut:
			s.put = q.MarkIV
			s.hasP = true
		}
	}
	var pick *strikeIV
	bestGap := math.MaxFloat64
	for _, s := range byStrike {
		gap := math.Abs(s.strike - index)
		if gap < bestGap {
			pick = s
			bestGap = gap
		}
	}
	if pick == nil || (!pick.hasC && !pick.hasP) {
		return nil
	}
	atm := &OptionsATM{
		Strike:       pick.strike,
		Expiry:       best.expiry,
		DaysToExpiry: best.dte,
		CallIV:       pick.call,
		PutIV:        pick.put,
		HasCall:      pick.hasC,
		HasPut:       pick.hasP,
	}
	switch {
	case pick.hasC && pick.hasP:
		atm.IV = (pick.call + pick.put) / 2
	case pick.hasC:
		atm.IV = pick.call
	default:
		atm.IV = pick.put
	}
	return atm
}
