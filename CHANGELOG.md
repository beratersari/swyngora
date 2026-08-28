# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **AI retry status:** CLI spinner and Process stream show `Retrying {model} (2/4)…`, `failed after 4 tries`, and `Falling back to {model}…` (`docs/features/ai-assistant.md`)
- **Scanner combo rules:** pick RSI, MA crossover, and/or volume increase on one rule and choose `matchMode` `all` (every selected condition) or `any` (one is enough) (`POST /api/v1/scanner/rules`, MCP `create_scanner_rule`) (`docs/features/indicator-scanner.md`)
- **Scanner rule edit:** enable/disable a rule without deleting it, and change interval, conditions, periods, and thresholds later (`PATCH /api/v1/scanner/rules/{id}`, MCP `update_scanner_rule`)
- **Funding arb:** compare Binance vs Bybit predicted funding using each venue's published settlement clocks only (8h at 00:00/08:00/16:00 UTC). No clock in the hold window → funding is 0. `trade` and scan rows are omitted unless after-fee net is positive (`GET /api/v1/market/funding-arb`, `/scan`, MCP `get_funding_arb` / `scan_funding_arb`, Telegram `/fundingarb`) (`docs/features/funding-arb.md`)
- **Funding arb history:** settled Binance/Bybit prints in a date range, grouped into long/short stretches that beat round-trip fees. The first clock of a run is the entry signal (not collected profit); a same-clock start/end is not a winner. A direction flip at settlement still pays the old sides, then starts a new run (`GET /api/v1/market/funding-arb/history`, MCP `get_funding_arb_history`, Telegram `/fundingarb hist`)
- **Funding arb watches:** omit `symbol` to follow the funding-arb scan and notify when a new coin's after-fee net is at least `minProfit`. Same coin + same long/short while still above the floor does not re-notify; a drop then re-cross notifies again. A coin missing from the scan is re-quoted and closed only when net is really below the floor. Pause/resume and PATCH settings. Optional `durationHours` stops the follow and closes open signals when the time is over (`POST .../pause`, `POST .../resume`, `PATCH /api/v1/funding-arb/watches/{id}`, MCP `pause_funding_arb_watch` / `update_funding_arb_watch`) (`docs/features/funding-arb.md`)
- **AI model fallback:** after the existing same-model retries are exhausted, `create_agent` tries the other allowed provider (`ChatXAI` ↔ `ChatOllama`) via LangChain `ModelFallbackMiddleware`. Ollama without `XAI_API_KEY` stays retry-only.
- **AI model retries:** `create_agent` uses LangChain `ModelRetryMiddleware` (3 retries, exponential backoff) so a short Grok or Ollama blip does not abort the turn. Grok’s OpenAI-client retries are off (`max_retries=0`) so only the middleware retries.
- **Holder address names:** top wallets include a `label` when the address is a widely published attribution (Binance, Bitfinex, Robinhood, Upbit, OKX, Bybit, Crypto.com, Gate.io, Tether, Silk Road FBI, Satoshi genesis, and other public tags). Not identity proof (`docs/features/holders.md`)
- **Post-delist movement:** after a venue halt, coin detail shows a labeled off-venue last price and candles (another listed venue or CoinGecko USD) instead of inventing flat Binance bars (`docs/features/delist-schedule.md`)
- **Delist last print:** already-removed pairs keep last / high / low / volume from the venue kline at halt so Markets and coin detail are not blank (`docs/features/delist-schedule.md`)
- **Delist market cap:** scheduled delist rows missing Binance circulating supply use CoinGecko public markets so the mcap column is not empty (`docs/features/delist-schedule.md`)
- **Delist announcement date:** amber Delist tags include when the venue published the notice (`announcedAt` from Binance CMS `releaseDate` and Bybit `publishTime`), not only the halt clock (`docs/features/delist-schedule.md`)
- **Multi-exchange delist tags:** Binance official schedule (plus CMS “Will Delist” titles) and Bybit announcement dates (article HTML when the list feed is empty). Pairs that delist in the next ~31 days **or in the last 30 days** stay on the default Markets list with an amber Delist tag (and are injected if the venue already halted them). Coinbase / Nasdaq / BIST have no public calendar (`docs/features/delist-schedule.md`)
- **Price-diff executable quote:** walk the buy venue asks and sell venue bids for a size (`notional` USDT or `quantity`) and return average buy/sell, slippage, profit after fees, and the largest still-profitable size (`GET /api/v1/price-diff/quote`, `GET /api/v1/price-diff/opportunities/{id}/quote`, MCP `quote_price_diff` / `quote_price_diff_opportunity`) (`docs/features/price-diff.md`)
- **Price-diff all-venue scan:** enter one amount and rank every Binance / Coinbase / Bybit buy-sell route after live book depth and fees, including how much of that money can actually be used (`GET /api/v1/price-diff/quote/scan`, `GET /api/v1/price-diff/watches/{id}/quote`, MCP `scan_price_diff_quotes` / `quote_price_diff_watch`) (`docs/features/price-diff.md`)
- **Price-diff executable quote:** walk the buy venue asks and sell venue bids for a size (`notional` USDT or `quantity`) and return average buy/sell, slippage, profit after fees, usable money, and the largest still-profitable size (`GET /api/v1/price-diff/quote`, `GET /api/v1/price-diff/opportunities/{id}/quote`, MCP `quote_price_diff` / `quote_price_diff_opportunity`) (`docs/features/price-diff.md`)

### Changed
- **Price-diff watches:** set `notional` and `minProfit`; opportunities open only when that size fills on live books and after-fee profit (including slippage) meets the floor — not from ticker last-price gaps (`docs/features/price-diff.md`)
- **Price-diff scan:** missing venue books are listed as `unavailable` (not ranked or chosen as best); `minProfitPct` / `minProfitAmount` hide tiny fills (`docs/features/price-diff.md`)
- **Price-diff max size:** keep filling while the **total** after-fee profit is still positive, even if the next book level loses money on its own (`docs/features/price-diff.md`)

### Fixed
- **AI both-LLM error:** when Grok and the Ollama fallback both fail, the CLI prints both errors (primary then fallback), not only the last Ollama message (`docs/features/ai-assistant.md`)
- **Dashboard 24h on halted coins:** delisted rows use the same off-venue 24h **delta and %** as coin detail (other venue or CoinGecko `price_change_24h`), not a frozen home-book window or a blank; Coinbase last/open/high/low and % stay on one stats window (`docs/features/market-data.md`, `docs/features/delist-schedule.md`)
- **AI concurrent tenant bind:** overlapping chats no longer send the master token plus the other user’s `X-Client-Id` (or inherit their `canTrade`) when desk gather runs on a thread pool (`docs/features/ai-assistant.md`)
- **Scanner backtest reuse:** starting the same symbol and date range after editing the rule creates a new job instead of returning the old completed run
- **Scanner backtest snapshot:** a queued backtest keeps the rule params from start time; later edits do not change that run
- **Scanner onset hits:** live results and backtest signals fire once when a condition becomes true, not on every later bar that stays true
- **Scanner combo lookback:** combo rules fetch candles for the longest selected condition (for example MA 200) instead of a fixed 100 bars
- **Holders without `cmcUniqueId`:** Binance marketing slugs resolve CoinMarketCap detail (`?slug=`) and a positive `cdpTotalHolder` fills the count when the holder table is empty (`docs/features/holders.md`)
- **Holders empty on many coins:** CMC’s public detail omitted the holder table for the default Go HTTP/2 client; requests now use HTTP/1.1 and browser Origin/Referer. ETH-style `dailyActive`-only payloads are shown instead of 404 (`docs/features/holders.md`)
- **PIVX (and other catalog-miss UTXO coins):** Binance marketing has no `cmcUniqueId` for PIVX; holders then fall back to public Chainz CryptoID address counts and rich list (`docs/features/holders.md`)
- **Holders coverage:** coin detail tries CoinMarketCap (id then slug), Coin Metrics address counts, GeckoTerminal, Ethplorer, Routescan (Chiliz/CITY), then Tronscan (`docs/features/holders.md`)
- **Delist chart line:** announce/halt markers snap to the candle that contains the timestamp on every interval (1m–1M), not the nearest bar open (`docs/features/delist-schedule.md`)
- **Coin-detail delist venue:** a previous exchange’s delist schedule is not treated as this venue’s halt while the new schedule loads (`docs/features/coin-detail.md`)
- **Multi-book paper submits:** trade, cash, and margin tickets stay disabled until a book is selected (`docs/features/paper-trading.md`)
- **AI paper book id:** LangChain paper/account tools accept `portfolio_id` without TypeError; margin open, recurring-buy create, basket create, and cancel-all send `portfolioId` (`docs/features/ai-assistant.md`)
- **Delist chart after halt:** leftover venue klines after a midnight schedule stay visible on the home-venue chart; days after the last print come from the off-venue panel (`docs/features/delist-schedule.md`)
- **Multi-book risk limits:** GET/PUT risk brakes and buy/margin guards no longer treat the book id as the actor; a second paper book cannot 400 the first book’s limits or block its buys (`docs/features/risk-limits.md`)
- **Recurring buy overspend:** DCA sizes and fills on one last print so a ticker move between sizing and fill cannot debit more than the plan amount (`docs/features/recurring-buys.md`)
- **Paper import UUID squat:** extra books always get a new id so an unused UUID-shaped client id cannot be occupied as another tenant’s first book (`docs/features/user-data-import.md`)
- **Multi-book paper desk:** snapshot query skips until a book is selected; AI and MCP paper tools accept `portfolioId` (`docs/features/paper-trading.md`, `docs/features/ai-assistant.md`)
- **USD chart native fallback:** ETHBTC (and other non-fiat quotes) no longer plot raw BTC prices on a USD chart when FX cannot convert (`docs/features/display-currency.md`)
- **Compact limit sell:** coin-detail paper ticket keeps Buy/Sell for limit orders
- **Heatmap venue URL:** selected exchange is written to `?exchange=` so header jump search stays on Coinbase/Bybit/Nasdaq/BIST
- **Back to Markets filters:** opening a coin from a filtered list and going back restores search, quote, tag, and page
- **Symbol typeahead quote pin:** search is not forced to the venue default quote, so ETHBTC is findable
- **WS halted last-print:** price ticks include `halted` so the tape does not treat a delist last print as live (`docs/features/realtime.md`)
- **Locked-API browser/stdio auth:** user-issued keys go on REST `Authorization` and WS `?token=`; stdio MCP sends `API_AUTH_TOKEN` / `SWYNGORA_API_TOKEN`
- **Paper import first-book squat:** extra non-UUID book ids (and UUIDs already owned by another tenant) are reminted so an import cannot occupy another client’s first-book primary key (`docs/features/user-data-import.md`)
- **Nasdaq/BIST live marks:** Yahoo ticker/candle misses fail closed instead of serving an expired last-good price to paper fills and alerts; equity caches join the process cleanup tick (`docs/features/market-data.md`)
- **Symbol suggest venue switch:** jump-search / paper / alert typeahead no longer offers the previous exchange’s symbols while the new venue loads
- **Coin-detail candle first paint:** klines start immediately (no intervals waterfall), a 100-bar request paints the viewport while the 300-bar window loads, and the chart no longer waits on indicators or remounts when switching tabs (`docs/features/coin-detail.md`)
- **Coin-detail EMA on pan:** EMA overlays are computed from every loaded candle so the line continues when you scroll into older history (`docs/features/coin-detail.md`)
- **Multi-book margin adjust/repay:** after the cash move, reload the position by book id instead of treating the book UUID as the actor — a second paper book no longer 4xxs a successful debit so a retry cannot double-charge (`docs/features/paper-margin.md`)
- **AccountGate tenant match:** `X-Client-Id`, `?clientId=`, and JSON `clientId` must agree; a decoy header can no longer trade or mutate a closed tenant. Paper `PlaceOrder` also rejects a closed owner (`docs/features/account-close.md`)
- **Telegram confirm after close:** `/buy` `/sell` Confirm re-checks `RequireActive` and does not fill (`docs/features/telegram-bot.md`)
- **AI tool scope on worker threads:** tenant id and read-only `canTrade` survive LangChain threads that drop ContextVars (`docs/features/ai-assistant.md`)
- **Desk tape venue switch:** the sticky tape no longer relabels the previous venue’s prices as the newly selected venue (`frontend/src/libs/hooks/useDeskPriceTape.ts`)
- **Coin-detail pump markers:** live pump events use the current pair only so BTC arrows cannot snap onto an ETH chart (`docs/features/coin-detail.md`)
- **Similar-move scores:** a thin overlap (only one field present) can no longer look like a high match — missing selected data counts as 0, and each case lists used / missing / coverage (`docs/features/around.md`)
- **Similar-move unique events:** overlapping matches or same-move legs are one past event (highest similarity kept); 15m / 1h / 4h results report how many unique events were used (`docs/features/around.md`)
- **Similar-move unique events:** overlapping matches, shared setup windows, or chained nearby legs are one past event (highest similarity kept); 15m / 1h / 4h results report how many unique events were used (`docs/features/around.md`)
- **Similar-move unique events:** two matches are one past event only when their price-move ranges overlap — shared setup windows no longer merge different moves (`docs/features/around.md`)
- **Similar-move short horizons:** 5m (and other short) after-windows use matching candle size instead of 15m bars (`docs/features/around.md`)
- **Similar-move look-ahead:** a past match only uses tape available before that move started; each case reports `dataFrom` / `dataTo` (`docs/features/around.md`)
- **Similar-move field scores:** a metric that was not compared no longer shows `score: 0`; similarity uses only fields that actually had data, and thin cases still need `minCoverage` (`docs/features/around.md`)
- **Similar-move scores:** each case lists used / missing / coverage so a thin overlap is visible (`docs/features/around.md`)
- **Combo common rate:** a precursor combo is common only when it hits often **overall** (hits / all sampled before-windows ≥ 60%), not when it is frequent on just one side (`docs/features/around.md`)
- **Liquidity sweep levels:** a later swing high/low no longer joins an older cluster and rewrites that shelf — a sweep that already printed keeps the level as it was at the poke (`docs/features/liquidity-sweeps.md`)
- **CVD divergence runs:** a quiet gap no longer glues two same-kind splits into one long episode (`docs/features/cvd.md`)
- **Combined CVD 5m buckets:** combined uses the overlapping time range and treats a missing 5-minute slot as 0. When both venues already have 24h, combined stays `complete` even if the first shared bucket is one bar late (`docs/features/cvd.md`)
- **Combined CVD completeness:** combined CVD only uses the time range both Binance and Bybit have. It is not marked `complete` while Bybit history is still filling — a full Binance series no longer makes a Binance-heavy combined look finished (`docs/features/cvd.md`)
- **Bybit open interest double-count:** current and historical Bybit figures now use `singleOpenInterest` / `singleOpenInterestValue` (one side). The older `openInterest` field is still both sides and was ~2× the UI value; if the single field is missing we halve the bilateral figure (`docs/features/open-interest.md`)

### Changed
- **AI streaming:** Process/CLI progress uses LangChain `stream_events` v3 (`tool_calls` / `messages`); leaf tools emit once (no wrap + stream pair)
- **AI tool progress:** specialist leaf tools emit Process events via LangChain `@wrap_tool_call` middleware instead of cloning each tool
- **AI LangChain deps:** drop unused `langchain-community`; require 1.x LTS floors (`langchain>=1.0`, `langchain-core>=1.0`, `langgraph>=1.0`, `langchain-ollama>=1.0`, `langchain-xai>=1.0`)
- **Signal trigger time:** each swing setup, scanner hit, and lab signal shows the exact UTC bar time (seconds + timezone), not a rounded local stamp
- **Coin detail tabs:** header and 24h/supply stay put; chart, order book, holders, indicators, and paper trade each get their own tab (`?tab=`) so the desk is not one long stack
- **Grok default model:** `GROK_MODEL` is now `grok-4.3` (`grok-3-mini` is retired) (`docs/features/ai-assistant.md`)
- **Web desk visual system:** charcoal + amber institutional palette (IBM Plex), underline nav, and sharper chrome — no more teal/neon green (`docs/design/frontend-design-system.md`)
- **Web terminal chrome:** left venue rail, utility bar, full-bleed workspace
- **Web CoinMarketCap-style look:** light canvas, blue brand, Inter, global stats bar, consumer price table
- **Web heatmap:** CoinMarketCap-style treemap — discrete green/red tiles, centered labels, hover tooltip, market-cap size by default, fullscreen (`/heatmap`)
- **Live spot order books:** Binance, Coinbase, and Bybit keep a local book over each venue’s depth websocket; a gap or drop invalidates and resyncs instead of serving stale data (`docs/features/order-book.md`)

### Fixed
- **Holder balances:** high-supply tokens no longer show `0` / `0.004` for wallets that own a real share of supply; the table uses share × circulating supply and an estimated USD value

### Added
- **Similar-move after horizons:** pick the after-windows (`horizons=5m,30m,2h`, default 15m/1h/4h); each reports up/down, average, median, sample, and the same stats in similarity bands 40–60 / 60–80 / 80–100 (`GET /api/v1/market/around/similar`, MCP `find_around_similar`) (`docs/features/around.md`)
- **Similar-move weights and coverage:** each tape field has its own importance (default book and OI 3, takers 1.5, volume and price 1); `minCoverage` (default 60) sends thin cases to `skipped` with missing metrics instead of treating them as matches (`GET /api/v1/market/around/similar`, MCP `find_around_similar`) (`docs/features/around.md`)
- **Similar-move field pick:** choose which tape to compare (`fields=volume,book,oi` skips price) (`GET /api/v1/market/around/similar`, MCP `find_around_similar`) (`docs/features/around.md`)
- **Similar past moves:** compare the current tape to the setup before past important moves and show what price did after the closest cases (`GET /api/v1/market/around/similar`, MCP `find_around_similar`) (`docs/features/around.md`)
- **Move precursor combos:** conditions that often fire together before a move (volume + book + OI, …) and whether that group leans up or down (`GET /api/v1/market/around/precursors`, MCP `find_around_precursors`) (`docs/features/around.md`)
- **Move precursors:** scan important up/down legs and list what often changed in the tape before them (`GET /api/v1/market/around/precursors`, MCP `find_around_precursors`) (`docs/features/around.md`)
- **Important moves:** find the strongest recent up and down legs on a coin and attach the around-the-move tape for each (`GET /api/v1/market/around/moves`, MCP `find_around_moves`) (`docs/features/around.md`)
- **Compare two times:** how two moves of the same coin differed — price, volume, vs typical, order book, OI, funding, liquidations, and sweeps (`GET /api/v1/market/around/compare`, MCP `compare_around`) (`docs/features/around.md`)
- **Around a time:** pick a coin and a time to see what happened before, during, and after that move — price, volume, buy/sell, VWAP, volume vs typical, POC, sweeps, and stored book/futures history (`GET /api/v1/market/around`, MCP `get_around`) (`docs/features/around.md`)
- **VWAP:** volume-weighted average price from a start time (or window) to now; last vs VWAP and total volume; Binance, Bybit, and combined (`GET /api/v1/market/vwap`, MCP `get_vwap`) (`docs/features/vwap.md`)
- **Volume surge:** latest 5m / 15m / 1h quote volume versus that coin's own typical (median), with buy/sell split; scan ranks coins well above typical (`GET /api/v1/market/volume-surge`, `.../scan`, MCP `get_volume_surge` / `scan_volume_surges`) (`docs/features/volume-surge.md`)
- **Liquidity sweeps:** poke through a prior high or low that had already turned price, then a return; level, excursion, time to reclaim, and volume (`GET /api/v1/market/liquidity-sweeps`, MCP `get_liquidity_sweeps`) (`docs/features/liquidity-sweeps.md`)
- **Absorption:** large market buys or sells versus how far price actually moved; which side is absorbing (`bid` / `ask`) and how strong (0–100); 15m / 1h / 4h / 24h plus the current run (`GET /api/v1/market/absorption`, MCP `get_absorption`) (`docs/features/absorption.md`)
- **Volume profile:** traded quote volume by price over a chosen range — total plus buy/sell, point of control, and the 70% value area — for Binance, Bybit, and combined (`GET /api/v1/market/volume-profile`, MCP `get_volume_profile`) (`docs/features/volume-profile.md`)
- **Spot vs futures CVD:** separate spot and futures tapes plus `spotFutures` when they move opposite ways; Binance spot from kline taker-buy volume, Bybit spot from live trades (`GET /api/v1/market/cvd`, MCP `get_cvd`) (`docs/features/cvd.md`)
- **CVD windows and venue split:** 15m / 1h / 4h / 24h CVD change (not only the latest total); `venueSplit` when Binance CVD rises and Bybit falls (or the reverse); divergence includes duration and how far price vs CVD moved (`GET /api/v1/market/cvd`, MCP `get_cvd`) (`docs/features/cvd.md`)
- **CVD divergence and venue share:** each 5m point and 1h / 4h / 24h window flags price-up/CVD-down or price-down/CVD-up; combined CVD shows how much Binance and Bybit each added (`GET /api/v1/market/cvd`, MCP `get_cvd`) (`docs/features/cvd.md`)
- **CVD:** cumulative market-buy minus market-sell notional over time versus price (1h / 4h / 24h confirms, opposite, or absorption) for Binance, Bybit, and combined (`GET /api/v1/market/cvd`, MCP `get_cvd`) (`docs/features/cvd.md`)
- **Iceberg refill:** detect when visible buy or sell size at one price is eaten and a similar clip comes back, repeatedly — both sides (`GET /api/v1/market/orderbook/icebergs`, MCP `get_orderbook_icebergs`; live walls get `behavior=iceberg`) (`docs/features/icebergs.md`)
- **Order-book history:** 1-minute samples of bid/ask levels, spread, liquidity, imbalance, and walls; look up a time and compare two times to see which price levels gained or lost liquidity (`GET /api/v1/market/orderbook/history`, `.../history/compare`, MCP `get_orderbook_history` / `compare_orderbook_history`) (`docs/features/book-history.md`)
- **Whale trades:** clustered large futures buys/sells (aggressive long/short) and liquidations, sorted biggest first, with average price, first/last time, total size, and a flag when the print is large versus circulating market cap (`GET /api/v1/market/whales`, MCP `get_whale_trades`) (`docs/features/whales.md`)
- **Support and resistance:** price-history + volume zones checked against the live order book; distance, test count, nearby bid/ask liquidity, and a breakout score from volume, book thickness, and taker flow (`GET /api/v1/market/levels`, MCP `get_support_resistance`) (`docs/features/levels.md`)
- **Market snapshot:** price, volume, market cap, open interest, funding, long/short, and taker buy/sell together for one coin, with 1h / 4h / 24h changes (`GET /api/v1/market/snapshot`, MCP `get_market_snapshot`) (`docs/features/snapshot.md`)
- **Price volatility:** how much a coin moved over 1h / 4h / 24h (net + high–low range), whether that range is higher or lower than normal and expanding or shrinking, and whether the coin is jumpy or calm versus BTC/ETH (`GET /api/v1/market/volatility`, MCP `get_price_volatility`) (`docs/features/volatility.md`)
- **Market breadth:** how many of the liquid coins we follow are up vs down over 1h / 4h / 24h (count and percent), plus whether BTC/ETH are moving with the pack or a few large coins are carrying (`GET /api/v1/market/breadth`, MCP `get_market_breadth`) (`docs/features/breadth.md`)
- **Price correlation vs BTC/ETH:** how similarly a coin has been moving with Bitcoin and Ethereum over 1h / 4h / 24h (correlation, beta, same-direction share, lead/lag) (`GET /api/v1/market/correlation`, MCP `get_price_correlation`) (`docs/features/correlation.md`)
- **Futures basis:** perp vs spot/index premium or discount on Binance and Bybit, dollar + percent, expanding/shrinking, plus funding/OI read and venue agreement (`GET /api/v1/market/basis`, MCP `get_basis`) (`docs/features/basis.md`)
- **Taker flow:** aggressive futures buy vs sell volume for 5m / 1h / 4h on Binance USD-M and Bybit linear, with delta and a short read vs price, OI, and funding (`GET /api/v1/market/taker-flow`, MCP `get_taker_flow`) (`docs/features/taker-flow.md`)
- **Venue divergence:** compare Binance vs Bybit on OI, funding, crowding, and positioning; flag same vs opposite with a short why (`GET /api/v1/market/venue-divergence`, MCP `get_venue_divergence`) (`docs/features/venue-divergence.md`)
- **Positioning (price + OI):** long buildup / short buildup / long unwinding / short covering for Binance and Bybit, with short reasons and a combined market direction (`GET /api/v1/market/positioning`, MCP `get_positioning`) (`docs/features/positioning.md`)
- **Squeeze risk:** long-squeeze and short-squeeze scores (0–100) for any coin on Binance USD-M and Bybit linear, with reasons and an OI-weighted combined view (`GET /api/v1/market/squeeze-risk`, MCP `get_squeeze_risk`) (`docs/features/squeeze-risk.md`)
- **Liquidation hunt (hypothetical):** per-venue model of where long/short pressure sits if spot is walked up or down, how much visible spot size that takes, and a rough desk result (book-only unwind vs cascade exit). Not evidence of exchange behavior (`GET /api/v1/market/liquidation-hunt`, MCP `estimate_liquidation_hunt`) (`docs/features/liquidation-hunt.md`)
- **Durable futures history:** SQLite archive of open interest, funding, long/short, and liquidations for Binance USD-M and Bybit linear; 5m sampler, restart-safe, no duplicate rows, per-venue fail-soft (`GET /api/v1/market/futures-history`, MCP `get_futures_history`) (`docs/features/futures-history.md`)
- **Futures long/short ratio:** share of accounts that are long vs short plus recent 5m history for Binance USD-M and Bybit linear; also attached on the open-interest payload (`GET /api/v1/market/long-short-ratio`, MCP `get_long_short_ratio`, Telegram `/ls`) (`docs/features/long-short-ratio.md`)
- **Futures funding rate:** predicted next perpetual funding plus recent settlements for Binance USD-M and Bybit linear; also attached on the open-interest payload (`GET /api/v1/market/funding-rate`, MCP `get_funding_rate`, Telegram `/funding`) (`docs/features/funding-rate.md`)
- **Futures open interest:** current outstanding size plus 5m / 1h / 4h / 24h change (contracts and USDT notional) from Binance USD-M and Bybit linear perpetual; `exchange=all` sums both (`GET /api/v1/market/open-interest`, MCP `get_open_interest`, Telegram `/oi`) (`docs/features/open-interest.md`)
- **Crypto holders:** coin detail shows on-chain wallet count, top 10/50/100 concentration, and top addresses from `GET /api/v1/market/holders` / MCP `get_holders` (`docs/features/holders.md`)
- **Grok reasoning:** `GROK_REASONING_EFFORT` (`none` | `low` | `medium` | `high`), default `low`; Grok calls only, Ollama unchanged (`docs/features/ai-assistant.md`)
- **Order heatmap:** coin detail paints a Bookmap-style liquidity map (thermal size, wide current-book column, history to the left) from `GET /api/v1/market/orderbook/heatmap` / MCP `get_orderbook_heatmap`. Not executed volume.
- **Futures liquidations:** rolling 5m / 1h / 4h / 24h long vs short notional, count, and biggest hit from Binance USD-M and Bybit linear perpetual streams; `complete` / `coverageSeconds` count only live websocket time per coin and venue (`GET /api/v1/market/liquidations`, MCP `get_liquidations`) (`docs/features/liquidations.md`)
- **Liquidity score:** 0–100 grade from live bid/ask notional only in ±0.1 / ±0.5 / ±1% bands the book actually reaches; market-wide uses the common venue range (`GET /api/v1/market/orderbook/liquidity`, MCP `get_market_liquidity`) (`docs/features/order-book.md`)
- **Wall persistence:** order-book walls now include `behavior` (`short` / `persistent` / `suspicious`) plus how long they have been present and how often they flicker (`docs/features/order-book.md`)
- **Market impact / slippage:** simulate a market buy or sell by walking live order-book levels; returns average fill, slippage vs mid/best, and price impact as the new best ask/bid after leftover size (0 if the touch level is not fully consumed). If visible depth is wiped, impact is not calculated (`GET /api/v1/market/orderbook/impact`, MCP `estimate_market_impact`) (`docs/features/order-book.md`)
- **Market-wide order book:** combined Binance + Coinbase + Bybit live depth; totals use a symmetric ±% both sides can reach (requested band only when all venues cover it both ways) (`GET /api/v1/market/orderbook/combined`, MCP `analyze_market_orderbook`) (`docs/features/order-book.md`)
- **Order-book alerts:** create imbalance or wall alerts on the existing alert API; the background checker uses the live local book and does not re-fire while the same condition stays true (`POST /api/v1/alerts` `kind=imbalance|wall`, MCP `create_orderbook_alert`) (`docs/features/price-alerts.md`)
- **Spot order-book analysis:** buy/sell pressure, notional imbalance, and large walls from live depth within ±`rangePct` of mid (default 2%, not only the top rows) on Binance, Coinbase, and Bybit (`GET /api/v1/market/orderbook`, MCP `analyze_spot_orderbook`) (`docs/features/order-book.md`)
- **Spot order book:** grouped bid/ask depth with buy/sell wall flags and suggested price steps (`GET /api/v1/market/orderbook`, MCP `get_spot_orderbook`); grouping is done on the backend; coin detail shows the ladder next to the chart (`docs/features/order-book.md`)
- **Paper trade idempotency keys:** `Idempotency-Key` header or `idempotencyKey` on market/pending/margin place and margin close; same key + same request returns the original fill; different request with that key is 409 (`docs/features/paper-trading.md`)
- **Paper portfolio export/import:** backup and restore owned books (cash, positions, trades, open orders, tax lots, recurring buys, margin, shares) via the existing export/import jobs; merge skips existing id/name, replace wipes owned books first (`docs/features/user-data-export.md`, `docs/features/user-data-import.md`)
- **Paper trading fees and slippage:** per-exchange taker fee and adverse slippage on market, pending, recurring, and margin fills; buy cash and tax-lot cost include the fee; sell realized PnL is after the fee; pending buy reservations cover worst-case slip + fee (`GET /api/v1/portfolio/trading-costs`, MCP `get_paper_trading_costs`) (`docs/features/paper-trading.md`)
- **Paper tax lots:** each buy opens a lot (qty, price, time); sells choose FIFO or LIFO (`lotMethod`); partial lots keep remaining qty; market, pending, and recurring fills share the same ledger (`GET /api/v1/portfolio/lots`) (`docs/features/paper-trading.md`)
- **Realtime WebSocket:** `GET /api/v1/ws` subscribe/unsubscribe selected coin prices and one paper portfolio’s order/position/cash events; reconnect resubscribes with snapshots; access checked per book; frontend uses the stream instead of polling when connected (`docs/features/realtime.md`)
- **Paper cash transfer:** owner-only move of available cash between your own books (`POST /api/v1/portfolio/transfers`); both histories get `transfer_out`/`transfer_in` with counterpart book name (not a deposit/withdrawal); MCP `transfer_portfolio_cash`; Telegram `/transfer` (`docs/features/paper-trading.md`)
- **Paper portfolio sharing:** owner grants `viewer` (read snapshot/trades/performance) or `trader` (also place/cancel orders) to another clientId; deposit/withdraw/delete/share stay owner-only; `POST/GET/PATCH/DELETE /api/v1/portfolio/shares` and `GET /api/v1/portfolios/shared` (`docs/features/paper-trading.md`)
- **Multiple named paper portfolios:** each client can create up to 20 books with their own cash, positions, orders, and performance; `GET /api/v1/portfolios`, `PATCH`/`DELETE /api/v1/portfolios/{id}`, optional `portfolioId` / `X-Portfolio-Id` on actions; MCP `list_portfolios` / `rename_portfolio` / `delete_portfolio`; Telegram `/portfolio list` `/use` (`docs/features/paper-trading.md`)
- **Per-account API keys:** named `read` or `trade` keys (`POST/GET/DELETE /api/v1/account/api-keys`); bots use `swy_…` secrets instead of the process master token (`docs/features/api-keys.md`)
- **Paper cash in/out:** `POST /portfolio/deposits` and `/withdrawals` plus `GET /cash-movements` history; deposits do not count as P&L; MCP + Telegram `/deposit` `/withdraw` `/cash` (`docs/features/paper-trading.md`)
- **Telegram paper trading:** `/portfolio` view/create plus `/buy` and `/sell` with last-price preview and Confirm/Cancel inline buttons (`docs/features/telegram-bot.md`)
- **AI chat sources:** assistant returns `references` (title + URL) from web/X research; `/ai` shows a Sources list; `web_research` uses Wikipedia + Google News RSS + CoinGecko + HN (DuckDuckGo is timed/optional) (`docs/features/ai-assistant.md`)
- **Paper portfolio performance:** `GET /api/v1/portfolio/performance?period=1d|1w|1m|3m` returns equity time series plus period P&L amount and percent; 15m snapshot worker; MCP `get_portfolio_performance` (`docs/features/paper-trading.md`)
- **Desk motion:** page enter, watchlist star pop, live price flash, connection pulse, hover lifts; ticker tape GPU + pause when tab hidden; Markets/Watchlist table columns memoized to cut poll jank (`frontend/src/styles/tokens/motion.ts`)
- **Swing signals desk (`/signals`):** watchlist scanner UI — confluence setup cards (EMA + RSI + volume), rule CRUD + 4h swing stack, live hits, historical lab with 1/5/20d returns; coin-detail chart markers (`docs/features/indicator-scanner.md`)
- **Web desk chrome:** sticky two-row header, brand mark, jump-to-pair search, live/offline connection pill, volume ticker tape, 1600px desk canvas, shared page headers (`frontend/`)

### Fixed
- **Order heatmap 15m window:** the live tape now keeps 30 minutes of 1s samples so a 15m lookback is not clipped at ~10–12 minutes
- **Paper recurring buys:** the worker now fills the book that owns the plan after a second paper portfolio exists (Main and named books); it no longer records `failed` and skips the period
- **AI chat layout:** assistant reply renders as markdown; thinking is a collapsed step list; tool chips show names only (no JSON dumps)
- **Dark UI contrast:** exchange/status chips no longer use Ant `Tag color="processing"` (unreadable green-on-green); BrandTag + global Tag overrides; tertiary text/placeholders raised off stone (`frontend/`)
- **Paper cash races:** serialize portfolio cash/position mutations per `clientId` so concurrent multi-symbol fills and HTTP orders cannot last-write-wins balances (`docs/features/paper-trading.md`)
- **MCP + account close:** in-process tools with `clientId` now require an active account (HTTP `/mcp` still skips header AccountGate because clientId is in the tool body)
- **Coin detail history:** reset paged candles on series change during render (no mixed old/new chart); pump-threshold toggle no longer sticks `historyLoading`
- **Docs:** paper-margin equity/liq/debt, risk-limit UTC first-touch baseline + DCA `order failed`, account-close MCP note

### Added
- **Paper order amend:** `GET`/`PATCH /api/v1/portfolio/orders/{id}` to change trigger price and remaining size of an open GTC limit/stop without canceling; same id, reservation recalc, 409 on concurrent fill; MCP `get_portfolio_order` / `amend_portfolio_order` (`docs/features/paper-trading.md`)
- **Paper cancel-all:** `POST /api/v1/portfolio/orders/cancel-all` cancels every open paper order or one market (`symbol`); MCP `cancel_all_portfolio_orders`
- **Named recurring buys:** plan `name` plus `weekly`+`weekday`, `monthly`+`dayOfMonth` (salary day), and `interval`+`intervalHours` (e.g. every 12h); `PATCH` to rename/reschedule; MCP `update_recurring_buy`
- **Allocation baskets:** named target mixes (e.g. 50/30/20) with live drift; **user-triggered** rebalance only (`POST .../baskets/{id}/rebalance`); no auto-rebalance (`docs/features/allocation-baskets.md`)
- **Risk limits:** optional `maxDailyLossPct` and `maxAssetWeightPct`; block new spot buys and new margin opens only — never close positions; GET returns live status for the settings screen (`docs/features/risk-limits.md`)
- **Mobile home dashboard** (`mobile/`): Home tab shows live favorites strip, top movers, highest volume, pump teaser, and quick chips; pull-to-refresh and AppState poll pause (`docs/features/mobile-home-dashboard.md`)
- **Mobile AI chat** (`mobile/`): Ask tab multi-turn chat via `POST /api/v1/ai/chat`; OpenAPI `postAiChat`; RTK mutation; context from coin detail; 503 UX when AI offline (`docs/features/mobile-ai-chat.md`)
- **Mobile chart pump overlays** (`mobile/`): coin detail chart toggles for pump/dump markers and high–low margin price lines (same pump events as the list section)
- **Mobile chart history pan** (`mobile/`): coin detail candle chart loads older bars when scrolling left (`endTime` pages, time-range viewport so empty left fills on all intervals; indicators limit follows series length)
- **Mobile icons** (`mobile/`): Lucide (`lucide-react-native` + `react-native-svg`) with `atoms/icon` wrapper; tab bar, favorites star, filters, back, language
- **Mobile localization** (`mobile/`): flexible i18next setup (`libs/i18n`) with en/tr catalogs, feature namespaces, language switcher on Home, tab/page copy wired for core screens
- **Mobile batch indicators** (`mobile/`): Favorites and Markets lists enrich rows with latest RSI via `POST /api/v1/market/indicators/batch` (≤50 symbols/exchange, AppState pause, partial-failure safe); coin detail series unchanged (`docs/features/mobile-batch-indicators.md`)
- **Mobile pumps radar** (`mobile/`): Pumps tab scans top-volume pairs via `/pumps/scan`; coin detail shows pump/dump events via `/pumps`; filters for lookback/threshold/direction (`docs/features/mobile-pumps.md`)
- **Mobile watchlist** (`mobile/`): star/unstar on Markets + Coin detail; Watchlist tab with ticker quotes; device clientId + optimistic local/server merge (`docs/features/mobile-watchlist.md`)
- **Mobile coin detail** (`mobile/`): CoinDetailPage from markets row — ticker/supply stats, interval toolbar, Lightweight Charts OHLCV + EMA overlays, RSI pane; RTK detail endpoints; Atomic organisms (`docs/design/mobile-coin-detail.md`)
- **Mobile multi-exchange markets dashboard** (`mobile/`): Markets tab lists Binance/Coinbase/Bybit spot via RTK Query; filters, sort, pagination, AppState poll pause; View+ViewModel (`docs/design/mobile-markets-dashboard.md`)
- **Product mobile scaffold** (`mobile/`): React Native (no Expo) + **react-native-web** for Chrome (`npm run web` → http://localhost:5180); Atomic components; modules own pages with View+ViewModel; RTK Query + OpenAPI codegen; brand tokens shared with frontend


### Added
- **Frontend i18n:** i18next + react-i18next with `en`/`tr` catalogs (`libs/i18n/`), language switcher, Ant Design locale sync; UI copy moved off hard-coded English

### Changed
- **Frontend layout (Option A):** removed `src/features/`; domain UI lives under `components/organisms/`; pages under `components/pages/` only; decision `project-management/decisions/002-no-features-folder-atomic-only.md`
- **Frontend brand palette:** navy/indigo blues replaced with green system — Rich Black `#000F0F`, Dark Green `#032221`, Bangladesh Green `#03624C`, Mountain Meadow `#4FD4A5`, Caribbean Green `#00FF81`, Anti-Flash White `#F1F7F6`, secondary Pine–Mint, neutrals Stone/Pistachio (`frontend/src/styles/tokens/colors.ts`)

### Added
- **Multi-agent AI assistant** (`ai/`): LangGraph orchestrator with market, web, X/Twitter-signal, and analyst specialists; Ollama + Grok only
- **Go MCP** embedded in `cmd/server` at `/mcp` (in-process tools); optional stdio `cmd/mcp`
- **Telegram `/ask` and `/ai`** plus `POST /api/v1/ai/chat` proxy; optional AI auto-start child process
- AGENTS.md rule: new agent-useful features must ship with MCP tools in the same MR; restart backend after server changes
- Docs: `docs/features/ai-assistant.md`, ADR 0002 multi-agent + MCP
- **Product coin detail + indicators UI** (`frontend/`): route `/markets/:exchange/:symbol`, Lightweight Charts OHLCV + EMA overlays, RSI pane (30/70 bands), ticker/supply stats; RTK endpoints for candles, ticker, supply, intervals, indicators (`docs/features/coin-detail.md`)
- **Frontend design + scaffold:** system design, project-init epic design, multi-exchange spot markets feature doc, `frontend/` libs/Atomic skeleton, GitLab PM definitions (`docs/pm/`)
- **Frontend stack decisions:** **Ant Design** + **TradingView Lightweight Charts**; local **`project-management/`** board (INIT/MKT tasks)
- **Product frontend scaffold** (`frontend/`): Vite + React + TS, Ant Design dark theme, Lightweight Charts host, RTK Query `libs/api`, OpenAPI codegen, Markets placeholder; scripts `dev`/`build`/`test`/`codegen:api`
- **Frontend styling:** **styled-components** only; colocated `*.styles.ts` (no CSS modules)
- **Frontend design system:** brand palette `#111844` / `#4B5694` / `#7288AE` / `#EAE0CF`, type scale (DM Sans + JetBrains Mono), `Text` + `Skeleton` atoms, shared `isLoading` contract (`docs/design/frontend-design-system.md`)
- **Multi-exchange spot markets UI** (`frontend/`): exchange tabs, search/quote/tag filters, sortable Ant Table, URL query sync, 10s poll with visibility pause (`GET /exchanges`, `/tags`, `/spot`)
- **Telegram bot** integrated in backend (`transport/telegram`): market commands, watchlist, `/lowmcap` / `/lowmcap all` (no AI; enabled via `TELEGRAM_BOT_TOKEN`)
- Technical indicators: **RSI** (Wilder) and **EMA** via `GET /api/v1/market/indicators`
- OpenAPI for **`POST /api/v1/market/indicators/batch`**; document `exchange` on ticker/intervals/tags
- **Watchlist** API (`/api/v1/watchlist`) + dashboard stars / filter (client id + localStorage)
- Multi-exchange spot data: **Coinbase** + **Bybit** via `?exchange=`; `GET /api/v1/market/exchanges`
- Binance product-catalog **tags** on spot markets: `tags` field, `tag`/`tags` filter (OR), `sort=tags`, and `GET /api/v1/market/tags`
- Simple frontend tag filter dropdown and Tags column

### Fixed
- Watchlist `Add` max-items race (enforced under store lock); reject empty/`default` clientId
- Indicator series no longer collapses bad closes (fail instead of inventing RSI/EMA gaps)
- Coinbase symbol normalization shared with watchlist (`BTCUSD` → `BTC-USD`)
- Empty/unparseable volumes sort nulls-last (not as zero); tag sort uses sorted tag join
- Binance product-meta/exchangeInfo singleflight detaches caller context
- Bybit: only `Trading` instruments; candle `CloseTime` from interval; safer error mapping
- Coinbase: hard-fail invalid OHLC rows; error if product pagination hits safety cap
- Telegram: fail-closed without allowlist; `/rsi` accepts either arg order; escape HTML errors; free-text not `/price`
- JSON POST/PUT body size capped (1 MiB); CORS origin allowlist via `CORS_ALLOW_ORIGINS`
- Simple frontend: Coinbase dashboard uses `quote=USD`; watchlist DELETE tombstones prevent re-merge
- Market-cap ranking: nulls last, no infinite max without price, collapse multi-quote pairs, refuse empty-supply mcap sorts
- Supply snapshot: atomic replace, USDT-pair preference, strict bapi success checks, last-good retained on failure; **retry with backoff** on failed refresh; default **48h safety TTL**
- Non-crypto filter: fail-closed on empty/soft catalog **and** when catalog is down with no last-good snapshot (spot list errors instead of listing equities)
- Candle/ticker thundering herd (singleflight); unbounded candle cache keys (range queries uncached + max entries)
- Ingress per-IP rate limit with **max bucket map**; watchlist **Add max items** + **max clients**; indicator batch **process-wide** upstream semaphore
- Sanitized public API errors; zero-duration config footguns
- Simple frontend: detail-page load race (stale symbol paint); watchlist **merge** sync (no wipe after offline adds); multi-exchange star paint vs click; tiny prices no longer format as `0`; XSS formatters; supply `asOf` cues

### Security
- Telegram bot does not start with token alone unless `TELEGRAM_ALLOW_ALL=true` or chat allowlist is set
- Configurable CORS (default `*` for local dev; restrict in production)


### Changed

- Supply (circulating / total / max) comes **only from Binance** marketing symbol list; CoinGecko adapter removed
- Supply snapshot still daily @ 03:00 UTC (+ startup); request path remains cache-only
- Max supply is null when Binance does not publish a hard cap
- Spot list and supply exclude non-crypto products (`bStocks` e.g. NVDAB/TSLAB, `tCommodities` e.g. PAXG)

### Added

- Daily supply/mcap snapshot refresh (default 03:00 UTC); user requests are cache-only
- Binance spot market list with search, metric sort, and pagination (`GET /api/v1/market/spot`)

## [0.1.0] - 2026-07-25

### Added

- Go backend (N-layered) with OpenAPI contract for market data
- Binance candlesticks (`/api/v1/market/candles`) with multi-interval support
- Binance 24h ticker including base and quote volume (`/api/v1/market/ticker/24h`)
- Asset circulating / total / max supply via free CoinGecko (`/api/v1/market/supply`)
- In-memory TTL caches with background cleanup for candles, ticker, and supply
- `simple-frontend/` static test harness (product UI reserved under `frontend/`)
- Feature docs and ADR for data-source choice

[0.1.0]: https://nova.teachx.ai/trace-analysis/swyngora/-/tags/v0.1.0
