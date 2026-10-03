package domain

// LiquidRecorderLimit is how many USDT perpetuals the background recorder
// keeps sampling. The original seed pairs stay inside this cap. The startup
// archive backfills this same list. Names outside it fill in from live samples.
// Binance books outside the seed list stay on the top 100 levels and a slower
// cadence so the wider set does not trip the shared REST pause.
const LiquidRecorderLimit = 100

// MergeLiquidSymbols keeps seeds first, then ranked names, until limit.
// Duplicate and blank symbols are dropped. Seeds win when the cap is tight.
func MergeLiquidSymbols(seeds, ranked []string, limit int) []string {
	if limit <= 0 {
		limit = LiquidRecorderLimit
	}
	out := make([]string, 0, limit)
	seen := map[string]struct{}{}
	add := func(raw string) {
		if len(out) >= limit {
			return
		}
		s := NormalizeLiquidationSymbol(raw)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	for _, s := range seeds {
		add(s)
	}
	for _, s := range ranked {
		add(s)
	}
	return out
}
