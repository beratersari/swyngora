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
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/radar"
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
	Radar      *radar.Service
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

func (b *Backend) GetBookLiquidity(ctx context.Context, exchange, symbol, window, startTime, endTime string, minPrice, maxPrice float64) (json.RawMessage, error) {
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
	got, err := b.Market.GetBookLiquidity(ctx, exchange, symbol, window, startPtr, endPtr, minPrice, maxPrice)
	if err != nil {
		return nil, err
	}
	return mustJSON(bookLiquidityMap(got))
}

func bookLiquidityMap(got *domain.BookLiquidityReport) map[string]any {
	if got == nil {
		return map[string]any{}
	}
	venues := make([]map[string]any, 0, len(got.Venues))
	for _, v := range got.Venues {
		venues = append(venues, bookLiquidityVenueMap(v))
	}
	out := map[string]any{
		"symbol": got.Symbol, "exchange": got.Exchange, "window": got.Window,
		"from": got.From.UTC().Format(time.RFC3339Nano), "to": got.To.UTC().Format(time.RFC3339Nano),
		"asOf": got.AsOf.UTC().Format(time.RFC3339Nano), "venues": venues, "summary": got.Summary, "note": got.Note,
	}
	if got.MinPrice > 0 {
		out["minPrice"] = formatQtyMCP(got.MinPrice)
	}
	if got.MaxPrice > 0 {
		out["maxPrice"] = formatQtyMCP(got.MaxPrice)
	}
	if got.Combined != nil {
		out["combined"] = bookLiquidityVenueMap(*got.Combined)
	}
	return out
}

func bookLiquidityVenueMap(v domain.BookLiquidityVenue) map[string]any {
	out := map[string]any{
		"exchange": string(v.Exchange), "symbol": v.Symbol,
		"from": v.From.UTC().Format(time.RFC3339Nano), "to": v.To.UTC().Format(time.RFC3339Nano),
		"tickSize": formatQtyMCP(v.TickSize), "samples": v.Samples,
		"bids": bookLiquiditySideMap(v.Bids), "asks": bookLiquiditySideMap(v.Asks),
		"series": bookLiquiditySeriesMap(v.Series), "complete": v.Complete, "truncated": v.Truncated,
		"summary": v.Summary,
	}
	if !v.KnownFrom.IsZero() {
		out["knownFrom"] = v.KnownFrom.UTC().Format(time.RFC3339Nano)
	}
	if !v.KnownTo.IsZero() {
		out["knownTo"] = v.KnownTo.UTC().Format(time.RFC3339Nano)
	}
	if v.Error != "" {
		out["error"] = v.Error
	}
	return out
}

func bookLiquiditySideMap(s domain.BookLiquiditySide) map[string]any {
	levels := make([]map[string]any, 0, len(s.Levels))
	for _, lv := range s.Levels {
		levels = append(levels, map[string]any{
			"side": lv.Side, "price": formatQtyMCP(lv.Price),
			"fromNotional": formatQtyMCP(lv.FromNotional), "toNotional": formatQtyMCP(lv.ToNotional),
			"added": formatQtyMCP(lv.Added), "pulled": formatQtyMCP(lv.Pulled),
			"traded": formatQtyMCP(lv.Traded), "remaining": formatQtyMCP(lv.Remaining), "read": lv.Read,
		})
	}
	return map[string]any{
		"side": s.Side, "fromNotional": formatQtyMCP(s.FromNotional), "toNotional": formatQtyMCP(s.ToNotional),
		"added": formatQtyMCP(s.Added), "pulled": formatQtyMCP(s.Pulled), "traded": formatQtyMCP(s.Traded),
		"read": s.Read, "levels": levels,
	}
}

func bookLiquiditySeriesMap(in []domain.BookLiquidityPoint) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, p := range in {
		out = append(out, map[string]any{
			"from": p.From.UTC().Format(time.RFC3339Nano), "to": p.To.UTC().Format(time.RFC3339Nano),
			"bidAdded": formatQtyMCP(p.BidAdded), "bidPulled": formatQtyMCP(p.BidPulled), "bidTraded": formatQtyMCP(p.BidTraded),
			"askAdded": formatQtyMCP(p.AskAdded), "askPulled": formatQtyMCP(p.AskPulled), "askTraded": formatQtyMCP(p.AskTraded),
		})
	}
	return out
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

func (b *Backend) ScanSuspiciousWalls(ctx context.Context, exchange, quote string, limit int) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.ScanSuspiciousWalls(ctx, exchange, quote, limit)
	if err != nil {
		return nil, err
	}
	if got == nil {
		return mustJSON(map[string]any{})
	}
	hits := make([]map[string]any, 0, len(got.Hits))
	for _, h := range got.Hits {
		hits = append(hits, map[string]any{
			"exchange": h.Exchange, "symbol": h.Symbol, "side": h.Side, "price": h.Price,
			"notional": h.Notional, "distancePct": h.DistancePct, "share": h.Share,
			"kind": h.Kind, "appearCount": h.AppearCount, "ageSeconds": h.AgeSeconds,
			"presentForSeconds": h.PresentForSeconds, "visibleSeconds": h.VisibleSeconds,
			"present": h.Present, "lastSeen": h.LastSeen,
		})
	}
	return mustJSON(map[string]any{
		"exchange": got.Exchange, "quote": got.Quote, "universe": got.Universe, "asOf": got.AsOf,
		"suspicious": got.Suspicious, "flickers": got.Flickers, "hits": hits,
		"summary": got.Summary, "note": got.Note,
	})
}

func (b *Backend) GetBookGaps(ctx context.Context, exchange, symbol string, rangePct float64) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetBookGaps(ctx, exchange, symbol, rangePct)
	if err != nil {
		return nil, err
	}
	return mustJSON(bookGapsJSON(got))
}

func (b *Backend) ScanBookGaps(ctx context.Context, exchange, quote string, rangePct float64, limit int) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.ScanBookGaps(ctx, exchange, quote, rangePct, limit)
	if err != nil {
		return nil, err
	}
	if got == nil {
		return mustJSON(map[string]any{})
	}
	hits := make([]map[string]any, 0, len(got.Hits))
	for _, h := range got.Hits {
		hits = append(hits, map[string]any{
			"symbol": h.Symbol, "scope": h.Scope, "easier": h.Easier,
			"upNotional": h.UpNotional, "downNotional": h.DownNotional,
			"upGapPct": h.UpGapPct, "downGapPct": h.DownGapPct,
			"widestSide": h.WidestSide, "widestPct": h.WidestPct,
			"summary": h.Summary, "error": h.Error,
		})
	}
	return mustJSON(map[string]any{
		"exchange": got.Exchange, "quote": got.Quote, "rangePct": got.RangePct,
		"universe": got.Universe, "hits": hits, "summary": got.Summary, "note": got.Note,
	})
}

func (b *Backend) GetWideSpread(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetWideSpread(ctx, exchange, symbol)
	if err != nil {
		return nil, err
	}
	if got == nil {
		return mustJSON(map[string]any{})
	}
	return mustJSON(map[string]any{
		"symbol": got.Symbol, "exchange": got.Exchange,
		"currentPct": formatQtyMCP(got.CurrentPct), "typicalPct": formatQtyMCP(got.TypicalPct),
		"ratio": formatQtyMCP(got.Ratio), "samples": got.Samples, "opened": got.Opened,
		"summary": got.Summary, "note": got.Note,
	})
}

func (b *Backend) ScanWideSpreads(ctx context.Context, exchange, quote string, limit int) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.ScanWideSpreads(ctx, exchange, quote, limit)
	if err != nil {
		return nil, err
	}
	if got == nil {
		return mustJSON(map[string]any{"hits": []any{}})
	}
	hits := make([]map[string]any, 0, len(got.Hits))
	for _, h := range got.Hits {
		hits = append(hits, map[string]any{
			"exchange": string(h.Exchange), "symbol": h.Symbol, "base": h.Base,
			"currentPct": formatQtyMCP(h.CurrentPct), "typicalPct": formatQtyMCP(h.TypicalPct),
			"ratio": formatQtyMCP(h.Ratio), "samples": h.Samples,
		})
	}
	return mustJSON(map[string]any{
		"exchange": got.Exchange, "quote": got.Quote, "known": got.Known, "ready": got.Ready,
		"hits": hits, "summary": got.Summary, "note": got.Note,
	})
}

func bookGapsJSON(got *domain.BookGaps) map[string]any {
	if got == nil {
		return map[string]any{}
	}
	gaps := make([]map[string]any, 0, len(got.Gaps))
	for _, g := range got.Gaps {
		gaps = append(gaps, map[string]any{
			"side": g.Side, "direction": g.Direction, "fromPrice": g.FromPrice, "toPrice": g.ToPrice,
			"fromPct": g.FromPct, "toPct": g.ToPct, "widthPct": g.WidthPct, "notional": g.Notional,
		})
	}
	return map[string]any{
		"symbol": got.Symbol, "exchange": got.Exchange, "midPrice": got.MidPrice, "rangePct": got.RangePct,
		"easier": got.Easier, "upNotional": got.UpNotional, "downNotional": got.DownNotional,
		"upGapPct": got.UpGapPct, "downGapPct": got.DownGapPct, "gaps": gaps,
		"summary": got.Summary, "note": got.Note,
	}
}

func (b *Backend) ScanOrderBookImpact(ctx context.Context, exchange, quote, side string, notional float64, limit int) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.ScanOrderBookImpact(ctx, exchange, quote, side, notional, limit)
	if err != nil {
		return nil, err
	}
	if got == nil {
		return mustJSON(map[string]any{})
	}
	hits := make([]map[string]any, 0, len(got.Hits))
	for _, h := range got.Hits {
		row := map[string]any{
			"symbol": h.Symbol, "scope": h.Scope, "side": h.Side,
			"slippagePct": h.SlippagePct, "slippageVsBestPct": h.SlippageVsBestPct,
			"impactPct": h.ImpactPct, "impactAvailable": h.ImpactAvailable, "exhausted": h.Exhausted,
			"averagePrice": h.AveragePrice, "bestPrice": h.BestPrice, "spentNotional": h.SpentNotional,
			"levelsUsed": h.LevelsUsed, "summary": h.Summary,
		}
		if h.Error != "" {
			row["error"] = h.Error
		}
		hits = append(hits, row)
	}
	return mustJSON(map[string]any{
		"exchange": got.Exchange, "quote": got.Quote, "side": got.Side, "notional": got.Notional,
		"universe": got.Universe, "asOf": got.AsOf.UTC().Format(time.RFC3339Nano),
		"medianSlippagePct": got.MedianSlippagePct, "exhaustedCoins": got.ExhaustedCoins,
		"hits": hits, "summary": got.Summary, "note": got.Note,
	})
}

func (b *Backend) GetOptions(ctx context.Context, exchange, asset string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetOptions(ctx, exchange, asset)
	if err != nil {
		return nil, err
	}
	expiries := make([]map[string]any, 0, len(got.Expiries))
	for _, e := range got.Expiries {
		row := map[string]any{
			"expiry": e.Expiry.UTC().Format(time.RFC3339), "label": e.Label, "daysToExpiry": e.DaysToExpiry,
			"openInterest": e.OpenInterest, "openInterestUsd": e.OpenInterestUSD,
			"volume": e.Volume, "volumeUsd": e.VolumeUSD,
			"callOi": e.CallOI, "putOi": e.PutOI, "callVolume": e.CallVolume, "putVolume": e.PutVolume,
			"sharePct": e.SharePct, "contractCount": e.ContractCount,
		}
		if e.HasPutCallOI {
			row["putCallOi"] = e.PutCallOI
		}
		if e.HasAtmIV {
			row["atmIv"] = e.AtmIV
		}
		if e.HasCallIV {
			row["callIv"] = e.CallIV
		}
		if e.HasPutIV {
			row["putIv"] = e.PutIV
		}
		if e.HasSkew {
			row["skew"] = e.Skew
		}
		if e.HasExpectedMove {
			row["expectedMoveAbs"] = e.ExpectedMoveAbs
			row["expectedMovePct"] = e.ExpectedMovePct
			row["expectedLow"] = e.ExpectedLow
			row["expectedHigh"] = e.ExpectedHigh
		}
		if e.HasMaxPain {
			row["maxPain"] = e.MaxPain
			row["maxPainPayout"] = e.MaxPainPayout
		}
		if e.HasGamma {
			row["gammaNetUsd"] = e.GammaNetUSD
			row["gammaCallUsd"] = e.GammaCallUSD
			row["gammaPutUsd"] = e.GammaPutUSD
			row["gammaRegime"] = e.GammaRegime
		}
		if e.HasGammaFlip {
			row["gammaFlip"] = e.GammaFlip
			row["gammaFlipSide"] = e.GammaFlipSide
			row["gammaFlipDistPct"] = e.GammaFlipDistPct
		}
		if len(e.Strikes) > 0 {
			strikes := make([]map[string]any, 0, len(e.Strikes))
			for _, s := range e.Strikes {
				st := map[string]any{
					"strike": s.Strike, "callOi": s.CallOI, "putOi": s.PutOI,
					"payout": s.Payout, "isMaxPain": s.IsMaxPain, "belowSpot": s.BelowSpot,
				}
				if s.HasGamma {
					st["gammaNetUsd"] = s.GammaNetUSD
					st["gammaCallUsd"] = s.GammaCallUSD
					st["gammaPutUsd"] = s.GammaPutUSD
				}
				strikes = append(strikes, st)
			}
			row["strikes"] = strikes
		}
		expiries = append(expiries, row)
	}
	term := make([]map[string]any, 0, len(got.TermStructure))
	for _, p := range got.TermStructure {
		row := map[string]any{
			"expiry": p.Expiry.UTC().Format(time.RFC3339), "label": p.Label, "daysToExpiry": p.DaysToExpiry,
		}
		if p.Strike > 0 {
			row["strike"] = p.Strike
		}
		if p.HasAtm {
			row["atmIv"] = p.AtmIV
		}
		if p.HasCall {
			row["callIv"] = p.CallIV
		}
		if p.HasPut {
			row["putIv"] = p.PutIV
		}
		if p.HasSkew {
			row["skew"] = p.Skew
		}
		term = append(term, row)
	}
	out := map[string]any{
		"asset": got.Asset, "exchange": got.Exchange, "indexPrice": got.IndexPrice,
		"asOf":         got.AsOf.UTC().Format(time.RFC3339Nano),
		"openInterest": got.OpenInterest, "openInterestUsd": got.OpenInterestUSD,
		"volume": got.Volume, "volumeUsd": got.VolumeUSD,
		"callOi": got.CallOI, "putOi": got.PutOI, "callVolume": got.CallVolume, "putVolume": got.PutVolume,
		"listed": got.Listed, "withOi": got.WithOI, "withIv": got.WithIV,
		"termStructure": term, "expiries": expiries, "note": got.Note,
	}
	if vm := optionsVsMarketMap(got.VsMarket); vm != nil {
		out["vsMarket"] = vm
	}
	if hist := optionsHistoryMap(got.History); hist != nil {
		out["history"] = hist
	}
	if strike := optionsStrikeOIMap(got.StrikeOI); strike != nil {
		out["strikeOi"] = strike
	}
	if mv := optionsMovesMap(got.Moves); mv != nil {
		out["moves"] = mv
	}
	if got.HasPutCallOI {
		out["putCallOi"] = got.PutCallOI
		out["lean"] = got.Lean
	}
	if got.HasPutCallVol {
		out["putCallVolume"] = got.PutCallVolume
	}
	if got.HasDVOL {
		out["dvol"] = got.DVOL
	}
	if got.HasDVOLChange {
		out["dvolChange24h"] = got.DVOLChange24h
	}
	if got.ATM != nil {
		atm := map[string]any{
			"iv": got.ATM.IV, "strike": got.ATM.Strike,
			"expiry": got.ATM.Expiry.UTC().Format(time.RFC3339), "daysToExpiry": got.ATM.DaysToExpiry,
		}
		if got.ATM.HasCall {
			atm["callIv"] = got.ATM.CallIV
		}
		if got.ATM.HasPut {
			atm["putIv"] = got.ATM.PutIV
		}
		out["atm"] = atm
	}
	if gm := optionsGammaMap(got.Gamma); gm != nil {
		out["gamma"] = gm
	}
	if vc := optionsVolCompareMap(got.VolCompare); vc != nil {
		out["volCompare"] = vc
	}
	if got.Skew != nil {
		sk := map[string]any{"lean": got.Skew.Lean}
		if got.Skew.HasAtmSkew {
			sk["atmSkew"] = got.Skew.AtmSkew
			sk["callIv"] = got.Skew.CallIV
			sk["putIv"] = got.Skew.PutIV
		}
		if got.Skew.HasTerm {
			sk["frontIv"] = got.Skew.FrontIV
			sk["backIv"] = got.Skew.BackIV
			sk["termSlope"] = got.Skew.TermSlope
			sk["frontDte"] = got.Skew.FrontDTE
			sk["backDte"] = got.Skew.BackDTE
		}
		if got.Skew.HasWing {
			sk["wingSkew"] = got.Skew.WingSkew
			sk["wingPutIv"] = got.Skew.WingPutIV
			sk["wingCallIv"] = got.Skew.WingCallIV
		}
		out["skew"] = sk
	}
	return mustJSON(out)
}

func optionsVolCompareMap(v *domain.OptionsVolCompare) map[string]any {
	if v == nil || (!v.HasDVOL && !v.HasRealized) {
		return nil
	}
	out := map[string]any{"days": v.Days, "lean": v.Lean}
	if v.Bars > 0 {
		out["bars"] = v.Bars
	}
	if v.Summary != "" {
		out["summary"] = v.Summary
	}
	if v.HasDVOL {
		out["dvol"] = v.DVOL
	}
	if v.HasRealized {
		out["realized"] = v.Realized
	}
	if v.HasGap {
		out["gap"] = v.Gap
	}
	return out
}

func optionsGammaMap(g *domain.OptionsGamma) map[string]any {
	if g == nil {
		return nil
	}
	out := map[string]any{
		"netUsd": g.NetUSD, "callUsd": g.CallUSD, "putUsd": g.PutUSD,
		"regime": g.Regime, "used": g.Used,
	}
	if g.Skipped > 0 {
		out["skipped"] = g.Skipped
	}
	if g.HasFlip {
		out["flip"] = g.Flip
		out["flipSide"] = g.FlipSide
		out["flipDistPct"] = g.FlipDistPct
	}
	if len(g.Profile) > 0 {
		profile := make([]map[string]any, 0, len(g.Profile))
		for _, p := range g.Profile {
			profile = append(profile, map[string]any{
				"spot": p.Spot, "netUsd": p.NetUSD, "callUsd": p.CallUSD, "putUsd": p.PutUSD,
			})
		}
		out["profile"] = profile
	}
	return out
}

func optionsVsMarketMap(v *domain.OptionsVsMarket) map[string]any {
	if v == nil || !v.Compared {
		return nil
	}
	findings := make([]map[string]any, 0, len(v.Findings))
	for _, f := range v.Findings {
		row := map[string]any{
			"kind": f.Kind, "topic": f.Topic, "severity": f.Severity,
			"title": f.Title, "detail": f.Detail,
		}
		if f.HasNums {
			row["options"] = f.Options
			row["market"] = f.Market
		}
		findings = append(findings, row)
	}
	out := map[string]any{"compared": true, "findings": findings}
	if v.Summary != "" {
		out["summary"] = v.Summary
	}
	if v.HasSpot && v.SpotLast > 0 {
		out["spotLast"] = v.SpotLast
	}
	if v.ImpliedIV > 0 {
		out["impliedIv"] = v.ImpliedIV
	}
	if v.HasVol {
		out["realizedAnn"] = v.RealizedAnn
	}
	if v.Range24hPct > 0 {
		out["range24hPct"] = v.Range24hPct
	}
	if v.HasFunding {
		out["fundingRate"] = v.FundingRate
	}
	if v.HasLongPct {
		out["longPct"] = v.LongPct
	}
	return out
}

func optionsStrikeOIMap(h *domain.OptionsStrikeOI) map[string]any {
	if h == nil {
		return nil
	}
	windows := make([]map[string]any, 0, len(h.Windows))
	for _, w := range h.Windows {
		strikes := make([]map[string]any, 0, len(w.Strikes))
		for _, s := range w.Strikes {
			row := map[string]any{
				"expiry": s.Expiry.UTC().Format(time.RFC3339), "label": s.Label, "strike": s.Strike,
				"callOi": s.CallOI, "putOi": s.PutOI, "callPast": s.CallPast, "putPast": s.PutPast,
				"callChange": s.CallChange, "putChange": s.PutChange, "netChange": s.NetChange,
			}
			if s.Opened {
				row["opened"] = true
			}
			if s.Closed {
				row["closed"] = true
			}
			strikes = append(strikes, row)
		}
		row := map[string]any{"window": w.Window, "complete": w.Complete, "strikes": strikes}
		if !w.ComparedAt.IsZero() {
			row["comparedAt"] = w.ComparedAt.UTC().Format(time.RFC3339)
		}
		windows = append(windows, row)
	}
	return map[string]any{"windows": windows, "note": h.Note}
}

func optionsHistoryMap(h *domain.OptionsHistory) map[string]any {
	if h == nil {
		return nil
	}
	windows := make([]map[string]any, 0, len(h.Windows))
	for _, w := range h.Windows {
		row := map[string]any{"window": w.Window, "complete": w.Complete}
		if w.OI.Complete {
			row["oi"] = optionsHistoryChangeMap(w.OI)
		}
		if w.PutCall.Complete {
			row["putCall"] = optionsHistoryChangeMap(w.PutCall)
		}
		if w.DVOL.Complete {
			row["dvol"] = optionsHistoryChangeMap(w.DVOL)
		}
		if w.Realized.Complete {
			row["realized"] = optionsHistoryChangeMap(w.Realized)
		}
		if w.VolGap.Complete {
			row["volGap"] = optionsHistoryChangeMap(w.VolGap)
		}
		if w.AtmIV.Complete {
			row["atmIv"] = optionsHistoryChangeMap(w.AtmIV)
		}
		if w.Gamma.Complete {
			row["gamma"] = optionsHistoryChangeMap(w.Gamma)
		}
		if w.Flip.Complete {
			row["flip"] = optionsHistoryChangeMap(w.Flip)
		}
		if w.Regime != "" {
			row["regime"] = w.Regime
		}
		if w.RegimeWas != "" {
			row["regimeWas"] = w.RegimeWas
		}
		if w.FlipSide != "" {
			row["flipSide"] = w.FlipSide
		}
		if w.FlipSideWas != "" {
			row["flipSideWas"] = w.FlipSideWas
		}
		windows = append(windows, row)
	}
	points := make([]map[string]any, 0, len(h.Points))
	for _, p := range h.Points {
		row := map[string]any{"time": p.Time.UTC().Format(time.RFC3339)}
		if p.HasOI {
			row["openInterest"] = p.OpenInterest
			if p.OpenInterestUSD > 0 {
				row["openInterestUsd"] = p.OpenInterestUSD
			}
		}
		if p.HasPutCallOI {
			row["putCallOi"] = p.PutCallOI
		}
		if p.HasDVOL {
			row["dvol"] = p.DVOL
		}
		if p.HasRealized {
			row["realizedVol"] = p.Realized
		}
		if p.HasVolGap {
			row["volGap"] = p.VolGap
		}
		if p.HasAtmIV {
			row["atmIv"] = p.AtmIV
		}
		if p.HasIndex {
			row["indexPrice"] = p.IndexPrice
		}
		if p.HasGamma {
			row["gammaNetUsd"] = p.GammaNetUSD
			row["gammaRegime"] = p.GammaRegime
		}
		if p.HasGammaFlip {
			row["gammaFlip"] = p.GammaFlip
			row["gammaFlipSide"] = p.GammaFlipSide
		}
		points = append(points, row)
	}
	out := map[string]any{
		"windows": windows, "points": points,
		"from": h.From.UTC().Format(time.RFC3339), "to": h.To.UTC().Format(time.RFC3339),
	}
	if h.Note != "" {
		out["note"] = h.Note
	}
	return out
}

func optionsMovesMap(m *domain.OptionsMoves) map[string]any {
	if m == nil || len(m.Findings) == 0 {
		return nil
	}
	findings := make([]map[string]any, 0, len(m.Findings))
	for _, f := range m.Findings {
		row := map[string]any{
			"kind": f.Kind, "topic": f.Topic, "window": f.Window, "severity": f.Severity,
			"title": f.Title, "detail": f.Detail,
		}
		if f.HasNums {
			row["current"] = f.Current
			row["typical"] = f.Typical
			row["ratio"] = f.Ratio
		}
		findings = append(findings, row)
	}
	out := map[string]any{"findings": findings}
	if m.Summary != "" {
		out["summary"] = m.Summary
	}
	return out
}

func optionsHistoryChangeMap(c domain.SnapshotChange) map[string]any {
	return map[string]any{
		"window": c.Window, "current": c.Current, "past": c.Past,
		"change": c.Change, "changePct": c.ChangePct,
		"direction": c.Direction, "complete": c.Complete,
	}
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
			"direction": w.Direction, "flow": w.Flow, "openedValue": w.OpenedValue, "closedValue": w.ClosedValue,
			"complete": w.Complete,
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
				"direction": w.Direction, "flow": w.Flow, "openedValue": w.OpenedValue, "closedValue": w.ClosedValue,
				"complete": w.Complete,
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

func (b *Backend) CompareSpotPerpLiquidity(ctx context.Context, symbol string, rangePct float64) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetSpotPerpLiquidity(ctx, symbol, rangePct)
	if err != nil {
		return nil, err
	}
	return mustJSON(spotPerpMap(got))
}

func spotPerpMap(got *domain.SpotPerpLiquidity) map[string]any {
	if got == nil {
		return map[string]any{}
	}
	bands := make([]map[string]any, 0, len(got.Bands))
	for _, b := range got.Bands {
		bands = append(bands, map[string]any{
			"rangePct": formatQtyMCP(b.RangePct), "spotBid": formatQtyMCP(b.SpotBid), "spotAsk": formatQtyMCP(b.SpotAsk),
			"perpBid": formatQtyMCP(b.PerpBid), "perpAsk": formatQtyMCP(b.PerpAsk),
			"thinnerBook": b.ThinnerBook, "strongerBook": b.StrongerBook,
			"spotThinnerSide": b.SpotThinnerSide, "perpThinnerSide": b.PerpThinnerSide, "complete": b.Complete,
		})
	}
	venues := make([]map[string]any, 0, len(got.Venues))
	for _, v := range got.Venues {
		venues = append(venues, map[string]any{
			"exchange": string(v.Exchange), "spotBid": formatQtyMCP(v.SpotBid), "spotAsk": formatQtyMCP(v.SpotAsk),
			"perpBid": formatQtyMCP(v.PerpBid), "perpAsk": formatQtyMCP(v.PerpAsk),
			"thinnerBook": v.ThinnerBook, "spotThinnerSide": v.SpotThinnerSide, "perpThinnerSide": v.PerpThinnerSide,
			"complete": v.Complete, "error": v.Error,
		})
	}
	return map[string]any{
		"symbol": got.Symbol, "rangePct": formatQtyMCP(got.RangePct),
		"spot": spotPerpBookMap(got.Spot), "perp": spotPerpBookMap(got.Perp),
		"thinnerBook": got.ThinnerBook, "strongerBook": got.StrongerBook, "ratio": formatQtyMCP(got.Ratio),
		"bands": bands, "venues": venues, "summary": got.Summary, "note": got.Note,
	}
}

func spotPerpBookMap(b domain.SpotPerpBook) map[string]any {
	return map[string]any{
		"bidNotional": formatQtyMCP(b.BidNotional), "askNotional": formatQtyMCP(b.AskNotional), "total": formatQtyMCP(b.Total),
		"thinnerSide": b.ThinnerSide, "strongerSide": b.StrongerSide, "mid": formatQtyMCP(b.Mid),
		"coverPct": formatQtyMCP(b.CoverPct), "venueCount": b.VenueCount, "complete": b.Complete,
	}
}

func (b *Backend) GetPerpDepth(ctx context.Context, exchange, symbol string, rangePct float64) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetPerpDepth(ctx, exchange, symbol, rangePct)
	if err != nil {
		return nil, err
	}
	return mustJSON(perpDepthMap(got))
}

func (b *Backend) GetDerivativesGrid(ctx context.Context, exchange, metric, rangeID string, limit int) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetDerivativesGrid(ctx, exchange, metric, rangeID, limit)
	if err != nil {
		return nil, err
	}
	return mustJSON(derivativesGridMap(got))
}

func derivativesGridMap(got *domain.DerivativesGrid) map[string]any {
	if got == nil {
		return map[string]any{}
	}
	rows := make([]map[string]any, 0, len(got.Rows))
	for _, row := range got.Rows {
		cells := make([]map[string]any, 0, len(row.Cells))
		for _, cell := range row.Cells {
			item := map[string]any{
				"time":     cell.Time.UTC().Format(time.RFC3339),
				"definite": cell.Definite,
			}
			if len(cell.Missing) > 0 {
				item["missing"] = cell.Missing
			}
			if cell.Definite {
				item["value"] = cell.Value
				if cell.Text != "" {
					item["text"] = cell.Text
				}
				if cell.Weight != "" {
					item["weight"] = cell.Weight
				}
				if cell.HasHeat {
					item["heat"] = cell.Heat
				}
			}
			cells = append(cells, item)
		}
		rows = append(rows, map[string]any{
			"symbol": row.Symbol,
			"base":   row.Base,
			"cells":  cells,
		})
	}
	return map[string]any{
		"exchange": got.Exchange,
		"metric":   got.Metric,
		"range":    got.Range,
		"stepSec":  got.StepSec,
		"asOf":     got.AsOf,
		"columns":  got.Columns,
		"rows":     rows,
		"universe": got.Universe,
		"note":     got.Note,
	}
}

func (b *Backend) GetDerivativesBoard(ctx context.Context, exchange string, limit int) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetDerivativesBoard(ctx, exchange, limit)
	if err != nil {
		return nil, err
	}
	return mustJSON(derivativesBoardMap(got))
}

func derivativesBoardMap(got *domain.DerivativesBoard) map[string]any {
	if got == nil {
		return map[string]any{}
	}
	coins := make([]map[string]any, 0, len(got.Coins))
	for _, c := range got.Coins {
		venues := make([]map[string]any, 0, len(c.Venues))
		for _, v := range c.Venues {
			venues = append(venues, map[string]any{
				"exchange": v.Exchange, "oiValue": v.OIValue, "oiStatus": v.OIStatus,
				"fundingRate": v.FundingRate, "fundingRatePct": v.FundingRatePct, "fundingPayer": v.FundingPayer,
				"fundingStatus": v.FundingStatus, "markPrice": v.MarkPrice,
				"liqLong": v.LiqLong, "liqShort": v.LiqShort, "liqStatus": v.LiqStatus,
				"longPct": v.LongPct, "longStatus": v.LongStatus,
				"takerBuyPct": v.TakerBuyPct, "takerStatus": v.TakerStatus,
				"gapPct": v.GapPct, "gapStatus": v.GapStatus,
				"cvdDay": v.CVDDay, "cvdStatus": v.CVDStatus,
				"printChangePct": v.PrintChangePct, "printStatus": v.PrintStatus,
				"liqGrade": v.LiqGrade, "liqHeatStatus": v.LiqHeatStatus,
				"bidPct": v.BidPct, "bookStatus": v.BookStatus,
			})
		}
		coin := map[string]any{
			"symbol": c.Symbol, "base": c.Base, "price": c.Price, "priceExchange": c.PriceExchange,
			"fundingRate": c.FundingRate, "fundingRatePct": c.FundingRatePct, "fundingPayer": c.FundingPayer,
			"fundingWeight": c.FundingWeight, "fundingDefinite": c.FundingDefinite,
			"openInterest": c.OpenInterest, "openInterestDefinite": c.OpenInterestDefinite,
			"change1h": c.Change1h, "change1hPct": c.Change1hPct, "change1hDefinite": c.Change1hDefinite,
			"change24h": c.Change24h, "change24hPct": c.Change24hPct, "change24hDefinite": c.Change24hDefinite,
			"liqLong": c.LiqLong, "liqShort": c.LiqShort, "liqDefinite": c.LiqDefinite,
			"longPct": c.LongPct, "longDefinite": c.LongDefinite,
			"takerBuyPct": c.TakerBuyPct, "takerDefinite": c.TakerDefinite,
			"gapPct": c.GapPct, "gapDefinite": c.GapDefinite, "gapScope": c.GapScope,
			"cvdDay": c.CVDDay, "cvdDayDefinite": c.CVDDayDefinite,
			"printChangePct": c.PrintChangePct, "printRead": c.PrintRead, "printDefinite": c.PrintDefinite,
			"liqHeat": c.LiqHeat, "liqHeatRatio": c.LiqHeatRatio, "liqHeatVenue": c.LiqHeatVenue, "liqHeatDefinite": c.LiqHeatDefinite,
			"bidPct": c.BidPct, "bidDefinite": c.BidDefinite,
			"venues": venues, "coverage": coverageMap(c.Coverage),
		}
		if len(c.Change24hMissing) > 0 {
			coin["change24hMissing"] = c.Change24hMissing
		}
		coins = append(coins, coin)
	}
	return map[string]any{
		"exchange": got.Exchange, "universe": got.Universe, "asOf": got.AsOf.UTC(),
		"coins": coins, "coverage": coverageMap(got.Coverage), "definite": got.Definite,
		"summary": got.Summary, "note": got.Note,
	}
}

func (b *Backend) GetPerpBookHistory(ctx context.Context, exchange, symbol string, windowSec int) (json.RawMessage, error) {
	if b == nil || b.Market == nil {
		return nil, fmt.Errorf("market service unavailable")
	}
	got, err := b.Market.GetPerpBookHistory(ctx, exchange, symbol, windowSec, nil, nil)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetPerpDepthOverview(ctx context.Context, exchange string, rangePct float64, limit int) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetPerpDepthOverview(ctx, exchange, rangePct, limit)
	if err != nil {
		return nil, err
	}
	return mustJSON(perpDepthOverviewMap(got))
}

func perpDepthOverviewMap(got *domain.PerpDepthOverview) map[string]any {
	if got == nil {
		return map[string]any{}
	}
	coins := make([]map[string]any, 0, len(got.Coins))
	for _, c := range got.Coins {
		row := map[string]any{
			"symbol": c.Symbol, "base": c.Base, "bid": formatQtyMCP(c.Bid), "ask": formatQtyMCP(c.Ask),
			"total": formatQtyMCP(c.Total), "sharePct": formatQtyMCP(c.SharePct),
			"bidSharePct": formatQtyMCP(c.BidSharePct), "askSharePct": formatQtyMCP(c.AskSharePct),
			"lean": c.Lean, "complete": c.Complete,
		}
		if c.Error != "" {
			row["error"] = c.Error
		}
		coins = append(coins, row)
	}
	wins := make([]map[string]any, 0, len(got.Windows))
	for _, w := range got.Windows {
		row := map[string]any{
			"window": w.Window, "bidPct": formatQtyMCP(w.BidPct), "askPct": formatQtyMCP(w.AskPct),
			"kind": w.Kind, "complete": w.Complete, "summary": w.Summary,
		}
		if w.Complete {
			row["pastBidPct"] = formatQtyMCP(w.PastBidPct)
			row["pastAskPct"] = formatQtyMCP(w.PastAskPct)
			row["changePct"] = domain.FormatSignedPct(w.ChangePct)
			row["total"] = formatQtyMCP(w.Total)
			row["pastTotal"] = formatQtyMCP(w.PastTotal)
			row["totalChange"] = domain.FormatSignedQty(w.TotalChange)
			row["totalChangePct"] = domain.FormatSignedPct(w.TotalChangePct)
			row["depthKind"] = w.DepthKind
			if !w.SampleTime.IsZero() {
				row["sampleTime"] = w.SampleTime.UTC().Format(time.RFC3339Nano)
			}
			drivers := make([]map[string]any, 0, len(w.Drivers))
			for _, d := range w.Drivers {
				drivers = append(drivers, map[string]any{
					"symbol": d.Symbol, "base": d.Base, "changePct": domain.FormatSignedPct(d.ChangePct), "kind": d.Kind,
				})
			}
			row["drivers"] = drivers
			sizes := make([]map[string]any, 0, len(w.SizeDrivers))
			for _, d := range w.SizeDrivers {
				sizes = append(sizes, map[string]any{
					"symbol": d.Symbol, "base": d.Base, "change": domain.FormatSignedQty(d.Change), "kind": d.Kind,
				})
			}
			row["sizeDrivers"] = sizes
		}
		wins = append(wins, row)
	}
	history := make([]map[string]any, 0, len(got.History))
	for _, p := range got.History {
		row := map[string]any{
			"time": p.Time.UTC().Format(time.RFC3339Nano), "total": formatQtyMCP(p.Total),
		}
		if p.Price > 0 {
			row["price"] = formatQtyMCP(p.Price)
		}
		history = append(history, row)
	}
	moves := make([]map[string]any, 0, len(got.Moves))
	for _, m := range got.Moves {
		row := map[string]any{
			"kind": m.Kind, "from": m.From.UTC().Format(time.RFC3339Nano), "to": m.To.UTC().Format(time.RFC3339Nano),
			"pastTotal": formatQtyMCP(m.PastTotal), "total": formatQtyMCP(m.Total),
			"change": domain.FormatSignedQty(m.Change), "changePct": domain.FormatSignedPct(m.ChangePct),
			"summary": m.Summary,
		}
		if m.PriceFrom > 0 {
			row["priceFrom"] = formatQtyMCP(m.PriceFrom)
			row["priceTo"] = formatQtyMCP(m.PriceTo)
			row["priceChangePct"] = domain.FormatSignedPct(m.PriceChangePct)
			row["priceKind"] = m.PriceKind
		}
		if m.PriceAfter > 0 {
			row["priceAfter"] = formatQtyMCP(m.PriceAfter)
			row["priceAfterPct"] = domain.FormatSignedPct(m.PriceAfterPct)
			row["afterKind"] = m.AfterKind
		}
		drivers := make([]map[string]any, 0, len(m.Drivers))
		for _, d := range m.Drivers {
			drivers = append(drivers, map[string]any{
				"symbol": d.Symbol, "base": d.Base, "change": domain.FormatSignedQty(d.Change),
				"sharePct": domain.FormatSignedPct(d.SharePct), "kind": d.Kind,
			})
		}
		row["drivers"] = drivers
		if m.Span != "" {
			row["span"] = m.Span
		}
		if m.AfterTotal > 0 {
			row["afterTotal"] = formatQtyMCP(m.AfterTotal)
			row["follow"] = m.Follow
			row["followPct"] = domain.FormatSignedPct(m.FollowPct)
		}
		moves = append(moves, row)
	}
	gaps := make([]map[string]any, 0, len(got.Gaps))
	for _, g := range got.Gaps {
		missing, stale := g.Missing, g.Stale
		if missing == nil {
			missing = []string{}
		}
		if stale == nil {
			stale = []string{}
		}
		gaps = append(gaps, map[string]any{
			"time": g.Time.UTC().Format(time.RFC3339Nano), "missing": missing, "stale": stale,
		})
	}
	bands := make([]map[string]any, 0, len(got.Bands))
	for _, b := range got.Bands {
		rowCoins := make([]map[string]any, 0, len(b.Coins))
		for _, c := range b.Coins {
			rowCoins = append(rowCoins, map[string]any{
				"symbol": c.Symbol, "base": c.Base, "bid": formatQtyMCP(c.Bid), "ask": formatQtyMCP(c.Ask),
				"total": formatQtyMCP(c.Total), "sharePct": formatQtyMCP(c.SharePct),
			})
		}
		row := map[string]any{
			"rangePct": formatQtyMCP(b.RangePct), "bid": formatQtyMCP(b.Bid), "ask": formatQtyMCP(b.Ask),
			"total": formatQtyMCP(b.Total), "bidPct": formatQtyMCP(b.BidPct), "askPct": formatQtyMCP(b.AskPct),
			"lean": b.Lean, "ofHeadlinePct": formatQtyMCP(b.OfHeadlinePct), "addedPct": formatQtyMCP(b.AddedPct),
			"addedBid": formatQtyMCP(b.AddedBid), "addedAsk": formatQtyMCP(b.AddedAsk), "complete": b.Complete,
			"leaders": b.Leaders, "coins": rowCoins, "summary": b.Summary,
		}
		if len(b.AddedLeaders) > 0 {
			row["addedLeaders"] = b.AddedLeaders
		}
		bands = append(bands, row)
	}
	out := map[string]any{
		"exchange": got.Exchange, "rangePct": formatQtyMCP(got.RangePct), "universe": got.Universe, "known": got.Known,
		"asOf": got.AsOf.UTC().Format(time.RFC3339Nano), "bid": formatQtyMCP(got.Bid), "ask": formatQtyMCP(got.Ask),
		"total": formatQtyMCP(got.Total), "imbalance": domain.FormatSignedQty(got.Imbalance),
		"bidPct": formatQtyMCP(got.BidPct), "askPct": formatQtyMCP(got.AskPct), "lean": got.Lean,
		"coins": coins, "leaders": got.Leaders, "windows": wins, "history": history, "moves": moves, "gaps": gaps, "bands": bands, "summary": got.Summary, "note": got.Note,
		"coverage": coverageMap(got.Coverage), "definite": got.Definite,
	}
	if len(got.Missing) > 0 {
		out["missing"] = got.Missing
	}
	return out
}

func perpDepthMap(got *domain.PerpDepthReport) map[string]any {
	if got == nil {
		return map[string]any{}
	}
	venues := make([]map[string]any, 0, len(got.Venues))
	for _, v := range got.Venues {
		venues = append(venues, perpDepthVenueMap(v))
	}
	out := map[string]any{
		"symbol": got.Symbol, "exchange": got.Exchange, "rangePct": formatQtyMCP(got.RangePct),
		"asOf": got.AsOf.UTC().Format(time.RFC3339Nano), "venues": venues, "summary": got.Summary, "note": got.Note,
	}
	if got.Combined != nil {
		out["combined"] = perpDepthVenueMap(*got.Combined)
	}
	return out
}

func perpDepthVenueMap(v domain.PerpDepthVenue) map[string]any {
	bands := make([]map[string]any, 0, len(v.Bands))
	for _, b := range v.Bands {
		bands = append(bands, perpDepthBandMap(b))
	}
	out := map[string]any{
		"exchange": string(v.Exchange), "symbol": v.Symbol, "mid": formatQtyMCP(v.Mid),
		"coverBidPct": formatQtyMCP(v.CoverBidPct), "coverAskPct": formatQtyMCP(v.CoverAskPct),
		"headline": perpDepthBandMap(v.Headline), "bands": bands, "summary": v.Summary,
	}
	if v.Error != "" {
		out["error"] = v.Error
	}
	return out
}

func perpDepthBandMap(b domain.PerpDepthBand) map[string]any {
	return map[string]any{
		"rangePct": formatQtyMCP(b.RangePct),
		"below": map[string]any{
			"notional": formatQtyMCP(b.Below.Notional), "quantity": formatQtyMCP(b.Below.Quantity),
			"levels": b.Below.Levels, "edgePrice": formatQtyMCP(b.Below.EdgePrice),
		},
		"above": map[string]any{
			"notional": formatQtyMCP(b.Above.Notional), "quantity": formatQtyMCP(b.Above.Quantity),
			"levels": b.Above.Levels, "edgePrice": formatQtyMCP(b.Above.EdgePrice),
		},
		"imbalance": domain.FormatSignedQty(b.Imbalance), "ratio": formatQtyMCP(b.Ratio),
		"lean": b.Lean, "complete": b.Complete,
	}
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

func (b *Backend) GetLiquidationPrintHeatmap(ctx context.Context, exchange, symbol, rawRange string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetLiquidationPrintHeatmap(ctx, exchange, symbol, rawRange)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetLiquidationOpenLevels(ctx context.Context, exchange, symbol, rawRange, anchor string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetLiquidationOpenLevels(ctx, exchange, symbol, rawRange, anchor)
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

func (b *Backend) GetNetPosition(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetNetPosition(ctx, exchange, symbol)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetVolumeShare(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetVolumeShare(ctx, exchange, symbol)
	if err != nil {
		return nil, err
	}
	return mustJSON(volumeShareMap(got))
}

func volumeShareMap(got *domain.VolumeShareReport) map[string]any {
	if got == nil {
		return map[string]any{}
	}
	windows := make([]map[string]any, 0, len(got.Windows))
	for _, w := range got.Windows {
		venues := make([]map[string]any, 0, len(w.Venues))
		for _, v := range w.Venues {
			row := map[string]any{
				"exchange": string(v.Exchange), "volume": v.Volume, "sharePct": v.SharePct,
				"kind": v.Kind, "complete": v.Complete,
			}
			row["priorKnown"] = v.PriorKnown
			if v.PriorKnown {
				row["priorVolume"] = v.PriorVolume
				row["priorSharePct"] = v.PriorSharePct
				row["changePct"] = v.ChangePct
			}
			venues = append(venues, row)
		}
		win := map[string]any{
			"window": w.Window, "total": w.Total, "venues": venues,
			"complete": w.Complete, "summary": w.Summary,
		}
		if w.Prior > 0 {
			win["prior"] = w.Prior
		}
		if len(w.Gained) > 0 {
			win["gained"] = w.Gained
		}
		if len(w.Lost) > 0 {
			win["lost"] = w.Lost
		}
		if len(w.Missing) > 0 {
			win["missing"] = w.Missing
		}
		windows = append(windows, win)
	}
	return map[string]any{
		"symbol": got.Symbol, "exchange": got.Exchange, "asOf": got.AsOf.UTC().Format(time.RFC3339Nano),
		"windows": windows, "note": got.Note,
	}
}

func (b *Backend) GetParticipation(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetParticipation(ctx, exchange, symbol)
	if err != nil {
		return nil, err
	}
	return mustJSON(participationMap(got))
}

func participationMap(got *domain.ParticipationReport) map[string]any {
	if got == nil {
		return map[string]any{}
	}
	venues := make([]map[string]any, 0, len(got.Venues))
	for _, v := range got.Venues {
		venues = append(venues, participationVenueMap(v))
	}
	out := map[string]any{
		"symbol": got.Symbol, "exchange": got.Exchange, "asOf": got.AsOf.UTC().Format(time.RFC3339Nano),
		"venues": venues, "note": got.Note,
	}
	if got.Combined != nil {
		out["combined"] = participationVenueMap(*got.Combined)
	}
	return out
}

func participationVenueMap(v domain.ParticipationVenue) map[string]any {
	wins := make([]map[string]any, 0, len(v.Windows))
	for _, w := range v.Windows {
		row := map[string]any{
			"window": w.Window, "volume": w.Volume, "oiDelta": w.OIDelta,
			"lean": w.Lean, "complete": w.Complete, "summary": w.Summary,
		}
		if w.Volume > 0 {
			row["ratio"] = w.Ratio
			row["ratioPct"] = w.RatioPct
			row["transfer"] = w.Transfer
		}
		wins = append(wins, row)
	}
	out := map[string]any{
		"exchange": string(v.Exchange), "symbol": v.Symbol, "windows": wins, "summary": v.Summary,
	}
	if v.Error != "" {
		out["error"] = v.Error
	}
	return out
}

func (b *Backend) GetOrderCount(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetOrderCount(ctx, exchange, symbol)
	if err != nil {
		return nil, err
	}
	return mustJSON(orderCountMap(got))
}

func (b *Backend) GetOrderCountOverview(ctx context.Context, exchange, quote string, limit int) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetOrderCountOverview(ctx, exchange, quote, limit)
	if err != nil {
		return nil, err
	}
	return mustJSON(orderCountOverviewMap(got))
}

func coverageMap(c domain.MarketCoverage) map[string]any {
	venues := make([]map[string]any, 0, len(c.Venues))
	for _, v := range c.Venues {
		row := map[string]any{"exchange": string(v.Exchange), "status": v.Status, "ageSec": v.AgeSec}
		if !v.AsOf.IsZero() {
			row["asOf"] = v.AsOf.UTC().Format(time.RFC3339Nano)
		}
		venues = append(venues, row)
	}
	missing, stale := c.Missing, c.Stale
	if missing == nil {
		missing = []string{}
	}
	if stale == nil {
		stale = []string{}
	}
	return map[string]any{"complete": c.Complete, "venues": venues, "missing": missing, "stale": stale}
}

func orderCountOverviewMap(got *domain.OrderCountOverview) map[string]any {
	if got == nil {
		return map[string]any{}
	}
	wins := make([]map[string]any, 0, len(got.Windows))
	for _, w := range got.Windows {
		wins = append(wins, map[string]any{
			"window": w.Window, "buyCount": w.BuyCount, "sellCount": w.SellCount,
			"total": w.Total, "delta": w.Delta, "ratio": w.Ratio, "lean": w.Lean,
			"buyCoins": w.BuyCoins, "sellCoins": w.SellCoins, "balancedCoins": w.BalancedCoins,
			"quietCoins": w.QuietCoins, "coins": w.Coins, "complete": w.Complete, "summary": w.Summary,
		})
	}
	coins := make([]map[string]any, 0, len(got.Coins))
	for _, c := range got.Coins {
		coins = append(coins, orderCountVenueMap(c))
	}
	return map[string]any{
		"exchange": got.Exchange, "quote": got.Quote, "universe": got.Universe,
		"asOf": got.AsOf.UTC().Format(time.RFC3339Nano), "windows": wins, "coins": coins,
		"summary": got.Summary, "note": got.Note,
		"coverage": coverageMap(got.Coverage), "definite": got.Definite,
	}
}

func orderCountMap(got *domain.OrderCountReport) map[string]any {
	if got == nil {
		return map[string]any{}
	}
	venues := make([]map[string]any, 0, len(got.Venues))
	for _, v := range got.Venues {
		venues = append(venues, orderCountVenueMap(v))
	}
	out := map[string]any{
		"symbol": got.Symbol, "exchange": got.Exchange, "asOf": got.AsOf.UTC().Format(time.RFC3339Nano),
		"venues": venues, "summary": got.Summary, "note": got.Note,
	}
	if got.Combined != nil {
		out["combined"] = orderCountVenueMap(*got.Combined)
	}
	return out
}

func orderCountVenueMap(v domain.OrderCountVenue) map[string]any {
	wins := make([]map[string]any, 0, len(v.Windows))
	for _, w := range v.Windows {
		wins = append(wins, map[string]any{
			"window": w.Window, "buyCount": w.BuyCount, "sellCount": w.SellCount,
			"total": w.Total, "delta": w.Delta, "ratio": w.Ratio, "lean": w.Lean,
			"complete": w.Complete, "summary": w.Summary,
		})
	}
	out := map[string]any{
		"exchange": string(v.Exchange), "symbol": v.Symbol, "windows": wins, "summary": v.Summary,
	}
	if v.Error != "" {
		out["error"] = v.Error
	}
	return out
}

func (b *Backend) GetBotTracker(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetBotTracker(ctx, exchange, symbol)
	if err != nil {
		return nil, err
	}
	return mustJSON(botTrackerMap(got))
}

func botTrackerMap(got *domain.BotTrackerReport) map[string]any {
	if got == nil {
		return map[string]any{}
	}
	venues := make([]map[string]any, 0, len(got.Venues))
	for _, v := range got.Venues {
		venues = append(venues, botTrackerVenueMap(v))
	}
	return map[string]any{
		"symbol": got.Symbol, "exchange": got.Exchange, "asOf": got.AsOf.UTC().Format(time.RFC3339Nano),
		"window": got.Window, "burst": got.Burst, "minCount": got.MinCount,
		"venues": venues, "clusters": botClustersMap(got.Clusters),
		"buyCount": got.BuyCount, "sellCount": got.SellCount, "delta": got.Delta,
		"summary": got.Summary, "note": got.Note,
	}
}

func botTrackerVenueMap(v domain.BotTrackerVenue) map[string]any {
	out := map[string]any{
		"exchange": string(v.Exchange), "symbol": v.Symbol,
		"clusters": botClustersMap(v.Clusters),
		"buyCount": v.BuyCount, "sellCount": v.SellCount, "delta": v.Delta,
		"summary": v.Summary,
	}
	if v.Error != "" {
		out["error"] = v.Error
	}
	return out
}

func botClustersMap(in []domain.BotCluster) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, c := range in {
		sec := int(c.Span.Round(time.Second) / time.Second)
		if sec < 0 {
			sec = 0
		}
		out = append(out, map[string]any{
			"exchange": string(c.Exchange), "side": c.Side,
			"notional": c.Notional, "count": c.Count, "spanSec": sec,
			"from": c.From.UTC().Format(time.RFC3339Nano), "to": c.To.UTC().Format(time.RFC3339Nano),
			"totalNotional": c.TotalNotional,
		})
	}
	return out
}

func (b *Backend) GetPrintSize(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetPrintSize(ctx, exchange, symbol)
	if err != nil {
		return nil, err
	}
	return mustJSON(printSizeMap(got))
}

func (b *Backend) GetPrintSizeOverview(ctx context.Context, exchange, quote string, limit int) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetPrintSizeOverview(ctx, exchange, quote, limit)
	if err != nil {
		return nil, err
	}
	return mustJSON(printSizeOverviewMap(got))
}

func printSizeOverviewMap(got *domain.PrintSizeOverview) map[string]any {
	if got == nil {
		return map[string]any{}
	}
	wins := make([]map[string]any, 0, len(got.Windows))
	for _, w := range got.Windows {
		wins = append(wins, map[string]any{
			"window": w.Window, "buy": printSizeSideMap(w.Buy), "sell": printSizeSideMap(w.Sell),
			"read": w.Read, "largerCoins": w.LargerCoins, "smallerCoins": w.SmallerCoins,
			"fewLargeCoins": w.FewLargeCoins, "manySmallCoins": w.ManySmallCoins,
			"stableCoins": w.StableCoins, "quietCoins": w.QuietCoins, "coins": w.Coins,
			"complete": w.Complete, "summary": w.Summary,
		})
	}
	coins := make([]map[string]any, 0, len(got.Coins))
	for _, c := range got.Coins {
		coins = append(coins, printSizeVenueMap(c))
	}
	return map[string]any{
		"exchange": got.Exchange, "quote": got.Quote, "universe": got.Universe,
		"asOf": got.AsOf.UTC().Format(time.RFC3339Nano), "windows": wins, "coins": coins,
		"summary": got.Summary, "note": got.Note,
		"coverage": coverageMap(got.Coverage), "definite": got.Definite,
	}
}

func printSizeMap(got *domain.PrintSizeReport) map[string]any {
	if got == nil {
		return map[string]any{}
	}
	venues := make([]map[string]any, 0, len(got.Venues))
	for _, v := range got.Venues {
		venues = append(venues, printSizeVenueMap(v))
	}
	out := map[string]any{
		"symbol": got.Symbol, "exchange": got.Exchange, "asOf": got.AsOf.UTC().Format(time.RFC3339Nano),
		"venues": venues, "note": got.Note,
	}
	if got.Combined != nil {
		out["combined"] = printSizeVenueMap(*got.Combined)
	}
	return out
}

func printSizeVenueMap(v domain.PrintSizeVenue) map[string]any {
	wins := make([]map[string]any, 0, len(v.Windows))
	for _, w := range v.Windows {
		row := map[string]any{
			"window": w.Window, "buy": printSizeSideMap(w.Buy), "sell": printSizeSideMap(w.Sell),
			"complete": w.Complete, "read": w.Read, "summary": w.Summary,
		}
		if !w.From.IsZero() {
			row["from"] = w.From.UTC().Format(time.RFC3339Nano)
		}
		if !w.To.IsZero() {
			row["to"] = w.To.UTC().Format(time.RFC3339Nano)
		}
		wins = append(wins, row)
	}
	out := map[string]any{
		"exchange": string(v.Exchange), "symbol": v.Symbol, "windows": wins, "summary": v.Summary,
	}
	if v.Error != "" {
		out["error"] = v.Error
	}
	return out
}

func printSizeSideMap(s domain.PrintSizeSide) map[string]any {
	return map[string]any{
		"avg": formatQtyMCP(s.Avg), "priorAvg": formatQtyMCP(s.PriorAvg),
		"notional": formatQtyMCP(s.Notional), "priorNotional": formatQtyMCP(s.PriorNotional),
		"count": s.Count, "priorCount": s.PriorCount, "ratio": formatQtyMCP(s.Ratio), "kind": s.Kind,
	}
}

func formatQtyMCP(v float64) string {
	if v == 0 {
		return "0"
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func (b *Backend) GetMarketPressure(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetMarketPressure(ctx, exchange, symbol)
	if err != nil {
		return nil, err
	}
	return mustJSON(marketPressureMap(got))
}

func marketPressureMap(got *domain.MarketPressureReport) map[string]any {
	if got == nil {
		return map[string]any{}
	}
	comps := make([]map[string]any, 0, len(got.Current.Components))
	for _, c := range got.Current.Components {
		comps = append(comps, map[string]any{
			"id": c.ID, "label": c.Label, "value": domain.FormatMarketPressureValue(c.Value),
			"weight": c.Weight, "used": c.Used, "lean": c.Lean, "detail": c.Detail,
		})
	}
	wins := make([]map[string]any, 0, len(got.Windows))
	for _, w := range got.Windows {
		row := map[string]any{
			"window": w.Window, "value": w.ValueText, "pastValue": w.PastText,
			"change": w.ChangeText, "lean": w.Lean, "kind": w.Kind, "complete": w.Complete,
		}
		if !w.SampleTime.IsZero() {
			row["sampleTime"] = w.SampleTime.UTC().Format(time.RFC3339Nano)
		}
		wins = append(wins, row)
	}
	asOf := ""
	if !got.AsOf.IsZero() {
		asOf = got.AsOf.UTC().Format(time.RFC3339Nano)
	}
	return map[string]any{
		"symbol": got.Symbol, "exchange": got.Exchange, "asOf": asOf,
		"current": map[string]any{
			"value": got.Current.ValueText, "lean": got.Current.Lean, "agreement": got.Current.Agreement,
			"components": comps, "headline": got.Current.Headline, "detail": got.Current.Detail,
		},
		"change1h": got.Change1h, "windows": wins, "note": got.Note,
	}
}

func (b *Backend) GetCVD(ctx context.Context, exchange, symbol, rangeID, anchor string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetCVD(ctx, exchange, symbol, rangeID, anchor)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetTapeBacktest(ctx context.Context, exchange, symbol, condition, side string, threshold float64, from, to string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	fromT, err := parseTapeClock(from)
	if err != nil {
		return nil, err
	}
	toT, err := parseTapeClock(to)
	if err != nil {
		return nil, err
	}
	got, err := b.Market.GetTapeBacktest(ctx, exchange, symbol, condition, side, threshold, fromT, toT)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func parseTapeClock(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("%w: time must be RFC3339", domain.ErrInvalidArgument)
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

func (b *Backend) GetSweepRecovery(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetSweepRecovery(ctx, exchange, symbol)
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

func (b *Backend) ReviewSetupRadar(ctx context.Context, exchange, symbol, since string, limit int) (json.RawMessage, error) {
	if b.Radar == nil {
		return nil, fmt.Errorf("%w: radar history not configured", domain.ErrUpstream)
	}
	got, err := b.Radar.Review(ctx, exchange, symbol, since, limit)
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

func (b *Backend) GetAroundSimilar(ctx context.Context, exchange, symbol, lookback, interval, direction string, minReturnPct float64, limit int, window, during, fields, weights, minCoverage, horizons, at string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	var when time.Time
	if at != "" {
		t, err := parseMCPTime(at)
		if err != nil {
			return nil, err
		}
		when = t
	}
	got, err := b.Market.GetAroundSimilar(ctx, exchange, symbol, lookback, interval, direction, minReturnPct, limit, window, during, fields, weights, minCoverage, horizons, when)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetForensics(ctx context.Context, exchange, symbol, at, window, during, lookback, interval, direction string, minReturnPct float64, limit int, fields, weights, minCoverage, horizons, asOf string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	when, err := parseMCPTime(at)
	if err != nil {
		return nil, err
	}
	var asOfTime time.Time
	if strings.TrimSpace(asOf) != "" {
		asOfTime, err = parseMCPTime(asOf)
		if err != nil {
			return nil, err
		}
	}
	got, err := b.Market.GetForensics(ctx, exchange, symbol, window, during, when, lookback, interval, direction, minReturnPct, limit, fields, weights, minCoverage, horizons, asOfTime)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) ScanForensics(ctx context.Context, exchange, quote, lookback, interval, direction string, minReturnPct float64, limit int, window, during, fields, weights, minCoverage, horizons string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.ScanForensics(ctx, exchange, quote, lookback, interval, direction, minReturnPct, limit, window, during, fields, weights, minCoverage, horizons)
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

func (b *Backend) GetFlowAtPrices(ctx context.Context, exchange, symbol, window, startTime, endTime string, tickSize, minPrice, maxPrice float64) (json.RawMessage, error) {
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
	got, err := b.Market.GetFlowAtPrices(ctx, exchange, symbol, window, startPtr, endPtr, tickSize, minPrice, maxPrice)
	if err != nil {
		return nil, err
	}
	return mustJSON(flowAtPricesMap(got))
}

func (b *Backend) GetCandleFootprint(ctx context.Context, exchange, symbol, interval, startTime, endTime string) (json.RawMessage, error) {
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
	got, err := b.Market.GetCandleFootprint(ctx, exchange, symbol, interval, startPtr, endPtr)
	if err != nil {
		return nil, err
	}
	return mustJSON(candleFootprintMap(got))
}

func candleFootprintMap(got *domain.CandleFootprintReport) map[string]any {
	if got == nil {
		return map[string]any{}
	}
	missing := make([]string, 0, len(got.Missing))
	for _, ex := range got.Missing {
		missing = append(missing, string(ex))
	}
	venues := make([]map[string]any, 0, len(got.Venues))
	for _, v := range got.Venues {
		venues = append(venues, candleFootprintBookMap(v))
	}
	out := map[string]any{
		"symbol": got.Symbol, "exchange": got.Exchange, "interval": got.Interval,
		"stepSec": int(got.Step.Seconds()),
		"from":    got.From.UTC(), "to": got.To.UTC(), "asOf": got.AsOf.UTC(),
		"definite": got.Definite, "truncated": got.Truncated, "missing": missing,
		"venues": venues, "summary": got.Summary, "note": got.Note,
	}
	if got.Combined != nil {
		out["combined"] = candleFootprintBookMap(*got.Combined)
	}
	return out
}

func candleFootprintBookMap(v domain.CandleFootprintBook) map[string]any {
	candles := make([]map[string]any, 0, len(v.Candles))
	for _, c := range v.Candles {
		rows := make([]map[string]any, 0, len(c.Rows))
		for _, row := range c.Rows {
			item := map[string]any{
				"price": formatHistQty(row.Price), "high": formatHistQty(row.High),
				"buyVolume": formatHistQty(row.Buy), "sellVolume": formatHistQty(row.Sell),
				"delta":    domain.FormatSignedQty(row.Buy - row.Sell),
				"buyCount": row.BuyN, "sellCount": row.SellN, "isPoc": row.IsPoc,
			}
			if row.Imbalance != "" {
				item["imbalance"] = row.Imbalance
			}
			rows = append(rows, item)
		}
		candle := map[string]any{
			"time": c.Open.UTC(), "buyVolume": formatHistQty(c.Buy), "sellVolume": formatHistQty(c.Sell),
			"delta":    domain.FormatSignedQty(c.Buy - c.Sell),
			"buyCount": c.BuyN, "sellCount": c.SellN, "rows": rows,
		}
		if c.POC > 0 {
			candle["poc"] = formatHistQty(c.POC)
		}
		candles = append(candles, candle)
	}
	return map[string]any{"exchange": string(v.Exchange), "candles": candles}
}

func flowAtPricesMap(got *domain.FlowAtPricesReport) map[string]any {
	if got == nil {
		return map[string]any{}
	}
	venues := make([]map[string]any, 0, len(got.Venues))
	for _, v := range got.Venues {
		venues = append(venues, flowAtPricesVenueMap(v))
	}
	out := map[string]any{
		"symbol": got.Symbol, "exchange": got.Exchange, "window": got.Window,
		"from": got.From.UTC().Format(time.RFC3339Nano), "to": got.To.UTC().Format(time.RFC3339Nano),
		"asOf": got.AsOf.UTC().Format(time.RFC3339Nano), "tickSize": formatQtyMCP(got.TickSize),
		"venues": venues, "summary": got.Summary, "note": got.Note,
	}
	if got.MinPrice > 0 {
		out["minPrice"] = formatQtyMCP(got.MinPrice)
	}
	if got.MaxPrice > 0 {
		out["maxPrice"] = formatQtyMCP(got.MaxPrice)
	}
	if got.Combined != nil {
		out["combined"] = flowAtPricesVenueMap(*got.Combined)
	}
	return out
}

func flowAtPricesVenueMap(v domain.FlowAtPricesVenue) map[string]any {
	levels := make([]map[string]any, 0, len(v.Levels))
	for _, lv := range v.Levels {
		levels = append(levels, map[string]any{
			"price": formatQtyMCP(lv.Price), "high": formatQtyMCP(lv.High),
			"buyVolume": formatQtyMCP(lv.BuyVolume), "sellVolume": formatQtyMCP(lv.SellVolume),
			"volume": formatQtyMCP(lv.Volume), "buyCount": lv.BuyCount, "sellCount": lv.SellCount,
			"trades": lv.Trades, "delta": domain.FormatSignedQty(lv.Delta),
			"buyPct": formatQtyMCP(lv.BuyPct), "sharePct": formatQtyMCP(lv.SharePct),
			"dominant": lv.Dominant, "isPoc": lv.IsPoc,
		})
	}
	out := map[string]any{
		"exchange": string(v.Exchange), "symbol": v.Symbol,
		"from": v.From.UTC().Format(time.RFC3339Nano), "to": v.To.UTC().Format(time.RFC3339Nano),
		"tickSize": formatQtyMCP(v.TickSize), "lastPrice": formatQtyMCP(v.LastPrice),
		"high": formatQtyMCP(v.High), "low": formatQtyMCP(v.Low),
		"buyVolume": formatQtyMCP(v.BuyVolume), "sellVolume": formatQtyMCP(v.SellVolume),
		"volume": formatQtyMCP(v.Volume), "delta": domain.FormatSignedQty(v.Delta),
		"buyCount": v.BuyCount, "sellCount": v.SellCount, "trades": v.Trades, "dominant": v.Dominant,
		"poc": map[string]any{
			"price": formatQtyMCP(v.POC.Price), "high": formatQtyMCP(v.POC.High),
			"buyVolume": formatQtyMCP(v.POC.BuyVolume), "sellVolume": formatQtyMCP(v.POC.SellVolume),
			"volume": formatQtyMCP(v.POC.Volume), "delta": domain.FormatSignedQty(v.POC.Delta),
			"trades": v.POC.Trades, "sharePct": formatQtyMCP(v.POC.SharePct),
		},
		"levels": levels, "complete": v.Complete, "truncated": v.Truncated, "summary": v.Summary,
	}
	if !v.KnownFrom.IsZero() {
		out["knownFrom"] = v.KnownFrom.UTC().Format(time.RFC3339Nano)
	}
	if !v.KnownTo.IsZero() {
		out["knownTo"] = v.KnownTo.UTC().Format(time.RFC3339Nano)
	}
	if v.Error != "" {
		out["error"] = v.Error
	}
	return out
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

func (b *Backend) GetCoinSituation(ctx context.Context, symbol string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetCoinSituation(ctx, symbol)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetFearGreed(ctx context.Context) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetFearGreed(ctx)
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

func (b *Backend) GetFuturesHistory(ctx context.Context, metric, exchange, symbol, from, to, bucket string, limit int) (json.RawMessage, error) {
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
	got, err := b.Market.GetFuturesHistory(ctx, metric, exchange, symbol, fromPtr, toPtr, limit, bucket)
	if err != nil {
		return nil, err
	}
	body := futuresHistoryMap(metric, exchange, symbol, got)
	if step, err := domain.ParseFuturesBucket(bucket); err == nil {
		label, _ := domain.FuturesBucketLabel(step)
		body["bucket"] = label
		body["note"] = domain.FuturesPaneNote(step)
	}
	return mustJSON(body)
}

func (b *Backend) GetPositioningAnchor(ctx context.Context, exchange, symbol, anchor, bucket, from, to string) (json.RawMessage, error) {
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
	got, err := b.Market.GetPositioningAnchor(ctx, exchange, symbol, anchor, bucket, fromPtr, toPtr)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetHistorySpan(ctx context.Context, symbol string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.HistorySpan(ctx, symbol)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) GetUnusual(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.GetUnusual(ctx, exchange, symbol)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
}

func (b *Backend) ScanUnusual(ctx context.Context, exchange, metric string, limit int) (json.RawMessage, error) {
	if b.Market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	got, err := b.Market.ScanUnusual(ctx, exchange, metric, limit)
	if err != nil {
		return nil, err
	}
	return mustJSON(got)
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
				if r.SizeLongShare > 0 {
					item["sizeLongPct"] = formatHistQty(r.SizeLongShare * 100)
				}
				if r.TopAccLongShare > 0 {
					item["topAccLongPct"] = formatHistQty(r.TopAccLongShare * 100)
				}
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
		row := map[string]any{
			"exchange": v.Exchange, "kind": v.Kind, "period": v.Period,
			"current": longShortLevelMap(v.Current), "change": v.Change,
			"history": longShortLevelsMap(v.History),
		}
		if v.TopAccounts != nil {
			row["topAccounts"] = longShortLevelMap(*v.TopAccounts)
		}
		if v.TopPositions != nil {
			row["topPositions"] = longShortLevelMap(*v.TopPositions)
		}
		if v.Split.Alignment != "" && v.Split.Alignment != domain.AlignUnknown {
			row["split"] = longShortSplitMap(v.Split)
		}
		if v.Gap != nil {
			row["gap"] = longShortGapMap(*v.Gap)
		}
		if hist := longShortGapHistoryMap(v.GapHistory); len(hist) > 0 {
			row["gapHistory"] = hist
		}
		if wins := longShortGapWindowsMap(v.GapWindows); len(wins) > 0 {
			row["gapWindows"] = wins
		}
		if moves := longShortGapMovesMap(v.GapMoves); len(moves) > 0 {
			row["gapMoves"] = moves
		}
		if v.Divergence != nil {
			row["divergence"] = longShortDivergenceMap(*v.Divergence)
		}
		venues = append(venues, row)
	}
	asOf := ""
	if !got.AsOf.IsZero() {
		asOf = got.AsOf.UTC().Format(time.RFC3339Nano)
	}
	out := map[string]any{
		"symbol": got.Symbol, "exchange": got.Exchange, "kind": got.Kind, "period": got.Period,
		"venues": venues, "history": longShortLevelsMap(got.History),
		"asOf": asOf, "venueCount": got.VenueCount,
		"note": "Share of accounts that are long vs short (not position size). On Binance, gap is top long% minus crowd long% in percentage points; gapWindows (1h/4h/24h) say if that spread widened, narrowed, or flipped. divergence is when top traders get more long while price falls (or more short while price rises), with since/duration. Combined all does not average venues.",
	}
	if got.Current != nil {
		out["current"] = longShortLevelMap(*got.Current)
	}
	if got.Gap != nil {
		out["gap"] = longShortGapMap(*got.Gap)
	}
	if hist := longShortGapHistoryMap(got.GapHistory); len(hist) > 0 {
		out["gapHistory"] = hist
	}
	if wins := longShortGapWindowsMap(got.GapWindows); len(wins) > 0 {
		out["gapWindows"] = wins
	}
	if moves := longShortGapMovesMap(got.GapMoves); len(moves) > 0 {
		out["gapMoves"] = moves
	}
	if got.Divergence != nil {
		out["divergence"] = longShortDivergenceMap(*got.Divergence)
	}
	return out
}

func longShortSplitMap(s domain.LongShortSplit) map[string]any {
	return map[string]any{
		"alignment": s.Alignment, "important": s.Important,
		"general": s.General, "topAccounts": s.TopAccounts, "topPositions": s.TopPositions,
		"sizeBias": s.SizeBias, "headline": s.Headline, "detail": s.Detail,
	}
}

func longShortGapMap(g domain.LongShortGap) map[string]any {
	out := map[string]any{
		"versus": g.Versus, "crowdLongPct": g.CrowdLongPct, "topLongPct": g.TopLongPct,
		"gapPct": g.GapPct, "lean": g.Lean, "headline": g.Headline, "detail": g.Detail,
	}
	if !g.Time.IsZero() {
		out["time"] = g.Time.UTC().Format(time.RFC3339Nano)
	}
	return out
}

func longShortGapHistoryMap(in []domain.LongShortGapPoint) []map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(in))
	for _, p := range in {
		row := map[string]any{
			"versus": p.Versus, "crowdLongPct": p.CrowdLongPct, "topLongPct": p.TopLongPct,
			"gapPct": p.GapPct, "lean": p.Lean,
		}
		if !p.Time.IsZero() {
			row["time"] = p.Time.UTC().Format(time.RFC3339Nano)
		}
		if p.Price != "" {
			row["price"] = p.Price
		}
		if p.PriceChangePct != "" {
			row["priceChangePct"] = p.PriceChangePct
		}
		if p.TopChangePct != "" {
			row["topChangePct"] = p.TopChangePct
		}
		if p.VsPrice != "" {
			row["vsPrice"] = p.VsPrice
		}
		out = append(out, row)
	}
	return out
}

func longShortDivergenceMap(d domain.LongShortPriceDivergence) map[string]any {
	out := map[string]any{
		"kind": d.Kind, "versus": d.Versus, "duration": d.Duration,
		"durationSeconds": d.DurationSeconds, "priceChangePct": d.PriceChangePct,
		"topChangePct": d.TopChangePct, "gapChangePct": d.GapChangePct,
		"headline": d.Headline, "detail": d.Detail, "bars": d.Bars,
	}
	if !d.Since.IsZero() {
		out["since"] = d.Since.UTC().Format(time.RFC3339Nano)
	}
	if !d.LastAt.IsZero() {
		out["lastAt"] = d.LastAt.UTC().Format(time.RFC3339Nano)
	}
	return out
}

func longShortGapWindowsMap(in []domain.LongShortGapWindow) []map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(in))
	for _, w := range in {
		row := map[string]any{
			"window": w.Window, "pastGapPct": w.PastGapPct, "changePct": w.ChangePct,
			"pastLean": w.PastLean, "lean": w.Lean, "kind": w.Kind, "complete": w.Complete,
		}
		if !w.SampleTime.IsZero() {
			row["sampleTime"] = w.SampleTime.UTC().Format(time.RFC3339Nano)
		}
		out = append(out, row)
	}
	return out
}

func longShortGapMovesMap(in []domain.LongShortGapMove) []map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(in))
	for _, m := range in {
		row := map[string]any{
			"kind": m.Kind, "fromLean": m.FromLean, "toLean": m.ToLean,
			"gapPct": m.GapPct, "changePct": m.ChangePct, "summary": m.Summary,
		}
		if !m.Time.IsZero() {
			row["time"] = m.Time.UTC().Format(time.RFC3339Nano)
		}
		out = append(out, row)
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

func (b *Backend) GetFundingArbSignal(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.FundingArb == nil {
		return nil, fmt.Errorf("%w: funding-arb watches not configured", domain.ErrUpstream)
	}
	got, err := b.FundingArb.GetSignal(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	return mustJSON(fundingArbSignalMap(got))
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
	if s.Expected != nil {
		m["expectedNet"] = s.Expected.ExpectedNet
	}
	if s.Review != nil {
		m["review"] = map[string]any{
			"complete": s.Review.Complete, "stillWorthIt": s.Review.StillWorthIt,
			"expectedNet": s.Review.ExpectedNet, "actualNet": s.Review.ActualNet,
			"deltaNet": s.Review.DeltaNet, "summary": s.Review.Summary,
			"pendingCount": s.Review.PendingCount, "settledCount": s.Review.SettledCount,
		}
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
		"note": "Predicted next funding plus recent settlements. rate is decimal; ratePct is percent. payer=long means longs pay shorts. Per-venue rows stay separate. pressure is OI-weighted combined predicted funding (USDT open interest share) plus pressureHistory / pressureWindows / pressureMoves.",
	}
	if got.Current != nil {
		out["current"] = fundingCurrentMap(*got.Current)
	}
	if got.Pressure != nil {
		out["pressure"] = fundingPressureMap(*got.Pressure)
	}
	if len(got.PressureHistory) > 0 {
		hist := make([]map[string]any, 0, len(got.PressureHistory))
		for _, p := range got.PressureHistory {
			hist = append(hist, fundingPressurePointMap(p))
		}
		out["pressureHistory"] = hist
	}
	if len(got.PressureWindows) > 0 {
		wins := make([]map[string]any, 0, len(got.PressureWindows))
		for _, w := range got.PressureWindows {
			row := map[string]any{
				"window": w.Window, "pastRate": w.PastRate, "pastRatePct": w.PastRatePct,
				"pastPayer": w.PastPayer, "change": w.Change, "changePct": w.ChangePct,
				"payer": w.Payer, "kind": w.Kind, "complete": w.Complete,
			}
			if !w.SampleTime.IsZero() {
				row["sampleTime"] = w.SampleTime.UTC().Format(time.RFC3339Nano)
			}
			wins = append(wins, row)
		}
		out["pressureWindows"] = wins
	}
	if len(got.PressureMoves) > 0 {
		moves := make([]map[string]any, 0, len(got.PressureMoves))
		for _, m := range got.PressureMoves {
			row := map[string]any{
				"kind": m.Kind, "fromPayer": m.FromPayer, "toPayer": m.ToPayer,
				"ratePct": m.RatePct, "changePct": m.ChangePct, "summary": m.Summary,
			}
			if !m.Time.IsZero() {
				row["time"] = m.Time.UTC().Format(time.RFC3339Nano)
			}
			moves = append(moves, row)
		}
		out["pressureMoves"] = moves
	}
	return out
}

func fundingPressureMap(p domain.FundingPressure) map[string]any {
	venues := make([]map[string]any, 0, len(p.Venues))
	for _, v := range p.Venues {
		venues = append(venues, map[string]any{
			"exchange": v.Exchange, "rate": v.Rate, "ratePct": v.RatePct, "payer": v.Payer,
			"openInterestValue": v.OpenInterestValue, "weightPct": v.WeightPct,
		})
	}
	out := map[string]any{
		"rate": p.Rate, "ratePct": p.RatePct, "payer": p.Payer,
		"openInterestValue": p.OpenInterestValue, "venueCount": p.VenueCount,
		"dominantVenue": p.DominantVenue, "venues": venues, "summary": p.Summary,
		"intervalHours": p.IntervalHours,
	}
	if !p.NextFundingTime.IsZero() {
		out["nextFundingTime"] = p.NextFundingTime.UTC().Format(time.RFC3339Nano)
	}
	if !p.Time.IsZero() {
		out["time"] = p.Time.UTC().Format(time.RFC3339Nano)
	}
	return out
}

func fundingPressurePointMap(p domain.FundingPressurePoint) map[string]any {
	out := map[string]any{
		"rate": p.Rate, "ratePct": p.RatePct, "payer": p.Payer,
		"openInterestValue": p.OpenInterestValue, "venueCount": p.VenueCount,
		"dominantVenue": p.DominantVenue,
	}
	if !p.Time.IsZero() {
		out["time"] = p.Time.UTC().Format(time.RFC3339Nano)
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
		"feed":     liquidationFeedJSON(got.Feed),
		"coverage": coverageMap(got.Coverage), "definite": got.Definite,
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
		if ep.StartedBy != "" {
			row["startedBy"] = ep.StartedBy
		}
		if ep.LargestBy != "" {
			row["largestBy"] = ep.LargestBy
		}
		if ep.OrderUnknown {
			row["orderUnknown"] = true
		}
		if len(ep.VenueStarts) > 0 {
			starts := make([]map[string]any, 0, len(ep.VenueStarts))
			for _, v := range ep.VenueStarts {
				s := map[string]any{
					"exchange": v.Exchange, "lagSec": v.LagSec,
					"longNotional": v.LongNotional, "shortNotional": v.ShortNotional,
					"totalNotional": v.TotalNotional, "sharePct": v.SharePct, "count": v.Count,
				}
				if !v.StartedAt.IsZero() {
					s["startedAt"] = v.StartedAt.UTC().Format(time.RFC3339Nano)
				}
				starts = append(starts, s)
			}
			row["venueStarts"] = starts
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
		"windowSeconds": got.WindowSeconds, "sampleEveryMs": got.SampleEveryMs, "source": got.Source,
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

func (b *Backend) CreateSituationAlert(ctx context.Context, clientID, exchange, symbol, mode string) (json.RawMessage, error) {
	if b.Alerts == nil {
		return nil, fmt.Errorf("%w: alerts not configured", domain.ErrUpstream)
	}
	a, err := b.Alerts.Create(ctx, pricealert.CreateInput{
		ClientID: clientID, Exchange: exchange, Symbol: symbol,
		Kind: string(domain.AlertKindSituation), Condition: "leaves", Mode: mode,
	})
	if err != nil {
		return nil, err
	}
	return alertJSON(a)
}

func (b *Backend) CreateTapeAlert(ctx context.Context, clientID, exchange, symbol, kind, condition string, target float64, mode string) (json.RawMessage, error) {
	if b.Alerts == nil {
		return nil, fmt.Errorf("%w: alerts not configured", domain.ErrUpstream)
	}
	a, err := b.Alerts.Create(ctx, pricealert.CreateInput{
		ClientID: clientID, Exchange: exchange, Symbol: symbol, Kind: kind,
		Condition: condition, TargetPrice: target, Mode: mode,
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

func (b *Backend) CreateLiquidationLevelAlert(ctx context.Context, clientID, exchange, symbol, minReliability string, minConfidence float64, mode string) (json.RawMessage, error) {
	if b.Alerts == nil {
		return nil, fmt.Errorf("%w: alerts not configured", domain.ErrUpstream)
	}
	a, err := b.Alerts.Create(ctx, pricealert.CreateInput{
		ClientID:    clientID,
		Exchange:    exchange,
		Symbol:      symbol,
		Kind:        string(domain.AlertKindLiqLevel),
		Condition:   minReliability,
		TargetPrice: minConfidence,
		Mode:        mode,
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

func (b *Backend) CreateSetupRadarAlert(ctx context.Context, clientID, exchange, symbol, lean string, minSimilarity float64, mode string) (json.RawMessage, error) {
	if b.Alerts == nil {
		return nil, fmt.Errorf("%w: alerts not configured", domain.ErrUpstream)
	}
	a, err := b.Alerts.Create(ctx, pricealert.CreateInput{
		ClientID:    clientID,
		Exchange:    exchange,
		Symbol:      symbol,
		Kind:        string(domain.AlertKindSetupRadar),
		Condition:   lean,
		TargetPrice: minSimilarity,
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

func (b *Backend) StressPortfolio(ctx context.Context, clientID, moves string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("portfolio service unavailable")
	}
	parsed := parseStressMoves(moves)
	rep, err := b.Portfolio.Stress(ctx, clientID, PortfolioIDFrom(ctx), "", parsed)
	if err != nil {
		return nil, err
	}
	return mustJSON(stressReportMap(rep))
}

func parseStressMoves(raw string) []domain.StressMove {
	var out []domain.StressMove
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		asset, pct, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		f, err := strconv.ParseFloat(strings.TrimSpace(pct), 64)
		if err != nil {
			continue
		}
		out = append(out, domain.StressMove{Asset: asset, MovePct: f})
	}
	return out
}

func stressReportMap(rep *domain.StressReport) map[string]any {
	if rep == nil {
		return map[string]any{}
	}
	lines := make([]map[string]any, 0, len(rep.Lines))
	for _, ln := range rep.Lines {
		lines = append(lines, map[string]any{
			"kind": ln.Kind, "symbol": ln.Symbol, "side": ln.Side, "asset": ln.Asset,
			"movePct": ln.MovePct, "mark": ln.Mark, "shockedMark": ln.ShockedMark,
			"value": ln.Value, "shockedValue": ln.ShockedValue,
			"unrealized": ln.Unrealized, "shockedUnrealized": ln.ShockedUnrealized,
			"liquidated": ln.Liquidated,
		})
	}
	return map[string]any{
		"equity": rep.Equity, "shockedEquity": rep.ShockedEquity, "equityDelta": rep.EquityDelta,
		"totalPnl": rep.TotalPnL, "shockedTotalPnl": rep.ShockedTotalPnL,
		"unrealizedPnl": rep.UnrealizedPnL, "shockedUnrealizedPnl": rep.ShockedUnrealizedPnL,
		"positionsValue": rep.PositionsValue, "shockedPositionsValue": rep.ShockedPositionsValue,
		"marginEquity": rep.MarginEquity, "shockedMarginEquity": rep.ShockedMarginEquity,
		"liquidated": rep.Liquidated, "lines": lines, "note": rep.Note,
	}
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

func (b *Backend) PutPortfolioRiskLimits(ctx context.Context, clientID string, maxDailyLossPct, maxAssetWeightPct, maxExposurePct *float64) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	v, err := b.Portfolio.SetRiskLimits(ctx, portfolio.RiskLimitsInput{
		PortfolioID: PortfolioIDFrom(ctx),
		ClientID:    clientID, MaxDailyLossPct: maxDailyLossPct, MaxAssetWeightPct: maxAssetWeightPct, MaxExposurePct: maxExposurePct,
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
	lim := map[string]any{
		"maxDailyLossPct": v.Limits.MaxDailyLossPct, "maxAssetWeightPct": v.Limits.MaxAssetWeightPct,
		"maxExposurePct": v.Limits.MaxExposurePct,
	}
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
			"dailyLossLimitHit": v.Status.DailyLossLimitHit,
			"exposureValue":     v.Status.ExposureValue, "exposurePct": v.Status.ExposurePct,
			"exposureLimitHit": v.Status.ExposureLimitHit, "assets": assets,
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
	res, err := b.Portfolio.Transfer(ctx, portfolio.TransferInput{
		ClientID: clientID, FromPortfolioID: fromPortfolioID, ToPortfolioID: toPortfolioID,
		Amount: amount, Note: note, IdempotencyKey: IdempotencyKeyFrom(ctx),
	})
	if err != nil {
		return nil, err
	}
	fromJSON, err := cashMoveJSON(res.Out, res.From)
	if err != nil {
		return nil, err
	}
	toJSON, err := cashMoveJSON(res.In, res.To)
	if err != nil {
		return nil, err
	}
	var fromMap, toMap map[string]any
	_ = json.Unmarshal(fromJSON, &fromMap)
	_ = json.Unmarshal(toJSON, &toMap)
	return mustJSON(map[string]any{
		"from": fromMap, "to": toMap,
		"replayed":       res.Replayed,
		"attempt":        res.Attempt,
		"idempotencyKey": res.IdempotencyKey,
		"note":           "Internal transfer between your paper portfolios. Not a deposit or withdrawal.",
	})
}

func (b *Backend) ListTransferAttempts(ctx context.Context, clientID string, limit, offset int) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	list, total, err := b.Portfolio.ListTransferAttempts(ctx, clientID, limit, offset, PortfolioIDFrom(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(list))
	for i := range list {
		a := list[i]
		items = append(items, map[string]any{
			"id": a.ID, "idempotencyKey": a.IdempotencyKey, "status": a.Status, "attempt": a.Attempt,
			"fromPortfolioId": a.FromPortfolioID, "toPortfolioId": a.ToPortfolioID,
			"fromPortfolioName": a.FromPortfolioName, "toPortfolioName": a.ToPortfolioName,
			"amount": a.Amount, "note": a.Note, "outMovementId": a.OutMovementID, "inMovementId": a.InMovementID,
			"createdAt": a.CreatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	return mustJSON(map[string]any{"clientId": clientID, "attempts": items, "count": len(items), "total": total})
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
	res, err := b.Portfolio.ExecuteAllocationRebalance(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	return mustJSON(rebalanceResultMap(res, true))
}

func (b *Backend) ListPortfolioRebalances(ctx context.Context, clientID, basketID string, limit, offset int) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	list, err := b.Portfolio.ListRebalanceResults(ctx, clientID, basketID, limit, offset, PortfolioIDFrom(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(list))
	for i := range list {
		items = append(items, rebalanceResultMap(&list[i], false))
	}
	return mustJSON(map[string]any{"clientId": clientID, "basketId": basketID, "rebalances": items, "count": len(items)})
}

func (b *Backend) GetPortfolioRebalance(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	res, err := b.Portfolio.GetRebalanceResult(ctx, clientID, id, PortfolioIDFrom(ctx))
	if err != nil {
		return nil, err
	}
	return mustJSON(rebalanceResultMap(res, true))
}

func (b *Backend) ContinuePortfolioRebalance(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	res, err := b.Portfolio.ContinueAllocationRebalance(ctx, clientID, id, PortfolioIDFrom(ctx))
	if err != nil {
		return nil, err
	}
	return mustJSON(rebalanceResultMap(res, true))
}

func rebalanceResultMap(r *domain.RebalanceResult, kids bool) map[string]any {
	m := map[string]any{
		"id": r.ID, "clientId": r.ClientID, "basketId": r.BasketID, "basketName": r.BasketName,
		"currency": r.Currency, "status": r.Status,
		"plannedLegs": r.PlannedLegs, "filledLegs": r.FilledLegs, "failedLegs": r.FailedLegs,
		"beforeEquity": r.BeforeEquity, "afterEquity": r.AfterEquity,
		"beforeCash": r.BeforeCash, "afterCash": r.AfterCash,
		"remainingAbsDriftPct": r.RemainingAbsDriftPct, "maxSleeveDriftPct": r.MaxSleeveDriftPct,
		"remainingNotional": r.RemainingNotional,
		"totalFee":          r.TotalFee, "totalSlippage": r.TotalSlippage, "totalCost": r.TotalCost,
		"summary": r.Summary, "note": r.Note,
		"createdAt":    r.CreatedAt.UTC().Format(time.RFC3339Nano),
		"attemptCount": r.AttemptCount, "canContinue": r.CanContinue,
	}
	if !kids {
		return m
	}
	legs := make([]map[string]any, 0, len(r.Legs))
	for _, l := range r.Legs {
		legs = append(legs, map[string]any{
			"side": string(l.Side), "asset": l.Asset, "exchange": string(l.Exchange), "symbol": l.Symbol,
			"reason": l.Reason, "plannedQuantity": l.PlannedQty, "plannedPrice": l.PlannedPrice,
			"plannedNotional": l.PlannedNotional, "status": l.Status, "tradeId": l.TradeID,
			"filledQuantity": l.FilledQty, "fillPrice": l.FillPrice, "fillNotional": l.FillNotional,
			"lastPrice": l.LastPrice, "fee": l.Fee, "slippage": l.Slippage,
			"error": l.Error,
		})
	}
	lines := make([]map[string]any, 0, len(r.Lines))
	for _, ln := range r.Lines {
		lines = append(lines, map[string]any{
			"asset": ln.Asset, "exchange": string(ln.Exchange), "symbol": ln.Symbol, "isCash": ln.IsCash,
			"targetPct": ln.TargetPct, "beforePct": ln.BeforePct, "afterPct": ln.AfterPct,
			"beforeValue": ln.BeforeValue, "afterValue": ln.AfterValue, "targetValue": ln.TargetValue,
			"afterDelta": ln.AfterDelta,
		})
	}
	m["legs"] = legs
	m["lines"] = lines
	return m
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

func (b *Backend) CompleteRecurringBuyRun(ctx context.Context, clientID, planID, runID string) (json.RawMessage, error) {
	if b.Portfolio == nil {
		return nil, fmt.Errorf("%w: portfolio not configured", domain.ErrUpstream)
	}
	run, tr, err := b.Portfolio.CompleteRecurringBuyRun(ctx, clientID, planID, runID, PortfolioIDFrom(ctx))
	if err != nil {
		return nil, err
	}
	out := map[string]any{"run": recurringRunMap(run)}
	if tr != nil {
		out["tradeId"] = tr.ID
	}
	return mustJSON(out)
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
		"timing":       string(r.Timing),
		"canComplete":  domain.RecurringRunCompletable(*r),
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

func backtestMap(job *domain.ScannerBacktest) map[string]any {
	if job == nil {
		return map[string]any{}
	}
	kind := job.Kind
	if kind == "" {
		if domain.IsCustomProfileBacktest(*job) {
			kind = domain.BacktestKindCustom
		} else {
			kind = domain.BacktestKindRule
		}
	}
	scope := domain.BacktestScopeSymbol
	if domain.IsWatchlistBacktest(*job) {
		scope = domain.BacktestScopeWatchlist
	}
	m := map[string]any{
		"id": job.ID, "clientId": job.ClientID, "ruleId": job.RuleID, "kind": kind,
		"exchange": string(job.Exchange), "symbol": job.Symbol, "scope": scope,
		"interval":   job.Interval,
		"rangeStart": job.RangeStart.UTC().Format(time.RFC3339Nano),
		"rangeEnd":   job.RangeEnd.UTC().Format(time.RFC3339Nano),
		"status":     string(job.Status), "progressPct": job.ProgressPct,
		"processedBars": job.ProcessedBars, "totalBars": job.TotalBars,
		"signalCount": job.SignalCount, "skippedBars": job.SkippedBars,
		"createdAt": job.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if domain.IsCustomProfileBacktest(*job) {
		if name, _, _, err := domain.DecodeCustomProfileSnapshot(job.RuleJSON); err == nil {
			m["sourceName"] = name
		}
	}
	if job.ErrorMessage != "" {
		m["errorMessage"] = job.ErrorMessage
	}
	if len(job.MissingFactors) > 0 {
		m["missingFactors"] = job.MissingFactors
	}
	if job.Performance != nil {
		m["performance"] = job.Performance
	}
	if mk := domain.CustomBacktestMarkets(*job); len(mk) > 0 {
		m["markets"] = mk
	}
	return m
}

func (b *Backend) StartBacktest(ctx context.Context, clientID, ruleID, profileID, exchange, symbol, rangeStart, rangeEnd string, startingBalance, riskPct, maxOpenRiskPct float64, useTP2 bool, tp1ClosePct float64, stopAfterTP1 string, feePct, slipPct float64) (json.RawMessage, error) {
	if b.Scanner == nil {
		return nil, fmt.Errorf("%w: scanner not configured", domain.ErrUpstream)
	}
	start, err := time.Parse(time.RFC3339Nano, rangeStart)
	if err != nil {
		start, err = time.Parse(time.RFC3339, rangeStart)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: rangeStart must be RFC3339", domain.ErrInvalidArgument)
	}
	end, err := time.Parse(time.RFC3339Nano, rangeEnd)
	if err != nil {
		end, err = time.Parse(time.RFC3339, rangeEnd)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: rangeEnd must be RFC3339", domain.ErrInvalidArgument)
	}
	job, err := b.Scanner.StartBacktest(ctx, scanner.StartBacktestInput{
		ClientID: clientID, RuleID: ruleID, ProfileID: profileID,
		Exchange: exchange, Symbol: symbol, StartingBalance: startingBalance, RiskPct: riskPct, MaxOpenRiskPct: maxOpenRiskPct,
		UseTP2: useTP2, TP1ClosePct: tp1ClosePct, StopAfterTP1: stopAfterTP1,
		FeePct: feePct, SlipPct: slipPct,
		RangeStart: start.UTC(), RangeEnd: end.UTC(),
	})
	if err != nil {
		return nil, err
	}
	return mustJSON(backtestMap(job))
}

func (b *Backend) ListBacktests(ctx context.Context, clientID string, limit, offset int) (json.RawMessage, error) {
	if b.Scanner == nil {
		return nil, fmt.Errorf("%w: scanner not configured", domain.ErrUpstream)
	}
	list, total, err := b.Scanner.ListBacktests(ctx, clientID, limit, offset)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(list))
	for i := range list {
		items = append(items, backtestMap(&list[i]))
	}
	return mustJSON(map[string]any{"clientId": clientID, "backtests": items, "count": len(items), "total": total})
}

func (b *Backend) GetBacktest(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.Scanner == nil {
		return nil, fmt.Errorf("%w: scanner not configured", domain.ErrUpstream)
	}
	job, err := b.Scanner.GetBacktest(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	return mustJSON(backtestMap(job))
}

func (b *Backend) CancelBacktest(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.Scanner == nil {
		return nil, fmt.Errorf("%w: scanner not configured", domain.ErrUpstream)
	}
	job, err := b.Scanner.CancelBacktest(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	return mustJSON(backtestMap(job))
}

func (b *Backend) ListBacktestSignals(ctx context.Context, clientID, id string, limit, offset int) (json.RawMessage, error) {
	if b.Scanner == nil {
		return nil, fmt.Errorf("%w: scanner not configured", domain.ErrUpstream)
	}
	list, total, err := b.Scanner.ListBacktestSignals(ctx, clientID, id, limit, offset)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(list))
	for _, s := range list {
		row := map[string]any{
			"id": s.ID, "backtestId": s.BacktestID,
			"exchange": string(s.Exchange), "symbol": s.Symbol,
			"signalAt":   s.SignalAt.UTC().Format(time.RFC3339Nano),
			"closePrice": s.ClosePrice, "summary": s.Summary,
			"return1d": s.Return1d, "return5d": s.Return5d, "return20d": s.Return20d,
			"metrics": s.Metrics, "factors": s.Factors,
		}
		if s.Trade != nil {
			row["trade"] = s.Trade
		}
		items = append(items, row)
	}
	perf, _ := b.Scanner.BacktestPerformance(ctx, clientID, id)
	return mustJSON(map[string]any{"backtestId": id, "signals": items, "count": len(items), "total": total, "performance": perf})
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
		sec := map[string]any{
			"valid": v.Valid, "invalid": v.Invalid, "willAdd": v.WillAdd,
			"willUpdate": v.WillUpdate, "willConflict": v.WillConflict, "duplicates": v.Duplicates,
		}
		if len(v.AddNames) > 0 {
			sec["addNames"] = v.AddNames
		}
		if len(v.UpdateNames) > 0 {
			sec["updateNames"] = v.UpdateNames
		}
		if len(v.Conflicts) > 0 {
			sec["conflicts"] = v.Conflicts
		}
		secs[string(k)] = sec
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
			"willAdd": j.Totals.WillAdd, "willUpdate": j.Totals.WillUpdate,
			"willConflict": j.Totals.WillConflict, "duplicates": j.Totals.Duplicates,
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

func (b *Backend) ConfirmImport(ctx context.Context, clientID, id, mode string, resolutions []domain.CustomSignalResolution) (json.RawMessage, error) {
	if b.Import == nil {
		return nil, fmt.Errorf("%w: import not configured", domain.ErrUpstream)
	}
	job, err := b.Import.Confirm(ctx, clientID, id, mode, resolutions)
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
	rev := domain.BuildPriceDiffOppReview(*o, time.Now().UTC())
	m := map[string]any{
		"id": o.ID, "watchId": o.WatchID, "clientId": o.ClientID, "symbol": o.Symbol,
		"buyExchange": string(o.BuyExchange), "sellExchange": string(o.SellExchange),
		"buyPrice": o.BuyPrice, "sellPrice": o.SellPrice,
		"grossDiffPct": o.GrossDiffPct, "netDiffPct": o.NetDiffPct, "minNetDiffPct": o.MinNetDiffPct,
		"status":         string(o.Status),
		"openedAt":       o.OpenedAt.UTC().Format(time.RFC3339Nano),
		"lastSeenAt":     o.LastSeenAt.UTC().Format(time.RFC3339Nano),
		"openNetDiffPct": o.OpenNetDiffPct,
		"openProfit":     o.OpenProfit,
		"peakNetDiffPct": o.PeakNetDiffPct,
		"peakProfit":     o.PeakProfit,
		"review": map[string]any{
			"openDurationSec": rev.OpenDurationSec,
			"openNetDiffPct":  rev.OpenNetDiffPct,
			"openProfit":      rev.OpenProfit,
			"peakNetDiffPct":  rev.PeakNetDiffPct,
			"peakProfit":      rev.PeakProfit,
			"peakDeltaPct":    rev.PeakDeltaPct,
			"lastNetDiffPct":  rev.LastNetDiffPct,
			"lastProfit":      rev.LastProfit,
			"closeReason":     string(rev.CloseReason),
			"summary":         rev.Summary,
		},
	}
	if o.ClosedAt != nil {
		m["closedAt"] = o.ClosedAt.UTC().Format(time.RFC3339Nano)
	}
	if !o.PeakAt.IsZero() {
		m["peakAt"] = o.PeakAt.UTC().Format(time.RFC3339Nano)
		if rm, ok := m["review"].(map[string]any); ok {
			rm["peakAt"] = o.PeakAt.UTC().Format(time.RFC3339Nano)
		}
	}
	if o.FadeStartedAt != nil {
		m["fadeStartedAt"] = o.FadeStartedAt.UTC().Format(time.RFC3339Nano)
		if rm, ok := m["review"].(map[string]any); ok {
			rm["fadeStartedAt"] = o.FadeStartedAt.UTC().Format(time.RFC3339Nano)
			if rev.FadeFromPeakSec != nil {
				rm["fadeFromPeakSec"] = *rev.FadeFromPeakSec
			}
		}
	}
	if o.CloseReason != "" {
		m["closeReason"] = string(o.CloseReason)
		m["closeNetDiffPct"] = o.CloseNetDiffPct
		m["closeProfit"] = o.CloseProfit
	}
	if len(o.Ticks) > 0 {
		ticks := make([]map[string]any, 0, len(o.Ticks))
		for _, tick := range o.Ticks {
			ticks = append(ticks, map[string]any{
				"at":       tick.At.UTC().Format(time.RFC3339Nano),
				"buyPrice": tick.BuyPrice, "sellPrice": tick.SellPrice,
				"grossDiffPct": tick.GrossDiffPct, "netDiffPct": tick.NetDiffPct,
				"profit": tick.Profit, "executable": tick.Executable,
			})
		}
		m["ticks"] = ticks
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

func parseCustomSignalWeightsJSON(raw string) (map[string]float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("%w: weightsJson is required", domain.ErrInvalidArgument)
	}
	var out map[string]float64
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("%w: weightsJson must be a JSON object of factor id to percent", domain.ErrInvalidArgument)
	}
	return out, nil
}

func customProfileMap(p *domain.CustomSignalProfile) map[string]any {
	if p == nil {
		return map[string]any{}
	}
	pct := p.Weights.Pct
	if pct == nil {
		pct = map[string]float64{}
	}
	return map[string]any{
		"id": p.ID, "clientId": p.ClientID, "name": p.Name, "weights": pct,
		"source": p.Weights.Source, "isDefault": p.IsDefault,
		"createdAt": p.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updatedAt": p.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func (b *Backend) ListCustomSignalFactors(context.Context) (json.RawMessage, error) {
	if b.Swing == nil {
		return nil, fmt.Errorf("%w: swing service not configured", domain.ErrUpstream)
	}
	return mustJSON(map[string]any{
		"factors": b.Swing.ListFactors(),
		"note":    "Built-in percents match the swing engine. Hunt, forensics, and similar setups are off until you give them weight.",
	})
}

func (b *Backend) ListCustomSignalProfiles(ctx context.Context, clientID string) (json.RawMessage, error) {
	if b.Swing == nil {
		return nil, fmt.Errorf("%w: swing service not configured", domain.ErrUpstream)
	}
	list, err := b.Swing.ListProfiles(ctx, clientID)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(list))
	for i := range list {
		items = append(items, customProfileMap(&list[i]))
	}
	return mustJSON(map[string]any{"profiles": items, "count": len(items)})
}

func (b *Backend) CreateCustomSignalProfile(ctx context.Context, clientID, name string, weights map[string]float64, isDefault bool) (json.RawMessage, error) {
	if b.Swing == nil {
		return nil, fmt.Errorf("%w: swing service not configured", domain.ErrUpstream)
	}
	p, err := b.Swing.CreateProfile(ctx, swing.CreateProfileInput{ClientID: clientID, Name: name, Weights: weights, IsDefault: isDefault})
	if err != nil {
		return nil, err
	}
	return mustJSON(customProfileMap(p))
}

func (b *Backend) UpdateCustomSignalProfile(ctx context.Context, clientID, id, name string, weights map[string]float64, isDefault *bool) (json.RawMessage, error) {
	if b.Swing == nil {
		return nil, fmt.Errorf("%w: swing service not configured", domain.ErrUpstream)
	}
	in := swing.UpdateProfileInput{ClientID: clientID, ID: id, HasWeights: weights != nil, IsDefault: isDefault}
	if strings.TrimSpace(name) != "" {
		n := name
		in.Name = &n
	}
	in.Weights = weights
	p, err := b.Swing.UpdateProfile(ctx, in)
	if err != nil {
		return nil, err
	}
	return mustJSON(customProfileMap(p))
}

func (b *Backend) DuplicateCustomSignalProfile(ctx context.Context, clientID, id, name string) (json.RawMessage, error) {
	if b.Swing == nil {
		return nil, fmt.Errorf("%w: swing service not configured", domain.ErrUpstream)
	}
	p, err := b.Swing.DuplicateProfile(ctx, clientID, id, name)
	if err != nil {
		return nil, err
	}
	return mustJSON(customProfileMap(p))
}

func (b *Backend) SetDefaultCustomSignalProfile(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.Swing == nil {
		return nil, fmt.Errorf("%w: swing service not configured", domain.ErrUpstream)
	}
	p, err := b.Swing.SetDefaultProfile(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	return mustJSON(customProfileMap(p))
}

func (b *Backend) DeleteCustomSignalProfile(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	if b.Swing == nil {
		return nil, fmt.Errorf("%w: swing service not configured", domain.ErrUpstream)
	}
	if err := b.Swing.DeleteProfile(ctx, clientID, id); err != nil {
		return nil, err
	}
	return mustJSON(map[string]any{"ok": true})
}

func (b *Backend) ScanCustomSignals(ctx context.Context, clientID, profileID, exchange string, limit int, weights domain.CustomSignalWeights, hasWeights bool) (json.RawMessage, error) {
	if b.Swing == nil {
		return nil, fmt.Errorf("%w: swing service not configured", domain.ErrUpstream)
	}
	wts, resolvedID, err := b.Swing.ResolveCustomWeights(ctx, clientID, profileID, weights, hasWeights)
	if err != nil {
		return nil, err
	}
	res, err := b.Swing.ScanCustom(ctx, swing.CustomScanInput{
		ClientID: clientID, Exchange: exchange, Limit: limit, ProfileID: resolvedID, Weights: wts,
	})
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(res.Items))
	for i := range res.Items {
		it := res.Items[i]
		row := map[string]any{
			"exchange": string(it.Exchange), "symbol": it.Symbol, "interval": it.Interval,
			"price": it.Price, "custom": it.Custom, "scoreCompare": it.Compare,
		}
		if !it.BarTime.IsZero() {
			row["barTime"] = it.BarTime.UTC().Format(time.RFC3339Nano)
		}
		if it.HuntDetail != "" {
			row["huntDetail"] = it.HuntDetail
		}
		if it.ForensicsDetail != "" {
			row["forensicsDetail"] = it.ForensicsDetail
		}
		if it.SimilarDetail != "" {
			row["similarDetail"] = it.SimilarDetail
		}
		if it.TapeSetups > 0 {
			row["tapeSetups"] = it.TapeSetups
			row["tapeUsed"] = it.TapeUsed
			row["tapeThin"] = it.TapeThin
		}
		if it.TapeFresh != nil && it.TapeFresh.Note != "" {
			row["tapeFresh"] = it.TapeFresh
		}
		if it.Engine != nil {
			row["engine"] = swingDecisionMap(it.Engine)
		}
		items = append(items, row)
	}
	return mustJSON(map[string]any{
		"profileId": res.ProfileID, "source": res.Weights.Source, "weights": res.Weights.Pct,
		"items": items, "count": res.Count, "note": res.Note,
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
			"takeProfit1": d.Levels.TakeProfit1, "takeProfit2": d.Levels.TakeProfit2,
			"riskPct": d.Levels.RiskPct, "rewardPct": d.Levels.RewardPct, "rr": d.Levels.RR, "atr": d.Levels.ATR,
			"leverage": d.Levels.Leverage,
		}
	}
	return m
}
