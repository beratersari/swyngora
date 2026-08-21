package apikey

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// Service manages named user API keys.
type Service struct {
	store domain.APIKeyPort
	now   func() time.Time
}

// New constructs an API key service.
func New(store domain.APIKeyPort) *Service {
	return &Service{store: store, now: func() time.Time { return time.Now().UTC() }}
}

// CreateInput names a new key.
type CreateInput struct {
	ClientID   string
	Name       string
	Permission string // read | trade
}

// Created is metadata plus the one-time secret.
type Created struct {
	Key    *domain.APIKey
	Secret string
}

// Create issues a new key. Secret is returned only here.
func (s *Service) Create(ctx context.Context, in CreateInput) (*Created, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: api key store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(in.ClientID)
	if err != nil {
		return nil, err
	}
	name, err := domain.ValidateAPIKeyName(in.Name)
	if err != nil {
		return nil, err
	}
	perm, err := domain.ParseAPIKeyPermission(in.Permission)
	if err != nil {
		return nil, err
	}
	n, err := s.store.CountActiveAPIKeys(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if n >= domain.MaxAPIKeysPerClient {
		return nil, fmt.Errorf("%w: at most %d active API keys per account", domain.ErrInvalidArgument, domain.MaxAPIKeysPerClient)
	}
	secret, prefix, hash, err := domain.NewAPIKeySecret()
	if err != nil {
		return nil, err
	}
	k, err := s.store.CreateAPIKey(ctx, domain.APIKey{
		ID: uuid.NewString(), ClientID: clientID, Name: name, Prefix: prefix, Hash: hash,
		Permission: perm, CreatedAt: s.now(),
	})
	if err != nil {
		return nil, err
	}
	return &Created{Key: k, Secret: secret}, nil
}

// List returns keys for the client (no secrets).
func (s *Service) List(ctx context.Context, clientID string) ([]domain.APIKey, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: api key store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	return s.store.ListAPIKeys(ctx, clientID)
}

// Revoke cancels a key. Idempotent if already revoked.
func (s *Service) Revoke(ctx context.Context, clientID, id string) (*domain.APIKey, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: api key store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: id is required", domain.ErrInvalidArgument)
	}
	return s.store.RevokeAPIKey(ctx, clientID, id, s.now())
}

// Authenticate validates a presented secret. Used by HTTP middleware.
func (s *Service) Authenticate(ctx context.Context, secret string) (*domain.APIKey, error) {
	if s.store == nil {
		return nil, domain.ErrNotFound
	}
	secret = strings.TrimSpace(secret)
	if !domain.LooksLikeUserAPIKey(secret) {
		return nil, domain.ErrNotFound
	}
	k, err := s.store.GetAPIKeyByHash(ctx, domain.HashAPIKeySecret(secret))
	if err != nil {
		return nil, err
	}
	if k.IsRevoked() {
		return nil, domain.ErrNotFound
	}
	_ = s.store.TouchAPIKeyLastUsed(ctx, k.ID, s.now())
	return k, nil
}

// DeleteByClient is used on account purge.
func (s *Service) DeleteByClient(ctx context.Context, clientID string) error {
	if s.store == nil {
		return nil
	}
	return s.store.DeleteAPIKeysByClient(ctx, clientID)
}

func normalizeClientID(id string) (string, error) {
	return domain.NormalizeClientID(id)
}
