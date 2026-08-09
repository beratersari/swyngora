package domain

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	APIKeySecretPrefix  = "swy_"
	APIKeySecretBytes   = 24
	MaxAPIKeysPerClient = 20
	MaxAPIKeyNameLen    = 64
	APIKeyDisplayPrefix = 12 // "swy_" + 8 hex
)

// APIKeyPermission is read (GET only) or trade (mutations except key/account admin).
type APIKeyPermission string

const (
	APIKeyPermissionRead  APIKeyPermission = "read"
	APIKeyPermissionTrade APIKeyPermission = "trade"
)

// ParseAPIKeyPermission accepts read|trade.
func ParseAPIKeyPermission(raw string) (APIKeyPermission, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", string(APIKeyPermissionRead):
		return APIKeyPermissionRead, nil
	case string(APIKeyPermissionTrade):
		return APIKeyPermissionTrade, nil
	default:
		return "", fmt.Errorf("%w: permission must be read or trade", ErrInvalidArgument)
	}
}

// ValidateAPIKeyName trims and checks length.
func ValidateAPIKeyName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("%w: name is required", ErrInvalidArgument)
	}
	if utf8.RuneCountInString(name) > MaxAPIKeyNameLen {
		return "", fmt.Errorf("%w: name must be at most %d characters", ErrInvalidArgument, MaxAPIKeyNameLen)
	}
	return name, nil
}

// APIKey is a named, scoped credential for one clientId. Secret is never stored.
type APIKey struct {
	ID         string
	ClientID   string
	Name       string
	Prefix     string // display only, e.g. swy_ab12cd34
	Hash       string // sha256 hex of secret
	Permission APIKeyPermission
	CreatedAt  time.Time
	LastUsedAt *time.Time
	RevokedAt  *time.Time
}

// IsRevoked reports whether the key can no longer authenticate.
func (k *APIKey) IsRevoked() bool {
	return k != nil && k.RevokedAt != nil && !k.RevokedAt.IsZero()
}

// CanTrade reports whether mutations beyond read are allowed.
func (k *APIKey) CanTrade() bool {
	return k != nil && !k.IsRevoked() && k.Permission == APIKeyPermissionTrade
}

// APIKeyPort persists user API keys.
type APIKeyPort interface {
	CreateAPIKey(ctx context.Context, k APIKey) (*APIKey, error)
	GetAPIKeyByHash(ctx context.Context, hash string) (*APIKey, error)
	GetAPIKey(ctx context.Context, clientID, id string) (*APIKey, error)
	ListAPIKeys(ctx context.Context, clientID string) ([]APIKey, error)
	CountActiveAPIKeys(ctx context.Context, clientID string) (int, error)
	RevokeAPIKey(ctx context.Context, clientID, id string, at time.Time) (*APIKey, error)
	TouchAPIKeyLastUsed(ctx context.Context, id string, at time.Time) error
	DeleteAPIKeysByClient(ctx context.Context, clientID string) error
}

// HashAPIKeySecret returns hex SHA-256 of the secret.
func HashAPIKeySecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// NewAPIKeySecret generates a high-entropy token and its prefix + hash.
func NewAPIKeySecret() (secret, prefix, hash string, err error) {
	raw := make([]byte, APIKeySecretBytes)
	if _, err = rand.Read(raw); err != nil {
		return "", "", "", fmt.Errorf("%w: generate api key: %v", ErrUpstream, err)
	}
	secret = APIKeySecretPrefix + hex.EncodeToString(raw)
	prefix = secret
	if len(prefix) > APIKeyDisplayPrefix {
		prefix = prefix[:APIKeyDisplayPrefix]
	}
	hash = HashAPIKeySecret(secret)
	return secret, prefix, hash, nil
}

// LooksLikeUserAPIKey reports whether token uses the user-key prefix.
func LooksLikeUserAPIKey(secret string) bool {
	return strings.HasPrefix(strings.TrimSpace(secret), APIKeySecretPrefix)
}
