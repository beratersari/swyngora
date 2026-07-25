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
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/bybit"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/cache"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/coinbase"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/platform/config"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/watchliststore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/supplyjob"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/watchlist"
	httpx "gitlab.com/trace-analysis/swyngora/backend/internal/transport/http"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	httpClient := &http.Client{Timeout: cfg.HTTPClientTimeout}

	// Shared short TTL caches are per-adapter (keys would collide across venues).
	binanceCandles := cache.NewWithOptions[[]domain.Candle](cfg.CandleCacheTTL, cache.Options{MaxEntries: cfg.CandleCacheMaxEntries})
	binanceTickers := cache.New[*domain.Ticker24h](cfg.TickerCacheTTL)
	binanceSpot := cache.New[[]domain.SpotMarket](cfg.SpotMarketCacheTTL)
	coinbaseCandles := cache.NewWithOptions[[]domain.Candle](cfg.CandleCacheTTL, cache.Options{MaxEntries: cfg.CandleCacheMaxEntries})
	coinbaseTickers := cache.New[*domain.Ticker24h](cfg.TickerCacheTTL)
	coinbaseSpot := cache.New[[]domain.SpotMarket](cfg.SpotMarketCacheTTL)
	bybitCandles := cache.NewWithOptions[[]domain.Candle](cfg.CandleCacheTTL, cache.Options{MaxEntries: cfg.CandleCacheMaxEntries})
	bybitTickers := cache.New[*domain.Ticker24h](cfg.TickerCacheTTL)
	bybitSpot := cache.New[[]domain.SpotMarket](cfg.SpotMarketCacheTTL)

	// Supply: Binance marketing list only (asset-level, used for all venues' mcap enrichment).
	supplyCache := cache.New[*domain.AssetSupply](cfg.SupplyCacheTTL)

	stopCleanup := make(chan struct{})
	go func() {
		t := time.NewTicker(cfg.CacheCleanupEvery)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				binanceCandles.Cleanup()
				binanceTickers.Cleanup()
				binanceSpot.Cleanup()
				coinbaseCandles.Cleanup()
				coinbaseTickers.Cleanup()
				coinbaseSpot.Cleanup()
				bybitCandles.Cleanup()
				bybitTickers.Cleanup()
				bybitSpot.Cleanup()
				supplyCache.Cleanup()
			case <-stopCleanup:
				return
			}
		}
	}()

	binanceClient := binance.NewClient(binance.Options{
		BaseURL:         cfg.BinanceBaseURL,
		ProductBaseURL:  cfg.BinanceProductBaseURL,
		HTTPClient:      httpClient,
		CandleCache:     binanceCandles,
		TickerCache:     binanceTickers,
		SpotMarketCache: binanceSpot,
		SupplyCache:     supplyCache,
	})
	coinbaseClient := coinbase.NewClient(coinbase.Options{
		BaseURL:         cfg.CoinbaseBaseURL,
		ExchangeURL:     cfg.CoinbaseExchangeURL,
		HTTPClient:      httpClient,
		CandleCache:     coinbaseCandles,
		TickerCache:     coinbaseTickers,
		SpotMarketCache: coinbaseSpot,
	})
	bybitClient := bybit.NewClient(bybit.Options{
		BaseURL:         cfg.BybitBaseURL,
		HTTPClient:      httpClient,
		CandleCache:     bybitCandles,
		TickerCache:     bybitTickers,
		SpotMarketCache: bybitSpot,
	})

	marketSvc := market.NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance:  binanceClient,
		domain.ExchangeCoinbase: coinbaseClient,
		domain.ExchangeBybit:    bybitClient,
	}, binanceClient)

	watchSvc := watchlist.New(watchliststore.NewMemory())

	handler := httpx.NewRouterWithOptions(marketSvc, watchSvc, httpx.RouterOptions{
		RateLimitRPS:   cfg.RateLimitRPS,
		RateLimitBurst: cfg.RateLimitBurst,
	})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	job := &supplyjob.Runner{
		Supply:     binanceClient,
		Hour:       cfg.SupplyRefreshHour,
		Minute:     cfg.SupplyRefreshMinute,
		Loc:        cfg.SupplyRefreshLocation,
		RunOnStart: cfg.SupplyRefreshOnStartup,
		Logger:     logger,
	}
	go job.Start(ctx)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("server listening", "addr", cfg.HTTPAddr, "exchanges", []string{"binance", "coinbase", "bybit"})
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")

	close(stopCleanup)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "err", err)
	}
}
