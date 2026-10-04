package domain

import (
	"fmt"
	"sort"
	"time"
)

const (
	PrintSizeWindow5m  = "5m"
	PrintSizeWindow15m = "15m"

	PrintSizeKindJumped = "jumped"
	PrintSizeKindFaded  = "faded"
	PrintSizeKindStable = "stable"

	PrintSizeReadFewLarge  = "few_large"
	PrintSizeReadManySmall = "many_small"
	PrintSizeReadLarger    = "larger"
	PrintSizeReadSmaller   = "smaller"
	PrintSizeReadStable    = "stable"

	printSizeJumpRatio  = 2.5
	printSizeFadeRatio  = 1.0 / 2.5
	printSizeVolRatio   = 1.3
	printSizeCountRatio = 1.3
	printSizeLookback   = 30 * time.Minute // 15m current + 15m prior
	printSizeMaxPages   = 15
)

// PrintSizeWindows is current 5m / 15m vs the same length just before.
var PrintSizeWindows = []struct {
	ID  string
	Dur time.Duration
}{
	{PrintSizeWindow5m, 5 * time.Minute},
	{PrintSizeWindow15m, 15 * time.Minute},
}

// PrintSizeLookback is how far back ListPrints should load (current 15m + prior 15m).
func PrintSizeLookback() time.Duration { return printSizeLookback }

// MemoryPrintPort returns prints already in memory. The boolean is false when
// the tape does not reach from. Callers must not treat that as an empty tape
// and must not fall back to a REST history walk.
type MemoryPrintPort interface {
	MemoryPrints(symbol string, from, to time.Time) ([]TakerPrint, bool)
}

// PrintSizeMaxPages caps REST pagination when walking aggTrades.
func PrintSizeMaxPages() int { return printSizeMaxPages }

// PrintSizeSide is average aggressive print size on one side of the book.
type PrintSizeSide struct {
	Avg           float64
	PriorAvg      float64
	Notional      float64
	PriorNotional float64
	Count         int
	PriorCount    int
	Ratio         float64 // current avg / prior avg; 0 if prior unknown
	Kind          string  // jumped | faded | stable
}

// PrintSizeWindow is one lookback vs the equal window just before it.
type PrintSizeWindow struct {
	Window   string
	Buy      PrintSizeSide
	Sell     PrintSizeSide
	Complete bool
	From     time.Time
	To       time.Time
	Read     string // few_large | many_small | larger | smaller | stable
	Summary  string
}

// PrintSizeVenue is one venue's print-size picture.
type PrintSizeVenue struct {
	Exchange Exchange
	Symbol   string
	Windows  []PrintSizeWindow
	Summary  string
	Error    string
	Span     time.Duration // oldest→newest print we actually have
	Newest   time.Time
}

// PrintSizeReport is the API result.
type PrintSizeReport struct {
	Symbol   string
	Exchange string
	AsOf     time.Time
	Venues   []PrintSizeVenue
	Combined *PrintSizeVenue
	Note     string
}

// BuildPrintSizeVenue folds prints into 5m / 15m average size vs the prior window.
func BuildPrintSizeVenue(ex Exchange, symbol string, prints []TakerPrint, now time.Time) PrintSizeVenue {
	symbol = NormalizeLiquidationSymbol(symbol)
	now = now.UTC()
	out := PrintSizeVenue{
		Exchange: ex,
		Symbol:   symbol,
		Windows:  make([]PrintSizeWindow, 0, len(PrintSizeWindows)),
	}
	clean := make([]TakerPrint, 0, len(prints))
	var oldest, newest time.Time
	for _, p := range prints {
		if p.Notional <= 0 || p.Time.IsZero() {
			continue
		}
		t := p.Time.UTC()
		if t.After(now) {
			continue
		}
		p.Time = t
		p.Symbol = symbol
		if p.Exchange == "" {
			p.Exchange = ex
		}
		clean = append(clean, p)
		if oldest.IsZero() || t.Before(oldest) {
			oldest = t
		}
		if newest.IsZero() || t.After(newest) {
			newest = t
		}
	}
	if !oldest.IsZero() && !newest.IsZero() {
		out.Span = newest.Sub(oldest)
		out.Newest = newest
	}
	sort.SliceStable(clean, func(i, j int) bool { return clean[i].Time.Before(clean[j].Time) })
	for _, w := range PrintSizeWindows {
		out.Windows = append(out.Windows, summarizePrintSizeWindow(clean, w.ID, w.Dur, now, oldest))
	}
	out.Summary = ExplainPrintSize(out)
	return out
}

func summarizePrintSizeWindow(prints []TakerPrint, id string, dur time.Duration, now, oldest time.Time) PrintSizeWindow {
	curFrom := now.Add(-dur)
	priorFrom := now.Add(-2 * dur)
	buy, sell := tallyPrints(prints, curFrom, now.Add(time.Millisecond))
	pBuy, pSell := tallyPrints(prints, priorFrom, curFrom)
	out := PrintSizeWindow{
		Window: id,
		Buy:    printSizeSide(buy.notional, buy.count, pBuy.notional, pBuy.count),
		Sell:   printSizeSide(sell.notional, sell.count, pSell.notional, pSell.count),
		From:   curFrom,
		To:     now,
	}
	// Covered if we have a print at or before the window open (slack 20% of the window, min 30s).
	slack := dur / 5
	if slack < 30*time.Second {
		slack = 30 * time.Second
	}
	out.Complete = !oldest.IsZero() && !oldest.After(curFrom.Add(slack)) && (buy.count+sell.count) > 0
	out.Read = printSizeRead(out.Buy, out.Sell)
	out.Summary = printSizeWindowSummary(id, out)
	return out
}

type printTally struct {
	notional float64
	count    int
}

func tallyPrints(prints []TakerPrint, from, to time.Time) (buy, sell printTally) {
	for _, p := range prints {
		// [from, to)
		if p.Time.Before(from) || !p.Time.Before(to) {
			continue
		}
		switch p.Side {
		case TakerSideBuy:
			buy.notional += p.Notional
			buy.count++
		case TakerSideSell:
			sell.notional += p.Notional
			sell.count++
		}
	}
	return buy, sell
}

func printSizeSide(notional float64, count int, priorNotional float64, priorCount int) PrintSizeSide {
	out := PrintSizeSide{
		Notional:      notional,
		PriorNotional: priorNotional,
		Count:         count,
		PriorCount:    priorCount,
		Kind:          PrintSizeKindStable,
	}
	if count > 0 {
		out.Avg = notional / float64(count)
	}
	if priorCount > 0 {
		out.PriorAvg = priorNotional / float64(priorCount)
	}
	if out.PriorAvg > 0 && out.Avg > 0 {
		out.Ratio = out.Avg / out.PriorAvg
		switch {
		case out.Ratio >= printSizeJumpRatio:
			out.Kind = PrintSizeKindJumped
		case out.Ratio <= printSizeFadeRatio:
			out.Kind = PrintSizeKindFaded
		}
	}
	return out
}

func printSizeRead(buy, sell PrintSizeSide) string {
	b := sideRead(buy)
	s := sideRead(sell)
	if b == PrintSizeReadFewLarge || s == PrintSizeReadFewLarge {
		return PrintSizeReadFewLarge
	}
	if b == PrintSizeReadManySmall || s == PrintSizeReadManySmall {
		return PrintSizeReadManySmall
	}
	if b == PrintSizeReadLarger || s == PrintSizeReadLarger {
		return PrintSizeReadLarger
	}
	if b == PrintSizeReadSmaller || s == PrintSizeReadSmaller {
		return PrintSizeReadSmaller
	}
	return PrintSizeReadStable
}

func sideRead(s PrintSizeSide) string {
	if s.PriorCount <= 0 || s.Count <= 0 {
		if s.Kind == PrintSizeKindJumped {
			return PrintSizeReadLarger
		}
		if s.Kind == PrintSizeKindFaded {
			return PrintSizeReadSmaller
		}
		return PrintSizeReadStable
	}
	volUp := s.Notional >= s.PriorNotional*printSizeVolRatio
	countUp := float64(s.Count) >= float64(s.PriorCount)*printSizeCountRatio
	countFlat := float64(s.Count) <= float64(s.PriorCount)*printSizeCountRatio
	switch {
	case volUp && s.Kind == PrintSizeKindJumped && countFlat:
		return PrintSizeReadFewLarge
	case volUp && s.Kind != PrintSizeKindJumped && countUp:
		return PrintSizeReadManySmall
	case s.Kind == PrintSizeKindJumped:
		return PrintSizeReadLarger
	case s.Kind == PrintSizeKindFaded:
		return PrintSizeReadSmaller
	default:
		return PrintSizeReadStable
	}
}

func printSizeWindowSummary(id string, w PrintSizeWindow) string {
	bits := make([]string, 0, 2)
	if w.Buy.Count > 0 {
		bits = append(bits, sideSummary("buy", w.Buy))
	}
	if w.Sell.Count > 0 {
		bits = append(bits, sideSummary("sell", w.Sell))
	}
	if len(bits) == 0 {
		return fmt.Sprintf("No aggressive prints in the last %s.", id)
	}
	head := fmt.Sprintf("Last %s: %s.", id, joinAnd(bits))
	switch w.Read {
	case PrintSizeReadFewLarge:
		return head + " That looks like a few large market orders, not many small ones."
	case PrintSizeReadManySmall:
		return head + " Volume came from many small prints, not a few large ones."
	default:
		return head
	}
}

func sideSummary(side string, s PrintSizeSide) string {
	cur := formatQty(s.Avg)
	if s.PriorAvg <= 0 {
		return fmt.Sprintf("%s prints averaged %s USDT (%d fills)", side, cur, s.Count)
	}
	prior := formatQty(s.PriorAvg)
	switch s.Kind {
	case PrintSizeKindJumped:
		return fmt.Sprintf("%s prints averaged %s USDT vs %s before (%d vs %d fills)", side, cur, prior, s.Count, s.PriorCount)
	case PrintSizeKindFaded:
		return fmt.Sprintf("%s prints averaged %s USDT vs %s before (%d vs %d fills)", side, cur, prior, s.Count, s.PriorCount)
	default:
		return fmt.Sprintf("%s prints averaged %s USDT (%d fills, prior %s)", side, cur, s.Count, prior)
	}
}

// ExplainPrintSize prefers the 15m window, then 5m.
func ExplainPrintSize(v PrintSizeVenue) string {
	if v.Error != "" {
		return v.Error
	}
	w := pickPrintSizeWindow(v.Windows, PrintSizeWindow15m)
	if w.Buy.Count+w.Sell.Count == 0 {
		w = pickPrintSizeWindow(v.Windows, PrintSizeWindow5m)
	}
	if w.Summary == "" {
		return "No aggressive print-size tape yet."
	}
	ex := string(v.Exchange)
	switch ex {
	case "", "all":
		ex = "Combined"
	case string(ExchangeBinance):
		ex = "Binance"
	case string(ExchangeBybit):
		ex = "Bybit"
	}
	return ex + ": " + w.Summary
}

func pickPrintSizeWindow(ws []PrintSizeWindow, id string) PrintSizeWindow {
	for _, w := range ws {
		if w.Window == id {
			return w
		}
	}
	return PrintSizeWindow{Window: id}
}

// CombinePrintSizeVenues sums notional and counts, then re-averages.
func CombinePrintSizeVenues(symbol string, venues []PrintSizeVenue) *PrintSizeVenue {
	if len(venues) == 0 {
		return nil
	}
	out := &PrintSizeVenue{
		Exchange: "all",
		Symbol:   NormalizeLiquidationSymbol(symbol),
		Windows:  make([]PrintSizeWindow, 0, len(PrintSizeWindows)),
	}
	var span time.Duration
	for _, v := range venues {
		if v.Span > span {
			span = v.Span
		}
	}
	out.Span = span
	now := time.Now().UTC()
	for _, spec := range PrintSizeWindows {
		var buyN, buyPN, sellN, sellPN float64
		var buyC, buyPC, sellC, sellPC int
		complete := true
		any := false
		var from, to time.Time
		for _, v := range venues {
			if v.Error != "" {
				complete = false
			}
			for _, w := range v.Windows {
				if w.Window != spec.ID {
					continue
				}
				buyN += w.Buy.Notional
				buyPN += w.Buy.PriorNotional
				buyC += w.Buy.Count
				buyPC += w.Buy.PriorCount
				sellN += w.Sell.Notional
				sellPN += w.Sell.PriorNotional
				sellC += w.Sell.Count
				sellPC += w.Sell.PriorCount
				any = true
				if !w.Complete {
					complete = false
				}
				if from.IsZero() || w.From.Before(from) {
					from = w.From
				}
				if to.IsZero() || w.To.After(to) {
					to = w.To
				}
			}
		}
		if !any {
			complete = false
		}
		if to.IsZero() {
			to = now
			from = now.Add(-spec.Dur)
		}
		win := PrintSizeWindow{
			Window:   spec.ID,
			Buy:      printSizeSide(buyN, buyC, buyPN, buyPC),
			Sell:     printSizeSide(sellN, sellC, sellPN, sellPC),
			Complete: complete,
			From:     from,
			To:       to,
		}
		win.Read = printSizeRead(win.Buy, win.Sell)
		win.Summary = printSizeWindowSummary(spec.ID, win)
		out.Windows = append(out.Windows, win)
	}
	out.Summary = ExplainPrintSize(*out)
	return out
}

const (
	printSizeOverviewDefault = 20
	printSizeOverviewMin     = 5
	printSizeOverviewMax     = 30
)

// PrintSizeMarketWindow is one lookback averaged across the followed coins.
// Notionals and counts are added first, then the average is divided.
// Coin counts are one vote per pair.
type PrintSizeMarketWindow struct {
	Window         string
	Buy            PrintSizeSide
	Sell           PrintSizeSide
	Read           string
	LargerCoins    int
	SmallerCoins   int
	FewLargeCoins  int
	ManySmallCoins int
	StableCoins    int
	QuietCoins     int
	Coins          int
	Complete       bool
	Summary        string
}

// PrintSizeOverview is the whole-market average print size.
type PrintSizeOverview struct {
	Exchange string
	Quote    string
	Universe int
	AsOf     time.Time
	Windows  []PrintSizeMarketWindow
	Coins    []PrintSizeVenue
	Summary  string
	Note     string
	Coverage MarketCoverage
	Definite bool
}

// ParsePrintSizeOverviewLimit clamps how many liquid pairs the desk follows.
func ParsePrintSizeOverviewLimit(n int) int {
	if n <= 0 {
		return printSizeOverviewDefault
	}
	if n < printSizeOverviewMin {
		return printSizeOverviewMin
	}
	if n > printSizeOverviewMax {
		return printSizeOverviewMax
	}
	return n
}

// BuildPrintSizeOverview sums notional and fill counts across coins, then re-averages.
// Coins are ordered by the 15m size multiple, largest jump first.
func BuildPrintSizeOverview(exchange, quote string, coins []PrintSizeVenue, now time.Time) PrintSizeOverview {
	usable := make([]PrintSizeVenue, 0, len(coins))
	for _, c := range coins {
		if c.Error != "" || c.Symbol == "" {
			continue
		}
		usable = append(usable, c)
	}
	out := PrintSizeOverview{
		Exchange: exchange,
		Quote:    quote,
		Universe: len(usable),
		AsOf:     now.UTC(),
		Windows:  make([]PrintSizeMarketWindow, 0, len(PrintSizeWindows)),
		Coins:    usable,
	}
	for _, spec := range PrintSizeWindows {
		out.Windows = append(out.Windows, summarizePrintSizeMarket(usable, spec.ID, spec.Dur, now))
	}
	sort.SliceStable(out.Coins, func(i, j int) bool {
		ri, rj := printSizeJump(out.Coins[i], PrintSizeWindow15m), printSizeJump(out.Coins[j], PrintSizeWindow15m)
		if ri != rj {
			return ri > rj
		}
		return out.Coins[i].Symbol < out.Coins[j].Symbol
	})
	out.Summary = explainPrintSizeOverview(out)
	return out
}

func summarizePrintSizeMarket(coins []PrintSizeVenue, window string, dur time.Duration, now time.Time) PrintSizeMarketWindow {
	var buyN, buyPN, sellN, sellPN float64
	var buyC, buyPC, sellC, sellPC int
	var larger, smaller, few, many, stable, quiet int
	complete := false
	for _, c := range coins {
		w := pickPrintSizeWindow(c.Windows, window)
		buyN += w.Buy.Notional
		buyPN += w.Buy.PriorNotional
		buyC += w.Buy.Count
		buyPC += w.Buy.PriorCount
		sellN += w.Sell.Notional
		sellPN += w.Sell.PriorNotional
		sellC += w.Sell.Count
		sellPC += w.Sell.PriorCount
		if w.Complete {
			complete = true
		}
		switch printSizeCoinVote(w) {
		case PrintSizeReadFewLarge:
			few++
		case PrintSizeReadManySmall:
			many++
		case PrintSizeReadLarger:
			larger++
		case PrintSizeReadSmaller:
			smaller++
		case "quiet":
			quiet++
		default:
			stable++
		}
	}
	win := PrintSizeWindow{
		Window: window,
		Buy:    printSizeSide(buyN, buyC, buyPN, buyPC),
		Sell:   printSizeSide(sellN, sellC, sellPN, sellPC),
		From:   now.Add(-dur),
		To:     now,
	}
	win.Read = printSizeRead(win.Buy, win.Sell)
	out := PrintSizeMarketWindow{
		Window: win.Window, Buy: win.Buy, Sell: win.Sell, Read: win.Read,
		LargerCoins: larger, SmallerCoins: smaller, FewLargeCoins: few, ManySmallCoins: many,
		StableCoins: stable, QuietCoins: quiet, Coins: len(coins), Complete: complete,
	}
	out.Summary = explainPrintSizeMarketWindow(out, win)
	return out
}

func printSizeCoinVote(w PrintSizeWindow) string {
	if w.Buy.Count+w.Sell.Count == 0 {
		return "quiet"
	}
	if w.Read != "" {
		return w.Read
	}
	return printSizeRead(w.Buy, w.Sell)
}

func printSizeJump(v PrintSizeVenue, window string) float64 {
	w := pickPrintSizeWindow(v.Windows, window)
	jump := w.Buy.Ratio
	if w.Sell.Ratio > jump {
		jump = w.Sell.Ratio
	}
	return jump
}

func explainPrintSizeMarketWindow(m PrintSizeMarketWindow, w PrintSizeWindow) string {
	head := printSizeWindowSummary(m.Window, w)
	return fmt.Sprintf("%s %d coins larger, %d smaller, %d few large, %d many small, %d stable, %d quiet.",
		head, m.LargerCoins, m.SmallerCoins, m.FewLargeCoins, m.ManySmallCoins, m.StableCoins, m.QuietCoins)
}

func explainPrintSizeOverview(r PrintSizeOverview) string {
	for _, w := range r.Windows {
		if w.Window == PrintSizeWindow15m && w.Summary != "" {
			return w.Summary
		}
	}
	if len(r.Windows) > 0 {
		return r.Windows[0].Summary
	}
	return "No aggressive print-size tape yet."
}

// FilterPrintsInRange keeps fills in [from, to].
func FilterPrintsInRange(prints []TakerPrint, from, to time.Time) []TakerPrint {
	if len(prints) == 0 {
		return prints
	}
	from, to = from.UTC(), to.UTC()
	out := make([]TakerPrint, 0, len(prints))
	for _, p := range prints {
		t := p.Time.UTC()
		if t.Before(from) || t.After(to) {
			continue
		}
		p.Time = t
		out = append(out, p)
	}
	return out
}
