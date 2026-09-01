package pricealert

import (
	"errors"
	"net"
	"strings"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestValidateWebhookURL_RejectsPrivateAndLocal(t *testing.T) {
	cases := []string{
		"http://127.0.0.1/hook",
		"http://127.0.0.1:8080/hook",
		"http://[::1]/hook",
		"http://10.0.0.5/hook",
		"http://192.168.1.1/hook",
		"http://172.16.0.1/hook",
		"http://169.254.169.254/latest/meta-data",
		"http://100.64.0.1/hook",
		"http://0.0.0.0/hook",
		"http://localhost/hook",
		"http://metadata.google.internal/",
		"ftp://example.com/x",
		"https://user:pass@example.com/x",
		"not-a-url",
		"",
	}
	for _, raw := range cases {
		_, err := validateWebhookURL(raw, false)
		if err == nil {
			t.Fatalf("expected error for %q", raw)
		}
		if !errors.Is(err, domain.ErrInvalidArgument) && raw != "not-a-url" && raw != "" {
			// parse failures also wrap invalid argument
			if !strings.Contains(err.Error(), "webhook") && !errors.Is(err, domain.ErrInvalidArgument) {
				t.Fatalf("%q: %v", raw, err)
			}
		}
	}
}

func TestValidateWebhookURL_AllowsPublicIP(t *testing.T) {
	// 8.8.8.8 is global unicast public.
	got, err := validateWebhookURL("https://8.8.8.8/hooks/alert", false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://8.8.8.8/hooks/alert" {
		t.Fatalf("got %q", got)
	}
}

func TestValidateWebhookURL_AllowPrivateOptIn(t *testing.T) {
	got, err := validateWebhookURL("http://127.0.0.1:9/hook", true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "127.0.0.1") {
		t.Fatalf("got %q", got)
	}
}

func TestValidateWebhookURL_DNSResolvesToPrivate(t *testing.T) {
	prev := lookupIP
	t.Cleanup(func() { lookupIP = prev })
	lookupIP = func(host string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("10.1.2.3")}, nil
	}
	_, err := validateWebhookURL("https://evil.example/hook", false)
	if err == nil {
		t.Fatal("expected private resolution rejection")
	}
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
}

func TestValidateWebhookURL_DNSResolvesToPublic(t *testing.T) {
	prev := lookupIP
	t.Cleanup(func() { lookupIP = prev })
	lookupIP = func(host string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("1.1.1.1")}, nil
	}
	got, err := validateWebhookURL("https://hooks.example.com/x", false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://hooks.example.com/x" {
		t.Fatalf("got %q", got)
	}
}

func TestRedactWebhookErr_StripsPathToken(t *testing.T) {
	const raw = "http://127.0.0.1:1/api/webhooks/999/DISCORD_WEBHOOK_TOKEN_LEAKME_xyz"
	err := errors.New(`Post "` + raw + `": dial tcp 127.0.0.1:1: connect: connection refused`)
	got := redactWebhookErr(err, raw)
	if got == nil {
		t.Fatal("expected redacted error")
	}
	if strings.Contains(got.Error(), "DISCORD_WEBHOOK_TOKEN_LEAKME_xyz") {
		t.Fatalf("token leaked: %s", got.Error())
	}
	if !strings.Contains(got.Error(), "***") && !strings.Contains(got.Error(), "%2A%2A%2A") {
		t.Fatalf("expected redacted path: %s", got.Error())
	}
}

func TestIsGlobalUnicastPublic(t *testing.T) {
	if isGlobalUnicastPublic(net.ParseIP("1.1.1.1")) != true {
		t.Fatal("1.1.1.1")
	}
	if isGlobalUnicastPublic(net.ParseIP("127.0.0.1")) {
		t.Fatal("loopback")
	}
	if isGlobalUnicastPublic(net.ParseIP("169.254.169.254")) {
		t.Fatal("metadata")
	}
}
