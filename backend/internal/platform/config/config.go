package config

import (
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
	HTTPClientTimeout     time.Duration
	CandleCacheTTL        time.Duration
	TickerCacheTTL        time.Duration
	SupplyCacheTTL        time.Duration
	CacheCleanupEvery     time.Duration
	SpotMarketCacheTTL    time.Duration

	// Daily supply snapshot refresh (Binance product catalog). User requests read cache only.
	SupplyRefreshHour      int
	SupplyRefreshMinute    int
	SupplyRefreshLocation  *time.Location
	SupplyRefreshOnStartup bool
}

// Load reads configuration from environment variables with safe defaults.
func Load() Config {
	loc := loadLocation(getenv("SUPPLY_REFRESH_TZ", "UTC"))
	return Config{
		HTTPAddr:              getenv("HTTP_ADDR", ":8080"),
		BinanceBaseURL:        getenv("BINANCE_BASE_URL", "https://api.binance.com"),
		BinanceProductBaseURL: getenv("BINANCE_PRODUCT_BASE_URL", "https://www.binance.com"),
		HTTPClientTimeout:     durationEnv("HTTP_CLIENT_TIMEOUT", 15*time.Second),
		CandleCacheTTL:        durationEnv("CANDLE_CACHE_TTL", 30*time.Second),
		TickerCacheTTL:        durationEnv("TICKER_CACHE_TTL", 15*time.Second),
		// Safety TTL slightly over 24h so a late refresh still serves yesterday's snapshot.
		SupplyCacheTTL:     durationEnv("SUPPLY_CACHE_TTL", 26*time.Hour),
		CacheCleanupEvery:  durationEnv("CACHE_CLEANUP_EVERY", 1*time.Minute),
		SpotMarketCacheTTL: durationEnv("SPOT_MARKET_CACHE_TTL", 30*time.Second),

		SupplyRefreshHour:      intEnv("SUPPLY_REFRESH_HOUR", 3),
		SupplyRefreshMinute:    intEnv("SUPPLY_REFRESH_MINUTE", 0),
		SupplyRefreshLocation:  loc,
		SupplyRefreshOnStartup: boolEnv("SUPPLY_REFRESH_ON_STARTUP", true),
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func durationEnv(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
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

func intEnv(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
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
