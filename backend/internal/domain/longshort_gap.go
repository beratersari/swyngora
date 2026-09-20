package domain

import (
	"fmt"
	"math"
	"sort"
	"time"
)

const (
	LongShortGapLookback  = 24 * time.Hour
	LongShortGapMaxPoints = 288 // 24h of 5m samples
	LongShortGapMaxMoves  = 8
	longShortGapBucket    = 5 * time.Minute

	// Below this, crowd and top long-% are treated as even.
	longShortGapEvenPP = 1.0
	// 1pp change over a window is widened / narrowed.
	longShortGapWindowMovePP = 1.0
	// Larger jump to call out a widen / narrow / flip event.
	longShortGapEventPP = 3.0

	LongShortGapWidened  = "widened"
	LongShortGapNarrowed = "narrowed"
	LongShortGapFlipped  = "flipped"
	LongShortGapStable   = "stable"

	LongShortGapLeanTopLong  = "top_more_long"
	LongShortGapLeanTopShort = "top_more_short"
	LongShortGapLeanEven     = "even"
)

// LongShortGapWindows is 1h / 4h / 24h of top-minus-crowd long share.
var LongShortGapWindows = []struct {
	ID  string
	Dur time.Duration
}{
	{SnapshotWindow1h, time.Hour},
	{SnapshotWindow4h, 4 * time.Hour},
	{SnapshotWindow24h, 24 * time.Hour},
}

// LongShortGap is top-trader long % minus crowd long %, in percentage points.
// +10 means top traders are 10pp more long than other accounts (65% vs 55%).
type LongShortGap struct {
	Versus       string // top_positions | top_accounts
	CrowdLongPct string
	TopLongPct   string
	GapPct       string // signed percentage points (top − crowd)
	Lean         string // top_more_long | top_more_short | even
	Headline     string
	Detail       string
	Time         time.Time
	gap          float64
}

// LongShortGapPoint is the crowd-vs-top gap at one 5m sample.
type LongShortGapPoint struct {
	Time         time.Time
	Versus       string
	CrowdLongPct string
	TopLongPct   string
	GapPct       string
	Lean         string
}

// LongShortGapWindow is live gap vs a lookback.
type LongShortGapWindow struct {
	Window     string
	PastGapPct string
	ChangePct  string // signed pp, now − past
	PastLean   string
	Lean       string
	Kind       string // widened | narrowed | flipped | stable
	Complete   bool
	SampleTime time.Time
}

// LongShortGapMove is when the crowd-vs-top gap flipped side or moved a lot.
type LongShortGapMove struct {
	Time      time.Time
	Kind      string
	FromLean  string
	ToLean    string
	GapPct    string
	ChangePct string
	Summary   string
}

type longShortGapRaw struct {
	Time   time.Time
	Versus string
	Crowd  float64 // 0–1 long share
	Top    float64
}

func (r longShortGapRaw) gapPP() float64 {
	return (r.Top - r.Crowd) * 100
}

// AttachLongShortGap fills Gap / history / windows / moves from crowd vs top-trader layers.
func AttachLongShortGap(s *LongShortSeries) {
	if s == nil {
		return
	}
	versus, layer := longShortGapLayer(s)
	if layer == nil {
		s.Gap, s.GapHistory, s.GapWindows, s.GapMoves = nil, nil, nil, nil
		return
	}
	crowd := NormalizeLongShortPoint(s.Current)
	top := NormalizeLongShortPoint(layer.Current)
	at := crowd.Time
	if at.IsZero() {
		at = top.Time
	}
	live := longShortGapRaw{Time: at.UTC(), Versus: versus, Crowd: crowd.LongShare, Top: top.LongShare}
	g := formatLongShortGap(live)
	s.Gap = &g
	pts := alignLongShortGapPoints(s, layer, versus)
	if len(pts) == 0 || !pts[len(pts)-1].Time.Equal(TruncateToBucket(live.Time, longShortGapBucket)) {
		if !live.Time.IsZero() {
			pts = append(pts, live)
		}
	} else {
		pts[len(pts)-1] = live
	}
	if len(pts) > LongShortGapMaxPoints {
		pts = pts[len(pts)-LongShortGapMaxPoints:]
	}
	s.GapHistory = formatLongShortGapHistory(pts)
	s.GapWindows = longShortGapWindows(pts, live.Time, live.gapPP(), LongShortGapLean(live.gapPP()))
	s.GapMoves = detectLongShortGapMoves(pts)
}

// EnrichLongShortFromStore prepends older crowd / top-trader prints from durable samples.
func EnrichLongShortFromStore(series []*LongShortSeries, stored []FuturesSnapshot) {
	if len(series) == 0 || len(stored) == 0 {
		return
	}
	byEx := map[Exchange][]FuturesSnapshot{}
	for _, r := range stored {
		if r.LongShare <= 0 && r.ShortShare <= 0 && r.SizeLongShare <= 0 && r.TopAccLongShare <= 0 {
			continue
		}
		byEx[r.Exchange] = append(byEx[r.Exchange], r)
	}
	for _, s := range series {
		if s == nil {
			continue
		}
		rows := byEx[s.Exchange]
		if len(rows) == 0 {
			continue
		}
		crowdHave := longShortTimes(s)
		topPosHave := longShortTimes(s.TopPositions)
		topAccHave := longShortTimes(s.TopAccounts)
		cutoff := s.Current.Time
		for _, r := range rows {
			at := r.SampledAt.UTC()
			if at.IsZero() {
				continue
			}
			if !cutoff.IsZero() && !at.Before(cutoff) {
				continue
			}
			t := TruncateToBucket(at, longShortGapBucket)
			if (r.LongShare > 0 || r.ShortShare > 0) && !crowdHave[t.Unix()] {
				p := NormalizeLongShortPoint(LongShortPoint{
					Time: at, LongShare: r.LongShare, ShortShare: r.ShortShare, Ratio: r.Ratio,
				})
				s.History = append(s.History, p)
				crowdHave[t.Unix()] = true
			}
			if r.SizeLongShare > 0 && s.TopPositions != nil {
				appendLongShortLayerPoint(s.TopPositions, at, r.SizeLongShare, topPosHave)
			}
			if r.TopAccLongShare > 0 && s.TopAccounts != nil {
				appendLongShortLayerPoint(s.TopAccounts, at, r.TopAccLongShare, topAccHave)
			}
		}
		s.History = SortLongShortNewestFirst(s.History)
		if s.TopPositions != nil {
			s.TopPositions.History = SortLongShortNewestFirst(s.TopPositions.History)
		}
		if s.TopAccounts != nil {
			s.TopAccounts.History = SortLongShortNewestFirst(s.TopAccounts.History)
		}
	}
}

func appendLongShortLayerPoint(layer *LongShortSeries, at time.Time, longShare float64, have map[int64]bool) {
	if layer == nil {
		return
	}
	if have == nil {
		have = map[int64]bool{}
	}
	t := TruncateToBucket(at.UTC(), longShortGapBucket)
	if have[t.Unix()] {
		return
	}
	p := NormalizeLongShortPoint(LongShortPoint{
		Time: at.UTC(), LongShare: longShare, ShortShare: 1 - longShare,
	})
	if !layer.Current.Time.IsZero() && !at.Before(layer.Current.Time) {
		return
	}
	layer.History = append(layer.History, p)
	have[t.Unix()] = true
}

func longShortTimes(s *LongShortSeries) map[int64]bool {
	out := map[int64]bool{}
	if s == nil {
		return out
	}
	add := func(p LongShortPoint) {
		if p.Time.IsZero() {
			return
		}
		out[TruncateToBucket(p.Time.UTC(), longShortGapBucket).Unix()] = true
	}
	add(s.Current)
	for _, p := range s.History {
		add(p)
	}
	return out
}

func longShortGapLayer(s *LongShortSeries) (versus string, layer *LongShortSeries) {
	if s == nil {
		return "", nil
	}
	if s.TopPositions != nil && longShortHasShare(s.TopPositions.Current) {
		return LongShortKindTopPositions, s.TopPositions
	}
	if s.TopAccounts != nil && longShortHasShare(s.TopAccounts.Current) {
		return LongShortKindTopAccounts, s.TopAccounts
	}
	return "", nil
}

func longShortHasShare(p LongShortPoint) bool {
	p = NormalizeLongShortPoint(p)
	return p.LongShare > 0 || p.ShortShare > 0
}

func alignLongShortGapPoints(crowd, top *LongShortSeries, versus string) []longShortGapRaw {
	if crowd == nil || top == nil {
		return nil
	}
	byTop := map[int64]float64{}
	for _, p := range longShortAllPoints(top) {
		if p.Time.IsZero() || !longShortHasShare(p) {
			continue
		}
		t := TruncateToBucket(p.Time.UTC(), longShortGapBucket)
		byTop[t.Unix()] = NormalizeLongShortPoint(p).LongShare
	}
	seen := map[int64]struct{}{}
	var out []longShortGapRaw
	for _, p := range longShortAllPoints(crowd) {
		if p.Time.IsZero() || !longShortHasShare(p) {
			continue
		}
		t := TruncateToBucket(p.Time.UTC(), longShortGapBucket)
		topShare, ok := byTop[t.Unix()]
		if !ok {
			continue
		}
		if _, dup := seen[t.Unix()]; dup {
			continue
		}
		seen[t.Unix()] = struct{}{}
		out = append(out, longShortGapRaw{
			Time: t, Versus: versus,
			Crowd: NormalizeLongShortPoint(p).LongShare, Top: topShare,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	from := time.Time{}
	if len(out) > 0 {
		from = out[len(out)-1].Time.Add(-LongShortGapLookback)
	}
	trimmed := out[:0]
	for _, p := range out {
		if !from.IsZero() && p.Time.Before(from) {
			continue
		}
		trimmed = append(trimmed, p)
	}
	return trimmed
}

func longShortAllPoints(s *LongShortSeries) []LongShortPoint {
	if s == nil {
		return nil
	}
	out := make([]LongShortPoint, 0, 1+len(s.History))
	if !s.Current.Time.IsZero() || longShortHasShare(s.Current) {
		out = append(out, s.Current)
	}
	out = append(out, s.History...)
	return out
}

func formatLongShortGap(raw longShortGapRaw) LongShortGap {
	gap := raw.gapPP()
	lean := LongShortGapLean(gap)
	out := LongShortGap{
		Versus:       raw.Versus,
		CrowdLongPct: formatFixed(raw.Crowd*100, 2),
		TopLongPct:   formatFixed(raw.Top*100, 2),
		GapPct:       FormatSignedPct(gap),
		Lean:         lean,
		Headline:     longShortGapHeadline(lean, gap, raw.Versus),
		Detail:       longShortGapDetail(raw, lean, gap),
		gap:          gap,
	}
	if !raw.Time.IsZero() {
		out.Time = raw.Time.UTC()
	}
	return out
}

func formatLongShortGapHistory(pts []longShortGapRaw) []LongShortGapPoint {
	if len(pts) == 0 {
		return nil
	}
	out := make([]LongShortGapPoint, 0, len(pts))
	for i := len(pts) - 1; i >= 0; i-- {
		p := pts[i]
		gap := p.gapPP()
		row := LongShortGapPoint{
			Versus:       p.Versus,
			CrowdLongPct: formatFixed(p.Crowd*100, 2),
			TopLongPct:   formatFixed(p.Top*100, 2),
			GapPct:       FormatSignedPct(gap),
			Lean:         LongShortGapLean(gap),
		}
		if !p.Time.IsZero() {
			row.Time = p.Time.UTC()
		}
		out = append(out, row)
	}
	return out
}

func LongShortGapLean(gapPP float64) string {
	switch {
	case gapPP >= longShortGapEvenPP:
		return LongShortGapLeanTopLong
	case gapPP <= -longShortGapEvenPP:
		return LongShortGapLeanTopShort
	default:
		return LongShortGapLeanEven
	}
}

func longShortGapHeadline(lean string, gapPP float64, versus string) string {
	who := "Top traders"
	if versus == LongShortKindTopAccounts {
		who = "Top-trader accounts"
	}
	abs := formatFixed(math.Abs(gapPP), 2)
	switch lean {
	case LongShortGapLeanTopLong:
		return fmt.Sprintf("%s are %spp more long than other accounts.", who, abs)
	case LongShortGapLeanTopShort:
		return fmt.Sprintf("%s are %spp more short than other accounts.", who, abs)
	default:
		return fmt.Sprintf("%s and other accounts are within 1pp on long share.", who)
	}
}

func longShortGapDetail(raw longShortGapRaw, lean string, gapPP float64) string {
	who := "top-trader size"
	if raw.Versus == LongShortKindTopAccounts {
		who = "top-trader accounts"
	}
	base := fmt.Sprintf("Other accounts %.0f%% long; %s %.0f%% long (gap %s pp).",
		raw.Crowd*100, who, raw.Top*100, FormatSignedPct(gapPP))
	switch lean {
	case LongShortGapLeanTopLong:
		return base + " Bigger books lean more long than the crowd."
	case LongShortGapLeanTopShort:
		return base + " Bigger books lean more short than the crowd."
	default:
		return base
	}
}

func findLongShortGapSample(hist []longShortGapRaw, target time.Time, slack time.Duration) (longShortGapRaw, bool) {
	var best longShortGapRaw
	found := false
	for _, p := range hist {
		if p.Time.IsZero() || p.Time.After(target) {
			continue
		}
		if !found || p.Time.After(best.Time) {
			best = p
			found = true
		}
	}
	if !found {
		return longShortGapRaw{}, false
	}
	return best, !best.Time.Before(target.Add(-slack))
}

func longShortGapWindows(hist []longShortGapRaw, now time.Time, liveGap float64, liveLean string) []LongShortGapWindow {
	out := make([]LongShortGapWindow, 0, len(LongShortGapWindows))
	for _, w := range LongShortGapWindows {
		row := LongShortGapWindow{Window: w.ID, Lean: liveLean}
		past, complete := findLongShortGapSample(hist, now.Add(-w.Dur), OpenInterestSampleSlack(w.Dur))
		if past.Time.IsZero() {
			out = append(out, row)
			continue
		}
		pastGap := past.gapPP()
		row.PastGapPct = FormatSignedPct(pastGap)
		row.ChangePct = FormatSignedPct(liveGap - pastGap)
		row.PastLean = LongShortGapLean(pastGap)
		row.Kind = longShortGapKind(pastGap, liveGap, longShortGapWindowMovePP)
		row.Complete = complete
		row.SampleTime = past.Time.UTC()
		out = append(out, row)
	}
	return out
}

func detectLongShortGapMoves(hist []longShortGapRaw) []LongShortGapMove {
	if len(hist) < 2 {
		return nil
	}
	var out []LongShortGapMove
	for i := 1; i < len(hist); i++ {
		prev, cur := hist[i-1], hist[i]
		kind := longShortGapKind(prev.gapPP(), cur.gapPP(), longShortGapEventPP)
		if kind == LongShortGapStable {
			continue
		}
		from, to := LongShortGapLean(prev.gapPP()), LongShortGapLean(cur.gapPP())
		out = append(out, LongShortGapMove{
			Time:      cur.Time.UTC(),
			Kind:      kind,
			FromLean:  from,
			ToLean:    to,
			GapPct:    FormatSignedPct(cur.gapPP()),
			ChangePct: FormatSignedPct(cur.gapPP() - prev.gapPP()),
			Summary:   longShortGapMoveSummary(kind, from, to, cur.gapPP()),
		})
	}
	if len(out) > LongShortGapMaxMoves {
		out = out[len(out)-LongShortGapMaxMoves:]
	}
	return out
}

func longShortGapKind(past, now, thresh float64) string {
	pastL, nowL := LongShortGapLean(past), LongShortGapLean(now)
	if pastL != nowL && pastL != LongShortGapLeanEven && nowL != LongShortGapLeanEven {
		return LongShortGapFlipped
	}
	dAbs := math.Abs(now) - math.Abs(past)
	switch {
	case dAbs > thresh:
		return LongShortGapWidened
	case dAbs < -thresh:
		return LongShortGapNarrowed
	default:
		return LongShortGapStable
	}
}

func longShortGapMoveSummary(kind, from, to string, gapPP float64) string {
	switch kind {
	case LongShortGapFlipped:
		return fmt.Sprintf("Crowd vs top-trader gap flipped from %s to %s.", longShortGapLeanPhrase(from), longShortGapLeanPhrase(to))
	case LongShortGapWidened:
		return fmt.Sprintf("Crowd vs top-trader gap widened to %s pp.", FormatSignedPct(gapPP))
	case LongShortGapNarrowed:
		return fmt.Sprintf("Crowd vs top-trader gap narrowed to %s pp.", FormatSignedPct(gapPP))
	default:
		return ""
	}
}

func longShortGapLeanPhrase(lean string) string {
	switch lean {
	case LongShortGapLeanTopLong:
		return "top more long"
	case LongShortGapLeanTopShort:
		return "top more short"
	default:
		return "even"
	}
}
