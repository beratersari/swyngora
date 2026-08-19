package market

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const icebergNote = "An iceberg here means visible size at one price was eaten (or hit at the touch) and then a similar clip came back at least twice. Book-pattern only — not proof of a hidden exchange order. Informational only — not financial advice."

// GetIcebergs returns prices where visible size was eaten and then came back.
func (s *Service) GetIcebergs(ctx context.Context, exchange, symbol string, minNotional float64) (*domain.IcebergReport, error) {
	if strings.TrimSpace(symbol) == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	minNotional = domain.ParseIcebergMinNotional(minNotional)

	var venues []domain.Exchange
	rawEx := strings.ToLower(strings.TrimSpace(exchange))
	if rawEx == "" || rawEx == "all" {
		venues = s.ListExchanges()
	} else {
		ex, err := s.ResolveExchange(exchange)
		if err != nil {
			return nil, err
		}
		venues = []domain.Exchange{ex}
	}
	if len(venues) == 0 {
		return nil, fmt.Errorf("%w: no exchanges configured", domain.ErrUpstream)
	}

	if s.icebergs == nil {
		s.icebergs = domain.NewIcebergMemory()
	}
	var levels []domain.IcebergLevel
	for _, ex := range venues {
		sym := normalizeSymbolForExchange(ex, symbol)
		s.noteWallWatch(ex, sym)
		s.noteBook(ex, sym)
		p, err := s.port(ex)
		if err != nil {
			continue
		}
		raw, err := p.GetOrderBook(ctx, domain.OrderBookQuery{Symbol: sym, Limit: domain.MaxOrderBookRawLimit})
		if err != nil || raw == nil {
			continue
		}
		s.observeIcebergs(ctx, ex, sym, raw)
		levels = append(levels, s.icebergs.Active(string(ex), strings.ToUpper(sym), minNotional)...)
	}
	levels = dedupeIcebergs(levels)
	sort.SliceStable(levels, func(i, j int) bool {
		if levels[i].ClipNotional != levels[j].ClipNotional {
			return levels[i].ClipNotional > levels[j].ClipNotional
		}
		return levels[i].Refills > levels[j].Refills
	})
	exLabel := "all"
	if len(venues) == 1 {
		exLabel = string(venues[0])
	}
	asks, bids := splitIcebergs(levels)
	return &domain.IcebergReport{
		Symbol:   domain.NormalizeSymbol(domain.ExchangeBinance, symbol),
		Exchange: exLabel,
		AsOf:     time.Now().UTC(),
		Asks:     asks,
		Bids:     bids,
		Summary:  domain.ExplainIcebergs(levels, symbol),
		Note:     icebergNote,
	}, nil
}

func (s *Service) observeIcebergs(ctx context.Context, ex domain.Exchange, symbol string, raw *domain.RawOrderBook) {
	if s == nil || raw == nil {
		return
	}
	if s.icebergs == nil {
		s.icebergs = domain.NewIcebergMemory()
	}
	var prints []domain.TakerPrint
	if p := s.printPort(ex); p != nil {
		if got, err := p.GetRecentPrints(ctx, domain.NormalizeSymbol(domain.ExchangeBinance, symbol)); err == nil {
			prints = got
		}
	}
	s.icebergs.ObserveBook(time.Now().UTC(), string(ex), symbol, *raw, prints, 0)
}

func splitIcebergs(in []domain.IcebergLevel) (asks, bids []domain.IcebergLevel) {
	for _, e := range in {
		switch e.Side {
		case "ask":
			asks = append(asks, e)
		case "bid":
			bids = append(bids, e)
		}
	}
	return asks, bids
}

func dedupeIcebergs(in []domain.IcebergLevel) []domain.IcebergLevel {
	type key struct {
		ex, sym, side string
		px            int64
	}
	seen := map[key]int{}
	out := make([]domain.IcebergLevel, 0, len(in))
	for _, e := range in {
		k := key{string(e.Exchange), e.Symbol, e.Side, int64(e.Price * 100)}
		if i, ok := seen[k]; ok {
			if e.Refills > out[i].Refills {
				out[i] = e
			}
			continue
		}
		seen[k] = len(out)
		out = append(out, e)
	}
	return out
}
