package pricealert

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// resolvePublicWebhookIP resolves host and returns a single public IP to dial.
// When allowPrivate is true, any resolved IP is accepted (local tests).
func resolvePublicWebhookIP(host string, allowPrivate bool) (net.IP, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return nil, fmt.Errorf("empty host")
	}
	if ip := net.ParseIP(host); ip != nil {
		if !allowPrivate && !isGlobalUnicastPublic(ip) {
			return nil, fmt.Errorf("webhook URL must not target private or local addresses")
		}
		return ip, nil
	}
	ips, err := lookupIP(host)
	if err != nil || len(ips) == 0 {
		return nil, fmt.Errorf("webhook URL host could not be resolved")
	}
	for _, ip := range ips {
		if allowPrivate || isGlobalUnicastPublic(ip) {
			return ip, nil
		}
	}
	return nil, fmt.Errorf("webhook URL must not target private or local addresses")
}

// newWebhookHTTPClient returns an HTTP client that never follows redirects and,
// when allowPrivate is false, dials only IPs that pass the public-host check
// (pins connect address to block DNS rebinding between validate and dial).
func newWebhookHTTPClient(timeout time.Duration, allowPrivate bool) *http.Client {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	dialer := &net.Dialer{Timeout: timeout}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			ip, err := resolvePublicWebhookIP(host, allowPrivate)
			if err != nil {
				return nil, fmt.Errorf("webhook dial blocked: %w", err)
			}
			// Dial the validated IP only (Host/SNI remain on the request).
			return dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          4,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	c := &http.Client{Timeout: timeout, Transport: transport}
	hardenWebhookHTTPClient(c)
	return c
}

// pinWebhookRequestURL rewrites the request URL host to a public IP while
// preserving the original Host header for virtual hosting / TLS SNI via
// Go's http.Request.Host. Used when a custom client is injected without
// our dialer (tests still re-check destination at dial time when possible).
func pinWebhookRequestURL(req *http.Request, allowPrivate bool) error {
	if req == nil || req.URL == nil {
		return fmt.Errorf("nil request")
	}
	host := req.URL.Hostname()
	ip, err := resolvePublicWebhookIP(host, allowPrivate)
	if err != nil {
		return err
	}
	port := req.URL.Port()
	if port == "" {
		if req.URL.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	// Keep logical host for TLS ServerName / Host header.
	if req.Host == "" {
		req.Host = req.URL.Host
	}
	if req.URL.Scheme == "http" || req.URL.Scheme == "https" {
		req.URL.Host = net.JoinHostPort(ip.String(), port)
	}
	return nil
}

// rewriteWebhookURLForPin is a helper for tests.
func rewriteWebhookURLForPin(raw string, allowPrivate bool) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	ip, err := resolvePublicWebhookIP(u.Hostname(), allowPrivate)
	if err != nil {
		return "", err
	}
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	u.Host = net.JoinHostPort(ip.String(), port)
	return u.String(), nil
}
