package domain

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	WallScanKindSuspicious = "suspicious"
	WallScanKindFlicker    = "flicker"

	// wallFlickerMax is a rest that left before it could be called persistent.
	wallFlickerMax = 20 * time.Second
	// wallFlickerKeep is how long a pulled wall stays on the market scan.
	wallFlickerKeep = 10 * time.Minute

	DefaultWallScanLimit = 40
	MaxWallScanHits      = 80
)

// WallScanHit is one suspicious or briefly pulled wall on the market scan.
type WallScanHit struct {
	Exchange          string
	Symbol            string
	Side              string
	Price             string
	Notional          string
	DistancePct       string
	Share             float64
	Kind              string
	AppearCount       int
	AgeSeconds        float64
	PresentForSeconds float64
	VisibleSeconds    float64
	Present           bool
	LastSeen          time.Time
}

// WallScan ranks suspicious walls and brief pulls across watched books.
type WallScan struct {
	Exchange   string
	Quote      string
	Universe   int
	AsOf       time.Time
	Suspicious int
	Flickers   int
	Hits       []WallScanHit
	Summary    string
	Note       string
}

// ParseWallScanExchange accepts binance, coinbase, bybit, or all.
func ParseWallScanExchange(raw string) (string, error) {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" || raw == "all" {
		return "all", nil
	}
	switch raw {
	case string(ExchangeBinance), string(ExchangeCoinbase), string(ExchangeBybit):
		return raw, nil
	default:
		return "", fmt.Errorf("%w: exchange must be binance, coinbase, bybit, or all", ErrInvalidArgument)
	}
}

// ParseWallScanLimit clamps how many hits the scan returns.
func ParseWallScanLimit(n int) int {
	if n <= 0 {
		return DefaultWallScanLimit
	}
	if n > MaxWallScanHits {
		return MaxWallScanHits
	}
	return n
}

// Export lists walls that are flickering or were pulled after a short rest.
func (m *WallMemory) Export(now time.Time) []WallScanHit {
	if m == nil {
		return nil
	}
	now = now.UTC()
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]WallScanHit, 0)
	for _, tr := range m.tracks {
		if tr == nil {
			continue
		}
		ref := tr.lastSeen
		if ref.IsZero() {
			ref = tr.firstSeen
		}
		if !ref.IsZero() && now.Sub(ref) > wallTrackTTL {
			continue
		}
		life := wallLife(tr, now)
		kind := ""
		if life.behavior == WallBehaviorSuspicious {
			kind = WallScanKindSuspicious
		} else if briefPull(tr, now) {
			kind = WallScanKindFlicker
		}
		if kind == "" {
			continue
		}
		price := tr.priceText
		if price == "" {
			price = strconv.FormatFloat(tr.price, 'f', -1, 64)
		}
		out = append(out, WallScanHit{
			Exchange:          tr.exchange,
			Symbol:            tr.symbol,
			Side:              tr.side,
			Price:             price,
			Notional:          tr.notional,
			DistancePct:       tr.distance,
			Share:             tr.share,
			Kind:              kind,
			AppearCount:       life.appears,
			AgeSeconds:        life.age,
			PresentForSeconds: life.streak,
			VisibleSeconds:    life.visible,
			Present:           tr.present,
			LastSeen:          tr.lastSeen,
		})
	}
	return out
}

func briefPull(tr *wallTrack, now time.Time) bool {
	if tr == nil || tr.present || tr.notional == "" || tr.lastStreak <= 0 || tr.lastStreak > wallFlickerMax {
		return false
	}
	if tr.lastAbsent.IsZero() {
		return false
	}
	gone := now.Sub(tr.lastAbsent)
	return gone >= wallAbsenceGrace && gone <= wallFlickerKeep
}

// BuildWallScan keeps one venue (or all) and ranks suspicious walls ahead of brief pulls.
func BuildWallScan(exchange, quote string, universe int, rows []WallScanHit, now time.Time, limit int) WallScan {
	exchange = strings.ToLower(strings.TrimSpace(exchange))
	if exchange == "" {
		exchange = "all"
	}
	quote = strings.ToUpper(strings.TrimSpace(quote))
	if quote == "" {
		quote = "USDT"
	}
	limit = ParseWallScanLimit(limit)
	kept := make([]WallScanHit, 0, len(rows))
	suspicious, flickers := 0, 0
	for _, row := range rows {
		if exchange != "all" && row.Exchange != exchange {
			continue
		}
		kept = append(kept, row)
		switch row.Kind {
		case WallScanKindSuspicious:
			suspicious++
		case WallScanKindFlicker:
			flickers++
		}
	}
	sort.SliceStable(kept, func(i, j int) bool {
		return wallScanBefore(kept[i], kept[j])
	})
	if len(kept) > limit {
		kept = kept[:limit]
	}
	if kept == nil {
		kept = []WallScanHit{}
	}
	return WallScan{
		Exchange:   exchange,
		Quote:      quote,
		Universe:   universe,
		AsOf:       now.UTC(),
		Suspicious: suspicious,
		Flickers:   flickers,
		Hits:       kept,
		Summary:    wallScanSummary(suspicious, flickers, universe),
	}
}

func wallScanBefore(a, b WallScanHit) bool {
	if a.Kind != b.Kind {
		return a.Kind == WallScanKindSuspicious
	}
	if a.AppearCount != b.AppearCount {
		return a.AppearCount > b.AppearCount
	}
	an := wallNotional(a.Notional)
	bn := wallNotional(b.Notional)
	if an != bn {
		return an > bn
	}
	if a.Symbol != b.Symbol {
		return a.Symbol < b.Symbol
	}
	if a.Side != b.Side {
		return a.Side < b.Side
	}
	return a.Price < b.Price
}

func wallNotional(raw string) float64 {
	n, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return 0
	}
	return n
}

func wallScanSummary(suspicious, flickers, universe int) string {
	return fmt.Sprintf("%d suspicious walls and %d brief flickers across %d watched books", suspicious, flickers, universe)
}
