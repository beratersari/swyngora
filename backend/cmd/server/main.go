package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/alertstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/binance"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/bybit"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/cache"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/coinbase"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/portfoliostore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/scannerstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/watchliststore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/platform/aistart"
	"gitlab.com/trace-analysis/swyngora/backend/internal/platform/config"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/aiagent"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/portfolio"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/pricealert"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/scanner"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/supplyjob"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/watchlist"
	httpx "gitlab.com/trace-analysis/swyngora/backend/internal/transport/http"
	mcpx "gitlab.com/trace-analysis/swyngora/backend/internal/transport/mcp"
	"gitlab.com/trace-analysis/swyngora/backend/internal/transport/telegram"
)

func main() {
	// Optional local secrets (does not override existing env).
	config.LoadDotEnv(".env")
	config.LoadDotEnv("backend/.env")

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

	watchStore, err := watchliststore.OpenSQLite(cfg.WatchlistDBPath)
	if err != nil {
		logger.Error("watchlist sqlite open failed", "path", cfg.WatchlistDBPath, "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := watchStore.Close(); err != nil {
			logger.Error("watchlist sqlite close", "err", err)
		}
	}()
	watchSvc := watchlist.New(watchStore)
	logger.Info("watchlist store ready", "driver", "sqlite", "path", watchStore.Path())

	alertStore, err := alertstore.Open(cfg.AlertsDBPath)
	if err != nil {
		logger.Error("alerts sqlite open failed", "path", cfg.AlertsDBPath, "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := alertStore.Close(); err != nil {
			logger.Error("alerts sqlite close", "err", err)
		}
	}()
	alertSvc := pricealert.New(alertStore)
	logger.Info("price alerts store ready", "driver", "sqlite", "path", alertStore.Path())

	portfolioStore, err := portfoliostore.Open(cfg.PortfolioDBPath)
	if err != nil {
		logger.Error("portfolio sqlite open failed", "path", cfg.PortfolioDBPath, "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := portfolioStore.Close(); err != nil {
			logger.Error("portfolio sqlite close", "err", err)
		}
	}()
	portfolioSvc := portfolio.New(portfolioStore, marketSvc)
	logger.Info("paper portfolio store ready", "driver", "sqlite", "path", portfolioStore.Path())

	scannerStore, err := scannerstore.Open(cfg.ScannerDBPath)
	if err != nil {
		logger.Error("scanner sqlite open failed", "path", cfg.ScannerDBPath, "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := scannerStore.Close(); err != nil {
			logger.Error("scanner sqlite close", "err", err)
		}
	}()
	scannerSvc := scanner.New(scannerStore, marketSvc, watchSvc)
	logger.Info("indicator scanner store ready", "driver", "sqlite", "path", scannerStore.Path())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	orderFiller := &portfolio.OrderFiller{
		Portfolio: portfolioSvc,
		Market:    marketSvc,
		Interval:  cfg.PortfolioOrderCheckInterval,
		Logger:    logger,
	}
	go orderFiller.Start(ctx)

	scannerChecker := &scanner.Checker{
		Scanner:  scannerSvc,
		Interval: cfg.ScannerCheckInterval,
		Logger:   logger,
	}
	go scannerChecker.Start(ctx)

	backtestWorker := &scanner.BacktestWorker{
		Scanner:  scannerSvc,
		Interval: 2 * time.Second,
		Logger:   logger,
	}
	go backtestWorker.Start(ctx)

	// Optional: start Python multi-agent HTTP as a child of this process.
	aiProc, err := aistart.Start(ctx, aistart.Options{
		Enabled: cfg.AIAutoStart,
		Python:  cfg.AIPython,
		WorkDir: cfg.AIWorkDir,
		Host:    cfg.AIListenHost,
		Port:    cfg.AIListenPort,
		Logger:  logger,
	})
	if err != nil {
		logger.Error("AI auto-start failed", "err", err)
	} else if aiProc != nil {
		defer aiProc.Stop()
	}

	aiClient := aiagent.New(cfg.AIServiceURL, cfg.AITimeout)

	// MCP tools run in-process (same binary / same port as REST). No second server.
	mcpServer := mcpx.NewInProcessServer(marketSvc, watchSvc, alertSvc, portfolioSvc, scannerSvc)
	mcpHTTP := mcpx.NewHTTPHandler(mcpServer)

	handler := httpx.NewRouterWithOptions(marketSvc, watchSvc, httpx.RouterOptions{
		RateLimitRPS:     cfg.RateLimitRPS,
		RateLimitBurst:   cfg.RateLimitBurst,
		CORSAllowOrigins: cfg.CORSAllowOrigins,
		MCPHandler:       mcpHTTP,
		AI:              aiClient,
		AITimeout:       cfg.AITimeout,
		Alerts:          alertSvc,
		Portfolio:       portfolioSvc,
		Scanner:         scannerSvc,
	})

	job := &supplyjob.Runner{
		Supply:     binanceClient,
		Hour:       cfg.SupplyRefreshHour,
		Minute:     cfg.SupplyRefreshMinute,
		Loc:        cfg.SupplyRefreshLocation,
		RunOnStart: cfg.SupplyRefreshOnStartup,
		Logger:     logger,
	}
	go job.Start(ctx)

	alertChecker := &pricealert.Checker{
		Alerts:   alertSvc,
		Market:   marketSvc,
		Interval: cfg.AlertCheckInterval,
		Logger:   logger,
	}
	go alertChecker.Start(ctx)

	webhookDeliverer := &pricealert.Deliverer{
		Alerts:      alertSvc,
		HTTP:        &http.Client{Timeout: cfg.WebhookHTTPTimeout},
		Interval:    cfg.WebhookDeliveryInterval,
		MaxAttempts: cfg.WebhookMaxAttempts,
		Logger:      logger,
	}
	go webhookDeliverer.Start(ctx)

	// Optional Telegram bot transport (same process, in-process services).
	// Fail closed: token without allowlist and without TELEGRAM_ALLOW_ALL does not start.
	if cfg.TelegramBotToken != "" {
		if len(cfg.TelegramAllowedChats) == 0 && !cfg.TelegramAllowAll {
			logger.Error("telegram bot NOT started: set TELEGRAM_CHAT_ID or TELEGRAM_ALLOWED_CHAT_IDS, or explicitly TELEGRAM_ALLOW_ALL=true (public bot)")
		} else {
			tgClient := telegram.NewClient(cfg.TelegramBotToken, cfg.HTTPClientTimeout+cfg.TelegramPollTimeout+10*time.Second)
			router := telegram.NewRouter(marketSvc, watchSvc, telegram.Options{
				DefaultExchange: cfg.TelegramDefaultExchange,
				LowMcapLimit:    cfg.TelegramLowMcapLimit,
				AllowedChatIDs:  cfg.TelegramAllowedChats,
				AllowAll:        cfg.TelegramAllowAll,
				AI:              aiClient,
				AITimeout:       cfg.AITimeout,
			})
			bot := &telegram.Bot{
				Client:      tgClient,
				Router:      router,
				Logger:      logger,
				PollTimeout: cfg.TelegramPollTimeout,
			}
			go bot.Start(ctx)
			logger.Info("telegram bot enabled",
				"allowlist", len(cfg.TelegramAllowedChats),
				"allow_all", cfg.TelegramAllowAll,
			)
		}
	} else {
		logger.Info("telegram bot disabled (set TELEGRAM_BOT_TOKEN to enable)")
	}

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		// AI chat can take longer than market endpoints.
		WriteTimeout:      180 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("server listening",
			"addr", cfg.HTTPAddr,
			"exchanges", []string{"binance", "coinbase", "bybit"},
			"mcp", "/mcp",
			"ai_service", cfg.AIServiceURL,
		)
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
