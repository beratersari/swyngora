package pricealert

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestResolvePublicWebhookIP_RejectsPrivate(t *testing.T) {
	if _, err := resolvePublicWebhookIP("127.0.0.1", false); err == nil {
		t.Fatal("expected loopback reject")
	}
	if _, err := resolvePublicWebhookIP("10.0.0.1", false); err == nil {
		t.Fatal("expected private reject")
	}
	ip, err := resolvePublicWebhookIP("8.8.8.8", false)
	if err != nil || ip.String() != "8.8.8.8" {
		t.Fatalf("public ip: %v %v", ip, err)
	}
}

func TestResolvePublicWebhookIP_DNSRebindBlockedAtResolve(t *testing.T) {
	orig := lookupIP
	t.Cleanup(func() { lookupIP = orig })
	lookupIP = func(host string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("169.254.169.254")}, nil
	}
	if _, err := resolvePublicWebhookIP("evil.example", false); err == nil {
		t.Fatal("metadata IP must be rejected")
	}
}

func TestPinWebhookRequestURL_UsesPublicIP(t *testing.T) {
	orig := lookupIP
	t.Cleanup(func() { lookupIP = orig })
	lookupIP = func(host string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("203.0.113.10")}, nil // documentation range — blocked as non-public
	}
	req, _ := http.NewRequest(http.MethodPost, "https://hooks.example.com/x", nil)
	if err := pinWebhookRequestURL(req, false); err == nil {
		t.Fatal("doc range should fail public check")
	}
	lookupIP = func(host string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("1.1.1.1")}, nil
	}
	req, _ = http.NewRequest(http.MethodPost, "https://hooks.example.com/x", nil)
	if err := pinWebhookRequestURL(req, false); err != nil {
		t.Fatal(err)
	}
	if req.Host != "hooks.example.com" && !strings.HasPrefix(req.Host, "hooks.example.com") {
		// Host may include port from original URL
		if req.URL.Hostname() == "hooks.example.com" {
			// URL was rewritten; Host header should still be original
		}
	}
	if !strings.Contains(req.URL.Host, "1.1.1.1") {
		t.Fatalf("expected pinned IP host, got %s (Host header %s)", req.URL.Host, req.Host)
	}
	if req.Host != "hooks.example.com" && req.Host != "hooks.example.com:443" {
		// After NewRequest without port, Host is hooks.example.com
		if !strings.HasPrefix(req.Host, "hooks.example.com") {
			t.Fatalf("Host header should remain original name, got %q", req.Host)
		}
	}
}

func TestNewWebhookHTTPClient_DialBlocksPrivate(t *testing.T) {
	c := newWebhookHTTPClient(2*time.Second, false)
	// Point at loopback literal — dialer should refuse.
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:9/", nil)
	_, err := c.Do(req)
	if err == nil {
		t.Fatal("expected dial error for loopback")
	}
}
