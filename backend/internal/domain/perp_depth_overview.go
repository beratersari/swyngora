package domain

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	PerpDepthOverviewDefaultN = 20
	PerpDepthOverviewMinN     = 5
	PerpDepthOverviewMaxN     = 40
	perpDepthOverviewLeaders  = 5
	perpOverviewBandLeaders   = 3
)

// PerpDepthCoinInput is one coin's headline bid/ask used to build the market total.
type PerpDepthCoinInput struct {
	Symbol   string
	Base     string
	Bid      float64
	Ask      float64
	Lean     string
	Complete bool
	Error    string
	Bands    []PerpDepthCoinBandInput
	Venues   []PerpDepthVenueBook
}

// PerpDepthVenueBook is one venue's headline bid/ask for a coin.
type PerpDepthVenueBook struct {
	Exchange string
	Bid      float64
	Ask      float64
}

// PerpDepthCoinBandInput is one coin's bid/ask inside a ±% of its own mid.
type PerpDepthCoinBandInput struct {
	RangePct float64
	Bid      float64
	Ask      float64
	Complete bool
}

// PerpDepthCoinShare is one coin's slice of market-wide perp bid/ask.
type PerpDepthCoinShare struct {
	Symbol      string
	Base        string
	Bid         float64
	Ask         float64
	Total       float64
	SharePct    float64 // of bid+ask total
	BidSharePct float64 // of market bid
	AskSharePct float64 // of market ask
	Lean        string
	Complete    bool
	Error       string
}

// PerpDepthOverview is resting perp bid/ask across the liquid coins we track.
type PerpDepthOverview struct {
	Exchange  string
	RangePct  float64
	Universe  int
	Known     int
	AsOf      time.Time
	Bid       float64
	Ask       float64
	Total     float64
	Imbalance float64
	BidPct    float64 // 100 * bid / (bid+ask)
	AskPct    float64
	Lean      string
	Coins     []PerpDepthCoinShare
	Leaders   []string
	Missing   []string
	Windows   []PerpDepthShareWindow
	History   []PerpDepthTotalPoint
	Moves     []PerpDepthBookMove
	Bands     []PerpDepthOverviewBand
	Summary   string
	Note      string
	Coverage  MarketCoverage
	Definite  bool
}

// PerpDepthOverviewBandCoin is one coin's slice of a ±% market-wide band.
type PerpDepthOverviewBandCoin struct {
	Symbol   string
	Base     string
	Bid      float64
	Ask      float64
	Total    float64
	SharePct float64
}

// PerpDepthOverviewBand is market-wide bid/ask inside ±RangePct of each coin's mid.
type PerpDepthOverviewBand struct {
	RangePct      float64
	Bid           float64
	Ask           float64
	Total         float64
	BidPct        float64
	AskPct        float64
	Lean          string
	OfHeadlinePct float64 // this band's total as % of the headline (e.g. 1%) book
	AddedPct      float64 // ring vs the previous tighter band, as % of the headline book
	AddedBid      float64
	AddedAsk      float64
	Complete      bool
	Leaders       []string
	AddedLeaders  []string
	Coins         []PerpDepthOverviewBandCoin
	Summary       string
}

// ParsePerpDepthOverviewLimit clamps how many liquid perps to scan.
func ParsePerpDepthOverviewLimit(n int) int {
	if n <= 0 {
		return PerpDepthOverviewDefaultN
	}
	if n < PerpDepthOverviewMinN {
		return PerpDepthOverviewMinN
	}
	if n > PerpDepthOverviewMaxN {
		return PerpDepthOverviewMaxN
	}
	return n
}

// BuildPerpDepthOverview sums headline bid/ask across coins and ranks their share of that book.
func BuildPerpDepthOverview(exchange string, rangePct float64, coins []PerpDepthCoinInput, asOf time.Time) *PerpDepthOverview {
	if rangePct <= 0 {
		rangePct = DefaultPerpDepthRangePct
	}
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	out := &PerpDepthOverview{
		Exchange: exchange,
		RangePct: rangePct,
		Universe: len(coins),
		AsOf:     asOf.UTC(),
		Lean:     PerpDepthLeanEven,
		Coins:    []PerpDepthCoinShare{},
		Leaders:  []string{},
		Missing:  []string{},
		Windows:  []PerpDepthShareWindow{},
		History:  []PerpDepthTotalPoint{},
		Moves:    []PerpDepthBookMove{},
		Bands:    []PerpDepthOverviewBand{},
		Note:     "Resting USD-M / linear perpetual depth across the most liquid USDT coins we follow. Bid is notional within N% below each coin's own mid; ask is notional within N% above. bands show how that book grows from 0.1% out to the headline (e.g. 80M inside 0.1% vs 1.0B at 1%) and which coins fill each distance. Combined exchange=all adds Binance, Bybit, and OKX for each coin. windows compare bid vs ask share of that book 15m / 1h / 4h ago (sampled while the desk runs), which coins moved the mix, and whether total bid+ask got thicker or thinner (at least 5%), plus which coins added or removed the most of that size. history is that total each minute for the last 6 hours, with BTC's close on the same minutes. moves are the largest jumps and drops of at least 5% that finish within 15 minutes, plus what BTC did during each and 15 minutes later, which coins caused most of each move, whether depth came back or kept going in the next 15 minutes, and whether Binance, Bybit, and OKX moved together or only one venue did. Informational only — not financial advice.",
	}
	rows := make([]PerpDepthCoinShare, 0, len(coins))
	var bid, ask float64
	for _, c := range coins {
		row := PerpDepthCoinShare{
			Symbol:   NormalizeLiquidationSymbol(c.Symbol),
			Base:     perpOverviewBase(c.Base, c.Symbol),
			Bid:      c.Bid,
			Ask:      c.Ask,
			Total:    c.Bid + c.Ask,
			Lean:     c.Lean,
			Complete: c.Complete,
			Error:    c.Error,
		}
		row.Lean = perpOverviewLean(row.Lean, row.Bid, row.Ask)
		if row.Error != "" || row.Total <= 0 {
			if row.Symbol != "" {
				out.Missing = append(out.Missing, row.Symbol)
			}
			rows = append(rows, row)
			continue
		}
		bid += row.Bid
		ask += row.Ask
		out.Known++
		rows = append(rows, row)
	}
	out.Bid = bid
	out.Ask = ask
	out.Total = bid + ask
	if out.Total > 0 {
		out.Imbalance = round4((bid - ask) / out.Total)
	}
	out.Lean = perpOverviewLean("", bid, ask)
	out.BidPct = PerpBookBidPct(bid, ask)
	out.AskPct = round4(100 - out.BidPct)
	if out.Total <= 0 {
		out.AskPct = 0
	}
	for i := range rows {
		if rows[i].Error != "" || rows[i].Total <= 0 {
			continue
		}
		if out.Total > 0 {
			rows[i].SharePct = round4(100 * rows[i].Total / out.Total)
		}
		if bid > 0 {
			rows[i].BidSharePct = round4(100 * rows[i].Bid / bid)
		}
		if ask > 0 {
			rows[i].AskSharePct = round4(100 * rows[i].Ask / ask)
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Total != rows[j].Total {
			return rows[i].Total > rows[j].Total
		}
		return rows[i].Symbol < rows[j].Symbol
	})
	out.Coins = rows
	for _, row := range rows {
		if len(out.Leaders) >= perpDepthOverviewLeaders {
			break
		}
		if row.Total <= 0 || row.Error != "" || row.Base == "" {
			continue
		}
		out.Leaders = append(out.Leaders, row.Base)
	}
	out.Bands = buildPerpDepthOverviewBands(rangePct, coins)
	out.Summary = ExplainPerpDepthOverview(*out)
	return out
}

// ExplainPerpDepthOverview writes total bid vs ask plus the coins that dominate that book.
func ExplainPerpDepthOverview(o PerpDepthOverview) string {
	if o.Known == 0 {
		return "No perpetual depth across the liquid coins we track."
	}
	head := fmt.Sprintf("%d liquid perps: %s bid vs %s ask within %g%% of mid",
		o.Known, formatQty(o.Bid), formatQty(o.Ask), o.RangePct)
	switch o.Lean {
	case "bid":
		head += " (bid-heavy)."
	case "ask":
		head += " (ask-heavy)."
	default:
		head += "."
	}
	parts := make([]string, 0, len(o.Leaders))
	byBase := map[string]PerpDepthCoinShare{}
	for _, c := range o.Coins {
		if c.Base != "" {
			byBase[c.Base] = c
		}
	}
	for _, name := range o.Leaders {
		c, ok := byBase[name]
		if !ok {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %s%%", name, formatQty(c.SharePct)))
	}
	if len(parts) > 0 {
		head += " " + joinList(parts) + " of that book."
	}
	if o.BidPct > 0 || o.AskPct > 0 {
		head += fmt.Sprintf(" Bids are %s%% of the book, asks %s%%.", formatQty(o.BidPct), formatQty(o.AskPct))
	}
	if near := ExplainPerpDepthNearness(o); near != "" {
		head += " " + near
	}
	for _, w := range o.Windows {
		if w.Complete && w.Summary != "" {
			head += " " + w.Summary
			break
		}
	}
	return head
}

func perpOverviewBase(base, symbol string) string {
	b := prettyBase(base)
	if b != "" && b != base {
		return b
	}
	if base != "" {
		return base
	}
	return prettyBase(symbol)
}

func perpOverviewLean(lean string, bid, ask float64) string {
	switch strings.ToLower(strings.TrimSpace(lean)) {
	case "bid", TakerSideBuy:
		return "bid"
	case "ask", TakerSideSell:
		return "ask"
	case TakerSideEven, "even":
		return "balanced"
	}
	switch TakerDominant(bid, ask) {
	case TakerSideBuy:
		return "bid"
	case TakerSideSell:
		return "ask"
	default:
		return "balanced"
	}
}

// IsPerpDepthEligible is the breadth universe minus stables (USDT/USDC perps are not the book).
func IsPerpDepthEligible(base string) bool {
	if !IsBreadthEligible(base) {
		return false
	}
	switch strings.ToUpper(strings.TrimSpace(base)) {
	case "USDT", "USDC", "BUSD", "FDUSD", "DAI", "TUSD", "USDE", "USD1":
		return false
	default:
		return true
	}
}

// ExplainPerpDepthNearness says how much of the headline book sits close to mid.
func ExplainPerpDepthNearness(o PerpDepthOverview) string {
	if len(o.Bands) < 2 {
		return ""
	}
	inner := o.Bands[0]
	outer := o.Bands[len(o.Bands)-1]
	for _, b := range o.Bands {
		if math.Abs(b.RangePct-o.RangePct) < 1e-9 {
			outer = b
			break
		}
	}
	if inner.RangePct >= outer.RangePct-1e-9 || outer.Total <= 0 {
		return ""
	}
	innerWho := perpBandLeaderLabel(inner.Leaders)
	outerWho := perpBandLeaderLabel(outer.Leaders)
	if inner.OfHeadlinePct < 40 {
		return fmt.Sprintf("Inside %g%% is only %s%% of that book%s; most size sits out to %g%%%s.",
			inner.RangePct, formatQty(inner.OfHeadlinePct), innerWho, outer.RangePct, outerWho)
	}
	return fmt.Sprintf("Inside %g%% already holds %s%% of the %g%% book%s.",
		inner.RangePct, formatQty(inner.OfHeadlinePct), outer.RangePct, innerWho)
}

func perpBandLeaderLabel(leaders []string) string {
	if len(leaders) == 0 || leaders[0] == "" {
		return ""
	}
	return " (" + leaders[0] + ")"
}

func buildPerpDepthOverviewBands(headlinePct float64, coins []PerpDepthCoinInput) []PerpDepthOverviewBand {
	pcts := overviewBandPcts(headlinePct, coins)
	if len(pcts) == 0 {
		return []PerpDepthOverviewBand{}
	}
	type row struct {
		symbol, base string
		bid, ask     float64
		complete     bool
	}
	byPct := make([][]row, len(pcts))
	for _, c := range coins {
		if c.Error != "" {
			continue
		}
		base := perpOverviewBase(c.Base, c.Symbol)
		for i, pct := range pcts {
			bid, ask, ok, complete := coinBandAt(c, headlinePct, pct)
			if !ok || bid+ask <= 0 {
				continue
			}
			byPct[i] = append(byPct[i], row{
				symbol: NormalizeLiquidationSymbol(c.Symbol), base: base,
				bid: bid, ask: ask, complete: complete,
			})
		}
	}
	out := make([]PerpDepthOverviewBand, 0, len(pcts))
	var prevBid, prevAsk float64
	var prevRows []row
	headlineTotal := 0.0
	for i, pct := range pcts {
		if math.Abs(pct-headlinePct) < 1e-9 {
			var bid, ask float64
			for _, r := range byPct[i] {
				bid += r.bid
				ask += r.ask
			}
			headlineTotal = bid + ask
			break
		}
	}
	for i, pct := range pcts {
		b := PerpDepthOverviewBand{
			RangePct:     pct,
			Lean:         PerpDepthLeanEven,
			Leaders:      []string{},
			AddedLeaders: []string{},
			Coins:        []PerpDepthOverviewBandCoin{},
		}
		completeN, n := 0, 0
		for _, r := range byPct[i] {
			b.Bid += r.bid
			b.Ask += r.ask
			n++
			if r.complete {
				completeN++
			}
		}
		b.Total = b.Bid + b.Ask
		b.Complete = n > 0 && completeN == n
		b.Lean = perpOverviewLean("", b.Bid, b.Ask)
		b.BidPct = PerpBookBidPct(b.Bid, b.Ask)
		b.AskPct = round4(100 - b.BidPct)
		if b.Total <= 0 {
			b.AskPct = 0
		}
		if headlineTotal > 0 {
			b.OfHeadlinePct = round4(100 * b.Total / headlineTotal)
		}
		addedBid := b.Bid - prevBid
		addedAsk := b.Ask - prevAsk
		if addedBid < 0 {
			addedBid = 0
		}
		if addedAsk < 0 {
			addedAsk = 0
		}
		b.AddedBid = addedBid
		b.AddedAsk = addedAsk
		if headlineTotal > 0 {
			b.AddedPct = round4(100 * (addedBid + addedAsk) / headlineTotal)
		}
		coinRows := make([]PerpDepthOverviewBandCoin, 0, len(byPct[i]))
		for _, r := range byPct[i] {
			row := PerpDepthOverviewBandCoin{Symbol: r.symbol, Base: r.base, Bid: r.bid, Ask: r.ask, Total: r.bid + r.ask}
			if b.Total > 0 {
				row.SharePct = round4(100 * row.Total / b.Total)
			}
			coinRows = append(coinRows, row)
		}
		sort.SliceStable(coinRows, func(a, b int) bool {
			if coinRows[a].Total != coinRows[b].Total {
				return coinRows[a].Total > coinRows[b].Total
			}
			return coinRows[a].Symbol < coinRows[b].Symbol
		})
		if len(coinRows) > perpOverviewBandLeaders {
			b.Coins = coinRows[:perpOverviewBandLeaders]
		} else {
			b.Coins = coinRows
		}
		b.Leaders = bandLeaderNames(coinRows, perpOverviewBandLeaders)
		if i > 0 {
			addedRows := make([]PerpDepthOverviewBandCoin, 0, len(byPct[i]))
			prevBy := map[string]row{}
			for _, r := range prevRows {
				prevBy[r.symbol] = r
			}
			for _, r := range byPct[i] {
				p := prevBy[r.symbol]
				ab, aa := r.bid-p.bid, r.ask-p.ask
				if ab < 0 {
					ab = 0
				}
				if aa < 0 {
					aa = 0
				}
				if ab+aa <= 0 {
					continue
				}
				addedRows = append(addedRows, PerpDepthOverviewBandCoin{
					Symbol: r.symbol, Base: r.base, Bid: ab, Ask: aa, Total: ab + aa,
				})
			}
			sort.SliceStable(addedRows, func(a, c int) bool {
				if addedRows[a].Total != addedRows[c].Total {
					return addedRows[a].Total > addedRows[c].Total
				}
				return addedRows[a].Symbol < addedRows[c].Symbol
			})
			b.AddedLeaders = bandLeaderNames(addedRows, perpOverviewBandLeaders)
		}
		b.Summary = ExplainPerpDepthOverviewBand(b, headlinePct)
		out = append(out, b)
		prevBid, prevAsk, prevRows = b.Bid, b.Ask, byPct[i]
	}
	return out
}

func ExplainPerpDepthOverviewBand(b PerpDepthOverviewBand, headlinePct float64) string {
	head := fmt.Sprintf("%g%%: %s bid vs %s ask", b.RangePct, formatQty(b.Bid), formatQty(b.Ask))
	if headlinePct > 0 && math.Abs(b.RangePct-headlinePct) > 1e-9 {
		head += fmt.Sprintf(" (%s%% of the %g%% book)", formatQty(b.OfHeadlinePct), headlinePct)
	}
	if len(b.Leaders) > 0 {
		head += "; " + joinList(b.Leaders)
	}
	head += "."
	return head
}

func overviewBandPcts(headline float64, coins []PerpDepthCoinInput) []float64 {
	seen := map[float64]struct{}{}
	out := make([]float64, 0, len(PerpDepthBandPcts)+1)
	add := func(p float64) {
		if p <= 0 || p > headline+1e-9 {
			return
		}
		key := math.Round(p*10000) / 10000
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, p)
	}
	add(headline)
	for _, c := range coins {
		for _, b := range c.Bands {
			add(b.RangePct)
		}
	}
	sort.Float64s(out)
	return out
}

func coinBandAt(c PerpDepthCoinInput, headline, pct float64) (bid, ask float64, ok, complete bool) {
	for _, b := range c.Bands {
		if math.Abs(b.RangePct-pct) < 1e-9 {
			return b.Bid, b.Ask, true, b.Complete
		}
	}
	if math.Abs(pct-headline) < 1e-9 && c.Bid+c.Ask > 0 {
		return c.Bid, c.Ask, true, c.Complete
	}
	return 0, 0, false, false
}

func bandLeaderNames(rows []PerpDepthOverviewBandCoin, n int) []string {
	out := make([]string, 0, n)
	for _, r := range rows {
		if len(out) >= n {
			break
		}
		if r.Base == "" || r.Total <= 0 {
			continue
		}
		out = append(out, r.Base)
	}
	return out
}
