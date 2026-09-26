package domain

import (
	"fmt"
	"math"
	"sort"
	"time"
)

const (
	BotTrackerWindow      = 5 * time.Minute
	BotTrackerBurst       = 60 * time.Second
	BotTrackerMinCount    = 4
	BotTrackerMinNotional = 5000.0
	botTrackerSizeFactor  = 1.002 // prints within about 0.2% share a size
	BotTrackerMaxClusters = 8
)

// BotCluster is one size that repeated on one side inside a short burst.
type BotCluster struct {
	Exchange      Exchange
	Side          string
	Notional      float64
	Count         int
	Span          time.Duration
	From          time.Time
	To            time.Time
	TotalNotional float64
}

// BotTrackerVenue is repeated same-size market orders on one venue.
type BotTrackerVenue struct {
	Exchange  Exchange
	Symbol    string
	Clusters  []BotCluster
	BuyCount  int
	SellCount int
	Delta     int // buy prints minus sell prints inside the flagged bursts
	Summary   string
	Error     string
}

// BotTrackerReport is the API result.
type BotTrackerReport struct {
	Symbol    string
	Exchange  string
	AsOf      time.Time
	Window    string
	Burst     string
	MinCount  int
	Venues    []BotTrackerVenue
	Clusters  []BotCluster
	BuyCount  int
	SellCount int
	Delta     int
	Summary   string
	Note      string
}

// BuildBotTracker flags sizes that repeat at least BotTrackerMinCount times
// inside any BotTrackerBurst during the last BotTrackerWindow.
func BuildBotTracker(ex Exchange, symbol string, prints []TakerPrint, now time.Time) BotTrackerVenue {
	symbol = NormalizeLiquidationSymbol(symbol)
	now = now.UTC()
	cut := now.Add(-BotTrackerWindow)
	out := BotTrackerVenue{Exchange: ex, Symbol: symbol, Clusters: []BotCluster{}}
	groups := map[string][]TakerPrint{}
	for _, p := range prints {
		if p.Notional < BotTrackerMinNotional || p.Time.IsZero() || p.Time.After(now) || p.Time.Before(cut) {
			continue
		}
		side := p.Side
		if side != TakerSideBuy && side != TakerSideSell {
			continue
		}
		key := fmt.Sprintf("%s:%d", side, botSizeKey(p.Notional))
		p.Time = p.Time.UTC()
		groups[key] = append(groups[key], p)
	}
	for _, g := range groups {
		if c, ok := densestBotBurst(ex, g); ok {
			out.Clusters = append(out.Clusters, c)
			if c.Side == TakerSideBuy {
				out.BuyCount += c.Count
			} else {
				out.SellCount += c.Count
			}
		}
	}
	sort.Slice(out.Clusters, func(i, j int) bool {
		if out.Clusters[i].Count != out.Clusters[j].Count {
			return out.Clusters[i].Count > out.Clusters[j].Count
		}
		return out.Clusters[i].Notional > out.Clusters[j].Notional
	})
	if len(out.Clusters) > BotTrackerMaxClusters {
		out.Clusters = out.Clusters[:BotTrackerMaxClusters]
		out.BuyCount, out.SellCount = 0, 0
		for _, c := range out.Clusters {
			if c.Side == TakerSideBuy {
				out.BuyCount += c.Count
			} else {
				out.SellCount += c.Count
			}
		}
	}
	out.Delta = out.BuyCount - out.SellCount
	out.Summary = explainBotVenue(out)
	return out
}

// MergeBotClusters keeps the busiest bursts across venues.
func MergeBotClusters(venues []BotTrackerVenue) []BotCluster {
	var all []BotCluster
	for _, v := range venues {
		if v.Error != "" {
			continue
		}
		all = append(all, v.Clusters...)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Count != all[j].Count {
			return all[i].Count > all[j].Count
		}
		return all[i].Notional > all[j].Notional
	})
	if len(all) > BotTrackerMaxClusters {
		all = all[:BotTrackerMaxClusters]
	}
	return all
}

func botSizeKey(n float64) int64 {
	if n <= 0 {
		return 0
	}
	return int64(math.Round(math.Log(n) / math.Log(botTrackerSizeFactor)))
}

func densestBotBurst(ex Exchange, prints []TakerPrint) (BotCluster, bool) {
	if len(prints) < BotTrackerMinCount {
		return BotCluster{}, false
	}
	sort.Slice(prints, func(i, j int) bool { return prints[i].Time.Before(prints[j].Time) })
	bestN, bestL, bestR := 0, 0, 0
	l := 0
	for r := range prints {
		for prints[r].Time.Sub(prints[l].Time) > BotTrackerBurst {
			l++
		}
		if n := r - l + 1; n > bestN {
			bestN, bestL, bestR = n, l, r
		}
	}
	if bestN < BotTrackerMinCount {
		return BotCluster{}, false
	}
	slice := prints[bestL : bestR+1]
	var sum float64
	vals := make([]float64, len(slice))
	for i, p := range slice {
		sum += p.Notional
		vals[i] = p.Notional
	}
	sort.Float64s(vals)
	return BotCluster{
		Exchange: ex, Side: slice[0].Side, Notional: vals[len(vals)/2],
		Count: bestN, Span: slice[len(slice)-1].Time.Sub(slice[0].Time),
		From: slice[0].Time, To: slice[len(slice)-1].Time, TotalNotional: sum,
	}, true
}

func explainBotVenue(v BotTrackerVenue) string {
	if v.Error != "" {
		return v.Error
	}
	if len(v.Clusters) == 0 {
		return fmt.Sprintf("No same-size market order repeated %d or more times inside %s during the last %s.",
			BotTrackerMinCount, BotTrackerBurst, BotTrackerWindow)
	}
	parts := make([]string, 0, len(v.Clusters))
	limit := len(v.Clusters)
	if limit > 3 {
		limit = 3
	}
	for _, c := range v.Clusters[:limit] {
		parts = append(parts, fmt.Sprintf("%s %s × %d in %s", c.Side, formatBotNotional(c.Notional), c.Count, formatBotSpan(c.Span)))
	}
	head := "Repeated size: " + joinBot(parts) + "."
	if v.Delta != 0 {
		side := "buy"
		n := v.Delta
		if n < 0 {
			side = "sell"
			n = -n
		}
		head += fmt.Sprintf(" Net %d more bot %s prints.", n, side)
	}
	return head
}

func ExplainBotReport(clusters []BotCluster, delta int) string {
	if len(clusters) == 0 {
		return fmt.Sprintf("No same-size market order repeated %d or more times inside %s during the last %s.",
			BotTrackerMinCount, BotTrackerBurst, BotTrackerWindow)
	}
	v := BotTrackerVenue{Clusters: clusters, Delta: delta}
	for _, c := range clusters {
		if c.Side == TakerSideBuy {
			v.BuyCount += c.Count
		} else {
			v.SellCount += c.Count
		}
	}
	return explainBotVenue(v)
}

func formatBotNotional(v float64) string {
	abs := math.Abs(v)
	switch {
	case abs >= 1_000_000:
		return trimBotFloat(abs/1_000_000) + "M"
	case abs >= 1_000:
		return trimBotFloat(abs/1_000) + "K"
	default:
		return trimBotFloat(abs)
	}
}

func trimBotFloat(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	for len(s) > 0 && s[len(s)-1] == '0' {
		s = s[:len(s)-1]
	}
	if len(s) > 0 && s[len(s)-1] == '.' {
		s = s[:len(s)-1]
	}
	return s
}

func formatBotSpan(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	sec := int(d.Round(time.Second) / time.Second)
	if sec < 60 {
		return fmt.Sprintf("%ds", sec)
	}
	return fmt.Sprintf("%dm%ds", sec/60, sec%60)
}

func joinBot(parts []string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	case 2:
		return parts[0] + " and " + parts[1]
	default:
		out := parts[0]
		for _, p := range parts[1 : len(parts)-1] {
			out += ", " + p
		}
		return out + ", and " + parts[len(parts)-1]
	}
}
