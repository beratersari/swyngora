package domain

import (
	"fmt"
	"math"
	"sort"
	"time"
)

const (
	DefaultLongShortHistoryLimit = 24
	MaxLongShortHistoryLimit     = 100
	LongShortPeriod5m            = "5m"
	LongShortKindAccounts        = "accounts"      // all accounts, count not size
	LongShortKindTopAccounts     = "top_accounts"  // Binance top 20% by margin, account count
	LongShortKindTopPositions    = "top_positions" // Binance top 20% by margin, position size
)

// LongShortPoint is one long/short account-ratio sample.
type LongShortPoint struct {
	Time       time.Time
	LongShare  float64 // 0–1 share of accounts that are long
	ShortShare float64 // 0–1
	Ratio      float64 // long/short; 1 = even
}

// LongShortSeries is one venue's latest sample plus recent history.
type LongShortSeries struct {
	Exchange Exchange
	Symbol   string
	Kind     string // accounts | top_accounts | top_positions
	Period   string // 5m
	Current  LongShortPoint
	History  []LongShortPoint // newest first, excluding current
	// TopAccounts / TopPositions are Binance-only extra layers on the
	// global account series. Nil on Bybit or when the extra feed fails.
	TopAccounts  *LongShortSeries
	TopPositions *LongShortSeries
	Split        LongShortSplit
}

// LongShortLevel is a formatted ratio print.
type LongShortLevel struct {
	Time      time.Time
	LongPct   string
	ShortPct  string
	Ratio     string
	Bias      string // long | short | balanced
	LongShare string // 0–1 decimal, for AI
}

// LongShortVenueSnap is one venue inside a snapshot.
type LongShortVenueSnap struct {
	Exchange string
	Kind     string
	Period   string
	Current  LongShortLevel
	Change   string // signed ratio change vs previous print
	History  []LongShortLevel
	// TopAccounts is Binance top-trader account-count ratio (nil elsewhere).
	TopAccounts *LongShortLevel
	// TopPositions is Binance top-trader position-size ratio (nil elsewhere).
	TopPositions *LongShortLevel
	Split        LongShortSplit
}

// LongShortSplit is how the crowd (all accounts) lines up with top traders.
type LongShortSplit struct {
	Alignment    string `json:"alignment,omitempty"` // same | opposite | mixed | unknown
	Important    bool   `json:"important,omitempty"`
	General      string `json:"general,omitempty"`      // long | short | balanced
	TopAccounts  string `json:"topAccounts,omitempty"`  // long | short | balanced
	TopPositions string `json:"topPositions,omitempty"` // long | short | balanced
	SizeBias     string `json:"sizeBias,omitempty"`     // lean of bigger books
	Headline     string `json:"headline,omitempty"`
	Detail       string `json:"detail,omitempty"`
}

// LongShortSnapshot is current long/short ratio plus recent 5m history.
type LongShortSnapshot struct {
	Symbol     string
	Exchange   string // binance | bybit | all
	Kind       string
	Period     string
	Current    *LongShortLevel
	Venues     []LongShortVenueSnap
	History    []LongShortLevel
	AsOf       time.Time
	VenueCount int
}

// ClampLongShortHistoryLimit bounds history size.
func ClampLongShortHistoryLimit(n int) int {
	if n <= 0 {
		return DefaultLongShortHistoryLimit
	}
	if n > MaxLongShortHistoryLimit {
		return MaxLongShortHistoryLimit
	}
	return n
}

// LongShortRatioFromShares is long/short; 0 short-share → +Inf treated as 0 invalid.
func LongShortRatioFromShares(longShare, shortShare float64) float64 {
	if shortShare > 1e-12 {
		return longShare / shortShare
	}
	if longShare > 1e-12 {
		return 99
	}
	return 0
}

// LongShortBias labels crowding. 5% band around 1.0 is balanced.
func LongShortBias(ratio float64) string {
	switch {
	case ratio >= 1.05:
		return "long"
	case ratio > 0 && ratio <= 0.95:
		return "short"
	default:
		return "balanced"
	}
}

// FormatLongShortLevel renders one sample.
func FormatLongShortLevel(p LongShortPoint) LongShortLevel {
	out := LongShortLevel{
		LongPct:   formatFixed(p.LongShare*100, 2),
		ShortPct:  formatFixed(p.ShortShare*100, 2),
		Ratio:     formatFixed(p.Ratio, 4),
		Bias:      LongShortBias(p.Ratio),
		LongShare: formatFixed(p.LongShare, 4),
	}
	if !p.Time.IsZero() {
		out.Time = p.Time.UTC()
	}
	return out
}

// SortLongShortNewestFirst sorts and dedups by timestamp (keeps last).
func SortLongShortNewestFirst(hist []LongShortPoint) []LongShortPoint {
	if len(hist) == 0 {
		return hist
	}
	sort.SliceStable(hist, func(i, j int) bool {
		return hist[i].Time.After(hist[j].Time)
	})
	out := hist[:0]
	for _, p := range hist {
		if p.Time.IsZero() {
			continue
		}
		if len(out) > 0 && out[len(out)-1].Time.Equal(p.Time) {
			out[len(out)-1] = p
			continue
		}
		out = append(out, p)
	}
	return out
}

func venueLongShortSnap(s *LongShortSeries) LongShortVenueSnap {
	out := LongShortVenueSnap{
		Exchange: string(s.Exchange),
		Kind:     s.Kind,
		Period:   s.Period,
		Current:  FormatLongShortLevel(s.Current),
		History:  make([]LongShortLevel, 0, len(s.History)),
		Split:    s.Split,
	}
	if len(s.History) > 0 {
		out.Change = FormatSignedQty(s.Current.Ratio - s.History[0].Ratio)
	}
	for _, p := range s.History {
		out.History = append(out.History, FormatLongShortLevel(p))
	}
	if s.TopAccounts != nil {
		lvl := FormatLongShortLevel(s.TopAccounts.Current)
		out.TopAccounts = &lvl
	}
	if s.TopPositions != nil {
		lvl := FormatLongShortLevel(s.TopPositions.Current)
		out.TopPositions = &lvl
	}
	return out
}

func oppositeLongShortBias(a, b string) bool {
	return (a == "long" && b == "short") || (a == "short" && b == "long")
}

func layerPoint(s *LongShortSeries) LongShortPoint {
	if s == nil {
		return LongShortPoint{}
	}
	return NormalizeLongShortPoint(s.Current)
}

// AttachLongShortSplit fills Split from the global print plus any top-trader layers.
func AttachLongShortSplit(s *LongShortSeries) {
	if s == nil {
		return
	}
	s.Split = CompareLongShortLayers(s.Current, layerPoint(s.TopAccounts), layerPoint(s.TopPositions))
}

// CompareLongShortLayers contrasts all-account crowding with top-trader
// account count and top-trader position size.
func CompareLongShortLayers(general, topAcc, topPos LongShortPoint) LongShortSplit {
	g := NormalizeLongShortPoint(general)
	a := NormalizeLongShortPoint(topAcc)
	p := NormalizeLongShortPoint(topPos)
	out := LongShortSplit{General: LongShortBias(g.Ratio)}
	if a.Ratio > 0 {
		out.TopAccounts = LongShortBias(a.Ratio)
	}
	if p.Ratio > 0 {
		out.TopPositions = LongShortBias(p.Ratio)
	}
	if out.TopAccounts == "" && out.TopPositions == "" {
		out.Alignment = AlignUnknown
		return out
	}
	out.SizeBias = out.TopPositions
	if out.SizeBias == "" {
		out.SizeBias = out.TopAccounts
	}
	switch {
	case oppositeLongShortBias(out.General, out.TopPositions):
		out.Alignment = AlignOpposite
		out.Important = true
		out.Headline = "Crowd accounts and top-trader positions lean opposite."
		out.Detail = fmt.Sprintf("All accounts are %s (%.0f%% long) while top-trader positions are %s (%.0f%% long). Size is with the bigger books, not the crowd.",
			out.General, g.LongShare*100, out.TopPositions, p.LongShare*100)
	case oppositeLongShortBias(out.General, out.TopAccounts) && out.TopPositions == "":
		out.Alignment = AlignOpposite
		out.Important = true
		out.Headline = "Crowd accounts and top-trader accounts lean opposite."
		out.Detail = fmt.Sprintf("All accounts are %s (%.0f%% long) while top-trader accounts are %s (%.0f%% long).",
			out.General, g.LongShare*100, out.TopAccounts, a.LongShare*100)
	case oppositeLongShortBias(out.TopAccounts, out.TopPositions):
		out.Alignment = AlignMixed
		out.Headline = "Top traders: more accounts on one side, more size on the other."
		out.Detail = fmt.Sprintf("Top-trader accounts are %s (%.0f%% long) but their positions are %s (%.0f%% long).",
			out.TopAccounts, a.LongShare*100, out.TopPositions, p.LongShare*100)
	case out.General == out.SizeBias || out.General == "balanced" || out.SizeBias == "balanced":
		out.Alignment = AlignSame
		out.Headline = "Crowd and top traders lean the same way."
		out.Detail = fmt.Sprintf("All accounts %s (%.0f%% long); top-trader size %s.",
			out.General, g.LongShare*100, out.SizeBias)
	default:
		out.Alignment = AlignMixed
		out.Headline = "Crowd and top traders are not a clean match."
		out.Detail = fmt.Sprintf("All accounts %s; top-trader size %s.", out.General, out.SizeBias)
	}
	return out
}

// HuntBlendShares is the long/short mix the live hunt map uses: all-account
// count unless crowd vs top-trader size are opposite, then size wins, then
// BlendAccountShare. Matches BuildHuntVenue.
func HuntBlendShares(accountLong, sizeLong float64) (estLong, estShort float64) {
	ser := &LongShortSeries{
		Current: LongShortPoint{LongShare: accountLong, ShortShare: 1 - accountLong},
	}
	if sizeLong > 0 {
		ser.TopPositions = &LongShortSeries{
			Kind:    LongShortKindTopPositions,
			Current: LongShortPoint{LongShare: sizeLong, ShortShare: 1 - sizeLong},
		}
	}
	AttachLongShortSplit(ser)
	blend := SizeWeightedLongShare(ser)
	if blend <= 0 {
		if accountLong <= 0 && sizeLong <= 0 {
			return 0.5, 0.5
		}
		if accountLong > 0 {
			blend = accountLong
		} else {
			blend = sizeLong
		}
	}
	return BlendAccountShare(blend)
}

// HuntBlendFromSnapshot rebuilds the live hunt mix from a stored LS print.
func HuntBlendFromSnapshot(ls *FuturesSnapshot) (estLong, estShort float64) {
	if ls == nil {
		return 0.5, 0.5
	}
	return HuntBlendShares(ls.LongShare, ls.SizeLongShare)
}

// HuntBlendKind is crowd or top_positions — what the frozen map used.
func HuntBlendKind(accountLong, sizeLong float64) string {
	if sizeLong <= 0 {
		return "crowd"
	}
	ser := &LongShortSeries{
		Current: LongShortPoint{LongShare: accountLong, ShortShare: 1 - accountLong},
		TopPositions: &LongShortSeries{
			Kind:    LongShortKindTopPositions,
			Current: LongShortPoint{LongShare: sizeLong, ShortShare: 1 - sizeLong},
		},
	}
	AttachLongShortSplit(ser)
	if ser.Split.Alignment == AlignOpposite {
		return "top_positions"
	}
	return "crowd"
}

// SizeWeightedLongShare is the long share analysis should use for "who is
// actually positioned." When the crowd and top-trader size lean opposite,
// size wins. Otherwise the all-account print is kept.
func SizeWeightedLongShare(s *LongShortSeries) float64 {
	if s == nil {
		return 0
	}
	g := NormalizeLongShortPoint(s.Current)
	if s.Split.Alignment == "" {
		AttachLongShortSplit(s)
	}
	if s.Split.Alignment == AlignOpposite && s.TopPositions != nil {
		p := NormalizeLongShortPoint(s.TopPositions.Current)
		if p.LongShare > 0 || p.ShortShare > 0 {
			return p.LongShare
		}
	}
	return g.LongShare
}

// BuildLongShortSnapshot folds one or more venue series.
func BuildLongShortSnapshot(exchange, symbol string, series []*LongShortSeries, now time.Time) *LongShortSnapshot {
	symbol = NormalizeLiquidationSymbol(symbol)
	now = now.UTC()
	out := &LongShortSnapshot{
		Symbol:   symbol,
		Exchange: exchange,
		Kind:     LongShortKindAccounts,
		Period:   LongShortPeriod5m,
		AsOf:     now,
		Venues:   []LongShortVenueSnap{},
		History:  []LongShortLevel{},
	}
	clean := make([]*LongShortSeries, 0, len(series))
	for _, s := range series {
		if s == nil {
			continue
		}
		s.History = SortLongShortNewestFirst(s.History)
		if s.Kind == "" {
			s.Kind = LongShortKindAccounts
		}
		if s.Period == "" {
			s.Period = LongShortPeriod5m
		}
		if s.Current.Time.IsZero() && len(s.History) > 0 {
			s.Current = s.History[0]
			s.History = s.History[1:]
		}
		AttachLongShortSplit(s)
		clean = append(clean, s)
	}
	sort.Slice(clean, func(i, j int) bool {
		return string(clean[i].Exchange) < string(clean[j].Exchange)
	})
	out.VenueCount = len(clean)
	for _, s := range clean {
		out.Venues = append(out.Venues, venueLongShortSnap(s))
	}
	if len(clean) == 1 {
		cur := FormatLongShortLevel(clean[0].Current)
		out.Current = &cur
		out.History = append([]LongShortLevel(nil), out.Venues[0].History...)
		out.Kind = clean[0].Kind
		out.Period = clean[0].Period
	}
	return out
}

// NormalizeLongShortPoint fills ratio from shares when missing.
func NormalizeLongShortPoint(p LongShortPoint) LongShortPoint {
	if p.Ratio <= 0 && (p.LongShare > 0 || p.ShortShare > 0) {
		p.Ratio = LongShortRatioFromShares(p.LongShare, p.ShortShare)
	}
	if p.LongShare+p.ShortShare > 0 {
		sum := p.LongShare + p.ShortShare
		if math.Abs(sum-1) > 0.02 && sum > 0 {
			p.LongShare /= sum
			p.ShortShare /= sum
		}
	}
	return p
}
