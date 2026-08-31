package domain

import (
	"math"
	"sort"
	"time"
)

const (
	// HuntMaintenanceMargin is a BTC-like USD-M / linear maintenance rate.
	HuntMaintenanceMargin = 0.004
	// HuntAccountBlend pulls account long/short toward 50/50 because the
	// published ratio is account count, not position size.
	HuntAccountBlend = 0.6
	// HuntLiquidationTakeRate is an insurance-fund-like slice of liquidated
	// notional (leftover margin / clear fee stand-in). Not a published fee.
	HuntLiquidationTakeRate = 0.005
	// HuntSpotTakerFee is a typical 0.10% spot taker on each side of the tour.
	HuntSpotTakerFee = 0.001
	// HuntCascadeFillRate is the share of estimated liquidation notional
	// assumed to become exit flow at the target (others trade / hide / fail).
	HuntCascadeFillRate = 0.5
	// HuntClusterBucketPct groups observed liquidations.
	HuntClusterBucketPct = 0.5
	maxHuntBands         = 8
)

// DefaultHuntLeverageMix is a stylized isolated-leverage distribution.
// Venues do not publish this; weights sum to 1.
var DefaultHuntLeverageMix = []HuntLeverageBucket{
	{Leverage: 5, Weight: 0.15},
	{Leverage: 10, Weight: 0.25},
	{Leverage: 25, Weight: 0.30},
	{Leverage: 50, Weight: 0.18},
	{Leverage: 75, Weight: 0.07},
	{Leverage: 100, Weight: 0.04},
	{Leverage: 125, Weight: 0.01},
}

// HuntLeverageBucket is one assumed leverage share of open interest.
type HuntLeverageBucket struct {
	Leverage float64 `json:"leverage"`
	Weight   float64 `json:"weight"`
}

// HuntAssumptions are the labeled model knobs returned to the caller.
type HuntAssumptions struct {
	MaintenanceMargin   float64              `json:"maintenanceMargin"`
	AccountBlend        float64              `json:"accountBlend"`
	LiquidationTakeRate float64              `json:"liquidationTakeRate"`
	SpotTakerFee        float64              `json:"spotTakerFee"`
	CascadeFillRate     float64              `json:"cascadeFillRate"`
	LeverageMix         []HuntLeverageBucket `json:"leverageMix"`
}

// DefaultHuntAssumptions returns the documented defaults.
func DefaultHuntAssumptions() HuntAssumptions {
	mix := append([]HuntLeverageBucket(nil), DefaultHuntLeverageMix...)
	return HuntAssumptions{
		MaintenanceMargin:   HuntMaintenanceMargin,
		AccountBlend:        HuntAccountBlend,
		LiquidationTakeRate: HuntLiquidationTakeRate,
		SpotTakerFee:        HuntSpotTakerFee,
		CascadeFillRate:     HuntCascadeFillRate,
		LeverageMix:         mix,
	}
}

// HuntBand is estimated (and maybe observed) liquidation pressure at one price.
type HuntBand struct {
	Side             string  `json:"side"`
	Direction        string  `json:"direction"`
	Leverage         float64 `json:"leverage,omitempty"`
	Price            float64 `json:"price"`
	MovePct          float64 `json:"movePct"`
	EstNotional      float64 `json:"estNotional"`
	ObservedNotional float64 `json:"observedNotional,omitempty"`
	Source           string  `json:"source"`
}

// HuntCluster is 24h realized liquidations in a price bucket.
type HuntCluster struct {
	Side     string  `json:"side"`
	Price    float64 `json:"price"`
	MovePct  float64 `json:"movePct"`
	Notional float64 `json:"notional"`
	Count    int     `json:"count"`
}

// HuntScenario is one hypothetical desk tour: push spot, hope for liquidations.
type HuntScenario struct {
	Direction           string    `json:"direction"`
	Thesis              string    `json:"thesis"`
	Target              HuntBand  `json:"target"`
	Spot                BookReach `json:"spot"`
	EstLiquidated       float64   `json:"estLiquidated"`
	CascadeExitNotional float64   `json:"cascadeExitNotional"`
	BookOnlyPnl         float64   `json:"bookOnlyPnl"`
	CascadeInventoryPnl float64   `json:"cascadeInventoryPnl"`
	LiquidationTake     float64   `json:"liquidationTake"`
	Fees                float64   `json:"fees"`
	NetBookOnly         float64   `json:"netBookOnly"`
	NetWithCascade      float64   `json:"netWithCascade"`
	HouseEdge           string    `json:"houseEdge"`
	Efficiency          float64   `json:"efficiency"`
}

// HuntVenueReport is the hunt model for one futures venue.
type HuntVenueReport struct {
	Exchange           Exchange           `json:"exchange"`
	Symbol             string             `json:"symbol"`
	Price              float64            `json:"price"`
	OpenInterestValue  float64            `json:"openInterestValue"`
	EstLongNotional    float64            `json:"estLongNotional"`
	EstShortNotional   float64            `json:"estShortNotional"`
	LongShare          float64            `json:"longShare"`
	ShortShare         float64            `json:"shortShare"`
	EstLongShare       float64            `json:"estLongShare"`
	EstShortShare      float64            `json:"estShortShare"`
	FundingRate        float64            `json:"fundingRate"`
	FundingPayer       string             `json:"fundingPayer"`
	VisibleBidNotional float64            `json:"visibleBidNotional"`
	VisibleAskNotional float64            `json:"visibleAskNotional"`
	UpPressure         []HuntBand         `json:"upPressure"`
	DownPressure       []HuntBand         `json:"downPressure"`
	Observed           []HuntCluster      `json:"observed"`
	UpHunt             HuntScenario       `json:"upHunt"`
	DownHunt           HuntScenario       `json:"downHunt"`
	UpScore            HuntDirectionScore `json:"upScore"`
	DownScore          HuntDirectionScore `json:"downScore"`
	Bias               HuntBias           `json:"bias"`
	Coverage           HuntCoverage       `json:"coverage"`
	Error              string             `json:"error,omitempty"`
}

// HuntReport is the API/use-case result. Venues are never averaged.
type HuntReport struct {
	Symbol      string            `json:"symbol"`
	Exchange    string            `json:"exchange"`
	AsOf        time.Time         `json:"asOf"`
	Assumptions HuntAssumptions   `json:"assumptions"`
	Venues      []HuntVenueReport `json:"venues"`
	Bias        *HuntBias         `json:"bias,omitempty"`
	Coverage    *HuntCoverage     `json:"coverage,omitempty"`
	Note        string            `json:"note,omitempty"`
}

const (
	HuntInputOK      = "ok"
	HuntInputWeak    = "weak"
	HuntInputMissing = "missing"
	HuntInputError   = "error"

	HuntCoverageComplete     = "complete"
	HuntCoverageUsable       = "usable"
	HuntCoverageThin         = "thin"
	HuntCoverageInsufficient = "insufficient"
)

// HuntInputStatus is one data source used by the hunt score.
type HuntInputStatus struct {
	ID       string  `json:"id"`
	Label    string  `json:"label"`
	Status   string  `json:"status"`
	Weight   float64 `json:"weight"`
	Detail   string  `json:"detail"`
	Have     string  `json:"have,omitempty"`
	Need     string  `json:"need,omitempty"`
	CoverPct float64 `json:"coverPct,omitempty"`
}

// HuntSpan is how much of a lookback window we actually have.
type HuntSpan struct {
	HaveSec  float64 `json:"haveSec"`
	NeedSec  float64 `json:"needSec"`
	CoverPct float64 `json:"coverPct"`
}

// HuntCoverage is how complete the hunt inputs are for one venue or the combined read.
type HuntCoverage struct {
	Score   float64           `json:"score"`
	Level   string            `json:"level"`
	Usable  bool              `json:"usable"`
	Inputs  []HuntInputStatus `json:"inputs"`
	Missing []string          `json:"missing"`
	Weak    []string          `json:"weak"`
	Summary string            `json:"summary"`
}

// HuntInputs feeds BuildHuntVenue.
type HuntInputs struct {
	Exchange     Exchange
	Symbol       string
	Price        float64
	OIValue      float64
	LongShare    float64
	ShortShare   float64
	FundingRate  float64
	Asks         []ImpactSourceLevel
	Bids         []ImpactSourceLevel
	Liquidations []LiquidationEvent
}

// HuntLiqDistance is isolated distance from entry as a fraction of price:
// 1/leverage − maintenance. Floored so extreme leverage still has a band.
func HuntLiqDistance(leverage, mmr float64) float64 {
	if leverage < 1 {
		leverage = 1
	}
	d := 1/leverage - mmr
	if d < 0.002 {
		return 0.002
	}
	return d
}

// BlendAccountShare converts account long-share into a cautious position proxy.
func BlendAccountShare(longShare float64) (estLong, estShort float64) {
	if longShare < 0 {
		longShare = 0
	}
	if longShare > 1 {
		longShare = 1
	}
	estLong = 0.5 + (longShare-0.5)*HuntAccountBlend
	if estLong < 0.15 {
		estLong = 0.15
	}
	if estLong > 0.85 {
		estLong = 0.85
	}
	return estLong, 1 - estLong
}

// TiltLeverageMix shifts weight toward high leverage on the crowded (paying) side.
func TiltLeverageMix(base []HuntLeverageBucket, crowded bool) []HuntLeverageBucket {
	out := append([]HuntLeverageBucket(nil), base...)
	if len(out) < 3 {
		return normalizeMix(out)
	}
	shift := 0.10
	if crowded {
		shift = 0.15
	}
	lowN, highN := 2, 2
	if !crowded {
		// receiving funding: more weight on low leverage
		give := shift / float64(highN)
		take := shift / float64(lowN)
		for i := len(out) - highN; i < len(out); i++ {
			if i >= 0 {
				out[i].Weight -= give
				if out[i].Weight < 0 {
					out[i].Weight = 0
				}
			}
		}
		for i := 0; i < lowN && i < len(out); i++ {
			out[i].Weight += take
		}
		return normalizeMix(out)
	}
	give := shift / float64(lowN)
	take := shift / float64(highN)
	for i := 0; i < lowN && i < len(out); i++ {
		out[i].Weight -= give
		if out[i].Weight < 0 {
			out[i].Weight = 0
		}
	}
	for i := len(out) - highN; i < len(out); i++ {
		if i >= 0 {
			out[i].Weight += take
		}
	}
	return normalizeMix(out)
}

func normalizeMix(in []HuntLeverageBucket) []HuntLeverageBucket {
	var sum float64
	for _, b := range in {
		if b.Weight > 0 {
			sum += b.Weight
		}
	}
	if sum <= 0 {
		return in
	}
	for i := range in {
		if in[i].Weight < 0 {
			in[i].Weight = 0
		}
		in[i].Weight /= sum
	}
	return in
}

// ClusterHuntLiquidations groups 24h events into ±bucketPct price bands.
func ClusterHuntLiquidations(events []LiquidationEvent, mid, bucketPct float64) []HuntCluster {
	if mid <= 0 || bucketPct <= 0 {
		return nil
	}
	step := mid * bucketPct / 100
	type acc struct {
		n     float64
		c     int
		sumPx float64
		side  string
	}
	buckets := map[int]*acc{}
	for _, e := range events {
		if e.Price <= 0 || e.Notional <= 0 {
			continue
		}
		key := int(math.Round((e.Price - mid) / step))
		// encode side in the key sign space via offset
		sideKey := key
		if e.Side == LiquidationSideShort {
			sideKey = key + 1_000_000
		} else {
			sideKey = key - 1_000_000
		}
		a := buckets[sideKey]
		if a == nil {
			a = &acc{side: e.Side}
			if a.side == "" {
				a.side = LiquidationSideLong
			}
			buckets[sideKey] = a
		}
		a.n += e.Notional
		a.c++
		a.sumPx += e.Price
	}
	out := make([]HuntCluster, 0, len(buckets))
	for _, a := range buckets {
		px := a.sumPx / float64(a.c)
		out = append(out, HuntCluster{
			Side:     a.side,
			Price:    px,
			MovePct:  (px - mid) / mid * 100,
			Notional: a.n,
			Count:    a.c,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Notional == out[j].Notional {
			return math.Abs(out[i].MovePct) < math.Abs(out[j].MovePct)
		}
		return out[i].Notional > out[j].Notional
	})
	if len(out) > 12 {
		out = out[:12]
	}
	return out
}

// BuildHuntVenue runs the hypothetical hunt model for one exchange.
func BuildHuntVenue(in HuntInputs) HuntVenueReport {
	in.Symbol = NormalizeLiquidationSymbol(in.Symbol)
	out := HuntVenueReport{
		Exchange:           in.Exchange,
		Symbol:             in.Symbol,
		Price:              in.Price,
		OpenInterestValue:  in.OIValue,
		LongShare:          in.LongShare,
		ShortShare:         in.ShortShare,
		FundingRate:        in.FundingRate,
		FundingPayer:       FundingPayer(in.FundingRate),
		VisibleAskNotional: LevelsNotional(in.Asks),
		VisibleBidNotional: LevelsNotional(in.Bids),
		UpPressure:         []HuntBand{},
		DownPressure:       []HuntBand{},
		Observed:           ClusterHuntLiquidations(in.Liquidations, in.Price, HuntClusterBucketPct),
	}
	if in.LongShare == 0 && in.ShortShare == 0 {
		in.LongShare, in.ShortShare = 0.5, 0.5
		out.LongShare, out.ShortShare = 0.5, 0.5
	}
	estLong, estShort := BlendAccountShare(in.LongShare)
	out.EstLongShare, out.EstShortShare = estLong, estShort
	out.EstLongNotional = in.OIValue * estLong
	out.EstShortNotional = in.OIValue * estShort

	if in.Price <= 0 {
		out.Error = "no spot price"
		return out
	}

	longsCrowded := in.FundingRate > 1e-12
	shortsCrowded := in.FundingRate < -1e-12
	longMix := TiltLeverageMix(DefaultHuntLeverageMix, longsCrowded)
	shortMix := TiltLeverageMix(DefaultHuntLeverageMix, shortsCrowded)

	up := modelBands(LiquidationSideShort, "up", in.Price, out.EstShortNotional, shortMix, 1)
	down := modelBands(LiquidationSideLong, "down", in.Price, out.EstLongNotional, longMix, -1)
	mergeObserved(&up, out.Observed, LiquidationSideShort, "up", in.Price)
	mergeObserved(&down, out.Observed, LiquidationSideLong, "down", in.Price)
	sortBands(up)
	sortBands(down)
	out.UpPressure = clipBands(up)
	out.DownPressure = clipBands(down)

	out.UpHunt = buildHuntScenario("up",
		"Buy spot to lift price and force shorts to cover.",
		out.UpPressure, in.Price, in.Asks, in.Bids, ImpactSideBuy)
	out.DownHunt = buildHuntScenario("down",
		"Sell spot to push price down, liquidate longs, then buy back cheaper.",
		out.DownPressure, in.Price, in.Bids, in.Asks, ImpactSideSell)
	return out
}

func modelBands(side, dir string, mid, sideOI float64, mix []HuntLeverageBucket, sign float64) []HuntBand {
	out := make([]HuntBand, 0, len(mix))
	for _, b := range mix {
		if b.Leverage < 1 || b.Weight <= 0 {
			continue
		}
		dist := HuntLiqDistance(b.Leverage, HuntMaintenanceMargin)
		px := mid * (1 + sign*dist)
		if px <= 0 {
			continue
		}
		out = append(out, HuntBand{
			Side:        side,
			Direction:   dir,
			Leverage:    b.Leverage,
			Price:       px,
			MovePct:     sign * dist * 100,
			EstNotional: sideOI * b.Weight,
			Source:      "model",
		})
	}
	return out
}

func mergeObserved(bands *[]HuntBand, clusters []HuntCluster, side, dir string, mid float64) {
	if bands == nil || mid <= 0 {
		return
	}
	for _, c := range clusters {
		if c.Side != side || c.Price <= 0 {
			continue
		}
		if dir == "up" && c.Price <= mid {
			continue
		}
		if dir == "down" && c.Price >= mid {
			continue
		}
		merged := false
		for i := range *bands {
			b := &(*bands)[i]
			if mid > 0 && math.Abs(b.Price-c.Price)/mid <= 0.0025 {
				b.ObservedNotional += c.Notional
				b.Source = "both"
				merged = true
				break
			}
		}
		if merged {
			continue
		}
		*bands = append(*bands, HuntBand{
			Side:             side,
			Direction:        dir,
			Price:            c.Price,
			MovePct:          (c.Price - mid) / mid * 100,
			ObservedNotional: c.Notional,
			Source:           "observed",
		})
	}
}

func sortBands(in []HuntBand) {
	sort.SliceStable(in, func(i, j int) bool {
		ai := in[i].EstNotional + in[i].ObservedNotional
		aj := in[j].EstNotional + in[j].ObservedNotional
		if ai == aj {
			return math.Abs(in[i].MovePct) < math.Abs(in[j].MovePct)
		}
		return ai > aj
	})
}

func clipBands(in []HuntBand) []HuntBand {
	if len(in) > maxHuntBands {
		return in[:maxHuntBands]
	}
	return in
}

func buildHuntScenario(dir, thesis string, bands []HuntBand, mid float64, push, unwind []ImpactSourceLevel, side string) HuntScenario {
	out := HuntScenario{Direction: dir, Thesis: thesis}
	target, ok := pickHuntTarget(bands, mid, push, side)
	if !ok {
		out.HouseEdge = "unreachable"
		return out
	}
	out.Target = target
	reach := WalkBookToPrice(side, mid, push, target.Price)
	out.Spot = reach
	// If the modeled cluster sits past visible depth, score wiping the
	// book instead — that is the most an on-venue desk can do right now.
	if !reach.Reachable && reach.MaxReachablePrice > 0 {
		horizon := reach.MaxReachablePrice
		out.Target.Price = horizon
		out.Target.MovePct = (horizon - mid) / mid * 100
		out.Target.Leverage = 0
		out.Target.Source = "visible_book"
		out.Target.EstNotional = liquidatedUpTo(bands, dir, horizon)
		if out.Thesis != "" {
			out.Thesis += " Visible spot depth does not reach the first modeled cluster; figures are for consuming the visible book only."
		}
		reach.TargetPrice = horizon
		reach.Reachable = true
		reach.Exhausted = true
		out.Spot = reach
	}
	out.EstLiquidated = liquidatedUpTo(bands, dir, out.Target.Price)
	if reach.Notional > 0 {
		out.Efficiency = out.EstLiquidated / reach.Notional
	}
	out.LiquidationTake = out.EstLiquidated * HuntLiquidationTakeRate
	out.CascadeExitNotional = out.EstLiquidated * HuntCascadeFillRate

	// Book-only unwind: sell (up) or buy back (down) the same qty on the opposite side.
	// Size that does not fit is marked at mid so a thin opposite book cannot invent P&L.
	var unwindSpent float64
	if reach.Quantity > 0 {
		filled, spent, _, _, exh := WalkBookQuantity(unwind, reach.Quantity)
		unwindSpent = spent
		if exh && mid > 0 && reach.Quantity > filled {
			unwindSpent += (reach.Quantity - filled) * mid
		}
	}
	if side == ImpactSideBuy {
		out.BookOnlyPnl = unwindSpent - reach.Notional
	} else {
		out.BookOnlyPnl = reach.Notional - unwindSpent
	}
	out.Fees = (reach.Notional + unwindSpent) * HuntSpotTakerFee
	out.NetBookOnly = out.BookOnlyPnl + out.LiquidationTake - out.Fees

	// Cascade: part of inventory exits at the target instead of walking back.
	cascadeNotional := out.CascadeExitNotional
	if cascadeNotional > reach.Notional {
		cascadeNotional = reach.Notional
	}
	var cascadeQty float64
	exitPx := out.Target.Price
	if reach.AveragePrice > 0 && exitPx > 0 {
		cascadeQty = cascadeNotional / exitPx
		if cascadeQty > reach.Quantity {
			cascadeQty = reach.Quantity
		}
	}
	rest := reach.Quantity - cascadeQty
	var restSpent float64
	if rest > 1e-12 {
		filled, spent, _, _, exh := WalkBookQuantity(unwind, rest)
		restSpent = spent
		if exh && mid > 0 && rest > filled {
			restSpent += (rest - filled) * mid
		}
	}
	if side == ImpactSideBuy {
		// bought at avg; sell cascadeQty at target, rest into old bids
		out.CascadeInventoryPnl = cascadeQty*exitPx + restSpent - reach.Notional
	} else {
		// sold at avg; buy cascadeQty at target, rest into old asks
		out.CascadeInventoryPnl = reach.Notional - (cascadeQty*exitPx + restSpent)
	}
	cascadeFees := (reach.Notional + cascadeQty*exitPx + restSpent) * HuntSpotTakerFee
	out.NetWithCascade = out.CascadeInventoryPnl + out.LiquidationTake - cascadeFees
	if !reach.Reachable {
		out.HouseEdge = "unreachable"
		return out
	}
	if out.NetWithCascade > 0 {
		out.HouseEdge = "profit"
	} else {
		out.HouseEdge = "loss"
	}
	return out
}

func pickHuntTarget(bands []HuntBand, mid float64, push []ImpactSourceLevel, side string) (HuntBand, bool) {
	if len(bands) == 0 || mid <= 0 {
		return HuntBand{}, false
	}
	bestIdx := -1
	bestScore := math.Inf(-1)
	foundReachable := false
	closestIdx := -1
	closestMove := math.Inf(1)
	for i, b := range bands {
		if b.Price <= 0 {
			continue
		}
		move := math.Abs(b.MovePct)
		if move < closestMove {
			closestMove = move
			closestIdx = i
		}
		reach := WalkBookToPrice(side, mid, push, b.Price)
		if !reach.Reachable {
			continue
		}
		liq := liquidatedUpTo(bands, b.Direction, b.Price)
		score := liq - reach.Notional
		if !foundReachable || score > bestScore {
			foundReachable = true
			bestScore = score
			bestIdx = i
		}
	}
	if foundReachable {
		return bands[bestIdx], true
	}
	if closestIdx >= 0 {
		return bands[closestIdx], true
	}
	return HuntBand{}, false
}

func liquidatedUpTo(bands []HuntBand, dir string, target float64) float64 {
	var n float64
	for _, b := range bands {
		if dir == "up" {
			if b.Price <= target {
				n += b.EstNotional
			}
		} else if b.Price >= target {
			n += b.EstNotional
		}
	}
	return n
}
