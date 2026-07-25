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
		// 0 = never expire supply entries; ReplaceAll on successful refresh only.
		// A positive SUPPLY_CACHE_TTL still works as a safety TTL if set.
		SupplyCacheTTL:     durationEnvAllowZero("SUPPLY_CACHE_TTL", 0),
		CacheCleanupEvery:  positiveDurationEnv("CACHE_CLEANUP_EVERY", 1*time.Minute),
		SpotMarketCacheTTL: positiveDurationEnv("SPOT_MARKET_CACHE_TTL", 5*time.Second),

		SupplyRefreshHour:      intEnvClamp("SUPPLY_REFRESH_HOUR", 3, 0, 23),
		SupplyRefreshMinute:    intEnvClamp("SUPPLY_REFRESH_MINUTE", 0, 0, 59),
		SupplyRefreshLocation:  loc,
		SupplyRefreshOnStartup: boolEnv("SUPPLY_REFRESH_ON_STARTUP", true),

		RateLimitRPS:   floatEnv("RATE_LIMIT_RPS", 20),
		RateLimitBurst: positiveIntEnv("RATE_LIMIT_BURST", 40),
	}
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
