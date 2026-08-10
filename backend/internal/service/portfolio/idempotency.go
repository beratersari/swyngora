package portfolio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type idempIDs struct {
	TradeID      string `json:"tradeId,omitempty"`
	OrderID      string `json:"orderId,omitempty"`
	TakeProfitID string `json:"takeProfitId,omitempty"`
	StopLossID   string `json:"stopLossId,omitempty"`
	EntryID      string `json:"entryId,omitempty"`
	PositionID   string `json:"positionId,omitempty"`
}

func hashParts(parts ...any) string {
	s := make([]string, 0, len(parts))
	for _, p := range parts {
		switch v := p.(type) {
		case string:
			s = append(s, strings.TrimSpace(v))
		case float64:
			s = append(s, strconv.FormatFloat(v, 'g', 16, 64))
		case int:
			s = append(s, strconv.Itoa(v))
		case bool:
			if v {
				s = append(s, "1")
			} else {
				s = append(s, "0")
			}
		case *float64:
			if v == nil {
				s = append(s, "")
			} else {
				s = append(s, strconv.FormatFloat(*v, 'g', 16, 64))
			}
		default:
			s = append(s, fmt.Sprint(v))
		}
	}
	return domain.IdempotencyRequestHash(s...)
}

func (s *Service) checkIdempotency(ctx context.Context, bookID, key, hash string) (*domain.IdempotencyRecord, error) {
	if s.store == nil || key == "" {
		return nil, nil
	}
	rec, err := s.store.GetIdempotency(ctx, bookID, key)
	if err == domain.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if rec.RequestHash != hash {
		return nil, fmt.Errorf("%w: idempotency key reused with a different request", domain.ErrConflict)
	}
	return rec, nil
}

func (s *Service) withIdempotency(ctx context.Context, bookID, key, hash, kind string, ids idempIDs) context.Context {
	if key == "" {
		return ctx
	}
	raw, _ := json.Marshal(ids)
	now := time.Now().UTC()
	return domain.ContextWithIdempotency(ctx, &domain.IdempotencyRecord{
		ClientID: bookID, Key: key, RequestHash: hash, Kind: kind, ResultJSON: string(raw),
		CreatedAt: now, ExpiresAt: now.Add(domain.DefaultIdempotencyTTL),
	})
}

func (s *Service) replayAfterHit(ctx context.Context, bookID, key, hash string) (*domain.IdempotencyRecord, error) {
	rec, err := s.checkIdempotency(ctx, bookID, key, hash)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, fmt.Errorf("%w: idempotency key already used", domain.ErrConflict)
	}
	return rec, nil
}

func fillIdempCloseIDs(ctx context.Context, positionID, tradeID string) {
	rec := domain.IdempotencyFromContext(ctx)
	if rec == nil {
		return
	}
	ids := idempIDs{PositionID: positionID, TradeID: tradeID}
	raw, _ := json.Marshal(ids)
	rec.Kind = domain.IdempotencyKindMarginClose
	rec.ResultJSON = string(raw)
}

func parseIdempIDs(raw string) idempIDs {
	var ids idempIDs
	_ = json.Unmarshal([]byte(raw), &ids)
	return ids
}

func (s *Service) replayTrade(ctx context.Context, rec *domain.IdempotencyRecord, actor, portfolioID string) (*domain.Trade, *domain.PortfolioView, error) {
	ids := parseIdempIDs(rec.ResultJSON)
	tr, err := s.store.GetTrade(ctx, rec.ClientID, ids.TradeID)
	if err != nil {
		return nil, nil, err
	}
	view, err := s.View(ctx, actor, portfolioID)
	if err != nil {
		return tr, nil, err
	}
	return tr, view, nil
}

func (s *Service) replayPending(ctx context.Context, rec *domain.IdempotencyRecord) (*domain.PendingOrder, error) {
	ids := parseIdempIDs(rec.ResultJSON)
	return s.store.GetPendingOrder(ctx, rec.ClientID, ids.OrderID)
}

func (s *Service) replayOCO(ctx context.Context, rec *domain.IdempotencyRecord) (*domain.PendingOrder, *domain.PendingOrder, error) {
	ids := parseIdempIDs(rec.ResultJSON)
	tp, err := s.store.GetPendingOrder(ctx, rec.ClientID, ids.TakeProfitID)
	if err != nil {
		return nil, nil, err
	}
	sl, err := s.store.GetPendingOrder(ctx, rec.ClientID, ids.StopLossID)
	return tp, sl, err
}

func (s *Service) replayBracket(ctx context.Context, rec *domain.IdempotencyRecord) (*domain.PendingOrder, *domain.PendingOrder, *domain.PendingOrder, error) {
	ids := parseIdempIDs(rec.ResultJSON)
	entry, err := s.store.GetPendingOrder(ctx, rec.ClientID, ids.EntryID)
	if err != nil {
		return nil, nil, nil, err
	}
	tp, err := s.store.GetPendingOrder(ctx, rec.ClientID, ids.TakeProfitID)
	if err != nil {
		return entry, nil, nil, err
	}
	sl, err := s.store.GetPendingOrder(ctx, rec.ClientID, ids.StopLossID)
	return entry, tp, sl, err
}

func (s *Service) replayMarginOpen(ctx context.Context, rec *domain.IdempotencyRecord) (*domain.MarginPosition, *domain.MarginOrder, error) {
	ids := parseIdempIDs(rec.ResultJSON)
	if ids.PositionID != "" {
		pos, err := s.store.GetMarginPosition(ctx, rec.ClientID, ids.PositionID)
		return pos, nil, err
	}
	if ids.OrderID != "" {
		o, err := s.store.GetMarginOrder(ctx, rec.ClientID, ids.OrderID)
		return nil, o, err
	}
	return nil, nil, domain.ErrNotFound
}

func (s *Service) replayMarginClose(ctx context.Context, rec *domain.IdempotencyRecord) (*domain.MarginPosition, *domain.MarginTrade, error) {
	ids := parseIdempIDs(rec.ResultJSON)
	pos, err := s.store.GetMarginPosition(ctx, rec.ClientID, ids.PositionID)
	if err != nil {
		return nil, nil, err
	}
	tr, err := s.store.GetMarginTrade(ctx, rec.ClientID, ids.TradeID)
	return pos, tr, err
}

func isIdempotencyHit(err error) bool {
	return err != nil && (errors.Is(err, domain.ErrIdempotencyHit) || strings.Contains(err.Error(), domain.ErrIdempotencyHit.Error()))
}
