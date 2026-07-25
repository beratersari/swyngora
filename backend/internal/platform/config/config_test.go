package config

import (
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("BINANCE_BASE_URL", "")
	t.Setenv("BINANCE_PRODUCT_BASE_URL", "")
	t.Setenv("HTTP_CLIENT_TIMEOUT", "")
	t.Setenv("CANDLE_CACHE_TTL", "")
	t.Setenv("TICKER_CACHE_TTL", "")
	t.Setenv("SUPPLY_CACHE_TTL", "")
	t.Setenv("CACHE_CLEANUP_EVERY", "")
	t.Setenv("SPOT_MARKET_CACHE_TTL", "")
	t.Setenv("SUPPLY_REFRESH_HOUR", "")
	t.Setenv("SUPPLY_REFRESH_MINUTE", "")
	t.Setenv("SUPPLY_REFRESH_TZ", "")
	t.Setenv("SUPPLY_REFRESH_ON_STARTUP", "")

	cfg := Load()
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("HTTPAddr=%q", cfg.HTTPAddr)
	}
	if cfg.BinanceProductBaseURL != "https://www.binance.com" {
		t.Fatalf("product base=%q", cfg.BinanceProductBaseURL)
	}
	if cfg.SupplyCacheTTL != 26*time.Hour {
		t.Fatalf("supply ttl=%v", cfg.SupplyCacheTTL)
	}
	if cfg.SpotMarketCacheTTL != 5*time.Second {
		t.Fatalf("spot ttl=%v", cfg.SpotMarketCacheTTL)
	}
	if cfg.SupplyRefreshHour != 3 || cfg.SupplyRefreshMinute != 0 {
		t.Fatalf("refresh at %d:%d", cfg.SupplyRefreshHour, cfg.SupplyRefreshMinute)
	}
	if !cfg.SupplyRefreshOnStartup {
		t.Fatal("startup refresh default true")
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("HTTP_ADDR", ":9090")
	t.Setenv("BINANCE_PRODUCT_BASE_URL", "https://example.test")
	t.Setenv("SUPPLY_CACHE_TTL", "30h")
	t.Setenv("SUPPLY_REFRESH_HOUR", "4")
	t.Setenv("SUPPLY_REFRESH_MINUTE", "15")
	t.Setenv("SUPPLY_REFRESH_ON_STARTUP", "false")

	cfg := Load()
	if cfg.HTTPAddr != ":9090" || cfg.SupplyCacheTTL != 30*time.Hour {
		t.Fatalf("cfg=%+v", cfg)
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
}

func TestDurationEnv_InvalidFallsBack(t *testing.T) {
	t.Setenv("HTTP_CLIENT_TIMEOUT", "not-a-duration")
	cfg := Load()
	if cfg.HTTPClientTimeout != 15*time.Second {
		t.Fatalf("want default on invalid, got %v", cfg.HTTPClientTimeout)
	}
}
