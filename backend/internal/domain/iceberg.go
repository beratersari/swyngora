package domain

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	WallBehaviorIceberg = "iceberg"

	DefaultIcebergMinNotional = 25_000.0
	MinIcebergMinNotional     = 5_000.0
	icebergMinRefills         = 2
	icebergEatFrac            = 0.45
	icebergRefillFrac         = 0.70
	icebergRefillWindow       = 3 * time.Minute
	icebergTouchFrac          = 0.0015 // 0.15% from the touch
	icebergTTL                = 20 * time.Minute
	maxIcebergTracks          = 1500
)

// IcebergLevel is one price where size keeps coming back after being eaten.
type IcebergLevel struct {
	Exchange         Exchange
	Symbol           string
	Side             string // bid | ask
	Price            float64
	ClipQuantity     float64
	ClipNotional     float64
	VisibleQuantity  float64
	VisibleNotional  float64
	Refills          int
	ExecutedNotional float64
	PrintHits        int
	Confidence       string // likely | possible
	FirstSeen        time.Time
	LastRefill       time.Time
	Summary          string
}

// IcebergReport is the API result for one coin.
type IcebergReport struct {
	Symbol   string
	Exchange string
	AsOf     time.Time
	Asks     []IcebergLevel
	Bids     []IcebergLevel
	Summary  string
	Note     string
}

// IcebergMemory watches quantity at a price across book snapshots.
type IcebergMemory struct {
	mu     sync.Mutex
	tracks []*icebergTrack
}

type icebergTrack struct {
	exchange, symbol, side string
	price                  float64
	qty, notional          float64
	clipQty, clipNotional  float64
	refills                int
	eatenNotional          float64
	printHits              int
	pendingEat             bool
	eatAt                  time.Time
	firstSeen              time.Time
	lastSeen               time.Time
	lastRefill             time.Time
	atTouch                bool
}

// NewIcebergMemory constructs an empty tracker.
func NewIcebergMemory() *IcebergMemory {
	return &IcebergMemory{}
}

// ParseIcebergMinNotional clamps the clip floor.
func ParseIcebergMinNotional(v float64) float64 {
	if v <= 0 {
		return DefaultIcebergMinNotional
	}
	if v < MinIcebergMinNotional {
		return MinIcebergMinNotional
	}
	return v
}

// ObserveBook records visible size at each significant price and looks for refill cycles.
func (m *IcebergMemory) ObserveBook(now time.Time, exchange, symbol string, raw RawOrderBook, prints []TakerPrint, minNotional float64) {
	if m == nil {
		return
	}
	now = now.UTC()
	exchange = strings.ToLower(strings.TrimSpace(exchange))
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	minNotional = ParseIcebergMinNotional(minNotional)
	samples := icebergSamples(raw, minNotional)
	bestBid, bestAsk := icebergTouch(raw)
	m.mu.Lock()
	defer m.mu.Unlock()
	seen := make(map[*icebergTrack]struct{}, len(samples))
	for _, sm := range samples {
		tr := m.matchLocked(exchange, symbol, sm.side, sm.price)
		if tr == nil {
			if sm.notional < minNotional {
				continue
			}
			tr = &icebergTrack{
				exchange: exchange, symbol: symbol, side: sm.side, price: sm.price,
				qty: sm.qty, notional: sm.notional,
				clipQty: sm.qty, clipNotional: sm.notional,
				firstSeen: now, lastSeen: now,
				atTouch: icebergNearTouch(sm.side, sm.price, bestBid, bestAsk),
			}
			m.tracks = append(m.tracks, tr)
			seen[tr] = struct{}{}
			continue
		}
		m.stepLocked(tr, now, sm, bestBid, bestAsk, prints)
		seen[tr] = struct{}{}
	}
	for _, tr := range m.tracks {
		if tr.exchange != exchange || tr.symbol != symbol {
			continue
		}
		if _, ok := seen[tr]; ok {
			continue
		}
		// Level gone: treat as a full eat if it was at the touch or prints hit.
		zero := icebergSample{side: tr.side, price: tr.price}
		m.stepLocked(tr, now, zero, bestBid, bestAsk, prints)
	}
	m.expireLocked(now)
}

// AnnotateWalls copies iceberg flags onto detected walls.
func (m *IcebergMemory) AnnotateWalls(exchange, symbol string, walls []OrderBookWall) {
	if m == nil {
		return
	}
	exchange = strings.ToLower(strings.TrimSpace(exchange))
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range walls {
		px, err := parseHistFloatErr(walls[i].Price)
		if err != nil || px <= 0 {
			continue
		}
		tr := m.matchLocked(exchange, symbol, walls[i].Side, px)
		if tr == nil || tr.refills < icebergMinRefills {
			continue
		}
		walls[i].Iceberg = true
		walls[i].IcebergRefills = tr.refills
		walls[i].IcebergClip = formatQty(tr.clipNotional)
		walls[i].Behavior = WallBehaviorIceberg
	}
}

// AnnotateCombinedWalls copies iceberg flags onto combined walls.
func (m *IcebergMemory) AnnotateCombinedWalls(walls []CombinedWall) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range walls {
		px, err := parseHistFloatErr(walls[i].Price)
		if err != nil || px <= 0 {
			continue
		}
		tr := m.matchAnySymbolLocked(strings.ToLower(walls[i].Exchange), walls[i].Side, px)
		if tr == nil || tr.refills < icebergMinRefills {
			continue
		}
		walls[i].Iceberg = true
		walls[i].IcebergRefills = tr.refills
		walls[i].IcebergClip = formatQty(tr.clipNotional)
		walls[i].Behavior = WallBehaviorIceberg
	}
}

// Active returns confirmed icebergs for a venue+symbol (empty exchange = all).
func (m *IcebergMemory) Active(exchange, symbol string, minNotional float64) []IcebergLevel {
	if m == nil {
		return nil
	}
	exchange = strings.ToLower(strings.TrimSpace(exchange))
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	minNotional = ParseIcebergMinNotional(minNotional)
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]IcebergLevel, 0, 8)
	for _, tr := range m.tracks {
		if tr.refills < icebergMinRefills || tr.clipNotional < minNotional {
			continue
		}
		if symbol != "" && tr.symbol != symbol {
			continue
		}
		if exchange != "" && exchange != "all" && tr.exchange != exchange {
			continue
		}
		out = append(out, icebergFromTrack(tr))
	}
	return out
}

func (m *IcebergMemory) matchLocked(exchange, symbol, side string, price float64) *icebergTrack {
	side = strings.ToLower(side)
	for _, tr := range m.tracks {
		if tr.exchange == exchange && tr.symbol == symbol && tr.side == side && sameIcebergPrice(tr.price, price) {
			return tr
		}
	}
	return nil
}

func (m *IcebergMemory) matchAnySymbolLocked(exchange, side string, price float64) *icebergTrack {
	side = strings.ToLower(side)
	var best *icebergTrack
	for _, tr := range m.tracks {
		if tr.exchange != exchange || tr.side != side || !sameIcebergPrice(tr.price, price) {
			continue
		}
		if best == nil || tr.refills > best.refills || (tr.refills == best.refills && tr.lastSeen.After(best.lastSeen)) {
			best = tr
		}
	}
	return best
}

func (m *IcebergMemory) stepLocked(tr *icebergTrack, now time.Time, sm icebergSample, bestBid, bestAsk float64, prints []TakerPrint) {
	prevQty, prevNotional := tr.qty, tr.notional
	if tr.clipQty <= 0 {
		tr.clipQty = prevQty
		tr.clipNotional = prevNotional
	}
	dropped := prevQty > 0 && (sm.qty <= prevQty*(1-icebergEatFrac) || (prevNotional-sm.notional) >= tr.clipNotional*icebergEatFrac)
	if dropped && !tr.pendingEat {
		nowTouch := icebergNearTouch(tr.side, tr.price, bestBid, bestAsk)
		hit := tr.atTouch || nowTouch || icebergPrintHit(tr.side, tr.price, prints)
		if hit {
			tr.pendingEat = true
			tr.eatAt = now
			if prevNotional > sm.notional {
				tr.eatenNotional += prevNotional - sm.notional
			}
			if icebergPrintHit(tr.side, tr.price, prints) {
				tr.printHits++
			}
		}
	} else if tr.pendingEat && now.Sub(tr.eatAt) <= icebergRefillWindow &&
		sm.qty >= tr.clipQty*icebergRefillFrac && sm.notional >= tr.clipNotional*icebergRefillFrac {
		tr.refills++
		tr.lastRefill = now
		tr.pendingEat = false
		// Visible clip is the repeating slice, not the residual after a partial eat.
		if sm.qty > 0 {
			tr.clipQty = sm.qty
			tr.clipNotional = sm.notional
		}
	} else if tr.pendingEat && now.Sub(tr.eatAt) > icebergRefillWindow {
		tr.pendingEat = false
	}
	if sm.qty > 0 {
		tr.qty = sm.qty
		tr.notional = sm.notional
		tr.price = sm.price
		tr.atTouch = icebergNearTouch(tr.side, tr.price, bestBid, bestAsk)
	} else {
		tr.qty = 0
		tr.notional = 0
		tr.atTouch = false
	}
	tr.lastSeen = now
}

func (m *IcebergMemory) expireLocked(now time.Time) {
	keep := m.tracks[:0]
	for _, tr := range m.tracks {
		ref := tr.lastSeen
		if ref.IsZero() {
			ref = tr.firstSeen
		}
		if now.Sub(ref) <= icebergTTL {
			keep = append(keep, tr)
		}
	}
	m.tracks = keep
	if len(m.tracks) <= maxIcebergTracks {
		return
	}
	// Keep most recently seen.
	sorted := append([]*icebergTrack(nil), m.tracks...)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].lastSeen.After(sorted[i].lastSeen) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	m.tracks = sorted[:maxIcebergTracks]
}

type icebergSample struct {
	side     string
	price    float64
	qty      float64
	notional float64
}

func icebergSamples(raw RawOrderBook, minNotional float64) []icebergSample {
	bestBid, bestAsk := icebergTouch(raw)
	out := make([]icebergSample, 0, 16)
	add := func(side string, lv PriceLevel, touch float64) {
		if !validLevel(lv) {
			return
		}
		n := lv.Price * lv.Quantity
		if n < minNotional && !sameIcebergPrice(lv.Price, touch) {
			return
		}
		out = append(out, icebergSample{side: side, price: lv.Price, qty: lv.Quantity, notional: n})
	}
	for _, lv := range raw.Bids {
		add("bid", lv, bestBid)
	}
	for _, lv := range raw.Asks {
		add("ask", lv, bestAsk)
	}
	return out
}

func icebergTouch(raw RawOrderBook) (bestBid, bestAsk float64) {
	if len(raw.Bids) > 0 {
		bestBid = raw.Bids[0].Price
	}
	if len(raw.Asks) > 0 {
		bestAsk = raw.Asks[0].Price
	}
	return bestBid, bestAsk
}

const icebergPriceMatchFrac = 0.0001 // 0.01% — same displayed price, tighter than walls

func sameIcebergPrice(a, b float64) bool {
	if a <= 0 || b <= 0 {
		return false
	}
	ref := a
	if b > ref {
		ref = b
	}
	return absFloat(a-b) <= ref*icebergPriceMatchFrac
}

func icebergNearTouch(side string, price, bestBid, bestAsk float64) bool {
	if price <= 0 {
		return false
	}
	switch side {
	case "bid":
		if bestBid <= 0 {
			return false
		}
		return absFloat(price-bestBid) <= bestBid*icebergTouchFrac
	case "ask":
		if bestAsk <= 0 {
			return false
		}
		return absFloat(price-bestAsk) <= bestAsk*icebergTouchFrac
	default:
		return false
	}
}

func icebergPrintHit(side string, price float64, prints []TakerPrint) bool {
	want := TakerSideBuy
	if side == "bid" {
		want = TakerSideSell
	}
	for _, p := range prints {
		if p.Side != want || p.Price <= 0 {
			continue
		}
		if sameIcebergPrice(p.Price, price) {
			return true
		}
	}
	return false
}

func icebergFromTrack(tr *icebergTrack) IcebergLevel {
	conf := "possible"
	if tr.printHits > 0 || tr.eatenNotional >= tr.clipNotional*2 {
		conf = "likely"
	}
	out := IcebergLevel{
		Exchange: Exchange(tr.exchange), Symbol: tr.symbol, Side: tr.side,
		Price: tr.price, ClipQuantity: tr.clipQty, ClipNotional: tr.clipNotional,
		VisibleQuantity: tr.qty, VisibleNotional: tr.notional,
		Refills: tr.refills, ExecutedNotional: tr.eatenNotional, PrintHits: tr.printHits,
		Confidence: conf, FirstSeen: tr.firstSeen, LastRefill: tr.lastRefill,
	}
	out.Summary = ExplainIceberg(out)
	return out
}

// ExplainIceberg writes a short refill read.
func ExplainIceberg(e IcebergLevel) string {
	verb := "buying"
	if e.Side == "ask" {
		verb = "selling"
	}
	when := ""
	if !e.LastRefill.IsZero() {
		when = " last refill " + e.LastRefill.UTC().Format("15:04:05")
	}
	return fmt.Sprintf("%s iceberg %s at %s: visible clip %s came back %d time(s)%s (executed ~%s).",
		e.Side, verb, formatQty(e.Price), formatQty(e.ClipNotional), e.Refills, when, formatQty(e.ExecutedNotional))
}

// ExplainIcebergs writes a biggest-clip read.
func ExplainIcebergs(levels []IcebergLevel, symbol string) string {
	if len(levels) == 0 {
		if symbol != "" {
			return prettyBase(symbol) + ": no refill icebergs at the same price in the recent book."
		}
		return "No refill icebergs at the same price in the recent book."
	}
	top := levels[0]
	for _, e := range levels[1:] {
		if e.ClipNotional > top.ClipNotional {
			top = e
		}
	}
	return fmt.Sprintf("%d iceberg(s). Largest: %s.", len(levels), ExplainIceberg(top))
}

func parseHistFloatErr(s string) (float64, error) {
	v := parseHistFloat(s)
	if v <= 0 {
		return 0, fmt.Errorf("not a price")
	}
	return v, nil
}

const icebergDisclaimer = "An iceberg here means visible size at one price was eaten (or hit at the touch) and then a similar clip came back at least twice. It is a book-pattern read, not proof of a hidden exchange order. Informational only — not financial advice."
