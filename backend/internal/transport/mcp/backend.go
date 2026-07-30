package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/portfolio"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/pricealert"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/watchlist"
)

// Backend is an in-process facade for MCP tools (no second process, no self-HTTP).
type Backend struct {
	Market    *market.Service
	Watch     *watchlist.Service
	Alerts    *pricealert.Service
	Portfolio *portfolio.Service
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

func (b *Backend) ListPriceAlerts(ctx context.Context, clientID string) (json.RawMessage, error) {
	if b.Alerts == nil {
		return nil, fmt.Errorf("%w: alerts not configured", domain.ErrUpstream)
	}
	list, err := b.Alerts.List(ctx, clientID)
	if err != nil {
		return nil, err
	}
	return alertsJSON(clientID, list)
}

func (b *Backend) CreatePriceAlert(ctx context.Context, clientID, exchange, symbol, condition string, targetPrice float64, mode string) (json.RawMessage, error) {
	if b.Alerts == nil {
		return nil, fmt.Errorf("%w: alerts not configured", domain.ErrUpstream)
	}
	a, err := b.Alerts.Create(ctx, pricealert.CreateInput{
		ClientID:    clientID,
		Exchange:    exchange,
		Symbol:      symbol,
		Condition:   condition,
		TargetPrice: targetPrice,
		Mode:        mode,
	})
	if err != nil {
		return nil, err
	}
	return alertJSON(a)
}

func (b *Backend) DeletePriceAlert(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.Alerts == nil {
		return nil, fmt.Errorf("%w: alerts not configured", domain.ErrUpstream)
	}
	if err := b.Alerts.Delete(ctx, clientID, id); err != nil {
		return nil, err
	}
	return mustJSON(map[string]any{"deleted": true, "id": id})
}

func (b *Backend) GetAlertWebhook(ctx context.Context, clientID string) (json.RawMessage, error) {
	if b.Alerts == nil {
		return nil, fmt.Errorf("%w: alerts not configured", domain.ErrUpstream)
	}
	wh, err := b.Alerts.GetWebhook(ctx, clientID)
	if err != nil {
		return nil, err
	}
	return webhookJSON(wh)
}

func (b *Backend) SetAlertWebhook(ctx context.Context, clientID, url string) (json.RawMessage, error) {
	return b.SetAlertWebhookWithMode(ctx, clientID, url, string(domain.DeliveryImmediate))
}

func (b *Backend) SetAlertWebhookWithMode(ctx context.Context, clientID, url, deliveryMode string) (json.RawMessage, error) {
	return b.SetAlertWebhookSettings(ctx, clientID, url, deliveryMode, "UTC", false, "", "")
}

func (b *Backend) SetAlertWebhookSettings(ctx context.Context, clientID, url, deliveryMode, timeZone string, quietEnabled bool, quietStart, quietEnd string) (json.RawMessage, error) {
	if b.Alerts == nil {
		return nil, fmt.Errorf("%w: alerts not configured", domain.ErrUpstream)
	}
	wh, err := b.Alerts.SetWebhook(ctx, clientID, domain.WebhookSettings{
		URL:               url,
		DeliveryMode:      deliveryMode,
		TimeZone:          timeZone,
		QuietHoursEnabled: quietEnabled,
		QuietStart:        quietStart,
		QuietEnd:          quietEnd,
	})
	if err != nil {
		return nil, err
	}
	return webhookJSON(wh)
}

func (b *Backend) DeleteAlertWebhook(ctx context.Context, clientID string) (json.RawMessage, error) {
	if b.Alerts == nil {
		return nil, fmt.Errorf("%w: alerts not configured", domain.ErrUpstream)
	}
	if err := b.Alerts.DeleteWebhook(ctx, clientID); err != nil {
		return nil, err
	}
	return mustJSON(map[string]any{"clientId": clientID, "deleted": true, "configured": false})
}

func (b *Backend) CreatePortfolio(ctx context.Context, clientID string, startingBalance float64, currency string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	if _, err := b.Portfolio.Create(ctx, portfolio.CreateInput{
		ClientID: clientID, StartingBalance: startingBalance, Currency: currency,
	}); err != nil {
		return nil, err
	}
	return b.GetPortfolio(ctx, clientID)
}

func (b *Backend) GetPortfolio(ctx context.Context, clientID string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	v, err := b.Portfolio.View(ctx, clientID)
	if err != nil {
		return nil, err
	}
	return portfolioViewJSON(v)
}

func (b *Backend) PlacePortfolioOrder(ctx context.Context, clientID, exchange, symbol, side string, quantity float64) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	tr, v, err := b.Portfolio.PlaceOrder(ctx, portfolio.OrderInput{
		ClientID: clientID, Exchange: exchange, Symbol: symbol, Side: side, Quantity: quantity,
	})
	if err != nil {
		return nil, err
	}
	pv, err := portfolioViewJSON(v)
	if err != nil {
		return nil, err
	}
	var pmap map[string]any
	_ = json.Unmarshal(pv, &pmap)
	return mustJSON(map[string]any{
		"trade": map[string]any{
			"id": tr.ID, "exchange": string(tr.Exchange), "symbol": tr.Symbol, "side": string(tr.Side),
			"quantity": tr.Quantity, "price": tr.Price, "notional": tr.Notional, "realizedPnL": tr.RealizedPnL,
			"createdAt": tr.CreatedAt.UTC().Format(time.RFC3339Nano),
		},
		"portfolio": pmap,
		"note":      v.Note,
	})
}

func (b *Backend) ListPortfolioTrades(ctx context.Context, clientID string, limit, offset int) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	list, total, err := b.Portfolio.ListTrades(ctx, clientID, limit, offset)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(list))
	for _, t := range list {
		items = append(items, map[string]any{
			"id": t.ID, "exchange": string(t.Exchange), "symbol": t.Symbol, "side": string(t.Side),
			"quantity": t.Quantity, "price": t.Price, "notional": t.Notional, "realizedPnL": t.RealizedPnL,
			"pendingOrderId": t.PendingOrderID,
			"createdAt":      t.CreatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	return mustJSON(map[string]any{"clientId": clientID, "trades": items, "count": len(items), "total": total})
}

func (b *Backend) PlacePortfolioPendingOrder(ctx context.Context, clientID, exchange, symbol, orderType string, quantity, triggerPrice float64, timeInForce, expiresAt string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	var exp *time.Time
	if expiresAt != "" {
		t, err := time.Parse(time.RFC3339Nano, expiresAt)
		if err != nil {
			t, err = time.Parse(time.RFC3339, expiresAt)
		}
		if err != nil {
			return nil, fmt.Errorf("%w: expiresAt must be RFC3339", domain.ErrInvalidArgument)
		}
		tu := t.UTC()
		exp = &tu
	}
	o, err := b.Portfolio.PlacePendingOrder(ctx, portfolio.PendingOrderInput{
		ClientID: clientID, Exchange: exchange, Symbol: symbol, Type: orderType,
		Quantity: quantity, TriggerPrice: triggerPrice, TimeInForce: timeInForce, ExpiresAt: exp,
	})
	if err != nil {
		return nil, err
	}
	return mustJSON(map[string]any{
		"type":  string(o.Type),
		"order": pendingOrderMap(o),
		"note":  "Paper pending order (GTC/IOC/FOK) with reservations. Not real money.",
	})
}

func (b *Backend) ListPortfolioOrders(ctx context.Context, clientID, status string, limit, offset int) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	list, err := b.Portfolio.ListPendingOrders(ctx, clientID, status, limit, offset)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(list))
	for i := range list {
		items = append(items, pendingOrderMap(&list[i]))
	}
	st := status
	if st == "" {
		st = "open"
	}
	return mustJSON(map[string]any{"clientId": clientID, "orders": items, "count": len(items), "status": st})
}

func (b *Backend) CancelPortfolioOrder(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	o, err := b.Portfolio.CancelPendingOrder(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	return mustJSON(map[string]any{"order": pendingOrderMap(o), "note": "Order canceled; unused reservation released; it will not execute."})
}

func pendingOrderMap(o *domain.PendingOrder) map[string]any {
	tif := string(o.TimeInForce)
	if tif == "" {
		tif = string(domain.TimeInForceGTC)
	}
	m := map[string]any{
		"id": o.ID, "clientId": o.ClientID, "exchange": string(o.Exchange), "symbol": o.Symbol,
		"type": string(o.Type), "side": string(o.Side), "quantity": o.Quantity,
		"filledQuantity": o.FilledQuantity, "remainingQuantity": o.RemainingQuantity,
		"triggerPrice": o.TriggerPrice, "reservedCash": o.ReservedCash, "reservedQuantity": o.ReservedQuantity,
		"timeInForce": tif,
		"status":      string(o.Status),
		"createdAt":   o.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updatedAt":   o.UpdatedAt.UTC().Format(time.RFC3339Nano),
		"fillTradeId": o.FillTradeID, "fillPrice": o.FillPrice, "rejectReason": o.RejectReason, "cancelReason": o.CancelReason,
	}
	if o.ExpiresAt != nil {
		m["expiresAt"] = o.ExpiresAt.UTC().Format(time.RFC3339Nano)
	}
	if o.FilledAt != nil {
		m["filledAt"] = o.FilledAt.UTC().Format(time.RFC3339Nano)
	}
	if o.CanceledAt != nil {
		m["canceledAt"] = o.CanceledAt.UTC().Format(time.RFC3339Nano)
	}
	return m
}

func portfolioViewJSON(v *domain.PortfolioView) (json.RawMessage, error) {
	pos := make([]map[string]any, 0, len(v.Positions))
	for _, p := range v.Positions {
		pos = append(pos, map[string]any{
			"exchange": string(p.Exchange), "symbol": p.Symbol, "quantity": p.Quantity,
			"reservedQuantity": p.ReservedQuantity, "availableQuantity": p.AvailableQuantity,
			"avgCost": p.AvgCost, "markPrice": p.MarkPrice, "marketValue": p.MarketValue,
			"unrealizedPnL": p.UnrealizedPnL, "costBasis": p.CostBasis,
		})
	}
	return mustJSON(map[string]any{
		"clientId": v.ClientID, "currency": v.Currency, "startingBalance": v.StartingBalance,
		"cashBalance": v.CashBalance, "reservedCash": v.ReservedCash, "availableCash": v.AvailableCash,
		"positionsValue": v.PositionsValue, "equity": v.Equity,
		"unrealizedPnL": v.UnrealizedPnL, "realizedPnLTotal": v.RealizedPnLTotal, "totalPnL": v.TotalPnL,
		"positions": pos, "note": v.Note,
		"createdAt": v.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updatedAt": v.UpdatedAt.UTC().Format(time.RFC3339Nano),
	})
}

func webhookJSON(wh *domain.ClientWebhook) (json.RawMessage, error) {
	mode := string(wh.DeliveryMode)
	if mode == "" {
		mode = string(domain.DeliveryImmediate)
	}
	tz := wh.TimeZone
	if tz == "" {
		tz = "UTC"
	}
	m := map[string]any{
		"clientId":     wh.ClientID,
		"url":          wh.URL,
		"deliveryMode": mode,
		"timeZone":     tz,
		"quietHours": map[string]any{
			"enabled": wh.QuietHoursEnabled,
			"start":   wh.QuietStart,
			"end":     wh.QuietEnd,
		},
		"configured": strings.TrimSpace(wh.URL) != "",
	}
	if !wh.UpdatedAt.IsZero() {
		m["updatedAt"] = wh.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}
	return mustJSON(m)
}

func alertJSON(a *domain.PriceAlert) (json.RawMessage, error) {
	mode := string(a.Mode)
	if mode == "" {
		mode = string(domain.AlertModeOneTime)
	}
	m := map[string]any{
		"id":          a.ID,
		"clientId":    a.ClientID,
		"exchange":    string(a.Exchange),
		"symbol":      a.Symbol,
		"condition":   string(a.Condition),
		"mode":        mode,
		"targetPrice": a.TargetPrice,
		"status":      string(a.Status),
		"createdAt":   a.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if a.Mode == domain.AlertModeRepeating {
		m["armed"] = a.Armed
	}
	if a.TriggeredAt != nil {
		m["triggeredAt"] = a.TriggeredAt.UTC().Format(time.RFC3339Nano)
	}
	if a.TriggeredPrice > 0 {
		m["triggeredPrice"] = a.TriggeredPrice
	}
	return mustJSON(m)
}

func alertsJSON(clientID string, list []domain.PriceAlert) (json.RawMessage, error) {
	items := make([]map[string]any, 0, len(list))
	for i := range list {
		raw, err := alertJSON(&list[i])
		if err != nil {
			return nil, err
		}
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return mustJSON(map[string]any{
		"clientId": clientID,
		"alerts":   items,
		"count":    len(items),
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
