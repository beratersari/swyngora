package domain

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	CorrWindow1h  = LiquidationWindow1h
	CorrWindow4h  = LiquidationWindow4h
	CorrWindow24h = LiquidationWindow24h

	CorrRelationFollows = "follows" // corr >= 0.70
	CorrRelationLoose   = "loose"   // 0.40–0.70
	CorrRelationMixed   = "mixed"   // |corr| < 0.40
	CorrRelationInverse = "inverse" // corr <= -0.40

	CorrTimingTogether = "together"
	CorrTimingLags     = "lags"  // coin follows the reference with a delay
	CorrTimingLeads    = "leads" // coin moves first

	corrFollowsMin  = 0.70
	corrLooseMin    = 0.40
	corrInverseMax  = -0.40
	corrMinSamples  = 15
	corrLagImprove  = 0.03 // lagged corr must beat contemporaneous by this
	corrZeroEpsilon = 1e-12
)

// PricePoint is a close aligned to a bar open time.
type PricePoint struct {
	Time  time.Time
	Close float64
}

// CorrelationVs is one coin versus one reference (BTC or ETH) in one window.
type CorrelationVs struct {
	Reference  string  // e.g. BTCUSDT
	Corr       float64 // Pearson of bar returns, -1..1
	Beta       float64 // asset return per 1 unit reference return
	SameDirPct float64 // % of bars that moved the same way
	Relation   string  // follows | loose | mixed | inverse
	Timing     string  // together | lags | leads
	LagBars    int     // signed: negative = coin lags, positive = coin leads
	LagMinutes int     // absolute delay of the best lag
	Samples    int
	Complete   bool
	Self       bool // coin is the reference itself
	Error      string
}

// CorrelationWindow is 1h / 4h / 24h vs BTC and ETH.
type CorrelationWindow struct {
	Window       string
	Interval     string
	Bars         int
	AssetMovePct float64
	BTCMovePct   float64
	ETHMovePct   float64
	BTC          CorrelationVs
	ETH          CorrelationVs
	Summary      string
}

// CorrelationReport is the API result for one coin.
type CorrelationReport struct {
	Symbol   string
	Exchange string
	AsOf     time.Time
	Windows  []CorrelationWindow
	Summary  string
	Note     string
}

// CorrWindowSpec is how we sample one lookback.
type CorrWindowSpec struct {
	ID       string
	Interval CandleInterval
	Bars     int // candles kept (returns = Bars-1)
	MaxLag   int // bars searched for lead/lag
}

// CorrelationWindows is 1m for 1h, 5m for 4h and 24h.
var CorrelationWindows = []CorrWindowSpec{
	{CorrWindow1h, Interval1m, 61, 6},   // ±6 minutes
	{CorrWindow4h, Interval5m, 49, 6},   // ±30 minutes
	{CorrWindow24h, Interval5m, 289, 6}, // ±30 minutes
}

// CorrelationRefs returns BTC and ETH pairs in the same quote as symbol.
// Crypto-quoted pairs (ETHBTC) fall back to USDT (or USD on Coinbase).
func CorrelationRefs(ex Exchange, symbol string) (btc, eth string) {
	symbol = NormalizeSymbol(ex, symbol)
	_, quote := SplitBaseQuote(ex, symbol)
	if quote == "" || quote == "BTC" || quote == "ETH" {
		if ex == ExchangeCoinbase {
			quote = "USD"
		} else {
			quote = "USDT"
		}
	}
	if ex == ExchangeCoinbase && quote == "USDT" {
		quote = "USD"
	}
	return PairSymbol(ex, "BTC", quote), PairSymbol(ex, "ETH", quote)
}

// PricePointsFromCandles extracts valid closes, oldest first.
func PricePointsFromCandles(candles []Candle) []PricePoint {
	out := make([]PricePoint, 0, len(candles))
	for _, c := range candles {
		px, err := parseClose(c.Close)
		if err != nil || px <= 0 || math.IsNaN(px) || math.IsInf(px, 0) {
			continue
		}
		t := c.OpenTime.UTC()
		if t.IsZero() {
			continue
		}
		out = append(out, PricePoint{Time: t, Close: px})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	return out
}

// TrimPricePoints keeps the newest n points.
func TrimPricePoints(pts []PricePoint, n int) []PricePoint {
	if n <= 0 || len(pts) <= n {
		return pts
	}
	return pts[len(pts)-n:]
}

// AlignPricePoints pairs closes that share an open time.
func AlignPricePoints(a, b []PricePoint) (ac, bc []float64, times []time.Time) {
	idx := make(map[int64]float64, len(b))
	for _, p := range b {
		if p.Time.IsZero() || p.Close <= 0 {
			continue
		}
		idx[p.Time.UnixMilli()] = p.Close
	}
	for _, p := range a {
		if p.Time.IsZero() || p.Close <= 0 {
			continue
		}
		q, ok := idx[p.Time.UnixMilli()]
		if !ok {
			continue
		}
		ac = append(ac, p.Close)
		bc = append(bc, q)
		times = append(times, p.Time)
	}
	return ac, bc, times
}

// PercentReturns is close-to-close percent change (length n-1).
func PercentReturns(closes []float64) []float64 {
	if len(closes) < 2 {
		return nil
	}
	out := make([]float64, 0, len(closes)-1)
	for i := 1; i < len(closes); i++ {
		if closes[i-1] <= 0 || math.IsNaN(closes[i]) || math.IsNaN(closes[i-1]) {
			return nil
		}
		out = append(out, (closes[i]/closes[i-1]-1)*100)
	}
	return out
}

// MovePct is last/first − 1 as a percent.
func MovePct(closes []float64) float64 {
	if len(closes) < 2 || closes[0] <= 0 {
		return 0
	}
	return (closes[len(closes)-1]/closes[0] - 1) * 100
}

// PearsonCorr is the sample Pearson correlation of x and y (same length).
func PearsonCorr(x, y []float64) float64 {
	n := len(x)
	if n < 2 || n != len(y) {
		return math.NaN()
	}
	var sx, sy float64
	for i := 0; i < n; i++ {
		if math.IsNaN(x[i]) || math.IsNaN(y[i]) {
			return math.NaN()
		}
		sx += x[i]
		sy += y[i]
	}
	mx, my := sx/float64(n), sy/float64(n)
	var num, dx, dy float64
	for i := 0; i < n; i++ {
		ax, ay := x[i]-mx, y[i]-my
		num += ax * ay
		dx += ax * ax
		dy += ay * ay
	}
	if dx < corrZeroEpsilon || dy < corrZeroEpsilon {
		if dx < corrZeroEpsilon && dy < corrZeroEpsilon {
			return 1
		}
		return 0
	}
	return num / math.Sqrt(dx*dy)
}

// RegressionBeta is cov(asset, ref) / var(ref).
func RegressionBeta(asset, ref []float64) float64 {
	n := len(asset)
	if n < 2 || n != len(ref) {
		return math.NaN()
	}
	var sa, sr float64
	for i := 0; i < n; i++ {
		sa += asset[i]
		sr += ref[i]
	}
	ma, mr := sa/float64(n), sr/float64(n)
	var cov, vr float64
	for i := 0; i < n; i++ {
		da, dr := asset[i]-ma, ref[i]-mr
		cov += da * dr
		vr += dr * dr
	}
	if vr < corrZeroEpsilon {
		return 0
	}
	return cov / vr
}

// SameDirectionPct is the share of bars where both returns have the same sign.
func SameDirectionPct(asset, ref []float64) float64 {
	n := 0
	same := 0
	for i := 0; i < len(asset) && i < len(ref); i++ {
		a, r := asset[i], ref[i]
		if math.Abs(a) < corrZeroEpsilon && math.Abs(r) < corrZeroEpsilon {
			continue
		}
		n++
		if a*r > 0 || (math.Abs(a) < corrZeroEpsilon && math.Abs(r) < corrZeroEpsilon) {
			same++
		}
	}
	if n == 0 {
		return 0
	}
	return float64(same) / float64(n) * 100
}

// BestReturnLag finds lag in [-maxLag, +maxLag] that maximises corr(asset[t], ref[t+lag]).
// Positive lag: coin leads the reference. Negative: coin lags (follows later).
func BestReturnLag(asset, ref []float64, maxLag int) (lag int, corr float64) {
	if maxLag < 0 {
		maxLag = 0
	}
	bestLag := 0
	best := PearsonCorr(asset, ref)
	if math.IsNaN(best) {
		best = -2
	}
	for l := -maxLag; l <= maxLag; l++ {
		if l == 0 {
			continue
		}
		a, r := shiftReturns(asset, ref, l)
		c := PearsonCorr(a, r)
		if math.IsNaN(c) {
			continue
		}
		if c > best+1e-12 || (math.Abs(c-best) <= 1e-12 && absInt(l) < absInt(bestLag)) {
			best = c
			bestLag = l
		}
	}
	if best < -1 {
		return 0, math.NaN()
	}
	return bestLag, best
}

func shiftReturns(asset, ref []float64, lag int) (a, r []float64) {
	// corr(asset[t], ref[t+lag]) on the overlap.
	n := len(asset)
	if n != len(ref) || n == 0 {
		return nil, nil
	}
	if lag >= 0 {
		if lag >= n {
			return nil, nil
		}
		return asset[:n-lag], ref[lag:]
	}
	k := -lag
	if k >= n {
		return nil, nil
	}
	return asset[k:], ref[:n-k]
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// ClassifyCorrRelation maps Pearson r onto follows / loose / mixed / inverse.
func ClassifyCorrRelation(corr float64) string {
	if math.IsNaN(corr) {
		return CorrRelationMixed
	}
	switch {
	case corr >= corrFollowsMin:
		return CorrRelationFollows
	case corr >= corrLooseMin:
		return CorrRelationLoose
	case corr <= corrInverseMax:
		return CorrRelationInverse
	default:
		return CorrRelationMixed
	}
}

// ClassifyCorrTiming says whether a non-zero lag is meaningful.
func ClassifyCorrTiming(lag int, contemporaneous, lagged float64) string {
	if lag == 0 || math.IsNaN(lagged) || math.IsNaN(contemporaneous) {
		return CorrTimingTogether
	}
	if lagged < contemporaneous+corrLagImprove {
		return CorrTimingTogether
	}
	if lag < 0 {
		return CorrTimingLags
	}
	return CorrTimingLeads
}

// IntervalMinutes is the bar size in minutes (0 if unknown).
func IntervalMinutes(iv CandleInterval) int {
	switch iv {
	case Interval1m:
		return 1
	case Interval3m:
		return 3
	case Interval5m:
		return 5
	case Interval15m:
		return 15
	case Interval30m:
		return 30
	case Interval1h:
		return 60
	default:
		return 0
	}
}

// CompareToReference scores one coin against BTC or ETH closes.
func CompareToReference(assetSym, refSym string, asset, ref []PricePoint, spec CorrWindowSpec) CorrelationVs {
	out := CorrelationVs{Reference: refSym}
	if assetSym != "" && refSym != "" && NormalizeSymbol(ExchangeBinance, assetSym) == NormalizeSymbol(ExchangeBinance, refSym) {
		out.Self = true
		out.Corr = 1
		out.Beta = 1
		out.SameDirPct = 100
		out.Relation = CorrRelationFollows
		out.Timing = CorrTimingTogether
		out.Complete = len(asset) >= 2
		if n := len(asset); n > 1 {
			out.Samples = n - 1
		}
		return out
	}
	ac, rc, _ := AlignPricePoints(asset, ref)
	ac = trimFloats(ac, spec.Bars)
	rc = trimFloats(rc, spec.Bars)
	ar := PercentReturns(ac)
	rr := PercentReturns(rc)
	if len(ar) < corrMinSamples || len(ar) != len(rr) {
		out.Error = "not enough overlapping bars"
		out.Relation = CorrRelationMixed
		out.Timing = CorrTimingTogether
		out.Samples = len(ar)
		return out
	}
	out.Samples = len(ar)
	out.Complete = true
	out.Corr = PearsonCorr(ar, rr)
	out.Beta = RegressionBeta(ar, rr)
	out.SameDirPct = SameDirectionPct(ar, rr)
	out.Relation = ClassifyCorrRelation(out.Corr)
	lag, lagged := BestReturnLag(ar, rr, spec.MaxLag)
	out.Timing = ClassifyCorrTiming(lag, out.Corr, lagged)
	if out.Timing != CorrTimingTogether {
		out.LagBars = lag
		mins := IntervalMinutes(spec.Interval)
		if mins > 0 {
			out.LagMinutes = absInt(lag) * mins
		}
	}
	return out
}

func trimFloats(v []float64, n int) []float64 {
	if n <= 0 || len(v) <= n {
		return v
	}
	return v[len(v)-n:]
}

// BuildCorrelationWindow compares the coin to BTC and ETH for one lookback.
func BuildCorrelationWindow(symbol string, spec CorrWindowSpec, asset, btc, eth []PricePoint, btcSym, ethSym string) CorrelationWindow {
	asset = TrimPricePoints(asset, spec.Bars)
	btc = TrimPricePoints(btc, spec.Bars)
	eth = TrimPricePoints(eth, spec.Bars)
	w := CorrelationWindow{
		Window:   spec.ID,
		Interval: string(spec.Interval),
		Bars:     len(asset),
		BTC:      CompareToReference(symbol, btcSym, asset, btc, spec),
		ETH:      CompareToReference(symbol, ethSym, asset, eth, spec),
	}
	if closes := closesOnly(asset); len(closes) >= 2 {
		w.AssetMovePct = MovePct(closes)
	}
	if closes := closesOnly(btc); len(closes) >= 2 {
		w.BTCMovePct = MovePct(closes)
	}
	if closes := closesOnly(eth); len(closes) >= 2 {
		w.ETHMovePct = MovePct(closes)
	}
	w.Summary = ExplainCorrWindow(symbol, w)
	return w
}

func closesOnly(pts []PricePoint) []float64 {
	out := make([]float64, 0, len(pts))
	for _, p := range pts {
		out = append(out, p.Close)
	}
	return out
}

// ExplainCorrWindow is a short read for one lookback.
func ExplainCorrWindow(symbol string, w CorrelationWindow) string {
	name := prettyBase(symbol)
	bits := make([]string, 0, 2)
	if s := explainVs(name, "BTC", w.BTC, w.Window); s != "" {
		bits = append(bits, s)
	}
	if s := explainVs(name, "ETH", w.ETH, w.Window); s != "" {
		bits = append(bits, s)
	}
	if len(bits) == 0 {
		return fmt.Sprintf("%s: not enough %s data to compare with BTC or ETH.", name, w.Window)
	}
	head := fmt.Sprintf("Over %s, %s is %s%% while BTC is %s%% and ETH is %s%%.",
		w.Window, name, FormatSignedPct(w.AssetMovePct), FormatSignedPct(w.BTCMovePct), FormatSignedPct(w.ETHMovePct))
	return head + " " + joinList(bits) + "."
}

func explainVs(name, ref string, v CorrelationVs, window string) string {
	_ = window
	if v.Self {
		return fmt.Sprintf("this is %s itself", ref)
	}
	if !v.Complete {
		return ""
	}
	rel := relationPhrase(v.Relation)
	s := fmt.Sprintf("%s %s (corr %s, beta %s, same direction %s%% of bars)",
		rel, ref, FormatSignedPct(v.Corr), FormatSignedPct(v.Beta), formatFixed(v.SameDirPct, 0))
	if v.Timing == CorrTimingLags && v.LagMinutes > 0 {
		s += fmt.Sprintf(", following about %d minutes later", v.LagMinutes)
	} else if v.Timing == CorrTimingLeads && v.LagMinutes > 0 {
		s += fmt.Sprintf(", leading by about %d minutes", v.LagMinutes)
	}
	return name + " " + s
}

func relationPhrase(rel string) string {
	switch rel {
	case CorrRelationFollows:
		return "follows"
	case CorrRelationLoose:
		return "loosely follows"
	case CorrRelationInverse:
		return "moves opposite"
	default:
		return "does not move with"
	}
}

func joinList(parts []string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	case 2:
		return parts[0] + " and " + parts[1]
	default:
		return strings.Join(parts[:len(parts)-1], ", ") + ", and " + parts[len(parts)-1]
	}
}

func prettyBase(symbol string) string {
	base, _ := SplitBaseQuote(ExchangeBinance, symbol)
	if base == "" {
		return symbol
	}
	return base
}

// ExplainCorrelationReport rolls 1h / 4h / 24h into one paragraph.
func ExplainCorrelationReport(symbol string, windows []CorrelationWindow) string {
	name := prettyBase(symbol)
	btc1 := pickCorrWindow(windows, CorrWindow1h)
	btc4 := pickCorrWindow(windows, CorrWindow4h)
	btc24 := pickCorrWindow(windows, CorrWindow24h)
	if btc1 == nil && btc4 == nil && btc24 == nil {
		return name + ": not enough price history to compare with BTC or ETH."
	}
	parts := make([]string, 0, 4)
	if btc1 != nil {
		parts = append(parts, fmt.Sprintf("1h vs BTC %s (corr %s)", relationPhrase(btc1.BTC.Relation), FormatSignedPct(btc1.BTC.Corr)))
		if btc1.ETH.Complete && !btc1.ETH.Self {
			parts = append(parts, fmt.Sprintf("vs ETH %s (corr %s)", relationPhrase(btc1.ETH.Relation), FormatSignedPct(btc1.ETH.Corr)))
		}
	}
	if btc4 != nil && btc4.BTC.Complete && !btc4.BTC.Self {
		parts = append(parts, fmt.Sprintf("4h vs BTC %s (corr %s)", relationPhrase(btc4.BTC.Relation), FormatSignedPct(btc4.BTC.Corr)))
	}
	if btc24 != nil && btc24.BTC.Complete && !btc24.BTC.Self {
		parts = append(parts, fmt.Sprintf("24h vs BTC %s (corr %s)", relationPhrase(btc24.BTC.Relation), FormatSignedPct(btc24.BTC.Corr)))
	}
	delay := firstDelay(windows)
	head := name + " " + joinList(parts) + "."
	if delay != "" {
		return head + " " + delay
	}
	return head
}

func pickCorrWindow(ws []CorrelationWindow, id string) *CorrelationWindow {
	for i := range ws {
		if ws[i].Window == id && (ws[i].BTC.Complete || ws[i].ETH.Complete) {
			return &ws[i]
		}
	}
	return nil
}

func firstDelay(ws []CorrelationWindow) string {
	for _, id := range []string{CorrWindow1h, CorrWindow4h, CorrWindow24h} {
		w := pickCorrWindow(ws, id)
		if w == nil {
			continue
		}
		if w.BTC.Timing == CorrTimingLags && w.BTC.LagMinutes > 0 {
			return fmt.Sprintf("It has been following BTC with about a %d-minute delay on the %s.", w.BTC.LagMinutes, w.Window)
		}
		if w.BTC.Timing == CorrTimingLeads && w.BTC.LagMinutes > 0 {
			return fmt.Sprintf("It has been leading BTC by about %d minutes on the %s.", w.BTC.LagMinutes, w.Window)
		}
	}
	return ""
}
