package portfolio

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// SituationChange is the open-position read: the newest remaining buy against the live tape.
// LiveOK is false when the live read could not be loaded. Comparison is then empty,
// so a failed read is not treated as a missing clause.
type SituationChange struct {
	Exchange   domain.Exchange
	Symbol     string
	Entry      *domain.CoinSituation
	EntryAt    time.Time
	Live       *domain.CoinSituation
	LiveOK     bool
	Comparison domain.SituationComparison
}

// SituationChange loads the entry read on the selected book and the live coin read.
// Equities return an empty comparison and do not call the futures tape.
// A flat book has no entry. A clause the entry did not store stays quiet.
func (s *Service) SituationChange(ctx context.Context, clientID, exchange, symbol string, refs ...string) (*SituationChange, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	p, err := s.requireAccessErr(ctx, clientID, domain.PortfolioRoleViewer, refs...)
	if err != nil {
		return nil, err
	}
	ex := domain.ParseExchange(exchange)
	if ex == "" {
		ex = domain.ExchangeBinance
	}
	sym := domain.NormalizeSymbol(ex, symbol)
	if sym == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	out := &SituationChange{Exchange: ex, Symbol: sym}
	if domain.IsEquityExchange(ex) {
		return out, nil
	}
	lots, err := s.store.ListTaxLots(ctx, p.BookID(), ex, sym, true)
	if err != nil {
		return nil, err
	}
	entry, at, err := s.entrySituation(ctx, p.BookID(), ex, lots)
	if err != nil {
		return nil, err
	}
	out.Entry = entry
	out.EntryAt = at
	if s.situations == nil {
		return out, nil
	}
	live, err := s.situations.GetCoinSituation(ctx, sym)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidArgument) {
			return nil, err
		}
		return out, nil
	}
	out.Live = live
	out.LiveOK = live != nil
	if out.LiveOK {
		out.Comparison = domain.CompareCoinSituation(entry, live)
	}
	return out, nil
}

// TradeSituation returns one fill plus, on a sell, the comparison with each buy it closed.
func (s *Service) TradeSituation(ctx context.Context, clientID, tradeID string, refs ...string) (*domain.Trade, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	p, err := s.requireAccessErr(ctx, clientID, domain.PortfolioRoleViewer, refs...)
	if err != nil {
		return nil, err
	}
	tradeID = strings.TrimSpace(tradeID)
	if tradeID == "" {
		return nil, fmt.Errorf("%w: tradeId is required", domain.ErrInvalidArgument)
	}
	tr, err := s.store.GetTrade(ctx, p.BookID(), tradeID)
	if err != nil {
		return nil, err
	}
	fills, err := s.store.ListTaxLotFillsForTrades(ctx, []string{tr.ID})
	if err != nil {
		return nil, err
	}
	tr.LotFills = fills
	list := []domain.Trade{*tr}
	if err := s.attachSituationLeft(ctx, p.BookID(), list); err != nil {
		return nil, err
	}
	return &list[0], nil
}

// attachSituationLeft fills Trade.Left on sells from the buys their lots closed.
// A buy that stored no read is skipped. A sell with no situation compares as a blank later read.
func (s *Service) attachSituationLeft(ctx context.Context, bookID string, list []domain.Trade) error {
	if s == nil || s.store == nil || len(list) == 0 {
		return nil
	}
	lotIDs := make([]string, 0)
	seenLot := map[string]struct{}{}
	for i := range list {
		if list[i].Side != domain.TradeSideSell {
			continue
		}
		for _, f := range list[i].LotFills {
			id := strings.TrimSpace(f.LotID)
			if id == "" {
				continue
			}
			if _, ok := seenLot[id]; ok {
				continue
			}
			seenLot[id] = struct{}{}
			lotIDs = append(lotIDs, id)
		}
	}
	if len(lotIDs) == 0 {
		return nil
	}
	lots, err := s.store.ListTaxLotsByIDs(ctx, bookID, lotIDs)
	if err != nil {
		return err
	}
	byLot := make(map[string]domain.TaxLot, len(lots))
	sourceIDs := make([]string, 0)
	seenSource := map[string]struct{}{}
	for _, lot := range lots {
		byLot[lot.ID] = lot
		id := strings.TrimSpace(lot.SourceTradeID)
		if id == "" {
			continue
		}
		if _, ok := seenSource[id]; ok {
			continue
		}
		seenSource[id] = struct{}{}
		sourceIDs = append(sourceIDs, id)
	}
	buys, err := s.store.ListTradesByIDs(ctx, bookID, sourceIDs)
	if err != nil {
		return err
	}
	byTrade := make(map[string]domain.Trade, len(buys))
	for _, tr := range buys {
		byTrade[tr.ID] = tr
	}
	for i := range list {
		if list[i].Side != domain.TradeSideSell {
			continue
		}
		seen := map[string]struct{}{}
		var exits []domain.SituationExit
		for _, f := range list[i].LotFills {
			lot, ok := byLot[f.LotID]
			if !ok {
				continue
			}
			id := strings.TrimSpace(lot.SourceTradeID)
			if id == "" {
				continue
			}
			if _, dup := seen[id]; dup {
				continue
			}
			seen[id] = struct{}{}
			buy, ok := byTrade[id]
			if !ok || buy.Situation == nil {
				continue
			}
			exits = append(exits, domain.SituationExit{
				SourceTradeID: id,
				EntryAt:       buy.CreatedAt,
				Entry:         buy.Situation,
				Comparison:    domain.CompareCoinSituation(buy.Situation, list[i].Situation),
			})
		}
		list[i].Left = exits
	}
	return nil
}
