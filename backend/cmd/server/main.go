package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/binance"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/cache"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/coingecko"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/platform/config"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
	httpx "gitlab.com/trace-analysis/swyngora/backend/internal/transport/http"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	httpClient := &http.Client{Timeout: cfg.HTTPClientTimeout}

	candleCache := cache.New[[]domain.Candle](cfg.CandleCacheTTL)
	tickerCache := cache.New[*domain.Ticker24h](cfg.TickerCacheTTL)
	supplyCache := cache.New[*domain.AssetSupply](cfg.SupplyCacheTTL)
	symbolCache := cache.New[string](24 * time.Hour)

	// Background cache hygiene: drop expired entries periodically.
	stopCleanup := make(chan struct{})
	go func() {
		t := time.NewTicker(cfg.CacheCleanupEvery)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				candleCache.Cleanup()
				tickerCache.Cleanup()
				supplyCache.Cleanup()
				symbolCache.Cleanup()
			case <-stopCleanup:
				return
			}
		}
	}()

	binanceClient := binance.NewClient(binance.Options{
		BaseURL:     cfg.BinanceBaseURL,
		HTTPClient:  httpClient,
		CandleCache: candleCache,
		TickerCache: tickerCache,
	})
	geckoClient := coingecko.NewClient(coingecko.Options{
		BaseURL:     cfg.CoinGeckoBaseURL,
		HTTPClient:  httpClient,
		SupplyCache: supplyCache,
		SymbolCache: symbolCache,
	})

	marketSvc := market.New(binanceClient, geckoClient)
	handler := httpx.NewRouter(marketSvc)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("server listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	logger.Info("shutting down")

	close(stopCleanup)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "err", err)
	}
}
