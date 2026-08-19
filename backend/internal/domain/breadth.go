package domain

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	BreadthWindow1h  = LiquidationWindow1h
	BreadthWindow4h  = LiquidationWindow4h
	BreadthWindow24h = LiquidationWindow24h

	BreadthDirUp   = "up"
	BreadthDirDown = "down"
	BreadthDirFlat = "flat"

	// Majors moving with the majority of coins.
	BreadthAlignWithMarket = "with_market"
	// BTC/ETH (or volume) up/down while most coins go the other way.
	BreadthAlignCarrying = "carrying"
	// Most coins up/down while BTC/ETH go the other way.
	BreadthAlignLagging = "lagging"
	BreadthAlignMixed   = "mixed"

	breadthFlatPct  = 0.05 // |change| < 5 bp counts as unchanged
	breadthLeanPct  = 55.0
	breadthCarryGap = 15.0 // volume share vs coin share
	breadthDefaultN = 80
	breadthMaxN     = 150
	breadthMinN     = 10
)

// CoinMove is one symbol's percent change in a lookback.
type CoinMove struct {
	Symbol      string
	Base        string
	ChangePct   float64
	QuoteVolume float64
	Known       bool
}

// WindowChange is a venue snapshot of percent change for requested symbols.
type WindowChange struct {
	Symbol    string
	ChangePct float64
}

// WindowChangePort loads rolling percent changes for many symbols at once.
type WindowChangePort interface {
	GetWindowChanges(ctx context.Context, window string, symbols []string) ([]WindowChange, error)
}

// BreadthCounts is how many coins (and how much volume) went up or down.
type BreadthCounts struct {
	Up            int
	Down          int
	Flat          int
	Total         int
	UpPct         float64
	DownPct       float64
	FlatPct       float64
	VolumeUpPct   float64
	VolumeDownPct float64
}

// BreadthWindow is one lookback of market breadth.
type BreadthWindow struct {
	Window    string
	Counts    BreadthCounts
	BTC       *CoinMove
	ETH       *CoinMove
	Alignment string
	Title     string
	Summary   string
	Complete  bool
}

// BreadthReport is the API result.
type BreadthReport struct {
	Exchange string
	Quote    string
	Universe int
	AsOf     time.Time
	Windows  []BreadthWindow
	Summary  string
	Note     string
}

// ParseBreadthLimit clamps the universe size.
func ParseBreadthLimit(n int) int {
	if n <= 0 {
		return breadthDefaultN
	}
	if n < breadthMinN {
		return breadthMinN
	}
	if n > breadthMaxN {
		return breadthMaxN
	}
	return n
}

// ClassifyMove is up / down / flat for a percent change.
func ClassifyMove(pct float64) string {
	if math.IsNaN(pct) || math.IsInf(pct, 0) {
		return BreadthDirFlat
	}
	switch {
	case pct > breadthFlatPct:
		return BreadthDirUp
	case pct < -breadthFlatPct:
		return BreadthDirDown
	default:
		return BreadthDirFlat
	}
}

// IsBreadthEligible drops leveraged / inverse tokens that are not "the market".
func IsBreadthEligible(base string) bool {
	b := strings.ToUpper(strings.TrimSpace(base))
	if b == "" {
		return false
	}
	for _, suf := range []string{"UP", "DOWN", "BULL", "BEAR", "3L", "3S", "2L", "2S", "5L", "5S"} {
		if strings.HasSuffix(b, suf) && len(b) > len(suf) {
			return false
		}
	}
	return true
}

// CountBreadth tallies up/down/flat and volume shares.
func CountBreadth(moves []CoinMove) BreadthCounts {
	var c BreadthCounts
	var vol, volUp, volDown float64
	for _, m := range moves {
		if !m.Known || math.IsNaN(m.ChangePct) {
			continue
		}
		c.Total++
		if m.QuoteVolume > 0 {
			vol += m.QuoteVolume
		}
		switch ClassifyMove(m.ChangePct) {
		case BreadthDirUp:
			c.Up++
			if m.QuoteVolume > 0 {
				volUp += m.QuoteVolume
			}
		case BreadthDirDown:
			c.Down++
			if m.QuoteVolume > 0 {
				volDown += m.QuoteVolume
			}
		default:
			c.Flat++
		}
	}
	if c.Total > 0 {
		n := float64(c.Total)
		c.UpPct = float64(c.Up) / n * 100
		c.DownPct = float64(c.Down) / n * 100
		c.FlatPct = float64(c.Flat) / n * 100
	}
	if vol > 0 {
		c.VolumeUpPct = volUp / vol * 100
		c.VolumeDownPct = volDown / vol * 100
	}
	return c
}

// MarketLean is the side most coins took, or mixed.
func MarketLean(c BreadthCounts) string {
	if c.Total == 0 {
		return BreadthDirFlat
	}
	if c.UpPct >= breadthLeanPct {
		return BreadthDirUp
	}
	if c.DownPct >= breadthLeanPct {
		return BreadthDirDown
	}
	return BreadthAlignMixed
}

func findMove(moves []CoinMove, base string) *CoinMove {
	want := strings.ToUpper(base)
	for i := range moves {
		if strings.EqualFold(moves[i].Base, want) {
			cp := moves[i]
			return &cp
		}
	}
	return nil
}

// BuildBreadthWindow scores one lookback and writes the BTC/ETH read.
func BuildBreadthWindow(id string, moves []CoinMove) BreadthWindow {
	w := BreadthWindow{
		Window: id,
		Counts: CountBreadth(moves),
		BTC:    findMove(moves, "BTC"),
		ETH:    findMove(moves, "ETH"),
	}
	w.Complete = w.Counts.Total >= breadthMinN
	w.Alignment, w.Title, w.Summary = ExplainBreadthWindow(w)
	return w
}

// ExplainBreadthWindow says whether BTC/ETH agree with the pack.
func ExplainBreadthWindow(w BreadthWindow) (align, title, summary string) {
	c := w.Counts
	if c.Total == 0 {
		return BreadthAlignMixed, "No breadth yet",
			"Not enough price changes to say how many coins are up or down."
	}
	summary = fmt.Sprintf("%d of %d coins are up (%s%%) and %d are down (%s%%) over %s.",
		c.Up, c.Total, formatFixed(c.UpPct, 0), c.Down, formatFixed(c.DownPct, 0), w.Window)
	if c.Flat > 0 {
		summary += fmt.Sprintf(" %d are roughly unchanged.", c.Flat)
	}

	lean := MarketLean(c)
	btcDir, ethDir := "", ""
	if w.BTC != nil && w.BTC.Known {
		btcDir = ClassifyMove(w.BTC.ChangePct)
	}
	if w.ETH != nil && w.ETH.Known {
		ethDir = ClassifyMove(w.ETH.ChangePct)
	}
	majors := majorLean(btcDir, ethDir)
	carry := isCarrying(c, lean, majors)

	switch {
	case carry:
		align = BreadthAlignCarrying
		if majors == BreadthDirUp || (majors == "" && c.VolumeUpPct >= c.VolumeDownPct) {
			title = "A few large coins are carrying the market up"
			summary += " BTC/ETH or the high-volume names are up, but most coins are not — the tape is narrower than the big coins suggest."
		} else {
			title = "A few large coins are dragging the market down"
			summary += " BTC/ETH or the high-volume names are down, but most coins are holding up better."
		}
	case (majors == "" || majors == BreadthDirFlat) && (lean == BreadthDirUp || lean == BreadthDirDown):
		align = BreadthAlignMixed
		title = "Most coins are " + lean + "; BTC and ETH are little changed"
		summary += " The pack is " + lean + ", while the large coins are roughly flat."
	case majors != "" && lean != BreadthAlignMixed && majors == lean:
		align = BreadthAlignWithMarket
		title = "BTC and ETH are moving with the rest of the market"
		summary += " " + majorsPhrase(majors) + " with the majority of coins."
	case majors != "" && lean != BreadthAlignMixed && majors != lean && majors != BreadthDirFlat:
		align = BreadthAlignLagging
		title = "BTC and ETH are not moving with the rest of the market"
		summary += " Most coins are " + lean + ", while " + majorsPhrase(majors) + "."
	default:
		align = BreadthAlignMixed
		title = "The market is split"
		if majors != "" && majors != BreadthAlignMixed && majors != BreadthDirFlat {
			summary += " " + majorsPhrase(majors) + "; coin internals are mixed."
		} else {
			summary += " Neither side has a clear majority."
		}
	}
	return align, title, summary
}

func majorLean(btcDir, ethDir string) string {
	if btcDir == "" && ethDir == "" {
		return ""
	}
	if btcDir == ethDir {
		return btcDir
	}
	if btcDir != "" && (ethDir == "" || ethDir == BreadthDirFlat) {
		return btcDir
	}
	if ethDir != "" && (btcDir == "" || btcDir == BreadthDirFlat) {
		return ethDir
	}
	if btcDir != ethDir && btcDir != "" && ethDir != "" && btcDir != BreadthDirFlat && ethDir != BreadthDirFlat {
		return BreadthAlignMixed
	}
	return btcDir
}

func isCarrying(c BreadthCounts, lean, majors string) bool {
	if c.Total == 0 {
		return false
	}
	// Volume tells a different story than the coin count (index vs internals).
	if c.VolumeUpPct-c.UpPct >= breadthCarryGap && c.VolumeUpPct >= 55 && c.UpPct < 45 {
		return true
	}
	if c.VolumeDownPct-c.DownPct >= breadthCarryGap && c.VolumeDownPct >= 55 && c.DownPct < 45 {
		return true
	}
	// BTC/ETH up while most coins are down — the familiar "few names carrying" tape.
	// The reverse (majors down, alts up) is lagging, not carrying.
	if majors == BreadthDirUp && lean == BreadthDirDown {
		return true
	}
	return false
}

func majorsPhrase(dir string) string {
	switch dir {
	case BreadthDirUp:
		return "BTC and ETH are up"
	case BreadthDirDown:
		return "BTC and ETH are down"
	case BreadthDirFlat:
		return "BTC and ETH are roughly unchanged"
	default:
		return "BTC and ETH disagree"
	}
}

// ExplainBreadthReport rolls the three windows into one paragraph.
func ExplainBreadthReport(windows []BreadthWindow) string {
	var h1, h4, h24 *BreadthWindow
	for i := range windows {
		w := &windows[i]
		switch w.Window {
		case BreadthWindow1h:
			h1 = w
		case BreadthWindow4h:
			h4 = w
		case BreadthWindow24h:
			h24 = w
		}
	}
	parts := make([]string, 0, 3)
	if h1 != nil && h1.Counts.Total > 0 {
		parts = append(parts, fmt.Sprintf("1h: %d/%d up (%s%%)", h1.Counts.Up, h1.Counts.Total, formatFixed(h1.Counts.UpPct, 0)))
	}
	if h4 != nil && h4.Counts.Total > 0 {
		parts = append(parts, fmt.Sprintf("4h: %d/%d up (%s%%)", h4.Counts.Up, h4.Counts.Total, formatFixed(h4.Counts.UpPct, 0)))
	}
	if h24 != nil && h24.Counts.Total > 0 {
		parts = append(parts, fmt.Sprintf("24h: %d/%d up (%s%%)", h24.Counts.Up, h24.Counts.Total, formatFixed(h24.Counts.UpPct, 0)))
	}
	if len(parts) == 0 {
		return "Not enough coins to measure market breadth."
	}
	head := joinList(parts) + "."
	// Prefer the 24h read for the "who is carrying" line; fall back to 1h.
	src := h24
	if src == nil || src.Alignment == "" {
		src = h1
	}
	if src != nil && src.Title != "" && src.Counts.Total > 0 {
		return head + " " + src.Title + " on the " + src.Window + "."
	}
	return head
}

// ParseChangePct parses a venue percent string (Binance-style).
func ParseChangePct(raw string) (float64, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, false
	}
	v, err := parseClose(s)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	return v, true
}
