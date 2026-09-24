package domain

import (
	"fmt"
	"strings"
)

const portfolioStressNote = "What-if on the current paper book. Spot marks and margin P&L move with each coin. An isolated margin position that crosses its liquidation price loses the margin locked on it. Cash does not change. Nothing is traded. Informational only."

// StressMove is a percent change for one base asset (BTC -10).
type StressMove struct {
	Asset   string
	MovePct float64
}

// StressLine is one spot or margin row after the shock.
type StressLine struct {
	Kind              string
	Symbol            string
	Side              string
	Asset             string
	MovePct           float64
	Mark              float64
	ShockedMark       float64
	Value             float64
	ShockedValue      float64
	Unrealized        float64
	ShockedUnrealized float64
	Liquidated        bool
}

// StressReport compares the book now with the same book after the moves.
type StressReport struct {
	Equity                float64
	ShockedEquity         float64
	EquityDelta           float64
	TotalPnL              float64
	ShockedTotalPnL       float64
	UnrealizedPnL         float64
	ShockedUnrealizedPnL  float64
	PositionsValue        float64
	ShockedPositionsValue float64
	MarginEquity          float64
	ShockedMarginEquity   float64
	Liquidated            int
	Lines                 []StressLine
	Note                  string
}

// ParseStressMoves checks a short list of asset shocks.
func ParseStressMoves(in []StressMove) ([]StressMove, error) {
	if len(in) == 0 {
		return nil, fmt.Errorf("%w: at least one move is required", ErrInvalidArgument)
	}
	if len(in) > 12 {
		return nil, fmt.Errorf("%w: at most 12 moves", ErrInvalidArgument)
	}
	out := make([]StressMove, 0, len(in))
	seen := map[string]struct{}{}
	for _, m := range in {
		asset := stressAsset(m.Asset)
		if asset == "" {
			return nil, fmt.Errorf("%w: asset is required", ErrInvalidArgument)
		}
		if m.MovePct <= -100 || m.MovePct > 500 || m.MovePct != m.MovePct {
			return nil, fmt.Errorf("%w: movePct must be greater than -100 and at most 500", ErrInvalidArgument)
		}
		if _, ok := seen[asset]; ok {
			return nil, fmt.Errorf("%w: duplicate asset %s", ErrInvalidArgument, asset)
		}
		seen[asset] = struct{}{}
		out = append(out, StressMove{Asset: asset, MovePct: m.MovePct})
	}
	return out, nil
}

// StressPortfolio reprices the current view. Coins without a move stay put.
func StressPortfolio(v PortfolioView, moves []StressMove) (StressReport, error) {
	parsed, err := ParseStressMoves(moves)
	if err != nil {
		return StressReport{}, err
	}
	by := map[string]float64{}
	for _, m := range parsed {
		by[m.Asset] = m.MovePct
	}
	rep := StressReport{
		Equity: v.Equity, TotalPnL: v.TotalPnL, UnrealizedPnL: v.UnrealizedPnL,
		PositionsValue: v.PositionsValue, MarginEquity: v.MarginEquity,
		Note: portfolioStressNote,
	}
	var spotValue, spotU, marginU float64
	for _, p := range v.Positions {
		asset := stressAsset(p.Symbol)
		pct, ok := by[asset]
		mark := p.MarkPrice
		if mark <= 0 {
			mark = p.AvgCost
		}
		next := mark
		if ok && mark > 0 {
			next = mark * (1 + pct/100)
		}
		value := p.Quantity * next
		u := p.UnrealizedPnL + (next-mark)*p.Quantity
		if !ok {
			value = p.MarketValue
			u = p.UnrealizedPnL
		}
		spotValue += value
		spotU += u
		if ok {
			rep.Lines = append(rep.Lines, StressLine{
				Kind: "spot", Symbol: p.Symbol, Asset: asset, MovePct: pct,
				Mark: mark, ShockedMark: next, Value: p.MarketValue, ShockedValue: value,
				Unrealized: p.UnrealizedPnL, ShockedUnrealized: u,
			})
		}
	}
	for _, p := range v.MarginPositions {
		if p.Status != "" && p.Status != MarginPositionOpen {
			continue
		}
		asset := stressAsset(p.Symbol)
		pct, ok := by[asset]
		mark := p.MarkPrice
		next := mark
		u := p.UnrealizedPnL
		liq := false
		if ok && mark > 0 {
			next = mark * (1 + pct/100)
			u = p.UnrealizedPnL + MarginUnrealizedPnL(p.Side, p.Quantity, mark, next)
			if ShouldLiquidate(p.Side, next, p.LiquidationPrice) {
				liq = true
				u = -p.Margin
				rep.Liquidated++
			}
		}
		marginU += u
		if ok {
			rep.Lines = append(rep.Lines, StressLine{
				Kind: "margin", Symbol: p.Symbol, Side: string(p.Side), Asset: asset, MovePct: pct,
				Mark: mark, ShockedMark: next, Value: p.Margin + p.UnrealizedPnL, ShockedValue: p.Margin + u,
				Unrealized: p.UnrealizedPnL, ShockedUnrealized: u, Liquidated: liq,
			})
		}
	}
	rep.ShockedPositionsValue = spotValue
	rep.ShockedUnrealizedPnL = spotU + marginU
	rep.ShockedMarginEquity = (v.MarginEquity - v.MarginUnrealizedPnL) + marginU
	// Equity = cash + spot value + margin locked + margin unrealized.
	// Margin locked is MarginEquity - MarginUnrealized, already inside ShockedMarginEquity.
	rep.ShockedEquity = v.Equity - v.PositionsValue - v.MarginEquity + rep.ShockedPositionsValue + rep.ShockedMarginEquity
	rep.EquityDelta = rep.ShockedEquity - v.Equity
	rep.ShockedTotalPnL = v.TotalPnL + rep.EquityDelta
	return rep, nil
}

func stressAsset(raw string) string {
	s := strings.ToUpper(strings.TrimSpace(raw))
	s = strings.ReplaceAll(s, "/", "")
	if i := strings.Index(s, "-"); i > 0 {
		return s[:i]
	}
	return LiquidationBaseAsset(s)
}
