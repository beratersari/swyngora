package domain

import (
	"fmt"
	"math"
	"strconv"
	"time"
)

// PumpDetectMode selects how return is measured.
type PumpDetectMode string

const (
	// PumpModeCloseReturn measures close[i-window] → close[i] percent change.
	PumpModeCloseReturn PumpDetectMode = "close_return"
	// PumpModeCandleBody measures open → close of the ending bar (single-bar body).
	PumpModeCandleBody PumpDetectMode = "candle_body"
	// PumpModeHighFromLow measures low → high of the ending bar (wick-inclusive spike).
	PumpModeHighFromLow PumpDetectMode = "high_from_low"
)

// PumpDirection filters event side.
type PumpDirection string

const (
	PumpDirectionUp   PumpDirection = "up"
	PumpDirectionDown PumpDirection = "down"
	PumpDirectionBoth PumpDirection = "both"
)

// PumpEvent is one detected rapid move on a candle series (informational).
type PumpEvent struct {
	Index       int       // bar index in the series (0-based)
	OpenTime    time.Time // open of the ending bar
	CloseTime   time.Time
	StartPrice  float64
	EndPrice    float64
	ReturnPct   float64 // signed % (positive = pump, negative = dump)
	High        float64
	Low         float64
	Volume      float64
	VolumeRatio float64 // bar volume / series median volume; 0 if unknown
	Mode        PumpDetectMode
	WindowBars  int
}

// PumpDetectOptions configures DetectPumpEvents.
type PumpDetectOptions struct {
	// MinReturnPct is the absolute threshold in percent (e.g. 5 = ±5%). Required > 0.
	MinReturnPct float64
	// WindowBars is the lookback in bars for close_return (default 1). Clamped 1–100.
	WindowBars int
	// Mode defaults to close_return.
	Mode PumpDetectMode
	// Direction defaults to up (pumps only).
	Direction PumpDirection
	// MinVolumeRatio requires volume/median ≥ this (0 = disabled).
	MinVolumeRatio float64
	// MaxEvents caps returned events after ranking by |return| (0 = no cap beyond full scan).
	MaxEvents int
}

// DetectPumpEvents finds bars where return exceeds the threshold.
// Candles must be chronological oldest-first. Pure function — no I/O.
func DetectPumpEvents(candles []Candle, opts PumpDetectOptions) ([]PumpEvent, error) {
	if opts.MinReturnPct <= 0 {
		return nil, fmt.Errorf("%w: minReturnPct must be > 0", ErrInvalidArgument)
	}
	if opts.WindowBars <= 0 {
		opts.WindowBars = 1
	}
	if opts.WindowBars > 100 {
		return nil, fmt.Errorf("%w: windowBars must be <= 100", ErrInvalidArgument)
	}
	if opts.Mode == "" {
		opts.Mode = PumpModeCloseReturn
	}
	switch opts.Mode {
	case PumpModeCloseReturn, PumpModeCandleBody, PumpModeHighFromLow:
	default:
		return nil, fmt.Errorf("%w: mode must be close_return|candle_body|high_from_low", ErrInvalidArgument)
	}
	if opts.Direction == "" {
		opts.Direction = PumpDirectionUp
	}
	switch opts.Direction {
	case PumpDirectionUp, PumpDirectionDown, PumpDirectionBoth:
	default:
		return nil, fmt.Errorf("%w: direction must be up|down|both", ErrInvalidArgument)
	}
	if opts.MinVolumeRatio < 0 {
		return nil, fmt.Errorf("%w: minVolumeRatio must be >= 0", ErrInvalidArgument)
	}
	if len(candles) < 2 && opts.Mode == PumpModeCloseReturn {
		return nil, nil
	}
	if len(candles) == 0 {
		return nil, nil
	}

	vols := make([]float64, len(candles))
	for i, c := range candles {
		v, _ := strconv.ParseFloat(c.Volume, 64)
		vols[i] = v
	}
	medianVol := medianFloat(vols)

	var events []PumpEvent
	startI := opts.WindowBars
	if opts.Mode != PumpModeCloseReturn {
		startI = 0
	}
	for i := startI; i < len(candles); i++ {
		c := candles[i]
		open, err1 := strconv.ParseFloat(c.Open, 64)
		high, err2 := strconv.ParseFloat(c.High, 64)
		low, err3 := strconv.ParseFloat(c.Low, 64)
		closeP, err4 := strconv.ParseFloat(c.Close, 64)
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			continue
		}
		if open <= 0 || closeP <= 0 || high <= 0 || low <= 0 {
			continue
		}

		var startPx, endPx, ret float64
		switch opts.Mode {
		case PumpModeCloseReturn:
			prevClose, err := strconv.ParseFloat(candles[i-opts.WindowBars].Close, 64)
			if err != nil || prevClose <= 0 {
				continue
			}
			startPx, endPx = prevClose, closeP
			ret = (endPx/startPx - 1) * 100
		case PumpModeCandleBody:
			startPx, endPx = open, closeP
			ret = (endPx/startPx - 1) * 100
		case PumpModeHighFromLow:
			startPx, endPx = low, high
			if startPx <= 0 {
				continue
			}
			ret = (endPx/startPx - 1) * 100
		}

		if !passesDirection(ret, opts.MinReturnPct, opts.Direction) {
			continue
		}

		volRatio := 0.0
		if medianVol > 0 {
			volRatio = vols[i] / medianVol
		}
		if opts.MinVolumeRatio > 0 && volRatio < opts.MinVolumeRatio {
			continue
		}

		events = append(events, PumpEvent{
			Index:       i,
			OpenTime:    c.OpenTime,
			CloseTime:   c.CloseTime,
			StartPrice:  startPx,
			EndPrice:    endPx,
			ReturnPct:   ret,
			High:        high,
			Low:         low,
			Volume:      vols[i],
			VolumeRatio: volRatio,
			Mode:        opts.Mode,
			WindowBars:  opts.WindowBars,
		})
	}

	// Rank by absolute return descending.
	sortPumpEventsByAbsReturn(events)
	if opts.MaxEvents > 0 && len(events) > opts.MaxEvents {
		events = events[:opts.MaxEvents]
	}
	return events, nil
}

func passesDirection(ret, minAbs float64, dir PumpDirection) bool {
	switch dir {
	case PumpDirectionUp:
		return ret >= minAbs
	case PumpDirectionDown:
		return ret <= -minAbs
	case PumpDirectionBoth:
		return math.Abs(ret) >= minAbs
	default:
		return false
	}
}

func sortPumpEventsByAbsReturn(events []PumpEvent) {
	// simple insertion for small N; events typically << 1000
	for i := 1; i < len(events); i++ {
		j := i
		for j > 0 && math.Abs(events[j-1].ReturnPct) < math.Abs(events[j].ReturnPct) {
			events[j-1], events[j] = events[j], events[j-1]
			j--
		}
	}
}

func medianFloat(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	cp := append([]float64(nil), xs...)
	// insertion sort
	for i := 1; i < len(cp); i++ {
		j := i
		for j > 0 && cp[j-1] > cp[j] {
			cp[j-1], cp[j] = cp[j], cp[j-1]
			j--
		}
	}
	n := len(cp)
	if n%2 == 1 {
		return cp[n/2]
	}
	return (cp[n/2-1] + cp[n/2]) / 2
}

// IntervalApproxDuration returns an approximate bar duration for lookback math.
func IntervalApproxDuration(iv CandleInterval) (time.Duration, error) {
	switch iv {
	case Interval1m:
		return time.Minute, nil
	case Interval3m:
		return 3 * time.Minute, nil
	case Interval5m:
		return 5 * time.Minute, nil
	case Interval15m:
		return 15 * time.Minute, nil
	case Interval30m:
		return 30 * time.Minute, nil
	case Interval1h:
		return time.Hour, nil
	case Interval2h:
		return 2 * time.Hour, nil
	case Interval4h:
		return 4 * time.Hour, nil
	case Interval6h:
		return 6 * time.Hour, nil
	case Interval8h:
		return 8 * time.Hour, nil
	case Interval12h:
		return 12 * time.Hour, nil
	case Interval1d:
		return 24 * time.Hour, nil
	case Interval3d:
		return 72 * time.Hour, nil
	case Interval1w:
		return 7 * 24 * time.Hour, nil
	case Interval1M:
		return 30 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("%w: unknown interval %q", ErrInvalidArgument, iv)
	}
}

// BarsForLookbackHours estimates how many bars cover lookbackHours for an interval.
func BarsForLookbackHours(iv CandleInterval, lookbackHours float64) (int, error) {
	if lookbackHours <= 0 {
		return 0, fmt.Errorf("%w: lookbackHours must be > 0", ErrInvalidArgument)
	}
	d, err := IntervalApproxDuration(iv)
	if err != nil {
		return 0, err
	}
	if d <= 0 {
		return 0, fmt.Errorf("%w: invalid interval duration", ErrInvalidArgument)
	}
	bars := int(math.Ceil(lookbackHours * float64(time.Hour) / float64(d)))
	if bars < 2 {
		bars = 2
	}
	if bars > 1000 {
		bars = 1000
	}
	return bars, nil
}
