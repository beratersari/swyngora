package config

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds process configuration from the environment.
type Config struct {
	HTTPAddr               string
	BinanceBaseURL         string
	BinanceProductBaseURL  string
	CMCBaseURL             string
	BinanceAPIKey          string
	BinanceWSURL           string
	BinanceFuturesWSURL    string
	BinanceFuturesBaseURL  string
	OpenInterestCacheTTL   time.Duration
	OrderBookIdleTTL       time.Duration
	OrderBookSyncTimeout   time.Duration
	DelistRefreshEvery     time.Duration
	DelistRefreshOnStartup bool
	CoinbaseBaseURL        string
	CoinbaseExchangeURL    string
	CoinbaseWSURL          string
	BybitBaseURL           string
	BybitWSURL             string
	BybitLinearWSURL       string
	HTTPClientTimeout      time.Duration
	CandleCacheTTL         time.Duration
	CandleCacheMaxEntries  int
	TickerCacheTTL         time.Duration
	OrderBookCacheTTL      time.Duration
	SupplyCacheTTL         time.Duration
	HoldersCacheTTL        time.Duration
	CacheCleanupEvery      time.Duration
	SpotMarketCacheTTL     time.Duration

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
	TelegramBotToken     string
	TelegramAllowedChats map[int64]struct{}
	// TelegramAllowAll permits any chat when the allowlist is empty.
	// Default false: token without allowlist refuses to start the bot (fail closed).
	TelegramAllowAll        bool
	TelegramDefaultExchange string
	TelegramPollTimeout     time.Duration
	TelegramLowMcapLimit    int

	// AI multi-agent service (Python). Telegram /ask and POST /api/v1/ai/chat.
	AIServiceURL string
	AITimeout    time.Duration
	AIAutoStart  bool
	AIPython     string // interpreter for auto-start, e.g. ai/.venv/bin/python
	AIWorkDir    string // cwd for auto-start (repo ai/ package)
	AIListenHost string
	AIListenPort int
	// AIServiceToken, when set, is sent as Authorization: Bearer to the Python AI HTTP API.
	// Empty = Python accepts unauthenticated localhost (dev).
	AIServiceToken string

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

	// RealtimePriceInterval is how often subscribed tickers are pushed on the WebSocket.
	RealtimePriceInterval time.Duration

	// PortfolioDBPath is the SQLite file for paper-trading portfolios.
	PortfolioDBPath string
	// PortfolioOrderCheckInterval is how often open pending paper orders are evaluated.
	PortfolioOrderCheckInterval time.Duration
	// RecurringBuyInterval is how often due recurring buy plans are evaluated.
	RecurringBuyInterval time.Duration
	// PortfolioSnapshotInterval is how often equity is sampled for performance charts.
	PortfolioSnapshotInterval time.Duration
	// PortfolioSnapshotRetention is how long equity samples are kept (must cover 3m).
	PortfolioSnapshotRetention time.Duration
	// MarginInterestInterval is how often open margin debts are interest-accrued.
	// Catch-up after downtime is O(1) per position (not hour-by-hour).
	MarginInterestInterval time.Duration

	// ScannerDBPath is the SQLite file for technical indicator scanner rules/results.
	ScannerDBPath string
	// ScannerCheckInterval is how often scanner rules are evaluated against watchlists.
	ScannerCheckInterval time.Duration

	// PriceDiffDBPath is the SQLite file for cross-exchange price difference watches/opportunities.
	PriceDiffDBPath string
	// PriceDiffCheckInterval is how often active price-diff watches are evaluated.
	PriceDiffCheckInterval time.Duration

	// FundingArbDBPath is the SQLite file for funding-arb follow watches.
	FundingArbDBPath string
	// FundingArbCheckInterval is how often active funding-arb watches are evaluated.
	FundingArbCheckInterval time.Duration

	// ExportDBPath is the SQLite file for user data export job metadata.
	ExportDBPath string
	// ExportFileDir is the directory where export download files are written.
	ExportFileDir string
	// ExportFileTTL is how long completed export files remain downloadable.
	ExportFileTTL time.Duration
	// ExportWorkerInterval is how often the export worker polls for pending jobs.
	ExportWorkerInterval time.Duration

	// ImportDBPath is the SQLite file for user data import job metadata.
	ImportDBPath string
	// ImportFileDir is the directory for uploaded import files and payloads.
	ImportFileDir string
	// ImportFileTTL is how long preview/source files remain before cleanup.
	ImportFileTTL time.Duration
	// ImportWorkerInterval is how often the import worker polls for pending jobs.
	ImportWorkerInterval time.Duration

	// AccountDBPath is the SQLite file for account close/reopen state.
	AccountDBPath string
	// AccountPurgeInterval is how often closed accounts past grace are purged.
	AccountPurgeInterval time.Duration

	// FuturesHistoryDBPath is the SQLite file for durable OI/funding/LS/liquidation history.
	FuturesHistoryDBPath string
	// FuturesHistoryInterval is how often snapshots are sampled.
	FuturesHistoryInterval time.Duration
	// FuturesHistoryRetention is how long samples and events are kept.
	FuturesHistoryRetention time.Duration
	// FuturesHistorySymbols extra pairs to always sample (CSV). Seeds are always included.
	FuturesHistorySymbols []string

	// OrderBookHistoryDBPath is the SQLite file for durable spot book samples.
	OrderBookHistoryDBPath string
	// OrderBookHistoryInterval is how often books are sampled.
	OrderBookHistoryInterval time.Duration
	// OrderBookHistoryRetention is how long book samples are kept.
	OrderBookHistoryRetention time.Duration
	// OrderBookHistorySymbols extra pairs to always sample (CSV). Seeds are always included.
	OrderBookHistorySymbols []string

	// APIAuthToken, when non-empty, requires Authorization: Bearer or X-API-Key on
	// tenant routes (watchlist/alerts/portfolio/scanner/AI) and /mcp. Market GETs stay public.
	// Empty = open local-dev mode (not multi-tenant safe).
	APIAuthToken string
	// AllowOpenAuth permits empty APIAuthToken when HTTPAddr is not loopback.
	// Default false — refuse open auth on 0.0.0.0 / LAN binds.
	AllowOpenAuth bool
	// AllowMasterImpersonate lets a remote client use the process master token
	// as any X-Client-Id. Default false — only loopback (local Vite, AI HTTP
	// tools) may impersonate. User keys stay bound to their client.
	AllowMasterImpersonate bool
	// MCPEnabled mounts streamable MCP at /mcp (default true). Set false to disable the agent surface.
	MCPEnabled bool
	// WebhookAllowPrivate permits loopback/RFC1918 webhook targets (local tests only). Default false (SSRF-safe).
	WebhookAllowPrivate bool
}

// Load reads configuration from environment variables with safe defaults.
// Invalid or non-positive durations fall back to defaults (never zero/negative).
func Load() Config {
	loc := loadLocation(getenv("SUPPLY_REFRESH_TZ", "UTC"))
	return Config{
		// Loopback by default so empty API_AUTH_TOKEN open mode is not LAN-wide.
		HTTPAddr:               getenv("HTTP_ADDR", "127.0.0.1:8080"),
		BinanceBaseURL:         getenv("BINANCE_BASE_URL", "https://api.binance.com"),
		BinanceProductBaseURL:  getenv("BINANCE_PRODUCT_BASE_URL", "https://www.binance.com"),
		CMCBaseURL:             getenv("CMC_BASE_URL", "https://api.coinmarketcap.com"),
		BinanceAPIKey:          strings.TrimSpace(os.Getenv("BINANCE_API_KEY")),
		BinanceWSURL:           getenv("BINANCE_WS_URL", "wss://stream.binance.com:9443"),
		BinanceFuturesWSURL:    getenv("BINANCE_FUTURES_WS_URL", "wss://fstream.binance.com"),
		BinanceFuturesBaseURL:  getenv("BINANCE_FUTURES_BASE_URL", "https://fapi.binance.com"),
		OpenInterestCacheTTL:   positiveDurationEnv("OPEN_INTEREST_CACHE_TTL", 30*time.Second),
		OrderBookIdleTTL:       positiveDurationEnv("ORDERBOOK_IDLE_TTL", 90*time.Second),
		OrderBookSyncTimeout:   positiveDurationEnv("ORDERBOOK_SYNC_TIMEOUT", 8*time.Second),
		DelistRefreshEvery:     positiveDurationEnv("DELIST_REFRESH_EVERY", time.Hour),
		DelistRefreshOnStartup: boolEnv("DELIST_REFRESH_ON_STARTUP", true),
		CoinbaseBaseURL:        getenv("COINBASE_BASE_URL", "https://api.coinbase.com"),
		CoinbaseExchangeURL:    getenv("COINBASE_EXCHANGE_URL", "https://api.exchange.coinbase.com"),
		CoinbaseWSURL:          getenv("COINBASE_WS_URL", "wss://ws-feed.exchange.coinbase.com"),
		BybitBaseURL:           getenv("BYBIT_BASE_URL", "https://api.bybit.com"),
		BybitWSURL:             getenv("BYBIT_WS_URL", "wss://stream.bybit.com/v5/public/spot"),
		BybitLinearWSURL:       getenv("BYBIT_LINEAR_WS_URL", "wss://stream.bybit.com/v5/public/linear"),
		HTTPClientTimeout:      positiveDurationEnv("HTTP_CLIENT_TIMEOUT", 15*time.Second),
		CandleCacheTTL:         positiveDurationEnv("CANDLE_CACHE_TTL", 30*time.Second),
		CandleCacheMaxEntries:  positiveIntEnv("CANDLE_CACHE_MAX_ENTRIES", 512),
		TickerCacheTTL:         positiveDurationEnv("TICKER_CACHE_TTL", 15*time.Second),
		OrderBookCacheTTL:      positiveDurationEnv("ORDERBOOK_CACHE_TTL", 2*time.Second),
		// Safety TTL so supply/mcap cannot stay forever after failed refreshes.
		// Successful daily ReplaceAll resets expiry. 0 = never expire (opt-in).
		SupplyCacheTTL:     durationEnvAllowZero("SUPPLY_CACHE_TTL", 48*time.Hour),
		HoldersCacheTTL:    positiveDurationEnv("HOLDERS_CACHE_TTL", time.Hour),
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
		AIAutoStart:    boolEnv("AI_AUTOSTART", false),
		AIPython:       strings.TrimSpace(os.Getenv("AI_PYTHON")),
		AIWorkDir:      strings.TrimSpace(getenv("AI_WORKDIR", "ai")),
		AIListenHost:   getenv("AI_LISTEN_HOST", "127.0.0.1"),
		AIListenPort:   positiveIntEnv("AI_LISTEN_PORT", 8090),
		AIServiceToken: strings.TrimSpace(os.Getenv("AI_SERVICE_TOKEN")),

		// Durable watchlist storage (SQLite). Default relative to process cwd.
		WatchlistDBPath: getenv("WATCHLIST_DB_PATH", "data/watchlist.db"),

		// Durable price alerts + background check cadence.
		AlertsDBPath:       getenv("ALERTS_DB_PATH", "data/alerts.db"),
		AlertCheckInterval: positiveDurationEnv("ALERT_CHECK_INTERVAL", 30*time.Second),

		WebhookDeliveryInterval: positiveDurationEnv("WEBHOOK_DELIVERY_INTERVAL", 5*time.Second),
		WebhookHTTPTimeout:      positiveDurationEnv("WEBHOOK_HTTP_TIMEOUT", 10*time.Second),
		WebhookMaxAttempts:      positiveIntEnv("WEBHOOK_MAX_ATTEMPTS", 8),

		RealtimePriceInterval: positiveDurationEnv("REALTIME_PRICE_INTERVAL", 5*time.Second),

		PortfolioDBPath:             getenv("PORTFOLIO_DB_PATH", "data/portfolio.db"),
		PortfolioOrderCheckInterval: positiveDurationEnv("PORTFOLIO_ORDER_CHECK_INTERVAL", 15*time.Second),
		RecurringBuyInterval:        positiveDurationEnv("RECURRING_BUY_INTERVAL", 30*time.Second),
		PortfolioSnapshotInterval:   positiveDurationEnv("PORTFOLIO_SNAPSHOT_INTERVAL", 15*time.Minute),
		PortfolioSnapshotRetention:  positiveDurationEnv("PORTFOLIO_SNAPSHOT_RETENTION", 100*24*time.Hour),
		MarginInterestInterval:      positiveDurationEnv("MARGIN_INTEREST_INTERVAL", time.Minute),

		ScannerDBPath:        getenv("SCANNER_DB_PATH", "data/scanner.db"),
		ScannerCheckInterval: positiveDurationEnv("SCANNER_CHECK_INTERVAL", 60*time.Second),

		PriceDiffDBPath:        getenv("PRICE_DIFF_DB_PATH", "data/pricediff.db"),
		PriceDiffCheckInterval: positiveDurationEnv("PRICE_DIFF_CHECK_INTERVAL", 30*time.Second),

		FundingArbDBPath:        getenv("FUNDING_ARB_DB_PATH", "data/fundingarb.db"),
		FundingArbCheckInterval: positiveDurationEnv("FUNDING_ARB_CHECK_INTERVAL", 30*time.Second),

		ExportDBPath:         getenv("EXPORT_DB_PATH", "data/export.db"),
		ExportFileDir:        getenv("EXPORT_FILE_DIR", "data/exports"),
		ExportFileTTL:        positiveDurationEnv("EXPORT_FILE_TTL", 1*time.Hour),
		ExportWorkerInterval: positiveDurationEnv("EXPORT_WORKER_INTERVAL", 2*time.Second),

		ImportDBPath:         getenv("IMPORT_DB_PATH", "data/import.db"),
		ImportFileDir:        getenv("IMPORT_FILE_DIR", "data/imports"),
		ImportFileTTL:        positiveDurationEnv("IMPORT_FILE_TTL", 1*time.Hour),
		ImportWorkerInterval: positiveDurationEnv("IMPORT_WORKER_INTERVAL", 2*time.Second),

		AccountDBPath:        getenv("ACCOUNT_DB_PATH", "data/accounts.db"),
		AccountPurgeInterval: positiveDurationEnv("ACCOUNT_PURGE_INTERVAL", 1*time.Hour),

		FuturesHistoryDBPath:    getenv("FUTURES_HISTORY_DB_PATH", "data/futures.db"),
		FuturesHistoryInterval:  positiveDurationEnv("FUTURES_HISTORY_INTERVAL", 5*time.Minute),
		FuturesHistoryRetention: positiveDurationEnv("FUTURES_HISTORY_RETENTION", 30*24*time.Hour),
		FuturesHistorySymbols:   parseCSVList(os.Getenv("FUTURES_HISTORY_SYMBOLS")),

		OrderBookHistoryDBPath:    getenv("ORDERBOOK_HISTORY_DB_PATH", "data/orderbook.db"),
		OrderBookHistoryInterval:  positiveDurationEnv("ORDERBOOK_HISTORY_INTERVAL", time.Minute),
		OrderBookHistoryRetention: positiveDurationEnv("ORDERBOOK_HISTORY_RETENTION", 7*24*time.Hour),
		OrderBookHistorySymbols:   parseCSVList(os.Getenv("ORDERBOOK_HISTORY_SYMBOLS")),

		APIAuthToken: strings.TrimSpace(os.Getenv("API_AUTH_TOKEN")),
		// AllowOpenAuth permits empty API_AUTH_TOKEN when HTTP_ADDR is non-loopback.
		// Default false: refuse to start open auth on 0.0.0.0 / LAN binds.
		AllowOpenAuth: boolEnv("ALLOW_OPEN_AUTH", false),
		// AllowMasterImpersonate permits the master token to act as any clientId
		// from a non-loopback peer. Default false.
		AllowMasterImpersonate: boolEnv("ALLOW_MASTER_IMPERSONATE", false),
		MCPEnabled:             boolEnv("MCP_ENABLED", true),
		WebhookAllowPrivate:    boolEnv("WEBHOOK_ALLOW_PRIVATE", false),
	}
}

// ValidateSecurity checks unsafe auth/bind combinations.
// Open auth (no API_AUTH_TOKEN) is only allowed on loopback unless ALLOW_OPEN_AUTH=true.
func (c Config) ValidateSecurity() error {
	if strings.TrimSpace(c.APIAuthToken) != "" {
		return nil
	}
	if c.AllowOpenAuth {
		return nil
	}
	if isLoopbackHTTPAddr(c.HTTPAddr) {
		return nil
	}
	return fmt.Errorf("API_AUTH_TOKEN is empty and HTTP_ADDR %q is not loopback; set a token or ALLOW_OPEN_AUTH=true for explicit open mode", c.HTTPAddr)
}

func isLoopbackHTTPAddr(addr string) bool {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return true
	}
	// ":8080" binds all interfaces.
	if strings.HasPrefix(addr, ":") {
		return false
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		// bare host or port-only forms
		if addr == "localhost" || addr == "127.0.0.1" || addr == "::1" {
			return true
		}
		return false
	}
	if host == "" || host == "0.0.0.0" || host == "::" || host == "[::]" {
		return false
	}
	ip := net.ParseIP(host)
	if ip != nil {
		return ip.IsLoopback()
	}
	return strings.EqualFold(host, "localhost")
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
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		_ = os.Setenv(key, val)
	}
}
