# MCP transport (`internal/transport/mcp`)

Exposes Swyngora market and watchlist HTTP APIs as **MCP tools** for AI agents.

## Tools

| Tool | Purpose |
|------|---------|
| `health` | API liveness |
| `realtime_stream_info` | WebSocket subscribe/unsubscribe protocol for live prices + paper portfolio |
| `list_exchanges` | Configured venues |
| `get_fx_rates` | Spot FX (USD base) for display conversion |
| `get_ticker` | 24h ticker |
| `get_spot_orderbook` | Live grouped spot bids/asks + walls; `analysis` from ±rangePct of mid; wall `behavior` |
| `analyze_spot_orderbook` | Buy/sell pressure, imbalance, and large walls (short / persistent / suspicious) |
| `analyze_market_orderbook` | Combined Binance+Coinbase+Bybit pressure in one shared price band |
| `get_market_liquidity` | 0–100 liquidity score from ±0.1/0.5/1% depth; per venue + market-wide |
| `get_orderbook_heatmap` | Resting bid/ask size over time. Up to 30 minutes is the live tape; longer windows (up to 30 days) use stored ladders plus brief walls and size moves |
| `get_liquidations` | Rolling 5m/1h/4h/24h long/short futures liquidations + `feed` last print / still-unfilled gaps / `missingSeconds` / missing venues |
| `get_liquidation_overview` | Market-wide 1h/4h/12h/24h long/short totals + coins ranked for a treemap + `feed` |
| `get_liquidation_levels` | CoinGlass-style 10x/25x/50x/100x price bars + walk-from-last cumulative; live bars scaled by later-print confidence and by realized/predicted amount after enough hits (`sizeScale`); `review` scores later same-venue liquidations; `symbol=all` time bars |
| `get_liquidation_cascade` | Short-burst long/short cascade vs typical (1m/5m/15m); `episodes` = start, duration, long/short notional, price; combined waves include `startedBy`, follow lag, and per-venue `sharePct` / notional |
| `scan_liquidation_cascades` | Market cascade plus coins currently bursting; `both` when the same side is cascade or hotter on Binance, Bybit, and OKX |
| `get_open_interest` | Current futures OI plus 5m/1h/4h/24h opened or closed USDT amount and percent, per venue and combined; includes `funding`. A failed or missing asked venue blanks the combined current and windows and is named in `missing`, and is not counted as zero. A window that lacks a sample on any asked venue is blank even if the others have it. `exchange=hyperliquid` is mark times coin size and is not inside `exchange=all` |
| `compare_spot_perp_liquidity` | Spot vs perp resting size at ±0.1/0.5/1% of mid; which book is thinner and which side (bid or ask) is thinner on each. Spot is Binance+Coinbase+Bybit; perp is Binance USD-M+Bybit linear+OKX. Binance and Bybit rows compare that venue's own books |
| `get_perp_depth` | Perpetual bid-below vs ask-above notional (default ±1%); Binance USD-M + Bybit linear + OKX USDT-swap + combined |
| `get_perp_depth_overview` | Total perp bid vs ask across the liquid coins we track, plus which coins make up the biggest share of that book; `bands` show how close that book is to mid (0.1% vs 1%) and who fills each distance; 15m/1h/4h bid-share vs now, who moved the mix, and whether total bid+ask got thicker or thinner (5%), plus which coins added or removed the most size; history is total bid+ask each minute for the last 6 hours with BTC's close; `chart` folds that total across the stored window, up to 30 days; moves say whether Binance, Bybit, and OKX moved together |
| `get_perp_book_history` | Stored perp bid and ask within 0.1%, 0.5%, and 1% of mid for one coin. A minute is kept only when Binance, Bybit, and OKX are all fresh. `windowSec` default 24h, max 30 days |
| `get_derivatives_grid` | Coins down the side, stored hours (`24h`) or UTC days (`7d`) across. Metric is funding, open-interest change, CVD, or liquidation notional. A cell stays blank until every expected venue has that column. A missing venue is named and is not zero. `venues` lists each exchange and its own number when that exchange has the column. `exchange=hyperliquid` fills funding and open interest only. CVD and liquidation cells stay blank and name that venue |
| `get_derivatives_board` | Liquid USDT perpetuals (default 36, max 100): mark, funding, open interest, 1h/24h open-interest change, crowd long percent, top-minus-crowd gap, day CVD, the current price-versus-CVD run, 15m print-size change, 15m order-count ratio, the farthest unusual reading, 5m liquidation heat, 1% bid share, 1h taker buy share, 24h long/short liquidations, and how far back each stored series goes. A combined cell is empty until every expected venue is fresh for that column. The 24h open-interest change uses the stored sample within 90 minutes of 24 hours ago when the live series does not reach it, and `change24hMissing` names a venue with no sample. Unusual names its venue and is not averaged. Opening that standout shows `get_coin_situation` for the symbol. `exchange=hyperliquid` is mark, open interest, and hourly funding. Liquidation, tape, and book stay blank and are named unsupported. It is not part of `exchange=all` |
| `get_options_overview` | BTC/ETH Deribit options: OI, 24h volume, put/call, DVOL + ATM IV, 30-day realized vol and `volCompare` (DVOL minus realized in vol points); term structure, call/put skew, expiry mix, expected move, max pain, dealer gamma (`netUsd` / flip / regime); `history` over hours/days including DVOL vs realized / volGap, net GEX, and flip vs spot; `moves` when OI/PCR/IV jumps vs typical or gamma / flip-side changes; `strikeOi` is call/put open-interest change at each strike over 1h/4h/24h/7d; `vsMarket` when IV or put/call disagrees with spot/futures |
| `get_funding_rate` | Predicted next perpetual funding plus recent settlements (Binance USD-M + Bybit linear + OKX USDT-swap); `pressure` is OI-weighted combined predicted funding plus 1h/4h/24h change. Combined pressure stays blank until every asked venue is in the blend and is named in `pressure.missing`. A venue with no open interest is still skipped inside a complete blend. `exchange=hyperliquid` is the hourly rate and is not inside `pressure` |
| `get_funding_arb` | Long cheaper-funding venue / short richer venue; sized after-fee payout + spot-perp gap |
| `scan_funding_arb` | Rank liquid USDT coins that beat fees on published settlements |
| `get_funding_arb_history` | Past after-fee winning stretches from settled Binance, Bybit, and OKX prints (first clock is entry only; flip clock still pays the old sides) |
| `create_funding_arb_watch` | Follow the scan (omit symbol) or one pair; notify when after-fee net ≥ minProfit |
| `list_funding_arb_watches` | List funding-arb follow watches |
| `get_funding_arb_watch` | Get one funding-arb follow watch |
| `update_funding_arb_watch` | Change minProfit and other settings without deleting |
| `pause_funding_arb_watch` | Pause a follow without deleting it |
| `resume_funding_arb_watch` | Resume a paused follow |
| `delete_funding_arb_watch` | Delete watch and its signals |
| `list_funding_arb_signals` | List min-profit crossings (open/closed/all) with predicted-vs-settled review when clocks have passed |
| `get_funding_arb_signal` | One crossing plus predicted-vs-settled review |
| `get_long_short_ratio` | Account long/short on Binance, Bybit, and OKX. Binance and OKX also return top-trader layers; `gap` is top long% minus crowd long%; `divergence` is top vs price with `since` / duration; `gapWindows` say if that spread is widening; `split` when crowd and size disagree |
| `get_futures_history` | Durable stored OI / funding / long-short / liquidation history. Optional `bucket` `5m`, `30m`, `2h`, or `4h` returns one sample per step across at most 90 days. A step with no sample is omitted. Bybit long/short has no top-trader share. `exchange=hyperliquid` is sampled open interest and hourly funding. It has no long/short or liquidation history |
| `get_history_span` | How long futures open interest, the spot book, flow-at-prices, and recorded liquidation prints have been stored for one coin. The startup archive covers the liquid set, up to 100 names. Binance tape for that list is walked back about 7 days. Venues with no rows are omitted |
| `get_unusual_context` | Funding, 24h open-interest change, completed-hour liquidations, completed-day CVD, and hourly spread versus that coin’s own stored samples. Venues stay separate. The span is the stored span. Print size stays unknown |
| `scan_unusual` | Same readings across the liquid USDT list (up to 100). Keeps only high and low bands. Typical and unknown rows are left off. Venues stay separate. Opening a row shows `get_coin_situation` for that symbol |
| `get_liquidation_max_pain` | Largest estimated liquidation pockets above (shorts) and below (longs) last price — not the hunt target |
| `estimate_liquidation_hunt` | Hypothetical per-venue hunt: spot size to reach liq zones + desk result + up/down ease scores (optional custom weight* mix summing to 100; percents stay as given — missing factors add 0 and are not redistributed; scoreCompare has separate upFactors/downFactors, usedVenues/missingVenues on combined rows, and upLargestChange/downLargestChange) and cascade paths (running fuel pool, hop strength, stall) |
| `get_liquidation_heatmap` | Labeled leverage model: price × time liquidation intensity (12h/24h/3d/7d/14d/30d/90d; Binance, Bybit, OKX, combined only when all three have open interest; own-venue prices; windows past a day read one stored open-interest, funding, and long/short sample per bucket, and a column with no sample stays empty; a stored open-interest hole uses public hourly history within the documented lookback; leverage mix; 1h/4h/12h hit review still checks the newest 2,500 prints) |
| `get_liquidation_print_heatmap` | Measured liquidation prints by price and time (12h–90d). A quiet recorded column is a real zero. `hasData` false is time that was not recorded. Combined sums a column only when Binance, Bybit, and OKX were all recorded. Each requested book includes `printAnchors` since 00:00 UTC and the current 4-hour open. An unrecorded clock stays blank and names the venue. Empty leverage mix and no review horizons |
| `get_liquidation_open_levels` | Estimated levels that stay until a later candle trades through them (12h–90d). Windows past a day read one stored open-interest sample per bucket, and a column with no sample stays empty. The first column places open interest already on the books. Later columns add only an increase. A hit is not put back. An open-interest drop closes the oldest slices and is not a liquidation. Each definite book reports still-open long, short, and long-minus-short change since 00:00 UTC and the current 4-hour open. Optional `anchor` adds the same change since that candle. A clock before the first column is not complete and is not zero. Combined is withheld, with no anchors, when any venue has no price and open interest |
| `get_squeeze_risk` | Long/short squeeze risk scores (0–100) per venue + OI-weighted combined. An errored or zero-OI venue withholds the blend (`agreement=withheld`). A zero score is not that blend |
| `get_positioning` | Price+OI regime: long/short buildup, unwinding, covering + market combined. An errored or zero-OI venue withholds the blend (`agreement=withheld`) and `missing` names it |
| `get_venue_divergence` | Binance, Bybit, and OKX: who shares a lean and which venue is the outlier |
| `get_net_position` | Net long and net short USDT over 15m / 1h / 4h from open-interest change and taker buy/sell. Positive means that side grew. An incomplete window has empty numeric strings and `complete` false. A missing venue is named and is not a flat zero |
| `get_taker_flow` | Aggressive futures buy vs sell volume (5m/1h/4h) per venue + combined |
| `get_participation` | OI USDT delta ÷ futures taker volume (15m/1h/4h); 2B volume and OI +300M is 15% new positions. A venue without a complete window blanks the combined ratio and is named in `missing`, and is not counted as zero |
| `get_volume_share` | How BTC futures volume splits across Binance / Bybit / OKX (e.g. 55/30/15) plus who gained or lost share vs the prior hour |
| `get_print_size` | Average aggressive print size (USDT per fill) 5m/15m vs the prior window; few large vs many small |
| `get_print_size_overview` | Same average summed across the recorder's liquid USDT list (default 100, seeds first), versus the window just before |
| `get_bot_tracker` | Same-size aggressive prints repeating at least 4 times inside 60s over the last 5m; buy, sell, and delta count only those clips |
| `get_order_count_ratio` | (buy print count − sell print count) / (buy + sell) from −1 to +1 over 1m/5m/15m |
| `get_order_count_overview` | Same ratio summed across that recorder list. Coin leans are one vote each |
| `get_flow_at_prices` | Buy vs sell USDT, print count, and delta at each price. `window` 5m/15m/1h/4h/1d/3d/7d or `startTime`/`endTime` (max 7d). Live tape is about 1h (Bybit: newest ~1000 trades); stored minutes while the process runs cover the longer lookback. Optional `minPrice`/`maxPrice`/`tickSize` |
| `get_candle_footprint` | Buy versus sell inside each candle. `interval` matches the chart. Optional `startTime`/`endTime` (max 30 days and 400 candles). Stored minutes, about 30 days. The open minute uses the live tape when it covers. `combined` only when every venue recorded. `missing` names a venue and is not treated as zero |
| `get_market_pressure` | One −1…+1 score from resting book, CVD, and absorption when they disagree; `change1h` is that score vs ~1h ago |
| `get_cvd` | Spot and futures CVD versus price; `range` 24h (5m), 7d (30m), 30d (2h), or 90d (4h) from stored minutes; 15m/1h/4h/24h change on those bars; anchored day (00:00 UTC) and current 4h open; optional `anchor` (RFC3339) adds a click anchor from that candle; venue split, spot-vs-futures split, divergence duration. `whaleDelta` / `retailDelta` split a futures bar when print size was stored (whale is one aggressive print of at least 100,000 USD; retail is smaller). A minute stored before size was known omits both and is not zero. Combined size needs every venue's bar at that step sized. Spot has no split. A minute inside the overlap that one venue missed is left out and is not filled from the other venue. A quiet minute nobody printed is not a gap |
| `get_positioning_anchor` | Open interest, funding, and liquidation notional restarted at one RFC3339 candle. A venue with no sample by that candle is named and left off. Liquidation sum is short up, long down; a first print after the candle is late. `exchange=all` includes Hyperliquid on open interest and hourly funding when a sample exists, and does not name it as a missing liquidation feed |
| `get_tape_backtest` | Replay stored CVD hours, liquidation prints, or spot-book share for one coin and score the spot close 1h, 4h, and 24h later. Combined is withheld when any expected venue has no complete bar. Liquidation bursts are measured prints, not estimated leverage levels |
| `get_volume_profile` | Aggressive buy/sell by printed price (POC + 70% value area) plus OI opened/closed and long/short liquidations on the same rows. Binance, Bybit, and OKX. `combined` only when every venue reaches the window open; `missing` names a later start. `window` or `startTime`/`endTime` |
| `get_around` | What happened around a time: before / during / after (price, volume, VWAP, vs typical, POC, sweeps, stored book/futures, BTC/ETH options when sampled, listed futures curve, tapeRead flow-at-prices; venueTiming says who printed first, whether the other followed, and whether OI or takers had already changed before the laggard's real start; last OI/taker print much earlier than price, a price-start taker bar, or a hole longer than one missing bar at the tape interval, omits continued/faded/reversed) |
| `compare_around` | Compare two times for the same coin: how the two moves differed (price, volume, book, OI, sweeps) |
| `find_around_moves` | Find strong up/down legs and show what happened during each (around tape) |
| `find_around_precursors` | What often changed before those moves, including conditions that fire together and whether they lean up or down |
| `find_around_similar` | Past important-move setups; matching-bar after horizons with similarity bands; each match lists `whySimilar` and per-field `compared` scores; optional `at` uses that time's before-tape |
| `get_market_forensics` | Before / during / after tape plus similar past moves; `why` explains the move in plain language; `timeline` is one clock of OI / takers / price / follow-through / liquidations / options / dealer gamma / futures curve / tapeRead / sweeps; timeline and similar after-windows only include prints known by `asOf` (`timeline.knownUntil`, default now); overlapping bar/sample stamps set `orderUnknown` and do not claim who printed first; a late first sample after missing venue tape is not a follow; unusual BTC/ETH options prints in the before-window are called out (lead is the real start, not the 4h/24h window open); dealer gamma / flip vs spot on BTC/ETH when those samples exist; listed futures curve (contango / backwardation and near-vs-far gap) before the move; last OI/taker print much earlier than price omits continued/faded/reversed |
| `scan_market_forensics` | Coins whose current tape looks like past pre-move setups, which tape pieces matched, plus what usually happened after |
| `review_setup_radar` | Saved Setup Radar hits and whether price later matched the lean |
| `get_vwap` | Volume-weighted average price from a start time (or window) to now; last vs VWAP; Binance, Bybit, combined |
| `get_absorption` | Large market buys/sells vs little price move; absorbing side + strength (15m/1h/4h/24h + current run) |
| `get_sweep_recovery` | How long spot depth took to refill after a liquidity sweep |
| `get_liquidity_sweeps` | Poke through a prior high/low that comes back; level, excursion, reclaim time, volume |
| `get_volume_surge` | Current 5m/15m/1h volume vs that coin's typical (median); buy/sell split |
| `scan_volume_surges` | Rank coins whose volume is much higher than typical |
| `get_rsi_heatmap` | Ranked Wilder RSI scatter for top listed pairs (stables omitted) |
| `get_basis` | Perp vs spot/index premium or discount on Binance, Bybit, and OKX; a missing or unusable quote leaves agreement unknown. `curve` is near vs far dated futures (contango / backwardation) plus `history` / `windows` / `moves` when that spread flipped or got steeper |
| `get_price_correlation` | How similarly a coin moves with BTC and ETH (1h / 4h / 24h) |
| `get_market_breadth` | How many followed coins are up vs down (1h / 4h / 24h), plus BTC/ETH vs the pack |
| `get_coin_situation` | Price versus futures CVD, 1h open interest, the closer estimated liquidation pocket, measured 24h force-order prints, the same prints since 00:00 UTC and the current 4-hour open, OI-weighted funding, Binance and OKX top versus crowd, and whale versus retail. A missing publisher is named and that clause or clock stays blank. A quiet recorded clock is a real zero. A situation alert still compares the 24-hour print side only. A live clause may include `since` when the stored minutes reach a different kind. `changes` lists each minute a stored kind changed. A gap is not a change. The liquidation model leaves `since` off and is not in `changes`. A stored fill omits both. A resting order stores a new read at the fill. Optional `at` (RFC3339) reads the stored five-minute minute at a candle close and leaves the estimated pocket off. Omit `at` for the live read. `/derivatives` and `/unusual` open these clauses under an outlier row |
| `create_situation_alert` | Fire when the live read leaves the clauses stored when the newest open paper buy filled. A blank newest buy does not fire. Repeating re-arm keeps the last fire. The alert still follows the newest open buy. The day and 4-hour print clocks are not part of that comparison |
| `get_fear_greed` | 0–100 fear and greed from our own tape (breadth, BTC move, volatility, funding, long/short, liquidations, perp book, taker flow), which input moved the score the most, and stored 1h/4h/24h/7d history. Not the Binance or CoinGlass index |
| `get_price_volatility` | How much a coin moved over 1h / 4h / 24h vs its normal range and vs BTC/ETH |
| `get_market_snapshot` | Price, volume, mcap, OI, funding, LS, and taker buy/sell together (1h / 4h / 24h) |
| `get_support_resistance` | Support/resistance from price, volume, and the order book, plus breakout score |
| `get_whale_trades` | Largest recent aggressive buys/sells and liquidations; OI says open vs close, plus price and OI 15m later |
| `get_orderbook_history` | Stored spot book at a time (levels, spread, liquidity, walls, brief walls that vanished between minute samples, and other resting-size moves the ladder missed) |
| `compare_orderbook_history` | Which price levels gained or lost liquidity between two times |
| `get_orderbook_icebergs` | Same-price clip eaten then refilled (bid and ask) |
| `get_orderbook_liquidity_flow` | Resting size traded vs pulled vs added (bid and ask) over 5m/15m/1h; `series` is the path over stored samples |
| `estimate_market_impact` | Walk live depth for a simulated market buy/sell; impact only when a new best remains |
| `scan_market_impact` | Rank liquid coins by slippage for the same quote-sized market order |
| `scan_suspicious_walls` | Suspicious and flickering spot walls across the liquid USDT pairs |
| `get_book_gaps` | Empty stretches in the live spot book and whether up or down is thinner |
| `scan_book_gaps` | Liquid pairs ranked by which side of the book is thinner |
| `get_wide_spread` | One coin's top-of-book spread versus the median of earlier samples of that book. Opened at 1.5× typical |
| `scan_wide_spreads` | Coins whose latest spread is at least 1.5× their own typical, widest first |
| `get_candles` | OHLCV |
| `get_supply` | Supply snapshot |
| `get_holders` | Holder count, concentration, top wallets (CMC → Coin Metrics → GeckoTerminal → Ethplorer → Routescan → Tronscan; CryptoID for some UTXO coins). `label` is a public attribution when known |
| `get_asset_profile` | Logo URL, listing date, and published token contracts |
| `list_spot_markets` | Search/sort spot/equity list (`binance` / `coinbase` / `bybit` / `nasdaq` / `bist`) |
| `list_delist_schedule` | Scheduled spot delists per venue (halt `delistTime` + `announcedAt` when the notice was published) |
| `get_post_delist` | Off-venue last + candles after this venue halted the pair (other listed venue or CoinGecko USD) |
| `get_indicators` | RSI/EMA |
| `analyze_swing` | 4h+1d swing engine (quality gates, ATR stop/TP) |
| `scan_swing_setups` | Watchlist swing scan |
| `list_custom_signal_factors` | Mixable swing patterns + hunt_up / forensics / similar_setup and built-in percents |
| `list_custom_signal_profiles` | Saved custom-signal mixes |
| `create_custom_signal_profile` | Save a named mix (weights sum to 100) |
| `update_custom_signal_profile` | Edit a saved mix |
| `duplicate_custom_signal_profile` | Copy a saved mix |
| `set_default_custom_signal_profile` | Mix the Custom tab opens with |
| `delete_custom_signal_profile` | Delete a saved mix |
| `scan_custom_signals` | Watchlist scored with a custom mix (`profileId` or weight*; optional forensics / similar_setup; `tapeSetups` / `tapeUsed` / `tapeThin` / `tapeFresh`; `custom.evidence` strong/fair/weak plus fresh/old/missing factor counts) |
| `start_backtest` | Historical rule or custom-profile backtest (`ruleId` or `profileId=built_in`). `scope=watchlist` (or `symbol=watchlist`) runs a custom mix on every watchlist coin; `startingBalance` (default 10000), `riskPct` (default 1), and `maxOpenRiskPct` (default 3) size fills on mark-to-market equity (open trades included) and cap combined open risk. `useTp2` / `tp1ClosePct` / `stopAfterTp1` scale at TP1; the runner stop can stay, move to entry, or trail with price. `feePct` / `slipPct` apply on each fill (including TP1 slices). A missing next-bar open stores the signal with no trade. Still-open trades stay marked in-range. Custom jobs skip timestamps when a selected factor has no history (`skippedBars` / `missingFactors`) |
| `list_backtests` | List Lab jobs |
| `get_backtest` | Job progress / summary |
| `cancel_backtest` | Cancel pending/running job |
| `list_backtest_signals` | Signals + 1/5/20d returns, first-touch TP/SL trades (same-bar uses a smaller interval when available), and performance (avg / median / win rate / n plus trade stats) |
| `detect_pump_events` | Pump/dump events on one symbol (threshold, interval, lookback) |
| `scan_pump_events` | Scan top-volume symbols for recent pumps (`maxTotalEvents` caps total events; response includes resolved defaults) |
| `get_watchlist` | Read watchlist (optional `ownerClientId` for shared lists) |
| `add_watchlist_item` | Add/update watch item (owner/editor; optional `ownerClientId`) |
| `remove_watchlist_item` | Remove watch item (owner/editor; optional `ownerClientId`) |
| `share_watchlist` | Grant viewer/editor access (owner only; no duplicate grantee) |
| `update_watchlist_share` | Change share role (owner only) |
| `revoke_watchlist_share` | Remove access (owner only) |
| `list_watchlist_shares` | List outgoing shares |
| `list_shared_watchlists` | Lists shared with this client |
| `list_watchlist_audit` | Who changed the list and when |
| `list_price_alerts` | List price alerts for a client. A situation alert includes `situationWatch` (`stored`, `blank`, or `flat`) and, after a fire, `situationFire` (clauses that left, entry kind, later kind, missing venues). Repeating re-arm does not clear that fire |
| `create_price_alert` | Create one-shot above/below price alert |
| `create_orderbook_alert` | Live imbalance or wall alert (edge re-arm, no re-fire while still true) |
| `create_liquidation_feed_alert` | Fire when Binance/Bybit liquidation feed stays down; no re-fire until it recovers |
| `create_liquidation_cascade_alert` | Fire on a big cascade; payload names the exchange and coin |
| `create_liquidation_notional_alert` | Fire when a coin's long/short/total liquidations in 1m/5m/15m/1h exceed a USDT amount; one fire per wave; payload `importance` follows nearby predicted-level confidence |
| `create_liquidation_level_alert` | Fire when last or later prints hit a predicted 10x/25x/50x/100x band that clears min reliability/confidence; high-confidence bands are high importance |
| `create_setup_radar_alert` | Fire when a coin (or any scanned coin) looks like a past pre-move setup; one fire until the match clears |
| `create_tape_alert` | Funding %, open interest USDT, 5m taker buy share, or fear-and-greed score above/below a level. Funding `exchange=all` fires when any venue crosses |
| `delete_price_alert` | Delete a price alert by id |
| `get_alert_webhook` | Get client webhook URL for alert notifications |
| `set_alert_webhook` | Set http(s) webhook URL (durable outbox on trigger) |
| `delete_alert_webhook` | Clear client webhook URL |
| `create_api_key` | Create named `read`/`trade` API key (secret once) |
| `list_api_keys` | List key metadata (no secrets) |
| `revoke_api_key` | Revoke a named API key |
| `create_portfolio` | Create a named paper book (starting balance + optional name) |
| `list_portfolios` | List paper books for a client |
| `rename_portfolio` | Rename a paper book |
| `delete_portfolio` | Delete a paper book and its data |
| `share_portfolio` | Share a book with another client (`viewer` or `trader`) |
| `update_portfolio_share` | Change share role |
| `revoke_portfolio_share` | Remove a share |
| `list_portfolio_shares` | Outgoing shares (owner) |
| `list_shared_portfolios` | Books shared with you |
| `get_situation_change` | Which stored clauses changed. `symbol` compares the newest open buy with the live read. `openReads` lists every remaining buy, newest first, with its size, fill time, stored read, and that buy's comparison when `liveOk` is true. A buy with no stored read keeps its size and time. The situation alert still follows the newest buy. `tradeId` compares a sell with each buy it closed. A blank later clause is missing, not a new kind. `liveOk` false omits the comparison |
| `get_situation_record` | What this book's closed crypto lots made, by the read stored on the buy. Each row is one entry kind and lists those closes, newest first, with symbol, sell time, and realized profit. The same close can appear on more than one row. `skipped` is a crypto close with no stored read. Equities and open positions are left out |
| `get_situation_leave` | Whether those closes kept each entry kind or sold after it left. `still` kept the kind. `left` changed it or left the clause blank. A change in print dollars with the same side stays in `still`. `unreadable` is a buy read with no sell read, and it stays out of both groups. The same close can appear on more than one row. Equities and open positions are left out |
| `get_fill_pocket` | Open paper fill against the nearer estimated liquidation pocket. `gapPct` is the percent from the fill to the pocket. `marginLiqPrice` is the paper liquidation price. A missing venue leaves the pocket blank. `exchange` and `symbol` are required |
| `get_portfolio` | Cash, positions, P&L snapshot. Crypto positions include `entrySituation` from the newest buy still held and `openReads` for every remaining buy. Every open position includes `moves` (1h/4h/24h price change after the newest open lot, including equities). `pending` has not closed its minute. `missing` is a blank close, not zero. The live read is `get_coin_situation`. What changed on each buy is `get_situation_change`. Optional `portfolioId`; required when the client has 2+ books |
| `deposit_portfolio_cash` | Add virtual cash (not trading P&L) |
| `transfer_portfolio_cash` | Move cash between your own books (owner only; pass `idempotencyKey` so retries do not double-move) |
| `list_transfer_attempts` | Transfer request tries: completed, replayed, or conflict |
| `withdraw_portfolio_cash` | Withdraw available virtual cash |
| `list_portfolio_cash_movements` | Deposit/withdraw history (optional `portfolioId`) |
| `get_portfolio_performance` | Equity history + period trading P&L (1d/1w/1m/3m). Deposits, withdrawals, and transfers are `netFlow`, not profit or loss. Largest performance drop with its start and end times |
| `stress_portfolio` | Preview equity, total P&L, and margin after `BTC:-10,ETH:-15` |
| `get_portfolio_correlation` | How open paper coins moved together over the last day. `pairs` leads with the strongest absolute correlation. `swingPct` vs `apartPct`. Optional add line. A coin without enough minutes is named and is not zero |
| `get_portfolio_risk_limits` | Optional risk limits + live status |
| `set_portfolio_risk_limits` | Set daily-loss %, max coin weight %, and/or max total exposure % |
| `clear_portfolio_risk_limits` | Remove all risk limits |
| `get_paper_trading_costs` | Per-exchange paper fee and slippage rates |
| `preview_portfolio_order` | Price a spot or margin order: fee, slippage, cash, and whether risk or the borrow cap would block it. `riskPercent` and `stopLossPrice` size a spot buy, bracket, or margin open from portfolio equity. A spot buy also returns the live coin read (`situationStatus`). `reason` is the clauses that would be stored. When that reason is set and the read is ready, `reasonHistory` is this book's past closes of each selected live kind, split into `still` and `left`. Both counts at zero means no past close stored that kind. The rows are not one total. Nothing is written |
| `place_portfolio_order` | Paper market buy/sell (slipped fill + taker fee; optional idempotencyKey). `reason` names the clauses a buy stores. Omit it to store the whole read. `none` stores no read. A sell keeps the whole read |
| `place_portfolio_pending_order` | Paper limit_buy / limit_sell / stop_loss / trailing_stop (optional idempotencyKey) |
| `place_portfolio_oco_order` | Paper OCO take-profit + stop-loss (linked; optional idempotencyKey) |
| `place_portfolio_bracket_order` | Paper bracket: entry limit buy + pending TP/SL (optional idempotencyKey) |
| `list_portfolio_orders` | List pending paper orders |
| `get_portfolio_order` | Get one pending order + last price + amend hints |
| `amend_portfolio_order` | Amend open GTC limit/stop price and/or remaining size |
| `cancel_all_portfolio_orders` | Cancel all open paper orders, or one market |
| `cancel_portfolio_order` | Cancel open pending paper order |
| `list_portfolio_trades` | Paper trade history. Each fill includes `moves` (1h/4h/24h venue price change after the fill). A blank window is not zero and is not profit |
| `list_portfolio_lots` | Open (or closed) tax lots for FIFO/LIFO sells |
| `create_recurring_buy` | Create named paper recurring buy (schedule, DST-aware timeZone, maxPrice, budget, endDate) |
| `update_recurring_buy` | Update recurring buy name, amount, schedule, timezone/clock, maxPrice, budget, or end |
| `list_recurring_buys` | List recurring buy plans |
| `get_recurring_buy` | Get one recurring buy plan |
| `pause_recurring_buy` | Pause a plan |
| `resume_recurring_buy` | Resume a paused plan |
| `delete_recurring_buy` | Delete a plan and its runs |
| `list_recurring_buy_runs` | Execution history for a plan (succeeded / failed / missed; timing on_time or late) |
| `complete_recurring_buy_run` | Buy a missed slot now at last price (late; does not move next run) |
| `create_portfolio_basket` | Named target mix (does not trade) |
| `list_portfolio_baskets` | List saved baskets |
| `get_portfolio_basket` | Basket + live drift preview |
| `update_portfolio_basket` | Update name/targets (no trades) |
| `delete_portfolio_basket` | Delete basket |
| `preview_portfolio_rebalance` | Proposed sells/buys (no trades) |
| `rebalance_portfolio_basket` | User-triggered rebalance (stores a result) |
| `list_portfolio_rebalances` | Stored rebalance runs (filled / failed / leftover drift) |
| `get_portfolio_rebalance` | One stored run with legs, leftover mix, and attempts |
| `continue_portfolio_rebalance` | Re-plan leftover mix from the current book (new attempt) |
| `set_margin_mode` | Set isolated or cross (locked if open pos/orders) |
| `place_margin_order` | Paper margin long/short open (market/limit, 1x–10x). Stores the coin read from placement. Optional idempotencyKey |
| `list_margin_positions` | Open margin positions, including `entrySituation` from placement |
| `close_margin_position` | Full/partial margin close (optional idempotencyKey) |
| `adjust_margin` | Add/remove isolated position margin |
| `repay_margin_debt` | Pay interest then principal without closing |
| `set_margin_brackets` | Stop-loss / take-profit |
| `list_margin_orders` | Margin limit orders |
| `cancel_margin_order` | Cancel margin limit (releases reserve) |
| `list_margin_trades` | Margin trade history |
| `create_price_diff_watch` | Track a coin on chosen venues; open only when a fresh-book fill meets minProfit for minDurationSec |
| `list_price_diff_watches` | List price-diff watches |
| `get_price_diff_watch` | Get one price-diff watch |
| `update_price_diff_watch` | Change notional, minProfit, minDurationSec, fees, or exchanges; resets duration timer |
| `pause_price_diff_watch` | Pause; close open opportunities; stop searching |
| `resume_price_diff_watch` | Resume; duration timer starts from zero |
| `delete_price_diff_watch` | Delete watch and opportunities |
| `list_price_diff_opportunities` | List open/closed opportunities (duration, peak, fade, close reason, review) |
| `get_price_diff_opportunity` | Get one opportunity plus sampled ticks |
| `quote_price_diff` | Walk two live spot books for a size: avg buy/sell, slippage, profit after fees, usable money, max size |
| `quote_price_diff_opportunity` | Same quote for a stored opportunity (uses that watch's fees) |
| `scan_price_diff_quotes` | Rank every venue pair at one size; missing books listed as unavailable; optional minProfitPct / minProfitAmount |
| `quote_price_diff_watch` | Same scan using a stored watch's symbol and fees |
| `create_scanner_rule` | Create watchlist scanner rule (conditions + matchMode all\|any) |
| `list_scanner_rules` | List scanner rules |
| `update_scanner_rule` | Enable/disable a rule or edit conditions, periods, and thresholds |
| `delete_scanner_rule` | Delete scanner rule |
| `list_scanner_results` | Scanner match history, confluence setups, hits24h |
| `start_export` | Start JSON/CSV export of watchlist, shares, alerts, backtests, portfolios (baskets, rebalance history, risk limits), customSignals |
| `get_export` | Export job status / progress |
| `list_exports` | List recent export jobs |
| `cancel_export` | Cancel pending/running export |
| `preview_import` | Preview restore of an export file (paper books include baskets, rebalance history, and risk limits; counts plus custom-signal addNames / updateNames / conflicts) |
| `confirm_import` | Apply preview with merge or replace |
| `get_import` | Import job status / progress |
| `list_imports` | List recent import jobs |
| `cancel_import` | Cancel previewed/pending/running import |

## Run (integrated — preferred)

MCP is **embedded in the main API process**. One command:

```bash
cd backend && go run ./cmd/server
# REST:  http://localhost:8080/api/v1/...
# MCP:   http://localhost:8080/mcp   (streamable HTTP)
```

### Security

| Control | Env | Notes |
|---------|-----|--------|
| Shared API token | `API_AUTH_TOKEN` | When set, `/mcp` requires master token or a user `trade` key (`Authorization: Bearer` / `X-API-Key`). Query-string secrets are rejected. |
| Remote master | `ALLOW_MASTER_IMPERSONATE` | Default false: a remote master token cannot pass `clientId` on tenant tools. Loopback (AI HTTP tools) still can. |
| Closed account | (in-process tools) | Tools that send `clientId` call `RequireActive` — closed clients get a tool error (HTTP AccountGate cannot read JSON body on `/mcp`) |
| User API key | HTTP identity | `bindMCPTenant` forces tool `clientId` to the key binding (mismatch → error); `create_api_key` / `list_api_keys` / `revoke_api_key` denied for user keys (mirror REST) |
| Disable MCP | `MCP_ENABLED=false` | Do not mount `/mcp` at all |
| Webhooks | (service) | `set_alert_webhook` rejects private/local targets unless `WEBHOOK_ALLOW_PRIVATE=true` |

Leave `API_AUTH_TOKEN` empty only for local single-user development. Do not expose `/mcp` on a public interface without a token.

Optional **stdio** adapter only for hosts that cannot speak HTTP MCP:

```bash
# only if you need pure stdio — API should already be running, or use in-process via server
SWYNGORA_API_URL=http://localhost:8080 go run ./cmd/mcp
# If the API has API_AUTH_TOKEN set, the HTTP client path must send the same token (not yet wired in stdio client — prefer in-process /mcp with auth).
```

## Tests

```bash
go test ./internal/transport/mcp/...
```

## Design

- **In-process** tools call market/watchlist services (no second process, no self-HTTP hop).
- HTTP mount uses mark3labs streamable MCP at `/mcp`.
- Failures return MCP tool errors (not panics).
- New product features that AI should use **must** add a matching MCP tool (see root `AGENTS.md`).