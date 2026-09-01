package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/apikey"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/dataimport"
	exportsvc "gitlab.com/trace-analysis/swyngora/backend/internal/service/export"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/fundingarb"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/portfolio"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/pricealert"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/pricediff"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/scanner"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/swing"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/watchlist"
)

// Backend is an in-process facade for MCP tools (no second process, no self-HTTP).
type Backend struct {
	Market     *market.Service
	Watch      *watchlist.Service
	Alerts     *pricealert.Service
	Portfolio  *portfolio.Service
	Scanner    *scanner.Service
	Export     *exportsvc.Service
	Import     *dataimport.Service
	PriceDiff  *pricediff.Service
	Swing      *swing.Service
	APIKeys    *apikey.Service
	FundingArb *fundingarb.Service
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
		"halted":             tkr.Halted,
	})
}

func (b *Backend) GetOrderBook(ctx context.Context, exchange, symbol, group string, limit int, rangePct float64) (json.RawMessage, error) {
	book, err := b.Market.GetSpotOrderBook(ctx, exchange, symbol, group, limit, rangePct)
	if err != nil {
		return nil, err
	}
	return mustJSON(map[string]any{
		"exchange":            string(book.Exchange),
		"symbol":              book.Symbol,
		"lastPrice":           book.LastPrice,
		"bestBid":             book.BestBid,
		"bestAsk":             book.BestAsk,
		"spread":              book.Spread,
		"spreadPct":           book.SpreadPct,
		"groupSize":           book.GroupSize,
		"suggestedGroupSizes": book.SuggestedGroupSizes,
		"levels":              book.Levels,
		"bids":                book.Bids,
		"asks":                book.Asks,
		"bidVolume":           book.BidVolume,
		"askVolume":           book.AskVolume,
		"imbalance":           book.Imbalance,
		"bidWalls":            book.BidWalls,
		"askWalls":            book.AskWalls,
		"analysis":            orderBookAnalysisMap(book.Analysis),
		"updatedAt":           book.UpdatedAt.UTC().Format(time.RFC3339Nano),
		"live":                book.Live,
		"source":              book.Source,
	})
}

func (b *Backend) GetBookHistory(ctx context.Context, exchange, symbol, at, from, to string, limit int) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	var atPtr, fromPtr, toPtr *time.Time
	if strings.TrimSpace(at) != "" {
		t, err := parseMCPTime(at)
		if err != nil {
			return nil, err
		}
		atPtr = &t
	}
	if strings.TrimSpace(from) != "" {
		t, err := parseMCPTime(from)
		if err != nil {
			return nil, err
		}
		fromPtr = &t
	}
	if strings.TrimSpace(to) != "" {
		t, err := parseMCPTime(to)
		if err != nil {
			return nil, err
		}
		toPtr = &t
	}
	got, err := b.Market.GetBookHistory(ctx, exchange, symbol, atPtr, fromPtr, toPtr, limit)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetIcebergs(ctx context.Context, exchange, symbol string, minNotional float64) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetIcebergs(ctx, exchange, symbol, minNotional)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) CompareBookHistory(ctx context.Context, exchange, symbol, from, to string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	fromT, err := parseMCPTime(from)
	if err != nil {
		return nil, err
	}
	toT, err := parseMCPTime(to)
	if err != nil {
		return nil, err
	}
	got, err := b.Market.CompareBookHistory(ctx, exchange, symbol, fromT, toT)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) EstimateOrderBookImpact(ctx context.Context, exchange, symbol, side string, quantity, notional float64) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.EstimateOrderBookImpact(ctx, exchange, symbol, side, quantity, notional)
	if err != nil {
		return nil, err
	}
	fills := make([]map[string]any, 0, len(got.Fills))
	for _, f := range got.Fills {
		fills = append(fills, map[string]any{
			"exchange": f.Exchange, "price": f.Price, "quantity": f.Quantity, "notional": f.Notional,
			"cumulativeQuantity": f.CumulativeQuantity, "cumulativeNotional": f.CumulativeNotional,
		})
	}
	return mustJSON(map[string]any{
		"symbol": got.Symbol, "scope": got.Scope, "side": got.Side,
		"midPrice": got.MidPrice, "bestPrice": got.BestPrice, "averagePrice": got.AveragePrice, "endPrice": got.EndPrice,
		"newBestPrice":      got.NewBestPrice,
		"requestedQuantity": got.RequestedQuantity, "requestedNotional": got.RequestedNotional,
		"filledQuantity": got.FilledQuantity, "spentNotional": got.SpentNotional,
		"unfilledQuantity": got.UnfilledQuantity, "unfilledNotional": got.UnfilledNotional,
		"visibleQuantity": got.VisibleQuantity, "visibleNotional": got.VisibleNotional,
		"slippagePct": got.SlippagePct, "slippageVsBestPct": got.SlippageVsBestPct, "impactPct": got.ImpactPct,
		"impactAvailable": got.ImpactAvailable, "impactNote": got.ImpactNote,
		"exhausted": got.Exhausted, "levelsUsed": got.LevelsUsed, "venueCount": got.VenueCount, "live": got.Live,
		"fills": fills,
		"note":  "Simulated market order walking live resting depth. Not a quote, fill, or financial advice. Visible book may be thinner than the real market.",
	})
}

func (b *Backend) GetOpenInterest(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetOpenInterest(ctx, exchange, symbol)
	if err != nil {
		return nil, err
	}
	wins := make([]map[string]any, 0, len(got.Windows))
	for _, w := range got.Windows {
		row := map[string]any{
			"window": w.Window, "openInterest": w.OpenInterest, "openInterestValue": w.OpenInterestValue,
			"change": w.Change, "changePct": w.ChangePct, "changeValue": w.ChangeValue, "changeValuePct": w.ChangeValuePct,
			"direction": w.Direction, "complete": w.Complete,
		}
		if !w.SampleTime.IsZero() {
			row["sampleTime"] = w.SampleTime.UTC().Format(time.RFC3339Nano)
		}
		wins = append(wins, row)
	}
	venues := make([]map[string]any, 0, len(got.Venues))
	for _, v := range got.Venues {
		vw := make([]map[string]any, 0, len(v.Windows))
		for _, w := range v.Windows {
			row := map[string]any{
				"window": w.Window, "openInterest": w.OpenInterest, "openInterestValue": w.OpenInterestValue,
				"change": w.Change, "changePct": w.ChangePct, "changeValue": w.ChangeValue, "changeValuePct": w.ChangeValuePct,
				"direction": w.Direction, "complete": w.Complete,
			}
			if !w.SampleTime.IsZero() {
				row["sampleTime"] = w.SampleTime.UTC().Format(time.RFC3339Nano)
			}
			vw = append(vw, row)
		}
		cur := map[string]any{"contracts": v.Current.Contracts, "value": v.Current.Value}
		if !v.Current.Time.IsZero() {
			cur["time"] = v.Current.Time.UTC().Format(time.RFC3339Nano)
		}
		venues = append(venues, map[string]any{"exchange": v.Exchange, "current": cur, "windows": vw})
	}
	cur := map[string]any{"contracts": got.Current.Contracts, "value": got.Current.Value}
	if !got.Current.Time.IsZero() {
		cur["time"] = got.Current.Time.UTC().Format(time.RFC3339Nano)
	}
	asOf := ""
	if !got.AsOf.IsZero() {
		asOf = got.AsOf.UTC().Format(time.RFC3339Nano)
	}
	out := map[string]any{
		"symbol": got.Symbol, "exchange": got.Exchange, "unit": got.Unit,
		"current": cur, "windows": wins, "venues": venues, "asOf": asOf, "venueCount": got.VenueCount,
		"note": "Binance USD-M and Bybit linear perpetual open interest. contracts is base-asset size (Bybit is single-sided). value is USDT notional. Change is current minus ~window ago. funding is predicted next rate plus recent settlements. Informational only.",
	}
	if got.Funding != nil {
		out["funding"] = fundingSnapshotMap(got.Funding)
	}
	if got.LongShort != nil {
		out["longShort"] = longShortSnapshotMap(got.LongShort)
	}
	return mustJSON(out)
}

func (b *Backend) GetLongShortRatio(ctx context.Context, exchange, symbol string, limit int) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetLongShortRatio(ctx, exchange, symbol, limit)
	if err != nil {
		return nil, err
	}
	return mustJSON(longShortSnapshotMap(got))
}

func (b *Backend) GetLiquidationHunt(ctx context.Context, exchange, symbol string, weights domain.HuntScoreWeights) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetLiquidationHuntWeighted(ctx, exchange, symbol, weights)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetLiquidationMaxPain(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetLiquidationMaxPain(ctx, exchange, symbol)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetLiquidationHuntHeatmap(ctx context.Context, exchange, symbol, rawRange string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetLiquidationHuntHeatmap(ctx, exchange, symbol, rawRange)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetSqueezeRisk(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetSqueezeRisk(ctx, exchange, symbol)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetPositioning(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetPositioning(ctx, exchange, symbol)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetVenueDivergence(ctx context.Context, symbol string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetVenueDivergence(ctx, symbol)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetTakerFlow(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetTakerFlow(ctx, exchange, symbol)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetCVD(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetCVD(ctx, exchange, symbol)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetVolumeSurge(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetVolumeSurge(ctx, exchange, symbol)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) ScanVolumeSurges(ctx context.Context, exchange, quote string, minRatio float64, limit int) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.ScanVolumeSurges(ctx, exchange, quote, minRatio, limit)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetRSIHeatmap(ctx context.Context, exchange, quote, intervals, sort string, limit, period int) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetRSIHeatmap(ctx, exchange, quote, intervals, sort, limit, period)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetLiquiditySweeps(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetLiquiditySweeps(ctx, exchange, symbol)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetAbsorption(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetAbsorption(ctx, exchange, symbol)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetAround(ctx context.Context, exchange, symbol, at, window, during string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	when, err := parseMCPTime(at)
	if err != nil {
		return nil, err
	}
	got, err := b.Market.GetAround(ctx, exchange, symbol, window, during, when)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) CompareAround(ctx context.Context, exchange, symbol, from, to, window, during string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	a, err := parseMCPTime(from)
	if err != nil {
		return nil, err
	}
	bAt, err := parseMCPTime(to)
	if err != nil {
		return nil, err
	}
	got, err := b.Market.CompareAround(ctx, exchange, symbol, window, during, a, bAt)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) FindAroundMoves(ctx context.Context, exchange, symbol, lookback, interval, direction string, minReturnPct float64, limit int, window, during string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.FindAroundMoves(ctx, exchange, symbol, lookback, interval, direction, minReturnPct, limit, window, during)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetAroundPrecursors(ctx context.Context, exchange, symbol, lookback, interval, direction string, minReturnPct float64, limit int, window, during string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetAroundPrecursors(ctx, exchange, symbol, lookback, interval, direction, minReturnPct, limit, window, during)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetAroundSimilar(ctx context.Context, exchange, symbol, lookback, interval, direction string, minReturnPct float64, limit int, window, during, fields, weights, minCoverage, horizons string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetAroundSimilar(ctx, exchange, symbol, lookback, interval, direction, minReturnPct, limit, window, during, fields, weights, minCoverage, horizons)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetVWAP(ctx context.Context, exchange, symbol, window, startTime, endTime string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	var startPtr, endPtr *time.Time
	if startTime != "" {
		t, err := parseMCPTime(startTime)
		if err != nil {
			return nil, err
		}
		startPtr = &t
	}
	if endTime != "" {
		t, err := parseMCPTime(endTime)
		if err != nil {
			return nil, err
		}
		endPtr = &t
	}
	got, err := b.Market.GetVWAP(ctx, exchange, symbol, window, startPtr, endPtr)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetVolumeProfile(ctx context.Context, exchange, symbol, window, startTime, endTime string, tickSize float64) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	var startPtr, endPtr *time.Time
	if startTime != "" {
		t, err := parseMCPTime(startTime)
		if err != nil {
			return nil, err
		}
		startPtr = &t
	}
	if endTime != "" {
		t, err := parseMCPTime(endTime)
		if err != nil {
			return nil, err
		}
		endPtr = &t
	}
	got, err := b.Market.GetVolumeProfile(ctx, exchange, symbol, window, startPtr, endPtr, tickSize)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetBasis(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetBasis(ctx, exchange, symbol)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetCorrelation(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetCorrelation(ctx, exchange, symbol)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetBreadth(ctx context.Context, exchange string, limit int) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetBreadth(ctx, exchange, limit)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetVolatility(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetVolatility(ctx, exchange, symbol)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetSnapshot(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetSnapshot(ctx, exchange, symbol)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetLevels(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetLevels(ctx, exchange, symbol)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetWhales(ctx context.Context, exchange, symbol string, minNotional float64, limit int) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetWhales(ctx, exchange, symbol, minNotional, limit)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetFuturesHistory(ctx context.Context, metric, exchange, symbol, from, to string, limit int) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	var fromPtr, toPtr *time.Time
	if strings.TrimSpace(from) != "" {
		t, err := parseMCPTime(from)
		if err != nil {
			return nil, err
		}
		fromPtr = &t
	}
	if strings.TrimSpace(to) != "" {
		t, err := parseMCPTime(to)
		if err != nil {
			return nil, err
		}
		toPtr = &t
	}
	got, err := b.Market.GetFuturesHistory(ctx, metric, exchange, symbol, fromPtr, toPtr, limit)
	if err != nil {
		return nil, err
	}
	return mustJSON(futuresHistoryMap(metric, exchange, symbol, got))
}

func futuresHistoryMap(metric, exchange, symbol string, raw any) map[string]any {
	metric, _ = domain.ParseFuturesMetric(metric)
	exchange, _ = domain.ParseOpenInterestExchange(exchange)
	symbol = domain.NormalizeLiquidationSymbol(symbol)
	out := map[string]any{
		"metric": metric, "exchange": exchange, "symbol": symbol,
		"note": "Durable SQLite history. Duplicates are ignored. One venue failing does not drop the other. Informational only.",
	}
	switch rows := raw.(type) {
	case []domain.FuturesSnapshot:
		items := make([]map[string]any, 0, len(rows))
		for _, r := range rows {
			item := map[string]any{
				"exchange":  string(r.Exchange),
				"sampledAt": r.SampledAt.UTC().Format(time.RFC3339Nano),
			}
			switch r.Metric {
			case domain.FuturesMetricOpenInterest:
				item["contracts"] = formatHistQty(r.Contracts)
				item["value"] = formatHistQty(r.Value)
			case domain.FuturesMetricFunding:
				dec, pct := domain.FormatFundingRate(r.FundingRate)
				item["rate"] = dec
				item["ratePct"] = pct
				item["predicted"] = r.Predicted
				if r.IntervalHours > 0 {
					item["intervalHours"] = r.IntervalHours
				}
			case domain.FuturesMetricLongShort:
				item["longPct"] = formatHistQty(r.LongShare * 100)
				item["shortPct"] = formatHistQty(r.ShortShare * 100)
				item["ratio"] = formatHistQty(r.Ratio)
				item["bias"] = domain.LongShortBias(r.Ratio)
			}
			items = append(items, item)
		}
		out["items"] = items
		out["count"] = len(items)
	case []domain.LiquidationEvent:
		items := make([]map[string]any, 0, len(rows))
		for _, e := range rows {
			items = append(items, map[string]any{
				"exchange": string(e.Exchange), "side": e.Side,
				"price": formatHistQty(e.Price), "quantity": formatHistQty(e.Quantity),
				"notional": formatHistQty(e.Notional),
				"time":     e.Time.UTC().Format(time.RFC3339Nano),
			})
		}
		out["items"] = items
		out["count"] = len(items)
	default:
		out["items"] = []any{}
		out["count"] = 0
	}
	return out
}

func formatHistQty(v float64) string {
	s := domain.FormatSignedQty(v)
	return strings.TrimPrefix(s, "+")
}

func parseMCPTime(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC(), nil
	}
	if ms, err := strconv.ParseInt(raw, 10, 64); err == nil {
		if ms > 1e12 {
			return time.UnixMilli(ms).UTC(), nil
		}
		return time.Unix(ms, 0).UTC(), nil
	}
	return time.Time{}, fmt.Errorf("%w: time must be RFC3339 or unix ms", domain.ErrInvalidArgument)
}

func longShortSnapshotMap(got *domain.LongShortSnapshot) map[string]any {
	if got == nil {
		return map[string]any{}
	}
	venues := make([]map[string]any, 0, len(got.Venues))
	for _, v := range got.Venues {
		venues = append(venues, map[string]any{
			"exchange": v.Exchange, "kind": v.Kind, "period": v.Period,
			"current": longShortLevelMap(v.Current), "change": v.Change,
			"history": longShortLevelsMap(v.History),
		})
	}
	asOf := ""
	if !got.AsOf.IsZero() {
		asOf = got.AsOf.UTC().Format(time.RFC3339Nano)
	}
	out := map[string]any{
		"symbol": got.Symbol, "exchange": got.Exchange, "kind": got.Kind, "period": got.Period,
		"venues": venues, "history": longShortLevelsMap(got.History),
		"asOf": asOf, "venueCount": got.VenueCount,
		"note": "Share of accounts that are long vs short (not position size). ratio is long/short. Combined all does not average venues.",
	}
	if got.Current != nil {
		out["current"] = longShortLevelMap(*got.Current)
	}
	return out
}

func longShortLevelMap(p domain.LongShortLevel) map[string]any {
	out := map[string]any{"longPct": p.LongPct, "shortPct": p.ShortPct, "ratio": p.Ratio, "bias": p.Bias, "longShare": p.LongShare}
	if !p.Time.IsZero() {
		out["time"] = p.Time.UTC().Format(time.RFC3339Nano)
	}
	return out
}

func longShortLevelsMap(in []domain.LongShortLevel) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, p := range in {
		out = append(out, longShortLevelMap(p))
	}
	return out
}

func (b *Backend) GetFundingArb(ctx context.Context, symbol string, notional, holdHours float64, feeBinancePct, feeBybitPct *float64) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetFundingArb(ctx, market.FundingArbParams{
		Symbol: symbol, Notional: notional, HoldHours: holdHours,
		FeeBinancePct: feeBinancePct, FeeBybitPct: feeBybitPct,
	})
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) ScanFundingArb(ctx context.Context, quote string, notional, holdHours float64, feeBinancePct, feeBybitPct *float64, limit int) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.ScanFundingArb(ctx, market.FundingArbScanParams{
		Quote: quote, Notional: notional, HoldHours: holdHours,
		FeeBinancePct: feeBinancePct, FeeBybitPct: feeBybitPct, SymbolLimit: limit,
	})
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetFundingArbHistory(ctx context.Context, symbol, fromRaw, toRaw string, notional float64, feeBinancePct, feeBybitPct *float64) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	from, err := parseFundingArbTime(fromRaw)
	if err != nil {
		return nil, fmt.Errorf("%w: from", domain.ErrInvalidArgument)
	}
	to, err := parseFundingArbTime(toRaw)
	if err != nil {
		return nil, fmt.Errorf("%w: to", domain.ErrInvalidArgument)
	}
	got, err := b.Market.GetFundingArbHistory(ctx, market.FundingArbHistoryParams{
		Symbol: symbol, From: from, To: to, Notional: notional,
		FeeBinancePct: feeBinancePct, FeeBybitPct: feeBybitPct,
	})
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) CreateFundingArbWatch(ctx context.Context, clientID, symbol string, notional, holdHours, minProfit float64, quote string, limit int, durationHours float64, feeBinancePct, feeBybitPct *float64) (json.RawMessage, error) {
	if b.FundingArb == nil {
		return nil, fmt.Errorf("%w: funding-arb watches not configured", domain.ErrUpstream)
	}
	got, err := b.FundingArb.CreateWatch(ctx, fundingarb.CreateInput{
		ClientID: clientID, Symbol: symbol, Notional: notional, HoldHours: holdHours, MinProfit: minProfit,
		Quote: quote, SymbolLimit: limit, DurationHours: durationHours,
		FeeBinancePct: feeBinancePct, FeeBybitPct: feeBybitPct,
	})
	if err != nil {
		return nil, err
	}
	return mustJSON(fundingArbWatchMap(got))
}

func (b *Backend) ListFundingArbWatches(ctx context.Context, clientID string) (json.RawMessage, error) {
	if b.FundingArb == nil {
		return nil, fmt.Errorf("%w: funding-arb watches not configured", domain.ErrUpstream)
	}
	got, err := b.FundingArb.ListWatches(ctx, clientID)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(got))
	for i := range got {
		items = append(items, fundingArbWatchMap(&got[i]))
	}
	return mustJSON(map[string]any{"watches": items, "count": len(items)})
}

func (b *Backend) GetFundingArbWatch(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.FundingArb == nil {
		return nil, fmt.Errorf("%w: funding-arb watches not configured", domain.ErrUpstream)
	}
	got, err := b.FundingArb.GetWatch(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	return mustJSON(fundingArbWatchMap(got))
}

func (b *Backend) UpdateFundingArbWatch(ctx context.Context, clientID, id string, notional, holdHours, minProfit *float64, quote *string, limit *int, durationHours *float64, feeBinancePct, feeBybitPct *float64) (json.RawMessage, error) {
	if b.FundingArb == nil {
		return nil, fmt.Errorf("%w: funding-arb watches not configured", domain.ErrUpstream)
	}
	got, err := b.FundingArb.UpdateWatch(ctx, fundingarb.UpdateInput{
		ClientID: clientID, ID: id,
		Notional: notional, HoldHours: holdHours, MinProfit: minProfit,
		Quote: quote, SymbolLimit: limit, DurationHours: durationHours,
		FeeBinancePct: feeBinancePct, FeeBybitPct: feeBybitPct,
	})
	if err != nil {
		return nil, err
	}
	return mustJSON(fundingArbWatchMap(got))
}

func (b *Backend) PauseFundingArbWatch(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.FundingArb == nil {
		return nil, fmt.Errorf("%w: funding-arb watches not configured", domain.ErrUpstream)
	}
	got, err := b.FundingArb.PauseWatch(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	return mustJSON(fundingArbWatchMap(got))
}

func (b *Backend) ResumeFundingArbWatch(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.FundingArb == nil {
		return nil, fmt.Errorf("%w: funding-arb watches not configured", domain.ErrUpstream)
	}
	got, err := b.FundingArb.ResumeWatch(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	return mustJSON(fundingArbWatchMap(got))
}

func (b *Backend) DeleteFundingArbWatch(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.FundingArb == nil {
		return nil, fmt.Errorf("%w: funding-arb watches not configured", domain.ErrUpstream)
	}
	if err := b.FundingArb.DeleteWatch(ctx, clientID, id); err != nil {
		return nil, err
	}
	return mustJSON(map[string]any{"deleted": true, "id": id})
}

func (b *Backend) ListFundingArbSignals(ctx context.Context, clientID, status string, limit int) (json.RawMessage, error) {
	if b.FundingArb == nil {
		return nil, fmt.Errorf("%w: funding-arb watches not configured", domain.ErrUpstream)
	}
	got, err := b.FundingArb.ListSignals(ctx, clientID, status, limit)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(got))
	for i := range got {
		items = append(items, fundingArbSignalMap(&got[i]))
	}
	return mustJSON(map[string]any{"signals": items, "count": len(items)})
}

func fundingArbWatchMap(w *domain.FundingArbWatch) map[string]any {
	if w == nil {
		return map[string]any{}
	}
	scope := "symbol"
	if w.IsScan() {
		scope = "scan"
	}
	m := map[string]any{
		"id": w.ID, "clientId": w.ClientID, "scope": scope, "symbol": w.Symbol,
		"quote": w.Quote, "symbolLimit": w.SymbolLimit,
		"notional": w.Notional, "holdHours": w.HoldHours, "minProfit": w.MinProfit,
		"feeBinancePct": w.FeeBinancePct, "feeBybitPct": w.FeeBybitPct,
		"status": string(w.Status), "armed": w.Armed,
		"createdAt": w.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updatedAt": w.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if w.ExpiresAt != nil {
		m["expiresAt"] = w.ExpiresAt.UTC().Format(time.RFC3339Nano)
	}
	return m
}

func fundingArbSignalMap(s *domain.FundingArbSignal) map[string]any {
	if s == nil {
		return map[string]any{}
	}
	m := map[string]any{
		"id": s.ID, "watchId": s.WatchID, "clientId": s.ClientID, "symbol": s.Symbol,
		"longExchange": string(s.LongExchange), "shortExchange": string(s.ShortExchange),
		"netAfterFees": s.NetAfterFees, "minProfit": s.MinProfit, "status": string(s.Status),
		"openedAt":   s.OpenedAt.UTC().Format(time.RFC3339Nano),
		"lastSeenAt": s.LastSeenAt.UTC().Format(time.RFC3339Nano),
	}
	if s.ClosedAt != nil {
		m["closedAt"] = s.ClosedAt.UTC().Format(time.RFC3339Nano)
	}
	return m
}

func parseFundingArbTime(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("empty")
	}
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse("2006-01-02", raw); err == nil {
		return t.UTC(), nil
	}
	if ms, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return time.UnixMilli(ms).UTC(), nil
	}
	return time.Time{}, fmt.Errorf("time")
}

func (b *Backend) GetFundingRate(ctx context.Context, exchange, symbol string, limit int) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetFundingRate(ctx, exchange, symbol, limit)
	if err != nil {
		return nil, err
	}
	return mustJSON(fundingSnapshotMap(got))
}

func fundingSnapshotMap(got *domain.FundingSnapshot) map[string]any {
	if got == nil {
		return map[string]any{}
	}
	venues := make([]map[string]any, 0, len(got.Venues))
	for _, v := range got.Venues {
		row := map[string]any{
			"exchange": v.Exchange, "current": fundingCurrentMap(v.Current),
			"avgLast3": v.AvgLast3, "avgLast3Pct": v.AvgLast3Pct,
			"history": fundingPrintsMap(v.History),
		}
		if v.LastSettled != nil {
			row["lastSettled"] = fundingPrintMap(*v.LastSettled)
		}
		venues = append(venues, row)
	}
	asOf := ""
	if !got.AsOf.IsZero() {
		asOf = got.AsOf.UTC().Format(time.RFC3339Nano)
	}
	out := map[string]any{
		"symbol": got.Symbol, "exchange": got.Exchange, "venues": venues,
		"history": fundingPrintsMap(got.History), "asOf": asOf, "venueCount": got.VenueCount,
		"note": "Predicted next funding plus recent settlements. rate is decimal; ratePct is percent. payer=long means longs pay shorts. Combined all does not average venues.",
	}
	if got.Current != nil {
		out["current"] = fundingCurrentMap(*got.Current)
	}
	return out
}

func fundingCurrentMap(c domain.FundingCurrent) map[string]any {
	out := map[string]any{"rate": c.Rate, "ratePct": c.RatePct, "payer": c.Payer, "intervalHours": c.IntervalHours}
	if !c.NextFundingTime.IsZero() {
		out["nextFundingTime"] = c.NextFundingTime.UTC().Format(time.RFC3339Nano)
	}
	if !c.Time.IsZero() {
		out["time"] = c.Time.UTC().Format(time.RFC3339Nano)
	}
	return out
}

func fundingPrintMap(p domain.FundingPrint) map[string]any {
	out := map[string]any{"rate": p.Rate, "ratePct": p.RatePct, "payer": p.Payer, "predicted": p.Predicted}
	if !p.Time.IsZero() {
		out["time"] = p.Time.UTC().Format(time.RFC3339Nano)
	}
	if p.MarkPrice != "" {
		out["markPrice"] = p.MarkPrice
	}
	return out
}

func fundingPrintsMap(in []domain.FundingPrint) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, p := range in {
		out = append(out, fundingPrintMap(p))
	}
	return out
}

func (b *Backend) GetLiquidations(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetLiquidations(ctx, exchange, symbol)
	if err != nil {
		return nil, err
	}
	wins := make([]map[string]any, 0, len(got.Windows))
	for _, w := range got.Windows {
		row := map[string]any{
			"window": w.Window, "longNotional": w.LongNotional, "shortNotional": w.ShortNotional,
			"totalNotional": w.TotalNotional, "count": w.Count,
			"coverageSeconds": w.CoverageSeconds, "complete": w.Complete,
		}
		if w.Biggest != nil {
			row["biggest"] = map[string]any{
				"exchange": w.Biggest.Exchange, "side": w.Biggest.Side,
				"price": w.Biggest.Price, "quantity": w.Biggest.Quantity, "notional": w.Biggest.Notional,
				"time": w.Biggest.Time.UTC().Format(time.RFC3339Nano),
			}
		}
		wins = append(wins, row)
	}
	since := ""
	if !got.CollectingSince.IsZero() {
		since = got.CollectingSince.UTC().Format(time.RFC3339Nano)
	}
	return mustJSON(map[string]any{
		"symbol": got.Symbol, "exchange": got.Exchange, "collectingSince": since,
		"live": got.Live, "venueCount": got.VenueCount, "windows": wins,
		"feed": liquidationFeedJSON(got.Feed),
	})
}

func liquidationFeedJSON(f domain.LiquidationFeed) map[string]any {
	venues := make([]map[string]any, 0, len(f.Venues))
	for _, v := range f.Venues {
		row := map[string]any{
			"exchange": v.Exchange, "live": v.Live, "coverageSeconds": v.CoverageSeconds,
		}
		if !v.LastEventAt.IsZero() {
			row["lastEventAt"] = v.LastEventAt.UTC().Format(time.RFC3339Nano)
		}
		if !v.LastSeenAt.IsZero() {
			row["lastSeenAt"] = v.LastSeenAt.UTC().Format(time.RFC3339Nano)
		}
		gaps := make([]map[string]any, 0, len(v.Gaps))
		for _, g := range v.Gaps {
			gap := map[string]any{"from": g.From.UTC().Format(time.RFC3339Nano), "seconds": g.Seconds}
			if !g.To.IsZero() {
				gap["to"] = g.To.UTC().Format(time.RFC3339Nano)
			}
			gaps = append(gaps, gap)
		}
		row["gaps"] = gaps
		venues = append(venues, row)
	}
	missing := f.Missing
	if missing == nil {
		missing = []string{}
	}
	return map[string]any{"venues": venues, "missing": missing}
}

func (b *Backend) GetLiquidationOverview(ctx context.Context, exchange, window string, limit int) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetLiquidationOverview(ctx, exchange, window, limit)
	if err != nil {
		return nil, err
	}
	wins := make([]map[string]any, 0, len(got.Windows))
	for _, w := range got.Windows {
		row := map[string]any{
			"window": w.Window, "longNotional": w.LongNotional, "shortNotional": w.ShortNotional,
			"totalNotional": w.TotalNotional, "count": w.Count,
			"coverageSeconds": w.CoverageSeconds, "complete": w.Complete,
		}
		if w.Biggest != nil {
			row["biggest"] = map[string]any{
				"exchange": w.Biggest.Exchange, "side": w.Biggest.Side,
				"price": w.Biggest.Price, "quantity": w.Biggest.Quantity, "notional": w.Biggest.Notional,
				"time": w.Biggest.Time.UTC().Format(time.RFC3339Nano),
			}
		}
		wins = append(wins, row)
	}
	coins := make([]map[string]any, 0, len(got.Coins))
	for _, c := range got.Coins {
		row := map[string]any{
			"symbol": c.Symbol, "base": c.Base,
			"longNotional": c.LongNotional, "shortNotional": c.ShortNotional,
			"totalNotional": c.TotalNotional, "count": c.Count,
		}
		if c.Biggest != nil {
			row["biggest"] = map[string]any{
				"exchange": c.Biggest.Exchange, "side": c.Biggest.Side,
				"price": c.Biggest.Price, "quantity": c.Biggest.Quantity, "notional": c.Biggest.Notional,
				"time": c.Biggest.Time.UTC().Format(time.RFC3339Nano),
			}
		}
		coins = append(coins, row)
	}
	since := ""
	if !got.CollectingSince.IsZero() {
		since = got.CollectingSince.UTC().Format(time.RFC3339Nano)
	}
	return mustJSON(map[string]any{
		"exchange": got.Exchange, "coinWindow": got.CoinWindow, "collectingSince": since,
		"live": got.Live, "venueCount": got.VenueCount, "windows": wins, "coins": coins,
		"feed": liquidationFeedJSON(got.Feed),
	})
}

func (b *Backend) GetLiquidationLevels(ctx context.Context, exchange, symbol, rawRange string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetLiquidationLevels(ctx, exchange, symbol, rawRange)
	if err != nil {
		return nil, err
	}
	levels := make([]map[string]any, 0, len(got.Levels))
	for _, lv := range got.Levels {
		lev := make([]map[string]any, 0, len(lv.ByLeverage))
		for _, sl := range lv.ByLeverage {
			lev = append(lev, map[string]any{
				"leverage": sl.Leverage, "longNotional": sl.LongNotional, "shortNotional": sl.ShortNotional,
			})
		}
		levels = append(levels, map[string]any{
			"price": lv.Price, "longNotional": lv.LongNotional, "shortNotional": lv.ShortNotional,
			"totalNotional": lv.TotalNotional, "cumLong": lv.CumLong, "cumShort": lv.CumShort,
			"cumTotal": lv.CumTotal, "byLeverage": lev,
		})
	}
	bars := make([]map[string]any, 0, len(got.Bars))
	for _, bar := range got.Bars {
		bars = append(bars, map[string]any{
			"t":            bar.Time.UTC().Format(time.RFC3339Nano),
			"longNotional": bar.LongNotional, "shortNotional": bar.ShortNotional,
			"totalNotional": bar.TotalNotional, "count": bar.Count,
		})
	}
	from, to := "", ""
	if !got.From.IsZero() {
		from = got.From.UTC().Format(time.RFC3339Nano)
	}
	if !got.To.IsZero() {
		to = got.To.UTC().Format(time.RFC3339Nano)
	}
	return mustJSON(map[string]any{
		"kind": got.Kind, "symbol": got.Symbol, "exchange": got.Exchange, "range": got.Range,
		"from": from, "to": to, "lastPrice": got.LastPrice, "lastPrices": got.LastPrices,
		"levels": levels, "bars": bars, "feed": liquidationFeedJSON(got.Feed),
		"missing": got.Missing, "note": got.Note,
	})
}

func (b *Backend) GetLiquidationCascade(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetLiquidationCascade(ctx, exchange, symbol)
	if err != nil {
		return nil, err
	}
	return mustJSON(cascadeReportJSON(got))
}

func (b *Backend) ScanLiquidationCascades(ctx context.Context, exchange string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.ScanLiquidationCascades(ctx, exchange)
	if err != nil {
		return nil, err
	}
	hits := make([]map[string]any, 0, len(got.Hits))
	for _, h := range got.Hits {
		hits = append(hits, map[string]any{
			"symbol": h.Symbol, "side": h.Side, "grade": h.Grade, "score": h.Score,
			"hottest": h.Hottest, "both": h.Both, "summary": h.Summary,
		})
	}
	return mustJSON(map[string]any{
		"exchange": got.Exchange,
		"asOf":     got.AsOf.UTC().Format(time.RFC3339Nano),
		"market":   cascadeReportJSON(&got.Market),
		"hits":     hits,
		"summary":  got.Summary,
		"note":     got.Note,
	})
}

func cascadeReportJSON(got *domain.CascadeReport) map[string]any {
	if got == nil {
		return map[string]any{"venues": []any{}}
	}
	venues := make([]map[string]any, 0, len(got.Venues))
	for _, v := range got.Venues {
		windows := make([]map[string]any, 0, len(v.Windows))
		for _, w := range v.Windows {
			windows = append(windows, map[string]any{
				"window": w.Window, "longNotional": w.LongNotional, "shortNotional": w.ShortNotional,
				"totalNotional": w.TotalNotional, "longTypical": w.LongTypical, "shortTypical": w.ShortTypical,
				"longRatio": w.LongRatio, "shortRatio": w.ShortRatio, "maxRatio": w.MaxRatio,
				"side": w.Side, "grade": w.Grade, "count": w.Count,
				"sampleBuckets": w.SampleBuckets, "complete": w.Complete,
			})
		}
		row := map[string]any{
			"exchange": string(v.Exchange), "symbol": v.Symbol, "windows": windows,
			"side": v.Side, "grade": v.Grade, "score": v.Score, "hottest": v.Hottest,
			"summary": v.Summary,
		}
		if !v.StartedAt.IsZero() {
			row["startedAt"] = v.StartedAt.UTC().Format(time.RFC3339Nano)
		}
		venues = append(venues, row)
	}
	episodes := make([]map[string]any, 0, len(got.Episodes))
	for _, ep := range got.Episodes {
		row := map[string]any{
			"symbol": ep.Symbol, "exchange": ep.Exchange, "combined": ep.Combined,
			"side": ep.Side, "grade": ep.Grade, "score": ep.Score, "open": ep.Open,
			"durationSec": ep.DurationSec, "longNotional": ep.LongNotional,
			"shortNotional": ep.ShortNotional, "totalNotional": ep.TotalNotional,
			"count": ep.Count, "peakRatio": ep.PeakRatio, "summary": ep.Summary,
		}
		if !ep.StartedAt.IsZero() {
			row["startedAt"] = ep.StartedAt.UTC().Format(time.RFC3339Nano)
		}
		if !ep.EndedAt.IsZero() {
			row["endedAt"] = ep.EndedAt.UTC().Format(time.RFC3339Nano)
		}
		if ep.PriceOpen != "" {
			row["priceOpen"] = ep.PriceOpen
			row["priceClose"] = ep.PriceClose
			row["priceHigh"] = ep.PriceHigh
			row["priceLow"] = ep.PriceLow
			row["priceChangePct"] = ep.PriceChangePct
		}
		episodes = append(episodes, row)
	}
	out := map[string]any{
		"symbol": got.Symbol, "exchange": got.Exchange,
		"asOf":   got.AsOf.UTC().Format(time.RFC3339Nano),
		"venues": venues, "episodes": episodes, "summary": got.Summary, "note": got.Note,
	}
	if got.Both != nil {
		out["both"] = map[string]any{
			"agree": got.Both.Agree, "side": got.Both.Side, "grade": got.Both.Grade,
			"score": got.Both.Score, "hottest": got.Both.Hottest, "summary": got.Both.Summary,
		}
	}
	return out
}

func (b *Backend) GetOrderBookHeatmap(ctx context.Context, exchange, symbol, group string, windowSec int) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetOrderBookHeatmap(ctx, exchange, symbol, group, windowSec)
	if err != nil {
		return nil, err
	}
	cols := make([]map[string]any, 0, len(got.Columns))
	mapLevels := func(in []domain.HeatmapLevel) []map[string]any {
		out := make([]map[string]any, 0, len(in))
		for _, lv := range in {
			out = append(out, map[string]any{"price": lv.Price, "notional": lv.Notional, "isWall": lv.IsWall})
		}
		return out
	}
	for _, c := range got.Columns {
		cols = append(cols, map[string]any{
			"t": c.Time.UTC().Format(time.RFC3339Nano), "mid": c.Mid,
			"bids": mapLevels(c.Bids), "asks": mapLevels(c.Asks),
		})
	}
	from, to := "", ""
	if !got.From.IsZero() {
		from = got.From.UTC().Format(time.RFC3339Nano)
	}
	if !got.To.IsZero() {
		to = got.To.UTC().Format(time.RFC3339Nano)
	}
	return mustJSON(map[string]any{
		"exchange": string(got.Exchange), "symbol": got.Symbol, "groupSize": got.GroupSize,
		"windowSeconds": got.WindowSeconds, "sampleEveryMs": got.SampleEveryMs,
		"from": from, "to": to, "columns": cols, "live": got.Live,
		"note": "Resting bid/ask notional over time. Not executed volume.",
	})
}

func (b *Backend) GetMarketLiquidity(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetMarketLiquidity(ctx, exchange, symbol)
	if err != nil {
		return nil, err
	}
	mapScore := func(s domain.LiquidityScore) map[string]any {
		bands := make([]map[string]any, 0, len(s.Bands))
		for _, b := range s.Bands {
			bands = append(bands, map[string]any{
				"rangePct": b.RangePct, "bidNotional": b.BidNotional, "askNotional": b.AskNotional,
				"bidQuantity": b.BidQuantity, "askQuantity": b.AskQuantity, "totalNotional": b.TotalNotional,
				"imbalance": b.Imbalance, "score": b.Score,
			})
		}
		return map[string]any{
			"midPrice": s.MidPrice, "usedRangePct": s.UsedRangePct,
			"score": s.Score, "grade": s.Grade,
			"weakerSide": s.WeakerSide, "weakness": s.Weakness, "bands": bands,
		}
	}
	venues := make([]map[string]any, 0, len(got.Venues))
	for _, v := range got.Venues {
		row := mapScore(v.LiquidityScore)
		row["exchange"] = string(v.Exchange)
		row["symbol"] = v.Symbol
		row["live"] = v.Live
		if v.Error != "" {
			row["error"] = v.Error
		}
		venues = append(venues, row)
	}
	return mustJSON(map[string]any{
		"symbol": got.Symbol, "venueCount": got.VenueCount,
		"market": mapScore(got.Market), "venues": venues,
	})
}

func (b *Backend) AnalyzeCombinedOrderBook(ctx context.Context, symbol string, rangePct float64) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetCombinedOrderBookAnalysis(ctx, symbol, rangePct)
	if err != nil {
		return nil, err
	}
	venues := make([]map[string]any, 0, len(got.Venues))
	for _, v := range got.Venues {
		venues = append(venues, map[string]any{
			"exchange": string(v.Exchange), "symbol": v.Symbol, "live": v.Live, "source": v.Source,
			"bidNotional": v.BidNotional, "askNotional": v.AskNotional,
			"bidQuantity": v.BidQuantity, "askQuantity": v.AskQuantity,
			"imbalance": v.Imbalance, "pressure": v.Pressure,
			"bidLevels": v.BidLevels, "askLevels": v.AskLevels, "error": v.Error,
		})
	}
	walls := make([]map[string]any, 0, len(got.Walls))
	for _, w := range got.Walls {
		walls = append(walls, map[string]any{
			"exchange": w.Exchange, "side": w.Side, "price": w.Price, "quantity": w.Quantity,
			"notional": w.Notional, "distancePct": w.DistancePct, "share": w.Share,
			"behavior": w.Behavior, "ageSeconds": w.AgeSeconds, "presentForSeconds": w.PresentForSeconds,
			"visibleSeconds": w.VisibleSeconds, "appearCount": w.AppearCount,
			"iceberg": w.Iceberg, "icebergRefills": w.IcebergRefills, "icebergClip": w.IcebergClip,
		})
	}
	return mustJSON(map[string]any{
		"symbol": got.Symbol, "rangePct": got.RangePct, "midPrice": got.MidPrice,
		"usedLow": got.UsedLow, "usedHigh": got.UsedHigh, "usedRangePct": got.UsedRangePct, "requestedReached": got.RequestedReached,
		"bidNotional": got.BidNotional, "askNotional": got.AskNotional,
		"bidQuantity": got.BidQuantity, "askQuantity": got.AskQuantity,
		"imbalance": got.Imbalance, "pressure": got.Pressure,
		"bidLevels": got.BidLevels, "askLevels": got.AskLevels,
		"coveredBidPct": got.CoveredBidPct, "coveredAskPct": got.CoveredAskPct,
		"walls": walls, "bands": got.Bands, "venues": venues, "venueCount": got.VenueCount,
	})
}

func (b *Backend) AnalyzeOrderBook(ctx context.Context, exchange, symbol string, rangePct float64) (json.RawMessage, error) {
	book, err := b.Market.GetSpotOrderBook(ctx, exchange, symbol, "", 5, rangePct)
	if err != nil {
		return nil, err
	}
	return mustJSON(map[string]any{
		"exchange":  string(book.Exchange),
		"symbol":    book.Symbol,
		"lastPrice": book.LastPrice,
		"bestBid":   book.BestBid,
		"bestAsk":   book.BestAsk,
		"spread":    book.Spread,
		"spreadPct": book.SpreadPct,
		"live":      book.Live,
		"source":    book.Source,
		"updatedAt": book.UpdatedAt.UTC().Format(time.RFC3339Nano),
		"analysis":  orderBookAnalysisMap(book.Analysis),
	})
}

func orderBookAnalysisMap(a domain.OrderBookAnalysis) map[string]any {
	return map[string]any{
		"rangePct":      a.RangePct,
		"midPrice":      a.MidPrice,
		"bidNotional":   a.BidNotional,
		"askNotional":   a.AskNotional,
		"bidQuantity":   a.BidQuantity,
		"askQuantity":   a.AskQuantity,
		"imbalance":     a.Imbalance,
		"pressure":      a.Pressure,
		"bidLevels":     a.BidLevels,
		"askLevels":     a.AskLevels,
		"coveredBidPct": a.CoveredBidPct,
		"coveredAskPct": a.CoveredAskPct,
		"walls":         a.Walls,
		"bands":         a.Bands,
	}
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
		"stale":             sup.Stale,
	})
}

func (b *Backend) GetAssetProfile(ctx context.Context, asset string) (json.RawMessage, error) {
	got, err := b.Market.GetAssetProfile(ctx, asset)
	if err != nil {
		return nil, err
	}
	rows := make([]map[string]any, 0, len(got.Contracts))
	for _, c := range got.Contracts {
		rows = append(rows, map[string]any{"chain": c.Chain, "address": c.Address})
	}
	listing := ""
	if got.ListingDate != nil && !got.ListingDate.IsZero() {
		listing = got.ListingDate.UTC().Format("2006-01-02")
	}
	return mustJSON(map[string]any{
		"asset":       got.Asset,
		"name":        got.Name,
		"slug":        got.Slug,
		"providerId":  got.ProviderID,
		"logoUrl":     got.LogoURL,
		"listingDate": listing,
		"contracts":   rows,
		"asOf":        got.AsOf.UTC().Format(time.RFC3339Nano),
		"source":      got.Source,
	})
}

func (b *Backend) GetHolders(ctx context.Context, asset string) (json.RawMessage, error) {
	got, err := b.Market.GetHolders(ctx, asset)
	if err != nil {
		return nil, err
	}
	rows := make([]map[string]any, 0, len(got.TopHolders))
	for _, row := range got.TopHolders {
		item := map[string]any{
			"address":  row.Address,
			"balance":  row.Balance,
			"sharePct": row.SharePct,
		}
		if row.Label != "" {
			item["label"] = row.Label
		}
		rows = append(rows, item)
	}
	return mustJSON(map[string]any{
		"asset":              got.Asset,
		"name":               got.Name,
		"providerId":         got.ProviderID,
		"holderCount":        got.HolderCount,
		"dailyActive":        got.DailyActive,
		"topTenSharePct":     got.TopTenSharePct,
		"topTwentySharePct":  got.TopTwentySharePct,
		"topFiftySharePct":   got.TopFiftySharePct,
		"topHundredSharePct": got.TopHundredSharePct,
		"topHolders":         rows,
		"asOf":               got.AsOf.UTC().Format(time.RFC3339Nano),
		"source":             got.Source,
		"stale":              got.Stale,
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
	ser, err := b.Market.GetIndicators(ctx, exchange, symbol, interval, limit, rsiPeriod, periods, nil)
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
			"rsi":  ser.LatestRSI,
			"zone": string(ser.LatestZone),
			"ema":  latestEMA,
		},
		"points": points,
		"note":   "Informational only — not financial advice.",
	})
}

func (b *Backend) GetFxRates(ctx context.Context) (json.RawMessage, error) {
	got, err := b.Market.GetFxRates(ctx)
	if err != nil {
		return nil, err
	}
	asOf := ""
	rates := map[string]float64{}
	note := ""
	stale := false
	if got != nil {
		if !got.AsOf.IsZero() {
			asOf = got.AsOf.UTC().Format(time.RFC3339)
		}
		rates = got.Rates
		note = got.Note
		stale = got.Stale
	}
	return mustJSON(map[string]any{
		"base":            domain.FxBaseUSD,
		"asOf":            asOf,
		"rates":           rates,
		"venueQuotes":     domain.DisplayVenueQuotes(),
		"marketCapQuotes": domain.DisplayMarketCapQuotes(),
		"aliases":         domain.DisplayFxAliases(),
		"stale":           stale,
		"note":            note,
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
		"exchanges":       out,
		"default":         string(domain.DefaultExchange),
		"venueQuotes":     domain.DisplayVenueQuotes(),
		"marketCapQuotes": domain.DisplayMarketCapQuotes(),
	})
}

func (b *Backend) ListDelistSchedule(ctx context.Context, exchange string) (json.RawMessage, error) {
	_ = ctx
	entries, err := b.Market.ListDelistSchedule(exchange)
	if err != nil {
		return nil, err
	}
	ex, _ := b.Market.ResolveExchange(exchange)
	items := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		row := map[string]any{
			"symbol":     e.Symbol,
			"delistTime": e.DelistTime.UTC().Format(time.RFC3339),
		}
		if !e.AnnouncedAt.IsZero() {
			row["announcedAt"] = e.AnnouncedAt.UTC().Format(time.RFC3339)
		}
		items = append(items, row)
	}
	return mustJSON(map[string]any{
		"exchange": string(ex),
		"enabled":  b.Market.DelistEnabledFor(string(ex)),
		"items":    items,
	})
}

func (b *Backend) GetPostDelist(ctx context.Context, exchange, symbol, interval string, limit int) (json.RawMessage, error) {
	view, err := b.Market.GetPostDelist(ctx, exchange, symbol, interval, limit)
	if err != nil {
		return nil, err
	}
	out := map[string]any{
		"exchange":           view.Exchange,
		"symbol":             view.Symbol,
		"base":               view.Base,
		"available":          view.Available,
		"source":             view.Source,
		"sourceLabel":        view.SourceLabel,
		"note":               view.Note,
		"lastPrice":          view.LastPrice,
		"priceChangePercent": view.PriceChangePercent,
		"quote":              view.Quote,
		"interval":           view.Interval,
	}
	if !view.DelistTime.IsZero() {
		out["delistTime"] = view.DelistTime.UTC().Format(time.RFC3339)
	}
	if !view.AsOf.IsZero() {
		out["asOf"] = view.AsOf.UTC().Format(time.RFC3339)
	}
	if len(view.Candles) > 0 {
		items := make([]map[string]any, 0, len(view.Candles))
		for _, c := range view.Candles {
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
		out["candles"] = items
	}
	return mustJSON(out)
}

func (b *Backend) GetWatchlist(ctx context.Context, clientID string) (json.RawMessage, error) {
	return b.GetWatchlistOwned(ctx, clientID, "")
}

// GetWatchlistOwned reads actor's own list or a shared list (ownerClientID).
func (b *Backend) GetWatchlistOwned(ctx context.Context, actorClientID, ownerClientID string) (json.RawMessage, error) {
	if b.Watch == nil {
		return nil, fmt.Errorf("watchlist not configured")
	}
	acc, err := b.Watch.Get(ctx, actorClientID, ownerClientID)
	if err != nil {
		return nil, err
	}
	return watchlistAccessJSON(acc)
}

func (b *Backend) AddWatchlistItem(ctx context.Context, clientID, exchange, symbol, note string) (json.RawMessage, error) {
	return b.AddWatchlistItemOwned(ctx, clientID, "", exchange, symbol, note)
}

// AddWatchlistItemOwned adds a symbol; ownerClientID empty = actor's list.
func (b *Backend) AddWatchlistItemOwned(ctx context.Context, actorClientID, ownerClientID, exchange, symbol, note string) (json.RawMessage, error) {
	if b.Watch == nil {
		return nil, fmt.Errorf("watchlist not configured")
	}
	acc, err := b.Watch.Add(ctx, actorClientID, ownerClientID, exchange, symbol, note, domain.WatchlistUnconditionalVersion)
	if err != nil {
		return nil, err
	}
	return watchlistAccessJSON(acc)
}

func (b *Backend) RemoveWatchlistItem(ctx context.Context, clientID, exchange, symbol string) (json.RawMessage, error) {
	return b.RemoveWatchlistItemOwned(ctx, clientID, "", exchange, symbol)
}

// RemoveWatchlistItemOwned removes a symbol; ownerClientID empty = actor's list.
func (b *Backend) RemoveWatchlistItemOwned(ctx context.Context, actorClientID, ownerClientID, exchange, symbol string) (json.RawMessage, error) {
	if b.Watch == nil {
		return nil, fmt.Errorf("watchlist not configured")
	}
	acc, err := b.Watch.Remove(ctx, actorClientID, ownerClientID, exchange, symbol, domain.WatchlistUnconditionalVersion)
	if err != nil {
		return nil, err
	}
	return watchlistAccessJSON(acc)
}

func (b *Backend) ShareWatchlist(ctx context.Context, ownerClientID, granteeClientID, role string) (json.RawMessage, error) {
	if b.Watch == nil {
		return nil, fmt.Errorf("watchlist not configured")
	}
	sh, err := b.Watch.Share(ctx, ownerClientID, granteeClientID, role)
	if err != nil {
		return nil, err
	}
	return shareJSON(sh)
}

func (b *Backend) UpdateWatchlistShare(ctx context.Context, ownerClientID, granteeClientID, role string) (json.RawMessage, error) {
	if b.Watch == nil {
		return nil, fmt.Errorf("watchlist not configured")
	}
	sh, err := b.Watch.UpdateShareRole(ctx, ownerClientID, granteeClientID, role)
	if err != nil {
		return nil, err
	}
	return shareJSON(sh)
}

func (b *Backend) RevokeWatchlistShare(ctx context.Context, ownerClientID, granteeClientID string) (json.RawMessage, error) {
	if b.Watch == nil {
		return nil, fmt.Errorf("watchlist not configured")
	}
	if err := b.Watch.RevokeShare(ctx, ownerClientID, granteeClientID); err != nil {
		return nil, err
	}
	return mustJSON(map[string]any{"revoked": true, "granteeClientId": granteeClientID})
}

func (b *Backend) ListWatchlistShares(ctx context.Context, ownerClientID string) (json.RawMessage, error) {
	if b.Watch == nil {
		return nil, fmt.Errorf("watchlist not configured")
	}
	list, err := b.Watch.ListShares(ctx, ownerClientID)
	if err != nil {
		return nil, err
	}
	return sharesListJSON(ownerClientID, list)
}

func (b *Backend) ListSharedWatchlists(ctx context.Context, granteeClientID string) (json.RawMessage, error) {
	if b.Watch == nil {
		return nil, fmt.Errorf("watchlist not configured")
	}
	list, err := b.Watch.ListSharedWithMe(ctx, granteeClientID)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(list))
	for i := range list {
		items = append(items, shareMap(&list[i]))
	}
	return mustJSON(map[string]any{"clientId": granteeClientID, "shares": items, "count": len(items)})
}

func (b *Backend) ListWatchlistAudit(ctx context.Context, ownerClientID string, limit, offset int) (json.RawMessage, error) {
	if b.Watch == nil {
		return nil, fmt.Errorf("watchlist not configured")
	}
	list, err := b.Watch.ListAudit(ctx, ownerClientID, limit, offset)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(list))
	for _, ev := range list {
		items = append(items, map[string]any{
			"id":            ev.ID,
			"ownerClientId": ev.OwnerClientID,
			"actorClientId": ev.ActorClientID,
			"action":        string(ev.Action),
			"exchange":      ev.Exchange,
			"symbol":        ev.Symbol,
			"detail":        ev.Detail,
			"createdAt":     ev.CreatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	return mustJSON(map[string]any{"ownerClientId": ownerClientID, "events": items, "count": len(items)})
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

func (b *Backend) CreateOrderBookAlert(ctx context.Context, clientID, exchange, symbol, kind, condition string, threshold, rangePct float64, mode string) (json.RawMessage, error) {
	if b.Alerts == nil {
		return nil, fmt.Errorf("%w: alerts not configured", domain.ErrUpstream)
	}
	a, err := b.Alerts.Create(ctx, pricealert.CreateInput{
		ClientID:    clientID,
		Exchange:    exchange,
		Symbol:      symbol,
		Kind:        kind,
		Condition:   condition,
		TargetPrice: threshold,
		RangePct:    rangePct,
		Mode:        mode,
	})
	if err != nil {
		return nil, err
	}
	return alertJSON(a)
}

func (b *Backend) CreateLiquidationFeedAlert(ctx context.Context, clientID, exchange string, minDownSeconds float64, mode string) (json.RawMessage, error) {
	if b.Alerts == nil {
		return nil, fmt.Errorf("%w: alerts not configured", domain.ErrUpstream)
	}
	a, err := b.Alerts.Create(ctx, pricealert.CreateInput{
		ClientID:    clientID,
		Exchange:    exchange,
		Kind:        string(domain.AlertKindLiqFeed),
		TargetPrice: minDownSeconds,
		Mode:        mode,
	})
	if err != nil {
		return nil, err
	}
	return alertJSON(a)
}

func (b *Backend) CreateLiquidationCascadeAlert(ctx context.Context, clientID, exchange, symbol, minGrade, mode string) (json.RawMessage, error) {
	if b.Alerts == nil {
		return nil, fmt.Errorf("%w: alerts not configured", domain.ErrUpstream)
	}
	a, err := b.Alerts.Create(ctx, pricealert.CreateInput{
		ClientID:  clientID,
		Exchange:  exchange,
		Symbol:    symbol,
		Kind:      string(domain.AlertKindLiqCascade),
		Condition: minGrade,
		Mode:      mode,
	})
	if err != nil {
		return nil, err
	}
	return alertJSON(a)
}

func (b *Backend) CreateLiquidationNotionalAlert(ctx context.Context, clientID, exchange, symbol, side string, notional float64, window, mode string) (json.RawMessage, error) {
	if b.Alerts == nil {
		return nil, fmt.Errorf("%w: alerts not configured", domain.ErrUpstream)
	}
	a, err := b.Alerts.Create(ctx, pricealert.CreateInput{
		ClientID:    clientID,
		Exchange:    exchange,
		Symbol:      symbol,
		Kind:        string(domain.AlertKindLiqNotional),
		Condition:   side,
		TargetPrice: notional,
		Window:      window,
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

func (b *Backend) CreateAPIKey(ctx context.Context, clientID, name, permission string) (json.RawMessage, error) {
	if b.APIKeys == nil {
		return nil, fmt.Errorf("%w: api keys not configured", domain.ErrUpstream)
	}
	created, err := b.APIKeys.Create(ctx, apikey.CreateInput{ClientID: clientID, Name: name, Permission: permission})
	if err != nil {
		return nil, err
	}
	return mustJSON(apiKeyMap(created.Key, created.Secret))
}

func (b *Backend) ListAPIKeys(ctx context.Context, clientID string) (json.RawMessage, error) {
	if b.APIKeys == nil {
		return nil, fmt.Errorf("%w: api keys not configured", domain.ErrUpstream)
	}
	list, err := b.APIKeys.List(ctx, clientID)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(list))
	for i := range list {
		items = append(items, apiKeyMap(&list[i], ""))
	}
	return mustJSON(map[string]any{"clientId": clientID, "keys": items, "count": len(items)})
}

func (b *Backend) RevokeAPIKey(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.APIKeys == nil {
		return nil, fmt.Errorf("%w: api keys not configured", domain.ErrUpstream)
	}
	k, err := b.APIKeys.Revoke(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	return mustJSON(apiKeyMap(k, ""))
}

func apiKeyMap(k *domain.APIKey, secret string) map[string]any {
	m := map[string]any{
		"id": k.ID, "name": k.Name, "prefix": k.Prefix, "permission": string(k.Permission),
		"revoked": k.IsRevoked(), "createdAt": k.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if k.LastUsedAt != nil {
		m["lastUsedAt"] = k.LastUsedAt.UTC().Format(time.RFC3339Nano)
	}
	if k.RevokedAt != nil {
		m["revokedAt"] = k.RevokedAt.UTC().Format(time.RFC3339Nano)
	}
	if secret != "" {
		m["secret"] = secret
	}
	return m
}

func (b *Backend) CreatePortfolio(ctx context.Context, clientID string, startingBalance float64, currency string) (json.RawMessage, error) {
	return b.CreateNamedPortfolio(ctx, clientID, startingBalance, currency, "")
}

func (b *Backend) CreateNamedPortfolio(ctx context.Context, clientID string, startingBalance float64, currency, name string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	p, err := b.Portfolio.Create(ctx, portfolio.CreateInput{
		ClientID: clientID, Name: name, StartingBalance: startingBalance, Currency: currency,
	})
	if err != nil {
		return nil, err
	}
	return b.GetPortfolio(WithPortfolioID(ctx, p.ID), clientID)
}

func (b *Backend) ListPortfolios(ctx context.Context, clientID string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	list, err := b.Portfolio.List(ctx, clientID)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(list))
	for _, p := range list {
		items = append(items, map[string]any{
			"id": p.ID, "clientId": p.ClientID, "name": p.Name, "currency": p.Currency,
			"startingBalance": p.StartingBalance, "cashBalance": p.CashBalance,
			"createdAt": p.CreatedAt.UTC().Format(time.RFC3339Nano),
			"updatedAt": p.UpdatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	return mustJSON(map[string]any{"clientId": clientID, "portfolios": items, "count": len(items)})
}

func (b *Backend) RenamePortfolio(ctx context.Context, clientID, id, name string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	p, err := b.Portfolio.Rename(ctx, clientID, id, name)
	if err != nil {
		return nil, err
	}
	return mustJSON(map[string]any{
		"id": p.ID, "clientId": p.ClientID, "name": p.Name, "currency": p.Currency,
		"startingBalance": p.StartingBalance, "cashBalance": p.CashBalance,
		"updatedAt": p.UpdatedAt.UTC().Format(time.RFC3339Nano),
	})
}

func (b *Backend) DeletePortfolio(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	if err := b.Portfolio.Delete(ctx, clientID, id); err != nil {
		return nil, err
	}
	return mustJSON(map[string]any{"deleted": true, "id": id, "clientId": clientID})
}

func (b *Backend) SharePortfolio(ctx context.Context, clientID, portfolioID, granteeClientID, role string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	sh, err := b.Portfolio.Share(ctx, clientID, portfolioID, granteeClientID, role)
	if err != nil {
		return nil, err
	}
	return mustJSON(portfolioShareMap(sh))
}

func (b *Backend) UpdatePortfolioShare(ctx context.Context, clientID, portfolioID, granteeClientID, role string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	sh, err := b.Portfolio.UpdateShareRole(ctx, clientID, portfolioID, granteeClientID, role)
	if err != nil {
		return nil, err
	}
	return mustJSON(portfolioShareMap(sh))
}

func (b *Backend) RevokePortfolioShare(ctx context.Context, clientID, portfolioID, granteeClientID string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	if err := b.Portfolio.RevokeShare(ctx, clientID, portfolioID, granteeClientID); err != nil {
		return nil, err
	}
	return mustJSON(map[string]any{"revoked": true, "granteeClientId": granteeClientID})
}

func (b *Backend) ListPortfolioShares(ctx context.Context, clientID, portfolioID string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	list, err := b.Portfolio.ListShares(ctx, clientID, portfolioID)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(list))
	for i := range list {
		items = append(items, portfolioShareMap(&list[i]))
	}
	return mustJSON(map[string]any{"ownerClientId": clientID, "shares": items, "count": len(items)})
}

func (b *Backend) ListSharedPortfolios(ctx context.Context, clientID string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	list, err := b.Portfolio.ListSharedWithMe(ctx, clientID)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(list))
	for _, bkn := range list {
		p := bkn.Portfolio
		items = append(items, map[string]any{
			"id": p.ID, "clientId": p.ClientID, "name": p.Name, "role": string(bkn.Role),
			"currency": p.Currency, "cashBalance": p.CashBalance,
		})
	}
	return mustJSON(map[string]any{"clientId": clientID, "portfolios": items, "count": len(items)})
}

func portfolioShareMap(sh *domain.PortfolioShare) map[string]any {
	return map[string]any{
		"portfolioId": sh.PortfolioID, "ownerClientId": sh.OwnerClientID, "granteeClientId": sh.GranteeClientID,
		"role":      string(sh.Role),
		"createdAt": sh.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updatedAt": sh.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func (b *Backend) GetPaperTradingCosts(_ context.Context, exchange string) (json.RawMessage, error) {
	note := domain.PaperTradingCostsNote
	ex := strings.TrimSpace(exchange)
	if ex != "" {
		v := domain.PaperTradingCostViewFor(domain.ParseExchange(ex))
		return mustJSON(map[string]any{
			"exchange": string(v.Exchange), "feeRate": v.FeeRate, "slippageRate": v.SlippageRate,
			"feePct": v.FeePct, "slippagePct": v.SlippagePct, "note": note,
		})
	}
	all := domain.AllPaperTradingCosts()
	items := make([]map[string]any, 0, len(all))
	for _, v := range all {
		items = append(items, map[string]any{
			"exchange": string(v.Exchange), "feeRate": v.FeeRate, "slippageRate": v.SlippageRate,
			"feePct": v.FeePct, "slippagePct": v.SlippagePct,
		})
	}
	return mustJSON(map[string]any{"items": items, "note": note})
}

func (b *Backend) GetPortfolioRiskLimits(ctx context.Context, clientID string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	v, err := b.Portfolio.GetRiskLimitsView(ctx, clientID, PortfolioIDFrom(ctx))
	if err != nil {
		return nil, err
	}
	return mustJSON(riskLimitsViewMap(v))
}

func (b *Backend) PutPortfolioRiskLimits(ctx context.Context, clientID string, maxDailyLossPct, maxAssetWeightPct *float64) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	v, err := b.Portfolio.SetRiskLimits(ctx, portfolio.RiskLimitsInput{
		PortfolioID: PortfolioIDFrom(ctx),
		ClientID:    clientID, MaxDailyLossPct: maxDailyLossPct, MaxAssetWeightPct: maxAssetWeightPct,
	})
	if err != nil {
		return nil, err
	}
	return mustJSON(riskLimitsViewMap(v))
}

func (b *Backend) DeletePortfolioRiskLimits(ctx context.Context, clientID string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	if err := b.Portfolio.ClearRiskLimits(ctx, clientID, PortfolioIDFrom(ctx)); err != nil {
		return nil, err
	}
	return mustJSON(map[string]any{"deleted": true, "clientId": clientID})
}

func riskLimitsViewMap(v *portfolio.RiskLimitsView) map[string]any {
	lim := map[string]any{"maxDailyLossPct": v.Limits.MaxDailyLossPct, "maxAssetWeightPct": v.Limits.MaxAssetWeightPct}
	if !v.Limits.UpdatedAt.IsZero() {
		lim["updatedAt"] = v.Limits.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}
	assets := make([]map[string]any, 0, len(v.Status.Assets))
	for _, a := range v.Status.Assets {
		assets = append(assets, map[string]any{
			"asset": a.Asset, "value": a.Value, "weightPct": a.WeightPct, "atOrOverLimit": a.AtOrOverLimit,
		})
	}
	return map[string]any{
		"clientId": v.Limits.ClientID, "limits": lim,
		"status": map[string]any{
			"dayKey": v.Status.DayKey, "timezone": "UTC",
			"startOfDayEquity": v.Status.StartOfDayEquity, "equity": v.Status.Equity,
			"dailyPnl": v.Status.DailyPnL, "dailyPnlPct": v.Status.DailyPnLPct,
			"dailyLossLimitHit": v.Status.DailyLossLimitHit, "assets": assets,
			"canOpenSpotBuy": v.Status.CanOpenSpotBuy, "canOpenMargin": v.Status.CanOpenMargin,
			"blockReasons": v.Status.BlockReasons,
		},
		"note": v.Note,
	}
}

func (b *Backend) GetPortfolio(ctx context.Context, clientID string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	v, err := b.Portfolio.View(ctx, clientID, PortfolioIDFrom(ctx))
	if err != nil {
		return nil, err
	}
	return portfolioViewJSON(v)
}

func (b *Backend) DepositPortfolioCash(ctx context.Context, clientID string, amount float64, note string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	m, v, err := b.Portfolio.Deposit(ctx, portfolio.CashMoveInput{ClientID: clientID, PortfolioID: PortfolioIDFrom(ctx), Amount: amount, Note: note})
	if err != nil {
		return nil, err
	}
	return cashMoveJSON(m, v)
}

func (b *Backend) WithdrawPortfolioCash(ctx context.Context, clientID string, amount float64, note string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	m, v, err := b.Portfolio.Withdraw(ctx, portfolio.CashMoveInput{ClientID: clientID, PortfolioID: PortfolioIDFrom(ctx), Amount: amount, Note: note})
	if err != nil {
		return nil, err
	}
	return cashMoveJSON(m, v)
}

func (b *Backend) ListPortfolioCashMovements(ctx context.Context, clientID string, limit, offset int) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	list, total, err := b.Portfolio.ListCashMovements(ctx, clientID, limit, offset, PortfolioIDFrom(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(list))
	for i := range list {
		items = append(items, cashMovementMap(&list[i]))
	}
	return mustJSON(map[string]any{
		"clientId": clientID, "movements": items, "count": len(items), "total": total, "limit": limit, "offset": offset,
	})
}

func cashMovementMap(m *domain.CashMovement) map[string]any {
	out := map[string]any{
		"id": m.ID, "kind": string(m.Kind), "amount": m.Amount,
		"cashAfter": m.CashAfter, "netDepositsAfter": m.NetDepositsAfter,
		"createdAt": m.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if m.Note != "" {
		out["note"] = m.Note
	}
	if m.CounterpartyPortfolioID != "" {
		out["counterpartyPortfolioId"] = m.CounterpartyPortfolioID
		out["counterpartyPortfolioName"] = m.CounterpartyPortfolioName
		out["peerMovementId"] = m.PeerMovementID
	}
	return out
}

func (b *Backend) TransferPortfolioCash(ctx context.Context, clientID, fromPortfolioID, toPortfolioID string, amount float64, note string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	if fromPortfolioID == "" {
		fromPortfolioID = PortfolioIDFrom(ctx)
	}
	out, in, fromV, toV, err := b.Portfolio.Transfer(ctx, portfolio.TransferInput{
		ClientID: clientID, FromPortfolioID: fromPortfolioID, ToPortfolioID: toPortfolioID,
		Amount: amount, Note: note,
	})
	if err != nil {
		return nil, err
	}
	fromJSON, err := cashMoveJSON(out, fromV)
	if err != nil {
		return nil, err
	}
	toJSON, err := cashMoveJSON(in, toV)
	if err != nil {
		return nil, err
	}
	var fromMap, toMap map[string]any
	_ = json.Unmarshal(fromJSON, &fromMap)
	_ = json.Unmarshal(toJSON, &toMap)
	return mustJSON(map[string]any{
		"from": fromMap, "to": toMap,
		"note": "Internal transfer between your paper portfolios. Not a deposit or withdrawal.",
	})
}

func cashMoveJSON(m *domain.CashMovement, v *domain.PortfolioView) (json.RawMessage, error) {
	pv, err := portfolioViewJSON(v)
	if err != nil {
		return nil, err
	}
	var pmap map[string]any
	if err := json.Unmarshal(pv, &pmap); err != nil {
		return nil, err
	}
	return mustJSON(map[string]any{"movement": cashMovementMap(m), "portfolio": pmap})
}

func (b *Backend) GetPortfolioPerformance(ctx context.Context, clientID, period string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	p, err := b.Portfolio.GetPerformance(ctx, clientID, period, PortfolioIDFrom(ctx))
	if err != nil {
		return nil, err
	}
	pts := make([]map[string]any, 0, len(p.Points))
	for _, pt := range p.Points {
		pts = append(pts, map[string]any{
			"t":      pt.Time.UTC().Format(time.RFC3339Nano),
			"equity": pt.Equity, "cashBalance": pt.CashBalance,
			"positionsValue": pt.PositionsValue, "marginEquity": pt.MarginEquity,
		})
	}
	m := map[string]any{
		"clientId": p.ClientID, "currency": p.Currency, "period": string(p.Period),
		"startAt":     p.StartAt.UTC().Format(time.RFC3339Nano),
		"endAt":       p.EndAt.UTC().Format(time.RFC3339Nano),
		"startEquity": p.StartEquity, "endEquity": p.EndEquity,
		"changeAmount": p.ChangeAmount, "changePct": p.ChangePct,
		"partial": p.Partial, "pointCount": p.PointCount, "points": pts, "note": p.Note,
	}
	return mustJSON(m)
}

func (b *Backend) PlacePortfolioOrder(ctx context.Context, clientID, exchange, symbol, side string, quantity float64, lotMethod string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	tr, v, err := b.Portfolio.PlaceOrder(ctx, portfolio.OrderInput{
		ClientID: clientID, PortfolioID: PortfolioIDFrom(ctx), Exchange: exchange, Symbol: symbol, Side: side, Quantity: quantity, LotMethod: lotMethod,
		IdempotencyKey: IdempotencyKeyFrom(ctx),
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
			"lotMethod": string(tr.LotMethod), "fee": tr.Fee, "lastPrice": tr.LastPrice,
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
	list, total, err := b.Portfolio.ListTrades(ctx, clientID, limit, offset, PortfolioIDFrom(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(list))
	for _, t := range list {
		items = append(items, map[string]any{
			"id": t.ID, "exchange": string(t.Exchange), "symbol": t.Symbol, "side": string(t.Side),
			"quantity": t.Quantity, "price": t.Price, "notional": t.Notional, "realizedPnL": t.RealizedPnL,
			"pendingOrderId": t.PendingOrderID, "fee": t.Fee, "lastPrice": t.LastPrice,
			"createdAt": t.CreatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	return mustJSON(map[string]any{"clientId": clientID, "trades": items, "count": len(items), "total": total})
}

func (b *Backend) ListPortfolioLots(ctx context.Context, clientID, exchange, symbol, status string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	lots, err := b.Portfolio.ListLots(ctx, clientID, exchange, symbol, status, PortfolioIDFrom(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(lots))
	for _, l := range lots {
		m := map[string]any{
			"id": l.ID, "exchange": string(l.Exchange), "symbol": l.Symbol,
			"quantity": l.Quantity, "originalQuantity": l.OriginalQuantity, "price": l.Price,
			"openedAt": l.OpenedAt.UTC().Format(time.RFC3339Nano), "sourceTradeId": l.SourceTradeID,
		}
		if l.ClosedAt != nil {
			m["closedAt"] = l.ClosedAt.UTC().Format(time.RFC3339Nano)
		}
		items = append(items, m)
	}
	return mustJSON(map[string]any{"lots": items, "count": len(items)})
}

func (b *Backend) PlacePortfolioPendingOrder(ctx context.Context, clientID, exchange, symbol, orderType string, quantity, triggerPrice float64, timeInForce, expiresAt, trailType string, trailValue float64, lotMethod string) (json.RawMessage, error) {
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
		PortfolioID: PortfolioIDFrom(ctx),
		ClientID:    clientID, Exchange: exchange, Symbol: symbol, Type: orderType,
		Quantity: quantity, TriggerPrice: triggerPrice, TimeInForce: timeInForce, ExpiresAt: exp,
		TrailType: trailType, TrailValue: trailValue, LotMethod: lotMethod,
		IdempotencyKey: IdempotencyKeyFrom(ctx),
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

func (b *Backend) PlacePortfolioBracketOrder(ctx context.Context, clientID, exchange, symbol string, quantity, entryPrice, takeProfitPrice, stopLossPrice float64, expiresAt string) (json.RawMessage, error) {
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
	entry, tp, sl, err := b.Portfolio.PlaceBracketOrder(ctx, portfolio.BracketOrderInput{
		PortfolioID: PortfolioIDFrom(ctx),
		ClientID:    clientID, Exchange: exchange, Symbol: symbol, Quantity: quantity,
		EntryPrice: entryPrice, TakeProfitPrice: takeProfitPrice, StopLossPrice: stopLossPrice, ExpiresAt: exp,
	})
	if err != nil {
		return nil, err
	}
	return mustJSON(map[string]any{
		"type": "bracket", "bracketId": entry.BracketID,
		"entry": pendingOrderMap(entry), "takeProfit": pendingOrderMap(tp), "stopLoss": pendingOrderMap(sl),
		"note": "Paper bracket: exits pending until entry fills; size tracks filled qty; OCO exits.",
	})
}

func (b *Backend) PlacePortfolioOCOOrder(ctx context.Context, clientID, exchange, symbol string, quantity, takeProfitPrice, stopLossPrice float64, expiresAt string) (json.RawMessage, error) {
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
	tp, sl, err := b.Portfolio.PlaceOCOOrder(ctx, portfolio.OCOOrderInput{
		PortfolioID: PortfolioIDFrom(ctx),
		ClientID:    clientID, Exchange: exchange, Symbol: symbol, Quantity: quantity,
		TakeProfitPrice: takeProfitPrice, StopLossPrice: stopLossPrice, ExpiresAt: exp,
	})
	if err != nil {
		return nil, err
	}
	return mustJSON(map[string]any{
		"type":       "oco",
		"ocoGroupId": tp.OCOGroupID,
		"takeProfit": pendingOrderMap(tp),
		"stopLoss":   pendingOrderMap(sl),
		"note":       "Paper OCO: take-profit + stop-loss for the same size; one fill cancels or shrinks the other.",
	})
}

func (b *Backend) ListPortfolioOrders(ctx context.Context, clientID, status string, limit, offset int) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	list, err := b.Portfolio.ListPendingOrders(ctx, clientID, status, limit, offset, PortfolioIDFrom(ctx))
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

func (b *Backend) GetPortfolioOrder(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	d, err := b.Portfolio.GetPendingOrderDetail(ctx, clientID, id, PortfolioIDFrom(ctx))
	if err != nil {
		return nil, err
	}
	return mustJSON(map[string]any{
		"order":     pendingOrderMap(&d.Order),
		"lastPrice": d.LastPrice,
		"editable":  d.Editable,
		"amend": map[string]any{
			"availableCashForOrder":     d.AvailableCashForOrder,
			"availableQuantityForOrder": d.AvailableQuantityForOrder,
			"maxRemainingQuantity":      d.MaxRemainingQuantity,
			"minRemainingQuantity":      d.MinRemainingQuantity,
		},
		"note": "Paper pending order. Amend keeps the same id and updates reservations. Not real money.",
	})
}

func (b *Backend) AmendPortfolioOrder(ctx context.Context, clientID, id string, triggerPrice, remainingQuantity *float64) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	o, view, err := b.Portfolio.AmendPendingOrder(ctx, portfolio.AmendPendingOrderInput{
		PortfolioID: PortfolioIDFrom(ctx),
		ClientID:    clientID, OrderID: id, TriggerPrice: triggerPrice, RemainingQuantity: remainingQuantity,
	})
	if err != nil {
		return nil, err
	}
	pv, err := portfolioViewJSON(view)
	if err != nil {
		return nil, err
	}
	var pmap map[string]any
	_ = json.Unmarshal(pv, &pmap)
	note := "Order amended in place; reservations updated. Not real money."
	if o.Status == domain.PendingStatusFilled {
		note = "Order amended and filled at last price. Not real money."
	}
	return mustJSON(map[string]any{"order": pendingOrderMap(o), "portfolio": pmap, "note": note})
}

func (b *Backend) CancelAllPortfolioOrders(ctx context.Context, clientID, exchange, symbol string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	list, view, err := b.Portfolio.CancelOpenPendingOrders(ctx, portfolio.CancelOpenOrdersInput{
		PortfolioID: PortfolioIDFrom(ctx),
		ClientID:    clientID, Exchange: exchange, Symbol: symbol,
	})
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(list))
	for i := range list {
		items = append(items, pendingOrderMap(&list[i]))
	}
	scope := "all"
	if strings.TrimSpace(symbol) != "" {
		scope = "market"
	} else if strings.TrimSpace(exchange) != "" {
		scope = "exchange"
	}
	out := map[string]any{"orders": items, "canceled": len(items), "scope": scope, "note": "Open paper orders canceled; unused reservations released. Not real money."}
	if view != nil {
		pv, err := portfolioViewJSON(view)
		if err != nil {
			return nil, err
		}
		var pmap map[string]any
		_ = json.Unmarshal(pv, &pmap)
		out["portfolio"] = pmap
	}
	return mustJSON(out)
}

func (b *Backend) CancelPortfolioOrder(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	o, err := b.Portfolio.CancelPendingOrder(ctx, clientID, id, PortfolioIDFrom(ctx))
	if err != nil {
		return nil, err
	}
	return mustJSON(map[string]any{"order": pendingOrderMap(o), "note": "Order canceled; unused reservation released; it will not execute."})
}

func (b *Backend) CreateRecurringBuyPlan(ctx context.Context, clientID, exchange, symbol string, amount float64, frequency, startAt, name, weekday string, dayOfMonth, intervalHours int, timeZone string, hour, minute int, maxPrice, budget float64, endDate, endsAt string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	var start *time.Time
	if startAt != "" {
		t, err := time.Parse(time.RFC3339Nano, startAt)
		if err != nil {
			t, err = time.Parse(time.RFC3339, startAt)
		}
		if err != nil {
			return nil, fmt.Errorf("%w: startAt must be RFC3339", domain.ErrInvalidArgument)
		}
		tu := t.UTC()
		start = &tu
	}
	in := portfolio.RecurringBuyCreateInput{
		PortfolioID: PortfolioIDFrom(ctx),
		ClientID:    clientID, Exchange: exchange, Symbol: symbol, Name: name,
		Amount: amount, Frequency: frequency, Weekday: weekday,
		DayOfMonth: dayOfMonth, IntervalHours: intervalHours, StartAt: start,
		TimeZone: timeZone, MaxPrice: maxPrice, Budget: budget, EndDate: endDate,
	}
	if hour != domain.RecurringHourUnset {
		h, m := hour, minute
		in.Hour, in.Minute = &h, &m
	}
	if endsAt != "" {
		t, err := time.Parse(time.RFC3339Nano, endsAt)
		if err != nil {
			t, err = time.Parse(time.RFC3339, endsAt)
		}
		if err != nil {
			return nil, fmt.Errorf("%w: endsAt must be RFC3339", domain.ErrInvalidArgument)
		}
		tu := t.UTC()
		in.EndsAt = &tu
	}
	plan, err := b.Portfolio.CreateRecurringBuyPlan(ctx, in)
	if err != nil {
		return nil, err
	}
	return mustJSON(recurringPlanMap(plan))
}

func (b *Backend) UpdateRecurringBuyPlan(ctx context.Context, clientID, id, name, frequency, weekday, startAt string, amount float64, dayOfMonth, intervalHours int, timeZone string, hour, minute int, maxPrice, budget float64, endDate, endsAt string, clearEnds bool) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	in := portfolio.RecurringBuyUpdateInput{ClientID: clientID, PlanID: id, PortfolioID: PortfolioIDFrom(ctx)}
	if name != "" {
		in.Name = &name
	}
	if frequency != "" {
		in.Frequency = &frequency
	}
	if weekday != "" {
		in.Weekday = &weekday
	}
	if amount > 0 {
		in.Amount = &amount
	}
	if dayOfMonth > 0 {
		in.DayOfMonth = &dayOfMonth
	}
	if intervalHours > 0 {
		in.IntervalHours = &intervalHours
	}
	if timeZone != "" {
		in.TimeZone = &timeZone
	}
	if hour != domain.RecurringHourUnset {
		h, m := hour, minute
		in.Hour, in.Minute = &h, &m
	}
	if maxPrice >= 0 {
		in.MaxPrice = &maxPrice
	}
	if budget >= 0 {
		in.Budget = &budget
	}
	if clearEnds {
		in.ClearEnds = true
	} else {
		if endDate != "" {
			in.EndDate = &endDate
		}
		if endsAt != "" {
			t, err := time.Parse(time.RFC3339Nano, endsAt)
			if err != nil {
				t, err = time.Parse(time.RFC3339, endsAt)
			}
			if err != nil {
				return nil, fmt.Errorf("%w: endsAt must be RFC3339", domain.ErrInvalidArgument)
			}
			tu := t.UTC()
			in.EndsAt = &tu
		}
	}
	if startAt != "" {
		t, err := time.Parse(time.RFC3339Nano, startAt)
		if err != nil {
			t, err = time.Parse(time.RFC3339, startAt)
		}
		if err != nil {
			return nil, fmt.Errorf("%w: startAt must be RFC3339", domain.ErrInvalidArgument)
		}
		tu := t.UTC()
		in.StartAt = &tu
	}
	plan, err := b.Portfolio.UpdateRecurringBuyPlan(ctx, in)
	if err != nil {
		return nil, err
	}
	return mustJSON(recurringPlanMap(plan))
}

func (b *Backend) ListRecurringBuyPlans(ctx context.Context, clientID string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	list, err := b.Portfolio.ListRecurringBuyPlans(ctx, clientID, PortfolioIDFrom(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(list))
	for i := range list {
		items = append(items, recurringPlanMap(&list[i]))
	}
	return mustJSON(map[string]any{"clientId": clientID, "plans": items, "count": len(items)})
}

func (b *Backend) GetRecurringBuyPlan(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	plan, err := b.Portfolio.GetRecurringBuyPlan(ctx, clientID, id, PortfolioIDFrom(ctx))
	if err != nil {
		return nil, err
	}
	return mustJSON(recurringPlanMap(plan))
}

func (b *Backend) PauseRecurringBuyPlan(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	plan, err := b.Portfolio.PauseRecurringBuyPlan(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	return mustJSON(recurringPlanMap(plan))
}

func (b *Backend) ResumeRecurringBuyPlan(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	plan, err := b.Portfolio.ResumeRecurringBuyPlan(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	return mustJSON(recurringPlanMap(plan))
}

func parseAllocationTargetsJSON(raw string) ([]domain.AllocationTarget, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("%w: targetsJSON is required", domain.ErrInvalidArgument)
	}
	var items []struct {
		Asset     string  `json:"asset"`
		Exchange  string  `json:"exchange"`
		WeightPct float64 `json:"weightPct"`
	}
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, fmt.Errorf("%w: targetsJSON must be a JSON array of {asset,weightPct}", domain.ErrInvalidArgument)
	}
	out := make([]domain.AllocationTarget, 0, len(items))
	for _, it := range items {
		out = append(out, domain.AllocationTarget{
			Asset: it.Asset, Exchange: domain.Exchange(it.Exchange), WeightPct: it.WeightPct,
		})
	}
	return out, nil
}

func allocationBasketMap(b *domain.AllocationBasket) map[string]any {
	tg := make([]map[string]any, 0, len(b.Targets))
	for _, t := range b.Targets {
		m := map[string]any{"asset": t.Asset, "weightPct": t.WeightPct}
		if t.Exchange != "" {
			m["exchange"] = string(t.Exchange)
		}
		tg = append(tg, m)
	}
	return map[string]any{
		"id": b.ID, "clientId": b.ClientID, "name": b.Name, "targets": tg,
		"createdAt": b.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updatedAt": b.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func allocationViewMap(v *portfolio.AllocationBasketView, trades []domain.Trade) map[string]any {
	lines := make([]map[string]any, 0, len(v.Plan.Lines))
	for _, ln := range v.Plan.Lines {
		lines = append(lines, map[string]any{
			"asset": ln.Asset, "exchange": string(ln.Exchange), "symbol": ln.Symbol, "isCash": ln.IsCash,
			"targetPct": ln.TargetPct, "actualPct": ln.ActualPct,
			"currentValue": ln.CurrentValue, "targetValue": ln.TargetValue, "deltaValue": ln.DeltaValue,
			"markPrice": ln.MarkPrice,
		})
	}
	legs := make([]map[string]any, 0, len(v.Plan.Legs))
	for _, l := range v.Plan.Legs {
		legs = append(legs, map[string]any{
			"side": string(l.Side), "asset": l.Asset, "exchange": string(l.Exchange), "symbol": l.Symbol,
			"quantity": l.Quantity, "price": l.Price, "notional": l.Notional, "reason": l.Reason,
		})
	}
	out := map[string]any{
		"basket": allocationBasketMap(&v.Basket), "currency": v.Plan.Currency,
		"equity": v.Plan.Equity, "cash": v.Plan.Cash, "availableCash": v.Plan.AvailableCash,
		"allocations": lines, "legs": legs, "note": v.Note,
	}
	if trades != nil {
		items := make([]map[string]any, 0, len(trades))
		for i := range trades {
			t := trades[i]
			items = append(items, map[string]any{
				"id": t.ID, "symbol": t.Symbol, "side": string(t.Side),
				"quantity": t.Quantity, "price": t.Price, "notional": t.Notional,
			})
		}
		out["trades"] = items
		out["tradeCount"] = len(items)
	}
	return out
}

func (b *Backend) CreatePortfolioBasket(ctx context.Context, clientID, name, targetsJSON string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	tg, err := parseAllocationTargetsJSON(targetsJSON)
	if err != nil {
		return nil, err
	}
	basket, err := b.Portfolio.CreateAllocationBasket(ctx, portfolio.AllocationBasketCreateInput{
		PortfolioID: PortfolioIDFrom(ctx),
		ClientID:    clientID, Name: name, Targets: tg,
	})
	if err != nil {
		return nil, err
	}
	return mustJSON(allocationBasketMap(basket))
}

func (b *Backend) ListPortfolioBaskets(ctx context.Context, clientID string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	list, err := b.Portfolio.ListAllocationBaskets(ctx, clientID, PortfolioIDFrom(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(list))
	for i := range list {
		items = append(items, allocationBasketMap(&list[i]))
	}
	return mustJSON(map[string]any{"clientId": clientID, "baskets": items, "count": len(items)})
}

func (b *Backend) GetPortfolioBasket(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	v, err := b.Portfolio.PreviewAllocationRebalance(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	return mustJSON(allocationViewMap(v, nil))
}

func (b *Backend) UpdatePortfolioBasket(ctx context.Context, clientID, id, name, targetsJSON string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	in := portfolio.AllocationBasketUpdateInput{ClientID: clientID, BasketID: id}
	if name != "" {
		in.Name = &name
	}
	if strings.TrimSpace(targetsJSON) != "" {
		tg, err := parseAllocationTargetsJSON(targetsJSON)
		if err != nil {
			return nil, err
		}
		in.Targets = tg
	}
	basket, err := b.Portfolio.UpdateAllocationBasket(ctx, in)
	if err != nil {
		return nil, err
	}
	return mustJSON(allocationBasketMap(basket))
}

func (b *Backend) DeletePortfolioBasket(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	if err := b.Portfolio.DeleteAllocationBasket(ctx, clientID, id); err != nil {
		return nil, err
	}
	return mustJSON(map[string]any{"deleted": true, "id": id})
}

func (b *Backend) PreviewPortfolioRebalance(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	v, err := b.Portfolio.PreviewAllocationRebalance(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	return mustJSON(allocationViewMap(v, nil))
}

func (b *Backend) RebalancePortfolioBasket(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	v, trades, err := b.Portfolio.ExecuteAllocationRebalance(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	return mustJSON(allocationViewMap(v, trades))
}

func (b *Backend) DeleteRecurringBuyPlan(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	if err := b.Portfolio.DeleteRecurringBuyPlan(ctx, clientID, id); err != nil {
		return nil, err
	}
	return mustJSON(map[string]any{"deleted": true, "id": id})
}

func (b *Backend) ListRecurringBuyRuns(ctx context.Context, clientID, planID string, limit, offset int) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	list, err := b.Portfolio.ListRecurringBuyRuns(ctx, clientID, planID, limit, offset, PortfolioIDFrom(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(list))
	for i := range list {
		items = append(items, recurringRunMap(&list[i]))
	}
	return mustJSON(map[string]any{"planId": planID, "runs": items, "count": len(items)})
}

func (b *Backend) SetMarginMode(ctx context.Context, clientID, mode string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	p, err := b.Portfolio.SetMarginMode(ctx, portfolio.SetMarginModeInput{ClientID: clientID, PortfolioID: PortfolioIDFrom(ctx), Mode: mode})
	if err != nil {
		return nil, err
	}
	return mustJSON(map[string]any{
		"clientId": p.ClientID, "marginMode": string(p.MarginMode),
		"updatedAt": p.UpdatedAt.UTC().Format(time.RFC3339Nano),
	})
}

func (b *Backend) AdjustMargin(ctx context.Context, clientID, positionID string, delta float64) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	pos, err := b.Portfolio.AdjustMargin(ctx, portfolio.MarginAdjustInput{
		PortfolioID: PortfolioIDFrom(ctx),
		ClientID:    clientID, PositionID: positionID, Delta: delta,
	})
	if err != nil {
		return nil, err
	}
	return mustJSON(marginPosMap(pos))
}

func (b *Backend) RepayMarginDebt(ctx context.Context, clientID, positionID string, amount float64) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	pos, tr, err := b.Portfolio.RepayMarginDebt(ctx, portfolio.MarginRepayInput{
		PortfolioID: PortfolioIDFrom(ctx),
		ClientID:    clientID, PositionID: positionID, Amount: amount,
	})
	if err != nil {
		return nil, err
	}
	return mustJSON(map[string]any{"position": marginPosMap(pos), "trade": marginTradeMap(tr)})
}

func (b *Backend) PlaceMarginOrder(ctx context.Context, clientID, exchange, symbol, side, orderType string, quantity float64, leverage int, limitPrice float64, stopLoss, takeProfit *float64) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	pos, ord, err := b.Portfolio.PlaceMarginOrder(ctx, portfolio.MarginOrderInput{
		PortfolioID: PortfolioIDFrom(ctx),
		ClientID:    clientID, Exchange: exchange, Symbol: symbol, Side: side, Type: orderType,
		Quantity: quantity, Leverage: leverage, LimitPrice: limitPrice, StopLoss: stopLoss, TakeProfit: takeProfit,
		IdempotencyKey: IdempotencyKeyFrom(ctx),
	})
	if err != nil {
		return nil, err
	}
	if pos != nil {
		return mustJSON(map[string]any{"type": "market", "position": marginPosMap(pos), "note": "Paper margin position. Not real money."})
	}
	return mustJSON(map[string]any{"type": "limit", "order": marginOrdMap(ord), "note": "Paper margin limit order. Not real money."})
}

func (b *Backend) ListMarginPositions(ctx context.Context, clientID string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	list, err := b.Portfolio.ListMarginPositions(ctx, clientID, PortfolioIDFrom(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(list))
	for i := range list {
		items = append(items, marginPosMap(&list[i]))
	}
	return mustJSON(map[string]any{"clientId": clientID, "positions": items, "count": len(items)})
}

func (b *Backend) GetMarginPosition(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	pos, err := b.Portfolio.GetMarginPosition(ctx, clientID, id, PortfolioIDFrom(ctx))
	if err != nil {
		return nil, err
	}
	return mustJSON(marginPosMap(pos))
}

func (b *Backend) CloseMarginPosition(ctx context.Context, clientID, id string, quantity float64) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	pos, tr, err := b.Portfolio.CloseMarginPosition(ctx, portfolio.MarginCloseInput{
		PortfolioID: PortfolioIDFrom(ctx),
		ClientID:    clientID, PositionID: id, Quantity: quantity,
		IdempotencyKey: IdempotencyKeyFrom(ctx),
	})
	if err != nil {
		return nil, err
	}
	return mustJSON(map[string]any{"position": marginPosMap(pos), "trade": marginTradeMap(tr)})
}

func (b *Backend) SetMarginBrackets(ctx context.Context, clientID, id string, stopLoss, takeProfit *float64, clearSL, clearTP bool) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	pos, err := b.Portfolio.SetMarginBrackets(ctx, portfolio.MarginBracketsInput{
		PortfolioID: PortfolioIDFrom(ctx),
		ClientID:    clientID, PositionID: id, StopLoss: stopLoss, TakeProfit: takeProfit,
		ClearStopLoss: clearSL, ClearTakeProfit: clearTP,
	})
	if err != nil {
		return nil, err
	}
	return mustJSON(marginPosMap(pos))
}

func (b *Backend) ListMarginOrders(ctx context.Context, clientID, status string, limit, offset int) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	list, err := b.Portfolio.ListMarginOrders(ctx, clientID, status, limit, offset, PortfolioIDFrom(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(list))
	for i := range list {
		items = append(items, marginOrdMap(&list[i]))
	}
	return mustJSON(map[string]any{"clientId": clientID, "orders": items, "count": len(items)})
}

func (b *Backend) CancelMarginOrder(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	o, err := b.Portfolio.CancelMarginOrder(ctx, clientID, id, PortfolioIDFrom(ctx))
	if err != nil {
		return nil, err
	}
	return mustJSON(map[string]any{"order": marginOrdMap(o)})
}

func (b *Backend) ListMarginTrades(ctx context.Context, clientID string, limit, offset int) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	list, err := b.Portfolio.ListMarginTrades(ctx, clientID, limit, offset, PortfolioIDFrom(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(list))
	for i := range list {
		items = append(items, marginTradeMap(&list[i]))
	}
	return mustJSON(map[string]any{"clientId": clientID, "trades": items, "count": len(items)})
}

func marginPosMap(p *domain.MarginPosition) map[string]any {
	mode := string(p.Mode)
	if mode == "" {
		mode = string(domain.MarginModeIsolated)
	}
	m := map[string]any{
		"id": p.ID, "clientId": p.ClientID, "exchange": string(p.Exchange), "symbol": p.Symbol,
		"side": string(p.Side), "mode": mode, "quantity": p.Quantity, "entryPrice": p.EntryPrice, "leverage": p.Leverage,
		"margin": p.Margin, "debtPrincipal": p.DebtPrincipal, "debtInterest": p.DebtInterest,
		"debtAsset": string(p.DebtAsset), "debtNotional": p.DebtNotional,
		"liquidationPrice": p.LiquidationPrice, "status": string(p.Status),
		"markPrice": p.MarkPrice, "unrealizedPnL": p.UnrealizedPnL, "realizedPnL": p.RealizedPnL,
		"closeReason": p.CloseReason,
		"openedAt":    p.OpenedAt.UTC().Format(time.RFC3339Nano),
		"updatedAt":   p.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if !p.LastInterestAt.IsZero() {
		m["lastInterestAt"] = p.LastInterestAt.UTC().Format(time.RFC3339Nano)
	}
	if p.StopLoss != nil {
		m["stopLoss"] = *p.StopLoss
	}
	if p.TakeProfit != nil {
		m["takeProfit"] = *p.TakeProfit
	}
	if p.ClosedAt != nil {
		m["closedAt"] = p.ClosedAt.UTC().Format(time.RFC3339Nano)
	}
	return m
}

func marginOrdMap(o *domain.MarginOrder) map[string]any {
	m := map[string]any{
		"id": o.ID, "clientId": o.ClientID, "exchange": string(o.Exchange), "symbol": o.Symbol,
		"side": string(o.Side), "type": string(o.Type), "quantity": o.Quantity, "leverage": o.Leverage,
		"limitPrice": o.LimitPrice, "reservedMargin": o.ReservedMargin, "status": string(o.Status),
		"positionId": o.PositionID, "rejectReason": o.RejectReason, "cancelReason": o.CancelReason,
		"createdAt": o.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updatedAt": o.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if o.StopLoss != nil {
		m["stopLoss"] = *o.StopLoss
	}
	if o.TakeProfit != nil {
		m["takeProfit"] = *o.TakeProfit
	}
	return m
}

func marginTradeMap(t *domain.MarginTrade) map[string]any {
	return map[string]any{
		"id": t.ID, "positionId": t.PositionID, "exchange": string(t.Exchange), "symbol": t.Symbol,
		"side": string(t.Side), "action": t.Action, "quantity": t.Quantity, "price": t.Price,
		"notional": t.Notional, "realizedPnL": t.RealizedPnL, "marginDelta": t.MarginDelta,
		"principalPaid": t.PrincipalPaid, "interestPaid": t.InterestPaid, "leverage": t.Leverage,
		"fee":       t.Fee,
		"createdAt": t.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func recurringPlanMap(p *domain.RecurringBuyPlan) map[string]any {
	m := map[string]any{
		"id": p.ID, "clientId": p.ClientID, "exchange": string(p.Exchange), "symbol": p.Symbol, "name": p.Name,
		"amount": p.Amount, "frequency": string(p.Frequency), "status": string(p.Status),
		"weekday": p.Weekday, "dayOfMonth": p.DayOfMonth, "intervalHours": p.IntervalHours,
		"nextRunAt": p.NextRunAt.UTC().Format(time.RFC3339Nano), "lastPeriodKey": p.LastPeriodKey,
		"createdAt": p.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updatedAt": p.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if p.TimeZone != "" {
		m["timeZone"] = p.TimeZone
	}
	if p.HasLocalTime || p.TimeZone != "" {
		m["hour"] = p.Hour
		m["minute"] = p.Minute
	}
	if p.MaxPrice > 0 {
		m["maxPrice"] = p.MaxPrice
	}
	if p.Budget > 0 {
		m["budget"] = p.Budget
		m["spent"] = p.Spent
		rem := domain.RecurringRemainingBudget(*p)
		if rem < 1e18 {
			m["remainingBudget"] = rem
		}
	}
	if p.EndsAt != nil && !p.EndsAt.IsZero() {
		m["endsAt"] = p.EndsAt.UTC().Format(time.RFC3339Nano)
		m["endDate"] = domain.RecurringEndDate(*p)
	}
	if p.LastRunAt != nil {
		m["lastRunAt"] = p.LastRunAt.UTC().Format(time.RFC3339Nano)
	}
	return m
}

func recurringRunMap(r *domain.RecurringBuyRun) map[string]any {
	return map[string]any{
		"id": r.ID, "planId": r.PlanID, "periodKey": r.PeriodKey, "status": string(r.Status),
		"amount": r.Amount, "quantity": r.Quantity, "price": r.Price, "tradeId": r.TradeID,
		"failReason":   r.FailReason,
		"scheduledFor": r.ScheduledFor.UTC().Format(time.RFC3339Nano),
		"executedAt":   r.ExecutedAt.UTC().Format(time.RFC3339Nano),
	}
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
		"ocoGroupId":  o.OCOGroupID, "ocoPeerId": o.OCOPeerID,
		"bracketId": o.BracketID, "bracketRole": o.BracketRole,
		"trailType": o.TrailType, "trailValue": o.TrailValue, "trailPeak": o.TrailPeak,
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
	mpos := make([]map[string]any, 0, len(v.MarginPositions))
	for i := range v.MarginPositions {
		p := v.MarginPositions[i]
		m := map[string]any{
			"id": p.ID, "clientId": p.ClientID, "exchange": string(p.Exchange), "symbol": p.Symbol,
			"side": string(p.Side), "mode": string(p.Mode), "quantity": p.Quantity,
			"entryPrice": p.EntryPrice, "leverage": p.Leverage, "margin": p.Margin,
			"debtPrincipal": p.DebtPrincipal, "debtInterest": p.DebtInterest, "debtAsset": string(p.DebtAsset),
			"debtNotional": p.DebtNotional, "markPrice": p.MarkPrice, "unrealizedPnL": p.UnrealizedPnL,
			"liquidationPrice": p.LiquidationPrice, "status": string(p.Status),
			"openedAt":  p.OpenedAt.UTC().Format(time.RFC3339Nano),
			"updatedAt": p.UpdatedAt.UTC().Format(time.RFC3339Nano),
		}
		if p.StopLoss != nil {
			m["stopLoss"] = *p.StopLoss
		}
		if p.TakeProfit != nil {
			m["takeProfit"] = *p.TakeProfit
		}
		mpos = append(mpos, m)
	}
	return mustJSON(map[string]any{
		"id": v.ID, "clientId": v.ClientID, "name": v.Name, "role": string(v.Role),
		"currency": v.Currency, "startingBalance": v.StartingBalance,
		"cashBalance": v.CashBalance, "netDeposits": v.NetDeposits, "contributedCapital": v.ContributedCapital,
		"reservedCash": v.ReservedCash, "reservedMargin": v.ReservedMargin,
		"availableCash": v.AvailableCash, "positionsValue": v.PositionsValue,
		"marginMode": string(v.MarginMode), "marginLocked": v.MarginLocked,
		"marginUnrealizedPnL": v.MarginUnrealizedPnL, "marginEquity": v.MarginEquity,
		"equity": v.Equity, "unrealizedPnL": v.UnrealizedPnL, "realizedPnLTotal": v.RealizedPnLTotal,
		"totalPnL": v.TotalPnL, "positions": pos, "marginPositions": mpos, "note": v.Note,
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
	kind := string(domain.EffectiveAlertKind(*a))
	m := map[string]any{
		"id":          a.ID,
		"clientId":    a.ClientID,
		"exchange":    string(a.Exchange),
		"symbol":      a.Symbol,
		"kind":        kind,
		"condition":   string(a.Condition),
		"mode":        mode,
		"targetPrice": a.TargetPrice,
		"status":      string(a.Status),
		"createdAt":   a.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if (domain.IsBookAlert(a.Kind) || domain.IsLiqNotionalAlert(a.Kind)) && a.RangePct > 0 {
		m["rangePct"] = a.RangePct
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

func watchlistAccessJSON(acc *domain.WatchlistAccess) (json.RawMessage, error) {
	items := make([]map[string]any, 0, len(acc.Items))
	for _, it := range acc.Items {
		items = append(items, map[string]any{
			"exchange": string(it.Exchange),
			"symbol":   it.Symbol,
			"note":     it.Note,
			"addedAt":  it.AddedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	return mustJSON(map[string]any{
		"clientId":      acc.ClientID,
		"ownerClientId": acc.OwnerClientID,
		"role":          string(acc.Role),
		"version":       acc.Version,
		"items":         items,
		"updatedAt":     acc.Updated.UTC().Format(time.RFC3339Nano),
	})
}

func shareMap(sh *domain.WatchlistShare) map[string]any {
	return map[string]any{
		"ownerClientId":   sh.OwnerClientID,
		"granteeClientId": sh.GranteeClientID,
		"role":            string(sh.Role),
		"createdAt":       sh.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updatedAt":       sh.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func shareJSON(sh *domain.WatchlistShare) (json.RawMessage, error) {
	return mustJSON(shareMap(sh))
}

func sharesListJSON(ownerClientID string, list []domain.WatchlistShare) (json.RawMessage, error) {
	items := make([]map[string]any, 0, len(list))
	for i := range list {
		items = append(items, shareMap(&list[i]))
	}
	return mustJSON(map[string]any{"ownerClientId": ownerClientID, "shares": items, "count": len(items)})
}

func mustJSON(v any) (json.RawMessage, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

func (b *Backend) CreateScannerRule(ctx context.Context, args map[string]any) (json.RawMessage, error) {
	if b.Scanner == nil {
		return nil, fmt.Errorf("%w: scanner not configured", domain.ErrUpstream)
	}
	clientID, _ := args["clientId"].(string)
	typ, _ := args["type"].(string)
	interval, _ := args["interval"].(string)
	in := scanner.CreateInput{
		ClientID: clientID, Type: typ, Interval: interval,
		Conditions:     stringSliceFromAny(args["conditions"]),
		MatchMode:      strFromAny(args["matchMode"], ""),
		RSIPeriod:      intFromAny(args["rsiPeriod"], 14),
		RSICondition:   strFromAny(args["rsiCondition"], "below"),
		RSIThreshold:   floatFromAny(args["rsiThreshold"], 30),
		MAFastPeriod:   intFromAny(args["maFastPeriod"], 12),
		MASlowPeriod:   intFromAny(args["maSlowPeriod"], 26),
		MADirection:    strFromAny(args["maDirection"], "golden_cross"),
		VolumeLookback: intFromAny(args["volumeLookback"], 20),
		VolumeMinRatio: floatFromAny(args["volumeMinRatio"], 2),
	}
	rule, err := b.Scanner.Create(ctx, in)
	if err != nil {
		return nil, err
	}
	return scannerRuleJSON(rule)
}

func (b *Backend) ListScannerRules(ctx context.Context, clientID string) (json.RawMessage, error) {
	if b.Scanner == nil {
		return nil, fmt.Errorf("%w: scanner not configured", domain.ErrUpstream)
	}
	list, err := b.Scanner.List(ctx, clientID)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(list))
	for i := range list {
		m, err := scannerRuleMap(&list[i])
		if err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return mustJSON(map[string]any{"clientId": clientID, "rules": items, "count": len(items)})
}

func (b *Backend) UpdateScannerRule(ctx context.Context, args map[string]any) (json.RawMessage, error) {
	if b.Scanner == nil {
		return nil, fmt.Errorf("%w: scanner not configured", domain.ErrUpstream)
	}
	in := scanner.UpdateInput{
		ClientID: strFromAny(args["clientId"], ""),
		ID:       strFromAny(args["id"], strFromAny(args["ruleId"], "")),
	}
	if v, ok := args["enabled"]; ok {
		if en, parsed := boolFromAny(v); parsed {
			in.Enabled = &en
		}
	}
	if v, ok := args["interval"]; ok {
		s := strFromAny(v, "")
		in.Interval = &s
	}
	if _, ok := args["conditions"]; ok {
		in.Conditions = stringSliceFromAny(args["conditions"])
	}
	if v, ok := args["matchMode"]; ok {
		s := strFromAny(v, "")
		in.MatchMode = &s
	}
	if v, ok := args["rsiPeriod"]; ok {
		n := intFromAny(v, 0)
		in.RSIPeriod = &n
	}
	if v, ok := args["rsiCondition"]; ok {
		s := strFromAny(v, "")
		in.RSICondition = &s
	}
	if v, ok := args["rsiThreshold"]; ok {
		n := floatFromAny(v, 0)
		in.RSIThreshold = &n
	}
	if v, ok := args["maFastPeriod"]; ok {
		n := intFromAny(v, 0)
		in.MAFastPeriod = &n
	}
	if v, ok := args["maSlowPeriod"]; ok {
		n := intFromAny(v, 0)
		in.MASlowPeriod = &n
	}
	if v, ok := args["maDirection"]; ok {
		s := strFromAny(v, "")
		in.MADirection = &s
	}
	if v, ok := args["volumeLookback"]; ok {
		n := intFromAny(v, 0)
		in.VolumeLookback = &n
	}
	if v, ok := args["volumeMinRatio"]; ok {
		n := floatFromAny(v, 0)
		in.VolumeMinRatio = &n
	}
	rule, err := b.Scanner.Update(ctx, in)
	if err != nil {
		return nil, err
	}
	return scannerRuleJSON(rule)
}

func (b *Backend) DeleteScannerRule(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.Scanner == nil {
		return nil, fmt.Errorf("%w: scanner not configured", domain.ErrUpstream)
	}
	if err := b.Scanner.Delete(ctx, clientID, id); err != nil {
		return nil, err
	}
	return mustJSON(map[string]any{"deleted": true, "id": id})
}

func (b *Backend) ListScannerResults(ctx context.Context, clientID string, limit, offset int) (json.RawMessage, error) {
	if b.Scanner == nil {
		return nil, fmt.Errorf("%w: scanner not configured", domain.ErrUpstream)
	}
	page, err := b.Scanner.ListResultsPage(ctx, clientID, limit, offset)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(page.Results))
	for _, r := range page.Results {
		items = append(items, map[string]any{
			"id": r.ID, "clientId": r.ClientID, "ruleId": r.RuleID, "exchange": string(r.Exchange),
			"symbol": r.Symbol, "ruleType": string(r.RuleType), "interval": r.Interval,
			"marketDataKey": r.MarketDataKey, "matchedAt": r.MatchedAt.UTC().Format(time.RFC3339Nano),
			"summary": r.Summary, "metrics": r.Metrics,
		})
	}
	setups := make([]map[string]any, 0, len(page.Setups))
	for _, s := range page.Setups {
		setups = append(setups, scanner.SetupMap(s))
	}
	return mustJSON(map[string]any{
		"clientId": clientID, "results": items, "count": len(items), "total": page.Total,
		"setups": setups, "hits24h": page.Hits24h,
	})
}

func exportJobJSON(j *domain.ExportJob) (json.RawMessage, error) {
	secs := make([]string, 0, len(j.Sections))
	for _, s := range j.Sections {
		secs = append(secs, string(s))
	}
	m := map[string]any{
		"id": j.ID, "clientId": j.ClientID, "format": string(j.Format), "sections": secs,
		"status": string(j.Status), "progressPct": j.ProgressPct, "stage": j.Stage,
		"errorMessage": j.ErrorMessage, "fileName": j.FileName, "byteSize": j.ByteSize,
		"createdAt": j.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if j.ExpiresAt != nil {
		m["expiresAt"] = j.ExpiresAt.UTC().Format(time.RFC3339Nano)
	}
	if j.StartedAt != nil {
		m["startedAt"] = j.StartedAt.UTC().Format(time.RFC3339Nano)
	}
	if j.FinishedAt != nil {
		m["finishedAt"] = j.FinishedAt.UTC().Format(time.RFC3339Nano)
	}
	if j.Status == domain.ExportCompleted {
		m["downloadUrl"] = "/api/v1/export/" + j.ID + "/download"
	}
	return mustJSON(m)
}

func (b *Backend) StartExport(ctx context.Context, clientID, format string, sections []string) (json.RawMessage, error) {
	if b.Export == nil {
		return nil, fmt.Errorf("%w: export not configured", domain.ErrUpstream)
	}
	if format == "" {
		format = "json"
	}
	job, err := b.Export.Start(ctx, exportsvc.StartInput{ClientID: clientID, Format: format, Sections: sections})
	if err != nil {
		return nil, err
	}
	return exportJobJSON(job)
}

func (b *Backend) GetExport(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.Export == nil {
		return nil, fmt.Errorf("%w: export not configured", domain.ErrUpstream)
	}
	job, err := b.Export.Get(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	return exportJobJSON(job)
}

func (b *Backend) ListExports(ctx context.Context, clientID string, limit, offset int) (json.RawMessage, error) {
	if b.Export == nil {
		return nil, fmt.Errorf("%w: export not configured", domain.ErrUpstream)
	}
	list, err := b.Export.List(ctx, clientID, limit, offset)
	if err != nil {
		return nil, err
	}
	items := make([]json.RawMessage, 0, len(list))
	for i := range list {
		raw, err := exportJobJSON(&list[i])
		if err != nil {
			return nil, err
		}
		items = append(items, raw)
	}
	return mustJSON(map[string]any{"clientId": clientID, "exports": items, "count": len(items)})
}

func (b *Backend) CancelExport(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.Export == nil {
		return nil, fmt.Errorf("%w: export not configured", domain.ErrUpstream)
	}
	job, err := b.Export.Cancel(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	return exportJobJSON(job)
}

func importJobJSON(j *domain.ImportJob) (json.RawMessage, error) {
	secs := map[string]any{}
	for k, v := range j.SectionCounts {
		secs[string(k)] = map[string]any{
			"valid": v.Valid, "invalid": v.Invalid, "willAdd": v.WillAdd, "duplicates": v.Duplicates,
		}
	}
	added := map[string]int{}
	for k, v := range j.AddedCounts {
		added[string(k)] = v
	}
	m := map[string]any{
		"id": j.ID, "clientId": j.ClientID, "format": string(j.Format), "mode": string(j.Mode),
		"status": string(j.Status), "progressPct": j.ProgressPct, "stage": j.Stage,
		"errorMessage": j.ErrorMessage, "sections": secs,
		"totals": map[string]any{
			"valid": j.Totals.Valid, "invalid": j.Totals.Invalid,
			"willAdd": j.Totals.WillAdd, "duplicates": j.Totals.Duplicates,
		},
		"added": added, "fileName": j.FileName, "byteSize": j.ByteSize,
		"createdAt": j.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if j.ExpiresAt != nil {
		m["expiresAt"] = j.ExpiresAt.UTC().Format(time.RFC3339Nano)
	}
	if j.StartedAt != nil {
		m["startedAt"] = j.StartedAt.UTC().Format(time.RFC3339Nano)
	}
	if j.FinishedAt != nil {
		m["finishedAt"] = j.FinishedAt.UTC().Format(time.RFC3339Nano)
	}
	return mustJSON(m)
}

func (b *Backend) PreviewImport(ctx context.Context, clientID, fileName, format string, fileBytes []byte) (json.RawMessage, error) {
	if b.Import == nil {
		return nil, fmt.Errorf("%w: import not configured", domain.ErrUpstream)
	}
	job, err := b.Import.Preview(ctx, dataimport.PreviewInput{
		ClientID: clientID, FileName: fileName, FileBytes: fileBytes, FormatHint: format,
	})
	if err != nil {
		return nil, err
	}
	return importJobJSON(job)
}

func (b *Backend) ConfirmImport(ctx context.Context, clientID, id, mode string) (json.RawMessage, error) {
	if b.Import == nil {
		return nil, fmt.Errorf("%w: import not configured", domain.ErrUpstream)
	}
	job, err := b.Import.Confirm(ctx, clientID, id, mode)
	if err != nil {
		return nil, err
	}
	return importJobJSON(job)
}

func (b *Backend) GetImport(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.Import == nil {
		return nil, fmt.Errorf("%w: import not configured", domain.ErrUpstream)
	}
	job, err := b.Import.Get(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	return importJobJSON(job)
}

func (b *Backend) ListImports(ctx context.Context, clientID string, limit, offset int) (json.RawMessage, error) {
	if b.Import == nil {
		return nil, fmt.Errorf("%w: import not configured", domain.ErrUpstream)
	}
	list, err := b.Import.List(ctx, clientID, limit, offset)
	if err != nil {
		return nil, err
	}
	items := make([]json.RawMessage, 0, len(list))
	for i := range list {
		raw, err := importJobJSON(&list[i])
		if err != nil {
			return nil, err
		}
		items = append(items, raw)
	}
	return mustJSON(map[string]any{"clientId": clientID, "imports": items, "count": len(items)})
}

func (b *Backend) CancelImport(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.Import == nil {
		return nil, fmt.Errorf("%w: import not configured", domain.ErrUpstream)
	}
	job, err := b.Import.Cancel(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	return importJobJSON(job)
}

func (b *Backend) CreatePriceDiffWatch(ctx context.Context, clientID, symbol string, notional, minProfit, minNetDiffPct, minDurationSec, feeBinance, feeCoinbase, feeBybit float64, exchanges []string) (json.RawMessage, error) {
	if b.PriceDiff == nil {
		return nil, fmt.Errorf("%w: price-diff not configured", domain.ErrUpstream)
	}
	w, err := b.PriceDiff.CreateWatch(ctx, pricediff.CreateInput{
		ClientID: clientID, Symbol: symbol, Notional: notional, MinProfit: minProfit,
		MinNetDiffPct: minNetDiffPct, MinDurationSec: minDurationSec,
		FeeBinancePct: feeBinance, FeeCoinbasePct: feeCoinbase, FeeBybitPct: feeBybit,
		Exchanges: exchanges,
	})
	if err != nil {
		return nil, err
	}
	return mustJSON(priceDiffWatchMap(w))
}

func (b *Backend) ListPriceDiffWatches(ctx context.Context, clientID string) (json.RawMessage, error) {
	if b.PriceDiff == nil {
		return nil, fmt.Errorf("%w: price-diff not configured", domain.ErrUpstream)
	}
	list, err := b.PriceDiff.ListWatches(ctx, clientID)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(list))
	for i := range list {
		items = append(items, priceDiffWatchMap(&list[i]))
	}
	return mustJSON(map[string]any{"clientId": clientID, "watches": items, "count": len(items)})
}

func (b *Backend) GetPriceDiffWatch(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.PriceDiff == nil {
		return nil, fmt.Errorf("%w: price-diff not configured", domain.ErrUpstream)
	}
	w, err := b.PriceDiff.GetWatch(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	return mustJSON(priceDiffWatchMap(w))
}

func (b *Backend) UpdatePriceDiffWatch(ctx context.Context, clientID, id string, notional, minProfit, minNetDiffPct, minDurationSec, feeBinance, feeCoinbase, feeBybit *float64, exchanges []string) (json.RawMessage, error) {
	if b.PriceDiff == nil {
		return nil, fmt.Errorf("%w: price-diff not configured", domain.ErrUpstream)
	}
	in := pricediff.UpdateInput{
		ClientID: clientID, ID: id,
		Notional: notional, MinProfit: minProfit, MinNetDiffPct: minNetDiffPct,
		MinDurationSec: minDurationSec,
		FeeBinancePct:  feeBinance, FeeCoinbasePct: feeCoinbase, FeeBybitPct: feeBybit,
	}
	if len(exchanges) > 0 {
		in.Exchanges = &exchanges
	}
	w, err := b.PriceDiff.UpdateWatch(ctx, in)
	if err != nil {
		return nil, err
	}
	return mustJSON(priceDiffWatchMap(w))
}

func (b *Backend) PausePriceDiffWatch(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.PriceDiff == nil {
		return nil, fmt.Errorf("%w: price-diff not configured", domain.ErrUpstream)
	}
	w, err := b.PriceDiff.PauseWatch(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	return mustJSON(priceDiffWatchMap(w))
}

func (b *Backend) ResumePriceDiffWatch(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.PriceDiff == nil {
		return nil, fmt.Errorf("%w: price-diff not configured", domain.ErrUpstream)
	}
	w, err := b.PriceDiff.ResumeWatch(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	return mustJSON(priceDiffWatchMap(w))
}

func (b *Backend) DeletePriceDiffWatch(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.PriceDiff == nil {
		return nil, fmt.Errorf("%w: price-diff not configured", domain.ErrUpstream)
	}
	if err := b.PriceDiff.DeleteWatch(ctx, clientID, id); err != nil {
		return nil, err
	}
	return mustJSON(map[string]any{"deleted": true, "id": id})
}

func (b *Backend) ListPriceDiffOpportunities(ctx context.Context, clientID, status string, limit, offset int) (json.RawMessage, error) {
	if b.PriceDiff == nil {
		return nil, fmt.Errorf("%w: price-diff not configured", domain.ErrUpstream)
	}
	list, err := b.PriceDiff.ListOpportunities(ctx, clientID, status, limit, offset)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(list))
	for i := range list {
		items = append(items, priceDiffOppMap(&list[i]))
	}
	return mustJSON(map[string]any{"clientId": clientID, "opportunities": items, "count": len(items)})
}

func (b *Backend) GetPriceDiffOpportunity(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.PriceDiff == nil {
		return nil, fmt.Errorf("%w: price-diff not configured", domain.ErrUpstream)
	}
	o, err := b.PriceDiff.GetOpportunity(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	return mustJSON(priceDiffOppMap(o))
}

func (b *Backend) QuotePriceDiff(ctx context.Context, symbol, buyExchange, sellExchange string, notional, quantity, feeBuyPct, feeSellPct, minNetDiffPct float64) (json.RawMessage, error) {
	if b.PriceDiff == nil {
		return nil, fmt.Errorf("%w: price-diff not configured", domain.ErrUpstream)
	}
	q, err := b.PriceDiff.Quote(ctx, pricediff.QuoteInput{
		Symbol: symbol, BuyExchange: buyExchange, SellExchange: sellExchange,
		BuyFeePct: feeBuyPct, SellFeePct: feeSellPct, Notional: notional, Quantity: quantity,
		MinNetDiffPct: minNetDiffPct,
	})
	if err != nil {
		return nil, err
	}
	return mustJSON(priceDiffQuoteMap(q))
}

func (b *Backend) ScanPriceDiffQuotes(ctx context.Context, symbol string, notional, quantity, feeBinance, feeCoinbase, feeBybit, minNetDiffPct, minProfitPct, minProfitAmount float64) (json.RawMessage, error) {
	if b.PriceDiff == nil {
		return nil, fmt.Errorf("%w: price-diff not configured", domain.ErrUpstream)
	}
	s, err := b.PriceDiff.QuoteScan(ctx, pricediff.ScanInput{
		Symbol: symbol, Notional: notional, Quantity: quantity,
		FeeBinancePct: feeBinance, FeeCoinbasePct: feeCoinbase, FeeBybitPct: feeBybit,
		MinNetDiffPct: minNetDiffPct, MinProfitPct: minProfitPct, MinProfitAmount: minProfitAmount,
	})
	if err != nil {
		return nil, err
	}
	return mustJSON(priceDiffScanMap(s))
}

func (b *Backend) QuotePriceDiffWatch(ctx context.Context, clientID, watchID string, notional, quantity, minProfitPct, minProfitAmount float64) (json.RawMessage, error) {
	if b.PriceDiff == nil {
		return nil, fmt.Errorf("%w: price-diff not configured", domain.ErrUpstream)
	}
	s, err := b.PriceDiff.QuoteWatch(ctx, clientID, watchID, pricediff.ScanInput{
		Notional: notional, Quantity: quantity, MinProfitPct: minProfitPct, MinProfitAmount: minProfitAmount,
	})
	if err != nil {
		return nil, err
	}
	return mustJSON(priceDiffScanMap(s))
}

func (b *Backend) QuotePriceDiffOpportunity(ctx context.Context, clientID, id string, notional, quantity float64) (json.RawMessage, error) {
	if b.PriceDiff == nil {
		return nil, fmt.Errorf("%w: price-diff not configured", domain.ErrUpstream)
	}
	q, err := b.PriceDiff.QuoteOpportunity(ctx, clientID, id, notional, quantity)
	if err != nil {
		return nil, err
	}
	return mustJSON(priceDiffQuoteMap(q))
}

func priceDiffWatchMap(w *domain.PriceDiffWatch) map[string]any {
	exs := w.WatchExchanges()
	names := make([]string, 0, len(exs))
	for _, ex := range exs {
		names = append(names, string(ex))
	}
	return map[string]any{
		"id": w.ID, "clientId": w.ClientID, "symbol": w.Symbol,
		"notional": w.Notional, "minProfit": w.MinProfit, "minNetDiffPct": w.MinNetDiffPct,
		"minDurationSec": w.MinDurationSec,
		"feeBinancePct":  w.FeeBinancePct, "feeCoinbasePct": w.FeeCoinbasePct, "feeBybitPct": w.FeeBybitPct,
		"exchanges": names,
		"status":    string(w.Status),
		"createdAt": w.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updatedAt": w.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func priceDiffQuoteMap(q *domain.PriceDiffQuote) map[string]any {
	if q == nil {
		return map[string]any{}
	}
	m := map[string]any{
		"symbol": q.Symbol, "buyExchange": string(q.BuyExchange), "sellExchange": string(q.SellExchange),
		"filledQuantity": q.FilledQuantity, "filledRequested": q.FilledRequested,
		"averageBuyPrice": q.AverageBuyPrice, "averageSellPrice": q.AverageSellPrice,
		"buySlippagePct": q.BuySlippagePct, "sellSlippagePct": q.SellSlippagePct, "slippagePct": q.SlippagePct,
		"buyNotional": q.BuyNotional, "sellNotional": q.SellNotional,
		"buyFeePct": q.BuyFeePct, "sellFeePct": q.SellFeePct, "buyFee": q.BuyFee, "sellFee": q.SellFee,
		"costAfterFees": q.CostAfterFees, "proceedsAfterFees": q.ProceedsAfterFees,
		"profitAfterFees": q.ProfitAfterFees, "profitPct": q.ProfitPct,
		"grossProfit": q.GrossProfit, "grossPct": q.GrossPct,
		"buyExhausted": q.BuyExhausted, "sellExhausted": q.SellExhausted,
		"profitable": q.Profitable, "executable": q.Executable, "meetsMinNet": q.MeetsMinNet,
		"live": q.Live, "buyLive": q.BuyLive, "sellLive": q.SellLive,
		"visibleBuyQuantity": q.VisibleBuyQty, "visibleBuyNotional": q.VisibleBuyNotional,
		"visibleSellQuantity": q.VisibleSellQty, "visibleSellNotional": q.VisibleSellNotional,
		"maxLimitedBy": q.MaxLimitedBy, "note": q.Note,
	}
	if q.RequestedNotional != "" {
		m["requestedNotional"] = q.RequestedNotional
	}
	if q.RequestedQuantity != "" {
		m["requestedQuantity"] = q.RequestedQuantity
	}
	if q.BestAsk != "" {
		m["bestAsk"] = q.BestAsk
	}
	if q.BestBid != "" {
		m["bestBid"] = q.BestBid
	}
	if q.MinNetDiffPct != 0 {
		m["minNetDiffPct"] = q.MinNetDiffPct
	}
	if !q.AsOf.IsZero() {
		m["asOf"] = q.AsOf.UTC().Format(time.RFC3339Nano)
	}
	if q.MaxQuantity != "" {
		m["maxQuantity"] = q.MaxQuantity
		m["maxNotional"] = q.MaxNotional
		m["maxAverageBuyPrice"] = q.MaxAverageBuyPrice
		m["maxAverageSellPrice"] = q.MaxAverageSellPrice
		m["maxProfitAfterFees"] = q.MaxProfitAfterFees
		m["maxProfitPct"] = q.MaxProfitPct
	}
	if q.UsedNotional != "" {
		m["usedNotional"] = q.UsedNotional
		m["unusedNotional"] = q.UnusedNotional
	}
	if q.UsedQuantity != "" {
		m["usedQuantity"] = q.UsedQuantity
		m["unusedQuantity"] = q.UnusedQuantity
	}
	m["usedPct"] = q.UsedPct
	return m
}

func priceDiffScanMap(s *domain.PriceDiffQuoteScan) map[string]any {
	if s == nil {
		return map[string]any{"routes": []any{}}
	}
	routes := make([]map[string]any, 0, len(s.Routes))
	for i := range s.Routes {
		routes = append(routes, priceDiffQuoteMap(&s.Routes[i]))
	}
	unavail := make([]map[string]any, 0, len(s.Unavailable))
	for i := range s.Unavailable {
		u := s.Unavailable[i]
		item := map[string]any{"reason": u.Reason, "message": u.Message}
		if u.Exchange != "" {
			item["exchange"] = u.Exchange
		}
		if u.BuyExchange != "" {
			item["buyExchange"] = u.BuyExchange
		}
		if u.SellExchange != "" {
			item["sellExchange"] = u.SellExchange
		}
		unavail = append(unavail, item)
	}
	m := map[string]any{
		"symbol": s.Symbol, "routes": routes, "unavailable": unavail,
		"profitableCount": s.ProfitableCount, "skippedCount": s.SkippedCount,
		"venueCount": s.VenueCount, "loadedVenueCount": s.LoadedVenueCount, "note": s.Note,
	}
	if s.RequestedNotional != "" {
		m["requestedNotional"] = s.RequestedNotional
	}
	if s.RequestedQuantity != "" {
		m["requestedQuantity"] = s.RequestedQuantity
	}
	if s.MinProfitPct != 0 {
		m["minProfitPct"] = s.MinProfitPct
	}
	if s.MinProfitAmount != 0 {
		m["minProfitAmount"] = s.MinProfitAmount
	}
	if s.BestRoute != nil {
		m["bestRoute"] = priceDiffQuoteMap(s.BestRoute)
	}
	return m
}

func priceDiffOppMap(o *domain.PriceDiffOpportunity) map[string]any {
	m := map[string]any{
		"id": o.ID, "watchId": o.WatchID, "clientId": o.ClientID, "symbol": o.Symbol,
		"buyExchange": string(o.BuyExchange), "sellExchange": string(o.SellExchange),
		"buyPrice": o.BuyPrice, "sellPrice": o.SellPrice,
		"grossDiffPct": o.GrossDiffPct, "netDiffPct": o.NetDiffPct, "minNetDiffPct": o.MinNetDiffPct,
		"status":     string(o.Status),
		"openedAt":   o.OpenedAt.UTC().Format(time.RFC3339Nano),
		"lastSeenAt": o.LastSeenAt.UTC().Format(time.RFC3339Nano),
	}
	if o.ClosedAt != nil {
		m["closedAt"] = o.ClosedAt.UTC().Format(time.RFC3339Nano)
	}
	return m
}

func scannerRuleJSON(r *domain.ScannerRule) (json.RawMessage, error) {
	m, err := scannerRuleMap(r)
	if err != nil {
		return nil, err
	}
	return mustJSON(m)
}

func scannerRuleMap(r *domain.ScannerRule) (map[string]any, error) {
	conds := r.SelectedConditions()
	names := make([]string, 0, len(conds))
	for _, c := range conds {
		names = append(names, string(c))
	}
	mode := string(r.MatchMode)
	if mode == "" {
		mode = string(domain.ScannerMatchAll)
	}
	m := map[string]any{
		"id": r.ID, "clientId": r.ClientID, "type": string(r.Type),
		"conditions": names, "matchMode": mode,
		"interval": r.Interval, "enabled": r.Enabled,
		"createdAt": r.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updatedAt": r.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	for _, c := range conds {
		switch c {
		case domain.ScannerRuleRSI:
			m["rsiPeriod"] = r.RSIPeriod
			m["rsiCondition"] = string(r.RSICondition)
			m["rsiThreshold"] = r.RSIThreshold
		case domain.ScannerRuleMACrossover:
			m["maFastPeriod"] = r.MAFastPeriod
			m["maSlowPeriod"] = r.MASlowPeriod
			m["maDirection"] = r.MADirection
		case domain.ScannerRuleVolumeIncrease:
			m["volumeLookback"] = r.VolumeLookback
			m["volumeMinRatio"] = r.VolumeMinRatio
		}
	}
	return m, nil
}

func boolFromAny(v any) (bool, bool) {
	switch t := v.(type) {
	case bool:
		return t, true
	case string:
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "true", "1", "yes", "on":
			return true, true
		case "false", "0", "no", "off":
			return false, true
		}
	}
	return false, false
}

func intFromAny(v any, def int) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	default:
		return def
	}
}

func floatFromAny(v any, def float64) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	default:
		return def
	}
}

func strFromAny(v any, def string) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return def
}

func stringSliceFromAny(v any) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, x := range t {
			if s, ok := x.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		t = strings.TrimSpace(t)
		if t == "" {
			return nil
		}
		parts := strings.Split(t, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		return out
	default:
		return nil
	}
}

func (b *Backend) AnalyzeSwing(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	if b.Swing == nil {
		return nil, fmt.Errorf("%w: swing service not configured", domain.ErrUpstream)
	}
	dec, err := b.Swing.Analyze(ctx, exchange, symbol)
	if err != nil {
		return nil, err
	}
	return mustJSON(swingDecisionMap(dec))
}

func (b *Backend) ScanSwingSetups(ctx context.Context, clientID, exchange string, limit int) (json.RawMessage, error) {
	if b.Swing == nil {
		return nil, fmt.Errorf("%w: swing service not configured", domain.ErrUpstream)
	}
	list, err := b.Swing.ScanWatchlist(ctx, clientID, exchange, limit)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(list))
	accepted := 0
	for i := range list {
		items = append(items, swingDecisionMap(&list[i]))
		if list[i].Accepted {
			accepted++
		}
	}
	return mustJSON(map[string]any{
		"items": items, "count": len(items), "accepted": accepted,
		"note": "Informational only — not financial advice.",
	})
}

func swingDecisionMap(d *domain.SwingDecision) map[string]any {
	if d == nil {
		return map[string]any{}
	}
	pats := make([]map[string]any, 0, len(d.Patterns))
	for _, p := range d.Patterns {
		pats = append(pats, map[string]any{
			"name": p.Name, "score": p.Score, "description": p.Description,
			"timeframe": p.Timeframe, "fresh": p.Fresh,
		})
	}
	m := map[string]any{
		"exchange": string(d.Exchange), "symbol": d.Symbol, "interval": d.Interval,
		"accepted": d.Accepted, "stage": d.Stage, "setupType": d.SetupType,
		"swingScore": d.SwingScore, "grade": d.Grade, "fresh": d.Fresh,
		"btcRegime": d.BTCRegime, "side": d.Side, "price": d.Price,
		"ema4h": d.EMA4h, "ema1d": d.EMA1d, "patterns": pats, "reasons": d.Reasons,
		"note": d.Note,
	}
	if d.ADX4h != nil {
		m["adx4h"] = *d.ADX4h
	}
	if d.ADX1d != nil {
		m["adx1d"] = *d.ADX1d
	}
	if d.RSI != nil {
		m["rsi"] = *d.RSI
	}
	if !d.BarTime.IsZero() {
		m["barTime"] = d.BarTime.UTC().Format(time.RFC3339Nano)
	}
	if d.Levels != nil {
		m["levels"] = map[string]any{
			"entry": d.Levels.Entry, "stopLoss": d.Levels.StopLoss, "takeProfit": d.Levels.TakeProfit,
			"riskPct": d.Levels.RiskPct, "rewardPct": d.Levels.RewardPct, "rr": d.Levels.RR, "atr": d.Levels.ATR,
		}
	}
	return m
}
