# Feature: Crypto holder snapshots

## Problem / goal

Coin detail already shows supply. Users also want **who holds the coin**: address count,
concentration (top 10 / 50 / 100 %), and the largest wallets.

## Behavior

- `GET /api/v1/market/holders?asset=BTC` (or `symbol=BTCUSDT`)
- Crypto only. Equities skip the query in product UI.
- `404` `catalog_unmapped` / `holders_unpublished` only after every source misses.
- Fetch uses `cmcUniqueId` when present, otherwise `GET …/detail?slug=` from the Binance marketing slug **or** the lowercased base ticker (so PIVXUSDT still hits CMC even when marketing has no id).
- When the CMC `holders` object is empty, a positive `cdpTotalHolder` is used as the wallet count.
- Response includes `holderCount`, optional `dailyActive`, top-10/20/50/100 share
  percents, up to 20 `topHolders` (each may have a `label`), and `stale` when last-good is served.
- `label` is a conservative public attribution when the exact address is widely
  published (BitInfoCharts / ChainQuery / Etherscan nametags, plus famous
  historical wallets). Unknown wallets stay unlabeled. This is **not** identity proof.
- Sources (in order, all free/public, no paid plan):
  1. CoinMarketCap public detail by Binance marketing `cmcUniqueId`
  2. Same CMC detail by `slug` (catalog slug, then lowercased ticker)
  3. Coin Metrics community `AdrBalCnt` (addresses with a balance)
  4. GeckoTerminal token `/info` holder count (CMC profile contracts, then CoinGecko)
  5. Ethplorer ERC-20 count + top wallets (`freekey`)
  6. Routescan EVM explorer (Chiliz fan tokens such as CITY, plus other mapped chains)
  7. Tronscan public `token_trc20` (JST, SUN, WIN, and other TRC-20)
  8. Chainz CryptoID (`/{ticker}/api.dws?q=addresses` + `q=rich`) for UTXO coins such as PIVX
- Pair forms (`BTC-USD`, `ETHTRY`, `ETHBTC`) normalize to the base asset.
  Bare tickers that only happen to end in `BTC` / `ETH` / `BNB` (`WBTC`,
  `STETH`, `WSTETH`) stay intact so the 1h cache cannot serve Wormhole for
  Wrapped Bitcoin (or `ST` for Lido staked ETH).
- Request path uses a 1h TTL cache (env `HOLDERS_CACHE_TTL`). A 429, upstream
  error, or empty CMC blip serves last-good (`stale: true`) when present.
  Unpublished assets are negative-cached so they do not hammer CMC.
- Informational only — not financial advice.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/holders.go` |
| Catalog | Binance marketing list (`LookupAsset`) + CoinGecko platforms |
| Adapter | `cmc`, `geckoterminal`, `ethplorer`, cascade `adapter/holders` |
| Service | `GetHolders` |
| HTTP | `GET /api/v1/market/holders` |
| MCP / AI | `get_holders` |
| Profile | `GET /api/v1/market/asset-profile` + MCP `get_asset_profile` |
| Web | `frontend/src/components/organisms/HolderPanel/` on coin detail |
| Mobile | holder count + top-10 share on coin-detail stats |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/adapter/cmc/ ./internal/adapter/binance/ ./internal/service/market/ ./internal/transport/http/handler/
curl -s 'http://localhost:8080/api/v1/market/holders?asset=BTC' | jq '{asset, holderCount, topTenSharePct, topHolders: .topHolders[:3]}'
```

Open `/markets/binance/BTCUSDT?tab=holders` — Holders tab. The API fills `resolvedBalance` (share × circulating when CMC balance is dust-scale) and `usdValue`. The web table only formats those fields.

## Limits

- Coverage is best-effort. Coins missing from every public feed still 404. The API
  response `source` field says which hop succeeded.
- Go’s default HTTP/2 client used to get `"holders":{}` even when the public coin
  page has a table (BTC, PEPE). The CMC adapter now uses HTTP/1.1 plus browser
  `Origin` / `Referer` so the payload matches the website.
- **2026-08-23 full venue scan** (783 unique Binance/Bybit/Coinbase bases):
  364 published. Remaining gaps are classified in this file’s limits, not a
  volume-top sample — mid-cap fan tokens such as CITY were previously omitted
  from that sample.
- **Hard gaps (no free address-count feed):** native L1s (SOL, AVAX, SUI, APT,
  HBAR, STX, SEI, TIA, TON, VET, MINA, …), BRC-20 (ORDI, 1000SATS), tokenized
  stocks (`AAPLB` / `TSLAX`), and Cosmos / Hedera / VeChain / similar chains.
- **Often recoverable but flaky:** BSC/Base/Optimism tokens (ASTER, KAITO,
  SAPIEN, LAZIO, AERO) when GeckoTerminal 429s. Retry the Holders tab.
- **PIVX** is catalog-unmapped on Binance marketing but **is** fetched via CryptoID
  (do not treat it as permanently empty).
- ETH often has **daily active** only (wallet count stays 0) when no other hop has a table.
- Stocks (`nasdaq` / `bist`) are skipped.
- Public web JSON can change shape — parsing is isolated and fixture-tested.
- Top wallets get a `label` from a curated map of public attributions: Binance
  (incl. BTCB / pool), Bitfinex, Robinhood, Upbit, OKX, Bybit, Crypto.com,
  Gate.io, Bitbank, Tether reserve, Coinbase / Kraken / KuCoin / HTX / Gemini /
  Bitstamp / Poloniex / MEXC (EVM), plus Mt. Gox, Bitfinex hack recovery,
  Silk Road / Silk Road FBI, UK government, PlusToken, Satoshi genesis,
  Hal Finney, and the Bitcoin Pizza payout. CoinMarketCap’s holder list has no
  names. Speculative single-source tags are omitted.
- **Satoshi:** the genesis address `1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa` is labeled
  `Satoshi (genesis)`. It is not in the BTC top 20 (~50 BTC plus donations). The
  large early-miner stash is many addresses (Patoshi clustering), not one wallet —
  we do **not** tag those as Satoshi. `12cbQLTFMXRnSzktFkuoG3eHoMeFtpTu3S` is
  labeled Hal Finney (the first known spend from Satoshi).
- Web Holders tab explains unmapped vs unpublished. Mobile shows the same reason
  string and hides holder tiles on equities.
