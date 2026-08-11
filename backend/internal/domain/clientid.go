package domain

import (
	"context"
	"fmt"
	"strings"
	"unicode"
)

// MaxClientIDLen is the hard cap on opaque tenant keys.
const MaxClientIDLen = 128

// ReservedClientIDs are shared/guessable names that must never persist as tenants.
var ReservedClientIDs = []string{
	"default",
	"anonymous",
	"http-default",
	"ai-assistant",
}

// NormalizeClientID trims and validates an opaque tenant key.
// Empty, oversized, reserved, and enumerable Telegram-style ids (tg-<digits>) are rejected.
func NormalizeClientID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("%w: clientId is required", ErrInvalidArgument)
	}
	if len(id) > MaxClientIDLen {
		return "", fmt.Errorf("%w: clientId too long", ErrInvalidArgument)
	}
	for _, reserved := range ReservedClientIDs {
		if strings.EqualFold(id, reserved) {
			return "", fmt.Errorf("%w: clientId must not be the shared name %q", ErrInvalidArgument, reserved)
		}
	}
	if isEnumerableTelegramClientID(id) {
		return "", fmt.Errorf("%w: clientId must not be an enumerable telegram user id", ErrInvalidArgument)
	}
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return "", fmt.Errorf("%w: clientId has invalid characters", ErrInvalidArgument)
	}
	return id, nil
}

func isEnumerableTelegramClientID(id string) bool {
	if len(id) < 4 || !strings.HasPrefix(strings.ToLower(id), "tg-") {
		return false
	}
	rest := id[3:]
	if rest == "" {
		return false
	}
	for _, r := range rest {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// TelegramIdentityPort maps Telegram numeric user ids to unguessable clientIds.
type TelegramIdentityPort interface {
	ClientIDForTelegramUser(ctx context.Context, telegramUserID int64) (string, error)
}
