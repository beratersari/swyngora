package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds process configuration from the environment.
type Config struct {
	HTTPAddr              string
	BinanceBaseURL        string
	BinanceProductBaseURL string
	CoinbaseBaseURL       string
	CoinbaseExchangeURL   string
	BybitBaseURL          string
	HTTPClientTimeout     time.Duration
	CandleCacheTTL        time.Duration
	CandleCacheMaxEntries int
	TickerCacheTTL        time.Duration
	SupplyCacheTTL        time.Duration
	CacheCleanupEvery     time.Duration
	SpotMarketCacheTTL    time.Duration

	// Daily supply snapshot refresh (Binance product catalog). User requests read cache only.
	SupplyRefreshHour      int
	SupplyRefreshMinute    int
	SupplyRefreshLocation  *time.Location
	SupplyRefreshOnStartup bool

	// Ingress rate limit (per client IP). 0 disables.
	RateLimitRPS   float64
	RateLimitBurst int

	// CORSAllowOrigins: empty or ["*"] allows any origin (local dev default).
	// Set comma-separated exact origins for production, e.g. https://app.example.com
	CORSAllowOrigins []string

	// Telegram bot (optional). Empty token disables the bot.
	TelegramBotToken        string
	TelegramAllowedChats    map[int64]struct{}
	// TelegramAllowAll permits any chat when the allowlist is empty.
	// Default false: token without allowlist refuses to start the bot (fail closed).
	TelegramAllowAll        bool
	TelegramDefaultExchange string
	TelegramPollTimeout     time.Duration
	TelegramLowMcapLimit    int

	// AI multi-agent service (Python). Telegram /ask and POST /api/v1/ai/chat.
	AIServiceURL   string
	AITimeout      time.Duration
	AIAutoStart    bool
	AIPython       string // interpreter for auto-start, e.g. ai/.venv/bin/python
	AIWorkDir      string // cwd for auto-start (repo ai/ package)
	AIListenHost   string
	AIListenPort   int

	// WatchlistDBPath is the SQLite file for durable watchlists (survives restarts).
	// Relative paths are resolved from the process working directory.
	WatchlistDBPath string

	// AlertsDBPath is the SQLite file for durable price alerts (survives restarts).
	AlertsDBPath string
	// AlertCheckInterval is how often active alerts are evaluated against last price.
	AlertCheckInterval time.Duration
	// WebhookDeliveryInterval is how often the outbox deliverer drains pending notifications.
	WebhookDeliveryInterval time.Duration
	// WebhookHTTPTimeout is the per-delivery HTTP client timeout.
	WebhookHTTPTimeout time.Duration
	// WebhookMaxAttempts is permanent failure threshold for webhook deliveries.
	WebhookMaxAttempts int

	// PortfolioDBPath is the SQLite file for paper-trading portfolios.
	PortfolioDBPath string
	// PortfolioOrderCheckInterval is how often open pending paper orders are evaluated.
	PortfolioOrderCheckInterval time.Duration

	// ScannerDBPath is the SQLite file for technical indicator scanner rules/results.
	ScannerDBPath string
	// ScannerCheckInterval is how often scanner rules are evaluated against watchlists.
	ScannerCheckInterval time.Duration
}

// Load reads configuration from environment variables with safe defaults.
// Invalid or non-positive durations fall back to defaults (never zero/negative).
func Load() Config {
	loc := loadLocation(getenv("SUPPLY_REFRESH_TZ", "UTC"))
	return Config{
		HTTPAddr:              getenv("HTTP_ADDR", ":8080"),
		BinanceBaseURL:        getenv("BINANCE_BASE_URL", "https://api.binance.com"),
		BinanceProductBaseURL: getenv("BINANCE_PRODUCT_BASE_URL", "https://www.binance.com"),
		CoinbaseBaseURL:       getenv("COINBASE_BASE_URL", "https://api.coinbase.com"),
		CoinbaseExchangeURL:   getenv("COINBASE_EXCHANGE_URL", "https://api.exchange.coinbase.com"),
		BybitBaseURL:          getenv("BYBIT_BASE_URL", "https://api.bybit.com"),
		HTTPClientTimeout:     positiveDurationEnv("HTTP_CLIENT_TIMEOUT", 15*time.Second),
		CandleCacheTTL:        positiveDurationEnv("CANDLE_CACHE_TTL", 30*time.Second),
		CandleCacheMaxEntries: positiveIntEnv("CANDLE_CACHE_MAX_ENTRIES", 512),
		TickerCacheTTL:        positiveDurationEnv("TICKER_CACHE_TTL", 15*time.Second),
		// Safety TTL so supply/mcap cannot stay forever after failed refreshes.
		// Successful daily ReplaceAll resets expiry. 0 = never expire (opt-in).
		SupplyCacheTTL:     durationEnvAllowZero("SUPPLY_CACHE_TTL", 48*time.Hour),
		CacheCleanupEvery:  positiveDurationEnv("CACHE_CLEANUP_EVERY", 1*time.Minute),
		SpotMarketCacheTTL: positiveDurationEnv("SPOT_MARKET_CACHE_TTL", 5*time.Second),

		SupplyRefreshHour:      intEnvClamp("SUPPLY_REFRESH_HOUR", 3, 0, 23),
		SupplyRefreshMinute:    intEnvClamp("SUPPLY_REFRESH_MINUTE", 0, 0, 59),
		SupplyRefreshLocation:  loc,
		SupplyRefreshOnStartup: boolEnv("SUPPLY_REFRESH_ON_STARTUP", true),

		RateLimitRPS:   floatEnv("RATE_LIMIT_RPS", 40),
		RateLimitBurst: positiveIntEnv("RATE_LIMIT_BURST", 80),

		CORSAllowOrigins: parseCSVList(getenv("CORS_ALLOW_ORIGINS", "*")),

		TelegramBotToken:        strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN")),
		TelegramAllowedChats:    parseTelegramChatIDs(),
		TelegramAllowAll:        boolEnv("TELEGRAM_ALLOW_ALL", false),
		TelegramDefaultExchange: strings.ToLower(strings.TrimSpace(getenv("BOT_DEFAULT_EXCHANGE", "binance"))),
		TelegramPollTimeout:     positiveDurationEnv("BOT_POLL_TIMEOUT", 30*time.Second),
		TelegramLowMcapLimit:    clampInt(positiveIntEnv("BOT_LOWMCAP_LIMIT", 10), 1, 25),

		AIServiceURL: getenv("AI_SERVICE_URL", "http://127.0.0.1:8090"),
		// Multi-agent /ask (market + web + X) often needs >2 minutes.
		AITimeout: positiveDurationEnv("AI_TIMEOUT", 300*time.Second),
		// Auto-start Python AI with the backend when true (default true if AI_PYTHON is set, else false).
		AIAutoStart:  boolEnv("AI_AUTOSTART", false),
		AIPython:     strings.TrimSpace(os.Getenv("AI_PYTHON")),
		AIWorkDir:    strings.TrimSpace(getenv("AI_WORKDIR", "ai")),
		AIListenHost: getenv("AI_LISTEN_HOST", "127.0.0.1"),
		AIListenPort: positiveIntEnv("AI_LISTEN_PORT", 8090),

		// Durable watchlist storage (SQLite). Default relative to process cwd.
		WatchlistDBPath: getenv("WATCHLIST_DB_PATH", "data/watchlist.db"),

		// Durable price alerts + background check cadence.
		AlertsDBPath:       getenv("ALERTS_DB_PATH", "data/alerts.db"),
		AlertCheckInterval: positiveDurationEnv("ALERT_CHECK_INTERVAL", 30*time.Second),

		WebhookDeliveryInterval: positiveDurationEnv("WEBHOOK_DELIVERY_INTERVAL", 5*time.Second),
		WebhookHTTPTimeout:      positiveDurationEnv("WEBHOOK_HTTP_TIMEOUT", 10*time.Second),
		WebhookMaxAttempts:      positiveIntEnv("WEBHOOK_MAX_ATTEMPTS", 8),

		PortfolioDBPath:             getenv("PORTFOLIO_DB_PATH", "data/portfolio.db"),
		PortfolioOrderCheckInterval: positiveDurationEnv("PORTFOLIO_ORDER_CHECK_INTERVAL", 15*time.Second),

		ScannerDBPath:        getenv("SCANNER_DB_PATH", "data/scanner.db"),
		ScannerCheckInterval: positiveDurationEnv("SCANNER_CHECK_INTERVAL", 60*time.Second),
	}
}

// parseCSVList splits a comma-separated env value; empty → nil.
func parseCSVList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// positiveDurationEnv parses a duration; invalid or <=0 values fall back to def.
func positiveDurationEnv(key string, def time.Duration) time.Duration {
	d := parseDuration(os.Getenv(key), def)
	if d <= 0 {
		return def
	}
	return d
}

// durationEnvAllowZero allows 0 (never-expire). Negative/invalid → def.
func durationEnvAllowZero(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d := parseDuration(v, def)
	if d < 0 {
		return def
	}
	return d
}

func parseDuration(v string, def time.Duration) time.Duration {
	if v == "" {
		return def
	}
	if d, err := time.ParseDuration(v); err == nil {
		return d
	}
	if sec, err := strconv.Atoi(v); err == nil {
		return time.Duration(sec) * time.Second
	}
	return def
}

func intEnvClamp(key string, def, min, max int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}

func positiveIntEnv(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

func floatEnv(key string, def float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f < 0 {
		return def
	}
	return f
}

func boolEnv(key string, def bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return def
	}
	switch v {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return def
	}
}

func loadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return loc
}

func parseTelegramChatIDs() map[int64]struct{} {
	out := map[int64]struct{}{}
	add := func(raw string) {
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			id, err := strconv.ParseInt(part, 10, 64)
			if err != nil {
				continue
			}
			out[id] = struct{}{}
		}
	}
	add(os.Getenv("TELEGRAM_ALLOWED_CHAT_IDS"))
	add(os.Getenv("TELEGRAM_CHAT_ID"))
	if len(out) == 0 {
		return nil
	}
	return out
}

func clampInt(n, min, max int) int {
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}

// LoadDotEnv loads KEY=VALUE from path without overriding existing env vars.
func LoadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		i := strings.IndexByte(line, '=')
		if i <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		val := strings.TrimSpace(line[i+1:])
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
}
