package pricealert

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// lookupIP is overridable in tests (defaults to net.LookupIP).
var lookupIP = net.LookupIP

// validateWebhookURL parses and hardens a webhook URL against SSRF.
// When allowPrivate is false (default), loopback, RFC1918, link-local, and
// other non-global unicast destinations are rejected after DNS resolution.
func validateWebhookURL(raw string, allowPrivate bool) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("%w: webhook URL is required", domain.ErrInvalidArgument)
	}
	if len(raw) > domain.MaxWebhookURLLen {
		return "", fmt.Errorf("%w: webhook URL too long", domain.ErrInvalidArgument)
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("%w: webhook URL must be an absolute http(s) URL", domain.ErrInvalidArgument)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("%w: webhook URL scheme must be http or https", domain.ErrInvalidArgument)
	}
	// Disallow credentials in URL (often used to confuse parsers / logs).
	if u.User != nil {
		return "", fmt.Errorf("%w: webhook URL must not include userinfo", domain.ErrInvalidArgument)
	}
	host := u.Hostname()
	if host == "" {
		return "", fmt.Errorf("%w: webhook URL host is required", domain.ErrInvalidArgument)
	}
	if isBlockedHostname(host) {
		return "", fmt.Errorf("%w: webhook URL host is not allowed", domain.ErrInvalidArgument)
	}
	if !allowPrivate {
		if err := assertPublicHost(host); err != nil {
			return "", err
		}
	}
	u.Fragment = ""
	return u.String(), nil
}

// redactWebhookURL keeps scheme+host so operators can debug, but drops the
// path token (Discord/Slack put the secret in the path).
func redactWebhookURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return "(webhook)"
	}
	u.Path = "/***"
	u.RawQuery = ""
	u.Fragment = ""
	u.User = nil
	return u.String()
}

// redactWebhookErr strips the full webhook URL (and its path) from a transport error.
func redactWebhookErr(err error, rawURL string) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if raw := strings.TrimSpace(rawURL); raw != "" {
		msg = strings.ReplaceAll(msg, raw, redactWebhookURL(raw))
		if u, perr := url.Parse(raw); perr == nil && u.Path != "" && u.Path != "/" {
			msg = strings.ReplaceAll(msg, u.Path, "/***")
		}
	}
	if msg == err.Error() {
		return err
	}
	return fmt.Errorf("%s", msg)
}

func isBlockedHostname(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	switch h {
	case "localhost", "localhost.localdomain", "metadata", "metadata.google.internal",
		"metadata.goog", "instance-data", "kubernetes.default", "kubernetes.default.svc":
		return true
	}
	if strings.HasSuffix(h, ".localhost") || strings.HasSuffix(h, ".local") {
		return true
	}
	return false
}

func assertPublicHost(host string) error {
	// Literal IP (v4/v6).
	if ip := net.ParseIP(host); ip != nil {
		if !isGlobalUnicastPublic(ip) {
			return fmt.Errorf("%w: webhook URL must not target private or local addresses", domain.ErrInvalidArgument)
		}
		return nil
	}
	ips, err := lookupIP(host)
	if err != nil || len(ips) == 0 {
		return fmt.Errorf("%w: webhook URL host could not be resolved", domain.ErrInvalidArgument)
	}
	for _, ip := range ips {
		if !isGlobalUnicastPublic(ip) {
			return fmt.Errorf("%w: webhook URL must not target private or local addresses", domain.ErrInvalidArgument)
		}
	}
	return nil
}

// isGlobalUnicastPublic returns true only for IPs safe as webhook egress targets
// when private targets are disallowed (no loopback, RFC1918, link-local, CGNAT, etc.).
func isGlobalUnicastPublic(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() || ip.IsInterfaceLocalMulticast() {
		return false
	}
	// IPv4-mapped handling: use To4 when present.
	if v4 := ip.To4(); v4 != nil {
		// 0.0.0.0/8, 169.254.0.0/16 already covered; block cloud metadata explicitly.
		if v4[0] == 169 && v4[1] == 254 {
			return false
		}
		// CGNAT 100.64.0.0/10
		if v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127 {
			return false
		}
		// Benchmark 198.18.0.0/15
		if v4[0] == 198 && (v4[1] == 18 || v4[1] == 19) {
			return false
		}
		// Documentation ranges often used in tests — treat as non-public egress.
		// 192.0.2.0/24, 198.51.100.0/24, 203.0.113.0/24
		if v4[0] == 192 && v4[1] == 0 && v4[2] == 2 {
			return false
		}
		if v4[0] == 198 && v4[1] == 51 && v4[2] == 100 {
			return false
		}
		if v4[0] == 203 && v4[1] == 0 && v4[2] == 113 {
			return false
		}
		return true
	}
	// IPv6: unique local fc00::/7
	if len(ip) == net.IPv6len && (ip[0]&0xfe) == 0xfc {
		return false
	}
	return ip.IsGlobalUnicast()
}
