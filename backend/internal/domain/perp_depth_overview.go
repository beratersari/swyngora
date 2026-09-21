package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	PerpDepthOverviewDefaultN = 20
	PerpDepthOverviewMinN     = 5
	PerpDepthOverviewMaxN     = 40
	perpDepthOverviewLeaders  = 5
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
	Lean      string
	Coins     []PerpDepthCoinShare
	Leaders   []string
	Missing   []string
	Summary   string
	Note      string
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
		Note:     "Resting USD-M / linear perpetual depth across the most liquid USDT coins we follow. Bid is notional within N% below each coin's own mid; ask is notional within N% above. Combined exchange=all adds Binance and Bybit for each coin. Informational only — not financial advice.",
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
