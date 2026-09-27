package domain

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	SpreadWidenMinSamples = 5
	SpreadWidenMinRatio   = 1.5
	// Ignore a wider print that is only noise on an already tight book.
	spreadWidenMinGapPct   = 0.005
	spreadWidenKeep        = 48
	DefaultSpreadScanLimit = 20
	MaxSpreadScanLimit     = 40
)

// SpreadSample is one top-of-book spread observation.
type SpreadSample struct {
	Time      time.Time
	SpreadPct float64
}

// SpreadTrack is one venue book we have been watching.
type SpreadTrack struct {
	Exchange Exchange
	Symbol   string
	Samples  []SpreadSample
}

// SpreadHit is a coin whose latest spread is wide versus its own typical.
type SpreadHit struct {
	Exchange   Exchange
	Symbol     string
	Base       string
	CurrentPct float64
	TypicalPct float64
	Ratio      float64
	Samples    int
	AsOf       time.Time
}

// SpreadScan ranks books that have opened versus their own recent spread.
type SpreadScan struct {
	Exchange string
	Quote    string
	Known    int
	Ready    int
	Hits     []SpreadHit
	Summary  string
	Note     string
}

// SpreadReport is one coin's current spread versus its own typical.
type SpreadReport struct {
	Symbol     string
	Exchange   string
	CurrentPct float64
	TypicalPct float64
	Ratio      float64
	Samples    int
	Opened     bool
	AsOf       time.Time
	Summary    string
	Note       string
}

const spreadWidenNote = "Current top-of-book spread versus the median of earlier samples of that same book. A coin is wide at 1.5× its own typical. Samples collect while this process watches the book. Not financial advice."

// SpreadMemory keeps recent bid-ask spreads per venue book.
type SpreadMemory struct {
	mu    sync.Mutex
	books map[string]*spreadBook
}

type spreadBook struct {
	ex      Exchange
	symbol  string
	samples []SpreadSample
}

// NewSpreadMemory returns an empty tracker.
func NewSpreadMemory() *SpreadMemory {
	return &SpreadMemory{books: map[string]*spreadBook{}}
}

// Observe records the top-of-book spread. Bids must be best-first.
func (m *SpreadMemory) Observe(now time.Time, ex Exchange, symbol string, book RawOrderBook) {
	if m == nil {
		return
	}
	pct, ok := TopSpreadPct(book)
	if !ok {
		return
	}
	symbol = NormalizeSymbol(ex, symbol)
	if symbol == "" {
		symbol = strings.ToUpper(strings.TrimSpace(book.Symbol))
	}
	if symbol == "" {
		return
	}
	key := string(ex) + "|" + symbol
	m.mu.Lock()
	defer m.mu.Unlock()
	b := m.books[key]
	if b == nil {
		b = &spreadBook{ex: ex, symbol: symbol}
		m.books[key] = b
	}
	b.samples = append(b.samples, SpreadSample{Time: now.UTC(), SpreadPct: pct})
	if len(b.samples) > spreadWidenKeep {
		b.samples = b.samples[len(b.samples)-spreadWidenKeep:]
	}
}

// Tracks copies the books currently stored.
func (m *SpreadMemory) Tracks() []SpreadTrack {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]SpreadTrack, 0, len(m.books))
	for _, b := range m.books {
		cp := make([]SpreadSample, len(b.samples))
		copy(cp, b.samples)
		out = append(out, SpreadTrack{Exchange: b.ex, Symbol: b.symbol, Samples: cp})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Exchange == out[j].Exchange {
			return out[i].Symbol < out[j].Symbol
		}
		return out[i].Exchange < out[j].Exchange
	})
	return out
}

// TopSpreadPct is (best ask − best bid) / mid × 100.
func TopSpreadPct(book RawOrderBook) (float64, bool) {
	var bid, ask float64
	for _, lv := range book.Bids {
		if validLevel(lv) {
			bid = lv.Price
			break
		}
	}
	for _, lv := range book.Asks {
		if validLevel(lv) {
			ask = lv.Price
			break
		}
	}
	if bid <= 0 || ask <= bid {
		return 0, false
	}
	mid := (bid + ask) / 2
	if mid <= 0 {
		return 0, false
	}
	return round4((ask - bid) / mid * 100), true
}

// ParseSpreadScanLimit clamps the ranking size.
func ParseSpreadScanLimit(n int) int {
	if n <= 0 {
		return DefaultSpreadScanLimit
	}
	if n > MaxSpreadScanLimit {
		return MaxSpreadScanLimit
	}
	return n
}

// BuildSpreadScan keeps books whose latest spread is at least 1.5× their own median.
func BuildSpreadScan(exchange, quote string, tracks []SpreadTrack, limit int) SpreadScan {
	limit = ParseSpreadScanLimit(limit)
	out := SpreadScan{
		Exchange: exchange,
		Quote:    quote,
		Known:    len(tracks),
		Hits:     []SpreadHit{},
		Note:     spreadWidenNote,
	}
	var hits []SpreadHit
	for _, tr := range tracks {
		if exchange != "" && exchange != "all" && string(tr.Exchange) != exchange {
			continue
		}
		hit, ok := widenHit(tr)
		if !ok {
			if len(tr.Samples) >= SpreadWidenMinSamples {
				out.Ready++
			}
			continue
		}
		out.Ready++
		hits = append(hits, hit)
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Ratio == hits[j].Ratio {
			return hits[i].Symbol < hits[j].Symbol
		}
		return hits[i].Ratio > hits[j].Ratio
	})
	if len(hits) > limit {
		hits = hits[:limit]
	}
	out.Hits = hits
	out.Summary = explainSpreadScan(out)
	return out
}

// SpreadReportFor builds the single-coin read. exchange "all" picks the widest ready book.
func SpreadReportFor(symbol, exchange string, tracks []SpreadTrack) SpreadReport {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	out := SpreadReport{Symbol: symbol, Exchange: exchange, Note: spreadWidenNote}
	var best *SpreadHit
	samples := 0
	for _, tr := range tracks {
		if !strings.EqualFold(tr.Symbol, symbol) && !strings.EqualFold(NormalizeSymbol(tr.Exchange, symbol), tr.Symbol) {
			continue
		}
		if exchange != "" && exchange != "all" && string(tr.Exchange) != exchange {
			continue
		}
		if len(tr.Samples) > samples {
			samples = len(tr.Samples)
		}
		hit, ok := widenHit(tr)
		if !ok {
			continue
		}
		if best == nil || hit.Ratio > best.Ratio {
			cp := hit
			best = &cp
		}
	}
	if best == nil {
		out.Samples = samples
		out.Summary = "Not enough samples yet to compare this spread with its own typical."
		return out
	}
	out.Exchange = string(best.Exchange)
	out.CurrentPct = best.CurrentPct
	out.TypicalPct = best.TypicalPct
	out.Ratio = best.Ratio
	out.Samples = best.Samples
	out.Opened = true
	out.AsOf = best.AsOf
	out.Summary = fmt.Sprintf("%s spread is %s× its typical (%s%% now, typical %s%%).",
		out.Symbol, formatQty(out.Ratio), formatQty(out.CurrentPct), formatQty(out.TypicalPct))
	return out
}

func widenHit(tr SpreadTrack) (SpreadHit, bool) {
	if len(tr.Samples) < SpreadWidenMinSamples {
		return SpreadHit{}, false
	}
	last := tr.Samples[len(tr.Samples)-1]
	prior := make([]float64, 0, len(tr.Samples)-1)
	for _, s := range tr.Samples[:len(tr.Samples)-1] {
		if s.SpreadPct > 0 {
			prior = append(prior, s.SpreadPct)
		}
	}
	if len(prior) < SpreadWidenMinSamples-1 || last.SpreadPct <= 0 {
		return SpreadHit{}, false
	}
	typical := medianFloat(prior)
	if typical <= 0 {
		return SpreadHit{}, false
	}
	ratio := last.SpreadPct / typical
	if ratio < SpreadWidenMinRatio || last.SpreadPct-typical < spreadWidenMinGapPct {
		return SpreadHit{}, false
	}
	base, _ := SplitBaseQuote(tr.Exchange, tr.Symbol)
	return SpreadHit{
		Exchange: tr.Exchange, Symbol: tr.Symbol, Base: base,
		CurrentPct: round4(last.SpreadPct), TypicalPct: round4(typical), Ratio: round4(ratio),
		Samples: len(tr.Samples), AsOf: last.Time,
	}, true
}

func explainSpreadScan(s SpreadScan) string {
	if len(s.Hits) == 0 {
		return fmt.Sprintf("No book is 1.5× wider than its own typical yet (%d watched, %d with enough samples).", s.Known, s.Ready)
	}
	top := s.Hits[0]
	return fmt.Sprintf("%d books are at least 1.5× their own typical spread. Widest is %s on %s at %s× (%s%% now, typical %s%%).",
		len(s.Hits), top.Symbol, top.Exchange, formatQty(top.Ratio), formatQty(top.CurrentPct), formatQty(top.TypicalPct))
}
