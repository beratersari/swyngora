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
	ExportedAt    string                 `json:"exportedAt"`
	ClientID      string                 `json:"clientId"`
	Sections      []string               `json:"sections"`
	Watchlist     *watchlistPayload      `json:"watchlist,omitempty"`
	Shares        []sharePayload         `json:"shares,omitempty"`
	Alerts        []alertPayload         `json:"alerts,omitempty"`
	Backtests     []backtestPayload      `json:"backtests,omitempty"`
	Portfolios    []portfolioBookPayload `json:"portfolios,omitempty"`
	CustomSignals []customSignalPayload  `json:"customSignals,omitempty"`
}

type customSignalPayload struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	Weights   map[string]float64 `json:"weights"`
	IsDefault bool               `json:"isDefault"`
	CreatedAt string             `json:"createdAt"`
	UpdatedAt string             `json:"updatedAt"`
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
	ID         string                `json:"id"`
	SignalAt   string                `json:"signalAt"`
	ClosePrice float64               `json:"closePrice"`
	Summary    string                `json:"summary"`
	Return1d   *float64              `json:"return1d,omitempty"`
	Return5d   *float64              `json:"return5d,omitempty"`
	Return20d  *float64              `json:"return20d,omitempty"`
	Metrics    map[string]float64    `json:"metrics,omitempty"`
	Trade      *domain.BacktestTrade `json:"trade,omitempty"`
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
								Metrics: sig.Metrics, Trade: sig.Trade,
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
		case domain.ExportSectionCustomSignals:
			if s.data.CustomSignals == nil {
				break
			}
			list, err := s.data.CustomSignals.ListProfiles(ctx, job.ClientID)
			if err != nil {
				return err
			}
			for _, p := range list {
				pct := p.Weights.Pct
				if pct == nil {
					pct = map[string]float64{}
				}
				pl.CustomSignals = append(pl.CustomSignals, customSignalPayload{
					ID: p.ID, Name: p.Name, Weights: pct, IsDefault: p.IsDefault,
					CreatedAt: p.CreatedAt.UTC().Format(time.RFC3339Nano),
					UpdatedAt: p.UpdatedAt.UTC().Format(time.RFC3339Nano),
				})
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
				side, lev, entry, sl, tp, outcome, pnl := "", "", "", "", "", "", ""
				if sig.Trade != nil {
					side = sig.Trade.Side
					lev = strconv.Itoa(sig.Trade.Leverage)
					entry = strconv.FormatFloat(sig.Trade.Entry, 'f', -1, 64)
					sl = strconv.FormatFloat(sig.Trade.StopLoss, 'f', -1, 64)
					tp = strconv.FormatFloat(sig.Trade.TakeProfit, 'f', -1, 64)
					outcome = sig.Trade.Outcome
					pnl = strconv.FormatFloat(sig.Trade.PnLPct, 'f', -1, 64)
				}
				sigRows = append(sigRows, []string{
					sig.ID, bt.ID, sig.SignalAt, strconv.FormatFloat(sig.ClosePrice, 'f', -1, 64), sig.Summary,
					fmtPtr(sig.Return1d), fmtPtr(sig.Return5d), fmtPtr(sig.Return20d),
					side, lev, entry, sl, tp, outcome, pnl,
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
			"side", "leverage", "entry", "stopLoss", "takeProfit", "outcome", "pnlPct",
		}, sigRows); err != nil {
			return nil, err
		}
	}
	if len(pl.Portfolios) > 0 {
		bookRows := make([][]string, 0, len(pl.Portfolios))
		posRows, tradeRows, orderRows, lotRows, fillRows := [][]string{}, [][]string{}, [][]string{}, [][]string{}, [][]string{}
		recRows, runRows, mposRows, mordRows, mtrRows, shRows := [][]string{}, [][]string{}, [][]string{}, [][]string{}, [][]string{}, [][]string{}
		basketRows, targetRows, rbRows, rbLegRows, rbLineRows, rbAttRows, riskRows := [][]string{}, [][]string{}, [][]string{}, [][]string{}, [][]string{}, [][]string{}, [][]string{}
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
					f64(t.RealizedPnL), t.PendingOrderID, t.LotMethod, f64(t.Fee), f64(t.LastPrice), situationCSV(t.Situation), t.CreatedAt,
				})
			}
			for _, o := range b.OpenOrders {
				orderRows = append(orderRows, []string{
					b.ID, o.ID, o.Exchange, o.Symbol, o.Type, o.Side, f64(o.Quantity), f64(o.FilledQuantity),
					f64(o.RemainingQuantity), f64(o.TriggerPrice), f64(o.ReservedCash), f64(o.ReservedQuantity),
					o.TimeInForce, derefStr(o.ExpiresAt), o.Status, o.OCOGroupID, o.OCOPeerID, o.TrailType,
					f64(o.TrailValue), f64(o.TrailPeak), o.BracketID, o.BracketRole, f64(o.TargetPercent), o.LotMethod, o.CreatedAt, o.UpdatedAt,
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
					r.TradeID, r.FailReason, r.ScheduledFor, r.ExecutedAt, r.Timing,
				})
			}
			for _, m := range b.MarginPositions {
				mposRows = append(mposRows, []string{
					b.ID, m.ID, m.Exchange, m.Symbol, m.Side, m.Mode, f64(m.Quantity), f64(m.EntryPrice), i64(m.Leverage),
					f64(m.Margin), f64(m.DebtPrincipal), f64(m.DebtInterest), m.DebtAsset, m.LastInterestAt,
					f64(m.LiquidationPrice), derefF64(m.StopLoss), derefF64(m.TakeProfit), m.Status, f64(m.RealizedPnL),
					m.CloseReason, m.OpenedAt, m.UpdatedAt, derefStr(m.ClosedAt), situationCSV(m.EntrySituation),
				})
			}
			for _, o := range b.MarginOrders {
				mordRows = append(mordRows, []string{
					b.ID, o.ID, o.Exchange, o.Symbol, o.Side, o.Type, f64(o.Quantity), i64(o.Leverage), f64(o.LimitPrice),
					f64(o.ReservedMargin), derefF64(o.StopLoss), derefF64(o.TakeProfit), o.Status, o.PositionID, o.CreatedAt, o.UpdatedAt,
					situationCSV(o.Situation),
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
			for _, bk := range b.Baskets {
				basketRows = append(basketRows, []string{b.ID, bk.ID, bk.Name, bk.CreatedAt, bk.UpdatedAt})
				for _, t := range bk.Targets {
					targetRows = append(targetRows, []string{b.ID, bk.ID, t.Asset, t.Exchange, f64(t.WeightPct)})
				}
			}
			for _, r := range b.Rebalances {
				rbRows = append(rbRows, []string{
					b.ID, r.ID, r.BasketID, r.BasketName, r.Currency, r.Status,
					i64(r.PlannedLegs), i64(r.FilledLegs), i64(r.FailedLegs),
					f64(r.BeforeEquity), f64(r.AfterEquity), f64(r.BeforeCash), f64(r.AfterCash),
					f64(r.RemainingAbsDriftPct), f64(r.MaxSleeveDriftPct), f64(r.RemainingNotional),
					f64(r.TotalFee), f64(r.TotalSlippage), f64(r.TotalCost), r.Summary, r.Note,
					r.CreatedAt, r.UpdatedAt, i64(r.AttemptCount),
				})
				for _, leg := range r.Legs {
					rbLegRows = append(rbLegRows, csvRebalanceLeg(b.ID, r.ID, leg))
				}
				for _, ln := range r.Lines {
					rbLineRows = append(rbLineRows, csvRebalanceLine(b.ID, r.ID, ln))
				}
				for _, att := range r.Attempts {
					rbAttRows = append(rbAttRows, []string{
						b.ID, r.ID, i64(att.Seq), att.Status, i64(att.PlannedLegs), i64(att.FilledLegs), i64(att.FailedLegs),
						f64(att.BeforeEquity), f64(att.AfterEquity), f64(att.BeforeCash), f64(att.AfterCash),
						f64(att.TotalFee), f64(att.TotalSlippage), f64(att.TotalCost), att.Summary, att.CreatedAt,
					})
				}
			}
			if b.RiskLimits != nil {
				known := "false"
				if b.RiskLimits.DayStartFlowKnown {
					known = "true"
				}
				riskRows = append(riskRows, []string{
					b.ID, derefF64(b.RiskLimits.MaxDailyLossPct), derefF64(b.RiskLimits.MaxAssetWeightPct), derefF64(b.RiskLimits.MaxExposurePct),
					b.RiskLimits.DayKey, f64(b.RiskLimits.DayStartEquity), f64(b.RiskLimits.DayStartNetDeposits), known, b.RiskLimits.UpdatedAt,
				})
			}
		}
		if err := writeSection("portfolios", []string{
			"id", "name", "currency", "startingBalance", "cashBalance", "realizedPnLTotal", "netDeposits", "marginMode", "createdAt", "updatedAt",
		}, bookRows); err != nil {
			return nil, err
		}
		_ = writeSection("portfolio_positions", []string{"portfolioId", "exchange", "symbol", "quantity", "avgCost", "updatedAt"}, posRows)
		_ = writeSection("portfolio_trades", []string{
			"portfolioId", "id", "exchange", "symbol", "side", "quantity", "price", "notional", "realizedPnL", "pendingOrderId", "lotMethod", "fee", "lastPrice", "situation", "createdAt",
		}, tradeRows)
		_ = writeSection("portfolio_orders", []string{
			"portfolioId", "id", "exchange", "symbol", "type", "side", "quantity", "filledQuantity", "remainingQuantity",
			"triggerPrice", "reservedCash", "reservedQuantity", "timeInForce", "expiresAt", "status", "ocoGroupId", "ocoPeerId",
			"trailType", "trailValue", "trailPeak", "bracketId", "bracketRole", "targetPercent", "lotMethod", "createdAt", "updatedAt",
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
			"portfolioId", "id", "planId", "periodKey", "status", "amount", "quantity", "price", "tradeId", "failReason", "scheduledFor", "executedAt", "timing",
		}, runRows)
		_ = writeSection("portfolio_margin_positions", []string{
			"portfolioId", "id", "exchange", "symbol", "side", "mode", "quantity", "entryPrice", "leverage", "margin",
			"debtPrincipal", "debtInterest", "debtAsset", "lastInterestAt", "liquidationPrice", "stopLoss", "takeProfit",
			"status", "realizedPnL", "closeReason", "openedAt", "updatedAt", "closedAt", "entrySituation",
		}, mposRows)
		_ = writeSection("portfolio_margin_orders", []string{
			"portfolioId", "id", "exchange", "symbol", "side", "type", "quantity", "leverage", "limitPrice", "reservedMargin",
			"stopLoss", "takeProfit", "status", "positionId", "createdAt", "updatedAt", "situation",
		}, mordRows)
		_ = writeSection("portfolio_margin_trades", []string{
			"portfolioId", "id", "positionId", "exchange", "symbol", "side", "action", "quantity", "price", "notional",
			"realizedPnL", "marginDelta", "principalPaid", "interestPaid", "leverage", "fee", "createdAt",
		}, mtrRows)
		_ = writeSection("portfolio_shares", []string{"portfolioId", "granteeClientId", "role", "createdAt", "updatedAt"}, shRows)
		_ = writeSection("portfolio_baskets", []string{"portfolioId", "id", "name", "createdAt", "updatedAt"}, basketRows)
		_ = writeSection("portfolio_basket_targets", []string{"portfolioId", "basketId", "asset", "exchange", "weightPct"}, targetRows)
		_ = writeSection("portfolio_rebalances", []string{
			"portfolioId", "id", "basketId", "basketName", "currency", "status",
			"plannedLegs", "filledLegs", "failedLegs", "beforeEquity", "afterEquity", "beforeCash", "afterCash",
			"remainingAbsDriftPct", "maxSleeveDriftPct", "remainingNotional", "totalFee", "totalSlippage", "totalCost",
			"summary", "note", "createdAt", "updatedAt", "attemptCount",
		}, rbRows)
		_ = writeSection("portfolio_rebalance_legs", []string{
			"portfolioId", "rebalanceId", "side", "asset", "exchange", "symbol", "reason",
			"plannedQty", "plannedPrice", "plannedNotional", "status", "tradeId",
			"filledQty", "fillPrice", "fillNotional", "lastPrice", "fee", "slippage", "error", "attempt",
		}, rbLegRows)
		_ = writeSection("portfolio_rebalance_lines", []string{
			"portfolioId", "rebalanceId", "asset", "exchange", "symbol", "isCash",
			"targetPct", "beforePct", "afterPct", "beforeValue", "afterValue", "targetValue", "afterDelta", "attempt",
		}, rbLineRows)
		_ = writeSection("portfolio_rebalance_attempts", []string{
			"portfolioId", "rebalanceId", "seq", "status", "plannedLegs", "filledLegs", "failedLegs",
			"beforeEquity", "afterEquity", "beforeCash", "afterCash", "totalFee", "totalSlippage", "totalCost",
			"summary", "createdAt",
		}, rbAttRows)
		_ = writeSection("portfolio_risk_limits", []string{
			"portfolioId", "maxDailyLossPct", "maxAssetWeightPct", "maxExposurePct", "dayKey", "dayStartEquity", "dayStartNetDeposits", "dayStartFlowKnown", "updatedAt",
		}, riskRows)
	}
	if pl.CustomSignals != nil {
		rows := make([][]string, 0, len(pl.CustomSignals))
		for _, p := range pl.CustomSignals {
			wb, _ := json.Marshal(p.Weights)
			def := "false"
			if p.IsDefault {
				def = "true"
			}
			rows = append(rows, []string{p.ID, p.Name, string(wb), def, p.CreatedAt, p.UpdatedAt})
		}
		if err := writeSection("customSignals", []string{"id", "name", "weights", "isDefault", "createdAt", "updatedAt"}, rows); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}

func csvRebalanceLeg(bookID, rebalanceID string, leg portfolioRebalanceLegPayload) []string {
	return []string{
		bookID, rebalanceID, leg.Side, leg.Asset, leg.Exchange, leg.Symbol, leg.Reason,
		f64(leg.PlannedQty), f64(leg.PlannedPrice), f64(leg.PlannedNotional), leg.Status, leg.TradeID,
		f64(leg.FilledQty), f64(leg.FillPrice), f64(leg.FillNotional), f64(leg.LastPrice),
		f64(leg.Fee), f64(leg.Slippage), leg.Error, i64(leg.Attempt),
	}
}

func csvRebalanceLine(bookID, rebalanceID string, ln portfolioRebalanceLinePayload) []string {
	cash := "false"
	if ln.IsCash {
		cash = "true"
	}
	return []string{
		bookID, rebalanceID, ln.Asset, ln.Exchange, ln.Symbol, cash,
		f64(ln.TargetPct), f64(ln.BeforePct), f64(ln.AfterPct),
		f64(ln.BeforeValue), f64(ln.AfterValue), f64(ln.TargetValue), f64(ln.AfterDelta), i64(ln.Attempt),
	}
}

func fmtPtr(p *float64) string {
	if p == nil {
		return ""
	}
	return strconv.FormatFloat(*p, 'f', -1, 64)
}
