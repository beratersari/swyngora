package portfolio

import "gitlab.com/trace-analysis/swyngora/backend/internal/domain"

// OrderPreviewPayload is the public JSON object for a paper order preview.
func OrderPreviewPayload(p *domain.OrderPreview) map[string]any {
	if p == nil {
		return map[string]any{}
	}
	legs := make([]any, 0, len(p.Legs))
	for _, leg := range p.Legs {
		legs = append(legs, previewLegPayload(leg))
	}
	out := map[string]any{
		"product":              p.Product,
		"type":                 p.Type,
		"side":                 p.Side,
		"exchange":             string(p.Exchange),
		"symbol":               p.Symbol,
		"quantity":             p.Quantity,
		"lastPrice":            p.LastPrice,
		"feeRate":              p.FeeRate,
		"slippageRate":         p.SlippageRate,
		"referencePrice":       p.ReferencePrice,
		"fillPrice":            p.FillPrice,
		"notional":             p.Notional,
		"fee":                  p.Fee,
		"cashDebit":            p.CashDebit,
		"cashCredit":           p.CashCredit,
		"reservedCash":         p.ReservedCash,
		"reservedQuantity":     p.ReservedQuantity,
		"availableCash":        p.AvailableCash,
		"availableQuantity":    p.AvailableQuantity,
		"cashAfter":            p.CashAfter,
		"availableAfter":       p.AvailableAfter,
		"insufficientCash":     p.InsufficientCash,
		"insufficientQuantity": p.InsufficientQuantity,
		"borrowLimitHit":       p.BorrowLimitHit,
		"fillsNow":             p.FillsNow,
		"bookSource":           p.BookSource,
		"bookExhausted":        p.BookExhausted,
		"bookVisibleQuantity":  p.BookVisibleQuantity,
		"bookFilledQuantity":   p.BookFilledQuantity,
		"bookLevels":           p.BookLevels,
		"leverage":             p.Leverage,
		"initialMargin":        p.InitialMargin,
		"liquidationPrice":     p.LiquidationPrice,
		"borrowedPrincipal":    p.BorrowedPrincipal,
		"borrowedAsset":        p.BorrowedAsset,
		"risk":                 previewRiskPayload(p.Risk),
		"legs":                 legs,
		"note":                 p.Note,
	}
	if p.MoveStopToEntry {
		out["moveStopToEntry"] = true
		out["moveStopPrice"] = p.MoveStopPrice
	}
	if p.Sizing != nil {
		out["sizing"] = previewSizingPayload(*p.Sizing)
	}
	if p.EstimatedRealizedPnL != nil {
		out["estimatedRealizedPnl"] = *p.EstimatedRealizedPnL
	}
	if p.SituationStatus != "" {
		out["situationStatus"] = p.SituationStatus
	}
	if p.Situation != nil {
		out["situation"] = p.Situation
	}
	if p.ReasonSet {
		reason := p.Reason
		if reason == nil {
			reason = []string{}
		}
		out["reason"] = reason
		if len(p.ReasonMissing) > 0 {
			out["reasonMissing"] = p.ReasonMissing
		}
	}
	if len(p.ReasonHistory) > 0 {
		out["reasonHistory"] = p.ReasonHistory
	}
	if p.Correlation != nil {
		out["correlation"] = p.Correlation
	}
	if p.Routed {
		out["routed"] = true
		legs := make([]any, 0, len(p.RouteLegs))
		for _, leg := range p.RouteLegs {
			legs = append(legs, routeLegPayload(leg))
		}
		out["routeLegs"] = legs
		if len(p.RouteMissing) > 0 {
			out["missingVenues"] = p.RouteMissing
		}
		if len(p.RouteSkipped) > 0 {
			skipped := make([]any, 0, len(p.RouteSkipped))
			for _, row := range p.RouteSkipped {
				skipped = append(skipped, map[string]any{
					"exchange": string(row.Exchange),
					"symbol":   row.Symbol,
					"reason":   row.Reason,
					"detail":   row.Detail,
				})
			}
			out["skippedVenues"] = skipped
		}
		if p.RouteCompare != nil {
			out["compare"] = routeComparePayload(*p.RouteCompare)
		}
		if p.RouteNote != "" {
			out["routeNote"] = p.RouteNote
		}
		if p.MaxSlippagePct > 0 {
			out["maxSlippagePct"] = p.MaxSlippagePct
			out["slippageTouch"] = p.SlippageTouch
			out["slippageLimitPrice"] = p.SlippageLimitPrice
			out["slippageInsideQuantity"] = p.SlippageInsideQuantity
			out["slippageLimited"] = p.SlippageLimited
		}
	}
	if p.Spread {
		out["spread"] = true
		out["spreadMinutes"] = p.SpreadMinutes
		out["sliceCount"] = p.SliceCount
		out["sliceQuantity"] = p.SliceQuantity
		out["sliceIntervalSec"] = p.SliceIntervalSec
	}
	return out
}

func routeLegPayload(leg domain.PaperRouteLeg) map[string]any {
	out := map[string]any{
		"exchange":   string(leg.Exchange),
		"symbol":     leg.Symbol,
		"quote":      leg.Quote,
		"aliased":    leg.Aliased,
		"quantity":   leg.Quantity,
		"fillPrice":  leg.FillPrice,
		"notional":   leg.Notional,
		"fee":        leg.Fee,
		"feeRate":    leg.FeeRate,
		"cashDebit":  leg.CashDebit,
		"cashCredit": leg.CashCredit,
		"levelsUsed": leg.LevelsUsed,
	}
	if leg.EstimatedRealizedPnL != nil {
		out["estimatedRealizedPnl"] = *leg.EstimatedRealizedPnL
	}
	return out
}

func routeComparePayload(c domain.PaperRouteCompare) map[string]any {
	out := map[string]any{
		"exchange":       string(c.Exchange),
		"symbol":         c.Symbol,
		"known":          c.Known,
		"exhausted":      c.Exhausted,
		"inventoryShort": c.InventoryShort,
	}
	if c.Known && !c.Exhausted && !c.InventoryShort {
		out["cash"] = c.Cash
		out["savedCash"] = c.SavedCash
	}
	return out
}

func previewLegPayload(leg domain.OrderPreviewLeg) map[string]any {
	out := map[string]any{
		"role":           leg.Role,
		"side":           string(leg.Side),
		"referencePrice": leg.ReferencePrice,
		"fillPrice":      leg.FillPrice,
		"notional":       leg.Notional,
		"fee":            leg.Fee,
		"cashDebit":      leg.CashDebit,
		"cashCredit":     leg.CashCredit,
	}
	if leg.Quantity > 0 {
		out["quantity"] = leg.Quantity
	}
	if leg.Percent > 0 {
		out["percent"] = leg.Percent
	}
	if leg.OffsetPercent > 0 {
		out["offsetPercent"] = leg.OffsetPercent
	}
	if leg.EstimatedRealizedPnL != nil {
		out["estimatedRealizedPnl"] = *leg.EstimatedRealizedPnL
	}
	return out
}

func previewSizingPayload(z domain.OrderPreviewSizing) map[string]any {
	return map[string]any{
		"riskPercent":           z.RiskPercent,
		"equity":                z.Equity,
		"riskBudget":            z.RiskBudget,
		"quantity":              z.Quantity,
		"entryPrice":            z.EntryPrice,
		"stopPrice":             z.StopPrice,
		"lossPerUnit":           z.LossPerUnit,
		"estimatedLoss":         z.EstimatedLoss,
		"stopAttached":          z.StopAttached,
		"stopBeyondLiquidation": z.StopBeyondLiquidation,
	}
}

func previewRiskPayload(r domain.OrderPreviewRisk) map[string]any {
	reasons := r.Reasons
	if reasons == nil {
		reasons = []string{}
	}
	out := map[string]any{
		"applies":           r.Applies,
		"blocked":           r.Blocked,
		"reasons":           reasons,
		"asset":             r.Asset,
		"assetValue":        r.AssetValue,
		"weightLimitHit":    r.WeightLimitHit,
		"exposureLimitHit":  r.ExposureLimitHit,
		"dailyPnl":          r.DailyPnL,
		"dailyPnlPct":       r.DailyPnLPct,
		"dailyLossLimitHit": r.DailyLossLimitHit,
	}
	if r.ProjectedWeightPct != nil {
		out["projectedWeightPct"] = *r.ProjectedWeightPct
	}
	if r.WeightLimitPct != nil {
		out["weightLimitPct"] = *r.WeightLimitPct
	}
	if r.ProjectedExposurePct != nil {
		out["projectedExposurePct"] = *r.ProjectedExposurePct
	}
	if r.ExposureLimitPct != nil {
		out["exposureLimitPct"] = *r.ExposureLimitPct
	}
	if r.DailyLossLimitPct != nil {
		out["dailyLossLimitPct"] = *r.DailyLossLimitPct
	}
	return out
}
