package domain

import "math"

// ATR computes Wilder's Average True Range on the last bar. ok=false if not enough history.
func ATR(bars []OHLC, period int) (float64, bool) {
	series, ok := ATRSeries(bars, period)
	if !ok {
		return 0, false
	}
	return series[len(series)-1], true
}

// ATRSeries returns Wilder ATR aligned to bars (index 0 unused/zero until warm-up).
func ATRSeries(bars []OHLC, period int) ([]float64, bool) {
	if period < 2 || len(bars) < period+1 {
		return nil, false
	}
	tr := make([]float64, len(bars))
	for i := 1; i < len(bars); i++ {
		h, l, pc := bars[i].High, bars[i].Low, bars[i-1].Close
		tr[i] = math.Max(h-l, math.Max(math.Abs(h-pc), math.Abs(l-pc)))
	}
	out := make([]float64, len(bars))
	var sum float64
	for i := 1; i <= period; i++ {
		sum += tr[i]
	}
	out[period] = sum / float64(period)
	for i := period + 1; i < len(bars); i++ {
		out[i] = (out[i-1]*float64(period-1) + tr[i]) / float64(period)
	}
	return out, true
}

// ADX returns Wilder ADX, +DI, −DI on the last bar.
func ADX(bars []OHLC, period int) (adx, plusDI, minusDI float64, ok bool) {
	if period < 2 || len(bars) < period*2+1 {
		return 0, 0, 0, false
	}
	n := len(bars)
	plusDM := make([]float64, n)
	minusDM := make([]float64, n)
	tr := make([]float64, n)
	for i := 1; i < n; i++ {
		up := bars[i].High - bars[i-1].High
		down := bars[i-1].Low - bars[i].Low
		if up > down && up > 0 {
			plusDM[i] = up
		}
		if down > up && down > 0 {
			minusDM[i] = down
		}
		h, l, pc := bars[i].High, bars[i].Low, bars[i-1].Close
		tr[i] = math.Max(h-l, math.Max(math.Abs(h-pc), math.Abs(l-pc)))
	}
	var smTR, smPlus, smMinus float64
	for i := 1; i <= period; i++ {
		smTR += tr[i]
		smPlus += plusDM[i]
		smMinus += minusDM[i]
	}
	dxs := make([]float64, 0, n)
	pdi, mdi := diPair(smPlus, smMinus, smTR)
	if pdi+mdi > 0 {
		dxs = append(dxs, 100*math.Abs(pdi-mdi)/(pdi+mdi))
	} else {
		dxs = append(dxs, 0)
	}
	for i := period + 1; i < n; i++ {
		smTR = smTR - smTR/float64(period) + tr[i]
		smPlus = smPlus - smPlus/float64(period) + plusDM[i]
		smMinus = smMinus - smMinus/float64(period) + minusDM[i]
		pdi, mdi = diPair(smPlus, smMinus, smTR)
		if pdi+mdi > 0 {
			dxs = append(dxs, 100*math.Abs(pdi-mdi)/(pdi+mdi))
		} else {
			dxs = append(dxs, 0)
		}
	}
	if len(dxs) < period {
		return 0, 0, 0, false
	}
	var adxSum float64
	for i := 0; i < period; i++ {
		adxSum += dxs[i]
	}
	adxVal := adxSum / float64(period)
	for i := period; i < len(dxs); i++ {
		adxVal = (adxVal*float64(period-1) + dxs[i]) / float64(period)
	}
	return adxVal, pdi, mdi, true
}

func diPair(plus, minus, tr float64) (plusDI, minusDI float64) {
	if tr <= 0 {
		return 0, 0
	}
	return 100 * plus / tr, 100 * minus / tr
}

// SuperTrend returns the last SuperTrend value and direction (+1 up, −1 down).
func SuperTrend(bars []OHLC, period int, multiplier float64) (value float64, dir int, ok bool) {
	atr, okATR := ATRSeries(bars, period)
	if !okATR || multiplier <= 0 {
		return 0, 0, false
	}
	n := len(bars)
	upper := make([]float64, n)
	lower := make([]float64, n)
	st := make([]float64, n)
	d := make([]int, n)
	start := period
	for i := start; i < n; i++ {
		mid := (bars[i].High + bars[i].Low) / 2
		bu := mid + multiplier*atr[i]
		bl := mid - multiplier*atr[i]
		if i == start {
			upper[i], lower[i] = bu, bl
			if bars[i].Close <= bu {
				st[i], d[i] = bu, -1
			} else {
				st[i], d[i] = bl, 1
			}
			continue
		}
		if bu < upper[i-1] || bars[i-1].Close > upper[i-1] {
			upper[i] = bu
		} else {
			upper[i] = upper[i-1]
		}
		if bl > lower[i-1] || bars[i-1].Close < lower[i-1] {
			lower[i] = bl
		} else {
			lower[i] = lower[i-1]
		}
		if d[i-1] == 1 {
			if bars[i].Close < lower[i] {
				d[i], st[i] = -1, upper[i]
			} else {
				d[i], st[i] = 1, lower[i]
			}
		} else {
			if bars[i].Close > upper[i] {
				d[i], st[i] = 1, lower[i]
			} else {
				d[i], st[i] = -1, upper[i]
			}
		}
	}
	return st[n-1], d[n-1], true
}

// MACD returns MACD line, signal, histogram series (nil until warm-up).
func MACD(closes []float64, fast, slow, signal int) (macd, sig, hist []*float64) {
	n := len(closes)
	macd = make([]*float64, n)
	sig = make([]*float64, n)
	hist = make([]*float64, n)
	if n < slow+signal {
		return macd, sig, hist
	}
	ef := EMA(closes, fast)
	es := EMA(closes, slow)
	line := make([]float64, n)
	validFrom := slow - 1
	for i := validFrom; i < n; i++ {
		if ef[i] == nil || es[i] == nil {
			continue
		}
		line[i] = *ef[i] - *es[i]
		macd[i] = ptr(line[i])
	}
	// Signal: EMA of MACD line from first valid index
	raw := make([]float64, 0, n-validFrom)
	idx := make([]int, 0, n-validFrom)
	for i := validFrom; i < n; i++ {
		if macd[i] != nil {
			raw = append(raw, *macd[i])
			idx = append(idx, i)
		}
	}
	sigSeries := EMA(raw, signal)
	for j, i := range idx {
		if j < len(sigSeries) && sigSeries[j] != nil {
			sig[i] = sigSeries[j]
			h := *macd[i] - *sigSeries[j]
			hist[i] = ptr(h)
		}
	}
	return macd, sig, hist
}
