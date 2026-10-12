# Swyngora Backend

Go HTTP API for market data across **Binance**, **Coinbase**, **Bybit** (crypto spot), **Nasdaq**, and **BIST** (cash equities). Futures liquidations / open interest / funding also include **OKX** USDT-swap (not a spot listing venue). Crypto circulating supply remains from the Binance daily snapshot and is **not** applied to stocks. Equity market caps come from the Nasdaq.com screener (USD) and the public TradingView Turkey scanner (TRY).

## Architecture (N-layered)

| Layer | Path | Role |
|---|---|---|
| Transport | `internal/transport/http` | HTTP + WebSocket handlers, CORS, rate limit, JSON mapping |
| Transport | `internal/transport/telegram` | Optional Telegram bot (long-poll → same services) |
| Application | `internal/service/market` | Validation + use-case orchestration |
| Application | `internal/service/realtime` | WebSocket hub: price pump + portfolio event fan-out |
| Domain | `internal/domain` | Entities, ports, sentinel errors |
| Infrastructure | `internal/adapter/*` | Binance (market + supply), Coinbase, Bybit, **OKX (USDT-swap OI / funding / liquidations)**, **Deribit (BTC/ETH options)**, **equities (Nasdaq screener + BIST TradingView scanner; Yahoo for candles)**, TTL cache, **SQLite watchlist** |
| Platform | `internal/platform/config` | Env config |

OpenAPI contract: [`api/openapi/openapi.yaml`](api/openapi/openapi.yaml).

## API

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Liveness |
| `GET` | `/api/v1/realtime` | WebSocket protocol description |
| `GET` | `/api/v1/ws` | WebSocket: live prices + paper portfolio events |
| `GET` | `/api/v1/market/exchanges` | Supported venues |
| `GET` | `/api/v1/market/fx` | Spot FX rates (USD base) plus venue/mcap quote maps and stable aliases |
| `GET` | `/api/v1/market/intervals` | Candle intervals (per `exchange`) |
| `GET` | `/api/v1/market/tags` | Unique Binance product-catalog tags (crypto) |
| `GET` | `/api/v1/market/spot?q=btc&quote=USDT&tag=Meme&sort=quoteVolume` | List/search/filter/sort spot markets |
| `GET` | `/api/v1/market/indicators?symbol=BTCUSDT&interval=1h` | RSI (Wilder) + EMA series |
| `POST` | `/api/v1/market/indicators/batch` | Latest RSI/EMA for up to 50 symbols (bounded concurrency) |
| `GET` | `/api/v1/market/rsi-heatmap` | Ranked Wilder RSI scatter for top listed pairs (stables omitted, forming bar dropped, ~120-bar seed, 60s cache + warm default) |
| `GET` | `/api/v1/market/delist-schedule` | Cached spot delist schedule (`exchange=`; halt `delistTime` + `announcedAt`; last 30 days + next ~31 days; Binance needs `BINANCE_API_KEY`, Bybit is public announcements) |
| `GET` | `/api/v1/market/post-delist` | Off-venue last + candles after this exchange halted the pair (other listed venue or CoinGecko USD; informational) |
| `GET` | `/api/v1/market/orderbook` | Grouped live spot book + ±% pressure/wall analysis |
| `GET` | `/api/v1/market/orderbook/history` | Stored book at a time, or a newest-first list; `flashes` are walls that rested only a few seconds inside that minute; `moves` are other size changes the ladder missed |
| `GET` | `/api/v1/market/orderbook/history/compare` | Which price levels gained or lost liquidity between two times |
| `GET` | `/api/v1/market/orderbook/icebergs` | Same-price clip eaten then refilled (bid and ask) |
| `GET` | `/api/v1/market/orderbook/liquidity-flow` | Resting size that was traded vs pulled vs added (bid and ask) over 5m/15m/1h |
| `GET` | `/api/v1/market/orderbook/combined` | Market-wide pressure from all three venues in one price band |
| `GET` | `/api/v1/market/orderbook/impact` | Simulated market-order fill: average price, slippage, exhausted |
| `GET` | `/api/v1/market/orderbook/impact/scan` | Same quote-sized order ranked by slippage across the liquid pairs |
| `GET` | `/api/v1/market/orderbook/walls/scan` | Suspicious walls (added and pulled at least 4 times in 2 minutes) and brief flickers across the liquid USDT pairs |
| `GET` | `/api/v1/market/orderbook/gaps` | Empty price stretches inside ±2% of mid, and whether a move up or down meets less resting size |
| `GET` | `/api/v1/market/orderbook/gaps/scan` | Same read across the liquid USDT pairs, thinnest one-sided books first |
| `GET` | `/api/v1/market/orderbook/spreads/scan` | Coins whose latest bid-ask spread is at least 1.5× that book's own typical |
| `GET` | `/api/v1/market/orderbook/spread` | One coin's current spread versus its own typical |
| `GET` | `/api/v1/market/orderbook/liquidity` | 0–100 liquidity score from ±0.1/0.5/1% depth; per venue + market-wide |
| `GET` | `/api/v1/market/orderbook/spot-perp` | Spot vs perp resting size at the same ±0.1/0.5/1% of mid; which book is thinner and which side (bid or ask) is thinner |
| `GET` | `/api/v1/market/orderbook/heatmap` | Resting bid/ask size over time. Up to 30 minutes is the live tape; longer `window` values (up to 30 days) use stored ladders plus brief walls and size moves. The coin chart requests the window for the candles on screen |
| `GET` | `/api/v1/market/liquidations` | Rolling 5m/1h/4h/24h futures long/short liquidations (Binance USD-M + Bybit linear); reconnect holes filled from that venue's history when available |
| `GET` | `/api/v1/market/liquidations/overview` | Market-wide 1h/4h/12h/24h totals from stored prints + coins ranked for a treemap |
| `GET` | `/api/v1/market/holders` | Crypto holder count, concentration, and top wallets (CMC → Coin Metrics → GeckoTerminal → Ethplorer → Routescan → Tronscan; CryptoID fallback for UTXO coins like PIVX). Known addresses include a public `label` |
| `GET` | `/api/v1/market/open-interest` | Current futures OI + 1h/4h/24h opened or closed USDT amount and percent, per venue and combined; includes funding. A failed or missing asked venue blanks the combined current and windows and is named in `missing`, and is not counted as zero. A window that lacks a sample on any asked venue is blank even if the others have it |
| `GET` | `/api/v1/market/perp-depth` | Perpetual bid notional within N% below mid vs ask notional above; Binance USD-M + Bybit linear + OKX USDT-swap + combined |
| `GET` | `/api/v1/market/perp-depth/overview` | Total perp bid vs ask across the liquid coins we track, plus which coins make up the biggest share of that book; `bands` show 0.1% vs 1% of mid and who fills each distance; 15m/1h/4h windows include whether total bid+ask got thicker or thinner; `history` is the last 6 hours of minutes; `chart` folds that total across the stored window, up to 30 days; `moves` are the largest 6h jumps and drops with BTC around each, the coins that caused them, and whether Binance, Bybit, and OKX moved together |
| `GET` | `/api/v1/market/perp-depth/book` | Stored perp bid and ask within 0.1%, 0.5%, and 1% of mid for one coin. A minute is kept only when Binance, Bybit, and OKX are all fresh |
| `GET` | `/api/v1/market/derivatives-grid` | Same liquid USDT names, coins down the side and stored time across (`24h` hourly or `7d` UTC days). Series: funding, open-interest change, CVD, or liquidation notional. A cell stays blank until every expected venue has that column. `exchange=hyperliquid` fills funding and open interest only; CVD and liquidation cells stay blank and name that venue |
| `GET` | `/api/v1/market/derivatives-board` | Liquid USDT perpetuals in one table (default 36, max 100): mark, funding, open interest, 1h/24h open-interest change, crowd long percent, top-minus-crowd gap, day CVD, the current price-versus-CVD run, 15m print-size change, 15m order-count ratio, the farthest unusual reading versus that coin's own samples, 5m liquidation heat, 1% bid share, 1h taker buy share, 24h long/short liquidations, and how far back each stored series goes. A combined cell stays empty until every venue that publishes that column is fresh. The 24h open-interest change uses the stored sample within 90 minutes of 24 hours ago when the live series does not reach it, and `change24hMissing` names a venue with no sample. Unusual names its venue and is not averaged. `exchange=hyperliquid` is mark, open interest (mark times coin size), and hourly funding only. Liquidation, tape, and book columns stay blank and are named unsupported. It is not part of `exchange=all` |
| `GET` | `/api/v1/market/options` | BTC/ETH Deribit options: listed OI, 24h volume, put/call, DVOL + ATM IV, 30-day realized vol and `volCompare` (DVOL minus realized); term structure, call/put skew, expiry mix, expected move, max pain, dealer gamma + flip; `history` over hours/days including DVOL vs realized / volGap, net GEX, and flip vs spot; `strikeOi` is call/put open-interest change at each strike over 1h/4h/24h/7d; `moves` when OI/PCR/IV jumps vs typical or gamma / flip-side changes; `vsMarket` when IV or put/call disagrees with spot/futures |
| `GET` | `/api/v1/market/print-size` | Average USDT per aggressive buy/sell fill (5m/15m vs the prior window; few large vs many small). Stored flow minutes fill the window when the live tape does not |
| `GET` | `/api/v1/market/print-size/overview` | Same average summed across the recorder's liquid USDT perpetuals (default 100, max 100, seeds first), plus how many coins got larger or smaller |
| `GET` | `/api/v1/market/bot-tracker` | Same-size aggressive prints that repeat at least 4 times inside any 60s during the last 5m. Buy, sell, and delta count only those clips |
| `GET` | `/api/v1/market/order-count` | Aggressive print counts and (buy − sell) / (buy + sell) from −1 to +1 over 1m/5m/15m. Stored flow minutes fill the window when the live tape does not |
| `GET` | `/api/v1/market/order-count/overview` | Same ratio summed across that recorder list, plus how many coins lean buy or sell. `definite` is false when Binance, Bybit, or OKX is missing or old |
| `GET` | `/api/v1/market/funding-rate` | Predicted next perpetual funding + recent settlements (Binance USD-M + Bybit linear + OKX USDT-swap); `pressure` is OI-weighted combined predicted funding plus 1h/4h/24h change. Combined pressure stays blank until every asked venue is in the blend and is named in `pressure.missing`. A venue with no open interest is still skipped inside a complete blend. `exchange=hyperliquid` is the hourly rate and is not inside that pressure |
| `GET` | `/api/v1/market/funding-arb` | Long cheaper-funding venue / short richer one; sized after-fee payout + spot-perp |
| `GET` | `/api/v1/market/funding-arb/scan` | After-fee winners only (published settlements in the hold window) |
| `GET` | `/api/v1/market/funding-arb/history` | Past after-fee stretches for one coin (`from`/`to`; Binance, Bybit, and OKX; first clock is entry only) |
| `POST`/`GET` | `/api/v1/funding-arb/watches` | Follow the scan (omit `symbol`) or one pair; notify when after-fee net ≥ `minProfit` |
| `GET`/`PATCH`/`DELETE` | `/api/v1/funding-arb/watches/{id}` | Get / edit settings / delete a funding-arb follow |
| `POST` | `/api/v1/funding-arb/watches/{id}/pause` | Pause a follow without deleting it |
| `POST` | `/api/v1/funding-arb/watches/{id}/resume` | Resume a paused follow |
| `GET` | `/api/v1/funding-arb/signals` | Open/closed min-profit crossings (predicted vs settled when clocks have passed) |
| `GET` | `/api/v1/funding-arb/signals/{id}` | One crossing plus predicted-vs-settled review |
| `GET` | `/api/v1/market/long-short-ratio` | Account long/short ratio + recent 5m history on Binance, Bybit, and OKX. Binance and OKX `gap` is top long% minus crowd long%; `divergence` is top vs price |
| `GET` | `/api/v1/market/futures-history` | Durable stored OI / funding / long-short / liquidation history. Optional `bucket` of `5m`, `30m`, `2h`, or `4h` returns one sample per step across at most 90 days. A step with no sample is omitted. Bybit long/short has no top-trader share |
| `GET` | `/api/v1/market/positioning-anchor` | Open interest, funding, and liquidation notional restarted at one candle. A venue with no sample by that candle is named and left off |
| `GET` | `/api/v1/market/unusual` | Whether funding, 24h open-interest change, completed-hour liquidations, completed-day CVD, and hourly spread are high or low versus that coin’s own stored samples. Venues stay separate. Print size stays unknown |
| `GET` | `/api/v1/market/unusual/scan` | Liquid USDT perpetuals (up to 100) whose same readings are high or low versus their own stored samples. Typical and unknown rows are left off |
| `GET` | `/api/v1/market/history-span` | How long futures open interest, the spot book, and flow-at-prices have actually been stored for one coin. Venues with no rows are omitted. Binance tape for the liquid list is walked back about 7 days |
| `GET` | `/api/v1/market/liquidation-max-pain` | Largest estimated liquidation pockets above (shorts) and below (longs) last price — not the hunt target |
| `GET` | `/api/v1/market/liquidation-hunt` | Hypothetical per-venue hunt: spot size to reach liq zones + desk result + up/down ease scores (requested percents stay as given; missing factors add 0; `scoreCompare` has separate up/down factor effects and the largest custom-weight mover per side) + cascade paths |
| `GET` | `/api/v1/market/liquidation-hunt/heatmap` | Labeled leverage model only (prints stay on the measured map): price × time intensity (12h–90d; own-venue prices; Binance, Bybit, OKX; combined only when all three have open interest; windows past a day read one stored open-interest, funding, and long/short sample per bucket, and a column with no sample stays empty; a stored open-interest hole uses that venue's public hourly history within the documented lookback; leverage mix; 1h/4h/12h review still checks the newest 2,500 prints) |
| `GET` | `/api/v1/market/liquidation-prints/heatmap` | Measured prints: stored force-order notional by price and time (12h–90d). A quiet recorded column is a real zero. A column before the watch, or inside a gap with no print, is unrecorded. Combined sums a column only when Binance, Bybit, and OKX were all recorded. No leverage mix and no review horizons |
| `GET` | `/api/v1/market/liquidation-open-levels` | Estimated liquidation levels that stay until a later candle trades through them (12h–90d; own-venue price and open interest; windows past a day read one stored open-interest sample per bucket, and a column with no sample stays empty; first column places the book; later columns add only an increase; a hit is not refilled; combined withheld when any venue has none; leverage mix). Each definite book also reports still-open size since 00:00 UTC and the current 4-hour open. Optional `anchor` adds the same change since that candle. A clock before the first column is not complete and is not zero |
| `GET` | `/api/v1/market/liquidation-levels` | CoinGlass-style price bars. 5x–125x is drawn as 10x/25x/50x/100x at each leverage's own distance, plus the walk from last and a later-print review (12h–30d). Live bars scale by hit rate and, after enough hits, by realized/predicted amount (`sizeScale`). `symbol=all` is stored-print time bars for that range (12h–30d) |
| `GET` | `/api/v1/market/liquidation-cascade` | Short-burst long/short cascade vs typical (1m/5m/15m); combined episodes include who started, follow lag, and per-venue share of the wave |
| `GET` | `/api/v1/market/liquidation-cascade/scan` | Market cascade plus coins currently bursting (venues stay separate) |
| `GET` | `/api/v1/market/squeeze-risk` | Long/short squeeze risk scores per venue + OI-weighted combined. An errored or zero-OI venue withholds the blend (`agreement=withheld`). A zero score is not that blend |
| `GET` | `/api/v1/market/positioning` | Price+OI regime (buildup / unwinding / covering) per venue + combined market. An errored or zero-OI venue withholds the blend (`agreement=withheld`) and `missing` names it |
| `GET` | `/api/v1/market/venue-divergence` | Binance, Bybit, and OKX: who moves together and who moves differently |
| `GET` | `/api/v1/market/taker-flow` | Aggressive futures buy vs sell volume (5m/1h/4h) per venue + combined. Binance, Bybit, and OKX sum measured prints (price × size) |
| `GET` | `/api/v1/market/net-position` | Net long and net short over 15m / 1h / 4h from open interest and taker buy/sell. An incomplete window has empty numeric strings and `complete` false. A missing venue is named and is not a flat zero |
| `GET` | `/api/v1/market/participation` | OI USDT delta ÷ futures taker volume (15m/1h/4h); 2B volume and OI +300M is 15% new positions. A venue without a complete window blanks the combined ratio and is named, not counted as zero |
| `GET` | `/api/v1/market/volume-share` | How BTC futures volume splits across Binance / Bybit / OKX (e.g. 55/30/15) plus who gained or lost share vs the prior window |
| `GET` | `/api/v1/market/pressure` | Combined −1…+1 pressure from resting book, CVD, and absorption, plus 1h change |
| `GET` | `/api/v1/market/cvd` | Spot and futures CVD versus price. Futures CVD is measured prints on Binance, Bybit, and OKX. `range=24h` (default) is 5-minute bars. `7d` is 30-minute, `30d` is 2-hour, and `90d` is 4-hour, folded from stored minutes. 15m/1h/4h/24h change; anchored day (00:00 UTC) and current 4h open. Optional `anchor` (RFC3339) adds a click anchor from that candle. The coin chart plots those anchors, a clicked candle, the running sum on a longer range, and each candle's buy−sell delta. `whaleDelta` and `retailDelta` split a futures bar when print size was stored: a whale is one aggressive print of at least 100,000 USD, and retail is smaller. A minute stored before size was known omits both and is not zero. Spot points have no split. A minute one venue missed inside the overlap is left out and is not zero. A minute nobody printed is not a gap |
| `GET` | `/api/v1/market/tape-backtest` | Replay a stored CVD hour, liquidation-print burst, spot-book share, settled funding print, 24h open-interest change, crowd-versus-top gap, or large-print share. Scores the spot close 1h, 4h, and 24h later, including the worst drop on the way. Combined is withheld when any expected venue has no complete bar (`docs/features/tape-backtest.md`) |
| `GET` | `/api/v1/market/volume-profile` | Aggressive buy/sell by printed price (POC + 70% value area) plus OI opened/closed and long/short liquidations on the same rows. Binance, Bybit, and OKX. Combined only when every venue reaches the window open |
| `GET` | `/api/v1/market/flow-at-prices` | Buy vs sell USDT, print count, and delta at each price. Windows 5m to 7d. Live tape is about 1h (Bybit: newest ~1000 trades). Binance minutes for the liquid list are filled from public aggTrades, about 7 days. Bybit and OKX stay on minutes this process recorded |
| `GET` | `/api/v1/market/flow-at-prices/candles` | Buy and sell inside each candle from stored minutes (about 30 days). The open minute uses the live tape when it covers. Combined only when every venue recorded. At most 400 candles and 24 price rows |
| `GET` | `/api/v1/market/around` | What happened around a time (before / during / after): price, volume, VWAP, vs typical, POC, sweeps, stored book/futures, BTC/ETH options when sampled; `tapeRead` is flow at prices |
| `GET` | `/api/v1/market/around/compare` | How two times / moves of the same coin differed (price, volume, book, OI, sweeps) |
| `GET` | `/api/v1/market/around/moves` | Strongest recent up/down legs plus what happened during each |
| `GET` | `/api/v1/market/around/precursors` | What often changed before those moves, including conditions that fire together |
| `GET` | `/api/v1/market/around/similar` | Past important-move setups like the current tape (optional `at`); unique events; caller-picked after horizons |
| `GET` | `/api/v1/market/forensics` | Around-the-move tape plus similar past setups for a chosen time; `timeline` and the similar after-windows stop at `asOf` / `knownUntil` |
| `GET` | `/api/v1/market/forensics/scan` | Coins whose current tape looks like past pre-move setups, plus what usually happened after |
| `GET` | `/api/v1/market/forensics/radar` | Saved Setup Radar hits and what price did after, plus lean usefulness |
| `GET` | `/api/v1/market/vwap` | Volume-weighted average price from a start time; last vs VWAP; Binance, Bybit, combined |
| `GET` | `/api/v1/market/absorption` | Large market buys/sells vs price hold; which side is absorbing and how strong |
| `GET` | `/api/v1/market/liquidity-sweeps` | Poke through a prior high/low that comes back; level, excursion, time, volume |
| `GET` | `/api/v1/market/orderbook/recovery` | How long spot depth near a swept level took to return to 80% of its pre-poke size |
| `GET` | `/api/v1/market/orderbook/recovery` | How long spot depth near a swept level took to return to 80% of its pre-poke size |
| `GET` | `/api/v1/market/volume-surge` | Current 5m/15m/1h volume vs that coin's typical; buy/sell split |
| `GET` | `/api/v1/market/volume-surge/scan` | Rank coins whose volume is much higher than their own typical |
| `GET` | `/api/v1/market/basis` | Perp vs spot/index premium or discount, trend, funding/OI read; `curve` is near vs far dated futures plus history / windows / moves when that spread flipped or got steeper |
| `GET` | `/api/v1/market/correlation` | How similarly a coin moves with BTC and ETH (1h / 4h / 24h) |
| `GET` | `/api/v1/market/breadth` | How many followed coins are up vs down (1h / 4h / 24h) |
| `GET` | `/api/v1/market/fear-greed` | 0–100 fear and greed from our own tape (breadth, BTC move, volatility, funding, long/short, liquidations, perp book, taker flow), which input moved the score, and stored past scores |
| `GET` | `/api/v1/market/situation` | Combined price-versus-CVD, 1h open interest, the closer estimated liquidation pocket, and measured 24h force-order prints. A missing venue is named and that clause stays blank. A live clause may include `since` when the stored minutes reach a different kind. The liquidation model leaves `since` off. A paper fill stores the read from the fill without that time. An unusual standout on `/derivatives` and each `/unusual` row open this read |
| `GET` | `/api/v1/market/volatility` | How much a coin moved (range + vs normal / BTC / ETH) |
| `GET` | `/api/v1/market/snapshot` | Price, volume, mcap, OI, funding, LS, and taker flow together |
| `GET` | `/api/v1/market/levels` | Support and resistance areas plus breakout strength |
| `GET` | `/api/v1/market/whales` | Large trades and liquidations, biggest first |
| `GET` | `/api/v1/market/pumps` | Mechanical pump/dump events for one symbol |
| `GET` | `/api/v1/market/pumps/scan` | Ranked pump hits across top-volume symbols |
| `GET` | `/api/v1/watchlist` | Get watchlist + `version` (`clientId` / optional `ownerClientId`) |
| `POST` | `/api/v1/watchlist/items` | Add symbol (owner or editor; optional `baseVersion`) |
| `DELETE` | `/api/v1/watchlist/items?exchange=&symbol=` | Remove symbol (`baseVersion` / If-Match) |
| `PUT` | `/api/v1/watchlist` | Replace list (owner; `baseVersion` + optional `baseItems` for multi-device) |
| `GET`/`POST`/`PATCH`/`DELETE` | `/api/v1/watchlist/shares` | List / grant / update role / revoke shares |
| `GET` | `/api/v1/watchlist/shared` | Lists shared with the caller |
| `GET` | `/api/v1/watchlist/audit` | Change history (who/when) |
| `GET` | `/api/v1/alerts` | List price alerts (`clientId` / `X-Client-Id`) |
| `POST` | `/api/v1/alerts` | Create price, book, liquidation, or Setup Radar alert |
| `GET` | `/api/v1/alerts/{id}` | Get one alert |
| `DELETE` | `/api/v1/alerts/{id}` | Delete an alert |
| `GET`/`PUT`/`DELETE` | `/api/v1/alerts/webhook` | Get / set / clear alert webhook URL |
| `POST` | `/api/v1/portfolio` | Create named paper book (starting balance, optional name) |
| `GET` | `/api/v1/portfolio` | Cash, positions, P&L for selected book (`portfolioId`) |
| `GET` | `/api/v1/portfolios` | List paper books |
| `GET` | `/api/v1/portfolios/shared` | Books shared with the caller |
| `PATCH`/`DELETE` | `/api/v1/portfolios/{id}` | Rename or delete a book |
| `POST`/`GET`/`PATCH`/`DELETE` | `/api/v1/portfolio/shares` | Share a book (`viewer`/`trader`), list, update, revoke |
| `POST` | `/api/v1/portfolio/deposits` | Add virtual cash |
| `POST` | `/api/v1/portfolio/withdrawals` | Withdraw available virtual cash |
| `POST` | `/api/v1/portfolio/transfers` | Move cash between own books (owner only; optional Idempotency-Key) |
| `GET` | `/api/v1/portfolio/transfers/attempts` | Transfer request tries (completed / replayed / conflict) |
| `GET` | `/api/v1/portfolio/cash-movements` | Deposit / withdraw / transfer history |
| `GET` | `/api/v1/portfolio/performance` | Equity history + period P&L (`1d`/`1w`/`1m`/`3m`) plus the largest peak-to-trough drop and when it ran |
| `POST` | `/api/v1/portfolio/stress` | What-if: equity, total P&L, and margin after asset percent moves (e.g. BTC −10, ETH −15) |
| `GET`/`PUT`/`DELETE` | `/api/v1/portfolio/risk-limits` | Optional risk brakes (daily loss %, max coin weight, max total exposure); block new buys/margin only |
| `GET` | `/api/v1/portfolio/trading-costs` | Per-exchange paper taker fee and slippage |
| `POST` | `/api/v1/portfolio/orders` | Paper market or pending (`limit_buy` / `limit_sell` / `stop_loss`); optional `Idempotency-Key` |
| `GET` | `/api/v1/portfolio/orders` | List pending orders (default: open) |
| `GET` | `/api/v1/portfolio/orders/{id}` | One pending order + last price + amend hints |
| `PATCH` | `/api/v1/portfolio/orders/{id}` | Amend open GTC limit/stop price and/or remaining size |
| `POST` | `/api/v1/portfolio/orders/cancel-all` | Cancel all open paper orders, or one market (`symbol`) |
| `DELETE` | `/api/v1/portfolio/orders/{id}` | Cancel an open pending order |
| `GET` | `/api/v1/portfolio/situation-change` | Which stored situation clauses changed on an open crypto position (`symbol`) or on one fill (`tradeId`). A blank later clause is missing, not a new kind |
| `GET` | `/api/v1/portfolio/situation-record` | Closed crypto lots grouped by the situation stored on the buy. Each row is one entry kind. The same close can appear on more than one row. A crypto buy with no stored read is skipped. Equities and open positions are left out |
| `GET` | `/api/v1/portfolio/trades` | Paper trade history. A sell includes `left`, the buys it closed compared with the read stored on the sell |
| `GET` | `/api/v1/portfolio/lots` | Tax lots (FIFO/LIFO remaining buys) |
| `POST`/`GET` | `/api/v1/portfolio/recurring-buys` | Create / list named paper recurring buy (DCA) plans |
| `GET`/`PATCH`/`DELETE` | `/api/v1/portfolio/recurring-buys/{id}` | Get / update name, schedule, timezone, maxPrice, budget, endDate / delete plan |
| `POST`/`GET` | `/api/v1/portfolio/baskets` | Create / list named allocation baskets (target % mix) |
| `GET`/`PATCH`/`DELETE` | `/api/v1/portfolio/baskets/{id}` | Get with live drift / update / delete |
| `GET`/`POST` | `/api/v1/portfolio/baskets/{id}/preview` · `/rebalance` | Preview or user-triggered rebalance (stores a result) |
| `GET` | `/api/v1/portfolio/baskets/{id}/rebalances` · `/api/v1/portfolio/rebalances` · `/{id}` | Stored rebalance results (fills, failures, leftover mix) |
| `POST` | `/api/v1/portfolio/rebalances/{id}/continue` | Continue leftover mix from the current book (new attempt) |
| `POST` | `/api/v1/portfolio/recurring-buys/{id}/pause` | Pause plan |
| `POST` | `/api/v1/portfolio/recurring-buys/{id}/resume` | Resume plan |
| `GET` | `/api/v1/portfolio/recurring-buys/{id}/runs` | Recurring buy execution history (includes missed slots; `timing` on fills) |
| `POST` | `/api/v1/portfolio/recurring-buys/{id}/runs/{runId}/complete` | Buy a missed slot now (late; does not move next run) |
| `POST`/`GET` | `/api/v1/portfolio/margin/orders` | Paper margin open (market/limit long/short 1x–10x; optional `Idempotency-Key`) / list |
| `DELETE` | `/api/v1/portfolio/margin/orders/{id}` | Cancel margin limit order |
| `GET` | `/api/v1/portfolio/margin/positions` | Open margin positions |
| `GET` | `/api/v1/portfolio/margin/positions/{id}` | Get margin position |
| `POST` | `/api/v1/portfolio/margin/positions/{id}/close` | Full or partial margin close (optional `Idempotency-Key`) |
| `PUT` | `/api/v1/portfolio/margin/positions/{id}/brackets` | Stop-loss / take-profit |
| `GET` | `/api/v1/portfolio/margin/trades` | Margin trade history |
| `POST`/`GET` | `/api/v1/price-diff/watches` | Create / list cross-exchange price difference watches |
| `GET`/`PATCH`/`DELETE` | `/api/v1/price-diff/watches/{id}` | Get / edit settings / delete watch |
| `POST` | `/api/v1/price-diff/watches/{id}/pause` | Pause: stop searching and close open opportunities |
| `POST` | `/api/v1/price-diff/watches/{id}/resume` | Resume: start searching again; duration timer starts from zero |
| `GET` | `/api/v1/price-diff/opportunities` | List opportunities (`status=open\|closed\|all`) with duration / peak / fade / close reason |
| `GET` | `/api/v1/price-diff/opportunities/{id}` | Get opportunity plus sampled path (`ticks`) |
| `GET` | `/api/v1/price-diff/opportunities/{id}/quote` | Walk both books for a size (avg buy/sell, slippage, profit after fees, usable money, max size) |
| `GET` | `/api/v1/price-diff/quote` | Same walk without a stored opportunity (`buyExchange` / `sellExchange` / `notional`) |
| `GET` | `/api/v1/price-diff/quote/scan` | Rank every venue pair at one size (fees + live depth + how much money can be used) |
| `GET` | `/api/v1/price-diff/watches/{id}/quote` | Same scan using a watch's symbol and fees |
| `POST` | `/api/v1/scanner/rules` | Create scanner rule (RSI / MA / volume, combined with matchMode all\|any) |
| `GET` | `/api/v1/scanner/rules` | List scanner rules |
| `GET` | `/api/v1/scanner/rules/{id}` | Get scanner rule |
| `PATCH` | `/api/v1/scanner/rules/{id}` | Enable/disable or edit conditions, periods, and thresholds |
| `DELETE` | `/api/v1/scanner/rules/{id}` | Delete scanner rule |
| `GET` | `/api/v1/scanner/results` | Scanner match history plus confluence `setups` and `hits24h` |
| `POST` | `/api/v1/scanner/backtests` | Start historical rule or custom-profile backtest (`ruleId` or `profileId`; `scope=watchlist` runs a custom mix on every watchlist coin) |
| `GET` | `/api/v1/scanner/backtests` | List backtests |
| `GET` | `/api/v1/scanner/backtests/{id}` | Backtest progress/summary |
| `POST` | `/api/v1/scanner/backtests/{id}/cancel` | Cancel backtest |
| `GET` | `/api/v1/scanner/backtests/{id}/signals` | Backtest signals + 1/5/20d returns |
| `GET` | `/api/v1/swing/custom/factors` | Mixable swing + hunt + forensics/similar factors and built-in percents |
| `GET`/`POST` | `/api/v1/swing/custom/profiles` | List / save a named custom mix (weights sum to 100) |
| `GET`/`PATCH`/`DELETE` | `/api/v1/swing/custom/profiles/{id}` | Get / update / delete a saved mix |
| `GET` | `/api/v1/swing/custom/scan` | Watchlist scored with a mix (`profileId` or `weight*`); tape rows include `tapeSetups` / `tapeUsed` / `tapeThin` / `tapeFresh`; `custom.evidence` is strong/fair/weak plus `fresh` / `old` / `missing` factor counts |
| `POST` | `/api/v1/export` | Start JSON/CSV export of own watchlist/shares/alerts/backtests/portfolios/customSignals |
| `GET` | `/api/v1/export` | List export jobs |
| `GET` | `/api/v1/export/{id}` | Export progress / status |
| `POST` | `/api/v1/export/{id}/cancel` | Cancel pending/running export |
| `GET` | `/api/v1/export/{id}/download` | Download completed file (owner only; TTL) |
| `POST` | `/api/v1/import/preview` | Upload export file; get valid/invalid/willAdd/willUpdate/willConflict plus custom-signal `addNames` / `updateNames` / `conflicts` |
| `POST` | `/api/v1/import/{id}/confirm` | Apply preview (`merge` or `replace`) in background |
| `GET` | `/api/v1/import` | List import jobs |
| `GET` | `/api/v1/import/{id}` | Import status / progress |
| `POST` | `/api/v1/import/{id}/cancel` | Cancel preview or running import |
| `GET` | `/api/v1/account` | Account status (active/closed, purgeAt, canReopen) |
| `POST` | `/api/v1/account/close` | Close account (7-day grace; data retained) |
| `POST` | `/api/v1/account/reopen` | Reopen within grace period |
| `GET`/`POST` | `/api/v1/account/api-keys` | List / create named API keys (`read` or `trade`) |
| `DELETE` | `/api/v1/account/api-keys/{id}` | Revoke a key |
| `POST` | `/api/v1/ai/chat` | Proxy to Python multi-agent assistant |
| `POST` | `/api/v1/ai/chat/stream` | NDJSON thinking/tool/final events (web Process panel) |
| `GET` | `/api/v1/market/candles?symbol=BTCUSDT&interval=1h&limit=100` | OHLCV from Binance |
| `GET` | `/api/v1/market/ticker/24h?symbol=BTCUSDT` | 24h stats + base/quote volume |
| `GET` | `/api/v1/market/supply?asset=BTC` | Circulating supply (Binance product catalog) |

Intervals: `1m`, `3m`, `5m`, `15m`, `30m`, `1h`, `2h`, `4h`, `6h`, `8h`, `12h`, `1d`, `3d`, `1w`, `1M`.

Optional candle params: `startTime`, `endTime` (RFC3339 or Unix ms).

**Supply / mcap note:** Circulating / total / max supply is loaded from Binance’s public **marketing symbol list** on a **daily schedule** (default **03:00 UTC**, plus once on startup). Failed refreshes **retry with backoff** (1m→1h) before waiting for the next daily slot. User requests (`/supply`, spot mcap columns) **read from cache only**. Snapshots are **atomically replaced** on successful refresh (last-good retained on failure until safety TTL). Default **`SUPPLY_CACHE_TTL=48h`** so entries cannot live forever after a long outage (set `0` to never expire). Max is null when Binance does not define a hard cap (max mcap may show as infinite **only when a USD price exists**). Sorting by market-cap fields **collapses to one preferred quote pair per base** (USDT-first). Empty supply snapshot → market-cap sort returns `502`. **Delist rows** missing that snapshot get circulating supply from CoinGecko public `/coins/markets` so the Markets mcap column is not blank.

**Watchlist persistence:** client watchlists are stored in **SQLite** (default `data/watchlist.db`) so they survive process restarts. Configure path via `WATCHLIST_DB_PATH`. **Sharing:** owners grant `viewer` or `editor` access to other `clientId`s; editors may add/remove symbols only; all mutations write an audit log (`docs/features/watchlist-sharing.md`). **Multi-device sync:** each list has a monotonic `version`; send `baseVersion` on writes — non-overlapping adds auto-merge; delete-vs-update conflicts return **409** with both sides (`docs/features/watchlist-sync.md`).

**Price alerts:** above/below thresholds (`POST /api/v1/alerts`) with `mode=one_time` or `mode=repeating`. Also `kind=liquidation_feed` (Binance/Bybit socket down too long), `kind=liquidation_cascade` (big cascade on a coin; payload names venue + symbol), `kind=liquidation_notional` (long/short/total USDT in 1m/5m/15m/1h; payload `importance` follows nearby predicted-level confidence), and `kind=liquidation_level` (price or prints hit a predicted 10x/25x/50x/100x band that clears later-print confidence). Repeating kinds fire once, stay quiet while still true, and re-arm when the condition clears. Optional webhook (`/api/v1/alerts/webhook`) supports `deliveryMode=immediate` or `hourly_digest`, plus **quiet hours** (`timeZone` + local start/end; midnight-crossing ranges OK). Delivery waits until quiet hours end; pending rows survive restarts. Webhook URLs are **SSRF-hardened** (no loopback/RFC1918/link-local/metadata; no HTTP redirects). Set `WEBHOOK_ALLOW_PRIVATE=true` only for local tests.

**Cross-exchange price diff:** watches (`/api/v1/price-diff/watches`) take a `notional`, `minProfit`, and optional `minDurationSec`. The worker walks **fresh** live books and opens an opportunity only when that size can be bought and sold and after-fee profit (including slippage) meets the floor for the whole duration — not from ticker last-price gaps or a one-tick spike. Stale books are ignored. Open state is durable; no duplicate while open; re-opens after the fill drops and holds again. While open, each tick stores the latest fill, the **peak** after-fee edge, and when the edge first dropped at least 0.01% from that peak. Close records **why** (`below_min_profit` / `below_min_net` / `thin_book` / `paused` / `venues_changed`) plus the last sample. `GET .../opportunities/{id}` returns that review and the sampled ticks. `PATCH` edits notional, min profit, duration, fees, and `exchanges` (resets the duration timer). Pause marks the watch paused first so an in-flight tick cannot open, then **closes** open opportunities; resume starts the duration timer from zero. Optional `exchanges` (at least two of binance/coinbase/bybit) limits which books are walked. Interval `PRICE_DIFF_CHECK_INTERVAL` (default `30s`). **Executable quote** (`/price-diff/quote` or `/opportunities/{id}/quote`) walks the buy asks and sell bids for a `notional` or `quantity` and returns average prices, slippage, profit after fees, usable money, and max still-profitable size. **Scan** (`/price-diff/quote/scan` or `/watches/{id}/quote`) ranks every venue pair at that size.

**Funding-arb follows:** `POST /api/v1/funding-arb/watches` with `minProfit` and no `symbol` follows the funding-arb scan (one scan follow per client). A background checker (`FUNDING_ARB_CHECK_INTERVAL`, default `30s`) re-scans and notifies the client's alert webhook (`type=funding_arb.triggered`) when a new coin first clears the floor. Same coin + same long/short while still above is not re-notified; a drop then re-cross notifies again. A coin missing from the volume-limited scan is re-quoted and closed only if net is really below the floor. `PATCH` edits min profit and other settings; pause/resume stop evaluation without deleting. Optional `durationHours` stops the follow and closes open signals when time is up. Optional `symbol` follows one pair. SQLite path `FUNDING_ARB_DB_PATH` (default `data/fundingarb.db`). See `docs/features/funding-arb.md`.

**Paper trading:** virtual portfolio (`/api/v1/portfolio`) with starting cash, market buy/sell at last price **plus per-exchange slippage and taker fee**. An open crypto position includes `entrySituation` from the moment the newest buy still held filled; the live read stays `GET /api/v1/market/situation`. The live Now line says `since` when the archive reaches a different kind. The stored entry does not. A resting fill stores a new read then, not the read from placement. Equities omit it. Also pending limit/stop orders with cash/position **reservations** (buy reserve covers slip + fee), **partial fills**, **in-place amend** of open GTC limit/stop (`PATCH .../orders/{id}`), and **GTC/IOC/FOK** (+ optional GTC `expiresAt`) via the background filler, open positions, realized/unrealized P&L, trade history, **recurring buy (DCA) plans** (IANA `timeZone` + local clock, optional `maxPrice` after fee and slippage), and **isolated margin** long/short (1x–10x, market/limit, liquidation, partial close, SL/TP). Simulated only — not real money. SQLite path `PORTFOLIO_DB_PATH` (default `data/portfolio.db`); order check interval `PORTFOLIO_ORDER_CHECK_INTERVAL` (default `15s`); recurring buy interval `RECURRING_BUY_INTERVAL` (default `30s`). Live prices + order/position events: `GET /api/v1/ws` (`docs/features/realtime.md`). Rates: `GET /api/v1/portfolio/trading-costs`.

**Indicator scanner:** create RSI / EMA crossover / volume-increase rules for the client's watchlist (`/api/v1/scanner/rules`). Select one or more conditions and `matchMode` `all` (every selected condition) or `any` (one is enough); a hit stores one combined result. Combo rules load enough candles for the longest selected condition. `PATCH` enables/disables a rule or edits periods and thresholds without deleting it. A background job evaluates **enabled** rules on `SCANNER_CHECK_INTERVAL` (default `60s`) and writes a hit only when the rule **becomes** true (not on every later bar that stays true). **Historical backtests** (`/api/v1/scanner/backtests`) snapshot a scanner rule **or** a custom signal profile (`profileId`, or `built_in`) when queued, then replay that copy over a date range for one symbol, track progress, support cancel, and report 1/5/20-day forward returns per signal. Custom-mix jobs use closed 4h swing factors; the score still uses whatever factors have data. A selected factor with no history skips that timestamp (no signal) and the job reports skippedBars / missingFactors. A signal fires when the mix first crosses 62 on a complete bar. `scope=watchlist` with a custom mix replays every watchlist coin in one job (zero-signal and fetch-error coins stay in `performance.markets`). Optional `startingBalance` (default 10000), `riskPct` (default 1), and `maxOpenRiskPct` (default 3) size each fill and cap combined open-stop risk. Still-open trades stay marked at the price at range end (not a later candle close). Same-time signals give risk to the higher mix score first. `rangeEndBalance` is the pot at the date range; `afterRangeBalance` includes later TP/SL only. Fills start at the next bar open (missing open = no trade) and size risk from mark-to-market equity including open fills. Optional `useTp2` closes part of the fill at TP1 and leaves a runner for TP2 (`tp1ClosePct`, `stopAfterTp1=keep|entry|trail`). `feePct` applies on each fill. `slipPct` is the fallback when that minute has no stored order book; otherwise the shared pot walks the stored ladder for the trade size. A close only fills the size on that side — selling 10 into a 0.01 bid leaves the rest open. Job results include 1d / 5d / 20d average, median, win rate, and sample count, plus first-touch TP/SL trades (side, leverage, entry, stop, target, outcome, leveraged PnL). If one bar tags both TP and SL, a smaller interval decides which printed first. SQLite path `SCANNER_DB_PATH` (default `data/scanner.db`). **Stored tape replay** (`GET /api/v1/market/tape-backtest`) is separate from those jobs: it scores a finished CVD hour, a 5-minute liquidation-print burst, or a 15-minute spot-book share over the next 1h, 4h, and 24h. Combined is withheld when any expected venue has no complete bar (`docs/features/tape-backtest.md`).

**Swing engine:** `GET /api/v1/market/swing?symbol=` analyzes one pair on **closed** 4h+1d bars (Wilder RSI/ADX/ATR, SuperTrend, MACD, volume/BB, BTC regime, min R:R 1.8 for trigger, suggested 1–10x leverage on the stop). `GET /api/v1/swing/setups` scans the watchlist (max 25). **Custom mixes** (`/api/v1/swing/custom/*`) let a client pick those pattern weights plus optional liquidation-hunt up score (must sum to 100; missing hunt adds 0). Saved profiles live in the scanner SQLite file. Informational only.

**User data export:** `POST /api/v1/export` queues a JSON or CSV dump of the caller's watchlist, shares, alerts, backtests, paper portfolios (including allocation baskets, rebalance history, and risk limits), and custom signal profiles. One active job per client; poll progress; cancel supported; download is owner-only; files expire (`EXPORT_FILE_TTL`, default 1h). See `docs/features/user-data-export.md`.

**User data import:** `POST /api/v1/import/preview` uploads a prior export and returns valid/invalid/willAdd/willUpdate/willConflict counts; custom signal mixes include `addNames` (new), `updateNames` (exactly one saved mix), and `conflicts` (more than one file row or saved mix with the same name). `confirm` with `merge` or `replace` applies in the background; pass `customSignalResolutions` to pick which saved mix to update — omitted conflicts are skipped. See `docs/features/user-data-import.md`.

**Account close:** `POST /api/v1/account/close` closes a `clientId` for 7 days (reopen allowed); product APIs and shared-list access stop; paper recurring plans are paused and open paper/margin orders canceled; workers skip the tenant. `X-Client-Id`, `?clientId=`, and JSON `clientId` must agree (else 400). Paper `PlaceOrder` and Telegram Confirm also refuse a closed owner. After grace, watchlists/shares/alerts/backtests/import-export files, paper books, price-diff watches, and funding-arb watches are purged. See `docs/features/account-close.md`.

**Hardening:** per-IP rate limits with **capped bucket map**; Binance REST is capped at 2 in flight and pauses only Binance after HTTP 429, HTTP 418, or a used-weight header at 1,800 or more, so background sampling cannot keep calling a banned IP while Coinbase, Bybit, and OKX continue; sanitized public errors; candle/ticker singleflight; bounded candle + watchlist client maps; non-crypto product filter **fails closed** without last-good catalog (no equities/commodities as crypto); indicator batch uses process-wide upstream semaphore; **webhook SSRF blocks** private destinations; paper portfolio mutations **serialized per `clientId`** (service mutex + store write lock); optional **`API_AUTH_TOKEN`** protects tenant APIs + `/mcp` (market GETs stay public); closed `clientId`s are blocked on tenant REST and on MCP tools that send `clientId`; **`MCP_ENABLED=false`** unmounts MCP.

**Auth note:** `clientId` / `X-Client-Id` is still a client-supplied label (not end-user login). For any network exposure set `API_AUTH_TOKEN` (and prefer non-`*` CORS). Users mint named `swy_…` keys (`read` or `trade`) so they do not share the master token (`docs/features/api-keys.md`). **User keys force their bound `clientId`**. The master token may impersonate a tenant only from **loopback** (or `ALLOW_MASTER_IMPERSONATE=true`). Query-string secrets are rejected; browsers mint `POST /api/v1/realtime/ticket`. Empty `API_AUTH_TOKEN` open mode is only allowed on **loopback** unless `ALLOW_OPEN_AUTH=true`. Default listen is `127.0.0.1:8080`. Full multi-user identity (JWT/session) is a separate follow-up.

## Run

```bash
# from backend/
go run ./cmd/server
# listens on 127.0.0.1:8080 by default
# optional Telegram bot starts when TELEGRAM_BOT_TOKEN is set
```

Optional local secrets (gitignored):

```bash
cp .env.example .env   # or place .env at repo root
# set TELEGRAM_BOT_TOKEN + TELEGRAM_CHAT_ID
go run ./cmd/server
```

After editing `.env` / bot tokens, **restart the server** so config reloads.

### Telegram bot commands

Enabled when `TELEGRAM_BOT_TOKEN` is non-empty. Calls **market**, **watchlist**, and **paper portfolio** services in-process (no HTTP hop).

| Command | Description |
|---------|-------------|
| `/price <symbol> [exchange]` | 24h ticker |
| `/spot [exchange] [query]` | Top by quote volume |
| `/lowmcap [exchange\|all] [n]` | Lowest circulating market cap |
| `/mcap <asset\|pair>` | Supply snapshot |
| `/rsi <symbol> [interval] [exchange]` | RSI + EMA |
| `/oi <symbol> [binance\|bybit\|all]` | Futures open interest + change + funding |
| `/perpliq [binance\|bybit\|all] [n]` | Total perp bid vs ask across liquid coins |
| `/funding <symbol> [binance\|bybit\|all]` | Current funding rate + recent history |
| `/fundingarb <symbol> [notional] [hours]` | Long/short venues + after-fee funding (`/fundingarb scan`) |
| `/ls <symbol> [binance\|bybit\|all]` | Long/short account ratio + recent history |
| `/exchanges` | Venues |
| `/watch` · `add` · `del` · `top` | Per-user watchlist (`tg-<user_id>`) |
| `/portfolio` · `/portfolio create [balance]` | Paper portfolio (`tg-<user_id>`) |
| `/deposit` · `/withdraw` `<amount> [note]` | Add or remove virtual cash |
| `/cash` | Deposit/withdraw history |
| `/buy` · `/sell` `<symbol> <qty> [exchange]` | Paper trade with Confirm / Cancel buttons |

See [`docs/features/telegram-bot.md`](../docs/features/telegram-bot.md).

### Environment

| Variable | Default | Purpose |
|---|---|---|
| `HTTP_ADDR` | `127.0.0.1:8080` | Listen address (loopback default) |
| `API_AUTH_TOKEN` | _(empty = open local mode on loopback only)_ | Master token for key management + loopback tenant access; user `swy_…` keys for remote tenants |
| `ALLOW_OPEN_AUTH` | `false` | Permit empty `API_AUTH_TOKEN` when `HTTP_ADDR` is non-loopback (`0.0.0.0`, `:8080`, LAN) |
| `ALLOW_MASTER_IMPERSONATE` | `false` | Let a remote client use the master token as any `X-Client-Id` |
| `MCP_ENABLED` | `true` | Mount streamable MCP at `/mcp`; set `false` to disable |
| `WEBHOOK_ALLOW_PRIVATE` | `false` | Allow loopback/private webhook targets (local tests only; SSRF risk if true) |
| `TELEGRAM_BOT_TOKEN` | _(empty = disabled)_ | BotFather token; enables Telegram transport |
| `TELEGRAM_CHAT_ID` | — | Allowed chat id (or use `TELEGRAM_ALLOWED_CHAT_IDS`) |
| `TELEGRAM_ALLOWED_CHAT_IDS` | — | Comma-separated allowed chat ids |
| `TELEGRAM_ALLOW_ALL` | `false` | If true and allowlist empty, bot is public; otherwise token without allowlist **does not start** |
| `CORS_ALLOW_ORIGINS` | `*` | Comma-separated browser origins (`*` = any; set exact origins in production) |
| `BOT_DEFAULT_EXCHANGE` | `binance` | Default venue for bot commands |
| `BOT_POLL_TIMEOUT` | `30s` | Telegram long-poll wait |
| `BOT_LOWMCAP_LIMIT` | `10` | Default `/lowmcap` size (max 25) |
| `BINANCE_BASE_URL` | `https://api.binance.com` | Binance Spot REST base |
| `BINANCE_API_KEY` | _(empty)_ | Enables hourly spot delist schedule refresh |
| `DELIST_REFRESH_EVERY` | `1h` | Delist schedule poll interval |
| `DELIST_REFRESH_ON_STARTUP` | `true` | Fetch delist schedule once on start |
| `BINANCE_PRODUCT_BASE_URL` | `https://www.binance.com` | Host for marketing symbol list (supply) |
| `CMC_BASE_URL` | `https://api.coinmarketcap.com` | Public data-api host for holder snapshots |
| `HOLDERS_CACHE_TTL` | `1h` | Per-asset holder snapshot TTL; unpublished assets are negative-cached for the same window |
| `COINBASE_BASE_URL` | `https://api.coinbase.com` | Coinbase public market products |
| `COINBASE_EXCHANGE_URL` | `https://api.exchange.coinbase.com` | Coinbase public candles |
| `BYBIT_BASE_URL` | `https://api.bybit.com` | Bybit v5 public market API |
| `HTTP_CLIENT_TIMEOUT` | `15s` | Upstream HTTP timeout |
| `CANDLE_CACHE_TTL` | `30s` | Candle response TTL (latest-N queries only; ranges not cached) |
| `CANDLE_CACHE_MAX_ENTRIES` | `512` | Max candle cache keys (memory bound) |
| `TICKER_CACHE_TTL` | `15s` | Ticker response TTL |
| `ORDERBOOK_CACHE_TTL` | `2s` | Reserved TTL for any leftover REST book cache (live books are not cached) |
| `BINANCE_WS_URL` | `wss://stream.binance.com:9443` | Binance spot stream host for the live local book |
| `BINANCE_FUTURES_BASE_URL` | `https://fapi.binance.com` | Binance USD-M REST (open interest) |
| `HYPERLIQUID_INFO_URL` | `https://api.hyperliquid.xyz` | Hyperliquid public info. Mark, open interest, and hourly funding. No key |
| `DERIBIT_BASE_URL` | `https://www.deribit.com` | Deribit public REST (BTC/ETH options) |
| `OPTIONS_CACHE_TTL` | `30s` | Options overview cache |
| `OPTIONS_HISTORY_DB_PATH` | `data/options.db` | Sampled options OI / put-call / IV |
| `OPTIONS_HISTORY_INTERVAL` | `5m` | How often BTC/ETH desk prints are stored |
| `OPTIONS_HISTORY_RETENTION` | `14d` | How long those samples are kept |
| `BINANCE_FUTURES_WS_URL` | `wss://fstream.binance.com` | Binance USD-M stream for liquidations |
| `OPEN_INTEREST_CACHE_TTL` | `30s` | Futures open-interest snapshot TTL |
| `COINBASE_WS_URL` | `wss://ws-feed.exchange.coinbase.com` | Coinbase Exchange feed for the live local book |
| `BYBIT_WS_URL` | `wss://stream.bybit.com/v5/public/spot` | Bybit spot stream for the live local book |
| `BYBIT_LINEAR_WS_URL` | `wss://stream.bybit.com/v5/public/linear` | Bybit linear stream for liquidations |
| `ORDERBOOK_IDLE_TTL` | `90s` | Drop unused live depth streams |
| `ORDERBOOK_SYNC_TIMEOUT` | `8s` | Max wait for a synced local book |
| `REALTIME_PRICE_INTERVAL` | `5s` | How often subscribed WebSocket prices are pushed |
| `SUPPLY_CACHE_TTL` | `48h` | Safety TTL for supply snapshot (`0` = never expire until successful replace) |
| `SPOT_MARKET_CACHE_TTL` | `5s` | Joined spot prices TTL (exchangeInfo / product tags cached longer in-adapter) |
| `SUPPLY_REFRESH_HOUR` | `3` | Daily product-catalog snapshot hour (local to TZ) |
| `SUPPLY_REFRESH_MINUTE` | `0` | Daily snapshot minute |
| `SUPPLY_REFRESH_TZ` | `UTC` | Timezone for daily schedule (DST-safe calendar math) |
| `SUPPLY_REFRESH_ON_STARTUP` | `true` | Run one snapshot load on process start |
| `CACHE_CLEANUP_EVERY` | `1m` | Background expired-entry cleanup (must be > 0) |
| `RATE_LIMIT_RPS` | `40` | Per-IP request rate (0 disables) |
| `RATE_LIMIT_BURST` | `80` | Per-IP burst size |
| `WATCHLIST_DB_PATH` | `data/watchlist.db` | SQLite file for durable watchlists (created if missing) |
| `ALERTS_DB_PATH` | `data/alerts.db` | SQLite file for durable price alerts |
| `ALERT_CHECK_INTERVAL` | `30s` | How often active alerts are evaluated against last price |
| `WEBHOOK_DELIVERY_INTERVAL` | `5s` | Outbox drain interval for alert webhooks |
| `WEBHOOK_HTTP_TIMEOUT` | `10s` | Per-webhook HTTP timeout |
| `WEBHOOK_MAX_ATTEMPTS` | `8` | Permanent failure after this many delivery attempts |
| `PORTFOLIO_DB_PATH` | `data/portfolio.db` | SQLite file for paper-trading portfolios |
| `PORTFOLIO_ORDER_CHECK_INTERVAL` | `15s` | How often open pending paper orders are evaluated |
| `RECURRING_BUY_INTERVAL` | `30s` | How often due recurring buy plans are evaluated |
| `PORTFOLIO_SNAPSHOT_INTERVAL` | `15m` | Equity sample cadence for performance charts |
| `PORTFOLIO_SNAPSHOT_RETENTION` | `2400h` | How long equity samples are kept (~100 days) |
| `MARGIN_INTEREST_INTERVAL` | `1m` | How often margin debt interest is catch-up accrued (O(1) per position) |
| `PRICE_DIFF_DB_PATH` | `data/pricediff.db` | SQLite for cross-exchange price difference watches/opportunities |
| `PRICE_DIFF_CHECK_INTERVAL` | `30s` | How often active price-diff watches are evaluated |
| `FUNDING_ARB_DB_PATH` | `data/fundingarb.db` | SQLite for funding-arb follow watches/signals |
| `FUNDING_ARB_CHECK_INTERVAL` | `30s` | How often active funding-arb watches are evaluated |
| `RADAR_DB_PATH` | `data/radar.db` | SQLite for saved Setup Radar hits and after-fills |
| `RADAR_SCAN_INTERVAL` | `15m` | How often the worker scans and records new Setup Radar hits |
| `RADAR_FILL_INTERVAL` | `1m` | How often due radar after-windows are filled from candles |
| `SCANNER_DB_PATH` | `data/scanner.db` | SQLite file for indicator scanner rules/results |
| `EXPORT_DB_PATH` | `data/export.db` | SQLite file for user data export jobs |
| `EXPORT_FILE_DIR` | `data/exports` | Directory for export download files |
| `EXPORT_FILE_TTL` | `1h` | How long completed export files remain downloadable |
| `EXPORT_WORKER_INTERVAL` | `2s` | How often pending exports are claimed |
| `IMPORT_DB_PATH` | `data/import.db` | SQLite file for user data import jobs |
| `IMPORT_FILE_DIR` | `data/imports` | Directory for uploaded import files |
| `IMPORT_FILE_TTL` | `1h` | How long preview uploads remain before cleanup |
| `IMPORT_WORKER_INTERVAL` | `2s` | How often pending imports are claimed |
| `ACCOUNT_DB_PATH` | `data/accounts.db` | SQLite for account close/reopen state |
| `ACCOUNT_PURGE_INTERVAL` | `1h` | How often expired closed accounts are purged |
| `FUTURES_HISTORY_DB_PATH` | `data/futures.db` | SQLite archive for OI, funding, long/short, liquidations |
| `FUTURES_HISTORY_INTERVAL` | `5m` | Snapshot worker cadence |
| `FUTURES_HISTORY_RETENTION` | `2160h` | How long stored futures rows and recorded liquidation prints are kept. Startup pages the liquid set (up to 100 names, seeds first) only as far as each endpoint documents (Binance open interest and long/short: 30 days; OKX hourly open interest and long/short: 1,440 points; OKX funding: three months; Bybit open interest back to listing, requested here for 90 days). A Binance pause retries the unfinished series. Public liquidation history still refills about 7 days; older prints are the ones this process already recorded |
| `FUTURES_HISTORY_SYMBOLS` | _(optional CSV)_ | Extra pairs always sampled (majors are built-in) |
| `ORDERBOOK_HISTORY_DB_PATH` | `data/orderbook.db` | SQLite archive for spot book samples |
| `ORDERBOOK_HISTORY_INTERVAL` | `1m` | Book snapshot worker cadence |
| `ORDERBOOK_HISTORY_RETENTION` | `720h` | How long stored books are kept (30 days) |
| `ORDERBOOK_HISTORY_SYMBOLS` | _(optional CSV)_ | Extra pairs always sampled (majors are built-in) |
| `PERP_DEPTH_HISTORY_DB_PATH` | `data/perpdepth.db` | SQLite archive for market-wide perp bid/ask samples |
| `PERP_DEPTH_HISTORY_INTERVAL` | `1m` | Liquid-perp book snapshot cadence |
| `PERP_DEPTH_HISTORY_RETENTION` | `720h` | How long perp-depth samples and per-coin book bands are kept (30 days) |
| `FEAR_GREED_HISTORY_DB_PATH` | `data/feargreed.db` | SQLite archive for fear-and-greed score samples |
| `FEAR_GREED_HISTORY_INTERVAL` | `15m` | How often the index is sampled |
| `FEAR_GREED_HISTORY_RETENTION` | `2160h` | How long those samples are kept (90 days) |
| `FLOW_AT_PRICES_DB_PATH` | `data/flowprices.db` | SQLite archive of buy/sell size at each price |
| `FLOW_AT_PRICES_INTERVAL` | `10s` | How often closed minutes are stored |
| `FLOW_AT_PRICES_RETENTION` | `720h` | How long those minute rows are kept (30 days) |
| `SCANNER_CHECK_INTERVAL` | `60s` | How often scanner rules are evaluated |

No API keys are required for the public endpoints used here. Respect upstream rate limits.

## MCP (AI tools) — same process as HTTP

MCP is **integrated into `cmd/server`**. Starting the backend exposes:

| Surface | URL |
|---------|-----|
| REST API | `http://localhost:8080/api/v1/...` |
| MCP (streamable HTTP) | `http://localhost:8080/mcp` |

```bash
go run ./cmd/server   # both REST and MCP
```

Package: `internal/transport/mcp` (in-process tools → market/watchlist services).  
Optional stdio-only binary: `SWYNGORA_API_URL=http://localhost:8080 API_AUTH_TOKEN=… go run ./cmd/mcp` (not required for normal use; `API_AUTH_TOKEN` or `SWYNGORA_API_TOKEN` is sent as `Authorization: Bearer` when the API is locked).

**Rule:** new agent-useful features must add MCP tools in the same change (root `AGENTS.md` §6.5).

## Test

```bash
go test ./...
go test ./internal/transport/http/ -count=1 -run TestE2E_
go test ./internal/transport/mcp/...
go vet ./...
```

Unit tests mock upstream HTTP; they do not call live Binance. `e2e_findings_test.go` hits the real router (`httptest`) for paper/auth regressions.

### Test coverage by layer (AGENTS.md §6.3 / §6.7)

| Layer | Package | Tests |
|---|---|---|
| Domain | `internal/domain` | `candle_test.go`, `errors_test.go`, `ports_test.go`, `ticker_test.go`, `supply_test.go`, `open_interest_test.go`, `volume_profile_test.go`, `flow_at_prices_test.go`, `book_liquidity_test.go`, `perp_depth_test.go`, `perp_depth_overview_test.go`, `perp_depth_history_test.go`, `absorption_test.go`, `liquidity_sweep_test.go`, `volume_surge_test.go`, `vwap_test.go`, `around_test.go`, `around_compare_test.go`, `around_moves_test.go`, `around_precursors_test.go`, `around_similar_test.go`, `around_options_test.go`, `around_forensics_test.go`, `around_forensics_timeline_test.go`, `around_venue_timing_test.go`, `pricediff_quote_test.go`, `funding_arb_test.go`, `funding_pressure_test.go`, `print_size_test.go`, `longshort_gap_test.go`, `longshort_price_test.go`, `market_pressure_test.go`, `liquidation_test.go`, `liquidation_history_test.go` |
| Application | `internal/service/market` | `service_test.go`, `volumeprofile_test.go`, `flowatprices_test.go`, `bookliquidity_test.go`, `perpdepth_test.go`, `absorption_test.go`, `sweep_test.go`, `volumesurge_test.go`, `vwap_test.go`, `around_test.go`, `funding_arb_test.go` (fakes for ports) |
| Application | `internal/service/futureshist` | `sink_test.go`, `backfill_test.go` (reconnect history fill, no cross-venue) |
| Application | `internal/service/fundingarb` | `service_test.go` (create, min-profit notify, re-arm) |
| Infrastructure | `internal/adapter/fundingarbstore` | `sqlite_test.go` |
| Infrastructure | `internal/adapter/binance` | `client_test.go`, `supply_test.go`, `openinterest_test.go` (`httptest`) |
| Infrastructure | `internal/adapter/cache` | `ttl_test.go` |
| Infrastructure | `internal/adapter/watchliststore` | `memory_test.go`, `sqlite_test.go` (incl. reopen/restart persistence) |
| Infrastructure | `internal/adapter/alertstore` | `sqlite_test.go` (CRUD, one-shot trigger, reopen) |
| Application | `internal/service/pricealert` | `service_test.go` (validation, max, checker once-only, liquidation feed/cascade no re-fire) |
| Transport handlers | `internal/transport/http/handler` | `market_test.go`, `health_test.go`, `respond_test.go`, `pricediff_test.go`, `funding_arb_watch_test.go` |
| Transport middleware | `internal/transport/http/middleware` | `cors_test.go`, `ratelimit_test.go` |
| Transport router | `internal/transport/http` | `router_test.go` |
| Platform | `internal/platform/config` | `config_test.go` |
| Entrypoint | `cmd/server` | No unit tests — process wiring only; covered by router + adapter tests |

`cmd/server/main.go` is intentionally thin (wire deps + listen + graceful shutdown). Behavioral coverage lives in the layers it composes.

## Example curl

```bash
curl -s 'http://localhost:8080/api/v1/market/candles?symbol=BTCUSDT&interval=1h&limit=3' | jq .
curl -s 'http://localhost:8080/api/v1/market/ticker/24h?symbol=BTCUSDT' | jq .
curl -s 'http://localhost:8080/api/v1/market/spot?quote=USDT&sort=quoteVolume&limit=5' | jq .
curl -s 'http://localhost:8080/api/v1/market/supply?asset=BTC' | jq .
```
