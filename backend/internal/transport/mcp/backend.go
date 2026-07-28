package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/watchlist"
)

// Backend is an in-process facade for MCP tools (no second process, no self-HTTP).
type Backend struct {
	Market *market.Service
	Watch  *watchlist.Service
}

func (b *Backend) GetTicker(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	tkr, err := b.Market.GetTicker24h(ctx, exchange, symbol)
	if err != nil {
		return nil, err
	}
	ex, _ := b.Market.ResolveExchange(exchange)
	return mustJSON(map[string]any{
		"exchange":           string(ex),
		"symbol":             tkr.Symbol,
		"priceChange":        tkr.PriceChange,
		"priceChangePercent": tkr.PriceChangePercent,
		"lastPrice":          tkr.LastPrice,
		"openPrice":          tkr.OpenPrice,
		"highPrice":          tkr.HighPrice,
		"lowPrice":           tkr.LowPrice,
		"volume":             tkr.Volume,
		"quoteVolume":        tkr.QuoteVolume,
		"openTime":           tkr.OpenTime.UTC().Format(time.RFC3339Nano),
		"closeTime":          tkr.CloseTime.UTC().Format(time.RFC3339Nano),
		"tradeCount":         tkr.TradeCount,
	})
}

func (b *Backend) GetCandles(ctx context.Context, exchange, symbol, interval string, limit int) (json.RawMessage, error) {
	candles, err := b.Market.GetCandles(ctx, exchange, symbol, interval, limit, nil, nil)
	if err != nil {
		return nil, err
	}
	ex, _ := b.Market.ResolveExchange(exchange)
	items := make([]map[string]any, 0, len(candles))
	for _, c := range candles {
		items = append(items, map[string]any{
			"openTime":    c.OpenTime.UTC().Format(time.RFC3339Nano),
			"open":        c.Open,
			"high":        c.High,
			"low":         c.Low,
			"close":       c.Close,
			"volume":      c.Volume,
			"closeTime":   c.CloseTime.UTC().Format(time.RFC3339Nano),
			"quoteVolume": c.QuoteVolume,
			"tradeCount":  c.TradeCount,
		})
	}
	return mustJSON(map[string]any{
		"exchange": string(ex),
		"symbol":   strings.ToUpper(strings.TrimSpace(symbol)),
		"interval": interval,
		"candles":  items,
	})
}

func (b *Backend) GetSupply(ctx context.Context, asset string) (json.RawMessage, error) {
	sup, err := b.Market.GetSupply(ctx, asset)
	if err != nil {
		return nil, err
	}
	return mustJSON(map[string]any{
		"asset":             sup.Asset,
		"name":              sup.Name,
		"providerId":        sup.ProviderID,
		"circulatingSupply": sup.CirculatingSupply,
		"totalSupply":       sup.TotalSupply,
		"maxSupply":         sup.MaxSupply,
		"currentPriceUsd":   sup.CurrentPriceUSD,
		"asOf":              sup.AsOf.UTC().Format(time.RFC3339Nano),
		"source":            sup.Source,
	})
}

func (b *Backend) ListSpot(ctx context.Context, exchange, query, quote, sort, order, tag string, limit, offset int) (json.RawMessage, error) {
	var tags []string
	if tag != "" {
		tags = []string{tag}
	}
	res, err := b.Market.ListSpotMarkets(ctx, exchange, domain.SpotListQuery{
		Query:      query,
		QuoteAsset: quote,
		Tags:       tags,
		SortBy:     domain.SpotSortField(sort),
		Order:      domain.SortOrder(order),
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(res.Items))
	for _, m := range res.Items {
		items = append(items, map[string]any{
			"symbol":               m.Symbol,
			"lastPrice":            m.LastPrice,
			"priceChangePercent":   m.PriceChangePercent,
			"volume":               m.Volume,
			"quoteVolume":          m.QuoteVolume,
			"tradeCount":           m.TradeCount,
			"tags":                 m.Tags,
			"circulatingSupply":    m.CirculatingSupply,
			"totalSupply":          m.TotalSupply,
			"maxSupply":            m.MaxSupply,
			"marketCapCirculating": m.MarketCapCirculating,
			"marketCapTotal":       m.MarketCapTotal,
			"marketCapMax":         m.MarketCapMax,
			"marketCapMaxInfinite": m.MarketCapMaxInfinite,
		})
	}
	return mustJSON(map[string]any{
		"exchange": string(res.Exchange),
		"query":    res.Query,
		"sort":     string(res.SortBy),
		"order":    string(res.Order),
		"total":    res.Total,
		"limit":    res.Limit,
		"offset":   res.Offset,
		"items":    items,
	})
}

func (b *Backend) GetIndicators(ctx context.Context, exchange, symbol, interval string, limit, rsiPeriod int, emaPeriodsCSV string) (json.RawMessage, error) {
	var periods []int
	if emaPeriodsCSV != "" {
		for _, p := range strings.Split(emaPeriodsCSV, ",") {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			var n int
			if _, err := fmt.Sscanf(p, "%d", &n); err == nil {
				periods = append(periods, n)
			}
		}
	}
	ser, err := b.Market.GetIndicators(ctx, exchange, symbol, interval, limit, rsiPeriod, periods)
	if err != nil {
		return nil, err
	}
	latestEMA := map[string]float64{}
	for k, v := range ser.LatestEMA {
		if v != nil {
			latestEMA[fmt.Sprintf("%d", k)] = *v
		}
	}
	points := make([]map[string]any, 0, len(ser.Points))
	for _, p := range ser.Points {
		ema := map[string]float64{}
		for k, v := range p.EMA {
			if v != nil {
				ema[fmt.Sprintf("%d", k)] = *v
			}
		}
		points = append(points, map[string]any{
			"openTime": p.OpenTime.UTC().Format(time.RFC3339Nano),
			"close":    p.Close,
			"rsi":      p.RSI,
			"ema":      ema,
		})
	}
	return mustJSON(map[string]any{
		"exchange":   string(ser.Exchange),
		"symbol":     ser.Symbol,
		"interval":   string(ser.Interval),
		"rsiPeriod":  ser.RSIPeriod,
		"emaPeriods": ser.EMAPeriods,
		"latest": map[string]any{
			"rsi": ser.LatestRSI,
			"ema": latestEMA,
		},
		"points": points,
		"note":   "Informational only — not financial advice.",
	})
}

func (b *Backend) ListExchanges(ctx context.Context) (json.RawMessage, error) {
	_ = ctx
	exs := b.Market.ListExchanges()
	out := make([]string, len(exs))
	for i, e := range exs {
		out[i] = string(e)
	}
	return mustJSON(map[string]any{
		"exchanges": out,
		"default":   string(domain.DefaultExchange),
	})
}

func (b *Backend) GetWatchlist(ctx context.Context, clientID string) (json.RawMessage, error) {
	if b.Watch == nil {
		return nil, fmt.Errorf("watchlist not configured")
	}
	wl, err := b.Watch.Get(ctx, clientID)
	if err != nil {
		return nil, err
	}
	return watchlistJSON(wl)
}

func (b *Backend) AddWatchlistItem(ctx context.Context, clientID, exchange, symbol, note string) (json.RawMessage, error) {
	if b.Watch == nil {
		return nil, fmt.Errorf("watchlist not configured")
	}
	wl, err := b.Watch.Add(ctx, clientID, exchange, symbol, note)
	if err != nil {
		return nil, err
	}
	return watchlistJSON(wl)
}

func (b *Backend) RemoveWatchlistItem(ctx context.Context, clientID, exchange, symbol string) (json.RawMessage, error) {
	if b.Watch == nil {
		return nil, fmt.Errorf("watchlist not configured")
	}
	wl, err := b.Watch.Remove(ctx, clientID, exchange, symbol)
	if err != nil {
		return nil, err
	}
	return watchlistJSON(wl)
}

func (b *Backend) Health(ctx context.Context) (json.RawMessage, error) {
	_ = ctx
	return mustJSON(map[string]any{
		"status": "ok",
		"mcp":    "in-process",
		"time":   time.Now().UTC().Format(time.RFC3339Nano),
	})
}

func (b *Backend) DetectPumpEvents(ctx context.Context, args map[string]any) (json.RawMessage, error) {
	q := market.PumpQuery{
		Exchange:  strArg(args, "exchange", "binance"),
		Symbol:    strArg(args, "symbol", ""),
		Interval:  strArg(args, "interval", "1h"),
		Mode:      domain.PumpDetectMode(strArg(args, "mode", "close_return")),
		Direction: domain.PumpDirection(strArg(args, "direction", "up")),
	}
	q.LookbackHours = floatArg(args, "lookbackHours", 0)
	q.Limit = intArg(args, "limit", 0)
	q.MinReturnPct = floatArg(args, "minReturnPct", 5)
	q.WindowBars = intArg(args, "windowBars", 1)
	q.MinVolumeRatio = floatArg(args, "minVolumeRatio", 0)
	q.MaxEvents = intArg(args, "maxEvents", 20)
	if s := strArg(args, "startTime", ""); s != "" {
		t, err := time.Parse(time.RFC3339Nano, s)
		if err != nil {
			t, err = time.Parse(time.RFC3339, s)
		}
		if err != nil {
			return nil, fmt.Errorf("%w: startTime: %v", domain.ErrInvalidArgument, err)
		}
		q.StartTime = &t
	}
	if s := strArg(args, "endTime", ""); s != "" {
		t, err := time.Parse(time.RFC3339Nano, s)
		if err != nil {
			t, err = time.Parse(time.RFC3339, s)
		}
		if err != nil {
			return nil, fmt.Errorf("%w: endTime: %v", domain.ErrInvalidArgument, err)
		}
		q.EndTime = &t
	}
	res, err := b.Market.DetectPumpEvents(ctx, q)
	if err != nil {
		return nil, err
	}
	events := make([]map[string]any, 0, len(res.Events))
	for _, e := range res.Events {
		events = append(events, map[string]any{
			"index":       e.Index,
			"openTime":    e.OpenTime.UTC().Format(time.RFC3339Nano),
			"closeTime":   e.CloseTime.UTC().Format(time.RFC3339Nano),
			"startPrice":  e.StartPrice,
			"endPrice":    e.EndPrice,
			"returnPct":   e.ReturnPct,
			"high":        e.High,
			"low":         e.Low,
			"volume":      e.Volume,
			"volumeRatio": e.VolumeRatio,
			"mode":        string(e.Mode),
			"windowBars":  e.WindowBars,
		})
	}
	return mustJSON(map[string]any{
		"exchange":      string(res.Exchange),
		"symbol":        res.Symbol,
		"interval":      res.Interval,
		"lookbackHours": res.LookbackHours,
		"barsAnalyzed":  res.BarsAnalyzed,
		"minReturnPct":  res.MinReturnPct,
		"windowBars":    res.WindowBars,
		"mode":          string(res.Mode),
		"direction":     string(res.Direction),
		"events":        events,
		"eventCount":    len(events),
		"note":          res.Note,
	})
}

func (b *Backend) ScanPumpEvents(ctx context.Context, args map[string]any) (json.RawMessage, error) {
	q := market.PumpScanQuery{
		Exchange:       strArg(args, "exchange", "binance"),
		QuoteAsset:     strArg(args, "quote", "USDT"),
		Interval:       strArg(args, "interval", "15m"),
		LookbackHours:  floatArg(args, "lookbackHours", 24),
		MinReturnPct:   floatArg(args, "minReturnPct", 8),
		WindowBars:     intArg(args, "windowBars", 1),
		Mode:           domain.PumpDetectMode(strArg(args, "mode", "close_return")),
		Direction:      domain.PumpDirection(strArg(args, "direction", "up")),
		MinVolumeRatio: floatArg(args, "minVolumeRatio", 0),
		SymbolLimit:    intArg(args, "symbolLimit", 15),
		MaxTotalEvents: intArg(args, "maxTotalEvents", 30),
	}
	res, err := b.Market.ScanPumpEvents(ctx, q)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(res.Hits))
	totalEvents := 0
	for _, h := range res.Hits {
		totalEvents += len(h.Events)
		evs := make([]map[string]any, 0, len(h.Events))
		for _, e := range h.Events {
			evs = append(evs, map[string]any{
				"openTime":    e.OpenTime.UTC().Format(time.RFC3339Nano),
				"returnPct":   e.ReturnPct,
				"startPrice":  e.StartPrice,
				"endPrice":    e.EndPrice,
				"volumeRatio": e.VolumeRatio,
				"mode":        string(e.Mode),
			})
		}
		items = append(items, map[string]any{
			"symbol":        h.Symbol,
			"exchange":      string(h.Exchange),
			"interval":      h.Interval,
			"bestReturnPct": h.BestReturnPct,
			"events":        evs,
		})
	}
	return mustJSON(map[string]any{
		"exchange":       string(res.Exchange),
		"quote":          res.QuoteAsset,
		"interval":       res.Interval,
		"lookbackHours":  res.LookbackHours,
		"minReturnPct":   res.MinReturnPct,
		"windowBars":     res.WindowBars,
		"mode":           string(res.Mode),
		"direction":      string(res.Direction),
		"symbolLimit":    res.SymbolLimit,
		"maxTotalEvents": res.MaxTotalEvents,
		"hits":           items,
		"hitCount":       len(items),
		"eventCount":     totalEvents,
		"note":           res.Note,
	})
}

func strArg(m map[string]any, k, def string) string {
	if m == nil {
		return def
	}
	v, ok := m[k]
	if !ok || v == nil {
		return def
	}
	switch t := v.(type) {
	case string:
		if t == "" {
			return def
		}
		return t
	default:
		return fmt.Sprint(t)
	}
}

func floatArg(m map[string]any, k string, def float64) float64 {
	if m == nil {
		return def
	}
	v, ok := m[k]
	if !ok || v == nil {
		return def
	}
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case json.Number:
		f, err := t.Float64()
		if err != nil {
			return def
		}
		return f
	case string:
		var f float64
		if _, err := fmt.Sscanf(t, "%f", &f); err == nil {
			return f
		}
		return def
	default:
		return def
	}
}

func intArg(m map[string]any, k string, def int) int {
	return int(floatArg(m, k, float64(def)))
}

func watchlistJSON(wl *domain.Watchlist) (json.RawMessage, error) {
	items := make([]map[string]any, 0, len(wl.Items))
	for _, it := range wl.Items {
		items = append(items, map[string]any{
			"exchange": string(it.Exchange),
			"symbol":   it.Symbol,
			"note":     it.Note,
			"addedAt":  it.AddedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	return mustJSON(map[string]any{
		"clientId":  wl.ClientID,
		"items":     items,
		"updatedAt": wl.Updated.UTC().Format(time.RFC3339Nano),
	})
}

func mustJSON(v any) (json.RawMessage, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}
