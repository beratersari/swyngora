# adapter/coinmetrics

Public Coin Metrics community API (no key):

`GET /v4/timeseries/asset-metrics?assets={ticker}&metrics=AdrBalCnt&frequency=1d&paging_from=end&page_size=1`

Used as holder fallback 2 after CoinMarketCap. A 403 means that ticker is not in the free community set.

```bash
cd backend && go test ./internal/adapter/coinmetrics/
```
