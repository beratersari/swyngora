package domain

import (
	"math"
	"sort"
	"time"
)

const (
	// OpenLevelStatusOpen is still on the book.
	OpenLevelStatusOpen = "open"
	// OpenLevelStatusHit means a later candle traded through the level.
	OpenLevelStatusHit = "hit"
	// OpenLevelStatusClosed means estimated open interest fell. Not a liquidation.
	OpenLevelStatusClosed = "closed"

	// openLevelClusterFrac groups lots of the same side and leverage
	// whose prices sit within 0.2% of the reference.
	openLevelClusterFrac = 0.002
	openLevelMaxOpen     = 80
	openLevelMaxHit      = 40
	openLevelMaxClosed   = 24
	// openLevelChurnFrac ignores open-interest noise smaller than 0.2%.
	openLevelChurnFrac = 0.002
)

// OpenLevelSegment is one estimated band drawn from its open time
// until price trades through it or open interest on that side falls.
type OpenLevelSegment struct {
	Exchange string    `json:"exchange"`
	Side     string    `json:"side"`
	Leverage int       `json:"leverage"`
	Price    float64   `json:"price"`
	Notional float64   `json:"notional"`
	OpenedAt time.Time `json:"openedAt"`
	// ClosedAt is nil while the band is still open. encoding/json omitempty
	// keeps a zero time.Time, which would publish year 1 as a close.
	ClosedAt *time.Time `json:"closedAt,omitempty"`
	Status   string     `json:"status"`
}

// OpenLevelPoint is uncapped open long and short notional at one column.
// Delta is long minus short. Caps on Segments do not change this series.
type OpenLevelPoint struct {
	Time          time.Time `json:"time"`
	LongNotional  float64   `json:"longNotional"`
	ShortNotional float64   `json:"shortNotional"`
	Delta         float64   `json:"delta"`
}

// OpenLevelBook is one venue, or the combined book when every venue contributed.
type OpenLevelBook struct {
	Exchange      string             `json:"exchange"`
	Segments      []OpenLevelSegment `json:"segments"`
	Cumulative    []OpenLevelPoint   `json:"cumulative"`
	OpenLong      float64            `json:"openLong"`
	OpenShort     float64            `json:"openShort"`
	Delta         float64            `json:"delta"`
	Coverage      float64            `json:"coverage"`
	ColumnsWithOI int                `json:"columnsWithOi"`
	HitLots       int                `json:"hitLots"`
	ClosedLots    int                `json:"closedLots"`
	AccountBlend  string             `json:"accountBlend,omitempty"`
	// Definite is false for combined when any venue has no price and open interest,
	// or when the venues never overlap on a column.
	Definite bool `json:"definite"`
}

// OpenLevelsReport is the persistent liquidation-level read.
type OpenLevelsReport struct {
	Symbol        string               `json:"symbol"`
	Range         string               `json:"range"`
	From          time.Time            `json:"from"`
	To            time.Time            `json:"to"`
	StepSec       int                  `json:"stepSec"`
	LeverageMix   []HuntLeverageBucket `json:"leverageMix,omitempty"`
	Binance       OpenLevelBook        `json:"binance"`
	Bybit         OpenLevelBook        `json:"bybit"`
	OKX           OpenLevelBook        `json:"okx"`
	Combined      OpenLevelBook        `json:"combined"`
	MissingVenues []string             `json:"missingVenues,omitempty"`
	Note          string               `json:"note,omitempty"`
}

// OpenLevelsInput feeds BuildOpenLevels. Venues are the same series the heatmap loads.
type OpenLevelsInput struct {
	Symbol string
	Spec   HuntHeatmapSpec
	To     time.Time
	Venues []HuntHeatmapVenueSeries
}

const openLevelsNote = "Estimated liquidation levels that stay at the price where that slice of open interest was placed, until a later candle trades through them. The first column places the whole open interest then on the books, because earlier entries are unknown. Later columns only add an increase in estimated open interest. A level that price has already traded through is not put back. When estimated open interest on a side falls, the oldest open slices on that side are closed; that is an inventory change, not a liquidation. A long level is hit when a later candle low reaches it. A short level is hit when a later candle high reaches it. The candle that placed a level cannot hit it. Binance, Bybit, and OKX each use only their own price and open interest. Combined sums a column only when all three have both, and combined is withheld when any venue has none in this window. The leverage mix is the labeled hunt model with 0.40% maintenance. Not exchange liquidation-engine output. Binance public open interest is 30 days, so older columns exist only when this archive already stored them. Not financial advice."

type openLot struct {
	side     string
	display  int
	price    float64
	notional float64
	openedAt time.Time
	closedAt time.Time
	status   string
}

type openLevelWalk struct {
	book    OpenLevelBook
	hasData []bool
	cum     []OpenLevelPoint
}

// BuildOpenLevels walks each venue's own price and open interest and keeps
// a level until a later candle trades through it.
func BuildOpenLevels(in OpenLevelsInput) OpenLevelsReport {
	spec := in.Spec
	if spec.Step <= 0 {
		spec, _ = ParseHuntHeatmapRange("24h")
	}
	to := in.To.UTC()
	if to.IsZero() {
		to = time.Now().UTC()
	}
	times := HuntHeatmapColumns(spec, to)
	from := to.Add(-spec.Window)
	if len(times) > 0 {
		from = times[0].Add(-spec.Step)
	}
	out := OpenLevelsReport{
		Symbol:      NormalizeLiquidationSymbol(in.Symbol),
		Range:       string(spec.Range),
		From:        from,
		To:          to,
		StepSec:     int(spec.Step.Seconds()),
		LeverageMix: huntLeverageMixCopy(),
		Note:        openLevelsNote,
	}
	byEx := map[Exchange]HuntHeatmapVenueSeries{}
	for _, v := range in.Venues {
		byEx[v.Exchange] = v
	}
	binance := walkOpenLevels("binance", byEx[ExchangeBinance], times, spec.Step)
	bybit := walkOpenLevels("bybit", byEx[ExchangeBybit], times, spec.Step)
	okx := walkOpenLevels("okx", byEx[ExchangeOKX], times, spec.Step)
	out.Binance = binance.book
	out.Bybit = bybit.book
	out.OKX = okx.book
	out.Combined = combineOpenLevels(times, []openLevelWalk{binance, bybit, okx})
	out.MissingVenues = missingOpenVenues(out.Binance, out.Bybit, out.OKX)
	return out
}

func missingOpenVenues(binance, bybit, okx OpenLevelBook) []string {
	var out []string
	if binance.ColumnsWithOI == 0 {
		out = append(out, "binance")
	}
	if bybit.ColumnsWithOI == 0 {
		out = append(out, "bybit")
	}
	if okx.ColumnsWithOI == 0 {
		out = append(out, "okx")
	}
	return out
}

func walkOpenLevels(ex string, ser HuntHeatmapVenueSeries, times []time.Time, step time.Duration) openLevelWalk {
	n := len(times)
	out := openLevelWalk{
		hasData: make([]bool, n),
		cum:     make([]OpenLevelPoint, n),
		book: OpenLevelBook{
			Exchange:   ex,
			Segments:   []OpenLevelSegment{},
			Cumulative: []OpenLevelPoint{},
		},
	}
	if n == 0 {
		return out
	}
	stale := huntMatchStale(step)
	candles := append([]HuntHeatmapPricePoint(nil), ser.Prices...)
	sort.SliceStable(candles, func(i, j int) bool {
		return candles[i].Time.Before(candles[j].Time)
	})
	var live []openLot
	var finished []openLot
	ci := 0
	hitUntil := func(until time.Time) {
		for ci < len(candles) && !candles[ci].Time.After(until) {
			live = hitOpenLots(live, &finished, candles[ci])
			ci++
		}
	}
	var targetLong, targetShort float64
	storedN, evenN := 0, 0
	withOI := 0
	for i, t := range times {
		hitUntil(t)
		px := nearestHuntPrice(ser.Prices, t, stale)
		oi := nearestHuntSnapshot(ser.OI, t, stale)
		if px > 0 && oi > 0 {
			out.hasData[i] = true
			withOI++
			estLong, estShort, stored := nearestHuntBlendShares(ser.LongShort, t, stale*2)
			if stored {
				storedN++
			} else {
				evenN++
			}
			fund := nearestHuntFunding(ser.Funding, t, 8*time.Hour)
			longMix := TiltLeverageMix(DefaultHuntLeverageMix, fund > 1e-12)
			shortMix := TiltLeverageMix(DefaultHuntLeverageMix, fund < -1e-12)
			targetLong = syncOpenSide(&live, &finished, LiquidationSideLong, targetLong, oi*estLong, px, t, longMix, -1)
			targetShort = syncOpenSide(&live, &finished, LiquidationSideShort, targetShort, oi*estShort, px, t, shortMix, 1)
		}
		longN, shortN := openSideNotionals(live)
		out.cum[i] = OpenLevelPoint{Time: t, LongNotional: longN, ShortNotional: shortN, Delta: longN - shortN}
	}
	for ci < len(candles) {
		live = hitOpenLots(live, &finished, candles[ci])
		ci++
	}
	longN, shortN := openSideNotionals(live)
	out.cum[n-1] = OpenLevelPoint{Time: times[n-1], LongNotional: longN, ShortNotional: shortN, Delta: longN - shortN}
	hitLots, closedLots := 0, 0
	for _, lot := range finished {
		switch lot.status {
		case OpenLevelStatusHit:
			hitLots++
		case OpenLevelStatusClosed:
			closedLots++
		}
	}
	lots := make([]openLot, 0, len(live)+len(finished))
	lots = append(lots, live...)
	lots = append(lots, finished...)
	out.book.Segments = capOpenSegments(clusterOpenLots(ex, lots, openLevelRef(ser)))
	out.book.Cumulative = trimOpenCumulative(out.cum)
	out.book.OpenLong = longN
	out.book.OpenShort = shortN
	out.book.Delta = longN - shortN
	out.book.ColumnsWithOI = withOI
	out.book.Coverage = float64(withOI) / float64(n)
	out.book.HitLots = hitLots
	out.book.ClosedLots = closedLots
	out.book.AccountBlend = huntAccountBlendLabel(storedN, evenN)
	out.book.Definite = withOI > 0
	return out
}

func combineOpenLevels(times []time.Time, parts []openLevelWalk) OpenLevelBook {
	book := OpenLevelBook{
		Exchange:   "combined",
		Segments:   []OpenLevelSegment{},
		Cumulative: []OpenLevelPoint{},
	}
	if len(times) == 0 || len(parts) == 0 {
		return book
	}
	for _, p := range parts {
		if p.book.ColumnsWithOI == 0 {
			return book
		}
	}
	overlap := 0
	cum := make([]OpenLevelPoint, 0, len(times))
	for i, t := range times {
		ok := true
		var longN, shortN float64
		for _, p := range parts {
			if i >= len(p.hasData) || !p.hasData[i] {
				ok = false
				break
			}
			if i < len(p.cum) {
				longN += p.cum[i].LongNotional
				shortN += p.cum[i].ShortNotional
			}
		}
		if !ok {
			continue
		}
		overlap++
		cum = append(cum, OpenLevelPoint{Time: t, LongNotional: longN, ShortNotional: shortN, Delta: longN - shortN})
	}
	if overlap == 0 {
		return book
	}
	var segs []OpenLevelSegment
	stored, even := 0, 0
	hit, closed := 0, 0
	for _, p := range parts {
		segs = append(segs, p.book.Segments...)
		hit += p.book.HitLots
		closed += p.book.ClosedLots
		switch p.book.AccountBlend {
		case "stored":
			stored++
		case "even":
			even++
		case "mixed":
			stored++
			even++
		}
	}
	book.Segments = capOpenSegments(segs)
	book.Cumulative = trimOpenCumulative(cum)
	if len(book.Cumulative) == 0 {
		book.Segments = []OpenLevelSegment{}
		return book
	}
	last := book.Cumulative[len(book.Cumulative)-1]
	book.OpenLong = last.LongNotional
	book.OpenShort = last.ShortNotional
	book.Delta = last.Delta
	book.ColumnsWithOI = overlap
	book.Coverage = float64(overlap) / float64(len(times))
	book.HitLots = hit
	book.ClosedLots = closed
	book.AccountBlend = huntAccountBlendLabel(stored, even)
	book.Definite = true
	return book
}

func syncOpenSide(live *[]openLot, finished *[]openLot, side string, target, want, entry float64, at time.Time, mix []HuntLeverageBucket, sign float64) float64 {
	if math.IsNaN(want) || math.IsInf(want, 0) || want < 0 {
		want = 0
	}
	if !openLevelNotionalChanged(target, want) {
		return target
	}
	if want > target {
		if entry > 0 {
			addOpenLots(live, side, want-target, entry, at, mix, sign)
		}
		return want
	}
	cur := openSideNotional(*live, side)
	if cur > want {
		shrinkOpenLots(live, finished, side, cur-want, at)
	}
	return want
}

func openLevelNotionalChanged(cur, want float64) bool {
	diff := math.Abs(want - cur)
	if diff <= 1 {
		return false
	}
	base := cur
	if want > base {
		base = want
	}
	if base <= 0 {
		return diff > 0
	}
	return diff/base > openLevelChurnFrac
}

func addOpenLots(live *[]openLot, side string, delta, entry float64, at time.Time, mix []HuntLeverageBucket, sign float64) {
	if delta <= 0 || entry <= 0 {
		return
	}
	for _, b := range mix {
		if b.Leverage < 1 || b.Weight <= 0 {
			continue
		}
		n := delta * b.Weight
		if n <= 0 || math.IsNaN(n) {
			continue
		}
		dist := HuntLiqDistance(b.Leverage, HuntMaintenanceMargin)
		px := entry * (1 + sign*dist)
		if px <= 0 || math.IsNaN(px) {
			continue
		}
		*live = append(*live, openLot{
			side:     side,
			display:  ChartLeverageBucket(b.Leverage),
			price:    px,
			notional: n,
			openedAt: at,
			status:   OpenLevelStatusOpen,
		})
	}
}

func shrinkOpenLots(live *[]openLot, finished *[]openLot, side string, reduce float64, at time.Time) {
	if reduce <= 0 || live == nil {
		return
	}
	idx := make([]int, 0, len(*live))
	for i := range *live {
		if (*live)[i].side == side {
			idx = append(idx, i)
		}
	}
	sort.SliceStable(idx, func(a, b int) bool {
		return (*live)[idx[a]].openedAt.Before((*live)[idx[b]].openedAt)
	})
	drop := map[int]struct{}{}
	for _, i := range idx {
		if reduce <= 1e-6 {
			break
		}
		lot := (*live)[i]
		if lot.notional <= reduce+1e-9 {
			reduce -= lot.notional
			lot.status = OpenLevelStatusClosed
			lot.closedAt = at
			*finished = append(*finished, lot)
			drop[i] = struct{}{}
			continue
		}
		closed := lot
		closed.notional = reduce
		closed.status = OpenLevelStatusClosed
		closed.closedAt = at
		*finished = append(*finished, closed)
		(*live)[i].notional -= reduce
		reduce = 0
	}
	if len(drop) == 0 {
		return
	}
	kept := make([]openLot, 0, len(*live)-len(drop))
	for i, lot := range *live {
		if _, ok := drop[i]; ok {
			continue
		}
		kept = append(kept, lot)
	}
	*live = kept
}

func hitOpenLots(live []openLot, finished *[]openLot, c HuntHeatmapPricePoint) []openLot {
	if c.Time.IsZero() {
		return live
	}
	high, low, ok := openLevelCandleRange(c)
	if !ok {
		return live
	}
	kept := make([]openLot, 0, len(live))
	for _, lot := range live {
		hit := false
		if lot.side == LiquidationSideLong && low <= lot.price {
			hit = true
		}
		if lot.side == LiquidationSideShort && high >= lot.price {
			hit = true
		}
		if !hit {
			kept = append(kept, lot)
			continue
		}
		lot.status = OpenLevelStatusHit
		lot.closedAt = c.Time
		*finished = append(*finished, lot)
	}
	return kept
}

func openLevelCandleRange(p HuntHeatmapPricePoint) (high, low float64, ok bool) {
	px := p.Price
	if px <= 0 {
		px = p.High
	}
	if px <= 0 {
		px = p.Low
	}
	if px <= 0 {
		return 0, 0, false
	}
	high, low = p.High, p.Low
	if high <= 0 {
		high = px
	}
	if low <= 0 {
		low = px
	}
	if low > high {
		low, high = high, low
	}
	return high, low, true
}

func openSideNotional(lots []openLot, side string) float64 {
	var n float64
	for _, lot := range lots {
		if lot.side == side {
			n += lot.notional
		}
	}
	return n
}

func openSideNotionals(lots []openLot) (longN, shortN float64) {
	for _, lot := range lots {
		if lot.side == LiquidationSideShort {
			shortN += lot.notional
		} else {
			longN += lot.notional
		}
	}
	return longN, shortN
}

func openLevelRef(ser HuntHeatmapVenueSeries) float64 {
	var ref float64
	var latest time.Time
	for _, p := range ser.Prices {
		px := p.Price
		if px <= 0 {
			px = p.High
		}
		if px <= 0 {
			continue
		}
		if ref == 0 || !p.Time.Before(latest) {
			ref = px
			latest = p.Time
		}
	}
	return ref
}

type openClusterKey struct {
	exchange string
	side     string
	status   string
	lev      int
	bucket   int
}

func clusterOpenLots(ex string, lots []openLot, ref float64) []OpenLevelSegment {
	step := ref * openLevelClusterFrac
	if step <= 0 {
		step = 0.01
	}
	type acc struct {
		notional float64
		wpx      float64
		opened   time.Time
		closed   time.Time
	}
	grouped := map[openClusterKey]*acc{}
	for _, lot := range lots {
		if lot.notional <= 0 || lot.price <= 0 || math.IsNaN(lot.price) {
			continue
		}
		status := lot.status
		if status == "" {
			status = OpenLevelStatusOpen
		}
		key := openClusterKey{
			exchange: ex,
			side:     lot.side,
			status:   status,
			lev:      lot.display,
			bucket:   int(math.Round(lot.price / step)),
		}
		a := grouped[key]
		if a == nil {
			a = &acc{opened: lot.openedAt, closed: lot.closedAt}
			grouped[key] = a
		}
		a.notional += lot.notional
		a.wpx += lot.price * lot.notional
		if lot.openedAt.Before(a.opened) {
			a.opened = lot.openedAt
		}
		if status != OpenLevelStatusOpen && !lot.closedAt.IsZero() && (a.closed.IsZero() || lot.closedAt.After(a.closed)) {
			a.closed = lot.closedAt
		}
	}
	out := make([]OpenLevelSegment, 0, len(grouped))
	for key, a := range grouped {
		seg := OpenLevelSegment{
			Exchange: key.exchange,
			Side:     key.side,
			Leverage: key.lev,
			Price:    a.wpx / a.notional,
			Notional: a.notional,
			OpenedAt: a.opened,
			Status:   key.status,
		}
		if key.status != OpenLevelStatusOpen && !a.closed.IsZero() {
			closed := a.closed.UTC()
			seg.ClosedAt = &closed
		}
		out = append(out, seg)
	}
	return out
}

func capOpenSegments(in []OpenLevelSegment) []OpenLevelSegment {
	groups := map[string][]OpenLevelSegment{}
	for _, seg := range in {
		groups[seg.Status] = append(groups[seg.Status], seg)
	}
	limits := []struct {
		status string
		n      int
	}{
		{OpenLevelStatusOpen, openLevelMaxOpen},
		{OpenLevelStatusHit, openLevelMaxHit},
		{OpenLevelStatusClosed, openLevelMaxClosed},
	}
	out := make([]OpenLevelSegment, 0, len(in))
	for _, lim := range limits {
		rows := groups[lim.status]
		sort.Slice(rows, func(i, j int) bool {
			if rows[i].Notional == rows[j].Notional {
				return rows[i].Price > rows[j].Price
			}
			return rows[i].Notional > rows[j].Notional
		})
		if len(rows) > lim.n {
			rows = rows[:lim.n]
		}
		out = append(out, rows...)
	}
	if out == nil {
		return []OpenLevelSegment{}
	}
	return out
}

func trimOpenCumulative(in []OpenLevelPoint) []OpenLevelPoint {
	start := 0
	for start < len(in) && in[start].LongNotional == 0 && in[start].ShortNotional == 0 {
		start++
	}
	if start >= len(in) {
		return []OpenLevelPoint{}
	}
	out := make([]OpenLevelPoint, len(in)-start)
	copy(out, in[start:])
	return out
}
