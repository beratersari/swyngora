package domain

import (
	"math"
	"sort"
	"time"
)

const (
	GammaRegimeLong  = "long"  // dealers long gamma — fade moves (stabler)
	GammaRegimeShort = "short" // dealers short gamma — chase moves (faster)
	GammaRegimeFlat  = "flat"

	GammaFlipAbove = "above" // index is above the flip
	GammaFlipBelow = "below"
	GammaFlipAt    = "at"

	// Floor so same-day options do not explode gamma.
	optionsGammaMinYearFrac = 1.0 / (365.0 * 24.0)

	// |net| / (|call|+|put|) below this is treated as flat.
	optionsGammaFlatRatio = 0.05

	// Flip profile is listed strikes inside this band of the index.
	optionsGammaProfileBand = 0.20
	optionsGammaProfileMax  = 61
)

// OptionsGamma is dealer gamma exposure at the current index plus the flip.
//
// Convention (SpotGamma-style, documented): dealers are treated as short
// calls and long puts. Call GEX is negative, put GEX is positive.
// Units are USD of delta for a 1% spot move (γ × OI × S² × 0.01).
// Positive net = long gamma = they fade the tape (price can stay calmer).
// Negative net = short gamma = they chase (moves can run faster).
// Flip is the index where net GEX crosses zero when the book is re-gammad.
type OptionsGamma struct {
	NetUSD      float64
	CallUSD     float64
	PutUSD      float64
	Regime      string
	Flip        float64
	HasFlip     bool
	FlipSide    string
	FlipDistPct float64
	Used        int
	Skipped     int
	Profile     []OptionsGammaLevel
}

// OptionsGammaLevel is dealer GEX if the index were at Spot (recomputed gamma).
type OptionsGammaLevel struct {
	Spot    float64
	NetUSD  float64
	CallUSD float64
	PutUSD  float64
}

type gammaContract struct {
	strike float64
	side   string
	oi     float64
	iv     float64 // decimal vol
	t      float64 // year fraction
}

// BlackScholesGamma is vanilla BS gamma (call = put). r = q = 0.
func BlackScholesGamma(spot, strike, sigma, yearFrac float64) float64 {
	if spot <= 0 || strike <= 0 || sigma <= 0 || yearFrac <= 0 {
		return 0
	}
	t := yearFrac
	if t < optionsGammaMinYearFrac {
		t = optionsGammaMinYearFrac
	}
	sqrtT := math.Sqrt(t)
	den := sigma * sqrtT
	if den <= 0 || math.IsInf(den, 0) || math.IsNaN(den) {
		return 0
	}
	d1 := (math.Log(spot/strike) + 0.5*sigma*sigma*t) / den
	if math.IsInf(d1, 0) || math.IsNaN(d1) {
		return 0
	}
	nd1 := math.Exp(-0.5*d1*d1) / math.Sqrt(2*math.Pi)
	g := nd1 / (spot * den)
	if math.IsInf(g, 0) || math.IsNaN(g) || g < 0 {
		return 0
	}
	return g
}

// DealerGammaUSD is USD delta change for a 1% spot move on one contract's OI.
// Call contribution is negative; put contribution is positive.
func DealerGammaUSD(spot, strike, sigma, yearFrac, oi float64, side string) (callUSD, putUSD float64) {
	if oi <= 0 || spot <= 0 {
		return 0, 0
	}
	g := BlackScholesGamma(spot, strike, sigma, yearFrac)
	if g <= 0 {
		return 0, 0
	}
	scale := g * oi * spot * spot * 0.01
	switch side {
	case OptionSideCall:
		return -scale, 0
	case OptionSidePut:
		return 0, scale
	default:
		return 0, 0
	}
}

func collectGammaContracts(quotes []OptionQuote, asOf time.Time) (used []gammaContract, skipped int) {
	for i := range quotes {
		q := quotes[i]
		if q.OpenInterest <= 0 || q.Strike <= 0 {
			continue
		}
		if q.MarkIV <= 0 || q.Expiry.IsZero() {
			skipped++
			continue
		}
		yf := OptionsYearFraction(OptionsDaysToExpiry(q.Expiry, asOf), q.Expiry, asOf)
		if yf <= 0 {
			skipped++
			continue
		}
		used = append(used, gammaContract{
			strike: q.Strike,
			side:   q.Side,
			oi:     q.OpenInterest,
			iv:     q.MarkIV / 100,
			t:      yf,
		})
	}
	return used, skipped
}

func sumDealerGamma(contracts []gammaContract, spot float64) (net, call, put float64) {
	for _, c := range contracts {
		cu, pu := DealerGammaUSD(spot, c.strike, c.iv, c.t, c.oi, c.side)
		call += cu
		put += pu
	}
	return call + put, call, put
}

func gammaRegime(net, call, put float64) string {
	den := math.Abs(call) + math.Abs(put)
	if den <= 0 {
		return GammaRegimeFlat
	}
	if math.Abs(net)/den < optionsGammaFlatRatio {
		return GammaRegimeFlat
	}
	if net > 0 {
		return GammaRegimeLong
	}
	return GammaRegimeShort
}

func gammaFlipSide(index, flip float64) (side string, distPct float64) {
	if index <= 0 || flip <= 0 {
		return "", 0
	}
	distPct = 100 * (index - flip) / index
	switch {
	case math.Abs(distPct) < 0.05:
		return GammaFlipAt, 0
	case index > flip:
		return GammaFlipAbove, distPct
	default:
		return GammaFlipBelow, distPct
	}
}

func uniqueStrikesInBand(contracts []gammaContract, index, band float64) []float64 {
	if index <= 0 {
		return nil
	}
	lo, hi := index*(1-band), index*(1+band)
	seen := map[float64]struct{}{}
	out := make([]float64, 0, 32)
	for _, c := range contracts {
		if c.strike < lo || c.strike > hi {
			continue
		}
		if _, ok := seen[c.strike]; ok {
			continue
		}
		seen[c.strike] = struct{}{}
		out = append(out, c.strike)
	}
	sort.Float64s(out)
	return out
}

func downsampleSpots(spots []float64, index float64, max int) []float64 {
	if max < 3 || len(spots) <= max {
		return spots
	}
	// Keep endpoints, the strike nearest the index, and an even sample.
	keep := map[float64]struct{}{
		spots[0]:            {},
		spots[len(spots)-1]: {},
	}
	nearest := spots[0]
	best := math.Abs(spots[0] - index)
	for _, s := range spots[1:] {
		d := math.Abs(s - index)
		if d < best {
			nearest = s
			best = d
		}
	}
	keep[nearest] = struct{}{}
	step := float64(len(spots)-1) / float64(max-1)
	for i := 0; i < max; i++ {
		idx := int(math.Round(float64(i) * step))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(spots) {
			idx = len(spots) - 1
		}
		keep[spots[idx]] = struct{}{}
	}
	out := make([]float64, 0, len(keep))
	for s := range keep {
		out = append(out, s)
	}
	sort.Float64s(out)
	return out
}

func interpolateGammaFlip(profile []OptionsGammaLevel, index float64) (flip float64, ok bool) {
	if len(profile) < 2 {
		return 0, false
	}
	type cross struct {
		spot        float64
		plusToMinus bool
	}
	var crosses []cross
	for i := 0; i < len(profile)-1; i++ {
		a, b := profile[i], profile[i+1]
		if a.Spot >= b.Spot {
			continue
		}
		if a.NetUSD == 0 {
			crosses = append(crosses, cross{spot: a.Spot, plusToMinus: b.NetUSD < 0})
			continue
		}
		if a.NetUSD*b.NetUSD > 0 {
			continue
		}
		if b.NetUSD == 0 {
			crosses = append(crosses, cross{spot: b.Spot, plusToMinus: a.NetUSD > 0})
			continue
		}
		den := math.Abs(a.NetUSD) + math.Abs(b.NetUSD)
		if den <= 0 {
			continue
		}
		spot := a.Spot + (b.Spot-a.Spot)*math.Abs(a.NetUSD)/den
		crosses = append(crosses, cross{spot: spot, plusToMinus: a.NetUSD > 0 && b.NetUSD < 0})
	}
	if len(crosses) == 0 {
		return 0, false
	}
	// Prefer the classic + → − crossing closest to the index; else nearest any zero.
	best := crosses[0]
	bestDist := math.Abs(best.spot - index)
	bestPref := best.plusToMinus
	for _, c := range crosses[1:] {
		d := math.Abs(c.spot - index)
		pref := c.plusToMinus
		if pref && !bestPref {
			best, bestDist, bestPref = c, d, pref
			continue
		}
		if pref == bestPref && d < bestDist {
			best, bestDist = c, d
		}
	}
	return best.spot, true
}

// BuildOptionsGamma aggregates dealer GEX and the zero-gamma flip from listed quotes.
func BuildOptionsGamma(quotes []OptionQuote, index float64, asOf time.Time) *OptionsGamma {
	if index <= 0 {
		return nil
	}
	used, skipped := collectGammaContracts(quotes, asOf)
	if len(used) == 0 {
		return nil
	}
	net, call, put := sumDealerGamma(used, index)
	out := &OptionsGamma{
		NetUSD:  net,
		CallUSD: call,
		PutUSD:  put,
		Regime:  gammaRegime(net, call, put),
		Used:    len(used),
		Skipped: skipped,
	}
	spots := uniqueStrikesInBand(used, index, optionsGammaProfileBand)
	spots = downsampleSpots(spots, index, optionsGammaProfileMax)
	if len(spots) == 0 {
		spots = []float64{index}
	}
	out.Profile = make([]OptionsGammaLevel, 0, len(spots))
	for _, s := range spots {
		n, c, p := sumDealerGamma(used, s)
		out.Profile = append(out.Profile, OptionsGammaLevel{Spot: s, NetUSD: n, CallUSD: c, PutUSD: p})
	}
	if flip, ok := interpolateGammaFlip(out.Profile, index); ok {
		out.Flip = flip
		out.HasFlip = true
		out.FlipSide, out.FlipDistPct = gammaFlipSide(index, flip)
	}
	return out
}

func attachStrikeGamma(b *OptionsExpiryBucket, quotes []OptionQuote, index float64, asOf time.Time) {
	if b == nil || len(b.Strikes) == 0 || index <= 0 {
		return
	}
	type sideIV struct {
		callOI, putOI float64
		callIV, putIV float64
		hasC, hasP    bool
		t             float64
	}
	byStrike := map[float64]*sideIV{}
	for i := range quotes {
		q := quotes[i]
		if q.Strike <= 0 || q.OpenInterest <= 0 || q.MarkIV <= 0 || q.Expiry.IsZero() {
			continue
		}
		yf := OptionsYearFraction(OptionsDaysToExpiry(q.Expiry, asOf), q.Expiry, asOf)
		if yf <= 0 {
			continue
		}
		s := byStrike[q.Strike]
		if s == nil {
			s = &sideIV{t: yf}
			byStrike[q.Strike] = s
		}
		switch q.Side {
		case OptionSideCall:
			s.callOI += q.OpenInterest
			s.callIV = q.MarkIV / 100
			s.hasC = true
		case OptionSidePut:
			s.putOI += q.OpenInterest
			s.putIV = q.MarkIV / 100
			s.hasP = true
		}
	}
	for i := range b.Strikes {
		s := byStrike[b.Strikes[i].Strike]
		if s == nil {
			continue
		}
		var call, put float64
		if s.hasC {
			c, _ := DealerGammaUSD(index, b.Strikes[i].Strike, s.callIV, s.t, s.callOI, OptionSideCall)
			call = c
		}
		if s.hasP {
			_, p := DealerGammaUSD(index, b.Strikes[i].Strike, s.putIV, s.t, s.putOI, OptionSidePut)
			put = p
		}
		if call == 0 && put == 0 {
			continue
		}
		b.Strikes[i].GammaCallUSD = call
		b.Strikes[i].GammaPutUSD = put
		b.Strikes[i].GammaNetUSD = call + put
		b.Strikes[i].HasGamma = true
	}
}

func attachOptionsGamma(out *OptionsSnapshot, quotes []OptionQuote, asOf time.Time) {
	if out == nil {
		return
	}
	if g := BuildOptionsGamma(quotes, out.IndexPrice, asOf); g != nil {
		out.Gamma = g
	}
	byExp := map[int64][]OptionQuote{}
	for i := range quotes {
		q := quotes[i]
		if q.Expiry.IsZero() {
			continue
		}
		k := q.Expiry.UTC().Unix()
		byExp[k] = append(byExp[k], q)
	}
	for i := range out.Expiries {
		eq := byExp[out.Expiries[i].Expiry.Unix()]
		if len(eq) == 0 {
			continue
		}
		if g := BuildOptionsGamma(eq, out.IndexPrice, asOf); g != nil {
			out.Expiries[i].GammaNetUSD = g.NetUSD
			out.Expiries[i].GammaCallUSD = g.CallUSD
			out.Expiries[i].GammaPutUSD = g.PutUSD
			out.Expiries[i].GammaRegime = g.Regime
			out.Expiries[i].HasGamma = true
			if g.HasFlip {
				out.Expiries[i].GammaFlip = g.Flip
				out.Expiries[i].HasGammaFlip = true
				out.Expiries[i].GammaFlipSide = g.FlipSide
				out.Expiries[i].GammaFlipDistPct = g.FlipDistPct
			}
		}
		attachStrikeGamma(&out.Expiries[i], eq, out.IndexPrice, asOf)
	}
}
