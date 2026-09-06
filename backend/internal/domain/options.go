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

	// ATM put IV minus ATM call IV (vol points) for a "puts richer" lean.
	optionsSkewPuts  = 1.0
	optionsSkewCalls = -1.0
	// OptionsWingPct is the OTM distance used for wing skew (~5% from spot).
	OptionsWingPct = 0.05
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
	AtmIV           float64
	CallIV          float64
	PutIV           float64
	Skew            float64 // ATM put IV − call IV (vol points)
	HasAtmIV        bool
	HasCallIV       bool
	HasPutIV        bool
	HasSkew         bool
	// 1-SD expected move from ATM IV until this expiry.
	ExpectedMoveAbs float64
	ExpectedMovePct float64
	ExpectedLow     float64
	ExpectedHigh    float64
	HasExpectedMove bool
	// Max pain: listed strike that minimizes call+put intrinsic payout.
	MaxPain       float64
	MaxPainPayout float64
	HasMaxPain    bool
	Strikes       []OptionsExpiryStrike
}

// OptionsExpiryStrike is call/put OI and total intrinsic if spot expires there.
type OptionsExpiryStrike struct {
	Strike    float64
	CallOI    float64
	PutOI     float64
	Payout    float64 // USD intrinsic if spot = strike (coin OI × USD payoff)
	IsMaxPain bool
	BelowSpot bool
}

// OptionsVolPoint is ATM implied vol on one expiry (term structure / skew).
type OptionsVolPoint struct {
	Expiry       time.Time
	Label        string
	DaysToExpiry int
	Strike       float64
	AtmIV        float64
	CallIV       float64
	PutIV        float64
	Skew         float64
	HasAtm       bool
	HasCall      bool
	HasPut       bool
	HasSkew      bool
}

// OptionsSkewSummary is headline term slope and call/put vol difference.
type OptionsSkewSummary struct {
	AtmSkew    float64
	HasAtmSkew bool
	CallIV     float64
	PutIV      float64
	Lean       string // puts | calls | balanced — which side is richer
	FrontIV    float64
	BackIV     float64
	TermSlope  float64 // back ATM − front ATM (vol points)
	HasTerm    bool
	FrontDTE   int
	BackDTE    int
	WingSkew   float64 // OTM put IV − OTM call IV on the 30d tenor
	HasWing    bool
	WingPutIV  float64
	WingCallIV float64
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
	TermStructure   []OptionsVolPoint
	Skew            *OptionsSkewSummary
	Expiries        []OptionsExpiryBucket
	Listed          int
	WithOI          int
	WithIV          int
	VsMarket        *OptionsVsMarket
	History         *OptionsHistory
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
	mon, ok := deribitMonth[raw[monStart:monStart+3]]
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

	buckets := map[int64]*OptionsExpiryBucket{}
	strikeBooks := map[int64]map[float64]*optionsStrikeBook{}
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
		if q.Strike > 0 && q.OpenInterest > 0 {
			book := strikeBooks[key]
			if book == nil {
				book = map[float64]*optionsStrikeBook{}
				strikeBooks[key] = book
			}
			st := book[q.Strike]
			if st == nil {
				st = &optionsStrikeBook{strike: q.Strike}
				book[q.Strike] = st
			}
			switch q.Side {
			case OptionSideCall:
				st.callOI += q.OpenInterest
			case OptionSidePut:
				st.putOI += q.OpenInterest
			}
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

	term, groups := buildOptionsTermStructure(quotes, out.IndexPrice, asOf)
	out.TermStructure = term
	volByExp := map[int64]OptionsVolPoint{}
	for _, p := range term {
		volByExp[p.Expiry.Unix()] = p
	}
	for i := range out.Expiries {
		p, ok := volByExp[out.Expiries[i].Expiry.Unix()]
		if !ok {
			continue
		}
		out.Expiries[i].AtmIV = p.AtmIV
		out.Expiries[i].CallIV = p.CallIV
		out.Expiries[i].PutIV = p.PutIV
		out.Expiries[i].Skew = p.Skew
		out.Expiries[i].HasAtmIV = p.HasAtm
		out.Expiries[i].HasCallIV = p.HasCall
		out.Expiries[i].HasPutIV = p.HasPut
		out.Expiries[i].HasSkew = p.HasSkew
	}
	for i := range out.Expiries {
		attachExpectedMove(&out.Expiries[i], out.IndexPrice, asOf)
		attachMaxPain(&out.Expiries[i], strikeBooks[out.Expiries[i].Expiry.Unix()], out.IndexPrice)
	}
	if atm := pickATMIVFromGroups(groups, out.IndexPrice); atm != nil {
		out.ATM = atm
	}
	out.Skew = buildOptionsSkew(out.ATM, term, groups, out.IndexPrice)
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

type optionExpGroup struct {
	expiry time.Time
	dte    int
	quotes []OptionQuote
}

type strikeIV struct {
	strike float64
	call   float64
	put    float64
	hasC   bool
	hasP   bool
}

func groupOptionQuotesByExpiry(quotes []OptionQuote, asOf time.Time) []*optionExpGroup {
	byExp := map[int64]*optionExpGroup{}
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
			g = &optionExpGroup{expiry: q.Expiry.UTC(), dte: dte}
			byExp[key] = g
		}
		g.quotes = append(g.quotes, q)
	}
	out := make([]*optionExpGroup, 0, len(byExp))
	for _, g := range byExp {
		out = append(out, g)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].expiry.Before(out[j].expiry)
	})
	return out
}

func closestStrikeIV(quotes []OptionQuote, target float64) *strikeIV {
	if target <= 0 || len(quotes) == 0 {
		return nil
	}
	byStrike := map[float64]*strikeIV{}
	for _, q := range quotes {
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
		gap := math.Abs(s.strike - target)
		if gap < bestGap {
			pick = s
			bestGap = gap
		}
	}
	if pick == nil || (!pick.hasC && !pick.hasP) {
		return nil
	}
	return pick
}

func volPointFromStrike(g *optionExpGroup, s *strikeIV) OptionsVolPoint {
	p := OptionsVolPoint{
		Expiry:       g.expiry,
		Label:        OptionsExpiryLabel(g.expiry),
		DaysToExpiry: g.dte,
		Strike:       s.strike,
		CallIV:       s.call,
		PutIV:        s.put,
		HasCall:      s.hasC,
		HasPut:       s.hasP,
	}
	switch {
	case s.hasC && s.hasP:
		p.AtmIV = (s.call + s.put) / 2
		p.HasAtm = true
		p.Skew = s.put - s.call
		p.HasSkew = true
	case s.hasC:
		p.AtmIV = s.call
		p.HasAtm = true
	default:
		p.AtmIV = s.put
		p.HasAtm = true
	}
	return p
}

func buildOptionsTermStructure(quotes []OptionQuote, index float64, asOf time.Time) ([]OptionsVolPoint, []*optionExpGroup) {
	groups := groupOptionQuotesByExpiry(quotes, asOf)
	if index <= 0 {
		return nil, groups
	}
	out := make([]OptionsVolPoint, 0, len(groups))
	for _, g := range groups {
		s := closestStrikeIV(g.quotes, index)
		if s == nil {
			continue
		}
		out = append(out, volPointFromStrike(g, s))
	}
	return out, groups
}

func pickATMIVFromGroups(groups []*optionExpGroup, index float64) *OptionsATM {
	if len(groups) == 0 {
		return nil
	}
	var preferred []*optionExpGroup
	for _, g := range groups {
		if g.dte >= OptionsATMMinDTE && g.dte <= OptionsATMMaxDTE {
			preferred = append(preferred, g)
		}
	}
	pool := preferred
	if len(pool) == 0 {
		pool = groups
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
	target := index
	if target <= 0 {
		var sum float64
		n := 0
		for _, q := range best.quotes {
			if q.Underlying > 0 {
				target = q.Underlying
				break
			}
			sum += q.Strike
			n++
		}
		if target <= 0 && n > 0 {
			target = sum / float64(n)
		}
	}
	s := closestStrikeIV(best.quotes, target)
	if s == nil {
		return nil
	}
	p := volPointFromStrike(best, s)
	return &OptionsATM{
		Strike:       p.Strike,
		Expiry:       p.Expiry,
		DaysToExpiry: p.DaysToExpiry,
		CallIV:       p.CallIV,
		PutIV:        p.PutIV,
		HasCall:      p.HasCall,
		HasPut:       p.HasPut,
		IV:           p.AtmIV,
	}
}

func buildOptionsSkew(atm *OptionsATM, term []OptionsVolPoint, groups []*optionExpGroup, index float64) *OptionsSkewSummary {
	if atm == nil && len(term) == 0 {
		return nil
	}
	out := &OptionsSkewSummary{}
	if atm != nil && atm.HasCall && atm.HasPut {
		out.CallIV = atm.CallIV
		out.PutIV = atm.PutIV
		out.AtmSkew = atm.PutIV - atm.CallIV
		out.HasAtmSkew = true
		out.Lean = optionsVolLean(out.AtmSkew)
	}
	if len(term) >= 2 {
		front, back := term[0], term[len(term)-1]
		if front.HasAtm && back.HasAtm {
			out.FrontIV = front.AtmIV
			out.BackIV = back.AtmIV
			out.FrontDTE = front.DaysToExpiry
			out.BackDTE = back.DaysToExpiry
			out.TermSlope = back.AtmIV - front.AtmIV
			out.HasTerm = true
		}
	}
	if index > 0 && atm != nil {
		var g *optionExpGroup
		for _, cand := range groups {
			if cand.expiry.Equal(atm.Expiry) {
				g = cand
				break
			}
		}
		if g != nil {
			putS := closestStrikeIV(filterSide(g.quotes, OptionSidePut), index*(1-OptionsWingPct))
			callS := closestStrikeIV(filterSide(g.quotes, OptionSideCall), index*(1+OptionsWingPct))
			if putS != nil && putS.hasP && callS != nil && callS.hasC {
				out.WingPutIV = putS.put
				out.WingCallIV = callS.call
				out.WingSkew = putS.put - callS.call
				out.HasWing = true
			}
		}
	}
	if !out.HasAtmSkew && !out.HasTerm && !out.HasWing {
		return nil
	}
	return out
}

func filterSide(quotes []OptionQuote, side string) []OptionQuote {
	out := make([]OptionQuote, 0, len(quotes))
	for _, q := range quotes {
		if q.Side == side {
			out = append(out, q)
		}
	}
	return out
}

func optionsVolLean(skew float64) string {
	switch {
	case skew >= optionsSkewPuts:
		return OptionsLeanPuts
	case skew <= optionsSkewCalls:
		return OptionsLeanCalls
	default:
		return OptionsLeanBalanced
	}
}

type optionsStrikeBook struct {
	strike float64
	callOI float64
	putOI  float64
}

// OptionsYearFraction is calendar time to expiry in years (365-day).
func OptionsYearFraction(dte int, expiry, asOf time.Time) float64 {
	if dte > 0 {
		return float64(dte) / 365
	}
	rem := expiry.Sub(asOf.UTC())
	if rem <= 0 {
		return 0
	}
	return rem.Hours() / (365 * 24)
}

// OptionsExpectedMove is the 1-SD ATM move: spot × (IV/100) × √T.
func OptionsExpectedMove(index, atmIVPct, yearFrac float64) (abs, pct, low, high float64, ok bool) {
	if index <= 0 || atmIVPct <= 0 || yearFrac <= 0 {
		return 0, 0, 0, 0, false
	}
	pct = atmIVPct * math.Sqrt(yearFrac)
	abs = index * pct / 100
	return abs, pct, index - abs, index + abs, true
}

func attachExpectedMove(b *OptionsExpiryBucket, index float64, asOf time.Time) {
	if b == nil || !b.HasAtmIV {
		return
	}
	yf := OptionsYearFraction(b.DaysToExpiry, b.Expiry, asOf)
	abs, pct, low, high, ok := OptionsExpectedMove(index, b.AtmIV, yf)
	if !ok {
		return
	}
	b.ExpectedMoveAbs = abs
	b.ExpectedMovePct = pct
	b.ExpectedLow = low
	b.ExpectedHigh = high
	b.HasExpectedMove = true
}

func optionPayoutAt(spot float64, book []*optionsStrikeBook) float64 {
	var p float64
	for _, s := range book {
		if spot > s.strike {
			p += s.callOI * (spot - s.strike)
		} else if spot < s.strike {
			p += s.putOI * (s.strike - spot)
		}
	}
	return p
}

func attachMaxPain(b *OptionsExpiryBucket, book map[float64]*optionsStrikeBook, index float64) {
	if b == nil || len(book) == 0 {
		return
	}
	list := make([]*optionsStrikeBook, 0, len(book))
	for _, s := range book {
		list = append(list, s)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].strike < list[j].strike })
	best := list[0].strike
	bestPay := optionPayoutAt(best, list)
	for _, s := range list[1:] {
		pay := optionPayoutAt(s.strike, list)
		if pay < bestPay {
			best = s.strike
			bestPay = pay
		}
	}
	b.MaxPain = best
	b.MaxPainPayout = bestPay
	b.HasMaxPain = true
	strikes := make([]OptionsExpiryStrike, 0, len(list))
	for _, s := range list {
		strikes = append(strikes, OptionsExpiryStrike{
			Strike:    s.strike,
			CallOI:    s.callOI,
			PutOI:     s.putOI,
			Payout:    optionPayoutAt(s.strike, list),
			IsMaxPain: s.strike == best,
			BelowSpot: index > 0 && s.strike < index,
		})
	}
	b.Strikes = strikes
}
