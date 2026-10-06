# Stored tape replay

## Goal

The signals lab replays candle rules and scores the next 1, 5, and 20 days.
Minutes of CVD, liquidation prints, and spot-book samples are already stored.
This replay scores those minutes and the spot close 1 hour, 4 hours, and 24 hours later.

## Behavior

`GET /api/v1/market/tape-backtest?symbol=BTCUSDT` is synchronous. It is not a scanner backtest job.

- `condition=cvd` fires when a finished UTC hour, with at least 50 distinct minute buckets, crosses a buy or sell share. The previous clock hour must exist and must be below the line. Default share is 0.65 (0.55–0.90). `65` is read as 0.65. The in-progress hour does not count.
- `condition=liquidation` fires when a 5-minute notional of the chosen side is at least a multiple of the median of that side's positive 5-minute buckets in the loaded window. Default multiple is 4 (2–12). At least 5 positive buckets are required. The side is the position that was liquidated (`long` or `short`). These are measured force-order prints. They are not estimated leverage levels.
- `condition=book` resamples stored spot ladders to 15 minutes and keeps the last sample in the block. Bid or ask share uses the same 0.65 band. Spot venues are Binance, Coinbase, and Bybit. Samples with no bid and ask size are skipped.
- A cross is the first bar that reaches the line. Staying above it does not fire again.
- `exchange=all` adds a combined series only at timestamps where every expected venue has a complete bar. Notionals are summed, then the share is recomputed. A venue with no complete bar is named in `missing`, `definite` is false, and `combined` is omitted. That venue is not filled with zero. Per-venue sections still show the venues that have bars.
- CVD and liquidation expect Binance, Bybit, and OKX. The book expects Binance, Coinbase, and Bybit.
- Forward returns use the spot 5-minute close at the signal, then 1 hour, 4 hours, and 24 hours later. A horizon stays blank until that candle has closed. Prices prefer Binance spot, then Bybit, then Coinbase.
- The default window is the last 7 days. The maximum is 30 days. The UI offers 1, 3, and 7 days.
- Liquidation history used here is the newest 2,500 prints. Book history is the newest 500 snapshots. When a cap is hit, older rows are absent and the note says so. Stats use every cross. The list shows the newest 100.

The Signals page **Tape** tab runs this replay. Informational only — not financial advice.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/tape_backtest.go` |
| Service | `GetTapeBacktest` in `backend/internal/service/market/tape_backtest.go` |
| HTTP | `GET /api/v1/market/tape-backtest` |
| MCP / AI | `get_tape_backtest` |
| Web | `/signals` tab Tape (`TapeBacktestPanel`) |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ ./internal/transport/http/handler/ ./internal/adapter/futuresstore/ -count=1 -run "TapeBacktest|ListTakerBuckets_OKX"
curl "http://127.0.0.1:8080/api/v1/market/tape-backtest?symbol=BTCUSDT&condition=cvd&exchange=binance"
```

Open `/signals`, choose the Tape tab, and press Replay.
