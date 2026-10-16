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
	if p.Sizing != nil {
		out["sizing"] = previewSizingPayload(*p.Sizing)
	}
	if p.EstimatedRealizedPnL != nil {
		out["estimatedRealizedPnl"] = *p.EstimatedRealizedPnL
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
