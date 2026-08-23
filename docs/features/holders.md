# Feature: Crypto holder snapshots

## Problem / goal

Coin detail already shows supply. Users also want **who holds the coin**: address count,
concentration (top 10 / 50 / 100 %), and the largest wallets.

## Behavior

- `GET /api/v1/market/holders?asset=BTC` (or `symbol=BTCUSDT`)
- Crypto only. Equities skip the query in product UI.
- `404` `catalog_unmapped`: no Binance marketing `cmcUniqueId` **and** no marketing `slug`.
- `404` `holders_unpublished`: CMC answered but published no holder table (`holders` empty and `cdpTotalHolder` is 0/absent).
- Fetch uses `cmcUniqueId` when present, otherwise `GET …/detail?slug=` from the Binance marketing slug **or** the lowercased base ticker (so PIVXUSDT still hits CMC even when marketing has no id).
- When the `holders` object is empty, a positive `cdpTotalHolder` is used as the wallet count.
- If CMC still has no table (`holders_unpublished`) or no catalog row, **Chainz CryptoID** is tried (`/{ticker}/api.dws?q=addresses` + `q=rich`). That is how PIVX (and similar UTXO coins) get a holder count and top wallets.
- Response includes `holderCount`, optional `dailyActive`, top-10/20/50/100 share
  percents, up to 20 `topHolders` (each may have a `label`), and `stale` when last-good is served.
- `label` is a conservative public attribution when the exact address is widely
  published (BitInfoCharts / ChainQuery / Etherscan nametags, plus famous
  historical wallets). Unknown wallets stay unlabeled. This is **not** identity proof.
- Ticker → CoinMarketCap id comes from the daily Binance marketing snapshot
  (`cmcUniqueId`), including rows that have an id but no supply numbers.
  Pair forms (`BTC-USD`, `ETHTRY`) normalize to the base asset.
- Request path uses a 1h TTL cache (env `HOLDERS_CACHE_TTL`). A 429, upstream
  error, or empty CMC blip serves last-good (`stale: true`) when present.
  Unpublished assets are negative-cached so they do not hammer CMC.
- Informational only — not financial advice.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/holders.go` |
| Catalog | Binance marketing list (`LookupAsset`) |
| Adapter | `backend/internal/adapter/cmc/` |
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

Open `/markets/binance/BTCUSDT?tab=holders` — Holders tab. Wallet size uses share × circulating supply when the raw CMC balance is dust-scale, plus an estimated USD value.

## Limits

- Coverage follows CoinMarketCap’s published holder tables, not every Binance pair.
- Go’s default HTTP/2 client used to get `"holders":{}` even when the public coin
  page has a table (BTC, PEPE). The adapter now uses HTTP/1.1 plus browser
  `Origin` / `Referer` so the payload matches the website.
- Some large coins still have **no CMC holder table** (`holdersFlag: false`,
  empty `holders`). There is no free public explorer API that fills those
  without a key-gated vendor (Etherscan v2, Mobula, Debank). ETH often has
  **daily active** only (wallet count stays 0).
- **Verified unpublished (CMC has no table):** ACE, SOL, ADA, AVAX, SUI, XRP, WIF, ARB
  (and others with the same empty `holders` object).
- **Verified unmapped (no Binance id/slug and no CryptoID coin):** EUR, UTK, TON
  (plus other catalog misses such as NFP, LRC, HFT, VANRY). **PIVX** is catalog-unmapped
  on Binance marketing but **is** fetched via CryptoID (do not treat it as permanently empty).
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
