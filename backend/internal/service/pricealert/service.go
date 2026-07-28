package pricealert

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const maxClientIDLen = 128

// CreateInput is the application input for creating a price alert.
type CreateInput struct {
	ClientID    string
	Exchange    string
	Symbol      string
	Condition   string // above | below
	TargetPrice float64
}

// Service orchestrates price-alert use cases.
type Service struct {
	store domain.PriceAlertPort
}

// New constructs a price-alert service.
func New(store domain.PriceAlertPort) *Service {
	return &Service{store: store}
}

// Create validates input and persists a new active alert.
func (s *Service) Create(ctx context.Context, in CreateInput) (*domain.PriceAlert, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: alert store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(in.ClientID)
	if err != nil {
		return nil, err
	}
	ex, sym, err := normalizeExchangeSymbol(in.Exchange, in.Symbol)
	if err != nil {
		return nil, err
	}
	cond := domain.AlertCondition(strings.ToLower(strings.TrimSpace(in.Condition)))
	if !domain.IsValidAlertCondition(string(cond)) {
		return nil, fmt.Errorf("%w: condition must be above or below", domain.ErrInvalidArgument)
	}
	if in.TargetPrice <= 0 || math.IsNaN(in.TargetPrice) || math.IsInf(in.TargetPrice, 0) {
		return nil, fmt.Errorf("%w: targetPrice must be a positive number", domain.ErrInvalidArgument)
	}
	n, err := s.store.CountByClient(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if n >= domain.MaxPriceAlertsPerClient {
		return nil, fmt.Errorf("%w: max %d alerts per client", domain.ErrInvalidArgument, domain.MaxPriceAlertsPerClient)
	}
	alert := domain.PriceAlert{
		ID:          uuid.NewString(),
		ClientID:    clientID,
		Exchange:    ex,
		Symbol:      sym,
		Condition:   cond,
		TargetPrice: in.TargetPrice,
		Status:      domain.AlertStatusActive,
		CreatedAt:   time.Now().UTC(),
	}
	return s.store.Create(ctx, alert)
}

// Get returns one alert for the client.
func (s *Service) Get(ctx context.Context, clientID, id string) (*domain.PriceAlert, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: alert store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: alert id is required", domain.ErrInvalidArgument)
	}
	return s.store.Get(ctx, clientID, id)
}

// List returns all alerts for a client.
func (s *Service) List(ctx context.Context, clientID string) ([]domain.PriceAlert, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: alert store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	return s.store.ListByClient(ctx, clientID)
}

// Delete removes an alert owned by the client.
func (s *Service) Delete(ctx context.Context, clientID, id string) error {
	if s.store == nil {
		return fmt.Errorf("%w: alert store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("%w: alert id is required", domain.ErrInvalidArgument)
	}
	return s.store.Delete(ctx, clientID, id)
}

// ListActive returns active alerts for the background checker.
func (s *Service) ListActive(ctx context.Context) ([]domain.PriceAlert, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: alert store not configured", domain.ErrUpstream)
	}
	return s.store.ListActive(ctx)
}

// MarkTriggered records a one-shot trigger.
func (s *Service) MarkTriggered(ctx context.Context, id string, price float64, at time.Time) (*domain.PriceAlert, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: alert store not configured", domain.ErrUpstream)
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: alert id is required", domain.ErrInvalidArgument)
	}
	return s.store.MarkTriggered(ctx, id, price, at)
}

func normalizeClientID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("%w: clientId is required", domain.ErrInvalidArgument)
	}
	if len(id) > maxClientIDLen {
		return "", fmt.Errorf("%w: clientId too long", domain.ErrInvalidArgument)
	}
	if strings.EqualFold(id, "default") {
		return "", fmt.Errorf("%w: clientId must not be the shared name \"default\"", domain.ErrInvalidArgument)
	}
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return "", fmt.Errorf("%w: clientId has invalid characters", domain.ErrInvalidArgument)
	}
	return id, nil
}

func normalizeExchangeSymbol(exchange, symbol string) (domain.Exchange, string, error) {
	rawEx := strings.TrimSpace(exchange)
	var ex domain.Exchange
	if rawEx == "" {
		ex = domain.DefaultExchange
	} else {
		if !domain.IsValidExchange(rawEx) {
			return "", "", fmt.Errorf("%w: exchange must be one of %v", domain.ErrInvalidArgument, domain.SupportedExchanges)
		}
		ex = domain.ParseExchange(rawEx)
	}
	sym := domain.NormalizeSymbol(ex, symbol)
	if sym == "" {
		return "", "", fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	return ex, sym, nil
}