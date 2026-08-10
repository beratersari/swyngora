package dataimport

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// Normalized payload ready to apply (IDs remapped to importer where needed).
type payload struct {
	WatchlistItems []domain.WatchlistItem     `json:"watchlistItems"`
	Shares         []domain.WatchlistShare    `json:"shares"`
	Alerts         []domain.PriceAlert        `json:"alerts"`
	Backtests      []backtestBundle           `json:"backtests"`
	Portfolios     []domain.PortfolioSnapshot `json:"portfolios"`
	FileOwnerID    string                     `json:"fileOwnerId,omitempty"`
}

type backtestBundle struct {
	Job     domain.ScannerBacktest         `json:"job"`
	Signals []domain.ScannerBacktestSignal `json:"signals"`
}

// wire shapes match export JSON.
type wirePayload struct {
	Watchlist *struct {
		Items []struct {
			Exchange string `json:"exchange"`
			Symbol   string `json:"symbol"`
			Note     string `json:"note"`
			AddedAt  string `json:"addedAt"`
		} `json:"items"`
	} `json:"watchlist"`
	Shares []struct {
		OwnerClientID   string `json:"ownerClientId"`
		GranteeClientID string `json:"granteeClientId"`
		Role            string `json:"role"`
		CreatedAt       string `json:"createdAt"`
		UpdatedAt       string `json:"updatedAt"`
	} `json:"shares"`
	Alerts []struct {
		ID             string  `json:"id"`
		Exchange       string  `json:"exchange"`
		Symbol         string  `json:"symbol"`
		Kind           string  `json:"kind"`
		Condition      string  `json:"condition"`
		TargetPrice    float64 `json:"targetPrice"`
		RangePct       float64 `json:"rangePct"`
		Mode           string  `json:"mode"`
		Status         string  `json:"status"`
		CreatedAt      string  `json:"createdAt"`
		TriggeredAt    *string `json:"triggeredAt"`
		TriggeredPrice float64 `json:"triggeredPrice"`
	} `json:"alerts"`
	ClientID   string              `json:"clientId"`
	Portfolios []wirePortfolioBook `json:"portfolios"`
	Backtests  []struct {
		ID           string `json:"id"`
		RuleID       string `json:"ruleId"`
		Exchange     string `json:"exchange"`
		Symbol       string `json:"symbol"`
		Interval     string `json:"interval"`
		RangeStart   string `json:"rangeStart"`
		RangeEnd     string `json:"rangeEnd"`
		Status       string `json:"status"`
		SignalCount  int    `json:"signalCount"`
		CreatedAt    string `json:"createdAt"`
		ErrorMessage string `json:"errorMessage"`
		Signals      []struct {
			ID         string             `json:"id"`
			SignalAt   string             `json:"signalAt"`
			ClosePrice float64            `json:"closePrice"`
			Summary    string             `json:"summary"`
			Return1d   *float64           `json:"return1d"`
			Return5d   *float64           `json:"return5d"`
			Return20d  *float64           `json:"return20d"`
			Metrics    map[string]float64 `json:"metrics"`
		} `json:"signals"`
	} `json:"backtests"`
}

type parseResult struct {
	Payload payload
	// Invalid counts per section (rows that failed validation).
	Invalid map[domain.ExportSection]int
	// Valid items (after de-dupe within file).
	// FileDuplicates counts dropped as duplicate within the file.
	FileDuplicates map[domain.ExportSection]int
}

func detectFormat(filename string, data []byte) domain.ExportFormat {
	name := strings.ToLower(filename)
	if strings.HasSuffix(name, ".csv") {
		return domain.ExportFormatCSV
	}
	if strings.HasSuffix(name, ".json") {
		return domain.ExportFormatJSON
	}
	trim := bytes.TrimSpace(data)
	if len(trim) > 0 && trim[0] == '{' {
		return domain.ExportFormatJSON
	}
	return domain.ExportFormatCSV
}

func parseExportFile(format domain.ExportFormat, data []byte, importerClientID string) (*parseResult, error) {
	switch format {
	case domain.ExportFormatJSON:
		return parseJSON(data, importerClientID)
	case domain.ExportFormatCSV:
		return parseCSV(data, importerClientID)
	default:
		return nil, fmt.Errorf("%w: unsupported format", domain.ErrInvalidArgument)
	}
}

func parseJSON(data []byte, importer string) (*parseResult, error) {
	var w wirePayload
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, fmt.Errorf("%w: invalid JSON export: %v", domain.ErrInvalidArgument, err)
	}
	res := &parseResult{
		Invalid:        map[domain.ExportSection]int{},
		FileDuplicates: map[domain.ExportSection]int{},
	}
	// Watchlist
	seenWL := map[string]struct{}{}
	if w.Watchlist != nil {
		for _, it := range w.Watchlist.Items {
			item, err := normalizeWLItem(it.Exchange, it.Symbol, it.Note, it.AddedAt)
			if err != nil {
				res.Invalid[domain.ExportSectionWatchlist]++
				continue
			}
			key := string(item.Exchange) + "|" + item.Symbol
			if _, ok := seenWL[key]; ok {
				res.FileDuplicates[domain.ExportSectionWatchlist]++
				continue
			}
			seenWL[key] = struct{}{}
			res.Payload.WatchlistItems = append(res.Payload.WatchlistItems, item)
		}
	}
	// Shares — owner forced to importer
	seenSh := map[string]struct{}{}
	for _, sh := range w.Shares {
		share, err := normalizeShare(importer, sh.GranteeClientID, sh.Role, sh.CreatedAt, sh.UpdatedAt)
		if err != nil {
			res.Invalid[domain.ExportSectionShares]++
			continue
		}
		key := share.GranteeClientID
		if _, ok := seenSh[key]; ok {
			res.FileDuplicates[domain.ExportSectionShares]++
			continue
		}
		seenSh[key] = struct{}{}
		res.Payload.Shares = append(res.Payload.Shares, share)
	}
	// Alerts
	seenAl := map[string]struct{}{}
	for _, a := range w.Alerts {
		al, err := normalizeAlert(importer, a.ID, a.Exchange, a.Symbol, a.Kind, a.Condition, a.TargetPrice, a.RangePct, a.Mode, a.Status, a.CreatedAt, a.TriggeredAt, a.TriggeredPrice)
		if err != nil {
			res.Invalid[domain.ExportSectionAlerts]++
			continue
		}
		if _, ok := seenAl[al.ID]; ok {
			res.FileDuplicates[domain.ExportSectionAlerts]++
			continue
		}
		seenAl[al.ID] = struct{}{}
		res.Payload.Alerts = append(res.Payload.Alerts, al)
	}
	// Backtests
	seenBT := map[string]struct{}{}
	for _, bt := range w.Backtests {
		wsigs := make([]wireSignal, 0, len(bt.Signals))
		for _, s := range bt.Signals {
			wsigs = append(wsigs, wireSignal{
				ID: s.ID, SignalAt: s.SignalAt, ClosePrice: s.ClosePrice, Summary: s.Summary,
				Return1d: s.Return1d, Return5d: s.Return5d, Return20d: s.Return20d, Metrics: s.Metrics,
			})
		}
		bundle, err := normalizeBacktest(importer, bt.ID, bt.RuleID, bt.Exchange, bt.Symbol, bt.Interval,
			bt.RangeStart, bt.RangeEnd, bt.Status, bt.SignalCount, bt.CreatedAt, bt.ErrorMessage, wsigs)
		if err != nil {
			res.Invalid[domain.ExportSectionBacktests]++
			continue
		}
		if _, ok := seenBT[bundle.Job.ID]; ok {
			res.FileDuplicates[domain.ExportSectionBacktests]++
			continue
		}
		seenBT[bundle.Job.ID] = struct{}{}
		res.Payload.Backtests = append(res.Payload.Backtests, bundle)
	}
	fileOwner := strings.TrimSpace(w.ClientID)
	res.Payload.FileOwnerID = fileOwner
	seenPF := map[string]struct{}{}
	for _, pb := range w.Portfolios {
		snap, err := normalizePortfolioBook(pb, fileOwner, importer)
		if err != nil {
			res.Invalid[domain.ExportSectionPortfolios]++
			continue
		}
		key := snap.Book.ID + "|" + strings.ToLower(snap.Book.Name)
		if _, ok := seenPF[snap.Book.ID]; ok {
			res.FileDuplicates[domain.ExportSectionPortfolios]++
			continue
		}
		if _, ok := seenPF[key]; ok {
			res.FileDuplicates[domain.ExportSectionPortfolios]++
			continue
		}
		seenPF[snap.Book.ID] = struct{}{}
		seenPF[key] = struct{}{}
		res.Payload.Portfolios = append(res.Payload.Portfolios, snap)
	}
	return res, nil
}

type wireSignal struct {
	ID         string
	SignalAt   string
	ClosePrice float64
	Summary    string
	Return1d   *float64
	Return5d   *float64
	Return20d  *float64
	Metrics    map[string]float64
}

func parseCSV(data []byte, importer string) (*parseResult, error) {
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	res := &parseResult{
		Invalid:        map[domain.ExportSection]int{},
		FileDuplicates: map[domain.ExportSection]int{},
	}
	var section string
	var header []string
	seenWL := map[string]struct{}{}
	seenSh := map[string]struct{}{}
	seenAl := map[string]struct{}{}
	seenBT := map[string]struct{}{}
	// signals attached after all backtests by id
	signalsByBT := map[string][]domain.ScannerBacktestSignal{}
	backtestsByID := map[string]*backtestBundle{}
	csvBooks := map[string]*wirePortfolioBook{}
	fileOwner := ""

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: invalid CSV: %v", domain.ErrInvalidArgument, err)
		}
		if len(rec) == 0 {
			continue
		}
		// meta line
		if len(rec) > 0 && strings.HasPrefix(rec[0], "# meta") {
			for i, cell := range rec {
				if cell == "clientId" && i+1 < len(rec) {
					fileOwner = strings.TrimSpace(rec[i+1])
				}
			}
			continue
		}
		// section marker
		if len(rec) > 0 && strings.HasPrefix(rec[0], "# section:") {
			section = strings.TrimPrefix(rec[0], "# section:")
			header = nil
			continue
		}
		if section == "" {
			continue
		}
		if header == nil {
			header = rec
			continue
		}
		row := map[string]string{}
		for i, h := range header {
			if i < len(rec) {
				row[h] = rec[i]
			}
		}
		switch section {
		case "watchlist":
			item, err := normalizeWLItem(row["exchange"], row["symbol"], row["note"], row["addedAt"])
			if err != nil {
				res.Invalid[domain.ExportSectionWatchlist]++
				continue
			}
			key := string(item.Exchange) + "|" + item.Symbol
			if _, ok := seenWL[key]; ok {
				res.FileDuplicates[domain.ExportSectionWatchlist]++
				continue
			}
			seenWL[key] = struct{}{}
			res.Payload.WatchlistItems = append(res.Payload.WatchlistItems, item)
		case "shares":
			share, err := normalizeShare(importer, row["granteeClientId"], row["role"], row["createdAt"], row["updatedAt"])
			if err != nil {
				res.Invalid[domain.ExportSectionShares]++
				continue
			}
			if _, ok := seenSh[share.GranteeClientID]; ok {
				res.FileDuplicates[domain.ExportSectionShares]++
				continue
			}
			seenSh[share.GranteeClientID] = struct{}{}
			res.Payload.Shares = append(res.Payload.Shares, share)
		case "alerts":
			tp, _ := strconv.ParseFloat(row["targetPrice"], 64)
			trigP, _ := strconv.ParseFloat(row["triggeredPrice"], 64)
			var trigAt *string
			if t := strings.TrimSpace(row["triggeredAt"]); t != "" {
				trigAt = &t
			}
			rng, _ := strconv.ParseFloat(row["rangePct"], 64)
			al, err := normalizeAlert(importer, row["id"], row["exchange"], row["symbol"], row["kind"], row["condition"],
				tp, rng, row["mode"], row["status"], row["createdAt"], trigAt, trigP)
			if err != nil {
				res.Invalid[domain.ExportSectionAlerts]++
				continue
			}
			if _, ok := seenAl[al.ID]; ok {
				res.FileDuplicates[domain.ExportSectionAlerts]++
				continue
			}
			seenAl[al.ID] = struct{}{}
			res.Payload.Alerts = append(res.Payload.Alerts, al)
		case "backtests":
			sc, _ := strconv.Atoi(row["signalCount"])
			bundle, err := normalizeBacktest(importer, row["id"], row["ruleId"], row["exchange"], row["symbol"], row["interval"],
				row["rangeStart"], row["rangeEnd"], row["status"], sc, row["createdAt"], row["errorMessage"], nil /* signals later */)
			if err != nil {
				res.Invalid[domain.ExportSectionBacktests]++
				continue
			}
			if _, ok := seenBT[bundle.Job.ID]; ok {
				res.FileDuplicates[domain.ExportSectionBacktests]++
				continue
			}
			seenBT[bundle.Job.ID] = struct{}{}
			cp := bundle
			backtestsByID[bundle.Job.ID] = &cp
		case "backtest_signals":
			btID := row["backtestId"]
			sigAt, err := parseTimeFlexible(row["signalAt"])
			if err != nil {
				res.Invalid[domain.ExportSectionBacktests]++
				continue
			}
			cp, _ := strconv.ParseFloat(row["closePrice"], 64)
			id := strings.TrimSpace(row["id"])
			if id == "" || btID == "" {
				res.Invalid[domain.ExportSectionBacktests]++
				continue
			}
			sig := domain.ScannerBacktestSignal{
				ID: id, BacktestID: btID, SignalAt: sigAt, ClosePrice: cp, Summary: row["summary"],
				Return1d: parseOptFloat(row["return1d"]), Return5d: parseOptFloat(row["return5d"]), Return20d: parseOptFloat(row["return20d"]),
			}
			signalsByBT[btID] = append(signalsByBT[btID], sig)
		case "portfolios":
			id := strings.TrimSpace(row["id"])
			if id == "" {
				res.Invalid[domain.ExportSectionPortfolios]++
				continue
			}
			b := &wirePortfolioBook{
				ID: id, Name: row["name"], Currency: row["currency"],
				StartingBalance: csvF64(row, "startingBalance"), CashBalance: csvF64(row, "cashBalance"),
				RealizedPnLTotal: csvF64(row, "realizedPnLTotal"), NetDeposits: csvF64(row, "netDeposits"),
				MarginMode: row["marginMode"], CreatedAt: row["createdAt"], UpdatedAt: row["updatedAt"],
			}
			csvBooks[id] = b
		case "portfolio_positions":
			if b := csvBooks[row["portfolioId"]]; b != nil {
				b.Positions = append(b.Positions, wirePFPos{
					Exchange: row["exchange"], Symbol: row["symbol"], Quantity: csvF64(row, "quantity"),
					AvgCost: csvF64(row, "avgCost"), UpdatedAt: row["updatedAt"],
				})
			}
		case "portfolio_trades":
			if b := csvBooks[row["portfolioId"]]; b != nil {
				b.Trades = append(b.Trades, wirePFTrade{
					ID: row["id"], Exchange: row["exchange"], Symbol: row["symbol"], Side: row["side"],
					Quantity: csvF64(row, "quantity"), Price: csvF64(row, "price"), Notional: csvF64(row, "notional"),
					RealizedPnL: csvF64(row, "realizedPnL"), PendingOrderID: row["pendingOrderId"], LotMethod: row["lotMethod"],
					Fee: csvF64(row, "fee"), LastPrice: csvF64(row, "lastPrice"), CreatedAt: row["createdAt"],
				})
			}
		case "portfolio_orders":
			if b := csvBooks[row["portfolioId"]]; b != nil {
				b.OpenOrders = append(b.OpenOrders, wirePFOrder{
					ID: row["id"], Exchange: row["exchange"], Symbol: row["symbol"], Type: row["type"], Side: row["side"],
					Quantity: csvF64(row, "quantity"), FilledQuantity: csvF64(row, "filledQuantity"),
					RemainingQuantity: csvF64(row, "remainingQuantity"), TriggerPrice: csvF64(row, "triggerPrice"),
					ReservedCash: csvF64(row, "reservedCash"), ReservedQuantity: csvF64(row, "reservedQuantity"),
					TimeInForce: row["timeInForce"], ExpiresAt: csvOptStr(row, "expiresAt"), Status: row["status"],
					OCOGroupID: row["ocoGroupId"], OCOPeerID: row["ocoPeerId"], TrailType: row["trailType"],
					TrailValue: csvF64(row, "trailValue"), TrailPeak: csvF64(row, "trailPeak"),
					BracketID: row["bracketId"], BracketRole: row["bracketRole"], LotMethod: row["lotMethod"],
					CreatedAt: row["createdAt"], UpdatedAt: row["updatedAt"],
				})
			}
		case "portfolio_lots":
			if b := csvBooks[row["portfolioId"]]; b != nil {
				b.Lots = append(b.Lots, wirePFLot{
					ID: row["id"], Exchange: row["exchange"], Symbol: row["symbol"], Quantity: csvF64(row, "quantity"),
					OriginalQuantity: csvF64(row, "originalQuantity"), Price: csvF64(row, "price"),
					OpenedAt: row["openedAt"], SourceTradeID: row["sourceTradeId"], ClosedAt: csvOptStr(row, "closedAt"),
				})
			}
		case "portfolio_lot_fills":
			if b := csvBooks[row["portfolioId"]]; b != nil {
				b.LotFills = append(b.LotFills, wirePFLotFill{
					ID: row["id"], TradeID: row["tradeId"], LotID: row["lotId"], Quantity: csvF64(row, "quantity"),
					CostPrice: csvF64(row, "costPrice"), SellPrice: csvF64(row, "sellPrice"), RealizedPnL: csvF64(row, "realizedPnL"),
				})
			}
		case "portfolio_recurring_buys":
			if b := csvBooks[row["portfolioId"]]; b != nil {
				b.RecurringBuys = append(b.RecurringBuys, wirePFRecurring{
					ID: row["id"], Exchange: row["exchange"], Symbol: row["symbol"], Name: row["name"],
					Amount: csvF64(row, "amount"), Frequency: row["frequency"], Weekday: row["weekday"],
					DayOfMonth: csvI(row, "dayOfMonth"), IntervalHours: csvI(row, "intervalHours"), Status: row["status"],
					NextRunAt: row["nextRunAt"], LastRunAt: csvOptStr(row, "lastRunAt"), LastPeriodKey: row["lastPeriodKey"],
					CreatedAt: row["createdAt"], UpdatedAt: row["updatedAt"],
				})
			}
		case "portfolio_recurring_runs":
			if b := csvBooks[row["portfolioId"]]; b != nil {
				b.RecurringRuns = append(b.RecurringRuns, wirePFRecurringRun{
					ID: row["id"], PlanID: row["planId"], PeriodKey: row["periodKey"], Status: row["status"],
					Amount: csvF64(row, "amount"), Quantity: csvF64(row, "quantity"), Price: csvF64(row, "price"),
					TradeID: row["tradeId"], FailReason: row["failReason"], ScheduledFor: row["scheduledFor"], ExecutedAt: row["executedAt"],
				})
			}
		case "portfolio_margin_positions":
			if b := csvBooks[row["portfolioId"]]; b != nil {
				b.MarginPositions = append(b.MarginPositions, wirePFMarginPos{
					ID: row["id"], Exchange: row["exchange"], Symbol: row["symbol"], Side: row["side"], Mode: row["mode"],
					Quantity: csvF64(row, "quantity"), EntryPrice: csvF64(row, "entryPrice"), Leverage: csvI(row, "leverage"),
					Margin: csvF64(row, "margin"), DebtPrincipal: csvF64(row, "debtPrincipal"), DebtInterest: csvF64(row, "debtInterest"),
					DebtAsset: row["debtAsset"], LastInterestAt: row["lastInterestAt"], LiquidationPrice: csvF64(row, "liquidationPrice"),
					StopLoss: csvOptF64(row, "stopLoss"), TakeProfit: csvOptF64(row, "takeProfit"), Status: row["status"],
					RealizedPnL: csvF64(row, "realizedPnL"), CloseReason: row["closeReason"],
					OpenedAt: row["openedAt"], UpdatedAt: row["updatedAt"], ClosedAt: csvOptStr(row, "closedAt"),
				})
			}
		case "portfolio_margin_orders":
			if b := csvBooks[row["portfolioId"]]; b != nil {
				b.MarginOrders = append(b.MarginOrders, wirePFMarginOrd{
					ID: row["id"], Exchange: row["exchange"], Symbol: row["symbol"], Side: row["side"], Type: row["type"],
					Quantity: csvF64(row, "quantity"), Leverage: csvI(row, "leverage"), LimitPrice: csvF64(row, "limitPrice"),
					ReservedMargin: csvF64(row, "reservedMargin"), StopLoss: csvOptF64(row, "stopLoss"), TakeProfit: csvOptF64(row, "takeProfit"),
					Status: row["status"], PositionID: row["positionId"], CreatedAt: row["createdAt"], UpdatedAt: row["updatedAt"],
				})
			}
		case "portfolio_margin_trades":
			if b := csvBooks[row["portfolioId"]]; b != nil {
				b.MarginTrades = append(b.MarginTrades, wirePFMarginTr{
					ID: row["id"], PositionID: row["positionId"], Exchange: row["exchange"], Symbol: row["symbol"],
					Side: row["side"], Action: row["action"], Quantity: csvF64(row, "quantity"), Price: csvF64(row, "price"),
					Notional: csvF64(row, "notional"), RealizedPnL: csvF64(row, "realizedPnL"), MarginDelta: csvF64(row, "marginDelta"),
					PrincipalPaid: csvF64(row, "principalPaid"), InterestPaid: csvF64(row, "interestPaid"),
					Leverage: csvI(row, "leverage"), Fee: csvF64(row, "fee"), CreatedAt: row["createdAt"],
				})
			}
		case "portfolio_shares":
			if b := csvBooks[row["portfolioId"]]; b != nil {
				b.Shares = append(b.Shares, wirePFShare{
					GranteeClientID: row["granteeClientId"], Role: row["role"], CreatedAt: row["createdAt"], UpdatedAt: row["updatedAt"],
				})
			}
		}
	}
	for id, b := range backtestsByID {
		b.Signals = signalsByBT[id]
		res.Payload.Backtests = append(res.Payload.Backtests, *b)
	}
	seenPF := map[string]struct{}{}
	for _, pb := range csvBooks {
		snap, err := normalizePortfolioBook(*pb, fileOwner, importer)
		if err != nil {
			res.Invalid[domain.ExportSectionPortfolios]++
			continue
		}
		if _, ok := seenPF[snap.Book.ID]; ok {
			res.FileDuplicates[domain.ExportSectionPortfolios]++
			continue
		}
		seenPF[snap.Book.ID] = struct{}{}
		res.Payload.Portfolios = append(res.Payload.Portfolios, snap)
	}
	return res, nil
}

func parseOptFloat(s string) *float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}

func parseTimeFlexible(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty time")
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("bad time")
}

func normalizeWLItem(exchange, symbol, note, addedAt string) (domain.WatchlistItem, error) {
	rawEx := strings.TrimSpace(exchange)
	var ex domain.Exchange
	if rawEx == "" {
		ex = domain.DefaultExchange
	} else {
		if !domain.IsValidExchange(rawEx) {
			return domain.WatchlistItem{}, fmt.Errorf("bad exchange")
		}
		ex = domain.ParseExchange(rawEx)
	}
	sym := domain.NormalizeSymbol(ex, symbol)
	if sym == "" {
		return domain.WatchlistItem{}, fmt.Errorf("symbol required")
	}
	added := time.Now().UTC()
	if t, err := parseTimeFlexible(addedAt); err == nil {
		added = t
	}
	return domain.WatchlistItem{Exchange: ex, Symbol: sym, Note: strings.TrimSpace(note), AddedAt: added}, nil
}

func normalizeShare(owner, grantee, role, createdAt, updatedAt string) (domain.WatchlistShare, error) {
	grantee = strings.TrimSpace(grantee)
	if grantee == "" {
		return domain.WatchlistShare{}, fmt.Errorf("grantee required")
	}
	if grantee == owner {
		return domain.WatchlistShare{}, fmt.Errorf("cannot share with self")
	}
	r, err := domain.NormalizeWatchlistShareRole(role)
	if err != nil {
		return domain.WatchlistShare{}, err
	}
	now := time.Now().UTC()
	cAt, uAt := now, now
	if t, err := parseTimeFlexible(createdAt); err == nil {
		cAt = t
	}
	if t, err := parseTimeFlexible(updatedAt); err == nil {
		uAt = t
	}
	return domain.WatchlistShare{
		OwnerClientID: owner, GranteeClientID: grantee, Role: r, CreatedAt: cAt, UpdatedAt: uAt,
	}, nil
}

func normalizeAlert(clientID, id, exchange, symbol, kind, condition string, target, rangePct float64, mode, status, createdAt string, triggeredAt *string, triggeredPrice float64) (domain.PriceAlert, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.PriceAlert{}, fmt.Errorf("id required")
	}
	rawEx := strings.TrimSpace(exchange)
	var ex domain.Exchange
	if rawEx == "" {
		ex = domain.DefaultExchange
	} else {
		if !domain.IsValidExchange(rawEx) {
			return domain.PriceAlert{}, fmt.Errorf("bad exchange")
		}
		ex = domain.ParseExchange(rawEx)
	}
	sym := domain.NormalizeSymbol(ex, symbol)
	if sym == "" {
		return domain.PriceAlert{}, fmt.Errorf("symbol required")
	}
	k, ok := domain.NormalizeAlertKind(kind)
	if !ok {
		return domain.PriceAlert{}, fmt.Errorf("bad kind")
	}
	if err := domain.ValidateAlertSpec(k, condition, target, rangePct); err != nil {
		return domain.PriceAlert{}, err
	}
	m, ok := domain.NormalizeAlertMode(mode)
	if !ok {
		m = domain.AlertModeOneTime
	}
	st := domain.AlertStatus(strings.ToLower(strings.TrimSpace(status)))
	if st != domain.AlertStatusActive && st != domain.AlertStatusTriggered {
		st = domain.AlertStatusActive
	}
	cAt := time.Now().UTC()
	if t, err := parseTimeFlexible(createdAt); err == nil {
		cAt = t
	}
	var tAt *time.Time
	if triggeredAt != nil {
		if t, err := parseTimeFlexible(*triggeredAt); err == nil {
			tAt = &t
		}
	}
	if domain.IsBookAlert(k) && rangePct <= 0 {
		rangePct = domain.DefaultOrderBookRangePct
	}
	if !domain.IsBookAlert(k) {
		rangePct = 0
	}
	return domain.PriceAlert{
		ID: id, ClientID: clientID, Exchange: ex, Symbol: sym, Kind: k,
		Condition:   domain.AlertCondition(strings.ToLower(strings.TrimSpace(condition))),
		TargetPrice: target, RangePct: rangePct, Mode: m, Status: st, CreatedAt: cAt,
		TriggeredAt: tAt, TriggeredPrice: triggeredPrice,
	}, nil
}

func normalizeBacktest(clientID, id, ruleID, exchange, symbol, interval, rangeStart, rangeEnd, status string, signalCount int, createdAt, errMsg string, wireSignals []wireSignal) (backtestBundle, error) {
	id = strings.TrimSpace(id)
	ruleID = strings.TrimSpace(ruleID)
	if id == "" || ruleID == "" {
		return backtestBundle{}, fmt.Errorf("id/ruleId required")
	}
	rawEx := strings.TrimSpace(exchange)
	var ex domain.Exchange
	if rawEx == "" {
		ex = domain.DefaultExchange
	} else {
		if !domain.IsValidExchange(rawEx) {
			return backtestBundle{}, fmt.Errorf("bad exchange")
		}
		ex = domain.ParseExchange(rawEx)
	}
	sym := domain.NormalizeSymbol(ex, symbol)
	if sym == "" {
		return backtestBundle{}, fmt.Errorf("symbol required")
	}
	rs, err1 := parseTimeFlexible(rangeStart)
	re, err2 := parseTimeFlexible(rangeEnd)
	if err1 != nil || err2 != nil {
		return backtestBundle{}, fmt.Errorf("bad range")
	}
	st := domain.ScannerBacktestStatus(strings.ToLower(strings.TrimSpace(status)))
	switch st {
	case domain.BacktestPending, domain.BacktestRunning, domain.BacktestCompleted, domain.BacktestCanceled, domain.BacktestFailed:
	default:
		st = domain.BacktestCompleted
	}
	cAt := time.Now().UTC()
	if t, err := parseTimeFlexible(createdAt); err == nil {
		cAt = t
	}
	iv := strings.TrimSpace(interval)
	if iv == "" {
		iv = "1h"
	}
	job := domain.ScannerBacktest{
		ID: id, ClientID: clientID, RuleID: ruleID, Exchange: ex, Symbol: sym, Interval: iv,
		RangeStart: rs, RangeEnd: re, Status: st, SignalCount: signalCount,
		ErrorMessage: errMsg, CreatedAt: cAt, ProgressPct: 100,
	}
	var sigs []domain.ScannerBacktestSignal
	for _, ws := range wireSignals {
		sigID := strings.TrimSpace(ws.ID)
		if sigID == "" {
			continue
		}
		sat, err := parseTimeFlexible(ws.SignalAt)
		if err != nil {
			continue
		}
		sigs = append(sigs, domain.ScannerBacktestSignal{
			ID: sigID, BacktestID: id, SignalAt: sat, ClosePrice: ws.ClosePrice, Summary: ws.Summary,
			Return1d: ws.Return1d, Return5d: ws.Return5d, Return20d: ws.Return20d, Metrics: ws.Metrics,
		})
	}
	return backtestBundle{Job: job, Signals: sigs}, nil
}
