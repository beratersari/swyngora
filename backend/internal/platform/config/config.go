package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds process configuration from the environment.
type Config struct {
	HTTPAddr           string
	BinanceBaseURL     string
	CoinGeckoBaseURL   string
	HTTPClientTimeout  time.Duration
	CandleCacheTTL     time.Duration
	TickerCacheTTL     time.Duration
	SupplyCacheTTL     time.Duration
	CacheCleanupEvery  time.Duration
}

// Load reads configuration from environment variables with safe defaults.
func Load() Config {
	return Config{
		HTTPAddr:          getenv("HTTP_ADDR", ":8080"),
		BinanceBaseURL:    getenv("BINANCE_BASE_URL", "https://api.binance.com"),
		CoinGeckoBaseURL:  getenv("COINGECKO_BASE_URL", "https://api.coingecko.com"),
		HTTPClientTimeout: durationEnv("HTTP_CLIENT_TIMEOUT", 15*time.Second),
		CandleCacheTTL:    durationEnv("CANDLE_CACHE_TTL", 30*time.Second),
		TickerCacheTTL:    durationEnv("TICKER_CACHE_TTL", 15*time.Second),
		SupplyCacheTTL:    durationEnv("SUPPLY_CACHE_TTL", 5*time.Minute),
		CacheCleanupEvery: durationEnv("CACHE_CLEANUP_EVERY", 1*time.Minute),
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
	// Accept Go duration strings ("30s") or integer seconds.
	if d, err := time.ParseDuration(v); err == nil {
		return d
	}
	if sec, err := strconv.Atoi(v); err == nil {
		return time.Duration(sec) * time.Second
	}
	return def
}
