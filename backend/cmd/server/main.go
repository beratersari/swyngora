package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/accountstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/alertstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/binance"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/bookhiststore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/bybit"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/cache"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/cmc"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/coinbase"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/deliststore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/equities"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/exportstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/futuresstore"
	fxrates "gitlab.com/trace-analysis/swyngora/backend/internal/adapter/fx"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/importstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/portfoliostore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/pricediffstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/scannerstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/watchliststore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/platform/aistart"
	"gitlab.com/trace-analysis/swyngora/backend/internal/platform/config"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/account"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/aiagent"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/apikey"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/bookhist"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/dataimport"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/delistjob"
	exportsvc "gitlab.com/trace-analysis/swyngora/backend/internal/service/export"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/futureshist"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/portfolio"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/pricealert"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/pricediff"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/realtime"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/scanner"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/supplyjob"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/swing"
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
	if err := cfg.ValidateSecurity(); err != nil {
		slog.Error("unsafe auth/bind configuration", "err", err)
		os.Exit(1)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	httpClient := &http.Client{Timeout: cfg.HTTPClientTimeout}

	// Shared short TTL caches are per-adapter (keys would collide across venues).
	binanceCandles := cache.NewWithOptions[[]domain.Candle](cfg.CandleCacheTTL, cache.Options{MaxEntries: cfg.CandleCacheMaxEntries})
	binanceTickers := cache.New[*domain.Ticker24h](cfg.TickerCacheTTL)
	binanceBooks := cache.NewWithOptions[*domain.RawOrderBook](cfg.OrderBookCacheTTL, cache.Options{MaxEntries: 256})
	binanceSpot := cache.New[[]domain.SpotMarket](cfg.SpotMarketCacheTTL)
	binanceOI := cache.New[*domain.OpenInterestSeries](cfg.OpenInterestCacheTTL)
	coinbaseCandles := cache.NewWithOptions[[]domain.Candle](cfg.CandleCacheTTL, cache.Options{MaxEntries: cfg.CandleCacheMaxEntries})
	coinbaseTickers := cache.New[*domain.Ticker24h](cfg.TickerCacheTTL)
	coinbaseBooks := cache.NewWithOptions[*domain.RawOrderBook](cfg.OrderBookCacheTTL, cache.Options{MaxEntries: 256})
	coinbaseSpot := cache.New[[]domain.SpotMarket](cfg.SpotMarketCacheTTL)
	bybitCandles := cache.NewWithOptions[[]domain.Candle](cfg.CandleCacheTTL, cache.Options{MaxEntries: cfg.CandleCacheMaxEntries})
	bybitTickers := cache.New[*domain.Ticker24h](cfg.TickerCacheTTL)
	bybitBooks := cache.NewWithOptions[*domain.RawOrderBook](cfg.OrderBookCacheTTL, cache.Options{MaxEntries: 256})
	bybitSpot := cache.New[[]domain.SpotMarket](cfg.SpotMarketCacheTTL)
	bybitOI := cache.New[*domain.OpenInterestSeries](cfg.OpenInterestCacheTTL)

	// Supply: Binance marketing list only (asset-level, used for all venues' mcap enrichment).
	supplyCache := cache.New[*domain.AssetSupply](cfg.SupplyCacheTTL)
	catalogCache := cache.New[*domain.AssetCatalogEntry](cfg.SupplyCacheTTL)
	holdersCache := cache.NewWithOptions[*domain.AssetHolders](cfg.HoldersCacheTTL, cache.Options{MaxEntries: 512})

	stopCleanup := make(chan struct{})
	go func() {
		t := time.NewTicker(cfg.CacheCleanupEvery)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				binanceCandles.Cleanup()
				binanceTickers.Cleanup()
				binanceBooks.Cleanup()
				binanceSpot.Cleanup()
				coinbaseCandles.Cleanup()
				coinbaseTickers.Cleanup()
				coinbaseBooks.Cleanup()
				coinbaseSpot.Cleanup()
				bybitCandles.Cleanup()
				bybitTickers.Cleanup()
				bybitBooks.Cleanup()
				bybitSpot.Cleanup()
				binanceOI.Cleanup()
				bybitOI.Cleanup()
				supplyCache.Cleanup()
				catalogCache.Cleanup()
				holdersCache.Cleanup()
			case <-stopCleanup:
				return
			}
		}
	}()

	binanceClient := binance.NewClient(binance.Options{
		BaseURL:           cfg.BinanceBaseURL,
		ProductBaseURL:    cfg.BinanceProductBaseURL,
		APIKey:            cfg.BinanceAPIKey,
		HTTPClient:        httpClient,
		CandleCache:       binanceCandles,
		TickerCache:       binanceTickers,
		OrderBookCache:    binanceBooks,
		SpotMarketCache:   binanceSpot,
		SupplyCache:       supplyCache,
		CatalogCache:      catalogCache,
		WSURL:             cfg.BinanceWSURL,
		DepthIdle:         cfg.OrderBookIdleTTL,
		DepthWait:         cfg.OrderBookSyncTimeout,
		FuturesBaseURL:    cfg.BinanceFuturesBaseURL,
		OpenInterestCache: binanceOI,
	})
	defer binanceClient.Close()
	coinbaseClient := coinbase.NewClient(coinbase.Options{
		BaseURL:         cfg.CoinbaseBaseURL,
		ExchangeURL:     cfg.CoinbaseExchangeURL,
		HTTPClient:      httpClient,
		CandleCache:     coinbaseCandles,
		TickerCache:     coinbaseTickers,
		OrderBookCache:  coinbaseBooks,
		SpotMarketCache: coinbaseSpot,
		WSURL:           cfg.CoinbaseWSURL,
		DepthIdle:       cfg.OrderBookIdleTTL,
		DepthWait:       cfg.OrderBookSyncTimeout,
	})
	defer coinbaseClient.Close()
	bybitClient := bybit.NewClient(bybit.Options{
		BaseURL:           cfg.BybitBaseURL,
		HTTPClient:        httpClient,
		CandleCache:       bybitCandles,
		TickerCache:       bybitTickers,
		OrderBookCache:    bybitBooks,
		SpotMarketCache:   bybitSpot,
		WSURL:             cfg.BybitWSURL,
		DepthIdle:         cfg.OrderBookIdleTTL,
		DepthWait:         cfg.OrderBookSyncTimeout,
		OpenInterestCache: bybitOI,
	})
	defer bybitClient.Close()

	delistStore := deliststore.NewMemory()
	binanceDelist := cfg.BinanceAPIKey != ""
	liqBook := domain.NewLiquidationBook()

	futuresStore, err := futuresstore.Open(cfg.FuturesHistoryDBPath)
	if err != nil {
		logger.Error("futures history sqlite open failed", "path", cfg.FuturesHistoryDBPath, "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := futuresStore.Close(); err != nil {
			logger.Error("futures history sqlite close", "err", err)
		}
	}()
	histSeeds := append([]string{}, domain.DefaultFuturesHistorySymbols...)
	histSeeds = append(histSeeds, cfg.FuturesHistorySymbols...)
	futuresHist := &futureshist.Service{
		Store:      futuresStore,
		TakerStore: futuresStore,
		OI:         map[domain.Exchange]domain.OpenInterestPort{domain.ExchangeBinance: binanceClient, domain.ExchangeBybit: bybitClient},
		Funding:    map[domain.Exchange]domain.FundingRatePort{domain.ExchangeBinance: binanceClient, domain.ExchangeBybit: bybitClient},
		LS:         map[domain.Exchange]domain.LongShortRatioPort{domain.ExchangeBinance: binanceClient, domain.ExchangeBybit: bybitClient},
		Taker:      map[domain.Exchange]domain.TakerBucketPort{domain.ExchangeBinance: binanceClient, domain.ExchangeBybit: bybitClient},
		Logger:     logger,
		Seeds:      histSeeds,
	}
	liqSink := futureshist.NewPersistSink(liqBook, futuresHist)
	binanceLiq := binance.NewLiquidationHub(binance.LiquidationOptions{
		WSURL: cfg.BinanceFuturesWSURL,
		Sink:  liqSink,
	})
	bybitLiq := bybit.NewLiquidationHub(bybit.LiquidationOptions{
		WSURL:   cfg.BybitLinearWSURL,
		BaseURL: cfg.BybitBaseURL,
		HTTP:    httpClient,
		Sink:    liqSink,
	})
	defer binanceLiq.Close()
	defer bybitLiq.Close()

	marketSvc := market.NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance:  binanceClient,
		domain.ExchangeCoinbase: coinbaseClient,
		domain.ExchangeBybit:    bybitClient,
		domain.ExchangeNasdaq:   equities.NewNasdaq(equities.Options{HTTPClient: httpClient}),
		domain.ExchangeBist:     equities.NewBist(equities.Options{HTTPClient: httpClient}),
	}, binanceClient).WithDelistStore(delistStore).WithDelistSource(domain.ExchangeBinance, binanceDelist).WithDelistSource(domain.ExchangeBybit, true).WithLiquidations(liqBook, bybitLiq).WithFx(fxrates.New(httpClient)).WithHolders(cmc.New(cmc.Options{
		BaseURL:    cfg.CMCBaseURL,
		HTTPClient: httpClient,
		Catalog:    binanceClient,
		Cache:      holdersCache,
	})).WithOpenInterest(map[domain.Exchange]domain.OpenInterestPort{
		domain.ExchangeBinance: binanceClient,
		domain.ExchangeBybit:   bybitClient,
	}).WithFundingRate(map[domain.Exchange]domain.FundingRatePort{
		domain.ExchangeBinance: binanceClient,
		domain.ExchangeBybit:   bybitClient,
	}).WithLongShortRatio(map[domain.Exchange]domain.LongShortRatioPort{
		domain.ExchangeBinance: binanceClient,
		domain.ExchangeBybit:   bybitClient,
	}).WithTakerFlow(map[domain.Exchange]domain.TakerFlowPort{
		domain.ExchangeBinance: binanceClient,
		domain.ExchangeBybit:   bybitClient,
	}).WithTakerBucketStore(futuresStore).WithBasis(map[domain.Exchange]domain.BasisPort{
		domain.ExchangeBinance: binanceClient,
		domain.ExchangeBybit:   bybitClient,
	}).WithWindowChanges(map[domain.Exchange]domain.WindowChangePort{
		domain.ExchangeBinance: binanceClient,
	})
	bybitTrades := bybit.NewTradeHub(bybit.TradeHubOptions{
		WSURL: cfg.BybitLinearWSURL,
		Book:  bybitClient.TakerBook(),
	})
	bybitClient.SetTakerWatch(bybitTrades.Watch)
	defer bybitTrades.Close()
	marketSvc.SetOnFuturesSymbol(futuresHist.NoteSymbol)
	marketSvc.SetFuturesHistory(futuresHist)
	logger.Info("futures history store ready", "driver", "sqlite", "path", futuresStore.Path())

	bookStore, err := bookhiststore.Open(cfg.OrderBookHistoryDBPath)
	if err != nil {
		logger.Error("order book history sqlite open failed", "path", cfg.OrderBookHistoryDBPath, "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := bookStore.Close(); err != nil {
			logger.Error("order book history sqlite close", "err", err)
		}
	}()
	bookSeeds := append([]string{}, domain.DefaultBookHistorySymbols...)
	bookSeeds = append(bookSeeds, cfg.OrderBookHistorySymbols...)
	bookHist := &bookhist.Service{
		Store:  bookStore,
		Books:  map[domain.Exchange]domain.MarketDataPort{domain.ExchangeBinance: binanceClient, domain.ExchangeCoinbase: coinbaseClient, domain.ExchangeBybit: bybitClient},
		Logger: logger,
		Seeds:  bookSeeds,
	}
	marketSvc.WithBookHistory(bookHist)
	logger.Info("order book history store ready", "driver", "sqlite", "path", bookStore.Path())

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
	alertSvc.AllowPrivateWebhooks = cfg.WebhookAllowPrivate
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

	realtimeHub := realtime.NewHub(realtime.Options{
		Market:   marketSvc,
		Access:   portfolioSvc,
		Interval: cfg.RealtimePriceInterval,
	})
	portfolioSvc.SetChangeSink(realtimeHub)

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

	priceDiffStore, err := pricediffstore.Open(cfg.PriceDiffDBPath)
	if err != nil {
		logger.Error("price-diff sqlite open failed", "path", cfg.PriceDiffDBPath, "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := priceDiffStore.Close(); err != nil {
			logger.Error("price-diff sqlite close", "err", err)
		}
	}()
	priceDiffSvc := pricediff.New(priceDiffStore, marketSvc)
	logger.Info("price-diff store ready", "driver", "sqlite", "path", priceDiffStore.Path())

	exportStore, err := exportstore.Open(cfg.ExportDBPath)
	if err != nil {
		logger.Error("export sqlite open failed", "path", cfg.ExportDBPath, "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := exportStore.Close(); err != nil {
			logger.Error("export sqlite close", "err", err)
		}
	}()
	exportSvc, err := exportsvc.New(exportStore, exportsvc.DataSources{
		Watchlist: watchStore,
		Alerts:    alertStore,
		Scanner:   scannerStore,
		Portfolio: portfolioStore,
	}, exportsvc.Options{
		FileDir: cfg.ExportFileDir,
		FileTTL: cfg.ExportFileTTL,
	})
	if err != nil {
		logger.Error("export service init failed", "err", err)
		os.Exit(1)
	}
	logger.Info("export store ready", "driver", "sqlite", "path", exportStore.Path(), "files", exportSvc.FileDir())

	importStore, err := importstore.Open(cfg.ImportDBPath)
	if err != nil {
		logger.Error("import sqlite open failed", "path", cfg.ImportDBPath, "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := importStore.Close(); err != nil {
			logger.Error("import sqlite close", "err", err)
		}
	}()
	importSvc, err := dataimport.New(importStore, dataimport.DataSources{
		Watchlist: watchStore,
		Alerts:    alertStore,
		Scanner:   scannerStore,
		Portfolio: portfolioStore,
	}, dataimport.Options{
		FileDir: cfg.ImportFileDir,
		FileTTL: cfg.ImportFileTTL,
	})
	if err != nil {
		logger.Error("import service init failed", "err", err)
		os.Exit(1)
	}
	logger.Info("import store ready", "driver", "sqlite", "path", importStore.Path(), "files", importSvc.FileDir())

	accountStore, err := accountstore.Open(cfg.AccountDBPath)
	if err != nil {
		logger.Error("account sqlite open failed", "path", cfg.AccountDBPath, "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := accountStore.CloseDB(); err != nil {
			logger.Error("account sqlite close", "err", err)
		}
	}()
	apiKeySvc := apikey.New(accountStore)
	accountSvc := account.New(accountStore, account.DataPurgeDeps{
		Watchlist: watchStore,
		Alerts:    alertStore,
		Scanner:   scannerStore,
		Exports:   exportStore,
		Imports:   importStore,
		APIKeys:   accountStore,
		Paper:     portfolioSvc,
		PriceDiff: priceDiffSvc,
	})
	watchSvc.SetAccountChecker(accountSvc)
	portfolioSvc.SetAccountChecker(accountSvc)
	scannerSvc.SetAccountChecker(accountSvc)
	priceDiffSvc.SetAccountChecker(accountSvc)
	logger.Info("account store ready", "driver", "sqlite", "path", accountStore.Path(), "grace", domain.AccountCloseGrace.String())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	nLiq := futureshist.RestoreBook(ctx, liqBook, futuresHist, time.Now().UTC())
	if nLiq > 0 {
		logger.Info("futures liquidations restored", "events", nLiq)
	}
	nTaker := 0
	fromTaker := time.Now().UTC().Add(-4 * time.Hour)
	for _, sym := range histSeeds {
		recs, err := futuresStore.ListTakerBuckets(ctx, string(domain.ExchangeBybit), sym, fromTaker, time.Now().UTC())
		if err != nil || len(recs) == 0 {
			continue
		}
		bybitClient.TakerBook().LoadBuckets(recs)
		nTaker += len(recs)
	}
	if nTaker > 0 {
		logger.Info("bybit taker buckets restored", "bars", nTaker)
	}
	go liqSink.Start(ctx)
	go bybitTrades.Start(ctx)
	go (&futureshist.Worker{
		Hist:     futuresHist,
		Interval: cfg.FuturesHistoryInterval,
		Retain:   cfg.FuturesHistoryRetention,
		Logger:   logger,
	}).Start(ctx)
	go (&bookhist.Worker{
		Hist:     bookHist,
		Interval: cfg.OrderBookHistoryInterval,
		Retain:   cfg.OrderBookHistoryRetention,
		Logger:   logger,
	}).Start(ctx)

	go realtimeHub.Start(ctx)

	orderFiller := &portfolio.OrderFiller{
		Portfolio: portfolioSvc,
		Market:    marketSvc,
		Interval:  cfg.PortfolioOrderCheckInterval,
		Logger:    logger,
	}
	go orderFiller.Start(ctx)

	recurringBuyWorker := &portfolio.RecurringBuyWorker{
		Portfolio: portfolioSvc,
		Interval:  cfg.RecurringBuyInterval,
		Logger:    logger,
	}
	go recurringBuyWorker.Start(ctx)

	snapshotWorker := &portfolio.SnapshotWorker{
		Portfolio: portfolioSvc,
		Interval:  cfg.PortfolioSnapshotInterval,
		Retention: cfg.PortfolioSnapshotRetention,
		Logger:    logger,
	}
	go snapshotWorker.Start(ctx)

	marginInterestWorker := &portfolio.MarginInterestWorker{
		Portfolio: portfolioSvc,
		Interval:  cfg.MarginInterestInterval,
		Logger:    logger,
	}
	go marginInterestWorker.Start(ctx)

	scannerChecker := &scanner.Checker{
		Scanner:  scannerSvc,
		Interval: cfg.ScannerCheckInterval,
		Logger:   logger,
	}
	go scannerChecker.Start(ctx)

	priceDiffChecker := &pricediff.Checker{
		Service:  priceDiffSvc,
		Interval: cfg.PriceDiffCheckInterval,
		Logger:   logger,
	}
	go priceDiffChecker.Start(ctx)

	backtestWorker := &scanner.BacktestWorker{
		Scanner:  scannerSvc,
		Interval: 2 * time.Second,
		Logger:   logger,
	}
	go backtestWorker.Start(ctx)

	exportWorker := &exportsvc.Worker{
		Export:   exportSvc,
		Interval: cfg.ExportWorkerInterval,
		Logger:   logger,
	}
	go exportWorker.Start(ctx)

	importWorker := &dataimport.Worker{
		Import:   importSvc,
		Interval: cfg.ImportWorkerInterval,
		Logger:   logger,
	}
	go importWorker.Start(ctx)

	accountPurge := &account.PurgeWorker{
		Accounts: accountSvc,
		Interval: cfg.AccountPurgeInterval,
		Logger:   logger,
	}
	go accountPurge.Start(ctx)

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

	aiClient := aiagent.New(cfg.AIServiceURL, cfg.AITimeout).WithServiceToken(cfg.AIServiceToken)

	swingSvc := swing.New(marketSvc, watchSvc)

	// MCP tools run in-process (same binary / same port as REST). Optional via MCP_ENABLED.
	var mcpHTTP http.Handler
	if cfg.MCPEnabled {
		mcpServer := mcpx.NewInProcessServer(marketSvc, watchSvc, alertSvc, portfolioSvc, scannerSvc, exportSvc, importSvc, priceDiffSvc, apiKeySvc, accountSvc, swingSvc)
		mcpHTTP = mcpx.NewHTTPHandler(mcpServer)
	}

	handler := httpx.NewRouterWithOptions(marketSvc, watchSvc, httpx.RouterOptions{
		RateLimitRPS:     cfg.RateLimitRPS,
		RateLimitBurst:   cfg.RateLimitBurst,
		CORSAllowOrigins: cfg.CORSAllowOrigins,
		APIAuthToken:     cfg.APIAuthToken,
		APIKeys:          apiKeySvc,
		MCPHandler:       mcpHTTP,
		AI:               aiClient,
		AITimeout:        cfg.AITimeout,
		Alerts:           alertSvc,
		Portfolio:        portfolioSvc,
		PriceDiff:        priceDiffSvc,
		Scanner:          scannerSvc,
		Swing:            swingSvc,
		Export:           exportSvc,
		Import:           importSvc,
		Accounts:         accountSvc,
		Realtime:         realtimeHub,
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

	if binanceDelist {
		go (&delistjob.Runner{
			Source:     binanceClient,
			Store:      delistStore,
			Interval:   cfg.DelistRefreshEvery,
			RunOnStart: cfg.DelistRefreshOnStartup,
			Logger:     logger,
			Exchange:   domain.ExchangeBinance,
		}).Start(ctx)
		logger.Info("binance delist schedule refresh enabled",
			"every", cfg.DelistRefreshEvery.String(),
			"on_startup", cfg.DelistRefreshOnStartup,
		)
	} else {
		logger.Info("binance delist schedule disabled (set BINANCE_API_KEY to enable hourly refresh)")
	}
	go (&delistjob.Runner{
		Source:     bybitClient,
		Store:      delistStore,
		Interval:   cfg.DelistRefreshEvery,
		RunOnStart: cfg.DelistRefreshOnStartup,
		Logger:     logger,
		Exchange:   domain.ExchangeBybit,
	}).Start(ctx)
	logger.Info("bybit delist announcement refresh enabled",
		"every", cfg.DelistRefreshEvery.String(),
		"on_startup", cfg.DelistRefreshOnStartup,
	)

	go binanceLiq.Start(ctx)
	go bybitLiq.Start(ctx)
	go marketSvc.StartWallSampler(ctx)
	go marketSvc.StartHeatmapWarmer(ctx)
	logger.Info("order heatmap warmer started", "venues", []string{"binance", "coinbase", "bybit"})

	alertChecker := &pricealert.Checker{
		Alerts:   alertSvc,
		Market:   marketSvc,
		Books:    marketSvc,
		Accounts: accountSvc,
		Interval: cfg.AlertCheckInterval,
		Logger:   logger,
	}
	go alertChecker.Start(ctx)

	webhookClient := &http.Client{Timeout: cfg.WebhookHTTPTimeout}
	// Deliverer hardens CheckRedirect; explicit client keeps timeout from config.
	webhookDeliverer := &pricealert.Deliverer{
		Alerts:      alertSvc,
		Accounts:    accountSvc,
		HTTP:        webhookClient,
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
				Identities:      accountStore,
				Accounts:        accountSvc,
				Portfolio:       portfolioSvc,
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
		// Must cover AI multi-agent turns (AITimeout, default 300s) plus a small buffer.
		WriteTimeout: cfg.AITimeout + 30*time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		mcpPath := ""
		if cfg.MCPEnabled {
			mcpPath = "/mcp"
		}
		logger.Info("server listening",
			"addr", cfg.HTTPAddr,
			"exchanges", []string{"binance", "coinbase", "bybit", "nasdaq", "bist"},
			"mcp", mcpPath,
			"mcp_enabled", cfg.MCPEnabled,
			"api_auth", cfg.APIAuthToken != "",
			"webhook_allow_private", cfg.WebhookAllowPrivate,
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
