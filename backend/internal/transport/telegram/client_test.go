package telegram

import (
	"errors"
	"strings"
	"testing"
)

func TestRedactSecret_StripsBotTokenFromTransportError(t *testing.T) {
	const token = "123456:LEAKME_BOT_TOKEN_SECRET"
	err := errors.New(`Get "https://api.telegram.org/bot` + token + `/getMe": context deadline exceeded`)
	got := redactSecret(err, token)
	if got == nil {
		t.Fatal("expected redacted error")
	}
	if strings.Contains(got.Error(), token) {
		t.Fatalf("token leaked: %s", got.Error())
	}
	if !strings.Contains(got.Error(), "***") {
		t.Fatalf("expected placeholder: %s", got.Error())
	}
}

func TestTelegramCommandVerb_DropsArgs(t *testing.T) {
	if got := telegramCommandVerb("/ask my key is swy_SECRET"); got != "/ask" {
		t.Fatalf("got %q", got)
	}
	if got := telegramCommandVerb("/ask@SwyngoraBot hello"); got != "/ask" {
		t.Fatalf("got %q", got)
	}
	if got := telegramCommandVerb("plain text with swy_SECRET"); got != "(text)" {
		t.Fatalf("got %q", got)
	}
}
