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
	WatchlistItems []domain.WatchlistItem  `json:"watchlistItems"`
	Shares         []domain.WatchlistShare `json:"shares"`
	Alerts         []domain.PriceAlert     `json:"alerts"`
	Backtests      []backtestBundle        `json:"backtests"`
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
		Condition      string  `json:"condition"`
		TargetPrice    float64 `json:"targetPrice"`
		Mode           string  `json:"mode"`
		Status         string  `json:"status"`
		CreatedAt      string  `json:"createdAt"`
		TriggeredAt    *string `json:"triggeredAt"`
		TriggeredPrice float64 `json:"triggeredPrice"`
	} `json:"alerts"`
	Backtests []struct {
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
		al, err := normalizeAlert(importer, a.ID, a.Exchange, a.Symbol, a.Condition, a.TargetPrice, a.Mode, a.Status, a.CreatedAt, a.TriggeredAt, a.TriggeredPrice)
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
			al, err := normalizeAlert(importer, row["id"], row["exchange"], row["symbol"], row["condition"],
				tp, row["mode"], row["status"], row["createdAt"], trigAt, trigP)
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
		}
	}
	for id, b := range backtestsByID {
		b.Signals = signalsByBT[id]
		res.Payload.Backtests = append(res.Payload.Backtests, *b)
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

func normalizeAlert(clientID, id, exchange, symbol, condition string, target float64, mode, status, createdAt string, triggeredAt *string, triggeredPrice float64) (domain.PriceAlert, error) {
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
	cond := domain.AlertCondition(strings.ToLower(strings.TrimSpace(condition)))
	if !domain.IsValidAlertCondition(string(cond)) {
		return domain.PriceAlert{}, fmt.Errorf("bad condition")
	}
	if target <= 0 {
		return domain.PriceAlert{}, fmt.Errorf("bad target")
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
	return domain.PriceAlert{
		ID: id, ClientID: clientID, Exchange: ex, Symbol: sym, Condition: cond,
		TargetPrice: target, Mode: m, Status: st, CreatedAt: cAt,
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
