package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	AlignSame     = "same"
	AlignOpposite = "opposite"
	AlignMixed    = "mixed"
	AlignUnknown  = "unknown"

	LeanBullish = "bullish" // long buildup or short covering
	LeanBearish = "bearish" // short buildup or long unwinding
	LeanMixed   = "mixed"
)

// VenueReading is one exchange's print inside a compared metric.
type VenueReading struct {
	Exchange string `json:"exchange"`
	Value    string `json:"value"`
}

// VenueSignalDiff is one compared metric across Binance, Bybit, and OKX.
type VenueSignalDiff struct {
	Metric       string         `json:"metric"` // oi_change | funding | crowding | positioning
	Label        string         `json:"label"`
	Readings     []VenueReading `json:"readings"`
	Same         []string       `json:"same,omitempty"`
	Different    []string       `json:"different,omitempty"`
	Binance      string         `json:"binance,omitempty"`
	Bybit        string         `json:"bybit,omitempty"`
	Alignment    string         `json:"alignment"` // same | opposite | mixed | unknown
	Important    bool           `json:"important"`
	WhyItMatters string         `json:"whyItMatters"`
}

// VenueLeanView is one exchange's direction.
type VenueLeanView struct {
	Exchange string `json:"exchange"`
	Lean     string `json:"lean"`
	Regime   string `json:"regime"`
	OIChange string `json:"oiChange,omitempty"`
}

// VenueLeanGroup lists exchanges that share one lean.
type VenueLeanGroup struct {
	Lean      string   `json:"lean"`
	Exchanges []string `json:"exchanges"`
}

// VenueDivergenceReport compares Binance, Bybit, and OKX for one coin.
type VenueDivergenceReport struct {
	Symbol          string            `json:"symbol"`
	AsOf            time.Time         `json:"asOf"`
	Alignment       string            `json:"alignment"` // same | opposite | mixed | unknown
	Venues          []VenueLeanView   `json:"venues"`
	Groups          []VenueLeanGroup  `json:"groups"`
	Outliers        []string          `json:"outliers,omitempty"`
	BinanceLean     string            `json:"binanceLean,omitempty"`
	BybitLean       string            `json:"bybitLean,omitempty"`
	OKXLean         string            `json:"okxLean,omitempty"`
	Important       bool              `json:"important"`
	Title           string            `json:"title"`
	Summary         string            `json:"summary"`
	Diffs           []VenueSignalDiff `json:"diffs"`
	BinanceRegime   string            `json:"binanceRegime,omitempty"`
	BybitRegime     string            `json:"bybitRegime,omitempty"`
	OKXRegime       string            `json:"okxRegime,omitempty"`
	BinanceOIChange string            `json:"binanceOiChange,omitempty"`
	BybitOIChange   string            `json:"bybitOiChange,omitempty"`
	OKXOIChange     string            `json:"okxOiChange,omitempty"`
	Note            string            `json:"note,omitempty"`
}

// VenueLeanFromRegime maps positioning to a market lean.
func VenueLeanFromRegime(regime string) string {
	switch regime {
	case RegimeLongBuildup, RegimeShortCovering:
		return LeanBullish
	case RegimeShortBuildup, RegimeLongUnwinding:
		return LeanBearish
	default:
		return LeanMixed
	}
}

func leanFromCrowdingAndOI(crowded, oiDir string) string {
	if oiDir == "up" && crowded == SqueezeSideLong {
		return LeanBullish
	}
	if oiDir == "up" && crowded == SqueezeSideShort {
		return LeanBearish
	}
	if oiDir == "down" && crowded == SqueezeSideLong {
		return LeanBearish // longs leaving
	}
	if oiDir == "down" && crowded == SqueezeSideShort {
		return LeanBullish // shorts leaving
	}
	return LeanMixed
}

// CompareVenuePositioning contrasts every futures venue that printed.
// Long is not used here: lean is bullish or bearish. Venues that share a lean
// land in the same group. A smaller group is the outlier.
func CompareVenuePositioning(symbol string, venues []PositioningVenueReport, now time.Time) VenueDivergenceReport {
	symbol = NormalizeLiquidationSymbol(symbol)
	usable := make([]PositioningVenueReport, 0, len(venues))
	for _, v := range venues {
		if v.Error != "" || v.Exchange == "" {
			continue
		}
		usable = append(usable, v)
	}
	sort.SliceStable(usable, func(i, j int) bool {
		return string(usable[i].Exchange) < string(usable[j].Exchange)
	})
	out := VenueDivergenceReport{
		Symbol: symbol,
		AsOf:   now.UTC(),
		Venues: []VenueLeanView{},
		Groups: []VenueLeanGroup{},
		Diffs:  []VenueSignalDiff{},
		Note:   "Compares Binance USD-M, Bybit linear, and OKX USDT-swap using the same positioning, funding, and account long/short we already publish. A venue with no account long/short is left out of the crowding row. Informational only — not financial advice.",
	}
	if len(usable) < 2 {
		out.Alignment = AlignUnknown
		out.Title = "Not enough venue data to compare"
		out.Summary = "Need at least two of Binance, Bybit, and OKX for this coin."
		return out
	}

	views := make([]venueSnap, 0, len(usable))
	for _, v := range usable {
		views = append(views, snapVenue(v))
	}
	for _, s := range views {
		out.Venues = append(out.Venues, VenueLeanView{
			Exchange: string(s.ex), Lean: s.lean, Regime: s.regime, OIChange: s.oiChange,
		})
		switch s.ex {
		case ExchangeBinance:
			out.BinanceLean, out.BinanceRegime, out.BinanceOIChange = s.lean, s.regime, s.oiChange
		case ExchangeBybit:
			out.BybitLean, out.BybitRegime, out.BybitOIChange = s.lean, s.regime, s.oiChange
		case ExchangeOKX:
			out.OKXLean, out.OKXRegime, out.OKXOIChange = s.lean, s.regime, s.oiChange
		}
	}

	out.Diffs = append(out.Diffs,
		compareOIViews(views),
		compareFundingViews(views),
		compareCrowdingViews(views),
		compareRegimeViews(views),
	)
	out.Groups, out.Outliers, out.Alignment, out.Important = leanGroups(views)
	out.Title, out.Summary = divergenceCopy(out)
	return out
}

type venueSnap struct {
	ex        Exchange
	lean      string
	regime    string
	label     string
	oiDir     string
	oiChange  string
	oiWindow  string
	funding   float64
	payer     string
	longShare float64
	crowd     string
	hasCrowd  bool
}

func snapVenue(v PositioningVenueReport) venueSnap {
	win := pickCompareWindow(v)
	lean := VenueLeanFromRegime(v.Regime)
	crowd := ""
	has := hasAccountCrowd(v)
	if has {
		crowd = CrowdedSideFromShares(v.LongShare)
	}
	if lean == LeanMixed && has {
		lean = leanFromCrowdingAndOI(crowd, win.OIDir)
	}
	label := v.Label
	if label == "" {
		label = RegimeLabel(v.Regime)
	}
	return venueSnap{
		ex: v.Exchange, lean: lean, regime: v.Regime, label: label,
		oiDir: win.OIDir, oiChange: FormatSignedPct(win.OIChangePct), oiWindow: win.Window,
		funding: v.FundingRate, payer: FundingPayer(v.FundingRate),
		longShare: v.LongShare, crowd: crowd, hasCrowd: has,
	}
}

func hasAccountCrowd(v PositioningVenueReport) bool {
	if v.CrowdingSplit.General != "" || v.CrowdingSplit.Alignment != "" {
		return true
	}
	if v.LongShare == 0.5 && v.ShortShare == 0.5 {
		return false
	}
	return v.LongShare > 0 || v.ShortShare > 0
}

func pickCompareWindow(v PositioningVenueReport) PositioningWindow {
	if v.Primary.Window != "" {
		return v.Primary
	}
	for _, w := range v.Windows {
		if w.Window == "4h" {
			return w
		}
	}
	if len(v.Windows) > 0 {
		return v.Windows[0]
	}
	return PositioningWindow{}
}

func compareOIViews(views []venueSnap) VenueSignalDiff {
	window := "4h"
	rows := make([]keyedReading, 0, len(views))
	for _, v := range views {
		if v.oiWindow != "" {
			window = v.oiWindow
		}
		text := fmt.Sprintf("%s (%s)", v.oiDir, v.oiChange)
		key := ""
		if v.oiDir == "up" || v.oiDir == "down" {
			key = v.oiDir
		}
		rows = append(rows, keyedReading{ex: v.ex, key: key, text: text})
	}
	d := finishDiff("oi_change", "Open interest ("+window+")", rows, true)
	switch d.Alignment {
	case AlignSame:
		d.WhyItMatters = "These venues are changing open interest in the same direction."
	case AlignOpposite:
		d.WhyItMatters = "Open interest is rising on some venues and falling on others, so leverage is not one global book."
	case AlignUnknown:
		d.WhyItMatters = "Not enough venues printed an open-interest direction."
	default:
		d.WhyItMatters = "At least one venue’s open interest is flat, so there is no clean split yet."
	}
	return d
}

func compareFundingViews(views []venueSnap) VenueSignalDiff {
	rows := make([]keyedReading, 0, len(views))
	for _, v := range views {
		dec, _ := FormatFundingRate(v.funding)
		key := ""
		if v.payer == "long" || v.payer == "short" {
			key = v.payer
		}
		rows = append(rows, keyedReading{ex: v.ex, key: key, text: fmt.Sprintf("%s (%s)", v.payer, dec)})
	}
	d := finishDiff("funding", "Funding payer", rows, true)
	switch d.Alignment {
	case AlignSame:
		d.WhyItMatters = "The same side pays funding on the venues that printed a rate."
	case AlignOpposite:
		d.WhyItMatters = "Opposite funding means one book is long-crowded while another is short-crowded."
	case AlignUnknown:
		d.WhyItMatters = "Not enough venues printed a funding payer."
	default:
		d.WhyItMatters = "Funding is near zero on at least one venue, so the payer split is weak."
	}
	return d
}

func compareCrowdingViews(views []venueSnap) VenueSignalDiff {
	rows := make([]keyedReading, 0, len(views))
	for _, v := range views {
		if !v.hasCrowd {
			rows = append(rows, keyedReading{ex: v.ex, text: "no account long/short"})
			continue
		}
		key := ""
		if v.crowd == SqueezeSideLong || v.crowd == SqueezeSideShort {
			key = v.crowd
		}
		rows = append(rows, keyedReading{
			ex: v.ex, key: key,
			text: fmt.Sprintf("%s (%.1f%% long)", v.crowd, v.longShare*100),
		})
	}
	d := finishDiff("crowding", "Account long/short", rows, true)
	switch d.Alignment {
	case AlignSame:
		d.WhyItMatters = "Account crowds lean the same way on the venues that publish them."
	case AlignOpposite:
		d.WhyItMatters = "More accounts are long on one venue and short on another. A move can liquidate different crowds."
	case AlignUnknown:
		d.WhyItMatters = "Fewer than two venues publish account long/short, so crowding cannot split the books."
	default:
		d.WhyItMatters = "At least one venue is balanced, so crowding is not a strong split."
	}
	return d
}

func compareRegimeViews(views []venueSnap) VenueSignalDiff {
	rows := make([]keyedReading, 0, len(views))
	for _, v := range views {
		key := ""
		if v.lean == LeanBullish || v.lean == LeanBearish {
			key = v.lean
		}
		rows = append(rows, keyedReading{ex: v.ex, key: key, text: v.label})
	}
	d := finishDiff("positioning", "Price + OI regime", rows, true)
	switch d.Alignment {
	case AlignSame:
		d.WhyItMatters = "The books that have a direction show the same price and open-interest lean."
	case AlignOpposite:
		d.WhyItMatters = "One venue is bullish while another is bearish. Combined open interest can hide that split."
	case AlignUnknown:
		d.WhyItMatters = "Not enough venues have a bullish or bearish regime."
	default:
		d.WhyItMatters = "Regimes differ, but they are not a clean bullish-versus-bearish split."
	}
	return d
}

type keyedReading struct {
	ex   Exchange
	key  string
	text string
}

func finishDiff(metric, label string, rows []keyedReading, oppositeMatters bool) VenueSignalDiff {
	d := VenueSignalDiff{
		Metric: metric, Label: label,
		Readings: make([]VenueReading, 0, len(rows)),
	}
	known := make([]keyedReading, 0, len(rows))
	for _, row := range rows {
		d.Readings = append(d.Readings, VenueReading{Exchange: string(row.ex), Value: row.text})
		switch row.ex {
		case ExchangeBinance:
			d.Binance = row.text
		case ExchangeBybit:
			d.Bybit = row.text
		}
		if row.key != "" {
			known = append(known, row)
		}
	}
	align, same, different := splitKeys(known)
	d.Alignment = align
	d.Same = same
	d.Different = different
	d.Important = oppositeMatters && align == AlignOpposite
	return d
}

// splitKeys groups known keys. One key is same. Two opposite keys name the
// smaller side as different. Three keys, or two that are not opposites, are mixed.
func splitKeys(rows []keyedReading) (alignment string, same, different []string) {
	if len(rows) < 2 {
		return AlignUnknown, nil, nil
	}
	buckets := map[string][]string{}
	order := []string{}
	for _, row := range rows {
		if _, ok := buckets[row.key]; !ok {
			order = append(order, row.key)
		}
		buckets[row.key] = append(buckets[row.key], string(row.ex))
	}
	if len(order) == 1 {
		return AlignSame, buckets[order[0]], nil
	}
	sort.SliceStable(order, func(i, j int) bool {
		if len(buckets[order[i]]) != len(buckets[order[j]]) {
			return len(buckets[order[i]]) > len(buckets[order[j]])
		}
		return order[i] < order[j]
	})
	if len(order) == 2 && keysOppose(order[0], order[1]) {
		same = buckets[order[0]]
		different = buckets[order[1]]
		if len(same) == len(different) {
			return AlignOpposite, nil, append(append([]string{}, same...), different...)
		}
		return AlignOpposite, same, different
	}
	all := make([]string, 0, len(rows))
	for _, row := range rows {
		all = append(all, string(row.ex))
	}
	return AlignMixed, nil, all
}

func keysOppose(a, b string) bool {
	switch {
	case (a == "up" && b == "down") || (a == "down" && b == "up"):
		return true
	case (a == LeanBullish && b == LeanBearish) || (a == LeanBearish && b == LeanBullish):
		return true
	case (a == "long" && b == "short") || (a == "short" && b == "long"):
		return true
	default:
		return false
	}
}

func leanGroups(views []venueSnap) (groups []VenueLeanGroup, outliers []string, alignment string, important bool) {
	rows := make([]keyedReading, 0, len(views))
	for _, v := range views {
		key := ""
		if v.lean == LeanBullish || v.lean == LeanBearish {
			key = v.lean
		}
		rows = append(rows, keyedReading{ex: v.ex, key: key, text: v.lean})
	}
	buckets := map[string][]string{}
	order := []string{}
	for _, row := range rows {
		if row.key == "" {
			continue
		}
		if _, ok := buckets[row.key]; !ok {
			order = append(order, row.key)
		}
		buckets[row.key] = append(buckets[row.key], string(row.ex))
	}
	sort.SliceStable(order, func(i, j int) bool {
		if len(buckets[order[i]]) != len(buckets[order[j]]) {
			return len(buckets[order[i]]) > len(buckets[order[j]])
		}
		return order[i] < order[j]
	})
	for _, key := range order {
		groups = append(groups, VenueLeanGroup{Lean: key, Exchanges: buckets[key]})
	}
	if groups == nil {
		groups = []VenueLeanGroup{}
	}
	switch len(order) {
	case 0:
		return groups, nil, AlignMixed, false
	case 1:
		return groups, nil, AlignSame, false
	default:
		if keysOppose(order[0], order[1]) && len(order) == 2 {
			if len(buckets[order[0]]) > len(buckets[order[1]]) {
				outliers = buckets[order[1]]
			}
			return groups, outliers, AlignOpposite, true
		}
		return groups, nil, AlignMixed, false
	}
}

func divergenceCopy(out VenueDivergenceReport) (title, summary string) {
	names := venueList(out.Venues)
	switch out.Alignment {
	case AlignUnknown:
		return out.Title, out.Summary
	case AlignSame:
		lean := ""
		if len(out.Groups) > 0 {
			lean = out.Groups[0].Lean
		}
		title = names + " are moving the same way"
		summary = fmt.Sprintf("%s all lean %s. %s", names, lean, sameMetricSentence(out.Diffs))
	case AlignOpposite:
		title = outlierTitle(out)
		summary = groupSentence(out) + " " + importantDiffPhrases(out.Diffs)
	default:
		title = names + " are mixed"
		summary = groupSentence(out) + " " + importantDiffPhrases(out.Diffs)
	}
	return title, strings.TrimSpace(summary)
}

func outlierTitle(out VenueDivergenceReport) string {
	if len(out.Outliers) == 1 {
		return venueTitle(out.Outliers[0]) + " is moving differently"
	}
	if len(out.Outliers) > 1 {
		return joinNames(out.Outliers) + " are moving differently"
	}
	return venueList(out.Venues) + " do not share one direction"
}

func groupSentence(out VenueDivergenceReport) string {
	if len(out.Groups) == 0 {
		return "No venue has a clear bullish or bearish lean."
	}
	parts := make([]string, 0, len(out.Groups))
	for _, g := range out.Groups {
		parts = append(parts, fmt.Sprintf("%s lean %s", joinNames(g.Exchanges), g.Lean))
	}
	return strings.Join(parts, ". ") + "."
}

func sameMetricSentence(diffs []VenueSignalDiff) string {
	n := 0
	for _, d := range diffs {
		if d.Alignment == AlignSame {
			n++
		}
	}
	if n == 0 {
		return "The overall lean matches even where a single print is soft."
	}
	return fmt.Sprintf("%d of %d compared signals also match.", n, len(diffs))
}

func importantDiffPhrases(diffs []VenueSignalDiff) string {
	var bits []string
	for _, d := range diffs {
		if d.Alignment != AlignOpposite && d.Alignment != AlignMixed {
			continue
		}
		if len(d.Different) == 0 && d.Alignment != AlignOpposite {
			continue
		}
		bits = append(bits, fmt.Sprintf("%s: %s.", d.Label, readingPhrase(d)))
	}
	if len(bits) == 0 {
		return "The leans differ even where a single print is soft."
	}
	return strings.Join(bits, " ")
}

func readingPhrase(d VenueSignalDiff) string {
	parts := make([]string, 0, len(d.Readings))
	for _, r := range d.Readings {
		parts = append(parts, r.Exchange+" "+r.Value)
	}
	return strings.Join(parts, ", ")
}

func venueList(views []VenueLeanView) string {
	names := make([]string, 0, len(views))
	for _, v := range views {
		names = append(names, venueTitle(v.Exchange))
	}
	return joinNames(names)
}

func venueTitle(ex string) string {
	switch Exchange(ex) {
	case ExchangeBinance:
		return "Binance"
	case ExchangeBybit:
		return "Bybit"
	case ExchangeOKX:
		return "OKX"
	default:
		if ex == "" {
			return "unknown"
		}
		return ex
	}
}

func joinNames(names []string) string {
	titled := make([]string, len(names))
	for i, n := range names {
		titled[i] = venueTitle(n)
	}
	switch len(titled) {
	case 0:
		return ""
	case 1:
		return titled[0]
	case 2:
		return titled[0] + " and " + titled[1]
	default:
		return strings.Join(titled[:len(titled)-1], ", ") + ", and " + titled[len(titled)-1]
	}
}

// HasMeaningfulDivergence is true when the user should be alerted.
func HasMeaningfulDivergence(r VenueDivergenceReport) bool {
	return r.Important && r.Alignment == AlignOpposite
}
