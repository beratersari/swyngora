package domain

import (
	"strconv"
	"strings"
	"sync"
	"time"
)

// Wall behavior labels on analysis.walls.
const (
	WallBehaviorShort      = "short"
	WallBehaviorPersistent = "persistent"
	WallBehaviorSuspicious = "suspicious"
)

const (
	// WallPersistentMin is how long a wall must rest near the same price
	// before it is labeled persistent. The book sampler must stay attached
	// longer than this plus one sample tick, or the last look lands short.
	WallPersistentMin = 2 * time.Minute

	wallSuspiciousWindow  = 2 * time.Minute
	wallSuspiciousAppears = 4
	wallAbsenceGrace      = 8 * time.Second
	wallPriceMatchFrac    = 0.0008 // 0.08% — same “zone”
	wallTrackTTL          = 20 * time.Minute
	maxWallTracks         = 2000
	wallPersistentDuty    = 0.7
)

// WallMemory watches detected walls across snapshots so we can tell a resting
// support/resistance print from a short-lived or flickering order.
type WallMemory struct {
	mu     sync.Mutex
	tracks []*wallTrack
}

type wallTrack struct {
	exchange, symbol, side string
	price                  float64
	firstSeen              time.Time
	lastSeen               time.Time
	lastAbsent             time.Time
	streakStart            time.Time
	present                bool
	appearCount            int
	visible                time.Duration
	appears                []time.Time
}

// NewWallMemory constructs an empty tracker.
func NewWallMemory() *WallMemory {
	return &WallMemory{}
}

// Observe records the current walls for one venue book and annotates them.
func (m *WallMemory) Observe(now time.Time, exchange, symbol string, walls []OrderBookWall) {
	if m == nil {
		return
	}
	now = now.UTC()
	exchange = strings.ToLower(strings.TrimSpace(exchange))
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	m.mu.Lock()
	defer m.mu.Unlock()
	seen := make(map[*wallTrack]struct{}, len(walls))
	for i := range walls {
		px, err := strconv.ParseFloat(walls[i].Price, 64)
		if err != nil || px <= 0 {
			continue
		}
		tr := m.matchLocked(exchange, symbol, walls[i].Side, px)
		if tr == nil {
			tr = &wallTrack{
				exchange:    exchange,
				symbol:      symbol,
				side:        strings.ToLower(walls[i].Side),
				price:       px,
				firstSeen:   now,
				lastSeen:    now,
				streakStart: now,
				appearCount: 1,
				appears:     []time.Time{now},
				present:     true,
			}
			m.tracks = append(m.tracks, tr)
		} else {
			m.touchPresentLocked(tr, now, px)
		}
		seen[tr] = struct{}{}
		annotateWall(&walls[i], tr, now)
	}
	for _, tr := range m.tracks {
		if tr.exchange != exchange || tr.symbol != symbol {
			continue
		}
		if _, ok := seen[tr]; ok {
			continue
		}
		if tr.present {
			if !tr.lastSeen.IsZero() && now.After(tr.lastSeen) {
				tr.visible += now.Sub(tr.lastSeen)
			}
			tr.present = false
			tr.lastAbsent = now
		}
	}
	m.expireLocked(now)
}

// ApplyCombined copies tracker life onto combined walls (after per-venue Observe).
func (m *WallMemory) ApplyCombined(now time.Time, walls []CombinedWall) {
	if m == nil {
		return
	}
	now = now.UTC()
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range walls {
		px, err := strconv.ParseFloat(walls[i].Price, 64)
		if err != nil || px <= 0 {
			continue
		}
		ex := strings.ToLower(strings.TrimSpace(walls[i].Exchange))
		tr := m.matchAnySymbolLocked(ex, walls[i].Side, px)
		if tr == nil {
			walls[i].Behavior = WallBehaviorShort
			continue
		}
		annotateCombined(&walls[i], tr, now)
	}
}

func (m *WallMemory) matchLocked(exchange, symbol, side string, price float64) *wallTrack {
	side = strings.ToLower(side)
	for _, tr := range m.tracks {
		if tr.exchange == exchange && tr.symbol == symbol && tr.side == side && sameWallPrice(tr.price, price) {
			return tr
		}
	}
	return nil
}

func (m *WallMemory) matchAnySymbolLocked(exchange, side string, price float64) *wallTrack {
	side = strings.ToLower(side)
	var best *wallTrack
	for _, tr := range m.tracks {
		if tr.exchange != exchange || tr.side != side || !sameWallPrice(tr.price, price) {
			continue
		}
		if best == nil || tr.lastSeen.After(best.lastSeen) {
			best = tr
		}
	}
	return best
}

func (m *WallMemory) touchPresentLocked(tr *wallTrack, now time.Time, price float64) {
	if tr.present {
		if !tr.lastSeen.IsZero() && now.After(tr.lastSeen) {
			tr.visible += now.Sub(tr.lastSeen)
		}
	} else {
		gap := now.Sub(tr.lastAbsent)
		if tr.lastAbsent.IsZero() || gap >= wallAbsenceGrace {
			tr.appearCount++
			tr.appears = append(tr.appears, now)
			tr.streakStart = now
		}
		tr.present = true
	}
	tr.price = price
	tr.lastSeen = now
}

func (m *WallMemory) expireLocked(now time.Time) {
	keep := m.tracks[:0]
	for _, tr := range m.tracks {
		ref := tr.lastSeen
		if ref.IsZero() {
			ref = tr.firstSeen
		}
		if now.Sub(ref) <= wallTrackTTL {
			keep = append(keep, tr)
		}
	}
	m.tracks = keep
	if len(m.tracks) <= maxWallTracks {
		return
	}
	// Keep the most recently seen maxWallTracks.
	sorted := append([]*wallTrack(nil), m.tracks...)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].lastSeen.After(sorted[i].lastSeen) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	m.tracks = sorted[:maxWallTracks]
}

func annotateWall(w *OrderBookWall, tr *wallTrack, now time.Time) {
	life := wallLife(tr, now)
	w.Behavior = life.behavior
	w.AgeSeconds = life.age
	w.PresentForSeconds = life.streak
	w.VisibleSeconds = life.visible
	w.AppearCount = life.appears
}

func annotateCombined(w *CombinedWall, tr *wallTrack, now time.Time) {
	life := wallLife(tr, now)
	w.Behavior = life.behavior
	w.AgeSeconds = life.age
	w.PresentForSeconds = life.streak
	w.VisibleSeconds = life.visible
	w.AppearCount = life.appears
}

type wallLifeView struct {
	behavior string
	age      float64
	streak   float64
	visible  float64
	appears  int
}

func wallLife(tr *wallTrack, now time.Time) wallLifeView {
	visible := tr.visible
	streak := time.Duration(0)
	if tr.present {
		if !tr.lastSeen.IsZero() && now.After(tr.lastSeen) {
			visible += now.Sub(tr.lastSeen)
		}
		start := tr.streakStart
		if start.IsZero() {
			start = tr.firstSeen
		}
		if now.After(start) {
			streak = now.Sub(start)
		}
	}
	age := time.Duration(0)
	if !tr.firstSeen.IsZero() && now.After(tr.firstSeen) {
		age = now.Sub(tr.firstSeen)
	}
	flips := 0
	cut := now.Add(-wallSuspiciousWindow)
	for _, a := range tr.appears {
		if !a.Before(cut) {
			flips++
		}
	}
	behavior := WallBehaviorShort
	if flips >= wallSuspiciousAppears {
		behavior = WallBehaviorSuspicious
	} else if streak >= WallPersistentMin {
		behavior = WallBehaviorPersistent
	} else if visible >= WallPersistentMin && age > 0 && float64(visible)/float64(age) >= wallPersistentDuty {
		behavior = WallBehaviorPersistent
	}
	return wallLifeView{
		behavior: behavior,
		age:      seconds1(age),
		streak:   seconds1(streak),
		visible:  seconds1(visible),
		appears:  tr.appearCount,
	}
}

func sameWallPrice(a, b float64) bool {
	if a <= 0 || b <= 0 {
		return false
	}
	ref := a
	if b > ref {
		ref = b
	}
	return absFloat(a-b) <= ref*wallPriceMatchFrac
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func seconds1(d time.Duration) float64 {
	if d <= 0 {
		return 0
	}
	return round4(d.Seconds())
}
