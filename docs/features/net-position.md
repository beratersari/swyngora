# Net long and net short

## Goal

Open interest says the futures book grew or shrank. Taker flow says who hit
the bid or the ask. Neither one, alone, says how much of that change went to
the long side and how much went to the short side.

## Behavior

`GET /api/v1/market/net-position?symbol=BTCUSDT`

For each of **15m**, **1h**, and **4h**:

- `buy` / `sell` — aggressive USDT volume
- `oiDelta` — open interest USDT now minus the reading at the start of the window
- `netLong = (buy − sell + oiDelta) / 2`
- `netShort = (sell − buy + oiDelta) / 2`

Positive means that side grew. Negative means it shrank. The two nets add up
to the open-interest change.

Example: 100M bought, 40M sold, open interest up 30M. Longs are **+45M** and
shorts are **−15M**.

If open interest moves by more than the taker volume, the extra is
`unexplained` and the split uses only the part the prints can account for.

Binance USD-M (5-minute taker bars) and Bybit linear (1-minute prints).
`exchange=all` sums the venues that have both inputs. OKX is not included.

This is not the account long/short ratio. Informational only.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/net_position.go` |
| Service | `backend/internal/service/market/netposition.go` |
| HTTP | `GET /api/v1/market/net-position` |
| MCP / AI | `get_net_position` |
| Web | Coin detail tape and Flow tab `PositionSidePanel` |
| Telegram | `/netpos BTCUSDT` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ -count=1 -run 'NetPosition|GetNetPosition'
curl -s "http://127.0.0.1:8080/api/v1/market/net-position?symbol=BTCUSDT"
```

Open a coin, Tape tab: 15m, 1h, and 4h each show long and short amounts.
