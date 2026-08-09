package domain

import (
	"strings"
	"testing"
	"time"
)

func TestParseAPIKeyPermission(t *testing.T) {
	p, err := ParseAPIKeyPermission("trade")
	if err != nil || p != APIKeyPermissionTrade {
		t.Fatalf("%s %v", p, err)
	}
	p, err = ParseAPIKeyPermission("")
	if err != nil || p != APIKeyPermissionRead {
		t.Fatalf("default %s %v", p, err)
	}
	if _, err := ParseAPIKeyPermission("admin"); err == nil {
		t.Fatal("want error")
	}
}

func TestNewAPIKeySecret_HashRoundTrip(t *testing.T) {
	sec, prefix, hash, err := NewAPIKeySecret()
	if err != nil || !strings.HasPrefix(sec, APIKeySecretPrefix) || !strings.HasPrefix(prefix, APIKeySecretPrefix) {
		t.Fatalf("%s %s %v", sec, prefix, err)
	}
	if HashAPIKeySecret(sec) != hash {
		t.Fatal("hash mismatch")
	}
	if !LooksLikeUserAPIKey(sec) || LooksLikeUserAPIKey("master-token") {
		t.Fatal("prefix detect")
	}
}

func TestAPIKey_RevokedCannotTrade(t *testing.T) {
	now := time.Now().UTC()
	k := &APIKey{Permission: APIKeyPermissionTrade, RevokedAt: &now}
	if k.CanTrade() || !k.IsRevoked() {
		t.Fatal("revoked")
	}
}
