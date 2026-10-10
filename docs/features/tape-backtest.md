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
- `condition=funding` fires when a settled funding print, shown in percent, crosses the line after the previous settled hour was on the other side. Default is 0.01 (0.01%), allowed 0.005–0.20. Side `long` means longs pay. Side `short` means shorts pay. Predicted funding is left out. Prints are often 8 hours apart, so the cross compares consecutive settled hours, not the empty hours between them.
- `condition=oi` fires when open interest is at least the chosen percent away from a sample about 24 hours earlier. Default is 5, allowed 1–50. Side `up` or `down`. A sample may sit up to 90 minutes from the hour. A missing sample is left out and is not treated as zero. The previous hour must exist. Combined sums the notionals, then recomputes the percent.
- `condition=gap` fires when top-trader long share minus the crowd, in percentage points, crosses the line. Default is 3, allowed 1–20. Side `top` means top traders are more long. Side `crowd` means the crowd is more long. Position share is used when it is stored. Otherwise the top-account share is used. Bybit does not publish a top-trader ratio, so `exchange=all` uses Binance and OKX, and `exchange=bybit` is rejected.
- `condition=size` fires when a finished UTC hour has at least 50 minutes whose print size is known, and prints of at least $100,000 are at least the chosen share of that hour's aggressive notional. Default is 0.40, allowed 0.15–0.80. Side `whale` is that large-print share. Side `retail` is the smaller-print share. Minutes stored before size was known are left out. Identical clip repeats are not stored, so this replays the size share, not a count of matching prints.
- A cross is the first bar that reaches the line. Staying above it does not fire again.
- `exchange=all` adds a combined series only at timestamps where every expected venue has a complete bar. Notionals are summed, then the share is recomputed. Funding and the crowd gap average the venues that all have the hour. A venue with no complete bar is named in `missing`, `definite` is false, and `combined` is omitted. That venue is not filled with zero. Per-venue sections still show the venues that have bars.
- CVD, liquidation, funding, open interest, and size expect Binance, Bybit, and OKX. The book expects Binance, Coinbase, and Bybit. The crowd gap expects Binance and OKX.
- Forward returns use the spot 5-minute close at the signal, then 1 hour, 4 hours, and 24 hours later. Each horizon also reports the worst drop: the lowest close versus the entry on the way there. Zero means price did not close below the entry. A horizon stays blank until that candle has closed. Prices prefer Binance spot, then Bybit, then Coinbase.
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
