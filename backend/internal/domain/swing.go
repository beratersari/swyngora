package domain

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Swing setup types (ported from crypto_analyzer, long-biased).
const (
	SwingSetupTrendPullback = "trend_pullback"
	SwingSetupBreakout      = "breakout"
	SwingSetupSqueeze       = "squeeze_release"
	SwingSetupMeanRevert    = "mean_reversion"
	SwingSetupUnknown       = "unknown"
)

// Signal stages.
const (
	SwingStageWatch    = "watch"
	SwingStageTrigger  = "trigger"
	SwingStageRejected = "rejected"
)

// BTC / market regimes.
const (
	SwingRegimeBull    = "bull"
	SwingRegimeBear    = "bear"
	SwingRegimeChop    = "chop"
	SwingRegimeUnknown = "unknown"
)

// Default swing parameters (literature-backed, stricter than the junior source).
const (
	SwingMinClosedBars       = 60
	SwingATRPeriod           = 14
	SwingADXPeriod           = 14
	SwingADXTrend            = 20.0
	SwingADXChop             = 18.0
	SwingADXStrong           = 25.0
	SwingMinRRTrigger        = 1.8
	SwingMinRRWatch          = 1.5
	SwingMaxRiskPct          = 8.0
	SwingMinQuoteVolumeUSDT  = 1_000_000.0
	SwingStoreMinScore       = 45.0
	SwingTriggerMinScore      = 62.0
	SwingMinFreshPatterns    = 1
	SwingMinPatternCount     = 2
	SwingStopATRMult         = 1.5
	SwingStopStructureBuffer = 0.25
	SwingTPATRMult           = 2.5
	SwingEMAFast             = 9
	SwingEMASlow             = 21
	SwingEMAMid              = 50
	SwingEMALong             = 200
	SwingSupertrendPeriod    = 10
	SwingSupertrendMult      = 3.0
	SwingMACDFast            = 12
	SwingMACDSlow            = 26
	SwingMACDSignal          = 9
	SwingBBPeriod            = 20
	SwingLookbackSwingLow    = 8
)

// SwingPattern is one detected factor with a 0–100 score.
type SwingPattern struct {
	Name        string  `json:"name"`
	Score       float64 `json:"score"`
	Description string  `json:"description"`
	Timeframe   string  `json:"timeframe"`
	Fresh       bool    `json:"fresh"`
}

// SwingLevels is an ATR/structure risk plan (informational).
type SwingLevels struct {
	Entry      float64 `json:"entry"`
	StopLoss   float64 `json:"stopLoss"`
	TakeProfit float64 `json:"takeProfit"`
	RiskPct    float64 `json:"riskPct"`
	RewardPct  float64 `json:"rewardPct"`
	RR         float64 `json:"rr"`
	ATR        float64 `json:"atr"`
}

// SwingDecision is the quality-gated result for one symbol.
type SwingDecision struct {
	Exchange   Exchange
	Symbol     string
	Interval   string
	Accepted   bool
	Stage      string
	SetupType  string
	SwingScore float64
	Grade      string
	Fresh      bool
	BTCRegime  string
	Side       string // long
	Price      float64
	ADX4h      *float64
	ADX1d      *float64
	RSI        *float64
	EMA4h      string // bullish|bearish|unknown
	EMA1d      string
	Patterns   []SwingPattern
	Levels     *SwingLevels
	Reasons    []string
	Note       string
	BarTime    time.Time
}

// SwingScanInput is market context for EvaluateSwing.
type SwingScanInput struct {
	Exchange     Exchange
	Symbol       string
	Primary      []OHLC // typically 4h, chronological, closed bars only
	Higher       []OHLC // typically 1d
	BTCPrimary   []OHLC // BTC 4h for regime (optional)
	BTCHigher    []OHLC // BTC 1d
	QuoteVolume  float64
	MarketCap    float64
	PrimaryTF    string
	HigherTF     string
	Now          time.Time
}

// OHLC is a parsed candle used by the swing engine.
type OHLC struct {
	OpenTime  time.Time
	CloseTime time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}

// ParseOHLC converts API candles to numeric bars. Invalid OHLC is an error (no gap-skip).
func ParseOHLC(candles []Candle) ([]OHLC, error) {
	out := make([]OHLC, 0, len(candles))
	for i, c := range candles {
		o, err := parseClose(c.Open)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid open at %d", ErrInvalidArgument, i)
		}
		h, err := parseClose(c.High)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid high at %d", ErrInvalidArgument, i)
		}
		l, err := parseClose(c.Low)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid low at %d", ErrInvalidArgument, i)
		}
		cl, err := parseClose(c.Close)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid close at %d", ErrInvalidArgument, i)
		}
		vol := 0.0
		if strings.TrimSpace(c.Volume) != "" {
			if v, e := parseClose(c.Volume); e == nil {
				vol = v
			}
		}
		if h < l || cl <= 0 || o <= 0 || math.IsNaN(h) || math.IsNaN(l) {
			return nil, fmt.Errorf("%w: inconsistent OHLC at %d", ErrInvalidArgument, i)
		}
		out = append(out, OHLC{
			OpenTime: c.OpenTime, CloseTime: c.CloseTime,
			Open: o, High: h, Low: l, Close: cl, Volume: vol,
		})
	}
	return out, nil
}

// ClosedBars drops the last bar when its close is still in the future (forming candle).
func ClosedBars(bars []OHLC, now time.Time) []OHLC {
	if len(bars) == 0 {
		return bars
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	last := bars[len(bars)-1]
	if !last.CloseTime.IsZero() && now.Before(last.CloseTime) {
		return bars[:len(bars)-1]
	}
	return bars
}

// EvaluateSwing runs pattern detection + quality gates on closed primary/higher TF bars.
// Informational only — not financial advice.
func EvaluateSwing(in SwingScanInput) (*SwingDecision, error) {
	primaryTF := in.PrimaryTF
	if primaryTF == "" {
		primaryTF = string(Interval4h)
	}
	higherTF := in.HigherTF
	if higherTF == "" {
		higherTF = string(Interval1d)
	}
	now := in.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	primary := ClosedBars(in.Primary, now)
	higher := ClosedBars(in.Higher, now)
	btcP := ClosedBars(in.BTCPrimary, now)
	btcH := ClosedBars(in.BTCHigher, now)

	d := &SwingDecision{
		Exchange:  in.Exchange,
		Symbol:    in.Symbol,
		Interval:  primaryTF,
		Stage:     SwingStageRejected,
		SetupType: SwingSetupUnknown,
		Side:      "long",
		BTCRegime: SwingRegimeUnknown,
		Note:      "Informational only — not financial advice.",
	}
	if len(primary) < SwingMinClosedBars {
		d.Reasons = []string{fmt.Sprintf("bars<%d", SwingMinClosedBars)}
		return d, nil
	}

	patterns, snap, err := detectSwingPatterns(primary, primaryTF)
	if err != nil {
		return nil, err
	}
	d.Price = snap.price
	d.ADX4h = snap.adx
	d.RSI = snap.rsi
	d.EMA4h = emaBias(primary)
	if len(higher) >= 50 {
		d.EMA1d = emaBias(higher)
		if adx, _, _, ok := ADX(higher, SwingADXPeriod); ok {
			v := adx
			d.ADX1d = &v
		}
	}
	d.Patterns = patterns
	d.BTCRegime = ClassifyBTCRegime(btcP, btcH, d.ADX4h)
	d.BarTime = primary[len(primary)-1].OpenTime

	levels, levReasons := planSwingLevels(primary, snap.atr, snap.price)
	if levels != nil {
		d.Levels = levels
	}
	d.Reasons = append(d.Reasons, levReasons...)

	filtered := filterFreshPatterns(patterns, true)
	d.Fresh = isFreshPatternSet(filtered)
	use := patterns
	if len(filtered) > 0 {
		use = filtered
	}
	d.SetupType = classifySetupType(use, d.ADX4h)
	d.SwingScore = computeSwingScore(use)
	d.Grade = swingGrade(d.SwingScore, d.Stage)

	if in.QuoteVolume > 0 && in.QuoteVolume < SwingMinQuoteVolumeUSDT {
		d.Reasons = append(d.Reasons, fmt.Sprintf("quote_volume<%.0f", SwingMinQuoteVolumeUSDT))
	}
	if !d.Fresh {
		d.Reasons = append(d.Reasons, "no_fresh_event")
	}
	if len(use) < SwingMinPatternCount {
		d.Reasons = append(d.Reasons, fmt.Sprintf("patterns<%d", SwingMinPatternCount))
	}
	if ok, why := regimeAllowsLong(d.BTCRegime, d.SetupType); !ok {
		d.Reasons = append(d.Reasons, why)
	}
	if d.SetupType != SwingSetupMeanRevert {
		if ok, why := multiTFAligned(d.EMA4h, d.EMA1d, d.ADX4h, d.ADX1d, d.SetupType); !ok {
			d.Reasons = append(d.Reasons, why)
		}
	} else if d.ADX4h != nil && *d.ADX4h >= 30 {
		d.Reasons = append(d.Reasons, "mean_reversion_blocked_by_strong_trend")
	}
	if levels != nil {
		if levels.RiskPct > SwingMaxRiskPct {
			d.Reasons = append(d.Reasons, fmt.Sprintf("risk_pct>%.1f", SwingMaxRiskPct))
		}
	}

	minRR := SwingMinRRWatch
	if d.SwingScore >= SwingTriggerMinScore && d.Fresh {
		minRR = SwingMinRRTrigger
	}
	if levels == nil {
		d.Reasons = append(d.Reasons, "missing_stop_or_rr")
	} else if levels.RR+1e-9 < minRR {
		d.Reasons = append(d.Reasons, fmt.Sprintf("rr<%.1f", minRR))
	}

	// Dedup reasons
	d.Reasons = uniqStrings(d.Reasons)

	hardBlock := hasAny(d.Reasons, "no_fresh_event", "btc_bear_blocks_long", "btc_chop_blocks_trend",
		"multi_tf_not_bullish", "both_tf_chop", "missing_stop_or_rr", "mean_reversion_blocked_by_strong_trend")
	if hardBlock || d.SwingScore < SwingStoreMinScore || len(use) < SwingMinPatternCount {
		d.Accepted = false
		d.Stage = SwingStageRejected
		d.Grade = swingGrade(d.SwingScore, d.Stage)
		return d, nil
	}

	d.Accepted = true
	if d.SwingScore >= SwingTriggerMinScore && d.Fresh && levels != nil && levels.RR+1e-9 >= SwingMinRRTrigger {
		d.Stage = SwingStageTrigger
	} else {
		d.Stage = SwingStageWatch
	}
	d.Grade = swingGrade(d.SwingScore, d.Stage)
	return d, nil
}

func swingGrade(score float64, stage string) string {
	if stage == SwingStageRejected {
		return "C"
	}
	if score >= 78 && stage == SwingStageTrigger {
		return "A"
	}
	if score >= 62 {
		return "B"
	}
	return "C"
}

type swingSnap struct {
	price float64
	atr   float64
	adx   *float64
	rsi   *float64
}

func detectSwingPatterns(bars []OHLC, tf string) ([]SwingPattern, swingSnap, error) {
	closes := make([]float64, len(bars))
	vols := make([]float64, len(bars))
	for i, b := range bars {
		closes[i] = b.Close
		vols[i] = b.Volume
	}
	price := closes[len(closes)-1]
	snap := swingSnap{price: price}

	atr, ok := ATR(bars, SwingATRPeriod)
	if ok {
		snap.atr = atr
	}

	var out []SwingPattern

	// 1. EMA 9/21 crossover (fresh event)
	e9 := lastEMA(closes, SwingEMAFast)
	e21 := lastEMA(closes, SwingEMASlow)
	e9p := emaAt(closes, SwingEMAFast, 2)
	e21p := emaAt(closes, SwingEMASlow, 2)
	if e9 != nil && e21 != nil && *e21 > 0 {
		score := 0.0
		fresh := false
		if *e9 > *e21 {
			score = 40
			if e9p != nil && e21p != nil && *e9p <= *e21p {
				score = 90
				fresh = true
			} else if emaAt(closes, SwingEMAFast, 3) != nil && emaAt(closes, SwingEMASlow, 3) != nil &&
				*emaAt(closes, SwingEMAFast, 3) <= *emaAt(closes, SwingEMASlow, 3) {
				score = 75
				fresh = true
			}
			gap := (*e9 - *e21) / *e21 * 100
			if gap > 0.2 && gap < 2.0 {
				score += 10
			}
		}
		if score > 0 {
			out = append(out, SwingPattern{
				Name: "ema_crossover", Score: clamp100(score), Fresh: fresh, Timeframe: tf,
				Description: fmt.Sprintf("EMA%d/EMA%d", SwingEMAFast, SwingEMASlow),
			})
		}
	}

	// 2. EMA 50/200 trend pullback
	e50 := lastEMA(closes, SwingEMAMid)
	e200 := lastEMA(closes, SwingEMALong)
	if e50 != nil {
		score := 0.0
		if e200 != nil && price > *e200 {
			dist := math.Abs(price-*e50) / *e50 * 100
			if dist < 2.0 {
				score = 90
			} else {
				score = 50
			}
		} else if price > *e50 {
			score = 30
		}
		if score > 0 {
			out = append(out, SwingPattern{
				Name: "ema_trend_bias", Score: clamp100(score), Fresh: false, Timeframe: tf,
				Description: "price vs EMA50/200",
			})
		}
	}

	// 3. SuperTrend
	st, dir, stOK := SuperTrend(bars, SwingSupertrendPeriod, SwingSupertrendMult)
	if stOK && dir == 1 {
		score := 70.0
		fresh := false
		if len(bars) > 2 {
			_, prevDir, prevOK := SuperTrend(bars[:len(bars)-1], SwingSupertrendPeriod, SwingSupertrendMult)
			if prevOK && prevDir == -1 {
				score = 95
				fresh = true
			}
		}
		if st > 0 && price > 0 {
			dist := (price - st) / price * 100
			if dist < 3 {
				score = math.Min(100, score+15)
			}
		}
		out = append(out, SwingPattern{
			Name: "supertrend", Score: clamp100(score), Fresh: fresh, Timeframe: tf,
			Description: "SuperTrend up",
		})
	}

	// 4. RSI recovery (Wilder)
	rsiSeries := RSIWilder(closes, DefaultRSIPeriod)
	rsi := lastPtr(rsiSeries)
	snap.rsi = rsi
	if rsi != nil {
		score := 0.0
		fresh := false
		switch {
		case *rsi >= 30 && *rsi <= 45:
			score = 80
		case *rsi < 30:
			score = 60
		case *rsi > 45 && *rsi < 55:
			score = 50
		case *rsi >= 55 && *rsi <= 65:
			score = 40
		}
		// bounce from oversold in last 5 closed bars
		oversold := false
		start := len(rsiSeries) - 6
		if start < 0 {
			start = 0
		}
		for i := start; i < len(rsiSeries)-1; i++ {
			if rsiSeries[i] != nil && *rsiSeries[i] < 35 {
				oversold = true
				break
			}
		}
		if oversold && *rsi > 35 {
			score = math.Max(score, 85)
			fresh = true
		}
		if score > 0 {
			out = append(out, SwingPattern{
				Name: "rsi_recovery", Score: clamp100(score), Fresh: fresh, Timeframe: tf,
				Description: fmt.Sprintf("RSI %.1f", *rsi),
			})
		}
	}

	// 5. MACD histogram
	_, _, hist := MACD(closes, SwingMACDFast, SwingMACDSlow, SwingMACDSignal)
	if len(hist) > 1 && hist[len(hist)-1] != nil {
		h := *hist[len(hist)-1]
		score := 0.0
		fresh := false
		if h > 0 {
			score = 60
			if hist[len(hist)-2] != nil && *hist[len(hist)-2] <= 0 {
				score = 90
				fresh = true
			}
		} else if h > -1e-9 {
			score = 40
		}
		if score > 0 {
			out = append(out, SwingPattern{
				Name: "macd_momentum", Score: clamp100(score), Fresh: fresh, Timeframe: tf,
				Description: "MACD histogram",
			})
		}
	}

	// 6. ADX
	adx, plusDI, minusDI, adxOK := ADX(bars, SwingADXPeriod)
	if adxOK {
		v := adx
		snap.adx = &v
		score := 0.0
		if adx >= SwingADXStrong && plusDI > minusDI {
			score = 80
			if adx >= 35 {
				score = 95
			}
		} else if adx >= SwingADXTrend && plusDI > minusDI {
			score = 50
		}
		if score > 0 {
			out = append(out, SwingPattern{
				Name: "adx_trend_strength", Score: clamp100(score), Fresh: false, Timeframe: tf,
				Description: fmt.Sprintf("ADX %.1f +DI>−DI", adx),
			})
		}
	}

	// 7. Accumulation (BB squeeze + volume dry-up)
	bbw := bollingerWidth(closes, SwingBBPeriod)
	vol20 := smaLast(vols, 20)
	vol5 := smaLast(vols, 5)
	accum := 0.0
	if bbw != nil {
		if *bbw < 4 {
			accum += 40
		} else if *bbw < 6 {
			accum += 25
		}
	}
	if vol20 != nil && *vol20 > 0 && vol5 != nil {
		ratio := *vol5 / *vol20
		if ratio < 0.6 {
			accum += 40
		} else if ratio < 0.8 {
			accum += 25
		}
	}
	if atr > 0 && price > 0 && (atr/price)*100 < 2 {
		accum += 20
	}
	if accum > 0 {
		out = append(out, SwingPattern{
			Name: "accumulation", Score: clamp100(accum), Fresh: false, Timeframe: tf,
			Description: "BB squeeze / volume dry-up",
		})
	}

	// 8. Volume breakout + range high
	if vol20 != nil && *vol20 > 0 && vols[len(vols)-1] > *vol20*2 {
		score := 70.0
		fresh := true
		if vols[len(vols)-1] > *vol20*3 {
			score = 90
		}
		if len(bars) >= 6 {
			hi := bars[len(bars)-6].High
			for _, b := range bars[len(bars)-5 : len(bars)-1] {
				if b.High > hi {
					hi = b.High
				}
			}
			if price > hi {
				score = math.Min(100, score+20)
			}
		}
		out = append(out, SwingPattern{
			Name: "volume_breakout", Score: clamp100(score), Fresh: fresh, Timeframe: tf,
			Description: "volume vs 20-bar avg",
		})
	}

	if snap.adx != nil && *snap.adx < SwingADXChop {
		// chop penalty applied later via gates; keep patterns
	}
	return out, snap, nil
}

func planSwingLevels(bars []OHLC, atr, entry float64) (*SwingLevels, []string) {
	if entry <= 0 {
		return nil, []string{"invalid_entry"}
	}
	n := SwingLookbackSwingLow
	if n >= len(bars) {
		n = len(bars) - 1
	}
	if n < 2 {
		return nil, []string{"missing_stop_or_rr"}
	}
	swingLow := bars[len(bars)-1-n].Low
	for _, b := range bars[len(bars)-n : len(bars)-1] { // exclude last closed bar wick as optional
		if b.Low < swingLow {
			swingLow = b.Low
		}
	}
	var stop float64
	if atr > 0 {
		structStop := swingLow - SwingStopStructureBuffer*atr
		atrStop := entry - SwingStopATRMult*atr
		// farther (lower) stop = more room under structure
		stop = math.Min(structStop, atrStop)
		// cap extremely wide stops so R:R remains possible
		if entry-stop > 3.5*atr {
			stop = entry - 2.5*atr
		}
	} else {
		stop = swingLow * 0.995
	}
	if stop <= 0 || stop >= entry {
		if atr > 0 {
			stop = entry - SwingStopATRMult*atr
		} else {
			stop = entry * 0.97
		}
	}
	if stop <= 0 || stop >= entry {
		return nil, []string{"missing_stop_or_rr"}
	}
	risk := entry - stop
	riskPct := risk / entry * 100
	rrTarget := SwingMinRRTrigger
	tpFromRR := entry + rrTarget*risk
	tp := tpFromRR
	if atr > 0 {
		atrTP := entry + SwingTPATRMult*atr
		if atrTP > tpFromRR {
			tp = atrTP
		}
	}
	rewardPct := (tp - entry) / entry * 100
	rr := 0.0
	if riskPct > 0 {
		rr = rewardPct / riskPct
	}
	return &SwingLevels{
		Entry: entry, StopLoss: round8(stop), TakeProfit: round8(tp),
		RiskPct: round8(riskPct), RewardPct: round8(rewardPct), RR: round8(rr), ATR: round8(atr),
	}, nil
}

var swingWeights = map[string]float64{
	"ema_crossover":      0.16,
	"ema_trend_bias":     0.12,
	"supertrend":         0.13,
	"rsi_recovery":       0.13,
	"macd_momentum":      0.12,
	"adx_trend_strength": 0.12,
	"accumulation":       0.10,
	"volume_breakout":    0.12,
}

var freshPatternNames = map[string]struct{}{
	"ema_crossover": {}, "supertrend": {}, "macd_momentum": {},
	"volume_breakout": {}, "rsi_recovery": {},
}

var continuationNames = map[string]struct{}{
	"ema_trend_bias": {}, "adx_trend_strength": {}, "accumulation": {},
}

var setupMembers = map[string]map[string]struct{}{
	SwingSetupTrendPullback: {"ema_trend_bias": {}, "supertrend": {}, "ema_crossover": {}, "adx_trend_strength": {}, "macd_momentum": {}},
	SwingSetupBreakout:      {"volume_breakout": {}, "macd_momentum": {}, "supertrend": {}},
	SwingSetupSqueeze:       {"accumulation": {}, "volume_breakout": {}, "macd_momentum": {}},
	SwingSetupMeanRevert:    {"rsi_recovery": {}, "accumulation": {}},
}

func computeSwingScore(patterns []SwingPattern) float64 {
	best := map[string]float64{}
	for _, p := range patterns {
		if p.Score <= 0 {
			continue
		}
		if v, ok := best[p.Name]; !ok || p.Score > v {
			best[p.Name] = p.Score
		}
	}
	if len(best) == 0 {
		return 0
	}
	var mass, acc float64
	for name, sc := range best {
		w := swingWeights[name]
		if w == 0 {
			w = 0.05
		}
		mass += w
		acc += sc * w
	}
	if mass <= 0 {
		return 0
	}
	score := acc / mass
	breadth := math.Min(1, float64(len(best))/5)
	score = score * (0.9 + 0.1*breadth)
	return math.Round(math.Min(100, math.Max(0, score))*10) / 10
}

func filterFreshPatterns(patterns []SwingPattern, freshOnly bool) []SwingPattern {
	if !freshOnly {
		return patterns
	}
	var fresh, ctx []SwingPattern
	for _, p := range patterns {
		if p.Fresh {
			fresh = append(fresh, p)
		} else if p.Score >= 70 {
			if _, ok := continuationNames[p.Name]; ok {
				ctx = append(ctx, p)
			}
		}
	}
	if len(fresh) == 0 {
		return nil
	}
	by := map[string]SwingPattern{}
	for _, p := range append(fresh, ctx...) {
		if prev, ok := by[p.Name]; !ok || p.Score > prev.Score {
			by[p.Name] = p
		}
	}
	out := make([]SwingPattern, 0, len(by))
	for _, p := range by {
		out = append(out, p)
	}
	return out
}

func isFreshPatternSet(patterns []SwingPattern) bool {
	for _, p := range patterns {
		if p.Fresh && p.Score >= 70 {
			if _, ok := freshPatternNames[p.Name]; ok {
				return true
			}
		}
	}
	return false
}

func classifySetupType(patterns []SwingPattern, adx *float64) string {
	if len(patterns) == 0 {
		return SwingSetupUnknown
	}
	scores := map[string]float64{
		SwingSetupTrendPullback: 0, SwingSetupBreakout: 0,
		SwingSetupSqueeze: 0, SwingSetupMeanRevert: 0,
	}
	for st, members := range setupMembers {
		for _, p := range patterns {
			if _, ok := members[p.Name]; ok {
				boost := p.Score
				if p.Fresh {
					boost *= 1.2
				}
				scores[st] += boost
			}
		}
	}
	if adx != nil {
		if *adx < SwingADXChop {
			scores[SwingSetupMeanRevert] *= 1.3
			scores[SwingSetupTrendPullback] *= 0.5
			scores[SwingSetupBreakout] *= 0.6
		} else if *adx >= SwingADXStrong {
			scores[SwingSetupTrendPullback] *= 1.2
			scores[SwingSetupBreakout] *= 1.1
			scores[SwingSetupMeanRevert] *= 0.4
		}
	}
	best, bestS := SwingSetupUnknown, 0.0
	for st, s := range scores {
		if s > bestS {
			best, bestS = st, s
		}
	}
	if bestS <= 0 {
		return SwingSetupUnknown
	}
	return best
}

// ClassifyBTCRegime uses EMA50/200 + ADX on BTC series (higher TF preferred).
func ClassifyBTCRegime(primary, higher []OHLC, adx4h *float64) string {
	series := higher
	if len(series) < 50 {
		series = primary
	}
	if len(series) < 50 {
		return SwingRegimeUnknown
	}
	if adx4h != nil && *adx4h < SwingADXChop {
		return SwingRegimeChop
	}
	closes := make([]float64, len(series))
	for i, b := range series {
		closes[i] = b.Close
	}
	price := closes[len(closes)-1]
	e50 := lastEMA(closes, SwingEMAMid)
	e200 := lastEMA(closes, SwingEMALong)
	if e50 == nil {
		return SwingRegimeUnknown
	}
	if e200 != nil {
		if price > *e200 && price >= *e50*0.98 {
			return SwingRegimeBull
		}
		if price < *e200 && price <= *e50*1.02 {
			return SwingRegimeBear
		}
		return SwingRegimeChop
	}
	if price >= *e50 {
		return SwingRegimeBull
	}
	return SwingRegimeBear
}

func regimeAllowsLong(regime, setup string) (bool, string) {
	switch regime {
	case SwingRegimeUnknown, SwingRegimeBull:
		return true, ""
	case SwingRegimeChop:
		if setup == SwingSetupMeanRevert || setup == SwingSetupSqueeze {
			return true, ""
		}
		return false, "btc_chop_blocks_trend"
	case SwingRegimeBear:
		if setup == SwingSetupMeanRevert {
			return true, ""
		}
		return false, "btc_bear_blocks_long"
	default:
		return true, ""
	}
}

func multiTFAligned(ema4h, ema1d string, adx4h, adx1d *float64, setup string) (bool, string) {
	if setup == SwingSetupMeanRevert {
		return true, ""
	}
	if ema4h != "bullish" || (ema1d != "" && ema1d != "bullish") {
		if ema1d != "" && ema1d != "unknown" {
			return false, "multi_tf_not_bullish"
		}
	}
	if adx4h != nil && adx1d != nil && *adx4h < SwingADXChop && *adx1d < SwingADXChop {
		return false, "both_tf_chop"
	}
	return true, ""
}

func emaBias(bars []OHLC) string {
	if len(bars) < SwingEMAMid {
		return "unknown"
	}
	closes := make([]float64, len(bars))
	for i, b := range bars {
		closes[i] = b.Close
	}
	e50 := lastEMA(closes, SwingEMAMid)
	if e50 == nil {
		return "unknown"
	}
	price := closes[len(closes)-1]
	if price >= *e50 {
		return "bullish"
	}
	return "bearish"
}

func lastEMA(closes []float64, period int) *float64 {
	s := EMA(closes, period)
	return lastPtr(s)
}

func emaAt(closes []float64, period, back int) *float64 {
	s := EMA(closes, period)
	i := len(s) - back
	if i < 0 || i >= len(s) {
		return nil
	}
	return s[i]
}

func lastPtr(s []*float64) *float64 {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] != nil {
			return s[i]
		}
	}
	return nil
}

func smaLast(vals []float64, period int) *float64 {
	if period <= 0 || len(vals) < period {
		return nil
	}
	var sum float64
	for _, v := range vals[len(vals)-period:] {
		sum += v
	}
	x := sum / float64(period)
	return &x
}

func bollingerWidth(closes []float64, period int) *float64 {
	if len(closes) < period || period < 2 {
		return nil
	}
	window := closes[len(closes)-period:]
	var mid float64
	for _, v := range window {
		mid += v
	}
	mid /= float64(period)
	if mid == 0 {
		return nil
	}
	var ss float64
	for _, v := range window {
		d := v - mid
		ss += d * d
	}
	std := math.Sqrt(ss / float64(period))
	w := (4 * std / mid) * 100
	return &w
}

func clamp100(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return math.Round(v*10) / 10
}

func round8(v float64) float64 {
	return math.Round(v*1e8) / 1e8
}

func uniqStrings(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func hasAny(list []string, keys ...string) bool {
	set := map[string]struct{}{}
	for _, k := range keys {
		set[k] = struct{}{}
	}
	for _, s := range list {
		if _, ok := set[s]; ok {
			return true
		}
	}
	return false
}
