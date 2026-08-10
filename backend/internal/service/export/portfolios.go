package export

import (
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type portfolioBookPayload struct {
	ID               string                     `json:"id"`
	Name             string                     `json:"name"`
	Currency         string                     `json:"currency"`
	StartingBalance  float64                    `json:"startingBalance"`
	CashBalance      float64                    `json:"cashBalance"`
	RealizedPnLTotal float64                    `json:"realizedPnLTotal"`
	NetDeposits      float64                    `json:"netDeposits"`
	MarginMode       string                     `json:"marginMode"`
	CreatedAt        string                     `json:"createdAt"`
	UpdatedAt        string                     `json:"updatedAt"`
	Positions        []portfolioPosPayload      `json:"positions,omitempty"`
	Trades           []portfolioTradePayload    `json:"trades,omitempty"`
	OpenOrders       []portfolioOrderPayload    `json:"openOrders,omitempty"`
	Lots             []portfolioLotPayload      `json:"lots,omitempty"`
	LotFills         []portfolioLotFillPayload  `json:"lotFills,omitempty"`
	RecurringBuys    []portfolioRecurringPayload `json:"recurringBuys,omitempty"`
	RecurringRuns    []portfolioRecurringRunPayload `json:"recurringRuns,omitempty"`
	MarginPositions  []portfolioMarginPosPayload `json:"marginPositions,omitempty"`
	MarginOrders     []portfolioMarginOrdPayload `json:"marginOrders,omitempty"`
	MarginTrades     []portfolioMarginTrPayload `json:"marginTrades,omitempty"`
	Shares           []portfolioSharePayload    `json:"shares,omitempty"`
}

type portfolioPosPayload struct {
	Exchange  string  `json:"exchange"`
	Symbol    string  `json:"symbol"`
	Quantity  float64 `json:"quantity"`
	AvgCost   float64 `json:"avgCost"`
	UpdatedAt string  `json:"updatedAt"`
}

type portfolioTradePayload struct {
	ID             string  `json:"id"`
	Exchange       string  `json:"exchange"`
	Symbol         string  `json:"symbol"`
	Side           string  `json:"side"`
	Quantity       float64 `json:"quantity"`
	Price          float64 `json:"price"`
	Notional       float64 `json:"notional"`
	RealizedPnL    float64 `json:"realizedPnL"`
	PendingOrderID string  `json:"pendingOrderId,omitempty"`
	LotMethod      string  `json:"lotMethod,omitempty"`
	Fee            float64 `json:"fee"`
	LastPrice      float64 `json:"lastPrice,omitempty"`
	CreatedAt      string  `json:"createdAt"`
}

type portfolioOrderPayload struct {
	ID                string   `json:"id"`
	Exchange          string   `json:"exchange"`
	Symbol            string   `json:"symbol"`
	Type              string   `json:"type"`
	Side              string   `json:"side"`
	Quantity          float64  `json:"quantity"`
	FilledQuantity    float64  `json:"filledQuantity"`
	RemainingQuantity float64  `json:"remainingQuantity"`
	TriggerPrice      float64  `json:"triggerPrice"`
	ReservedCash      float64  `json:"reservedCash"`
	ReservedQuantity  float64  `json:"reservedQuantity"`
	TimeInForce       string   `json:"timeInForce"`
	ExpiresAt         *string  `json:"expiresAt,omitempty"`
	Status            string   `json:"status"`
	OCOGroupID        string   `json:"ocoGroupId,omitempty"`
	OCOPeerID         string   `json:"ocoPeerId,omitempty"`
	TrailType         string   `json:"trailType,omitempty"`
	TrailValue        float64  `json:"trailValue,omitempty"`
	TrailPeak         float64  `json:"trailPeak,omitempty"`
	BracketID         string   `json:"bracketId,omitempty"`
	BracketRole       string   `json:"bracketRole,omitempty"`
	LotMethod         string   `json:"lotMethod,omitempty"`
	CreatedAt         string   `json:"createdAt"`
	UpdatedAt         string   `json:"updatedAt"`
}

type portfolioLotPayload struct {
	ID               string  `json:"id"`
	Exchange         string  `json:"exchange"`
	Symbol           string  `json:"symbol"`
	Quantity         float64 `json:"quantity"`
	OriginalQuantity float64 `json:"originalQuantity"`
	Price            float64 `json:"price"`
	OpenedAt         string  `json:"openedAt"`
	SourceTradeID    string  `json:"sourceTradeId,omitempty"`
	ClosedAt         *string `json:"closedAt,omitempty"`
}

type portfolioLotFillPayload struct {
	ID          string  `json:"id"`
	TradeID     string  `json:"tradeId"`
	LotID       string  `json:"lotId"`
	Quantity    float64 `json:"quantity"`
	CostPrice   float64 `json:"costPrice"`
	SellPrice   float64 `json:"sellPrice"`
	RealizedPnL float64 `json:"realizedPnL"`
}

type portfolioRecurringPayload struct {
	ID            string  `json:"id"`
	Exchange      string  `json:"exchange"`
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	Amount        float64 `json:"amount"`
	Frequency     string  `json:"frequency"`
	Weekday       string  `json:"weekday,omitempty"`
	DayOfMonth    int     `json:"dayOfMonth,omitempty"`
	IntervalHours int     `json:"intervalHours,omitempty"`
	Status        string  `json:"status"`
	NextRunAt     string  `json:"nextRunAt"`
	LastRunAt     *string `json:"lastRunAt,omitempty"`
	LastPeriodKey string  `json:"lastPeriodKey,omitempty"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     string  `json:"updatedAt"`
}

type portfolioRecurringRunPayload struct {
	ID           string  `json:"id"`
	PlanID       string  `json:"planId"`
	PeriodKey    string  `json:"periodKey"`
	Status       string  `json:"status"`
	Amount       float64 `json:"amount"`
	Quantity     float64 `json:"quantity"`
	Price        float64 `json:"price"`
	TradeID      string  `json:"tradeId,omitempty"`
	FailReason   string  `json:"failReason,omitempty"`
	ScheduledFor string  `json:"scheduledFor"`
	ExecutedAt   string  `json:"executedAt"`
}

type portfolioMarginPosPayload struct {
	ID               string   `json:"id"`
	Exchange         string   `json:"exchange"`
	Symbol           string   `json:"symbol"`
	Side             string   `json:"side"`
	Mode             string   `json:"mode"`
	Quantity         float64  `json:"quantity"`
	EntryPrice       float64  `json:"entryPrice"`
	Leverage         int      `json:"leverage"`
	Margin           float64  `json:"margin"`
	DebtPrincipal    float64  `json:"debtPrincipal"`
	DebtInterest     float64  `json:"debtInterest"`
	DebtAsset        string   `json:"debtAsset"`
	LastInterestAt   string   `json:"lastInterestAt,omitempty"`
	LiquidationPrice float64  `json:"liquidationPrice"`
	StopLoss         *float64 `json:"stopLoss,omitempty"`
	TakeProfit       *float64 `json:"takeProfit,omitempty"`
	Status           string   `json:"status"`
	RealizedPnL      float64  `json:"realizedPnL"`
	CloseReason      string   `json:"closeReason,omitempty"`
	OpenedAt         string   `json:"openedAt"`
	UpdatedAt        string   `json:"updatedAt"`
	ClosedAt         *string  `json:"closedAt,omitempty"`
}

type portfolioMarginOrdPayload struct {
	ID             string   `json:"id"`
	Exchange       string   `json:"exchange"`
	Symbol         string   `json:"symbol"`
	Side           string   `json:"side"`
	Type           string   `json:"type"`
	Quantity       float64  `json:"quantity"`
	Leverage       int      `json:"leverage"`
	LimitPrice     float64  `json:"limitPrice"`
	ReservedMargin float64  `json:"reservedMargin"`
	StopLoss       *float64 `json:"stopLoss,omitempty"`
	TakeProfit     *float64 `json:"takeProfit,omitempty"`
	Status         string   `json:"status"`
	PositionID     string   `json:"positionId,omitempty"`
	CreatedAt      string   `json:"createdAt"`
	UpdatedAt      string   `json:"updatedAt"`
}

type portfolioMarginTrPayload struct {
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
	PrincipalPaid float64 `json:"principalPaid,omitempty"`
	InterestPaid  float64 `json:"interestPaid,omitempty"`
	Leverage      int     `json:"leverage"`
	Fee           float64 `json:"fee"`
	CreatedAt     string  `json:"createdAt"`
}

type portfolioSharePayload struct {
	GranteeClientID string `json:"granteeClientId"`
	Role            string `json:"role"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

func rfcNano(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func rfcNanoPtr(t *time.Time) *string {
	if t == nil || t.IsZero() {
		return nil
	}
	s := t.UTC().Format(time.RFC3339Nano)
	return &s
}

func snapshotToPayload(snap domain.PortfolioSnapshot) portfolioBookPayload {
	p := portfolioBookPayload{
		ID: snap.Book.ID, Name: snap.Book.Name, Currency: snap.Book.Currency,
		StartingBalance: snap.Book.StartingBalance, CashBalance: snap.Book.CashBalance,
		RealizedPnLTotal: snap.Book.RealizedPnLTotal, NetDeposits: snap.Book.NetDeposits,
		MarginMode: string(snap.Book.MarginMode),
		CreatedAt: rfcNano(snap.Book.CreatedAt), UpdatedAt: rfcNano(snap.Book.UpdatedAt),
	}
	for _, pos := range snap.Positions {
		p.Positions = append(p.Positions, portfolioPosPayload{
			Exchange: string(pos.Exchange), Symbol: pos.Symbol, Quantity: pos.Quantity,
			AvgCost: pos.AvgCost, UpdatedAt: rfcNano(pos.UpdatedAt),
		})
	}
	for _, t := range snap.Trades {
		p.Trades = append(p.Trades, portfolioTradePayload{
			ID: t.ID, Exchange: string(t.Exchange), Symbol: t.Symbol, Side: string(t.Side),
			Quantity: t.Quantity, Price: t.Price, Notional: t.Notional, RealizedPnL: t.RealizedPnL,
			PendingOrderID: t.PendingOrderID, LotMethod: string(t.LotMethod), Fee: t.Fee, LastPrice: t.LastPrice,
			CreatedAt: rfcNano(t.CreatedAt),
		})
	}
	for _, o := range snap.OpenOrders {
		p.OpenOrders = append(p.OpenOrders, portfolioOrderPayload{
			ID: o.ID, Exchange: string(o.Exchange), Symbol: o.Symbol, Type: string(o.Type), Side: string(o.Side),
			Quantity: o.Quantity, FilledQuantity: o.FilledQuantity, RemainingQuantity: o.RemainingQuantity,
			TriggerPrice: o.TriggerPrice, ReservedCash: o.ReservedCash, ReservedQuantity: o.ReservedQuantity,
			TimeInForce: string(o.TimeInForce), ExpiresAt: rfcNanoPtr(o.ExpiresAt), Status: string(o.Status),
			OCOGroupID: o.OCOGroupID, OCOPeerID: o.OCOPeerID, TrailType: o.TrailType, TrailValue: o.TrailValue,
			TrailPeak: o.TrailPeak, BracketID: o.BracketID, BracketRole: o.BracketRole, LotMethod: string(o.LotMethod),
			CreatedAt: rfcNano(o.CreatedAt), UpdatedAt: rfcNano(o.UpdatedAt),
		})
	}
	for _, l := range snap.Lots {
		p.Lots = append(p.Lots, portfolioLotPayload{
			ID: l.ID, Exchange: string(l.Exchange), Symbol: l.Symbol, Quantity: l.Quantity,
			OriginalQuantity: l.OriginalQuantity, Price: l.Price, OpenedAt: rfcNano(l.OpenedAt),
			SourceTradeID: l.SourceTradeID, ClosedAt: rfcNanoPtr(l.ClosedAt),
		})
	}
	for _, f := range snap.LotFills {
		p.LotFills = append(p.LotFills, portfolioLotFillPayload{
			ID: f.ID, TradeID: f.TradeID, LotID: f.LotID, Quantity: f.Quantity,
			CostPrice: f.CostPrice, SellPrice: f.SellPrice, RealizedPnL: f.RealizedPnL,
		})
	}
	for _, r := range snap.RecurringPlans {
		p.RecurringBuys = append(p.RecurringBuys, portfolioRecurringPayload{
			ID: r.ID, Exchange: string(r.Exchange), Symbol: r.Symbol, Name: r.Name, Amount: r.Amount,
			Frequency: string(r.Frequency), Weekday: r.Weekday, DayOfMonth: r.DayOfMonth, IntervalHours: r.IntervalHours,
			Status: string(r.Status), NextRunAt: rfcNano(r.NextRunAt), LastRunAt: rfcNanoPtr(r.LastRunAt),
			LastPeriodKey: r.LastPeriodKey, CreatedAt: rfcNano(r.CreatedAt), UpdatedAt: rfcNano(r.UpdatedAt),
		})
	}
	for _, r := range snap.RecurringRuns {
		p.RecurringRuns = append(p.RecurringRuns, portfolioRecurringRunPayload{
			ID: r.ID, PlanID: r.PlanID, PeriodKey: r.PeriodKey, Status: string(r.Status),
			Amount: r.Amount, Quantity: r.Quantity, Price: r.Price, TradeID: r.TradeID, FailReason: r.FailReason,
			ScheduledFor: rfcNano(r.ScheduledFor), ExecutedAt: rfcNano(r.ExecutedAt),
		})
	}
	for _, m := range snap.MarginPositions {
		p.MarginPositions = append(p.MarginPositions, portfolioMarginPosPayload{
			ID: m.ID, Exchange: string(m.Exchange), Symbol: m.Symbol, Side: string(m.Side), Mode: string(m.Mode),
			Quantity: m.Quantity, EntryPrice: m.EntryPrice, Leverage: m.Leverage, Margin: m.Margin,
			DebtPrincipal: m.DebtPrincipal, DebtInterest: m.DebtInterest, DebtAsset: string(m.DebtAsset),
			LastInterestAt: rfcNano(m.LastInterestAt), LiquidationPrice: m.LiquidationPrice,
			StopLoss: m.StopLoss, TakeProfit: m.TakeProfit, Status: string(m.Status), RealizedPnL: m.RealizedPnL,
			CloseReason: m.CloseReason, OpenedAt: rfcNano(m.OpenedAt), UpdatedAt: rfcNano(m.UpdatedAt),
			ClosedAt: rfcNanoPtr(m.ClosedAt),
		})
	}
	for _, o := range snap.MarginOrders {
		p.MarginOrders = append(p.MarginOrders, portfolioMarginOrdPayload{
			ID: o.ID, Exchange: string(o.Exchange), Symbol: o.Symbol, Side: string(o.Side), Type: string(o.Type),
			Quantity: o.Quantity, Leverage: o.Leverage, LimitPrice: o.LimitPrice, ReservedMargin: o.ReservedMargin,
			StopLoss: o.StopLoss, TakeProfit: o.TakeProfit, Status: string(o.Status), PositionID: o.PositionID,
			CreatedAt: rfcNano(o.CreatedAt), UpdatedAt: rfcNano(o.UpdatedAt),
		})
	}
	for _, t := range snap.MarginTrades {
		p.MarginTrades = append(p.MarginTrades, portfolioMarginTrPayload{
			ID: t.ID, PositionID: t.PositionID, Exchange: string(t.Exchange), Symbol: t.Symbol,
			Side: string(t.Side), Action: t.Action, Quantity: t.Quantity, Price: t.Price, Notional: t.Notional,
			RealizedPnL: t.RealizedPnL, MarginDelta: t.MarginDelta, PrincipalPaid: t.PrincipalPaid,
			InterestPaid: t.InterestPaid, Leverage: t.Leverage, Fee: t.Fee, CreatedAt: rfcNano(t.CreatedAt),
		})
	}
	for _, sh := range snap.Shares {
		p.Shares = append(p.Shares, portfolioSharePayload{
			GranteeClientID: sh.GranteeClientID, Role: string(sh.Role),
			CreatedAt: rfcNano(sh.CreatedAt), UpdatedAt: rfcNano(sh.UpdatedAt),
		})
	}
	return p
}

func f64(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
func i64(v int) string     { return strconv.Itoa(v) }
func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
func derefF64(p *float64) string {
	if p == nil {
		return ""
	}
	return f64(*p)
}
