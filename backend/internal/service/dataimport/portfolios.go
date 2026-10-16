package dataimport

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type wirePortfolioBook struct {
	ID               string               `json:"id"`
	Name             string               `json:"name"`
	Currency         string               `json:"currency"`
	StartingBalance  float64              `json:"startingBalance"`
	CashBalance      float64              `json:"cashBalance"`
	RealizedPnLTotal float64              `json:"realizedPnLTotal"`
	NetDeposits      float64              `json:"netDeposits"`
	MarginMode       string               `json:"marginMode"`
	CreatedAt        string               `json:"createdAt"`
	UpdatedAt        string               `json:"updatedAt"`
	Positions        []wirePFPos          `json:"positions"`
	Trades           []wirePFTrade        `json:"trades"`
	OpenOrders       []wirePFOrder        `json:"openOrders"`
	Lots             []wirePFLot          `json:"lots"`
	LotFills         []wirePFLotFill      `json:"lotFills"`
	RecurringBuys    []wirePFRecurring    `json:"recurringBuys"`
	RecurringRuns    []wirePFRecurringRun `json:"recurringRuns"`
	MarginPositions  []wirePFMarginPos    `json:"marginPositions"`
	MarginOrders     []wirePFMarginOrd    `json:"marginOrders"`
	MarginTrades     []wirePFMarginTr     `json:"marginTrades"`
	Shares           []wirePFShare        `json:"shares"`
	Baskets          []wirePFBasket       `json:"baskets"`
	Rebalances       []wirePFRebalance    `json:"rebalances"`
	RiskLimits       *wirePFRisk          `json:"riskLimits"`
}

type wirePFBasket struct {
	ID        string               `json:"id"`
	Name      string               `json:"name"`
	Targets   []wirePFBasketTarget `json:"targets"`
	CreatedAt string               `json:"createdAt"`
	UpdatedAt string               `json:"updatedAt"`
}

type wirePFBasketTarget struct {
	Asset     string  `json:"asset"`
	Exchange  string  `json:"exchange"`
	WeightPct float64 `json:"weightPct"`
}

type wirePFRebalance struct {
	ID                   string                `json:"id"`
	BasketID             string                `json:"basketId"`
	BasketName           string                `json:"basketName"`
	Currency             string                `json:"currency"`
	Status               string                `json:"status"`
	PlannedLegs          int                   `json:"plannedLegs"`
	FilledLegs           int                   `json:"filledLegs"`
	FailedLegs           int                   `json:"failedLegs"`
	BeforeEquity         float64               `json:"beforeEquity"`
	AfterEquity          float64               `json:"afterEquity"`
	BeforeCash           float64               `json:"beforeCash"`
	AfterCash            float64               `json:"afterCash"`
	RemainingAbsDriftPct float64               `json:"remainingAbsDriftPct"`
	MaxSleeveDriftPct    float64               `json:"maxSleeveDriftPct"`
	RemainingNotional    float64               `json:"remainingNotional"`
	TotalFee             float64               `json:"totalFee"`
	TotalSlippage        float64               `json:"totalSlippage"`
	TotalCost            float64               `json:"totalCost"`
	Summary              string                `json:"summary"`
	Note                 string                `json:"note"`
	CreatedAt            string                `json:"createdAt"`
	UpdatedAt            string                `json:"updatedAt"`
	AttemptCount         int                   `json:"attemptCount"`
	Legs                 []wirePFRebalanceLeg  `json:"legs"`
	Lines                []wirePFRebalanceLine `json:"lines"`
	Attempts             []wirePFRebalanceAtt  `json:"attempts"`
}

type wirePFRebalanceLeg struct {
	Side            string  `json:"side"`
	Asset           string  `json:"asset"`
	Exchange        string  `json:"exchange"`
	Symbol          string  `json:"symbol"`
	Reason          string  `json:"reason"`
	PlannedQty      float64 `json:"plannedQty"`
	PlannedPrice    float64 `json:"plannedPrice"`
	PlannedNotional float64 `json:"plannedNotional"`
	Status          string  `json:"status"`
	TradeID         string  `json:"tradeId"`
	FilledQty       float64 `json:"filledQty"`
	FillPrice       float64 `json:"fillPrice"`
	FillNotional    float64 `json:"fillNotional"`
	LastPrice       float64 `json:"lastPrice"`
	Fee             float64 `json:"fee"`
	Slippage        float64 `json:"slippage"`
	Error           string  `json:"error"`
	Attempt         int     `json:"attempt"`
}

type wirePFRebalanceLine struct {
	Asset       string  `json:"asset"`
	Exchange    string  `json:"exchange"`
	Symbol      string  `json:"symbol"`
	IsCash      bool    `json:"isCash"`
	TargetPct   float64 `json:"targetPct"`
	BeforePct   float64 `json:"beforePct"`
	AfterPct    float64 `json:"afterPct"`
	BeforeValue float64 `json:"beforeValue"`
	AfterValue  float64 `json:"afterValue"`
	TargetValue float64 `json:"targetValue"`
	AfterDelta  float64 `json:"afterDelta"`
	Attempt     int     `json:"attempt"`
}

type wirePFRebalanceAtt struct {
	Seq           int                   `json:"seq"`
	Status        string                `json:"status"`
	PlannedLegs   int                   `json:"plannedLegs"`
	FilledLegs    int                   `json:"filledLegs"`
	FailedLegs    int                   `json:"failedLegs"`
	BeforeEquity  float64               `json:"beforeEquity"`
	AfterEquity   float64               `json:"afterEquity"`
	BeforeCash    float64               `json:"beforeCash"`
	AfterCash     float64               `json:"afterCash"`
	TotalFee      float64               `json:"totalFee"`
	TotalSlippage float64               `json:"totalSlippage"`
	TotalCost     float64               `json:"totalCost"`
	Summary       string                `json:"summary"`
	CreatedAt     string                `json:"createdAt"`
	Legs          []wirePFRebalanceLeg  `json:"legs"`
	Lines         []wirePFRebalanceLine `json:"lines"`
}

type wirePFRisk struct {
	MaxDailyLossPct     *float64 `json:"maxDailyLossPct"`
	MaxAssetWeightPct   *float64 `json:"maxAssetWeightPct"`
	MaxExposurePct      *float64 `json:"maxExposurePct"`
	DayKey              string   `json:"dayKey"`
	DayStartEquity      float64  `json:"dayStartEquity"`
	DayStartNetDeposits float64  `json:"dayStartNetDeposits"`
	DayStartFlowKnown   bool     `json:"dayStartFlowKnown"`
	UpdatedAt           string   `json:"updatedAt"`
}

type wirePFPos struct {
	Exchange  string  `json:"exchange"`
	Symbol    string  `json:"symbol"`
	Quantity  float64 `json:"quantity"`
	AvgCost   float64 `json:"avgCost"`
	UpdatedAt string  `json:"updatedAt"`
}

type wirePFTrade struct {
	ID             string                `json:"id"`
	Exchange       string                `json:"exchange"`
	Symbol         string                `json:"symbol"`
	Side           string                `json:"side"`
	Quantity       float64               `json:"quantity"`
	Price          float64               `json:"price"`
	Notional       float64               `json:"notional"`
	RealizedPnL    float64               `json:"realizedPnL"`
	PendingOrderID string                `json:"pendingOrderId"`
	LotMethod      string                `json:"lotMethod"`
	Fee            float64               `json:"fee"`
	LastPrice      float64               `json:"lastPrice"`
	Situation      *domain.CoinSituation `json:"situation,omitempty"`
	CreatedAt      string                `json:"createdAt"`
}

type wirePFOrder struct {
	ID                string  `json:"id"`
	Exchange          string  `json:"exchange"`
	Symbol            string  `json:"symbol"`
	Type              string  `json:"type"`
	Side              string  `json:"side"`
	Quantity          float64 `json:"quantity"`
	FilledQuantity    float64 `json:"filledQuantity"`
	RemainingQuantity float64 `json:"remainingQuantity"`
	TriggerPrice      float64 `json:"triggerPrice"`
	ReservedCash      float64 `json:"reservedCash"`
	ReservedQuantity  float64 `json:"reservedQuantity"`
	TimeInForce       string  `json:"timeInForce"`
	ExpiresAt         *string `json:"expiresAt"`
	Status            string  `json:"status"`
	OCOGroupID        string  `json:"ocoGroupId"`
	OCOPeerID         string  `json:"ocoPeerId"`
	TrailType         string  `json:"trailType"`
	TrailValue        float64 `json:"trailValue"`
	TrailPeak         float64 `json:"trailPeak"`
	BracketID         string  `json:"bracketId"`
	BracketRole       string  `json:"bracketRole"`
	LotMethod         string  `json:"lotMethod"`
	CreatedAt         string  `json:"createdAt"`
	UpdatedAt         string  `json:"updatedAt"`
}

type wirePFLot struct {
	ID               string  `json:"id"`
	Exchange         string  `json:"exchange"`
	Symbol           string  `json:"symbol"`
	Quantity         float64 `json:"quantity"`
	OriginalQuantity float64 `json:"originalQuantity"`
	Price            float64 `json:"price"`
	OpenedAt         string  `json:"openedAt"`
	SourceTradeID    string  `json:"sourceTradeId"`
	ClosedAt         *string `json:"closedAt"`
}

type wirePFLotFill struct {
	ID          string  `json:"id"`
	TradeID     string  `json:"tradeId"`
	LotID       string  `json:"lotId"`
	Quantity    float64 `json:"quantity"`
	CostPrice   float64 `json:"costPrice"`
	SellPrice   float64 `json:"sellPrice"`
	RealizedPnL float64 `json:"realizedPnL"`
}

type wirePFRecurring struct {
	ID            string  `json:"id"`
	Exchange      string  `json:"exchange"`
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	Amount        float64 `json:"amount"`
	Frequency     string  `json:"frequency"`
	Weekday       string  `json:"weekday"`
	DayOfMonth    int     `json:"dayOfMonth"`
	IntervalHours int     `json:"intervalHours"`
	Status        string  `json:"status"`
	NextRunAt     string  `json:"nextRunAt"`
	LastRunAt     *string `json:"lastRunAt"`
	LastPeriodKey string  `json:"lastPeriodKey"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     string  `json:"updatedAt"`
}

type wirePFRecurringRun struct {
	ID           string  `json:"id"`
	PlanID       string  `json:"planId"`
	PeriodKey    string  `json:"periodKey"`
	Status       string  `json:"status"`
	Amount       float64 `json:"amount"`
	Quantity     float64 `json:"quantity"`
	Price        float64 `json:"price"`
	TradeID      string  `json:"tradeId"`
	FailReason   string  `json:"failReason"`
	ScheduledFor string  `json:"scheduledFor"`
	ExecutedAt   string  `json:"executedAt"`
	Timing       string  `json:"timing"`
}

type wirePFMarginPos struct {
	ID               string                `json:"id"`
	Exchange         string                `json:"exchange"`
	Symbol           string                `json:"symbol"`
	Side             string                `json:"side"`
	Mode             string                `json:"mode"`
	Quantity         float64               `json:"quantity"`
	EntryPrice       float64               `json:"entryPrice"`
	Leverage         int                   `json:"leverage"`
	Margin           float64               `json:"margin"`
	DebtPrincipal    float64               `json:"debtPrincipal"`
	DebtInterest     float64               `json:"debtInterest"`
	DebtAsset        string                `json:"debtAsset"`
	LastInterestAt   string                `json:"lastInterestAt"`
	LiquidationPrice float64               `json:"liquidationPrice"`
	StopLoss         *float64              `json:"stopLoss"`
	TakeProfit       *float64              `json:"takeProfit"`
	Status           string                `json:"status"`
	RealizedPnL      float64               `json:"realizedPnL"`
	CloseReason      string                `json:"closeReason"`
	OpenedAt         string                `json:"openedAt"`
	UpdatedAt        string                `json:"updatedAt"`
	ClosedAt         *string               `json:"closedAt"`
	EntrySituation   *domain.CoinSituation `json:"entrySituation,omitempty"`
}

type wirePFMarginOrd struct {
	ID             string                `json:"id"`
	Exchange       string                `json:"exchange"`
	Symbol         string                `json:"symbol"`
	Side           string                `json:"side"`
	Type           string                `json:"type"`
	Quantity       float64               `json:"quantity"`
	Leverage       int                   `json:"leverage"`
	LimitPrice     float64               `json:"limitPrice"`
	ReservedMargin float64               `json:"reservedMargin"`
	StopLoss       *float64              `json:"stopLoss"`
	TakeProfit     *float64              `json:"takeProfit"`
	Status         string                `json:"status"`
	PositionID     string                `json:"positionId"`
	CreatedAt      string                `json:"createdAt"`
	UpdatedAt      string                `json:"updatedAt"`
	Situation      *domain.CoinSituation `json:"situation,omitempty"`
}

type wirePFMarginTr struct {
	ID            string  `json:"id"`
	PositionID    string  `json:"positionId"`
	Exchange      string  `json:"exchange"`
	Symbol        string  `json:"symbol"`
	Side          string  `json:"side"`
	Action        string  `json:"action"`
	Quantity      float64 `json:"quantity"`
	Price         float64 `json:"price"`
	Notional      float64 `json:"notional"`
	RealizedPnL   float64 `json:"realizedPnL"`
	MarginDelta   float64 `json:"marginDelta"`
	PrincipalPaid float64 `json:"principalPaid"`
	InterestPaid  float64 `json:"interestPaid"`
	Leverage      int     `json:"leverage"`
	Fee           float64 `json:"fee"`
	CreatedAt     string  `json:"createdAt"`
}

type wirePFShare struct {
	GranteeClientID string `json:"granteeClientId"`
	Role            string `json:"role"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

func normalizePortfolioBook(w wirePortfolioBook, fileOwner, importer string) (domain.PortfolioSnapshot, error) {
	name, err := domain.ValidatePortfolioName(w.Name)
	if err != nil {
		return domain.PortfolioSnapshot{}, err
	}
	cur := strings.ToUpper(strings.TrimSpace(w.Currency))
	if cur == "" {
		cur = domain.DefaultPaperCurrency
	}
	mode, err := domain.NormalizeMarginMode(w.MarginMode)
	if err != nil {
		mode = domain.MarginModeIsolated
	}
	id := strings.TrimSpace(w.ID)
	snap := domain.PortfolioSnapshot{
		Book: domain.Portfolio{
			ID: id, ClientID: fileOwner, Name: name, Currency: cur,
			StartingBalance: w.StartingBalance, CashBalance: w.CashBalance,
			RealizedPnLTotal: w.RealizedPnLTotal, NetDeposits: w.NetDeposits, MarginMode: mode,
			CreatedAt: parseTimeOrNow(w.CreatedAt), UpdatedAt: parseTimeOrNow(w.UpdatedAt),
		},
	}
	for _, p := range w.Positions {
		ex := domain.ParseExchange(p.Exchange)
		sym := domain.NormalizeSymbol(ex, p.Symbol)
		if sym == "" || p.Quantity <= 0 {
			continue
		}
		snap.Positions = append(snap.Positions, domain.Position{
			ClientID: id, Exchange: ex, Symbol: sym, Quantity: p.Quantity, AvgCost: p.AvgCost, UpdatedAt: parseTimeOrNow(p.UpdatedAt),
		})
	}
	for _, t := range w.Trades {
		if strings.TrimSpace(t.ID) == "" || t.Quantity <= 0 {
			continue
		}
		ex := domain.ParseExchange(t.Exchange)
		side := domain.TradeSide(strings.ToLower(t.Side))
		if !domain.IsValidTradeSide(string(side)) {
			continue
		}
		lm, _ := domain.NormalizeLotMethod(t.LotMethod)
		snap.Trades = append(snap.Trades, domain.Trade{
			ID: t.ID, ClientID: id, Exchange: ex, Symbol: domain.NormalizeSymbol(ex, t.Symbol), Side: side,
			Quantity: t.Quantity, Price: t.Price, Notional: t.Notional, RealizedPnL: t.RealizedPnL,
			PendingOrderID: t.PendingOrderID, LotMethod: lm, Fee: t.Fee, LastPrice: t.LastPrice,
			Situation: t.Situation, CreatedAt: parseTimeOrNow(t.CreatedAt),
		})
	}
	for _, o := range w.OpenOrders {
		if strings.TrimSpace(o.ID) == "" {
			continue
		}
		ex := domain.ParseExchange(o.Exchange)
		ot := domain.PendingOrderType(o.Type)
		if !domain.IsValidPendingOrderType(string(ot)) {
			continue
		}
		st := domain.PendingOrderStatus(o.Status)
		if st == "" {
			st = domain.PendingStatusOpen
		}
		lm, _ := domain.NormalizeLotMethod(o.LotMethod)
		ord := domain.PendingOrder{
			ID: o.ID, ClientID: id, Exchange: ex, Symbol: domain.NormalizeSymbol(ex, o.Symbol),
			Type: ot, Side: domain.TradeSide(o.Side), Quantity: o.Quantity, FilledQuantity: o.FilledQuantity,
			RemainingQuantity: o.RemainingQuantity, TriggerPrice: o.TriggerPrice, ReservedCash: o.ReservedCash,
			ReservedQuantity: o.ReservedQuantity, TimeInForce: domain.TimeInForce(o.TimeInForce), Status: st,
			OCOGroupID: o.OCOGroupID, OCOPeerID: o.OCOPeerID, TrailType: o.TrailType, TrailValue: o.TrailValue,
			TrailPeak: o.TrailPeak, BracketID: o.BracketID, BracketRole: o.BracketRole, LotMethod: lm,
			CreatedAt: parseTimeOrNow(o.CreatedAt), UpdatedAt: parseTimeOrNow(o.UpdatedAt),
		}
		if o.ExpiresAt != nil {
			if tm, err := parseTimeFlexible(*o.ExpiresAt); err == nil {
				ord.ExpiresAt = &tm
			}
		}
		snap.OpenOrders = append(snap.OpenOrders, ord)
	}
	for _, l := range w.Lots {
		if strings.TrimSpace(l.ID) == "" {
			continue
		}
		ex := domain.ParseExchange(l.Exchange)
		lot := domain.TaxLot{
			ID: l.ID, ClientID: id, Exchange: ex, Symbol: domain.NormalizeSymbol(ex, l.Symbol),
			Quantity: l.Quantity, OriginalQuantity: l.OriginalQuantity, Price: l.Price,
			OpenedAt: parseTimeOrNow(l.OpenedAt), SourceTradeID: l.SourceTradeID,
		}
		if l.ClosedAt != nil {
			if tm, err := parseTimeFlexible(*l.ClosedAt); err == nil {
				lot.ClosedAt = &tm
			}
		}
		snap.Lots = append(snap.Lots, lot)
	}
	for _, f := range w.LotFills {
		if strings.TrimSpace(f.ID) == "" {
			continue
		}
		snap.LotFills = append(snap.LotFills, domain.TaxLotFill{
			ID: f.ID, TradeID: f.TradeID, LotID: f.LotID, Quantity: f.Quantity,
			CostPrice: f.CostPrice, SellPrice: f.SellPrice, RealizedPnL: f.RealizedPnL,
		})
	}
	for _, r := range w.RecurringBuys {
		if strings.TrimSpace(r.ID) == "" {
			continue
		}
		ex := domain.ParseExchange(r.Exchange)
		plan := domain.RecurringBuyPlan{
			ID: r.ID, ClientID: id, Exchange: ex, Symbol: domain.NormalizeSymbol(ex, r.Symbol),
			Name: r.Name, Amount: r.Amount, Frequency: domain.RecurringBuyFrequency(r.Frequency),
			Weekday: r.Weekday, DayOfMonth: r.DayOfMonth, IntervalHours: r.IntervalHours,
			Status: domain.RecurringBuyPlanStatus(r.Status), NextRunAt: parseTimeOrNow(r.NextRunAt),
			LastPeriodKey: r.LastPeriodKey, CreatedAt: parseTimeOrNow(r.CreatedAt), UpdatedAt: parseTimeOrNow(r.UpdatedAt),
		}
		if r.LastRunAt != nil {
			if tm, err := parseTimeFlexible(*r.LastRunAt); err == nil {
				plan.LastRunAt = &tm
			}
		}
		snap.RecurringPlans = append(snap.RecurringPlans, plan)
	}
	for _, r := range w.RecurringRuns {
		if strings.TrimSpace(r.ID) == "" {
			continue
		}
		snap.RecurringRuns = append(snap.RecurringRuns, domain.RecurringBuyRun{
			ID: r.ID, PlanID: r.PlanID, ClientID: id, PeriodKey: r.PeriodKey,
			Status: domain.RecurringBuyRunStatus(r.Status), Amount: r.Amount, Quantity: r.Quantity, Price: r.Price,
			TradeID: r.TradeID, FailReason: r.FailReason, ScheduledFor: parseTimeOrNow(r.ScheduledFor),
			ExecutedAt: parseTimeOrNow(r.ExecutedAt), Timing: domain.RecurringBuyTiming(r.Timing),
		})
	}
	for _, m := range w.MarginPositions {
		if strings.TrimSpace(m.ID) == "" {
			continue
		}
		ex := domain.ParseExchange(m.Exchange)
		pos := domain.MarginPosition{
			ID: m.ID, ClientID: id, Exchange: ex, Symbol: domain.NormalizeSymbol(ex, m.Symbol),
			Side: domain.MarginSide(m.Side), Mode: domain.MarginMode(m.Mode), Quantity: m.Quantity,
			EntryPrice: m.EntryPrice, Leverage: m.Leverage, Margin: m.Margin, DebtPrincipal: m.DebtPrincipal,
			DebtInterest: m.DebtInterest, DebtAsset: domain.DebtAsset(m.DebtAsset),
			LastInterestAt: parseTimeOrNow(m.LastInterestAt), LiquidationPrice: m.LiquidationPrice,
			StopLoss: m.StopLoss, TakeProfit: m.TakeProfit, Status: domain.MarginPositionStatus(m.Status),
			RealizedPnL: m.RealizedPnL, CloseReason: m.CloseReason,
			OpenedAt: parseTimeOrNow(m.OpenedAt), UpdatedAt: parseTimeOrNow(m.UpdatedAt),
			EntrySituation: m.EntrySituation,
		}
		if m.ClosedAt != nil {
			if tm, err := parseTimeFlexible(*m.ClosedAt); err == nil {
				pos.ClosedAt = &tm
			}
		}
		snap.MarginPositions = append(snap.MarginPositions, pos)
	}
	for _, o := range w.MarginOrders {
		if strings.TrimSpace(o.ID) == "" {
			continue
		}
		ex := domain.ParseExchange(o.Exchange)
		snap.MarginOrders = append(snap.MarginOrders, domain.MarginOrder{
			ID: o.ID, ClientID: id, Exchange: ex, Symbol: domain.NormalizeSymbol(ex, o.Symbol),
			Side: domain.MarginSide(o.Side), Type: domain.MarginOrderType(o.Type), Quantity: o.Quantity,
			Leverage: o.Leverage, LimitPrice: o.LimitPrice, ReservedMargin: o.ReservedMargin,
			StopLoss: o.StopLoss, TakeProfit: o.TakeProfit, Status: domain.MarginOrderStatus(o.Status),
			PositionID: o.PositionID, CreatedAt: parseTimeOrNow(o.CreatedAt), UpdatedAt: parseTimeOrNow(o.UpdatedAt),
			Situation: o.Situation,
		})
	}
	for _, t := range w.MarginTrades {
		if strings.TrimSpace(t.ID) == "" {
			continue
		}
		ex := domain.ParseExchange(t.Exchange)
		snap.MarginTrades = append(snap.MarginTrades, domain.MarginTrade{
			ID: t.ID, ClientID: id, PositionID: t.PositionID, Exchange: ex, Symbol: domain.NormalizeSymbol(ex, t.Symbol),
			Side: domain.MarginSide(t.Side), Action: t.Action, Quantity: t.Quantity, Price: t.Price, Notional: t.Notional,
			RealizedPnL: t.RealizedPnL, MarginDelta: t.MarginDelta, PrincipalPaid: t.PrincipalPaid,
			InterestPaid: t.InterestPaid, Leverage: t.Leverage, Fee: t.Fee, CreatedAt: parseTimeOrNow(t.CreatedAt),
		})
	}
	for _, sh := range w.Shares {
		role, err := domain.NormalizePortfolioShareRole(sh.Role)
		if err != nil || strings.TrimSpace(sh.GranteeClientID) == "" {
			continue
		}
		snap.Shares = append(snap.Shares, domain.PortfolioShare{
			PortfolioID: id, OwnerClientID: fileOwner, GranteeClientID: strings.TrimSpace(sh.GranteeClientID),
			Role: role, CreatedAt: parseTimeOrNow(sh.CreatedAt), UpdatedAt: parseTimeOrNow(sh.UpdatedAt),
		})
	}
	for _, b := range w.Baskets {
		basket, ok := normalizeImportedBasket(b, id, cur)
		if !ok {
			continue
		}
		if len(snap.Baskets) >= domain.MaxAllocationBasketsPerClient {
			break
		}
		snap.Baskets = append(snap.Baskets, basket)
	}
	for _, r := range w.Rebalances {
		rb, ok := normalizeImportedRebalance(r, id)
		if !ok {
			continue
		}
		snap.Rebalances = append(snap.Rebalances, rb)
	}
	if w.RiskLimits != nil {
		if lim, ok := normalizeImportedRisk(*w.RiskLimits, id); ok {
			snap.RiskLimits = &lim
		}
	}
	mapped := domain.RemapPortfolioSnapshot(snap, fileOwner, importer)
	if err := domain.ValidatePortfolioSnapshot(mapped); err != nil {
		return domain.PortfolioSnapshot{}, err
	}
	return mapped, nil
}

func normalizeImportedBasket(b wirePFBasket, bookID, currency string) (domain.AllocationBasket, bool) {
	id := strings.TrimSpace(b.ID)
	if id == "" {
		return domain.AllocationBasket{}, false
	}
	name, err := domain.NormalizeAllocationName(b.Name)
	if err != nil {
		return domain.AllocationBasket{}, false
	}
	var targets []domain.AllocationTarget
	for _, t := range b.Targets {
		a := domain.NormalizeAllocationAsset(t.Asset)
		if a == "" {
			continue
		}
		ex := domain.Exchange("")
		if strings.TrimSpace(t.Exchange) != "" {
			ex = domain.ParseExchange(t.Exchange)
		}
		targets = append(targets, domain.AllocationTarget{Asset: a, Exchange: ex, WeightPct: t.WeightPct})
	}
	if err := domain.ValidateAllocationTargets(targets, currency); err != nil {
		return domain.AllocationBasket{}, false
	}
	return domain.AllocationBasket{
		ID: id, ClientID: bookID, Name: name, Targets: targets,
		CreatedAt: parseTimeOrNow(b.CreatedAt), UpdatedAt: parseTimeOrNow(b.UpdatedAt),
	}, true
}

func normalizeImportedRebalance(r wirePFRebalance, bookID string) (domain.RebalanceResult, bool) {
	id := strings.TrimSpace(r.ID)
	basketID := strings.TrimSpace(r.BasketID)
	if id == "" || basketID == "" {
		return domain.RebalanceResult{}, false
	}
	status := strings.TrimSpace(r.Status)
	if status == "" {
		status = domain.RebalanceResultCompleted
	}
	out := domain.RebalanceResult{
		ID: id, ClientID: bookID, BasketID: basketID, BasketName: strings.TrimSpace(r.BasketName),
		Currency: strings.ToUpper(strings.TrimSpace(r.Currency)), Status: status,
		PlannedLegs: r.PlannedLegs, FilledLegs: r.FilledLegs, FailedLegs: r.FailedLegs,
		BeforeEquity: r.BeforeEquity, AfterEquity: r.AfterEquity, BeforeCash: r.BeforeCash, AfterCash: r.AfterCash,
		RemainingAbsDriftPct: r.RemainingAbsDriftPct, MaxSleeveDriftPct: r.MaxSleeveDriftPct,
		RemainingNotional: r.RemainingNotional, TotalFee: r.TotalFee, TotalSlippage: r.TotalSlippage,
		TotalCost: r.TotalCost, Summary: r.Summary, Note: r.Note,
		CreatedAt: parseTimeOrNow(r.CreatedAt), UpdatedAt: parseTimeOrNow(r.UpdatedAt),
		AttemptCount: r.AttemptCount,
	}
	if out.TotalCost == 0 && (out.TotalFee != 0 || out.TotalSlippage != 0) {
		out.TotalCost = out.TotalFee + out.TotalSlippage
	}
	for _, leg := range r.Legs {
		out.Legs = append(out.Legs, normalizeImportedRebalanceLeg(leg))
	}
	for _, ln := range r.Lines {
		out.Lines = append(out.Lines, normalizeImportedRebalanceLine(ln))
	}
	for _, att := range r.Attempts {
		a := domain.RebalanceAttempt{
			Seq: att.Seq, Status: att.Status, PlannedLegs: att.PlannedLegs, FilledLegs: att.FilledLegs,
			FailedLegs: att.FailedLegs, BeforeEquity: att.BeforeEquity, AfterEquity: att.AfterEquity,
			BeforeCash: att.BeforeCash, AfterCash: att.AfterCash, TotalFee: att.TotalFee,
			TotalSlippage: att.TotalSlippage, TotalCost: att.TotalCost, Summary: att.Summary,
			CreatedAt: parseTimeOrNow(att.CreatedAt),
		}
		if a.Seq <= 0 {
			a.Seq = 1
		}
		if a.TotalCost == 0 && (a.TotalFee != 0 || a.TotalSlippage != 0) {
			a.TotalCost = a.TotalFee + a.TotalSlippage
		}
		for _, leg := range att.Legs {
			a.Legs = append(a.Legs, normalizeImportedRebalanceLeg(leg))
		}
		for _, ln := range att.Lines {
			a.Lines = append(a.Lines, normalizeImportedRebalanceLine(ln))
		}
		out.Attempts = append(out.Attempts, a)
	}
	if out.AttemptCount <= 0 {
		if n := len(out.Attempts); n > 0 {
			out.AttemptCount = n
		} else {
			out.AttemptCount = 1
		}
	}
	return out, true
}

func normalizeImportedRebalanceLeg(leg wirePFRebalanceLeg) domain.RebalanceLegOutcome {
	side := domain.TradeSide(strings.ToLower(strings.TrimSpace(leg.Side)))
	return domain.RebalanceLegOutcome{
		Side: side, Asset: domain.NormalizeAllocationAsset(leg.Asset),
		Exchange: domain.ParseExchange(leg.Exchange), Symbol: strings.TrimSpace(leg.Symbol),
		Reason: strings.TrimSpace(leg.Reason), PlannedQty: leg.PlannedQty, PlannedPrice: leg.PlannedPrice,
		PlannedNotional: leg.PlannedNotional, Status: strings.TrimSpace(leg.Status), TradeID: strings.TrimSpace(leg.TradeID),
		FilledQty: leg.FilledQty, FillPrice: leg.FillPrice, FillNotional: leg.FillNotional,
		LastPrice: leg.LastPrice, Fee: leg.Fee, Slippage: leg.Slippage, Error: strings.TrimSpace(leg.Error),
		Attempt: leg.Attempt,
	}
}

func normalizeImportedRebalanceLine(ln wirePFRebalanceLine) domain.RebalanceLineSnapshot {
	return domain.RebalanceLineSnapshot{
		Asset: domain.NormalizeAllocationAsset(ln.Asset), Exchange: domain.ParseExchange(ln.Exchange),
		Symbol: strings.TrimSpace(ln.Symbol), IsCash: ln.IsCash, TargetPct: ln.TargetPct,
		BeforePct: ln.BeforePct, AfterPct: ln.AfterPct, BeforeValue: ln.BeforeValue,
		AfterValue: ln.AfterValue, TargetValue: ln.TargetValue, AfterDelta: ln.AfterDelta, Attempt: ln.Attempt,
	}
}

func normalizeImportedRisk(w wirePFRisk, bookID string) (domain.RiskLimits, bool) {
	if err := domain.ValidateOptionalRiskPct(w.MaxDailyLossPct, "maxDailyLossPct"); err != nil {
		return domain.RiskLimits{}, false
	}
	if err := domain.ValidateOptionalRiskPct(w.MaxAssetWeightPct, "maxAssetWeightPct"); err != nil {
		return domain.RiskLimits{}, false
	}
	if err := domain.ValidateOptionalExposurePct(w.MaxExposurePct); err != nil {
		return domain.RiskLimits{}, false
	}
	return domain.RiskLimits{
		ClientID: bookID, MaxDailyLossPct: w.MaxDailyLossPct, MaxAssetWeightPct: w.MaxAssetWeightPct,
		MaxExposurePct: w.MaxExposurePct,
		DayKey:         strings.TrimSpace(w.DayKey), DayStartEquity: w.DayStartEquity,
		DayStartNetDeposits: w.DayStartNetDeposits, DayStartFlowKnown: w.DayStartFlowKnown,
		UpdatedAt: parseTimeOrNow(w.UpdatedAt),
	}, true
}

func parseTimeOrNow(s string) time.Time {
	if t, err := parseTimeFlexible(s); err == nil {
		return t
	}
	return time.Now().UTC()
}

func (s *Service) applyPortfolios(ctx context.Context, clientID string, mode domain.ImportMode, pl *payload) (int, error) {
	if s.data.Portfolio == nil {
		if len(pl.Portfolios) == 0 {
			return 0, nil
		}
		return 0, nil
	}
	replace := mode == domain.ImportModeReplace
	if len(pl.Portfolios) == 0 && !replace {
		return 0, nil
	}
	snaps := make([]domain.PortfolioSnapshot, 0, len(pl.Portfolios))
	for _, snap := range pl.Portfolios {
		snaps = append(snaps, domain.RekeyPortfolioSnapshot(snap, uuid.NewString))
	}
	return s.data.Portfolio.ImportOwnedPortfolios(ctx, clientID, snaps, replace)
}

func csvF64(row map[string]string, key string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(row[key]), 64)
	return v
}
func csvI(row map[string]string, key string) int {
	v, _ := strconv.Atoi(strings.TrimSpace(row[key]))
	return v
}
func csvOptStr(row map[string]string, key string) *string {
	s := strings.TrimSpace(row[key])
	if s == "" {
		return nil
	}
	return &s
}
func situationFromCSV(raw string) *domain.CoinSituation {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var s domain.CoinSituation
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return nil
	}
	return &s
}

func csvOptF64(row map[string]string, key string) *float64 {
	s := strings.TrimSpace(row[key])
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}
