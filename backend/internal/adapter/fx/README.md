# adapter/fx

Fetches **spot fiat FX** for display conversion (BIST TRY ↔ Nasdaq USD ↔ EUR).

## Source

[Frankfurter](https://www.frankfurter.app/) — public ECB reference rates, no API key.

`USDT` is treated as `USD` (no paid stablecoin feed).

## Tests

```bash
cd backend
go test ./internal/adapter/fx/...
```
