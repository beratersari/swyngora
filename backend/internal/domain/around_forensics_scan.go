package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	DefaultForensicsScanLimit    = 12
	MaxForensicsScanLimit        = 20
	DefaultForensicsScanLookback = "3d"
	DefaultForensicsScanQuote    = "USDT"
	forensicsScanWorkers         = 3
)

// ForensicsScanHit is one coin whose current tape looks like a past pre-move setup.
type ForensicsScanHit struct {
	Symbol           string
	Exchange         string
	Similarity       float64
	Coverage         float64
	Events           int
	Matches          int
	UpAfter          int
	DownAfter        int
	MedianAfterPct   float64
	AfterHorizons    []AroundSimilarHorizonStat
	ClosestAt        time.Time
	ClosestDirection string
	ClosestReturnPct float64
	Lean             string                    // up | down | mixed | unknown
	Compared         []AroundSimilarFieldScore // closest twin: price, volume, takers, oi, book, options
	WhySimilar       string
	AfterStory       string
	Summary          string
}

// ForensicsScanReport ranks coins whose current tape looks like past important-move setups.
type ForensicsScanReport struct {
	Exchange    string
	Quote       string
	Lookback    string
	Window      string
	Interval    string
	Direction   string
	Fields      []string
	MinCoverage float64
	Horizons    []string
	SymbolLimit int
	Scanned     int
	Compared    int
	AsOf        time.Time
	Hits        []ForensicsScanHit
	Summary     string
	Note        string
}

const ForensicsScanDisclaimer = "Compares each coin's current tape (last window) to the setup before that coin's own important past moves. Ranked by how close the current tape is. After-windows say what usually happened next on those past cases. Informational only — not financial advice."

// ClampForensicsScanLimit keeps the universe small enough to scan live.
func ClampForensicsScanLimit(n int) int {
	if n <= 0 {
		return DefaultForensicsScanLimit
	}
	if n > MaxForensicsScanLimit {
		return MaxForensicsScanLimit
	}
	return n
}

// ForensicsScanWorkers is how many coins to compare at once.
func ForensicsScanWorkers() int { return forensicsScanWorkers }

// ForensicsScanHitFromSimilar keeps a coin that has at least one similar past setup.
func ForensicsScanHitFromSimilar(rep *AroundSimilarReport) (ForensicsScanHit, bool) {
	if rep == nil || len(rep.Matches) == 0 {
		return ForensicsScanHit{}, false
	}
	top := rep.Matches[0]
	h := ForensicsScanHit{
		Symbol:           rep.Symbol,
		Exchange:         rep.Exchange,
		Similarity:       top.Similarity,
		Coverage:         top.Coverage,
		Events:           rep.Events,
		Matches:          len(rep.Matches),
		UpAfter:          rep.UpAfter,
		DownAfter:        rep.DownAfter,
		MedianAfterPct:   rep.MedianAfterPct,
		AfterHorizons:    append([]AroundSimilarHorizonStat(nil), rep.AfterHorizons...),
		ClosestAt:        top.Move.At,
		ClosestDirection: top.AfterDirection,
		ClosestReturnPct: top.AfterReturnPct,
		Compared:         append([]AroundSimilarFieldScore(nil), top.Compared...),
	}
	if h.Events == 0 {
		h.Events = len(rep.Matches)
	}
	up, down := rep.UpAfter, rep.DownAfter
	if hz := pickForensicsWhyHorizon(rep.AfterHorizons); hz.Sample > 0 || hz.Events > 0 {
		up, down = hz.Up, hz.Down
	}
	h.Lean = ForensicsScanLean(up, down)
	h.WhySimilar = ExplainForensicsScanWhy(h)
	h.AfterStory = ExplainForensicsScanAfter(h)
	h.Summary = ExplainForensicsScanHit(h)
	return h, true
}

const (
	forensicsScanFieldClose = 70.0
	forensicsScanFieldSome  = 50.0
)

func prettyScanField(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case AroundSimilarFieldPrice:
		return "price"
	case AroundSimilarFieldVolume:
		return "volume"
	case AroundSimilarFieldTakers:
		return "taker flow"
	case AroundSimilarFieldOI:
		return "open interest"
	case AroundSimilarFieldBook:
		return "the order book"
	case AroundSimilarFieldOptions:
		return "listed options"
	default:
		return name
	}
}

func scanFieldWere(items []string, singular, plural string) string {
	if len(items) == 1 {
		return items[0] + " " + singular
	}
	return joinList(items) + " " + plural
}

// ExplainForensicsScanWhy says which tape pieces matched the closest past setup.
func ExplainForensicsScanWhy(h ForensicsScanHit) string {
	var close, some, far, missing []string
	for _, c := range h.Compared {
		label := prettyScanField(c.Name)
		if label == "" {
			continue
		}
		if !c.Used {
			missing = append(missing, label)
			continue
		}
		switch {
		case c.Score+1e-9 >= forensicsScanFieldClose:
			close = append(close, label)
		case c.Score+1e-9 >= forensicsScanFieldSome:
			some = append(some, label)
		default:
			far = append(far, label)
		}
	}
	if len(close)+len(some)+len(far)+len(missing) == 0 {
		return ""
	}
	var parts []string
	switch {
	case len(close) > 0:
		parts = append(parts, "Closest past twin matched most on "+joinList(close)+".")
		if len(some) > 0 {
			parts = append(parts, scanFieldWere(some, "was only somewhat similar.", "were only somewhat similar."))
		}
		if len(far) > 0 {
			parts = append(parts, scanFieldWere(far, "looked different.", "looked different."))
		}
	case len(some) > 0:
		parts = append(parts, "Closest past twin was only somewhat similar on "+joinList(some)+".")
		if len(far) > 0 {
			parts = append(parts, scanFieldWere(far, "looked different.", "looked different."))
		}
	case len(far) > 0:
		parts = append(parts, "None of the compared tape pieces were close — "+scanFieldWere(far, "looked different.", "looked different."))
	}
	if len(missing) > 0 {
		parts = append(parts, scanFieldWere(missing, "could not be compared.", "could not be compared."))
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

// ForensicsScanLean is up / down / mixed / unknown from after counts.
func ForensicsScanLean(up, down int) string {
	total := up + down
	if total <= 0 {
		return "unknown"
	}
	if up*100 >= 65*total {
		return CVDDirUp
	}
	if down*100 >= 65*total {
		return CVDDirDown
	}
	return "mixed"
}

// ExplainForensicsScanAfter says what usually happened after similar setups.
func ExplainForensicsScanAfter(h ForensicsScanHit) string {
	name := prettyBase(h.Symbol)
	if name == "" {
		name = "This coin"
	}
	hz := pickForensicsWhyHorizon(h.AfterHorizons)
	up, down, horizon := h.UpAfter, h.DownAfter, ""
	if hz.Horizon != "" && (hz.Sample > 0 || hz.Events > 0) {
		up, down, horizon = hz.Up, hz.Down, hz.Horizon
	}
	total := up + down
	switch {
	case total <= 0 && h.ClosestDirection == CVDDirUp:
		return name + " has a close past twin that pushed higher afterward, but there is not enough of a sample to call a usual path."
	case total <= 0 && h.ClosestDirection == CVDDirDown:
		return name + " has a close past twin that sold off afterward, but there is not enough of a sample to call a usual path."
	case total <= 0:
		return name + " looks like a past setup, but there is not enough after-tape to say what usually came next."
	}
	when := "afterward"
	if horizon != "" {
		when = "over the next " + horizon
	}
	switch h.Lean {
	case CVDDirUp:
		return fmt.Sprintf("When %s looked like this before, price more often kept rising %s than it reversed.", name, when)
	case CVDDirDown:
		return fmt.Sprintf("When %s looked like this before, price more often fell %s than it continued higher.", name, when)
	default:
		return fmt.Sprintf("When %s looked like this before, similar cases split %s — the setup has not reliably picked a side.", name, when)
	}
}

// ExplainForensicsScanHit is one ranked coin in one sentence.
func ExplainForensicsScanHit(h ForensicsScanHit) string {
	name := prettyBase(h.Symbol)
	if name == "" {
		name = h.Symbol
	}
	head := fmt.Sprintf("%s currently looks like a past pre-move setup.", name)
	if why := strings.TrimSpace(h.WhySimilar); why != "" {
		head += " " + why
	}
	if s := strings.TrimSpace(h.AfterStory); s != "" {
		return head + " " + s
	}
	return head
}

// FinishForensicsScan ranks hits and writes the desk summary.
func FinishForensicsScan(r *ForensicsScanReport) {
	if r == nil {
		return
	}
	sort.SliceStable(r.Hits, func(i, j int) bool {
		if r.Hits[i].Similarity == r.Hits[j].Similarity {
			return r.Hits[i].Events > r.Hits[j].Events
		}
		return r.Hits[i].Similarity > r.Hits[j].Similarity
	})
	r.Summary = ExplainForensicsScan(*r)
}

// ExplainForensicsScan writes how many coins currently look like past setups.
func ExplainForensicsScan(r ForensicsScanReport) string {
	if r.Compared == 0 {
		return "No coins had enough current tape to compare."
	}
	if len(r.Hits) == 0 {
		return fmt.Sprintf("Scanned %d coin(s); none currently look like their own important-move setups.", r.Compared)
	}
	up, down, mixed := 0, 0, 0
	for _, h := range r.Hits {
		switch h.Lean {
		case CVDDirUp:
			up++
		case CVDDirDown:
			down++
		case "mixed":
			mixed++
		}
	}
	head := fmt.Sprintf("%d of %d scanned coin(s) currently look like a past pre-move setup.", len(r.Hits), r.Compared)
	switch {
	case up > 0 && down == 0 && mixed == 0:
		head += " Those similar cases more often kept rising afterward."
	case down > 0 && up == 0 && mixed == 0:
		head += " Those similar cases more often fell afterward."
	case up+down+mixed > 0:
		head += fmt.Sprintf(" After similar cases: %d usually rose, %d usually fell, %d split.", up, down, mixed)
	}
	top := r.Hits[0]
	name := prettyBase(top.Symbol)
	if name == "" {
		name = top.Symbol
	}
	head += " Closest now is " + name + "."
	return head
}
