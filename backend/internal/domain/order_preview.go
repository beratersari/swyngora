package domain

// Order preview is a read of what a paper order would cost and whether a risk
// limit would block it. It does not place the order.

const OrderPreviewNote = "Preview only — nothing was placed. A crypto fill walks the visible book. Equities use a fixed slippage. A size the book cannot hold is not filled. Risk limits block new buys and new margin opens. Sells are not blocked. Not real money."

// OrderPreviewLeg is one fill the ticket is about to rest or send.
// Role is fill, take_profit, or stop_loss.
type OrderPreviewLeg struct {
	Role                 string
	Side                 TradeSide
	ReferencePrice       float64
	FillPrice            float64
	Notional             float64
	Fee                  float64
	CashDebit            float64
	CashCredit           float64
	EstimatedRealizedPnL *float64
}

// OrderPreviewRisk is the risk-limit read for this order.
// Applies is true when the order is a new spot buy or a new margin open.
// Projected weights are informational on a sell (the coin gets smaller).
type OrderPreviewRisk struct {
	Applies              bool
	Blocked              bool
	Reasons              []string
	Asset                string
	AssetValue           float64
	ProjectedWeightPct   *float64
	WeightLimitPct       *float64
	WeightLimitHit       bool
	ProjectedExposurePct *float64
	ExposureLimitPct     *float64
	ExposureLimitHit     bool
	DailyPnL             float64
	DailyPnLPct          float64
	DailyLossLimitHit    bool
	DailyLossLimitPct    *float64
}

// OrderPreview is the cost and risk of one paper order before it is placed.
type OrderPreview struct {
	Product              string
	Type                 string
	Side                 string
	Exchange             Exchange
	Symbol               string
	Quantity             float64
	LastPrice            float64
	FeeRate              float64
	SlippageRate         float64
	ReferencePrice       float64
	FillPrice            float64
	Notional             float64
	Fee                  float64
	CashDebit            float64
	CashCredit           float64
	ReservedCash         float64
	ReservedQuantity     float64
	EstimatedRealizedPnL *float64
	AvailableCash        float64
	AvailableQuantity    float64
	CashAfter            float64
	AvailableAfter       float64
	InsufficientCash     bool
	InsufficientQuantity bool
	BorrowLimitHit       bool
	FillsNow             bool
	// BookSource is "book" when the fill walks the visible spot book, or "slippage" for equities and tests.
	BookSource          string
	BookExhausted       bool
	BookVisibleQuantity float64
	BookFilledQuantity  float64
	BookLevels          int
	Leverage            int
	InitialMargin       float64
	LiquidationPrice    float64
	BorrowedPrincipal   float64
	BorrowedAsset       string
	Risk                OrderPreviewRisk
	Legs                []OrderPreviewLeg
	Sizing              *OrderPreviewSizing
	Note                string
	// Situation is the live coin read for a spot buy, limit buy, or bracket.
	// SituationStatus is ready, blank, failed, or skipped.
	Situation       *CoinSituation
	SituationStatus string
	// ReasonSet reports that the preview named which clauses would be stored.
	ReasonSet     bool
	Reason        []string
	ReasonMissing []string
	// ReasonHistory is this book's past closes for each selected clause at the
	// kind the live read says now. Nil when this preview does not name a reason,
	// or when that close list could not be read. The price is still valid then.
	ReasonHistory []SituationLeaveRow
	// Routed is true when this preview splits a spot market order across venues.
	Routed       bool
	RouteLegs    []PaperRouteLeg
	RouteMissing []string
	RouteSkipped []PaperRouteSkip
	RouteCompare *PaperRouteCompare
	RouteNote    string
}

// QuoteFill is the slipped fill, notional, fee, and cash movement for a quantity.
func QuoteFill(side TradeSide, reference, qty float64, cost TradingCost) (fill, notional, fee, debit, credit float64) {
	fill = ApplySlippage(reference, side, cost.SlippageRate)
	if qty <= 0 || fill <= 0 {
		return fill, 0, 0, 0, 0
	}
	notional = qty * fill
	fee = FeeAmount(qty, fill, cost.FeeRate)
	switch side {
	case TradeSideBuy:
		debit = BuyCashDebit(qty, fill, cost.FeeRate)
	case TradeSideSell:
		credit = SellCashCredit(qty, fill, cost.FeeRate)
	}
	return fill, notional, fee, debit, credit
}

// BuildOrderPreviewRisk reports whether this add would be blocked, and the
// projected coin weight and total exposure. additionalNotional is the notional
// the order guard uses (0 when the order does not add risk).
func BuildOrderPreviewRisk(limits RiskLimits, startEquity, dailyMark, equityNow float64, asset string, assetValue, exposureValue, additionalNotional float64, applies bool) OrderPreviewRisk {
	reasons := RiskLimitMessages(limits, startEquity, dailyMark, equityNow, asset, assetValue, exposureValue, additionalNotional)
	if reasons == nil {
		reasons = []string{}
	}
	out := OrderPreviewRisk{
		Applies:           applies,
		Reasons:           reasons,
		Asset:             asset,
		AssetValue:        assetValue,
		DailyPnL:          dailyMark - startEquity,
		DailyPnLPct:       DailyPnLPct(startEquity, dailyMark),
		DailyLossLimitHit: limits.MaxDailyLossPct != nil && DailyLossLimitHit(startEquity, dailyMark, *limits.MaxDailyLossPct),
		DailyLossLimitPct: limits.MaxDailyLossPct,
	}
	if applies {
		out.Blocked = len(reasons) > 0
	}
	if additionalNotional > 0 && equityNow > PositionEpsilon {
		w := ProjectedAssetWeightPct(equityNow, assetValue, additionalNotional)
		e := ProjectedExposurePct(equityNow, exposureValue, additionalNotional)
		out.ProjectedWeightPct = &w
		out.ProjectedExposurePct = &e
		if limits.MaxAssetWeightPct != nil {
			out.WeightLimitPct = limits.MaxAssetWeightPct
			out.WeightLimitHit = AssetWeightLimitHit(equityNow, assetValue, additionalNotional, *limits.MaxAssetWeightPct)
		}
		if limits.MaxExposurePct != nil {
			out.ExposureLimitPct = limits.MaxExposurePct
			out.ExposureLimitHit = ExposureLimitHit(equityNow, exposureValue, additionalNotional, *limits.MaxExposurePct)
		}
	}
	return out
}
