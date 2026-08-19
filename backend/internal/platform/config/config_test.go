package config

import (
	"testing"
	"time"
)

func TestValidateSecurity_OpenAuthNonLoopback(t *testing.T) {
	cfg := Config{HTTPAddr: ":8080", APIAuthToken: ""}
	if err := cfg.ValidateSecurity(); err == nil {
		t.Fatal("expected error for open auth on all interfaces")
	}
	cfg.AllowOpenAuth = true
	if err := cfg.ValidateSecurity(); err != nil {
		t.Fatal(err)
	}
	cfg = Config{HTTPAddr: "127.0.0.1:8080", APIAuthToken: ""}
	if err := cfg.ValidateSecurity(); err != nil {
		t.Fatal(err)
	}
	cfg = Config{HTTPAddr: "0.0.0.0:8080", APIAuthToken: "secret"}
	if err := cfg.ValidateSecurity(); err != nil {
		t.Fatal(err)
	}
}

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("BINANCE_BASE_URL", "")
	t.Setenv("BINANCE_PRODUCT_BASE_URL", "")
	t.Setenv("HTTP_CLIENT_TIMEOUT", "")
	t.Setenv("CANDLE_CACHE_TTL", "")
	t.Setenv("CANDLE_CACHE_MAX_ENTRIES", "")
	t.Setenv("TICKER_CACHE_TTL", "")
	t.Setenv("SUPPLY_CACHE_TTL", "")
	t.Setenv("CACHE_CLEANUP_EVERY", "")
	t.Setenv("SPOT_MARKET_CACHE_TTL", "")
	t.Setenv("BINANCE_FUTURES_BASE_URL", "")
	t.Setenv("OPEN_INTEREST_CACHE_TTL", "")
	t.Setenv("SUPPLY_REFRESH_HOUR", "")
	t.Setenv("SUPPLY_REFRESH_MINUTE", "")
	t.Setenv("SUPPLY_REFRESH_TZ", "")
	t.Setenv("SUPPLY_REFRESH_ON_STARTUP", "")
	t.Setenv("RATE_LIMIT_RPS", "")
	t.Setenv("RATE_LIMIT_BURST", "")
	t.Setenv("CORS_ALLOW_ORIGINS", "")
	t.Setenv("TELEGRAM_ALLOW_ALL", "")
	t.Setenv("WATCHLIST_DB_PATH", "")
	t.Setenv("ALERTS_DB_PATH", "")
	t.Setenv("ALERT_CHECK_INTERVAL", "")
	t.Setenv("WEBHOOK_DELIVERY_INTERVAL", "")
	t.Setenv("WEBHOOK_HTTP_TIMEOUT", "")
	t.Setenv("WEBHOOK_MAX_ATTEMPTS", "")
	t.Setenv("PORTFOLIO_DB_PATH", "")

	cfg := Load()
	if cfg.HTTPAddr != "127.0.0.1:8080" {
		t.Fatalf("HTTPAddr=%q", cfg.HTTPAddr)
	}
	if err := cfg.ValidateSecurity(); err != nil {
		t.Fatalf("default bind should allow open auth: %v", err)
	}
	if cfg.BinanceProductBaseURL != "https://www.binance.com" {
		t.Fatalf("product base=%q", cfg.BinanceProductBaseURL)
	}
	if cfg.SupplyCacheTTL != 48*time.Hour {
		t.Fatalf("supply ttl default want 48h safety TTL, got %v", cfg.SupplyCacheTTL)
	}
	if cfg.HoldersCacheTTL != time.Hour {
		t.Fatalf("holders ttl=%v", cfg.HoldersCacheTTL)
	}
	if cfg.CMCBaseURL != "https://api.coinmarketcap.com" {
		t.Fatalf("cmc base=%q", cfg.CMCBaseURL)
	}
	if cfg.CandleCacheMaxEntries != 512 {
		t.Fatalf("candle max entries=%d", cfg.CandleCacheMaxEntries)
	}
	if cfg.SpotMarketCacheTTL != 5*time.Second {
		t.Fatalf("spot ttl=%v", cfg.SpotMarketCacheTTL)
	}
	if cfg.BinanceFuturesBaseURL != "https://fapi.binance.com" {
		t.Fatalf("futures base=%q", cfg.BinanceFuturesBaseURL)
	}
	if cfg.OpenInterestCacheTTL != 30*time.Second {
		t.Fatalf("oi ttl=%v", cfg.OpenInterestCacheTTL)
	}
	if cfg.SupplyRefreshHour != 3 || cfg.SupplyRefreshMinute != 0 {
		t.Fatalf("refresh at %d:%d", cfg.SupplyRefreshHour, cfg.SupplyRefreshMinute)
	}
	if !cfg.SupplyRefreshOnStartup {
		t.Fatal("startup refresh default true")
	}
	if cfg.RateLimitRPS != 40 || cfg.RateLimitBurst != 80 {
		t.Fatalf("rate limit %v/%d", cfg.RateLimitRPS, cfg.RateLimitBurst)
	}
	// Empty CORS_ALLOW_ORIGINS falls back to getenv default "*"
	if len(cfg.CORSAllowOrigins) != 1 || cfg.CORSAllowOrigins[0] != "*" {
		t.Fatalf("CORS default=%v", cfg.CORSAllowOrigins)
	}
	if cfg.TelegramAllowAll {
		t.Fatal("TelegramAllowAll default false")
	}
	if cfg.WatchlistDBPath != "data/watchlist.db" {
		t.Fatalf("WatchlistDBPath default=%q", cfg.WatchlistDBPath)
	}
	if cfg.AlertsDBPath != "data/alerts.db" {
		t.Fatalf("AlertsDBPath default=%q", cfg.AlertsDBPath)
	}
	if cfg.AlertCheckInterval != 30*time.Second {
		t.Fatalf("AlertCheckInterval default=%v", cfg.AlertCheckInterval)
	}
	if cfg.WebhookDeliveryInterval != 5*time.Second {
		t.Fatalf("WebhookDeliveryInterval default=%v", cfg.WebhookDeliveryInterval)
	}
	if cfg.WebhookHTTPTimeout != 10*time.Second {
		t.Fatalf("WebhookHTTPTimeout default=%v", cfg.WebhookHTTPTimeout)
	}
	if cfg.WebhookMaxAttempts != 8 {
		t.Fatalf("WebhookMaxAttempts default=%d", cfg.WebhookMaxAttempts)
	}
	if cfg.RealtimePriceInterval != 5*time.Second {
		t.Fatalf("RealtimePriceInterval default=%v", cfg.RealtimePriceInterval)
	}
	if cfg.PortfolioDBPath != "data/portfolio.db" {
		t.Fatalf("PortfolioDBPath default=%q", cfg.PortfolioDBPath)
	}
	if cfg.PortfolioOrderCheckInterval != 15*time.Second {
		t.Fatalf("PortfolioOrderCheckInterval default=%v", cfg.PortfolioOrderCheckInterval)
	}
	if cfg.PortfolioSnapshotInterval != 15*time.Minute {
		t.Fatalf("PortfolioSnapshotInterval default=%v", cfg.PortfolioSnapshotInterval)
	}
	if cfg.PortfolioSnapshotRetention != 100*24*time.Hour {
		t.Fatalf("PortfolioSnapshotRetention default=%v", cfg.PortfolioSnapshotRetention)
	}
	if cfg.ScannerDBPath != "data/scanner.db" {
		t.Fatalf("ScannerDBPath default=%q", cfg.ScannerDBPath)
	}
	if cfg.ScannerCheckInterval != 60*time.Second {
		t.Fatalf("ScannerCheckInterval default=%v", cfg.ScannerCheckInterval)
	}
	if cfg.APIAuthToken != "" {
		t.Fatalf("APIAuthToken default empty, got %q", cfg.APIAuthToken)
	}
	if !cfg.MCPEnabled {
		t.Fatal("MCPEnabled default true")
	}
	if cfg.WebhookAllowPrivate {
		t.Fatal("WebhookAllowPrivate default false")
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("HTTP_ADDR", ":9090")
	t.Setenv("BINANCE_PRODUCT_BASE_URL", "https://example.test")
	t.Setenv("SUPPLY_CACHE_TTL", "30h")
	t.Setenv("SUPPLY_REFRESH_HOUR", "4")
	t.Setenv("SUPPLY_REFRESH_MINUTE", "15")
	t.Setenv("SUPPLY_REFRESH_ON_STARTUP", "false")
	t.Setenv("RATE_LIMIT_RPS", "5")
	t.Setenv("RATE_LIMIT_BURST", "10")
	t.Setenv("WATCHLIST_DB_PATH", "C:/tmp/wl.db")

	cfg := Load()
	if cfg.HTTPAddr != ":9090" || cfg.SupplyCacheTTL != 30*time.Hour {
		t.Fatalf("cfg=%+v", cfg)
	}
	if cfg.WatchlistDBPath != "C:/tmp/wl.db" {
		t.Fatalf("WatchlistDBPath=%q", cfg.WatchlistDBPath)
	}
	if cfg.BinanceProductBaseURL != "https://example.test" {
		t.Fatalf("product base=%q", cfg.BinanceProductBaseURL)
	}
	if cfg.SupplyRefreshHour != 4 || cfg.SupplyRefreshMinute != 15 {
		t.Fatalf("time %d:%d", cfg.SupplyRefreshHour, cfg.SupplyRefreshMinute)
	}
	if cfg.SupplyRefreshOnStartup {
		t.Fatal("startup should be false")
	}
	if cfg.RateLimitRPS != 5 || cfg.RateLimitBurst != 10 {
		t.Fatalf("rate %v/%d", cfg.RateLimitRPS, cfg.RateLimitBurst)
	}
}

func TestDurationEnv_InvalidFallsBack(t *testing.T) {
	t.Setenv("HTTP_CLIENT_TIMEOUT", "not-a-duration")
	cfg := Load()
	if cfg.HTTPClientTimeout != 15*time.Second {
		t.Fatalf("want default on invalid, got %v", cfg.HTTPClientTimeout)
	}
}

func TestDurationEnv_ZeroFallsBackForClientTimeout(t *testing.T) {
	t.Setenv("HTTP_CLIENT_TIMEOUT", "0")
	t.Setenv("CACHE_CLEANUP_EVERY", "0s")
	cfg := Load()
	if cfg.HTTPClientTimeout != 15*time.Second {
		t.Fatalf("zero timeout must fall back, got %v", cfg.HTTPClientTimeout)
	}
	if cfg.CacheCleanupEvery != time.Minute {
		t.Fatalf("zero cleanup must fall back, got %v", cfg.CacheCleanupEvery)
	}
}

func TestSupplyTTL_ZeroAllowed(t *testing.T) {
	t.Setenv("SUPPLY_CACHE_TTL", "0")
	cfg := Load()
	if cfg.SupplyCacheTTL != 0 {
		t.Fatalf("want 0, got %v", cfg.SupplyCacheTTL)
	}
}

func TestTelegramChatIDs(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "tok")
	t.Setenv("TELEGRAM_CHAT_ID", "42")
	t.Setenv("TELEGRAM_ALLOWED_CHAT_IDS", "100,200")
	cfg := Load()
	if cfg.TelegramBotToken != "tok" {
		t.Fatalf("token=%q", cfg.TelegramBotToken)
	}
	if _, ok := cfg.TelegramAllowedChats[42]; !ok {
		t.Fatal("missing chat 42")
	}
	if _, ok := cfg.TelegramAllowedChats[100]; !ok {
		t.Fatal("missing chat 100")
	}
}

func TestTelegramDisabledByDefault(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_CHAT_ID", "")
	t.Setenv("TELEGRAM_ALLOWED_CHAT_IDS", "")
	cfg := Load()
	if cfg.TelegramBotToken != "" {
		t.Fatal("expected empty token")
	}
}

func TestCORSAllowOrigins_Parse(t *testing.T) {
	t.Setenv("CORS_ALLOW_ORIGINS", "http://localhost:5173, https://app.example.com")
	cfg := Load()
	if len(cfg.CORSAllowOrigins) != 2 {
		t.Fatalf("got %v", cfg.CORSAllowOrigins)
	}
	if cfg.CORSAllowOrigins[0] != "http://localhost:5173" || cfg.CORSAllowOrigins[1] != "https://app.example.com" {
		t.Fatalf("%v", cfg.CORSAllowOrigins)
	}
}

func TestTelegramAllowAll(t *testing.T) {
	t.Setenv("TELEGRAM_ALLOW_ALL", "true")
	t.Setenv("TELEGRAM_CHAT_ID", "")
	t.Setenv("TELEGRAM_ALLOWED_CHAT_IDS", "")
	cfg := Load()
	if !cfg.TelegramAllowAll {
		t.Fatal("want TelegramAllowAll true")
	}
	if cfg.TelegramAllowedChats != nil {
		t.Fatalf("empty allowlist want nil, got %v", cfg.TelegramAllowedChats)
	}
}
