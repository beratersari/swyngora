package config

import (
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear keys we care about so defaults apply (use t.Setenv).
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("BINANCE_BASE_URL", "")
	t.Setenv("COINGECKO_BASE_URL", "")
	t.Setenv("HTTP_CLIENT_TIMEOUT", "")
	t.Setenv("CANDLE_CACHE_TTL", "")
	t.Setenv("TICKER_CACHE_TTL", "")
	t.Setenv("SUPPLY_CACHE_TTL", "")
	t.Setenv("CACHE_CLEANUP_EVERY", "")

	// os.Getenv returns "" for unset; t.Setenv("", "") may still set empty.
	// Unset by setting then using Load - getenv treats "" as default.
	cfg := Load()
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("HTTPAddr=%q", cfg.HTTPAddr)
	}
	if cfg.BinanceBaseURL != "https://api.binance.com" {
		t.Fatalf("BinanceBaseURL=%q", cfg.BinanceBaseURL)
	}
	if cfg.CoinGeckoBaseURL != "https://api.coingecko.com" {
		t.Fatalf("CoinGeckoBaseURL=%q", cfg.CoinGeckoBaseURL)
	}
	if cfg.HTTPClientTimeout != 15*time.Second {
		t.Fatalf("HTTPClientTimeout=%v", cfg.HTTPClientTimeout)
	}
	if cfg.CandleCacheTTL != 30*time.Second || cfg.TickerCacheTTL != 15*time.Second {
		t.Fatalf("ttls candle=%v ticker=%v", cfg.CandleCacheTTL, cfg.TickerCacheTTL)
	}
	if cfg.SupplyCacheTTL != 5*time.Minute || cfg.CacheCleanupEvery != time.Minute {
		t.Fatalf("supply=%v cleanup=%v", cfg.SupplyCacheTTL, cfg.CacheCleanupEvery)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("HTTP_ADDR", ":9090")
	t.Setenv("BINANCE_BASE_URL", "https://example.test/binance")
	t.Setenv("COINGECKO_BASE_URL", "https://example.test/cg")
	t.Setenv("HTTP_CLIENT_TIMEOUT", "3s")
	t.Setenv("CANDLE_CACHE_TTL", "45s")
	t.Setenv("TICKER_CACHE_TTL", "10") // integer seconds
	t.Setenv("SUPPLY_CACHE_TTL", "2m")
	t.Setenv("CACHE_CLEANUP_EVERY", "30s")

	cfg := Load()
	if cfg.HTTPAddr != ":9090" {
		t.Fatalf("HTTPAddr=%q", cfg.HTTPAddr)
	}
	if cfg.BinanceBaseURL != "https://example.test/binance" {
		t.Fatalf("Binance=%q", cfg.BinanceBaseURL)
	}
	if cfg.CoinGeckoBaseURL != "https://example.test/cg" {
		t.Fatalf("CG=%q", cfg.CoinGeckoBaseURL)
	}
	if cfg.HTTPClientTimeout != 3*time.Second {
		t.Fatalf("timeout=%v", cfg.HTTPClientTimeout)
	}
	if cfg.CandleCacheTTL != 45*time.Second {
		t.Fatalf("candle ttl=%v", cfg.CandleCacheTTL)
	}
	if cfg.TickerCacheTTL != 10*time.Second {
		t.Fatalf("ticker ttl=%v (want integer-seconds parse)", cfg.TickerCacheTTL)
	}
	if cfg.SupplyCacheTTL != 2*time.Minute {
		t.Fatalf("supply ttl=%v", cfg.SupplyCacheTTL)
	}
	if cfg.CacheCleanupEvery != 30*time.Second {
		t.Fatalf("cleanup=%v", cfg.CacheCleanupEvery)
	}
}

func TestDurationEnv_InvalidFallsBack(t *testing.T) {
	t.Setenv("HTTP_CLIENT_TIMEOUT", "not-a-duration")
	cfg := Load()
	if cfg.HTTPClientTimeout != 15*time.Second {
		t.Fatalf("want default on invalid, got %v", cfg.HTTPClientTimeout)
	}
}
