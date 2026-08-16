package export

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// payload is the JSON export shape.
type payload struct {
	ExportedAt string                 `json:"exportedAt"`
	ClientID   string                 `json:"clientId"`
	Sections   []string               `json:"sections"`
	Watchlist  *watchlistPayload      `json:"watchlist,omitempty"`
	Shares     []sharePayload         `json:"shares,omitempty"`
	Alerts     []alertPayload         `json:"alerts,omitempty"`
	Backtests  []backtestPayload      `json:"backtests,omitempty"`
	Portfolios []portfolioBookPayload `json:"portfolios,omitempty"`
}

type watchlistPayload struct {
	ClientID  string                 `json:"clientId"`
	UpdatedAt string                 `json:"updatedAt"`
	Items     []watchlistItemPayload `json:"items"`
}

type watchlistItemPayload struct {
	Exchange string `json:"exchange"`
	Symbol   string `json:"symbol"`
	Note     string `json:"note,omitempty"`
	AddedAt  string `json:"addedAt"`
}

type sharePayload struct {
	OwnerClientID   string `json:"ownerClientId"`
	GranteeClientID string `json:"granteeClientId"`
	Role            string `json:"role"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

type alertPayload struct {
	ID             string  `json:"id"`
	Exchange       string  `json:"exchange"`
	Symbol         string  `json:"symbol"`
	Kind           string  `json:"kind,omitempty"`
	Condition      string  `json:"condition"`
	TargetPrice    float64 `json:"targetPrice"`
	RangePct       float64 `json:"rangePct,omitempty"`
	Mode           string  `json:"mode"`
	Status         string  `json:"status"`
	CreatedAt      string  `json:"createdAt"`
	TriggeredAt    *string `json:"triggeredAt,omitempty"`
	TriggeredPrice float64 `json:"triggeredPrice,omitempty"`
}

type backtestPayload struct {
	ID           string                  `json:"id"`
	RuleID       string                  `json:"ruleId"`
	Exchange     string                  `json:"exchange"`
	Symbol       string                  `json:"symbol"`
	Interval     string                  `json:"interval"`
	RangeStart   string                  `json:"rangeStart"`
	RangeEnd     string                  `json:"rangeEnd"`
	Status       string                  `json:"status"`
	SignalCount  int                     `json:"signalCount"`
	CreatedAt    string                  `json:"createdAt"`
	ErrorMessage string                  `json:"errorMessage,omitempty"`
	Signals      []backtestSignalPayload `json:"signals"`
}

type backtestSignalPayload struct {
	ID         string             `json:"id"`
	SignalAt   string             `json:"signalAt"`
	ClosePrice float64            `json:"closePrice"`
	Summary    string             `json:"summary"`
	Return1d   *float64           `json:"return1d,omitempty"`
	Return5d   *float64           `json:"return5d,omitempty"`
	Return20d  *float64           `json:"return20d,omitempty"`
	Metrics    map[string]float64 `json:"metrics,omitempty"`
}

// runJob builds the export file for a claimed job.
func (s *Service) runJob(ctx context.Context, job *domain.ExportJob) error {
	// Cancel check helper
	canceled := func() bool {
		st, err := s.store.GetStatus(ctx, job.ID)
		return err == nil && st == domain.ExportCanceled
	}
	if canceled() {
		return s.store.Finish(ctx, job.ID, domain.ExportCanceled, "", "", 0, nil, "", time.Now().UTC())
	}

	nSec := len(job.Sections)
	if nSec == 0 {
		nSec = 1
	}
	pl := payload{
		ExportedAt: time.Now().UTC().Format(time.RFC3339Nano),
		ClientID:   job.ClientID,
	}
	for _, sec := range job.Sections {
		pl.Sections = append(pl.Sections, string(sec))
	}

	for i, sec := range job.Sections {
		if canceled() {
			return s.store.Finish(ctx, job.ID, domain.ExportCanceled, "", "", 0, nil, "", time.Now().UTC())
		}
		pct := float64(i) / float64(nSec) * 100
		_ = s.store.UpdateProgress(ctx, job.ID, pct, string(sec))

		switch sec {
		case domain.ExportSectionWatchlist:
			if s.data.Watchlist == nil {
				return fmt.Errorf("watchlist store not configured")
			}
			wl, err := s.data.Watchlist.Get(ctx, job.ClientID)
			if err != nil {
				return err
			}
			items := make([]watchlistItemPayload, 0, len(wl.Items))
			for _, it := range wl.Items {
				items = append(items, watchlistItemPayload{
					Exchange: string(it.Exchange), Symbol: it.Symbol, Note: it.Note,
					AddedAt: it.AddedAt.UTC().Format(time.RFC3339Nano),
				})
			}
			pl.Watchlist = &watchlistPayload{
				ClientID: wl.ClientID, UpdatedAt: wl.Updated.UTC().Format(time.RFC3339Nano), Items: items,
			}
		case domain.ExportSectionShares:
			if s.data.Watchlist == nil {
				return fmt.Errorf("watchlist store not configured")
			}
			shares, err := s.data.Watchlist.ListSharesByOwner(ctx, job.ClientID)
			if err != nil {
				return err
			}
			for _, sh := range shares {
				pl.Shares = append(pl.Shares, sharePayload{
					OwnerClientID: sh.OwnerClientID, GranteeClientID: sh.GranteeClientID,
					Role:      string(sh.Role),
					CreatedAt: sh.CreatedAt.UTC().Format(time.RFC3339Nano),
					UpdatedAt: sh.UpdatedAt.UTC().Format(time.RFC3339Nano),
				})
			}
		case domain.ExportSectionAlerts:
			if s.data.Alerts == nil {
				return fmt.Errorf("alerts store not configured")
			}
			alerts, err := s.data.Alerts.ListByClient(ctx, job.ClientID)
			if err != nil {
				return err
			}
			for _, a := range alerts {
				ap := alertPayload{
					ID: a.ID, Exchange: string(a.Exchange), Symbol: a.Symbol,
					Kind:      string(domain.EffectiveAlertKind(a)),
					Condition: string(a.Condition), TargetPrice: a.TargetPrice,
					RangePct: a.RangePct,
					Mode:     string(a.Mode), Status: string(a.Status),
					CreatedAt:      a.CreatedAt.UTC().Format(time.RFC3339Nano),
					TriggeredPrice: a.TriggeredPrice,
				}
				if a.TriggeredAt != nil {
					t := a.TriggeredAt.UTC().Format(time.RFC3339Nano)
					ap.TriggeredAt = &t
				}
				pl.Alerts = append(pl.Alerts, ap)
			}
		case domain.ExportSectionBacktests:
			if s.data.Scanner == nil {
				return fmt.Errorf("scanner store not configured")
			}
			// Page through all backtests
			offset := 0
			const page = 100
			for {
				if canceled() {
					return s.store.Finish(ctx, job.ID, domain.ExportCanceled, "", "", 0, nil, "", time.Now().UTC())
				}
				list, err := s.data.Scanner.ListBacktests(ctx, job.ClientID, page, offset)
				if err != nil {
					return err
				}
				if len(list) == 0 {
					break
				}
				for _, bt := range list {
					bp := backtestPayload{
						ID: bt.ID, RuleID: bt.RuleID, Exchange: string(bt.Exchange), Symbol: bt.Symbol,
						Interval:   bt.Interval,
						RangeStart: bt.RangeStart.UTC().Format(time.RFC3339Nano),
						RangeEnd:   bt.RangeEnd.UTC().Format(time.RFC3339Nano),
						Status:     string(bt.Status), SignalCount: bt.SignalCount,
						CreatedAt:    bt.CreatedAt.UTC().Format(time.RFC3339Nano),
						ErrorMessage: bt.ErrorMessage,
					}
					// All signals for this backtest
					sigOff := 0
					for {
						sigs, err := s.data.Scanner.ListBacktestSignals(ctx, bt.ID, page, sigOff)
						if err != nil {
							return err
						}
						if len(sigs) == 0 {
							break
						}
						for _, sig := range sigs {
							bp.Signals = append(bp.Signals, backtestSignalPayload{
								ID: sig.ID, SignalAt: sig.SignalAt.UTC().Format(time.RFC3339Nano),
								ClosePrice: sig.ClosePrice, Summary: sig.Summary,
								Return1d: sig.Return1d, Return5d: sig.Return5d, Return20d: sig.Return20d,
								Metrics: sig.Metrics,
							})
						}
						if len(sigs) < page {
							break
						}
						sigOff += page
					}
					pl.Backtests = append(pl.Backtests, bp)
				}
				if len(list) < page {
					break
				}
				offset += page
			}
		case domain.ExportSectionPortfolios:
			if s.data.Portfolio == nil {
				break
			}
			snaps, err := s.data.Portfolio.ExportOwnedPortfolios(ctx, job.ClientID)
			if err != nil {
				return err
			}
			for _, snap := range snaps {
				pl.Portfolios = append(pl.Portfolios, snapshotToPayload(snap))
			}
		}
	}

	if canceled() {
		return s.store.Finish(ctx, job.ID, domain.ExportCanceled, "", "", 0, nil, "", time.Now().UTC())
	}
	_ = s.store.UpdateProgress(ctx, job.ID, 95, "writing")

	clientDir := filepath.Join(s.fileDir, sanitizePathPart(job.ClientID))
	if err := os.MkdirAll(clientDir, 0o755); err != nil {
		return err
	}
	ext := "json"
	if job.Format == domain.ExportFormatCSV {
		ext = "csv"
	}
	fileName := fmt.Sprintf("swyngora-export-%s.%s", job.ID[:8], ext)
	filePath := filepath.Join(clientDir, job.ID+"."+ext)

	var data []byte
	var err error
	if job.Format == domain.ExportFormatCSV {
		data, err = buildCSV(&pl)
	} else {
		data, err = json.MarshalIndent(&pl, "", "  ")
		if err == nil {
			data = append(data, '\n')
		}
	}
	if err != nil {
		return err
	}
	if err := os.WriteFile(filePath, data, 0o600); err != nil {
		return err
	}

	// Final cancel race: if canceled after write, drop file and mark canceled.
	if canceled() {
		_ = os.Remove(filePath)
		return s.store.Finish(ctx, job.ID, domain.ExportCanceled, "", "", 0, nil, "", time.Now().UTC())
	}

	now := time.Now().UTC()
	exp := now.Add(s.fileTTL)
	return s.store.Finish(ctx, job.ID, domain.ExportCompleted, fileName, filePath, int64(len(data)), &exp, "", now)
}

func sanitizePathPart(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	out := b.String()
	if out == "" || out == "." || out == ".." {
		return "_empty"
	}
	return out
}

func buildCSV(pl *payload) ([]byte, error) {
	var buf strings.Builder
	w := csv.NewWriter(&buf)

	writeSection := func(name string, header []string, rows [][]string) error {
		if err := w.Write([]string{"# section:" + name}); err != nil {
			return err
		}
		if err := w.Write(header); err != nil {
			return err
		}
		for _, row := range rows {
			if err := w.Write(row); err != nil {
				return err
			}
		}
		w.Flush()
		return w.Error()
	}

	if err := w.Write([]string{"# meta", "exportedAt", pl.ExportedAt, "clientId", pl.ClientID}); err != nil {
		return nil, err
	}
	w.Flush()

	if pl.Watchlist != nil {
		rows := make([][]string, 0, len(pl.Watchlist.Items))
		for _, it := range pl.Watchlist.Items {
			rows = append(rows, []string{it.Exchange, it.Symbol, it.Note, it.AddedAt})
		}
		if err := writeSection("watchlist", []string{"exchange", "symbol", "note", "addedAt"}, rows); err != nil {
			return nil, err
		}
	}
	if pl.Shares != nil {
		rows := make([][]string, 0, len(pl.Shares))
		for _, sh := range pl.Shares {
			rows = append(rows, []string{sh.OwnerClientID, sh.GranteeClientID, sh.Role, sh.CreatedAt, sh.UpdatedAt})
		}
		if err := writeSection("shares", []string{"ownerClientId", "granteeClientId", "role", "createdAt", "updatedAt"}, rows); err != nil {
			return nil, err
		}
	}
	if pl.Alerts != nil {
		rows := make([][]string, 0, len(pl.Alerts))
		for _, a := range pl.Alerts {
			trig := ""
			if a.TriggeredAt != nil {
				trig = *a.TriggeredAt
			}
			rows = append(rows, []string{
				a.ID, a.Exchange, a.Symbol, a.Kind, a.Condition,
				strconv.FormatFloat(a.TargetPrice, 'f', -1, 64),
				strconv.FormatFloat(a.RangePct, 'f', -1, 64),
				a.Mode, a.Status, a.CreatedAt, trig,
				strconv.FormatFloat(a.TriggeredPrice, 'f', -1, 64),
			})
		}
		if err := writeSection("alerts", []string{
			"id", "exchange", "symbol", "kind", "condition", "targetPrice", "rangePct", "mode", "status", "createdAt", "triggeredAt", "triggeredPrice",
		}, rows); err != nil {
			return nil, err
		}
	}
	if pl.Backtests != nil {
		btRows := make([][]string, 0)
		sigRows := make([][]string, 0)
		for _, bt := range pl.Backtests {
			btRows = append(btRows, []string{
				bt.ID, bt.RuleID, bt.Exchange, bt.Symbol, bt.Interval,
				bt.RangeStart, bt.RangeEnd, bt.Status, strconv.Itoa(bt.SignalCount), bt.CreatedAt, bt.ErrorMessage,
			})
			for _, sig := range bt.Signals {
				sigRows = append(sigRows, []string{
					sig.ID, bt.ID, sig.SignalAt, strconv.FormatFloat(sig.ClosePrice, 'f', -1, 64), sig.Summary,
					fmtPtr(sig.Return1d), fmtPtr(sig.Return5d), fmtPtr(sig.Return20d),
				})
			}
		}
		if err := writeSection("backtests", []string{
			"id", "ruleId", "exchange", "symbol", "interval", "rangeStart", "rangeEnd", "status", "signalCount", "createdAt", "errorMessage",
		}, btRows); err != nil {
			return nil, err
		}
		if err := writeSection("backtest_signals", []string{
			"id", "backtestId", "signalAt", "closePrice", "summary", "return1d", "return5d", "return20d",
		}, sigRows); err != nil {
			return nil, err
		}
	}
	if len(pl.Portfolios) > 0 {
		bookRows := make([][]string, 0, len(pl.Portfolios))
		posRows, tradeRows, orderRows, lotRows, fillRows := [][]string{}, [][]string{}, [][]string{}, [][]string{}, [][]string{}
		recRows, runRows, mposRows, mordRows, mtrRows, shRows := [][]string{}, [][]string{}, [][]string{}, [][]string{}, [][]string{}, [][]string{}
		for _, b := range pl.Portfolios {
			bookRows = append(bookRows, []string{
				b.ID, b.Name, b.Currency, f64(b.StartingBalance), f64(b.CashBalance),
				f64(b.RealizedPnLTotal), f64(b.NetDeposits), b.MarginMode, b.CreatedAt, b.UpdatedAt,
			})
			for _, p := range b.Positions {
				posRows = append(posRows, []string{b.ID, p.Exchange, p.Symbol, f64(p.Quantity), f64(p.AvgCost), p.UpdatedAt})
			}
			for _, t := range b.Trades {
				tradeRows = append(tradeRows, []string{
					b.ID, t.ID, t.Exchange, t.Symbol, t.Side, f64(t.Quantity), f64(t.Price), f64(t.Notional),
					f64(t.RealizedPnL), t.PendingOrderID, t.LotMethod, f64(t.Fee), f64(t.LastPrice), t.CreatedAt,
				})
			}
			for _, o := range b.OpenOrders {
				orderRows = append(orderRows, []string{
					b.ID, o.ID, o.Exchange, o.Symbol, o.Type, o.Side, f64(o.Quantity), f64(o.FilledQuantity),
					f64(o.RemainingQuantity), f64(o.TriggerPrice), f64(o.ReservedCash), f64(o.ReservedQuantity),
					o.TimeInForce, derefStr(o.ExpiresAt), o.Status, o.OCOGroupID, o.OCOPeerID, o.TrailType,
					f64(o.TrailValue), f64(o.TrailPeak), o.BracketID, o.BracketRole, o.LotMethod, o.CreatedAt, o.UpdatedAt,
				})
			}
			for _, l := range b.Lots {
				lotRows = append(lotRows, []string{
					b.ID, l.ID, l.Exchange, l.Symbol, f64(l.Quantity), f64(l.OriginalQuantity), f64(l.Price),
					l.OpenedAt, l.SourceTradeID, derefStr(l.ClosedAt),
				})
			}
			for _, f := range b.LotFills {
				fillRows = append(fillRows, []string{b.ID, f.ID, f.TradeID, f.LotID, f64(f.Quantity), f64(f.CostPrice), f64(f.SellPrice), f64(f.RealizedPnL)})
			}
			for _, r := range b.RecurringBuys {
				recRows = append(recRows, []string{
					b.ID, r.ID, r.Exchange, r.Symbol, r.Name, f64(r.Amount), r.Frequency, r.Weekday, i64(r.DayOfMonth),
					i64(r.IntervalHours), r.Status, r.NextRunAt, derefStr(r.LastRunAt), r.LastPeriodKey, r.CreatedAt, r.UpdatedAt,
				})
			}
			for _, r := range b.RecurringRuns {
				runRows = append(runRows, []string{
					b.ID, r.ID, r.PlanID, r.PeriodKey, r.Status, f64(r.Amount), f64(r.Quantity), f64(r.Price),
					r.TradeID, r.FailReason, r.ScheduledFor, r.ExecutedAt,
				})
			}
			for _, m := range b.MarginPositions {
				mposRows = append(mposRows, []string{
					b.ID, m.ID, m.Exchange, m.Symbol, m.Side, m.Mode, f64(m.Quantity), f64(m.EntryPrice), i64(m.Leverage),
					f64(m.Margin), f64(m.DebtPrincipal), f64(m.DebtInterest), m.DebtAsset, m.LastInterestAt,
					f64(m.LiquidationPrice), derefF64(m.StopLoss), derefF64(m.TakeProfit), m.Status, f64(m.RealizedPnL),
					m.CloseReason, m.OpenedAt, m.UpdatedAt, derefStr(m.ClosedAt),
				})
			}
			for _, o := range b.MarginOrders {
				mordRows = append(mordRows, []string{
					b.ID, o.ID, o.Exchange, o.Symbol, o.Side, o.Type, f64(o.Quantity), i64(o.Leverage), f64(o.LimitPrice),
					f64(o.ReservedMargin), derefF64(o.StopLoss), derefF64(o.TakeProfit), o.Status, o.PositionID, o.CreatedAt, o.UpdatedAt,
				})
			}
			for _, tr := range b.MarginTrades {
				mtrRows = append(mtrRows, []string{
					b.ID, tr.ID, tr.PositionID, tr.Exchange, tr.Symbol, tr.Side, tr.Action, f64(tr.Quantity), f64(tr.Price),
					f64(tr.Notional), f64(tr.RealizedPnL), f64(tr.MarginDelta), f64(tr.PrincipalPaid), f64(tr.InterestPaid),
					i64(tr.Leverage), f64(tr.Fee), tr.CreatedAt,
				})
			}
			for _, sh := range b.Shares {
				shRows = append(shRows, []string{b.ID, sh.GranteeClientID, sh.Role, sh.CreatedAt, sh.UpdatedAt})
			}
		}
		if err := writeSection("portfolios", []string{
			"id", "name", "currency", "startingBalance", "cashBalance", "realizedPnLTotal", "netDeposits", "marginMode", "createdAt", "updatedAt",
		}, bookRows); err != nil {
			return nil, err
		}
		_ = writeSection("portfolio_positions", []string{"portfolioId", "exchange", "symbol", "quantity", "avgCost", "updatedAt"}, posRows)
		_ = writeSection("portfolio_trades", []string{
			"portfolioId", "id", "exchange", "symbol", "side", "quantity", "price", "notional", "realizedPnL", "pendingOrderId", "lotMethod", "fee", "lastPrice", "createdAt",
		}, tradeRows)
		_ = writeSection("portfolio_orders", []string{
			"portfolioId", "id", "exchange", "symbol", "type", "side", "quantity", "filledQuantity", "remainingQuantity",
			"triggerPrice", "reservedCash", "reservedQuantity", "timeInForce", "expiresAt", "status", "ocoGroupId", "ocoPeerId",
			"trailType", "trailValue", "trailPeak", "bracketId", "bracketRole", "lotMethod", "createdAt", "updatedAt",
		}, orderRows)
		_ = writeSection("portfolio_lots", []string{
			"portfolioId", "id", "exchange", "symbol", "quantity", "originalQuantity", "price", "openedAt", "sourceTradeId", "closedAt",
		}, lotRows)
		_ = writeSection("portfolio_lot_fills", []string{"portfolioId", "id", "tradeId", "lotId", "quantity", "costPrice", "sellPrice", "realizedPnL"}, fillRows)
		_ = writeSection("portfolio_recurring_buys", []string{
			"portfolioId", "id", "exchange", "symbol", "name", "amount", "frequency", "weekday", "dayOfMonth", "intervalHours",
			"status", "nextRunAt", "lastRunAt", "lastPeriodKey", "createdAt", "updatedAt",
		}, recRows)
		_ = writeSection("portfolio_recurring_runs", []string{
			"portfolioId", "id", "planId", "periodKey", "status", "amount", "quantity", "price", "tradeId", "failReason", "scheduledFor", "executedAt",
		}, runRows)
		_ = writeSection("portfolio_margin_positions", []string{
			"portfolioId", "id", "exchange", "symbol", "side", "mode", "quantity", "entryPrice", "leverage", "margin",
			"debtPrincipal", "debtInterest", "debtAsset", "lastInterestAt", "liquidationPrice", "stopLoss", "takeProfit",
			"status", "realizedPnL", "closeReason", "openedAt", "updatedAt", "closedAt",
		}, mposRows)
		_ = writeSection("portfolio_margin_orders", []string{
			"portfolioId", "id", "exchange", "symbol", "side", "type", "quantity", "leverage", "limitPrice", "reservedMargin",
			"stopLoss", "takeProfit", "status", "positionId", "createdAt", "updatedAt",
		}, mordRows)
		_ = writeSection("portfolio_margin_trades", []string{
			"portfolioId", "id", "positionId", "exchange", "symbol", "side", "action", "quantity", "price", "notional",
			"realizedPnL", "marginDelta", "principalPaid", "interestPaid", "leverage", "fee", "createdAt",
		}, mtrRows)
		_ = writeSection("portfolio_shares", []string{"portfolioId", "granteeClientId", "role", "createdAt", "updatedAt"}, shRows)
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}

func fmtPtr(p *float64) string {
	if p == nil {
		return ""
	}
	return strconv.FormatFloat(*p, 'f', -1, 64)
}
