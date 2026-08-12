package domain

import "math"

// BookReach is the size needed to walk live depth until last price reaches a target.
type BookReach struct {
	Side              string  `json:"side"`
	TargetPrice       float64 `json:"targetPrice"`
	MidPrice          float64 `json:"midPrice"`
	BestPrice         float64 `json:"bestPrice"`
	Quantity          float64 `json:"quantity"`
	Notional          float64 `json:"notional"`
	AveragePrice      float64 `json:"averagePrice"`
	EndPrice          float64 `json:"endPrice"`
	Reachable         bool    `json:"reachable"`
	Exhausted         bool    `json:"exhausted"`
	MaxReachablePrice float64 `json:"maxReachablePrice"`
	VisibleNotional   float64 `json:"visibleNotional"`
	LevelsUsed        int     `json:"levelsUsed"`
}

// WalkBookToPrice consumes asks (buy) or bids (sell) until a fill is at or
// beyond target. The whole level that first crosses the target is taken so the
// cost includes punching that wall, not a one-lot print.
func WalkBookToPrice(side string, mid float64, levels []ImpactSourceLevel, target float64) BookReach {
	if side == "" {
		side = ImpactSideBuy
	}
	out := BookReach{Side: side, TargetPrice: target, MidPrice: mid}
	var vis float64
	for _, lv := range levels {
		vis += lv.Price * lv.Quantity
	}
	out.VisibleNotional = vis
	if len(levels) == 0 || target <= 0 || math.IsNaN(target) || math.IsInf(target, 0) {
		out.Exhausted = true
		return out
	}
	out.BestPrice = levels[0].Price
	buy := side != ImpactSideSell
	var qty, spent, last float64
	used := 0
	for _, lv := range levels {
		if lv.Price <= 0 || lv.Quantity <= 0 {
			continue
		}
		qty += lv.Quantity
		spent += lv.Price * lv.Quantity
		last = lv.Price
		used++
		if buy && lv.Price >= target {
			out.Reachable = true
			break
		}
		if !buy && lv.Price <= target {
			out.Reachable = true
			break
		}
	}
	out.Quantity = qty
	out.Notional = spent
	out.EndPrice = last
	out.MaxReachablePrice = last
	out.LevelsUsed = used
	if qty > 0 {
		out.AveragePrice = spent / qty
	}
	if !out.Reachable {
		out.Exhausted = true
	}
	return out
}

// WalkBookQuantity takes a fixed base size from the same ordered levels.
func WalkBookQuantity(levels []ImpactSourceLevel, quantity float64) (filled, spent, avg, end float64, exhausted bool) {
	if quantity <= 0 {
		return 0, 0, 0, 0, false
	}
	remain := quantity
	for _, lv := range levels {
		if lv.Price <= 0 || lv.Quantity <= 0 || remain <= 0 {
			continue
		}
		take := lv.Quantity
		if remain < take {
			take = remain
		}
		filled += take
		spent += take * lv.Price
		end = lv.Price
		remain -= take
	}
	if filled > 0 {
		avg = spent / filled
	}
	return filled, spent, avg, end, remain > 1e-12
}

// LevelsNotional is resting quote value of the side.
func LevelsNotional(levels []ImpactSourceLevel) float64 {
	var n float64
	for _, lv := range levels {
		if lv.Price > 0 && lv.Quantity > 0 {
			n += lv.Price * lv.Quantity
		}
	}
	return n
}
